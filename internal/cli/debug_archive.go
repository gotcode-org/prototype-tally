package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/config"
	"gotcode.org/tally/internal/store"
)

func newDebugArchiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "debug-archive",
		Short: "Print out all archived tasks to debug UI rendering issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := store.NewStore("")
			if err != nil {
				return err
			}
			tasks, err := s.ListTasks("")
			if err != nil {
				return err
			}
			
			cfg, _ := config.Load()
			archiveStates := []string{"Done", "Closed", "Removed"}
			if cfg != nil && len(cfg.UI.ArchiveStates) > 0 {
				archiveStates = cfg.UI.ArchiveStates
			}
			
			isArchive := func(status string) bool {
				for _, as := range archiveStates {
					if strings.EqualFold(status, as) {
						return true
					}
				}
				return false
			}

			fmt.Println("=== ARCHIVED TASKS IN FILESYSTEM ===")
			count := 0
			for _, t := range tasks {
				if isArchive(string(t.Status)) {
					fmt.Printf("ID: %s\n", t.ID)
					fmt.Printf("  Title:      %s\n", t.Title)
					fmt.Printf("  Status:     %s\n", t.Status)
					fmt.Printf("  ParentID:   '%s'\n", t.ParentID)
					fmt.Printf("  CreatedAt:  %s\n", t.CreatedAt.Format("2006-01-02 (Mon)"))
					fmt.Printf("  UpdatedAt:  %s\n", t.UpdatedAt.Format("2006-01-02 (Mon)"))
					fmt.Println()
					count++
				}
			}
			
			fmt.Printf("Total Archived Tasks Found: %d\n", count)
			return nil
		},
	}
}
