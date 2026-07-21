package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	tokenURL = "https://www.reddit.com/api/v1/access_token"

	// oauthScope is read, and only ever read. ADR-006 forbids granting a collector
	// credential any write scope; naming the scope explicitly keeps that visible here
	// rather than relying on what the client-credentials grant happens to default to.
	oauthScope = "read"

	// expiryMargin retires a token slightly early, so a tick that starts just before
	// expiry does not die holding one that goes stale mid-run.
	expiryMargin = time.Minute
)

// AuthError reports that the collector cannot authenticate.
//
// The two fields are separate because docs/DESIGN.md §5 requires an error to say what
// happened *and* what to do about it; a single free-text message makes the second half
// easy to forget.
type AuthError struct {
	What string
	Fix  string
	Err  error
}

func (e *AuthError) Error() string {
	return SourceName + " collector: " + e.What + " → " + e.Fix
}

func (e *AuthError) Unwrap() error { return e.Err }

// token returns a cached access token, fetching a fresh one when there is none, when the
// cached one has expired, or when force is set — which is what a 401 mid-tick means.
func (c *Collector) token(ctx context.Context, force bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !force && c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	// Read at use, not at construction: ADR-007 keeps secrets in the environment and the
	// config referring to them by name, so config.yaml stays committable.
	id, secret := os.Getenv(c.idEnv), os.Getenv(c.secretEnv)
	if id == "" || secret == "" {
		return "", &AuthError{
			What: "client credentials are missing",
			Fix:  "set " + c.idEnv + " and " + c.secretEnv + " in the environment, then restart",
		}
	}

	form := url.Values{"grant_type": {"client_credentials"}, "scope": {oauthScope}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("%s: build token request: %w", SourceName, err)
	}
	req.SetBasicAuth(id, secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", &AuthError{
			What: "the token request could not be sent",
			Fix:  "check network access to reddit.com",
			Err:  err,
		}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", &AuthError{
			What: "authentication was rejected (" + resp.Status + ")",
			Fix:  "check the credentials in " + c.idEnv + " and " + c.secretEnv + ", then reconnect in Sources",
		}
	}

	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", &AuthError{
			What: "the token response could not be read",
			Fix:  "retry; if it persists, check the credentials in Sources",
			Err:  err,
		}
	}
	if body.AccessToken == "" {
		return "", &AuthError{
			What: "the token response carried no access token",
			Fix:  "check the credentials in " + c.idEnv + " and " + c.secretEnv + ", then reconnect in Sources",
		}
	}

	c.accessToken = body.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(body.ExpiresIn)*time.Second - expiryMargin)
	return c.accessToken, nil
}
