package models

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/ui/common"
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
	case 2: // Onboarding mode (Original: borderless & bottom-left)
		dlg, _ := dialog.NewModels(com, true)
		m.models = dlg
	case 3: // Onboarding Framed & Centered (Custom variation: bordered & centered)
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

	var content string
	if m.stateIdx == 3 {
		// Framed and centered variant for onboarding flow (zero modification to crush upstream)
		com := mockws.NewCommon()
		t := com.Styles

		dialogStyle := t.Dialog.View
		titleStyle := t.Dialog.Title

		boxW := max(40, min(73, w-6))
		boxH := max(12, min(22, h-6))

		innerWidth := boxW - dialogStyle.GetHorizontalFrameSize()
		innerHeight := boxH - dialogStyle.GetVerticalFrameSize()

		subCanvas := uv.NewScreenBuffer(innerWidth, innerHeight)
		m.models.Draw(subCanvas, image.Rect(0, 0, innerWidth, innerHeight))

		rawLines := strings.Split(strings.ReplaceAll(subCanvas.Render(), "\r\n", "\n"), "\n")
		var validLines []string
		started := false
		for _, l := range rawLines {
			trimmed := strings.TrimRight(l, " ")
			if !started && trimmed == "" {
				continue
			}
			started = true
			validLines = append(validLines, trimmed)
		}
		for len(validLines) > 0 && validLines[len(validLines)-1] == "" {
			validLines = validLines[:len(validLines)-1]
		}
		innerContent := strings.Join(validLines, "\n")

		titleContentWidth := innerWidth - titleStyle.GetHorizontalFrameSize()
		title := common.DialogTitle(t, "Onboarding Setup", titleContentWidth, t.Dialog.TitleGradFromColor, t.Dialog.TitleGradToColor)
		header := titleStyle.Render(title)

		framed := dialogStyle.Width(boxW).Render(header + "\n\n" + innerContent)
		content = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, framed)
	} else {
		canvas := uv.NewScreenBuffer(w, h)
		m.models.Draw(canvas, image.Rect(0, 0, w, h))

		content = strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
		lines := strings.Split(content, "\n")
		for i, l := range lines {
			lines[i] = strings.TrimRight(l, " ")
		}
		content = strings.Join(lines, "\n")
	}

	hint := "\n\n  Model Picker: [Tab] switch Large/Small • [↑/↓] select • [Enter] confirm • Presets: [1] Large  [2] Small  [3] Onboarding (Original)  [4] Onboarding (Bordered)"
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
			{ID: "large", Name: "Large Task Model", Description: "Primary reasoning model selection (Bordered & Centered)"},
			{ID: "small", Name: "Small Task Model", Description: "Fast sub-task / extraction model (Bordered & Centered)"},
			{ID: "onboarding", Name: "Onboarding Flow", Description: "Initial setup model picker ('Find your fave', Borderless & Bottom-Left)"},
			{ID: "onboarding-framed", Name: "Onboarding (Bordered)", Description: "Framed border and centered placement without modifying crush core"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newModelsDialogModel(false))
		},
	})
}
