package run

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/henrywhitaker3/crunchy-users/internal/app"
	"github.com/henrywhitaker3/crunchy-users/internal/k8s"
	"github.com/henrywhitaker3/crunchy-users/internal/postgres"
	"github.com/spf13/cobra"
)

func NewCommand(app *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the crunchy postgres user reconcilitation loop",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				<-sigs
				fmt.Println("Received interrupt, stopping...")
				cancel()
			}()

			out, err := k8s.WatchClusters(ctx, app.Client)
			if err != nil {
				return err
			}

			run := true
			for run {
				select {
				case <-ctx.Done():
					run = false
				case res := <-out:
					postgres.HandleCluster(ctx, res)
				}
			}

			return nil
		},
	}
}
