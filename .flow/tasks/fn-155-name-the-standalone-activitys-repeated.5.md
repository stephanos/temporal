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


### Integrated review evidence pointers

The isolated worker evidence is retained at `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn155-task5/.flow/tmp/fn-155/task5/`: `handover-summary.md`, `handover-evidence.json`, and `combined-twelve-leaf-handover-addendum.md`. The current `proof-manifest-v4-twelve-detail.json` has SHA256 `9686edd7956cc935b6c9d96dbda2b44db217e470164225b024144e800c1fa11c`; the conductor independently verified all 280 sealed files.

`combined-twelve-leaf-v2.json` and `combined-twelve-leaf-proof-v2/report.json` retain the exact 31-file restoration under eleven Case line leaves and one manifest refusal Position, 79 rejected Case mutations, 19 rejected manifest mutations and four metadata-independence controls. The manifest is a constructed single-token projection of the immutable original tied to three fresh refusal outputs, not whole fresh generation. The original strict Case and manifest comparisons remain RED. Fresh joined generation, managed-tree agreement and full gates remain task .6 obligations.

`timeout-detail-proof/report.json`, `timeout-detail-production.json`, `timeout-detail-proof/input-pins.json` and `timeout-detail-proof/raw-output-comparison.json` retain eight exact original/current pairs rendered by production `expiryDetail`, using actual default ceilings, a 1024-byte Detail cap and unscaled 100% bounds. All eight strings are below the cap and restore their original bytes after only the eleven enumerated coordinates. These rendering observations and constructed INCONCLUSIVE identity controls establish no live Run, successful matching replay or semantic-equivalence credit.

The conductor evidence root is `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn-155-name-the-standalone-activitys-repeated/.flow/tmp/`: `fn155-integration/task5-integrated.json` pins normalized source base `b528cd346f7614cce5e83dbde910ecc9cf423db0` and source head `4d5982d2be198469e258028c7b48f0a36e6ea96c`; `fn155-integration/task5-pre-review-focused.json` and `fn155-integration/task5-pre-review-lint.json` record terminal exit 0 on that source head with their input hashes subsequently rechecked by the conductor. `notes_dir.20261009T235609Z-root` locates the conductor notes. This appendix adds evidence pointers only; requirements, acceptance, scope and dependencies remain unchanged.
## Acceptance
- [ ] TBD

## Done summary
# Realization integration

Single definitions now supply attempt records, Describe reads, status/run conditions, activation and run-scoped Describe requests. Only Realization.scala changed; identities, guards, ordered evidence, causal scopes, confirms and attempt ordinals remain unchanged. Worker source ad94da388e76cc528333ea3685a256beaca562f1 is reachable through source merge 4d5982d2be198469e258028c7b48f0a36e6ea96c. Normalized base b528cd346f7614cce5e83dbde910ecc9cf423db0; reviewed head 18896e446c7f567aecbe731e713dfc2342d382f0; finalized cumulative ledger 05654f348ef8f2cfb00a3b91211985f7f363814d. Source SHA256 is 5927eebc77c1dfbb5b5fa4aa69e8dc1068901b945cd53984ac6ca9209eddcee4.

The independently reviewed provenance amendment is committed in 331b6548891d2413fd5df6f797cd1e362b0a8c59 and integrated in 4fc8bf94a6cd581e9c49a5d2f81bc8cded2d22f6. The scoped proof restores all 31 immutable original raw files after exactly eleven fixed Case line integers and one separately fixed manifest refusal Position. It retains 79 rejected Case mutations, nineteen rejected manifest mutations and four metadata-independence controls. Root independently checked all 280 V4 sealed files, SHA256 9686edd7956cc935b6c9d96dbda2b44db217e470164225b024144e800c1fa11c. Full three-Model projections, sixteen complete authored-root partitions, twenty fresh Activity Cases, unchanged default Check/Find paths, exact current identities/bindings and crossed historical rejection are retained.

Eight exact old/current timeout Detail pairs come from production expiryDetail with actual default ceilings, 100% bounds and the 1024-byte cap. All are below the cap and recover original output bytes after only the enumerated coordinates. The complete affected current IR/Case selector inventory found no semantic Detail consumer and detected its seeded controls. No scaled-profile, truncation, live Run or successful matching replay credit is claimed.

Actual fresh-context codex:gpt-6.1-sol:high correctness, contracts and integration review draws all returned SHIP without findings (same GPT family). RID f94e4af1e82d45fba6da31376ebe59c5; receipt at agent/fn155-task5-review/.flow/tmp/fn155-task5-formal-review/receipt.json, SHA256 c3aac617ae6c369fd721e1f02f1b1c1b48d9dbc1f8b0dccc68bfcff2c4c0f397. The seven predecessor attempts remain an exact prefix. A reviewer's fresh Go-test attempt was blocked before execution by its read-only sandbox; it supplies no test credit.

Integrated pre-review Activity Scala tests and model lint passed with independently rechecked source inputs. Immediately after review-ledger integration at 05654f348ef8f2cfb00a3b91211985f7f363814d, all 24 top-level realization/lowering tests passed in 8.673 seconds and model lint passed in 31.761 seconds; shared-lock waits were 184.629 and 191.313 seconds. Both commands exited naturally 0 with unchanged source inputs, independently rechecked after completion. Root receipts are .flow/tmp/fn155-integration/task5-post-review-focused.json and task5-post-review-lint.json. All attributable commands have terminated.

stage: memory-capture - skipped(clean first-round SHIP; no review fixes)
stage: plan-sync - skipped(policy: rolling route)

Original strict Case/manifest comparisons remain RED, not rewritten as passes. The manifest proof is a constructed single-token projection tied to fresh refusals, not whole fresh generation. Task 6 still owes fresh joined production artifacts, exact joined source-provenance rederivation, all managed-tree and full gates. Inherited Batch 5 semantic failures, fn154 Quint memory and fn157 resource deferrals remain strict; no full-generation, full-suite, live or replay credit is supplied here.
## Evidence
- Commits: ad94da388e76cc528333ea3685a256beaca562f1, 4d5982d2be198469e258028c7b48f0a36e6ea96c, 18896e446c7f567aecbe731e713dfc2342d382f0, 05654f348ef8f2cfb00a3b91211985f7f363814d
- Tests: mise exec -- go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/realization -count=1 -v, mise exec -- go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/lower -run "^(TestHeartbeatPendingPublicationPrecedesItsFutureTimer|TestHeartbeatDetailsLowerOrderedTypedMessages|TestHeartbeatDetailsRefuseCrossedMessageLists|TestHeartbeatDetailsRefuseANilMessageList|TestTheFieldsAndTheSingleReadOfEvidenceAreCheckedAgainstTheirDescriptors|TestEvidenceReadFromOneMessageWithItsFieldsLowers)$" -count=1 -v, make lint-model, Integrated post-review checks passed: 24 top-level realization/lowering tests and model lint, unchanged input hashes independently rechecked, Sealed complete three-Model projection and 16 original authored-root partitions retained; full 31-original-file comparison restores only 11 Case integer leaves plus one manifest refusal Position, 79 Case and 19 manifest candidate mutations rejected; four metadata-independence controls passed; exact 20 current bindings and eight crossed historical identity rejections preserved, Eight exact production expiryDetail pairs at default unscaled ceilings; complete affected current IR/Case Detail-selector inventory with seeded detection controls, Root independently verified all 280 V4 sealed proof files, SHA256 9686edd7956cc935b6c9d96dbda2b44db217e470164225b024144e800c1fa11c, Actual Codex gpt-6.1-sol high correctness/contracts/integration SHIP, no findings; RID f94e4af1e82d45fba6da31376ebe59c5; seven predecessor review attempts preserved exactly, Original strict Case and manifest RED retained; constructed manifest not whole fresh generation; full gates and fresh joined output remain task6; no canonical/live/replay credit
- PRs: