---
satisfies: [R5, R6]
---
# fn-231-gomad-site-aware-minimizer-ordering-and.2 Symbolized reporting in minimize --json and inspect

**Touches:** tools/gomad/rust/gomad/src/cli/{minimize.rs,inspect.rs}, tools/gomad/testdata/parity/**, tools/gomad/CLI.md
**Semantic references only:** tools/gomad/internal/runner/minimize_operation.go, runner/inspect/inspect.go, CLI.md

## Description
Implement once on the Rust host after fn-232.9 and the existing site-spec predecessors; the Go locations below document existing semantics only. Port the behavior into the Rust ownership surface above and extend the shared golden/behavioral corpus. Do not create or extend a Go-host implementation.
Emit per accepted reduction `removed_in_scope`/`removed_out_of_scope` and the top-level `site_scope` block in `minimize --json`. In inspect, join each accepted reduction's removed decisions to symbols and add `final_forced_by_symbol` grouping the final forced decisions by package and function with counts; unavailable symbols group under their reason. Golden tests for both outputs; CLI.md documents the fields.

## Acceptance
Golden minimize and inspect JSON on a fixture minimized artifact show the new fields exactly as the spec's API Contracts shape; the minimization evidence block in the record is byte-unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
