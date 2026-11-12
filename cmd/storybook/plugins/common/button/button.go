package button

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/justwasm/storybook"
)

type buttonModel struct {
	stateIdx   int
	activeBtn  int
	clickCount int
}

func (m *buttonModel) Init() tea.Cmd {
	return nil
}

func (m *buttonModel) SetState(idx int) {
	m.stateIdx = idx
}

func (m *buttonModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "left", "h":
			m.activeBtn = 0
		case "right", "l", "tab":
			m.activeBtn = 1
		case "enter", " ":
			m.clickCount++
		}
	}
	return m, nil
}

func (m *buttonModel) View() tea.View {
	st := styles.CharmtonePantera()
	var b strings.Builder

	h1 := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render
	b.WriteString(h1("Crush Common Button Component") + "\n\n")

	// Render different states based on stateIdx
	switch m.stateIdx {
	case 1: // Focused / Selected
		b.WriteString("Selected State:\n\n")
		b.WriteString("  " + common.Button(&st, common.ButtonOpts{Text: "Confirm", Selected: true, UnderlineIndex: 0}) + "  ")
		b.WriteString(common.Button(&st, common.ButtonOpts{Text: "Cancel", Selected: false, UnderlineIndex: 0}) + "\n\n")
	case 2: // Hovered
		b.WriteString("Hovered State:\n\n")
		b.WriteString("  " + common.Button(&st, common.ButtonOpts{Text: "Confirm", Hovered: true, UnderlineIndex: 0}) + "  ")
		b.WriteString(common.Button(&st, common.ButtonOpts{Text: "Cancel", Hovered: false, UnderlineIndex: 0}) + "\n\n")
	case 3: // Inactive
		b.WriteString("Inactive State (Unfocused Pane):\n\n")
		b.WriteString("  " + common.Button(&st, common.ButtonOpts{Text: "Confirm", Inactive: true, UnderlineIndex: 0}) + "  ")
		b.WriteString(common.Button(&st, common.ButtonOpts{Text: "Cancel", Inactive: true, UnderlineIndex: 0}) + "\n\n")
	default: // Interactive Gallery
		b.WriteString("Interactive Button Pair (Use ←/→ or h/l to switch, [Enter/Space] to click):\n\n")
		btn1 := common.Button(&st, common.ButtonOpts{
			Text:           "Continue",
			Selected:       m.activeBtn == 0,
			UnderlineIndex: 0,
		})
		btn2 := common.Button(&st, common.ButtonOpts{
			Text:           "Dismiss",
			Selected:       m.activeBtn == 1,
			UnderlineIndex: 0,
		})
		b.WriteString("  " + btn1 + "   " + btn2 + "\n\n")
		b.WriteString(fmt.Sprintf("  Clicks recorded: %d\n", m.clickCount))
	}

	b.WriteString("\n  Presets: [1] Interactive  [2] Selected  [3] Hovered  [4] Inactive\n")
	return tea.NewView(b.String())
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "common-button",
		Name:        "Button Component",
		Category:    storybook.CategoryBasics,
		Level:       storybook.LevelAtom,
		Description: "Interactive button with selection, hover, inactive dimming, and mnemonic underline.",
		UpstreamRef: "internal/ui/common/button.go",
		States: []storybook.State{
			{ID: "interactive", Name: "Interactive", Description: "Live focusable button pair"},
			{ID: "selected", Name: "Selected", Description: "Focused bold variant"},
			{ID: "hovered", Name: "Hovered", Description: "Mouse-over highlight"},
			{ID: "inactive", Name: "Inactive", Description: "Muted background pane style"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(&buttonModel{})
		},
	})
}
