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
- Preserve strict inherited completion, fatal-failure and pause/resume assertions. Record baseline failures separately, verify their unchanged status after migration, and do not count them as passing checks or fix them under this meaning-preserving task; their correction remains Batch 5.

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
## Acceptance
- [ ] A complete first-batch Activity eligibility inventory records converted and retained triples, exact inbound readers and actual Protocol/facet ownership (R5).
- [ ] Native records-only/full-state retry differential covers every applicable machine row; the final witness preserves the old full-state predicate, including deadlines, with a frozen replay handoff to .9 for actual assessment proof (R5).
- [ ] Only hand-written terminate and its unshared source declarations disappear; generated terminateSettles, its bindings/dependencies and protected semantic assertions remain. Managed Case removal stays deferred (R6).
- [ ] Complete Query/core/path/expectation/realization inventories, all-row predicate truth comparison and explicit declaration-span/identity maps pass; differences are only R5/R6-authorized. Query names, live expectations and realization membership are unchanged except the named deletion (R5).
- [ ] Scala compilation, nonempty new migration proofs, scratch lifting, syntax and step-table equivalence pass. Strict inherited completion/fatal-failure/pause-resume assertion results are independently recorded before/after, never relaxed or credited as green; .9 retains the complete consumer/regeneration/live obligations.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
