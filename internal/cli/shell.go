package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/config"
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

	// If we are past the first word, check context
	command := args[0]
	lastArg := args[len(args)-1]
	prevArg := ""
	if len(args) > 1 {
		prevArg = args[len(args)-2]
	}

	// 1. Flag Value Autocompletion
	if prevArg == "--type" || prevArg == "-t" {
		types := []prompt.Suggest{
			{Text: "Story"}, {Text: "Technical Story"}, {Text: "Bug"}, {Text: "Task"},
		}
		return prompt.FilterHasPrefix(types, lastArg, true)
	}

	if prevArg == "--swimlane" {
		var lanes []prompt.Suggest
		if cfg, err := config.Load(); err == nil {
			for _, s := range cfg.ADO.Swimlanes {
				lanes = append(lanes, prompt.Suggest{Text: s})
			}
		}
		return prompt.FilterHasPrefix(lanes, lastArg, true)
	}

	// 2. Flag Name Autocompletion (for 'add' command)
	if command == "add" && strings.HasPrefix(lastArg, "-") {
		flags := []prompt.Suggest{
			{Text: "--name", Description: "The title or name of the new task (Required)"},
			{Text: "--type", Description: "ADO Work Item Type (e.g., Story, Bug)"},
			{Text: "--swimlane", Description: "The swimlane to put the task in"},
			{Text: "--tags", Description: "Comma-separated list of tags"},
			{Text: "--recur", Description: "Set a recurrence rule"},
			{Text: "--parent", Description: "The Tally ID of the parent Story"},
		}
		return prompt.FilterHasPrefix(flags, lastArg, true)
	}

	// 3. Task ID Autocompletion
	needsTaskID := false
	switch command {
	case "delete", "edit", "log", "points", "push", "state", "debug-task":
		needsTaskID = true
	}

	// Only suggest task IDs for the exact second argument, and if it's not starting a flag
	if needsTaskID && len(args) == 2 && !strings.HasPrefix(lastArg, "-") {
		return prompt.FilterHasPrefix(getTaskSuggestions(), lastArg, true)
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
