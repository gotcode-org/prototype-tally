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
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Interactively build your Tally configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("=== Tally Configuration Wizard ===")
			
			cfg, err := config.Load()
			if err != nil {
				cfg = &config.Config{}
			}

			reader := bufio.NewReader(os.Stdin)

			ask := func(prompt, current string) string {
				if current != "" {
					fmt.Printf("%s [%s]: ", prompt, current)
				} else {
					fmt.Printf("%s: ", prompt)
				}
				
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)
				
				if input == "" {
					return current
				}
				return input
			}

			cfg.User.Email = ask("Email Address", cfg.User.Email)
			cfg.ADO.Organization = ask("Azure DevOps Organization URL (e.g. https://dev.azure.com/gotcode)", cfg.ADO.Organization)
			cfg.ADO.DefaultProject = ask("Default ADO Project", cfg.ADO.DefaultProject)
			cfg.ADO.DefaultArea = ask("Default Area Path", cfg.ADO.DefaultArea)
			cfg.SevenPace.ActivityID = ask("7pace Default Activity ID", cfg.SevenPace.ActivityID)

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Println("\nConfiguration successfully saved to ~/.config/tally/config.yaml")
			fmt.Println("\nTo authenticate with Azure DevOps, please export your Personal Access Token (PAT) as an environment variable:")
			fmt.Println("  export TALLY_ADO_PAT=\"your_token_here\"")
			
			return nil
		},
	}
	return cmd
}
