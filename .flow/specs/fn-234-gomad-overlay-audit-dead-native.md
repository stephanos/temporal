# fn-234-gomad-overlay-audit-dead-native Gomad: overlay audit, dead native leftovers and inspection split

## Conversation Evidence

> user: "should we do another pass over the Go overlays? to see if we can simplify/shrink/minimize any of it without changing behavior?"
> user: "one-spec, two-task audit: delete the dead intercept list and any other native leftovers the sweep turns up, then split inspection code out of the runtime file with the behavioral gate above."

## Goal & Context

The cooperative runtime overlay is the part of Gomad that stays in Go after the Rust host rewrite: about 1,700 lines of `package runtime` and `package net` source spliced into a pinned stock Go 1.27.1 tree by an anchored rewrite table, plus the shared choice semantics the Rust helper is held equal to. A survey on 2026-10-20 found it lean, with two findings worth acting on. First, the checked-in intercept list for Go 1.27.1 names 111 net and os intercept functions that no source defines any more; nothing in the build, tests, generator or documentation reads it, and its only reference is a closed native-era task. It is a leftover of the deleted native overlay. Second, the inspection and debug support (the `gomadDebug*` state and functions: 25 functions and roughly 160 references) is interleaved with the scheduling, random-draw and timer code in the single wasip1 runtime file, so the determinism-critical path cannot be reviewed on its own.

This spec is a bounded audit, not a rewrite. It removes dead overlay-adjacent files and splits the inspection code into its own overlay file. Guest behavior is unchanged by construction: the same declarations end up in `package runtime` of the patched tree, only in two files instead of one. Shrinking capability is out of scope.

## Architecture & Data Models

**Dead-file sweep.** Within the Gomad module's overlay and toolchain surface (the toolchain package, the runtime-source package and module-root data files), a file is dead when no Go source, Makefile target, generator, test fixture loader or document reads it. The sweep is a recorded grep over the module plus `make validate`; each deletion names the search that proved it unreferenced. Files that carry `native` in their name but are read by WASM gates (the SQLite storage-crash controls, the runner's diagnostic-identity fixtures) are not leftovers and stay; the milestones already record that their filename does not make them native gates.

**Inspection split.** The wasip1 runtime overlay file is divided into two embedded sources: the existing cooperative runtime file keeps scheduling, random draws, timers, notes, choice recording and the host protocol; a new inspection file receives every `gomadDebug*` declaration (state, buffers, `gomadDebugMain`, `gomadDebugFindRunnable`, `gomadDebugAllowed`, dump and probe plumbing) and nothing else. Both carry the same `//go:build wasip1 && wasm` constraint and are copied into the patched runtime tree beside each other. The rewrite table is untouched: its anchors reference names, not files. The overlay implementation identity is the ordered digest of its embedded sources, so adding a file changes that identity and every derived installation identity; this is expected and recorded, not a defect.

**Behavioral gate.** Before changing embedded sources or refreshing caches, retain the pre-split source revision, patched-tree declaration list, actual compiled modules and their hashes, installation/configuration bytes and identities, compiler/helper/host identities, workload arguments, seeds, profiles and bounds for the toolchain choices fixture, helper feature cases and `TestFrontendSystemInfo` through the Temporal installation. Build and retain the corresponding post-split modules and configurations separately, with their actual new identities. Execute both versions freshly on the same fixed host/helper and compiler, using matching workload arguments, seeds, profiles and bounds; retain each version's raw results and its own identity-matching replay evidence. Compare outcome class and termination, first-divergence ordinal/reason and failure-group partitions over corresponding workload/seed labels. Missing pre-split inputs, missing executions or a semantic difference fail the gate; old observations alone are not a fresh execution. Identity/hash differences are retained, not normalized into equality. Conformance corpora remain unchanged and `make validate` must pass. Patched-tree equality is checked directly: the concatenated set of top-level declarations in the patched `runtime` package is identical before and after the split, as a sorted list of declaration names and kinds.

**Landing.** One batch, one commit per task, preparation and compile caches refreshed for the new identity only after the pre-split inputs are retained. The frozen pre-fn-228 Go baseline and its authenticated extracted source that fn-232 preserves are not modified. Its host-parity lane continues using that original source through its baseline-source argument and retains its existing archive/source hash checks; pointing it at the edited checkout is not supported. This overlay gate is a separate old-overlay/new-overlay comparison, not evidence supplied by Go/Rust host agreement on one overlay. Do not replace the frozen baseline, weaken its source authentication, or claim that a shared regression passing both hosts proves overlay preservation. Coordinate changed-candidate integration before final qualification; no earlier qualification result qualifies the new overlay identity.

## API Contracts

- No CLI, flag, status, environment variable, manifest format or wire schema changes.
- The overlay produces a patched `runtime` package whose declaration set (names and kinds, sorted) is byte-identical before and after the split; a diff of that list is a finding.
- `ImplementationSHA256` changes once, by adding the new embedded source to its ordered input; the identity string prefix and the digest construction are unchanged.
- Deleted files are listed in the task's done summary with the grep that proved each unreferenced.

## Edge Cases & Constraints

- The `.go.txt` suffix on overlay sources is deliberate: it keeps `package runtime` files out of the host module's build, vet, gofmt and linters and makes them embeddable. The new inspection file follows the same convention; no editor mappings are added.
- The split must not move or rename any non-debug declaration, and must not change declaration order within the inspection code beyond the move itself; `go/format` normalization already applied by the overlay builder is the only formatting change allowed.
- A debug symbol that is referenced from a rewrite-table anchor (for example `gomadDebugInspecting` in the scheduler and clock rewrites) stays a package-level name; the split changes which file declares it, never its name or type.
- Installations, preparation caches and compiled-engine caches derived from the previous identity are stale after this lands and must be rebuilt; the existing identity refusals handle a stale cache, nothing silently reuses one.
- No capability removal, no new overlay behavior, no change to what the runtime refuses.
- No PR, push or CI execution.

## Acceptance Criteria

- **R1:** The Go 1.27.1 intercept list and every other file in the overlay and toolchain surface that the recorded sweep proves unreferenced are deleted, with the proving search recorded per file; files read by any WASM gate stay. Errors: a deletion without a recorded search is a finding; a build, test or `make validate` failure after deletion is a finding.
- **R2:** The wasip1 runtime overlay is split into a cooperative runtime file and an inspection file containing exactly the `gomadDebug*` declarations; the sorted declaration list of the patched `runtime` package is identical before and after. Errors: a declaration missing, duplicated or renamed fails the declaration comparison with its name.
- **R3:** `ImplementationSHA256` includes the new source in its ordered input; the toolchain tests cover the new identity and the rebuilt overlay verifies its published copies. Errors: a stale installation is refused by the existing identity check, never reused.
- **R4:** Behavioral gate: capture and authenticate the pre-split modules/configurations before editing sources or refreshing caches, retain the corresponding post-split inputs separately, then freshly execute both versions of every fixture workload with matching arguments, seeds, profile and bounds on the same fixed host/helper/compiler. Compare outcome class/termination, first-divergence ordinal/reason and failure-group partitions; retain each version's actual identities and own replay evidence. Conformance corpora stay unchanged and `make validate` is green. Errors: missing inputs/executions or any semantic difference block the task; byte differences in records/modules are expected, and Go/Rust agreement on only one overlay does not satisfy this gate. Frozen fn-232 baseline inputs and authentication remain unchanged.
- **R5:** The toolchain README or WASM implementation guide names the two overlay files, states the `.go.txt` rationale in one paragraph, and records the identity change. Errors: a document that still names the deleted intercept list is a finding.

## Boundaries

- No capability removal, no shrinking of the choice, timer, network or note semantics.
- No change to the rewrite table or the stock Go version.
- No editor configuration for `.go.txt` files.
- No deletion outside the overlay and toolchain surface; Go host-side files named `native` are fn-232's retirement scope, not this audit's.
- No change to the frozen pre-fn-228 Go baseline fn-232 preserves.
- No PR, push or CI run.

## Decision Context

The owner asked whether another pass over the overlays could simplify or shrink them without changing behavior. The survey found the overlay already small and factored once (no duplicated functions between the shared semantics and the runtime file), so the honest answer was a bounded audit: one confirmed dead file and one readability split, each with a mechanical proof. The declaration-list comparison is the structural proof that the split cannot change behavior; the fresh-execution gate is the same standard fn-232 uses, so this spec adds no second notion of equivalence. Rejected: moving inspection behind a build tag so slim guests drop it (changes module bytes and behavior for slim guests, a capability decision, not a cleanup); rewriting the anchored rewrite table into patch files (the anchors are what keep the overlay honest against the pinned stock source); keeping the intercept list as documentation (it documents code that no longer exists).

## Quick commands

```sh
cd tools/gomad && go test ./internal/backends/wasm/toolchain ./internal/model/runtimesource && make validate
make -C tools/gomad test-wasm
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Dead intercept list and proven-unreferenced leftovers deleted. | .1 | — |
| R2 | Inspection split with identical declaration set. | .2 | — |
| R3 | Identity includes the new source; stale caches refused. | .2 | — |
| R4 | Behavioral gate on fresh executions. | .1, .2 | — |
| R5 | Documentation of files, rationale and identity change. | .1, .2 | — |
