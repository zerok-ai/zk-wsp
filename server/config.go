package server

import (
	"strconv"
	"time"
)

type Config struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Timeout     int    `yaml:"timeout"`
	IdleTimeout int    `yaml:"idle_timeout"`
}

func (c *Config) GetAddr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

func (c *Config) GetTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Millisecond
}
