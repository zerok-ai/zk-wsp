package client

type Config struct {
	ID           string   `yaml:"id"`
	Targets      []string `yaml:"targets"`
	PoolIdleSize int      `yaml:"poolIdleSize"`
	PoolMaxSize  int      `yaml:"poolMaxSize"`
	SecretKey    string   `yaml:"secretKey"`
}
