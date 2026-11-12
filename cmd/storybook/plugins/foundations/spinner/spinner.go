package spinner

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/ui/anim"
	"github.com/justwasm/storybook"
)

type tickAnimMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(anim.FrameInterval(), func(t time.Time) tea.Msg {
		return tickAnimMsg(t)
	})
}

type animSpinnerModel struct {
	anim      *anim.Anim
	startTime time.Time
	stateIdx  int
}

func newAnimSpinnerModel(stateIdx int) *animSpinnerModel {
	start := time.Now()
	elapsedFn := func() string {
		dur := time.Since(start).Truncate(time.Second)
		return fmt.Sprintf(" (%s)", dur)
	}

	var a *anim.Anim
	switch stateIdx {
	case 1: // Running tool (NoScramble)
		a = anim.New(anim.Settings{
			ID:          "tool-spinner",
			Label:       "Executing bash script",
			NoScramble:  true,
			Suffix:      elapsedFn,
			LabelColor:  color.RGBA{R: 0x60, G: 0xa5, B: 0xfa, A: 0xff},
			SuffixColor: color.RGBA{R: 0x9c, G: 0xa3, B: 0xaf, A: 0xff},
		})
	case 2: // File operation
		a = anim.New(anim.Settings{
			ID:          "file-spinner",
			Label:       "Writing diff changes to internal/ui/model/ui.go",
			NoScramble:  true,
			Suffix:      elapsedFn,
			LabelColor:  color.RGBA{R: 0x34, G: 0xd3, B: 0x99, A: 0xff},
			SuffixColor: color.RGBA{R: 0x9c, G: 0xa3, B: 0xaf, A: 0xff},
		})
	case 3: // Fast cycling high entropy
		a = anim.New(anim.Settings{
			ID:          "entropy-spinner",
			Label:       "Synthesizing architecture plan",
			Size:        16,
			CycleColors: true,
			GradColorA:  color.RGBA{R: 0xf4, G: 0x3f, B: 0x5e, A: 0xff},
			GradColorB:  color.RGBA{R: 0x8b, G: 0x5c, B: 0xf6, A: 0xff},
			Suffix:      elapsedFn,
			LabelColor:  color.RGBA{R: 0xfb, G: 0xbf, B: 0x24, A: 0xff},
		})
	default: // Standard thinking matrix
		a = anim.New(anim.Settings{
			ID:          "thinking-spinner",
			Label:       "Thinking",
			Size:        10,
			CycleColors: true,
			GradColorA:  color.RGBA{R: 0xa7, G: 0x8b, B: 0xfa, A: 0xff},
			GradColorB:  color.RGBA{R: 0xec, G: 0x48, B: 0x99, A: 0xff},
			Suffix:      elapsedFn,
			LabelColor:  color.RGBA{R: 0xe5, G: 0xe7, B: 0xeb, A: 0xff},
		})
	}

	return &animSpinnerModel{
		anim:      a,
		startTime: start,
		stateIdx:  stateIdx,
	}
}

func (m *animSpinnerModel) Init() tea.Cmd {
	return tickCmd()
}

func (m *animSpinnerModel) SetState(idx int) {
	m.stateIdx = idx
	m.anim = newAnimSpinnerModel(idx).anim
	m.startTime = time.Now()
}

func (m *animSpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tickAnimMsg:
		m.anim.Advance()
		return m, tickCmd()
	}
	return m, nil
}

func (m *animSpinnerModel) View() tea.View {
	var b strings.Builder
	b.WriteString("Crush Proprietary Gradient & Thinking Spinner Engine\n\n")
	b.WriteString("  Live Animation Preview:\n  ")
	b.WriteString(m.anim.Render())
	b.WriteString("\n\n(Presets: [1] Thinking Matrix  [2] Tool Execution  [3] File Operation  [4] High Entropy Cycle)")
	return tea.NewView(b.String())
}

func init() {
	storybook.Register(storybook.Story{
		ID:          "foundations-spinner",
		Name:        "Gradient & Thinking Spinner",
		Category:    storybook.CategoryFoundations,
		Level:       storybook.LevelAtom,
		Description: "Crush's proprietary animated gradient braille/matrix characters spinner with elapsed timer and scrambled glyph modes.",
		UpstreamRef: "internal/ui/anim/anim.go",
		States: []storybook.State{
			{ID: "thinking", Name: "Thinking Matrix", Description: "Staggered cycling runes with violet/pink gradient"},
			{ID: "tool", Name: "Tool Running", Description: "Clean ellipsis indicator for background shell execution"},
			{ID: "file", Name: "File Operation", Description: "Green accented status for disk modifications"},
			{ID: "entropy", Name: "High Entropy", Description: "Expanded 16-rune scrambled deliberation sequence"},
		},
		Factory: func() storybook.Instance {
			return storybook.WrapModel(newAnimSpinnerModel(0))
		},
	})
}
