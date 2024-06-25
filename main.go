package main

import (
	"context"
	"fmt"
	"os"

	"github.com/henrywhitaker3/crunchy-users/cmd/root"
	"github.com/henrywhitaker3/crunchy-users/internal/app"
	"github.com/henrywhitaker3/crunchy-users/internal/logger"
	"github.com/joho/godotenv"
)

var (
	version string = "unknown"
)

func main() {
	godotenv.Load()
	app, err := app.NewApp(version)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ctx := logger.Wrap(context.Background())

	root := root.NewRootCommand(app)
	root.SetContext(ctx)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
