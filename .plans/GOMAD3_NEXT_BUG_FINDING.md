# Gomad v3 remaining bug-finding work

Increase distinct, replayable failures per unit of compute and reduce their
reproductions. [GOMAD3_NEXT.md](GOMAD3_NEXT.md) places this track alongside the
active delivery work.

Choice tracing, coverage, exact tape replay, bounded prefix exploration, and the
existing combined-simulation reducer are documented in the
[README](../tools/gomad3/README.md) and
[specification](../tools/gomad3/SPEC.md#campaignchoicefrontier-choice-exploration).
Their implementation plans are complete; the proposals below extend them.

## Search evidence

Compare the existing frontier and seed sampling on representative Temporal failures
with equal execution and wall-time budgets. The pinned two-outcome `select` fixture
reaches both outcomes in sixteen executions under either strategy; it establishes
no efficiency advantage. Record failure signatures, semantic outcomes, trace and
artifact bytes, frontier growth, and recovery executions separately.

Keep raw prefix search as a bounded diagnostic baseline. A more complex policy
needs a demonstrated bug class or a measured discovery-cost improvement.

## BUG-5: Failure minimization

The remaining work is crash-resumable minimizer state and typed scenario-input
shrinkers. Every accepted candidate must preserve the normalized failure, outcome,
and exact choice/model replay under the same target and platform identities.

Checkpoint attempt order, budgets, accepted reductions, and parent identity so an
interruption resumes from a validated prior state. Publish a new artifact with
lineage; the parent stays immutable. Scenario owners supply typed shrinkers and
input-validity rules. Divergent candidates or changed signatures are rejected.
General input shrinking and causal minimality are separate claims.

## BUG-6: Deterministic fault plans

Simulation already has lifecycle, network, storage, and typed fault plans. Broader
explicit World adapters may justify additional error, readiness-delay,
cancellation, delivery-drop, or capacity-exhaustion actions.

Each adapter extension needs a named workload, stable resource/operation/occurrence
matching, separate planned and realized fault identities, hard bounds, and
fail-before-mutation replay. Missing, extra, reordered, or inapplicable faults must
diverge. Qualify a representative failure path against the same seed budget
without fault control before claiming better discovery power.

## BUG-7: Later research extensions

Candidates include PCT, preemption bounds, compiler checkpoints, deterministic GC
observations, partial-order reduction, a separate code-coverage profile, and multi-P
record/replay. Their requirements differ:

- PCT and preemption bounds need stable actor identities, enabled sets, and explicit
  preemption accounting. DPOR additionally needs reviewed semantic resource/access
  dependencies and causality metadata. Unknown transitions remain dependent.
- Reduction must agree with an unreduced reference search on outcomes and deadlocks
  in finite fixtures. Novelty hashes alone do not justify independence.
- Visited-state pruning requires every future-relevant part of an explicit model.
  A World digest cannot summarize native Go heap, stacks, timers, or runtime state.
- Compiler and coverage instrumentation perturb execution and need separate profile
  identities, overhead measurements, disabled-mode tests, and replay qualification.
- GC work needs a demonstrated collector channel and reviewed policy. The current
  constraints and findings belong to [milestones](GOMAD_MILESTONES.md).
- Multi-P and race detection remain separate profiles. Single-P schedule exploration
  does not cover racy reads or weak-memory behavior.

See the [comparative research](../docs/research/gomad/GOMAD_CMPv2.md) for candidate
policies and oracle design. Record fairness and logical progress thresholds when
checking recovery; bounded progress tests do not prove absence of nontermination.

## Extension gates

The choice protocol owns record validation, bounds, and projection. Search consumes
validated choices; it never reads runtime memory or grants capabilities. Every
extension retains explicit run/depth/byte/time budgets, durable remaining work,
typed capacity and divergence outcomes, and tests that reject a mismatched choice
before target-visible effects. Independently review workload generators and oracles,
and retain native integration testing alongside deterministic simulation.
