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

// TaskStore defines the interface for our storage engine.
// By declaring this interface here, we invert the dependency and prevent import cycles!
type TaskStore interface {
	Save(task *Task) error
	Load(id string) (*Task, error)
	Parse(path string) (*Task, error)
	GetNextID(date time.Time, isRecurring bool) (string, error)
	ListTasks(datePrefix string) ([]*Task, error)
	GetTaskPath(id string) string
	Delete(id string) error
}

// App acts as the CQRS orchestrator containing all business logic.
type App struct {
	Store TaskStore
}

// NewApp initializes the business logic layer.
func NewApp(s TaskStore) *App {
	return &App{Store: s}
}
