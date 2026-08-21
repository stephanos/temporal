# Gomad v3 Next: Bug-Finding Power

> **Status note:** This is the detailed track design. Current implementation status and cross-track ordering live in [GOMAD3_NEXT.md](GOMAD3_NEXT.md). The capability designs, invariants, verification plan, and exit criteria here remain normative.

## Goal

Increase the number and quality of concurrency failures Gomad finds per unit of compute, then turn each failure into a small, controlled reproduction. The recommended progression is:

> observe choices → control choices → explore alternatives → minimize failures → inject deterministic faults

The completed first deliverables are bounded runtime-choice tracing, exact
replay, and bounded alternative-prefix exploration. The raw frontier remains a
reference algorithm until workload evidence justifies later search techniques.

## What success means

Gomad should be able to answer:

- Which runtime-controlled choices occurred?
- Which choice sites and alternatives were exercised?
- Can this exact decision sequence be replayed before effects are applied?
- What alternative decision prefixes remain unexplored?
- What is the shortest controlled prefix that still produces the same failure signature?
- Which explicit fault caused a failure, and can it be replayed and minimized?

A successful campaign is not proof of correctness. It is evidence that a bounded search was performed, with enough information to reproduce and compare what was explored.

## Non-goals

- Exhaustive verification of arbitrary Go programs.
- Stable schedule identities across source, compiler, runtime, or architecture changes.
- Race-detector replacement or immediate support for true parallel memory races.
- Broad transparent network fault simulation before the current TCP semantics are correct.
- Unbounded traces, search frontiers, or failure minimization.

## BUG-1: Runtime-choice trace

Add a versioned, bounded stream of runtime decisions. Each record should contain only stable logical data:

- decision kind, such as runnable selection, channel/select winner, timer tie-break, map seed, or synchronization wake-up;
- a target-specific site fingerprint where one can be generated without pointers or host addresses;
- number of eligible alternatives and selected alternative;
- a monotonic choice ordinal;
- enough domain-specific identity to diagnose divergence without exposing unstable runtime internals.

The patched runtime should write records through a fixed-size shared-memory protocol similar to the I/O transcript. The process layer should return terminal state separately from payload so overflow, malformed records, and incomplete shutdown cannot be mistaken for a valid trace. Artifacts should record the trace schema, byte limit, record count, digest, and terminal status.

New user functions:

- `gomad inspect <artifact> --choices` to show counts, kinds, hot sites, and branching points;
- JSON projection of the same information for analysis tools;
- a concise choice summary in failure output.

The trace must be optional and identified as an execution profile because observation can perturb a schedule. Comparisons are valid within the same profile, not between traced and untraced runs.

### Deep module boundary

Introduce one choice protocol module that owns encoding, bounds, validation, terminal state, and projection. The runner should consume a validated choice artifact rather than know runtime record details. Runtime-specific encoding should not leak into guidance, replay, or CLI packages.

## BUG-2: Choice coverage and feedback

Extend semantic guidance with choice features:

- decision kinds reached;
- site fingerprints reached;
- sites observed with more than one alternative;
- selected alternative classes;
- bounded adjacent choice pairs;
- trace terminal status.

Use these features to retain novel successful seeds and rank the existing corpus. Report choice coverage as observed features, never as a percentage of a presumed complete schedule space.

Do not begin with source-code coverage. Compiler coverage instrumentation changes program structure and scheduling, and branch coverage says little about schedule diversity. Choice coverage directly measures the mechanism Gomad controls. Code coverage can later be a separate, explicitly perturbing profile.

## BUG-3: Exact choice-tape replay

Implemented in the v2 choice controller: retained complete traces derive an
identity-bound immutable tape, artifact replay supplies it automatically, and
the runtime rejects mismatches before applying a decision. V1 traces remain
readable observational evidence but cannot claim exact replay. The `prefix`
mode is available internally for BUG-4; this capability does not expose
a public tape or prefix flag.

Separate decision production from the runtime RNG. A choice controller should support four modes:

1. `seed`: current seeded behavior;
2. `record`: seeded behavior plus a validated choice tape;
3. `replay`: consume the complete expected tape;
4. `prefix`: force a supplied prefix, then return to seeded choices.

Replay must validate each choice before applying it. A mismatch in kind, site, alternative count, selected value, missing choice, or extra choice should produce a stable divergence record and stop the target. Exact choice replay remains bound to the same target and platform identities.

The choice tape should complement, not replace, I/O and World replay. Replay succeeds only when all enabled controllers consume their expected input and the final outcome/evidence matches.

## BUG-4: Bounded alternative-prefix exploration

Implemented as the `choice-frontier` campaign strategy. It uses one base seed,
breadth-first canonical prefixes, round-atomic persistence, durable resume, and
explicit run, depth, frontier-byte, wall-time, and failure bounds. The pinned
two-outcome `select` comparison reaches two declared outcomes in sixteen
executions under both frontier and seed sampling. This satisfies the requirement
to report where raw prefix search does not improve efficiency and does not yet
justify BUG-5, PCT, or dependency reduction.

Once choices can be recorded and forced, add an explorer that derives new runs from observed branching points. Start with a simple bounded frontier rather than DPOR:

- record the choice sequence for a run;
- identify sites with untried eligible alternatives;
- enqueue canonical forced prefixes that select one alternative differently;
- execute each prefix in a fresh process;
- deduplicate prefixes and semantic outcomes;
- stop at explicit run, depth, frontier-byte, wall-time, and failure budgets.

Suggested CLI shape:

```text
gomad explore --strategy=seed ...
gomad explore --strategy=choice-frontier --max-choice-depth=N --max-frontier-bytes=N ...
```

The campaign plan and artifact must record strategy and all search bounds. Resume should restore the frontier from immutable or journaled state without rediscovering completed prefixes.

This will not eliminate redundant schedules. It provides a controlled foundation on which happens-before reduction could later be evaluated using measurements from real workloads.

## BUG-5: Failure minimization

Add `gomad minimize <artifact>` for exact-replay failures. The predicate is preservation of the same normalized failure signature and replay-compatible outcome.

Use bounded passes:

1. remove an unnecessary suffix from the forced choice prefix;
2. simplify ranges of forced choices back to their seeded/default value;
3. remove redundant fault-plan entries;
4. reduce explicit scenario parameters when the target declares shrinkable values.

Every candidate runs in a fresh process under the same target and platform identities. The original artifact remains immutable; minimization publishes a new artifact linked to its parent, with attempt count, budget, accepted reductions, and final predicate evidence.

Do not claim general input shrinking. Target inputs are opaque unless a scenario adapter supplies a typed shrinker.

## BUG-6: Deterministic fault plans

Extend World with a versioned fault-plan interface for explicit adapters. Initial fault actions should be small and composable:

- return a declared adapter error;
- delay readiness until a logical instant;
- cancel a pending request;
- drop one explicit modeled delivery;
- exhaust a declared capacity.

Match faults on stable adapter, resource, operation, and occurrence identities. Record both planned and realized faults. Exact replay validates a fault before applying it; unused or extra fault entries are divergence.

Begin with mailbox and purpose-built Temporal test adapters. Do not retrofit transparent TCP into a distributed network simulator in this phase. Partitions, packet loss, and reordering require a richer network semantics and should be justified by workload evidence.

## BUG-7: Later research extensions

Evaluate these only after the choice controller is stable:

- compiler checkpoints at high-value blocking or synchronization sites;
- deterministic GC trigger points and GC choice records;
- bounded partial-order reduction based on observed synchronization;
- a separate code-coverage-guided profile;
- experimental multi-P record/replay.

Multi-P is last because true parallel execution introduces memory-order decisions that a single logical choice stream cannot fully control. A credible design needs a recorded synchronization protocol and a clear memory-model claim, not only `GOMAXPROCS > 1`.

## Data flow

```text
patched runtime
  → bounded choice protocol
  → process result
  → validated choice artifact
  → semantic guidance / replay controller / frontier builder
  → campaign journal and immutable failure artifact
  → inspect or minimize
```

The choice artifact is the stable handoff. Guidance must not parse runtime memory; the runtime must not know corpus or search policy; the runner must not implement choice semantics.

## Error handling and failure modes

- Trace overflow is a typed capacity outcome, not silent truncation.
- A malformed or unterminated trace is a runner/toolchain failure.
- Replay divergence stops before applying the mismatched decision.
- Frontier exhaustion is a successful bounded completion; frontier capacity exhaustion is reported separately.
- A minimization candidate that diverges or changes signature is rejected, not treated as a new minimized failure.
- Crashes during frontier or minimizer updates must leave a resumable prior state.

At 10× choice volume, memory must remain bounded by streaming validation, on-disk segmented frontier state, feature deduplication, and explicit trace limits. The explorer must apply backpressure rather than enqueue the full theoretical branch tree.

## Trade-offs

- Choice tracing adds runtime and storage overhead and may perturb schedules. Profile identity and A/B benchmarks are mandatory.
- Site fingerprints improve diagnostics but cannot be stable across arbitrary compiler changes.
- Prefix exploration is easier to trust than sophisticated reduction but produces more redundant runs.
- Fault injection improves failure-path coverage but expands adapter semantics and replay obligations.
- Compiler checkpoints expose more choices at the cost of a larger patch surface and more schedule instability after source changes.
- Choice traces reveal internal control-flow and synchronization metadata and must inherit artifact sensitivity, retention, and export policy.

## Verification plan

1. Golden protocol tests for every choice record and terminal condition.
2. Black-box tests proving identical target/seed/profile produces the same complete trace under different host load.
3. Tests proving different seeds reach different alternatives in known branching programs.
4. End-to-end replay tests that reject the first changed choice before target-visible mutation.
5. Small state-machine tests where frontier exploration enumerates all deliberately modeled outcomes within a known bound.
6. Crash/resume tests with a non-empty frontier and duplicate outcomes.
7. Minimizer tests proving the same signature with fewer forced choices and proving the parent artifact is unchanged.
8. Fault tests for unused, extra, reordered, and mismatched actions.
9. Benchmarks for trace overhead, frontier growth, artifact bytes, and executions per discovered outcome.

## Exit criteria

### Choice trace v1

- Every qualified runtime-choice corpus case emits a valid bounded trace.
- Repeated runs produce identical trace digests for the same execution identity.
- Overflow and protocol failures are explicit and tested.
- `inspect` can explain branching sites without runtime-internal knowledge.

### Controlled exploration v1

- Exact choice replay validates decisions before application.
- A benchmark corpus demonstrates that the frontier finds known alternate outcomes with fewer executions than seed-only sampling, or demonstrates clearly where it does not.
- Search bounds and remaining frontier are durable across resume.

### Minimization and faults v1

- Known failures are reduced while preserving their normalized signature.
- Fault plans replay exactly and unused/extra faults diverge.
- At least one representative Temporal workload finds a failure path not reached by the same seed budget without fault control.

## Recommended next slice

Run the implemented BUG-4 frontier against additional representative Temporal
workloads and compare failures and declared semantic outcomes per compute-hour
with equal-budget seed sampling. Advance to BUG-5 only after this evidence shows
that controlled prefixes find a distinct failure or materially improve useful
outcomes per execution; otherwise keep the raw frontier as an exact, bounded
diagnostic strategy.
