---
satisfies: [R1]
---
# fn-172-gomad-guest-side-cost-experiments-and.2 Committed diagnostic driver and guest-time attribution for Temporal selections

## Description
Add a committed, env-gated Go driver for diagnostic runs of the Temporal workflow and activity selections, plus a guest-stack attribution classifier. Use them to produce the R1 activity flame profile and attribution report.

Performance measurements follow the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

**Size:** M
**Files:** `tools/gomad3/backends/wasm/wasi/diagnostic_profile_test.go` (new: driver, classifier, unit tests), `tools/gomad3/backends/wasm/wasi/process.go` (test-only diagnostic revision switch), `tools/gomad3/backends/wasm/wasi/process_test.go`, `.flow/artifacts/fn172-guest-cost-experiments-<date>.md` (new)
**Touches:** [tools/gomad3/backends/wasm/wasi/diagnostic_profile_test.go, tools/gomad3/backends/wasm/wasi/process.go, tools/gomad3/backends/wasm/wasi/process_test.go, tools/gomad3/backends/wasm/wasi/integration_test.go, .flow/artifacts/fn172-*]

### Approach
- Test-only switch: an unexported `Request` field, like `remoteModel` (`process.go:132`), names an accepted diagnostic configuration revision. The `started` check (`process.go:653-690`) and authorize echo accept it only when set; the release path still requires `PinnedEngine()` (`process.go:217`). Add a negative test: without the switch, a diagnostic revision is refused.
- Driver: follow the scratch driver's shape (`wasi/zz_scratch_profile_test.go`; do not import or modify it). Build `./tests` once with `prototypeBuild` and run the workflow (`^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$`) and activity (`^TestActivityTestSuite$`) selections through `prototypeRequest` + `runDiagnosticProfile` (`integration_test.go:4653`). Gate it on a new root env var (for example `GOMAD_WASM_GUEST_PROFILE_ROOT`), pass the helper by absolute path, and use fixed seeds (0, 1000003, 2000006). Add the guest-profile request field when the helper is a `guest-profile` build.
- Classifier: self time bucketed by the first matching frame from leaf to root: GC (`runtime.gcBgMarkWorker`, `gcDrain*`, `scanobject`, `markroot*`, `gcAssistAlloc*`, `bgsweep`, `sweepone`, write-barrier helpers), scheduler (`runtime.schedule`, `findRunnable`, `gopark`, `park_m`, `goready`, `mcall`, `gogo`, `handleEvent`, `beforeIdle`, `wasm_pc_f_loop`), allocation (`mallocgc`…), test package, `go.temporal.io/server/...`, other runtime, stdlib, other. The Go driver reads the Firefox Profiler JSON (samples → stack → frame → func name tables) and unit-tests the classifier on a small synthetic profile.
- Report: write the activity attribution table, flame file digests, commit, helper/module digests, host and re-run command to the artifact. Raw profiles stay under the root, outside the repo.

### Investigation targets
**Required:**
- `tools/gomad3/backends/wasm/wasi/zz_scratch_profile_test.go` (driver shape only)
- `tools/gomad3/backends/wasm/wasi/integration_test.go:4630-4900` (`requireDiagnosticEvidence`, `runDiagnosticProfile`, `decodeDiagnosticReport`)
- `tools/gomad3/backends/wasm/wasi/process.go:120-240`, `:640-700`

**Optional:**
- `.flow/artifacts/fn156-workflow-profile-optimization-research-2026-10-11.md` (report format precedent)

### Quick commands
```bash
(cd tools/gomad3 && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run '^TestDiagnostic|^TestGuestTransport' ./backends/wasm/wasi)
(cd tools/gomad3 && GOMAD_WASM_GUEST_PROFILE_ROOT=/abs/root GOMAD_WASM_TEST_HELPER=/abs/guestprof/gomad3-wasmhost env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 60m -run '^TestDiagnosticGuestProfile' ./backends/wasm/wasi)
```

## Acceptance
- [ ] The driver is skipped without its env var; the release transport refuses diagnostic revisions unless the test-only switch is set (tests).
- [ ] Classifier unit tests cover each bucket and the leaf-to-root precedence.
- [ ] The artifact contains the activity selection's guest attribution (GC, scheduler, allocation, test code, server, runtime, other) with flame file digests and re-run command.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
