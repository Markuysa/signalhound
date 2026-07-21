// Package config loads and validates config.yaml, which is the source of truth for
// everything the agent does (ADR-007). The database never stores settings.
//
// Secrets are not in this file. Fields ending in _env name an environment variable to
// read the secret from, so config.yaml can be committed to a public repository. Load
// rejects a file that carries a literal-looking credential rather than trusting the
// author to have been careful.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// Config mirrors config.yaml. See PRD §2.2.5 for the canonical example.
type Config struct {
	ICP       ICP                 `yaml:"icp"`
	Sources   Sources             `yaml:"sources"`
	LLM       LLM                 `yaml:"llm"`
	Scoring   Scoring             `yaml:"scoring"`
	Notifiers map[string]Notifier `yaml:"notifiers"`
	Server    Server              `yaml:"server"`
	Store     Store               `yaml:"store"`
}

// ICP describes who the user is looking for. Changing any field here invalidates
// cached scores — see Fingerprint.
type ICP struct {
	Description     string   `yaml:"description"`
	PositiveSignals []string `yaml:"positive_signals"`
	NegativeSignals []string `yaml:"negative_signals"`
	Languages       []string `yaml:"languages"`
	Regions         []string `yaml:"regions"`
}

type Sources struct {
	HackerNews *HackerNewsSource `yaml:"hackernews"`
	Reddit     *RedditSource     `yaml:"reddit"`
	RSS        *RSSSource        `yaml:"rss"`
}

type HackerNewsSource struct {
	Enabled  bool     `yaml:"enabled"`
	Schedule string   `yaml:"schedule"`
	Keywords []string `yaml:"keywords"`
}

type RedditSource struct {
	Enabled         bool     `yaml:"enabled"`
	Schedule        string   `yaml:"schedule"`
	Subreddits      []string `yaml:"subreddits"`
	Keywords        []string `yaml:"keywords"`
	ClientIDEnv     string   `yaml:"client_id_env"`
	ClientSecretEnv string   `yaml:"client_secret_env"`
}

type RSSSource struct {
	Enabled  bool     `yaml:"enabled"`
	Schedule string   `yaml:"schedule"`
	Feeds    []string `yaml:"feeds"`
}

type LLM struct {
	Provider       string  `yaml:"provider"` // openai-compatible | anthropic
	BaseURL        string  `yaml:"base_url"`
	Model          string  `yaml:"model"`
	APIKeyEnv      string  `yaml:"api_key_env"`
	DailyBudgetUSD float64 `yaml:"daily_budget_usd"`
	MaxTokens      int     `yaml:"max_tokens"`
}

type Scoring struct {
	NotifyThreshold int `yaml:"notify_threshold"`
	MinContentLen   int `yaml:"min_content_length"`
}

type Notifier struct {
	Enabled     bool   `yaml:"enabled"`
	ChatID      string `yaml:"chat_id"`
	URL         string `yaml:"url"`
	BotTokenEnv string `yaml:"bot_token_env"`
}

type Server struct {
	Addr        string `yaml:"addr"`
	APITokenEnv string `yaml:"api_token_env"`
}

type Store struct {
	Driver string `yaml:"driver"` // sqlite
	DSN    string `yaml:"dsn"`
}

// Fingerprint returns a hash of the ICP block. It changes whenever any ICP field
// changes and is stable otherwise, including across map and slice ordering.
//
// score_cache is keyed on it so that editing the ICP invalidates cached scores rather
// than serving reasoning that was produced against a different definition of a lead.
func (c Config) Fingerprint() string { return c.ICP.Fingerprint() }

// Fingerprint hashes the ICP definition.
func (i ICP) Fingerprint() string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(i.Description))
	for _, xs := range [][]string{i.PositiveSignals, i.NegativeSignals, i.Languages, i.Regions} {
		// Sorted on a copy: reordering a YAML list is not a semantic change, so it
		// must not invalidate every cached score.
		cp := append([]string(nil), xs...)
		sort.Strings(cp)
		b.WriteByte('\x1f')
		b.WriteString(strings.Join(cp, "\x1e"))
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:16])
}

// FieldError names the config field that is wrong. Load and Validate return these so
// PUT /api/config can answer 422 with per-field errors (ADR-007).
type FieldError struct {
	Field  string
	Reason string
}

func (e FieldError) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

// Errors is a collection of FieldError, so validation reports everything wrong at once
// instead of making the user fix one field per attempt.
type Errors []FieldError

func (e Errors) Error() string {
	parts := make([]string, len(e))
	for i, fe := range e {
		parts[i] = fe.Error()
	}
	return strings.Join(parts, "; ")
}
