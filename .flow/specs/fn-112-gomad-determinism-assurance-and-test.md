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
- fn-105 D16 keeps the forward-clock decision (assessment Q5).
- fn-109 owns interface changes. Where this spec reshapes tests around an
  interface fn-109 migrates, the test change follows the verified migration.
- fn-110 owns patch relocation. Runtime edits here coordinate source-set and
  descriptor changes with it.
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
  evidence with diagnostics off fail this criterion.

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
- A one-command adapter regeneration is a maintenance candidate recorded in
  the milestones and is not part of this spec.

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
