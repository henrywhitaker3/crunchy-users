package config

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	KubeconfigPath string `env:"KUBE_CONFIG_PATH,default=~/.kube/config"`
}

func New() (*Config, error) {
	cfg := &Config{}
	if err := envconfig.Process(context.Background(), cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
