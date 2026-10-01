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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
