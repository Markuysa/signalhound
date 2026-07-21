// Package hn collects Hacker News stories and comments through the Algolia HN Search
// API, which is free and needs no credentials.
//
// The package exposes Collect and nothing else. Retries, rate limiting and scheduling
// belong to the collector runner, so those limits stay configurable without touching
// collector code (see docs/ARCHITECTURE.md, "External integrations").
package hn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Markuysa/signalhound/internal/core"
)

// SourceName is the Source every RawSignal from this collector carries.
const SourceName = "hackernews"

const (
	// search_by_date rather than search: the runner polls incrementally, so newest-first
	// is the ordering that pairs with a created_at_i lower bound.
	searchURL   = "https://hn.algolia.com/api/v1/search_by_date"
	itemURLBase = "https://news.ycombinator.com/item?id="
)

// Collector searches Hacker News for each configured keyword.
type Collector struct {
	keywords []string
	client   *http.Client
}

// New returns a collector searching keywords. A nil client falls back to
// http.DefaultClient; tests inject one that serves recorded fixtures.
func New(keywords []string, client *http.Client) *Collector {
	if client == nil {
		client = http.DefaultClient
	}
	return &Collector{keywords: keywords, client: client}
}

// Name implements core.Collector.
func (c *Collector) Name() string { return SourceName }

// Collect searches every keyword and merges the results, keeping the first hit for any
// objectID seen twice — one item routinely matches several keywords.
func (c *Collector) Collect(ctx context.Context, since time.Time) ([]core.RawSignal, error) {
	var out []core.RawSignal
	seen := make(map[string]bool)
	for _, kw := range c.keywords {
		hits, err := c.search(ctx, kw, since)
		if err != nil {
			return nil, err
		}
		for _, h := range hits {
			if h.ObjectID == "" || seen[h.ObjectID] {
				continue
			}
			seen[h.ObjectID] = true
			out = append(out, h.toRawSignal())
		}
	}
	return out, nil
}

func (c *Collector) search(ctx context.Context, keyword string, since time.Time) ([]hit, error) {
	q := url.Values{"query": {keyword}}
	if !since.IsZero() {
		// Algolia's only incremental knob. Omitted on the first run, where since is the
		// zero time and a bound would be meaningless.
		q.Set("numericFilters", "created_at_i>"+strconv.FormatInt(since.Unix(), 10))
	}

	// ponytail: first page only (20 hits per keyword per tick). Add paging if a keyword
	// can produce more than that between two ticks — after downtime, most likely.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("%s: build request for %q: %w", SourceName, keyword, err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: search %q: %w", SourceName, keyword, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("%s: search %q: unexpected status %s", SourceName, keyword, resp.Status)
	}

	var body struct {
		Hits []hit `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("%s: decode response for %q: %w", SourceName, keyword, err)
	}
	return body.Hits, nil
}

// hit is one Algolia search result. Stories and comments share the shape and are told
// apart by which text field is populated.
type hit struct {
	ObjectID    string `json:"objectID"`
	Title       string `json:"title"`
	StoryTitle  string `json:"story_title"`
	StoryText   string `json:"story_text"`
	CommentText string `json:"comment_text"`
	Author      string `json:"author"`
	CreatedAtI  int64  `json:"created_at_i"`
}

func (h hit) toRawSignal() core.RawSignal {
	// A story carries title and story_text; a comment carries comment_text and names its
	// parent story in story_title, which is the only title a comment has.
	title, content := h.Title, h.StoryText
	if h.CommentText != "" {
		title, content = h.StoryTitle, h.CommentText
	}
	return core.RawSignal{
		Source:      SourceName,
		ExternalID:  h.ObjectID,
		URL:         itemURLBase + h.ObjectID,
		Author:      h.Author,
		Title:       title,
		Content:     content,
		PublishedAt: time.Unix(h.CreatedAtI, 0).UTC(),
	}
}
