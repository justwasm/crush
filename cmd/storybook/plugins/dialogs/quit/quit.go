package quit

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type quitDialogModel struct {
	quit   *dialog.Quit
	width  int
	height int
	action string
}

func newQuitDialogModel() *quitDialogModel {
	com := common.DefaultCommon(nil)
	return &quitDialogModel{
		quit:   dialog.NewQuit(com, 0),
		width:  80,
		height: 20,
	}
}

func (m *quitDialogModel) Init() tea.Cmd {
	return nil
}

func (m *quitDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *quitDialogModel) SetState(idx int) {
	// 0: Reset, 1: Focus No, 2: Focus Yes
	m.quit = dialog.NewQuit(common.DefaultCommon(nil), 0)
	if idx == 2 {
		m.quit.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	}
}

func (m *quitDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		action := m.quit.HandleMsg(msg)
		if action != nil {
			m.action = "Action triggered: " + msg.String()
		}
	case tea.MouseMsg:
		_ = m.quit.HandleMsg(msg)
	}
	return m, nil
}

func (m *quitDialogModel) View() tea.View {
	w := max(40, m.width)
	h := max(12, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.quit.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  Interactive Quit Modal: Use [←/→] to select, [Enter/Space] to confirm, [y/n] quick keys"
	if m.action != "" {
		hint += " • " + m.action
	}
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-quit",
		Name:        "Quit Confirmation",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Interactive quit confirmation dialog with key and mouse hover handling.",
		UpstreamRef: "internal/ui/dialog/quit.go",
		States: []storybook.State{
			{ID: "default", Name: "Cancel Selected", Description: "No / Cancel button active"},
			{ID: "yes", Name: "Quit Selected", Description: "Yes / Quit button active"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newQuitDialogModel())
		},
	})
}
