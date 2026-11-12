package user

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/ui/chat"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/justwasm/storybook"
)

type userMessageModel struct {
	stateIdx int
	width    int
}

func (m *userMessageModel) Init() tea.Cmd {
	return nil
}

func (m *userMessageModel) SetSize(w, h int) {
	m.width = w
}

func (m *userMessageModel) SetState(idx int) {
	m.stateIdx = idx
}

func (m *userMessageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
	}
	return m, nil
}

func (m *userMessageModel) View() tea.View {
	st := styles.CharmtonePantera()
	w := max(40, m.width-4)

	var text string
	switch m.stateIdx {
	case 1:
		text = "Can you refactor the TUI layout to use a universal substrate pattern and extract common components into plugins?"
	case 2:
		text = "```go\nfunc BenchmarkRenderer(b *testing.B) {\n    // stress test frame cache\n}\n```\nPlease optimize this benchmark."
	default:
		text = "Hello! Please audit the workspace and run the test suite."
	}

	msg := message.Message{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: text},
		},
	}

	item := chat.NewUserMessageItem(&st, &msg, nil)
	rendered := item.RawRender(w)

	var b strings.Builder
	h1 := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render
	b.WriteString(h1("Chat: User Prompt Bubble") + "\n\n")
	b.WriteString(rendered)
	b.WriteString("\n\n  Presets: [1] Short Prompt  [2] Multi-line Refactor  [3] Code Snippet Query\n")

	return tea.NewView(b.String())
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "chat-user-message",
		Name:        "User Message Bubble",
		Category:    storybook.CategoryApps,
		Level:       storybook.LevelMolecule,
		Description: "User prompt message bubble with markdown formatting and timestamp alignment.",
		UpstreamRef: "internal/ui/chat/user.go",
		States: []storybook.State{
			{ID: "short", Name: "Short Prompt", Description: "Simple one-liner request"},
			{ID: "refactor", Name: "Refactor Query", Description: "Multi-line complex architectural question"},
			{ID: "code", Name: "Code Snippet", Description: "Prompt with embedded fenced code"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(&userMessageModel{width: 80})
		},
	})
}
