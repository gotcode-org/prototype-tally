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

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FieldType int

const (
	FieldText FieldType = iota
	FieldTextArea
	FieldSelector
	FieldBoolean
	FieldButton
)

type Field struct {
	Type     FieldType
	Name     string
	Label    string
	Help     string

	// Models for text
	TextInput textinput.Model
	TextArea  textarea.Model

	// State for selectors
	Options  []string
	Selected int

	// Action for buttons
	ButtonBgColor lipgloss.Color
	ButtonFgColor lipgloss.Color
	Action        func(form *FormModel) tea.Cmd
}

type FormModel struct {
	Title         string
	Fields        []*Field
	FocusIndex    int
	Width         int
	Height        int
	terminalWidth int
	terminalHeight int
	WidthPct      float64
	HeightPct     float64
	Quitting      bool
	Submitted     bool
}

func NewForm(title string) *FormModel {
	return &FormModel{
		Title:      title,
		WidthPct:   0.50,
		HeightPct:  0.60,
	}
}

func (f *FormModel) AddTextBox(name, label, placeholder, help string) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = "  "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ThemeBlue).Background(ThemeBase)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ThemeText).Background(ThemeBase)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(ThemeMauve).Background(ThemeBase)
	f.Fields = append(f.Fields, &Field{Type: FieldText, Name: name, Label: label, Help: help, TextInput: ti})
}

func (f *FormModel) AddTextArea(name, label, placeholder, help string) {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.ShowLineNumbers = false
	ta.SetHeight(3)
	ta.Prompt = "  "
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(ThemeText).Background(ThemeBase)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(ThemeBase)
	ta.FocusedStyle.Text = lipgloss.NewStyle().Background(ThemeBase)
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Background(ThemeBase)
	ta.Cursor.Style = lipgloss.NewStyle().Foreground(ThemeMauve).Background(ThemeBase)
	ta.BlurredStyle.Base = lipgloss.NewStyle().Foreground(ThemeSubtext).Background(ThemeBase)
	f.Fields = append(f.Fields, &Field{Type: FieldTextArea, Name: name, Label: label, Help: help, TextArea: ta})
}

func (f *FormModel) AddSelector(name, label string, options []string, help string) {
	f.Fields = append(f.Fields, &Field{Type: FieldSelector, Name: name, Label: label, Help: help, Options: options})
}

func (f *FormModel) AddBoolean(name, label, help string) {
	f.Fields = append(f.Fields, &Field{Type: FieldBoolean, Name: name, Label: label, Help: help, Options: []string{"True", "False"}})
}

func (f *FormModel) AddButton(label string, bgColor, fgColor lipgloss.Color, action func(form *FormModel) tea.Cmd) {
	f.Fields = append(f.Fields, &Field{Type: FieldButton, Name: label, Label: label, ButtonBgColor: bgColor, ButtonFgColor: fgColor, Action: action})
}

func (f *FormModel) GetString(name string) string {
	for _, field := range f.Fields {
		if field.Name == name {
			if field.Type == FieldText {
				return field.TextInput.Value()
			}
			if field.Type == FieldTextArea {
				return field.TextArea.Value()
			}
			if field.Type == FieldSelector || field.Type == FieldBoolean {
				return field.Options[field.Selected]
			}
		}
	}
	return ""
}

func (f *FormModel) updateFocus() {
	for i, field := range f.Fields {
		if field.Type == FieldText {
			if i == f.FocusIndex {
				field.TextInput.Focus()
			} else {
				field.TextInput.Blur()
			}
		} else if field.Type == FieldTextArea {
			if i == f.FocusIndex {
				field.TextArea.Focus()
			} else {
				field.TextArea.Blur()
			}
		}
	}
}

func (f *FormModel) Init() tea.Cmd {
	f.updateFocus()
	return nil
}

func (f *FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.terminalWidth = msg.Width
		f.terminalHeight = msg.Height
		f.Width = int(float64(msg.Width) * f.WidthPct)
		if f.Width < 60 {
			f.Width = 60
		}
		f.Height = int(float64(msg.Height) * f.HeightPct)
		
		for _, field := range f.Fields {
			if field.Type == FieldTextArea {
				field.TextArea.SetWidth(f.Width - 10)
			}
		}
		return f, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			f.Quitting = true
			return f, tea.Quit
		case "tab":
			f.FocusIndex = (f.FocusIndex + 1) % len(f.Fields)
			f.updateFocus()
			return f, nil
		case "shift+tab":
			f.FocusIndex = (f.FocusIndex - 1 + len(f.Fields)) % len(f.Fields)
			f.updateFocus()
			return f, nil
		case "enter":
			if f.Fields[f.FocusIndex].Type == FieldButton {
				return f, f.Fields[f.FocusIndex].Action(f)
			}
		case "left", "h":
			field := f.Fields[f.FocusIndex]
			if (field.Type == FieldSelector || field.Type == FieldBoolean) && field.Selected > 0 {
				field.Selected--
				return f, nil
			}
		case "right", "l":
			field := f.Fields[f.FocusIndex]
			if (field.Type == FieldSelector || field.Type == FieldBoolean) && field.Selected < len(field.Options)-1 {
				field.Selected++
				return f, nil
			}
		}
	}

	if len(f.Fields) > 0 {
		field := f.Fields[f.FocusIndex]
		if field.Type == FieldText {
			var cmd tea.Cmd
			field.TextInput, cmd = field.TextInput.Update(msg)
			return f, cmd
		} else if field.Type == FieldTextArea {
			var cmd tea.Cmd
			field.TextArea, cmd = field.TextArea.Update(msg)
			return f, cmd
		}
	}

	return f, nil
}

func (f *FormModel) View() string {
	if f.Quitting || f.Submitted {
		return ""
	}

	var sections []string
	headerFull := lipgloss.NewStyle().Width(f.Width).Align(lipgloss.Center).Background(ThemeMauve).Foreground(ThemeBase).Bold(true).Render(" " + strings.ToUpper(f.Title) + " ")
	sections = append(sections, headerFull)

	contentWidth := f.Width - 4

	var buttonViews []string

	for i, field := range f.Fields {
		focused := i == f.FocusIndex
		
		if field.Type == FieldButton {
			var view string
			if focused {
				view = lipgloss.NewStyle().Foreground(field.ButtonFgColor).Background(field.ButtonBgColor).Padding(0, 2).Bold(true).Render("▶ " + field.Label + " ◀")
			} else {
				view = lipgloss.NewStyle().Foreground(field.ButtonFgColor).Background(field.ButtonBgColor).Padding(0, 3).Render(field.Label)
			}
			buttonViews = append(buttonViews, view)
			continue
		}

		var lbl string
		if focused {
			lbl = lipgloss.NewStyle().Foreground(ThemeGreen).Background(ThemeOverlay).Bold(true).Width(contentWidth).Render("▶ " + field.Label)
		} else {
			lbl = lipgloss.NewStyle().Foreground(ThemeText).Background(ThemeOverlay).Bold(true).Width(contentWidth).Render("  " + field.Label)
		}

		var view string
		switch field.Type {
		case FieldText:
			tiView := field.TextInput.View()
			visibleLen := lipgloss.Width(tiView)
			targetWidth := contentWidth - 10
			
			var padded string
			if visibleLen < targetWidth {
				spaces := lipgloss.NewStyle().Background(ThemeBase).Render(strings.Repeat(" ", targetWidth - visibleLen))
				padded = tiView + spaces
			} else {
				padded = tiView
			}
			
			// Give it an inset border and a dark inner background
			inner := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ThemeSubtext).
				BorderBackground(ThemeOverlay).
				Background(ThemeBase).
				Padding(0, 1).
				Render(padded)
				
			view = lipgloss.NewStyle().Background(ThemeOverlay).Width(contentWidth).PaddingLeft(4).Align(lipgloss.Left).Render(inner)
		case FieldTextArea:
			taView := field.TextArea.View()
			
			// Text areas naturally handle their width/padding internally via Base style, 
			// so we just wrap it in a border.
			inner := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ThemeSubtext).
				BorderBackground(ThemeOverlay).
				Background(ThemeBase).
				Padding(0, 1).
				Render(taView)
				
			view = lipgloss.NewStyle().Background(ThemeOverlay).Width(contentWidth).PaddingLeft(4).Align(lipgloss.Left).Render(inner)
		case FieldSelector, FieldBoolean:
			statusStr := ""
			for j, opt := range field.Options {
				prefix := "○"
				if j == field.Selected {
					prefix = "⊙"
				}
				if focused && j == field.Selected {
					statusStr += lipgloss.NewStyle().Foreground(ThemeBase).Background(ThemeGreen).Bold(true).Render(fmt.Sprintf(" %s %s ", prefix, opt))
				} else {
					statusStr += lipgloss.NewStyle().Background(ThemeOverlay).Render(fmt.Sprintf(" %s %s ", prefix, opt))
				}
			}
			view = lipgloss.NewStyle().Background(ThemeOverlay).Width(contentWidth).PaddingLeft(4).Align(lipgloss.Left).Render(statusStr)
		}

		if lbl != "" {
			sections = append(sections, lbl)
		}
		
		sections = append(sections, view)
		
		if field.Help != "" {
			sections = append(sections, lipgloss.NewStyle().Foreground(ThemeSubtext).Background(ThemeOverlay).Italic(true).Width(contentWidth).Render("    "+field.Help))
		}
		
		sections = append(sections, lipgloss.NewStyle().Background(ThemeOverlay).Width(contentWidth).Render("")) // Spacing between fields
	}

	if len(buttonViews) > 0 {
		buttonsStr := strings.Join(buttonViews, lipgloss.NewStyle().Background(ThemeOverlay).Render("   "))
		// Align buttons to the center
		sections = append(sections, lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center).Background(ThemeOverlay).Render(buttonsStr))
	}

	sections = append(sections, lipgloss.NewStyle().Background(ThemeOverlay).Width(contentWidth).Render("")) // Spacing before footer

		helpText := " tab/shift+tab: move • ←/→: select • enter: submit • esc: quit "
	footer := lipgloss.NewStyle().Width(f.Width).Align(lipgloss.Center).Background(ThemeMauve).Foreground(ThemeBase).Bold(true).Render(helpText)

	// Combine body sections and force height so the footer is pushed exactly to the bottom
	bodyContent := strings.Join(sections[1:], "\n")
	paddedBody := lipgloss.NewStyle().Height(f.Height - 2).Background(ThemeOverlay).Width(contentWidth).Render(bodyContent)

	formContent := sections[0] + "\n" + paddedBody + "\n" + footer
	
	win := lipgloss.NewStyle().Background(ThemeOverlay).Width(f.Width).Height(f.Height).Render(formContent)
	return lipgloss.Place(f.terminalWidth, f.terminalHeight, lipgloss.Center, lipgloss.Center, win)
}
