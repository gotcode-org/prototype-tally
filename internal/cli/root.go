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
)

// Execute boots the Cobra CLI framework.
func Execute() {
	rootCmd := &cobra.Command{
		Use:   "tally",
		Short: "Tally is a blazing fast time tracker and task manager",
	}

	rootCmd.AddCommand(newAddCmd())
	rootCmd.AddCommand(newDeleteCmd())
	rootCmd.AddCommand(newActivitiesCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newLogCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newSyncCmd())
	rootCmd.AddCommand(newFetchCmd())
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newPushCmd())
	rootCmd.AddCommand(newDebugTaskCmd())
	rootCmd.AddCommand(newEditCmd())
	rootCmd.AddCommand(newStateCmd())
	rootCmd.AddCommand(newPointsCmd())
	rootCmd.AddCommand(newUICmd())
	rootCmd.AddCommand(newDebugCmd())
	rootCmd.AddCommand(newMigrateCmd())
	rootCmd.AddCommand(newDebugArchiveCmd())
	rootCmd.AddCommand(newStandupCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
