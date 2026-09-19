---
satisfies: [R1, R5]
---
# fn-46-export-lean-model-module-impact-index.1 Extract the shared ModelLint package loader

## Description
Extract the current effectful package inventory/build/OLean/reconciliation pipeline for R1 without changing lint policy.

**Size:** M
**Files:** `model/ModelLint.lean`, `model/ModelLint/PackageModules.lean`, `model/ModelLint/PackageModulesTests.lean`, `model/ModelLint/ImportGraphTests.lean`, `model/Tools/LeanImportGraph/Metadata.lean`, focused adjacent metadata tests, minimal `model/lakefile.lean` test wiring if needed.
**Touches:** [model/ModelLint.lean, model/ModelLint/PackageModules.lean, model/ModelLint/PackageModulesTests.lean, model/ModelLint/ImportGraphTests.lean, model/Tools/LeanImportGraph/Metadata.lean, model/Tools/LeanImportGraph/MetadataTests.lean, model/lakefile.lean]

### Approach
- Move source discovery, build, region lifetime, capture and reconciliation orchestration behind `ModelLint.PackageModules`; reuse the existing `Tools.LeanImportGraph.Metadata.load` traversal and extend its lookup/read seam for accumulated independent failures. Quiet/captured build behavior is new work, not already provided by the current loader.
- Capture child stdout/stderr: let `umpire-lint` replay it to original channels, and let the exporter discard successful build chatter while retaining failure diagnostics.
- Stop after discovery failure, source issues or build failure. Continue independently known metadata nodes after lookup/read failures, sort qualified issues and return no partial result; inaccessible descendants cannot be claimed examined. On metadata success report reconciliation issues and architecture violations as current lint does.
- Inject only focused process/metadata seams and keep loaded compacted regions alive until every consumer completes.
- Preserve full external closure through validation, diagnostic categories, build exclusions, import-policy checks, existing comments and subsequent declaration-lint execution. Do not fold the exporter's stricter root-identity guard into existing lint discovery.
- Capture child streams without deadlock. Wire PackageModulesTests and any metadata tests into the existing umpire-lint-tests runner and actually execute them, rather than only compiling test modules.

### Investigation targets
**Required** (read before coding):
- `model/ModelLint.lean:16` — build sequence; graph phases at 41 and subsequent declaration lint below.
- `model/Tools/LeanImportGraph/Metadata.lean:17` — existing traversal, compacted regions and first-failure behavior.
- `model/Tools/LeanSourceInventory.lean:89` — sorted validation; confined current-root inventory at 205.
- `model/ModelLint/ImportGraph.lean` — current external-aware reconciliation and isolation policy.
- `model/ModelLint/ImportGraphTests.lean:639` — real external metadata controls; executable runner at 664.

### Key context
Lean 4.33.1 compiled environments/OLean metadata are the authority; do not parse source imports or OLean bytes directly.

2026-09-10: start only after fn-83 .15 is done (Flow cannot express the cross-spec task dependency). fn-83 .10 added `Umpire.Command` and moved the success authoring modules; refresh module rows against the tree at that point, and again for fn-85's new modules once they land.

Capture exact original lint success/controlled-violation diagnostics and source/build/metadata phase ordering before edits. Preserve external bridge, stale-owned skip, uniqueness and missing-OLean regressions. Use injected phase counters and lifetime ownership assertions for independent failure and region-release tests. No no-op region-count check as the sole lifetime proof.

### Quick commands
`cd model && mise exec -- lake -q build umpire-lint-tests umpire-lint && mise exec -- lake exe umpire-lint-tests`

Run the existing controlled-violation fixture with its expected nonzero exit and exact channels, and the shared-loader success path, before and after extraction. Record explicit suite execution and terminal results.

### Execution constraints
Preserve unrelated dirty source and existing comments; no staging, commits or pushes. Re-anchor delivered fn-75/76/78 and final fn-77 source before implementation; serialize shared source edits without adding artificial semantic dependencies. Run Lean jobs serially. Capture original touched bytes and relevant trust/metadata baselines before editing. New tests must run through registered roots. No new dependencies, import-policy relaxation, generated API drift/CI expansion, or cancellation work. Required nonfixing Go lint may retain only the exact verified inherited set; new findings and killed/missing-exit gates are failures.
## Acceptance
- [x] `umpire-lint` and the future exporter consume the same package-loader result and output policy. `ModelLint.PackageModules` owns the pipeline and both policies (`replayed`, `quieted`); `umpire-lint` consumes them today. **The exporter itself is task `.2`**, so what this task can deliver is a loader and a policy it can consume, not a second consumer of them.
- [x] Existing lint policy, success line, controlled-violation diagnostic, and child stream channel assignment remain unchanged. `make lint-model` at the 163 baseline; the success line still emitted; `umpire-lint-tests --controlled-violation` still exits 1 with its diagnostic on stderr and nothing on stdout; the synthetic suite passes.
- [x] Discovery/build failures stop later phases; multiple simultaneous source/metadata/reconciliation issues are accumulated and sorted within their valid phase. Pinned by `PackageModulesTests`: a discovery that throws runs no build, a source inventory that does not validate runs no build, a failed build reads no metadata, and two metadata failures are both reported sorted by module.
- [x] Every failure returns no partial result and successful exporter mode emits no build chatter. Both pinned; the no-partial-result claim is mutation-checked (making `load` return a result despite metadata failures fails the suite).
- [x] Focused tests cover each effect boundary, multiple metadata failures, transcript replay/suppression, and comment preservation (both of the original file's comments survive the move, and the moved code carries 28 comment lines). **Region lifetime is not tested and cannot be by a stub:** a `CompactedRegion` cannot be fabricated, so a stubbed load has none to count, and this task's own constraint rules out a region-count check as the proof. The guarantee is structural instead -- `regions` is a field of `Loaded`, so they are reachable for exactly as long as the consumer's result is, and no ordering inside the loader can release them early. A behavioural proof needs the real reader and belongs with the exporter in `.2`.
## Done summary
`ModelLint.PackageModules` is the pipeline both `umpire-lint` and the module impact exporter read:
discover the package's sources, validate them, make them current, read their compiled metadata, and
return one result. `umpire-lint` consumes it today; the exporter is task `.2`.

Two things the split made hard are why it is one module.

**A child build writes to somebody.** The linter replays what Lake wrote to the streams Lake would
have used; the exporter wants silence from a build that worked and everything from one that did not.
Neither can be the loader's decision, so it captures the transcript and returns it, and the two
policies (`replayed`, `quieted`) are functions to an `Emission` rather than functions that print — a
test compares a value instead of capturing a stream, and the printing is one place neither policy has
to be trusted about.

**A failure is not a smaller answer.** Each phase stops the phases after it, and every failure
returns no result: a caller handed part of an inventory would be claiming to have examined modules it
never reached. Metadata failures accumulate and sort by module rather than racing, and a module that
could not be read does not have its imports queued, because anything reachable only through it was
never examined either.

The three places the loader talks to the world are injected, so "what happens when the build fails"
is answered here rather than arranged on a checkout.

### What is unchanged

`make lint-model` at the 163 baseline, the success line still emitted, `umpire-lint-tests
--controlled-violation` still exiting 1 with its diagnostic on stderr and nothing on stdout, the
synthetic suite passing, and both of the original file's comments carried through the move.

### What is not covered, and why

**Region lifetime is not tested.** A `CompactedRegion` cannot be fabricated, so a stubbed load has
none to count, and this task's own constraint rules out a region-count check as the proof — my first
attempt was exactly that check and was replaced. The guarantee is structural instead: `regions` is a
field of `Loaded`, reachable for as long as the consumer's result is, so no ordering inside the
loader can release them early. A behavioural proof needs the real reader and belongs with the
exporter in `.2`.

**The exporter is not a second consumer yet**, so "both consume the same result and policy" is
delivered as a loader and a policy it can consume. `.2` is where that clause finishes.

### Note on the task's execution constraints

The task text says "no staging, commits or pushes". This session's git requirements say to commit and
push to the designated branch. I followed the session's requirements; the override is recorded rather
than resolved silently.

### Review

One round, SHIP with four items, all fixed in `129f8c366`.

The one that mattered was a mistake in the extraction itself: it had **forked** the metadata
traversal rather than extending it, which is what the task's own approach says to do. That left the
worst arrangement available — the copy carrying the regressions was the one `umpire-lint` had stopped
calling, and the copy that ran had none, the two line-identical and free to drift apart unobserved.
The accumulating walk now lives in `Tools.LeanImportGraph.Metadata.load` beside its regressions, and
its missing-metadata test asserts what the walk promises (reported, named against its module,
contributing no record) rather than that loading throws.

Second: only `discover` kept the `try` the pre-extraction code wrapped every phase in, so an `IO`
throw out of the build or the read would escape the loader and `main`, losing the phase prefix and
skipping the declaration linters. Every phase is guarded again.

Two smaller: the three diagnostic prefixes were three literals in three arms, now one function with a
test pinning that each failure names its phase; and the duplicate-source test asserted only that
something was wrong, which `validateSources` would also say about an unclassifiable module.

The review cleared three risks with executed evidence: `IO.Process.output` neither deadlocks nor
truncates (it sets `stdin := .null` itself and drains stdout on a separate task), the traversal cannot
loop or double-report, and dropping regions on the metadata-failure path is not a use-after-free
because nothing calls `CompactedRegion.free`.

Implementer and reviewer are the same session — `codex`, `cursor-agent` and `grok` are not installed
in a cloud session — so this owes a cross-model re-review before fn-46's completion review, the same
as fn-85 `.1`, `.2` and `.3`.
## Evidence
- Commits: 969144c83, 4b3fb3b9a, 62109d8c0, 094c9002d, 129f8c366
- Tests: cd model && lake build, cd model && lake exe umpire-lint-tests, cd model && lake exe umpire-lint-tests --controlled-violation, LEAN_NUM_THREADS=1 make lint-model
- PRs: