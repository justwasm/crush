package question_form

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/cmd/storybook/mockws"
	"github.com/charmbracelet/crush/internal/question"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/justwasm/storybook"
)

type questionFormDialogModel struct {
	form     *dialog.QuestionForm
	width    int
	height   int
	stateIdx int
}

func newQuestionFormDialogModel(stateIdx int) *questionFormDialogModel {
	com := mockws.NewCommon()
	batch := question.Request{
		ID:           "clarify-batch-1",
		ConfirmTitle: "Confirm",
		Questions: []question.Question{
			{
				ID:    "q-yesno",
				Type:  question.TypeYesNo,
				Text:  "Should the agent proceed with automatic file modifications without extra confirmation?",
				Label: "Auto Modify",
			},
			{
				ID:    "q-tier",
				Type:  question.TypeSingleChoice,
				Text:  "Select the architectural tier where this component belongs:",
				Label: "Target Tier",
				Choices: []question.Choice{
					{ID: "t1", Label: "Tier 1: Core (Zero Dependencies)"},
					{ID: "t2", Label: "Tier 2: Features (Auth, Models)"},
					{ID: "t3", Label: "Tier 3: Infra (Backends, IPC)"},
					{ID: "t4", Label: "Tier 4: App (Agents, Session)"},
				},
			},
			{
				ID:    "q-scope",
				Type:  question.TypeMultiChoice,
				Text:  "Choose all target runtime environments to verify:",
				Label: "Environments",
				Choices: []question.Choice{
					{ID: "darwin", Label: "macOS (Darwin ARM64)"},
					{ID: "linux", Label: "Linux (Debian x86_64)"},
					{ID: "wasm", Label: "WebAssembly / WASI"},
				},
			},
		},
	}
	form := dialog.NewQuestionForm(com.Styles, batch)
	switch stateIdx {
	case 1: // Tab 2: Target Tier
		form.HandleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	case 2: // Tab 3: Environments
		form.HandleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
		form.HandleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	case 3: // Tab 4: Confirm
		form.HandleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
		form.HandleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
		form.HandleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	}
	return &questionFormDialogModel{
		form:   form,
		width:  80,
		height: 22,
	}
}

func (m *questionFormDialogModel) Init() tea.Cmd {
	return nil
}

func (m *questionFormDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *questionFormDialogModel) SetState(idx int) {
	m.stateIdx = idx
	m.form = newQuestionFormDialogModel(idx).form
}

func (m *questionFormDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.form == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		_, cmd := m.form.HandleKey(msg)
		return m, cmd
	}
	return m, nil
}

func (m *questionFormDialogModel) View() tea.View {
	if m.form == nil {
		return tea.NewView("Question form unavailable")
	}
	w := max(40, m.width)
	h := max(14, m.height)

	canvas := uv.NewScreenBuffer(w, h)
	m.form.Draw(canvas, uv.Rect(0, 0, w, h))

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	content = strings.Join(lines, "\n")

	hint := "\n\n  Interactive Clarification Form: [Tab/Shift+Tab] switch tabs • [↑/↓] select option • [Enter] answer • Presets: [1] Yes/No  [2] Choice  [3] Multi  [4] Confirm"
	return tea.NewView(content + hint)
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "dialog-question-form",
		Name:        "Clarification Questions Form",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelModal,
		Description: "Tabbed interactive clarification form combining Yes/No, single-choice, multi-choice, and batch confirmation tabs.",
		UpstreamRef: "internal/ui/dialog/question_form.go",
		States: []storybook.State{
			{ID: "tab1", Name: "Yes/No Question", Description: "Boolean choice prompt with immediate keyboard actions"},
			{ID: "tab2", Name: "Single Choice", Description: "Radio list of mutually exclusive architectural options"},
			{ID: "tab3", Name: "Multi Choice", Description: "Checkbox grid of selectable target environments"},
			{ID: "confirm", Name: "Confirm Summary", Description: "Final review tab summarizing all recorded answers"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newQuestionFormDialogModel(0))
		},
	})
}
