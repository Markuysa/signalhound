package hn

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Markuysa/signalhound/internal/core"
)

// Timestamps of the fixture hits, as created_at_i.
const (
	tsComment = 1784554920 // objectID 40003, newest
	tsStory1  = 1784546400 // objectID 40001
	tsStory2  = 1784491500 // objectID 40002, oldest
)

// fixtureTransport serves recorded Algolia responses from testdata instead of the
// network, and applies the created_at_i bound the way Algolia would — so a collector
// that forgot to send numericFilters fails the since test rather than passing it.
type fixtureTransport struct {
	t     *testing.T
	files map[string]string // query → testdata file

	status  int      // when non-zero, returned for every request
	rawBody string   // when non-empty, returned verbatim
	queries []string // every numericFilters value seen, "" when absent
}

func (f *fixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.t.Helper()
	q := req.URL.Query()
	f.queries = append(f.queries, q.Get("numericFilters"))

	respond := func(status int, body string) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Status:     strconv.Itoa(status) + " " + http.StatusText(status),
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	}

	if f.status != 0 {
		return respond(f.status, `{"error":"upstream said no"}`)
	}
	if f.rawBody != "" {
		return respond(http.StatusOK, f.rawBody)
	}

	name, ok := f.files[q.Get("query")]
	if !ok {
		f.t.Fatalf("unexpected query %q", q.Get("query"))
	}
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		f.t.Fatalf("read fixture: %v", err)
	}

	body := struct {
		Hits []map[string]any `json:"hits"`
	}{}
	if err := json.Unmarshal(raw, &body); err != nil {
		f.t.Fatalf("decode fixture %s: %v", name, err)
	}
	if bound, ok := parseCreatedAtBound(f.t, q.Get("numericFilters")); ok {
		kept := make([]map[string]any, 0, len(body.Hits))
		for _, h := range body.Hits {
			if created, _ := h["created_at_i"].(float64); int64(created) > bound {
				kept = append(kept, h)
			}
		}
		body.Hits = kept
	}
	filtered, err := json.Marshal(body)
	if err != nil {
		f.t.Fatalf("re-encode fixture %s: %v", name, err)
	}
	return respond(http.StatusOK, string(filtered))
}

// parseCreatedAtBound reads the "created_at_i>N" filter the collector is expected to send.
func parseCreatedAtBound(t *testing.T, filter string) (int64, bool) {
	t.Helper()
	if filter == "" {
		return 0, false
	}
	rest, ok := strings.CutPrefix(filter, "created_at_i>")
	if !ok {
		t.Fatalf("numericFilters %q does not bound created_at_i", filter)
	}
	n, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		t.Fatalf("numericFilters %q has a non-numeric bound: %v", filter, err)
	}
	return n, true
}

func newTestCollector(t *testing.T) (*Collector, *fixtureTransport) {
	t.Helper()
	tr := &fixtureTransport{
		t: t,
		files: map[string]string{
			"RAG production": "search_rag_production.json",
			"LLM cost":       "search_llm_cost.json",
		},
	}
	return New([]string{"RAG production", "LLM cost"}, &http.Client{Transport: tr}), tr
}

func TestName(t *testing.T) {
	var _ core.Collector = New(nil, nil) // the interface is the contract this package owes
	if got := New(nil, nil).Name(); got != "hackernews" {
		t.Errorf("Name() = %q, want %q", got, "hackernews")
	}
}

func TestCollectMergesKeywordsAndDeduplicates(t *testing.T) {
	c, tr := newTestCollector(t)

	got, err := c.Collect(context.Background(), time.Time{})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	// Both fixtures contain objectID 40001; it must survive exactly once.
	want := []string{"40001", "40002", "40003"}
	ids := make([]string, 0, len(got))
	for _, s := range got {
		ids = append(ids, s.ExternalID)
	}
	if len(ids) != len(want) {
		t.Fatalf("got %d signals %v, want %d %v", len(ids), ids, len(want), want)
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Errorf("objectID %s returned twice", id)
		}
		seen[id] = true
	}
	for _, id := range want {
		if !seen[id] {
			t.Errorf("objectID %s missing from %v", id, ids)
		}
	}

	// The first run has no cursor, so no bound is sent.
	for _, f := range tr.queries {
		if f != "" {
			t.Errorf("zero since sent numericFilters %q, want none", f)
		}
	}
}

func TestCollectMapsStoryAndComment(t *testing.T) {
	c, _ := newTestCollector(t)

	got, err := c.Collect(context.Background(), time.Time{})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	bySignalID := map[string]core.RawSignal{}
	for _, s := range got {
		bySignalID[s.ExternalID] = s
	}

	story, ok := bySignalID["40001"]
	if !ok {
		t.Fatal("story 40001 missing")
	}
	// A story: title and story_text, and a URL pointing at the HN item rather than the
	// article it links to.
	want := core.RawSignal{
		Source:      "hackernews",
		ExternalID:  "40001",
		URL:         "https://news.ycombinator.com/item?id=40001",
		Author:      "kmartens",
		Title:       "Our RAG pipeline fell over in production at 40 QPS",
		PublishedAt: time.Unix(tsStory1, 0).UTC(),
	}
	if story.Source != want.Source || story.ExternalID != want.ExternalID ||
		story.URL != want.URL || story.Author != want.Author || story.Title != want.Title ||
		!story.PublishedAt.Equal(want.PublishedAt) {
		t.Errorf("story mapped to %+v, want %+v", story, want)
	}
	if !strings.Contains(story.Content, "naive top-k retriever") {
		t.Errorf("story Content = %q, want the story_text", story.Content)
	}

	comment, ok := bySignalID["40003"]
	if !ok {
		t.Fatal("comment 40003 missing")
	}
	// A comment: comment_text as the body, and its parent story's title, since a comment
	// has no title of its own.
	if comment.Title != "The economics of running LLMs in production" {
		t.Errorf("comment Title = %q, want the parent story_title", comment.Title)
	}
	if !strings.Contains(comment.Content, "inference bill went from $800") {
		t.Errorf("comment Content = %q, want the comment_text", comment.Content)
	}
	if comment.Author != "rsvp_pierre" {
		t.Errorf("comment Author = %q, want rsvp_pierre", comment.Author)
	}
	if !comment.PublishedAt.Equal(time.Unix(tsComment, 0).UTC()) {
		t.Errorf("comment PublishedAt = %v, want %v", comment.PublishedAt, time.Unix(tsComment, 0).UTC())
	}
}

func TestCollectHonoursSince(t *testing.T) {
	tests := []struct {
		name  string
		since time.Time
		want  []string
	}{
		{"everything after the oldest hit", time.Unix(tsStory2, 0).UTC(), []string{"40001", "40003"}},
		{"only the newest hit", time.Unix(tsStory1, 0).UTC(), []string{"40003"}},
		{"since now returns nothing", time.Now(), nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, tr := newTestCollector(t)
			got, err := c.Collect(context.Background(), tc.since)
			if err != nil {
				t.Fatalf("Collect: %v", err)
			}
			ids := make([]string, 0, len(got))
			for _, s := range got {
				ids = append(ids, s.ExternalID)
			}
			if len(ids) != len(tc.want) {
				t.Fatalf("got %v, want %v", ids, tc.want)
			}
			for _, id := range tc.want {
				if !slices.Contains(ids, id) {
					t.Errorf("got %v, want it to contain %s", ids, id)
				}
			}
			// The bound has to reach Algolia, not be applied locally after the fact.
			for _, f := range tr.queries {
				if f != "created_at_i>"+strconv.FormatInt(tc.since.Unix(), 10) {
					t.Errorf("numericFilters = %q, want the created_at_i bound for since", f)
				}
			}
		})
	}
}

func TestCollectErrors(t *testing.T) {
	tests := []struct {
		name string
		tr   *fixtureTransport
		want string
	}{
		{"non-2xx", &fixtureTransport{status: http.StatusTooManyRequests}, "unexpected status"},
		{"malformed json", &fixtureTransport{rawBody: `{"hits": [{"objectID":`}, "decode response"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.tr.t = t
			c := New([]string{"RAG production"}, &http.Client{Transport: tc.tr})

			got, err := c.Collect(context.Background(), time.Time{})
			if err == nil {
				t.Fatalf("Collect returned %v, want an error", got)
			}
			if !strings.Contains(err.Error(), "hackernews") {
				t.Errorf("error %q does not name the source", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not mention %q", err, tc.want)
			}
			if got != nil {
				t.Errorf("Collect returned %v alongside an error, want nil", got)
			}
		})
	}
}
