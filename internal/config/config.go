package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

var Version = "dev"

type Config struct {
	Server ServerConfig `yaml:"server"`
	MySQL  MySQLConfig  `yaml:"mysql"`
	Auth   AuthConfig   `yaml:"auth"`
}

type ServerConfig struct {
	Address string `yaml:"address"`
	Mode    string `yaml:"mode"`
}

type MySQLConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	Database        string `yaml:"database"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
}

func (m MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		m.User, m.Password, m.Host, m.Port, m.Database,
	)
}

func (m MySQLConfig) ConnMaxLifetimeDuration() time.Duration {
	return time.Duration(m.ConnMaxLifetime) * time.Second
}

type AuthConfig struct {
	TokenHeader string   `yaml:"token_header"`
	ValidTokens []string `yaml:"valid_tokens"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}
	return &cfg, nil
}
