package arguments

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/commands"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type argumentsDialogModel struct {
	args     *dialog.Arguments
	width    int
	height   int
	stateIdx int
}

func newArgumentsDialogModel(stateIdx int) *argumentsDialogModel {
	com := mockws.NewCommon()
	cmdArgs := []commands.Argument{
		{
			ID:          "target_branch",
			Title:       "Target Branch",
			Description: "Name of the target git branch to rebase onto (e.g. main, develop)",
			Required:    true,
		},
		{
			ID:          "commit_message",
			Title:       "Commit Message",
			Description: "Summary header for squashed pull request changes",
			Required:    true,
		},
		{
			ID:          "extra_flags",
			Title:       "Extra Flags",
			Description: "Optional CLI parameters (e.g. --force-with-lease)",
			Required:    false,
		},
	}
	dlg := dialog.NewArguments(com, "Execute Git Rebase", "Configure arguments for /rebase workflow", cmdArgs, nil)
	switch stateIdx {
	case 1:
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	case 2:
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	}
	return &argumentsDialogModel{
		args:   dlg,
		width:  80,
		height: 24,
	}
}

func (m *argumentsDialogModel) Init() tea.Cmd {
	return nil
}

func (m *argumentsDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *argumentsDialogModel) SetState(idx int) {
	m.stateIdx = idx
	m.args = newArgumentsDialogModel(idx).args
}

func (m *argumentsDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.args == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.args.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.args.HandleMsg(msg)
	}
	return m, nil
}

func (m *argumentsDialogModel) View() tea.View {
	if m.args == nil {
		return tea.NewView("Arguments dialog unavailable")
	}
	w := max(40, m.width)
	h := max(16, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.args.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  Command Arguments: [Tab/↓] next field • [Shift+Tab/↑] previous • [Enter] confirm • Presets: [1] Field 1  [2] Field 2  [3] Field 3"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-arguments",
		Name:        "Command Arguments Form",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Interactive multi-field parameter entry modal for slash commands and MCP prompt execution.",
		UpstreamRef: "internal/ui/dialog/arguments.go",
		States: []storybook.State{
			{ID: "field1", Name: "First Field Focused", Description: "Target Branch input active"},
			{ID: "field2", Name: "Second Field Focused", Description: "Commit Message input active"},
			{ID: "field3", Name: "Third Field Focused", Description: "Extra Flags optional field active"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newArgumentsDialogModel(0))
		},
	})
}
