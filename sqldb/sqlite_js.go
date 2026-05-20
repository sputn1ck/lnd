//go:build js && wasm

package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/lightningnetwork/lnd/sqldb/sqlc"
	"github.com/sputn1ck/go-wasmsqlite"
	_ "github.com/sputn1ck/go-wasmsqlite"
	"github.com/stretchr/testify/require"
)

var (
	// sqliteSchemaReplacements maps schema strings to their SQLite
	// compatible replacements. Currently, no replacements are needed as our
	// SQL schema definition files are designed for SQLite compatibility.
	sqliteSchemaReplacements = map[string]string{}

	// Make sure SqliteStore implements the MigrationExecutor interface.
	_ MigrationExecutor = (*SqliteStore)(nil)

	// Make sure SqliteStore implements the DB interface.
	_ DB = (*SqliteStore)(nil)
)

// SqliteStore is a database store implementation that uses a sqlite backend.
type SqliteStore struct {
	cfg *SqliteConfig

	*BaseDB
}

// NewSqliteStore attempts to open a new sqlite database based on the passed
// config.
func NewSqliteStore(cfg *SqliteConfig, dbPath string) (*SqliteStore, error) {
	pragmaOptions := []string{
		"foreign_keys=on",
		"synchronous=full",
		"auto_vacuum=incremental",
	}
	pragmaOptions = append(pragmaOptions, cfg.PragmaOptions...)

	wasmOptions := make(url.Values)
	wasmOptions.Set("file", dbPath)
	wasmOptions.Set("vfs", "opfs")
	wasmOptions.Set("journal_mode", "WAL")
	wasmOptions.Set("busy_timeout", fmt.Sprintf("%d", cfg.busyTimeoutMs()))
	wasmOptions.Set("require_persistent", "true")
	wasmOptions.Set("pragma", strings.Join(pragmaOptions, ";"))

	db, err := sql.Open("wasmsqlite", wasmOptions.Encode())
	if err != nil {
		return nil, err
	}

	migrationTrackerSQL := `
	CREATE TABLE IF NOT EXISTS migration_tracker (
		version INTEGER UNIQUE NOT NULL,
		migration_time TIMESTAMP NOT NULL
	);`

	_, err = db.Exec(migrationTrackerSQL)
	if err != nil {
		return nil, fmt.Errorf("error creating migration tracker: %w",
			err)
	}

	maxConns := cfg.MaxConns()
	if maxConns > 1 {
		maxConns = 1
	}
	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(maxConns)
	db.SetConnMaxLifetime(connIdleLifetime)

	s := &SqliteStore{
		cfg: cfg,
		BaseDB: &BaseDB{
			DB:      db,
			Queries: sqlc.New(db),
		},
	}

	return s, nil
}

// GetBaseDB returns the underlying BaseDB instance for the SQLite store.
func (s *SqliteStore) GetBaseDB() *BaseDB {
	return s.BaseDB
}

// ApplyAllMigrations applies both the SQLC and custom in-code migrations to the
// SQLite database.
func (s *SqliteStore) ApplyAllMigrations(ctx context.Context,
	migrations []MigrationConfig) error {

	if s.cfg.SkipMigrations {
		return nil
	}

	return ApplyMigrations(ctx, s.BaseDB, s, migrations)
}

func errSqliteMigration(err error) error {
	return fmt.Errorf("error creating sqlite migration: %w", err)
}

// ExecuteMigrations runs migrations for the sqlite database.
func (s *SqliteStore) ExecuteMigrations(target MigrationTarget) error {
	driver, err := wasmsqlite.NewMigrateDriver(s.DB)
	if err != nil {
		return errSqliteMigration(err)
	}

	sqliteFS := newReplacerFS(sqlSchemas, sqliteSchemaReplacements)
	return applyMigrations(
		sqliteFS, driver, "sqlc/migrations", "sqlite", target,
	)
}

// GetSchemaVersion returns the current schema version of the SQLite database.
func (s *SqliteStore) GetSchemaVersion() (int, bool, error) {
	driver, err := wasmsqlite.NewMigrateDriver(s.DB)
	if err != nil {
		return 0, false, errSqliteMigration(err)
	}

	version, dirty, err := driver.Version()
	if err != nil {
		return 0, dirty, err
	}

	return version, dirty, nil
}

// SetSchemaVersion sets the schema version of the SQLite database.
//
// NOTE: This alters the internal database schema tracker. USE WITH CAUTION!!!
func (s *SqliteStore) SetSchemaVersion(version int, dirty bool) error {
	driver, err := wasmsqlite.NewMigrateDriver(s.DB)
	if err != nil {
		return errSqliteMigration(err)
	}

	return driver.SetVersion(version, dirty)
}

// NewTestSqliteDB is a helper function that creates an SQLite database for
// testing.
func NewTestSqliteDB(t testing.TB) *SqliteStore {
	t.Helper()

	dbFileName := fmt.Sprintf("/%s.db", t.Name())
	sqlDB, err := NewSqliteStore(&SqliteConfig{
		SkipMigrations: false,
	}, dbFileName)
	require.NoError(t, err)

	require.NoError(t, sqlDB.ApplyAllMigrations(
		context.Background(), GetMigrations()),
	)

	t.Cleanup(func() {
		require.NoError(t, sqlDB.DB.Close())
	})

	return sqlDB
}

// NewTestSqliteDBWithVersion is a helper function that creates an SQLite
// database for testing and migrates it to the given version.
func NewTestSqliteDBWithVersion(t *testing.T, version uint) *SqliteStore {
	t.Helper()

	dbFileName := fmt.Sprintf("/%s-%d.db", t.Name(), version)
	sqlDB, err := NewSqliteStore(&SqliteConfig{
		SkipMigrations: false,
	}, dbFileName)
	require.NoError(t, err)

	require.NoError(t, sqlDB.ExecuteMigrations(TargetVersion(version)))

	t.Cleanup(func() {
		require.NoError(t, sqlDB.DB.Close())
	})

	return sqlDB
}
