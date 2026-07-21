// Package reddit collects new posts from configured subreddits through the official
// Reddit API, authenticating app-only (client credentials) with read scope.
//
// The package exposes Collect and nothing else. Scheduling and rate limiting belong to
// the collector runner, so those limits stay configurable without touching collector
// code (see docs/ARCHITECTURE.md, "External integrations").
package reddit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Markuysa/signalhound/internal/core"
)

// SourceName prefixes the Source of every signal from this collector, which is
// "reddit:r/<subreddit>" — the subreddit is part of the provenance, not a detail.
const SourceName = "reddit"

const (
	apiBase = "https://oauth.reddit.com"
	webBase = "https://www.reddit.com"

	// The Reddit API rules require a descriptive, identifying User-Agent; a generic or
	// spoofed one is what gets an app rate-limited hardest.
	userAgent = "signalhound/0.1 (+https://github.com/Markuysa/signalhound)"

	pageSize = 100 // Reddit's maximum

	// ponytail: bounds a first run, where there is no cursor and /new would otherwise
	// page back through the whole subreddit. Raise it if a cold start proves to need more.
	maxPages = 10
)

// errUnauthorized marks a 401 from the listing endpoint, which is a refresh signal rather
// than a failure — it stays internal, callers see an AuthError only once a refresh fails.
var errUnauthorized = errors.New("unauthorized")

// Collector polls the configured subreddits for new posts.
type Collector struct {
	subreddits []string
	idEnv      string
	secretEnv  string
	client     *http.Client

	mu          sync.Mutex // guards the cached token; Collect may be called concurrently
	accessToken string
	tokenExpiry time.Time
}

// New returns a collector polling subreddits, reading its client credentials from the
// environment variables named by idEnv and secretEnv (ADR-007 — the config names the
// variables, it never holds the secrets). Empty names fall back to REDDIT_CLIENT_ID and
// REDDIT_CLIENT_SECRET. A nil client falls back to http.DefaultClient.
func New(subreddits []string, idEnv, secretEnv string, client *http.Client) *Collector {
	if idEnv == "" {
		idEnv = "REDDIT_CLIENT_ID"
	}
	if secretEnv == "" {
		secretEnv = "REDDIT_CLIENT_SECRET"
	}
	if client == nil {
		client = http.DefaultClient
	}
	subs := make([]string, 0, len(subreddits))
	for _, s := range subreddits {
		if s = normalizeSubreddit(s); s != "" {
			subs = append(subs, s)
		}
	}
	return &Collector{subreddits: subs, idEnv: idEnv, secretEnv: secretEnv, client: client}
}

// Name implements core.Collector.
func (c *Collector) Name() string { return SourceName }

// Collect polls every configured subreddit for posts newer than since.
func (c *Collector) Collect(ctx context.Context, since time.Time) ([]core.RawSignal, error) {
	var out []core.RawSignal
	for _, sub := range c.subreddits {
		got, err := c.collectSubreddit(ctx, sub, since)
		if err != nil {
			return nil, err
		}
		out = append(out, got...)
	}
	return out, nil
}

func (c *Collector) collectSubreddit(ctx context.Context, sub string, since time.Time) ([]core.RawSignal, error) {
	var out []core.RawSignal
	after := ""
	for range maxPages {
		body, err := c.listing(ctx, sub, after)
		if err != nil {
			return nil, err
		}
		for _, child := range body.Data.Children {
			p := child.Data
			published := time.Unix(int64(p.CreatedUTC), 0).UTC()
			// /new is newest-first, so the first post at or before the cursor means every
			// remaining post is older too — stop rather than page on.
			if !since.IsZero() && !published.After(since) {
				return out, nil
			}
			out = append(out, p.toRawSignal(sub, published))
		}
		if after = body.Data.After; after == "" {
			return out, nil
		}
	}
	return out, nil
}

// listing fetches one page, refreshing the token once if the cached one was rejected.
func (c *Collector) listing(ctx context.Context, sub, after string) (*listingBody, error) {
	body, err := c.fetchListing(ctx, sub, after, false)
	if !errors.Is(err, errUnauthorized) {
		return body, err
	}
	// The token was good enough to be cached and is not accepted now: refresh and retry.
	body, err = c.fetchListing(ctx, sub, after, true)
	if errors.Is(err, errUnauthorized) {
		// A 401 on a token minted seconds ago is a credential problem, not an expiry.
		return nil, &AuthError{
			What: "the access token was rejected right after a refresh",
			Fix:  "check the credentials in " + c.idEnv + " and " + c.secretEnv + ", then reconnect in Sources",
		}
	}
	return body, err
}

func (c *Collector) fetchListing(ctx context.Context, sub, after string, forceToken bool) (*listingBody, error) {
	token, err := c.token(ctx, forceToken)
	if err != nil {
		return nil, err
	}

	q := url.Values{"limit": {fmt.Sprint(pageSize)}}
	if after != "" {
		q.Set("after", after)
	}
	endpoint := apiBase + "/r/" + url.PathEscape(sub) + "/new?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: build request for r/%s: %w", SourceName, sub, err)
	}
	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: fetch r/%s: %w", SourceName, sub, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// 429 included: backing off is the runner's job, so it needs to see the failure.
		return nil, fmt.Errorf("%s: fetch r/%s: unexpected status %s", SourceName, sub, resp.Status)
	}

	var body listingBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("%s: decode listing for r/%s: %w", SourceName, sub, err)
	}
	return &body, nil
}

// normalizeSubreddit accepts "golang", "r/golang" and "/r/golang" alike — all three turn
// up in hand-written config.
func normalizeSubreddit(s string) string {
	s = strings.Trim(strings.TrimSpace(s), "/")
	return strings.TrimPrefix(s, "r/")
}

type listingBody struct {
	Data struct {
		After    string `json:"after"`
		Children []struct {
			Data post `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type post struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"` // fullname, e.g. t3_abc123
	Title      string  `json:"title"`
	SelfText   string  `json:"selftext"`
	Author     string  `json:"author"`
	Permalink  string  `json:"permalink"`
	CreatedUTC float64 `json:"created_utc"`
}

func (p post) toRawSignal(sub string, published time.Time) core.RawSignal {
	// The fullname is what Reddit's own pagination uses, so it is the id worth carrying.
	id := p.Name
	if id == "" {
		id = "t3_" + p.ID
	}
	return core.RawSignal{
		Source:      SourceName + ":r/" + sub,
		ExternalID:  id,
		URL:         webBase + p.Permalink,
		Author:      p.Author,
		Title:       p.Title,
		Content:     p.SelfText,
		PublishedAt: published,
	}
}
