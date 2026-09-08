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

import "github.com/charmbracelet/lipgloss"

type ThemePalette struct {
	Name    string
	Base    lipgloss.Color
	Text    lipgloss.Color
	Subtext lipgloss.Color
	Overlay lipgloss.Color
	Blue    lipgloss.Color
	Green   lipgloss.Color
	Red     lipgloss.Color
	Mauve   lipgloss.Color
	Peach   lipgloss.Color
}

var (
	CatppuccinMocha = ThemePalette{
		Name: "Catppuccin Mocha",
		Base:    lipgloss.Color("#1e1e2e"),
		Text:    lipgloss.Color("#cdd6f4"),
		Subtext: lipgloss.Color("#a6adc8"),
		Overlay: lipgloss.Color("#6c7086"),
		Blue:    lipgloss.Color("#89b4fa"),
		Green:   lipgloss.Color("#a6e3a1"),
		Red:     lipgloss.Color("#f38ba8"),
		Mauve:   lipgloss.Color("#cba6f7"),
		Peach:   lipgloss.Color("#fab387"),
	}

	Dracula = ThemePalette{
		Name: "Dracula",
		Base:    lipgloss.Color("#282a36"),
		Text:    lipgloss.Color("#f8f8f2"),
		Subtext: lipgloss.Color("#6272a4"),
		Overlay: lipgloss.Color("#44475a"),
		Blue:    lipgloss.Color("#8be9fd"),
		Green:   lipgloss.Color("#50fa7b"),
		Red:     lipgloss.Color("#ff5555"),
		Mauve:   lipgloss.Color("#ff79c6"), // Dracula Pink
		Peach:   lipgloss.Color("#ffb86c"),
	}

	Nord = ThemePalette{
		Name: "Nord",
		Base:    lipgloss.Color("#2e3440"),
		Text:    lipgloss.Color("#eceff4"),
		Subtext: lipgloss.Color("#d8dee9"),
		Overlay: lipgloss.Color("#4c566a"),
		Blue:    lipgloss.Color("#81a1c1"),
		Green:   lipgloss.Color("#a3be8c"),
		Red:     lipgloss.Color("#bf616a"),
		Mauve:   lipgloss.Color("#88c0d0"), // Nord Frost Blue
		Peach:   lipgloss.Color("#ebcb8b"),
	}
	Gruvbox = ThemePalette{
		Name: "Gruvbox",
		Base:    lipgloss.Color("#282828"),
		Text:    lipgloss.Color("#ebdbb2"),
		Subtext: lipgloss.Color("#a89984"),
		Overlay: lipgloss.Color("#504945"),
		Blue:    lipgloss.Color("#83a598"),
		Green:   lipgloss.Color("#b8bb26"),
		Red:     lipgloss.Color("#fb4934"),
		Mauve:   lipgloss.Color("#d3869b"),
		Peach:   lipgloss.Color("#fe8019"),
	}

	TokyoNight = ThemePalette{
		Name: "Tokyo Night",
		Base:    lipgloss.Color("#1a1b26"),
		Text:    lipgloss.Color("#c0caf5"),
		Subtext: lipgloss.Color("#a9b1d6"),
		Overlay: lipgloss.Color("#414868"),
		Blue:    lipgloss.Color("#7aa2f7"),
		Green:   lipgloss.Color("#9ece6a"),
		Red:     lipgloss.Color("#f7768e"),
		Mauve:   lipgloss.Color("#bb9af7"),
		Peach:   lipgloss.Color("#ff9e64"),
	}

	RosePine = ThemePalette{
		Name: "Rosé Pine",
		Base:    lipgloss.Color("#191724"),
		Text:    lipgloss.Color("#e0def4"),
		Subtext: lipgloss.Color("#908caa"),
		Overlay: lipgloss.Color("#44415a"),
		Blue:    lipgloss.Color("#9ccfd8"),
		Green:   lipgloss.Color("#31748f"),
		Red:     lipgloss.Color("#ebbcba"),
		Mauve:   lipgloss.Color("#c4a7e7"),
		Peach:   lipgloss.Color("#f6c177"),
	}

	Monokai = ThemePalette{
		Name: "Monokai",
		Base:    lipgloss.Color("#2d2a2e"),
		Text:    lipgloss.Color("#fcfcfa"),
		Subtext: lipgloss.Color("#939293"),
		Overlay: lipgloss.Color("#403e41"),
		Blue:    lipgloss.Color("#78dce8"),
		Green:   lipgloss.Color("#a9dc76"),
		Red:     lipgloss.Color("#ff6188"),
		Mauve:   lipgloss.Color("#ab9df2"),
		Peach:   lipgloss.Color("#fc9867"),
	}

	SolarizedDark = ThemePalette{
		Name: "Solarized Dark",
		Base:    lipgloss.Color("#002b36"),
		Text:    lipgloss.Color("#839496"),
		Subtext: lipgloss.Color("#586e75"),
		Overlay: lipgloss.Color("#073642"),
		Blue:    lipgloss.Color("#268bd2"),
		Green:   lipgloss.Color("#859900"),
		Red:     lipgloss.Color("#dc322f"),
		Mauve:   lipgloss.Color("#6c71c4"),
		Peach:   lipgloss.Color("#cb4b16"),
	}
)

var availableThemes = []ThemePalette{CatppuccinMocha, Dracula, Nord, Gruvbox, TokyoNight, RosePine, Monokai, SolarizedDark}
var currentThemeIdx = 0

var (
	ThemeBase    lipgloss.Color
	ThemeText    lipgloss.Color
	ThemeSubtext lipgloss.Color
	ThemeOverlay lipgloss.Color
	ThemeBlue    lipgloss.Color
	ThemeGreen   lipgloss.Color
	ThemeRed     lipgloss.Color
	ThemeMauve   lipgloss.Color
	ThemePeach   lipgloss.Color
)

var (
	WindowStyle       lipgloss.Style
	TitleStyle        lipgloss.Style
	LabelStyle        lipgloss.Style
	FocusedLabelStyle lipgloss.Style
	HelperStyle       lipgloss.Style
	ErrorStyle        lipgloss.Style
	ButtonStyle       lipgloss.Style
	ActiveButtonStyle lipgloss.Style
	SuccessTitleStyle lipgloss.Style
)

func init() {
	ApplyTheme(CatppuccinMocha)
}

func CycleTheme() string {
	currentThemeIdx = (currentThemeIdx + 1) % len(availableThemes)
	t := availableThemes[currentThemeIdx]
	ApplyTheme(t)
	return t.Name
}

func GetCurrentThemeName() string {
	if len(availableThemes) > 0 && currentThemeIdx >= 0 && currentThemeIdx < len(availableThemes) {
		return availableThemes[currentThemeIdx].Name
	}
	return "Unknown"
}

func ApplyThemeByName(name string) {
	for i, t := range availableThemes {
		if t.Name == name {
			currentThemeIdx = i
			ApplyTheme(t)
			return
		}
	}
	ApplyTheme(CatppuccinMocha)
}

func ApplyTheme(t ThemePalette) {
	ThemeBase = t.Base
	ThemeText = t.Text
	ThemeSubtext = t.Subtext
	ThemeOverlay = t.Overlay
	ThemeBlue = t.Blue
	ThemeGreen = t.Green
	ThemeRed = t.Red
	ThemeMauve = t.Mauve
	ThemePeach = t.Peach

	WindowStyle = lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ThemeMauve).
		Background(ThemeBase)

	TitleStyle = lipgloss.NewStyle().
		Foreground(ThemeBase).
		Background(ThemeMauve).
		Padding(0, 2).
		Bold(true).
		MarginBottom(1)

	LabelStyle = lipgloss.NewStyle().
		Foreground(ThemeText).
		Bold(true)

	FocusedLabelStyle = lipgloss.NewStyle().
		Foreground(ThemeGreen).
		Bold(true)

	HelperStyle = lipgloss.NewStyle().
		Foreground(ThemeOverlay).
		Italic(true)

	ErrorStyle = lipgloss.NewStyle().
		Foreground(ThemeRed).
		Bold(true)

	ButtonStyle = lipgloss.NewStyle().
		Foreground(ThemeText).
		Background(ThemeOverlay).
		Padding(0, 3).
		MarginTop(1)

	ActiveButtonStyle = lipgloss.NewStyle().
		Foreground(ThemeBase).
		Background(ThemeGreen).
		Padding(0, 3).
		Bold(true).
		MarginTop(1)

	SuccessTitleStyle = lipgloss.NewStyle().
		Foreground(ThemeBase).
		Background(ThemeGreen).
		Padding(0, 2).
		Bold(true).
		MarginBottom(1)
		
	// Refresh List UI globals too!
	HeaderStyle = lipgloss.NewStyle().Foreground(ThemeBase).Background(ThemeMauve).Bold(true)
	MilestoneRowStyle = lipgloss.NewStyle().Foreground(ThemeText).Background(ThemeOverlay)
	StoryRowStyle = lipgloss.NewStyle().Foreground(ThemeText).Background(ThemeOverlay)
	TaskRowStyle = lipgloss.NewStyle().Foreground(ThemeBase).Background(ThemeOverlay) // Darker text for lowest tier tasks
	ActiveRowStyle = lipgloss.NewStyle().Foreground(ThemeBase).Background(ThemeBlue).Bold(true) // Blue highlight for selected row
}

// Adjust dimensions based on terminal size
func AdjustDimensions(width, height int) (contentWidth, targetWidth, targetHeight int) {
	targetWidth = int(float64(width) * 0.95)
	if targetWidth < 40 {
		targetWidth = 40
	}
	targetHeight = int(float64(height) * 0.95)
	if targetHeight < 12 {
		targetHeight = 12
	}
	contentWidth = targetWidth - 8
	if contentWidth < 20 {
		contentWidth = 20
	}
	return
}
