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

			type dailyTaskStats struct {
				task    *core.Task
				seconds int
			}
			todayTasks := make(map[string]*dailyTaskStats)

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
								if _, ok := todayTasks[t.ID]; !ok {
									todayTasks[t.ID] = &dailyTaskStats{task: t, seconds: 0}
								}
								todayTasks[t.ID].seconds += log.Seconds
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
								if _, ok := todayTasks[t.ID]; !ok {
									todayTasks[t.ID] = &dailyTaskStats{task: t, seconds: 0}
								}
								todayTasks[t.ID].seconds += t.TotalSeconds
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

			if len(todayTasks) > 0 {
				fmt.Println("\n==== Today's Breakdown ====")
				for _, stat := range todayTasks {
					adoInfo := ""
					if stat.task.ADOID != nil {
						adoInfo = fmt.Sprintf("[ADO-%d]", *stat.task.ADOID)
					}
					
					typ := stat.task.ADOType
					if typ == "" {
						typ = "Task"
					}
					
					// Format: [20261002.001] [ADO-12345]   (Story)             Fix the firewall rules                   : 2.00 hours
					fmt.Printf("[%s] %-13s (%-17s) %-40s : %.2f hours\n", stat.task.ID, adoInfo, typ, stat.task.Title, float64(stat.seconds)/3600.0)
				}
			}

			return nil
		},
	}
}
