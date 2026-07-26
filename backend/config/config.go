package config

import (
	"net/http/httputil"
	"os"
)

type Listener struct {
	Addr       string
	IsTLS      bool
	SSHBackend string
}

type Route struct {
	Path   string
	Target string
	Proxy  *httputil.ReverseProxy
	IsGRPC bool
}

type Config struct {
	AppName         string
	AppKey          string
	AppAddr         string
	LogLevel        string
	LogFormat       string
	CacheIPPath     string
	CacheDomainPath string
	CacheISPPath    string
	CacheCityPath   string
	DBPath          string
	CertFile        string
	KeyFile         string
	DropbearAddr    string
	OpenSSHAddr     string
	XrayAPIAddr     string
	Routes          []Route
	Listeners       []Listener
}

func GetConfig() *Config {
	return &Config{
		AppName:         os.Getenv("APP_NAME"),
		AppKey:          os.Getenv("APP_KEY"),
		AppAddr:         os.Getenv("APP_ADDR"),
		LogLevel:        os.Getenv("LOG_LEVEL"),
		LogFormat:       os.Getenv("LOG_FORMAT"),
		CacheIPPath:     os.Getenv("CACHE_IP_PATH"),
		CacheDomainPath: os.Getenv("CACHE_DOMAIN_PATH"),
		CacheISPPath:    os.Getenv("CACHE_ISP_PATH"),
		CacheCityPath:   os.Getenv("CACHE_ISP_PATH"),
		DBPath:          os.Getenv("DB_PATH"),
		CertFile:        os.Getenv("CERT_FILE"),
		KeyFile:         os.Getenv("KEY_FILE"),
		DropbearAddr:    os.Getenv("DROPBEAR_ADDR"),
		OpenSSHAddr:     os.Getenv("OPENSSH_ADDR"),
		XrayAPIAddr:     os.Getenv("XRAY_API_ADDR"),
		Listeners: []Listener{
			{Addr: ":80", IsTLS: false, SSHBackend: "dropbear"},
			{Addr: ":8080", IsTLS: false, SSHBackend: "dropbear"},
			{Addr: ":443", IsTLS: true, SSHBackend: "dropbear"},
			{Addr: ":444", IsTLS: true, SSHBackend: "openssh"},
			{Addr: ":8443", IsTLS: true, SSHBackend: "dropbear"},
		},
		Routes: []Route{
			{Path: "/vmess-ws", Target: "127.0.0.1:1054", IsGRPC: false},
			{Path: "/vless-ws", Target: "127.0.0.1:1057", IsGRPC: false},
			{Path: "/trojan-ws", Target: "127.0.0.1:1060", IsGRPC: false},
			{Path: "/vmess-up", Target: "127.0.0.1:1056", IsGRPC: false},
			{Path: "/vless-up", Target: "127.0.0.1:1059", IsGRPC: false},
			{Path: "/trojan-up", Target: "127.0.0.1:1062", IsGRPC: false},
			{Path: "/vmess-grpc", Target: "127.0.0.1:1055", IsGRPC: true},
			{Path: "/vless-grpc", Target: "127.0.0.1:1058", IsGRPC: true},
			{Path: "/trojan-grpc", Target: "127.0.0.1:1061", IsGRPC: true},
			{Path: "/vps", Target: "127.0.0.1:3000", IsGRPC: false},
		},
	}
}
