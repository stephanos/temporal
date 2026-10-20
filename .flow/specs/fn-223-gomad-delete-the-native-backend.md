# Gomad: delete the native backend

## Conversation Evidence

> user: "we want to retire the native backend (not delete yet)"
> user: "yes file these specs"

## Goal & Context

After `fn-219-gomad-promote-the-model-projections-and` lands, nothing in the parent module outside `internal/backends/native` and outside `gomad_native`-tagged files references the native tree. The native backend is then a frozen nested module that no gate compiles by default, whose execution qualification has been deferred under fn-128 and fn-149 since the 2026-10-14 hibernation, and whose only remaining consumers are its own tagged CLI seam, tagged test fixtures, Make targets, CI jobs and archcheck tables. The owner decision of 2026-10-19 is that WASM is the product. This spec removes the native backend and its scaffolding and closes the two deferred qualification specs as won't-do with a recorded owner decision, while following the current closed-spec cleanup policy in AGENTS.md.

Measured on 2026-10-19, the native tree is about 110k Go lines plus the pinned Go 1.27.1 runtime patch and overlay; its subtrees are `toolchain` (34k), `build`, `transcript`, `execution`, `qualification`, `conformance`, `upgrade`, `compatibilitypack`, `generation`, `cmd`, `preparation` and `sourceinventory`.

## Architecture & Data Models

**Removed.**

- The nested module `internal/backends/native` in full, including `go.mod`/`go.sum`, the runtime patch and overlay, and the `replace` for it in `tools/gomad/go.mod`.
- The `gomad_native` CLI seam: `cmd/gomad/internal/cli/native_enabled*.go`, `native_disabled.go` and their tests; the `nativeBuiltIn`, `registerNative`, `configureNative`, `resolveInstallation` and `dispatchPrivateMode` seam functions collapse to the WASM-only path in `launcher.go` and `cli.go`, keeping the WASM grammar, defaults, exit statuses and recorded formats byte-for-byte on the existing CLI fixtures. The "native backend not built in" refusal for a native installation header becomes an ordinary unregistered-backend refusal with the existing status.
- Tagged fixtures and tests: `internal/backends/testfixture/native.go`, `internal/evidence/record/testfixture/native.go`, every `*_native_test.go` under `internal/runner` and `cmd/gomad/internal/cli`, and `internal/runner/internal/exploration/integration_test.go` if it is native-only (task .1 verifies).
- The cluster linkname glue against `internal/gomadio` and `internal/gomadsim` symbols in `internal/model/cluster/runtime_network.go`, `runtime_process.go` and `runtime_volume.go`; the `ErrRuntimeUnavailable` fallbacks become the only implementation.
- Native Make targets in `tools/gomad/Makefile` and the root `Makefile` (`toolchain`, `test-host`, `test-toolchain`, `overlay-test`, `test-simulation`, `core-qualification`, `upgrade-dossier`, `compatibility-pack-qualification`, `test-builder`, `test-live-capability`, `test-runtime`, `test-upstream`, `tests-qualification-generate`, `generate-native`, `validate-native`, the soak target and the `GOMAD_LINT_TAGS` selection); the native jobs and the `native` path-filter output in `.github/workflows/gomad.yml` and the native smoke manifest reference in `gomad-smoke.yml`.
- archcheck native ownership tables, inventories, exclusions, `NativeSelected` and `nativeModulePresent` in `internal/archcheck`.
- The `gomad` and `gomad_native` build tags where they only served native; the `gomad_wasm_slim` tag and its guest stubs stay. Task .2 lists every `//go:build gomad` site and decides each.

**Retained.** Flow spec/task records and Git history; inputs and evidence still required by open specs. Closed-owner worktrees, branches and evidence are deleted under AGENTS.md.

**Closed.** fn-128 and fn-149 are closed as won't-do with the owner decision recorded in their Decision Context and in the close-out artifact.

## API Contracts

- `gomad` WASM command grammar, defaults, exit statuses, recorded formats and diagnostics are unchanged on the existing CLI fixtures.
- A native installation header is refused with the existing unregistered-backend status; `doctor` no longer has a native row.
- Default `make -C tools/gomad` targets and CI jobs are unchanged in behavior; native targets and jobs no longer exist.
- `go build ./... && go vet ./...` and `go test -tags test_dep ./...` from `tools/gomad` cover the whole module with no selection.

## Edge Cases & Constraints

- Ordering: this spec starts only after fn-219 is closed; the fn-219 archcheck string check must pass on the default set before any deletion.
- Identity: deleting the overlay cannot rotate any WASM identity, because fn-219 already removed every shared consumer; task .3 records the WASM identities as unchanged by rerunning the identity probes used in fn-184 and fn-219.
- Historical evidence that names native identities stays valid as history; nothing is relabeled.
- `go.sum` pruning after the module removal is a recorded step with `go mod tidy` output retained.
- No WASM semantics, evidence format, helper or slim guest change.
- No CI run, PR or push.

## Acceptance Criteria

- **R1:** `internal/backends/native` and its `replace` are gone; `grep -rn "internal/backends/native\|gomad_native"` over `tools/gomad`, both Makefiles, `.github/workflows` and `internal/archcheck` returns nothing; `go build ./... && go vet ./... && go test -tags test_dep ./...` pass from `tools/gomad` with no selection. Errors: any remaining reference is a finding.
- **R2:** The CLI seam, tagged fixtures and tests, cluster linkname glue, native Make targets, CI jobs and archcheck tables are removed, and the existing WASM CLI fixtures pass byte-for-byte. Errors: a changed WASM output or status is a finding.
- **R3:** WASM identities (helper digest, overlay identity, choice, model and volume identities) are recorded as unchanged in a close-out artifact under `.flow/artifacts`, and `make -C tools/gomad test-wasm` exits 0 on the current candidate. Errors: an identity rotation is a finding and blocks completion.
- **R4:** fn-128 and fn-149 are closed as won't-do with the owner decision recorded; README, ARCHITECTURE and MILESTONES no longer describe a native backend except as removed history. Errors: a remaining description of native as available, deferred or qualified is a finding.

## Boundaries

- No change to WASM behavior, evidence, helper, slim guest build or qualification.
- Evidence cleanup follows the owner decision of 2026-10-20 in AGENTS.md; it does not delete live operational source or open-owner acceptance inputs.
- No start before fn-219 is closed.
- No CI run, PR or push.

## Decision Context

Owner decision (2026-10-19): retire native now, delete after fn-219. fn-184 isolated the backend and fn-219 removes the last shared dependencies on its sources, so deletion is a pure removal with no identity risk. Rejected: keeping the frozen module indefinitely (ongoing pin, lint and reader cost for code nobody runs); deleting before fn-219 (would take the model reference and generators with it).

## Early proof point

Task .1 removes only the `replace` and the nested module on a throwaway branch and runs `go build ./... && go vet ./...` on the default set. If anything outside the listed seam, fixtures and glue fails to compile, stop and report: fn-219 missed a consumer.

## Quick commands

```sh
cd tools/gomad && grep -rn "internal/backends/native\|gomad_native" . ../../Makefile ../../.github/workflows   # expect empty
cd tools/gomad && go build ./... && go vet ./... && go test -tags test_dep ./...
cd tools/gomad && make test-wasm
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Native module, replace and every reference removed; default build, vet and tests pass without a selection. | fn-223-gomad-delete-the-native-backend.1, fn-223-gomad-delete-the-native-backend.2 | — |
| R2 | Seam, fixtures, glue, Make targets, CI jobs and archcheck tables removed; WASM CLI fixtures byte-identical. | fn-223-gomad-delete-the-native-backend.2 | — |
| R3 | WASM identities unchanged and test-wasm green, recorded in a close-out artifact. | fn-223-gomad-delete-the-native-backend.3 | — |
| R4 | fn-128 and fn-149 closed as won't-do; docs describe native only as removed history. | fn-223-gomad-delete-the-native-backend.3 | — |
