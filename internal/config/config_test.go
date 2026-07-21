package config

import (
	"os"
	"strings"
	"testing"
)

// The config.yaml from PRD §2.2.5, verbatim. The acceptance criterion is that this
// document loads without error, so it is pinned here as a literal rather than read from
// a file that might drift.
const prdExample = `
icp:
  description: >
    Companies 20–500 people with a product adopting LLMs (RAG, agents),
    without a strong in-house ML team. Regions: EU, US. Languages: en, ru.
  positive_signals:
    - "complaining about LLM quality/cost in prod"
    - "hiring an LLM/AI engineer or consultant"
  negative_signals:
    - "student questions"
    - "large corporations with ML departments"

sources:
  hackernews:
    enabled: true
    schedule: "*/30 * * * *"
    keywords: ["RAG production", "LLM cost", "hiring LLM engineer"]
  reddit:
    enabled: true
    schedule: "0 * * * *"
    subreddits: ["LocalLLaMA", "MachineLearning", "golang"]
  rss:
    enabled: true
    schedule: "0 * * * *"
    feeds:
      - "https://example.com/jobs.rss"

llm:
  provider: openai-compatible
  base_url: "http://localhost:11434/v1"
  model: "qwen2.5:14b"
  daily_budget_usd: 2.0

scoring:
  notify_threshold: 70

notifiers:
  telegram:
    chat_id: "..."
  webhook:
    url: "https://..."
`

func TestParsePRDExample(t *testing.T) {
	c, err := Parse([]byte(prdExample))
	if err != nil {
		t.Fatalf("the PRD §2.2.5 example must load, got: %v", err)
	}
	if c.LLM.Model != "qwen2.5:14b" {
		t.Errorf("llm.model = %q", c.LLM.Model)
	}
	if c.Scoring.NotifyThreshold != 70 {
		t.Errorf("notify_threshold = %d", c.Scoring.NotifyThreshold)
	}
	if !c.Sources.HackerNews.Enabled || len(c.Sources.HackerNews.Keywords) != 3 {
		t.Errorf("hackernews source parsed wrong: %+v", c.Sources.HackerNews)
	}
	// Defaults must fill in what the example omits.
	if c.Store.Driver != "sqlite" || c.Server.Addr == "" {
		t.Errorf("defaults not applied: store=%+v server=%+v", c.Store, c.Server)
	}
}

func TestExampleFileLoads(t *testing.T) {
	// config.example.yaml ships in the repo; it must be a valid config so users can copy it.
	if _, err := Load("../../config.example.yaml"); err != nil {
		t.Fatalf("config.example.yaml must be valid: %v", err)
	}
}

func TestMissingRequiredFieldNamesIt(t *testing.T) {
	cases := map[string]string{
		"icp.description": `
llm: {provider: anthropic, model: claude-x}
sources: {hackernews: {enabled: true, keywords: ["x"]}}`,
		"llm.model": `
icp: {description: x}
llm: {provider: anthropic}
sources: {hackernews: {enabled: true, keywords: ["x"]}}`,
		"llm.provider": `
icp: {description: x}
llm: {model: m}
sources: {hackernews: {enabled: true, keywords: ["x"]}}`,
		"sources": `
icp: {description: x}
llm: {provider: anthropic, model: m}
sources: {hackernews: {enabled: false}}`,
	}
	for wantField, doc := range cases {
		_, err := Parse([]byte(doc))
		if err == nil {
			t.Errorf("%s: expected an error, got none", wantField)
			continue
		}
		if !strings.Contains(err.Error(), wantField) {
			t.Errorf("error should name %q, got: %v", wantField, err)
		}
	}
}

func TestInlineSecretRejected(t *testing.T) {
	doc := `
icp: {description: x}
llm: {provider: anthropic, model: m, api_key: "sk-ant-real-key-here"}
sources: {hackernews: {enabled: true, keywords: ["x"]}}`
	_, err := Parse([]byte(doc))
	if err == nil || !strings.Contains(err.Error(), "api_key") {
		t.Fatalf("an inline api_key must be rejected and named, got: %v", err)
	}
	// The _env form is the correct one and must pass.
	ok := strings.Replace(doc, `api_key: "sk-ant-real-key-here"`, `api_key_env: "MY_KEY"`, 1)
	if _, err := Parse([]byte(ok)); err != nil {
		t.Fatalf("the _env form must be accepted: %v", err)
	}
}

func TestFingerprintStableUnderReordering(t *testing.T) {
	a := ICP{Description: "x", PositiveSignals: []string{"a", "b"}, Languages: []string{"en", "ru"}}
	b := ICP{Description: "x", PositiveSignals: []string{"b", "a"}, Languages: []string{"ru", "en"}}
	if a.Fingerprint() != b.Fingerprint() {
		t.Error("reordering a list must not change the fingerprint")
	}
	c := ICP{Description: "x", PositiveSignals: []string{"a", "c"}}
	if a.Fingerprint() == c.Fingerprint() {
		t.Error("changing a signal must change the fingerprint")
	}
}

func TestSecretReadsEnv(t *testing.T) {
	t.Setenv("SH_TEST_KEY", "value")
	got, err := Secret("SH_TEST_KEY")
	if err != nil || got != "value" {
		t.Fatalf("Secret = %q, %v", got, err)
	}
	if _, err := Secret("SH_TEST_UNSET"); err == nil {
		t.Error("an unset variable must error")
	} else if !strings.Contains(err.Error(), "SH_TEST_UNSET") {
		t.Errorf("the error must name the variable, got: %v", err)
	}
	_ = os.Getenv
}
