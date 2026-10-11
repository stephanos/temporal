---
satisfies: [R5, R7]
---
# fn-140-one-sentence-witness-queries-with.10 Migrate remaining Scala witnesses and commit the source equivalence seal

## Description
**Size:** L
**Touches:** [model/temporal/**/*.scala, model/irgen/Order.scala, model/irgen/test/**, model/irgen/testdata/**, model/testdata/equivalence/batch-1d.map, model/testdata/equivalence/batch-1d.ledger.json, model/testdata/equivalence/dispatch-original/**, model/check/Equivalence.scala, model/check/Gate.scala, model/check/StepLedger.scala, model/check/test/Equivalence.test.scala, model/check/test/StepLedger.test.scala]

Complete the remaining Scala witness migration and source-aware eligible-triple lint after .4/.5. This is the independently completed, committed Scala source seal that releases fn-149 grouping under the owner's one-worker sequence. Deferred .6 retains full regeneration, Go, consumer/replay, full-gate/review/live closure and depends on this seal and consumer tasks .7/.8/.9. Do not wait for those owner-deferred tasks or claim they passed here.

### Approach
- Re-anchor to .4's complete incremental source pin and explicitly classified inherited debt on the integrated fn-156 baseline with actual fn-151/fn-155/fn-159/fn-160/fn-161/fn-162 mapping. Inventory every remaining pinned find and all Property/Scenario readers across all feature, foundation and actor packages; migrate every eligible Model, not only files that kept historical names. Layout now uses Protocol and facets. Preserve verify Queries, shared declarations, functions returning claim bundles/Query lists, monitor-only claims, exact repeated-tail paths and custom-start verify forms. Add no unrequested witness starts surface.
- Implement source-aware lint in the existing typed source/order pass (Order.scala), using collected references and exact witness admissibility. Position the diagnostic on the authored Query. Do not try to distinguish a core/sugar triple from its emitted IR or move the source rule to a Go consumer. Include independent fixtures for eligible private pinned triples, shared Property, shared Scenario, verify, monitor-only and exact input-bearing repeated tails; retain existing order/structure refusal classes and attribution.
- Preserve every Query name, path/input/start, runtime registration, Limits/total/free/live metadata and realization membership. Freeze complete pre/post source/scratch inventories, identity maps and explicit edited-declaration spans. Independently enumerate all applicable model rows for every rewritten Property and assert old/new truth equality; required ends predicates remain, including full state rather than weaker field comparisons. Reuse .4's baseline evidence only for unchanged declarations.
- Preserve strict inherited completion/fatal-failure/pause-resume Go assertion bytes and replay inputs; .9 owns their actual baseline/post Go execution. Never claim source/native results as those assessments. Preserve and independently audit the native strict duplicate-Property failure separately; no blanket Batch 5 waiver applies to source validation. All new migration/admission checks must pass.
- Integrate .5's docs/core-form/layout template and lift the template positively. Seal isolated scratch IR and Scala lift goldens with complete difference classification under R5/R6, never broad position stripping or alpha-normalization. Keep checked-in production IR, Cases and functional/canary mirrors untouched. Preserve baseline and identity/span mapping for deferred .6/.9 consumer comparisons; no native substitute for actual assessments.
- Run framework, Model, lifter and gate Scala suites, syntax and pinned step-table equivalence. Record concrete test coverage and full source-seal handoff. The committed closure releases fn-149.1 source implementation, then fn-149's source seal releases fn-164, then the pre-fault seal releases fn-123; no whole-spec Go closure is needed for that source order.

### Inherited fixture obligations

The fn-140.3 pre-edit fixture campaign recorded obsolete elapsed constructors, missing nested
monitor reason text, a Totals specimen rejected by the strict duplicate-Property rule, and stale
managed goldens selected accidentally by the broad '*expect*' name filter. Task .3 may repair
obsolete constructors and source expectation admission, then run a dedicated nonempty expectation
suite. That focused result does not repair or pass the old Totals/full fixture campaign.

Before this source seal, audit the retained fn140-3 baseline logs and prove all live/total order
assertions still execute. Repair a stale Totals source specimen by preserving its independent
pair/control semantics and the strict duplicate rule, never by weakening admission. Retain stale
managed-golden comparisons for the deferred regeneration/full sweep, with exact tests and results
identified. No excluded assertion earns passing credit.


The expanded fn-140.3 integrated formatter check also records inherited formatting debt in
`model/irgen/testdata/lifts/Hints.scala` (`fn140-3-root-format.log`, exit1). The separate constructor
checkpoint changes only its timerDeadline reference; the long WaitBound lines are unchanged from
that task's baseline. Repair the source fixture formatting in this sweep, preserving every positive
and invalid bound and mapping any refusal-coordinate changes explicitly. The nine-file .3 API
format check passed; no broader formatter success is claimed.

### Quick commands (serialize heavy commands)
```sh
mise exec -- scala-cli compile model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/irgen
mise exec -- scala-cli test model/check
make lint-model-syntax
make umpire-check-equivalence BASE=.flow/tmp/fn160/baseline-ae3f36ee51 MAP=model/testdata/equivalence/batch-1d.map STEPS_ONLY=1
```

No Go commands/edits, production regeneration, full gates, independent review or live runs while the owner restriction remains.

### Inherited source-equivalence and strict-native obligations

Task .4's pre-edit historical --steps-only comparison is red with 34 reported differences:
Activity Product/Protocol binding-row order, four new dispatching tables, and four unmatched
capability rename entries. Its separate zero-difference incremental pre/post comparison may
prove .4 changed no steps; it cannot close this historical debt. Audit the exact frozen
populations and provenance. Reconcile the complete historical-to-current tables using explicit
earlier-spec identity/row mappings and independently justified new compositions, or report an
unresolved source-seal blocker. Never discard ordering or positions broadly, regenerate the
baseline from current source, accept unused map entries, or call this Batch 5 assessment debt.
The complete original-baseline comparison and mandatory pre-fault seal must be satisfied before
admitting a meaning-changing source task.

Task .4 also records a pre-edit ActivitySubjectRegression strict duplicate-Property assertion
failure (blank native Property versus cancelIsRequested at Some()). Keep its independent
assertion strict and audit the actual cause before this seal. If it is a stale specimen, repair
only its source setup while preserving the pair/control semantic assertion and strict admission;
otherwise identify the real implementation gap and refine the repair scope before claiming
closure. This native failure is not one of the protected Go assessment trio and is not waived
as Batch 5. The final seal may report held managed-golden/Go checks separately, but it cannot
report passing native/source admission by excluding or relaxing these assertions.
### Verified dispatch-table provenance (conductor)

The four additional dispatch subjects originate in commit
`fb9ff2a09582b72dc5b703b19ff5b7c514359885` (`feat(model): bounded dispatch progress across the activity and its worker`, fn-150.5), which follows the immutable baseline capture
`ea5264bc23a9d994ca9e1bea33dafd4341684cab`. That commit first introduces
`system/DispatchProgress.scala` with DispatchingActivity, ServingWorker, ServedDispatch and
UnservedDispatch. Commit `fa32b76586f66aa1fbd3b8404568dd0e381e42fe` then splits the already-existing
Protocol declarations into their current one-machine-per-file layout; the file split is not
provenance for new behavior. The original dispatch additions intentionally restrict the activity
and worker catalogs and add served/unserved composition checks. They are not replacement names
for the deleted dispatchEligibility claim container.

This provenance does not waive any baseline difference or prove fn-150 closed: its current Flow
state retains unfinished source/consumer tasks. Before the historical seal, authenticate those
four complete declarations and every native row against their independently committed original
source, compose only the verified earlier-spec mappings, and account for their additions explicitly
alongside the untouched original-baseline population. Do not map unrelated tables together,
recapture the baseline, or claim the later file move introduced equivalent replacement tables.

### Verified rename-map and duplicate-reader findings (conductor)

The immutable baseline's actual capability origins are `temporal.capabilities.Retries` and
`temporal.capabilities.Deadline`; it has no `Retryable`, `Expirable`, `Pollable` or
`Describable` origin. The current batch map wrongly names the post-rename Retryable/Expirable
origins as old names and includes unused Pollable/Describable entries. Correct the former to
the authenticated old names and remove only the unjustified latter entries. Retain the
strict unused-entry check and verify actual origins and every justified mapping independently.
The map file is included in this task's narrow source write surface for this reconciliation.

The current equivalence implementation compares binding and row order exactly and supports only
renames and removals; its new-table diagnostic supplies no accounting syntax. Therefore neither
a global table sort nor a claim that the existing map accepts the four additions can satisfy
this seal. Any required checker extension needs an explicit scoped design and negative controls
before implementation. Preserve ordinary strict comparison, the untouched historical population
and the independently committed dispatch population; do not fabricate a passing map.

The strict Activity duplicate pair is confirmed in Cancellation.scala. Both Properties select
`client.requestCancel`; cancelRequested tests the phase and fact, while cancelIsRequested tests
the fact alone. Existing readers are cancelRequest and Reset's resetCancellation for
the stronger Property, and the live cancelIsRequested Query for the weaker Property. The latter
is named in HeldCancellationByIDRun. A repair must retain the stronger claim, preserve every
Query identity and metadata, record the actual property-reader identity change and prove both
old predicates agree over every applicable native result. Retain the strict distinctness test
and independently detect a seeded duplicate; removing a declaration from its asserted list
without eliminating the actual duplicate is not a repair. This remains fn-162.3's earlier
source obligation and does not close that task's held consumer acceptance.


### Scoped historical-step ledger design

The source seal compares every original-baseline table and explicitly accounts for the four later dispatch additions. The conductor adopts a typed sidecar ledger over the existing strict comparator. Restoring the old interleaved actor order would change the adopted facet assembly; a global table sort would conceal ordering mutations. The ledger records the two exact earlier-spec permutations and independently committed additions while ordinary comparisons retain byte and order equality.

The existing two-token rename/removal grammar stays unchanged. One exact metadata comment, `# step-ledger batch-1d.ledger.json`, locates the versioned sidecar relative to the map. Reject duplicate directives, unsupported versions, missing or escaping paths, duplicate table keys and unknown fields. Readers that ignore comments keep their existing rename semantics. Gate/Equivalence may load the source-step ledger for the steps-only comparison; full projection and interpreter proof remain strict, separate and deferred. This ledger earns no full consumer equivalence credit.

An order entry names the exact baseline and current table keys, the original table digest/count, the exact old/current binding vectors and the explicitly listed bijective action-block permutation after authorized row removals and renames. Check both vectors, unique bindings, one contiguous block per action and block order. Replay only that entry's action blocks. Compare all other headers and state lines in order and every class/input/state/result/alternative/fact/explanation byte inside each block. Missing, duplicate, fragmented, reordered inner rows or changed payloads fail. Include the table/ledger identity in comparison memoization so one permitted permutation cannot leak to a different key. A ledger entry must explain an actual change; unused entries fail.

An addition entry names one of dispatchingActivity, servingWorker, servedDispatch or unservedDispatch and points to the frozen native table population built independently from commit `fb9ff2a09582b72dc5b703b19ff5b7c514359885`. Authenticate the complete original declaration sources and relevant native execution inputs, retaining source commit, source hashes, artifact hashes, table keys/counts and the exact earlier-spec identity mappings. Verify payload hashes against actual bytes rather than trusting index.tsv. Compare all original addition rows against current rows under only those mappings; do not use current-candidate hashes as the addition oracle. Preserve the original historical baseline unchanged. Frozen regression data may live in `model/testdata/equivalence/dispatch-original/`; record its independent build and authentication before accepting it. Unknown, missing, duplicate, unauthenticated or unused additions fail, and every other new table still fails.

Write focused tests before checker implementation. Retain every existing ordinary mutation control. Add positive tests for the exact permitted permutation and authenticated addition, and negative tests for wrong old/current vectors, non-bijections, unused entries, another table sharing a digest, noncontiguous blocks, inner-row order, missing/duplicate pairs, changed results/alternative order/fact order/explanations, missing/extra added tables, changed addition payload/source hash, and unsupported or malformed sidecars. Run the controls with and without the ledger. No check, fixture assertion or baseline is relaxed to obtain a passing seal.

The cancellation repair remains a separately classified fn-162.3 source identity change. Retain cancelRequested, redirect the existing cancelIsRequested Query to it, prove complete applicable-row equality of both original predicates and the retained predicate, and preserve the strict duplicate detector with a seeded duplicate control. Witness R5 allows only the separately inventoried witness changes; this repair does not broaden that allowance or close fn-162.3's consumers.
## Acceptance
- [ ] Complete eligible/retained inventories cover all Model packages, with concrete reader/boundary justifications; every eligible private pinned triple is a witness (R5).
- [ ] Source-aware lint and focused positive/refusal fixtures cover private/shared/verify/monitor/repeated-input-tail forms with exact source ownership and unchanged other refusal classes (R5).
- [ ] Full Query/core/path/expectation/realization inventories, independently enumerated all-row truth equality and explicit source-span/identity maps prove only R5 rewrites and the R6 deletion; no protected expectation is changed (R5).
- [ ] Final author docs and core-form/layout template are integrated and positively lifted, proving R7.
- [ ] Scala compilation and framework/Model/lifter/gate source checks, new migration proofs, syntax and historical-baseline step-table equivalence pass, with complete justified identity/row mappings and no unresolved strict-native admission failure. Held managed-golden/Go checks and protected Batch 5 Go assessment obligations are explicitly separate, never passing credit or a native/source waiver. Scratch artifacts and mappings are frozen for deferred consumer proofs; managed outputs remain untouched.
- [ ] This Scala source seal is committed and its handoff to fn-149 is explicit; .6/.7/.8/.9 retain all full consumer/regeneration/assessment/gate/review/live obligations without claiming them complete.
- [ ] The historical-step ledger passes exact mapped-permutation/addition positives and all independent mutation controls; ordinary unmapped ordering/new-table/unused-map checks remain strict. The complete original baseline and independently authenticated original dispatch population compare with every current table accounted for.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:


