package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/charmbracelet/crush/internal/appserver"
	"github.com/charmbracelet/crush/internal/backend"
	"github.com/charmbracelet/crush/internal/config"
	crushlog "github.com/charmbracelet/crush/internal/log"
	"github.com/spf13/cobra"
)

var (
	appServerListen string
)

func init() {
	appServerCmd.Flags().StringVar(&appServerListen, "listen", "stdio://", "Listen address (default: stdio://)")
	rootCmd.AddCommand(appServerCmd)
}

var appServerCmd = &cobra.Command{
	Use:   "app-server",
	Short: "Start the Crush app-server (NDJSON over stdio / JSON-RPC 2.0)",
	Long: `Start the Crush app-server for subprocess integration via NDJSON over stdio.
Communicates using JSON-RPC 2.0, supporting interactive turn streaming, reasoning deltas,
and tool calls compatible with pigasus and other AI harnesses.

Usage:
  crush app-server --listen stdio://
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		dataDir, err := cmd.Flags().GetString("data-dir")
		if err != nil {
			return fmt.Errorf("failed to get data directory: %v", err)
		}
		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			return fmt.Errorf("failed to get debug flag: %v", err)
		}
		cwd, err := cmd.Flags().GetString("cwd")
		if err != nil {
			return fmt.Errorf("failed to get cwd flag: %v", err)
		}
		if cwd == "" {
			cwd = config.GlobalWorkspaceDir()
		}

		// Ensure logs go to stderr or file, NEVER stdout (which is the NDJSON channel).
		logFile := filepath.Join(config.GlobalCacheDir(), "app-server", "crush.log")
		crushlog.Setup(logFile, debug, os.Stderr)

		cfg, err := config.Load(cmd.Context(), cwd, dataDir, debug)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %v", err)
		}

		b := backend.New(context.Background(), cfg, nil)
		defer b.ShutdownWorkspaces(context.Background())

		srv := appserver.NewServer(cfg, b, slog.Default())
		return srv.Serve(cmd.Context(), os.Stdin, os.Stdout)
	},
}
