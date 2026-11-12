package models

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type modelsDialogModel struct {
	models   *dialog.Models
	width    int
	height   int
	stateIdx int
}

func newModelsDialogModel(onboarding bool) *modelsDialogModel {
	com := mockws.NewCommon()
	m, _ := dialog.NewModels(com, onboarding)
	return &modelsDialogModel{
		models: m,
		width:  80,
		height: 24,
	}
}

func (m *modelsDialogModel) Init() tea.Cmd {
	return nil
}

func (m *modelsDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *modelsDialogModel) SetState(idx int) {
	m.stateIdx = idx
	com := mockws.NewCommon()
	switch idx {
	case 1: // Small task model
		dlg, _ := dialog.NewModels(com, false)
		if dlg != nil {
			dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
		}
		m.models = dlg
	case 2: // Onboarding mode
		dlg, _ := dialog.NewModels(com, true)
		m.models = dlg
	default: // Large task model (standard)
		dlg, _ := dialog.NewModels(com, false)
		m.models = dlg
	}
}

func (m *modelsDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.models == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.models.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.models.HandleMsg(msg)
	}
	return m, nil
}

func (m *modelsDialogModel) View() tea.View {
	if m.models == nil {
		return tea.NewView("Models dialog unavailable")
	}
	w := max(40, m.width)
	h := max(16, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.models.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  Model Picker: [Tab] switch Large/Small • [↑/↓] select • [Enter] confirm • Presets: [1] Large  [2] Small  [3] Onboarding"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-models",
		Name:        "Model Selector",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Interactive LLM provider & model picker with multi-tier tasks (Large/Small), filter input, and onboarding mode.",
		UpstreamRef: "internal/ui/dialog/models.go",
		States: []storybook.State{
			{ID: "large", Name: "Large Task Model", Description: "Primary reasoning model selection"},
			{ID: "small", Name: "Small Task Model", Description: "Fast sub-task / extraction model"},
			{ID: "onboarding", Name: "Onboarding Flow", Description: "Initial setup model picker ('Find your fave')"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newModelsDialogModel(false))
		},
	})
}
