package store

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// InitGit ensures the BaseDir is a git repository.
func (s *Store) InitGit() error {
	gitDir := filepath.Join(s.BaseDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		cmd := exec.Command("git", "init")
		cmd.Dir = s.BaseDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to init git: %w", err)
		}
	}
	return nil
}

// CommitChanges stages all changes in BaseDir and commits them with the given message.
func (s *Store) CommitChanges(message string) error {
	if err := s.InitGit(); err != nil {
		return err
	}

	// Check if there are changes to commit
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = s.BaseDir
	out, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if len(strings.TrimSpace(string(out))) == 0 {
		// No changes to commit
		return nil
	}

	// Add all changes
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = s.BaseDir
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("failed to git add: %w", err)
	}

	// Commit
	commitCmd := exec.Command("git", "commit", "-m", message)
	commitCmd.Dir = s.BaseDir
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("failed to git commit: %w", err)
	}

	return nil
}
