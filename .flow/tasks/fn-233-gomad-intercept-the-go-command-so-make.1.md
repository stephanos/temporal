---
satisfies: [R4, R8]
---
# fn-233-gomad-intercept-the-go-command-so-make.1 run-guest launcher verb: per-seed execution behind go -exec with intercepted-origin evidence

## Description
**Touches:** tools/gomad/internal/backends/wasm/wasmhost/src/linked.rs and request-local sink owners, tools/gomad/rust/gomad/src/cli/run_guest.rs, tools/gomad/rust/gomad/src/campaign/origin.rs, tools/gomad/testdata/parity, tools/gomad/testdata/parity/**.
Cross-spec ordering (Flow dependencies cannot cross specs): starts after fn-232-gomad-rewrite-the-host-side-in-rust.7 (seed explore) and fn-232-gomad-rewrite-the-host-side-in-rust.11 (install) are done; no Go host implementation.
Public verb invoked by Go's -exec once per test binary. Resolve the installation (explicit, GOMAD_WASM_INSTALLATION; auto-derivation arrives in .3), run the module for the configured seeds sequentially, forward guest stdout verbatim, diagnostics to stderr, retain executions as one campaign per invocation in the Rust lineage with origin: intercepted-go plus the intercepted argv and the wrapper run identity; operator campaigns carry origin: operator. Stop at the first non-successful seed and return its guest failure status or the existing infrastructure status on capacity/fuel/helper terminations; return zero only when all requested seeds succeed; retain unexecuted seeds as unexecuted; last stderr line on failure is the exact replay command when a replayable artifact exists; otherwise explicitly report that no replayable artifact was retained and print no invented command. Qualification refuses intercepted-origin evidence. Early proof point: follow the parent's complete manual preparation recipe (pinned compiler, generated cooperative overlay, installation build tags including test_dep, configuration and admission bindings) and quote the launcher as -exec 'gomad run-guest' against TestFrontendSystemInfo with an explicit installation and show Go reporting PASS and the artifact replaying; stop and report if Go cannot parse forwarded output or -p parallel launchers cannot share one artifact root.

## Acceptance
Implement the parent request-local streaming callback interface in the linked helper (including wasmhost/src/linked.rs and its sink owners), retaining byte-identical bounded output. Prove a marker is observable before guest exit with a controlled handshake. Authenticate per-invocation metadata and retain exact original argv plus distinct invocation and wrapper identities; cover concurrent different-argv invocations.

Hand-written go test -exec invocation passes TestFrontendSystemInfo with Go reporting PASS; the retained artifact replays exactly under the explicit replay verb, is accepted by minimize and inspection, and is refused by qualify with the existing status; origin field present with argv and run identity; infrastructure terminations exit with the existing status and a termination diagnostic, followed by a replay command only when a replayable artifact exists; golden corpus cases for success, first-seed-failure with a later would-pass seed (later seed not executed), failure-with-replay-line, unprepared module, infrastructure termination and standalone structured reporting; Go -exec stdout remains guest-only and the shim never enables --json; all cases pass with byte-identical second regeneration.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
