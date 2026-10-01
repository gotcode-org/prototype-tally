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
		Use:   "fetch [id]",
		Short: "Fetch and restore missing tasks from ADO, or a specific task by ID",
		Args:  cobra.MaximumNArgs(1),
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

			var targetID *string
			if len(args) == 1 {
				targetID = &args[0]
				fmt.Printf("Fetching specific task %s from ADO...\n", *targetID)
			} else {
				fmt.Println("Connecting to Azure DevOps WIQL API...")
			}
			
			if _, err := app.Fetch(cfg, adoPat, sevenPaceToken, targetID, nil); err != nil {
				return err
			}
			
			if targetID != nil {
				s.CommitChanges(fmt.Sprintf("tally fetch: Synced task %s with ADO", *targetID))
			} else {
				s.CommitChanges("tally fetch: Synced all active tasks with ADO")
			}
			
			return nil
		},
	}
}
