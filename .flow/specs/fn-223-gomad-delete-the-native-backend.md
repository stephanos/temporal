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
- archcheck native ownership tables, inventories, exclusions, `NativeSelected` and `nativeModulePresent` in `internal/archcheck`; the root lint router's native ownership/allowlist readers and routing plus their matching fixtures under `cmd/tools/lintcode/**`, with focused router tests and all retained source-routing protections preserved.
- The `gomad` and `gomad_native` build tags where they only served native; the `gomad_wasm_slim` tag and its guest stubs stay. Task .2 lists every `//go:build gomad` site and decides each.

**Retained fixture consumers.** Redirect the five native-relative reads in WASM backend/integration_test.go (four) and command/command_test.go (one) to the byte-identical promoted io_failure_module fixture; verify both copies' SHA-256 and change no guest/assertion semantics. Preserve the root server module requirement consumed by dynamic WASM fixtures when tidying the parent module.

**Retained.** Flow spec/task records and Git history; inputs and evidence still required by open specs. Closed-owner worktrees, branches and evidence are deleted under AGENTS.md.

**Closed.** fn-128 and fn-149 are closed as won't-do with the owner decision recorded in their Decision Context and in the close-out artifact.

**Execution.** Tasks .1–.3 form one pure-deletion implementation batch under MILESTONES.md, permitted in parallel with the Rust-host first batch only after fn-219 closes. Preserve the initial module-removal build/vet proof, ordinary default Go build/vet/tests, lint, static/architecture checks, model corpora, focused regressions and applicable generated/pin validation. Remove obsolete test consumers with their fixtures and clean reference-bearing documentation before the zero-reference gate. Full changed/integrated-candidate runtime/workload/retained-replay qualification and the complete behavioral reach of test-wasm transfer to fn-232 R12/task .13's final frozen Rust cycle, not another full Go qualification ladder. Source acceptance requires proved ledger/docs/checks and an explicit pending receiver; dependencies still govern integration and completion.

## Source/runtime test partition

The literal unselected full-module test attempt exited 1 and remains a failure in the retained tests.command/tests.log and fullmodule-failure-classification.tsv; it is not relabeled a source pass. That initial table's environment-required labels alone do not establish guest execution. The implementation audit separates 48 environment-sensitive root tests into nine source/admission/preparation roots (task .1) and 39 actual live-guest roots (receiving task fn-232.13). The two additional archcheck failures caused by a concurrent temporary identity probe are invalidated observations, not passes; rerun after probe removal. Preserve the dynamic-server fixture dependency regression and its restored requirement as an actual repair, not an environmental excuse.

Retain the exact positive name lists and selectors in the closeout/input transfer: /home/agent/wt/evidence/fn223/{source-gate-names.txt,live-gate-names.txt,ordinary-skip-selector.txt,source-run-selector.txt}. Against the frozen candidate's package-qualified test inventory, prove the lists are disjoint, the union is exactly 48 roots, every selector matches precisely its named roots/subtrees, and all other source tests run in the ordinary lane. No package exclusion, broad substring selector or blanket environment-based skip is permitted. A new/unclassified root stays required until its actual behavior and owner are established.

The ordinary lane uses the full ./... package universe under test_dep and an exact anchored skip of that union in the default test environment. The positive nine-root lane uses explicit authenticated pinned Go and fn-232 baseline-helper inputs without skipping source assertions; compiler/preparation work is not transferred merely because it needs those inputs. The 39 live roots' full behavioral assertions, fixtures and bounds transfer to the final Rust receiver, in addition to the existing full test-wasm reach; renaming or regrouping tests cannot drop coverage. Both source lanes, ordinary build/vet/lint/static/model/generated checks and focused CLI/router regressions must pass before source acceptance. Keep exact commands, selected/matched inventories and numeric exits; no claim of an unselected full-module pass or completed Go qualification.

## API Contracts

- `gomad` WASM command grammar, defaults, exit statuses, recorded formats and diagnostics are unchanged on the existing CLI fixtures.
- A native installation header is refused with the existing unregistered-backend status; `doctor` no longer has a native row.
- Default `make -C tools/gomad` targets and CI jobs are unchanged in behavior; native targets and jobs no longer exist.
- `go build ./... && go vet ./...` from `tools/gomad` cover the whole module without selection. Source-test acceptance covers the same full package universe through the exact partition below, not an unsupported claim that an unselected live-guest Go run passed.

## Edge Cases & Constraints

- Ordering: this spec starts only after fn-219 is closed; the fn-219 archcheck string check must pass on the default set before any deletion.
- Identity: freeze and identify the native-deletion base and deletion-only candidate, with full exact source inventories and before/after identity inputs and values. Use this pair to attribute deletion effects, retaining the fn-219 inputs as provenance rather than pretending an older candidate includes unrelated changes. Removing the declared runtime glue rotates the shared cluster implementation digest because `internal/model/cluster/implementation.go` embeds those files' exact paths and bytes. The WASM cluster implementation composes that digest, so its boot, retained-execution, scenario-provenance, preparation BuildKey and choice-execution bindings rotate transitively. Record exact before/after inputs and values. Across that deletion-only pair, the choice source digest, helper, WASM overlay/runtime, volume, network, World, broker clock, interrupted-replay, strict-JSON, soft-assertion and WASI implementation identities remain unchanged. Module or tag edits can also change a guest's selected source inventory, BuildKey and choice-execution identity; account for each change with its exact authorized input and transitive cause. Do not infer compilation-key or guest-module changes from preparation-key changes. Any other rotation is a finding until its required deletion input and cause are established and explicitly recorded. Preserve full hash coverage and current-source digests. Separate fn-228 and Rust-host rotations on the later integrated candidate by their own exact inputs and causes; neither excuse an unexplained deletion rotation nor impose a false globally unchanged digest across unrelated batches. Task .3 proves source acceptance and qualification transfer; fn-232 R12/.13 qualifies the final frozen integrated Rust candidate.
- Historical evidence that names native identities stays valid as history; nothing is relabeled.
- `go.sum` pruning after the module removal is a recorded step with `go mod tidy` output retained.
- No WASM semantics, evidence format, helper or slim guest change.
- No CI run, PR or push.

## Acceptance Criteria

- **R1:** `internal/backends/native` and both its `require` and `replace` are gone; `git grep -n -e internal/backends/native -e gomad_native -- tools/gomad Makefile .github/workflows` over Git-tracked source, documentation and build inputs reports no matches (exit 1). Untracked toolchains, caches and historical evidence are not source inputs to this check. `go build ./... && go vet ./...` pass from `tools/gomad` without selection. All ordinary source tests pass across `./...` under `test_dep` with only the exact anchored 48-root union excluded from that lane: nine positively named source/preparation/admission tests pass separately with explicit pinned inputs, and the other 39 actual live-guest roots remain pending at fn-232 R12/task .13. Task .1 records the positive source list and selection proof; task .13 records the exact live list and behavioral mapping. No source assertion, input requirement or budget is weakened, and unresolved classification blocks source acceptance. Errors: any remaining tracked reference or scan error is a finding. Task .2 owns reference-bearing documentation cleanup before this gate, including the CLI and WASM guides; task .3 finishes historical wording and qualification links.
- **R2:** The CLI seam, tagged fixtures and tests, cluster linkname glue, native Make targets, CI jobs and archcheck tables are removed, and the existing WASM CLI fixtures pass byte-for-byte. Errors: a changed WASM output or status is a finding.
- **R3:** An identity ledger under `.flow/artifacts` records every retained WASM identity and preparation binding for the frozen native-deletion base and deletion-only candidate, with complete source inventories, exact causes for declared cluster-glue rotations and proven transitive changes; all other implementation identities are unchanged within that pair. Attribute later fn-228/Rust changes separately. Retain all ordinary source checks and CLI regressions. Transfer the full changed/integrated-candidate runtime qualification, mandatory workload/retained-replay qualification and complete behavioral test-wasm reach required by QUALIFICATION.md to fn-232 R12/task .13, mapped without missing subgates to its final frozen Rust cycle under the unchanged limit. Record receiver acceptance, exact pending gate/fixture mapping and input custody before source closeout; no new full Go ladder is required. The receiver retains actual commands, numeric exits and final candidate/source/helper/runtime/module/model bindings. Errors: unexplained rotations, weakened coverage, stale digests, failed source checks or an unproved transfer block this source acceptance; missing/nonzero final qualification blocks the Rust receiver, never becomes a pass here.
- **R4:** fn-128 and fn-149 are closed as won't-do with the owner decision recorded; retained README, ARCHITECTURE, CLI and WASM guides, MILESTONES and AGENTS no longer describe a native backend except as removed history. The ledger/docs explicitly name fn-232 R12/task .13 as pending receiver for R3/R4's full changed/integrated-candidate qualification and behavioral test-wasm reach, preserving prior failed/interrupted/unrun Go attempts and their dispositions. Errors: native described as available/deferred/qualified, missing receiving acceptance, or a transferred gate relabeled passed without Rust evidence is a finding.

## Boundaries

- No WASM semantics, evidence-format, helper-logic or slim-guest change. Qualification rigor and behavioral coverage remain required; the owner's Rust-first order changes execution ownership to fn-232 R12/.13, not gate meaning or a qualification pass. Preserve live `gomad`-tagged WASM inspection and determinism instrumentation when removing native-only selections.
- Evidence cleanup follows the owner decision of 2026-10-20 in AGENTS.md; it does not delete live operational source or open-owner acceptance inputs.
- No start before fn-219 is closed.
- No CI run, PR or push.

## Decision Context

Owner decision (2026-10-19): retire native now, delete after fn-219. fn-184 isolated the backend and fn-219 removes the last shared dependencies on its sources. Rejected: keeping the frozen module indefinitely (ongoing pin, lint and reader cost for code nobody runs); deleting before fn-219 (would take the model reference and generators with it).

Conductor decision (2026-10-20), under the owner's instruction to follow recommendations autonomously: the original unchanged-identity assumption was incorrect because the retained cluster implementation hashes the native glue that this spec removes. Permit only that declared source-driven rotation, proven transitive bindings and source-inventory changes caused by the exact authorized module/tag edits; retain unchanged implementation identities elsewhere and require full qualification of the frozen changed candidate. Preserve complete hash coverage and current-source digests. Native deletion still starts only after fn-219 closes. This corrects an acceptance assumption before implementation; it does not claim that the original unchanged-identity requirement passed.

Source-test overlap correction (2026-10-20): implementation exposed actual guest execution inside the former unselected full-module source command. Apply the exact nine-source/39-live partition above before acceptance; this implements the existing Rust-first transfer, not an exemption for source/preparation checks. Preserve the failed unselected result, probe-race invalidation and repaired fixture dependency history. Only the final guest behavior transfers; no assertion, budget or behavioral reach is reduced.

Owner-order reconciliation (2026-10-20): MILESTONES now permits this pure-deletion lane in parallel with the Rust-host foundation after fn-219 closes and forbids another Go-host qualification cycle. This supersedes the earlier requirement to rerun the full Go WASM ladder in task .3: all changed/integrated-candidate runtime/workload/retained-replay and behavioral test-wasm obligations transfer explicitly to fn-232 R12/.13. Task .3 may close source acceptance only after ledger/docs/source checks and proved receiver acceptance, while those final gates remain pending. Preserve prior failed/interrupted/unrun attempts and distinguish native-deletion identity causes from unrelated fn-228/Rust changes. No execution success is inferred.

Plan-review corrections (2026-10-20): remove the obsolete archcheck fixture consumer in task .1, move documentation cleanup needed by R1 into task .2, and retire all native smoke jobs before workflow validation. The reference check covers Git-tracked inputs so ignored compiler caches and retained historical artifacts cannot obscure its source contract; every tracked match still fails. Task .3 retains final historical wording and the explicit current-candidate qualification transfer.

## Early proof point

Task .1 removes both the `require` and `replace` declarations and the nested module on a throwaway branch and runs `go build ./... && go vet ./...` on the default set. Removing only `replace` would leave the nonexistent local-only `v0.0.0` requirement, which is not evidence of a source consumer. If anything outside the listed seam, fixtures and glue fails to compile, stop and report: fn-219 missed a consumer.

## Quick commands

```sh
git grep -n -e internal/backends/native -e gomad_native -- tools/gomad Makefile .github/workflows   # expect exit 1 with no matches; scan errors fail
cd tools/gomad && go build ./... && go vet ./...
# Test source acceptance: exact ordinary48 exclusion + positive9 pinned-input lane in task .1.
# Live39 roots: pending final Rust behavior mapping in fn-232.13, not an unselected Go pass.
make -C tools/gomad validate  # applicable generated/pin and model-corpus source checks
# Full runtime/workload/retained-replay and behavioral test-wasm reach: pending fn-232 R12/task .13 on the final Rust candidate, not a new Go ladder
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Native module/references removed; unselected build/vet and every source test pass through the exact ordinary-plus-positive9 partition; live39 transferred. | fn-223-gomad-delete-the-native-backend.1, fn-223-gomad-delete-the-native-backend.2 | — |
| R2 | Seam, fixtures, glue, Make targets, CI jobs and archcheck tables removed; WASM CLI fixtures byte-identical. | fn-223-gomad-delete-the-native-backend.2 | — |
| R3 | Account for deletion-only identities, preserve source checks and prove final Rust qualification transfer. | fn-223-gomad-delete-the-native-backend.3 | — |
| R4 | Native retirement history/docs and explicit pending Rust receiver for qualification. | fn-223-gomad-delete-the-native-backend.3 | — |
