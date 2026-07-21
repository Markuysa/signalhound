package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Markuysa/signalhound/internal/core"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // pure-Go SQLite driver, no cgo (ADR-002)
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// ErrNotFound is returned by the Get* methods when a row does not exist.
var ErrNotFound = errors.New("not found")

// SQLite is the Store implementation backing the MVP.
type SQLite struct {
	db *sql.DB
}

// OpenSQLite opens (creating if absent) the database at dsn and runs migrations to the
// latest version. Opening a database that is already current is a no-op, so calling it
// twice is safe.
func OpenSQLite(dsn string) (*SQLite, error) {
	// Foreign keys are off by default in SQLite; the ON DELETE CASCADEs depend on them.
	db, err := sql.Open("sqlite", dsn+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // one writer; modernc/sqlite serializes anyway and this avoids "database is locked"
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLite{db: db}, nil
}

func migrate(db *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

func (s *SQLite) Close() error { return s.db.Close() }

func (s *SQLite) SaveSignal(ctx context.Context, sig core.Signal) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO signals (id, source, url, author, title, content, lang, published_at, collected_at, content_hash)
		VALUES (?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(content_hash) DO NOTHING`,
		sig.ID, sig.Source, sig.URL, sig.Author, sig.Title, sig.Content, sig.Lang,
		sig.PublishedAt, sig.CollectedAt, sig.ContentHash)
	if err != nil {
		return false, fmt.Errorf("save signal: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SQLite) GetSignal(ctx context.Context, id string) (core.Signal, error) {
	var sig core.Signal
	err := s.db.QueryRowContext(ctx, `
		SELECT id, source, url, author, title, content, lang, published_at, collected_at, content_hash
		FROM signals WHERE id = ?`, id).Scan(
		&sig.ID, &sig.Source, &sig.URL, &sig.Author, &sig.Title, &sig.Content, &sig.Lang,
		&sig.PublishedAt, &sig.CollectedAt, &sig.ContentHash)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Signal{}, ErrNotFound
	}
	if err != nil {
		return core.Signal{}, fmt.Errorf("get signal: %w", err)
	}
	return sig, nil
}

func (s *SQLite) ListSignals(ctx context.Context, f SignalFilter) ([]core.Signal, string, error) {
	q := `SELECT s.id, s.source, s.url, s.author, s.title, s.content, s.lang, s.published_at, s.collected_at, s.content_hash
	      FROM signals s`
	var joins, where string
	var args []any
	if f.ScoreGTE > 0 || f.Intent != "" {
		joins = " JOIN scores sc ON sc.signal_id = s.id"
	}
	add := func(cond string, arg any) {
		if where == "" {
			where = " WHERE "
		} else {
			where += " AND "
		}
		where += cond
		args = append(args, arg)
	}
	if f.ScoreGTE > 0 {
		add("sc.value >= ?", f.ScoreGTE)
	}
	if f.Intent != "" {
		add("sc.intent_type = ?", f.Intent)
	}
	if f.Source != "" {
		add("s.source = ?", f.Source)
	}
	if !f.Since.IsZero() {
		add("s.published_at >= ?", f.Since)
	}
	if f.Cursor != "" {
		add("s.id < ?", f.Cursor) // ULIDs sort by time, so id < cursor is "older than"
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q += joins + where + " ORDER BY s.id DESC LIMIT ?"
	args = append(args, limit+1) // fetch one extra to know whether a next page exists

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list signals: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []core.Signal
	for rows.Next() {
		var sig core.Signal
		if err := rows.Scan(&sig.ID, &sig.Source, &sig.URL, &sig.Author, &sig.Title, &sig.Content,
			&sig.Lang, &sig.PublishedAt, &sig.CollectedAt, &sig.ContentHash); err != nil {
			return nil, "", fmt.Errorf("scan signal: %w", err)
		}
		out = append(out, sig)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	var next string
	if len(out) > limit {
		next = out[limit].ID
		out = out[:limit]
	}
	return out, next, nil
}

func (s *SQLite) SaveScore(ctx context.Context, sc core.Score) error {
	reasons, _ := json.Marshal(sc.Reasons)
	icp, _ := json.Marshal(sc.ICPMatch)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO scores (signal_id, value, intent_type, reasons, icp_match, draft_reply, model, cost_usd, latency_ms, scored_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(signal_id) DO UPDATE SET
			value=excluded.value, intent_type=excluded.intent_type, reasons=excluded.reasons,
			icp_match=excluded.icp_match, draft_reply=excluded.draft_reply, model=excluded.model,
			cost_usd=excluded.cost_usd, latency_ms=excluded.latency_ms, scored_at=excluded.scored_at`,
		sc.SignalID, sc.Value, sc.IntentType, string(reasons), string(icp), sc.DraftReply,
		sc.Model, sc.CostUSD, sc.LatencyMS, sc.ScoredAt)
	if err != nil {
		return fmt.Errorf("save score: %w", err)
	}
	return nil
}

func (s *SQLite) GetScore(ctx context.Context, signalID string) (core.Score, error) {
	return scanScore(s.db.QueryRowContext(ctx, `
		SELECT signal_id, value, intent_type, reasons, icp_match, draft_reply, model, cost_usd, latency_ms, scored_at
		FROM scores WHERE signal_id = ?`, signalID))
}

func scanScore(row *sql.Row) (core.Score, error) {
	var sc core.Score
	var reasons, icp string
	err := row.Scan(&sc.SignalID, &sc.Value, &sc.IntentType, &reasons, &icp, &sc.DraftReply,
		&sc.Model, &sc.CostUSD, &sc.LatencyMS, &sc.ScoredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Score{}, ErrNotFound
	}
	if err != nil {
		return core.Score{}, fmt.Errorf("scan score: %w", err)
	}
	_ = json.Unmarshal([]byte(reasons), &sc.Reasons)
	_ = json.Unmarshal([]byte(icp), &sc.ICPMatch)
	return sc, nil
}

func (s *SQLite) SaveLead(ctx context.Context, l core.Lead) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO leads (id, name, company, best_score, status, notes, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, company=excluded.company, best_score=excluded.best_score,
			status=excluded.status, notes=excluded.notes, updated_at=excluded.updated_at`,
		l.ID, l.Name, l.Company, l.BestScore, l.Status, l.Notes, l.CreatedAt, l.UpdatedAt); err != nil {
		return fmt.Errorf("save lead: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM lead_signals WHERE lead_id = ?`, l.ID); err != nil {
		return err
	}
	for _, sigID := range l.Signals {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO lead_signals (lead_id, signal_id) VALUES (?, ?)`, l.ID, sigID); err != nil {
			return fmt.Errorf("link lead signal: %w", err)
		}
	}
	return tx.Commit()
}

func (s *SQLite) GetLead(ctx context.Context, id string) (core.Lead, error) {
	var l core.Lead
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, company, best_score, status, notes, created_at, updated_at
		FROM leads WHERE id = ?`, id).Scan(
		&l.ID, &l.Name, &l.Company, &l.BestScore, &l.Status, &l.Notes, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Lead{}, ErrNotFound
	}
	if err != nil {
		return core.Lead{}, fmt.Errorf("get lead: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT signal_id FROM lead_signals WHERE lead_id = ?`, id)
	if err != nil {
		return core.Lead{}, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var sid string
		if err := rows.Scan(&sid); err != nil {
			return core.Lead{}, err
		}
		l.Signals = append(l.Signals, sid)
	}
	return l, rows.Err()
}

func (s *SQLite) ListLeads(ctx context.Context, status string) ([]core.Lead, error) {
	q := `SELECT id, name, company, best_score, status, notes, created_at, updated_at FROM leads`
	var args []any
	if status != "" {
		q += " WHERE status = ?"
		args = append(args, status)
	}
	q += " ORDER BY best_score DESC"
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list leads: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []core.Lead
	for rows.Next() {
		var l core.Lead
		if err := rows.Scan(&l.ID, &l.Name, &l.Company, &l.BestScore, &l.Status, &l.Notes,
			&l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *SQLite) GetCursor(ctx context.Context, source string) (Cursor, error) {
	var c Cursor
	err := s.db.QueryRowContext(ctx,
		`SELECT last_seen_at, last_external_id FROM source_cursors WHERE source = ?`, source).
		Scan(&c.LastSeenAt, &c.LastExternalID)
	if errors.Is(err, sql.ErrNoRows) {
		return Cursor{}, nil // no cursor yet means "from the beginning", not an error
	}
	if err != nil {
		return Cursor{}, fmt.Errorf("get cursor: %w", err)
	}
	return c, nil
}

func (s *SQLite) SetCursor(ctx context.Context, source string, c Cursor) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO source_cursors (source, last_seen_at, last_external_id) VALUES (?,?,?)
		ON CONFLICT(source) DO UPDATE SET last_seen_at=excluded.last_seen_at, last_external_id=excluded.last_external_id`,
		source, c.LastSeenAt, c.LastExternalID)
	if err != nil {
		return fmt.Errorf("set cursor: %w", err)
	}
	return nil
}

func (s *SQLite) AddSpend(ctx context.Context, day string, usd float64) (float64, error) {
	var total float64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO llm_spend (day, usd, calls) VALUES (?, ?, 1)
		ON CONFLICT(day) DO UPDATE SET usd = usd + excluded.usd, calls = calls + 1
		RETURNING usd`, day, usd).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("add spend: %w", err)
	}
	return total, nil
}

func (s *SQLite) GetSpend(ctx context.Context, day string) (Spend, error) {
	sp := Spend{Day: day}
	err := s.db.QueryRowContext(ctx, `SELECT usd, calls FROM llm_spend WHERE day = ?`, day).
		Scan(&sp.USD, &sp.Calls)
	if errors.Is(err, sql.ErrNoRows) {
		return sp, nil // no spend recorded is zero spend, not an error
	}
	if err != nil {
		return Spend{}, fmt.Errorf("get spend: %w", err)
	}
	return sp, nil
}

func (s *SQLite) GetCachedScore(ctx context.Context, hash, model, fp string) (core.Score, bool, error) {
	// score_cache stores the whole Score as one JSON blob, so decode that rather than
	// reconstructing it from columns.
	var jsonStr string
	err := s.db.QueryRowContext(ctx,
		`SELECT score_json FROM score_cache WHERE content_hash=? AND model=? AND icp_fingerprint=?`,
		hash, model, fp).Scan(&jsonStr)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Score{}, false, nil
	}
	if err != nil {
		return core.Score{}, false, fmt.Errorf("get cached score: %w", err)
	}
	var out core.Score
	if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
		return core.Score{}, false, fmt.Errorf("decode cached score: %w", err)
	}
	return out, true, nil
}

func (s *SQLite) PutCachedScore(ctx context.Context, hash, model, fp string, sc core.Score) error {
	blob, err := json.Marshal(sc)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO score_cache (content_hash, model, icp_fingerprint, score_json, cached_at)
		VALUES (?,?,?,?,?)
		ON CONFLICT(content_hash, model, icp_fingerprint) DO UPDATE SET score_json=excluded.score_json, cached_at=excluded.cached_at`,
		hash, model, fp, string(blob), time.Now().UTC())
	if err != nil {
		return fmt.Errorf("put cached score: %w", err)
	}
	return nil
}
