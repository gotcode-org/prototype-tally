package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/core"
	"gotcode.org/tally/internal/store"
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
		
		// Fix terminal raw mode before exit
		restoreCmd := exec.Command("stty", "-raw", "echo")
		restoreCmd.Stdin = os.Stdin
		_ = restoreCmd.Run()
		
		os.Exit(0)
	}

	if in == "clear" || in == "cls" {
		fmt.Print("\033[H\033[2J")
		return
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
	text := d.TextBeforeCursor()
	args := strings.Split(text, " ")

	// If we are typing the first word (the command)
	if len(args) <= 1 {
		var suggestions []prompt.Suggest
		for _, cmd := range rootCmd.Commands() {
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
		suggestions = append(suggestions, prompt.Suggest{Text: "clear", Description: "Clear the screen"})
		suggestions = append(suggestions, prompt.Suggest{Text: "help", Description: "Show help"})

		return prompt.FilterHasPrefix(suggestions, d.GetWordBeforeCursor(), true)
	}

	// If we are past the first word, check context for Task ID suggestions
	command := args[0]
	needsTaskID := false
	switch command {
	case "delete", "edit", "log", "points", "push", "state", "debug-task":
		needsTaskID = true
	}

	if needsTaskID && len(args) == 2 {
		return prompt.FilterHasPrefix(getTaskSuggestions(), d.GetWordBeforeCursor(), true)
	}

	return []prompt.Suggest{}
}

func getTaskSuggestions() []prompt.Suggest {
	var suggestions []prompt.Suggest
	s, err := store.NewStore("")
	if err != nil {
		return suggestions
	}
	app := core.NewApp(s)
	tasks, err := app.ListTasks("")
	if err != nil {
		return suggestions
	}

	for _, t := range tasks {
		// Suggest active tasks to avoid cluttering the autocomplete
		st := strings.ToLower(string(t.Status))
		if st == "closed" || st == "done" || st == "resolved" || st == "completed" || st == "removed" {
			continue
		}
		suggestions = append(suggestions, prompt.Suggest{
			Text:        t.ID,
			Description: t.Title,
		})
	}
	return suggestions
}
