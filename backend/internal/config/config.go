package config

import (
	"os"
)

type Config struct {
	ServerPort   string
	DatabasePath string
}

func Load() *Config {
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = ":8080"
	}

	databasePath := os.Getenv("DATABASE_PATH")
	if databasePath == "" {
		databasePath = "./balance_monitor.db"
	}

	return &Config{
		ServerPort:   serverPort,
		DatabasePath: databasePath,
	}
}

