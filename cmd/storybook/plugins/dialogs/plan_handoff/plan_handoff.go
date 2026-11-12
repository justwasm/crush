package plan_handoff

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type planHandoffDialogModel struct {
	handoff  *dialog.PlanHandoffInline
	width    int
	height   int
	stateIdx int
}

func newPlanHandoffDialogModel(stateIdx int) *planHandoffDialogModel {
	com := mockws.NewCommon()
	dlg := dialog.NewPlanHandoffInline(com)
	switch stateIdx {
	case 1: // YOLO
		dlg.HandleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	case 2: // Revise
		dlg.HandleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
		dlg.HandleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	}
	return &planHandoffDialogModel{
		handoff: dlg,
		width:   80,
		height:  16,
	}
}

func (m *planHandoffDialogModel) Init() tea.Cmd {
	return nil
}

func (m *planHandoffDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *planHandoffDialogModel) SetState(idx int) {
	m.stateIdx = idx
	m.handoff = newPlanHandoffDialogModel(idx).handoff
}

func (m *planHandoffDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.handoff == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_, cmd := m.handoff.HandleKey(msg)
		return m, cmd
	}
	return m, nil
}

func (m *planHandoffDialogModel) View() tea.View {
	if m.handoff == nil {
		return tea.NewView("Plan handoff inline prompt unavailable")
	}
	w := max(40, m.width)
	h := max(10, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.handoff.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  Plan Execution Handoff: [←/→] switch option • [Enter] confirm • Presets: [1] Start Coding  [2] Code (YOLO)  [3] Revise Plan"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-plan-handoff",
		Name:        "Plan Handoff & YOLO Mode",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelMolecule,
		Description: "Inline action prompt shown when architectural plan completes, allowing single-click transition to coder agent or YOLO execution.",
		UpstreamRef: "internal/ui/dialog/plan_handoff_inline.go",
		States: []storybook.State{
			{ID: "coding", Name: "Start Coding", Description: "Standard permission-gated code generation"},
			{ID: "yolo", Name: "Code (YOLO)", Description: "Autonomous auto-approved execution mode"},
			{ID: "revise", Name: "Revise Plan", Description: "Inline feedback textarea for plan revisions"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newPlanHandoffDialogModel(0))
		},
	})
}
