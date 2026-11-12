package sessions

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type sessionsDialogModel struct {
	sess     *dialog.Session
	width    int
	height   int
	stateIdx int
}

func newSessionsDialogModel(stateIdx int) *sessionsDialogModel {
	com := mockws.NewCommon()
	dlg, _ := dialog.NewSessions(com, "sess-storybook-migration")
	switch stateIdx {
	case 1: // Delete mode
		if dlg != nil {
			dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: 'd'}))
		}
	case 2: // Rename mode
		if dlg != nil {
			dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: 'r'}))
		}
	}
	return &sessionsDialogModel{
		sess:   dlg,
		width:  80,
		height: 22,
	}
}

func (m *sessionsDialogModel) Init() tea.Cmd {
	return nil
}

func (m *sessionsDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *sessionsDialogModel) SetState(idx int) {
	m.stateIdx = idx
	m.sess = newSessionsDialogModel(idx).sess
}

func (m *sessionsDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.sess == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.sess.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.sess.HandleMsg(msg)
	}
	return m, nil
}

func (m *sessionsDialogModel) View() tea.View {
	if m.sess == nil {
		return tea.NewView("Sessions dialog unavailable")
	}
	w := max(40, m.width)
	h := max(14, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.sess.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  Session Manager: [↑/↓] navigate • [Enter] switch • [r] rename • [d] delete • Presets: [1] List  [2] Delete  [3] Rename"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-sessions",
		Name:        "Session History Manager",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Interactive multi-session switcher with live fuzzy filtering, in-place title rename, and delete confirmation.",
		UpstreamRef: "internal/ui/dialog/sessions.go",
		States: []storybook.State{
			{ID: "normal", Name: "Session List", Description: "Browse past conversation sessions with metadata"},
			{ID: "delete", Name: "Delete Mode", Description: "Inline confirmation prompt to delete session"},
			{ID: "rename", Name: "Rename Mode", Description: "Inline text input to update session title"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newSessionsDialogModel(0))
		},
	})
}
