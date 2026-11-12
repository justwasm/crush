package mcp_toggles

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type mcpTogglesDialogModel struct {
	toggles  *dialog.MCPToggles
	width    int
	height   int
	stateIdx int
}

func newMCPTogglesDialogModel(stateIdx int) *mcpTogglesDialogModel {
	com := mockws.NewCommon()
	items := []dialog.MCPToggleItem{
		{
			Name:            "github",
			Disabled:        false,
			ConfigDisabled:  false,
			EnabledOverride: true,
			Status:          "connected (12 tools)",
		},
		{
			Name:            "filesystem",
			Disabled:        false,
			ConfigDisabled:  false,
			EnabledOverride: false,
			Status:          "connected (8 tools)",
		},
		{
			Name:            "postgres-db",
			Disabled:        true,
			ConfigDisabled:  true,
			EnabledOverride: false,
			Status:          "disabled by config",
		},
		{
			Name:            "slack-bot",
			Disabled:        false,
			ConfigDisabled:  false,
			EnabledOverride: false,
			Status:          "connecting...",
		},
	}
	dlg := dialog.NewMCPToggles(com, items)
	if stateIdx == 1 {
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	}
	return &mcpTogglesDialogModel{
		toggles: dlg,
		width:   80,
		height:  22,
	}
}

func (m *mcpTogglesDialogModel) Init() tea.Cmd {
	return nil
}

func (m *mcpTogglesDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *mcpTogglesDialogModel) SetState(idx int) {
	m.stateIdx = idx
	m.toggles = newMCPTogglesDialogModel(idx).toggles
}

func (m *mcpTogglesDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.toggles == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.toggles.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.toggles.HandleMsg(msg)
	}
	return m, nil
}

func (m *mcpTogglesDialogModel) View() tea.View {
	if m.toggles == nil {
		return tea.NewView("MCP Toggles dialog unavailable")
	}
	w := max(40, m.width)
	h := max(14, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.toggles.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  MCP Toggles: [Tab] switch Local/Global scope • [Space] toggle server • [↑/↓] navigate • Presets: [1] Local  [2] Global"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-mcp-toggles",
		Name:        "MCP Server Toggles",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Interactive toggle management for configured MCP servers with dual-scope overrides (Local repo vs Global config).",
		UpstreamRef: "internal/ui/dialog/mcp_toggles.go",
		States: []storybook.State{
			{ID: "local", Name: "Local Scope", Description: "Repository-scoped server activation overrides"},
			{ID: "global", Name: "Global Scope", Description: "Global user config persistent enable/disable"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newMCPTogglesDialogModel(0))
		},
	})
}
