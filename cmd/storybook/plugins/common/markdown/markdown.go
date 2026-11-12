package markdown

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/justwasm/storybook"
)

type markdownModel struct {
	stateIdx int
	width    int
	height   int
}

func (m *markdownModel) Init() tea.Cmd {
	return nil
}

func (m *markdownModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *markdownModel) SetState(idx int) {
	m.stateIdx = idx
}

func (m *markdownModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
	}
	return m, nil
}

const sampleDoc = `
# Agent Execution Architecture
Autonomous multi-step execution with **Antigravity** & **Crush**.

### Key Milestones
1. Initializing workspace sandbox
2. Synthesizing tool call execution graph
3. Streaming tokens with sub-millisecond AST parsing

| Stage | Latency | Status |
| :--- | :--- | :--- |
| Planning | 120ms | Completed |
| Verification | 45ms | Active |
`

func (m *markdownModel) View() tea.View {
	st := styles.CharmtonePantera()
	w := max(40, m.width-4)

	var renderer *glamour.TermRenderer
	var label string
	switch m.stateIdx {
	case 1:
		renderer = common.UserMarkdownRenderer(&st, w)
		label = "User Markdown Renderer"
	case 2:
		renderer = common.PlanMarkdownRenderer(&st, w)
		label = "Plan Markdown Renderer"
	case 3:
		renderer = common.QuietMarkdownRenderer(&st, w)
		label = "Quiet Markdown Renderer"
	default:
		renderer = common.MarkdownRenderer(&st, w)
		label = "Standard Assistant Markdown Renderer"
	}

	rendered, err := renderer.Render(sampleDoc)
	if err != nil {
		rendered = sampleDoc
	}

	hint := "\n  Mode: " + label + " • Presets: [1] Standard  [2] User  [3] Plan  [4] Quiet"
	return tea.NewView(strings.TrimRight(rendered, "\n") + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "common-markdown",
		Name:        "Glamour Markdown",
		Category:    storybook.CategoryBasics,
		Level:       storybook.LevelMolecule,
		Description: "Specialized Glamour markdown renderers (Standard, User, Plan, Quiet) with syntax highlighting.",
		UpstreamRef: "internal/ui/common/markdown.go",
		States: []storybook.State{
			{ID: "standard", Name: "Standard Assistant", Description: "Rich assistant output with headers, tables, code"},
			{ID: "user", Name: "User Bubble", Description: "Styled user input markdown"},
			{ID: "plan", Name: "Plan Document", Description: "Structured plan document formatting"},
			{ID: "quiet", Name: "Quiet Mode", Description: "Subtle muted prose presentation"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(&markdownModel{width: 80, height: 24})
		},
	})
}
