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
	"time"

	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/core"
	"gotcode.org/tally/internal/store"
)

func newSummaryCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "summary",
		Aliases: []string{"stats", "time"},
		Short:   "Display a summary of time logged (Day, Week, Month, Year)",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := store.NewStore("")
			if err != nil {
				return err
			}
			app := core.NewApp(s)

			// We need all tasks to calculate historical time
			tasks, err := app.ListTasks("")
			if err != nil {
				return err
			}

			now := time.Now()
			var daySec, weekSec, monthSec, yearSec int

			// Determine the start of the ISO week (Monday)
			offset := int(time.Monday - now.Weekday())
			if offset > 0 {
				offset = -6
			}
			startOfWeek := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, offset)
			endOfWeek := startOfWeek.AddDate(0, 0, 7)

			for _, t := range tasks {
				// Calculate granular time logs
				for _, log := range t.TimeLogs {
					if log.Timestamp.Year() == now.Year() {
						yearSec += log.Seconds
						if log.Timestamp.Month() == now.Month() {
							monthSec += log.Seconds
							if log.Timestamp.Day() == now.Day() {
								daySec += log.Seconds
							}
						}
						if !log.Timestamp.Before(startOfWeek) && log.Timestamp.Before(endOfWeek) {
							weekSec += log.Seconds
						}
					}
				}
				// Also support legacy TotalSeconds if TimeLogs is empty
				if len(t.TimeLogs) == 0 && t.TotalSeconds > 0 {
					if t.CreatedAt.Year() == now.Year() {
						yearSec += t.TotalSeconds
						if t.CreatedAt.Month() == now.Month() {
							monthSec += t.TotalSeconds
							if t.CreatedAt.Day() == now.Day() {
								daySec += t.TotalSeconds
							}
						}
						if !t.CreatedAt.Before(startOfWeek) && t.CreatedAt.Before(endOfWeek) {
							weekSec += t.TotalSeconds
						}
					}
				}
			}

			fmt.Println("==== Tally Time Summary ====")
			fmt.Printf("Today:      %.2f hours\n", float64(daySec)/3600.0)
			fmt.Printf("This Week:  %.2f hours\n", float64(weekSec)/3600.0)
			fmt.Printf("This Month: %.2f hours\n", float64(monthSec)/3600.0)
			fmt.Printf("This Year:  %.2f hours\n", float64(yearSec)/3600.0)

			return nil
		},
	}
}
