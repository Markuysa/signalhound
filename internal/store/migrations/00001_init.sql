-- +goose Up
CREATE TABLE signals (
    id            TEXT PRIMARY KEY,
    source        TEXT NOT NULL,
    url           TEXT NOT NULL,
    author        TEXT NOT NULL,
    title         TEXT NOT NULL,
    content       TEXT NOT NULL,
    lang          TEXT NOT NULL DEFAULT '',
    published_at  TIMESTAMP NOT NULL,
    collected_at  TIMESTAMP NOT NULL,
    content_hash  TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_signals_content_hash ON signals(content_hash);
CREATE INDEX idx_signals_source ON signals(source);
CREATE INDEX idx_signals_published_at ON signals(published_at);

CREATE TABLE scores (
    signal_id    TEXT PRIMARY KEY REFERENCES signals(id) ON DELETE CASCADE,
    value        INTEGER NOT NULL,
    intent_type  TEXT NOT NULL,
    reasons      TEXT NOT NULL,   -- JSON array
    icp_match    TEXT NOT NULL,   -- JSON array
    draft_reply  TEXT NOT NULL DEFAULT '',
    model        TEXT NOT NULL,
    cost_usd     REAL NOT NULL DEFAULT 0,
    latency_ms   INTEGER NOT NULL DEFAULT 0,
    scored_at    TIMESTAMP NOT NULL
);
CREATE INDEX idx_scores_value ON scores(value);

CREATE TABLE leads (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL DEFAULT '',
    company    TEXT NOT NULL DEFAULT '',
    best_score INTEGER NOT NULL DEFAULT 0,
    status     TEXT NOT NULL DEFAULT 'new',
    notes      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE INDEX idx_leads_status ON leads(status);

CREATE TABLE lead_signals (
    lead_id   TEXT NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    signal_id TEXT NOT NULL REFERENCES signals(id) ON DELETE CASCADE,
    PRIMARY KEY (lead_id, signal_id)
);

CREATE TABLE source_cursors (
    source           TEXT PRIMARY KEY,
    last_seen_at     TIMESTAMP NOT NULL,
    last_external_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE llm_spend (
    day   TEXT PRIMARY KEY,       -- YYYY-MM-DD
    usd   REAL NOT NULL DEFAULT 0,
    calls INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE score_cache (
    content_hash    TEXT NOT NULL,
    model           TEXT NOT NULL,
    icp_fingerprint TEXT NOT NULL,
    score_json      TEXT NOT NULL,
    cached_at       TIMESTAMP NOT NULL,
    PRIMARY KEY (content_hash, model, icp_fingerprint)
);

-- +goose Down
DROP TABLE score_cache;
DROP TABLE llm_spend;
DROP TABLE source_cursors;
DROP TABLE lead_signals;
DROP TABLE leads;
DROP TABLE scores;
DROP TABLE signals;
