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
	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/core"
	"gotcode.org/tally/internal/store"
)

func newAddCmd() *cobra.Command {
	var adoType string
	var tags []string
	var recurrence string
	var parentID string
	var swimlane string

	cmd := &cobra.Command{
		Use:   "add [title]",
		Short: "Create a new task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize storage (Presentation layer doing setup)
			s, err := store.NewStore("")
			if err != nil {
				return err
			}
			app := core.NewApp(s)

			// Execute business logic
			task, err := app.AddTask(args[0], adoType, tags, recurrence, parentID, swimlane)
			if err != nil {
				return err
			}

			// Format presentation output
			fmt.Printf("Created task %s: %s\n", task.ID, task.Title)
			return nil
		},
	}

	cmd.Flags().StringVar(&adoType, "type", "", "ADO Work Item Type (e.g., Story, Bug)")
	cmd.Flags().StringSliceVar(&tags, "tags", []string{}, "Comma-separated list of tags")
	cmd.Flags().StringVar(&recurrence, "recur", "", "Set a recurrence rule (e.g., daily)")
	cmd.Flags().StringVar(&parentID, "parent", "", "The Tally ID of the parent Story")
	cmd.Flags().StringVar(&swimlane, "swimlane", "", "The swimlane to put the task in")

	return cmd
}
