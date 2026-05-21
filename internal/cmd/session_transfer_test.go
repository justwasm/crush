package cmd

import (
	"bytes"
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/db"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestSessionCmd_TransferSubcommands(t *testing.T) {
	t.Parallel()

	require.Equal(t, sessionExportCmd, findSessionSubcommand(t, "export"))
	require.Equal(t, sessionImportCmd, findSessionSubcommand(t, "import"))
}

func TestSessionExportImportRoundTrip(t *testing.T) {
	ctx := t.Context()
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	seedSessionTransferFixture(t, ctx, srcDir)

	srcConn, err := db.Connect(ctx, srcDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Release(srcDir))
	})

	var exported bytes.Buffer
	srcServices := &sessionServices{
		q:    db.New(srcConn),
		conn: srcConn,
	}
	exportedCount, err := exportSessionTransfers(ctx, srcServices, &exported)
	require.NoError(t, err)
	require.Equal(t, 2, exportedCount)

	dstConn, err := db.Connect(ctx, dstDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Release(dstDir))
	})

	importedCount, err := importSessionTransfers(ctx, dstConn, bytes.NewReader(exported.Bytes()))
	require.NoError(t, err)
	require.Equal(t, 2, importedCount)

	var reexported bytes.Buffer
	dstServices := &sessionServices{
		q:    db.New(dstConn),
		conn: dstConn,
	}
	reexportedCount, err := exportSessionTransfers(ctx, dstServices, &reexported)
	require.NoError(t, err)
	require.Equal(t, 2, reexportedCount)

	require.JSONEq(t, "["+chompJSONL(exported.String())+"]", "["+chompJSONL(reexported.String())+"]")
}

func seedSessionTransferFixture(t *testing.T, ctx context.Context, dataDir string) {
	t.Helper()

	conn, err := db.Connect(ctx, dataDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Release(dataDir))
	})

	tx, err := conn.BeginTx(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	mustExec := func(query string, args ...any) {
		t.Helper()
		_, err := tx.ExecContext(ctx, query, args...)
		require.NoError(t, err)
	}

	mustExec(insertSessionQuery,
		"session-parent",
		sql.NullString{},
		"Parent session",
		0,
		42,
		21,
		1.5,
		100,
		90,
		sql.NullString{String: "message-parent-1", Valid: true},
		sql.NullString{String: `[{"content":"Ship it","status":"pending","active_form":"shipping"}]`, Valid: true},
	)
	mustExec(insertSessionQuery,
		"session-child",
		sql.NullString{String: "session-parent", Valid: true},
		"Child session",
		0,
		3,
		2,
		0.25,
		200,
		190,
		sql.NullString{},
		sql.NullString{},
	)

	mustExec(insertMessageQuery,
		"message-parent-1",
		"session-parent",
		"user",
		`[{"type":"text","text":"hello"}]`,
		sql.NullString{},
		sql.NullString{},
		int64(0),
		101,
		101,
		sql.NullInt64{},
	)
	mustExec(insertMessageQuery,
		"message-parent-2",
		"session-parent",
		"assistant",
		`[{"type":"text","text":"world"},{"type":"finish","reason":"stop","time":102}]`,
		sql.NullString{String: "claude-sonnet", Valid: true},
		sql.NullString{String: "anthropic", Valid: true},
		int64(1),
		102,
		102,
		sql.NullInt64{Int64: 102, Valid: true},
	)
	mustExec(insertMessageQuery,
		"message-child-1",
		"session-child",
		"user",
		`[{"type":"text","text":"child"}]`,
		sql.NullString{},
		sql.NullString{},
		int64(0),
		201,
		201,
		sql.NullInt64{},
	)

	mustExec(insertFileQuery,
		"file-parent-1",
		"session-parent",
		"README.md",
		"# title",
		1,
		103,
		103,
	)
	mustExec(insertReadFileQuery,
		"session-parent",
		"README.md",
		104,
	)

	require.NoError(t, tx.Commit())
}

func findSessionSubcommand(t *testing.T, name string) *cobra.Command {
	t.Helper()

	for _, cmd := range sessionCmd.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	t.Fatalf("missing subcommand %q", name)
	return nil
}

func chompJSONL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	return strings.ReplaceAll(s, "\n", ",")
}
