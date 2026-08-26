---
satisfies: [R7, R8]
---
# fn-22-deterministic-replay-semantic.7 Expose the bounded orchestration command and root target

## Description
Add the single `umpire replay-minimize-promote --set <result-set> --proposal <file>` command surface (or the repository's matching command binary convention) over the deep replay library and one thin root `umpire-replay-minimize-promote` Make target using SET and PROPOSAL. The target builds/installs the fixed Go command and its `temporal-model-promote` sibling before delegating; it exposes no executable/candidate override. Emit the spec's exact `umpire-replay-report/v1` canonical JSON terminal envelope, identity projection, field order, nullability, enums, and rerun semantics. Emit the bounded sanitized `umpire-replay-progress/v1` stderr stream at every phase/attempt/cleanup/deadline transition with fake-clock coverage. Write sealed proposal bytes with exclusive creation and no overwrite. Implement exact status 0/2/1, terminal stdout/stderr, cancellation, partial-output cleanup, and repeatability contracts without new user-settable runtime/reducer knobs.

**Size:** M
**Files:** `tools/umpire/replay/report.go`, `tools/umpire/replay/progress.go`, `tools/umpire/replay/report_test.go`, `tools/umpire/replay/progress_test.go`, `tools/umpire/cmd/umpire-replay-minimize-promote/main.go`, `tools/umpire/cmd/umpire-replay-minimize-promote/main_test.go`, `Makefile`
**Touches:** [tools/umpire/replay/report.go, tools/umpire/replay/progress.go, tools/umpire/replay/report_test.go, tools/umpire/replay/progress_test.go, tools/umpire/cmd/umpire-replay-minimize-promote/main.go, tools/umpire/cmd/umpire-replay-minimize-promote/main_test.go, Makefile]

## Acceptance
Status 0 is possible only for reproducible complete minimized/irreducible analysis plus an exclusively created sealed proposal. Status 2 covers valid not-reproduced, indeterminate, or exhausted analysis and writes no proposal. Status 1 covers admission/invariant/protocol/compiler/output failures and never reports success. Report bytes, report semantic identity, and allowed rerun differences match the exact spec; progress is bounded, flushed, sanitized, fake-clock deterministic, and always exposes cleanup/cancellation state. An existing proposal path is not overwritten, interrupted/failed writes leave no partial file, the root target delegates exactly, and no model-local Makefile or CI workflow changes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
