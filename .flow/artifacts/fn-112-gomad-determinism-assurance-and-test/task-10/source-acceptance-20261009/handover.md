# Task 10 source progress

Task `fn-112-gomad-determinism-assurance-and-test.10` remains `in_progress`.
Bounded shared-documentation corrections are ready for root's source review.
Source acceptance is not complete: unfiltered affected adapter lint remains
RED with 63 errcheck diagnostics, including four in the soak adapter outside
this task's admitted Touches. Root declined a write-scope expansion.

Tier: session (jev-unavailable(no_key)); optional judge no_key retained by root;
requested model is not actual-model telemetry.

stage: impl-review - skipped(policy: root owns review/lifecycle; required source lint is red)

No worker review verdict, Flow Done, staging, commit, push, PR, CI dispatch or
native revival occurred. All command handles are terminal. The workspace is
`/Users/stephan/Workspace/skunkworks/gomad/temporal`; admission and current HEAD
are both `15f56644664f3d3749bab2387aa97936a1cac6dd`, with an empty commit range.
`evidence.json` retains every actual command, exit, elapsed time, environment,
raw-log hash, source digest, tool/script hash, and current changed-path binding.

## Source changes and preservation

Worker-owned changes are exactly:

- `tools/gomad3/CLI.md`: `--batches` replaces the maximum and may reduce the
  minimum, rather than only capping the existing maximum.
- `tools/gomad3/README.md`: name deferred D12 and native soak owners.
- `tools/gomad3/TUTORIAL.md`: quote a cumulative per-platform, per-cohort clean
  repetition bound, not a platform-wide count.
- `tools/gomad3integration/README.md`: retain Linux intermittent expectations,
  distinguish historical clean results from current qualification, describe
  actual smoke completion/classification/replay checks, state 20% sizing
  headroom and deferred owners, and update the generated selection count to
  149 without claiming 149 executed passes.
- `tools/gomad3integration/qualification/soak.json`: clarify sizing prose and
  replace obsolete fn-105 R12 removal ownership with deferred fn-128.2.
- `.github/workflows/gomad3.yml`: the directly related D12-removal comment only.

The root's existing task record/body and MILESTONES status edits remain intact;
the fn-113/fn-114 completed-section cleanup and historical anchors are preserved.
Both protected untracked `.turbo` files were untouched. No tracked Go file
changed from admission. Artifact-only Go code is a filesystem diagnostic
control, not shipped runtime implementation.

Manifest bytes are **not** identical. Before SHA-256:
`373bbeba5aee579de261ae02793f20ceb67939649ef3e505c4b3265b6e61262b`;
after: `6399fe11ec2db65a016c5a7b3713da52451e25b9be90fafd4de5148d2f401516`.
`soak.go:280,295` hashes the complete raw manifest into report and ledger-run
identity, so these prose corrections participate in that identity. Execution
selection, minimum/maximum rounds, repetitions, budget, timeouts, loader count,
informational-platform keys and execution cohort identity are unchanged.
The executable workflow source is unchanged when comment-only lines are
removed. `sealed-source-audit.json` binds these before/after assertions.

## Retained requirements and evidence

| Requirement | Delivered source and current portable evidence | Remaining limit |
|---|---|---|
| R6 named scheduled selection | Five smoke/probe workloads, seeds 11/17, guarded frontend probe, repeat32, minimum2/maximum4, load2, budget120m and timeout55m; both workflow matrices match, schedule remains Monday09:17UTC and jobs180m. `TestSelectedManifestIsValid`, integration manifest/smoke-equivalence tests and sealed workflow audit pass. | No scheduled/native workload execution or measured bound claimed. |
| R6 cross-batch comparisons and retention | `TestCohortBatchSequenceAAThenBBIsADivergence` exercises exact AA/BB ledger observations; `TestRunRetainsBothTracesAndDifferOutputForACrossBatchDivergence` executes equal AAA then equal BBB qualification batches through injected qualification reports and real diagnostic trace/differ retention. Earlier-run baseline and within-batch localized-pair tests also pass. Root accepted the nonredundant AAA/BBB retention equivalence. | Injected qualification is portable control evidence, never native soak evidence. |
| R6 identity and cumulative counts | `TestNewToolchainIdentityStartsANewCohort`, `TestCumulativeCountsAccumulateAcrossRetainedRuns`, earlier-retained-run comparison and tampered-cohort rejection pass. Cohort key includes workload, seed, platform and execution identity; toolchain changes roll identity. | Native retained ledger/report owners remain deferred. |
| R6 non-pass outcomes | Overflow/infrastructure/divergence classification, failed baseline retention, exhausted budget, target failure classification, informational divergence and overflow-repetition exclusion tests pass. Final soak/set runs cover 16+43 top-level tests, 104 tests/subtests, no skips or failures. | Original FUSE failures remain retained; local-temp success does not establish universal FUSE correctness. |
| R7 honest closure/exclusions | README and SPEC retain closure without compiled `-gomadguard`, host netpoll, SIGPROF/CPU, enabled block/mutex profiling and nonvirtualized NumCPU exclusions; timer_ties/runq_shuffle seeded positive controls remain documented. Inventory negative controls pass two stock-host tests. | Actual patched runtime inventory/reference and supported-native conformance are not newly executed here. |
| R11 shared guide facts | Diagnostic flag/differ, draw inventory, seeded fixtures, declared model differences, gate behavior and bounds are present. Corrections match current implementations and Linux manifests/checker. Current independent inventory passes 39 help entries and 37 local guide links; integration README is also directly scanned and has no Markdown links. | Historical fn-111 procedure remains red; obsolete checker findings are retained, not rewritten into green. |
| Source preservation/gates | Vet and standalone errortype pass affected soak/set/adapter packages. Architecture/public-signature/purity and TestHostPackageVet pass both supported source sets plus developmental host. Check-only generated validate, package format check and whitespace check pass. Soak/set configured lint passes independently. | Combined adapter lint is RED63; fast lint's zero exit has zero changed-package coverage. |

Source requirements are audited against the current task/parent, all predecessor
Done summaries, historical task-10 handover, current admission and native transfer
manifest. Missing native receipts are not used to waive any portable/source gate.

## Baseline and filesystem investigation

The first subprocess cwd-only attempt observed the wrong module, selected zero
tests and failed setup: `baseline-soak-set` is **inconclusive**, not a product
baseline. The wrapper was corrected to an explicit `cd` plus physical cwd assert.
`baseline-soak-set-corrected` then ran actual packages pre-edit and exited1 on
the original FUSE temp filesystem, with TWO failed leaves:

- `TestPruneQualifiedCampaignsRemovesASharedTargetOnlyWithItsLastArtifact`:
  `prune_test.go:72: the target of the last pruned Campaign survived: <nil>`.
- `TestRunCountsRetainedRunnerFailureAsInfrastructure`: `sync replacement:
  sync /Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/
  TestRunCountsRetainedRunnerFailureAsInfrastructure3174432631/001/
  .safefile-1412649978: directory not empty`, with an incomplete set report.

Original temp is `fuseblk`, fs-id0. Root admitted one unchanged-source run with
TMPDIR/GOTMPDIR changed to private `/tmp/fn11210-portable.6ZZCZ0Ix`, `overlayfs`,
fs-id `e6312165da52bad5`; that pre-edit run passed, and the final source run also
passed. Other explicit Go/module/cache/proxy settings stayed fixed in receipts.

The stdlib hardlink/removal control retained fixture
`/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/fn11210-link-control-3054527208/shared`
(dev58, inode416060929), whose Nlink stayed3 after both linked directories were
removed. The same control on overlay retained
`/tmp/fn11210-portable.6ZZCZ0Ix/fn11210-link-control-791919736/shared`
(dev30, inode5637248), Nlink3→2→1. This independently reproduces a metadata
input to pruning's guard; it is not proof of the separate safefile-sync cause
or of all FUSE defects being resolved. Original Go-test tempdirs were cleaned
by the test runner; their original errors/paths are retained in raw logs.
No product guard, assertion or filesystem implementation was edited.

## Gates and remaining routing

Named current passes: `final-soak-set`, `portable-integration-manifests`,
`soak-command-invalid-input`, `affected-vet`, `standalone-errortype`,
`final-architecture-source-sets` (nine test/subtest passes), `final-validate`,
`draw-inventory-negative-controls`, `qualification-package-lint`, `format-check`
and `sealed-source-inventory`. The fast-lint Make target actually ran against
the admission base and printed `No changed Go packages to lint.`; exit0 is
explicitly **zero coverage**, not adapter lint acceptance.

`configured-lint` uses unchanged project configuration and no revision filter;
it exits1 with63 errcheck diagnostics, all in `cmd/gomadtool`. Four belong to
the soak adapter's unchecked diagnostic writes at `soak.go:36,46,55,62`.
That source is semantically task10 but outside its Touches; root declined
expansion. The other59 belong to preexisting maintainer command adapters,
including compatibility/upgrade/protocol/script/diagnostic commands; their exact
file counts are in `evidence.json` under `lint_red.by_file`. The existing
fn-109.51 shared packet retains the same scoped63 count and separate208
original-base diagnostics; count equality does not substitute for current
source bindings. Root owns prerequisite routing. No exclusions were added,
and independent soak/set green cannot turn the combined command green.
Unchanged full cmd/gomadtool Git-fixture failure was not blindly rerun; focused
soak argument coverage is not a full-package pass.

The retained fn-111 procedure ran before/after in separate output directories,
both exit1 with54 diagnostics. Its companion resolves all local links and
finds no missing glossary terms; its frozen baseline123 identifiers differs
from current134 by eleven added contracts and zero removed contracts.
The old procedure does not recognize preserved HTML milestone anchors,
splits indexed `compatibility-pack refresh` incorrectly, omits refresh from
its hardcoded PACK_ACTIONS, checks moved source paths/old clock semantics,
treats qualified-context shorthand/metavariables as literal commands, expects
removed completed milestone sections, and tries an unavailable pinned native
clock probe. Actual current refresh help/parser accepts impact-report and
pin-impact format=json. No valid docs or historical checker were changed merely
to manufacture a pass. The new bounded inventory is a separate ordinary source
check, not a retroactive green for the historical procedure. Its initial
artifact-only red omitted positional `--provenance`; correction is justified
by the actual parser check `arguments[1] != "--provenance"`, and all versions'
observations are retained.

The first architecture run passed while a manifest-prose edit changed its
aggregate source digest; that mixed-doc receipt is retained and not sealed as
the final frozen check. `final-architecture-source-sets` ran again frozen and
passed. A final integration README completion-rule clarification after that
run touched no Go bytes and was subsequently checked against the exact smoke
workflow in the sealed source audit. Check-only generation uses the Makefile's
explicit `.toolchain/generator-cache` override, as shown in its raw log.

No current actionlint/yamlfmt executable was available; no fresh workflow lint
pass is claimed. This diff changes only one workflow comment; executable
workflow byte preservation and whitespace checks are retained. Native full
test-host/soak, workload reports and measured bounds stay deferred to fn-149.4
and fn-128.5/.7; D12 removal policy stays with deferred fn-128.2. None was run
or represented as green. Formal source review and completion await root's
decision and the required red-gate resolution.
