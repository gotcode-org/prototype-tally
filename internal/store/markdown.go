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

package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"gotcode.org/tally/internal/core"
)

// Store handles all disk I/O for reading and writing Markdown task files.
type Store struct {
	BaseDir string
}

// NewStore initializes a new Store. If baseDir is empty, it defaults to the XDG data directory.
func NewStore(baseDir string) (*Store, error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not find user home directory: %w", err)
		}
		// Default XDG Data Directory structure
		baseDir = filepath.Join(home, ".local", "share", "tally", "tasks")
	}

	return &Store{BaseDir: baseDir}, nil
}

// getTaskPath calculates the exact YYYY/MM/DD path for a given task ID.
// Assumes ID is formatted like "20260901.001".
func (s *Store) GetTaskPath(id string) string {
	if strings.HasPrefix(id, "recur-") {
		return filepath.Join(s.BaseDir, "recurring", id+".md")
	}
	if strings.HasPrefix(id, "backlog-") {
		return filepath.Join(s.BaseDir, "backlog", id+".md")
	}

	if len(id) < 8 {
		// Fallback if ID is malformed
		return filepath.Join(s.BaseDir, "misc", id+".md")
	}

	year := id[0:4]
	month := id[4:6]
	day := id[6:8]

	return filepath.Join(s.BaseDir, year, month, day, id+".md")
}

// Save writes a core.Task to disk as a Markdown file with YAML frontmatter.
func (s *Store) Save(task *core.Task) error {
	path := s.GetTaskPath(task.ID)

	// Ensure the nested YYYY/MM/DD directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory tree: %w", err)
	}

	// Marshal the struct into YAML frontmatter
	frontmatter, err := yaml.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal yaml frontmatter: %w", err)
	}

	// Stitch it together: --- \n frontmatter \n --- \n body
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(frontmatter)
	buf.WriteString("---\n\n")
	buf.WriteString(strings.TrimSpace(task.Body))
	buf.WriteString("\n")

	// Write atomically (or just standard write for now)
	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write markdown file: %w", err)
	}

	return nil
}

// Load fetches a task strictly by its sequential ID.
func (s *Store) Load(id string) (*core.Task, error) {
	path := s.GetTaskPath(id)
	return s.Parse(path)
}

// Parse extracts a core.Task from a Markdown file reading the YAML frontmatter.
func (s *Store) Parse(path string) (*core.Task, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	task := &core.Task{}

	// Basic split to separate frontmatter from body
	parts := strings.SplitN(string(content), "---", 3)
	if len(parts) >= 3 {
		// Parse the YAML frontmatter
		if err := yaml.Unmarshal([]byte(parts[1]), task); err != nil {
			return nil, fmt.Errorf("failed to unmarshal frontmatter in %s: %w", path, err)
		}
		// The rest is the body
		task.Body = strings.TrimSpace(parts[2])
	} else {
		// No frontmatter found, treat whole file as body
		task.Body = strings.TrimSpace(string(content))
	}

	return task, nil
}

// GetNextID scans the directory for a given date and generates the next sequential ID (e.g. 20260901.003).
func (s *Store) GetNextID(date time.Time, isRecurring bool) (string, error) {
	year := date.Format("2006")
	month := date.Format("01")
	day := date.Format("02")
	
	var prefix string
	var dir string
	
	if isRecurring {
		prefix = fmt.Sprintf("recur-%s%s%s", year, month, day)
		dir = filepath.Join(s.BaseDir, "recurring")
	} else {
		prefix = fmt.Sprintf("%s%s%s", year, month, day)
		dir = filepath.Join(s.BaseDir, year, month, day)
	}
	
	// If the directory doesn't exist, this is the first task of the day
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return prefix + ".001", nil
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	maxSeq := 0
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
			name := strings.TrimSuffix(f.Name(), ".md")
			parts := strings.Split(name, ".")
			if len(parts) == 2 {
				var seq int
				if _, err := fmt.Sscanf(parts[1], "%d", &seq); err == nil {
					if seq > maxSeq {
						maxSeq = seq
					}
				}
			}
		}
	}

	return fmt.Sprintf("%s.%03d", prefix, maxSeq+1), nil
}

// ListTasks recursively finds all Markdown tasks, optionally scoped to a specific sharded directory based on a date prefix.
func (s *Store) ListTasks(datePrefix string) ([]*core.Task, error) {
	var tasks []*core.Task

	targetDir := s.BaseDir
	if len(datePrefix) >= 4 {
		targetDir = filepath.Join(targetDir, datePrefix[0:4])
	}
	if len(datePrefix) >= 6 {
		targetDir = filepath.Join(targetDir, datePrefix[4:6])
	}
	if len(datePrefix) >= 8 {
		targetDir = filepath.Join(targetDir, datePrefix[6:8])
	}

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return tasks, nil // Directory doesn't exist yet, no tasks for this date
	}

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			task, err := s.Parse(path)
			if err != nil {
				return nil
			}
			tasks = append(tasks, task)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk task directory: %w", err)
	}

	return tasks, nil
}


// Delete completely removes the markdown file for a specific task.
func (s *Store) Delete(id string) error {
	path := s.GetTaskPath(id)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete task file %s: %w", path, err)
	}
	return nil
}
