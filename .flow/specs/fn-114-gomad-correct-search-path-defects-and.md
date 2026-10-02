# Gomad: correct search-path defects and remove wasted work

**Plan date:** 2026-10-01

## Goal & Context

A developer who runs `explore`, `--guide`, choice exploration, or `minimize`
gets results that are bound to the right identity and spends compute on new
executions. Today four defects make those paths wrong in specific cases and six
make them waste time, trace bytes, or disk.

The [2026-10-01 assessment of GOMAD_CMP.md](../../.plans/GOMAD_CMP.md) and its
two code studies
([schedule search](../../docs/research/gomad/2026-10-01-feasibility-schedule-search.md),
[workload and diagnosis](../../docs/research/gomad/2026-10-01-feasibility-workload-diagnosis.md))
record the findings. They came from reading source and retained reports. Nothing
was executed, so each finding is re-anchored and reproduced before its fix.

This spec owns defects in shipped behavior. New capabilities from the same
assessment (yield points, scheduling policies, a per-execution input channel,
an application assertion API, causality analysis) stay research candidates in
GOMAD_CMP.md.

### Findings this spec owns

| ID | Kind | Finding | Evidence | Status |
| --- | --- | --- | --- | --- |
| C1 | Correctness | The guided corpus identity omits the target environment and clock-tick policy. Once Runner-managed targets receive `--env` entries (fn-105 task 30), one corpus can hold cases recorded under different environments or tick policies | `runner/internal/corpus/model.go` `Identity` and `targetProjection` bind target, argv, tags, adapters, packs, toolchain, and boundary only | confirmed — environment and tick inputs still omitted |
| C2 | Correctness | Goroutines created with no identified parent take a process-wide creation counter as identity. `time.AfterFunc` callbacks, including every context deadline, are created from the scheduler, so their identity depends on creation order and therefore on the schedule | `gomadChoiceAssignGoroutineIdentity` else-branch in the runtime overlay | changed — callback identities swap under schedule changes; valid same-seed swapped prefixes succeed |
| C3 | Correctness | A forced prefix that diverges in a choice-exploration candidate ends the whole campaign as a `HostError` and publishes no candidate execution records or typed divergence evidence | `execution/process_unix.go` returns a typed choice divergence; `runner.go` preserves it and `choice_exploration_campaign.go` returns from completion-error handling before processing outcomes | changed — symptom reproduced; executor-error path, not runner-domain branch |
| C4 | Correctness | `exec` provenance validation rejects `-race`, cgo, non-exe build modes, and external linking, and has no check for coverage instrumentation, so an unqualified instrumentation profile can pass | `validateDeterministicBuildInfo` in `target/target.go` | confirmed — no coverage-instrumentation rejection |
| E1 | Inefficiency | A guided campaign gives up to three quarters of its seeds to corpus cases. The corpus identity binds the exact target, so those seeds reproduce records the corpus already holds | `mixGuidedSelection` in `runner/seeds.go` | confirmed — selection repeats corpus seeds; consequence remains source inference |
| E2 | Inefficiency | Every artifact embeds the full target binary. `./tests` binaries are 155 to 179 MB, the representative qualification set retains about 11 GiB, and the 1 GiB corpus holds about six `./tests` cases | `artifact/publication.go` target payload; historical binary sizes in `.plans/GOMAD_CMP.md:85-86`, total in `tools/gomad3/README.md:334` | confirmed — full target payload; sizes and storage totals are historical |
| E3 | Inefficiency | The select hook records one decision per poll-order step of every multi-case `select`, ready or not. In `TestSignalWorkflowTestSuiteChasm` seed 11, 26,865 of 57,801 decisions are select-poll. Poll order can change behavior only when at least two cases are ready | patch hunk in `selectgo`; retained D14 qualification report | confirmed — all seven fixture shapes emit poll decisions; unreduced outcomes retained, suppression soundness remains task 12 |
| E4 | Inefficiency | System goroutines are recorded and explorable alternatives. The retained D21 control probe reports peak goroutines 2 and 26 branching decisions; its source does not deliberately start two user goroutines | `gomadChoiceRunqIndex`; retained D21 control report | changed — unfiltered alternatives and historical 26 confirmed; two-user premise unsupported |
| E5 | Inefficiency | Choice exploration skips every decision past `--max-choice-depth` counted from ordinal 0. The boot-only cluster probe records 4,562 choice records, so on a functional suite the strategy permutes bootstrap only | `expandCandidate` in `runner/internal/exploration/choice/engine.go` | confirmed — absolute decision ordinal bounds expansion |
| E6 | Inefficiency | The minimizer holds its sealed, self-validating state in memory and works in a temporary directory that `close` deletes. An interrupted run repeats every attempt | `runner/minimize_operation.go` | confirmed — state remains in memory and temporary workspace is deleted |

### Relationship to existing work

- fn-105 D12 keeps the linux/amd64 replay-divergence fix. fn-105 D15 keeps trace
  capacity; E3 and E4 report the bytes they save and change no capacity bound.
- fn-105 task 30 delivered the target environment and is done, so C1 is
  reachable today.
- fn-110 owns patch relocation and size. C2, E3, and E4 edit the runtime, so
  their hunks and source-set changes are coordinated with it.
- fn-112 adds a diagnostic trace and re-audits draw sites. Runtime and
  choice-wire edits from both specs are batched into as few toolchain
  identities as the work allows.
- fn-109 owns interface migrations. Changes to corpus, artifact, and minimizer
  interfaces follow its verified migrations where they overlap.
- The roadmap's BUG-5 item covers minimizer resume and typed shrinking. This
  spec delivers resume only.

## Architecture & Data Models

### Corpus identity (C1)

The corpus identity gains the recorded target environment and the clock-tick
policy, the same values that already enter Campaign and Artifact identity. A
corpus opened with a different environment or policy fails with the existing
changed-identity error. The corpus schema version rises; an existing corpus is
rejected visibly and never reinterpreted.

### Goroutine identity for runtime-created goroutines (C2)

A goroutine started for a timer callback derives its identity from the timer's
creator: the creating goroutine's identity, that goroutine's child ordinal at
creation, and the creation site. The timer carries those values from creation to
the callback start. Other parentless creations are inventoried and each is
either given a schedule-independent derivation or listed as a declared
exception with its reason. The changed derivation takes new versioned identity
labels; the labels are the scheme version, and no separate constant exists.

Task 2 reproduced callback identity swaps on the unchanged toolchain. Its seed-6
parent maps A/B to parentless ordinals 3/2; valid same-seed prefixes at decision
3/rank 2 and decision 4/rank 1 map them to 2/3 and succeed. All 16 one-decision
alternatives of that recorded parent succeed. This narrows the predicted
alternative-set-divergence failure: `BuildRankPrefix` truncates the suffix after
the changed choice. A cross-seed full-prefix experiment diverges at a select site,
but does not establish failure of supported seed-bound replay. Task 5 still owns
stable identities; successful same-seed prefix execution is behavior to preserve.
The retained [task 2 evidence](../artifacts/fn-114-gomad-correct-search-path-defects-and/runtime-reproduction/final-approved/search-reproduction.json)
binds these observations to the pre-edit toolchain and fixture source.

### Diverging exploration candidates (C3)

A candidate whose forced prefix diverges becomes a typed candidate result with
retained evidence: the candidate identity, the divergence ordinal and reason,
and the expected and observed records the terminal frame already carries. The
round commits, the campaign reports the divergence under the existing
`replay_divergence` classification, and the failure policy decides whether
exploration continues. Divergence of a prefix taken from a deterministic parent
remains a Gomad confidence failure and is never reported as a target outcome.

### Instrumentation in provenance (C4)

`exec` provenance validation rejects a binary whose build information shows
coverage instrumentation, with its own error text, until a coverage profile is
qualified.

### Guided selection (E1)

Guided selection stops scheduling a corpus seed whose retained record already
answers the execution. The reserved unguided share and the snapshot-bound,
deterministic selection stay. Re-running corpus cases remains available as an
explicit regression mode. When the corpus offers nothing new to run, the
campaign runs its requested seeds and reports that guidance selected none.

### Shared target payload (E2)

A Campaign, corpus, and qualification store keeps one content-addressed copy of
each prepared target, and artifacts in that store reference it by the SHA-256
and size they already record. Replay resolves and verifies the reference before
execution. An artifact copied or exported out of its store carries the target
with it, so a standalone artifact stays self-contained. Pruning removes a
target only when no retained artifact references it.

### No-op select decisions (E3)

The work has two parts with separate evidence. First, a fixture and a reduced
versus unreduced comparison establish whether poll order is unobservable when
fewer than two cases are ready, including the blocking path. Second, if it
holds, the runtime records the ready-case count and the explorer never expands
a select-poll decision with fewer than two ready cases. Whether such decisions
are also dropped from the Choice Trace is decided on the measured byte saving
and the replay-validation cost of losing them.

### System goroutines (E4)

The runtime orders runtime-owned goroutines by a fixed rule and offers only
user goroutines as alternatives. The rule is stated in the contract. This
changes schedules, so it is a new choice-controller identity.

### Exploration start (E5)

Choice exploration takes a start ordinal. Decisions before it are forced as
recorded and never expanded, and `--max-choice-depth` counts from it. The start
is part of the controller configuration and Campaign identity. The default
keeps today's behavior.

### Minimizer resume (E6)

`minimize` writes its sealed state and the last accepted artifact under the
output directory after each commit. A resume option reopens that state, checks
the parent artifact identity and budget, and continues without repeating
evaluated attempts.

## API Contracts

- CLI grammar and defaults are unchanged except for three additions: the
  exploration start ordinal, the minimizer resume option, and the explicit
  corpus regression mode.
- Corpus schema, choice-wire or identity-scheme versions, and artifact schema
  rise where their bytes change. Readers reject older or newer forms visibly.
  No existing artifact is rewritten.
- A runtime edit changes the toolchain build key. Retained artifacts keep their
  original identity, and qualification of the candidate records fresh ones.
- Failure classifications keep their names and precedence. C3 adds evidence to
  an existing classification.

## Edge Cases & Constraints

- The [milestone constraints](../../MILESTONES.md#constraints) apply: no policy
  widening, fail-closed boundaries, no test rewriting, and the collector patch
  prohibition. A fix that needs a prohibited file goes to the patch-policy
  owner.
- A finding that re-anchoring refutes closes with its evidence and no code
  change.
- C2, E3, and E4 change schedules or records. Each is qualified on the core,
  smoke, and representative sets, and existing dispositions are not weakened to
  obtain a pass.
- E2 must not break `replay --verify-only`, `resume`, shard merge, or
  qualification pruning. A missing or mismatched shared target fails before
  execution.
- E1 must keep campaign selection independent of completion timing and equal
  across resume.
- E3's reduction is sound only for the cases its fixture covers. Any select
  shape outside them stays expanded.
- linux/amd64 evidence needs a native host. A missing host leaves the affected
  criteria incomplete.

## Acceptance Criteria

- **R1:** Each finding C1 to C4 and E1 to E6 is re-anchored to current file and
  line and marked confirmed, changed, or refuted before its work starts. C2,
  C3, and E3 each have a fixture or test that reproduces the finding on the
  unmodified tree. Errors: a refuted finding closes with its evidence.

- **R2 (C1):** Opening a corpus with a different target environment or
  clock-tick policy fails with the changed-identity error, and a test covers
  both. Errors: a corpus written before the change is rejected by schema
  version.

- **R3 (C2):** In a fixture where two context deadlines fire in either order,
  each callback goroutine's identity is the same in both orders, and a forced
  prefix that swaps them replays without an alternative-set divergence. The
  inventory of remaining parentless creations is checked in with a reason per
  entry. Errors: a new parentless creation site after a Go upgrade fails the
  toolchain tier.

- **R4 (C3):** A choice-exploration campaign with a diverging candidate commits
  its round, retains the candidate's divergence evidence, reports
  `replay_divergence`, and resumes correctly. Errors: the campaign never reports
  the divergence as a target failure or a success.

- **R5 (C4):** `exec` provenance for a coverage-instrumented binary is rejected
  with a specific error, with a negative test.

- **R6 (E1):** A guided campaign over a corpus whose cases all bind the current
  target executes no seed whose record the corpus already holds, keeps its
  unguided share, and selects the same seeds after resume. The regression mode
  re-runs corpus cases on request. The report states how many executions were
  new.

- **R7 (E2):** A store with N artifacts of one target holds one copy of the
  target binary. Replay, verify-only, resume, merge, inspect, and pruning pass
  their existing tests. An exported artifact replays on its own. The
  representative qualification set's retained bytes are reported before and
  after. Errors: a missing or corrupt shared target fails before execution.

- **R8 (E3):** A reduced and an unreduced exploration of a finite fixture agree
  on outcomes and deadlocks, and the explorer expands no select-poll decision
  with fewer than two ready cases. Trace bytes and branching-decision counts
  for the Signal suite are reported before and after. Errors: if the no-op
  claim fails for any covered select shape, that shape stays expanded and the
  result is recorded.

- **R9 (E4):** A program with two user goroutines records only decisions among
  user goroutines, and the contract states the rule for runtime-owned
  goroutines. Decision counts for the control probe are reported before and
  after.

- **R10 (E5):** With a start ordinal at the first test body of a functional
  suite, choice exploration expands decisions at and after that ordinal only,
  and a resumed campaign keeps the same start. The default behavior is
  byte-identical to today for a fixed identity.

- **R11 (E6):** A `minimize` run killed after an accepted reduction resumes
  from its persisted state, repeats no evaluated attempt, and publishes the
  same result as an uninterrupted run. Errors: a changed parent artifact or
  budget is rejected.

- **R12:** The candidate toolchain and Runner pass `make -C tools/gomad3
  validate` and `test`, the core set, the smoke set, and the representative
  Temporal set on darwin/arm64 and linux/amd64, with exact replay where the
  manifests require it. README, CLI, ARCHITECTURE, and the milestones describe
  the delivered behavior. Errors: missing native-host evidence leaves this
  criterion incomplete.

## Boundaries

- No new search policy, yield point, input channel, assertion API, or oracle.
- No change to trace capacity (D15) or to the linux divergence fix (D12).
- No typed scenario shrinking; BUG-5 keeps it.
- No removal of guidance, choice exploration, or the minimizer.
- Documentation inconsistencies found by the same assessment (README statements
  on timer ties and ready-select ranks, the roadmap's search-evidence claim, the
  unreachable `gomadChoiceRunnextSeeded` hunk) are outside this spec.

## Decision Context

These ten items are grouped because they share one cost: C2, E3, and E4 each
change the toolchain identity and force requalification, and fn-110 and fn-112
edit the same runtime files. One coordinated revision is cheaper than three.

The Runner-side items (C1, C3, C4, E1, E5, E6) need no toolchain change and can
land first. E2 changes the artifact store and is ordered after fn-109's
Artifact-handle work if that lands first.

E3 is split into a soundness check and a change because the claim that poll
order is unobservable with fewer than two ready cases rests on reading
upstream `selectgo`. The reduction ships only for the shapes the check covers.

## Planning decisions (2026-10-01)

The planning pass re-read each finding against current source. All ten still
hold as stated. Three evidence references were corrected: the binary sizes are
in the assessment, the identity scheme has no version constant, and the D14 and
D21 reports are cited by the schedule-search study without a path. Task 1
repeats the check at its start commit and resolves the report paths.

**Order.** Task 1 gates every other task. The work then runs as two serial
chains that share no source file, and joins for the last three tasks:

- Runner chain: tasks 3 (C1, C4), 4 (C3), 7 (E1), 6 (E5), 8 (E6), 9 and 10 (E2).
  The user approved promoting guided seed deduplication on 2026-10-02: task 3
  supplies its identity prerequisite, while task 6 was only a shared-file ordering dependency.
- Runtime chain: tasks 2 (C2 and E3 reproductions), 5 (C2), 11 (E3 recording).
- Joined: tasks 12 (E3 rule), 13 (E4), 14 (qualification and docs).

Several dependencies inside a chain express a shared write surface and no
logical need, so that the whole spec can run serially in one working tree. Each
task says which kind its dependencies are. Correctness findings come before
efficiency findings in both chains.

**Toolchain identity.** Tasks 5, 11, and 13 each change the build key. They are
tested per task on the toolchain tiers and qualified once, in task 14, as one
candidate. If a runtime task of fn-110, fn-112, or fn-109 lands in the same
period, task 14 qualifies the combined candidate.

**C3 failure policy.** A divergence stops the campaign under `first` and does
not stop it under `budget` or `all`. It does not consume the distinct
failure-signature budget, because it is not a target failure.

**E1 predicate.** A corpus entry answers an execution when its identity equals
the campaign's corpus identity and its replay result is verified and matching.
Answered seeds are not executed in the default mode: they leave the guided
share, the unguided pool, and the requested selection. The campaign runs the
requested selection minus the answered seeds, substitutes nothing, and reports
how many were left out. A fully answered selection executes nothing and says
so.

**E2 form.** Task 9 chooses the sharing form. The recommended form is a
content-addressed pool with each artifact's target a hard link to it. The pool
belongs to the artifacts root a command was given, because the publication
stores in use today are per-kind and per-round directories and are narrower
than the sharing boundary; the corpus and the minimizer output root each own
one. The target keeps the mode it is published with today. With this form
the manifest, the artifact schema, and the readers do not change, retained
artifacts stay readable, and a plain recursive copy is the export. Rejected as
the default: a manifest reference with no target in the artifact directory,
because it needs a new record form, a schema decision for retained artifacts,
store-level resolution in a reader confined to the artifact directory, and an
export step the CLI does not have.

**E3 recording point.** Poll order is drawn before the channels are locked, so
readiness is recorded with the select result and carried onto the select-poll
decisions when the replay plan is projected. The record also carries shape
evidence (case count, default, and flags for nil, timer, closed, and repeated
channels), and the explorer's eligibility rule is an explicit list of shapes
that passed the soundness check. A ready count alone cannot tell a proven shape
from an unproven one. Dropping no-op decisions from the
trace would need a different recording point; task 12 decides on the measured
numbers and implements nothing.

**E6 exclusion.** `minimize` holds an exclusive lock on its output workspace
from before the state check to the end of final validation, the same host lock
campaign resume uses.

**Docs.** CLI usage text changes with each flag. README, CLI guide, SPEC, and
ARCHITECTURE prose is written once in task 14, except the scheduling rule that
R9 requires, which task 13 writes.

**Cross-spec ordering.** No spec-level dependency is recorded. fn-110 tasks 2
to 4, fn-112 tasks 3 and 5, and fn-109 tasks 2, 5, 6, 12, and 13 edit files this
spec edits and are all open. Each affected task checks their state first and
rebases onto whichever landed.

## Open Questions

1. C2: for a timer created on one goroutine and reset on another, task 5 keeps
   the creator. Whether collector worker creation is deterministic enough to
   leave as a declared exception is decided by the inventory.
2. E3: whether a select-poll decision and its select result are always adjacent
   in the trace is not established. Task 11 tests it and does not rely on it.
3. E5: `inspect --choices` is the intended way to find the ordinal of the first
   test body. If its numbering cannot match the replay plan's decision ordinals,
   the finding needs a marker, which is outside this spec.
4. E1: with the corpus bound to the exact target, default guidance may select no
   seed for an unchanged target. Whether guidance should then draw on corpora of
   other identities is a research question in the assessment.
5. Who owns the shared toolchain identity when fn-110, fn-112, and this spec
   land runtime edits together is undecided (fn-112 Open Question 2).

## Quick commands

```bash
go -C tools/gomad3 test -tags test_dep ./runner/... ./artifact/... ./target/... ./cmd/gomad/...
make -C tools/gomad3 validate test-toolchain test-runtime overlay-test
make -C tools/gomad3 core-qualification
make gomad3-smoke-qualification
```

## Early proof point

Task fn-114-gomad-correct-search-path-defects-and.1 validates the core approach
(the ten findings still hold at the start commit, and C3 reproduces in a Runner
test). If a finding is refuted, its task closes with the evidence and the
remaining order is re-checked before continuing with
fn-114-gomad-correct-search-path-defects-and.2 and later tasks.


## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Each finding C1 to C4 and E1 to E6 is re-anchored to current file and line and marked confirmed, changed, or refuted before its work starts. C2, C3, and E3 each have a fixture or test that reproduces the finding on the unmodified tree. Errors: a refuted finding closes with its evidence. | fn-114-gomad-correct-search-path-defects-and.1, fn-114-gomad-correct-search-path-defects-and.2 | — |
| R2 | (C1): Opening a corpus with a different target environment or clock-tick policy fails with the changed-identity error, and a test covers both. Errors: a corpus written before the change is rejected by schema version. | fn-114-gomad-correct-search-path-defects-and.3 | — |
| R3 | (C2): In a fixture where two context deadlines fire in either order, each callback goroutine's identity is the same in both orders, and a forced prefix that swaps them replays without an alternative-set divergence. The inventory of remaining parentless creations is checked in with a reason per entry. Errors: a new parentless creation site after a Go upgrade fails the toolchain tier. | fn-114-gomad-correct-search-path-defects-and.5 | — |
| R4 | (C3): A choice-exploration campaign with a diverging candidate commits its round, retains the candidate's divergence evidence, reports `replay_divergence`, and resumes correctly. Errors: the campaign never reports the divergence as a target failure or a success. | fn-114-gomad-correct-search-path-defects-and.4 | — |
| R5 | (C4): `exec` provenance for a coverage-instrumented binary is rejected with a specific error, with a negative test. | fn-114-gomad-correct-search-path-defects-and.3 | — |
| R6 | (E1): A guided campaign over a corpus whose cases all bind the current target executes no seed whose record the corpus already holds, keeps its unguided share, and selects the same seeds after resume. The regression mode re-runs corpus cases on request. The report states how many executions were new. | fn-114-gomad-correct-search-path-defects-and.7 | — |
| R7 | (E2): A store with N artifacts of one target holds one copy of the target binary. Replay, verify-only, resume, merge, inspect, and pruning pass their existing tests. An exported artifact replays on its own. The representative qualification set's retained bytes are reported before and after. Errors: a missing or corrupt shared target fails before execution. | fn-114-gomad-correct-search-path-defects-and.10, fn-114-gomad-correct-search-path-defects-and.9 | — |
| R8 | (E3): A reduced and an unreduced exploration of a finite fixture agree on outcomes and deadlocks, and the explorer expands no select-poll decision with fewer than two ready cases. Trace bytes and branching-decision counts for the Signal suite are reported before and after. Errors: if the no-op claim fails for any covered select shape, that shape stays expanded and the result is recorded. | fn-114-gomad-correct-search-path-defects-and.11, fn-114-gomad-correct-search-path-defects-and.12 | — |
| R9 | (E4): A program with two user goroutines records only decisions among user goroutines, and the contract states the rule for runtime-owned goroutines. Decision counts for the control probe are reported before and after. | fn-114-gomad-correct-search-path-defects-and.13 | — |
| R10 | (E5): With a start ordinal at the first test body of a functional suite, choice exploration expands decisions at and after that ordinal only, and a resumed campaign keeps the same start. The default behavior is byte-identical to today for a fixed identity. | fn-114-gomad-correct-search-path-defects-and.6 | — |
| R11 | (E6): A `minimize` run killed after an accepted reduction resumes from its persisted state, repeats no evaluated attempt, and publishes the same result as an uninterrupted run. Errors: a changed parent artifact or budget is rejected. | fn-114-gomad-correct-search-path-defects-and.8 | — |
| R12 | The candidate toolchain and Runner pass `make -C tools/gomad3 validate` and `test`, the core set, the smoke set, and the representative Temporal set on darwin/arm64 and linux/amd64, with exact replay where the manifests require it. README, CLI, ARCHITECTURE, and the milestones describe the delivered behavior. Errors: missing native-host evidence leaves this criterion incomplete. | fn-114-gomad-correct-search-path-defects-and.14 | — |

