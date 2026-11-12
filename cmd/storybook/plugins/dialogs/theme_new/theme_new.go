package theme_new

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type themeNewDialogModel struct {
	themeNew *dialog.ThemeNew
	width    int
	height   int
	stateIdx int
}

func newThemeNewDialogModel(stateIdx int) *themeNewDialogModel {
	com := mockws.NewCommon()
	dlg := dialog.NewThemeNew(com, "charmtone-panther")
	return &themeNewDialogModel{
		themeNew: dlg,
		width:    80,
		height:   18,
	}
}

func (m *themeNewDialogModel) Init() tea.Cmd {
	return nil
}

func (m *themeNewDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *themeNewDialogModel) SetState(idx int) {
	m.stateIdx = idx
	m.themeNew = newThemeNewDialogModel(idx).themeNew
}

func (m *themeNewDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.themeNew == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.themeNew.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.themeNew.HandleMsg(msg)
	}
	return m, nil
}

func (m *themeNewDialogModel) View() tea.View {
	if m.themeNew == nil {
		return tea.NewView("Theme New dialog unavailable")
	}
	w := max(40, m.width)
	h := max(12, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.themeNew.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  New Theme: Type theme name • [Enter] confirm • [Esc] cancel"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-theme-new",
		Name:        "Create New Theme",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Prompt dialog to name and scaffold a new custom theme inheriting from an existing base palette.",
		UpstreamRef: "internal/ui/dialog/theme_new.go",
		States: []storybook.State{
			{ID: "default", Name: "Prompt", Description: "Theme name input field with base inheritance indicator"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newThemeNewDialogModel(0))
		},
	})
}
