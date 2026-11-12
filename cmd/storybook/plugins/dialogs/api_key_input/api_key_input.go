package api_key_input

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/catwalk/v3/pkg/catwalk"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type apiKeyInputDialogModel struct {
	input    *dialog.APIKeyInput
	width    int
	height   int
	stateIdx int
}

func newAPIKeyInputDialogModel(state dialog.APIKeyInputState) *apiKeyInputDialogModel {
	com := mockws.NewCommon()
	prov := catwalk.Provider{
		ID:   "anthropic",
		Name: "Anthropic",
	}
	model := config.SelectedModel{
		Provider: "anthropic",
		Model:    "claude-3-7-sonnet",
	}
	dlg, _ := dialog.NewAPIKeyInput(com, false, prov, model, config.SelectedModelTypeLarge)
	if state != dialog.APIKeyInputStateInitial {
		dlg.HandleMsg(dialog.ActionChangeAPIKeyState{State: state})
	}
	return &apiKeyInputDialogModel{
		input:  dlg,
		width:  80,
		height: 20,
	}
}

func (m *apiKeyInputDialogModel) Init() tea.Cmd {
	return nil
}

func (m *apiKeyInputDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *apiKeyInputDialogModel) SetState(idx int) {
	m.stateIdx = idx
	switch idx {
	case 1:
		m.input = newAPIKeyInputDialogModel(dialog.APIKeyInputStateVerifying).input
	case 2:
		m.input = newAPIKeyInputDialogModel(dialog.APIKeyInputStateVerified).input
	case 3:
		m.input = newAPIKeyInputDialogModel(dialog.APIKeyInputStateError).input
	default:
		m.input = newAPIKeyInputDialogModel(dialog.APIKeyInputStateInitial).input
	}
}

func (m *apiKeyInputDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.input == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.input.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.input.HandleMsg(msg)
	}
	return m, nil
}

func (m *apiKeyInputDialogModel) View() tea.View {
	if m.input == nil {
		return tea.NewView("API Key input dialog unavailable")
	}
	w := max(40, m.width)
	h := max(14, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.input.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  Provider Token Entry: Type API key • [Enter] submit • Presets: [1] Initial  [2] Verifying  [3] Verified  [4] Error"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-api-key-input",
		Name:        "API Key & Token Input",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Provider authentication token entry screen with live validation feedback, spinner animation, and error states.",
		UpstreamRef: "internal/ui/dialog/api_key_input.go",
		States: []storybook.State{
			{ID: "initial", Name: "Initial Prompt", Description: "Placeholder prompt awaiting user token input"},
			{ID: "verifying", Name: "Verifying Spinner", Description: "Connecting and verifying token authenticity"},
			{ID: "verified", Name: "Verified Success", Description: "Green validation confirmation"},
			{ID: "error", Name: "Error State", Description: "Invalid credentials notification"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newAPIKeyInputDialogModel(dialog.APIKeyInputStateInitial))
		},
	})
}
