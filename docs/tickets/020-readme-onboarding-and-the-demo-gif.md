---
id: 20
title: README, onboarding, and the demo gif
role: dev
depends: [19]
status: todo
---

Write the public face of the project. PRD is explicit that README and screenshot quality are the
last thing to cut, not the first (§3.6).

Spec: `docs/PRD.md` §1.1, §1.4, §3.3, §3.4. Design: ADR-006, ADR-007.


## Likely files
- `README.md`, `LICENSE`, `docs/QUICK_REF.md`, `config.example.yaml` (documentation pass)
- `.github/ISSUE_TEMPLATE/*`

## Acceptance criteria
- [ ] README opens with the elevator pitch from PRD §1.1, a screenshot of the Feed, and a gif of
      the demo mode.
- [ ] A quickstart takes a reader from `docker run` to a first scored signal in under 10
      minutes, with the `--demo` path shown first as the zero-configuration route (PRD §3.3).
- [ ] "No outreach automation" appears as a stated philosophy section with its reasoning, not a
      footnote (ADR-006, PRD §1.4.4).
- [ ] Documents: the full `config.yaml` reference, the environment variable for every secret,
      BYO-LLM setup including a local Ollama example, and the requirement that the process can
      write to its own config file (ADR-007 consequence).
- [ ] AGPL-3.0 `LICENSE` file is present and referenced from the README.
- [ ] `docs/QUICK_REF.md` is filled in from its template and stays under 50 lines.
- [ ] Every command shown in the README has actually been run against the built binary — no
      aspirational snippets.
- [ ] No real API key, token, or live `config.yaml` appears anywhere in the docs (CLAUDE.md).
