---
satisfies: [R2]
---
# fn-105-gomad-follow-ups-deferred-scope.2 D2: shared retention policy without merging strategy transactions

## Description
Origin: fn-102.3 (F8 R3). Originally deferred 2026-09-29 with D1; revived 2026-09-30 under fn-108 R7, reused by fn-109. Depends on D1. Reuse or transfer this obligation exactly once during task breakdown. Brief: .flow/tasks/fn-102-gomad-architecture-consolidate.3.md. Fn-108 supplies current delivery and qualification criteria.

## Acceptance
Revival is recorded. Satisfy original fn-102 R3 together with fn-108 R7 and its preservation/verification requirements. Preserve separate strategy transactions, retention limits, publication failure visibility, recovery, and replay evidence. Cross-referencing or transferring the task does not claim completion.

## Done summary
D2 is delivered through fn-108-gomad-reduce-code-size-without-removing.6 (fn-108 R7), which fn-105 R2 names as its owner. Success-retention decisions, capacity mapping and artifact-input composition now have one private stateless owner in `tools/gomad3/runner/retention.go` (`decideSuccessRetention`, `successRetention.annotate`, `successPublicationFailure`, `executionArtifactInput`); seed ordinal commits, atomic exploration rounds and replay-verified corpus admission keep their separate transactions, and callers still advance novelty and counters only at their existing commit points. Characterization was written first; 31 fixed-identity projections of journal records and manifests are byte-identical before and after, and the public API and CLI help are unchanged. Production Go is down 20 code lines. Gates passed on darwin/arm64 with `-count=1` (runner tests, architecture tests, validate, test-harness, world-test, test-host, gomad3sim, integration, smoke qualification 4/4 with exact replay, plus guided and choice-exploration CLI runs). linux/amd64 was not run. Evidence and the SHIP review are in `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/task6-evidence.md` and `task6-review.md`. No commits; the user owns commits.

stage: impl-review - ran under fn-108.6 (raw codex bridge on working-tree diff; commits forbidden) (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: see fn-108-gomad-reduce-code-size-without-removing.6 evidence: .flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/task6-evidence.md
- PRs: