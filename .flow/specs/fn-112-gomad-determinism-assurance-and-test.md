# Gomad: determinism assurance and test strategy

**Plan date:** 2026-10-01

## Goal & Context

Give Gomad a checkable basis for its determinism claim and a test suite shaped
around that claim. Today a same-seed divergence is found only after a workload
emits it, and nothing reports where two runs first differed. Linux D12 has been
open on that basis: about one tier-3 seed-run in 26 diverges at choice ordinals
from 8 to about 85k, far from its cause.

The [2026-10-01 quality assessment](../../MILESTONES.md#quality-assessment-2026-10-01)
records the findings this spec owns. Its runtime findings came from source
reading and were not executed, so each one is re-anchored against the code
before work starts. The assessment's identifiers Q1, Q2, Q6, Q7, and Q8 are
used below.

"Never fails" is not provable while the collector, the allocator, and real
threads stay inside the determinism boundary. The claim this spec establishes
is a runtime invariant that is checked, plus a measured bound from repeated
fresh runs under host load.

### Relationship to existing work

- fn-105 D12 keeps ownership of the linux/amd64 fix and its R12 acceptance.
  This spec supplies the localiser D12 needs. The two D12 candidates from the
  assessment (the one-shot syscall wait in `suspendG`, and uncontrolled linux
  ASLR and `runtime.NumCPU`) are recorded on the D12 task.
- fn-105 D26 owns the forward-clock correction (assessment Q5).
- fn-109 owns interface changes. Where this spec reshapes tests around an
  interface fn-109 migrates, the test change follows the verified migration.
- fn-110 owns patch relocation. Runtime edits here coordinate source-set and
  descriptor changes with it.
- fn-114 corrects search-path defects and edits the runtime choice hooks
  (goroutine identity, select-poll records, system goroutines). Its runtime
  and choice-wire edits and this spec's are batched into as few toolchain
  identities as the work allows.
- Removing or freezing features without a caller (`tools/gomad3sim`, campaign
  plans and shards, guidance, choice exploration, `minimize`,
  `compare-support`) needs an owner decision and is outside this spec.

## Architecture & Data Models

### Divergence localiser (Q1)

An opt-in diagnostic trace records a runtime-state digest at every choice
point. The digest covers the seeded stream draw counters, the allocation count,
the GC cycle and phase, the virtual time, and the run-queue length. A differ
takes two diagnostic traces of fresh same-seed runs and reports the first
ordinal whose digest differs, the fields that differ, and the preceding
agreeing record.

The diagnostic trace is a separate record kind with its own identity. The v2
Choice Trace, the Decision Tape, and evidence digests keep their bytes when
diagnostics are off. Recording must not allocate on the Go heap or draw from
the seeded stream, because either would move the state it measures.

### Stream isolation (Q2)

Audit every draw site of the process-wide seeded stream and classify it as
target-ordered or host-timed. Host-timed sites draw from the M-local stream, as
the four already rerouted do. A runtime check, active in the diagnostic mode,
fails the process when a path marked host-timed draws from the seeded stream.
The classified inventory is checked in beside the clock inventory and fails on
an unclassified draw site after a Go upgrade.

### Determinism soak (Q6)

A scheduled gate runs N fresh repetitions per seed of a named workload
selection under bounded unrelated CPU load, with choice tracing and the
diagnostic trace on. It accepts zero divergences. The report states
repetitions, seeds, load, platform, and toolchain identity, so the bound it
establishes can be quoted. On a divergence the gate retains both diagnostic
traces and the differ output.

### Conformance fixtures (Q7, Q8)

Each uncontrolled or unverified channel gets one seeded black-box fixture with
a positive control, in the existing conformance campaign: netpoll readiness,
SIGPROF, block and mutex profile sampling on linux, `runtime.NumCPU`, timer-tie
draws, and run-queue shuffle draws. A channel that stays outside the contract
gets a fixture that proves it fails closed or a contract sentence that names
it. Closure mode's lack of compiled guards is stated in the contract, and the
soak selection includes one guarded-mode workload.

### Test suite shape

Four layers hold the suite. New tests belong to one of them.

1. Runtime conformance fixtures, one per channel, each with a positive control.
2. The determinism soak.
3. Model conformance: generated operation sequences run against the in-memory
   filesystem and loopback TCP models and against the host OS, comparing
   results and errors within each model's declared differences.
4. End-to-end CLI: `explore`, `replay`, and kill-then-resume through real
   processes, comparing a resumed Campaign with an uninterrupted one.

Tests that assert forwarded fields on injected fakes, repeated rejection
tables, and templated adapter tests are consolidated into table-driven or
generated form. Consolidation keeps every asserted behavior and removes
duplicated scaffolding.

### CLI-discovered corrections

The built-CLI layer exposed three existing defects, tracked as tasks 11–13.
Task 12 repairs native compiler selection in the standard host-test entrypoint.
Task 11 preserves a verified watchdog or cancellation when no I/O terminal was
written, while corrupt or ordinary incomplete results still fail. Task 13
repairs diagnostic replay of valid watchdog artifacts without claiming exact
replay or substituting live host inputs. Their retained reproductions and
acceptance criteria belong to the task records. These corrections advance R3
and R9 (and the host/compiler gate advances R2 and R8).

## API Contracts

CLI grammar and defaults, recorded formats, canonical bytes for fixed
identities, error classifications, and replay compatibility are unchanged when
diagnostics are off. The diagnostic trace is opt-in, carries its own versioned
schema, and is part of execution identity when enabled, like `--choices`.

A runtime edit changes the toolchain build key. Retained artifacts keep their
original identity, and qualification of the candidate records fresh artifacts.

## Edge Cases & Constraints

- The [milestone constraints](../../MILESTONES.md#constraints)
  apply, including the collector patch prohibition. A digest field that needs
  a prohibited file is dropped or goes to the patch-policy owner.
- The diagnostic mode must not perturb the run. A workload that qualifies with
  diagnostics off and diverges with them on is a defect in the diagnostics.
- Diagnostic traces are byte-bounded. Overflow is a Runner failure and cannot
  support a localisation claim.
- The soak selection and N are chosen so one platform's run fits a scheduled
  job. The bound it reports is per platform.
- Model conformance compares only operations the model declares it supports.
  Declared differences from host behavior are listed in one place and each has
  a test.
- Test consolidation cannot lower the count of distinct asserted behaviors.
  Each removed test maps to the table row or generated case that replaces it.
- linux/amd64 evidence needs a native host. A missing host leaves the affected
  criteria incomplete.

## Acceptance Criteria

- **R1:** `gomad3.yml` and the smoke workflow pass on both platforms at the
  spec's starting commit, beginning with `make -C tools/gomad3 validate`.
  Errors: a failing or skipped job is recorded with its cause; the open D12
  allowances stay as they are until fn-105 R12 removes them.

- **R2:** Existing tests that no target runs are executed by a gate: the tagged
  `tools/gomad3sim/*_toolchain_test.go` files, the overlay packages
  `internal/gomadio`, `internal/gomadsim`, `internal/gomadmodelwire`, `os`, and
  `cmd/internal/gomadcap`, and the `choice_replay` fixture. `./toolchain` runs
  in one tier per toolchain kind. Errors: a test that fails once it runs is
  fixed or recorded as a finding; deleting it needs a stated reason.

- **R3:** Every assessment finding this spec owns is re-anchored to current
  file and line, and marked confirmed, changed, or refuted, before its work
  starts. Errors: a refuted finding closes with its evidence.

- **R4:** A diagnostic trace records the runtime-state digest at every choice
  point, and a differ reports the first diverging ordinal and fields for two
  fresh same-seed runs. A fixture with a deliberate host-timed draw
  demonstrates localisation at the injected site. Errors: diagnostics that
  allocate on the Go heap, draw from the seeded stream, or change a workload's
  behavior with diagnostics off fail this criterion.

- **R5:** The seeded-stream draw-site inventory is complete and classified,
  host-timed sites use the M-local stream, and a diagnostic-mode check fails
  the process on a violation. A negative fixture triggers the check. Errors:
  an unclassified site fails the toolchain tier.

- **R6:** A scheduled soak gate on each platform reports zero divergences over
  a stated N fresh repetitions per seed under load, and retains both traces
  and the differ output on a divergence. Errors: on linux/amd64 the gate stays
  informational until fn-105 R12 closes D12, and says so in its report.

- **R7:** Each channel named under conformance fixtures has a seeded fixture
  with a positive control or a contract sentence that places it outside the
  contract. README and SPEC state that closure mode compiles no guards.

- **R8:** Generated-sequence model conformance tests compare the filesystem
  and loopback TCP models with the host OS on both platforms, with every
  declared difference listed and tested. Errors: an undeclared difference is
  a defect in the model or a new declared difference with a reason.

- **R9:** End-to-end tests drive `explore` and `replay` through the built CLI,
  and kill a real coordinator mid-campaign, resume it, and compare the result
  with an uninterrupted run of the same plan.

- **R10:** The change-detector tests named in the assessment are consolidated,
  with a retained mapping from each removed test to its replacement and no
  loss of asserted behavior. Test code lines and test count are reported
  before and after.

- **R11:** The milestones' quality-assessment section, README, and SPEC are
  updated to the delivered state, including the measured soak bound.

## Boundaries

- The D12 fix, the D16 clock correction, patch relocation, and interface
  migrations keep their fn-105, fn-110, and fn-109 owners.
- No feature is removed. Deterministic GC, multi-P scheduling, and rr-style
  machine recording remain research items.
- No capability is widened and no test in `./tests` is rewritten for Gomad.
- Pin-repair tooling, including one-command adapter regeneration, belongs to
  fn-113.

## Decision Context

The localiser comes first because D12 and each earlier darwin divergence took
a bisect or a per-event logging campaign to find, and the same search would
repeat for the next channel. A digest per choice point reuses the existing
choice-record path and gives a first-difference ordinal directly.

A soak with a stated N replaces two-repetition qualification as the source of
the determinism claim because a defect at D12's rate passes most
two-repetition checks, and exact replay forces run-queue and select choices
that would otherwise expose a divergence between fresh runs.

Suite reshaping is in the same spec because the assessment found the
assurance gaps and the test-mass imbalance together: the layers that are
missing are the ones that test the product claim.

## Planning decisions (2026-10-01)

Task breakdown settled the points below. Each is a default the owner can
change before the task that uses it starts.

- **R1 includes repair.** A failing job is reproduced and fixed on a committed
  tree. A failure whose cause stays unknown is recorded as a finding with its
  log and keeps R1 open.
- **R2 gates.** The `gomad3_toolchain` tests get their own Make target inside
  `test`. `./toolchain` runs once with the patched toolchain and once with
  stock Go, and leaves the host tier.
- **R3 has no task of its own.** Each task re-anchors the findings it uses as
  its first step and records confirmed, changed, or refuted.
- **R4 transport.** The diagnostic trace uses its own inherited descriptor and
  byte bound, separate from the 64 MiB Choice Trace. A `--diagnostics` flag on
  `explore` and `qualify` enables it. The differ is a `gomadtool` subcommand.
- **R4 off-mode comparison.** A runtime edit changes the toolchain build key
  and the choice implementation digest, which are part of evidence. Comparison
  across toolchain builds therefore uses a behavioral projection (output
  hashes, transcript, World identity, outcome, virtual time, peak goroutines,
  decision content) with those identities as the allowed differences.
  Byte-identity is required only within one toolchain identity.
- **Docs are written once.** Tasks 2 and 4 to 7 record names and contract
  sentences in their done summaries, and task 10 writes all documentation.
- **R4 digest fields** are limited to state the overlay can read without
  editing a collector file. Draw counters are included so a divergence on an
  untaped draw (timer ties, run-queue shuffle) shows as a counter delta at the
  next choice point.
- **R4 and R5 fixtures** inject their fault through an overlay-only,
  diagnostics-only environment switch. No test-only hunk enters the patch.
- **R5 seeded stream** means every draw that reaches the `GOMADSEED`-derived
  state through the patch's runtime rand helpers. The inventory test counts
  references in the patched source, as the clock inventory does.
- **R6 size.** The soak loops `qualify` at its 32-repetition bound. The first
  step measures per-run cost and picks the largest N that fits one scheduled
  job. Each `qualify` batch compares only against its own first execution, so
  the soak also compares the baseline across batches within one cohort
  (workload, seed, platform, execution identity). The report carries a
  cumulative count per cohort across scheduled runs, and the quotable bound is
  that per-cohort count. A new toolchain identity starts a new cohort.
- **R8 generator** uses the standard library's seeded random source with fixed
  seeds and bounded sequence lengths. Declared differences are per platform.
- **R9 kill point.** The test sends SIGKILL to the coordinator after a fixed
  number of journaled executions and compares decoded execution records and
  semantic summary counts with an uninterrupted run. Campaign IDs and artifact
  references are normalized. Wall-time fields, journal segmentation, and
  journal hashes are excluded, because recovery legitimately changes them, and
  each Campaign's storage integrity is validated separately.
- **R10 follows fn-109 task 6**, which rewrites the executor-injection tests.
  A behavior is one named subtest or table row. The mapping is retained under
  `.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/`.

## Open Questions

1. What cumulative soak count is the target bound, and may the soak use
   runners other than the GitHub-hosted ones?
2. fn-110 tasks 2 to 4, fn-109 task 13, fn-114 C2/E3/E4, and fn-105 D26 edit
   the same patch hunks and `runtime/gomad.go`. Which lands first, and who owns
   the batched toolchain identity? R5 rerouting shifts every seed's schedule,
   so it should share one identity bump with them. No task dependency is
   recorded, because the localiser is the prerequisite for D12.
3. If a digest field or a draw site turns out to need a collector file, who
   gives patch-policy approval, and is dropping the field acceptable?
4. R2 and R9 invest in `tools/gomad3sim` tests and `resume`. Both are on the
   list of features with no caller whose fate is undecided.

## Quick commands

```bash
make -C tools/gomad3 validate
make -C tools/gomad3 test
gh run list --repo stephanos/temporal --workflow gomad3.yml -L 5
```

## Early proof point

Task fn-112-gomad-determinism-assurance-and-test.3 validates the core approach:
a runtime-state digest recorded at each choice point without allocating or
drawing, with evidence unchanged when diagnostics are off. If recording
perturbs the run, re-evaluate the digest fields and the transport before
tasks 4, 5, and 10.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Workflows green on both platforms | .1 | — |
| R2 | Orphaned tests run in a gate | .2 | — |
| R3 | Findings re-anchored before work | .3, .5, .6 | First step of each task that uses an assessment finding |
| R4 | Diagnostic trace and differ | .3, .4 | — |
| R5 | Draw-site inventory and runtime check | .5 | — |
| R6 | Scheduled soak gate | .10 | — |
| R7 | Channel fixtures and closure-mode contract | .6, .10 | Task 6 drafts the contract sentences; task 10 writes them into README and SPEC |
| R8 | Model conformance against the host OS | .7 | — |
| R9 | End-to-end CLI and kill-then-resume | .8 | — |
| R10 | Change-detector tests consolidated | .9 | — |
| R11 | Docs updated to delivered state | .10 | — |
