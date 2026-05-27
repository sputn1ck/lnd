//go:build js && wasm && kvdb_sqlite

package sqlite

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"syscall/js"

	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/lightningnetwork/lnd/kvdb/sqlbase"
	_ "github.com/sputn1ck/go-wasmsqlite"
)

// defaultWasmSQLiteVFS uses go-wasmsqlite's automatic browser preference order:
// opfs-wl, opfs-sahpool, opfs, then memory if persistent storage is unavailable.
const defaultWasmSQLiteVFS = "auto"

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
	setWasmSQLiteStorage(wasmOptions, filepath.Join(dbPath, fileName))
	wasmOptions.Set("pragma", strings.Join(pragmaOptions, ";"))

	sqlCfg := &sqlbase.Config{
		DriverName:      "wasmsqlite",
		Dsn:             wasmOptions.Encode(),
		Timeout:         cfg.Timeout,
		TableNamePrefix: prefix,
	}

	return sqlbase.NewSqlBackend(ctx, sqlCfg)
}

func setWasmSQLiteStorage(values url.Values, fileName string) {
	vfs := defaultWasmSQLiteVFS
	globalVFS := js.Global().Get("lndWasmSQLiteVFS")
	if globalVFS.Type() == js.TypeString && globalVFS.String() != "" {
		vfs = globalVFS.String()
	}

	if vfs == "memory" {
		values.Set("file", ":memory:")
		values.Set("vfs", "memory")
		return
	}

	values.Set("file", fileName)
	values.Set("vfs", vfs)
	values.Set("journal_mode", "WAL")
}
