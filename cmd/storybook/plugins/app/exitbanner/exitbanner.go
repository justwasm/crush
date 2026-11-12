package exitbanner

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/crush/internal/ui/exitbanner"
	"github.com/justwasm/storybook"
)

type exitBannerModel struct {
	stateIdx int
	width    int
}

func (m *exitBannerModel) Init() tea.Cmd {
	return nil
}

func (m *exitBannerModel) SetSize(w, h int) {
	m.width = w
}

func (m *exitBannerModel) SetState(idx int) {
	m.stateIdx = idx
}

func (m *exitBannerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
	}
	return m, nil
}

func (m *exitBannerModel) View() tea.View {
	w := max(50, m.width-4)

	sess := &session.Session{
		ID:        "ses_01j98ab7cdef",
		Title:     "Refactor Crush TUI Component Harness",
		CreatedAt: 1731456000,
	}

	var bannerType config.ExitBanner
	var label string
	switch m.stateIdx {
	case 1:
		bannerType = config.ExitBannerCompact
		label = "Compact Resume Hint"
	default:
		bannerType = config.ExitBannerDefault
		label = "Full Brand Farewell Banner"
	}

	com := mockws.NewCommon()
	rendered := exitbanner.Render(com.Styles, bannerType, sess, w)

	var b strings.Builder
	h1 := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Render
	b.WriteString(h1("Exit Banner Treatment: "+label) + "\n\n")
	b.WriteString(rendered)
	b.WriteString("\n\n  Presets: [1] Full Banner  [2] Compact Resume Hint\n")

	return tea.NewView(b.String())
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "app-exitbanner",
		Name:        "Exit Farewell Banner",
		Category:    storybook.CategoryWorkspace,
		Level:       storybook.LevelOrganism,
		Description: "Post-TUI exit summary banner with ASCII logo and session resume instructions.",
		UpstreamRef: "internal/ui/exitbanner/exitbanner.go",
		States: []storybook.State{
			{ID: "full", Name: "Full Banner", Description: "Logo, parting message, and resume command"},
			{ID: "compact", Name: "Compact Resume", Description: "Single-line resume command only"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(&exitBannerModel{width: 80})
		},
	})
}
