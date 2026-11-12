package permissions

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type permissionsDialogModel struct {
	perm   *dialog.Permissions
	width  int
	height int
}

func newPermissionsDialogModel() *permissionsDialogModel {
	com := common.DefaultCommon(nil)
	req := permission.PermissionRequest{
		ID:          "perm-exec-01",
		ToolName:    "bash",
		Description: "Execute shell command in repository root",
		Action:      "git push origin main",
		Path:        "/Users/gear/GitHub/crush",
	}
	return &permissionsDialogModel{
		perm:   dialog.NewPermissions(com, req),
		width:  80,
		height: 22,
	}
}

func (m *permissionsDialogModel) Init() tea.Cmd {
	return nil
}

func (m *permissionsDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *permissionsDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.perm == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.perm.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.perm.HandleMsg(msg)
	}
	return m, nil
}

func (m *permissionsDialogModel) View() tea.View {
	if m.perm == nil {
		return tea.NewView("Permissions dialog unavailable")
	}
	w := max(40, m.width)
	h := max(14, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.perm.Draw(canvas, image.Rect(0, 0, w, h))

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
		ID:          "dialog-permissions",
		Name:        "Permission Request Modal",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Interactive tool execution approval dialog with Allow Once, Allow for Session, and Deny choices.",
		UpstreamRef: "internal/ui/dialog/permissions.go",
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newPermissionsDialogModel())
		},
	})
}
