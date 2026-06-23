package db

import (
	"cmp"
	"database/sql"
	"fmt"
	"os"

	"github.com/charmbracelet/crush/internal/event"
	"github.com/tursodatabase/libsql-client-go/libsql"
)

var (
	sqld     = os.Getenv("TURSO_DATABASE_URL")
	sqldAuth = os.Getenv("TURSO_AUTH_TOKEN")
	sqldNS   = cmp.Or(os.Getenv("TURSO_NS"), event.ID())
)

func OpenDB(dbPath string) (*sql.DB, error) {
	if sqld != "" {
		if sqldAuth != "" {
			return openDBRemote(sqld, sqldAuth)
		}
		return openDBRemote(sqld+"/"+sqldNS, "")
	}
	return openDB(dbPath)
}

func openDBRemote(url string, authToken string) (*sql.DB, error) {
	opts := []libsql.Option{}
	if authToken != "" {
		opts = append(opts, libsql.WithAuthToken(authToken))
	}
	connector, err := libsql.NewConnector(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("connector: %w", err)
	}
	db := sql.OpenDB(connector)
	return db, nil
}
