package main

import (
	"fmt"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	Server ServerConfig `koanf:"server"`
	Redis  RedisConfig  `koanf:"redis"`
}

type ServerConfig struct {
	Port string `koanf:"port"`
}

type RedisConfig struct {
	ConnectionString string `koanf:"connection_string"`
}

func LoadConfig(configPath string) (*Config, error) {
	k := koanf.New(".")

	// Load YAML config
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}

	var config Config
	if err := k.Unmarshal("", &config); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return &config, nil
}
