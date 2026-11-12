package notifications

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type notificationsDialogModel struct {
	notifs   *dialog.Notifications
	width    int
	height   int
	stateIdx int
}

func newNotificationsDialogModel(stateIdx int) *notificationsDialogModel {
	com := mockws.NewCommon()
	dlg := dialog.NewNotifications(com)
	switch stateIdx {
	case 1: // Native
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	case 2: // OSC
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	case 3: // Disabled
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	}
	return &notificationsDialogModel{
		notifs: dlg,
		width:  80,
		height: 20,
	}
}

func (m *notificationsDialogModel) Init() tea.Cmd {
	return nil
}

func (m *notificationsDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *notificationsDialogModel) SetState(idx int) {
	m.stateIdx = idx
	m.notifs = newNotificationsDialogModel(idx).notifs
}

func (m *notificationsDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.notifs == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.notifs.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.notifs.HandleMsg(msg)
	}
	return m, nil
}

func (m *notificationsDialogModel) View() tea.View {
	if m.notifs == nil {
		return tea.NewView("Notifications dialog unavailable")
	}
	w := max(40, m.width)
	h := max(14, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.notifs.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  Notification Backend: [↑/↓] select • [Enter] confirm • Presets: [1] Auto  [2] Native  [3] OSC  [4] Disabled"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-notifications",
		Name:        "Notification Preferences",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Dialog to select completion notification backend: Auto-detect, Native OS popup, Terminal OSC 777/9, Bell, or Disabled.",
		UpstreamRef: "internal/ui/dialog/notifications.go",
		States: []storybook.State{
			{ID: "auto", Name: "Auto Detect", Description: "Automatically choose best available notification channel"},
			{ID: "native", Name: "Native Desktop", Description: "System notification center popups"},
			{ID: "osc", Name: "Terminal OSC", Description: "Terminal-native escape sequence notifications"},
			{ID: "disabled", Name: "Disabled", Description: "Suppress all audible and visual alerts"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newNotificationsDialogModel(0))
		},
	})
}
