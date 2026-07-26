package config

import "os"

type Config struct {
	LogLevel  string
	LogFormat string
	DBPath    string
}

func GetConfig() *Config {
	return &Config{
		LogLevel:  os.Getenv("LOG_LEVEL"),
		LogFormat: os.Getenv("LOG_FORMAT"),
		DBPath:    os.Getenv("DB_PATH"),
	}
}
