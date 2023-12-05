package client

import (
	logsConfig "github.com/zerok-ai/zk-utils-go/logs/config"
	"strconv"
	"time"
)

type WspLoginConfig struct {
	Path                string `json:"path"`
	MaxRetries          int    `json:"maxRetries"`
	Host                string `json:"host"`
	Port                string `json:"port"`
	ClusterSecretName   string `yaml:"clusterSecretName"`
	ClusterKeyData      string `yaml:"clusterKeyData"`
	ClusterKeyNamespace string `yaml:"clusterKeyNamespace"`
}

// Config configures an Proxy
type Config struct {
	Target               *TargetConfig         `yaml:"target"`
	PoolIdleSize         int                   `yaml:"poolIdleSize"`
	PoolMaxSize          int                   `yaml:"poolMaxSize"`
	SecretKey            string                `yaml:"secretKey"`
	Host                 string                `yaml:"host"`
	Port                 int                   `yaml:"port"`
	Timeout              int                   `yaml:"timeout"`
	MaxRetryInterval     int                   `yaml:"maxRetryInterval"`
	DefaultRetryInterval int                   `yaml:"defaultRetryInterval"`
	LogsConfig           logsConfig.LogsConfig `yaml:"logs"`
	WspLogin             WspLoginConfig        `yaml:"wspLogin"`
}

type TargetConfig struct {
	URL                 string `yaml:"url"`
	ClusterSecretName   string `yaml:"clusterSecretName"`
	ClusterKeyData      string `yaml:"clusterKeyData"`
	ClusterKeyNamespace string `yaml:"clusterKeyNamespace"`
	MaxRetries          int    `yaml:"maxRetries"`
	SecretKey           string `yaml:"secretKey"`
}

func (c Config) GetTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Millisecond
}

func (c Config) GetAddr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

// NewConfig creates a new ProxyConfig
func NewConfig() (config *Config) {
	config = new(Config)

	config.Target = &TargetConfig{URL: "ws://127.0.0.1:8080/register", SecretKey: ""}
	//TODO: We need to create separate pool size for write and read conns.
	config.PoolIdleSize = 10
	config.PoolMaxSize = 100

	return
}
