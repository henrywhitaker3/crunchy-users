package app

import (
	"github.com/henrywhitaker3/crunchy-users/internal/config"
	"github.com/henrywhitaker3/crunchy-users/internal/k8s"
	"k8s.io/client-go/dynamic"
)

type App struct {
	Version string
	Config  *config.Config

	Client *dynamic.DynamicClient
}

func NewApp(version string) (*App, error) {
	app := &App{Version: version}
	cfg, err := config.New()
	if err != nil {
		return nil, err
	}
	app.Config = cfg

	client, err := k8s.NewClient(cfg.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	app.Client = client

	return app, nil
}
