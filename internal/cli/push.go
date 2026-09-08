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
		Use:   "push <id>",
		Short: "Push a single task to Azure DevOps and 7pace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetID := args[0]
			
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
				return err
			}

			s, err := store.NewStore("")
			if err != nil {
				return err
			}
			app := core.NewApp(s)

			fmt.Printf("Pushing isolated task %s to ADO...\n", targetID)
			_, err = app.SyncSingle(cfg, adoPat, sevenPaceToken, targetID, nil)
			return err
		},
	}
}
