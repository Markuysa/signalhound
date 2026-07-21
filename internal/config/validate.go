package config

import (
	"fmt"
	"os"
	"strings"
)

// Defaults applied before validation, so a minimal config.yaml still runs.
const (
	DefaultAddr            = "127.0.0.1:8080"
	DefaultStoreDriver     = "sqlite"
	DefaultStoreDSN        = "signalhound.db"
	DefaultNotifyThreshold = 70
	DefaultMinContentLen   = 40
	DefaultMaxTokens       = 1024
	DefaultLLMAPIKeyEnv    = "SIGNALHOUND_LLM_API_KEY"
	DefaultAPITokenEnv     = "SIGNALHOUND_API_TOKEN"
)

// applyDefaults fills in what the user may omit. It never overwrites a set value.
func (c *Config) applyDefaults() {
	if c.Server.Addr == "" {
		c.Server.Addr = DefaultAddr
	}
	if c.Server.APITokenEnv == "" {
		c.Server.APITokenEnv = DefaultAPITokenEnv
	}
	if c.Store.Driver == "" {
		c.Store.Driver = DefaultStoreDriver
	}
	if c.Store.DSN == "" {
		c.Store.DSN = DefaultStoreDSN
	}
	if c.Scoring.NotifyThreshold == 0 {
		c.Scoring.NotifyThreshold = DefaultNotifyThreshold
	}
	if c.Scoring.MinContentLen == 0 {
		c.Scoring.MinContentLen = DefaultMinContentLen
	}
	if c.LLM.MaxTokens == 0 {
		c.LLM.MaxTokens = DefaultMaxTokens
	}
	if c.LLM.APIKeyEnv == "" {
		c.LLM.APIKeyEnv = DefaultLLMAPIKeyEnv
	}
}

// Validate reports every problem at once. A missing required field names that field
// rather than panicking later at the point of use.
func (c Config) Validate() error {
	var errs Errors

	if strings.TrimSpace(c.ICP.Description) == "" {
		errs = append(errs, FieldError{"icp.description", "required: the scorer has nothing to match against without it"})
	}

	switch c.LLM.Provider {
	case "openai-compatible", "anthropic":
	case "":
		errs = append(errs, FieldError{"llm.provider", "required: one of openai-compatible, anthropic"})
	default:
		errs = append(errs, FieldError{"llm.provider", fmt.Sprintf("unknown provider %q: want openai-compatible or anthropic", c.LLM.Provider)})
	}
	if strings.TrimSpace(c.LLM.Model) == "" {
		errs = append(errs, FieldError{"llm.model", "required"})
	}
	if c.LLM.DailyBudgetUSD < 0 {
		errs = append(errs, FieldError{"llm.daily_budget_usd", "must not be negative"})
	}

	if c.Scoring.NotifyThreshold < 0 || c.Scoring.NotifyThreshold > 100 {
		errs = append(errs, FieldError{"scoring.notify_threshold", "must be between 0 and 100"})
	}

	if c.Store.Driver != "sqlite" {
		errs = append(errs, FieldError{"store.driver", fmt.Sprintf("unsupported driver %q: only sqlite ships in v0.1 (ADR-002)", c.Store.Driver)})
	}

	if !c.anySourceEnabled() {
		errs = append(errs, FieldError{"sources", "no source is enabled: the agent would collect nothing"})
	}
	if hn := c.Sources.HackerNews; hn != nil && hn.Enabled && len(hn.Keywords) == 0 {
		errs = append(errs, FieldError{"sources.hackernews.keywords", "required when the source is enabled"})
	}
	if r := c.Sources.Reddit; r != nil && r.Enabled && len(r.Subreddits) == 0 {
		errs = append(errs, FieldError{"sources.reddit.subreddits", "required when the source is enabled"})
	}
	if rs := c.Sources.RSS; rs != nil && rs.Enabled && len(rs.Feeds) == 0 {
		errs = append(errs, FieldError{"sources.rss.feeds", "required when the source is enabled"})
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (c Config) anySourceEnabled() bool {
	return (c.Sources.HackerNews != nil && c.Sources.HackerNews.Enabled) ||
		(c.Sources.Reddit != nil && c.Sources.Reddit.Enabled) ||
		(c.Sources.RSS != nil && c.Sources.RSS.Enabled)
}

// Secret reads the value of the named environment variable. It returns a FieldError
// naming the variable when it is unset, so a missing key is reported the same way a
// missing config field is.
func Secret(envVar string) (string, error) {
	if envVar == "" {
		return "", FieldError{"<secret>", "no environment variable named"}
	}
	v := os.Getenv(envVar)
	if v == "" {
		return "", FieldError{envVar, "environment variable is unset or empty"}
	}
	return v, nil
}

// LLMAPIKey returns the model API key from the environment.
func (c Config) LLMAPIKey() (string, error) { return Secret(c.LLM.APIKeyEnv) }

// APIToken returns the HTTP API token from the environment.
func (c Config) APIToken() (string, error) { return Secret(c.Server.APITokenEnv) }
