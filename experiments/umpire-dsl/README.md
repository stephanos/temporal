# Umpire DSL experiment

Intent, comparisons, measurements, and decisions live in
[UMPIRE_DSL_EXPERIMENT.md](../../.plans/UMPIRE_DSL_EXPERIMENT.md).
This package is independent of the production Lake workspace and Flow-Next.

## Run

Use Lean 4.33.1, matching `lean-toolchain`:

```sh
cd experiments/umpire-dsl
./run.sh
```

This builds the kernel-checked examples and executable differential tests, runs the
executable twice, and compares deterministic receipts. Local output is ignored under
`results.local/`. For an additional native execution timing, run
`.lake/build/bin/dsl-experiment --timing`.

```sh
./run.sh --all
```

The extended run also compares against the current Umpire Property evaluator, checks
the pinned Veil semantic adapter, and performs a transition edit in a temporary copy.
The baseline probe needs the existing production compiled dependencies; it does not
build or modify them. The Veil runner fetches only its pinned source checkout if
absent. It does not install the full checker dependency graph.

## What to read

| File | Experiment |
| --- | --- |
| `DslExperiment/Model.lean` | One transition authority for two operations, both resolutions, and a counted self-loop. |
| `DslExperiment/Property.lean` | Typed bounded temporal clause, passive monitor, independent suffix semantics, and chunking theorem. |
| `DslExperiment/Projection.lean` | Run-local correlation, partial evidence, deduplication, atomic rejection, and causal support. |
| `DslExperiment/Query.lean` | Exact/ordered/union scenarios and explicit endpoint, coverage, and work-limit semantics. |
| `DslExperimentTests.lean` | Positive and failure examples, including source-level mode/clock errors. |
| `Main.lean` | Bounded exhaustive semantic comparisons, evidence variants, and repeatable query receipts. |
| `Baseline.lean` | Actual Umpire `eventuallyWithin` comparison using an explicit per-operation fixture projection. |
| `veil/` | Actual pinned Veil types, adapter proofs, and a temporary one-transition edit. |

Both forms below are checked and definitionally equal:

```lean
def response : ResponseClause :=
  whenever (.event .requested) eventually (.anyOf [.canceled, .completed])
    within 1 operationTransitions

def responseExplicit : ResponseClause :=
  ⟨.event .requested, .anyOf [.canceled, .completed], 1, .operationTransitions⟩
```

The operation key is captured independently for every trigger. A response at the
trigger coordinate or inclusive deadline is accepted. Only admitted labeled steps
of that operation tick its clock. `evaluateAdmitted` checks model replay before
interpreting a trace; raw `evaluate` and `reference` are semantic interpreters whose
query/projection consumers enforce admission. Arbitrary words in differential tests
exercise the interpreters, not the model's accepted language.

Query callers must choose `closedFinitePrefixes`, `runtimePrefixes`, or
`terminalWorlds`. Closing selected finite prefixes deliberately makes an unanswered
trigger at the endpoint fail. Runtime prefixes retain pending obligations. A work
budget never closes a trace or proves absence. Receipts report the selected endpoint
policy, semantic depth, satisfiability, trigger coverage, and completeness.

## Boundaries

There is no live Temporal driver, portable Case producer, production fingerprint
implementation, full temporal logic, or symbolic checker in this package. The
projection consumes a synthetic normalized envelope; accepted identities are
immutable. A conflicting later append is rejected and reported separately, and
cannot retract a previously established violation. An append that fails while
releasing buffered evidence emits no new steps. Emissions retain transitive causal
support. Missing parents remain pending; detected cycles reject. The 128-envelope
limit is an experimental resource cap and yields inconclusive evidence.

Finite model exploration keeps full trace history and grows exponentially with
depth. Projection and monitor lists prioritize clarity over scale. No performance
claim extends to production loads. The experiment accepts ordinary Lean logical
axioms `propext`/`Quot.sound` where audited; it introduces no custom axioms,
`sorry`, or compiler-trusted theorem proofs. Executable differential checks are
bounded tests, not a general compiler-correctness theorem.

See [VEIL_RESULTS.md](VEIL_RESULTS.md) for the successful core seam and the full
checker setup failures. The shared relational view here still derives from finite
successors; it does not demonstrate removal of finite admission.
