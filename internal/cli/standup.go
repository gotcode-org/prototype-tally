package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/core"
)

import "gotcode.org/tally/internal/store"

func newStandupCmd() *cobra.Command {
	var outFile string

	cmd := &cobra.Command{
		Use:   "standup",
		Short: "Generate a markdown file with Active and Blocked tasks for standup",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := store.NewStore("")
			if err != nil {
				return err
			}
			app := core.NewApp(s)

			tasks, err := app.Store.ListTasks("")
			if err != nil {
				return fmt.Errorf("failed to list tasks: %w", err)
			}

			now := time.Now()
			todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			yesterdayStart := todayStart.AddDate(0, 0, -1)

			var active []*core.Task
			var blocked []*core.Task
			var finished []*core.Task

			for _, t := range tasks {
				status := strings.ToLower(string(t.Status))
				if status == "active" || status == "in progress" || status == "doing" {
					active = append(active, t)
				} else if status == "blocked" || status == "paused" || status == "on hold" || status == "waiting" {
					blocked = append(blocked, t)
				} else if status == "closed" || status == "done" || status == "resolved" || status == "completed" {
					if !t.UpdatedAt.Before(yesterdayStart) {
						finished = append(finished, t)
					}
				}
			}
			
			// Build Markdown
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("# 🌅 Standup - %s\n\n", now.Format("2006-01-02")))

			sb.WriteString("## 🏃 Active\n")
			if len(active) == 0 {
				sb.WriteString("- *No active items*\n")
			} else {
				for _, t := range active {
					adoTag := ""
					if t.ADOID != nil {
						adoTag = fmt.Sprintf("[ADO-%d] ", *t.ADOID)
					}
					
					timeStr := ""
					if t.TotalSeconds > 0 {
						timeStr = fmt.Sprintf(" *(%.1fh tracked)*", float64(t.TotalSeconds)/3600.0)
					}
					
					sb.WriteString(fmt.Sprintf("- **%s%s**%s\n", adoTag, t.Title, timeStr))
				}
			}
			sb.WriteString("\n")

			sb.WriteString("## 🛑 Blocked / Paused\n")
			if len(blocked) == 0 {
				sb.WriteString("- *No blocked items*\n")
			} else {
				for _, t := range blocked {
					adoTag := ""
					if t.ADOID != nil {
						adoTag = fmt.Sprintf("[ADO-%d] ", *t.ADOID)
					}
					sb.WriteString(fmt.Sprintf("- **%s%s**\n", adoTag, t.Title))
				}
			}
			sb.WriteString("\n")

			if outFile == "" {
				reportsDir := ""
				if home, err := os.UserHomeDir(); err == nil {
					reportsDir = filepath.Join(home, ".local", "share", "tally", "reports")
					os.MkdirAll(reportsDir, 0755)
				}
				outFile = filepath.Join(reportsDir, fmt.Sprintf("standup_%s.md", now.Format("2006-01-02")))
			}

			err = os.WriteFile(outFile, []byte(sb.String()), 0644)
			if err != nil {
				return fmt.Errorf("failed to write standup file: %w", err)
			}

			absPath, _ := filepath.Abs(outFile)
			fmt.Printf("✅ Standup report generated successfully!\n📄 Saved to: %s\n\n", absPath)
			fmt.Println(sb.String())

			return nil
		},
	}

	cmd.Flags().StringVarP(&outFile, "out", "o", "", "Output file path (default: standup_YYYY-MM-DD.md)")
	return cmd
}
