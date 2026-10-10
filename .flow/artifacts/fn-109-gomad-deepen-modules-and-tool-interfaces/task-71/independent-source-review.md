# Task 71 independent source and bounded behavior review

The frozen candidate correctly restores the three admitted completion-characterization consumers without changing their executors, tables, assertions, or production behavior. The retained worker logs show all 60 original failures becoming passes and the same 67 controls remaining passes. No new source defect was found: critical 0, important 0, minor 0. This is a bounded source-progress assessment, not formal SHIP, task completion, native qualification, or an aggregate green verdict.

## Review identity and authority

The conductor requested `gpt-6.1-sol` at high effort. Actual executed-model telemetry is unavailable and is not inferred from that request. Writer and reviewer were requested from the same GPT family. After a fresh-session attempt failed with ENOSPC, the conductor explicitly authorized reuse of this task-70 review context. This review is independently performed but is **not a fresh-context review**; that limitation remains part of its acceptance boundary.

The candidate reviewed is in `/Users/stephan/Workspace/skunkworks/.gomad-scripted-and-spin-corrections.gFiXmTVr/completion` (ROOT71). Its actual BASE and frozen HEAD are `30a8847362779670311486706701233d0043be07`; the frozen handover records no worker commit. PRIMARY remains the authoritative owner workspace, not the isolated historical owner-spec copy. I read the current AGENTS instructions, Gomad README and Milestones, authoritative owner requirements, task 71, admission, preparation note, and completion-next-slice research. The authoritative PRIMARY owner-spec SHA256 is `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c`. Task-body SHA256 is `65504e4cbc0226dde29b9d2c44383d53167ca8ff7a3c1804dea4d841d24cc3a8`; admission SHA256 is `cdf1ec74a039468e2126fde8c7abba8b4cdfdc57cfe39506668d618219bf46a6`. The admission's historical base does not override the actual BASE above.

I applied the review-code, verification-before-completion, and flow-next-prose instructions within the conductor's restricted review scope. This review used read-only source, Git-object, hash, JSON, and retained-log checks; it launched no Go/environment/tool probe, build, test, lint, vet, generator, wrapper, native check, additional agent, or lifecycle operation. Its only write is this report, outside the immutable worker seal. No product, sealed evidence, Git state, cache, Flow state, CI, PR, or publication was changed.

## Whole-file boundary and integration path

The sole changed product file is `tools/gomad3/runner/completion_characterization_test.go`:

- BASE SHA256: `b865856c22c519d3b9af29b65cbc5cf0c72b288b5380875f6809801c7a794e3f`.
- Candidate SHA256: `bd9ad5807a84c484dab8ed3d5f1e2ed1863b8a985bb0e3841125271b697bcd82`.
- Independently inspected Git diff: exactly three inserted lines and zero deleted lines.
- Independently removing those three exact assignment lines reconstructs the entire BASE file byte-for-byte, with the BASE SHA256 above. This preserves imports, comments, fault datums, expected errors, identities, counters, projection expectations, and all assertions—not merely the visible diff context.
- The product-scope diff against BASE names only this file. Shared helpers, seed-completion consumers, preparation owners, public APIs, bootstrap guards, runtime/toolchain inputs, and lint policy are unchanged. The shared preparation fixture remains BASE-equal, SHA256 `c43c4fb18ad07b9b9bbb6efba9dc5a6194a99d86ae2405cd0dc4b6088943402e`.

Every added line is `configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)` immediately before its existing `exploreWith` call:

| Consumer | Candidate line | Preserved dependency and final mutation |
| --- | --- | --- |
| TestCompletionFaultsKeepReasonPrecedenceAndEvidence | 343 | The outer faultExecutor returned by completionCampaign, including its fault, supervisory error, shared mutex, and captured result |
| TestCancellationIsAHostFailure | 375 | The final blockingExecutor replacement, progress-driven cancellation, and 10ms TerminateGrace |
| TestCompletionProjectsWorldCoverageAndChoicesForEveryStrategy | 432 | The outer result-capturing faultExecutor and final CollectExecutionEvidence=true |

The assignment passes the current `config.Preparer` and current outer executor. It does not unwrap `faultExecutor.base`, replace a strategy executor, or attach inside `completionCampaign`. That shared helper is also consumed by excluded seed-completion characterization; modifying it would have broadened this task improperly.

The existing scripted helper checks nonnil explicit dependencies, forwards the actual preparation request to the selected preparer, verifies the prepared fixture, rejects adapter substitutions, and supplies the explicitly synthetic bootstrap marker. Its marker is not a patched-toolchain frame. Local orchestration, strategy dispatch, completed-execution assessment, error classification, publication, journaling, and partial/resume-plan handling remain real source paths. Public Explore still starts with empty private dependencies; nil private preparation/bootstrap hooks retain their real defaults; isolated requests still reject injected execution/preparation before starting the coordinator, subject to the unchanged resume-preflight precedence.

The 17-fault table still runs each fault against seed, choice exploration, and simulation exploration. World decode/seed mismatch, semantic and choice failures, supervision failures, watchdog/cancellation combinations, strategy-specific cancellation causes, counters, publication failures, journals, and partial states keep their original expectations. Real artifact.OpenArtifact, canonical journal decoding, and filesystem partial inspection are not substituted. Cancellation retains errors.As/errors.Is checks, zero-failure/artifact assertions, and real seed resume-plan/partial checks; exploration's intentionally unsettled partial exclusion is unchanged. Positive projection retains real campaign.OpenCampaign, World decoding/composition, semantic-probe summary, projection of the captured trace, per-execution journal assertions, and DeepEqual comparison. Only the existing diagnostic canonical-record logging clears elapsed fields. No new assertion weakens these boundaries.

## Frozen packet and actual terminal coverage

I independently verified all 187 named members of `packet-worker-seal.sha256`, with zero hash mismatches and no duplicate member names. The seal SHA256 is `49a7cea954e743109f2d5fd177bbd8b80e5d444a99c12d52df31e59f7a1d02d6`. The worker summary SHA256 is `422f7b7e84a1c15250cef799f9c4f3d55e96290b45f126cbeece8cfe8412eaa8`; worker-evidence JSON SHA256 is `3161365e675991f830f9d6b7f1e08f206ab6cecc7137d79740c8a52887b58ad6`.

I parsed the retained raw Go JSON independently, counting only actual named terminal actions and rejecting duplicate terminals:

| Retained capture | Principal exit | PASS | FAIL | SKIP | Actual named structure |
| --- | --- | --- | --- | --- | --- |
| baseline-three | 1 | 0 | 60 | 0 | 3 parents, 57 leaves |
| baseline-controls | 0 | 67 | 0 | 0 | 36 parents, 31 leaves |
| final-three | 0 | 60 | 0 | 0 | The exact original 3 parents and 57 leaves |
| final-focused | 0 | 127 | 0 | 0 | Exact original/control union: 39 parents, 88 leaves |
| final-boundaries | 0 | 5 | 0 | 0 | 5 actual top-level boundary tests |

The 57 original leaves are 51 fault/strategy cases, three cancellation strategies, and three positive projections. All 54 fault/cancellation leaves print the preparation.stageError validation refusal; the three positive-projection leaves print the unsupported linux/arm64 deterministic-I/O refusal. The unchanged baseline therefore demonstrates a meaningful setup-stage RED, not a hypothesized defect. No slash intermediate outcome was synthesized.

The original terminal-name set is identical before and after. Each of its 60 names changes FAIL→PASS. Every one of the 67 actual control names remains PASS in the focused run; the focused set contains no additional or missing name. The control split is Runner 62 and deterministicio 5; the focused split is Runner 122 and deterministicio 5. Focused coverage was derived from the actual logs, not forecast from a prior task.

The raw control outcomes include real-default/bootstrap refusal (public, executor-only, prepare-only), all isolated executor/preparer/replayer refusals, isolated private preparation/bootstrap combinations and precedence cases, portable adapters/requirements/bootstrap/prepared guards, preparation error forwarding/stage boundaries, and local completion/finalization/cancellation controls. These remain genuine guards, not synthetic bootstrap acceptance claims. The five fresh boundary-test executions are package architecture, private Runner injection, external-module public request compilation, exact module edges, and pure-module host-effect checks; all five pass without a skip.

## Standards receipts and retained RED

The retained numeric principal receipts show format, whitespace/diff check, whole-file preservation, host Runner vet, standalone Runner errortype (`-style-check=false`), CGO=0 affected-source-set vet for darwin/arm64 and linux/amd64, required fast lint, and generated validation exiting 0. The static commands cover Runner, gomadtool conformance, and Runner execution; they are not full native executions or aggregate source-set qualification. Generated validation's raw log reaches version/protocol/boundary checks, compiler-test boundary validation, patch/script ownership checks, compatibility-pack checking, its host-pack test, and qualification-manifest checking.

Configured unfiltered Runner lint is exit 1 both before and after. I independently extracted all six complete header/source/caret blocks from the raw logs, confirmed no missing caret block, and compared their full bytes. Both block sets have SHA256 `0343dd7822baa18fb03d36e7917bd5c69ff5bd5b6e483f7a0f3c592ae52cb31b`: zero introduced or removed blocks. A comparison exit 0 means preservation, not lint green.

Fast lint's raw processing statistics show 50 post-policy diagnostics diff-filtered to zero; its exit 0 is scoped to that changed-line gate. Its successful Make recipe reaches the silent errortype recipe, and the standalone Runner errortype receipt supplies separately measured evidence. Neither result supersedes unfiltered Runner RED or the conductor's full-50 diagnostic comparison obligation.

Default lint-cache ENOSPC warnings and sparse-checkout find warnings are retained. The packet reports 1561 ENOSPC lines in each baseline/final Runner lint log, 3059 in fast lint, and six sparse Make find warnings. There was no lint-cache override or cleanup and no warning-free claim. The explicitly recorded setup transition is a previously absent git-ignored generator-cache symlink to the retained-success cache; its contents are not qualified by this review.

## Capture limits and acceptance boundary

I independently checked the hashes and actual numeric exit/elapsed/UTC metadata of all 22 principal receipts and their raw logs. All 44 retained raw environment observations hash to their named inputs. Independently normalizing exactly one numeric GOGCCFLAGS go-build mapping per observation makes each pre/post pair byte-identical; every other byte and option stays unchanged. The current final-validate source manifest contains 1287 present hash lines and 72 explicit absences; I checked every present hash and absence against the frozen workspace with zero mismatches. Its tool manifest has 42 hash lines representing 31 distinct literal paths; all 42 current hashes match. This is an explicit input domain, not a hermetic installation/cache/C-library guarantee.

The three original baseline captures lack separate post-helper rehashes of the raw environment proof inputs. That historical window remains unbound. Additive v2/v3 captures hash checker and input bytes around their measured proof and retain current proofs of the original raw pairs, but cannot retroactively repair the earlier window. The failed proof batch exit 3, its misleading first receipt name (actually the baseline-three input pair), and the unexecuted original outcome adapter remain preserved. The corrected successor inputs and names do not rewrite historical captures. These disclosed limitations prevent an unqualified baseline capture or full-task acceptance claim; they do not change the independently verified source boundary and retained terminal-name comparison above.

The v3 wrapper records actual Make parent/descendant PATH prefixes, absent prefix shadows, literal /bin/sh and resolved /usr/bin/dash, and relevant executable paths/hashes. Its legacy environment_capture_mode=make label means environment capture enabled, not that every principal command was Make. Current wrapper/input binding and current proofs are not historical retrobinding. The worker's receipt audit covers 21 earlier receipts/42 observations; the audit's own receipt and pair bring the actual packet to 22/44.

The retained process receipt exits 0 at `2026-10-10T04:49:26Z` with no attributed task gate child and nine disclosed process races/unreadable entries. It excludes checker ancestors and leaves the foreign bungee-lang Make untouched; it does not prove global process absence. The conductor reports the sole Go lane released after all worker handles became terminal. This reviewer ran no replacement process/tool/environment probe.

The immediate integrated baseline remains combined-70, ROOT70 HEAD `b4047ac5451693e01b0fc220b58650b600fddcca`, fixed20 seal `6adbc92107cc4ec2df71171b12da5ca02e780d975d4bc3652c4603999da5998b`, with 673 ordinary named outcomes (421 PASS, 240 FAIL, 12 SKIP) and unchanged full50 lint RED. Those are prior actual outcomes, not a task-71 aggregate forecast. This report does not certify an unexecuted future combined-71 wrapper or candidate ordinary/full50 comparison. The conductor still owns integration, its frozen current-candidate ordinary and full-lint packet, integrated review, checkpoint authorization, lifecycle, and acceptance. Native fn-128/fn-149 qualification remains deferred and unverified; stock linux/arm64 evidence is bounded developmental host-source coverage, not patched-runtime, replay/soak, process transport, adapter, or native qualification.

## Disposition

The exact three-line source change is correct and adequately covered for the admitted scripted completion slice. Critical 0, important 0, minor 0 new source findings. The worker's focused results and unchanged control/lint comparisons are supported by its frozen packet within the explicit historical-capture, reused-context, and host-source limits above. All task/source acceptance remains OPEN unless the conductor separately authorizes it after its required integrated checks; no formal Done or SHIP is asserted.
