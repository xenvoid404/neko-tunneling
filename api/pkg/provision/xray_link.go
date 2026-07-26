package provision

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type vmessJSON struct {
	V    string `json:"v"`
	Ps   string `json:"ps"`
	Add  string `json:"add"`
	Port string `json:"port"`
	Id   string `json:"id"`
	Aid  string `json:"aid"`
	Scy  string `json:"scy"`
	Net  string `json:"net"`
	Type string `json:"type"`
	Host string `json:"host"`
	Path string `json:"path"`
	Tls  string `json:"tls"`
	Sni  string `json:"sni"`
	Fp   string `json:"fp"`
}

func GenerateVmess(remark, domain string, port int, uuid, netType, pathOrServiceName string, isTLS bool) (string, error) {
	tlsStr, sniStr := "", ""
	if isTLS {
		tlsStr = "tls"
		sniStr = domain
	}

	cfg := vmessJSON{
		V:    "2",
		Ps:   remark,
		Add:  domain,
		Port: strconv.Itoa(port),
		Id:   uuid,
		Aid:  "0",
		Scy:  "auto",
		Net:  netType,
		Type: "none",
		Host: domain,
		Path: pathOrServiceName,
		Tls:  tlsStr,
		Sni:  sniStr,
		Fp:   "chrome",
	}

	jsonBytes, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("gagal marshal konfigurasi vmess: %w", err)
	}

	return "vmess://" + base64.StdEncoding.EncodeToString(jsonBytes), nil
}

func GenerateVless(remark, domain string, port int, uuid, netType, pathOrServiceName string, isTLS bool) string {
	security, sni := "none", ""
	if isTLS {
		security = "tls"
		sni = fmt.Sprintf("&sni=%s", domain)
	}

	var query string
	switch netType {
	case "grpc":
		query = fmt.Sprintf("mode=gun&security=%s&encryption=none&type=grpc&serviceName=%s%s", security, pathOrServiceName, sni)
	case "ws", "httpupgrade":
		query = fmt.Sprintf("path=%s&security=%s&encryption=none&host=%s&type=%s%s", pathOrServiceName, security, domain, netType, sni)
	}

	return fmt.Sprintf("vless://%s@%s:%d?%s#%s", uuid, domain, port, query, url.QueryEscape(remark))
}

func GenerateTrojan(remark, domain string, port int, password, netType, pathOrServiceName string, isTLS bool) string {
	security, sni := "none", ""
	if isTLS {
		security = "tls"
		sni = fmt.Sprintf("&sni=%s", domain)
	}

	var query string
	switch netType {
	case "grpc":
		query = fmt.Sprintf("mode=gun&security=%s&type=grpc&serviceName=%s%s", security, pathOrServiceName, sni)
	case "ws", "httpupgrade":
		query = fmt.Sprintf("path=%s&security=%s&host=%s&type=%s%s", pathOrServiceName, security, domain, netType, sni)
	}

	return fmt.Sprintf("trojan://%s@%s:%d?%s#%s", password, domain, port, query, url.QueryEscape(remark))
}
