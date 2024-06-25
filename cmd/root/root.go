package root

import (
	"github.com/henrywhitaker3/crunchy-users/cmd/run"
	"github.com/henrywhitaker3/crunchy-users/internal/app"
	"github.com/spf13/cobra"
)

func NewRootCommand(app *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "crunchy-users",
		Short: "Setup crunchy-postgres operator users and databases",
	}

	cmd.AddCommand(run.NewCommand(app))

	return cmd
}
