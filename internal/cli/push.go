package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/config"
	"gotcode.org/tally/internal/core"
	"gotcode.org/tally/internal/store"
)

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push [id]",
		Short: "Push all offline tasks, or a single task to Azure DevOps and 7pace",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			adoPat := os.Getenv("TALLY_ADO_PAT")
			if adoPat == "" {
				return fmt.Errorf("FATAL: TALLY_ADO_PAT environment variable is not set")
			}
			sevenPaceToken := os.Getenv("TALLY_7PACE_TOKEN")
			if sevenPaceToken == "" {
				// Fallback to ADO PAT just in case, but 7pace usually requires its own token
				sevenPaceToken = adoPat
			}

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if cfg.ADO.Organization == "" {
				return fmt.Errorf("FATAL: ADO Organization is missing. Run 'tally config'")
			}

			s, err := store.NewStore("")
			if err != nil {
				return err
			}
			app := core.NewApp(s)

			// If an ID is provided, sync just that single task
			if len(args) == 1 {
				targetID := args[0]
				fmt.Printf("Pushing isolated task %s to ADO...\n", targetID)
				_, err = app.SyncSingle(cfg, adoPat, sevenPaceToken, targetID, nil)
				if err == nil {
					s.CommitChanges(fmt.Sprintf("tally push: Pushed task %s to ADO", targetID))
				}
				return err
			}

			// Otherwise, sync all offline tasks
			fmt.Println("Initializing Tally Enterprise Push Engine...")
			if err := app.Sync(cfg, adoPat, sevenPaceToken, nil); err != nil {
				return err
			}

			s.CommitChanges("tally push: Pushed offline changes to ADO and 7pace")
			fmt.Println("\nPush complete.")
			return nil
		},
	}
}
