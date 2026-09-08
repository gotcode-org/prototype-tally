package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/store"
	"gopkg.in/yaml.v3"
)

func newDebugTaskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "debug-task <id>",
		Short: "Dump the raw yaml of a specific task to stdout for troubleshooting",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := store.NewStore("")
			if err != nil {
				return err
			}
			t, err := s.Load(args[0])
			if err != nil {
				return err
			}
			out, _ := yaml.Marshal(t)
			fmt.Println(string(out))
			return nil
		},
	}
}
