---
satisfies: [R5, R6]
---
# fn-140-one-sentence-witness-queries-with.4 Prove the first Activity witness migration and remove duplicate terminate

## Description
**Size:** M
**Touches:** [model/temporal/features/activity/standalone/protocol/Protocol.scala, model/temporal/features/activity/standalone/protocol/Completion.scala, model/temporal/features/activity/standalone/protocol/Retry.scala, model/temporal/features/activity/standalone/protocol/Cancellation.scala, model/temporal/features/activity/standalone/protocol/Pausing.scala, model/temporal/features/activity/standalone/protocol/Dispatching.scala, model/temporal/features/activity/standalone/protocol/Timeouts.scala, model/temporal/features/activity/standalone/ActivitySubjectRegression.test.scala, model/temporal/features/activity/standalone/ActivityResetRegression.test.scala, model/temporal/features/activity/standalone/ActivityRetryRegression.test.scala]

Apply the first real Activity witness conversion and prove its Scala source meaning before the remaining Model migration. The original full Go assessment/retry differential, Case membership and regenerated proof obligations are retained in deferred task .9. This task cannot claim those obligations using native execution or scratch lifting.

### Approach
- Begin after .3 and the integrated committed fn-151/fn-155/fn-156 source receipts/mappings. Their deferred whole-spec closures supply no passing credit; .9 retains the full-closure baseline requirement for actual consumer assessment proof. Current ownership is ActivityProtocol plus facet traits under protocol/, not historical claim-container Systems or RetryFailures.scala. Freeze the integrated pre-conversion source/scratch lift, baseline-to-current identity mappings and declaration spans in task scratch; checked-in IR/Cases are intentionally stale and are not this source baseline.
- Inventory every first-batch pinned find and all inbound Property/Scenario readers before choosing conversion. In the current fn-162 source, Completion's single completes Property is shared by its completion Query and Pausing's pauseResume Query; retain that actual shared edge unless a complete reader inventory establishes otherwise. Shared Cancellation declarations must likewise stay shared. Preserve verify/invariant/monitor-only forms and paths whose exact tail class repeats earlier. A triple is retained when any actual reader or witness boundary requires it; record that reason rather than guessing from the old names.
- Convert eligible private path/fact triples without changing Query names, paths/inputs/start values, limits, totals, live expectations or realization membership. Move runtime tests off deleted standalone declarations to the witness's generated core values while preserving independent original predicates, exact receipt assertions and negative checks.
- Preserve the retry claim's full terminal-state equality, including deadlines and attempt count, using ends where required. Run a native records-only versus full-state differential and compare both against the independent old predicate on every applicable machine row. Save the same declaration/path/baseline data for .9's required actual assessment replay; native truth equality does not prove a live assessment. Do not change expected assessments to fit conversion.
- Delete only the hand-written terminate Query and its otherwise-unread Property/Scenario. Keep capability-generated terminateSettles and all its dependencies and bindings. Deleting the managed Case is deferred to .9. Update literal source inventory assertions without weakening protected semantic assertions.
- Capture the complete before/after Query/core/path/expectation/realization inventory keyed by unchanged Query name. Enumerate applicable model rows independently from machine bindings and finite state/input catalogs; compare original and generated predicates on every row, not just successful pinned paths. Preserve rewritten Property truth and row applicability. Map every edited source position to its declaration span and record all changed core identities/functions/derived Limits under R5, plus only the R6 named deletion.
- Preserve strict inherited completion, fatal-failure and pause/resume Go assertion bytes and available prior recorded failure receipts/dispositions, explicitly identifying any missing originals. These path-assessment assertions are not native Scala tests. Freeze independently identified pre/post source, scratch IR, declaration mappings and replay inputs for .9 to execute their actual baseline/post Go assertions when the held consumer lane reopens. Do not invent a native assessment substitute, run Go here or credit inherited records as fresh execution. Their meaning correction remains Batch 5.
- Separately record genuine native baseline/post results, including the ActivitySubjectRegression strict duplicate-Property assertion that currently fails before migration (blank native Property versus activityProtocol.cancelIsRequested at Some()). Keep that assertion strict and its failure separately attributed; all new migration/inventory/all-row proofs must pass. Do not mislabel this source-validation debt as an executed Go assessment or silently waive it.

### Quick commands (serialize heavy commands)
```sh
mise exec -- scala-cli compile model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/project.scala model/framework model/temporal --test-only '*ActivitySubjectRegression*'
mise exec -- scala-cli test model/project.scala model/framework model/temporal --test-only '*ActivityRetryRegression*'
mise exec -- scala-cli test model/project.scala model/framework model/temporal --test-only '*ActivityResetRegression*'
make lint-model-syntax
make umpire-check-equivalence BASE=.flow/tmp/fn160/baseline-ae3f36ee51 MAP=model/testdata/equivalence/batch-1d.map STEPS_ONLY=1
```

Use the established lifter entry point for isolated before/after scratch IR with current jars. No Go commands/edits, production regeneration, full gates/review/live. New migration proofs must pass; separately evidenced inherited failures are not waived.
Owner source-command clarification: the frozen baseline lives in the main checkout at
`/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn160/baseline-ae3f36ee51`.
Its baseline.txt records the actual fn-160.5 capture commit ea5264bc23, clean model, and the
59-table/42-distinct/13,077,242-row index; the directory suffix identifies its pre-Batch source,
not the capture commit. Use that absolute immutable input rather than recapturing current source.
If Make's jar prerequisites attempt owner-held linked-API generation, retain its actual exit and
run the same established Scala gate directly with existing jars already proven by current source
compilation and scratch lifting:

```sh
mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/check -- --check-equivalence /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn160/baseline-ae3f36ee51 model/testdata/equivalence/batch-1d.map --steps-only
```

This performs the actual source StepTables comparison; it does not bypass a comparison, regenerate
managed artifacts, or run Go. Record input jar hashes and the inherited Make descriptor failure
separately; no canonical Make green credit is claimed from the direct source result.

### Independently red historical equivalence baseline

The pre-edit direct comparison on .4's integrated source produced 34 differences against the
immutable pre-Batch1d baseline: Activity Product/Protocol action-row order, four new dispatching
tables, and four unmatched capability rename entries. Preserve the actual log and complete
table populations; these are unresolved earlier source/equivalence obligations, not Batch 5
assessment corrections and not a passing comparison. Do not repair maps, normalize tables,
recapture the historical baseline or relax its comparator in this migration.

Pin the independently generated pre-edit 46-table population at
`model/build/history/steps.4120162900474836399` before changing source, including complete
index/payload hashes and source/jar provenance. The same established Scala --steps-only checker
must compare the post-edit source against that pre-edit population with an explicitly empty
identity map, yielding zero differences over every table/row. This incremental comparison
proves this migration changed no steps; it does not replace the historical baseline.
Also repeat the historical-baseline comparison after edits and demonstrate no additional
differences, with exact inherited drift classified separately from .4's R5/R6 changes.
Task .10 and the mandatory pre-fault seal retain reconciliation against the historical baseline;
their final admission cannot count the incremental comparison or inherited red as that closure.

### Historical receipt availability (verified by conductor)

The surviving historical disposition is preserved in main checkout
`.flow/tmp/fn145-integrated-done-summary.md`, `fn145-integrated-done-evidence.json` and
`fn145-integration-evidence.md`, and the retained fn-145.4 Flow Done summary. The referenced
original `/tmp/umpire-fn1454.aTUadX/.flow/tmp/fn1454` detailed receipt directory is absent.
Pin the surviving summary bytes and identify that missing-original limitation; summaries
are not authenticated underlying execution receipts. They record canonical exit1, completion
failed, fatal unfinished/OOM and pause unreached in that particular run, not three observed
assertion failures. Fn-151.3's later candidate canonical has no RUN/terminal for the trio.

Preserve the actual assertion route
`tools/umpire/conformance/played_test.go: TestALoweredActivityCaseRunsLiveAndReplaysAlike`
and its existing current-source replay seam, without running Go while held. The .9 consumer
proof must execute BOTH frozen pre/post populations afresh and preserve actual per-child
results. No historical summary or unrelated generated-Case test supplies that proof.
## Acceptance
- [ ] A complete first-batch Activity eligibility inventory records converted and retained triples, exact inbound readers and actual Protocol/facet ownership (R5).
- [ ] Native records-only/full-state retry differential covers every applicable machine row; the final witness preserves the old full-state predicate, including deadlines, with a frozen replay handoff to .9 for actual assessment proof (R5).
- [ ] Only hand-written terminate and its unshared source declarations disappear; generated terminateSettles, its bindings/dependencies and protected semantic assertions remain. Managed Case removal stays deferred (R6).
- [ ] Complete Query/core/path/expectation/realization inventories, all-row predicate truth comparison and explicit declaration-span/identity maps pass; differences are only R5/R6-authorized. Query names, live expectations and realization membership are unchanged except the named deletion (R5).
- [ ] Scala compilation, nonempty new migration proofs, scratch lifting, syntax and incremental pre/post step-table equivalence pass with zero differences. The historical-baseline comparison remains explicitly red, is repeated with no additional drift, and is retained for reconciliation at .10/pre-fault seal; Actual native inherited results are pinned before/after and never relaxed or credited as green. Protected Go completion/fatal-failure/pause-resume assertion bytes, available prior receipts/dispositions and frozen replay inputs are retained with missing-original limitations explicit; .9 owns their actual baseline/post Go execution and complete consumer/regeneration/live obligations.
## Done summary
Converted seven private Activity Protocol paths to witnesses while preserving their independent original predicates, full-state retry endings, exact limits, paths, expectations and memberships. Removed only the hand-written terminate Query and its unshared Property/Scenario; capability-generated terminateSettles and its dependencies remain. This closes .4's Scala-source contract, not the whole spec or held consumer proofs.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none; conductor owns the shared independent sweep)
stage: plan-sync - skipped(config: planSync.enabled != true)

### R5 eligibility and source meaning

The independent source pin is commit 89da620c7b36b6ab496eae2e6e4698951e508ef8. `fn140-4-source-baseline/`, `fn140-4-source-baseline-manifest.txt` and `fn140-4-baseline-source-sha256.txt` preserve its committed source, not the main checkout's user-owned SEMANTICS draft. Manifest hashes are b534414e04618a1f113dfcaaf279a091f886f712c4166e419c7d1ca1a37a1b1f (Git tree) and 7578580e4688e3fe5d0b3ede8c109cb8839c045a364e1b3b250c3bd8a96fdc8e (source payload pins). Final source is independently retained in `fn140-4-source-final/`; its complete payload-pin manifest hash is 08716a02a7550b2c80bcd28a66b30a4ad139e93826a4da9341daa77f87cab9dd.

`fn140-4-eligibility-and-inbound-readers.json` inventories all 18 first-batch pinned finds, exact Property/Scenario Query readers, source/test references and realization readers: seven converted, ten shared-boundary triples retained, one R6 deletion. Actual ownership is ActivityProtocol plus Completion/Retry/Timeouts and the retained facets. Completion's completes has six inbound Queries, including completion and pauseResume. Shared Cancellation triples remain; `fn140-4-retained-nonfind-boundaries.json` separately records unchanged verify/invariant/free boundaries, including retryExhaustionByFailures whose exact tail also appears earlier.

Conversions are scheduledCompletedByID, nonRetryableFailure, retry, heldFailedByID, startToCloseTimeout, retryAfterTimeout and retryExhaustion. Tests now read their generated Property/Scenario cores. Existing receipt/path/bounds/expectation assertions remain, alongside independent old predicates and negative full-state/fact checks. No stable Query reference in StandaloneActivityRun changed.

`fn140-4-{baseline,final}-query-core-inventory.json` and `fn140-4-{baseline,final}-full-core-inventory.json` capture all four fresh scratch Models, 339 before/338 after Queries. The complete core comparison is exact except the seven authorized Property/Scenario/holds identities, their generated ending functions, and the R6 deletion. Query paths, inputs, starts, limits (including search budgets), totals, free flags, expectedRun fields and realization memberships remain exact, except terminate's removal from standaloneActivityRun. Explicit original three/six Limits avoid witness default-budget drift.

`fn140-4-query-identity-and-limits-ledger.json` records each original/final Query and generated core, identity mapping and unchanged Limits. `fn140-4-identity-and-function-ledger.json` retains every original ordinary holds body, final generated holds body and added .ends body. Structural checks prove the exact original ending and records conjuncts under explicit step-parameter substitution; only conjunction order changes. For startToCloseTimeout, the original statusTimedOut(startToClose) constructor becomes the same typed enum literal with its startToClose field; that specific R5 representation change is authenticated separately, not hidden by normalization.

`fn140-4-{baseline,final}-declaration-spans.json` records 170/153 declarations. `fn140-4-edited-line-ranges.json`, `fn140-4-source-position-ledger.json` and `fn140-4-paired-position-translations.json` classify 5,394 positions, including 2,014 exact old/new translations. Unchanged lines must translate exactly; replaced cores map from their old declaration spans to the corresponding witness spans. Generic position removal alone is not the proof.

### R5 native differential

The new migration test was red before conversion (`fn140-4-witness-red.log`, exit1, ordinary Queries were not Witness instances), then green. ActivityRetryRegression is nonempty and passes 7/7, including the table-driven independent original predicate proof for all seven migrated claims. It enumerates finite State values and native bindings/input catalogs, not artifact rows: 5,616 states, 21 bindings, 668,304 source/input pairs and 121,176 enabled outputs. Exactly 39,312 applicable pairs are retained, including disabled rows. Enabled counts are 5,616 By-ID completion/failure, 2,160 fatal/retry/timeout-retry/exhaustion, and 1,080 terminal start-to-close rows; other applicable pairs remain disabled.

`fn140-4-{baseline,final}-native.tsv` are byte-identical, SHA256 93959a8f7417cb35baa40d77592378b7992cc3ee7b1dabe9dca93bb12d960e49. The independent archived ordinary predicates and generated predicates agree on every row. Records-only diverges from the old full-state retry predicate on 2,155/2,160 enabled completion rows (also 2,155 timeout-completion rows, 286 timeout-exhaustion rows). The regression rejects all eight changed State fields and missing evidence for an otherwise valid retry result. This is native truth evidence only, not actual Go assessment/live evidence.

### R6 deletion and held replay

`fn140-4-r6-deletion-ledger.json` retains the exact removed terminated Property, terminatedWhileScheduled Scenario, terminate Query and ordinary holds function. The complete core/step comparisons authenticate unchanged generated terminateSettles, bindings and dependencies. No production IR, managed Case, golden, schema, mirror or Go source was changed; managed Case deletion remains .9.

The protected actual Go route is tools/umpire/conformance/played_test.go: TestALoweredActivityCaseRunsLiveAndReplaysAlike, using its existing UMPIRE_ACTIVITY_IR_DIR replay seam. Entire source bytes are unchanged from the pinned commit and retained as `fn140-4-protected-go-replay.test.go`, SHA256 c63deec916b93b86b679785a5791ecbbe5b436f9bb21b006570882bc583bdc77. Both frozen source populations, fresh IR directories, declaration/core identities and native rows are the replay handoff to .9. That task must execute BOTH populations afresh and retain actual per-child completion/fatal/pause and retry assessment results; no .4 Go command or live proof occurred.

Surviving historical dispositions were copied/pinned as fn145-integrated-done-summary.md (0e9ecf083aebddfcc4f76e33fc112c450ee3577a9277b8f26485af591838b385), fn145-integrated-done-evidence.json (0bd4e7d2059f277e02654559d2d59c7480dd8524240c8f3e46df65d30acfc134) and fn145-integration-evidence.md (7d5a94718d1537d276eba352bfcd1086d912072bda4298fc9a6b62bd5295888a). They report canonical exit1, completion failed, fatal unfinished/OOM, pause unreached in that run. The original detailed /tmp/umpire-fn1454.aTUadX/.flow/tmp/fn1454 receipts are absent; these summaries are not authenticated underlying execution receipts. Fn-151.3 likewise supplied no RUN/terminal for the trio. No unrelated generated-Case pass supplies that proof.

### Native verification and inherited results

Compilation passes. ActivityRetryRegression 7/7 and ActivityResetRegression 8/8 pass; ActivitySubjectRegression remains exit1 with 2/3 passing. Its identical strict same-when predicate assertion still fails for cancellation.properties.cancelRequested versus cancellation.properties.cancelIsRequested on client.requestCancel: ` and activityProtocol.cancelIsRequested state one promise when Some()`. `fn140-4-protected-assertion-preservation.json` proves the assertion bytes unchanged. This is the earlier fn-162.3 duplicate-promise gap, not a Go assessment or .4/Batch 5 waiver. Its focused receipt and realization-membership tests pass before/after.

Fresh current-source packaging, printed classpath, authoritative scratch lifting, native enumeration and syntax all exit0. Baseline/current library hashes are d54e36aa6fec921bb93924147b8ef54175da1010a6bfcca778b58f774f1697d5 / 33f6e7af0aea8ef0896ad86008ffd57d6b5bfc97f8a5d40e38b5f7460bed8334. Both classpath files hash to 4f00d4b5d02bd93070c7bc8f927d750572e79416424a7b073363263069fcde98. Existing API/IR dependency jars remain ca66e13980809d30068c5d8e64302ee1bed71cca8e10379e349b9b744c9fe6f2 / 45654467e918f9b4d1cb41812adfd3d208ab264a45de5f484f9020dd188f4301. Authoritative logs carry `-authoritative` suffix. Failed initial attempts are disclosed: two wrong lift prefixes, strict red baseline, intended migration red, List[Any] strict-equality compiler error, unreachable fallback under Werror, and initial task-local inventory-proof iterations. The first empty-prefix lift's log was overwritten by the successful lift and survives only in the tool transcript; the second wrong-prefix log and other failed-attempt logs are retained. No printed compiler error was accepted as green.

Syntax command: `make lint-model-syntax MODEL_GATE='mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/check --'`, exit0. Canonical Make equivalence prerequisite attempt remains exit2 with the inherited realization.proto descriptor mismatch (`fn140-4-baseline-equivalence.log`); no Make green credit is claimed and no schema/build/map/comparator repair was attempted. One prerequisite unexpectedly entered its linked-API builder before failing; no independent Go test/run was issued afterward.

### Two distinct step-table baselines

Immutable historical input remains read-only at /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn160/baseline-ae3f36ee51. Actual capture commit is ea5264bc23a9d994ca9e1bea33dafd4341684cab, clean model, 59 tables/42 distinct/13,077,242 rows. baseline.txt SHA256 is 0da41ab3e54c8993b69dce150722e09be536f92158b66ee53cf8105cfa14c9fd; step-tables/index.tsv is b9910d76cef97a3c66e0ef4429534f11a60df69595b81856feba0e1f0c75a093. The suffix is not the capture commit. Nothing was overwritten or recaptured there.

The independently generated pre-edit integrated population is model/build/history/steps.4120162900474836399, retained at fn140-4-pre-edit-steps: 46 tables/2,189,510 rows. Index SHA256 is 3a8f32405bfc69ce0f940097398575d0c3313facdd43fb88dd2bac5d8b8f3bcd; complete compressed-payload pin manifest fn140-4-pre-edit-steps-pins.txt hashes to 5bdd70c8aa3060656b132e859044ba45f7b8f43e4f9019f36fe7d04ce9c1a7c3. The proof decompresses/authenticates every payload against its indexed digest and row count. The explicit empty map hashes to e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855.

`mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/check -- --check-equivalence .flow/tmp/scala-dsl/fn140-4-pre-edit-steps .flow/tmp/scala-dsl/fn140-4-empty.map --steps-only` exits0 with zero differences. Post-edit tables are steps.1430268193809240274; their complete index equals the pre-edit index byte-for-byte. This proves the migration preserved every native step row, not historical equivalence closure.

The same established direct checker against the immutable historical absolute baseline and model/testdata/equivalence/batch-1d.map exits1 both before/after, exactly 34 identical diagnostics. Post historical tables are steps.7235116530622136935, with the same pre-edit index. `fn140-4-historical-equivalence-drift.json` compares exact diagnostic bytes: Activity Product/Protocol action-row order, four new dispatching tables and four unmatched capability renames. These inherited differences remain .10/pre-fault reconciliation obligations, not Batch 5 corrections or a blanket waiver.

`python3 .flow/tmp/scala-dsl/fn140-4-proof.py verify` exits0 (`fn140-4-proof-authoritative.log`, `fn140-4-proof-result.json`). `git diff --check` passes. Heavy commands were serialized; every started handle reached terminal status. No full gate, independent review, production regeneration, push/PR, rebase/history rewrite or MILESTONES edit occurred. Owner-authored .4 task metadata changes accompany the implementation commit.

### Remaining obligations

Deferred .9 retains actual Go/native-IR consumer assessment replay for BOTH populations, retry assessment differential, managed Case removal, full fn-151/fn-155/fn-156 consumer closures, shared production regeneration, complete consumer/full gates and live evidence. .10/pre-fault seal retains historical-equivalence reconciliation and the existing fn-162.3 strict cancellation-duplicate obligation. Independent review remains explicitly skipped here. No whole-spec/goal completion is claimed.
## Evidence
- Commits:
- Tests:
- PRs:
