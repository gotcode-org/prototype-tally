package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// InitGit ensures the BaseDir is a git repository.
func (s *Store) InitGit() error {
	if err := os.MkdirAll(s.BaseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base dir: %w", err)
	}

	gitDir := filepath.Join(s.BaseDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		_, err := git.PlainInit(s.BaseDir, false)
		if err != nil && err != git.ErrRepositoryAlreadyExists {
			return fmt.Errorf("failed to init go-git: %w", err)
		}
	}
	return nil
}

// CommitChanges stages all changes in BaseDir and commits them with the given message.
func (s *Store) CommitChanges(message string) error {
	if err := s.InitGit(); err != nil {
		return err
	}

	r, err := git.PlainOpen(s.BaseDir)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if status.IsClean() {
		return nil
	}

	if err := w.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return fmt.Errorf("failed to git add: %w", err)
	}

	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Tally",
			Email: "tally@localhost",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to git commit: %w", err)
	}

	return nil
}
