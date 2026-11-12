package theme_editor

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type themeEditorDialogModel struct {
	editor   *dialog.ThemeEditor
	width    int
	height   int
	stateIdx int
}

func newThemeEditorDialogModel(stateIdx int) *themeEditorDialogModel {
	com := mockws.NewCommon()
	dlg := dialog.NewThemeEditor(com, "charmtone-panther")
	switch stateIdx {
	case 1:
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	case 2:
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	}
	return &themeEditorDialogModel{
		editor: dlg,
		width:  80,
		height: 24,
	}
}

func (m *themeEditorDialogModel) Init() tea.Cmd {
	return nil
}

func (m *themeEditorDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *themeEditorDialogModel) SetState(idx int) {
	m.stateIdx = idx
	m.editor = newThemeEditorDialogModel(idx).editor
}

func (m *themeEditorDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.editor == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.editor.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.editor.HandleMsg(msg)
	}
	return m, nil
}

func (m *themeEditorDialogModel) View() tea.View {
	if m.editor == nil {
		return tea.NewView("Theme Editor dialog unavailable")
	}
	w := max(40, m.width)
	h := max(16, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.editor.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  Theme Palette Editor: [Tab/↓] next color slot • [Shift+Tab/↑] previous • [Enter] save • Type hex codes to edit"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-theme-editor",
		Name:        "Theme Palette Editor",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Interactive in-TUI color palette editor with live preview, hex color validation, and slot cycling.",
		UpstreamRef: "internal/ui/dialog/theme_editor.go",
		States: []storybook.State{
			{ID: "slot0", Name: "Primary Color", Description: "First palette token selected"},
			{ID: "slot1", Name: "Secondary Color", Description: "Second palette token selected"},
			{ID: "slot2", Name: "Background Color", Description: "Surface background token selected"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newThemeEditorDialogModel(0))
		},
	})
}
