package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// CommandRun represents a single execution of an opencode-backed PR command.
type CommandRun struct {
	ID               int64
	CommandRequestID int64
	SessionID        string
	Status           string // queued, starting, running, blocked, completed, failed
	StatusReason     string
	TerminalSummary  string
	StartedAt        *time.Time
	CompletedAt      *time.Time
}

// CommandTimelineEntry represents a normalized human-readable output line for a command run.
type CommandTimelineEntry struct {
	ID           int64
	CommandRunID int64
	Sequence     int
	EntryType    string // text, tool_status, blocker, terminal, generic
	DisplayText  string
	Metadata     map[string]any
	CreatedAt    time.Time
}

// CreateCommandRun initializes a new command run record.
func (s *Store) CreateCommandRun(ctx context.Context, run *CommandRun) error {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO command_runs (command_request_id, session_id, status, status_reason, terminal_summary, started_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		run.CommandRequestID, run.SessionID, run.Status, run.StatusReason, run.TerminalSummary, run.StartedAt, run.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create command run: %w", err)
	}
	id, _ := result.LastInsertId()
	run.ID = id
	return nil
}

// GetCommandRunByCommandRequestID returns the command run for a given command request.
func (s *Store) GetCommandRunByCommandRequestID(ctx context.Context, commandRequestID int64) (*CommandRun, error) {
	var run CommandRun
	var sessionID, statusReason, terminalSummary sql.NullString
	var startedAt, completedAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, command_request_id, session_id, status, status_reason, terminal_summary, started_at, completed_at
		 FROM command_runs WHERE command_request_id = ?`,
		commandRequestID,
	).Scan(&run.ID, &run.CommandRequestID, &sessionID, &run.Status, &statusReason, &terminalSummary, &startedAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get command run: %w", err)
	}
	if sessionID.Valid {
		run.SessionID = sessionID.String
	}
	if statusReason.Valid {
		run.StatusReason = statusReason.String
	}
	if terminalSummary.Valid {
		run.TerminalSummary = terminalSummary.String
	}
	if startedAt.Valid {
		run.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		run.CompletedAt = &completedAt.Time
	}
	return &run, nil
}

// UpdateCommandRunStatus updates the status and optional reason of a command run.
func (s *Store) UpdateCommandRunStatus(ctx context.Context, runID int64, status, reason string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE command_runs SET status = ?, status_reason = ? WHERE id = ?`,
		status, reason, runID,
	)
	if err != nil {
		return fmt.Errorf("failed to update command run status: %w", err)
	}
	return nil
}

// UpdateCommandRunSessionID sets the canonical session ID for a command run.
func (s *Store) UpdateCommandRunSessionID(ctx context.Context, runID int64, sessionID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE command_runs SET session_id = ? WHERE id = ?`,
		sessionID, runID,
	)
	if err != nil {
		return fmt.Errorf("failed to update command run session id: %w", err)
	}
	return nil
}

// CompleteCommandRun marks a command run as terminal with a summary.
func (s *Store) CompleteCommandRun(ctx context.Context, runID int64, status, summary string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE command_runs SET status = ?, terminal_summary = ?, completed_at = ? WHERE id = ?`,
		status, summary, now, runID,
	)
	if err != nil {
		return fmt.Errorf("failed to complete command run: %w", err)
	}
	return nil
}

// StartCommandRun marks a command run as starting and sets the started time.
func (s *Store) StartCommandRun(ctx context.Context, runID int64) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE command_runs SET status = ?, started_at = ? WHERE id = ?`,
		"starting", now, runID,
	)
	if err != nil {
		return fmt.Errorf("failed to start command run: %w", err)
	}
	return nil
}

// AppendTimelineEntry adds a new display entry to a command run timeline.
func (s *Store) AppendTimelineEntry(ctx context.Context, entry *CommandTimelineEntry) error {
	metaJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO command_timeline_entries (command_run_id, sequence, entry_type, display_text, metadata_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		entry.CommandRunID, entry.Sequence, entry.EntryType, entry.DisplayText, string(metaJSON), entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to append timeline entry: %w", err)
	}
	id, _ := result.LastInsertId()
	entry.ID = id
	return nil
}

// GetTimelineEntries returns ordered timeline entries for a command run, bounded by limit and offset.
func (s *Store) GetTimelineEntries(ctx context.Context, commandRunID int64, limit, offset int) ([]CommandTimelineEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, command_run_id, sequence, entry_type, display_text, metadata_json, created_at
		 FROM command_timeline_entries
		 WHERE command_run_id = ?
		 ORDER BY sequence DESC
		 LIMIT ? OFFSET ?`,
		commandRunID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query timeline entries: %w", err)
	}
	defer rows.Close()

	var entries []CommandTimelineEntry
	for rows.Next() {
		var e CommandTimelineEntry
		var metaJSON string
		if err := rows.Scan(&e.ID, &e.CommandRunID, &e.Sequence, &e.EntryType, &e.DisplayText, &metaJSON, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan timeline entry: %w", err)
		}
		if metaJSON != "" {
			_ = json.Unmarshal([]byte(metaJSON), &e.Metadata)
		}
		if e.Metadata == nil {
			e.Metadata = make(map[string]any)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetNextTimelineSequence returns the next available sequence number for a command run.
func (s *Store) GetNextTimelineSequence(ctx context.Context, commandRunID int64) (int, error) {
	var seq sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT MAX(sequence) FROM command_timeline_entries WHERE command_run_id = ?`,
		commandRunID,
	).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("failed to get next timeline sequence: %w", err)
	}
	if !seq.Valid {
		return 1, nil
	}
	return int(seq.Int64) + 1, nil
}

// ListActiveCommandRuns returns command runs that are not in a terminal state.
func (s *Store) ListActiveCommandRuns(ctx context.Context) ([]CommandRun, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, command_request_id, session_id, status, status_reason, terminal_summary, started_at, completed_at
		 FROM command_runs
		 WHERE status IN ('queued', 'starting', 'running', 'blocked')
		 ORDER BY started_at DESC, id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list active command runs: %w", err)
	}
	defer rows.Close()

	var runs []CommandRun
	for rows.Next() {
		var run CommandRun
		var sessionID, statusReason, terminalSummary sql.NullString
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(&run.ID, &run.CommandRequestID, &sessionID, &run.Status, &statusReason, &terminalSummary, &startedAt, &completedAt); err != nil {
			return nil, fmt.Errorf("failed to scan command run: %w", err)
		}
		if sessionID.Valid {
			run.SessionID = sessionID.String
		}
		if statusReason.Valid {
			run.StatusReason = statusReason.String
		}
		if terminalSummary.Valid {
			run.TerminalSummary = terminalSummary.String
		}
		if startedAt.Valid {
			run.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			run.CompletedAt = &completedAt.Time
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}
