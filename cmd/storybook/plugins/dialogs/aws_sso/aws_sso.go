package aws_sso

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type awsSSODialogModel struct {
	sso      *dialog.AWSSSO
	width    int
	height   int
	stateIdx int
}

func newAWSSSODialogModel(stateIdx int) *awsSSODialogModel {
	com := mockws.NewCommon()
	dlg, _ := dialog.NewAWSSSO(com, "aws sso login --profile prod-engineer")
	switch stateIdx {
	case 1: // Initializing
		// No URL set, waiting state
	case 2: // Success
		dlg.Finish("")
	case 3: // Error
		dlg.Finish("AWS SSO token refresh request timed out after 180s")
	default: // Waiting with verification URL
		dlg.SetURL("https://device.sso.us-east-1.amazonaws.com/?user_code=ABCD-EFGH")
	}
	return &awsSSODialogModel{
		sso:    dlg,
		width:  80,
		height: 22,
	}
}

func (m *awsSSODialogModel) Init() tea.Cmd {
	return nil
}

func (m *awsSSODialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *awsSSODialogModel) SetState(idx int) {
	m.stateIdx = idx
	m.sso = newAWSSSODialogModel(idx).sso
}

func (m *awsSSODialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.sso == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.sso.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.sso.HandleMsg(msg)
	}
	return m, nil
}

func (m *awsSSODialogModel) View() tea.View {
	if m.sso == nil {
		return tea.NewView("AWS SSO dialog unavailable")
	}
	w := max(40, m.width)
	h := max(14, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.sso.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  AWS SSO Auth: [Enter] open in browser • [Esc] cancel • Presets: [1] URL Ready  [2] Waiting  [3] Success  [4] Error"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-aws-sso",
		Name:        "AWS SSO Authentication",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "AWS SSO credential refresh dialog with live spinner, verification URL display, and browser opening.",
		UpstreamRef: "internal/ui/dialog/aws_sso.go",
		States: []storybook.State{
			{ID: "url", Name: "URL Ready", Description: "Verification URL received with browser open hotkey"},
			{ID: "waiting", Name: "Waiting SSO", Description: "Command executing in background with spinner"},
			{ID: "success", Name: "SSO Success", Description: "Tokens refreshed successfully"},
			{ID: "error", Name: "SSO Error", Description: "Timeout or expired session notification"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newAWSSSODialogModel(0))
		},
	})
}
