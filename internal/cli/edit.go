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
	"os/exec"

	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/store"
)

func newEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit [id]",
		Short: "Open a task in your $EDITOR",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			s, err := store.NewStore("")
			if err != nil {
				return err
			}

			path := s.GetTaskPath(id)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("task %s does not exist at %s", id, path)
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vim" // Fallback to vim
			}

			fmt.Printf("Opening %s in %s...\n", id, editor)
			
			c := exec.Command(editor, path)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr

			return c.Run()
		},
	}
	return cmd
}
