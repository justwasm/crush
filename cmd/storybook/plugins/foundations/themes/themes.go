package themes

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/justwasm/storybook"
)

type themesModel struct {
	themeNames []string
	activeIdx  int
}

func newThemesModel() *themesModel {
	names := styles.BuiltinThemeNames()
	if len(names) == 0 {
		names = []string{"charmtone", "hyper", "gruvbox", "keanu"}
	}
	return &themesModel{
		themeNames: names,
		activeIdx:  0,
	}
}

func (m *themesModel) Init() tea.Cmd {
	return nil
}

func (m *themesModel) SetState(idx int) {
	if idx >= 0 && idx < len(m.themeNames) {
		m.activeIdx = idx
	}
}

func (m *themesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "j", "down", "right":
			m.activeIdx = (m.activeIdx + 1) % len(m.themeNames)
		case "k", "up", "left":
			m.activeIdx = (m.activeIdx - 1 + len(m.themeNames)) % len(m.themeNames)
		}
	}
	return m, nil
}

func (m *themesModel) View() tea.View {
	activeTheme := m.themeNames[m.activeIdx]
	pal, err := styles.ThemePalette(activeTheme)
	if err != nil {
		pal, _ = styles.ThemePalette("charmtone")
	}

	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	b.WriteString(titleStyle.Render(fmt.Sprintf("Crush Theme Palette: %s", strings.ToUpper(activeTheme))))
	b.WriteString(fmt.Sprintf("  (Theme %d of %d • [←/→] Cycle)\n\n", m.activeIdx+1, len(m.themeNames)))

	swatch := func(label string, hexStr string) string {
		c := lipgloss.Color(hexStr)
		block := lipgloss.NewStyle().Background(c).Foreground(lipgloss.Color("#ffffff")).Padding(0, 1).Render(hexStr)
		lbl := lipgloss.NewStyle().Width(14).Render(label)
		return fmt.Sprintf("  %s %s", lbl, block)
	}

	b.WriteString("Core Brand Tokens:\n")
	b.WriteString(swatch("Primary", pal.Primary) + "\n")
	b.WriteString(swatch("Secondary", pal.Secondary) + "\n")
	b.WriteString(swatch("Accent", pal.Accent) + "\n")
	b.WriteString(swatch("Keyword", pal.Keyword) + "\n\n")

	b.WriteString("Background Tonal Layers:\n")
	b.WriteString(swatch("Bg Base", pal.BgBase) + "\n")
	b.WriteString(swatch("Bg LessVis", pal.BgLessVisible) + "\n")
	b.WriteString(swatch("Bg MostVis", pal.BgMostVisible) + "\n\n")

	b.WriteString("Semantic Signals:\n")
	b.WriteString(swatch("Success", pal.Success) + "  " + swatch("Error", pal.Error) + "\n")
	b.WriteString(swatch("Warning", pal.Warning) + "  " + swatch("Attention", pal.Attention) + "\n")
	b.WriteString(swatch("Info", pal.Info) + "  " + swatch("Busy", pal.Busy) + "\n")

	return tea.NewView(b.String())
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "foundations-themes",
		Name:        "Theme Palettes",
		Category:    storybook.CategoryFoundations,
		Level:       storybook.LevelAtom,
		Description: "Built-in brand palettes and semantic design tokens across all Crush themes.",
		UpstreamRef: "internal/ui/styles/themes.go",
		States: []storybook.State{
			{ID: "charmtone", Name: "Charmtone Pantera", Description: "Signature dark theme"},
			{ID: "hyper", Name: "Hypercrush", Description: "Hyper variant theme"},
			{ID: "gruvbox", Name: "Gruvbox Dark", Description: "Earthy retro groove"},
			{ID: "keanu", Name: "Keanu", Description: "Cyberpunk high-contrast"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newThemesModel())
		},
	})
}
