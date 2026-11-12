package oauth

import (
	"errors"
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

type oauthDialogModel struct {
	oauth    *dialog.OAuth
	width    int
	height   int
	stateIdx int
}

func newOAuthDialogModel(stateIdx int) *oauthDialogModel {
	com := mockws.NewCommon()
	prov := catwalk.Provider{
		ID:   "github",
		Name: "GitHub Copilot",
	}
	model := config.SelectedModel{
		Provider: "github",
		Model:    "gpt-4o",
	}
	dlg, _ := dialog.NewOAuthCopilot(com, false, prov, model, config.SelectedModelTypeLarge)
	switch stateIdx {
	case 1: // Initializing
		// Default State is OAuthStateInitializing
	case 2: // Success
		dlg.HandleMsg(dialog.ActionInitiateOAuth{
			UserCode:        "WXYZ-9876",
			VerificationURL: "https://github.com/login/device",
			ExpiresIn:       900,
		})
		dlg.State = dialog.OAuthStateSuccess
	case 3: // Error
		dlg.HandleMsg(dialog.ActionOAuthErrored{
			Error: errors.New("authorization expired or cancelled by user"),
		})
	default: // Displaying Code (standard)
		dlg.HandleMsg(dialog.ActionInitiateOAuth{
			UserCode:        "WD7S-29BA",
			VerificationURL: "https://github.com/login/device",
			ExpiresIn:       900,
			Interval:        5,
		})
	}
	return &oauthDialogModel{
		oauth:  dlg,
		width:  80,
		height: 24,
	}
}

func (m *oauthDialogModel) Init() tea.Cmd {
	return nil
}

func (m *oauthDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *oauthDialogModel) SetState(idx int) {
	m.stateIdx = idx
	m.oauth = newOAuthDialogModel(idx).oauth
}

func (m *oauthDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.oauth == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_ = m.oauth.HandleMsg(msg)
	case tea.MouseMsg:
		_ = m.oauth.HandleMsg(msg)
	}
	return m, nil
}

func (m *oauthDialogModel) View() tea.View {
	if m.oauth == nil {
		return tea.NewView("OAuth dialog unavailable")
	}
	w := max(40, m.width)
	h := max(16, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.oauth.Draw(canvas, image.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  Device OAuth: [c] copy code • [u] copy URL • [Enter] copy & open • Presets: [1] Code  [2] Initializing  [3] Success  [4] Error"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-oauth",
		Name:        "OAuth Device Flow",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Device code authorization dialog with large verification code box, one-click copy shortcuts (c/u), and browser launch.",
		UpstreamRef: "internal/ui/dialog/oauth.go",
		States: []storybook.State{
			{ID: "display", Name: "Device Code Display", Description: "Rendered verification code box with copy hotkeys"},
			{ID: "init", Name: "Initializing", Description: "Connecting to provider authentication service"},
			{ID: "success", Name: "Authentication Success", Description: "Token persisted and models ready"},
			{ID: "error", Name: "Auth Failure", Description: "Timeout or rejection error display"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newOAuthDialogModel(0))
		},
	})
}
