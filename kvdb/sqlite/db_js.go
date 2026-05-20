//go:build js && wasm && kvdb_sqlite

package sqlite

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/lightningnetwork/lnd/kvdb/sqlbase"
	_ "github.com/sputn1ck/go-wasmsqlite"
)

// NewSqliteBackend returns a db object initialized with the passed backend
// config. The browser build uses go-wasmsqlite, which provides OPFS-backed
// database/sql access through the official SQLite WASM runtime.
func NewSqliteBackend(ctx context.Context, cfg *Config, dbPath, fileName,
	prefix string) (walletdb.DB, error) {

	pragmaOptions := []string{
		fmt.Sprintf("busy_timeout=%d", cfg.BusyTimeout.Milliseconds()),
		"foreign_keys=on",
		"auto_vacuum=incremental",
	}
	pragmaOptions = append(pragmaOptions, cfg.PragmaOptions...)

	wasmOptions := make(url.Values)
	wasmOptions.Set("file", filepath.Join(dbPath, fileName))
	wasmOptions.Set("vfs", "opfs")
	wasmOptions.Set("journal_mode", "WAL")
	wasmOptions.Set("require_persistent", "true")
	wasmOptions.Set("pragma", strings.Join(pragmaOptions, ";"))

	sqlCfg := &sqlbase.Config{
		DriverName:      "wasmsqlite",
		Dsn:             wasmOptions.Encode(),
		Timeout:         cfg.Timeout,
		TableNamePrefix: prefix,
	}

	return sqlbase.NewSqlBackend(ctx, sqlCfg)
}
