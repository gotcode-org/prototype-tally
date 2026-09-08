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
	"strings"
	"gotcode.org/tally/internal/core"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	HeaderStyle = lipgloss.NewStyle().Foreground(ThemeBase).Background(ThemeMauve).Bold(true)
	MilestoneRowStyle = lipgloss.NewStyle().Foreground(ThemeText)
	StoryRowStyle = lipgloss.NewStyle().Foreground(ThemeText)
	TaskRowStyle = lipgloss.NewStyle().Foreground(ThemeOverlay)
	ActiveRowStyle = lipgloss.NewStyle().Foreground(ThemeText).Background(ThemeOverlay).Bold(true)
)

type CreateNewTaskMsg struct{}
type EditTaskMsg struct{ ID string }
type EditorFinishedMsg struct{ Err error }
type SyncTasksMsg struct{}
type PullTasksMsg struct{}
type PushSingleTaskMsg struct{ ID string }
type CycleThemeMsg struct{}
type GenerateStandupMsg struct{}

type RefreshMsg struct{}
type SyncFinishedMsg struct{ Err error; Conflicts []*core.Task }

type ListItem struct {
	ID           string
	Title        string
	Type         string
	Status       string
	Progress     float64
	ProgressText string
	TimeText     string
	TypeColor    lipgloss.Color
	Expanded     bool
	Children     []*ListItem
	
	CachedWidth  int
	CacheNormal  []string
	CacheActive  []string
}

type FlatRow struct {
	Depth       int
	Expanded    bool
	HasChildren bool
	Item        *ListItem
}

type ListModel struct {
	title          string
	TimeStats      string
	items          []*ListItem
	cachedFlatRows []FlatRow
	cursor         int
	offset         int
	terminalWidth  int
	terminalHeight int
	widthPct       float64
	heightPct      float64
	confirmingID   string
	confirmingTitle string
}

func NewListModel(title string, items []*ListItem) ListModel {
	m := ListModel{
		title:     title,
		items:     items,
		cursor:    0,
		widthPct:  0.95,
		heightPct: 0.95,
	}
	m.rebuildFlatRows()
	return m
}

func (m ListModel) Init() tea.Cmd {
	return nil
}

func (m *ListModel) rebuildFlatRows() {
	var rows []FlatRow
	var flatten func(items []*ListItem, depth int)
	flatten = func(items []*ListItem, depth int) {
		for _, item := range items {
			hasChildren := len(item.Children) > 0
			rows = append(rows, FlatRow{
				Depth:       depth,
				Expanded:    item.Expanded,
				HasChildren: hasChildren,
				Item:        item,
			})
			if item.Expanded && hasChildren {
				flatten(item.Children, depth+1)
			}
		}
	}
	flatten(m.items, 0)
	m.cachedFlatRows = rows
}

func (m *ListModel) getFlatRows() []FlatRow {
	return m.cachedFlatRows
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.confirmingID != "" {
			if msg.String() == "y" || msg.String() == "Y" {
				id := m.confirmingID
				m.confirmingID = ""
				m.confirmingTitle = ""
				return m, func() tea.Msg { return DeleteTaskMsg{ID: id} }
			} else if msg.String() == "n" || msg.String() == "N" || msg.String() == "esc" {
				m.confirmingID = ""
				m.confirmingTitle = ""
				return m, nil
			} else if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "n":
			return m, func() tea.Msg { return CreateNewTaskMsg{} }
		case "c":
			flatRows := m.getFlatRows()
			if len(flatRows) > 0 {
				row := flatRows[m.cursor]
				if row.Item.ID != "" && !strings.HasPrefix(row.Item.ID, "backlog-") && !strings.HasPrefix(row.Item.ID, "recur-") {
					return m, func() tea.Msg { return CreateSubtaskMsg{ParentID: row.Item.ID} }
				}
			}
		case "v":
			return m, func() tea.Msg { return ShowVersionMsg{} }
		case "t":
			return m, func() tea.Msg { return CycleThemeMsg{} }
		case "r", "R":
			return m, func() tea.Msg { return GenerateStandupMsg{} }
		case "s":
			return m, func() tea.Msg { return SyncTasksMsg{} }
		case "p":
			return m, func() tea.Msg { return PullTasksMsg{} }
		case "u":
			flatRows := m.getFlatRows()
			if len(flatRows) > 0 {
				row := flatRows[m.cursor]
				if row.Item.ID != "" {
					return m, func() tea.Msg { return PushSingleTaskMsg{ID: row.Item.ID} }
				}
			}
		case "e", "enter":
			flatRows := m.getFlatRows()
			if len(flatRows) > 0 {
				row := flatRows[m.cursor]
				if row.Item.ID != "" {
					return m, func() tea.Msg { return EditTaskMsg{ID: row.Item.ID} }
				}
			}
		case "a":
			flatRows := m.getFlatRows()
			if len(flatRows) > 0 {
				row := flatRows[m.cursor]
				if row.Item.ID != "" {
					return m, func() tea.Msg { return LogTimeMsg{ID: row.Item.ID} }
				}
			}
		case "x", "delete":
			flatRows := m.getFlatRows()
			if len(flatRows) > 0 {
				row := flatRows[m.cursor]
				if row.Item.ID != "" {
					m.confirmingID = row.Item.ID
					m.confirmingTitle = row.Item.Title
					return m, nil
				}
			}
		case "m":
			flatRows := m.getFlatRows()
			if len(flatRows) > 0 {
				row := flatRows[m.cursor]
				if row.Item.ID != "" && strings.HasPrefix(row.Item.ID, "backlog-") {
					return m, func() tea.Msg { return StartTaskMsg{ID: row.Item.ID} }
				}
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.getFlatRows())-1 {
				m.cursor++
			}
		case "pgup":
			visibleRows := m.terminalHeight - 8
			if visibleRows < 1 {
				visibleRows = 1
			}
			m.cursor -= visibleRows
			if m.cursor < 0 {
				m.cursor = 0
			}
		case "pgdown":
			visibleRows := m.terminalHeight - 8
			if visibleRows < 1 {
				visibleRows = 1
			}
			flatRows := m.getFlatRows()
			m.cursor += visibleRows
			if len(flatRows) > 0 && m.cursor >= len(flatRows) {
				m.cursor = len(flatRows) - 1
			}
		case " ":
			flatRows := m.getFlatRows()
			if len(flatRows) > 0 {
				row := flatRows[m.cursor]
				row.Item.Expanded = !row.Item.Expanded
				m.rebuildFlatRows()
			}
		}
	}
	return m, nil
}

func (m ListModel) renderHeader(widths []int) string {
	cells := []string{}
	headers := []string{"NAME", "ID", "TYPE", "STATUS", "TIME"}

	for i, col := range headers {
		w := widths[i]
		var styled string
		if len(col) > w {
			styled = HeaderStyle.Render(col[:w-3] + "...")
		} else {
			styled = HeaderStyle.Render(col) + lipgloss.NewStyle().Background(ThemeMauve).Render(strings.Repeat(" ", w-len(col)))
		}
		cells = append(cells, styled)
	}

	prefix := lipgloss.NewStyle().Background(ThemeMauve).Render("  ")
	separator := lipgloss.NewStyle().Foreground(ThemeBase).Background(ThemeMauve).Render(" │ ") // Keep dark on Mauve
	middle := prefix + strings.Join(cells, separator)

	var parts []string
	for _, w := range widths {
		parts = append(parts, strings.Repeat("─", w))
	}
	borderStyle := lipgloss.NewStyle().Foreground(ThemeBase).Background(ThemeMauve)
	
	topBorder := borderStyle.Render("──" + strings.Join(parts, "─┬─"))
	botBorder := borderStyle.Render("──" + strings.Join(parts, "─┼─"))

	return topBorder + "\n" + middle + "\n" + botBorder
}

func formatStatusCell(status string, width int, isCursor bool, bgStyle lipgloss.Style) string {
	if status == "" {
		return ""
	}
	var fg lipgloss.Color

	switch status {
	case "ACTIVE", "In Progress":
		fg = ThemeBlue
	case "COMPLETED", "Done":
		fg = ThemeGreen
	case "CANCELLED":
		fg = ThemeRed
	case "BACKLOG", "Todo":
		fg = ThemeSubtext
	default:
		fg = ThemeSubtext
	}

	if isCursor {
		fg = ThemeBase
	}

	badge := lipgloss.NewStyle().Foreground(fg).Background(bgStyle.GetBackground()).Bold(true).Render(strings.ToUpper(status))
	visibleLen := lipgloss.Width(badge)

	if visibleLen > width {
		return badge[:width]
	}

	return badge
}

func renderProgressBar(pct float64, text string, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}

	filledLen := int(float64(width) * pct)
	emptyLen := width - filledLen

	filled := lipgloss.NewStyle().Foreground(ThemeGreen).Render(strings.Repeat("█", filledLen))
	empty := lipgloss.NewStyle().Foreground(ThemeOverlay).Render(strings.Repeat("░", emptyLen))

	bar := filled + empty
	
	if text != "" {
		textStyle := lipgloss.NewStyle().Foreground(ThemeOverlay).Italic(true)
		return fmt.Sprintf("%s %s", bar, textStyle.Render(text))
	}
	return bar
}

func getRowTitle(row FlatRow) string {
	icon := ""
	if row.HasChildren {
		if row.Expanded {
			icon = "▼ "
		} else {
			icon = "▶ "
		}
	} else {
		icon = "  "
	}
	indent := strings.Repeat("  ", row.Depth)
	return indent + icon + row.Item.Title
}

func renderRow(title, idStr, typeStr, statusCell, progress string, widths []int, isCursor bool, rowStyle lipgloss.Style) []string {
	formatColLeftRaw := func(txt string, w int) []string {
		runes := []rune(txt)
		if len(runes) <= w {
			spaces := strings.Repeat(" ", w-len(runes))
			return []string{rowStyle.Render(txt + spaces)}
		}
		
		var lines []string
		indentStr := ""
		// Find where the actual text starts by skipping leading spaces and tree icons
		textStartIndex := 0
		runes_txt := []rune(txt)
		for i, r := range runes_txt {
			if r != ' ' && r != '▼' && r != '▶' && r != '↻' {
				textStartIndex = i
				break
			}
		}
		
		// If there is an icon, we usually want to indent past it + 1 space
		if textStartIndex > 0 {
			indentStr = strings.Repeat(" ", textStartIndex)
		} else {
			indentStr = "    "
		}
		
		firstLineMax := w
		subsequentLineMax := w - len(indentStr)
		if subsequentLineMax < 10 {
			return []string{rowStyle.Render(string(runes[:w-3]) + "...")}
		}

		// Helper to find word break
		findBreak := func(r []rune, max int) int {
			if len(r) <= max {
				return len(r)
			}
			// Look for space from right to left
			for i := max; i >= 0; i-- {
				if r[i] == ' ' {
					return i
				}
			}
			// If no space found, hard break
			return max
		}

		// First line
		breakIdx := findBreak(runes, firstLineMax)
		chunk := string(runes[:breakIdx])
		pad := w - lipgloss.Width(chunk)
		if pad < 0 { pad = 0 }
		lines = append(lines, rowStyle.Render(chunk + strings.Repeat(" ", pad)))
		
		if breakIdx < len(runes) && runes[breakIdx] == ' ' {
			runes = runes[breakIdx+1:]
		} else {
			runes = runes[breakIdx:]
		}
		
		// Subsequent lines
		for len(runes) > 0 {
			breakIdx := findBreak(runes, subsequentLineMax)
			chunk := string(runes[:breakIdx])
			pad := subsequentLineMax - lipgloss.Width(chunk)
			if pad < 0 { pad = 0 }
			lines = append(lines, rowStyle.Render(indentStr + chunk + strings.Repeat(" ", pad)))
			
			if breakIdx < len(runes) && runes[breakIdx] == ' ' {
				runes = runes[breakIdx+1:]
			} else {
				runes = runes[breakIdx:]
			}
		}
		
		return lines
	}
	
	formatColLeftANSI := func(txt string, w int) string {
		visibleLen := lipgloss.Width(txt)
		if visibleLen > w {
			// This is technically unsafe for ANSI, but we only use it for short strings like TimeText
			return rowStyle.Render(txt[:w-3] + "...")
		}
		spaces := strings.Repeat(" ", w-visibleLen)
		return txt + rowStyle.Render(spaces)
	}
	
	formatColCenter := func(txt string, w int) string {
		visibleLen := lipgloss.Width(txt)
		if visibleLen > w {
			return rowStyle.Render(txt[:w-3] + "...")
		}
		if txt == "" {
			return rowStyle.Render(strings.Repeat(" ", w))
		}
		leftPad := (w - visibleLen) / 2
		rightPad := w - visibleLen - leftPad
		return rowStyle.Render(strings.Repeat(" ", leftPad)) + txt + rowStyle.Render(strings.Repeat(" ", rightPad))
	}

	if isCursor {
		rowStyle = ActiveRowStyle
	}

	c0Lines := formatColLeftRaw(title, widths[0])
	c1Cell := formatColLeftRaw(idStr, widths[1])[0]
	c2Cell := formatColCenter(typeStr, widths[2])
	c3Cell := formatColCenter(statusCell, widths[3])
	c4Cell := formatColLeftANSI(progress, widths[4])

	prefix := "  "
	if isCursor {
		prefix = lipgloss.NewStyle().Foreground(ThemeBase).Background(ThemeBlue).Bold(true).Render("▶ ")
	}
	prefix = rowStyle.Render(prefix)
	
	emptyPrefix := rowStyle.Render("  ")
	separator := lipgloss.NewStyle().Foreground(ThemeSubtext).Background(rowStyle.GetBackground()).Render(" │ ")
	
	var result []string
	
	maxLines := len(c0Lines)
	for i := 0; i < maxLines; i++ {
		c0Line := c0Lines[i]
		
		pref := emptyPrefix
		if i == 0 {
			pref = prefix
		}
		
		c1Line := formatColLeftRaw("", widths[1])[0]
		c2Line := formatColCenter("", widths[2])
		c3Line := formatColCenter("", widths[3])
		c4Line := formatColLeftANSI("", widths[4])
		
		if i == 0 {
			c1Line = c1Cell
			c2Line = c2Cell
			c3Line = c3Cell
			c4Line = c4Cell
		}
		
		line := pref + c0Line + separator + c1Line + separator + c2Line + separator + c3Line + separator + c4Line
		result = append(result, line)
	}

	return result
}

func (m ListModel) View() string {
	if m.terminalWidth == 0 || m.terminalHeight == 0 {
		return ""
	}

	targetWidth := int(float64(m.terminalWidth) * m.widthPct)
	targetHeight := int(float64(m.terminalHeight) * m.heightPct)

	windowStyle := lipgloss.NewStyle().
		Border(lipgloss.HiddenBorder()).
		Height(targetHeight)

	headerWidth := targetWidth
	
	// Subtract the width of prefix (2) and separators (3 * 3 = 9) = 11 total static chars
	availableWidth := headerWidth - 14 // 4 separators = 12 chars + 2 prefix = 14
	if availableWidth < 30 {
		availableWidth = 30
	}
	
	// Proportional widths so the grid stays balanced across different monitor sizes
	wName := int(float64(availableWidth) * 0.50)
	wID := int(float64(availableWidth) * 0.17)
	wType := int(float64(availableWidth) * 0.10)
	wStatus := int(float64(availableWidth) * 0.15)
	wTime := availableWidth - (wName + wID + wType + wStatus) // Absorbs the remaining ~8%

	widths := []int{wName, wID, wType, wStatus, wTime}

	flatRows := m.getFlatRows()
	var allLines []string
	
	visibleRows := targetHeight - 8
	
	// Windowing logic: we need to figure out which lines to show based on the cursor
	cursorStartLine := 0
	
	for i, row := range flatRows {
		isCursor := i == m.cursor
		
		if isCursor {
			cursorStartLine = len(allLines)
		}
		
		// CPU Optimization: Skip heavy lipgloss rendering for items far off-screen
		if i < m.cursor - (visibleRows * 2) || i > m.cursor + (visibleRows * 2) {
			allLines = append(allLines, "")
			continue
		}
		
		if row.Item.CachedWidth != availableWidth || len(row.Item.CacheNormal) == 0 {
			titleCell := getRowTitle(row)
			rowStyle := MilestoneRowStyle
			if row.Depth > 0 {
				rowStyle = StoryRowStyle
			}

			computeRender := func(active bool) []string {
				bgStyle := rowStyle
				if active {
					bgStyle = ActiveRowStyle
				}
				
				var typeCell string
				if row.Item.Type != "" {
					tColor := row.Item.TypeColor
					if active {
						tColor = ThemeBase
					}
					typeCell = lipgloss.NewStyle().Foreground(tColor).Background(bgStyle.GetBackground()).Bold(true).Render(strings.ToUpper(row.Item.Type))
				}
				
				statusCell := formatStatusCell(row.Item.Status, widths[2], active, bgStyle)

				var progressCell string
				if row.Item.TimeText != "" {
					pColor := ThemeSubtext
					if active {
						pColor = ThemeBase
					}
					progressCell = lipgloss.NewStyle().Foreground(pColor).Background(bgStyle.GetBackground()).Render(row.Item.TimeText)
				} else {
					progressCell = lipgloss.NewStyle().Foreground(ThemeSubtext).Background(bgStyle.GetBackground()).Render("-")
				}

				idCell := row.Item.ID
				if idCell == "" {
					idCell = " "
				}
				return renderRow(titleCell, idCell, typeCell, statusCell, progressCell, widths, active, rowStyle)
			}
			
			row.Item.CacheNormal = computeRender(false)
			row.Item.CacheActive = computeRender(true)
			row.Item.CachedWidth = availableWidth
		}
		
		if isCursor {
			allLines = append(allLines, row.Item.CacheActive...)
		} else {
			allLines = append(allLines, row.Item.CacheNormal...)
		}
	}

	// Calculate scroll offset
	if cursorStartLine < m.offset {
		m.offset = cursorStartLine
	} else if cursorStartLine >= m.offset + visibleRows {
		m.offset = cursorStartLine - visibleRows + 1
	}

	var listSections []string
	if len(allLines) > 0 {
		end := m.offset + visibleRows
		if end > len(allLines) {
			end = len(allLines)
		}
		if m.offset < len(allLines) {
			listSections = allLines[m.offset:end]
		}
	}

	for len(listSections) < visibleRows {
		listSections = append(listSections, renderRow("", "", "", "", "", widths, false, lipgloss.NewStyle().Foreground(ThemeBase).Background(ThemeOverlay))[0])
	}
	


	paddedList := lipgloss.NewStyle().
		Background(ThemeOverlay).
		Width(headerWidth).
		Height(visibleRows).
		Render(strings.Join(listSections, "\n"))

	cursorIndicator := ""
	if len(flatRows) > 0 {
		cursorIndicator = fmt.Sprintf(" [%d/%d]", m.cursor+1, len(flatRows))
	}
	headerFull := lipgloss.NewStyle().Width(headerWidth).Align(lipgloss.Center).Background(ThemeMauve).Foreground(ThemeBase).Bold(true).Render(" " + strings.ToUpper(m.title) + cursorIndicator + " ")
	headerRow := lipgloss.NewStyle().Background(ThemeMauve).Render(m.renderHeader(widths))
	
	themeDisplay := GetCurrentThemeName()
	helpStr := " ↑/↓: move • space: expand • enter/e: edit • n: new • a: add time • m: start (move) • x: delete • r: report • t: theme (" + themeDisplay + ") • esc: quit "
	
	footerContent := helpStr
	if m.TimeStats != "" {
		footerContent = m.TimeStats + "\n" + helpStr
	}
	
	if m.confirmingID != "" {
		footerContent = fmt.Sprintf("⚠ ARE YOU SURE YOU WANT TO PERMANENTLY DELETE '%s'? (y/N)", strings.ToUpper(m.confirmingTitle))
	}
	
	footer := lipgloss.NewStyle().Width(headerWidth).Align(lipgloss.Center).Background(ThemeMauve).Foreground(ThemeBase).Bold(true).Render(footerContent)
	
	finalContent := headerFull + "\n" + headerRow + "\n" + paddedList + "\n" + footer

	activeWindow := windowStyle.Copy().Width(targetWidth)
	result := activeWindow.Render(finalContent)

	return lipgloss.Place(m.terminalWidth, m.terminalHeight, lipgloss.Center, lipgloss.Top, result)
}

func RunList(title string, items []*ListItem) error {
	m := NewListModel(title, items)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
