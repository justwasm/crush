package reasoning

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type reasoningDialogModel struct {
	reasoning *dialog.Reasoning
	width     int
	height    int
	stateIdx  int
}

func newReasoningDialogModel(stateIdx int) *reasoningDialogModel {
	com := mockws.NewCommon()
	dlg, _ := dialog.NewReasoning(com)
	switch stateIdx {
	case 1:
		if dlg != nil {
			dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		}
	case 2:
		if dlg != nil {
			dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
			dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		}
	}
	return &reasoningDialogModel{
		reasoning: dlg,
		width:     80,
		height:    20,
	}
}

func (m *reasoningDialogModel) Init() tea.Cmd {
	return nil
}

func (m *reasoningDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *reasoningDialogModel) SetState(idx int) {
	m.stateIdx = idx
	m.reasoning = newReasoningDialogModel(idx).reasoning
}

func (m *reasoningDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.reasoning == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.reasoning.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.reasoning.HandleMsg(msg)
	}
	return m, nil
}

func (m *reasoningDialogModel) View() tea.View {
	if m.reasoning == nil {
		return tea.NewView("Reasoning dialog unavailable")
	}
	w := max(40, m.width)
	h := max(14, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.reasoning.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  Reasoning Effort: [↑/↓] select effort • [Enter] confirm • Presets: [1] Low  [2] Medium  [3] High"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-reasoning",
		Name:        "Reasoning Effort Selector",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Dialog to adjust model thinking / reasoning budget effort levels (None, Low, Medium, High).",
		UpstreamRef: "internal/ui/dialog/reasoning.go",
		States: []storybook.State{
			{ID: "low", Name: "Low Effort", Description: "Quick responses with minimal thinking time"},
			{ID: "medium", Name: "Medium Effort", Description: "Balanced deliberation for everyday coding"},
			{ID: "high", Name: "High Effort", Description: "Extended chain-of-thought for complex architecture"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newReasoningDialogModel(0))
		},
	})
}
