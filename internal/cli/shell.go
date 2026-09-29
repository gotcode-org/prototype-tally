package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/spf13/cobra"
)

func newShellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell",
		Short: "Launch the interactive Tally shell",
		Run: func(cmd *cobra.Command, args []string) {
			runInteractiveShell()
		},
	}
	return cmd
}

func runInteractiveShell() {
	fmt.Println("Welcome to the Tally Interactive Shell.")
	fmt.Println("Type 'help' to see available commands, or 'exit' to quit.")
	fmt.Println("Note: The full-screen TUI cannot be launched from within this shell.")

	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix("tally> "),
		prompt.OptionTitle("Tally Shell"),
		prompt.OptionPrefixTextColor(prompt.Green),
	)

	p.Run()
}

func executor(in string) {
	in = strings.TrimSpace(in)
	if in == "" {
		return
	}

	if in == "exit" || in == "quit" {
		fmt.Println("Goodbye!")
		os.Exit(0)
	}

	// Split input to pass to cobra
	args := strings.Fields(in)

	// Block TUI from shell to avoid breaking terminal state
	if args[0] == "ui" {
		fmt.Println("Error: The TUI cannot be launched from within the shell.")
		fmt.Println("Type 'exit' to leave the shell, then run 'tally ui' instead.")
		return
	}

	// Temporarily override the args of rootCmd
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		// Error is already printed by cobra if we don't silence it
	}
}

func completer(d prompt.Document) []prompt.Suggest {
	var suggestions []prompt.Suggest

	// Dynamically grab all available Tally commands for autocomplete
	for _, cmd := range rootCmd.Commands() {
		// Don't suggest the UI or shell recursively
		if cmd.Use == "ui" || cmd.Use == "shell" {
			continue
		}
		suggestions = append(suggestions, prompt.Suggest{
			Text:        cmd.Use,
			Description: cmd.Short,
		})
	}
	
	suggestions = append(suggestions, prompt.Suggest{Text: "exit", Description: "Exit the shell"})
	suggestions = append(suggestions, prompt.Suggest{Text: "quit", Description: "Exit the shell"})
	suggestions = append(suggestions, prompt.Suggest{Text: "help", Description: "Show help"})

	return prompt.FilterHasPrefix(suggestions, d.GetWordBeforeCursor(), true)
}
