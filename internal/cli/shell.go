package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/kballard/go-shellquote"
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

// sshParser wraps the standard posix parser to fix ANSI escape fragmentation and chunking over SSH.
type sshParser struct {
	prompt.ConsoleParser
	buf []byte
}

func (p *sshParser) Read() ([]byte, error) {
	if len(p.buf) == 0 {
		b, err := p.ConsoleParser.Read()
		if err != nil || len(b) == 0 {
			return b, err
		}

		// Anti-fragmentation: wait a tiny bit if it looks truncated
		if b[len(b)-1] == 27 || (len(b) >= 2 && b[len(b)-2] == 27 && b[len(b)-1] == '[') || (len(b) >= 2 && b[len(b)-2] == 27 && b[len(b)-1] == 'O') {
			time.Sleep(15 * time.Millisecond)
			b2, err2 := p.ConsoleParser.Read()
			if err2 == nil && len(b2) > 0 {
				b = append(b, b2...)
			}
		}
		p.buf = b
	}

	// Now we have bytes in p.buf. We must return EXACTLY ONE sequence or string of regular text.
	if p.buf[0] != 27 {
		idx := bytes.IndexByte(p.buf, 27)
		if idx == -1 {
			res := p.buf
			p.buf = nil
			return res, nil
		}
		res := p.buf[:idx]
		p.buf = p.buf[idx:]
		return res, nil
	}

	// It's an escape sequence starting with \x1b.
	if len(p.buf) == 1 {
		res := p.buf
		p.buf = nil
		return res, nil
	}

	endIdx := 1
	if p.buf[1] == '[' {
		// CSI sequence: \x1b [ ... <char 0x40-0x7E>
		endIdx = 2
		for endIdx < len(p.buf) {
			if p.buf[endIdx] >= 0x40 && p.buf[endIdx] <= 0x7E {
				endIdx++
				break
			}
			endIdx++
		}
	} else if p.buf[1] == 'O' {
		// SS3 sequence: \x1b O <char>
		if len(p.buf) >= 3 {
			endIdx = 3
		} else {
			endIdx = len(p.buf)
		}
	} else {
		// Alt+Key (e.g. \x1b b)
		endIdx = 2
	}

	res := p.buf[:endIdx]
	p.buf = p.buf[endIdx:]
	return res, nil
}

func runInteractiveShell() {
	fmt.Println("Welcome to the Tally Interactive Shell.")
	fmt.Println("Type 'help' to see available commands, or 'exit' to quit.")
	fmt.Println("Note: The full-screen TUI cannot be launched from within this shell.")

	parser := prompt.NewStandardInputParser()

	p := prompt.New(
		executor,
		completer,
		prompt.OptionParser(&sshParser{ConsoleParser: parser}),
		prompt.OptionPrefix("tally> "),
		prompt.OptionTitle("Tally Shell"),
		prompt.OptionPrefixTextColor(prompt.Green),
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.Home,
			Fn: func(buf *prompt.Buffer) {
				x := []rune(buf.Document().TextBeforeCursor())
				buf.CursorLeft(len(x))
			},
		}),
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.End,
			Fn: func(buf *prompt.Buffer) {
				x := []rune(buf.Document().TextAfterCursor())
				buf.CursorRight(len(x))
			},
		}),
		// Catch raw terminal byte sequences for Home
		prompt.OptionAddASCIICodeBind(
			prompt.ASCIICodeBind{ASCIICode: []byte{27, 91, 72}, Fn: goStart},        // \x1b[H
			prompt.ASCIICodeBind{ASCIICode: []byte{27, 91, 49, 126}, Fn: goStart},   // \x1b[1~
			prompt.ASCIICodeBind{ASCIICode: []byte{27, 79, 72}, Fn: goStart},        // \x1bOH
		),
		// Catch raw terminal byte sequences for End
		prompt.OptionAddASCIICodeBind(
			prompt.ASCIICodeBind{ASCIICode: []byte{27, 91, 70}, Fn: goEnd},          // \x1b[F
			prompt.ASCIICodeBind{ASCIICode: []byte{27, 91, 52, 126}, Fn: goEnd},     // \x1b[4~
			prompt.ASCIICodeBind{ASCIICode: []byte{27, 79, 70}, Fn: goEnd},          // \x1bOF
		),
	)

	p.Run()
}

func goStart(buf *prompt.Buffer) {
	x := []rune(buf.Document().TextBeforeCursor())
	buf.CursorLeft(len(x))
}

func goEnd(buf *prompt.Buffer) {
	x := []rune(buf.Document().TextAfterCursor())
	buf.CursorRight(len(x))
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

	if in == "history" {
		s, err := store.NewStore("")
		if err == nil {
			s.InitGit()
			cmd := exec.Command("git", "log", "--oneline", "--decorate", "--color=always", "-n", "20")
			cmd.Dir = s.BaseDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Run()
		}
		return
	}

	// Split input to pass to cobra, respecting shell quotes!
	args, err := shellquote.Split(in)
	if err != nil {
		fmt.Printf("Error parsing input: %v\n", err)
		return
	}

	if len(args) == 0 {
		return
	}
	
	command := args[0]
	
	if command == "diff" {
		if len(args) < 2 {
			fmt.Println("Usage: diff <TaskID>")
			return
		}
		s, err := store.NewStore("")
		if err == nil {
			s.InitGit()
			path := s.GetTaskPath(args[1])
			// Show the diff history of this specific file
			cmd := exec.Command("git", "log", "-p", "--color=always", "-n", "3", "--", path)
			cmd.Dir = s.BaseDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Run()
		}
		return
	}

	// Intercept the UI command and spawn a sandbox child process
	if command == "ui" {
		cmd := exec.Command(os.Args[0], "ui")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		_ = cmd.Run()
		
		// Clear the screen so the shell prompt redraws cleanly
		fmt.Print("\033[H\033[2J")
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
			if cmd.Use == "shell" {
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
		suggestions = append(suggestions, prompt.Suggest{Text: "history", Description: "View git change history"})
		suggestions = append(suggestions, prompt.Suggest{Text: "diff", Description: "View diff history of a specific task"})
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
			{Text: "Story"}, {Text: `"Technical Story"`}, {Text: "Bug"}, {Text: "Task"},
		}
		return prompt.FilterHasPrefix(types, lastArg, true)
	}

	if prevArg == "--swimlane" {
		var lanes []prompt.Suggest
		if cfg, err := config.Load(); err == nil {
			for _, s := range cfg.ADO.Swimlanes {
				if strings.Contains(s, " ") {
					s = fmt.Sprintf(`"%s"`, s)
				}
				lanes = append(lanes, prompt.Suggest{Text: s})
			}
		}
		return prompt.FilterHasPrefix(lanes, lastArg, true)
	}

	if prevArg == "--activity" {
		var acts []prompt.Suggest
		if cfg, err := config.Load(); err == nil {
			for name := range cfg.SevenPace.Activities {
				if strings.Contains(name, " ") {
					name = fmt.Sprintf(`"%s"`, name)
				}
				acts = append(acts, prompt.Suggest{Text: name})
			}
		}
		return prompt.FilterHasPrefix(acts, lastArg, true)
	}

	// 2. Flag Name Autocompletion (for 'add' and 'log' command)
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

	if command == "log" && strings.HasPrefix(lastArg, "-") {
		flags := []prompt.Suggest{
			{Text: "--activity", Description: "The friendly name of the activity type (Required)"},
			{Text: "--date", Description: "Date to log the time for (YYYY-MM-DD)"},
		}
		return prompt.FilterHasPrefix(flags, lastArg, true)
	}

	// 3. Task ID Autocompletion
	needsTaskID := false
	switch command {
	case "delete", "edit", "log", "points", "push", "state", "debug-task", "diff":
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
