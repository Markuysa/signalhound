package core

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCoreImportsNothingInternal guards the dependency rule from ADR-005: every feature
// package may depend on core, and core may depend on none of them. Break it and the
// packages stop being independently buildable, which is what makes tickets parallel.
//
// This is asserted rather than documented because it is the one architectural property
// that a single careless import silently destroys.
func TestCoreImportsNothingInternal(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
			continue
		}
		f, err := parser.ParseFile(fset, e.Name(), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", e.Name(), err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, "/internal/") {
				t.Errorf("%s imports %q; core must not import any internal package",
					e.Name(), path)
			}
		}
	}
}

func TestContentHashIgnoresFormatting(t *testing.T) {
	// The same post reached through two sources arrives with different whitespace and
	// punctuation. Dedup depends on those collapsing to one hash.
	same := [][2]string{
		{"Cutting LLM cost", "We  spend\t$4k/mo on   inference."},
		{"cutting llm cost", "We spend $4k/mo on inference."},
		{"Cutting  LLM   Cost!", "we spend $4k/mo on inference"},
	}
	want := ContentHash(same[0][0], same[0][1])
	for _, c := range same[1:] {
		if got := ContentHash(c[0], c[1]); got != want {
			t.Errorf("ContentHash(%q, %q) = %s, want %s (formatting must not matter)",
				c[0], c[1], got[:8], want[:8])
		}
	}

	// Hyphenation is a real difference in how people write, not in what they mean.
	if ContentHash("self-hosted agent", "") != ContentHash("self hosted agent", "") {
		t.Error("hyphenated and spaced forms must hash alike")
	}

	if ContentHash("Cutting LLM cost", "x") == ContentHash("Cutting LLM cost", "y") {
		t.Error("different content must not collide")
	}
}

func TestValidators(t *testing.T) {
	for _, s := range []string{IntentBuying, IntentPain, IntentHiring, IntentResearch, IntentNone} {
		if !ValidIntentType(s) {
			t.Errorf("ValidIntentType(%q) = false", s)
		}
	}
	if ValidIntentType("interested") {
		t.Error("ValidIntentType accepted a value outside the schema")
	}

	for _, s := range []string{LeadNew, LeadReviewed, LeadContacted, LeadIgnored, LeadCustomer} {
		if !ValidLeadStatus(s) {
			t.Errorf("ValidLeadStatus(%q) = false", s)
		}
	}
	if ValidLeadStatus("won") {
		t.Error("ValidLeadStatus accepted a value outside the schema")
	}
}

func TestPackageHasNoStrayFiles(t *testing.T) {
	// core is copied verbatim by nine downstream tickets; a stray file here is a
	// merge conflict waiting to happen.
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{
		"signal.go": true, "score.go": true, "lead.go": true,
		"interfaces.go": true, "errors.go": true, "core_test.go": true,
	}
	for _, e := range entries {
		if !allowed[e.Name()] {
			t.Errorf("unexpected file %q in internal/core", e.Name())
		}
	}
}
