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

func newDebugCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "debug [ado_id]",
		Short: "Fetch and dump raw ADO fields for a specific work item ID to find WEF fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			adoPat := os.Getenv("TALLY_ADO_PAT")
			if adoPat == "" {
				return fmt.Errorf("TALLY_ADO_PAT environment variable is required")
			}

			url := fmt.Sprintf("%s/_apis/wit/workitems/%s?api-version=7.1", strings.TrimRight(cfg.ADO.Organization, "/"), args[0])
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}
			req.SetBasicAuth("", adoPat)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			var payload map[string]interface{}
			if err := json.Unmarshal(body, &payload); err != nil {
				return err
			}

			fields, ok := payload["fields"].(map[string]interface{})
			if !ok {
				fmt.Printf("Raw Response:\n%s\n", string(body))
				return nil
			}

			fmt.Println("Available Fields:")
			for k, v := range fields {
				if strings.HasPrefix(k, "WEF_") || strings.Contains(strings.ToLower(k), "lane") {
					fmt.Printf("  ⭐ -> %s = %v\n", k, v)
				} else {
					fmt.Printf("  %s\n", k)
				}
			}

			return nil
		},
	}
}
