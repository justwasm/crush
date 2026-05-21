package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/event"
	"github.com/spf13/cobra"
)

const sessionTransferVersion = 1

const (
	listAllSessionsQuery = `
SELECT id, parent_session_id, title, message_count, prompt_tokens, completion_tokens, cost, updated_at, created_at, summary_message_id, todos
FROM sessions
ORDER BY created_at ASC, id ASC
`
	insertSessionQuery = `
INSERT INTO sessions (
	id,
	parent_session_id,
	title,
	message_count,
	prompt_tokens,
	completion_tokens,
	cost,
	updated_at,
	created_at,
	summary_message_id,
	todos
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	insertMessageQuery = `
INSERT INTO messages (
	id,
	session_id,
	role,
	parts,
	model,
	provider,
	is_summary_message,
	created_at,
	updated_at,
	finished_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	insertFileQuery = `
INSERT INTO files (
	id,
	session_id,
	path,
	content,
	version,
	created_at,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
`
	insertReadFileQuery = `
INSERT INTO read_files (
	session_id,
	path,
	read_at
) VALUES (?, ?, ?)
`
)

type sessionTransferRecord struct {
	Version   int                       `json:"version"`
	Session   sessionTransferSession    `json:"session"`
	Messages  []sessionTransferMessage  `json:"messages,omitempty"`
	Files     []sessionTransferFile     `json:"files,omitempty"`
	ReadFiles []sessionTransferReadFile `json:"read_files,omitempty"`
}

type sessionTransferSession struct {
	ID               string          `json:"id"`
	ParentSessionID  string          `json:"parent_session_id,omitempty"`
	Title            string          `json:"title"`
	MessageCount     int64           `json:"message_count"`
	PromptTokens     int64           `json:"prompt_tokens"`
	CompletionTokens int64           `json:"completion_tokens"`
	Cost             float64         `json:"cost"`
	UpdatedAt        int64           `json:"updated_at"`
	CreatedAt        int64           `json:"created_at"`
	SummaryMessageID string          `json:"summary_message_id,omitempty"`
	Todos            json.RawMessage `json:"todos,omitempty"`
}

type sessionTransferMessage struct {
	ID               string          `json:"id"`
	Role             string          `json:"role"`
	Parts            json.RawMessage `json:"parts"`
	Model            string          `json:"model,omitempty"`
	Provider         string          `json:"provider,omitempty"`
	IsSummaryMessage bool            `json:"is_summary_message,omitempty"`
	CreatedAt        int64           `json:"created_at"`
	UpdatedAt        int64           `json:"updated_at"`
	FinishedAt       *int64          `json:"finished_at,omitempty"`
}

type sessionTransferFile struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Content   string `json:"content"`
	Version   int64  `json:"version"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type sessionTransferReadFile struct {
	Path   string `json:"path"`
	ReadAt int64  `json:"read_at"`
}

func runSessionExport(cmd *cobra.Command, args []string) error {
	event.SetNonInteractive(true)

	ctx, svc, cleanup, err := sessionSetup(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	out, outCleanup, err := sessionTransferOutput(cmd.OutOrStdout(), args)
	if err != nil {
		return err
	}
	defer outCleanup()

	count, err := exportSessionTransfers(ctx, svc, out)
	if err != nil {
		return err
	}
	event.SessionExported(count)
	return nil
}

func runSessionImport(cmd *cobra.Command, args []string) error {
	event.SetNonInteractive(true)

	ctx, svc, cleanup, err := sessionSetup(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	in, inCleanup, err := sessionTransferInput(cmd.InOrStdin(), args)
	if err != nil {
		return err
	}
	defer inCleanup()

	count, err := importSessionTransfers(ctx, svc.conn, in)
	if err != nil {
		return err
	}
	event.SessionImported(count)
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Imported %d sessions\n", count)
	return err
}

func exportSessionTransfers(ctx context.Context, svc *sessionServices, w io.Writer) (int, error) {
	sessions, err := listAllDBSessions(ctx, svc.conn)
	if err != nil {
		return 0, fmt.Errorf("failed to list sessions: %w", err)
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	for _, sess := range sessions {
		record, err := buildSessionTransferRecord(ctx, svc, sess)
		if err != nil {
			return 0, err
		}
		if err := enc.Encode(record); err != nil {
			return 0, fmt.Errorf("failed to write session %q: %w", sess.ID, err)
		}
	}

	return len(sessions), nil
}

func buildSessionTransferRecord(ctx context.Context, svc *sessionServices, sess db.Session) (sessionTransferRecord, error) {
	messages, err := svc.q.ListMessagesBySession(ctx, sess.ID)
	if err != nil {
		return sessionTransferRecord{}, fmt.Errorf("failed to list messages for session %q: %w", sess.ID, err)
	}
	files, err := svc.q.ListFilesBySession(ctx, sess.ID)
	if err != nil {
		return sessionTransferRecord{}, fmt.Errorf("failed to list files for session %q: %w", sess.ID, err)
	}
	readFiles, err := svc.q.ListSessionReadFiles(ctx, sess.ID)
	if err != nil {
		return sessionTransferRecord{}, fmt.Errorf("failed to list read files for session %q: %w", sess.ID, err)
	}

	record := sessionTransferRecord{
		Version: sessionTransferVersion,
		Session: sessionTransferSession{
			ID:               sess.ID,
			ParentSessionID:  sess.ParentSessionID.String,
			Title:            sess.Title,
			MessageCount:     sess.MessageCount,
			PromptTokens:     sess.PromptTokens,
			CompletionTokens: sess.CompletionTokens,
			Cost:             sess.Cost,
			UpdatedAt:        sess.UpdatedAt,
			CreatedAt:        sess.CreatedAt,
			SummaryMessageID: sess.SummaryMessageID.String,
			Todos:            sessionTransferJSON(sess.Todos),
		},
		Messages:  make([]sessionTransferMessage, len(messages)),
		Files:     make([]sessionTransferFile, len(files)),
		ReadFiles: make([]sessionTransferReadFile, len(readFiles)),
	}

	for i, msg := range messages {
		record.Messages[i] = sessionTransferMessage{
			ID:               msg.ID,
			Role:             msg.Role,
			Parts:            sessionTransferRawBytes(msg.Parts),
			Model:            msg.Model.String,
			Provider:         msg.Provider.String,
			IsSummaryMessage: msg.IsSummaryMessage != 0,
			CreatedAt:        msg.CreatedAt,
			UpdatedAt:        msg.UpdatedAt,
			FinishedAt:       sessionTransferInt64Ptr(msg.FinishedAt),
		}
	}

	for i, file := range files {
		record.Files[i] = sessionTransferFile{
			ID:        file.ID,
			Path:      file.Path,
			Content:   file.Content,
			Version:   file.Version,
			CreatedAt: file.CreatedAt,
			UpdatedAt: file.UpdatedAt,
		}
	}

	for i, readFile := range readFiles {
		record.ReadFiles[i] = sessionTransferReadFile{
			Path:   readFile.Path,
			ReadAt: readFile.ReadAt,
		}
	}

	return record, nil
}

func importSessionTransfers(ctx context.Context, conn *sql.DB, r io.Reader) (int, error) {
	sqlConn, err := conn.Conn(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to reserve database connection: %w", err)
	}
	defer sqlConn.Close()

	if _, err := sqlConn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return 0, fmt.Errorf("failed to disable foreign keys: %w", err)
	}
	defer func() {
		_, _ = sqlConn.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")
	}()

	tx, err := sqlConn.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	dec := json.NewDecoder(r)
	count := 0
	for {
		var record sessionTransferRecord
		if err := dec.Decode(&record); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return 0, fmt.Errorf("failed to decode session record %d: %w", count+1, err)
		}
		if err := validateSessionTransferRecord(record); err != nil {
			return 0, fmt.Errorf("invalid session record %d: %w", count+1, err)
		}
		if err := importSessionTransferRecord(ctx, tx, record); err != nil {
			return 0, fmt.Errorf("failed to import session %q: %w", record.Session.ID, err)
		}
		count++
	}

	if err := checkSessionTransferForeignKeys(ctx, tx); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit import: %w", err)
	}

	return count, nil
}

func importSessionTransferRecord(ctx context.Context, tx *sql.Tx, record sessionTransferRecord) error {
	for _, msg := range record.Messages {
		if _, err := tx.ExecContext(ctx, insertMessageQuery,
			msg.ID,
			record.Session.ID,
			msg.Role,
			string(msg.Parts),
			nullString(msg.Model),
			nullString(msg.Provider),
			boolToInt64(msg.IsSummaryMessage),
			msg.CreatedAt,
			msg.UpdatedAt,
			nullInt64(msg.FinishedAt),
		); err != nil {
			return err
		}
	}

	for _, file := range record.Files {
		if _, err := tx.ExecContext(ctx, insertFileQuery,
			file.ID,
			record.Session.ID,
			file.Path,
			file.Content,
			file.Version,
			file.CreatedAt,
			file.UpdatedAt,
		); err != nil {
			return err
		}
	}

	for _, readFile := range record.ReadFiles {
		if _, err := tx.ExecContext(ctx, insertReadFileQuery,
			record.Session.ID,
			readFile.Path,
			readFile.ReadAt,
		); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, insertSessionQuery,
		record.Session.ID,
		nullString(record.Session.ParentSessionID),
		record.Session.Title,
		record.Session.MessageCount,
		record.Session.PromptTokens,
		record.Session.CompletionTokens,
		record.Session.Cost,
		record.Session.UpdatedAt,
		record.Session.CreatedAt,
		nullString(record.Session.SummaryMessageID),
		nullString(string(record.Session.Todos)),
	); err != nil {
		return err
	}

	return nil
}

func listAllDBSessions(ctx context.Context, conn *sql.DB) ([]db.Session, error) {
	rows, err := conn.QueryContext(ctx, listAllSessionsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []db.Session
	for rows.Next() {
		var sess db.Session
		if err := rows.Scan(
			&sess.ID,
			&sess.ParentSessionID,
			&sess.Title,
			&sess.MessageCount,
			&sess.PromptTokens,
			&sess.CompletionTokens,
			&sess.Cost,
			&sess.UpdatedAt,
			&sess.CreatedAt,
			&sess.SummaryMessageID,
			&sess.Todos,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

func validateSessionTransferRecord(record sessionTransferRecord) error {
	if record.Version != sessionTransferVersion {
		return fmt.Errorf("unsupported version %d", record.Version)
	}
	if strings.TrimSpace(record.Session.ID) == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(record.Session.Title) == "" {
		return fmt.Errorf("session title is required")
	}
	if record.Session.MessageCount < 0 {
		return fmt.Errorf("message_count must be non-negative")
	}
	if len(record.Session.Todos) > 0 && !json.Valid(record.Session.Todos) {
		return fmt.Errorf("session todos must be valid JSON")
	}
	for i, msg := range record.Messages {
		if strings.TrimSpace(msg.ID) == "" {
			return fmt.Errorf("message %d is missing an id", i)
		}
		if strings.TrimSpace(msg.Role) == "" {
			return fmt.Errorf("message %d is missing a role", i)
		}
		if len(msg.Parts) == 0 || !json.Valid(msg.Parts) {
			return fmt.Errorf("message %d has invalid parts JSON", i)
		}
	}
	for i, file := range record.Files {
		if strings.TrimSpace(file.ID) == "" {
			return fmt.Errorf("file %d is missing an id", i)
		}
		if strings.TrimSpace(file.Path) == "" {
			return fmt.Errorf("file %d is missing a path", i)
		}
	}
	for i, readFile := range record.ReadFiles {
		if strings.TrimSpace(readFile.Path) == "" {
			return fmt.Errorf("read_files entry %d is missing a path", i)
		}
	}
	return nil
}

func checkSessionTransferForeignKeys(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return fmt.Errorf("failed to validate imported data: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		var table string
		var rowID int64
		var parent string
		var fkIndex int64
		if err := rows.Scan(&table, &rowID, &parent, &fkIndex); err != nil {
			return fmt.Errorf("failed to read foreign key error: %w", err)
		}
		return fmt.Errorf("foreign key check failed for table %q row %d referencing %q", table, rowID, parent)
	}

	return rows.Err()
}

func sessionTransferInput(stdin io.Reader, args []string) (io.ReadCloser, func(), error) {
	if len(args) == 0 || args[0] == "-" {
		return io.NopCloser(stdin), func() {}, nil
	}

	f, err := os.Open(args[0])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open %q: %w", args[0], err)
	}
	return f, func() { _ = f.Close() }, nil
}

func sessionTransferOutput(stdout io.Writer, args []string) (io.WriteCloser, func(), error) {
	if len(args) == 0 || args[0] == "-" {
		return nopWriteCloser{Writer: stdout}, func() {}, nil
	}

	f, err := os.Create(args[0])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create %q: %w", args[0], err)
	}
	return f, func() { _ = f.Close() }, nil
}

func sessionTransferJSON(v sql.NullString) json.RawMessage {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return nil
	}
	return json.RawMessage(v.String)
}

func sessionTransferRawBytes(v string) json.RawMessage {
	if strings.TrimSpace(v) == "" {
		return json.RawMessage("null")
	}
	return json.RawMessage(v)
}

func sessionTransferInt64Ptr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	value := v.Int64
	return &value
}

func nullString(v string) sql.NullString {
	return sql.NullString{
		String: v,
		Valid:  strings.TrimSpace(v) != "",
	}
}

func nullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func boolToInt64(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}
