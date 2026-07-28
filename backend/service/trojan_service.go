package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xenvoid404/neko-tunneling/config"
)

type TrojanService struct {
	cfg    *config.Config
	fileMu sync.Mutex
}

func NewTrojanService(cfg *config.Config) *TrojanService {
	return &TrojanService{cfg: cfg}
}

func (s *TrojanService) AddUser(ctx context.Context, username, password string) error {
	if err := s.addToMemory(ctx, username, password); err != nil {
		return fmt.Errorf("gagal add user Trojan ke memory: %w", err)
	}

	if err := s.addToJSON(username, password); err != nil {
		if rbErr := s.delFromMemory(ctx, username); rbErr != nil {
			return fmt.Errorf("gagal add ke JSON: %w (PENTING: rollback memory juga gagal: %v)", err, rbErr)
		}
		return fmt.Errorf("gagal add ke JSON (berhasil rollback dari memory): %w", err)
	}

	return nil
}

func (s *TrojanService) DelUser(ctx context.Context, username string) error {
	memErr := s.delFromMemory(ctx, username)
	jsonErr := s.delFromJSON(username)

	switch {
	case memErr != nil && jsonErr != nil:
		return fmt.Errorf("gagal hapus dari memory (%v) dan JSON (%v)", memErr, jsonErr)
	case memErr != nil:
		return fmt.Errorf("gagal hapus dari memory, tapi JSON sukses: %w", memErr)
	case jsonErr != nil:
		return fmt.Errorf("hapus dari memory sukses, tapi JSON gagal: %w", jsonErr)
	}

	return nil
}

func (s *TrojanService) addToMemory(ctx context.Context, username, password string) error {
	confPath := s.cfg.XrayTrojanConfPath
	if confPath == "" {
		return fmt.Errorf("path file konfigurasi Trojan belum diatur di config")
	}

	s.fileMu.Lock()
	data, err := os.ReadFile(confPath)
	s.fileMu.Unlock()

	if err != nil {
		return fmt.Errorf("gagal membaca file %s: %w", confPath, err)
	}

	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("gagal parse json: %w", err)
	}

	inbounds, ok := root["inbounds"].([]interface{})
	if !ok {
		return fmt.Errorf("field \"inbounds\" tidak ditemukan di file config")
	}

	newClientObj := map[string]interface{}{
		"id":      password,
		"alterId": 0,
		"email":   username,
	}

	var filteredInbounds []interface{}
	for _, item := range inbounds {
		inbound, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		_, ok = inbound["settings"].(map[string]interface{})
		if !ok {
			continue
		}

		inboundCopy := make(map[string]interface{}, len(inbound))
		for k, v := range inbound {
			if subMap, ok := v.(map[string]interface{}); ok {
				subCopy := make(map[string]interface{}, len(subMap))
				for sk, sv := range subMap {
					subCopy[sk] = sv
				}
				inboundCopy[k] = subCopy
			} else {
				inboundCopy[k] = v
			}
		}

		inboundCopySettings := inboundCopy["settings"].(map[string]interface{})
		inboundCopySettings["clients"] = []interface{}{newClientObj}
		filteredInbounds = append(filteredInbounds, inboundCopy)
	}

	if len(filteredInbounds) == 0 {
		return fmt.Errorf("tidak ada inbound valid yang ditemukan")
	}

	minRoot := map[string]interface{}{"inbounds": filteredInbounds}
	tmpData, err := json.MarshalIndent(minRoot, "", "  ")
	if err != nil {
		return fmt.Errorf("gagal marshal data temp: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "xray-trojan-adu-*.json")
	if err != nil {
		return fmt.Errorf("gagal membuat file temp: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(tmpData); err != nil {
		tmpFile.Close()
		return fmt.Errorf("gagal menulis file temp: %w", err)
	}
	tmpFile.Close()

	cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "xray", "api", "adu", "--server="+s.cfg.XrayAPIAddr, tmpPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gagal eksekusi xray api adu: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	return nil
}

func (s *TrojanService) addToJSON(username, password string) error {
	confPath := s.cfg.XrayTrojanConfPath
	client := map[string]interface{}{
		"id":      password,
		"alterId": 0,
		"email":   username,
	}

	s.fileMu.Lock()
	defer s.fileMu.Unlock()

	data, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("gagal membaca %s: %w", confPath, err)
	}

	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("gagal parse json: %w", err)
	}

	inbounds, ok := root["inbounds"].([]interface{})
	if !ok {
		return fmt.Errorf("field \"inbounds\" tidak ditemukan di file config")
	}

	updatedCount := 0
	for _, item := range inbounds {
		inbound, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		settings, ok := inbound["settings"].(map[string]interface{})
		if !ok {
			continue
		}

		clients, _ := settings["clients"].([]interface{})
		settings["clients"] = append(clients, client)
		updatedCount++
	}

	if updatedCount == 0 {
		return fmt.Errorf("tidak ada inbound yang diperbarui di file JSON")
	}

	return s.atomicWriteJSON(confPath, root)
}

func (s *TrojanService) delFromMemory(ctx context.Context, username string) error {
	confPath := s.cfg.XrayTrojanConfPath
	if confPath == "" {
		return fmt.Errorf("path file konfigurasi Trojan belum diatur di config")
	}

	s.fileMu.Lock()
	data, err := os.ReadFile(confPath)
	s.fileMu.Unlock()

	if err != nil {
		return fmt.Errorf("gagal membaca file %s: %w", confPath, err)
	}

	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("gagal parse json: %w", err)
	}

	inbounds, ok := root["inbounds"].([]interface{})
	if !ok {
		return fmt.Errorf("field \"inbounds\" tidak ditemukan di file config")
	}

	var filteredInbounds []interface{}
	for _, item := range inbounds {
		inbound, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		settings, ok := inbound["settings"].(map[string]interface{})
		if !ok {
			continue
		}

		var targetClient interface{}
		clients, _ := settings["clients"].([]interface{})
		for _, c := range clients {
			cMap, cOk := c.(map[string]interface{})
			if cOk && cMap["email"] == username {
				targetClient = c
				break
			}
		}
		if targetClient == nil {
			continue
		}

		inboundCopy := make(map[string]interface{}, len(inbound))
		for k, v := range inbound {
			if subMap, ok := v.(map[string]interface{}); ok {
				subCopy := make(map[string]interface{}, len(subMap))
				for sk, sv := range subMap {
					subCopy[sk] = sv
				}
				inboundCopy[k] = subCopy
			} else {
				inboundCopy[k] = v
			}
		}

		inboundCopySettings := inboundCopy["settings"].(map[string]interface{})
		inboundCopySettings["clients"] = []interface{}{targetClient}
		filteredInbounds = append(filteredInbounds, inboundCopy)
	}

	if len(filteredInbounds) == 0 {
		return nil
	}

	minRoot := map[string]interface{}{"inbounds": filteredInbounds}
	tmpData, err := json.MarshalIndent(minRoot, "", "  ")
	if err != nil {
		return fmt.Errorf("gagal marshal data temp: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "xray-trojan-rmu-*.json")
	if err != nil {
		return fmt.Errorf("gagal membuat file temp: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(tmpData); err != nil {
		tmpFile.Close()
		return fmt.Errorf("gagal menulis file temp: %w", err)
	}
	tmpFile.Close()

	cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "xray", "api", "rmu", "--server="+s.cfg.XrayAPIAddr, tmpPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gagal eksekusi xray api rmu: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	return nil
}

func (s *TrojanService) delFromJSON(username string) error {
	confPath := s.cfg.XrayTrojanConfPath

	s.fileMu.Lock()
	defer s.fileMu.Unlock()

	data, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("gagal membaca %s: %w", confPath, err)
	}

	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("gagal parse json: %w", err)
	}

	inbounds, ok := root["inbounds"].([]interface{})
	if !ok {
		return fmt.Errorf("field \"inbounds\" tidak ditemukan di file config")
	}

	for _, item := range inbounds {
		inbound, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		settings, ok := inbound["settings"].(map[string]interface{})
		if !ok {
			continue
		}

		clients, _ := settings["clients"].([]interface{})
		filtered := clients[:0]
		for _, client := range clients {
			cMap, ok := client.(map[string]interface{})
			if ok && cMap["email"] == username {
				continue
			}
			filtered = append(filtered, client)
		}
		settings["clients"] = filtered
	}

	return s.atomicWriteJSON(confPath, root)
}

func (s *TrojanService) atomicWriteJSON(filePath string, root map[string]interface{}) error {
	newData, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("gagal marshal json: %w", err)
	}

	dir := filepath.Dir(filePath)
	tmp, err := os.CreateTemp(dir, ".config-*.json.tmp")
	if err != nil {
		return fmt.Errorf("gagal membuat file temp: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(newData); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("gagal menulis file temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("gagal menutup file temp: %w", err)
	}
	if err := os.Chmod(tmpPath, 0644); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("gagal set permission: %w", err)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("gagal rename file: %w", err)
	}

	return nil
}

func GenerateTrojanLink(remark, hostname string, port int, password, net, path string, isTLS bool) string {
	security, sni := "none", ""
	if isTLS {
		security = "tls"
		sni = fmt.Sprintf("&sni=%s", hostname)
	}

	var query string
	switch net {
	case "grpc":
		query = fmt.Sprintf("mode=gun&security=%s&type=grpc&serviceName=%s%s", security, path, sni)
	case "ws", "httpupgrade":
		query = fmt.Sprintf("path=%s&security=%s&host=%s&type=%s%s", path, security, hostname, net, sni)
	}

	return fmt.Sprintf("trojan://%s@%s:%d?%s#%s", password, hostname, port, query, url.QueryEscape(remark))
}
