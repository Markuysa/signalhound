package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads, defaults and validates a config file.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // the path is operator-supplied by design
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	return Parse(raw)
}

// Parse is Load without the filesystem, used by PUT /api/config to validate a submitted
// document before anything is written to disk (ADR-007).
func Parse(raw []byte) (Config, error) {
	var c Config
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	dec.KnownFields(true) // a typo'd key is an error, not a silently ignored setting
	if err := dec.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if err := rejectInlineSecrets(raw); err != nil {
		return Config{}, err
	}
	c.applyDefaults()
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

// secretKey matches a YAML key that would hold a credential rather than the name of an
// environment variable holding one.
var secretKey = regexp.MustCompile(`(?im)^\s*[a-z0-9_]*(api_key|apikey|secret|token|password|passwd)\s*:\s*(\S.*)$`)

// rejectInlineSecrets refuses a config that carries a literal credential.
//
// ADR-007 lets config.yaml be committed to a public repository, which only holds if
// secrets are never in it. Relying on the author to remember that is not a control, so
// the loader enforces it: keys are named with an _env suffix and hold a variable name.
func rejectInlineSecrets(raw []byte) error {
	var errs Errors
	for _, m := range secretKey.FindAllStringSubmatch(string(raw), -1) {
		key := strings.TrimSpace(strings.SplitN(m[0], ":", 2)[0])
		if strings.HasSuffix(key, "_env") {
			continue // names a variable, does not hold the value
		}
		value := strings.TrimSpace(m[2])
		if value == "" || strings.HasPrefix(value, "#") {
			continue
		}
		errs = append(errs, FieldError{key,
			"looks like an inline secret: keep credentials in the environment and name the variable here, e.g. " + key + "_env"})
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}
