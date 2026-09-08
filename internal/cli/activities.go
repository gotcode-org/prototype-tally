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

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gotcode.org/tally/internal/config"
)

type ActivityItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func newActivitiesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activities",
		Short: "Fetch and display available 7pace Activity Types and their GUIDs",
		RunE: func(cmd *cobra.Command, args []string) error {
			sevenPaceToken := os.Getenv("TALLY_7PACE_TOKEN")
			if sevenPaceToken == "" {
				sevenPaceToken = os.Getenv("TALLY_ADO_PAT") // Fallback
			}
			if sevenPaceToken == "" {
				return fmt.Errorf("FATAL: TALLY_7PACE_TOKEN environment variable is not set")
			}

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if cfg.ADO.Organization == "" {
				return fmt.Errorf("FATAL: ADO Organization is missing in config. Run 'tally config'")
			}

			// Clean up the org name in case the user provided a full URL like https://dev.azure.com/org
			orgName := strings.TrimRight(cfg.ADO.Organization, "/")
			parts := strings.Split(orgName, "/")
			if len(parts) > 0 {
				orgName = parts[len(parts)-1]
			}

			url := fmt.Sprintf("https://%s.timehub.7pace.com/api/rest/activityTypes?api-version=3.1", orgName)
			fmt.Printf("Fetching Activity Types from %s...\n\n", url)

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+sevenPaceToken)
			req.Header.Set("Accept", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("network error hitting 7pace: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("7pace API returned HTTP %d: %s", resp.StatusCode, string(body))
			}

			bodyBytes, _ := io.ReadAll(resp.Body)
			
			var genericPayload map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &genericPayload); err != nil {
				return fmt.Errorf("failed to decode JSON response: %w\nRaw Body: %s", err, string(bodyBytes))
			}

			var items []ActivityItem

			// 7pace API can be wildly inconsistent. We will dynamically search for the array.
			var targetArray []interface{}
			
			if dataObj, ok := genericPayload["data"].(map[string]interface{}); ok {
				if val, ok := dataObj["activityTypes"].([]interface{}); ok {
					targetArray = val
				}
			} else if val, ok := genericPayload["data"].([]interface{}); ok {
				targetArray = val
			} else if val, ok := genericPayload["value"].([]interface{}); ok {
				targetArray = val
			} else if val, ok := genericPayload["items"].([]interface{}); ok {
				targetArray = val
			}

			if targetArray != nil {
				for _, obj := range targetArray {
					if m, ok := obj.(map[string]interface{}); ok {
						id, _ := m["id"].(string)
						name, _ := m["name"].(string)
						if id != "" && name != "" {
							items = append(items, ActivityItem{ID: id, Name: name})
						}
					}
				}
			}

			if len(items) == 0 {
				fmt.Println("No activity types found or unexpected JSON format. Dumping raw JSON payload for inspection:\n")
				fmt.Println(string(bodyBytes))
				return nil
			}

			// Print the table
			fmt.Printf("%-40s | %s\n", "ACTIVITY NAME", "GUID")
			fmt.Println(strings.Repeat("-", 41) + "+" + strings.Repeat("-", 38))
			for _, item := range items {
				fmt.Printf("%-40s | %s\n", item.Name, item.ID)
			}
			fmt.Println()

			return nil
		},
	}
	return cmd
}
