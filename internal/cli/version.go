package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the application version and dependencies",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.GetVersionString())
			fmt.Println()
			fmt.Println(version.GetDepsString())
		},
	}
}
