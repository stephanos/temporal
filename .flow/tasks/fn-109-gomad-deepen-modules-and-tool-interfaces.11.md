---
satisfies: [R17]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.11 Separate capability collection, pure evaluation and linked projection, with one source-inventory owner

## Description
Stage 3, R17 (S5). `target/capability.go` (1,350 lines) mixes host evidence collection, validation, policy evaluation and linked projection, and adapter source-inventory hashing is exported from `target` only so `deterministicio` can wrap it. Separate the three concerns internally behind the unchanged review interface and give inventory hashing one neutral private owner.

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
## Evidence
- Commits:
- Tests:
- PRs:
