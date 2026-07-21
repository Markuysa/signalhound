package reddit

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Markuysa/signalhound/internal/core"
)

// created_utc of the fixture posts.
const (
	tsGolangNew  = 1784554920 // t3_r2p003, newest in r/golang
	tsGolangMid  = 1784546400 // t3_r2p002
	tsGolangOld  = 1784491500 // t3_r2p001, only reachable on page 2
	tsLocalLLaMA = 1784550000 // t3_l1p001
)

const (
	testIDEnv     = "TEST_REDDIT_CLIENT_ID"
	testSecretEnv = "TEST_REDDIT_CLIENT_SECRET"
)

// stubTransport answers both the token endpoint and the listing endpoint from testdata,
// with scriptable statuses so a 401, a refresh failure and a 429 can be exercised without
// a network or a live credential.
type stubTransport struct {
	t *testing.T

	tokenStatuses   []int             // consumed per token request; exhausted → 200
	listingStatuses []int             // consumed per listing request; exhausted → 200
	fixtures        map[string]string // "<sub>|<after>" → testdata file

	tokenForms   []url.Values // the form body of every token request
	tokenAuths   []string     // the Authorization header of every token request
	listingPaths []string     // "<sub>|<after>" for every listing request
	listingAuths []string
	userAgents   []string
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.t.Helper()
	s.userAgents = append(s.userAgents, req.Header.Get("User-Agent"))

	respond := func(status int, body string) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Status:     strconv.Itoa(status) + " " + http.StatusText(status),
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	}
	pop := func(queue *[]int) int {
		if len(*queue) == 0 {
			return http.StatusOK
		}
		status := (*queue)[0]
		*queue = (*queue)[1:]
		return status
	}

	if req.URL.Path == "/api/v1/access_token" {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			s.t.Fatalf("read token request body: %v", err)
		}
		form, err := url.ParseQuery(string(raw))
		if err != nil {
			s.t.Fatalf("parse token form %q: %v", raw, err)
		}
		s.tokenForms = append(s.tokenForms, form)
		s.tokenAuths = append(s.tokenAuths, req.Header.Get("Authorization"))

		if status := pop(&s.tokenStatuses); status != http.StatusOK {
			return respond(status, `{"error":"invalid_grant"}`)
		}
		return respond(http.StatusOK, readFixture(s.t, "token.json"))
	}

	// /r/<sub>/new
	sub := strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, "/r/"), "/new")
	key := sub + "|" + req.URL.Query().Get("after")
	s.listingPaths = append(s.listingPaths, key)
	s.listingAuths = append(s.listingAuths, req.Header.Get("Authorization"))

	if status := pop(&s.listingStatuses); status != http.StatusOK {
		return respond(status, `{"message":"Too Many Requests","error":429}`)
	}
	name, ok := s.fixtures[key]
	if !ok {
		s.t.Fatalf("unexpected listing request %q", key)
	}
	return respond(http.StatusOK, readFixture(s.t, name))
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(raw)
}

// newTestCollector wires a collector to the stub with credentials in the environment,
// the way ADR-007 expects them to arrive.
func newTestCollector(t *testing.T, subs ...string) (*Collector, *stubTransport) {
	t.Helper()
	t.Setenv(testIDEnv, "fixture-client-id")
	t.Setenv(testSecretEnv, "fixture-client-secret")

	tr := &stubTransport{
		t: t,
		fixtures: map[string]string{
			"golang|":          "listing_golang.json",
			"golang|t3_r2p001": "listing_golang_page2.json",
			"LocalLLaMA|":      "listing_localllama.json",
		},
	}
	if len(subs) == 0 {
		subs = []string{"golang"}
	}
	return New(subs, testIDEnv, testSecretEnv, &http.Client{Transport: tr}), tr
}

func TestImplementsCollector(t *testing.T) {
	var _ core.Collector = New(nil, "", "", nil) // the interface is this package's contract
	if got := New(nil, "", "", nil).Name(); got != "reddit" {
		t.Errorf("Name() = %q, want %q", got, "reddit")
	}
}

func TestCollectMapsPostsAndPages(t *testing.T) {
	c, tr := newTestCollector(t, "r/golang", "LocalLLaMA")

	got, err := c.Collect(context.Background(), time.Time{})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	// Two pages of r/golang plus one page of r/LocalLLaMA.
	wantIDs := []string{"t3_r2p003", "t3_r2p002", "t3_r2p001", "t3_l1p001"}
	if len(got) != len(wantIDs) {
		t.Fatalf("got %d signals, want %d: %+v", len(got), len(wantIDs), got)
	}
	for i, want := range wantIDs {
		if got[i].ExternalID != want {
			t.Errorf("signal %d ExternalID = %q, want %q", i, got[i].ExternalID, want)
		}
	}

	want := core.RawSignal{
		Source:      "reddit:r/golang",
		ExternalID:  "t3_r2p003",
		URL:         "https://www.reddit.com/r/golang/comments/r2p003/anyone_running_an_llm_scoring_pipeline_in_go/",
		Author:      "gopher_ana",
		Title:       "Anyone running an LLM scoring pipeline in Go in production?",
		PublishedAt: time.Unix(tsGolangNew, 0).UTC(),
	}
	first := got[0]
	if first.Source != want.Source || first.ExternalID != want.ExternalID ||
		first.URL != want.URL || first.Author != want.Author || first.Title != want.Title ||
		!first.PublishedAt.Equal(want.PublishedAt) {
		t.Errorf("post mapped to %+v, want %+v", first, want)
	}
	if !strings.Contains(first.Content, "bringing in outside help") {
		t.Errorf("Content = %q, want the selftext", first.Content)
	}

	// The subreddit is part of the provenance, and "r/golang" in config must normalize.
	if last := got[len(got)-1]; last.Source != "reddit:r/LocalLLaMA" {
		t.Errorf("Source = %q, want reddit:r/LocalLLaMA", last.Source)
	}
	if !got[len(got)-1].PublishedAt.Equal(time.Unix(tsLocalLLaMA, 0).UTC()) {
		t.Errorf("LocalLLaMA PublishedAt = %v", got[len(got)-1].PublishedAt)
	}

	// Paging follows Reddit's own "after" cursor, and stops when it comes back empty.
	wantPaths := []string{"golang|", "golang|t3_r2p001", "LocalLLaMA|"}
	if strings.Join(tr.listingPaths, ",") != strings.Join(wantPaths, ",") {
		t.Errorf("listing requests = %v, want %v", tr.listingPaths, wantPaths)
	}
}

func TestCollectStopsPagingAtCursor(t *testing.T) {
	tests := []struct {
		name      string
		since     time.Time
		wantIDs   []string
		wantPaths []string
	}{
		{
			name:      "cursor inside the first page stops before page two",
			since:     time.Unix(tsGolangMid, 0).UTC(),
			wantIDs:   []string{"t3_r2p003"},
			wantPaths: []string{"golang|"},
		},
		{
			name:      "cursor on page two still fetches page two",
			since:     time.Unix(tsGolangOld, 0).UTC(),
			wantIDs:   []string{"t3_r2p003", "t3_r2p002"},
			wantPaths: []string{"golang|", "golang|t3_r2p001"},
		},
		{
			name:      "since now returns nothing",
			since:     time.Now(),
			wantIDs:   nil,
			wantPaths: []string{"golang|"},
		},
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
			if strings.Join(ids, ",") != strings.Join(tc.wantIDs, ",") {
				t.Errorf("got %v, want %v", ids, tc.wantIDs)
			}
			// Paging must stop at the cursor, not fetch everything and filter after.
			if strings.Join(tr.listingPaths, ",") != strings.Join(tc.wantPaths, ",") {
				t.Errorf("listing requests = %v, want %v", tr.listingPaths, tc.wantPaths)
			}
		})
	}
}

func TestTokenRequestAsksForReadScopeOnly(t *testing.T) {
	c, tr := newTestCollector(t)

	if _, err := c.Collect(context.Background(), time.Time{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(tr.tokenForms) == 0 {
		t.Fatal("no token request was made")
	}
	form := tr.tokenForms[0]
	if got := form.Get("grant_type"); got != "client_credentials" {
		t.Errorf("grant_type = %q, want client_credentials (app-only)", got)
	}
	// ADR-006: a collector credential is never granted a write scope.
	if got := form.Get("scope"); got != "read" {
		t.Errorf("scope = %q, want read", got)
	}
	for _, forbidden := range []string{"submit", "edit", "privatemessages", "modposts", "*"} {
		if strings.Contains(form.Get("scope"), forbidden) {
			t.Errorf("scope %q requests write access, forbidden by ADR-006", form.Get("scope"))
		}
	}
	if !strings.HasPrefix(tr.tokenAuths[0], "Basic ") {
		t.Errorf("token request Authorization = %q, want basic auth", tr.tokenAuths[0])
	}
	// Reddit's API rules require a descriptive User-Agent on every request.
	for i, ua := range tr.userAgents {
		if !strings.Contains(ua, "signalhound") {
			t.Errorf("request %d User-Agent = %q, want it to identify signalhound", i, ua)
		}
	}
	for _, auth := range tr.listingAuths {
		if !strings.HasPrefix(auth, "bearer ") {
			t.Errorf("listing Authorization = %q, want a bearer token", auth)
		}
	}
}

func TestTokenIsCachedAcrossRequests(t *testing.T) {
	c, tr := newTestCollector(t, "golang", "LocalLLaMA")

	if _, err := c.Collect(context.Background(), time.Time{}); err != nil {
		t.Fatalf("first Collect: %v", err)
	}
	if _, err := c.Collect(context.Background(), time.Time{}); err != nil {
		t.Fatalf("second Collect: %v", err)
	}
	// Six listing requests across two ticks, one token.
	if len(tr.tokenForms) != 1 {
		t.Errorf("made %d token requests across %d listings, want 1 — the token must be cached",
			len(tr.tokenForms), len(tr.listingPaths))
	}
}

func TestTokenRefreshedOn401(t *testing.T) {
	c, tr := newTestCollector(t)
	// The first listing is rejected; the refreshed token must be retried on the same page.
	tr.listingStatuses = []int{http.StatusUnauthorized}

	got, err := c.Collect(context.Background(), time.Time{})
	if err != nil {
		t.Fatalf("Collect after refresh: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("got %d signals after refresh, want 3", len(got))
	}
	if len(tr.tokenForms) != 2 {
		t.Errorf("made %d token requests, want 2 (initial + refresh after 401)", len(tr.tokenForms))
	}
	// The rejected page is refetched, not skipped.
	if tr.listingPaths[0] != "golang|" || tr.listingPaths[1] != "golang|" {
		t.Errorf("listing requests = %v, want the 401'd page refetched", tr.listingPaths)
	}
}

func TestAuthErrors(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, tr *stubTransport)
		unsetCred bool
		wantIn    []string
	}{
		{
			name:      "missing credentials",
			unsetCred: true,
			wantIn:    []string{"credentials are missing", testIDEnv, testSecretEnv},
		},
		{
			name: "refresh rejected",
			setup: func(_ *testing.T, tr *stubTransport) {
				tr.tokenStatuses = []int{http.StatusUnauthorized}
			},
			wantIn: []string{"authentication was rejected", "401", "Sources"},
		},
		{
			name: "still 401 after a refresh",
			setup: func(_ *testing.T, tr *stubTransport) {
				tr.listingStatuses = []int{http.StatusUnauthorized, http.StatusUnauthorized}
			},
			wantIn: []string{"rejected right after a refresh", "Sources"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, tr := newTestCollector(t)
			if tc.unsetCred {
				t.Setenv(testIDEnv, "")
				t.Setenv(testSecretEnv, "")
			}
			if tc.setup != nil {
				tc.setup(t, tr)
			}

			_, err := c.Collect(context.Background(), time.Time{})
			if err == nil {
				t.Fatal("Collect succeeded, want an auth error")
			}
			var authErr *AuthError
			if !errors.As(err, &authErr) {
				t.Fatalf("error %v is %T, want *AuthError", err, err)
			}
			// docs/DESIGN.md §5: what happened, then how to fix it.
			if authErr.What == "" || authErr.Fix == "" {
				t.Errorf("AuthError = %+v, want both What and Fix populated", authErr)
			}
			if !strings.Contains(err.Error(), " → ") {
				t.Errorf("message %q does not follow the \"what → fix\" style", err)
			}
			if !strings.HasPrefix(err.Error(), "reddit collector: ") {
				t.Errorf("message %q does not name the collector", err)
			}
			for _, want := range tc.wantIn {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("message %q does not mention %q", err, want)
				}
			}
		})
	}
}

func TestRateLimitAndMalformedResponses(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(tr *stubTransport)
		wantIn string
	}{
		{
			name:   "rate limited",
			setup:  func(tr *stubTransport) { tr.listingStatuses = []int{http.StatusTooManyRequests} },
			wantIn: "unexpected status 429",
		},
		{
			name:   "server error",
			setup:  func(tr *stubTransport) { tr.listingStatuses = []int{http.StatusBadGateway} },
			wantIn: "unexpected status 502",
		},
		{
			name:   "malformed listing",
			setup:  func(tr *stubTransport) { tr.fixtures["golang|"] = "token.json" },
			wantIn: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, tr := newTestCollector(t)
			tc.setup(tr)

			got, err := c.Collect(context.Background(), time.Time{})
			if tc.name == "malformed listing" {
				// A listing-shaped decode of the wrong document yields no children rather
				// than an error; what matters is that it does not panic or invent signals.
				if err != nil {
					t.Fatalf("Collect: %v", err)
				}
				if len(got) != 0 {
					t.Errorf("got %d signals from a non-listing document, want 0", len(got))
				}
				return
			}
			if err == nil {
				t.Fatalf("Collect returned %v, want an error", got)
			}
			// A 429 is the runner's problem to back off from, so it must not masquerade
			// as an auth failure that tells the user to touch their credentials.
			var authErr *AuthError
			if errors.As(err, &authErr) {
				t.Errorf("error %v is an *AuthError, want a plain transport error", err)
			}
			if !strings.Contains(err.Error(), "reddit") {
				t.Errorf("error %q does not name the source", err)
			}
			if !strings.Contains(err.Error(), tc.wantIn) {
				t.Errorf("error %q does not mention %q", err, tc.wantIn)
			}
			if got != nil {
				t.Errorf("Collect returned %v alongside an error, want nil", got)
			}
		})
	}
}
