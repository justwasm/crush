package assistant

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/ui/chat"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/justwasm/storybook"
)

type assistantMessageModel struct {
	stateIdx int
	width    int
}

func (m *assistantMessageModel) Init() tea.Cmd {
	return nil
}

func (m *assistantMessageModel) SetSize(w, h int) {
	m.width = w
}

func (m *assistantMessageModel) SetState(idx int) {
	m.stateIdx = idx
}

func (m *assistantMessageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
	}
	return m, nil
}

func (m *assistantMessageModel) View() tea.View {
	st := styles.CharmtonePantera()
	w := max(40, m.width-4)

	var parts []message.ContentPart

	switch m.stateIdx {
	case 1: // Thinking only
		parts = []message.ContentPart{
			message.ReasoningContent{Thinking: "1. Inspecting package dependency graph\n2. Verifying bubbletea v2 event loop contracts\n3. Resolving sub-canvas coordinate mappings"},
		}
	case 2: // Plan handoff
		parts = []message.ContentPart{
			message.TextContent{Text: "### Implementation Plan\n\n- [x] Extract `github.com/justwasm/storybook` substrate\n- [x] Support `RegisterModel` zero-invasive wrapping\n- [ ] Migrate all 15+ Crush UI components"},
		}
	default: // Complete Turn (Thinking + Final Answer)
		parts = []message.ContentPart{
			message.ReasoningContent{Thinking: "Synthesizing full component inventory. All dialogs, chat bubbles, and brand assets are modularized into independent plugins."},
			message.TextContent{Text: "I have verified the component architecture. Everything builds cleanly with **100% host isolation** and zero cyclic dependencies."},
		}
	}

	msg := message.Message{
		ID:        "msg-assistant-1",
		SessionID: "sess-1",
		Role:      message.Assistant,
		Model:     "claude-3-7-sonnet",
		Provider:  "anthropic",
		Parts:     parts,
	}

	item := chat.NewAssistantMessageItem(&st, &msg)
	rendered := item.RawRender(w)

	var b strings.Builder
	h1 := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render
	b.WriteString(h1("Chat: Assistant Turn with Thinking Window") + "\n\n")
	b.WriteString(rendered)
	b.WriteString("\n\n  Presets: [1] Full Turn  [2] Thinking Window  [3] Plan Output\n")

	return tea.NewView(b.String())
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "chat-assistant-message",
		Name:        "Assistant Message",
		Category:    storybook.CategoryApps,
		Level:       storybook.LevelOrganism,
		Description: "Assistant response card with collapsible reasoning/thinking window and markdown streaming view.",
		UpstreamRef: "internal/ui/chat/assistant.go",
		States: []storybook.State{
			{ID: "complete", Name: "Complete Turn", Description: "Thinking block followed by response"},
			{ID: "thinking", Name: "Thinking Only", Description: "Active or collapsed reasoning window"},
			{ID: "plan", Name: "Plan Output", Description: "Markdown task checklist"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(&assistantMessageModel{width: 80})
		},
	})
}
