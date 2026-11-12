package landing

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/justwasm/storybook"
)

type landingModel struct {
	width  int
	height int
}

func (m *landingModel) Init() tea.Cmd {
	return nil
}

func (m *landingModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *landingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
	}
	return m, nil
}

func (m *landingModel) View() tea.View {
	st := styles.CharmtonePantera()
	w := max(60, m.width-4)

	var b strings.Builder

	// 1. CWD and git branch
	cwd := common.PrettyPathInline(&st, "/Users/gear/GitHub/crush", "main (clean)", w)
	b.WriteString(cwd + "\n\n")

	// 2. Active Model Banner
	ctx := &common.ModelContextInfo{
		ContextUsed:    18400,
		ModelContext:   200000,
		Cost:           0.018,
		EstimatedUsage: false,
	}
	modelBanner := common.ModelInfo(&st, "Claude 3.7 Sonnet", "Anthropic", "High Reasoning", ctx, w, nil)
	b.WriteString(modelBanner + "\n\n")

	// 3. Three-column status: LSP / MCP / Skills
	colW := max(18, (w-6)/3)
	colBox := lipgloss.NewStyle().Width(colW).Padding(0, 1).BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))

	lspContent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("LSP SERVERS") + "\n\n" +
		"• gopls (running)\n" +
		"• tsserver (idle)\n" +
		"0 diagnostics"
	lspCard := colBox.Render(lspContent)

	mcpContent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("MCP TOOLS") + "\n\n" +
		"• filesystem (active)\n" +
		"• github (connected)\n" +
		"14 tools ready"
	mcpCard := colBox.Render(mcpContent)

	skillsContent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("SKILLS") + "\n\n" +
		"• bubbletea-tui\n" +
		"• storybook-substrate\n" +
		"2 active skills"
	skillsCard := colBox.Render(skillsContent)

	threeCols := lipgloss.JoinHorizontal(lipgloss.Top, lspCard, " ", mcpCard, " ", skillsCard)
	b.WriteString(threeCols + "\n\n")

	// 4. Shortcut hints
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
	b.WriteString(hintStyle("  Shortcuts: [/] Commands  •  [ctrl+p] Switch Model  •  [ctrl+n] New Session  •  [q] Quit"))

	return tea.NewView(b.String())
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "app-landing",
		Name:        "Landing Welcome View",
		Category:    storybook.CategoryWorkspace,
		Level:       storybook.LevelOrganism,
		Description: "Workspace initial landing screen with active model info, LSP/MCP cards, and quick shortcuts.",
		UpstreamRef: "internal/ui/model/landing.go",
		Factory: func() storybook.Instance {
			return storybook.WrapModel(&landingModel{width: 80, height: 24})
		},
	})
}
