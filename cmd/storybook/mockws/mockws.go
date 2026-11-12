package mockws

import (
	"context"
	"time"

	"charm.land/catwalk/v3/pkg/catwalk"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/csync"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/workspace"
)

type MockWorkspace struct {
	workspace.Workspace
	Cfg *config.Config
}

func (w *MockWorkspace) Config() *config.Config {
	if w.Cfg == nil {
		providersMap := csync.NewMap[string, config.ProviderConfig]()
		providersMap.Set("anthropic", config.ProviderConfig{
			ID:   "anthropic",
			Name: "Anthropic",
			Models: []catwalk.Model{
				{
					ID:   "claude-3-7-sonnet",
					Name: "Claude 3.7 Sonnet",
					Reasoning: catwalk.Reasoning{
						Thinking:           catwalk.ThinkingToggleable,
						EffortLevels:       catwalk.NewEffortLevels("low", "medium", "high"),
						DefaultEffortLevel: "medium",
					},
				},
			},
		})

		w.Cfg = &config.Config{
			Options: &config.Options{
				TUI:                        &config.TUIOptions{},
				DisableProviderAutoUpdate: true,
			},
			Providers: providersMap,
			Agents: map[string]config.Agent{
				config.AgentCoder: {Model: config.SelectedModelTypeLarge},
			},
			Models: map[config.SelectedModelType]config.SelectedModel{
				config.SelectedModelTypeLarge: {
					Provider:        "anthropic",
					Model:           "claude-3-7-sonnet",
					ReasoningEffort: "medium",
				},
			},
		}
	}
	return w.Cfg
}

func (w *MockWorkspace) WorkingDir() string {
	return "/Users/gear/GitHub/crush"
}

func (w *MockWorkspace) ListSessions(ctx context.Context) ([]session.Session, error) {
	now := time.Now().Unix()
	return []session.Session{
		{
			ID:           "sess-storybook-migration",
			Title:        "Refactor Storybook integration and dual-axis preset support",
			MessageCount: 42,
			PromptTokens: 12540,
			Cost:         0.084,
			UpdatedAt:    now - 120,
			CreatedAt:    now - 3600,
		},
		{
			ID:           "sess-bubbletea-v2",
			Title:        "Upgrade project dependencies to Bubble Tea v2 and Lip Gloss v2",
			MessageCount: 18,
			PromptTokens: 8400,
			Cost:         0.032,
			UpdatedAt:    now - 86400,
			CreatedAt:    now - 90000,
		},
		{
			ID:           "sess-catwalk-providers",
			Title:        "Audit catwalk model catalog sync and hyper endpoints",
			MessageCount: 9,
			PromptTokens: 4120,
			Cost:         0.015,
			UpdatedAt:    now - 172800,
			CreatedAt:    now - 180000,
		},
	}, nil
}

// NewCommon returns a Common with a safe mock workspace and config.
func NewCommon() *common.Common {
	ws := &MockWorkspace{}
	return common.DefaultCommon(ws)
}
