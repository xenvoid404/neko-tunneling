package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/proxy/trojan"
	"github.com/xtls/xray-core/proxy/vless"
	"github.com/xtls/xray-core/proxy/vmess"
)

var (
	xrayOnce      sync.Once
	xrayConn      *grpc.ClientConn
	xrayClient    command.HandlerServiceClient
	xrayConfPaths map[string]string
	confFileMu    sync.Mutex
)

var protocolTags = map[string][]string{
	"vmess":  {"vmess-ws", "vmess-up", "vmess-grpc"},
	"vless":  {"vless-ws", "vless-up", "vless-grpc"},
	"trojan": {"trojan-ws", "trojan-up", "trojan-grpc"},
}

func InitXrayClient(addr string, confPaths map[string]string) error {
	xrayConfPaths = confPaths
	var initErr error
	xrayOnce.Do(func() {
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			initErr = fmt.Errorf("gagal terhubung ke API Xray: %w", err)
			return
		}
		xrayConn = conn
		xrayClient = command.NewHandlerServiceClient(conn)
	})
	return initErr
}

func CloseXrayClient() {
	if xrayConn != nil {
		slog.Info("Menutup koneksi Xray API...")
		if err := xrayConn.Close(); err != nil {
			slog.Error("Gagal menutup koneksi Xray API", slog.Any("error", err))
			return
		}
		slog.Info("Koneksi Xray API berhasil ditutup")
	}
}

func AddXrayUser(ctx context.Context, targetProtocol, username, password string) error {
	if err := addToMemory(ctx, targetProtocol, username, password); err != nil {
		return fmt.Errorf("addToMemory gagal: %w", err)
	}

	if err := addToJSON(targetProtocol, username, password); err != nil {
		if rbErr := delFromMemory(ctx, targetProtocol, username); rbErr != nil {
			return fmt.Errorf("addToJSON gagal: %w (PENTING: rollback addToMemory JUGA gagal -- user %s sekarang aktif di RAM Xray tapi tidak konsisten dengan file, hapus manual: %v)", err, username, rbErr)
		}
		return fmt.Errorf("addToJSON gagal (sudah di-rollback dari memory): %w", err)
	}
	return nil
}

func DelXrayUser(ctx context.Context, targetProtocol, username string) error {
	memErr := delFromMemory(ctx, targetProtocol, username)
	jsonErr := delFromJSON(targetProtocol, username)

	switch {
	case memErr != nil && jsonErr != nil:
		return fmt.Errorf("delFromMemory gagal (%v) dan delFromJSON gagal (%v) -- perlu pembersihan manual di kedua sisi untuk email=%s", memErr, jsonErr, username)
	case memErr != nil:
		return fmt.Errorf("delFromMemory gagal, tapi delFromJSON sukses -- user %s masih AKTIF di RAM Xray sampai restart berikutnya: %w", username, memErr)
	case jsonErr != nil:
		return fmt.Errorf("PENTING: delFromMemory sukses tapi delFromJSON gagal -- user %s akan RESURRECT saat Xray restart berikutnya, hapus manual entry email=%s dari file confdir terkait: %w", username, username, jsonErr)
	}
	return nil
}

func addToMemory(ctx context.Context, targetProtocol, username, password string) error {
	if xrayClient == nil {
		return fmt.Errorf("xray client belum diinisialisasi")
	}

	tags, ok := protocolTags[targetProtocol]
	if !ok {
		return fmt.Errorf("tidak ditemukan inbound tag untuk protocol: %s", targetProtocol)
	}

	account, err := buildAccMemory(targetProtocol, password)
	if err != nil {
		return fmt.Errorf("gagal memformat akun: %w", err)
	}

	operation := serial.ToTypedMessage(&command.AddUserOperation{
		User: &protocol.User{
			Level:   0,
			Email:   username,
			Account: account,
		},
	})

	for _, tag := range tags {
		callCtx, callCancel := context.WithTimeout(ctx, 3*time.Second)
		_, err := xrayClient.AlterInbound(callCtx, &command.AlterInboundRequest{
			Tag:       tag,
			Operation: operation,
		})
		callCancel()

		if err != nil {
			if rbErr := delFromMemory(ctx, targetProtocol, username); rbErr != nil {
				return fmt.Errorf("gagal add user ke inbound %s: %w (rollback tag sebelumnya juga gagal: %v)", tag, err, rbErr)
			}
			return fmt.Errorf("gagal add user ke inbound %s: %w", tag, err)
		}
	}

	return nil
}

func delFromMemory(ctx context.Context, targetProtocol, username string) error {
	if xrayClient == nil {
		return fmt.Errorf("xray client belum diinisialisasi")
	}

	tags, ok := protocolTags[targetProtocol]
	if !ok {
		return fmt.Errorf("tidak ditemukan inbound tag untuk protocol: %s", targetProtocol)
	}

	operation := serial.ToTypedMessage(&command.RemoveUserOperation{Email: username})

	var failed []string
	for _, tag := range tags {
		callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		_, err := xrayClient.AlterInbound(callCtx, &command.AlterInboundRequest{
			Tag:       tag,
			Operation: operation,
		})
		cancel()
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", tag, err))
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("gagal menghapus user dari sebagian inbound gRPC: %s", strings.Join(failed, "; "))
	}
	return nil
}

func addToJSON(targetProtocol, username, password string) error {
	tags, ok := protocolTags[targetProtocol]
	if !ok {
		return fmt.Errorf("tidak ditemukan inbound tag untuk protocol: %s", targetProtocol)
	}

	confPath, ok := xrayConfPaths[targetProtocol]
	if !ok || confPath == "" {
		return fmt.Errorf("path file confdir untuk protokol %s belum dikonfigurasi", targetProtocol)
	}

	client, err := buildAccJSON(targetProtocol, username, password)
	if err != nil {
		return fmt.Errorf("gagal memformat entri client: %w", err)
	}

	confFileMu.Lock()
	defer confFileMu.Unlock()

	data, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("gagal membaca file %s: %w", confPath, err)
	}

	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("gagal parse %s: %w", confPath, err)
	}

	inbounds, ok := root["inbounds"].([]interface{})
	if !ok {
		return fmt.Errorf("format %s tidak sesuai: field \"inbounds\" tidak ditemukan", confPath)
	}

	tagSet := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tagSet[tag] = true
	}

	found := 0
	for _, item := range inbounds {
		inbound, ok := item.(map[string]interface{})
		tagStr, tagOk := inbound["tag"].(string)
		if !ok || !tagOk || !tagSet[tagStr] {
			continue
		}

		settings, ok := inbound["settings"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("inbound %q di %s tidak punya field \"settings\" yang valid", inbound["tag"], confPath)
		}

		clients, _ := settings["clients"].([]interface{})
		settings["clients"] = append(clients, client)
		found++
	}

	if found != len(tags) {
		return fmt.Errorf("hanya %d dari %d inbound tag ditemukan di %s (%v)", found, len(tags), confPath, tags)
	}

	newData, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("gagal marshal %s: %w", confPath, err)
	}

	dir := filepath.Dir(confPath)
	tmp, err := os.CreateTemp(dir, ".config-*.json.tmp")
	if err != nil {
		return fmt.Errorf("gagal membuat file sementara: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(newData); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("gagal menulis file sementara: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("gagal menutup file sementara: %w", err)
	}
	if err := os.Chmod(tmpPath, 0644); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("gagal set permission file sementara: %w", err)
	}
	if err := os.Rename(tmpPath, confPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("gagal rename file sementara ke %s: %w", confPath, err)
	}

	return nil
}

func delFromJSON(targetProtocol, username string) error {
	tags, ok := protocolTags[targetProtocol]
	if !ok {
		return fmt.Errorf("tidak ditemukan inbound tag untuk protocol: %s", targetProtocol)
	}

	confPath, ok := xrayConfPaths[targetProtocol]
	if !ok || confPath == "" {
		return fmt.Errorf("path file confdir untuk protokol %s belum dikonfigurasi", targetProtocol)
	}

	confFileMu.Lock()
	defer confFileMu.Unlock()

	data, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("gagal membaca file %s: %w", confPath, err)
	}

	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("gagal parse %s: %w", confPath, err)
	}

	inbounds, ok := root["inbounds"].([]interface{})
	if !ok {
		return fmt.Errorf("format %s tidak sesuai: field \"inbounds\" tidak ditemukan", confPath)
	}

	tagSet := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tagSet[tag] = true
	}

	for _, item := range inbounds {
		inbound, ok := item.(map[string]interface{})
		tagStr, tagOk := inbound["tag"].(string)
		if !ok || !tagOk || !tagSet[tagStr] {
			continue
		}

		settings, ok := inbound["settings"].(map[string]interface{})
		if !ok {
			continue
		}

		clients, _ := settings["clients"].([]interface{})
		filtered := clients[:0]
		for _, client := range clients {
			clientMap, ok := client.(map[string]interface{})
			if ok && clientMap["email"] == username {
				continue
			}
			filtered = append(filtered, client)
		}
		settings["clients"] = filtered
	}

	newData, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("gagal marshal %s: %w", confPath, err)
	}

	dir := filepath.Dir(confPath)
	tmp, err := os.CreateTemp(dir, ".config-*.json.tmp")
	if err != nil {
		return fmt.Errorf("gagal membuat file sementara: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(newData); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("gagal menulis file sementara: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("gagal menutup file sementara: %w", err)
	}
	if err := os.Chmod(tmpPath, 0644); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("gagal set permission file sementara: %w", err)
	}
	if err := os.Rename(tmpPath, confPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("gagal rename file sementara ke %s: %w", confPath, err)
	}

	return nil
}

func buildAccMemory(targetProtocol, password string) (*serial.TypedMessage, error) {
	switch targetProtocol {
	case "vmess":
		return serial.ToTypedMessage(&vmess.Account{Id: password}), nil
	case "vless":
		return serial.ToTypedMessage(&vless.Account{Id: password}), nil
	case "trojan":
		return serial.ToTypedMessage(&trojan.Account{Password: password}), nil
	default:
		return nil, fmt.Errorf("protocol %s tidak didukung", targetProtocol)
	}
}

func buildAccJSON(targetProtocol, username, password string) (map[string]interface{}, error) {
	switch targetProtocol {
	case "vmess":
		return map[string]interface{}{"id": password, "alterId": 0, "email": username}, nil
	case "vless":
		return map[string]interface{}{"id": password, "email": username}, nil
	case "trojan":
		return map[string]interface{}{"password": password, "email": username}, nil
	default:
		return nil, fmt.Errorf("protocol %s tidak didukung", targetProtocol)
	}
}
