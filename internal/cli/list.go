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
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/core"
	"gotcode.org/tally/internal/store"
)

func newListCmd() *cobra.Command {
	var dateFilter string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List tasks (optionally filtered by date)",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := store.NewStore("")
			if err != nil {
				return err
			}
			app := core.NewApp(s)

			tasks, err := app.ListTasks(dateFilter)
			if err != nil {
				return err
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "ID	STATE	TYPE	TIME	TITLE")

			for _, t := range tasks {
				dur := time.Duration(t.TotalSeconds) * time.Second
				timeStr := dur.String()
				if t.TotalSeconds == 0 {
					timeStr = "0m"
				}

				adoType := t.ADOType
				if adoType == "" {
					adoType = "-"
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", t.ID, t.Status, adoType, timeStr, t.Title)
			}
			
			w.Flush()
			return nil
		},
	}
	
	cmd.Flags().StringVarP(&dateFilter, "date", "d", "", "Filter tasks by date (e.g. 2026, 202609, 20260901)")
	
	return cmd
}
