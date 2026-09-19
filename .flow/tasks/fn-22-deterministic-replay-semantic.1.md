---
satisfies: [R1, R2]
---

# fn-22-deterministic-replay-semantic.1 Admit the Case-native replay subject and semantic violation key
## Description
Define strict replay admission over canonical Case, exact preparation Profile/catalog identity, and one closed matching Run/Verdict. Derive the semantic violation key from Case/Contract identity, violated terminal state, responsible clause, and supporting Observation roles while excluding fresh Run transport values.

**Size:** M
**Touches:** `tools/umpire/replay/subject.go`, `tools/umpire/replay/subject_test.go`

## Acceptance
- [ ] Crossed, stale, noncanonical, incomplete, satisfied, duplicate, and N+1 inputs fail before target effects.
- [ ] Equivalent per-Run identities and timestamps do not change the semantic key; any semantic binding change does.
- [ ] No persisted replay bundle, audit digest, trust store, or compatibility reader is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
