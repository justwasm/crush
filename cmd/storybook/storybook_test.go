package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	_ "github.com/charmbracelet/crush/cmd/storybook/plugins/all"
	"github.com/justwasm/storybook"
)

func TestCatalogIntegrity(t *testing.T) {
	stories := storybook.DefaultRegistry.Stories()
	if len(stories) == 0 {
		t.Fatalf("expected registered stories in storybook, got 0")
	}

	t.Logf("Total registered Crush UI stories: %d", len(stories))

	seenIDs := make(map[string]bool)
	for _, s := range stories {
		if s.ID == "" {
			t.Errorf("story has empty ID: %+v", s)
		}
		if seenIDs[s.ID] {
			t.Errorf("duplicate story ID: %s", s.ID)
		}
		seenIDs[s.ID] = true

		if s.Name == "" {
			t.Errorf("story %s has empty Name", s.ID)
		}
		if s.Category == "" {
			t.Errorf("story %s has empty Category", s.ID)
		}
		if s.Description == "" {
			t.Errorf("story %s has empty Description", s.ID)
		}
		if s.Factory == nil {
			t.Errorf("story %s has nil Factory", s.ID)
		}
	}
}

func TestAllStoriesRenderAcrossAllStates(t *testing.T) {
	stories := storybook.DefaultRegistry.Stories()

	for _, s := range stories {
		t.Run(s.ID, func(t *testing.T) {
			statesCount := len(s.States)
			if statesCount == 0 {
				statesCount = 1
			}

			for stateIdx := 0; stateIdx < statesCount; stateIdx++ {
				inst := s.Factory()
				if inst == nil {
					t.Fatalf("factory returned nil instance for %s", s.ID)
				}

				if r, ok := inst.(storybook.Resizable); ok {
					r.SetSize(80, 24)
				}
				if st, ok := inst.(storybook.Stateful); ok && len(s.States) > 0 {
					st.SetState(stateIdx)
				}

				view := inst.View()
				if len(strings.TrimSpace(view)) == 0 {
					t.Errorf("empty view rendered for %s (state %d)", s.ID, stateIdx)
				}
			}
		})
	}
}

func TestInteractiveDialogKeyHandling(t *testing.T) {
	stories := storybook.DefaultRegistry.Stories()
	var quitStory *storybook.Story
	for _, s := range stories {
		if s.ID == "dialog-quit" {
			sc := s
			quitStory = &sc
			break
		}
	}

	if quitStory == nil {
		t.Fatalf("dialog-quit story not found")
	}

	inst := quitStory.Factory()
	if r, ok := inst.(storybook.Resizable); ok {
		r.SetSize(80, 20)
	}

	// Deliver right arrow key
	newInst, cmd := inst.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	if newInst == nil {
		t.Fatalf("instance became nil after key update")
	}
	_ = cmd

	v := newInst.View()
	if !strings.Contains(v, "Are you sure you want to quit?") {
		t.Fatalf("expected quit question in view, got: %s", v)
	}
}
