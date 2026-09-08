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

func newStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state [id] [state]",
		Short: "Change the state of a task (e.g. tally state 20260901.001 Closed)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			state := args[1]

			s, err := store.NewStore("")
			if err != nil {
				return err
			}
			app := core.NewApp(s)

			task, err := app.SetState(id, state)
			if err != nil {
				return err
			}

			fmt.Printf("Task %s is now marked as '%s'\n", task.ID, task.Status)
			return nil
		},
	}
	return cmd
}
