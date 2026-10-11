---
satisfies: [R1, R5, R7]
---
# fn-140-one-sentence-witness-queries-with.10 Migrate remaining Scala witnesses and commit the source equivalence seal

## Description
**Size:** L
**Touches:** [model/temporal/**/*.scala, model/framework/Facet.scala, model/framework/Claims.scala, model/framework/Syntax.scala, model/framework/Witness.test.scala, model/README.md, model/irgen/Syntax.scala, model/irgen/Conventions.scala, model/irgen/Order.scala, model/irgen/Lifting.scala, model/irgen/test/**, model/irgen/testdata/**, model/testdata/equivalence/batch-1d.map, model/testdata/equivalence/batch-1d.ledger.json, model/testdata/equivalence/dispatch-original/**, model/check/Equivalence.scala, model/check/Gate.scala, model/check/StepLedger.scala, model/check/test/Equivalence.test.scala, model/check/test/StepLedger.test.scala]

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

### Scoped generic status-table source repair

The gate baseline has a strict CapabilityVocabularySuite failure because Facet.scala's comment names a Temporal kind and Conventions.scala exempts an entire binding by the concrete Temporal capability name. Preserve the strict vocabulary check and the recorded fn-160 R9 status-table layering exception without concealing its name or waiving a source check. This is separately classified earlier-spec architecture debt, not a witness R5 allowance.

Replace the concrete-kind exception with a narrow typed constructor seam: a binding inside a capabilities section whose RHS is exactly the companion apply of a CapabilityOf case class, with exactly one case field typed framework.realize.StatusTable and a stable StatusTable reference as that argument. Admit only this precisely recognized binding, including its status-type arguments. Do not exempt arbitrary status-table declarations, non-capabilities, extra constructor fields, blocks or calls hiding unrelated API/layer references. Existing Lifecycle.scala's statusTable binding remains byte-for-byte unchanged. Facet.scala changes only its generic adjective example, not runtime code.

Write focused fixtures/tests first: a generic independently named positive proving no Temporal-kind identity is required; wrong section, extra field, hidden block/call and unrelated API/layer-reference negatives with exact authored R4 diagnostics. Preserve every existing convention refusal and native capability inventory. Run CapabilityVocabularySuite and focused conventions tests plus the complete source suites; compare production core/native inventories to prove no declaration, predicate, status mapping or Query meaning changed. Keep the full-consumer boundary unchanged.

### Scoped conditional-fact witness foundation and heartbeat migration

The conductor adopts the narrow conditional-fact recommendation under the owner's explicit instruction to follow its recommendations when in doubt. This resolves the previously pending API-versus-exception question without narrowing R5. Do not implement an arbitrary Step-predicate ending or retain heartbeat as an exception.

After mandatory primary fact selection `.records(primaryFact)` or `.excludes(primaryFact)`, a completed Witness admits `.recordsWhen(condition: S => Boolean, fact)`; a composition admits `.recordsWhen(condition, _.member -> MemberMachine, fact)` with the selected member's exact fact type and identity. Both are immutable modifiers, usable with the existing ends/limits/live/explore ordering. Conditions inspect only the after state. Multiple conditional clauses accumulate in written order; a condition that is false imposes no fact requirement. Preserve the builder's mandatory primary fact-selection admission and every existing compiler/refusal rule.

Native core semantics are `primaryRecorded(after) && (!condition1(after.state) || selectedFact1Recorded(after)) && ... && ending(after.state)`, with the ending omitted when absent. Use the established typed primary/member-fact machinery rather than a second suffix-based selector implementation. Preserve Query names, selected tail, owner, path, start, limits, total/free/live/exploration metadata and runtime registration. No schema, IR declaration kind, Go consumer, live generation or starts surface changes.

Extend the established WitnessSource/sugar hook in irgen/Syntax.scala to collect conditional clauses in their authored order and lower them to the ordinary Property Boolean expression and state-function calls. Preserve primary fact/path duplicate admission, composed selected-member identity, and exact authored source positions. One primary .records or .excludes continues to be required; .recordsWhen never finishes an unfinished builder. Every retained modifier must copy accumulated clauses.

Migrate heartbeatTimeoutExhausts with the original ordered path and metadata, primary heartbeatTimedOut, recordsWhen(exhausted, statusTimedOut(heartbeat)), and the state ending exhausted(state) implies phase == timedOut. The repeated exhausted condition preserves the complete original assertion; do not remove the state ending or change the original synthetic tests. Delete only its now-private Property/Scenario after its full reader inventory proves eligibility.

Tests precede implementation: native truth comparison with the exact original predicate for every applicable row; independently pinned synthetic same-state/same-primary-fact steps with correct/wrong/missing terminal facts; false-condition rows accepting absent or other terminal facts; primary-fact-absent refusal; accumulated-clause order and modifier preservation; typed native and composed-member positives; foreign/native/member facts, non-state condition, ambiguous/mismatched member and unfinished-builder compiler/lift refusals. Lift sugar and independently authored core forms and compare full Boolean IR/functions with only explicit parameter/identity/source mappings, never broad normalization. Existing heartbeat family, complete capability classifications and duplicate detector assertions remain unchanged.

Update README and the new public sugar's core-form documentation with the conditional clause, keeping the triple guidance for genuine shared/invariant/repeated-tail forms. The main checkout's user SEMANTICS draft remains untouched. After the foundation and migration, rerun current-source admission, complete all Query/core/path/expectation/realization/span inventories and native proof, positively lift layout, and repeat final source/checker/format/syntax/historical steps-only checks. Any earlier pre-edit-lifter inventory capture is scratch evidence only; final current-source lint/lift must pass and yield that same full inventory before closure.

This scope remains inside the original worker timebox; no extra budget, review, Go, managed regeneration or live-run authority is granted.

The native duplicate test's newly exposed None-selector pair is the two pre-existing global dispatch transition invariants, not witness selectors (native Witness stores the tail class). Keep actual selectors, the complete native row population and the strict pair routine. Augment it with independently pinned off-native transitions distinguishing those actual invariants; never assign invented selectors or exclude the global group to get a pass.

### Provisional historical checker checkpoint (2026-10-11)

The direct canonical Scala steps-only comparison in `fn140-10-final-equivalence-historical1.log` reaches its terminal equivalent-to-baseline message; the worker reports exit0. It compares all59 immutable baseline keys with46 current keys under the strict rename/removal map, two explicit binding-block permutations and four authenticated original dispatch additions. The conductor independently decompressed and hashed all45 unique current payloads, verified every count,46 unique keys and2,189,510 table lines. The current index SHA256 is `3a8f32405bfc69ce0f940097398575d0c3313facdd43fb88dd2bac5d8b8f3bcd`, identical to the prior complete incremental native population. Retained root audit: `.flow/tmp/scala-dsl/fn140-10-root-current-payload-audit.json`.

The worker's `fn140-10-check-tests2.log` reports a nonempty complete passing Scala checker campaign, including original Equivalence controls, actual source/compiled-artifact/receipt authentication and the vocabulary check. This is an intermediate checkpoint, not the final source seal. Static audit then found that current index keys can be silently collapsed by `toMap`, and the one-row-per-action mutation fixture does not independently establish inner pair ordering. Add strict duplicate original/current key refusals, multi-row same-action mutation controls, exact malformed/duplicate sidecar controls and the full-mode no-ledger regression; rerun the updated checker and final formatted-source historical comparison before any completion claim.

Typed source admission also needs unrelated `records` helper controls and a qualifier-call control for supposedly stable StatusTable references. Resolve actual framework fact methods and stable typed reference chains, rather than method names or rendered-source substring tests. Preserve every prior retained-boundary fixture. Heartbeat R5, complete Query/core/span inventories and the full source fixture obligations above remain open. The Make dry-run requires owner-held linked-API generation; the direct Scala invocation grants no canonical Make, generated artifact, Go, consumer, review or live-run credit.
### Scoped absent-fact witness foundation and exact reset guards

The conductor adopts the absent-fact decision in the parent spec under the owner's recommendation mandate. R5 remains every eligible private pinned find; negative-only predicates are not retained exceptions. R1's mandatory positive finisher is explicitly refined to mandatory typed fact selection.

Add `.excludes(fact)` on WitnessBuilder to finish a negative-primary witness and on Witness to append an immutable unconditional absent-fact guard. Both also support the established composition member selector `.excludes(_.member -> MemberMachine, fact)`, retaining that member's fact type and exact identity with the same resolver as records/recordsWhen. Native core is NOT(facts.contains(fact)); all later excludes/recordsWhen clauses are accumulated and short-circuited in authored order before ends, and every existing modifier copies them. Unfinished builders, foreign native/member fact types and invalid selectors remain refused. No arbitrary Step predicate, new start surface, outcome/fact additions or consumer/schema changes are permitted.

Extend WitnessSource in irgen/Syntax.scala to retain primary polarity and the ordered additional clauses. Emit ordinary NOT(CONTAINS) for each excludes and preserve helper/body/parameter/source-span parity. Only positive-primary witnesses enter the existing recorded-fact capability-duplicate registration; do not weaken or skip any existing positive duplicate controls. Add independent positive and negative-primary cases with the same path/fact to prove the distinction. No irgen/Claims.scala write is authorized by this refinement.

Migrate resetOutranksPause's original path to a witness excluding statusPaused with the original resetRequested state ending and Limits/total/live/realization metadata. Delete pauseLeavesResetPending and resetThenPaused only after the complete reader inventory confirms privacy. Restore resetFatality's absent statusFailed guard and resetTimeout's absent statusTimedOut(startToClose) guard through completed .excludes clauses. Preserve each full state ending. Include this newly eligible query in the complete frozen pre/post migration inventory and native enumeration; no hard-coded17/18 count or negative-only retention allowance may conceal it.

Tests precede implementation: native/compiler controls for negative-primary and positive-primary-with-excludes, missing versus present forbidden facts on independently pinned equal-state steps, correct versus wrong composed member recording the same fact, accumulated clause order/short-circuit and every modifier; lifter core equality for ordinary negative predicates with exact source metadata, retained positive duplicate refusal and negative-primary nonduplicate; typed source lint refusal for a private negative-only triple and positive retained shared/invariant/repeated-tail cases. Run the original reset and heartbeat synthetic/native assertions unchanged and add exact negative-guard controls. Final current-source admission and complete Query/core/helper/body/parameter/span mutation proofs must pass before the seal; an earlier native-only redundancy observation is not guard-preservation proof.

This refinement uses the already scoped framework/lifter/test/docs files and retains the original120-minute worker cap, size L and all held consumer/generation restrictions. The user-owned main SEMANTICS draft remains untouched.

### Verified intermediate native checkpoint (2026-10-11)

The final focused native `framework.WitnessTest` campaign in
`fn140-10-excludes-native-final2.log` executes all six tests successfully, including
independent single-member absent/present controls, live/exploration preservation
and ordered mixed-clause short-circuit counters. The complete original/current
native captures in `fn140-10-all-native-pre20.tsv` and
`fn140-10-all-native-post20.tsv` are byte-identical (SHA256
`0574109cbab5cef0aa95d89fc8c8d100927e0630c717314769b3e0f326091fcb`).
The conductor independently verified the exact 20-Query population, all 52,244
applicable rows, 27,540 enabled results and nonempty enabled coverage per Query.
This includes all 19 witness migrations, including resetOutranksPause, and the
separately classified cancellation redirect. Root audit:
`.flow/tmp/scala-dsl/fn140-10-root-native20-audit.json`.

This is native-row equality evidence, not the completed source seal or a substitute
for off-native forbidden-fact controls. Full original/rewritten Boolean core,
generated helper/body/parameter/source-span parity and its mutation controls remain
unverified. The latest complete lifter campaign has eight source failures; three have subsequent
focused passing repairs and five full-Fixtures contracts remain open, separately
from nine owner-held managed snapshot comparisons; neither
source failures nor excluded assertions earn passing credit. Fn-149 remains gated
on this task's verified committed closure, and the original worker cap is unchanged.

### Integrated partial and bounded continuation (2026-10-11)

Checkpoint `6fc3878be702154a28d1c5d07e78587f2480d298` is integrated locally.
The first implementation attempt returned terminal `BLOCKED:SCOPE_EXCEEDED` at
its original timebox, with this task still `in_progress`, a clean workspace and
no attributable live command. This is failure strike 1, classified as code present
but not finalized. It is not a completed source seal or a new authority blocker.

stage: worker-attempt-1 - failed(timebox: committed partial; exact core/span and five fixture contracts unresolved)

The complete irgen campaign3 returned exit1: 156 passing tests, eight source
failures, nine owner-held managed comparisons, and one pre-existing skipped test.
Fresh focused DefaultEnds (5), PhaseCapabilities (3) and StatusTableSeam (2)
campaigns subsequently passed. Five full-Fixtures obligations remain: the exact
DerivedTwins per-machine derived-ID refusal, the independent custom-state design
positive, the legacy layout template realization's count/ID/machine/source, and
fresh verification of Kettle/Reservoir coordinate repairs. Do not change expected
diagnostic stages or remove positive realization coverage to make these pass.

Other fresh focused observations include WitnessTest (6), production Reset guards
(9), checker tests (97, including eight strict StepLedger controls), actual
`make lint-model-syntax`, and current-source lifting of all four production roots.
These observations are not full acceptance. The complete main/helper/body/type/
parameter/polarity/order parity proof, authenticated heartbeat `timesOut` export
retirement and Construct-to-typed-literal timeout argument mapping, 26 edited spans
and LostStartAnswer's start-position map, independent mutations, rejection
reason/coordinate/no-IR controls, final format/idempotence and source/artifact/
classpath reauthentication still require completion.

Use one fresh-context re-anchoring continuation in the existing `w-1d` workspace,
starting from the integrated local tip. Preserve the implemented decisions and
original full task acceptance; independently re-verify evidence on the actual
artifact. The continuation has its own explicit 120-minute budget, not an extension
of attempt 1. Persist the exact start/deadline at dispatch. After a second non-done
return, stop retrying this task and escalate; no third dispatch is permitted.

Original handover:
`.worktrees/w-1d/.flow/tmp/scala-dsl/fn140-10-summary.md` and
`fn140-10-evidence.json`. Continuation handover uses separate
`fn140-10-retry-summary.md` and `fn140-10-retry-evidence.json` files.
Root native20, query-metadata and campaign3 failure audits remain under
`.flow/tmp/scala-dsl/fn140-10-root-*.json`.
No Go, managed regeneration, full gate, independent review, live run, publication
or edits to the user-owned main SEMANTICS draft are authorized. Fn-149 remains
gated on verified committed closure, not this checkpoint.

The continuation's first complete Fixtures reproduction returned exit1 with
100 passing tests, three source failures, nine held managed comparisons and the
same pre-existing skip. Kettle and Reservoir coordinate controls now execute and
pass; remaining source failures are DerivedTwins, the custom-state FacetRules
positive and layout realization collection. Root independently inspected the
terminal log and exact failure population; retained audit:
`.flow/tmp/scala-dsl/fn140-10-root-retry-fixtures-red-audit.json`. This does not
pass the suite or the source seal.

Root also independently identified 528 changed nested source positions in
same-name exported functions: 382 map to exact unchanged source lines and 146
require formatted/edited declaration mapping beyond the 26 Query spans. All
same-name function differences other than those positions are the independently
inventoried Nexus standalone witness parameter/body `step` to `after` rename.
Complete validator coverage must include all these nested positions rather than
strip or ignore them. Audit: `fn140-10-root-same-name-function-positions-audit.json`.

### DerivedTwins collection-stage reconciliation

The original shared-def fixture tested the whole-run `derivedIdTwins` check across
two IR files of one package. Fn-160 now collects one IR file per feature, placing
both named Queries in one context, where the existing strict Query registration
correctly refuses the duplicate first. This is an explicitly classified fn-160
collection-stage change, not permission to weaken either identity rule.

Retain an automatic-root test requiring the exact duplicate-registration refusal
at the authored shared-def position and no IR output. In addition, lift each
actual shared-def instantiation separately through its explicit Query-list root,
read both authentic source-derived IR Models, and run the unchanged whole-run
`derivedIdTwins` check over them. Require the original exact per-machine ID
collision reason, owning machines, source coordinate and one-error cardinality.
The two models must independently contain the intended distinct machines and the
same declared Query ID; no hand-constructed candidate Models or manufactured
error message substitutes for this source integration control. Preserve the
existing independent DerivedIds mutation/positive tests. No Claims.scala,
Lift.scala, declaration/discovery implementation or public API change is needed
or authorized by this repair. The automatic-root refusal alone does not pass the
original cross-file contract.

### Independently verified frozen core candidate (2026-10-11)

The conductor re-executed the strict complete validator and all 26 independent
core/metadata/order/span mutations successfully (terminal exit0). This accounts
for all 338 Queries, reconstructs each of the 19 witness main bodies and 14
ending helpers from original predicates, and separately classifies the
cancellation redirect and heartbeat helper retirement. All 134 frozen original
non-test framework/Model source files match committed base
`45798d12bbfa726b32e7f7472f02412619d3490e` byte for byte.

All 655 changed positions are classified: 442 exact unchanged-line mappings,
150 explicit token-identical formatted-declaration mappings and 63 explicit R5
edited-declaration mappings. The conductor reviewed the old/new source lines
and independently authenticated the 213 frozen path-specific non-exact records,
including each source span's bounds and SHA256. Frozen map SHA256:
`edb174349e757dc4f48f68a19a70d4f41ca449e37765e4f368ce606e1966ba85`;
reviewed validator SHA256:
`83180612b288d4c0669cfd9952d7872131bba663cfaa92aaa9b8f391ac799a89`.
Root audit: `.flow/tmp/scala-dsl/fn140-10-root-full-core-proof-audit.log`.

This earns frozen candidate core proof, not final current-source closure. The
typed full inbound-reader inventory, final re-lifts after fixture/collection
repairs, artifact/classpath authentication, source fixture checks, formatter
idempotence, syntax and final historical comparison remain required. Do not
silently replace the reviewed position map if final formatting changes it.

The subsequent complete Fixtures campaign2 is still exit1: 101 passing tests,
two source failures, nine held comparisons and one pre-existing skip. Its
unchanged custom-state/facet-owner positive and canonical negatives execute
and pass. DerivedTwins still fails its wrapper diagnostic assertion (now
corrected in source, not yet passing); its new authentic cross-file control
still requires execution. Layout used the rejected Temporal-kit specimen in
that run, so this run grants no original generic-layout proof. Log SHA256:
`5724527bdf7bd71307dd1fc563574b7915f2c8b558c53efffaecb34f0f986141`.

### Generic realization collection repair

Fresh original-template evidence localizes the layout failure to automatic root
collection, not the generic realization declaration. The same original compiled
`layout.jar` emits the complete core realization through the explicit typed root
`fixture.features.lamp.system.LampRealization$.system`, but automatic collection
omits it. `Lifting.liftRoot` already supports `framework.realize.Realization`
values; `irFiles` omits these values both from its roots and from the `asked`
predicate. Fn-160.2 requires every declared realization, including a feature
whose only claim is that realization. Evidence: the continuation's
`fn140-10-retry-layout-explicit2.log` and `fn140-10-retry-layout-explicit.json`
(artifact SHA256 `a2e4722d22d843f8c909eb5c06fdc05091e1e97572dfbe7145f16a0f767a8a18`).

Authorize only the corresponding typed core-value predicate in
`model/irgen/Lifting.scala`'s root collection and `asked` filtering. Preserve the
original generic layout payload, with no Temporal realization kit, Entity,
roles, observations, cleanup or changed correlation bounds. Before the fix,
add source integration assertions comparing the explicit and automatic complete
realization payloads, including exact ID, owner and positions. Include an
independent realization-only foundation or actor package so removing `asked`
support fails, plus an unrelated value negative control so collection does not
become an untyped value scan. Verify the original template and full source
fixture contracts after the fix; no removal of assertions earns credit.

This is a separately classified fn-160 collection repair needed for R7's positive
layout proof, not a witness semantics rewrite. Re-lift all four production roots
and confirm the frozen complete inventory is unchanged. Native
`framework.IrFile.discover` also appears to omit wrapper-held generic realization
getters; that distinct native boundary remains an explicit fn-160.2 source
obligation and is not authorized or claimed fixed here. This refinement adds
only Lifting.scala to implementation Touches and grants no extra worker budget,
third retry, managed regeneration, Go, review, live or publication authority.


### Independently verified source-fixture checkpoint (2026-10-11)

The continuation's complete `umpire.irgen.*` campaign `fn140-10-retry-irgen-final2.log`
is terminal exit1: 166 tests pass, all nine failures are the explicitly held
managed comparisons, and the pre-existing case-variant-header skip remains.
There are no source test failures in this run. The conductor independently
recounted the complete log and verified its exact failure population; SHA256
`8408bf12dcb1011c4b1da12b8637f9275a8ebbe2575d82fae667161b773a5f0f`.
Audit: `.flow/tmp/scala-dsl/fn140-10-root-retry-irgen-final2-audit.json`.

The authentic automatic and explicit DerivedTwins controls, custom-state facet
positive/canonical negatives, original generic layout complete realization
payload, independent realization-only package and unrelated-value negative
execute and pass. Original layout source SHA256 remains
`97a2cc2f13971de65f3d8c28ce3f3ff1b42caab88e902e4c83455e2065387171`.
The narrowly authorized typed realization roots/asked repair is exercised;
native reflection discovery is still the distinct fn-160.2 obligation.

This does not pass the full suite, regenerate managed files or prove the
combined original rejected-root population. Its independent reason/coordinate/
no-IR coverage remains required rather than waived by stale `rejects.txt`.
Final current re-lifts, source/artifact authentication, formatter, syntax and
historical equivalence remain open.

The typed-reader capture must resolve exact compiler Symbols, not rendered
full names or a composite display key. Initial names collide for real locals
and overloads; even the first composite signature/owner/span identity attempt
failed its strict uniqueness check. No colliding declarations may be dropped
or merged. The full reader inventory must independently validate all private
eligibility decisions and retained shared/bundle/capability/verify/free/monitor/
repeated-tail boundaries, including non-Query readers, rather than merely
attach typed rows to the earlier IR-based labels. This source seal and fn-149
admission remain unproven; the original continuation deadline and two-strike
limit are unchanged.


### Reproduced bare-statement reader omission (2026-10-11)

The conductor's independent source-anchored reader regression is terminal exit1:
both original and current typed graphs omit all three authored `overriding`
references in `activity/standalone/protocol/Retry.scala`, at lines 124, 130 and
135, to `resetFailureReturnsToWaiting`, `resetFailureEndsFailed` and
`resetFailurePauses`. These are ordinary non-inline calls whose arguments name
Property factories. The collector visits ValDef/DefDef right-hand sides, not
bare ClassDef body statements. Exact compiler identities and zero unresolved
targets do not prove that every authored reader was collected.
Regression: `.flow/tmp/scala-dsl/fn140-10-root-bare-reader-audit.py`;
red evidence: `fn140-10-root-bare-reader-red.log`.

Earlier private-boundary results are therefore declaration-RHS graph evidence,
not full source-reader completeness. Add source-identified bare statement reader
nodes without treating a whole ClassDef body as an extra reader of its private
members. Recapture both complete graphs, independently verify known overriding,
except and bound calls, and repeat all private and retained decisions. Generated
aliases, case-class fields and native/reflected capability laws require concrete
typed boundary explanations; a generic outside-Query label is insufficient.

Order.scala's eligibleWitnesses pass uses the same RHS-only traversal. Prove
whether a bare statement naming an otherwise private triple causes a false R5
refusal using an independent retained-reader fixture before changing that pass;
if reproduced, repair reader coverage within its existing scoped source lint.
Preserve all original private refusal and shared/verify/monitor/repeated-tail
controls. This clarifies existing all-reader acceptance; it grants no additional
worker time, third retry, unrelated implementation or deferred-consumer authority.
Fn-149 remains gated on the verified committed complete seal.


### Independently verified expanded-reader and final-core candidate (2026-10-11)

The reader omission also affects ClassDef parent-constructor arguments. The
independent `parentReader` fixture's intended red run records the erroneous
sixth R5 refusal at line 130; its first unused-parameter compilation failure is
not the reproduction. Final focused EligibleWitness fixtures pass both tests
after visiting only bare class-body Terms and Term parents. Five private
refusals still execute, while bare-statement and parent readers retain their
Queries in the 17-Query positive. Generated accessor filtering is unchanged.

Fresh graphs contain all 5,261/5,199 original/current declaration nodes plus
225 bare statements each and 825/820 parent terms, totaling 6,311/6,244 unique
nodes and 25,829/25,563 references. Exact compiler identities retain all
colliding locals, signatures and compiler clones. All known three reset
overriding calls are now captured on both sides, and no named Property/Scenario
reference has an unresolved target. Graph SHA256:
original `f827b8b705a1c64e773834027cffbc70410bd5d9581386143ba36a7f86252692`;
current `ca33925a6f42009394ebf30bb75eec873e8754b058060bfe1910c0954d922b23`.

The conductor independently repeated all 19 migrated reader proofs against
this expanded graph: 37 direct private boundaries and the exact Fenced factory
to private alias to heldStaleDeliveryQuery chain. The public factory stays
byte-identical. All 154 original claim nodes outside direct Query inputs have
concrete typed boundaries: 23 callable capability/feature Property factories
(including the recursive Idempotent forwarder), 111 claim-bundle parameter or
field compiler clones, 19 factory-local Properties returned in claim bundles,
and one unqueried ordinary authored Property, NexusProtocol.asyncStarts.
There is no unmatched Scenario. This is not a monitor reclassification or
permission to delete unused declarations. The complete validator re-execution
binds all 2,028 before/after Query/Property/Scenario positions successfully.
Root audits: `fn140-10-root-all-readers-structure-audit.json`,
`fn140-10-root-all-private-readers-audit.json`, and
`fn140-10-root-retained-claim-boundaries-audit.json` under `.flow/tmp/scala-dsl/`.

All four fresh final source re-lifts pass complete core verification, preserving
338 Queries, the 19 witness rewrites, 14 ending helpers and all 655 classified
positions. The conductor independently authenticates all 134 frozen original
sources to committed base 45798d12bbfa726b32e7f7472f02412619d3490e and rejects
all 26 independent core/metadata/order/span mutations. The immutable 213-record
reviewed map remains `edb174349e757dc4f48f68a19a70d4f41ca449e37765e4f368ce606e1966ba85`.

The final formatter edit changes only the exact resetOutranksPause method chain
from one line to three, with the entire Reset.scala file otherwise unchanged.
Seven explicitly reviewed replacement records cover that declaration's span
and the two-line shift of keepPausedReset/resetRepeated; all original positions
and spans stay frozen. The conductor derives every changed field independently
from the authored source rather than accepting an arbitrary replacement map.
Separate delta SHA256: `01c4749645f1964227d89b4cdb209d67155afda93b41bad1c8368db654c9bdf1`.
Root verification: `fn140-10-root-final-core-proof-audit.log`.

This is verified candidate evidence, not integrated task closure. Formatter
changes also move strict witness-fixture expression coordinates; focused repairs
must retain complete function/body/parameter/clause/span assertions and the full
source campaign must execute again. Final compile/native suites, complete
226-row refusal/no-IR contract, format/idempotence/syntax, historical equivalence
and artifact/classpath authentication still require final acceptance. The nine
held managed comparisons, consumer obligations and fn-149 gate remain unchanged.
The original continuation deadline and two-strike limit still apply.


### Independently verified final source campaign (2026-10-11)

The final complete irgen campaign returns exit1 with 166 passing tests, ten
stale managed comparisons, one pre-existing case-variant-header skip and no
source test failures. All thirteen WitnessFixtures and both EligibleWitness
fixtures execute. The complete authored refusal contract also executes and
passes. It authenticates the original 226-row refusal table's SHA256, preserves
its complete ordered reason population with three explicit vocabulary
adaptations and source-anchored formatter coordinates, and requires a nonzero
lift exit with no IR. The old managed rejects.txt comparison remains failing.

The ten comparisons include the previous nine and newly stale rules.json.
The conductor authenticates its original managed bytes and the complete
Rules.scala source against committed base 45798d12bbfa726b32e7f7472f02412619d3490e.
The only source edit joins Dimming's two-line object/extends header. Exactly
94 explicit source-line shifts derive from that edit, and applying those
shifts makes the complete original rules IR equal to the actual scratch IR.
No other field, list order or metadata changes. This separately classified
formatter drift earns no managed regeneration or golden equality credit.

Audit: .flow/tmp/scala-dsl/fn140-10-root-retry-irgen-final3-audit.json.
Terminal campaign SHA256:
5096e0a4bf0c0991e0e962f2a7dd2068303107afbbd0500c2eb9547e168102b1.
The conductor also re-executed all five independent typed-reader mutations;
extra standalone reader, missing sole Query reader, missing overriding reader,
wrong statement kind and unsupported compiler-clone classification are refused.

Final framework/Model, checker, syntax, formatter idempotence, native inventory,
historical equivalence and last source/artifact authentication remain separate
acceptance gates. The full irgen campaign remains red on held comparisons.
Fn-149 stays gated, and the original continuation deadline and two-strike limit
are unchanged.

## Acceptance
- [ ] Complete eligible/retained inventories cover all Model packages, with concrete reader/boundary justifications; every eligible private pinned triple is a witness (R5).
- [ ] Source-aware lint and focused positive/refusal fixtures cover private/shared/verify/monitor/repeated-input-tail forms with exact source ownership and unchanged other refusal classes (R5).
- [ ] Full Query/core/path/expectation/realization inventories, independently enumerated all-row truth equality and explicit source-span/identity maps prove only R5 rewrites and the R6 deletion; no protected expectation is changed (R5).
- [ ] Final author docs and core-form/layout template are integrated and positively lifted, proving R7.
- [ ] The original generic layout realization is collected automatically with its complete explicit-root payload and exact identity/source metadata; a realization-only package and unrelated-value control prove the typed roots and `asked` repair. Production inventories remain unchanged, and native reflection discovery is separately retained under fn-160.2 rather than claimed fixed (R7; separately classified fn-160 R2/R10).
- [ ] Scala compilation and framework/Model/lifter/gate source checks, new migration proofs, syntax and historical-baseline step-table equivalence pass, with complete justified identity/row mappings and no unresolved strict-native admission failure. Held managed-golden/Go checks and protected Batch 5 Go assessment obligations are explicitly separate, never passing credit or a native/source waiver. Scratch artifacts and mappings are frozen for deferred consumer proofs; managed outputs remain untouched.
- [ ] This Scala source seal is committed and its handoff to fn-149 is explicit; .6/.7/.8/.9 retain all full consumer/regeneration/assessment/gate/review/live obligations without claiming them complete.
- [ ] The historical-step ledger passes exact mapped-permutation/addition positives and all independent mutation controls; ordinary unmapped ordering/new-table/unused-map checks remain strict. The complete original baseline and independently authenticated original dispatch population compare with every current table accounted for.
- [ ] The strict framework/lifter capability-vocabulary check passes through a generic typed single-field status-table binding seam, with positive and exact-coordinate negative controls; Lifecycle's status table and all production declaration/native semantics remain unchanged. This is an explicitly separate fn-160 R9 source repair, not witness or consumer equivalence credit.
- [ ] Typed conditional-fact witness clauses preserve the exact original heartbeat assertion on independently pinned synthetic and all applicable native rows; mandatory primary fact, selected-member identity, immutable modifiers, compiler/lift refusals and complete core/span parity are verified before current source admission and the R5 seal (R1, R5).

- [ ] Typed absent-fact selection and modifiers preserve resetOutranksPause and the original fatal/timeout negative guards, with native/compiler/member-identity/lift/duplicate/source-lint controls and complete inventory/core/span parity (R1, R5).

## Done summary
# Task 10 Scala source seal

The original full source acceptance is satisfied and committed at b5374fbb0b. All 19 eligible private pinned witness conversions, the separately classified weak-cancellation deletion/redirect, complete source/core/reader contracts, original fixture contracts and historical native equivalence are verified. This seal hands frozen source and scratch artifacts to fn-149; it does not complete managed consumer regeneration, Go assessment, full gates, independent review, live runs, fn-150 whole-spec closure or the broader MILESTONES goal.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none; explicit owner/conductor override)

### Continuation and scope

Workspace: /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/w-1d. Range base remains 45798d12bbfa726b32e7f7472f02412619d3490e; the original checkpoint 6fc3878be702154a28d1c5d07e78587f2480d298 was continued, not restarted. The existing actor claim was retained. Attempt 2 is the only bounded retry: START 2026-10-11 10:46:51 UTC, DEADLINE 2026-10-11 12:46:51 UTC. No third dispatch or extension was taken.

Baseline is inherited red, not a green handoff. The original baseline logs and first-attempt partial handovers remain diagnostic history; capture-only receipts and failed/intermediate commands do not earn final passing credit. Normal tracking merges preserve authoritative conductor refinements. No force push, push/PR, target rebase/history rewrite, Go command/edit, managed production IR/Cases/golden/mirror regeneration, full repository gate, independent review or live run occurred. Gate classify returns FULL because of MILESTONES.md; the owner-held full/consumer gates were not run and no green full-gate receipt was fabricated.

The protected played_test.go SHA is c63deec916b93b86b679785a5791ecbbe5b436f9bb21b006570882bc583bdc77. Main user-owned model/SEMANTICS.md SHA is 89e1fbcb204ae44fb2cbfe4129f3a89f826af00e666707bf8d639a0dd2774841. Both remain unchanged.

### Complete source proof

The strict final inventory compares all 338 Query/Property/Scenario/core/path/input/expectation/limits/live/realization contracts. It independently derives all 19 rewritten main bodies and 14 ending helpers, full helper insertion/removal/type/parameter/member/polarity/order contracts, unchanged declarations, all 655 exact nested-position mappings, the LostStartAnswer constructor/argument mapping, heartbeat's authenticated timesOut retirement/substitution, and Reset's complete negative guards. Cancellation remains separately classified fn162.3. Twenty-six independent core/helper/guard/type/member/order/metadata/position mutants are refused.

The immutable 213-position map SHA edb174349e757dc4f48f68a19a70d4f41ca449e37765e4f368ce606e1966ba85 remains unchanged. The final Reset formatting overlay is only an exact whole-file single-chain replacement with seven explicitly bounded position changes; delta SHA 01c4749645f1964227d89b4cdb209d67155afda93b41bad1c8368db654c9bdf1. No position stripping or candidate-as-expected normalization is used.

Full typed reader capture uses exact compiler Symbol/AST declaration identities, preserving collisions/overloads and parameter clones. It covers declaration RHS, bare class-body Terms and parent-constructor Terms without falsely making whole classes readers of private members. Original/current graphs contain 6311/6244 nodes and 25829/25563 references; all 2028 Query/Property/Scenario anchors bind uniquely. Every migrated private chain is independently restricted to its designated Query, including the exact Fenced factory-to-alias chain. All retained boundaries are justified, including 23 callable Property factories, 111 claim-bundle parameter/field clones, 19 factory-local Properties and one ordinary unqueried authored Property; no unmatched Scenario remains. Non-claim class/module/synthetic-anonfun references are explicitly outside this semantic declaration graph. Five independent reader omission/extra-reader/classification mutants are refused.

Source-lint red-to-green tests establish the five intended private refusals while retaining 17 positive Queries, including independently proven bare-statement and constructor-parent readers. Original accessor admission is preserved. The generic core Realization discovery fix is precisely the authorized typed roots/asked predicate: original explicit and automatic layout payloads are fully equal, realization-only foundation and unrelated-val controls pass, and four production root inventories remain exact. Generic native getter discovery remains separate fn160.2 debt.

DerivedTwins preserves both the automatic duplicate Query refusal/no-IR contract and the original per-machine derived-ID collision through two authentic isolated source instantiations. The independent custom-state FacetRules positive and crossed canonical State refusal pass. Existing DefaultEnds, PhaseCapabilities, StatusTableSeam, Patterns independent pair controls and Totals controls pass. All five conditional/absent/member/guarded sugar/core twins compare exact complete main/helpers/type/parameter/source spans.

The complete ordered 226-row original authored refusal population passes exact reason/coordinate/no-IR validation. Original rejects.txt SHA efdd6380f6266a955434f28a2c486065829f077e980b1136939bc1994112bf90 is pinned. Three vocabulary adaptations are limited to named original rows, and exact authored source anchors account for fixture formatting coordinates. The Hopeless marker's because-only source repair restores the original intended refusal, not a changed expected outcome.

### Final campaigns and held observations

- Full model source suite: terminal rc1, 178 pass, exactly two held managed comparisons (IrFilesTest exports and MachineTablePins), zero other failures. Witness6, Reset9 and all strict native laws pass. Log: fn140-10-retry-model-final.log.
- Full irgen source suite: terminal rc1, 166 pass, ten held managed comparisons, one preexisting case-variant-header skip, zero source-control failures. Log: fn140-10-retry-irgen-final3.log. The held nine are realizations.json, taskqueue.json, retryCapabilities.json, retrySelection.json, deadlineCapabilities.json, hints.json, rejections.json, hintsRefused.json and rejects.txt. The distinct tenth is rules.json: independently authenticated whole-file Dimming header formatting induces exactly 94 position.line substitutions; all other data is unchanged. No held assertion is counted as passed or regenerated.
- Checker full source suite: rc0, 97 pass including all eight strict StepLedger controls. Log: fn140-10-retry-check-final.log.
- Actual syntax Make target: rc0 with only the server=false Scala invocation override; predicate unchanged. Log: fn140-10-retry-syntax-final.log.
- Final model package and compiled classpath: rc0/rc0; four current production roots freshly lifted rc0. Seven scratch IR/waiver artifacts are authenticated.
- Fresh native20 capture: rc0, all populations nonempty, 52244 applicable rows/27540 enabled rows, exact byte equality with immutable original capture; SHA 0574109cbab5cef0aa95d89fc8c8d100927e0630c717314769b3e0f326091fcb.
- Final direct historical steps-only gate: rc0 against absolute immutable /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn160/baseline-ae3f36ee51 and batch-1d.map. The fresh current index contains 46 keys/45 payloads/2189510 rows; the original has 59 keys/42 payloads/13077242 rows. Two exact binding-order ledger entries and four independently authenticated original dispatch additions are accounted for. Ordinary table grammar/order and strict refusal controls remain unchanged. Log: fn140-10-retry-historical-final.log.
- Authored exports: rc0, four files/51 subject rows/43 distinct exact kind+family.name subjects equal scratch IR. All 43 are present in the fresh native index, with exactly three explicitly classified anonymous derived keys. All seven original baseline IR files match their immutable independent pin hashes. Forty-four retained original pin rows and 17 explicit map removals are accounted for. All retained catalogs match exactly, with only the authorized ActivityProduct stop removal reconstructed to the exact original 115-action pin hash and the exact two ledger old/current binding orders. This counterpart is not a pass for the held managed tests.
- Whole 459-file formatter check: rc0. A second formatting pass is rc0 and the whole before/after source manifest is byte-identical. Log: fn140-10-retry-format-check-final.log and fn140-10-retry-format-idempotence.log. The earlier invalid --conf command is inconclusive, not a formatter pass.
- Final complete core validator, 26 mutations, reader validator, five reader mutations and git diff --check: rc0.

Canonical Make equivalence remains owner-held/unexecuted: its API prerequisite runs protoc/Go before the stale check and is outside authorized execution. No fake failure or Make/full-consumer credit is claimed; the documented direct Scala gate is the completed source route. Supplemental export captures that initially failed compilation are retained as diagnostic logs, superseded by the nonempty final successful capture.

### Frozen handoff

Immutable snapshot: .flow/tmp/scala-dsl/fn140-10-retry-seal. Its manifest.json SHA is e8a7dbf4a9e51ae442cff4ae72de521ef4aff7340dd43ed358cd199a5acecd1e and hashes 5272 copied files, including all 459 final source files, authenticated model jar, 20 actual dependency jars, classpath provenance, seven final scratch IR files, exact original/current typed graphs, frozen original sources/IR, position map/delta, complete inventories, proof scripts, final native table index/payloads and campaign logs. Every copy was source/destination SHA-checked and refuses an existing changed target. The compiled artifact's exact whole 1893 class/TASTy member set matches the compiled classpath byte-for-byte. Model/build paths may be overwritten later; dependent proofs must use this frozen snapshot and manifest, not mutable build paths.

Final authentication receipt: fn140-10-retry-final-authentication.json. Export/catalog counterpart: fn140-10-final-export-subject-catalog-proof.json. Complete typed inventory: fn140-10-complete-typed-reader-inventory.json. Exact final inventories, native captures and all commands/logs are frozen beneath the seal. Conductor independent audits remain in the main checkout's .flow/tmp/scala-dsl/fn140-10-root-* files.

All worker-owned foreground handles are terminal. The source seal is scoped to this task; fn149 release is conductor-owned and requires independently verified committed closure. Tasks .6/.7/.8/.9 retain all deferred consumer/regeneration/Go assessment/full-gate/review/live obligations.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b043cbf03a20fcb2a4dddadaf8c80c26f5c2684f, 45b56bd6775a56c33efd6e5743d4851745da7fc1, 8453cc9640f03ceb1c7f51dcc0cb8d19b4b6a54e, c1a45df0c64ed388f895fb3bf585a768ff75ab2c, 4abc3a617f37df0a684dde305c493000c06e8b3e, 620440e0ead78b59ecbf0565b294a4bc41dcd1b2, 736e1a318b17544b378cf72fed081404d473a10e, 6fc3878be702154a28d1c5d07e78587f2480d298, e8794d8bc51e9e82aa2556891f2411f1d6d7df27, 3e8cfa44489ef754dd558e69b3a718f038d08b91, c12f2992bf382087be244475d4e1ee75b6d9ef55, 12d2ba6b6761ab42091427335379ff1bc86021da, c224617bf1ede205fb44e4b6c62f5ed4f42ac3c9, 35624d8c60366668de9b19e7c79b99517d35e9a2, 5848f9e1555ab2011164c2a2fc3720a24db44a11, 3fecdda1e15448c29720fb6056c37c6b73372f55, 933059cbe0c587ff717af400dca28535b7cd274d, 20e0b96a2fae88dfdc7e49808992f92e6b6bdcf3, b8a75d4f488ae3625b11530439e3143335af5a21, 01042750a667c7589dbca210151bd326d819f8e7, 2f91350112216f20a4430cf2318d5b1b69c6814e, a4f40b53e58d6725da1755b8a8c1c647b8bb5089, dc36d68a549b52a842e53290bab76a98d0ca6f75, 844e62b9d6d9256bdc1897ba63cb949109d15c15, dce8fce7706c0bc2f0bfb02fc874f3469439b633, d0db2a710236667af6b3fad4b73af6158672178d, f80db4ef6ef8f919cb037787c87ce76f7cad91b4, 14d5043ee8155788bb06ff0bcbb35345d9b06a56, b5374fbb0bd13642dea60191f97619754a81acd8
- Tests: baseline: red (inherited original model/irgen source suites; no green handoff), timeout 600 mise exec -- scala-cli test model/project.scala model/framework model/temporal --server=false: rc1,178pass,2heldmanaged,0sourcefail, timeout 600 mise exec -- scala-cli test model/irgen --server=false: rc1,166pass,10heldmanaged,1preexistingskip,0sourcefail; complete226 authored refusals pass, timeout 600 mise exec -- scala-cli test model/check --server=false: rc0,97pass,8StepLedger controls, timeout 600 mise exec -- make lint-model-syntax MODEL_GATE="mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/check --": rc0, mise exec -- scala-cli --power package model/project.scala model/framework model/temporal --library --server=false -f -o model/build/model-scala.jar: rc0, mise exec -- scala-cli compile model/project.scala model/framework model/temporal --server=false --print-classpath: rc0, mise exec -- scala-cli run model/irgen --server=false --main-class umpire.irgen.lift -- --ir model/build/model-scala.jar=model/ model/build/model-scala.classpath .flow/tmp/scala-dsl/fn140-10-retry-final-ir activity-standalone nexus-workflow nexus-standalone taskqueue: rc0, timeout 600 mise exec -- scala-cli run model/project.scala model/framework model/temporal .flow/tmp/scala-dsl/fn140-10-all-native.scala --server=false --main-class framework.RemainingNativeInventory -- .flow/tmp/scala-dsl/fn140-10-retry-all-native-post20.tsv: rc0,20claims,27540enabled; cmp immutablepre20 rc0, timeout 600 mise exec -- scala-cli run model/check --server=false -- --check-equivalence /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn160/baseline-ae3f36ee51 model/testdata/equivalence/batch-1d.map --steps-only: rc0, timeout 600 mise exec -- scala-cli run model/project.scala model/framework model/temporal .flow/tmp/scala-dsl/fn140-10-final-exports.scala --server=false --main-class framework.FinalAuthoredExports -- .flow/tmp/scala-dsl/fn140-10-final-authored-exports.json: rc0,4files,51subjects, python3 .flow/tmp/scala-dsl/fn140-10-exports-proof.py: rc0,43distinctsubjectsets/nativekeys,44retainedpins/17explicitremovals, timeout 600 mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/project.scala model/framework model/temporal model/irgen model/check: rc0,459files, timeout 600 mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf model/project.scala model/framework model/temporal model/irgen model/check: rc0; wholebefore/after manifestcmp rc0 idempotent, python3 .flow/tmp/scala-dsl/fn140-10-inventory.py --retry-final --verify-final: rc0,338queries/19rewrites/14helpers/655exactpositions, python3 .flow/tmp/scala-dsl/fn140-10-inventory.py --retry-final --mutations: rc0,26independentmutants refused, python3 .flow/tmp/scala-dsl/fn140-10-reader-inventory.py --retry-final: rc0,2028anchors/6311originalnodes/6244currentnodes/fullretainedboundaryproof, python3 .flow/tmp/scala-dsl/fn140-10-reader-mutations.py: rc0,5independentmutants refused, python3 .flow/tmp/scala-dsl/fn140-10-final-auth.py: rc0,459sources/1893exactcompiledmembers/20deps/7IR, python3 .flow/tmp/scala-dsl/fn140-10-freeze-seal.py: rc0,5272immutablecopies; manifestSHAe8a7dbf4a9e51ae442cff4ae72de521ef4aff7340dd43ed358cd199a5acecd1e, git diff --check: rc0, Canonical Make/fullconsumer gates owner-held and unexecuted; no passing credit
- PRs: