---
satisfies: [R1, R4, R5]
---
# fn-234-gomad-overlay-audit-dead-native.1 Delete the dead Go 1.27.1 intercept list and proven-unreferenced overlay leftovers

## Description
Touches: tools/gomad/expected-intercepts-go1.27.1.txt (delete), tools/gomad/internal/backends/wasm/toolchain/**, tools/gomad/internal/model/runtimesource/**, tools/gomad/README.md, tools/gomad/internal/backends/wasm/README.md.
Sweep the overlay and toolchain surface for files no Go source, Makefile target, generator, fixture loader or document reads: grep the module for each filename and the native-era names (gomadIntercept, gomad3, native overlay); record the proving search per file in the done summary. Delete the intercept list (111 net/os intercepts with no defining source; only reference is the closed task fn-114.5). Keep files any WASM gate reads even when named native (qualification/native_crash.go, runner diagnostic-identity-*-native.json): they are not leftovers. Remove documentation mentions of deleted files. Run make validate and the behavioral gate on the fixture workloads; this task changes no overlay bytes, so the gate here is a build/test/doc check.

## Acceptance
Intercept list deleted; every other deletion has a recorded search proving it unreferenced; no file read by a WASM gate deleted; go build/vet/lint, toolchain and runtimesource tests, make validate green; no document names a deleted file; overlay implementation identity unchanged by this task.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
