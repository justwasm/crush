package theme

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type themeDialogModel struct {
	theme  *dialog.Theme
	width  int
	height int
}

func newThemeDialogModel() *themeDialogModel {
	com := mockws.NewCommon()
	return &themeDialogModel{
		theme:  dialog.NewTheme(com),
		width:  80,
		height: 22,
	}
}

func (m *themeDialogModel) Init() tea.Cmd {
	return nil
}

func (m *themeDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *themeDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.theme.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.theme.HandleMsg(msg)
	}
	return m, nil
}

func (m *themeDialogModel) View() tea.View {
	w := max(40, m.width)
	h := max(14, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.theme.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	return tea.NewView(content)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-theme",
		Name:        "Theme Selector Dialog",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Interactive theme picker modal with live filtering, fuzzy search, and keyboard navigation.",
		UpstreamRef: "internal/ui/dialog/theme.go",
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newThemeDialogModel())
		},
	})
}
