package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type VMessJSON struct {
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
	tlsStr := ""
	sniStr := ""
	if isTLS {
		tlsStr = "tls"
		sniStr = domain
	}

	cfg := VMessJSON{
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
		return "", err
	}

	base64Encoded := base64.StdEncoding.EncodeToString(jsonBytes)
	return "vmess://" + base64Encoded, nil
}

func GenerateVless(remark, domain string, port int, uuid, netType, pathOrServiceName string, isTLS bool) string {
	var query string

	security := "none"
	sni := ""
	if isTLS {
		security = "tls"
		sni = fmt.Sprintf("&sni=%s", domain)
	}

	switch netType {
	case "grpc":
		query = fmt.Sprintf("mode=gun&security=%s&encryption=none&type=grpc&serviceName=%s%s", security, pathOrServiceName, sni)
	case "ws", "httpupgrade":
		query = fmt.Sprintf("path=%s&security=%s&encryption=none&host=%s&type=%s%s", pathOrServiceName, security, domain, netType, sni)
	}

	return fmt.Sprintf("vless://%s@%s:%d?%s#%s", uuid, domain, port, query, url.QueryEscape(remark))
}

func GenerateTrojan(remark, domain string, port int, password, netType, pathOrServiceName string, isTLS bool) string {
	var query string

	security := "none"
	sni := ""
	if isTLS {
		security = "tls"
		sni = fmt.Sprintf("&sni=%s", domain)
	}

	switch netType {
	case "grpc":
		query = fmt.Sprintf("mode=gun&security=%s&type=grpc&serviceName=%s%s", security, pathOrServiceName, sni)
	case "ws", "httpupgrade":
		query = fmt.Sprintf("path=%s&security=%s&host=%s&type=%s%s", pathOrServiceName, security, domain, netType, sni)
	}

	return fmt.Sprintf("trojan://%s@%s:%d?%s#%s", password, domain, port, query, url.QueryEscape(remark))
}

func main() {
	domain := "tes.kesatu.biz.id"
	uuid := "7da3fe25-8945-47bc-8c6f-ecc28f3f34bf"

	fmt.Println("=== LINK GENERATOR XRAY ===")

	// 1. Contoh Pembuatan Link VMess (WS TLS & Non-TLS)
	vmessWS80, _ := GenerateVMess("VMess-WS-80", domain, 80, uuid, "ws", "/vmess-ws", false)
	vmessWS443, _ := GenerateVMess("VMess-WS-443", domain, 443, uuid, "ws", "/vmess-ws", true)
	fmt.Println("\n[VMESS]")
	fmt.Println(vmessWS80)
	fmt.Println(vmessWS443)

	// 2. Contoh Pembuatan Link VLESS (HTTPUpgrade TLS & Non-TLS)
	vlessUP80 := GenerateVLESS("VLESS-UP-80", domain, 80, uuid, "httpupgrade", "/vless-up", false)
	vlessUP443 := GenerateVLESS("VLESS-UP-443", domain, 443, uuid, "httpupgrade", "/vless-up", true)
	fmt.Println("\n[VLESS]")
	fmt.Println(vlessUP80)
	fmt.Println(vlessUP443)

	// 3. Contoh Pembuatan Link Trojan (gRPC TLS)
	trojanGRPC := GenerateTrojan("Trojan-gRPC-443", domain, 443, uuid, "grpc", "trojan-grpc", true)
	fmt.Println("\n[TROJAN]")
	fmt.Println(trojanGRPC)
}
