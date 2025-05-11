package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port                   int `yaml:"port"`
	ShutdownTimeoutSeconds int `yaml:"shutdown_timeout_seconds"`
	ChannelBufferSize      int `yaml:"channel_buffer_size"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Port == 0 {
		cfg.Port = 50051
	}
	if cfg.ChannelBufferSize <= 0 {
		cfg.ChannelBufferSize = 100
	}
	if cfg.ShutdownTimeoutSeconds <= 0 {
		cfg.ShutdownTimeoutSeconds = 10
	}
	return &cfg, nil
}

func (c *Config) Timeout() time.Duration {
	return time.Duration(c.ShutdownTimeoutSeconds) * time.Second
}
