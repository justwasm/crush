package logo

import (
	_ "embed"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/ui/logo"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/justwasm/storybook"
)

type logoModel struct {
	stateIdx int
	width    int
	height   int
}

func (m *logoModel) Init() tea.Cmd {
	return nil
}

func (m *logoModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *logoModel) SetState(idx int) {
	m.stateIdx = idx
}

func (m *logoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
	}
	return m, nil
}

func (m *logoModel) View() tea.View {
	s := styles.CharmtonePantera()
	w := m.width
	if w <= 0 {
		w = 80
	}

	opts := logo.Opts{
		FieldColor:   s.Logo.FieldColor,
		TitleColorA:  s.Logo.TitleColorA,
		TitleColorB:  s.Logo.TitleColorB,
		CharmColor:   s.Logo.CharmColor,
		VersionColor: s.Logo.VersionColor,
		Width:        w,
	}

	var content string
	switch m.stateIdx {
	case 1: // Compact Sidebar
		content = logo.Render(s.Logo.GradCanvas, "v1.2.0", true, opts)
	case 2: // Hypercrush
		opts.Hyper = true
		content = logo.Render(s.Logo.GradCanvas, "v1.2.0", false, opts)
	case 3: // Stretched Letterform
		opts.Unstable = true
		content = logo.Render(s.Logo.GradCanvas, "v1.2.0", false, opts)
	default: // Standard Wide
		content = logo.Render(s.Logo.GradCanvas, "v1.2.0", false, opts)
	}

	hint := "\n\n  Press [1] Standard Wide  [2] Compact Sidebar  [3] Hypercrush  [4] Stretched"
	return tea.NewView(strings.TrimRight(content, "\n") + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "foundations-logo",
		Name:        "Logo & Wordmark",
		Category:    storybook.CategoryFoundations,
		Level:       storybook.LevelAtom,
		Description: "Stylized Crush wordmark with diagonal field, gradient letterforms, and version badges.",
		UpstreamRef: "internal/ui/logo/logo.go",
		States: []storybook.State{
			{ID: "wide", Name: "Standard Wide", Description: "Main pane title art"},
			{ID: "compact", Name: "Compact Sidebar", Description: "Narrow sidebar wordmark"},
			{ID: "hyper", Name: "Hypercrush", Description: "Hyper edition branding variant"},
			{ID: "stretched", Name: "Stretched", Description: "Randomized letterform stretch"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(&logoModel{width: 80, height: 24})
		},
	})
}
