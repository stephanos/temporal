# Restore explicit scripted Unix campaign-mode coverage

Root admits fn-109.73 as a TODO task for the rank-1 Unix mode fixture, advancing R5/R18/R19 and open fn-109.63/fn-112.10 source acceptance. This admission authorizes no worker, execution lane, implementation or completion. Root alone assigns a fresh worker, grants serial execution, integrates, coordinates independent review, commits and controls lifecycle. The task has no predecessor completion dependency; retained RED owners remain in progress, and every shared gate still requires root's explicit serial grant.

Actual BASE is PRIMARY `fd9cfc4026db966a877597a9658a0af589f18aea`. The 39-line `tools/gomad3/runner/runner_mode_unix_test.go` is SHA-256 `ebdc20609fd89c246bf345e0df40f3c126b09765d898770dabe4389518228168`. Follow the retained [remaining-scripted-fixture-survey.md](../../fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/remaining-scripted-fixture-survey.md), SHA-256 `25d2f660a363458d4954d319556d280600dce10d17278fd180c3da42d5c8b1a7`, for the source trace and rank-1 boundary. The authoritative dirty PRIMARY owner spec remains SHA-256 `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c`; historical isolated owner copies supply no waiver.

## Exact boundary

**Touches:** [tools/gomad3/runner/runner_mode_unix_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-73/**]

Add exactly one syntactic assignment using the existing helper in `TestRunEnforcesBatchModesIndependentOfUmask`:

```go
configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
```

Insert after the original `defer syscall.Umask(oldUmask)` and immediately before the existing `exploreWith` call. Preserve the final outer `configDependencies.executor` and `config.Preparer`. Keep their original construction before `syscall.Umask(0o777)`, the entire 0777 umask interval, the original saved umask and deferred restoration, and serial test execution unchanged. Neither moving construction into the umask interval nor adding parallel execution is admitted.

Preserve all seven real filesystem assertions and their existing error handling:

| Existing path | Required permission |
| --- | --- |
| `.` | `0700` |
| `failures` | `0700` |
| `.partial` | `0700` |
| `executions` | `0700` |
| `campaign.json` | `0600` |
| `executions/index.json` | `0600` |
| `executions/00000000000000000000.jsonl` | `0600` |

Keep the Unix build tag, imports, comments, helper bodies, table data, executor/preparer implementation, target metadata and every remaining byte unchanged. Removing exactly that one assignment from the candidate must reconstruct the entire actual BASE file byte-for-byte, not merely a selected body. Preserve the original real `os.Stat` and permission comparisons; no mock filesystem, rewritten assertion, new fixture or weakening of mode enforcement is admitted. This whole-file source check reinstates no encoded-output compatibility requirement.

## Retained RED and observation boundary

The research's combined68 ordinary log records exactly the emitted top-level terminal FAIL `TestRunEnforcesBatchModesIndependentOfUmask`, at raw line 2366, preceded by the unsupported-host preparation diagnostic at 2364. Its log SHA-256 is `3b75cacdac2bcc7016dc5f7a99f924f6b958f0cbc7d58e59afe77980b7b34b1e`. The fixture has no emitted table-child names; synthesize none.

The later [combined69 checkpoint](../combined-69/checkpoint-note.md) binds frozen isolated HEAD `c668243e0ecab6e4080aa7dad0810ccc2cedb08f`, integrated as PRIMARY BASE `fd9cfc4026db966a877597a9658a0af589f18aea`. Its ordinary log SHA-256 is `73ceada3c52c462a03f655c0be752b2c191d7a3925d0f30f624ef48aa0c438d9`. The same selected name still emits FAIL at line 2363, with the unchanged unsupported-linux/arm64 preparation diagnostic at 2361. It is one of the 671 named outcomes unchanged from combined68. The explicit 20-member postcapture seal is SHA-256 `71b566f5f17961cb370c3791a8008116241503b0cc31f4b00f9fa406c06fa0b5`.

Combined69 observed 673 names with 394 PASS, 267 FAIL and 12 SKIP, ordinary exit 1, and original-base aggregate lint RED50, exit 2. These are historical observations, not a future PASS count or acceptance prediction. The retained preparation refusal precedes all seven mode observations and establishes no downstream behavior pass. Tasks70 and other sibling work retain their separate workers/workspaces and source ownership; this draft neither inspects nor depends on an active worker candidate.

## Worker verification after root admission

Before inserting the assignment, retain an actual unchanged-source preparation-stage RED for `^TestRunEnforcesBatchModesIndependentOfUmask$` on the admitted worker candidate. Use pinned Go 1.27.1, `-tags test_dep`, `-count=1`, JSON terminal events and an external wall bound. A missing tool, compilation failure, timeout or skip does not establish meaningful RED. Observe and retain unchanged controls before editing as well.

After insertion, execute the original fixture and observe every unchanged filesystem assertion with actual campaign publication, target copying/verification and journal files. Keep the preparation forwarding, operation-error, original-stage, default/bootstrap, isolated and public-profile controls unchanged:

- `TestPreparationDependenciesForwardRealFixtureInputs`
- `TestPreparationDependenciesOperationErrorsRemainUnchanged`
- `TestPreparationDependenciesFailuresStopAtOriginalStages`
- `TestPreparationDependenciesKeepRealDefaultsAndBootstrapGuard`
- `TestInjectionCharacterizationIsolatedPreparationDependencies`
- `TestInjectionCharacterizationIsolatedExploreRejectsEverySubstitution`
- `TestPortableProfilePublicGuardsRemainFirst` in `./deterministicio`

Retain the existing real preparation error/cancellation and local-phase controls where required by the inherited source gate. Do not weaken unsupported-host/default/bootstrap refusal or use synthetic bootstrap as a real decoded frame. Any newly reached original assertion, permission or validation failure returns to root for separate scope; this one-assignment admission provides no corrective authority beyond its exact boundary.

Use one compact handover referencing immutable raw receipts and existing evidence. Bind the actual workspace and HEAD, complete source and tool manifests before and after each execution, actual BASE blob, authoritative PRIMARY owner spec, task/admission/research inputs, checker and wrapper bytes, actual executable identities, selected and effective environment including actual Go settings, exact argv, numeric exit, UTC start/end, elapsed seconds and raw-log SHA-256. Require source/tool stability during each command. Bind every preservation/comparison/reconciliation checker and the actual Perl executable at execution time; later hashes or seals do not retroactively cure an omitted checker binding. Keep reused-cache and selectively captured environment limitations explicit.

Run the applicable formatting/body and whole-file preservation checks, affected vet and separately reached errortype, configured unfiltered Runner lint and required repository `make lint-code-fast`, with fixes disabled and the actual baseline recorded. Retain current generated validation, architecture/private-public boundaries and both supported source-set static checks, or reconcile retained evidence against exactly unchanged consumed inputs. No speculative native gate substitutes for these still-owned source requirements. Every Go/build/lint/vet/generator command is serialized by root; this draft grants no lane and does not override an existing serial hold. Shell execution must use `login:false`, `env -u BASH_ENV bash -c` and an explicit `cd` to the exact root-assigned workspace.

Root owns the next frozen ordinary comparison against its actual immediately preceding integrated baseline, accounting for separately admitted sibling scopes. Preserve the entire original emitted named set and permit only actual root-admitted outcome changes, including this exact terminal name. Require all current 50 complete original-base lint header/source/caret blocks unchanged, with zero introduced or removed blocks. Counts alone do not satisfy complete-block preservation. Keep aggregate nonzero exits, unfiltered lint RED and integrated errortype reachability explicit; standalone analyzer or fast-lint success supplies no aggregate GREEN claim.

Fresh independent integrated source/evidence review and all still-owned source requirements govern acceptance. A source-progress checkpoint is not formal Done or SHIP. Root minted the actual TODO task before worker dispatch and will record eventual candidate-bound evidence separately from this admission.

## Exclusions

Exclude every other consumer, the survey's diagnostic/bounds candidates, shared helpers, public/default paths, bootstrap decoding, real compiler/process execution, production/storage owners, replay/resume/minimize, guidance, adapter/installation identity, schema/generated inputs, runtime/toolchain and lint-policy changes. Preserve dirty PRIMARY owner/spec files, `MILESTONES.md` including fn-155, and all unrelated user files. Native fn-128/fn-149 remains deferred and unverified. This draft supplies no native qualification, full native test-host pass, replay/soak bound, formal acceptance, task/lane authority, CI, PR or push authority.
