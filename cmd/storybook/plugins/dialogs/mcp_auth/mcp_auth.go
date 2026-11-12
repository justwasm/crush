package mcp_auth

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	mcptools "github.com/charmbracelet/crush/internal/agent/tools/mcp"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type mcpAuthDialogModel struct {
	auth     *dialog.MCPAuth
	width    int
	height   int
	stateIdx int
}

func newMCPAuthDialogModel(stateIdx int) *mcpAuthDialogModel {
	com := mockws.NewCommon()
	pending := []mcptools.PendingAuthServer{
		{
			Name: "github-tools",
			URL:  "https://github.com/login/oauth/authorize?client_id=mcp-dev-client",
		},
		{
			Name: "linear-issues",
			URL:  "https://linear.app/oauth/authorize?client_id=mcp-linear",
		},
	}
	authFn := func(name string) string {
		return "https://auth.example.com/oauth/authorize?server=" + name
	}
	dlg, _ := dialog.NewMCPAuth(com, pending, authFn)
	return &mcpAuthDialogModel{
		auth:   dlg,
		width:  80,
		height: 22,
	}
}

func (m *mcpAuthDialogModel) Init() tea.Cmd {
	return nil
}

func (m *mcpAuthDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *mcpAuthDialogModel) SetState(idx int) {
	m.stateIdx = idx
	m.auth = newMCPAuthDialogModel(idx).auth
}

func (m *mcpAuthDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.auth == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.auth.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.auth.HandleMsg(msg)
	}
	return m, nil
}

func (m *mcpAuthDialogModel) View() tea.View {
	if m.auth == nil {
		return tea.NewView("MCP Auth dialog unavailable")
	}
	w := max(40, m.width)
	h := max(14, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.auth.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  MCP Server Auth: [c] copy auth URL • [Enter] open in browser • [s] skip • [Esc] close"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-mcp-auth",
		Name:        "MCP Server Auth",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Authorization dialog for third-party MCP servers with one-click URL copy shortcut (c/u) and browser sign-in.",
		UpstreamRef: "internal/ui/dialog/mcp_auth.go",
		States: []storybook.State{
			{ID: "prompt", Name: "Authorize Prompt", Description: "Pending authorization for connected MCP servers"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newMCPAuthDialogModel(0))
		},
	})
}
