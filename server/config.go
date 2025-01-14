package server

import (
	logsConfig "github.com/zerok-ai/zk-utils-go/logs/config"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"
)

type ZkCloudConfig struct {
	Host                   string `yaml:"host"`
	Port                   string `yaml:"port"`
	LoginPath              string `yaml:"loginPath"`
	ConnectionSyncInterval int    `yaml:"connectionSyncInterval"`
	ConnectionSyncPath     string `yaml:"connectionSyncPath"`
}

// Config configures an Server
type Config struct {
	Host        string                `yaml:"host"`
	Port        int                   `yaml:"port"`
	Timeout     int                   `yaml:"timeout"`
	IdleTimeout int                   `yaml:"idleTimeout"`
	SecretKey   string                `yaml:"secretKey"`
	PoolMaxSize int                   `yaml:"poolMaxSize"`
	ZkCloud     ZkCloudConfig         `yaml:"zkcloud"`
	LogsConfig  logsConfig.LogsConfig `yaml:"logs"`
}

// GetAddr returns the address to specify a HTTP server address
func (c Config) GetAddr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

// GetTimeout returns the time.Duration converted to millisecond
func (c Config) GetTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}

// NewConfig creates a new ProxyConfig
func NewConfig() (config *Config) {
	config = new(Config)
	config.Host = "127.0.0.1"
	config.Port = 8080
	config.Timeout = 1000 // millisecond
	config.IdleTimeout = 60000
	return
}

// LoadConfiguration loads configuration from a YAML file
func LoadConfiguration(path string) (config *Config, err error) {
	config = NewConfig()

	bytes, err := os.ReadFile(path)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		return
	}

	return
}
