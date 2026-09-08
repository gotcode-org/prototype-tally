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

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the user's global preferences.
type Config struct {
	User struct {
		Email string `yaml:"email"`
	} `yaml:"user"`

	ADO struct {
		Organization   string `yaml:"organization"`
		DefaultProject string `yaml:"default_project"`
		DefaultArea    string `yaml:"default_area_path"`
		SwimlaneField  string `yaml:"swimlane_field,omitempty"`
		DefaultSwimlane  string `yaml:"default_swimlane,omitempty"`
		Swimlanes        []string `yaml:"swimlanes,omitempty"`
		Debug          bool     `yaml:"debug,omitempty"`
		FetchDays      int      `yaml:"fetch_days,omitempty"`
	} `yaml:"ado"`

	SevenPace struct {
		ActivityID string            `yaml:"default_activity_id"`
		Activities map[string]string `yaml:"activities,omitempty"`
		Email      string            `yaml:"email,omitempty"`
	} `yaml:"7pace"`

	UI struct {
		Theme           string            `yaml:"theme"`
		DashboardStates []string          `yaml:"dashboard_states,omitempty"`
		ArchiveStates   []string          `yaml:"archive_states,omitempty"`
		StateAliases    map[string]string `yaml:"state_aliases,omitempty"`
	} `yaml:"ui"`
}

// GetConfigPath returns the XDG base directory for Tally config.
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "tally", "config.yaml"), nil
}

// Save writes the configuration to disk.
func Save(cfg *Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// Load reads the configuration from disk.
func Load() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil // Return empty if it doesn't exist
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
