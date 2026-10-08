---
satisfies: [R17]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.11 Separate capability collection, pure evaluation and linked projection, with one source-inventory owner

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Stage 3, R17 (S5). `target/capability.go` (1,350 lines) mixes host evidence collection, validation, policy evaluation and linked projection, and adapter source-inventory hashing is exported from `target` only so `deterministicio` can wrap it. Separate the three concerns internally behind the unchanged review interface and give inventory hashing one neutral private owner.

### Source-inventory lint revival (2026-10-05)

Revive only the remaining QF1012 formatted hash-write correction from reviewed integrated source `4695a9ad18de1aa49e032dad82154f73635e9c8d`. This source-progress exception permits starting while task10 acceptance remains open; preserve that dependency and every original completion criterion. Only the formatted per-file digest write in `internal/sourceinventory/inventory.go` may change. Preserve all other bytes, comments, tests, signatures, framing, traversal, limits, validation/error order, consumer mappings, pins and generated output. Retain explicit infallible SHA-256 write-result handling; add no error branch or production seam.

Before editing, run existing literal inventory/capacity/refusal controls and focused target/adapter consumer controls on unchanged BASE, and reproduce actual pinned unfiltered package lint. Run the same controls on final source, ownership/purity/architecture checks, errortype, formatting and the actual integrated lint gate with original comparison `951c5516e9e7b3066e7e069adda9565cfd68844c` and fix disabled. Inspect generator ownership. Record complete measured diagnostic delta and unreached stages, not subtraction-based claims. Keep historical evidence immutable; retain compact command/source/tool receipts under `task-11/inventory-format-20261005/`.

Root owns Flow, review and commits; one source/cache writer. Fresh independent source-progress review precedes a separate commit. Current-source scoped checks cannot complete original predecessor/preservation/matched-first-baseline/full/default/affected/functional/formal/native Darwin acceptance. Linux remains deferred and nonblocking under fn-128. Existing historical host and no-commit text below is superseded only by current evidence and MILESTONES instruction 5; original task scope/acceptance is retained.

Current correction progress: the sole formatted write is repaired with all other source bytes and existing assertions preserved. BASE/FINAL inventory 8/8 and target 10/10 pass; adapter controls retain the same one pass and two early missing-patched-toolchain failures. Scoped lint reproduces one QF1012 before the edit and reports zero afterward. Architecture/purity/edges, standalone errortype, formatting and check-only validation pass. Root's actual integrated 55-package gate reports 318 findings, with exactly QF1012 removed from the previous 319 and all remaining complete blocks byte-identical; Make exits 2 and integrated errortype is unreached. See [current progress](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-11/inventory-format-20261005/progress.md). Original completion and adapter inventory proof gaps remain open; historical claims below retain their historical source meaning.

**External ordering:** after `fn-108-gomad-reduce-code-size-without-removing.2` (local cleanup of `target/capability.go`; it deletes `validateGoCapabilityClosure`, `:184`).

**Size:** M
**Files:** `tools/gomad3/target/capability.go`, `target/capability_capacity.go`, new private files or sub-packages under `target/internal/`, a neutral private inventory package, `deterministicio/adapter_copy.go` and its callers, `architecture_test.go`, tests.
**Touches:** [tools/gomad3/target/**, tools/gomad3/deterministicio/*.go, tools/gomad3/internal/**, tools/gomad3/architecture_test.go]

### Approach
- Unchanged public contract: `ReviewCapabilityClosure` (`capability.go:195`), `ReviewCapabilities` (`:206`), `CapabilityReview`, `CapabilityFinding`, `UnsupportedCapabilityError`, `VerifyCompatibility` (`:1337`).
- Current concerns in one file: collection (`reviewGoCapabilityPackages` `:266`, `loadBuildOverlay` `:445`, `projectCapabilitySource` `:403`, `projectForeignSources` `:934`, `validateAdapterReplacementInputs` `:1062`, `matchAdapterReplacement` `:1128`); evaluation (`collectCapabilityFindings` `:695`, `collectImportFindings` `:728`, `collectLinknameFindings` `:742`, `builtInSimulationLinknameAllowed` `:782`, `forbiddenImport` `:876`); linked projection (`projectExecutableCapabilityReview` `:583`, `projectDeniedBoundaryFindings` `:631`, `capabilityManifest` `:667`).
- Evaluation becomes a pure function over collected evidence: no filesystem, no process, no clock. It reuses `internal/compatibilitypack` policy (`policy.go`, `v2_selection.go`); do not write a second evaluator or a registration mechanism.
- Inventory: `DigestAdapterSourceInventory` / `digestAdapterSourceInventory` (`capability.go:1182-1241`, limits 5000 files and 512 MiB) is consumed by `target` (`:1163`) and wrapped by `deterministicio/adapter_copy.go:144` for `libc_adapter.go`, `memory_adapter.go`, `xnet_adapter.go`, `grpc_adapter.go`, `adapter_rewrite.go`, `adapter_registry.go:402`. Move the hashing to one neutral private package both import; register its owner in `architecture_test.go`. Keep `AdapterCapacityError` mapping at each consumer.
- Do not split by file size alone: a move that leaves the same call order and knowledge in the caller does not meet R17.
- Equivalence harness: golden canonical `CapabilityReview` JSON for existing fixtures (ordered findings, live and eliminated blockers, adapter inventories) compared byte-for-byte before and after; inventory digests for the embedded adapters unchanged.

### Investigation targets
**Required:**
- `tools/gomad3/target/capability.go` (outline with `grep -n '^func \|^type '`, then the ranges above)
- `tools/gomad3/target/capability_test.go`, `capability_review_test.go`
- `tools/gomad3/internal/compatibilitypack/{policy.go,v2_selection.go}`
- `tools/gomad3/deterministicio/adapter_copy.go:130-160`
- `tools/gomad3/target/internal/livecap/project.go`
**Optional:**
- `tools/gomad3/deterministicio/adapter_rewrite.go:60-110`

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./target/... ./deterministicio/... ./internal/compatibilitypack/...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep . -run 'TestPackageArchitecture|TestCompatibilityPackHasOneOwner|TestExactModuleEdges'
make validate-compatibility
make test-live-capability
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Collection, pure evaluation and linked projection have separate private owners behind the unchanged `ReviewCapabilities` / `ReviewCapabilityClosure` contract; the evaluator performs no host effect and reuses the existing compatibility policy.
- [ ] Adapter source-inventory hashing has one neutral private owner used by both target and adapter preparation, with an architectural owner registered.
- [ ] Golden canonical reviews (ordered findings, live and eliminated blockers, inventories) and embedded-adapter inventory digests are byte-identical before and after.
- [ ] Source drift, invalid overlays or replacements, unsafe bridge directives, capacity exhaustion and malformed linked evidence still fail closed with their existing error types.
- [ ] Exact first-party simulation pins and the allowed bridge directives are unchanged; `make validate-compatibility` passes.
## Done summary
Blocked:
Blocked: R17 is implemented and reviewed (SHIP). Only native darwin/arm64 and linux/amd64 gates remain, and they belong to task 21.

Done (commits 58095d83a9 and c1cf9565fe on gomad-fn109, base 5d093b214d):
- Collection (`target/capability_collection.go`) owns every host effect of a review: listing, overlay, sources, adapter replacement digests and compatibility pack loading.
- Evaluation is pure. `target/internal/capabilitypolicy` makes the policy decisions through the existing `compatibility.SelectPacksForPlatform`, and the exact simulation bridge pins moved unchanged. `target/capability_evaluation.go` validates the evidence shape and projects the decisions.
- Linked projection (`target/capability_linked.go`) reads and narrows the embedded record.
- Callers use `reviewRecordedClosure` and `projectCapabilityReview` instead of sequencing the steps, and error order is unchanged.
- Adapter inventory hashing moved to `internal/sourceinventory`, a new architectural owner that may import only hostfs, which target and deterministicio both use. `target.DigestAdapterSourceInventory` is removed, and the removal is recorded in go-interface-changes.md. The bounded reader moved to `hostfs.ReadBounded`.
- `TestCapabilityEvaluationHasNoHostEffect` enforces evaluator purity, and `TestExactModuleEdges` enforces the inventory owner.

Local evidence (linux/arm64, developmental only, with the uncommitted shim; details in task-11/local-evidence.json):
- Golden canonical reviews were captured from the base implementation and are byte-identical after the change. Real-target canonical reviews are also identical before and after.
- The inventory digest is pinned to the base value.
- Quick commands: exit 1 at baseline (309s) and after (135s), with the identical set of 12 failures. Ten are shim-induced; two are pre-existing pin drift, see below.
- Architecture tests: exit 0. `make validate`, which includes validate-compatibility: exit 0. `make test-live-capability`: exit 0. go vet and gofmt: clean.
- Full `make test-host`: exit 2 (163s and 263s). Every failure reproduces on base sources, apart from three load-timing tests that pass on an isolated rerun.
- golangci-lint: not run, because the repository binary is a darwin build.

Pre-existing, not fixed by design: `TestBuiltInSimulationLinknamesPinCurrentFirstPartySources` and `TestClosureReviewSupportsSimulationFixtureAndRefusesHarnessTests` fail at base. `tools/gomad3sim/runtime_time_toolchain.go` gained a `gomadSimulationTimeCurrent` directive in ad90b462e0 without a pin update, and acceptance keeps the pins unchanged.

Remaining native gates: on darwin/arm64 and linux/amd64 hosts (task 21), full `make -C tools/gomad3 test-host`, the task Quick commands, `make validate-compatibility`, `make test-live-capability` and scoped golangci-lint.

Blocked:
ORIGINAL_QUALIFICATION_OPEN: sourceinventory QF1012 has reviewed source progress;
scoped lint is clean and literal inventory/target, architecture, errortype,
formatting and check-only validation controls pass. Actual integrated lint
still reports 318 findings and Make exits 2, leaving integrated errortype
unreached. Two BASE/FINAL adapter inventory controls fail before hashing because
the patched Go executable is absent. Original predecessor, matched-first-baseline,
complete preservation, full/default/functional/affected-consumer/formal/native
Darwin acceptance remains open where unproved. Linux belongs to fn-128 and
does not block this task. See task-11/inventory-format-20261005/progress.md.
Revive for a separately admitted source correction or changed original-gate
prerequisite; preserve every original requirement.
## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
