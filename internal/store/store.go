package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Store represents the SQLite-backed data store
type Store struct {
	db *sql.DB
}

// New creates a new Store instance
func New(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// Migrate creates or updates the database schema
func (s *Store) Migrate(ctx context.Context) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS provider_cursors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			scope_key TEXT NOT NULL UNIQUE,
			cursor_value TEXT NOT NULL,
			cursor_kind TEXT NOT NULL,
			last_polled_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS repositories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			repo_ref TEXT NOT NULL UNIQUE,
			owner TEXT NOT NULL,
			name TEXT NOT NULL,
			default_branch TEXT NOT NULL,
			branch_prefix TEXT NOT NULL DEFAULT 'heimdall',
			pr_monitor_label TEXT NOT NULL DEFAULT '',
			local_mirror_path TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT 1
		);`,
		`CREATE TABLE IF NOT EXISTS work_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			provider_work_item_id TEXT NOT NULL,
			work_item_key TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			state_name TEXT NOT NULL,
			lifecycle_bucket TEXT NOT NULL,
			project_name TEXT,
			team_key TEXT,
			labels_json TEXT NOT NULL DEFAULT '[]',
			last_seen_updated_at DATETIME,
			UNIQUE(provider, provider_work_item_id),
			UNIQUE(provider, work_item_key)
		);`,
		`CREATE TABLE IF NOT EXISTS work_item_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			work_item_id INTEGER NOT NULL,
			provider TEXT NOT NULL,
			provider_event_id TEXT,
			event_type TEXT NOT NULL,
			event_version TEXT,
			idempotency_key TEXT NOT NULL UNIQUE,
			occurred_at DATETIME NOT NULL,
			detected_at DATETIME NOT NULL,
			FOREIGN KEY (work_item_id) REFERENCES work_items(id)
		);`,
		`CREATE TABLE IF NOT EXISTS repo_bindings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			work_item_id INTEGER NOT NULL,
			repository_id INTEGER NOT NULL,
			branch_name TEXT NOT NULL UNIQUE,
			change_name TEXT NOT NULL UNIQUE,
			binding_status TEXT NOT NULL,
			last_head_sha TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE(work_item_id, repository_id),
			FOREIGN KEY (work_item_id) REFERENCES work_items(id),
			FOREIGN KEY (repository_id) REFERENCES repositories(id)
		);`,
		`CREATE TABLE IF NOT EXISTS pull_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER NOT NULL,
			repo_binding_id INTEGER,
			provider TEXT NOT NULL DEFAULT 'github',
			provider_pr_node_id TEXT UNIQUE,
			number INTEGER NOT NULL,
			title TEXT NOT NULL,
			base_branch TEXT NOT NULL,
			head_branch TEXT NOT NULL,
			state TEXT NOT NULL,
			url TEXT NOT NULL,
			UNIQUE(repository_id, number),
			FOREIGN KEY (repository_id) REFERENCES repositories(id),
			FOREIGN KEY (repo_binding_id) REFERENCES repo_bindings(id)
		);`,
		`CREATE TABLE IF NOT EXISTS command_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pull_request_id INTEGER NOT NULL,
			comment_node_id TEXT NOT NULL UNIQUE,
			command_name TEXT NOT NULL,
			command_args TEXT,
			requested_agent TEXT,
			actor_login TEXT NOT NULL,
			authorization_status TEXT NOT NULL,
			dedupe_key TEXT NOT NULL UNIQUE,
			workflow_run_id INTEGER,
			status TEXT NOT NULL,
			FOREIGN KEY (pull_request_id) REFERENCES pull_requests(id)
		);`,
		`CREATE TABLE IF NOT EXISTS workflow_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			work_item_id INTEGER NOT NULL,
			repository_id INTEGER NOT NULL,
			trigger_event_id INTEGER,
			run_type TEXT NOT NULL,
			status TEXT NOT NULL,
			status_reason TEXT,
			change_name TEXT NOT NULL,
			branch_name TEXT NOT NULL,
			worktree_path TEXT NOT NULL,
			requested_by_type TEXT NOT NULL,
			requested_by_login TEXT,
			attempt_count INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (work_item_id) REFERENCES work_items(id),
			FOREIGN KEY (repository_id) REFERENCES repositories(id)
		);`,
		`CREATE TABLE IF NOT EXISTS workflow_steps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workflow_run_id INTEGER NOT NULL,
			step_name TEXT NOT NULL,
			step_order INTEGER NOT NULL,
			status TEXT NOT NULL,
			executor TEXT,
			command_line TEXT,
			tool_version TEXT,
			attempt_count INTEGER NOT NULL DEFAULT 0,
			UNIQUE(workflow_run_id, step_order),
			FOREIGN KEY (workflow_run_id) REFERENCES workflow_runs(id)
		);`,
		`CREATE TABLE IF NOT EXISTS jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workflow_run_id INTEGER,
			command_request_id INTEGER,
			job_type TEXT NOT NULL,
			lock_key TEXT NOT NULL,
			status TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 100,
			run_after DATETIME NOT NULL,
			attempt_count INTEGER NOT NULL DEFAULT 0,
			max_attempts INTEGER NOT NULL DEFAULT 10,
			FOREIGN KEY (workflow_run_id) REFERENCES workflow_runs(id),
			FOREIGN KEY (command_request_id) REFERENCES command_requests(id)
		);`,
		`CREATE TABLE IF NOT EXISTS audit_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workflow_run_id INTEGER,
			command_request_id INTEGER,
			event_type TEXT NOT NULL,
			severity TEXT NOT NULL,
			actor_type TEXT NOT NULL,
			actor_login TEXT,
			agent_name TEXT,
			commit_sha TEXT,
			summary TEXT NOT NULL,
			occurred_at DATETIME NOT NULL,
			FOREIGN KEY (workflow_run_id) REFERENCES workflow_runs(id),
			FOREIGN KEY (command_request_id) REFERENCES command_requests(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_work_items_key ON work_items(provider, work_item_key);`,
		`CREATE INDEX IF NOT EXISTS idx_workflow_runs_status ON workflow_runs(work_item_id, status);`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status, run_after);`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_lock ON jobs(lock_key);`,
	}

	for _, migration := range migrations {
		if _, err := s.db.ExecContext(ctx, migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	columnMigrations := []string{
		`ALTER TABLE work_items ADD COLUMN description TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE work_items ADD COLUMN project_name TEXT`,
		`ALTER TABLE work_items ADD COLUMN labels_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE workflow_runs ADD COLUMN status_reason TEXT`,
		`ALTER TABLE repositories ADD COLUMN pr_monitor_label TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE repositories ADD COLUMN default_spec_writing_agent TEXT NOT NULL DEFAULT ''`,
	}
	for _, migration := range columnMigrations {
		if err := s.execOptionalMigration(ctx, migration); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) execOptionalMigration(ctx context.Context, query string) error {
	if _, err := s.db.ExecContext(ctx, query); err != nil {
		if isDuplicateColumnError(err) {
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}
	return nil
}

func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate column name")
}

// ProviderCursor operations
func (s *Store) GetProviderCursor(ctx context.Context, provider, scopeKey string) (*ProviderCursor, error) {
	var cursor ProviderCursor
	err := s.db.QueryRowContext(ctx,
		`SELECT id, provider, scope_key, cursor_value, cursor_kind, last_polled_at 
		 FROM provider_cursors WHERE provider = ? AND scope_key = ?`,
		provider, scopeKey,
	).Scan(&cursor.ID, &cursor.Provider, &cursor.ScopeKey, &cursor.CursorValue, &cursor.CursorKind, &cursor.LastPolledAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cursor, nil
}

func (s *Store) SetProviderCursor(ctx context.Context, cursor *ProviderCursor) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO provider_cursors (provider, scope_key, cursor_value, cursor_kind, last_polled_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(scope_key) DO UPDATE SET
		 cursor_value = excluded.cursor_value,
		 last_polled_at = excluded.last_polled_at`,
		cursor.Provider, cursor.ScopeKey, cursor.CursorValue, cursor.CursorKind, time.Now(),
	)
	return err
}

// ProviderCursor represents a polling cursor
type ProviderCursor struct {
	ID           int64
	Provider     string
	ScopeKey     string
	CursorValue  string
	CursorKind   string
	LastPolledAt *time.Time
}
