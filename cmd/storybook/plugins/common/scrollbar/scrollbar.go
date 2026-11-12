package scrollbar

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/justwasm/storybook"
)

type scrollbarModel struct {
	offset   int
	total    int
	viewSize int
	height   int
}

func newScrollbarModel() *scrollbarModel {
	return &scrollbarModel{
		offset:   0,
		total:    100,
		viewSize: 15,
		height:   15,
	}
}

func (m *scrollbarModel) Init() tea.Cmd {
	return nil
}

func (m *scrollbarModel) SetState(idx int) {
	switch idx {
	case 1:
		m.offset = 40 // middle
	case 2:
		m.offset = m.total - m.viewSize // bottom
	default:
		m.offset = 0 // top
	}
}

func (m *scrollbarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "j", "down":
			if m.offset < m.total-m.viewSize {
				m.offset++
			}
		case "k", "up":
			if m.offset > 0 {
				m.offset--
			}
		}
	}
	return m, nil
}

func (m *scrollbarModel) View() tea.View {
	st := styles.CharmtonePantera()
	var b strings.Builder

	h1 := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render
	b.WriteString(h1("Crush Viewport Scrollbar") + "  (Use [j/k] or [↑/↓] to scroll)\n\n")

	// Render mock list lines
	var listLines []string
	for i := m.offset; i < m.offset+m.viewSize && i < m.total; i++ {
		listLines = append(listLines, fmt.Sprintf("  • Item entry #%03d (Document log record)", i+1))
	}
	content := strings.Join(listLines, "\n")

	bar := common.Scrollbar(&st, m.height, m.total, m.viewSize, m.offset)
	joined := lipgloss.JoinHorizontal(lipgloss.Top, content, "    ", bar)
	b.WriteString(joined + "\n\n")
	b.WriteString(fmt.Sprintf("  Offset: %d / %d  •  Presets: [1] Top  [2] Middle  [3] Bottom\n", m.offset, m.total-m.viewSize))

	return tea.NewView(b.String())
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "common-scrollbar",
		Name:        "Scrollbar Indicator",
		Category:    storybook.CategoryBasics,
		Level:       storybook.LevelAtom,
		Description: "Proportional scrollbar indicator with thumb sizing and track rendering.",
		UpstreamRef: "internal/ui/common/scrollbar.go",
		States: []storybook.State{
			{ID: "top", Name: "Top", Description: "Scrolled to top"},
			{ID: "middle", Name: "Middle", Description: "Scrolled to midpoint"},
			{ID: "bottom", Name: "Bottom", Description: "Scrolled to bottom"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newScrollbarModel())
		},
	})
}
