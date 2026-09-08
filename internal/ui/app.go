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

package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sort"
	"time"
	"path/filepath"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gotcode.org/tally/internal/config"
	"gotcode.org/tally/internal/core"
	"gotcode.org/tally/internal/version"
)

type SyncLogMsg string
type CloseModalMsg struct{}
type ShowVersionMsg struct{}

type AppState int

const (
	StateDashboard AppState = iota
	StateCreateTask
	StateLogTime
	StateStandup
	StateSyncing
	StateVersion
	StateConflict
)

type MainModel struct {
	coreApp       *core.App
	state         AppState
	standupText   string
	standupScroll int
	versionScroll int
	syncLogs      []string
	syncErr       error
	logChannel    chan string
	syncTicks     int
	conflictQueue []*core.Task
	list          ListModel
	expandedState map[string]bool
	form    *FormModel
	
	selectedID     string
	terminalWidth  int
	terminalHeight int
}

func NewMainModel(app *core.App) *MainModel {
	if cfg, err := config.Load(); err == nil {
		if cfg.UI.Theme != "" {
			ApplyThemeByName(cfg.UI.Theme)
		}
	}
	m := &MainModel{
		coreApp:       app,
		state:         StateDashboard,
		expandedState: make(map[string]bool),
	}
	m.reloadList()
	return m
}

func (m *MainModel) reloadList() {
	// Capture current expansion state and cursor before wiping
	var selectedID string
	oldCursor := 0
	if m.list.items != nil {
		oldCursor = m.list.cursor
		flat := m.list.getFlatRows()
		if m.list.cursor >= 0 && m.list.cursor < len(flat) {
			selectedID = flat[m.list.cursor].Item.ID
		}
		
		var captureState func(items []*ListItem)
		captureState = func(items []*ListItem) {
			for _, item := range items {
				// Use ID if available, otherwise fallback to Title (for folders)
				key := item.ID
				if key == "" {
					key = "folder:" + item.Title
				}
				m.expandedState[key] = item.Expanded
				captureState(item.Children)
			}
		}
		captureState(m.list.items)
	}

	tasks, _ := m.coreApp.Store.ListTasks("")

	// Reverse tasks so newest are at the top
	for i, j := 0, len(tasks)-1; i < j; i, j = i+1, j-1 {
		tasks[i], tasks[j] = tasks[j], tasks[i]
	}

	var items []*ListItem
	
	
	
	
	
	// 1. Map out all task ListItems so we can nest children easily
	itemMap := make(map[string]*ListItem)
	
	// 2. Pre-generate all ListItems
	for _, t := range tasks {
		c := ThemeBlue
		if strings.Contains(strings.ToLower(t.ADOType), "story") {
			c = ThemeMauve
		} else if strings.Contains(strings.ToLower(t.ADOType), "bug") {
			c = ThemeRed
		}

		timeStr := ""
		if t.TotalSeconds > 0 {
			hours := float64(t.TotalSeconds) / 3600.0
			timeStr = fmt.Sprintf("%.1fh", hours)
		}

		titleStr := t.Title
		if t.Recurrence != "" {
			titleStr = "↻ " + titleStr
		}

		expanded := true
		if val, exists := m.expandedState[t.ID]; exists {
			expanded = val
		}

		itemMap[t.ID] = &ListItem{
			ID:        t.ID,
			Title:     titleStr,
			Type:      t.ADOType,
			Status:    string(t.Status),
			TimeText:  timeStr,
			TypeColor: c,
			Expanded:  expanded, // keep children visible by default or use captured state
		}
	}
	
	// 3. Nest children into parents, and build a list of top-level nodes
	var topLevelTasks []*core.Task
	for _, t := range tasks {
		if t.ParentID != "" {
			// Find parent in the map
			if parentNode, exists := itemMap[t.ParentID]; exists {
				// Attach this child to the parent!
				parentNode.Children = append(parentNode.Children, itemMap[t.ID])
			} else {
				// Parent is missing (maybe deleted?), render as top-level fallback
				topLevelTasks = append(topLevelTasks, t)
			}
		} else {
			// No parent, so it's a top-level node
			topLevelTasks = append(topLevelTasks, t)
		}
	}

	cfg, _ := config.Load()
	var dashStates []string
	var archiveStates []string
	
	if cfg != nil && len(cfg.UI.DashboardStates) > 0 {
		dashStates = cfg.UI.DashboardStates
	} else {
		dashStates = []string{"New", "Active", "In Progress", "Backlog"}
	}
	
	if cfg != nil && len(cfg.UI.ArchiveStates) > 0 {
		archiveStates = cfg.UI.ArchiveStates
	} else {
		archiveStates = []string{"Done", "Closed", "Removed"}
	}

	isArchive := func(s string) bool {
		for _, as := range archiveStates {
			if strings.EqualFold(s, as) {
				return true
			}
		}
		return false
	}
	
	stateNodes := make(map[string]*ListItem)
	var archiveRoot *ListItem
	
	var templatesNode *ListItem

	for _, t := range topLevelTasks {
		taskItem := itemMap[t.ID]

		// Intercept Templates
		if strings.HasPrefix(t.ID, "recur-") {
			if templatesNode == nil {
				expanded := false
				if val, ok := m.expandedState["folder:Templates"]; ok { expanded = val }
				templatesNode = &ListItem{Title: "Templates", Expanded: expanded} 
			}
			templatesNode.Children = append(templatesNode.Children, taskItem)
			continue
		}

		statusStr := string(t.Status)
		if statusStr == "" {
			statusStr = "New"
		}
		
		displayStr := statusStr
		if cfg != nil && cfg.UI.StateAliases != nil {
			if alias, ok := cfg.UI.StateAliases[statusStr]; ok {
				displayStr = alias
			}
		}
		
		if isArchive(statusStr) || isArchive(displayStr) {
			if archiveRoot == nil {
				expanded := false
				if val, ok := m.expandedState["folder:Archive"]; ok { expanded = val }
				archiveRoot = &ListItem{Title: "Archive", Expanded: expanded}
			}
			
			// Group archived tasks by when they were last touched (closed)
			yStr := t.UpdatedAt.Format("2006")
			mStr := t.UpdatedAt.Format("January")
			dStr := t.UpdatedAt.Format("02 (Mon)")
			
			var yNode, mNode, dNode *ListItem
			for _, child := range archiveRoot.Children {
				if child.Title == yStr {
					yNode = child
					break
				}
			}
			if yNode == nil {
				expanded := false
				if val, ok := m.expandedState["folder:"+yStr]; ok { expanded = val }
				yNode = &ListItem{Title: yStr, Expanded: expanded}
				archiveRoot.Children = append(archiveRoot.Children, yNode)
			}
			
			for _, child := range yNode.Children {
				if child.Title == mStr {
					mNode = child
					break
				}
			}
			if mNode == nil {
				expanded := false
				if val, ok := m.expandedState["folder:"+mStr]; ok { expanded = val }
				mNode = &ListItem{Title: mStr, Expanded: expanded}
				yNode.Children = append(yNode.Children, mNode)
			}
			
			for _, child := range mNode.Children {
				if child.Title == dStr {
					dNode = child
					break
				}
			}
			if dNode == nil {
				expanded := false
				if val, ok := m.expandedState["folder:"+dStr]; ok { expanded = val }
				dNode = &ListItem{Title: dStr, Expanded: expanded}
				mNode.Children = append(mNode.Children, dNode)
			}
			
			dNode.Children = append(dNode.Children, taskItem)
			continue
		}
		
		// It's an active task, group by state alias
		node, exists := stateNodes[displayStr]
		if !exists {
			expanded := true // Default to expanded for active states
			if val, ok := m.expandedState["folder:"+displayStr]; ok { expanded = val }
			node = &ListItem{Title: displayStr, Expanded: expanded}
			stateNodes[displayStr] = node
		}
		node.Children = append(node.Children, taskItem)
	}

	var finalItems []*ListItem
	
	// Add configured states in exact order
	for _, st := range dashStates {
		// We use case-insensitive matching in case config casing doesn't exactly match ADO casing
		for k, node := range stateNodes {
			if strings.EqualFold(k, st) {
				finalItems = append(finalItems, node)
				delete(stateNodes, k)
				break
			}
		}
	}
	
	// Add any unknown dynamically discovered states
	var unknownKeys []string
	for k := range stateNodes {
		unknownKeys = append(unknownKeys, k)
	}
	sort.Strings(unknownKeys)
	for _, k := range unknownKeys {
		finalItems = append(finalItems, stateNodes[k])
	}
	
	if archiveRoot != nil {
		finalItems = append(finalItems, archiveRoot)
	}
	if templatesNode != nil {
		finalItems = append(finalItems, templatesNode)
	}
	
	items = finalItems

	
	// Calculate Time Stats
	now := time.Now()
	var daySec, weekSec, monthSec, yearSec int
	
	// Determine the start of the ISO week (Monday)
	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6
	}
	startOfWeek := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, offset)
	
	for _, t := range tasks {
		// Calculate granular time logs
		for _, log := range t.TimeLogs {
			if log.Timestamp.Year() == now.Year() {
				yearSec += log.Seconds
				if log.Timestamp.Month() == now.Month() {
					monthSec += log.Seconds
					if log.Timestamp.Day() == now.Day() {
						daySec += log.Seconds
					}
				}
				if !log.Timestamp.Before(startOfWeek) {
					weekSec += log.Seconds
				}
			}
		}
		// Also support legacy TotalSeconds if TimeLogs is empty
		if len(t.TimeLogs) == 0 && t.TotalSeconds > 0 {
			if t.CreatedAt.Year() == now.Year() {
				yearSec += t.TotalSeconds
				if t.CreatedAt.Month() == now.Month() {
					monthSec += t.TotalSeconds
					if t.CreatedAt.Day() == now.Day() {
						daySec += t.TotalSeconds
					}
				}
				if !t.CreatedAt.Before(startOfWeek) {
					weekSec += t.TotalSeconds
				}
			}
		}
	}
	
	timeStatsStr := fmt.Sprintf(" TIME LOGGED - Day: %.1fh | Week: %.1fh | Month: %.1fh | Year: %.1fh ", 
		float64(daySec)/3600.0, float64(weekSec)/3600.0, float64(monthSec)/3600.0, float64(yearSec)/3600.0)

	m.list = NewListModel("TALLY DASHBOARD", items)
	m.list.TimeStats = timeStatsStr
	
	// Restore cursor position
	flat := m.list.getFlatRows()
	found := false
	if selectedID != "" {
		for i, row := range flat {
			if row.Item.ID == selectedID {
				m.list.cursor = i
				found = true
				break
			}
		}
	}
	// Fallback: if the item was deleted, try to put the cursor back where it roughly was
	if !found && len(flat) > 0 {
		if oldCursor >= len(flat) {
			m.list.cursor = len(flat) - 1
		} else if oldCursor >= 0 {
			m.list.cursor = oldCursor
		}
	}
	
	// Propagate dimensions if already set
	if m.terminalWidth > 0 {
		newM, _ := m.list.Update(tea.WindowSizeMsg{Width: m.terminalWidth, Height: m.terminalHeight})
		m.list = newM.(ListModel)
	}
}

func (m *MainModel) buildCreateForm(parentID string) {
	title := "CREATE NEW TASK"
	if parentID != "" {
		title = "CREATE SUBTASK FOR: " + parentID
	}
	f := NewForm(title)
	f.AddTextBox("title", "Title", "Enter task title...", "")
	f.AddSelector("type", "Type", []string{"Task", "Story", "Technical Story", "Bug"}, "")
	f.AddTextBox("tags", "Tags", "comma separated (e.g. urgent, backend)", "")
	f.AddSelector("recur", "Recurrence", []string{"", "daily", "weekdays", "weekly", "monthly"}, "")
	
	if cfg, err := config.Load(); err == nil && len(cfg.ADO.Swimlanes) > 0 {
		opts := []string{""}
		opts = append(opts, cfg.ADO.Swimlanes...)
		f.AddSelector("swimlane", "Swimlane", opts, cfg.ADO.DefaultSwimlane)
	}
	
	f.AddButton("CREATE", ThemeGreen, ThemeBase, func(form *FormModel) tea.Cmd {
		return func() tea.Msg {
			title := form.GetString("title")
			adoType := form.GetString("type")
			tagStr := form.GetString("tags")
			
			var tags []string
			if tagStr != "" {
				for _, t := range strings.Split(tagStr, ",") {
					tags = append(tags, strings.TrimSpace(t))
				}
			}

			if title != "" {
				m.coreApp.AddTask(title, adoType, tags, form.GetString("recur"), parentID, form.GetString("swimlane"))
			}
			return FormSubmitMsg{}
		}
	})
	
	f.AddButton("CANCEL", ThemeRed, ThemeBase, func(form *FormModel) tea.Cmd {
		return func() tea.Msg { return FormCancelMsg{} }
	})
	
	m.form = f
	if m.terminalWidth > 0 {
		m.form.Update(tea.WindowSizeMsg{Width: m.terminalWidth, Height: m.terminalHeight})
	}
	m.form.Init()
}

func (m *MainModel) buildLogTimeForm(id string) {
	// Try to load activities from config for the dropdown
	cfg, _ := config.Load()
	activityNames := []string{"Default"}
	if cfg != nil && cfg.SevenPace.Activities != nil {
		for name := range cfg.SevenPace.Activities {
			activityNames = append(activityNames, name)
		}
	}

	f := NewForm("LOG TIME: " + id)
	f.AddTextBox("time", "Time", "e.g., 5m, 1.5h", "")
	if len(activityNames) > 1 {
		f.AddSelector("activity", "Activity", activityNames, "")
	}
	
	f.AddButton("LOG TIME", ThemeGreen, ThemeBase, func(form *FormModel) tea.Cmd {
		return func() tea.Msg {
			timeStr := form.GetString("time")
			activityName := form.GetString("activity")
			
			var activityID string
			if cfg != nil {
				if activityName == "Default" || activityName == "" {
					activityID = cfg.SevenPace.ActivityID
				} else {
					activityID = cfg.SevenPace.Activities[activityName]
				}
			}

			if timeStr != "" {
				m.coreApp.LogTime(id, timeStr, activityID)
			}
			return FormSubmitMsg{}
		}
	})
	
	f.AddButton("CANCEL", ThemeRed, ThemeBase, func(form *FormModel) tea.Cmd {
		return func() tea.Msg { return FormCancelMsg{} }
	})
	
	m.form = f
	if m.terminalWidth > 0 {
		m.form.Update(tea.WindowSizeMsg{Width: m.terminalWidth, Height: m.terminalHeight})
	}
	m.form.Init()
}

type FormSubmitMsg struct{}
type FormCancelMsg struct{}
type LogTimeMsg struct{ ID string }
type DeleteTaskMsg struct{ ID string }
type CreateSubtaskMsg struct{ ParentID string }
type StartTaskMsg struct{ ID string }

func (m *MainModel) Init() tea.Cmd {
	return m.list.Init()
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		newM, _ := m.list.Update(msg)
		m.list = newM.(ListModel)
		if m.form != nil {
			newF, _ := m.form.Update(msg)
			m.form = newF.(*FormModel)
		}
		return m, nil
		
	case RefreshMsg:
		m.reloadList()
		return m, nil

	case CycleThemeMsg:
		newTheme := CycleTheme()
		
		// Save the new theme to config
		cfg, err := config.Load()
		if err == nil {
			cfg.UI.Theme = newTheme
			config.Save(cfg)
		}
		
		m.reloadList() // redraw with new global styles
		return m, nil

	case SyncTasksMsg:
		m.state = StateSyncing
		m.syncTicks = 0
		m.syncLogs = []string{"Starting Bulk Push..."}
		m.syncErr = nil
		m.logChannel = make(chan string, 100)
		return m, tea.Batch(m.startSyncProcess("sync", ""), listenToLogs(m.logChannel))
		
	case PullTasksMsg:
		m.state = StateSyncing
		m.syncTicks = 0
		m.syncLogs = []string{"Starting ADO Fetch..."}
		m.syncErr = nil
		m.logChannel = make(chan string, 100)
		return m, tea.Batch(m.startSyncProcess("fetch", ""), listenToLogs(m.logChannel))
		
	case PushSingleTaskMsg:
		m.state = StateSyncing
		m.syncTicks = 0
		m.syncLogs = []string{"Starting Targeted Push..."}
		m.syncErr = nil
		m.logChannel = make(chan string, 100)
		return m, tea.Batch(m.startSyncProcess("push_single", msg.ID), listenToLogs(m.logChannel))
		
	case SyncLogMsg:
		m.syncTicks++
		m.syncLogs = append(m.syncLogs, string(msg))
		// keep only last 50 lines to prevent memory issues
		if len(m.syncLogs) > 50 {
			m.syncLogs = m.syncLogs[len(m.syncLogs)-50:]
		}
		return m, listenToLogs(m.logChannel)

	case SyncFinishedMsg:
		m.coreApp.ReconcileRecurringTasks()
		m.reloadList()
		m.syncErr = msg.Err
		
		if msg.Err == nil {
			if len(msg.Conflicts) > 0 {
				// Don't auto dismiss to dashboard, transition to conflicts
				m.conflictQueue = msg.Conflicts
				m.state = StateConflict
				return m, nil
			}
			
			// Auto dismiss on success
			return m, func() tea.Msg {
				time.Sleep(800 * time.Millisecond)
				return CloseModalMsg{}
			}
		}
		return m, nil
		
	case CloseModalMsg:
		if m.state == StateSyncing && m.syncErr == nil {
			m.state = StateDashboard
		}
		return m, nil

	case EditTaskMsg:
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}
		path := m.coreApp.Store.GetTaskPath(msg.ID)
		c := exec.Command(editor, path)
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			return EditorFinishedMsg{Err: err}
		})
		
	case EditorFinishedMsg:
		m.reloadList() // reload data after editor closes
		return m, nil

	case ShowVersionMsg:
		m.state = StateVersion
		m.versionScroll = 0
		return m, nil

	case GenerateStandupMsg:
		m.generateStandup()
		m.standupScroll = 0
		m.state = StateStandup
		return m, nil

	case CreateNewTaskMsg:
		m.state = StateCreateTask
		m.buildCreateForm("")
		return m, nil
		
	case CreateSubtaskMsg:
		m.state = StateCreateTask
		m.buildCreateForm(msg.ParentID)
		return m, nil
		
	case LogTimeMsg:
		m.state = StateLogTime
		m.selectedID = msg.ID
		m.buildLogTimeForm(msg.ID)
		return m, nil
		
	case DeleteTaskMsg:
		m.coreApp.DeleteTask(msg.ID)
		m.reloadList()
		return m, nil
		
	case StartTaskMsg:
		m.coreApp.StartTask(msg.ID)
		m.reloadList()
		return m, nil
		
	case FormSubmitMsg, FormCancelMsg:
		m.state = StateDashboard
		m.form = nil
		m.coreApp.ReconcileRecurringTasks() // Force a spawn check in case they just created a template
		m.reloadList() // Pull fresh data
		return m, nil
	}

	var cmd tea.Cmd
	
	if m.state == StateConflict {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "o", "O":
				if len(m.conflictQueue) > 0 {
					m.coreApp.Store.Save(m.conflictQueue[0])
					m.conflictQueue = m.conflictQueue[1:]
				}
			case "k", "K":
				if len(m.conflictQueue) > 0 {
					remoteTask := m.conflictQueue[0]
					localTask, _ := m.coreApp.Store.Load(remoteTask.ID)
					if localTask != nil {
						localTask.ADORev = remoteTask.ADORev
						m.coreApp.Store.Save(localTask)
					}
					m.conflictQueue = m.conflictQueue[1:]
				}
			case "q", "esc":
				m.conflictQueue = nil
			}
			
			if len(m.conflictQueue) == 0 {
				m.state = StateDashboard
				m.reloadList()
			}
		}
		return m, nil
	}
	
	if m.state == StateDashboard {
		var newModel tea.Model
		newModel, cmd = m.list.Update(msg)
		m.list = newModel.(ListModel)
	} else if m.state == StateStandup || m.state == StateSyncing || m.state == StateVersion {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" || msg.String() == "q" || msg.String() == "enter" {
				m.state = StateDashboard
			}
			if m.state == StateStandup {
				if msg.String() == "up" || msg.String() == "k" {
					if m.standupScroll > 0 { m.standupScroll-- }
				}
				if msg.String() == "down" || msg.String() == "j" {
					m.standupScroll++ 
				}
				if msg.String() == "pgup" {
					m.standupScroll -= 10
					if m.standupScroll < 0 { m.standupScroll = 0 }
				}
				if msg.String() == "pgdown" {
					m.standupScroll += 10
				}
			} else if m.state == StateVersion {
				if msg.String() == "up" || msg.String() == "k" {
					if m.versionScroll > 0 { m.versionScroll-- }
				}
				if msg.String() == "down" || msg.String() == "j" {
					m.versionScroll++ 
				}
				if msg.String() == "pgup" {
					m.versionScroll -= 10
					if m.versionScroll < 0 { m.versionScroll = 0 }
				}
				if msg.String() == "pgdown" {
					m.versionScroll += 10
				}
			}
		case tea.WindowSizeMsg:
			m.terminalWidth = msg.Width
			m.terminalHeight = msg.Height
		}
	} else if m.state == StateCreateTask || m.state == StateLogTime {
		var newModel tea.Model
		newModel, cmd = m.form.Update(msg)
		m.form = newModel.(*FormModel)
	}
	
	return m, cmd
}

func (m *MainModel) View() string {
	if (m.state == StateCreateTask || m.state == StateLogTime) && m.form != nil {
		return m.form.View()
	}
	if m.state == StateSyncing {
		w := int(float64(m.terminalWidth) * 0.50)
		if w < 60 { w = 60 }
		if w > m.terminalWidth { w = m.terminalWidth }
		
		h := 12
		
		headerColor := ThemeMauve
		if m.syncErr != nil {
			headerColor = ThemeRed
		} else if len(m.syncLogs) > 0 && strings.Contains(m.syncLogs[len(m.syncLogs)-1], "✅") {
			headerColor = ThemeGreen
		}
		
		headerFull := lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Background(headerColor).Foreground(ThemeBase).Bold(true).Render(" " + strings.ToUpper("Synchronizing") + " ")
		
		contentWidth := w - 4

		lastLog := "Initializing..."
		if len(m.syncLogs) > 0 {
			lastLog = m.syncLogs[len(m.syncLogs)-1]
		}
		
		lastLog = strings.TrimSpace(lastLog)
		if strings.HasPrefix(lastLog, "->") {
			lastLog = strings.TrimSpace(strings.TrimPrefix(lastLog, "->"))
		}
		
		if lipgloss.Width(lastLog) > contentWidth - 10 {
			lastLog = lastLog[:contentWidth-13] + "..."
		}
		
		barWidth := contentWidth - 4
		var barStr string
		
		bgStyle := lipgloss.NewStyle().Background(ThemeOverlay).Foreground(ThemeText)
		
		if m.syncErr != nil || (len(m.syncLogs) > 0 && strings.Contains(m.syncLogs[len(m.syncLogs)-1], "✅")) {
			barStyle := lipgloss.NewStyle().Foreground(ThemeGreen).Background(ThemeOverlay).Bold(true)
			if m.syncErr != nil {
				barStyle = barStyle.Foreground(ThemeRed)
			}
			barStr = bgStyle.Render("[") + barStyle.Render(strings.Repeat("=", barWidth)) + bgStyle.Render("]")
		} else {
			blockWidth := 15
			if blockWidth > barWidth { blockWidth = barWidth }
			travel := barWidth - blockWidth
			if travel < 1 { travel = 1 }
			
			cycle := travel * 2
			pos := m.syncTicks % cycle
			if pos >= travel {
				pos = cycle - pos
			}
			
			leftSpace := pos
			rightSpace := barWidth - blockWidth - leftSpace
			if rightSpace < 0 { rightSpace = 0 }
			
			barStyle := lipgloss.NewStyle().Foreground(ThemeMauve).Background(ThemeOverlay).Bold(true)
			barStr = bgStyle.Render("[") + bgStyle.Render(strings.Repeat(" ", leftSpace)) + barStyle.Render(strings.Repeat("=", blockWidth)) + bgStyle.Render(strings.Repeat(" ", rightSpace)) + bgStyle.Render("]")
		}
		
		var sections []string
		sections = append(sections, lipgloss.NewStyle().Background(ThemeOverlay).Width(contentWidth).Height(2).Render(""))
		sections = append(sections, lipgloss.NewStyle().Background(ThemeOverlay).Width(contentWidth).Align(lipgloss.Center).Foreground(ThemeText).Render("Status: " + lastLog))
		sections = append(sections, lipgloss.NewStyle().Background(ThemeOverlay).Width(contentWidth).Height(1).Render(""))
		sections = append(sections, lipgloss.NewStyle().Background(ThemeOverlay).Width(contentWidth).Align(lipgloss.Center).Render(barStr))
		
		if m.syncErr != nil {
			errMsg := m.syncErr.Error()
			if lipgloss.Width(errMsg) > contentWidth - 10 {
				errMsg = errMsg[:contentWidth-13] + "..."
			}
			sections = append(sections, lipgloss.NewStyle().Background(ThemeOverlay).Width(contentWidth).Height(1).Render(""))
			sections = append(sections, lipgloss.NewStyle().Background(ThemeOverlay).Foreground(ThemeRed).Width(contentWidth).Align(lipgloss.Center).Render("Error: " + errMsg))
		}

		// Fill remaining
		for len(sections) < h - 2 {
			sections = append(sections, lipgloss.NewStyle().Background(ThemeOverlay).Width(contentWidth).Render(""))
		}
		if len(sections) > h - 2 {
			sections = sections[:h-2]
		}
		
		bodyContent := strings.Join(sections, "\n")
		paddedBody := lipgloss.NewStyle().Height(h - 2).Background(ThemeOverlay).Width(contentWidth).Render(bodyContent)
		
		footerText := " Syncing in progress... "
		if m.syncErr != nil {
			footerText = " [ERROR] Press esc/enter to dismiss "
		} else if len(m.syncLogs) > 0 && strings.Contains(m.syncLogs[len(m.syncLogs)-1], "✅") {
			footerText = " Complete! Returning to dashboard... "
		}
		
		footer := lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Background(headerColor).Foreground(ThemeBase).Bold(true).Render(footerText)
		
		win := lipgloss.NewStyle().Background(ThemeOverlay).Width(w).Height(h).Render(headerFull + "\n" + paddedBody + "\n" + footer)
		return lipgloss.Place(m.terminalWidth, m.terminalHeight, lipgloss.Center, lipgloss.Center, win)
	}
	if m.state == StateStandup {
		w := int(float64(m.terminalWidth) * 0.60)
		if w < 70 { w = 70 }
		if w > m.terminalWidth { w = m.terminalWidth }
		
		h := int(float64(m.terminalHeight) * 0.80)
		if h < 20 { h = 20 }
		if h > m.terminalHeight { h = m.terminalHeight }
		
		headerFull := lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Background(ThemeMauve).Foreground(ThemeBase).Bold(true).Render(" STANDUP REPORT ")
		
		bodyLines := strings.Split(m.standupText, "\n")
		contentWidth := w - 4
		maxLines := h - 2
		
		maxScroll := len(bodyLines) - maxLines
		if maxScroll < 0 { maxScroll = 0 }
		if m.standupScroll > maxScroll { m.standupScroll = maxScroll }
		
		var renderedLines []string
		endIdx := m.standupScroll + maxLines
		if endIdx > len(bodyLines) { endIdx = len(bodyLines) }
		
		for _, l := range bodyLines[m.standupScroll:endIdx] {
			rawLine := l
			if lipgloss.Width(rawLine) > contentWidth {
				rawLine = rawLine[:contentWidth-3] + "..."
			}
			renderedLines = append(renderedLines, lipgloss.NewStyle().Foreground(ThemeText).Background(ThemeOverlay).Width(contentWidth).PaddingLeft(4).Render(rawLine))
		}
		
		for len(renderedLines) < maxLines {
			renderedLines = append(renderedLines, lipgloss.NewStyle().Background(ThemeOverlay).Width(contentWidth).Render(""))
		}
		
		bodyContent := strings.Join(renderedLines, "\n")
		paddedBody := lipgloss.NewStyle().Height(maxLines).Background(ThemeOverlay).Width(contentWidth).Render(bodyContent)
		
		footer := lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Background(ThemeMauve).Foreground(ThemeBase).Bold(true).Render(" ↑/↓/pgup/pgdown: scroll • esc/q/enter: close ")
		
		win := lipgloss.NewStyle().Background(ThemeOverlay).Width(w).Height(h).Render(headerFull + "\n" + paddedBody + "\n" + footer)
		return lipgloss.Place(m.terminalWidth, m.terminalHeight, lipgloss.Center, lipgloss.Center, win)
	}
		if m.state == StateVersion {
		w := int(float64(m.terminalWidth) * 0.60)
		if w < 70 { w = 70 }
		if w > m.terminalWidth { w = m.terminalWidth }
		
		h := int(float64(m.terminalHeight) * 0.80)
		if h < 20 { h = 20 }
		if h > m.terminalHeight { h = m.terminalHeight }
		
		headerFull := lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Background(ThemeMauve).Foreground(ThemeBase).Bold(true).Render(" VERSION & DEPENDENCIES ")
		
		fullText := version.GetVersionString() + "\n\n" + version.GetDepsString()
		bodyLines := strings.Split(fullText, "\n")
		contentWidth := w - 4
		maxLines := h - 2
		
		maxScroll := len(bodyLines) - maxLines
		if maxScroll < 0 { maxScroll = 0 }
		if m.versionScroll > maxScroll { m.versionScroll = maxScroll }
		
		var renderedLines []string
		endIdx := m.versionScroll + maxLines
		if endIdx > len(bodyLines) { endIdx = len(bodyLines) }
		
		for _, l := range bodyLines[m.versionScroll:endIdx] {
			rawLine := l
			if lipgloss.Width(rawLine) > contentWidth {
				rawLine = rawLine[:contentWidth-3] + "..."
			}
			renderedLines = append(renderedLines, lipgloss.NewStyle().Foreground(ThemeText).Background(ThemeOverlay).Width(contentWidth).PaddingLeft(4).Render(rawLine))
		}
		
		for len(renderedLines) < maxLines {
			renderedLines = append(renderedLines, lipgloss.NewStyle().Background(ThemeOverlay).Width(contentWidth).Render(""))
		}
		
		bodyContent := strings.Join(renderedLines, "\n")
		paddedBody := lipgloss.NewStyle().Height(maxLines).Background(ThemeOverlay).Width(contentWidth).Render(bodyContent)
		
		footer := lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Background(ThemeMauve).Foreground(ThemeBase).Bold(true).Render(" ↑/↓/pgup/pgdown: scroll • esc/q/enter: close ")
		
		win := lipgloss.NewStyle().Background(ThemeOverlay).Width(w).Height(h).Render(headerFull + "\n" + paddedBody + "\n" + footer)
		return lipgloss.Place(m.terminalWidth, m.terminalHeight, lipgloss.Center, lipgloss.Center, win)
	}
	if m.state == StateConflict {
		if len(m.conflictQueue) == 0 {
			return ""
		}
		t := m.conflictQueue[0]
		w := int(float64(m.terminalWidth) * 0.50)
		h := 14
		
		header := lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Background(ThemeRed).Foreground(ThemeBase).Bold(true).Render(" CONFLICT DETECTED ")
		
		adoIDStr := "???"
		if t.ADOID != nil {
			adoIDStr = fmt.Sprintf("%d", *t.ADOID)
		}
		
		body := fmt.Sprintf("\n  ADO #%s has been modified remotely.\n\n  Task: %s\n  Remote Revision: %d\n\n  Do you want to overwrite your local file with the remote changes?\n\n", adoIDStr, t.Title, t.ADORev)
		
		paddedBody := lipgloss.NewStyle().Height(h - 2).Background(ThemeOverlay).Width(w).Foreground(ThemeText).Render(body)
		footer := lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Background(ThemeRed).Foreground(ThemeBase).Bold(true).Render(" [O]verwrite Local • [K]eep Local • [esc] Abort ")
		
		win := lipgloss.NewStyle().Background(ThemeOverlay).Width(w).Height(h).Render(header + "\n" + paddedBody + "\n" + footer)
		return lipgloss.Place(m.terminalWidth, m.terminalHeight, lipgloss.Center, lipgloss.Center, win)
	}

	return m.list.View()
}

func RunApp(app *core.App) error {
	p := tea.NewProgram(NewMainModel(app), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func (m *MainModel) generateStandup() {
	tasks, _ := m.coreApp.Store.ListTasks("")
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterdayStart := todayStart.AddDate(0, 0, -1)

	var active []*core.Task
	var blocked []*core.Task
	var finished []*core.Task

	for _, t := range tasks {
		status := strings.ToLower(string(t.Status))
		if status == "active" || status == "in progress" || status == "doing" {
			active = append(active, t)
		} else if status == "blocked" || status == "paused" || status == "on hold" || status == "waiting" {
			blocked = append(blocked, t)
		} else if status == "closed" || status == "done" || status == "resolved" || status == "completed" {
			if !t.UpdatedAt.Before(yesterdayStart) {
				finished = append(finished, t)
	}
		}
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# 🌅 Standup - %s\n\n", now.Format("2006-01-02")))

	sb.WriteString("## 🏃 Active\n")
	if len(active) == 0 {
		sb.WriteString("- *No active items*\n")
	} else {
		for _, t := range active {
			adoTag := ""
			if t.ADOID != nil {
				adoTag = fmt.Sprintf("[ADO-%d] ", *t.ADOID)
			}
			timeStr := ""
			if t.TotalSeconds > 0 {
				timeStr = fmt.Sprintf(" *(%.1fh tracked)*", float64(t.TotalSeconds)/3600.0)
			}
			sb.WriteString(fmt.Sprintf("- **%s%s**%s\n", adoTag, t.Title, timeStr))
		}
	}
	sb.WriteString("\n")

	sb.WriteString("## 🛑 Blocked / Paused\n")
	if len(blocked) == 0 {
		sb.WriteString("- *No blocked items*\n")
	} else {
		for _, t := range blocked {
			adoTag := ""
			if t.ADOID != nil {
				adoTag = fmt.Sprintf("[ADO-%d] ", *t.ADOID)
			}
			sb.WriteString(fmt.Sprintf("- **%s%s**\n", adoTag, t.Title))
		}
	}
	sb.WriteString("\n")

	sb.WriteString("## ✅ Finished (Today & Yesterday)\n")
	if len(finished) == 0 {
		sb.WriteString("- *No finished items*\n")
	} else {
		for _, t := range finished {
			adoTag := ""
			if t.ADOID != nil {
				adoTag = fmt.Sprintf("[ADO-%d] ", *t.ADOID)
			}
			sb.WriteString(fmt.Sprintf("- **%s%s**\n", adoTag, t.Title))
		}
	}

	reportsDir := ""
	if home, err := os.UserHomeDir(); err == nil {
		reportsDir = filepath.Join(home, ".local", "share", "tally", "reports")
		os.MkdirAll(reportsDir, 0755)
	}
	outFile := filepath.Join(reportsDir, fmt.Sprintf("standup_%s.md", now.Format("2006-01-02")))
	os.WriteFile(outFile, []byte(sb.String()), 0644)
	
	m.standupText = sb.String()
}

func listenToLogs(sub chan string) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-sub
		if !ok {
			return nil
		}
		return SyncLogMsg(msg)
	}
}
func (m *MainModel) startSyncProcess(jobType string, targetID string) tea.Cmd {
	return func() tea.Msg {
		cfg, _ := config.Load()
		adoPat := os.Getenv("TALLY_ADO_PAT")
		spToken := os.Getenv("TALLY_7PACE_TOKEN")
		if spToken == "" {
			spToken = adoPat
		}
		
		if adoPat == "" {
			m.logChannel <- "[ERROR] TALLY_ADO_PAT environment variable is missing!"
			time.Sleep(200 * time.Millisecond)
			return SyncFinishedMsg{Err: fmt.Errorf("missing ADO PAT")}
		}
		var err error

		var conflicts []*core.Task
		switch jobType {
		case "sync":
			err = m.coreApp.Sync(cfg, adoPat, spToken, m.logChannel)
		case "fetch":
			conflicts, err = m.coreApp.Fetch(cfg, adoPat, spToken, m.logChannel)
		case "push_single":
			conflicts, err = m.coreApp.SyncSingle(cfg, adoPat, spToken, targetID, m.logChannel)
		}

		if err != nil {
			m.logChannel <- "[ERROR] " + err.Error()
		} else {
			m.logChannel <- "✅ Complete!"
		}
		
		time.Sleep(200 * time.Millisecond) // let the UI drain the channel
		return SyncFinishedMsg{Err: err, Conflicts: conflicts}
	}
}
