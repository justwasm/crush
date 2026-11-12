package auth_method

import (
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/catwalk/v3/pkg/catwalk"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type authMethodDialogModel struct {
	auth   *dialog.AuthMethod
	width  int
	height int
}

func newAuthMethodDialogModel(selected int, onboarding bool) *authMethodDialogModel {
	com := mockws.NewCommon()
	prov := catwalk.Provider{
		ID:   "openai",
		Name: "OpenAI",
	}
	model := config.SelectedModel{
		Provider: "openai",
		Model:    "gpt-4o",
	}
	dlg := dialog.NewAuthMethod(com, onboarding, prov, model, config.SelectedModelTypeLarge)
	if selected == 1 {
		dlg.HandleMsg(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	}
	return &authMethodDialogModel{
		auth:   dlg,
		width:  80,
		height: 22,
	}
}

func (m *authMethodDialogModel) Init() tea.Cmd {
	return nil
}

func (m *authMethodDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *authMethodDialogModel) SetState(idx int) {
	switch idx {
	case 1:
		m.auth = newAuthMethodDialogModel(1, false).auth
	case 2:
		m.auth = newAuthMethodDialogModel(0, true).auth
	default:
		m.auth = newAuthMethodDialogModel(0, false).auth
	}
}

func (m *authMethodDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m *authMethodDialogModel) View() tea.View {
	if m.auth == nil {
		return tea.NewView("Auth method dialog unavailable")
	}
	w := max(40, m.width)
	h := max(16, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.auth.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  Auth Method Card Selector: [←/→] choose card • [Enter] accept • Presets: [1] OAuth  [2] API Key  [3] Onboarding"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-auth-method",
		Name:        "Auth Method Selector",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Dual-card selection dialog for choosing between OAuth browser login and manual API Key configuration.",
		UpstreamRef: "internal/ui/dialog/auth_method.go",
		States: []storybook.State{
			{ID: "oauth", Name: "OAuth Focused", Description: "Browser sign-in card selected"},
			{ID: "apikey", Name: "API Key Focused", Description: "Manual token entry card selected"},
			{ID: "onboarding", Name: "Onboarding Context", Description: "Initial launch provider credential flow"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newAuthMethodDialogModel(0, false))
		},
	})
}
