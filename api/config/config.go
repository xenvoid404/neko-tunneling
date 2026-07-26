package config

import (
	"time"
)

type Config struct {
	ListenAddr    string
	XrayAPIAddr   string
	DBPath        string
	DBMaxConn     int
	DBMaxIdleConn int
	DBMaxIdleTime time.Duration
	DBTimeout     time.Duration
}

func GetConfig() *Config {
	return &Config{
		ListenAddr:    "127.0.0.1:3000",
		XrayAPIAddr:   "127.0.0.1:62789",
		DBPath:        "/var/lib/nekotun/sqlite.db",
		DBMaxConn:     1,
		DBMaxIdleConn: 1,
		DBMaxIdleTime: 5 * time.Minute,
		DBTimeout:     10 * time.Second,
	}
}
