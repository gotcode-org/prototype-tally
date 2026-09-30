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
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/core"
	"gotcode.org/tally/internal/store"
)

func newListCmd() *cobra.Command {
	var dateFilter string
	var statusFilter string
	var page int
	var limit int

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List tasks (optionally filtered by date and status)",
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

			var filteredTasks []*core.Task
			for _, t := range tasks {
				// Apply status filtering
				if statusFilter != "" && !strings.EqualFold(statusFilter, "all") {
					if !strings.EqualFold(string(t.Status), statusFilter) {
						continue
					}
				} else if statusFilter == "" {
					// Default: Hide closed/done/resolved tasks
					st := strings.ToLower(string(t.Status))
					if st == "closed" || st == "done" || st == "resolved" || st == "completed" || st == "removed" {
						continue
					}
				}
				filteredTasks = append(filteredTasks, t)
			}
			
			total := len(filteredTasks)
			if total == 0 {
				fmt.Println("No tasks found matching the given filters.")
				return nil
			}

			// Sort by ID descending (chronological, newest first)
			sort.Slice(filteredTasks, func(i, j int) bool {
				return filteredTasks[i].ID > filteredTasks[j].ID
			})

			if limit <= 0 {
				limit = total
			}
			start := (page - 1) * limit
			if start >= total {
				fmt.Printf("Page %d is empty (only %d matching tasks).\n", page, total)
				return nil
			}
			end := start + limit
			if end > total {
				end = total
			}

			paginatedTasks := filteredTasks[start:end]

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "ID\tSTATE\tTYPE\tTIME\tTITLE")

			for _, t := range paginatedTasks {
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
			
			totalPages := (total + limit - 1) / limit
			fmt.Printf("\nPage %d of %d (Showing %d-%d of %d tasks)\n", page, totalPages, start+1, end, total)
			return nil
		},
	}
	
	cmd.Flags().StringVarP(&dateFilter, "date", "d", "", "Filter tasks by date (e.g. 2026, 202609, 20260901)")
	cmd.Flags().StringVarP(&statusFilter, "status", "s", "", "Filter tasks by status (e.g. Closed, active, New, all)")
	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number to display")
	cmd.Flags().IntVarP(&limit, "limit", "l", 40, "Number of tasks per page (set to 0 for unlimited)")
	
	return cmd
}
