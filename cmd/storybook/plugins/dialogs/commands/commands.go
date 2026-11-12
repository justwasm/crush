package commands

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type commandsDialogModel struct {
	commands *dialog.Commands
	width    int
	height   int
}

func newCommandsDialogModel() *commandsDialogModel {
	com := mockws.NewCommon()
	cmds, _ := dialog.NewCommands(com, "session-dev-1", true, false, false, false, nil, nil)
	return &commandsDialogModel{
		commands: cmds,
		width:    80,
		height:   22,
	}
}

func (m *commandsDialogModel) Init() tea.Cmd {
	return nil
}

func (m *commandsDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *commandsDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.commands == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.commands.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.commands.HandleMsg(msg)
	}
	return m, nil
}

func (m *commandsDialogModel) View() tea.View {
	if m.commands == nil {
		return tea.NewView("Commands dialog unavailable")
	}
	w := max(40, m.width)
	h := max(14, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.commands.Draw(canvas, image.Rect(0, 0, w, h))

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
		ID:          "dialog-commands",
		Name:        "Commands Palette",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Interactive slash commands palette with filter input, description list, and hotkey selection.",
		UpstreamRef: "internal/ui/dialog/commands.go",
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newCommandsDialogModel())
		},
	})
}
