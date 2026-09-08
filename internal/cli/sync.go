/*
Copyright (C) 2026 The GotCode Collective

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/config"
	"gotcode.org/tally/internal/core"
	"gotcode.org/tally/internal/store"
)

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Push all offline tasks and timesheets to Azure DevOps and 7pace",
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

			fmt.Println("Initializing Tally Enterprise Sync Engine...")
			if err := app.Sync(cfg, adoPat, sevenPaceToken, nil); err != nil {
				return err
			}

			fmt.Println("\nSync complete.")
			return nil
		},
	}
	return cmd
}
