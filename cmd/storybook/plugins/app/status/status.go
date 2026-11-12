package status

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/justwasm/storybook"
)

type statusModel struct {
	stateIdx int
	width    int
}

func (m *statusModel) Init() tea.Cmd {
	return nil
}

func (m *statusModel) SetSize(w, h int) {
	m.width = w
}

func (m *statusModel) SetState(idx int) {
	m.stateIdx = idx
}

func (m *statusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
	}
	return m, nil
}

func (m *statusModel) View() tea.View {
	st := styles.CharmtonePantera()
	w := max(50, m.width-4)

	var b strings.Builder
	h1 := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render
	b.WriteString(h1("Bottom Status Bar & Mode Badges") + "\n\n")

	var badge string
	switch m.stateIdx {
	case 1: // PLAN Mode
		badge = st.Status.ModeBadgePlan.String() + " "
	case 2: // YOLO Mode
		badge = st.Status.ModeBadgeYolo.String() + " "
	default: // Normal Coding Mode
		badge = ""
	}

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	keyBold := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)

	keys := []string{
		keyBold.Render("[/]") + keyStyle.Render(" commands"),
		keyBold.Render("[ctrl+p]") + keyStyle.Render(" models"),
		keyBold.Render("[shift+tab]") + keyStyle.Render(" plan mode"),
		keyBold.Render("[ctrl+y]") + keyStyle.Render(" yolo"),
		keyBold.Render("[esc]") + keyStyle.Render(" cancel"),
	}

	helpLine := strings.Join(keys, "  •  ")
	fullBar := badge + helpLine

	barStyle := lipgloss.NewStyle().
		Width(w).
		Padding(0, 1).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("238"))

	b.WriteString(barStyle.Render(fullBar) + "\n\n")
	b.WriteString("  Presets: [1] Normal Code Mode  [2] Plan Mode  [3] YOLO (Auto-Accept) Mode\n")

	return tea.NewView(b.String())
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "app-status",
		Name:        "Bottom Status Bar",
		Category:    storybook.CategoryWorkspace,
		Level:       storybook.LevelOrganism,
		Description: "Bottom status bar with mode badges (Plan / YOLO / Code) and global keybinding hints.",
		UpstreamRef: "internal/ui/model/status.go",
		States: []storybook.State{
			{ID: "normal", Name: "Normal Mode", Description: "Standard coding session status"},
			{ID: "plan", Name: "Plan Mode", Description: "Plan agent active mode badge"},
			{ID: "yolo", Name: "YOLO Mode", Description: "Auto-approve permissions badge"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(&statusModel{width: 80})
		},
	})
}
