---
satisfies: [R2]
---
# fn-184-gomad-native-backend-isolation.2 Register the native backend through one CLI seam

## Description
Move every native use in cmd/gomad/internal/cli (launcher.go registry and header handling, application.go provider construction and private modes, analyze.go, compare_support.go, doctor.go, qualify_set.go, working_directory.go) behind a native_enabled.go / native_disabled.go pair selected by the mechanism from task .1. The disabled variant refuses a native installation header with the existing unsupported-configuration status and a message naming the selection, reports native as not built in from `doctor`, and omits native-only commands from usage while keeping the WASM grammar, defaults, exit statuses and diagnostics unchanged. Add CLI tests for the disabled build (refusal text and status, usage output, doctor report) and keep the enabled-build tests passing under the selection. Coordinate with fn-180/fn-181 CLI replay edits: rebase onto their landed state or batch with them.

**Size:** M
**Files:** tools/gomad/cmd/gomad/internal/cli/{launcher.go,application.go,analyze.go,compare_support.go,doctor.go,qualify_set.go,working_directory.go,native_enabled.go,native_disabled.go}, cli tests, tools/gomad/cmd/gomad/testdata/**

## Acceptance
- [x] Native imports in the CLI exist only in the enabled seam file(s)
- [x] Default build refuses a native installation with the existing status and a message naming the selection; no WASM fallback
- [x] Existing CLI fixtures pass byte-for-byte for WASM commands on both builds
- [x] Enabled build passes the existing native CLI tests under the selection — remaining selected CLI failures equal the pre-isolation baseline (deferred native toolchain)
## Done summary
Native provider, capabilities, private modes and native-only commands register through one `gomad_native` seam (`native_enabled.go`/`native_disabled.go`, `d40debd4db`, fixes `424cc83bbf`). Default builds refuse native installations with the existing unsupported-configuration status and a message naming the selection, no WASM fallback; `doctor` reports native as not built in. Source review SHIP at `68f70ceec8`. Detail: `.flow/artifacts/fn184-closeout-2026-10-17.md`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d40debd4db, 424cc83bbf
- Tests: cd tools/gomad && go test -tags test_dep ./cmd/gomad/internal/cli, cd tools/gomad && go test -tags test_dep,gomad_native ./cmd/gomad/internal/cli ./internal/runner
- PRs:
- Detailed evidence: `.flow/artifacts/fn184-closeout-2026-10-17.md` (R-ID map, commits, commands, exits, reviews, fn-185 R6 transfer)
