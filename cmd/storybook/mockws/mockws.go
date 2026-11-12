package mockws

import (
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/workspace"
)

type MockWorkspace struct {
	workspace.Workspace
	Cfg *config.Config
}

func (w *MockWorkspace) Config() *config.Config {
	if w.Cfg == nil {
		w.Cfg = &config.Config{
			Options: &config.Options{
				TUI: &config.TUIOptions{},
			},
		}
	}
	return w.Cfg
}

func (w *MockWorkspace) WorkingDir() string {
	return "/Users/gear/GitHub/crush"
}

// NewCommon returns a Common with a safe mock workspace and config.
func NewCommon() *common.Common {
	ws := &MockWorkspace{}
	return common.DefaultCommon(ws)
}
