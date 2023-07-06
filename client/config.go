package client

import (
	"os"
	"strconv"
	"time"

	uuid "github.com/nu7hatch/gouuid"
	"gopkg.in/yaml.v2"
)

// Config configures an Proxy
type Config struct {
	ID           string          `yaml:"id"`
	Targets      []*TargetConfig `yaml:"targets"`
	PoolIdleSize int             `yaml:"poolIdleSize"`
	PoolMaxSize  int             `yaml:"poolMaxSize"`
	SecretKey    string          `yaml:"secretKey"`
	Host         string          `yaml:"host"`
	Port         int             `yaml:"port"`
	Timeout      int             `yaml:"timeout"`
}

type TargetConfig struct {
	URL                 string `yaml:"url"`
	ClusterSecretName   string `yaml:"clusterSecretName"`
	ClusterKeyData      string `yaml:"clusterKeyData"`
	ClusterKeyNamespace string `yaml:"clusterKeyNamespace"`
	MaxRetries          int    `yaml:"maxRetries"`
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

	id, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	config.ID = id.String()
	//fmt.Println("Client id is ", config.ID)

	config.Targets = []*TargetConfig{&TargetConfig{URL: "ws://127.0.0.1:8080/register"}}
	//TODO: We need to create separate pool size for write and read conns.
	config.PoolIdleSize = 10
	config.PoolMaxSize = 100

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
