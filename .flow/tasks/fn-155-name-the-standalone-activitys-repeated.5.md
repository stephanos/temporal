---
satisfies: [R6]
---
# fn-155-name-the-standalone-activitys-repeated.5 Single-definition Realization evidence builders

## Description
Implements the local items of spec section F in the Realization file. Each of these gets a single definition: the run-event attempt record, the Describe read, status and run-state conditions, the worker activation, and the run-scoped Describe base. The per-policy `deadlines` generator is deferred (spec Boundaries).

**Size:** M
**Files:** `model/temporal/features/activity/standalone/system/Realization.scala`
**Touches:** [model/temporal/features/activity/standalone/system/Realization.scala]

### Approach
- **Attempt record.** `attemptRecord(script, number, carrier, extraGuard*)` replaces `heartbeatDelivered`, `heartbeatRetryDelivered` and `resetDelivered`. Its guard must stay byte-identical in lowered Cases.
- **Describe read.** Rename `externalRead` to say it reads Describe. Reuse it for `heartbeatReceipt`, `heartbeatCompleted` and `heartbeatExpired`. Move `activityRunFieldForInfo` above its first use.
- **Conditions and activation.** `statusIs`/`runStateIs` conditions, an `activation` val, and a run-scoped Describe base carrying `runId := executionRun`.
- **Lift check.** Each helper must lift. The lifter follows helper defs with bound parameters (`model/irgen/Realizations.scala:86-144`). Check each one with a scratch lift.

### Source-provenance proof

- **Sealed inputs.** Reuse the original .1 seal and original producer/domain lower path. Record exact original/current Scala, lifted IR, producer and domain input hashes. The original source commit is `9a146bb59e7ee7ae2fc249d4017bdda77aca488e`. The frozen original artifacts reside in the existing .1 worktree's `.flow/tmp/fn-155/before/`. Pin the complete original file inventory independently, including 20 activity Cases, 10 Nexus Cases and `manifest.json`. Freshly lower all 20 original activity Query witnesses with original Check-per-Query and LowerFind inputs. Preserve the ten Nexus bytes and immutable original manifest archive; freshly compare the candidate manifest under its single separate diagnostic exception below. No diagnostic boolean establishes membership or equality.
- **Fixed exception membership.** Derive and seal the eleven typed locations below from the frozen originals before inspecting candidate changes. Use Case, entrypoint, instruction, hint, realization and complete ServerStep action identity together. Reject missing, duplicate, omitted, mislabeled or extra entries, mismatched selectors and missing/extra files. Query coverage and original unsupported/fatal dispositions remain strict.
- **Source identity.** Independently validate each original coordinate against the pinned original `model/temporal/features/activity/standalone/system/Realization.scala` and its lifted ServerStep; independently validate each candidate coordinate against exact current inputs and the corresponding declaration/full action. Require uniqueness. For `resetAfterHeartbeat`, require the explicit heartbeat ServerStep within `ResetAfterHeartbeat.serverSteps`, not merely an enclosing object or a matching statement elsewhere. Candidate line numbers below are observations to re-confirm from final source pins.
- **Raw bytes.** Verify every candidate file, including the twelve currently equal activity Cases. After the separately approved identity mapping, the comparator may restore only the eleven authorized Case integer leaves and the single independently selected manifest diagnostic coordinate below. All 31 raw original files must then match, including serialization, order, path, hint metadata, guard, evidence, cause/causal scope, confirms, bounds, expectations and refusal meaning. The existing IR projection remains a separate proof. Generic Position removal, source-token replacement without typed membership, or normalized-JSON equality cannot establish this contract.
- **Derived effects.** Use existing production CaseIdentity/CaseFingerprint/prepared-binding seams and timeout-detail rendering. Record old/current values and exact input hashes, then show new-Case/old-Run and new-Case/old-factory rejection while retaining original Case+Run bytes. Inventory affected current IR/Cases for semantic consumers of changed timeout Detail; if any exists, or completeness cannot be proved, report failure. Do not change production hashing, admission or timeout code to create equivalence.
- **Strict controls.** Exercise the executable full verifier on each of the eleven locations with wrong-current-line, wrong-file, stale-original-line and another valid declaration. Exercise omitted/mislabeled/duplicate allowlist entries, missing/extra Case files, a mutation to a currently equal Case, and changes outside the allowlist including other coordinates, bounds, run-event guards, evidence order and confirms/causal scope. Each mutant must fail. Changing report booleans alone is not a control.

### Fixed source-line inventory

Every entry uses `controller` as entrypoint and `model/temporal/features/activity/standalone/system/Realization.scala` as the unchanged source path. The allowlist contains only `source.line` under the named wait hint. Full actions and declaration identities are validated separately; these short hint names do not replace that check.

| Case filename suffix | Instruction | Hint | Realization | Original -> observed current line |
| --- | --- | --- | --- | --- |
| deferredResetCompletes | await-reset-completed | deadline.heartbeat | resetAfterHeartbeat | 1030 -> 950 |
| heartbeatTimeoutExhausts | await-heartbeat-expiration | deadline.heartbeat | exhaustAfterHeartbeat | 593 -> 579 |
| heartbeatTimeoutRetriesThenCompletes | await-heartbeat-retry-completion | deadline.heartbeat | retryAfterHeartbeat | 548 -> 526 |
| heartbeatTimeoutRetriesThenCompletes | await-heartbeat-retry-completion | deadline.backoff | retryAfterHeartbeat | 548 -> 526 |
| retry | await-completed | deadline.backoff | retryFailuresExecution | 465 -> 432 |
| retryAfterTimeout | await-completed | deadline.startToClose | retryAfterTimeout | 473 -> 440 |
| retryAfterTimeout | await-completed | deadline.backoff | retryAfterTimeout | 473 -> 440 |
| retryExhaustion | await-failed | deadline.startToClose | retryAfterTimeout | 473 -> 440 |
| retryExhaustion | await-failed | deadline.backoff | retryAfterTimeout | 473 -> 440 |
| scheduleToStartTimeout | await-timed-out | deadline.scheduleToStart | timeoutsExecution | 469 -> 436 |
| startDelayedCompletion | await-completed | deadline.startDelay | dispatchExecution | 468 -> 435 |

Filename pattern is `activity-standalone-<suffix>-case.json`. The five full actions are rooted at `temporal.features.activity`: deadlines are `.deadline.heartbeat`, `.deadline.startToClose`, `.deadline.scheduleToStart`; timers are `.timers.backoff` and `.timers.startDelay`. The verifier retains each entry's exact action rather than suffix matching.

Source-site identities are `RetryFailuresExecution extends DerivesFrom(Standalone, RetryFailures)`, `DispatchExecution extends DerivesFrom(Standalone, DispatchEligibility)`, `TimeoutsExecution extends DerivesFrom(Standalone, Timeouts)`, `RetryAfterTimeout`, `RetryAfterHeartbeat`, `ExhaustAfterHeartbeat`, and the explicit heartbeat `ServerStep` within `ResetAfterHeartbeat.serverSteps`.

### One separate manifest diagnostic coordinate

Only `manifest.json`'s original Query selector `{Family: temporal.features.activity.standalone.system, Owner: timeouts, Name: startToCloseTimeout}` and its `unsupported[0]` refusal may update `Position`. Freeze that selector and the original refusal `{Construct: "attempt that gives no answer", ID: "attempts", Owner: "none: a recorded limit of the prototype"}` together with its exact original Why. Standing remains `unsupported`. The original Position is `model/temporal/features/activity/standalone/system/Realization.scala:310`; original line 310 uniquely declares `private val attempts = script(`. The isolated task .5 source has that same declaration at observed line 278. Preserve its file path and independently bind both exact source declarations to the unique lifted Script named `attempts`.

Run the unchanged original producer with original/current default Check and Find against the original unsupported Query, using pinned original domains. Require fresh located-refusal receipts proving that standing, Construct, ID, Owner and Why are identical and Position is exactly the independently derived source-script coordinate. Source inspection alone is insufficient. `unanswered()` uses `locate(s.GetPosition())`; it is not changed to satisfy the comparison. A copied manifest provides no fresh canonical-manifest credit. This exception emits no Case and changes no Case content identity; derive the new manifest digest normally.

The executable verifier must reject wrong file, wrong current line, stale original line, another valid declaration, missing/duplicate/mislabeled Query or refusal entries, changed standing, Construct, ID, Owner or Why, and every other nonallowlisted manifest byte. Preserve all original archive bytes. Task .6 repeats this exact selector, fresh refusal and raw-byte proof against the joined source; no other manifest coordinate is authorized.

The isolated refusal receipt is `manifest-refusals.json` (exit 0) with `manifest-refusals.log`, `refusal-proof/input-contracts.json`, `refusal-proof/manifest-refusal-comparison.json` and `proof-manifest-v3-refusals.json` under the task .5 evidence directory below. It freshly runs the unchanged producer against original declaration entries rehydrated from the sealed original IR and pinned current partitions, retaining original scopes/domains. The timeout refusal is unchanged except 310 -> 278; both Cancellation refusals remain at `Kit.scala:248`, and none emits a Case. The single-token manifest projection protects all other original bytes but is not a freshly generated whole manifest. These receipts grant no canonical-generation credit and do not substitute for the required final joined manifest generation, source proofs or mutation controls.

### Proof handover and limits

Supporting receipts are `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn155-task5/.flow/tmp/fn-155/task5/`: `source-sites-v3.py`, `source-site-validation-v3.json`, `case-diagnostics.json`, `provenance-identity-map.json`, `content-identities-and-admissions.json`, and the sealed proof manifest. V3 independently freezes all 31 original files and eleven locations; 79 candidate mutations reject and two diagnostic-independence controls succeed. The source candidate is `ad94da388e76cc528333ea3685a256beaca562f1`; current source SHA256 is `5927eebc77c1dfbb5b5fa4aa69e8dc1068901b945cd53984ac6ca9209eddcee4`. These pins are provisional until final joined reproof. The root's rolling notes are at `/Users/stephan/Workspace/skunkworks/umpire/temporal/.git/flow-notes/fn-155-name-the-standalone-activitys-repeated-20261009T235609Z-root/conductor.md`.

The admission harness derives both production content identities and default preparation/binding for all 20 Cases, with crossed historical identity rejection for the eight changed Cases. Its matching records are constructed INCONCLUSIVE identity controls; it supplies no successful matching replay.Admit, actual violated replayable Run, live run or semantic-equivalence credit. Retain actual original Case+Run pairs and all existing true live/replay obligations. Never relabel a historical Run or receipt.

### Investigation targets
**Required:**
- Pre-split `system/Realization.scala:93-243`, `:658-751`, `:919-961`
- `model/irgen/Realizations.scala:86-144`, `:812-848`

### Key context
- **Trigger identity.** Joint conflict scopes need realized trigger identity (memory: joint-conflict-scopes-require-realized). Keep evidence ids and `confirms` unchanged.

### Acceptance
- [ ] The R6 full-inventory comparison restores only the eleven authorized Case line leaves and one separate manifest diagnostic coordinate after explicit identity substitutions, recovering all 31 original raw files; the original strict RED comparison remains retained. `make umpire-check-cases` remains required on regenerated managed artifacts at closure.
- [ ] Each listed builder has one definition
- [ ] Focused realization and lowering tests pass
- [ ] Both pinned source sides and complete realization/action identities are validated independently for all eleven leaves; every listed mutation rejects through the executable verifier and inventory-independent diagnostic controls succeed.
- [ ] Fresh original/current unchanged-producer default Check and Find reproduce the single manifest refusal at independently bound source/lifted-Script coordinates. Its standing, Construct, ID, Owner, Why and path remain identical; every manifest mutation above rejects.
- [ ] Production-derived identity/binding/timeout-detail effects are disclosed, changed-detail semantic consumers are absent from the affected current IR/Cases, and crossed old bindings reject. Constructed controls grant no live/replay/conformance credit.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
