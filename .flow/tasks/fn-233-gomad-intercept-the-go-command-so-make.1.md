---
satisfies: [R4, R8]
---
# fn-233-gomad-intercept-the-go-command-so-make.1 run-guest launcher verb: per-seed execution behind go -exec with intercepted-origin evidence

## Description
Touches: gomad/src/cli/run_guest.rs, gomad/src/campaign/origin.rs, testdata/parity, golden corpus.
Cross-spec ordering (Flow dependencies cannot cross specs): starts after fn-232-gomad-rewrite-the-host-side-in-rust.7 (seed explore) and fn-232-gomad-rewrite-the-host-side-in-rust.11 (install) are done; no Go host implementation.
Public verb invoked by Go's -exec once per test binary. Resolve the installation (explicit, GOMAD_WASM_INSTALLATION; auto-derivation arrives in .3), run the module for the configured seeds sequentially, forward guest stdout verbatim, diagnostics to stderr, retain executions as one campaign per invocation in the Rust lineage with origin: intercepted-go plus the intercepted argv and the wrapper run identity; operator campaigns carry origin: operator. Exit with the guest status on the last seed or the existing infrastructure status on capacity/fuel/helper terminations; last stderr line on failure is the exact replay command. Qualification refuses intercepted-origin evidence. Early proof point: run the hand-written GOOS=wasip1 GOARCH=wasm go test -exec gomad run-guest command against TestFrontendSystemInfo with an explicit installation and show Go reporting PASS and the artifact replaying; stop and report if Go cannot parse forwarded output or -p parallel launchers cannot share one artifact root.

## Acceptance
Hand-written go test -exec invocation passes TestFrontendSystemInfo with Go reporting PASS; the retained artifact replays exactly under the explicit replay verb, is accepted by minimize and inspection, and is refused by qualify with the existing status; origin field present with argv and run identity; infrastructure terminations exit with the existing status and one stderr line; golden corpus cases for success, failure-with-replay-line, unprepared module, infrastructure termination and --json rows pass with byte-identical second regeneration.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
