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
Implemented Rust minimize site_scope and per-reduction in/out counts, inspect removed-decision symbols and final symbol grouping, and documented the display fields without changing retained minimization evidence. Four complete actual acceptance phases passed on immutable a07dd6e2c404ce9120c448e8568c18906fa2f9aebd87d11ce8d53a7bac8a861b; both semantic goldens match SHA3095bc851f9fa9f821dac792e585f16228975a0ebd0d8dd0a0cafeaedf05f61e. Positive display snapshots authenticate both real outputs and match938B SHA4d50ea6bf4def5cee39e09755672f23d99dc52e2f62642b0e2090fc8a365549d. All16 portable display tests and independent two-output authentication passed in the integrated checkout. Independent same-family reviewer SHIP6506164bda (canonical JSON type repair, meaningful RED/GREEN). Complete raw command receipts and R1–R8 map are referenced in .flow/artifacts/fn231-rust-minimizer-handover-2026-10-22.md; final spec compliance review remains separate.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8536a84a8595b0f5e27295b93885908bd65f07bb, 864859b9f4, a4cfdf5ab7, c63cfc2999
- Tests: python3 tools/gomad/testdata/parity/site-minimizer-display-tests.py (16PASS), python3 tools/gomad/testdata/parity/site-minimizer-display.py --scope-output /home/agent/wt/evidence/fn231-public-provenancefix-01/scope-01 --scope-output /home/agent/wt/evidence/fn231-public-provenancefix-02/scope-02 (exit0), Four full actual scope/semantic acceptance phases:55/119/55/119 command receipts, all terminal0; two-run-comparison.json, AST checker/tests/live-hook and git diff --check exit0
- PRs: