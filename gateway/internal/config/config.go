package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	PostgresConfig PostgresConfig `yaml:"postgres"`
	ServerConfig   ServerConfig   `yaml:"server"`
}

type PostgresConfig struct {
	Host     string        `yaml:"host"`
	Port     string        `yaml:"port"`
	DBName   string        `yaml:"dbname"`
	User     string        `yaml:"user"`
	Password string        `yaml:"password"`
	Timeout  time.Duration `yaml:"timeout"`
}

type ServerConfig struct {
	Port          string        `yaml:"port"`
	CollabWeight  float64       `yaml:"collab_weight"`
	ContentWeight float64       `yaml:"content_weight"`
	TimeoutAPI    time.Duration `yaml:"timeout_api"`
}

func MustLoad() Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}
	var config Config
	err := cleanenv.ReadConfig(configPath, &config)
	if err != nil {
		log.Fatalf("config not read: %v", err)
	}
	return config
}
