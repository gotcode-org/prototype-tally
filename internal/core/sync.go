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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"strconv"
	"time"

	"gotcode.org/tally/internal/config"
	md "github.com/JohannesKaufmann/html-to-markdown"
)

func logf(logChan chan<- string, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if logChan != nil {
		// Non-blocking send or simple block
		logChan <- msg
	} else {
		fmt.Print(msg)
	}
}

type ADOResponse struct {
	ID int `json:"id"`
}

// Sync executes the network operations to push local data to ADO and 7pace.
func (a *App) Sync(cfg *config.Config, adoPat string, sevenPaceToken string, logChan chan<- string) error {
	tasks, err := a.Store.ListTasks("")
	if err != nil {
		return fmt.Errorf("failed to list tasks for sync: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	orgName := extractOrgName(cfg.ADO.Organization)

	for _, t := range tasks {
		// Never sync raw templates, only their instantiated clones!
		if strings.HasPrefix(t.ID, "recur-") {
			continue
		}

		if t.ADOID == nil {
			logf(logChan, "Syncing task %s to ADO...\n", t.ID)
			
			adoType := t.ADOType
			if adoType == "" {
				adoType = "Task"
			}

			patch := []map[string]interface{}{
				{"op": "add", "path": "/fields/System.Title", "value": t.Title},
				{"op": "add", "path": "/fields/System.AreaPath", "value": cfg.ADO.DefaultArea},
			}
			
			if cfg.User.Email != "" {
				patch = append(patch, map[string]interface{}{
					"op": "add", "path": "/fields/System.AssignedTo", "value": cfg.User.Email,
				})
			}
			
			if t.Body != "" {
				desc, ac := parseMarkdownSections(t.Body)
				
				if desc != "" {
					patch = append(patch, map[string]interface{}{
						"op": "add", "path": "/fields/System.Description", "value": desc,
					})
				}
				if ac != "" {
					patch = append(patch, map[string]interface{}{
						"op": "add", "path": "/fields/Microsoft.VSTS.Common.AcceptanceCriteria", "value": ac,
					})
				}
			}
			
			if len(t.Tags) > 0 {
				patch = append(patch, map[string]interface{}{
					"op": "add", "path": "/fields/System.Tags", "value": strings.Join(t.Tags, "; "),
				})
			}
			
			if t.Swimlane != "" && cfg.ADO.SwimlaneField != "" {
				patch = append(patch, map[string]interface{}{
					"op": "add", "path": "/fields/" + cfg.ADO.SwimlaneField, "value": t.Swimlane,
				})
			}
			
			if t.StoryPoints != nil {
				patch = append(patch, map[string]interface{}{
					"op": "add", "path": "/fields/Microsoft.VSTS.Scheduling.StoryPoints", "value": *t.StoryPoints,
				})
			}
			
			// Handle ADO Hierarchy Linking
			if t.ParentID != "" {
				parentTask, err := a.Store.Load(t.ParentID)
				if err == nil && parentTask.ADOID != nil {
					patch = append(patch, map[string]interface{}{
						"op": "add",
						"path": "/relations/-",
						"value": map[string]interface{}{
							"rel": "System.LinkTypes.Hierarchy-Reverse", // I am a child of this parent
							"url": fmt.Sprintf("%s/_apis/wit/workitems/%d", strings.TrimRight(cfg.ADO.Organization, "/"), *parentTask.ADOID),
							"attributes": map[string]interface{}{
								"comment": "Linked via Tally",
							},
						},
					})
				}
			}

			payload, _ := json.Marshal(patch)
			url := fmt.Sprintf("%s/%s/_apis/wit/workitems/$%s?api-version=7.0", strings.TrimRight(cfg.ADO.Organization, "/"), cfg.ADO.DefaultProject, strings.ReplaceAll(adoType, " ", "%20"))
			
			req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json-patch+json")
			req.SetBasicAuth("", adoPat)

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("network error hitting ADO: %w", err)
			}
			
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				var adoResp ADOResponse
				body, _ := io.ReadAll(resp.Body)
				json.Unmarshal(body, &adoResp)
				resp.Body.Close()
				
				t.ADOID = &adoResp.ID
				t.UpdatedAt = time.Now()
				a.Store.Save(t)
				logf(logChan, "  -> Successfully created ADO Work Item #%d\n", *t.ADOID)
			} else if resp.StatusCode == 400 && cfg.User.Email != "" {
				resp.Body.Close()
				logf(logChan, "  -> ADO rejected the AssignedTo identity. Retrying without assignment...\n")
				
				fallbackPatch := []map[string]interface{}{}
				for _, p := range patch {
					if p["path"] != "/fields/System.AssignedTo" {
						fallbackPatch = append(fallbackPatch, p)
					}
				}
				
				payload, _ = json.Marshal(fallbackPatch)
				req, _ = http.NewRequest("POST", url, bytes.NewBuffer(payload))
				req.Header.Set("Content-Type", "application/json-patch+json")
				req.SetBasicAuth("", adoPat)
				
				resp2, err := client.Do(req)
				if err == nil && resp2.StatusCode >= 200 && resp2.StatusCode < 300 {
					var adoResp ADOResponse
					body, _ := io.ReadAll(resp2.Body)
					json.Unmarshal(body, &adoResp)
					resp2.Body.Close()
					
					t.ADOID = &adoResp.ID
					t.UpdatedAt = time.Now()
					a.Store.Save(t)
					logf(logChan, "  -> Successfully created ADO Work Item #%d (Unassigned)\n", *t.ADOID)
				} else {
					if err == nil {
						body, _ := io.ReadAll(resp2.Body)
						resp2.Body.Close()
						logf(logChan, "  -> Fallback also failed (HTTP %d). Response: %s\n", resp2.StatusCode, string(body))
					}
				}
			} else {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				logf(logChan, "  -> Failed to create ADO item (HTTP %d). Response: %s\n", resp.StatusCode, string(body))
				continue
			}
		}

		// 1.5 Update State if Closed
		if t.ADOID != nil {
			patch := []map[string]interface{}{}
			
			// Always sync title
			patch = append(patch, map[string]interface{}{
				"op": "add", "path": "/fields/System.Title", "value": t.Title,
			})
			
			// Sync state if specified and not generic defaults
			if t.Status != "open" && t.Status != "active" {
				patch = append(patch, map[string]interface{}{
					"op": "add", "path": "/fields/System.State", "value": t.Status,
				})
			}
			
			if len(t.Tags) > 0 {
				patch = append(patch, map[string]interface{}{
					"op": "add", "path": "/fields/System.Tags", "value": strings.Join(t.Tags, "; "),
				})
			}
			
			if t.Swimlane != "" && cfg.ADO.SwimlaneField != "" {
				patch = append(patch, map[string]interface{}{
					"op": "add", "path": "/fields/" + cfg.ADO.SwimlaneField, "value": t.Swimlane,
				})
			}
			
			if t.StoryPoints != nil {
				patch = append(patch, map[string]interface{}{
					"op": "add", "path": "/fields/Microsoft.VSTS.Scheduling.StoryPoints", "value": *t.StoryPoints,
				})
			}
			
			// Handle ADO Hierarchy Linking
			if t.ParentID != "" {
				parentTask, err := a.Store.Load(t.ParentID)
				if err == nil && parentTask.ADOID != nil {
					patch = append(patch, map[string]interface{}{
						"op": "add",
						"path": "/relations/-",
						"value": map[string]interface{}{
							"rel": "System.LinkTypes.Hierarchy-Reverse", // I am a child of this parent
							"url": fmt.Sprintf("%s/_apis/wit/workitems/%d", strings.TrimRight(cfg.ADO.Organization, "/"), *parentTask.ADOID),
							"attributes": map[string]interface{}{
								"comment": "Linked via Tally",
							},
						},
					})
				}
			}
			
			if t.Body != "" {
				desc, ac := parseMarkdownSections(t.Body)
				if desc != "" {
					patch = append(patch, map[string]interface{}{
						"op": "add", "path": "/fields/System.Description", "value": desc,
					})
				}
				if ac != "" {
					patch = append(patch, map[string]interface{}{
						"op": "add", "path": "/fields/Microsoft.VSTS.Common.AcceptanceCriteria", "value": ac,
					})
				}
			}
			
			if len(patch) > 0 {
				payload, _ := json.Marshal(patch)
				url := fmt.Sprintf("%s/%s/_apis/wit/workitems/%d?api-version=7.0", strings.TrimRight(cfg.ADO.Organization, "/"), cfg.ADO.DefaultProject, *t.ADOID)
				
				req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(payload))
				req.Header.Set("Content-Type", "application/json-patch+json")
				req.SetBasicAuth("", adoPat)
				
				resp, err := client.Do(req)
				if err == nil {
					body, _ := io.ReadAll(resp.Body)
					if resp.StatusCode >= 400 {
						logf(logChan, "  -> ADO rejected update for task %s (HTTP %d). Response: %s\n", t.ID, resp.StatusCode, string(body))
					} else {
						var updateResp struct {
							Rev int `json:"rev"`
						}
						if err := json.Unmarshal(body, &updateResp); err == nil && updateResp.Rev > 0 {
							t.ADORev = updateResp.Rev
							a.Store.Save(t) // Force save immediately so we don't rely on TimeLog changes
						}
						logf(logChan, "  -> Successfully updated ADO Work Item #%d\n", *t.ADOID)
						// Dump the payload we sent and the ADO response if debugging is enabled
						if cfg.ADO.Debug {
							logf(logChan, "     [DEBUG] Sent Payload: %s\n", string(payload))
							respStr := string(body)
							if len(respStr) > 500 {
								respStr = respStr[:500] + "..."
							}
							logf(logChan, "     [DEBUG] ADO Response: %s\n", respStr)
						}
					}
					resp.Body.Close()
				}
			}
		}

		hasChanges := false
		for i := range t.TimeLogs {
			logEntry := &t.TimeLogs[i]
			if logEntry.Synced || t.ADOID == nil {
				continue
			}

			logf(logChan, "Pushing %d seconds of time to 7pace for ADO #%d...\n\n\n", logEntry.Seconds, *t.ADOID)
			
			logData := map[string]interface{}{
				"timestamp":  logEntry.Timestamp.Format(time.RFC3339),
				"length":     logEntry.Seconds,
				"workItemId": *t.ADOID,
				"comment":    "Logged via Tally terminal",
			}
			
			// Use the specific activity ID if set, otherwise fallback to default
			actID := logEntry.ActivityID
			if actID == "" {
				actID = cfg.SevenPace.ActivityID
			}
			
			if len(actID) >= 32 {
				logData["activityTypeId"] = actID
			}
			
			payload, _ := json.Marshal(logData)
			url := fmt.Sprintf("https://%s.timehub.7pace.com/api/rest/workLogs?api-version=3.1", orgName)
			
			req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+sevenPaceToken)

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("network error hitting 7pace: %w", err)
			}
			
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				logEntry.Synced = true
				hasChanges = true
				logf(logChan, "  -> Time successfully logged to 7pace!\n")
			} else {
				body, _ := io.ReadAll(resp.Body)
				logf(logChan, "  -> Failed to log time to 7pace (HTTP %d). Response: %s\n", resp.StatusCode, string(body))
			}
			resp.Body.Close()
		}
		
		if hasChanges {
			// Recalculate SyncedSeconds based on successful discrete logs
			synced := 0
			for _, l := range t.TimeLogs {
				if l.Synced {
					synced += l.Seconds
				}
			}
			t.SyncedSeconds = synced
			a.Store.Save(t)
		}
	}

	return nil
}

func extractOrgName(adoURL string) string {
	url := strings.TrimRight(adoURL, "/")
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// parseMarkdownSections splits the body into Description and Acceptance Criteria
func parseMarkdownSections(body string) (description, acceptanceCriteria string) {
	lines := strings.Split(body, "\n")
	
	var currentSection string = "description"
	var descBuilder strings.Builder
	var acBuilder strings.Builder
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		
		// Skip headers
		if strings.HasPrefix(lower, "# description") {
			currentSection = "description"
			continue
		} else if strings.HasPrefix(lower, "# acceptance criteria") {
			currentSection = "ac"
			continue
		}
		
		// ADO WYSIWYG requires proper block elements, bare text with <br> is often swallowed
		htmlLine := line
		if htmlLine == "" {
			htmlLine = "<br>"
		}
		
		if currentSection == "description" {
			descBuilder.WriteString(fmt.Sprintf("<div>%s</div>", htmlLine))
		} else if currentSection == "ac" {
			acBuilder.WriteString(fmt.Sprintf("<div>%s</div>", htmlLine))
		}
	}
	
	return strings.TrimSpace(descBuilder.String()), strings.TrimSpace(acBuilder.String())
}

// Fetch queries ADO for all work items assigned to the current user, and restores any missing local markdown files.
func (a *App) Fetch(cfg *config.Config, adoPat string, sevenPaceToken string, logChan chan<- string) ([]*Task, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	converter := md.NewConverter("", true, nil)
	
	fetchDays := cfg.ADO.FetchDays
	if fetchDays <= 0 {
		fetchDays = 30
	}
	query := fmt.Sprintf(`{"query": "Select [System.Id], [System.Title], [System.State], [System.WorkItemType] From WorkItems Where [System.AssignedTo] = @Me AND [System.ChangedDate] >= @Today - %d"}`, fetchDays)
	
	url := fmt.Sprintf("%s/%s/_apis/wit/wiql?api-version=7.0", strings.TrimRight(cfg.ADO.Organization, "/"), cfg.ADO.DefaultProject)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(query)))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("", adoPat)
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to hit WIQL endpoint: %w", err)
	}
	defer resp.Body.Close()
	
	var wiqlResp struct {
		WorkItems []struct {
			ID int `json:"id"`
		} `json:"workItems"`
	}
	
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("WIQL rejected (HTTP %d): %s", resp.StatusCode, string(body))
	}
	
	if err := json.Unmarshal(body, &wiqlResp); err != nil {
		return nil, err
	}
	
	tasks, _ := a.Store.ListTasks("")
	adoToLocal := make(map[int]string)
	for _, t := range tasks {
		if t.ADOID != nil {
			adoToLocal[*t.ADOID] = t.ID
		}
	}
	
	logf(logChan, "WIQL returned %d tasks assigned to you. Comparing against local files...\n", len(wiqlResp.WorkItems))
	
	var pendingTasks []*Task
	var conflicts []*Task
	childToParentADO := make(map[string]int)
	generatedSeqs := make(map[string]int)
	
	var allADOIDs []string
	for _, wi := range wiqlResp.WorkItems {
		allADOIDs = append(allADOIDs, strconv.Itoa(wi.ID))
	}
	
	for i := 0; i < len(allADOIDs); i += 50 {
		end := i + 50
		if end > len(allADOIDs) {
			end = len(allADOIDs)
		}
		chunk := allADOIDs[i:end]
		
		detailUrl := fmt.Sprintf("%s/_apis/wit/workitems?ids=%s&api-version=7.0&$expand=relations", strings.TrimRight(cfg.ADO.Organization, "/"), strings.Join(chunk, ","))
		req2, _ := http.NewRequest("GET", detailUrl, nil)
		req2.SetBasicAuth("", adoPat)
		
		resp2, err := client.Do(req2)
		if err != nil || resp2.StatusCode >= 400 {
			continue
		}
		
		var batchDetails struct {
			Value []struct {
				ID int `json:"id"`
				Rev int `json:"rev"`
				Fields map[string]interface{} `json:"fields"`
				Relations []struct {
					Rel string `json:"rel"`
					URL string `json:"url"`
				} `json:"relations"`
			} `json:"value"`
		}
		dBody, _ := io.ReadAll(resp2.Body)
		resp2.Body.Close()
		json.Unmarshal(dBody, &batchDetails)
		
		for _, details := range batchDetails.Value {
			localID := adoToLocal[details.ID]
			var existingTask *Task
			if localID != "" {
				existingTask, _ = a.Store.Load(localID)
			}
			
			if existingTask != nil && existingTask.ADORev >= details.Rev {
				// We are up to date or ahead
				continue
			}
			
			if existingTask == nil {
				logf(logChan, "Restoring missing task ADO #%d...\n", details.ID)
			} else {
				logf(logChan, "Conflict detected for ADO #%d. Remote Rev (%d) > Local (%d). Resolving...\n", details.ID, details.Rev, existingTask.ADORev)
			}
			
			title, _ := details.Fields["System.Title"].(string)
			state, _ := details.Fields["System.State"].(string)
			adoType, _ := details.Fields["System.WorkItemType"].(string)
			
			var spPtr *float64
			if spVal, ok := details.Fields["Microsoft.VSTS.Scheduling.StoryPoints"].(float64); ok {
				spPtr = &spVal
			} else {
				defaultSP := 1.0
				spPtr = &defaultSP
			}
			
			createdStr, _ := details.Fields["System.CreatedDate"].(string)
			changedStr, _ := details.Fields["System.ChangedDate"].(string)
			
			createdAt := time.Now()
			if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
				createdAt = t
			}
			
			updatedAt := time.Now()
			if t, err := time.Parse(time.RFC3339, changedStr); err == nil {
				updatedAt = t
			}
			
			descriptionHTML, _ := details.Fields["System.Description"].(string)
			acHTML, _ := details.Fields["Microsoft.VSTS.Common.AcceptanceCriteria"].(string)
			
			var bodyBuilder strings.Builder
			if descriptionHTML != "" {
				markdown, err := converter.ConvertString(descriptionHTML)
				if err == nil {
					bodyBuilder.WriteString(markdown)
				}
			}
			if acHTML != "" {
				markdown, err := converter.ConvertString(acHTML)
				if err == nil {
					if bodyBuilder.Len() > 0 {
						bodyBuilder.WriteString("\n\n---\n\n")
					}
					bodyBuilder.WriteString("### Acceptance Criteria\n\n")
					bodyBuilder.WriteString(markdown)
				}
			}
			
			adoIdVal := details.ID
			var newTask *Task
			isConflict := false
			
			if existingTask == nil {
				baseID, _ := a.Store.GetNextID(createdAt, false)
				prefix := baseID[:9]
				
				seq := 1
				if val, exists := generatedSeqs[prefix]; exists {
					seq = val + 1
				} else {
					if len(baseID) > 9 {
						fmt.Sscanf(baseID[9:], "%d", &seq)
					}
				}
				generatedSeqs[prefix] = seq
				newID := fmt.Sprintf("%s%03d", prefix, seq)
				
				newTask = &Task{
					ID: newID,
					CreatedAt: createdAt,
					TotalSeconds: 0,
					SyncedSeconds: 0,
				}
				adoToLocal[details.ID] = newID
			} else {
				taskCopy := *existingTask
				newTask = &taskCopy
				isConflict = true
			}
			
			newTask.Title = title
			newTask.Status = TaskState(state)
			newTask.ADOType = adoType
			newTask.StoryPoints = spPtr
			newTask.ADOID = &adoIdVal
			newTask.ADORev = details.Rev
			newTask.UpdatedAt = updatedAt
			newTask.Body = bodyBuilder.String()
			
			// If missing, also pull 7pace time
			if existingTask == nil {
				orgName := extractOrgName(cfg.ADO.Organization)
				if orgName != "" && sevenPaceToken != "" {
					sevenUrl := fmt.Sprintf("https://%s.timehub.7pace.com/api/rest/workLogs?api-version=3.1&$filter=WorkItemId%%20eq%%20%d", orgName, details.ID)
					sReq, _ := http.NewRequest("GET", sevenUrl, nil)
					sReq.Header.Set("Authorization", "Bearer "+sevenPaceToken)
					sResp, err := client.Do(sReq)
					if err == nil {
						sBody, _ := io.ReadAll(sResp.Body)
						if sResp.StatusCode == 200 {
							type WorkLog struct {
								Length int `json:"length"`
								WorkItemId int `json:"workItemId"`
								WorkItem struct {
									ID int `json:"id"`
								} `json:"workItem"`
								User struct {
									Email string `json:"email"`
								} `json:"user"`
							}
							
							var sData struct {
								Data  []WorkLog `json:"data"`
								Items []WorkLog `json:"items"`
								Value []WorkLog `json:"value"`
							}
							
							json.Unmarshal(sBody, &sData)
							
							totalTime := 0
							allLogs := append(sData.Data, append(sData.Items, sData.Value...)...)
							
							for _, l := range allLogs {
								if l.WorkItemId != details.ID && l.WorkItem.ID != details.ID {
									continue
								}
								targetEmail := cfg.SevenPace.Email
								if targetEmail == "" {
									targetEmail = cfg.User.Email
								}
								
								if targetEmail == "" || strings.EqualFold(l.User.Email, targetEmail) {
									totalTime += l.Length
								}
							}
							newTask.TotalSeconds = totalTime
							newTask.SyncedSeconds = totalTime
						}
						sResp.Body.Close()
					}
				}
			}
			
			if isConflict {
				conflicts = append(conflicts, newTask)
			} else {
				pendingTasks = append(pendingTasks, newTask)
			}
			
			// Check for parent relation
			for _, rel := range details.Relations {
				if rel.Rel == "System.LinkTypes.Hierarchy-Reverse" {
					parts := strings.Split(rel.URL, "/")
					if len(parts) > 0 {
						if parentID, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
							childToParentADO[newTask.ID] = parentID
						}
					}
				}
			}
		}
	}
	
	// Pass 1.5: Fetch any missing Parents on-demand (e.g. if they fell outside the 14-day WIQL window)
	for _, parentADO := range childToParentADO {
		if adoToLocal[parentADO] == "" {
			logf(logChan, "Fetching out-of-window parent ADO #%d...\n", parentADO)
			detailUrl := fmt.Sprintf("%s/_apis/wit/workitems/%d?api-version=7.0", strings.TrimRight(cfg.ADO.Organization, "/"), parentADO)
			req2, _ := http.NewRequest("GET", detailUrl, nil)
			req2.SetBasicAuth("", adoPat)
			
			resp2, err := client.Do(req2)
			if err != nil || resp2.StatusCode >= 400 {
				continue
			}
			
			var details struct {
				Fields map[string]interface{} `json:"fields"`
			}
			dBody, _ := io.ReadAll(resp2.Body)
			resp2.Body.Close()
			json.Unmarshal(dBody, &details)
			
			title, _ := details.Fields["System.Title"].(string)
			state, _ := details.Fields["System.State"].(string)
			adoType, _ := details.Fields["System.WorkItemType"].(string)
			
			var spPtr *float64
			if spVal, ok := details.Fields["Microsoft.VSTS.Scheduling.StoryPoints"].(float64); ok {
				spPtr = &spVal
			} else {
				defaultSP := 1.0
				spPtr = &defaultSP
			}
			
			createdStr, _ := details.Fields["System.CreatedDate"].(string)
			changedStr, _ := details.Fields["System.ChangedDate"].(string)
			
			createdAt := time.Now()
			if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
				createdAt = t
			}
			updatedAt := time.Now()
			if t, err := time.Parse(time.RFC3339, changedStr); err == nil {
				updatedAt = t
			}
			
			baseID, _ := a.Store.GetNextID(createdAt, false)
			prefix := baseID[:9]
			seq := 1
			if val, exists := generatedSeqs[prefix]; exists {
				seq = val + 1
			} else if len(baseID) > 9 {
				fmt.Sscanf(baseID[9:], "%d", &seq)
			}
			generatedSeqs[prefix] = seq
			newID := fmt.Sprintf("%s%03d", prefix, seq)
			
			adoIdVal := parentADO
			
			descriptionHTML, _ := details.Fields["System.Description"].(string)
			acHTML, _ := details.Fields["Microsoft.VSTS.Common.AcceptanceCriteria"].(string)
			
			var bodyBuilder strings.Builder
			if descriptionHTML != "" {
				markdown, err := converter.ConvertString(descriptionHTML)
				if err == nil {
					bodyBuilder.WriteString(markdown)
				}
			}
			if acHTML != "" {
				markdown, err := converter.ConvertString(acHTML)
				if err == nil {
					if bodyBuilder.Len() > 0 {
						bodyBuilder.WriteString("\n\n---\n\n")
					}
					bodyBuilder.WriteString("### Acceptance Criteria\n\n")
					bodyBuilder.WriteString(markdown)
				}
			}
			
			newTask := &Task{
				ID: newID,
				Title: title,
				Status: TaskState(state),
				ADOType: adoType,
				StoryPoints: spPtr,
				ADOID: &adoIdVal,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				Body: bodyBuilder.String(),
			}
			
			adoToLocal[parentADO] = newID
			pendingTasks = append(pendingTasks, newTask)
		}
	}

	// Pass 2: Reconstruct Hierarchy and Save
	for _, t := range pendingTasks {
		if parentADO, ok := childToParentADO[t.ID]; ok {
			if localParentID, exists := adoToLocal[parentADO]; exists {
				t.ParentID = localParentID
			}
		}
		a.Store.Save(t)
	}
	
	logf(logChan, "Successfully restored %d missing tasks from ADO!\n", len(pendingTasks))
	return conflicts, nil
}

// SyncSingle executes the network operations to push a single local task to ADO and 7pace.
func (a *App) SyncSingle(cfg *config.Config, adoPat string, sevenPaceToken string, taskID string, logChan chan<- string) ([]*Task, error) {
	t, err := a.Store.Load(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to load task %s: %w", taskID, err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	orgName := extractOrgName(cfg.ADO.Organization)

	if t.ADOID == nil {
		logf(logChan, "Task %s is not linked to ADO (no ADOID). Ignoring push.\n", t.ID)
	} else {
		logf(logChan, "Syncing task %s (ADO #%d...\no ADO...\n", t.ID, *t.ADOID)
		
		patch := []map[string]interface{}{}
		if t.ADORev > 0 {
			patch = append(patch, map[string]interface{}{
				"op": "test", "path": "/rev", "value": t.ADORev,
			})
		}
		patch = append(patch,
			map[string]interface{}{"op": "add", "path": "/fields/System.Title", "value": t.Title},
			map[string]interface{}{"op": "add", "path": "/fields/System.State", "value": string(t.Status)},
		)
		
		if t.Body != "" {
			desc, ac := parseMarkdownSections(t.Body)
			if desc != "" {
				patch = append(patch, map[string]interface{}{
					"op": "add", "path": "/fields/System.Description", "value": desc,
				})
			}
			if ac != "" {
				patch = append(patch, map[string]interface{}{
					"op": "add", "path": "/fields/Microsoft.VSTS.Common.AcceptanceCriteria", "value": ac,
				})
			}
		}
		
		if len(t.Tags) > 0 {
			patch = append(patch, map[string]interface{}{
				"op": "add", "path": "/fields/System.Tags", "value": strings.Join(t.Tags, "; "),
			})
		}
		
		if cfg.ADO.SwimlaneField != "" && t.Swimlane != "" {
			patch = append(patch, map[string]interface{}{
				"op": "add", "path": "/fields/" + cfg.ADO.SwimlaneField, "value": t.Swimlane,
			})
		}
		
		if t.StoryPoints != nil {
			patch = append(patch, map[string]interface{}{
				"op": "add", "path": "/fields/Microsoft.VSTS.Scheduling.StoryPoints", "value": *t.StoryPoints,
			})
		}
		
		// Handle ADO Hierarchy Linking
		if t.ParentID != "" {
			parentTask, err := a.Store.Load(t.ParentID)
			if err == nil && parentTask.ADOID != nil {
				patch = append(patch, map[string]interface{}{
					"op": "add",
					"path": "/relations/-",
					"value": map[string]interface{}{
						"rel": "System.LinkTypes.Hierarchy-Reverse",
						"url": fmt.Sprintf("%s/_apis/wit/workitems/%d", strings.TrimRight(cfg.ADO.Organization, "/"), *parentTask.ADOID),
						"attributes": map[string]interface{}{
							"comment": "Linked via Tally",
						},
					},
				})
			}
		}

		payload, _ := json.Marshal(patch)
		url := fmt.Sprintf("%s/%s/_apis/wit/workitems/%d?api-version=7.0", strings.TrimRight(cfg.ADO.Organization, "/"), cfg.ADO.DefaultProject, *t.ADOID)
		
		req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json-patch+json")
		req.SetBasicAuth("", adoPat)
		
		resp, err := client.Do(req)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode >= 400 {
				bodyStr := string(body)
				if strings.Contains(bodyStr, "TF401346") || strings.Contains(bodyStr, "rev") || resp.StatusCode == 412 || resp.StatusCode == 400 {
					logf(logChan, "  -> Conflict! ADO item #%d was modified remotely. Fetching details...\n", *t.ADOID)
					detailUrl := fmt.Sprintf("%s/_apis/wit/workitems/%d?api-version=7.0", strings.TrimRight(cfg.ADO.Organization, "/"), *t.ADOID)
					reqF, _ := http.NewRequest("GET", detailUrl, nil)
					reqF.SetBasicAuth("", adoPat)
					respF, err := client.Do(reqF)
					if err == nil && respF.StatusCode == 200 {
						var details struct {
							ID int `json:"id"`
							Rev int `json:"rev"`
							Fields map[string]interface{} `json:"fields"`
						}
						dBody, _ := io.ReadAll(respF.Body)
						json.Unmarshal(dBody, &details)
						respF.Body.Close()
						taskCopy := *t
						newTask := &taskCopy
						newTask.ADORev = details.Rev
						if val, ok := details.Fields["System.Title"].(string); ok { newTask.Title = val }
						if val, ok := details.Fields["System.State"].(string); ok { newTask.Status = TaskState(val) }
						
						converter := md.NewConverter("", true, nil)
						descHTML, _ := details.Fields["System.Description"].(string)
						acHTML, _ := details.Fields["Microsoft.VSTS.Common.AcceptanceCriteria"].(string)
						var bodyBuilder strings.Builder
						if descHTML != "" {
							if markdown, err := converter.ConvertString(descHTML); err == nil {
								bodyBuilder.WriteString(markdown)
							}
						}
						if acHTML != "" {
							if markdown, err := converter.ConvertString(acHTML); err == nil {
								if bodyBuilder.Len() > 0 {
									bodyBuilder.WriteString("\n\n---\n\n")
								}
								bodyBuilder.WriteString("### Acceptance Criteria\n\n")
								bodyBuilder.WriteString(markdown)
							}
						}
						newTask.Body = bodyBuilder.String()
						
						return []*Task{newTask}, nil
					}
				}
				logf(logChan, "  -> ADO rejected update for task %s (HTTP %d). Response: %s\n", t.ID, resp.StatusCode, bodyStr)
			} else {
				var updateResp struct {
					Rev int `json:"rev"`
				}
				if err := json.Unmarshal(body, &updateResp); err == nil && updateResp.Rev > 0 {
					t.ADORev = updateResp.Rev
					a.Store.Save(t)
				}
				logf(logChan, "  -> Successfully updated ADO Work Item #%d\n", *t.ADOID)
			}
			resp.Body.Close()
		}
	}

	hasChanges := false
	for i := range t.TimeLogs {
		logEntry := &t.TimeLogs[i]
		if logEntry.Synced || t.ADOID == nil {
			continue
		}

		logf(logChan, "Pushing %d seconds of time to 7pace for ADO #%d...\n\n", logEntry.Seconds, *t.ADOID)
		
		logData := map[string]interface{}{
			"timestamp":  logEntry.Timestamp.Format(time.RFC3339),
			"length":     logEntry.Seconds,
			"workItemId": *t.ADOID,
			"comment":    "Logged via Tally terminal",
		}
		
		actID := logEntry.ActivityID
		if actID == "" {
			actID = cfg.SevenPace.ActivityID
		}
		
		if len(actID) >= 32 {
			logData["activityTypeId"] = actID
		}
		
		payload, _ := json.Marshal(logData)
		url := fmt.Sprintf("https://%s.timehub.7pace.com/api/rest/workLogs?api-version=3.1", orgName)
		
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+sevenPaceToken)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("network error hitting 7pace: %w", err)
		}
		
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			logEntry.Synced = true
			hasChanges = true
			logf(logChan, "  -> Time successfully logged to 7pace!\n")
		} else {
			body, _ := io.ReadAll(resp.Body)
			logf(logChan, "  -> Failed to log time to 7pace (HTTP %d). Response: %s\n", resp.StatusCode, string(body))
		}
		resp.Body.Close()
	}
	
	if hasChanges {
		synced := 0
		for _, l := range t.TimeLogs {
			if l.Synced {
				synced += l.Seconds
			}
		}
		t.SyncedSeconds = synced
		a.Store.Save(t)
	}

	return nil, nil
}
