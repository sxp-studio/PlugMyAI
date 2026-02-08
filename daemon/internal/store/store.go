package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type App struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Scope     string    `json:"scope"` // "chat" or "full"
	Token     string    `json:"token,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `json:"revoked"`
	Providers []string  `json:"providers"` // allowed provider IDs; empty = unrestricted
}

type HistoryEntry struct {
	ID           string          `json:"id"`
	AppID        string          `json:"app_id"`
	AppName      string          `json:"app_name"`
	Model        string          `json:"model"`
	Provider     string          `json:"provider"`
	Messages     json.RawMessage `json:"messages"`
	Response     json.RawMessage `json:"response"`
	TokensIn     int             `json:"tokens_in"`
	TokensOut    int             `json:"tokens_out"`
	DurationMS   int64           `json:"duration_ms"`
	Status       string          `json:"status"` // "success", "error"
	ErrorMessage string          `json:"error_message,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type ConnectRequest struct {
	ID             string    `json:"id"`
	AppName        string    `json:"app_name"`
	AppURL         string    `json:"app_url"`
	AppIcon        string    `json:"app_icon,omitempty"`
	RequestedScope string    `json:"requested_scope"` // "chat" or "full"
	Status         string    `json:"status"`          // "pending", "approved", "denied"
	Token          string    `json:"token,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

func New(dataDir string) (*Store, error) {
	dbPath := filepath.Join(dataDir, "data.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrating database: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS apps (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			url TEXT NOT NULL DEFAULT '',
			scope TEXT NOT NULL DEFAULT 'chat',
			token TEXT NOT NULL UNIQUE,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			revoked INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS history (
			id TEXT PRIMARY KEY,
			app_id TEXT NOT NULL,
			app_name TEXT NOT NULL,
			model TEXT NOT NULL DEFAULT '',
			provider TEXT NOT NULL DEFAULT '',
			messages TEXT NOT NULL DEFAULT '[]',
			response TEXT NOT NULL DEFAULT '{}',
			tokens_in INTEGER NOT NULL DEFAULT 0,
			tokens_out INTEGER NOT NULL DEFAULT 0,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'success',
			error_message TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (app_id) REFERENCES apps(id)
		)`,
		`CREATE TABLE IF NOT EXISTS connect_requests (
			id TEXT PRIMARY KEY,
			app_name TEXT NOT NULL,
			app_url TEXT NOT NULL DEFAULT '',
			app_icon TEXT NOT NULL DEFAULT '',
			requested_scope TEXT NOT NULL DEFAULT 'chat',
			status TEXT NOT NULL DEFAULT 'pending',
			token TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_history_app_id ON history(app_id)`,
		`CREATE INDEX IF NOT EXISTS idx_history_created_at ON history(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_apps_token ON apps(token)`,
		`CREATE TABLE IF NOT EXISTS app_providers (
			app_id TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			PRIMARY KEY (app_id, provider_id),
			FOREIGN KEY (app_id) REFERENCES apps(id)
		)`,
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("executing migration: %w", err)
		}
	}
	return nil
}

// --- Apps ---

func (s *Store) CreateApp(id, name, url, scope, token string) error {
	if scope == "" {
		scope = "chat"
	}
	_, err := s.db.Exec(
		"INSERT INTO apps (id, name, url, scope, token) VALUES (?, ?, ?, ?, ?)",
		id, name, url, scope, token,
	)
	return err
}

func (s *Store) GetAppByToken(token string) (*App, error) {
	var a App
	err := s.db.QueryRow(
		"SELECT id, name, url, scope, token, created_at, revoked FROM apps WHERE token = ? AND revoked = 0",
		token,
	).Scan(&a.ID, &a.Name, &a.URL, &a.Scope, &a.Token, &a.CreatedAt, &a.Revoked)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.Providers, err = s.GetAppProviders(a.ID)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) ListApps() ([]App, error) {
	rows, err := s.db.Query(
		"SELECT id, name, url, scope, token, created_at, revoked FROM apps ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []App
	for rows.Next() {
		var a App
		if err := rows.Scan(&a.ID, &a.Name, &a.URL, &a.Scope, &a.Token, &a.CreatedAt, &a.Revoked); err != nil {
			return nil, err
		}
		providers, err := s.GetAppProviders(a.ID)
		if err != nil {
			return nil, err
		}
		a.Providers = providers
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

func (s *Store) RevokeApp(id string) error {
	_, err := s.db.Exec("UPDATE apps SET revoked = 1 WHERE id = ?", id)
	return err
}

// SetAppProviders replaces the allowed providers for an app.
// An empty slice means unrestricted access.
func (s *Store) SetAppProviders(appID string, providerIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM app_providers WHERE app_id = ?", appID); err != nil {
		return err
	}
	for _, pid := range providerIDs {
		if _, err := tx.Exec("INSERT INTO app_providers (app_id, provider_id) VALUES (?, ?)", appID, pid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetAppProviders returns the allowed provider IDs for an app.
// An empty slice means unrestricted.
func (s *Store) GetAppProviders(appID string) ([]string, error) {
	rows, err := s.db.Query("SELECT provider_id FROM app_providers WHERE app_id = ?", appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// --- History ---

func (s *Store) LogRequest(entry *HistoryEntry) error {
	_, err := s.db.Exec(
		`INSERT INTO history (id, app_id, app_name, model, provider, messages, response, tokens_in, tokens_out, duration_ms, status, error_message)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.AppID, entry.AppName, entry.Model, entry.Provider,
		string(entry.Messages), string(entry.Response),
		entry.TokensIn, entry.TokensOut, entry.DurationMS,
		entry.Status, entry.ErrorMessage,
	)
	return err
}

func (s *Store) ListHistory(limit, offset int) ([]HistoryEntry, error) {
	rows, err := s.db.Query(
		`SELECT id, app_id, app_name, model, provider, messages, response, tokens_in, tokens_out, duration_ms, status, error_message, created_at
		 FROM history ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		var msgs, resp string
		if err := rows.Scan(&e.ID, &e.AppID, &e.AppName, &e.Model, &e.Provider, &msgs, &resp, &e.TokensIn, &e.TokensOut, &e.DurationMS, &e.Status, &e.ErrorMessage, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Messages = json.RawMessage(msgs)
		e.Response = json.RawMessage(resp)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) GetHistoryEntry(id string) (*HistoryEntry, error) {
	var e HistoryEntry
	var msgs, resp string
	err := s.db.QueryRow(
		`SELECT id, app_id, app_name, model, provider, messages, response, tokens_in, tokens_out, duration_ms, status, error_message, created_at
		 FROM history WHERE id = ?`, id,
	).Scan(&e.ID, &e.AppID, &e.AppName, &e.Model, &e.Provider, &msgs, &resp, &e.TokensIn, &e.TokensOut, &e.DurationMS, &e.Status, &e.ErrorMessage, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	e.Messages = json.RawMessage(msgs)
	e.Response = json.RawMessage(resp)
	return &e, err
}

// HistoryFilter controls server-side filtering for history listings.
type HistoryFilter struct {
	Limit   int
	Offset  int
	AppName string // SQL LIKE %value%
	SortBy  string // "recent" (default) or "tokens"
}

// ListHistoryFiltered returns history entries matching the given filter,
// along with the total count for pagination.
func (s *Store) ListHistoryFiltered(f HistoryFilter) ([]HistoryEntry, int, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}

	where := ""
	var args []any
	if f.AppName != "" {
		where = " WHERE app_name LIKE ?"
		args = append(args, "%"+f.AppName+"%")
	}

	// Count total matching rows
	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM history"+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	orderBy := " ORDER BY created_at DESC"
	if f.SortBy == "tokens" {
		orderBy = " ORDER BY (tokens_in + tokens_out) DESC"
	}

	query := `SELECT id, app_id, app_name, model, provider, messages, response, tokens_in, tokens_out, duration_ms, status, error_message, created_at
		 FROM history` + where + orderBy + " LIMIT ? OFFSET ?"
	queryArgs := append(args, f.Limit, f.Offset)

	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		var msgs, resp string
		if err := rows.Scan(&e.ID, &e.AppID, &e.AppName, &e.Model, &e.Provider, &msgs, &resp, &e.TokensIn, &e.TokensOut, &e.DurationMS, &e.Status, &e.ErrorMessage, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		e.Messages = json.RawMessage(msgs)
		e.Response = json.RawMessage(resp)
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

// DeleteAllHistory removes all history entries.
func (s *Store) DeleteAllHistory() error {
	_, err := s.db.Exec("DELETE FROM history")
	return err
}

// --- Connect Requests ---

func (s *Store) CreateConnectRequest(id, appName, appURL, appIcon, requestedScope string, expiresAt time.Time) error {
	if requestedScope == "" {
		requestedScope = "chat"
	}
	_, err := s.db.Exec(
		"INSERT INTO connect_requests (id, app_name, app_url, app_icon, requested_scope, expires_at) VALUES (?, ?, ?, ?, ?, ?)",
		id, appName, appURL, appIcon, requestedScope, expiresAt,
	)
	return err
}

func (s *Store) GetConnectRequest(id string) (*ConnectRequest, error) {
	var cr ConnectRequest
	err := s.db.QueryRow(
		"SELECT id, app_name, app_url, app_icon, requested_scope, status, token, created_at, expires_at FROM connect_requests WHERE id = ?",
		id,
	).Scan(&cr.ID, &cr.AppName, &cr.AppURL, &cr.AppIcon, &cr.RequestedScope, &cr.Status, &cr.Token, &cr.CreatedAt, &cr.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &cr, err
}

func (s *Store) ApproveConnectRequest(id, token string) error {
	_, err := s.db.Exec(
		"UPDATE connect_requests SET status = 'approved', token = ? WHERE id = ? AND status = 'pending'",
		token, id,
	)
	return err
}

func (s *Store) DenyConnectRequest(id string) error {
	_, err := s.db.Exec(
		"UPDATE connect_requests SET status = 'denied' WHERE id = ? AND status = 'pending'",
		id,
	)
	return err
}

func (s *Store) ListPendingConnectRequests() ([]ConnectRequest, error) {
	rows, err := s.db.Query(
		"SELECT id, app_name, app_url, app_icon, requested_scope, status, token, created_at, expires_at FROM connect_requests WHERE status = 'pending' AND expires_at > CURRENT_TIMESTAMP ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []ConnectRequest
	for rows.Next() {
		var cr ConnectRequest
		if err := rows.Scan(&cr.ID, &cr.AppName, &cr.AppURL, &cr.AppIcon, &cr.RequestedScope, &cr.Status, &cr.Token, &cr.CreatedAt, &cr.ExpiresAt); err != nil {
			return nil, err
		}
		requests = append(requests, cr)
	}
	return requests, rows.Err()
}
