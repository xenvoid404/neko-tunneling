package config

import (
	"os"
)

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
	XrayAPIAddr     string
	DBPath          string
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
		XrayAPIAddr:     os.Getenv("XRAY_API_ADDR"),
		DBPath:          os.Getenv("DB_PATH"),
	}
}
