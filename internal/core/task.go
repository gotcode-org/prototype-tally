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

package core

import (
	"time"
)

// TaskState represents the current lifecycle state of a task.
type TaskState string

const (
	StateNew    TaskState = "New"
	StateActive TaskState = "active"
	StatePaused TaskState = "paused"
	StateClosed TaskState = "closed"
)

// Task represents a single unit of work in Tally.
// This struct maps directly to the YAML frontmatter in our Markdown files.
type Task struct {
	// Local Core Metadata
	ID        string    `yaml:"id"`         // e.g., "20260901.001"
	Title     string    `yaml:"title"`      // e.g., "Upgrade Redis Cluster"
	Status    TaskState `yaml:"status"`     // open, active, paused, closed
	Tags      []string  `yaml:"tags,omitempty"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`

	// Azure DevOps Metadata
	ADOID   *int   `yaml:"ado_id,omitempty"`   // Nil if not yet synced to ADO
	ADORev      int      `yaml:"ado_rev,omitempty"`  // ADO Revision for conflict resolution
	ADOType     string   `yaml:"ado_type,omitempty"` // e.g., "Story", "Technical Story", "Bug"
	StoryPoints *float64 `yaml:"story_points,omitempty"` // Mapped to ADO Story Points
	Swimlane    string   `yaml:"swimlane,omitempty"`

	// Recurring Tasks
	Recurrence string `yaml:"recurrence,omitempty"` // e.g., "daily", "weekly", "monthly"
	TemplateID string `yaml:"template_id,omitempty"` // ID of the template this was cloned from
	ParentID   string `yaml:"parent_id,omitempty"`   // Local Tally ID of the parent Story

	// Time Tracking State
	TotalSeconds  int        `yaml:"total_seconds"`          // Total time tracked locally
	SyncedSeconds int        `yaml:"synced_seconds"`         // Time successfully pushed to 7pace
	ActiveTimer   *time.Time `yaml:"active_timer,omitempty"` // When the timer was started (nil if paused)
	TimeLogs      []TimeLog  `yaml:"time_logs,omitempty"`    // Granular time entries

	// The Raw Markdown Body

	// The Raw Markdown Body
	// This is explicitly ignored by the YAML parser so we can handle it manually
	Body string `yaml:"-"` 
}


// TimeLog represents a discrete block of time logged to a task.
type TimeLog struct {
	Timestamp  time.Time `yaml:"timestamp"`
	Seconds    int       `yaml:"seconds"`
	ActivityID string    `yaml:"activity_id,omitempty"`
	Synced     bool      `yaml:"synced"`
}

// UnsyncedSeconds returns the amount of time that hasn't been pushed to 7pace yet.

func (t *Task) UnsyncedSeconds() int {
	return t.TotalSeconds - t.SyncedSeconds
}
