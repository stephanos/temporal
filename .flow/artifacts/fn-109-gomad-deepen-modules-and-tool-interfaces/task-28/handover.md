# Task 28 campaign source progress

Campaign cleanup now reports all thirteen mapped Close errors at their original
lifetimes. Both seed failure-policy switches use direct first/budget predicates.
The final unfiltered package lint removes the fifteen mapped findings and retains
the two original invariant panic findings. SOURCE_PROGRESS_ONLY applies.

Base is `d7c6695cff81a1160ee482cb82b62cf592f4f199` on branch `gomad`.
The worker changed seven of the eight admitted source/test paths.
`controller_completion_test.go` remains byte-identical to BASE. Both invariant
panic bodies and the original rejection tests remain unchanged. Root owns the
fresh independent review, Git/index/commit, Flow lifecycle and MILESTONES.
Worker commits and commit range are empty.

Tier is session (`jev-unavailable(no_key)`). The requested implementer was
`gpt-6.1-sol` at high. Actual execution metadata is unavailable.
Formal implementation review remains conductor-deferred while qualification
is red. This handover supplies no formal SHIP verdict.

## Preserved behavior

Constructor validation retains source/parallelism, zero budget, then unknown
policy precedence. Construction restores all initial counters. All continues
admission with retained failures; budget stops admission at or above its distinct
failure threshold. Completion retains accounting before the stopped guard,
first-only cancellation, budget drain and All's no-action behavior. Existing
whole-statistics vectors and zero-mutation invariant checks retain their values.

Nine deferred journal closes use inline closures with `t.Error` at the original
registration points. `publishMergeShard` closes before returning to its caller.
The torn-tail setup closes once on Write failure, reports a nonnil close error
independently, then fails with the original Write error. The successful setup
path and literal torn fragment remain unchanged.

`OpenCampaign`, `RecordPlan` and `ReadResumePlan` use named error results with
conditional deferred `errors.Join(primary, closeErr)`. A nil Close error leaves
the original error object, text and returned data untouched. Function types,
validation statements, publication/config-update order and canonical fixtures
remain unchanged. Existing-plan refusal retains its direct error; changed-segment
rejection retains the original `*IntegrityError` object; prepared-target errors
retain their two original causes in order and return the original zero data.

Before production edits, new literal characterization covered All and resumed
budget admission below/at/above the threshold, complete initial statistics,
existing-plan refusal, changed target identity and changed published segment.
The initial target-error assertion omitted the existing target-size metadata
cause. Source inspection of `hashValidatedFile` confirmed the cause; the corrected
literal passed before production edits. [The initial failure](characterization.log)
is an assertion calibration, and supplies no runtime RED claim.

The TDD source-policy regression is the actual pinned analyzer's fifteen mapped
diagnostics. Behavioral preservation tests pass on the original production
source. No source-grep test, fake close-error seam or relaxed expectation was added.
The first if/else-if implementation emitted two QF1003 findings in
[the intermediate lint output](final-lint.log). Direct compound predicates
remove those introduced diagnostics without a rule or configuration change.

## Verification

[Evidence](evidence.json) names raw logs and per-command receipts with exact argv,
cwd, environment, start/end, elapsed time, actual command exit, tool/config
hashes and campaign source hashes before/after. The capture wrapper records a
failed child command in its receipt even though the wrapper itself exits zero.
All recorded gate source snapshots are stable. Baseline campaign hashes match
BASE; all final campaign and boundary receipts match the handed-over source.

| Gate | Baseline | Final |
| --- | --- | --- |
| Entire ordinary campaign package | exit 0, 2.891s | exit 0, 6.735s |
| Focused controller/resume/cancel/plan/publication/torn-tail/rejection tests | exit 0, 0.982s, 45 top-level tests | exit 0, 4.925s, 46 top-level tests |
| New literal characterization before production | exit 0, 0.492s | covered by final focused/package gates |
| Actual unfiltered pinned campaign lint | exit 1, 17 findings | exit 1, 2 findings |
| Pinned campaign errortype | exit 0, no findings | exit 0, no findings |

The exact lint delta is thirteen errcheck and two exhaustive findings removed.
Both remaining forbidigo findings have their original panic messages and bodies.
The final affected-package lint remains red. The [baseline output](baseline-lint.log)
and [final output](final-predicates-lint.log) retain every diagnostic; evidence
lists their exact paths, lines, messages and rules.

The frozen root boundary gate passes in 5.475s. It runs
`TestPackageArchitecture`, `TestPublicPackagesDoNotExportTypeAliases`, and
`TestRunnerRequestsCompileInExternalModule`. `make validate` passes in 4.669s,
checking version, protocols, boundary/compiler fixtures, patch/overlay inputs,
script ownership, compatibility packs and qualification manifest drift.
Its receipt precedes only the final predicate refinement, which changes no
generator input. `git diff --check` and pinned gofmt pass.

Before editing production, the worker inspected Makefile `VERSION_INPUTS`,
`BOUNDARY_INPUTS` and `COMPATIBILITY_INPUTS`. The protected digest of 1,007 tracked
noncampaign Gomad/module/config files is identical before and after. This covers
generator consumers, runtime/overlay/protocol/toolchain/build-key inputs and
public consumers. The final product diff contains only the seven admitted
campaign paths. No pin, dependency, lint config or generated source changed;
validation used check modes and no download ran.

## Qualification limits

Pinned Go reports `go1.27.1 linux/arm64`; this evidence is developmental.
The patched `.toolchain/bin/go` is absent. Active pinned `os/root.go:96`
delegates Close to `root_openat.go:32`, whose Unix implementation always returns
nil while discarding the syscall close result. [Protected source hashes](protected-after.json)
bind that inspection to the active Go source. No real nonnil-root-close injection
exists in these functions. Conditional join order is inspected source behavior;
this task demonstrates no executed nonnil-root-close path.

Original R16/R18/R19, task3/predecessors, task21, first-baseline fixed-identity
evidence, full/formal gates and qualified darwin/arm64 and linux/amd64 execution
remain open wherever unproved. The two invariant findings remain open without
suppression or a completion-error redesign.

The [task27 handover](../task-27/handover.md) and
[R18 disclosure supplement](../task-21/preservation-disclosure-2026-10-04.md)
retain their historical limitations unchanged. Task25's
[root-fast receipt](../task-25/root-fast.receipt.json) retains the historical
419 nested findings. This task did not rerun full rootfast and supplies no new
whole-scope count. Root must review and commit this verified source progress
before admitting another writer.

Owned delegates are zero. Owned live command handles and pending commands are
zero. Worker Git writes, Flow lifecycle writes, bridges, pushes and reviews are
zero. The independent root-owned artifact scout is outside this worker scope.
