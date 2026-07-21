// Package core owns the domain types and every interface that crosses a package
// boundary. It imports nothing from internal/ — that rule is what lets the feature
// packages be built in parallel without importing each other. See ADR-005.
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
	"unicode"
)

// RawSignal is what a Collector emits, before normalization.
type RawSignal struct {
	Source      string
	ExternalID  string // source-native id, used for cursors and dedup
	URL         string
	Author      string
	Title       string
	Content     string
	PublishedAt time.Time
	Meta        map[string]string // source-specific extras: subreddit, points, …
}

// Signal is a normalized, deduplicated signal.
type Signal struct {
	ID          string // ULID
	Source      string // "hackernews", "reddit:r/golang", "rss:<feed-name>"
	URL         string
	Author      string
	Title       string
	Content     string
	Lang        string // detected; used by the pre-filter
	PublishedAt time.Time
	CollectedAt time.Time
	ContentHash string // sha256 of the normalized title+content
}

// ContentHash returns the deduplication key for a title and body: the sha256 of both
// lowercased, with runs of whitespace collapsed and punctuation dropped.
//
// The same post retrieved from two sources, or the same post retrieved twice with
// different surrounding markup, must produce the same hash — that is the whole point,
// so the normalization here is deliberately aggressive.
func ContentHash(title, content string) string {
	sum := sha256.Sum256([]byte(normalizeForHash(title + " " + content)))
	return hex.EncodeToString(sum[:])
}

func normalizeForHash(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	space := true // leading whitespace is skipped
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			space = false
			continue
		}
		// Whitespace, punctuation and symbols all collapse to a single separator, so
		// "self-hosted", "self hosted" and "self  hosted!" hash alike.
		if !space {
			b.WriteRune(' ')
			space = true
		}
	}
	return strings.TrimRight(b.String(), " ")
}
