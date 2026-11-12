package completions

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/completions"
	"github.com/justwasm/storybook"
)

type completionsModel struct {
	comp     *completions.Completions
	stateIdx int
}

func newCompletionsModel(stateIdx int) *completionsModel {
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ca3af")).Padding(0, 1)
	focusedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f9fafb")).Background(lipgloss.Color("#374151")).Bold(true).Padding(0, 1)
	matchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#60a5fa")).Bold(true)

	c := completions.New(normalStyle, focusedStyle, matchStyle)

	sampleFiles := []completions.FileCompletionValue{
		{Path: "cmd/crush/main.go"},
		{Path: "internal/ui/model/ui.go"},
		{Path: "internal/ui/dialog/models.go"},
		{Path: "internal/ui/chat/tools.go"},
		{Path: "go.mod"},
		{Path: "go.sum"},
		{Path: "README.md"},
	}

	sampleResources := []completions.ResourceCompletionValue{
		{
			MCPName: "github",
			URI:     "github://charmbracelet/crush/issues/open",
			Title:   "GitHub Issues (Open)",
		},
		{
			MCPName: "postgres",
			URI:     "postgres://localhost:5432/public/schema",
			Title:   "PostgreSQL Schema",
		},
		{
			MCPName: "linear",
			URI:     "linear://active-sprint",
			Title:   "Linear Sprint Tasks",
		},
	}

	switch stateIdx {
	case 1: // MCP Resources
		c.SetItems(nil, sampleResources)
	case 2: // Filtered
		c.SetItems(sampleFiles, sampleResources)
		c.Filter("model")
	default: // All files and resources
		c.SetItems(sampleFiles, sampleResources)
	}

	return &completionsModel{
		comp:     c,
		stateIdx: stateIdx,
	}
}

func (m *completionsModel) Init() tea.Cmd {
	return nil
}

func (m *completionsModel) SetState(idx int) {
	m.stateIdx = idx
	m.comp = newCompletionsModel(idx).comp
}

func (m *completionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.comp == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		_, _ = m.comp.Update(msg)
	}
	return m, nil
}

func (m *completionsModel) View() tea.View {
	if m.comp == nil {
		return tea.NewView("Completions popup unavailable")
	}
	var b strings.Builder
	b.WriteString("Crush Floating Autocomplete Popup (@files & MCP resources)\n\n")
	b.WriteString(m.comp.Render())
	b.WriteString("\n\n(Use [↑/↓] or [Ctrl+P/N] to navigate • [Tab/Enter] select • Presets: [1] Files & Resources  [2] Resources Only  [3] Filtered Query)")
	return tea.NewView(b.String())
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "common-completions",
		Name:        "Autocomplete Popup",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelMolecule,
		Description: "Floating dropdown completion menu triggered by @mentions, supporting files, directories, and MCP resources with fuzzy matching.",
		UpstreamRef: "internal/ui/completions/completions.go",
		States: []storybook.State{
			{ID: "all", Name: "Files & Resources", Description: "Full unified list of repository files and connected MCP resources"},
			{ID: "resources", Name: "MCP Resources", Description: "Third-party MCP context resources (GitHub issues, database schemas)"},
			{ID: "filtered", Name: "Filtered Query", Description: "Real-time fuzzy search filter applied ('model')"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newCompletionsModel(0))
		},
	})
}
