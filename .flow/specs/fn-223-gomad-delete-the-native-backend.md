# Gomad: delete the native backend

## Conversation Evidence

> user: "we want to retire the native backend (not delete yet)"
> user: "yes file these specs"

## Goal & Context

After `fn-219-gomad-promote-the-model-projections-and` lands, nothing in the parent module outside `internal/backends/native` and outside `gomad_native`-tagged files references the native tree. The native backend is then a frozen nested module that no gate compiles by default, whose execution qualification has been deferred under fn-128 and fn-149 since the 2026-10-14 hibernation, and whose only remaining consumers are its own tagged CLI seam, tagged test fixtures, Make targets, CI jobs and archcheck tables. The owner decision of 2026-10-19 is that WASM is the product. This spec removes the native backend and its scaffolding and closes the two deferred qualification specs as won't-do with a recorded owner decision, while following the current closed-spec cleanup policy in AGENTS.md.

Measured on 2026-10-19, the native tree is about 110k Go lines plus the pinned Go 1.27.1 runtime patch and overlay; its subtrees are `toolchain` (34k), `build`, `transcript`, `execution`, `qualification`, `conformance`, `upgrade`, `compatibilitypack`, `generation`, `cmd`, `preparation` and `sourceinventory`.

## Architecture & Data Models

**Removed.**

- The nested module `internal/backends/native` in full, including `go.mod`/`go.sum`, the runtime patch and overlay, and both its `require` and `replace` declarations in `tools/gomad/go.mod`.
- The `gomad_native` CLI seam: `cmd/gomad/internal/cli/native_enabled*.go`, `native_disabled.go` and their tests; the `nativeBuiltIn`, `registerNative`, `configureNative`, `resolveInstallation` and `dispatchPrivateMode` seam functions collapse to the WASM-only path in `launcher.go` and `cli.go`, keeping the WASM grammar, defaults, exit statuses and recorded formats byte-for-byte on the existing CLI fixtures. The "native backend not built in" refusal for a native installation header becomes an ordinary unregistered-backend refusal with the existing status.
- Tagged fixtures and tests: `internal/backends/testfixture/native.go`, `internal/evidence/record/testfixture/native.go`, every `*_native_test.go` under `internal/runner` and `cmd/gomad/internal/cli`, and `internal/runner/internal/exploration/integration_test.go` if it is native-only (task .1 verifies). Retire its obsolete `TestSimulationGateSelectsProcessNetworkHandles` archcheck consumer in the same step, before task .1's full-module test gate.
- The cluster linkname glue against `internal/gomadio` and `internal/gomadsim` symbols in `internal/model/cluster/runtime_domain.go`, `runtime_network.go`, `runtime_process.go`, `runtime_process_model.go` and `runtime_volume.go`, plus the native-only `runtime_time_toolchain.go` selection. The stock unavailable-runtime behavior remains; update the embedded source inventory when files are removed and keep complete coverage of retained owners.
- Native Make targets in `tools/gomad/Makefile` and the root `Makefile` (`toolchain`, `test-host`, `test-toolchain`, `overlay-test`, `test-simulation`, `core-qualification`, `upgrade-dossier`, `compatibility-pack-qualification`, `test-builder`, `test-live-capability`, `test-runtime`, `test-upstream`, `tests-qualification-generate`, `generate-native`, `validate-native`, the soak target and the `GOMAD_LINT_TAGS` selection); the native jobs and the `native` path-filter output in `.github/workflows/gomad.yml`. Retire the Linux and Darwin native jobs and routing in `gomad-smoke.yml`; delete that workflow if no retained jobs remain. Validate workflow source without dispatching CI.
- archcheck native ownership tables, inventories, exclusions, `NativeSelected` and `nativeModulePresent` in `internal/archcheck`.
- The `gomad` and `gomad_native` build tags where they only served native; the `gomad_wasm_slim` tag and its guest stubs stay. Task .2 lists every `//go:build gomad` site and decides each.

**Retained.** Flow spec/task records and Git history; inputs and evidence still required by open specs. Closed-owner worktrees, branches and evidence are deleted under AGENTS.md.

**Closed.** fn-128 and fn-149 are closed as won't-do with the owner decision recorded in their Decision Context and in the close-out artifact.

**Execution.** Tasks .1–.3 form one implementation batch under MILESTONES.md. Preserve the initial module-removal build/vet proof, remove obsolete test consumers with their fixtures, and clean reference-bearing documentation before the zero-reference gate. Validate the integrated batch and complete each task only when the batch evidence proves its requirements; dependencies govern integration and completion.

## API Contracts

- `gomad` WASM command grammar, defaults, exit statuses, recorded formats and diagnostics are unchanged on the existing CLI fixtures.
- A native installation header is refused with the existing unregistered-backend status; `doctor` no longer has a native row.
- Default `make -C tools/gomad` targets and CI jobs are unchanged in behavior; native targets and jobs no longer exist.
- `go build ./... && go vet ./...` and `go test -tags test_dep ./...` from `tools/gomad` cover the whole module with no selection.

## Edge Cases & Constraints

- Ordering: this spec starts only after fn-219 is closed; the fn-219 archcheck string check must pass on the default set before any deletion.
- Identity: removing the declared runtime glue rotates the shared cluster implementation digest because `internal/model/cluster/implementation.go` embeds those files' exact paths and bytes. The WASM cluster implementation composes that digest, so its boot, retained-execution, scenario-provenance, preparation BuildKey and choice-execution bindings rotate transitively. Record exact before/after inputs and values. The choice source digest, helper, WASM overlay/runtime, volume, network, World, broker clock, interrupted-replay, strict-JSON, soft-assertion and WASI implementation identities remain unchanged. Module or tag edits can also change a guest's selected source inventory, BuildKey and choice-execution identity; account for each change with its exact authorized input and transitive cause. Do not infer compilation-key or guest-module changes from preparation-key changes. Any other rotation is a finding until its required deletion input and cause are established and explicitly recorded. Preserve full hash coverage and current-source digests; task .3 requalifies the final frozen candidate.
- Historical evidence that names native identities stays valid as history; nothing is relabeled.
- `go.sum` pruning after the module removal is a recorded step with `go mod tidy` output retained.
- No WASM semantics, evidence format, helper or slim guest change.
- No CI run, PR or push.

## Acceptance Criteria

- **R1:** `internal/backends/native` and both its `require` and `replace` are gone; `git grep -n -e internal/backends/native -e gomad_native -- tools/gomad Makefile .github/workflows` over Git-tracked source, documentation and build inputs reports no matches (exit 1). Untracked toolchains, caches and historical evidence are not source inputs to this check. `go build ./... && go vet ./... && go test -tags test_dep ./...` pass from `tools/gomad` with no selection. Errors: any remaining tracked reference or scan error is a finding. Task .2 owns reference-bearing documentation cleanup before this gate, including the CLI and WASM guides; task .3 finishes historical wording and qualification links.
- **R2:** The CLI seam, tagged fixtures and tests, cluster linkname glue, native Make targets, CI jobs and archcheck tables are removed, and the existing WASM CLI fixtures pass byte-for-byte. Errors: a changed WASM output or status is a finding.
- **R3:** An identity ledger under `.flow/artifacts` records every retained WASM identity and preparation binding before and after deletion, with exact causes for the declared cluster-glue rotations and proven transitive changes. All remaining implementation identities are unchanged. On the final frozen changed candidate, the current WASM gates required by `internal/backends/wasm/QUALIFICATION.md` pass, including runtime qualification, mandatory workload qualification with retained replay, and `make -C tools/gomad test-wasm`; retain commands, numeric exits and candidate/source/helper/module/model bindings. Errors: unexplained rotations, weakened hash coverage, stale digests, missing qualification evidence or nonzero required exits block completion.
- **R4:** fn-128 and fn-149 are closed as won't-do with the owner decision recorded; retained README, ARCHITECTURE, CLI and WASM guides, MILESTONES and AGENTS no longer describe a native backend except as removed history. Errors: a remaining description of native as available, deferred or qualified is a finding.

## Boundaries

- No WASM semantics, evidence-format, helper-logic, slim-guest or qualification-contract change. Current-candidate requalification is required for the declared identity changes. Preserve live `gomad`-tagged WASM inspection and determinism instrumentation when removing native-only selections.
- Evidence cleanup follows the owner decision of 2026-10-20 in AGENTS.md; it does not delete live operational source or open-owner acceptance inputs.
- No start before fn-219 is closed.
- No CI run, PR or push.

## Decision Context

Owner decision (2026-10-19): retire native now, delete after fn-219. fn-184 isolated the backend and fn-219 removes the last shared dependencies on its sources. Rejected: keeping the frozen module indefinitely (ongoing pin, lint and reader cost for code nobody runs); deleting before fn-219 (would take the model reference and generators with it).

Conductor decision (2026-10-20), under the owner's instruction to follow recommendations autonomously: the original unchanged-identity assumption was incorrect because the retained cluster implementation hashes the native glue that this spec removes. Permit only that declared source-driven rotation, proven transitive bindings and source-inventory changes caused by the exact authorized module/tag edits; retain unchanged implementation identities elsewhere and require full qualification of the frozen changed candidate. Preserve complete hash coverage and current-source digests. Native deletion still starts only after fn-219 closes. This corrects an acceptance assumption before implementation; it does not claim that the original unchanged-identity requirement passed.

Plan-review corrections (2026-10-20): remove the obsolete archcheck fixture consumer in task .1, move documentation cleanup needed by R1 into task .2, and retire all native smoke jobs before workflow validation. The reference check covers Git-tracked inputs so ignored compiler caches and retained historical artifacts cannot obscure its source contract; every tracked match still fails. Task .3 retains final historical wording and current-candidate qualification.

## Early proof point

Task .1 removes both the `require` and `replace` declarations and the nested module on a throwaway branch and runs `go build ./... && go vet ./...` on the default set. Removing only `replace` would leave the nonexistent local-only `v0.0.0` requirement, which is not evidence of a source consumer. If anything outside the listed seam, fixtures and glue fails to compile, stop and report: fn-219 missed a consumer.

## Quick commands

```sh
git grep -n -e internal/backends/native -e gomad_native -- tools/gomad Makefile .github/workflows   # expect exit 1 with no matches; scan errors fail
cd tools/gomad && go build ./... && go vet ./... && go test -tags test_dep ./...
cd tools/gomad && make test-wasm
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Native module, replace and every reference removed; default build, vet and tests pass without a selection. | fn-223-gomad-delete-the-native-backend.1, fn-223-gomad-delete-the-native-backend.2 | — |
| R2 | Seam, fixtures, glue, Make targets, CI jobs and archcheck tables removed; WASM CLI fixtures byte-identical. | fn-223-gomad-delete-the-native-backend.2 | — |
| R3 | Account for declared identity changes and requalify the frozen WASM candidate. | fn-223-gomad-delete-the-native-backend.3 | — |
| R4 | fn-128 and fn-149 closed as won't-do; docs describe native only as removed history. | fn-223-gomad-delete-the-native-backend.3 | — |
