package diff

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/justwasm/storybook"
)

type diffModel struct {
	stateIdx int
	width    int
}

func (m *diffModel) Init() tea.Cmd {
	return nil
}

func (m *diffModel) SetSize(w, h int) {
	m.width = w
}

func (m *diffModel) SetState(idx int) {
	m.stateIdx = idx
}

func (m *diffModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
	}
	return m, nil
}

const beforeCode = `package main

import "fmt"

func HandleTurn(turn int) error {
	fmt.Printf("Executing turn %d\n", turn)
	return nil
}
`

const afterCode = `package main

import (
	"fmt"
	"time"
)

func HandleTurn(turn int) error {
	start := time.Now()
	defer func() {
		fmt.Printf("Turn %d completed in %v\n", turn, time.Since(start))
	}()
	return nil
}
`

func (m *diffModel) View() tea.View {
	st := styles.CharmtonePantera()
	w := max(50, m.width-4)

	formatter := common.DiffFormatter(&st).
		Before("engine.go", beforeCode).
		After("engine.go", afterCode).
		Width(w)

	var label string
	if m.stateIdx == 1 { // Unified
		label = "Unified Diff Format"
	} else { // Split Side-by-Side
		formatter = formatter.Split()
		label = "Side-by-Side Split Diff Format"
	}

	rendered := formatter.String()

	var b strings.Builder
	h1 := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render
	b.WriteString(h1("Diff Viewer: "+label) + "\n\n")
	b.WriteString(rendered)
	b.WriteString("\n\n  Presets: [1] Split Side-by-Side  [2] Unified Diff\n")

	return tea.NewView(b.String())
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "chat-diff-viewer",
		Name:        "Diff Viewer",
		Category:    storybook.CategoryLists,
		Level:       storybook.LevelOrganism,
		Description: "Syntax-highlighted code diff inspection supporting side-by-side split and unified views.",
		UpstreamRef: "internal/ui/diffview/diffview.go",
		States: []storybook.State{
			{ID: "split", Name: "Split Side-by-Side", Description: "Two-column before & after comparison"},
			{ID: "unified", Name: "Unified Diff", Description: "Single column hunk additions and deletions"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(&diffModel{width: 80})
		},
	})
}
