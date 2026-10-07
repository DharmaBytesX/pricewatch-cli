package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/DharmaBytesX/pricewatch-cli/internal/version"
)

func newVersionCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the version of pricewatch",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			_, err := fmt.Fprintf(app.Out, "pricewatch %s\n", version.String())
			return err
		},
	}
}
