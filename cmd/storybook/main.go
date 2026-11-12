package main

import (
	"fmt"
	"os"

	_ "github.com/charmbracelet/crush/cmd/storybook/plugins/all"
	"github.com/justwasm/storybook"
)

func main() {
	if err := storybook.Run(storybook.WithTitle("❖ CRUSH TUI Storybook — Agentic Workspace Components")); err != nil {
		fmt.Fprintf(os.Stderr, "Storybook error: %v\n", err)
		os.Exit(1)
	}
}
