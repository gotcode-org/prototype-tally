package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/core"
	"gotcode.org/tally/internal/store"
	"gopkg.in/yaml.v3"
)

func newMigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Deep sweep to repair corrupted legacy backlog IDs",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := store.NewStore("")
			if err != nil {
				return err
			}
			
			fmt.Printf("Initiating deep sweep of %s to repair corrupted frontmatter IDs...\n", s.BaseDir)
			
			repairedCount := 0
			
			err = filepath.Walk(s.BaseDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
					return nil
				}
				
				// The correct ID is simply the filename without .md
				correctID := strings.TrimSuffix(info.Name(), ".md")
				
				// Skip recurring templates as they use unique IDs
				if strings.HasPrefix(correctID, "recur-") {
					return nil
				}
				
				data, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				
				// Parse the YAML frontmatter
				parts := strings.SplitN(string(data), "---", 3)
				if len(parts) < 3 {
					return nil // Not a valid tally markdown file
				}
				
				var task core.Task
				if err := yaml.Unmarshal([]byte(parts[1]), &task); err != nil {
					return nil
				}
				
				// If the frontmatter ID is stuck as a legacy backlog ID
				if strings.HasPrefix(task.ID, "backlog-") {
					// We need to fix it!
					
					// If the file is still in the backlog folder, we need to generate a new ID and move it
					if strings.Contains(path, "/backlog/") {
						dateStr := correctID
						if strings.HasPrefix(dateStr, "backlog-") && len(dateStr) >= 16 {
							dateStr = dateStr[8:16]
						}
						// Strip prefix so it is a standard ID
						task.ID = strings.Replace(task.ID, "backlog-", "", 1)
						task.Status = "Backlog"
						
						s.Save(&task)
						os.Remove(path) // delete the old legacy file
						repairedCount++
						return nil
					}
					
					// The file was already moved to standard YYYY/MM/DD but frontmatter is corrupted
					task.ID = correctID
					task.Status = "Backlog"
					s.Save(&task)
					repairedCount++
				}
				
				return nil
			})
			
			if err != nil {
				return err
			}
			
			fmt.Printf("Deep sweep complete! Successfully repaired %d corrupted legacy tasks.\n", repairedCount)
			return nil
		},
	}
}
