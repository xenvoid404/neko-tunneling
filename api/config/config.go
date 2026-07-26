package config

import (
	"os"
)

type Config struct {
	AppName         string
	AppKey          string
	AppAddr         string
	LogLevel        string
	CacheIPPath     string
	CacheDomainPath string
	CacheISPPath    string
	CacheCityPath   string
	XrayAPIAddr     string
	DBPath          string
}

func GetConfig() *Config {
	return &Config{
		AppName:         os.Getenv("APP_NAME", "Neko Tunneling"),
		AppKey:          os.Getenv("APP_KEY", "f4228ff1859249734dfb0caa5cb5f922"),
		AppAddr:         os.Getenv("APP_ADDR", "127.0.0.1:3000"),
		LogLevel:        os.Getenv("LOG_LEVEL", "info"),
		CacheIPPath:     os.Getenv("CACHE_IP_PATH", "/var/lib/nekotun/cache/ip"),
		CacheDomainPath: os.Getenv("CACHE_DOMAIN_PATH", "/var/lib/nekotun/cache/domain"),
		CacheISPPath:    os.Getenv("CACHE_ISP_PATH", "/var/lib/nekotun/cache/isp"),
		CacheCityPath:   os.Getenv("CACHE_ISP_PATH", "/var/lib/nekotun/cache/city"),
		XrayAPIAddr:     os.Getenv("XRAY_API_ADDR", "127.0.0.1:62789"),
		DBPath:          os.Getenv("DB_PATH", "/var/lib/nekotun/sqlite.db"),
	}
}
