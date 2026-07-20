# ADR-007: config.yaml is the source of truth; the UI edits the file

Date: 2026-07-21
Status: accepted

## Context

ICP definition, sources, LLM settings and thresholds have to be editable from the UI (PRD §3.2.4)
and also gitable and shareable as code (PRD §1.4.2). Those pull in opposite directions. Options:

1. Config lives in `config.yaml`; `PUT /api/config` validates and rewrites the file; the process
   hot-reloads.
2. Config lives in the database; the YAML file is import-only, a bootstrap seed.
3. Both, with a sync/merge story.

Option 3 needs conflict resolution nobody wants to write for a v0.1. Option 2 is what most SaaS
does and it is what makes "config as code" a marketing claim rather than a fact — the file stops
being authoritative the moment someone clicks Save.

## Decision

`config.yaml` on disk is authoritative. The database stores signals, scores, leads, cursors and
ledgers — never settings. `PUT /api/config` parses and validates the submitted YAML, writes it
atomically (temp file + rename), and triggers a hot reload that rebuilds the collector schedules
and the scorer. On validation failure nothing is written and the endpoint returns 422 with
per-field errors.

Secrets stay out of the file: API keys and tokens are read from the environment and referenced
by name, so the config can be committed to a public repo.

## Consequences

- The file stays diffable, gitable and shareable — the claim is literally true, and an ICP
  written by one user can be pasted into another's repo.
- Hot reload has to rebuild live components. Anything holding config values in a struct field at
  startup will silently serve stale settings; components take a config accessor, not a snapshot.
- Editing the ICP changes the `icp_fingerprint` and therefore invalidates the score cache
  (ADR-003). That is correct — cached reasoning from an old ICP would be misleading — but it
  means a config save can cause a burst of re-scoring cost. The UI warns before saving ICP edits.
- The process needs write access to its own config file. That is a new deployment constraint for
  read-only container filesystems, documented in the README.
