package diffview

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/ui/diffview"
	"github.com/justwasm/storybook"
)

const beforeSample = `package main

import (
	"fmt"
	"os"
)

func runWorker(id int) {
	fmt.Printf("Worker %d starting\n", id)
}

func main() {
	fmt.Println("Legacy runner starting...")
	runWorker(1)
	os.Exit(0)
}
`

const afterSample = `package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

func runWorker(ctx context.Context, id int) error {
	fmt.Printf("Worker %d starting with context\n", id)
	select {
	case <-time.After(100 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	fmt.Println("Modern concurrent runner active")
	if err := runWorker(ctx, 1); err != nil {
		fmt.Fprintf(os.Stderr, "Worker error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
`

type diffviewModel struct {
	dv       *diffview.DiffView
	width    int
	stateIdx int
}

func newDiffviewModel(stateIdx int) *diffviewModel {
	dv := diffview.New().
		Before("runner.go", beforeSample).
		After("runner.go", afterSample)

	switch stateIdx {
	case 1: // Split
		dv.Split()
	case 2: // Light Style
		dv.Style(diffview.DefaultLightStyle()).Unified()
	default: // Unified
		dv.Unified()
	}

	return &diffviewModel{
		dv:       dv,
		width:    80,
		stateIdx: stateIdx,
	}
}

func (m *diffviewModel) Init() tea.Cmd {
	return nil
}

func (m *diffviewModel) SetSize(w, h int) {
	m.width = w
}

func (m *diffviewModel) SetState(idx int) {
	m.stateIdx = idx
	m.dv = newDiffviewModel(idx).dv
}

func (m *diffviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			if m.stateIdx == 0 {
				m.SetState(1)
			} else {
				m.SetState(0)
			}
		}
	}
	return m, nil
}

func (m *diffviewModel) View() tea.View {
	if m.dv == nil {
		return tea.NewView("DiffView unavailable")
	}
	w := max(50, m.width)
	m.dv.Width(w)
	var b strings.Builder
	b.WriteString("Crush Syntax-Highlighted Diff Viewer\n\n")
	b.WriteString(m.dv.String())
	b.WriteString("\n\n(Press [Tab] to toggle Unified/Split • Presets: [1] Unified  [2] Split Side-by-Side  [3] Light Theme)")
	return tea.NewView(b.String())
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "common-diffview",
		Name:        "Diff Viewer",
		Category:    storybook.CategoryPanels,
		Level:       storybook.LevelOrganism,
		Description: "Syntax-highlighted git diff comparison viewer supporting unified inline and split side-by-side layouts.",
		UpstreamRef: "internal/ui/diffview/diffview.go",
		States: []storybook.State{
			{ID: "unified", Name: "Unified Diff", Description: "Standard single-column git patch view with line numbers"},
			{ID: "split", Name: "Split Side-by-Side", Description: "Dual-pane before vs after visual comparison"},
			{ID: "light", Name: "Light Theme", Description: "High contrast bright background color palette"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newDiffviewModel(0))
		},
	})
}
