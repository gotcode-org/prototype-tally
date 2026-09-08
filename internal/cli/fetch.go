package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/config"
	"gotcode.org/tally/internal/core"
	"gotcode.org/tally/internal/store"
)

func newFetchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch",
		Short: "Fetch and restore missing tasks from ADO",
		RunE: func(cmd *cobra.Command, args []string) error {
			adoPat := os.Getenv("TALLY_ADO_PAT")
			if adoPat == "" {
				return fmt.Errorf("FATAL: TALLY_ADO_PAT environment variable is not set")
			}
			sevenPaceToken := os.Getenv("TALLY_7PACE_TOKEN")
			if sevenPaceToken == "" {
				sevenPaceToken = adoPat
			}
			
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			
			s, err := store.NewStore("")
			if err != nil {
				return err
			}
			app := core.NewApp(s)

			fmt.Println("Connecting to Azure DevOps WIQL API...")
			if _, err := app.Fetch(cfg, adoPat, sevenPaceToken, nil); err != nil {
				return err
			}
			return nil
		},
	}
}
