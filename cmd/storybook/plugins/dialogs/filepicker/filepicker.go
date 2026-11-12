package filepicker

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type filePickerDialogModel struct {
	picker   *dialog.FilePicker
	width    int
	height   int
	stateIdx int
}

func newFilePickerDialogModel(stateIdx int) *filePickerDialogModel {
	com := mockws.NewCommon()
	dlg, cmd := dialog.NewFilePicker(com)
	// Synchronously execute the init command to populate directory files immediately
	if cmd != nil {
		if msg := cmd(); msg != nil {
			dlg.HandleMsg(msg)
		}
	}
	switch stateIdx {
	case 1:
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	case 2:
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	}
	return &filePickerDialogModel{
		picker: dlg,
		width:  80,
		height: 24,
	}
}

func (m *filePickerDialogModel) Init() tea.Cmd {
	return nil
}

func (m *filePickerDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *filePickerDialogModel) SetState(idx int) {
	m.stateIdx = idx
	m.picker = newFilePickerDialogModel(idx).picker
}

func (m *filePickerDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.picker == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.picker.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.picker.HandleMsg(msg)
	}
	return m, nil
}

func (m *filePickerDialogModel) View() tea.View {
	if m.picker == nil {
		return tea.NewView("FilePicker dialog unavailable")
	}
	w := max(40, m.width)
	h := max(16, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.picker.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  FilePicker Dialog: [↑/↓] or [j/k] navigate • [→/l] enter directory • [←/h] go back • [Enter] select file"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-filepicker",
		Name:        "FilePicker Modal",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Interactive modal file browser supporting directory drilling, image previews, and file selection.",
		UpstreamRef: "internal/ui/dialog/filepicker.go",
		States: []storybook.State{
			{ID: "root", Name: "Working Directory", Description: "Workspace project root file list"},
			{ID: "down1", Name: "Second Entry Selected", Description: "Cursor down one item"},
			{ID: "down2", Name: "Third Entry Selected", Description: "Cursor down two items"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newFilePickerDialogModel(0))
		},
	})
}
