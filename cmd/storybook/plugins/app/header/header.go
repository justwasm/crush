package header

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/logo"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/justwasm/storybook"
)

type headerModel struct {
	stateIdx int
	width    int
}

func (m *headerModel) Init() tea.Cmd {
	return nil
}

func (m *headerModel) SetSize(w, h int) {
	m.width = w
}

func (m *headerModel) SetState(idx int) {
	m.stateIdx = idx
}

func (m *headerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
	}
	return m, nil
}

func (m *headerModel) View() tea.View {
	st := styles.CharmtonePantera()
	w := max(50, m.width-4)

	var b strings.Builder
	h1 := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render
	b.WriteString(h1("Workspace Header Treatment") + "\n\n")

	opts := logo.Opts{
		FieldColor:   st.Logo.FieldColor,
		TitleColorA:  st.Logo.TitleColorA,
		TitleColorB:  st.Logo.TitleColorB,
		CharmColor:   st.Logo.CharmColor,
		VersionColor: st.Logo.VersionColor,
		Width:        w,
	}

	switch m.stateIdx {
	case 1: // Compact Active Header with Session Details
		charmLogo := st.Header.Charm.Render("Charm™") + " " +
			styles.ApplyBoldForegroundGrad(st.Header.LogoGradCanvas, "CRUSH", st.Header.LogoGradFromColor, st.Header.LogoGradToColor)
		diagPattern := lipgloss.NewStyle().Foreground(st.Logo.FieldColor).Render(" ╱╱╱╱╱╱╱╱╱ ")
		sessBadge := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("Session: #42-refactor-storybook")
		modelBadge := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(" [Claude 3.7 Sonnet] ")
		branchBadge := common.PrettyPathInline(&st, "/Users/gear/GitHub/crush", "main", 30)

		line1 := charmLogo + diagPattern + sessBadge + modelBadge
		b.WriteString(line1 + "\n")
		b.WriteString("  " + branchBadge + "\n\n")

	default: // Wide Initial Header
		logoBanner := logo.Render(st.Logo.GradCanvas, "v1.2.0", false, opts)
		b.WriteString(logoBanner + "\n\n")
	}

	b.WriteString("  Presets: [1] Full Header  [2] Compact Session Bar\n")
	return tea.NewView(b.String())
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "app-header",
		Name:        "Session Header",
		Category:    storybook.CategoryWorkspace,
		Level:       storybook.LevelOrganism,
		Description: "Top session header with diagonal ASCII pattern, active model pill, and session title.",
		UpstreamRef: "internal/ui/model/header.go",
		States: []storybook.State{
			{ID: "wide", Name: "Wide Initial", Description: "Full ASCII wordmark and version"},
			{ID: "compact", Name: "Compact Session", Description: "Condensed status bar with session badges"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(&headerModel{width: 80})
		},
	})
}
