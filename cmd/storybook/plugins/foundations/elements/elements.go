package elements

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/justwasm/storybook"
)

type elementsModel struct {
	stateIdx int
}

func (m *elementsModel) Init() tea.Cmd {
	return nil
}

func (m *elementsModel) SetState(idx int) {
	m.stateIdx = idx
}

func (m *elementsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *elementsModel) View() tea.View {
	st := styles.CharmtonePantera()
	var b strings.Builder

	h1 := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Render
	b.WriteString(h1("Crush Design System — UI Atoms & Badges") + "\n\n")

	b.WriteString("1. Paths & Git Branch Badges:\n")
	b.WriteString("  " + common.PrettyPath(&st, "/Users/gear/GitHub/crush/cmd/storybook", 60) + "\n")
	b.WriteString("  " + common.PrettyPathWithBranch(&st, "/Users/gear/GitHub/crush", "main (clean)", 60) + "\n\n")

	b.WriteString("2. Reasoning Effort Levels:\n")
	b.WriteString("  Low: " + common.FormatReasoningEffort("low") + "\n")
	b.WriteString("  Medium: " + common.FormatReasoningEffort("medium") + "\n")
	b.WriteString("  High: " + common.FormatReasoningEffort("high") + "\n")
	b.WriteString("  X-High: " + common.FormatReasoningEffort("xhigh") + "\n\n")

	b.WriteString("3. Model Badges & Context Info:\n")
	ctxInfo := &common.ModelContextInfo{
		ContextUsed:  34200,
		ModelContext: 128000,
		Cost:         0.042,
	}
	modelBanner := common.ModelInfo(&st, "Claude 3.7 Sonnet (Thinking)", "Anthropic", "High Reasoning", ctxInfo, 70, nil)
	b.WriteString("  " + strings.ReplaceAll(modelBanner, "\n", "\n  ") + "\n\n")

	return tea.NewView(b.String())
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "foundations-elements",
		Name:        "Badges & Elements",
		Category:    storybook.CategoryFoundations,
		Level:       storybook.LevelAtom,
		Description: "Path labels, git branch badges, reasoning effort tags, and model context cards.",
		UpstreamRef: "internal/ui/common/elements.go",
		Factory: func() storybook.Instance {
			return storybook.WrapModel(&elementsModel{})
		},
	})
}
