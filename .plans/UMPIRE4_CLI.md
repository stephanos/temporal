# CLI

These actions assume that an executable Lean model already exists under `model/`. The primary users
are feature engineers who want ordinary tests and fuzzing, and operators who want to reuse the same
model-defined behavior in environment-specific tools. Model authoring is a separate workflow.

The model is the smart part of the system. It defines behavior, regressions, exploration spaces,
properties, coverage, and the meaning of observations. The CLIs are semantically thin: they select
named model declarations, translate versioned artifacts, bind actions to runtime adapters, and move
evidence between the model and a real system. CLI flags may select or tighten model-declared bounds,
but must not invent behavior or broaden those bounds.

## User-facing concepts

- A **property** is a claim the system should satisfy. Properties remain visible because they explain
  why a test passed, failed, or could not be established.
- A **scenario** is a named space of possible situations, including parameters, ordering choices,
  participant behavior, and faults.
- A **test** is one concrete deterministic trace selected from a scenario and checking one or more
  properties.
- A **run** is one execution of a test against a particular environment.
- A **result** is the qualified interpretation of a run: established, violated, unknown, conflict,
  or unsupported.

`ExperimentSpec`, semantic hashes, evidence artifacts, and Lean declaration forms remain inspectable
implementation details rather than the primary user vocabulary.

## Complete executable traces

A generated test trace defines its semantically relevant behavior in totality. It includes:

- required environment capabilities and initial conditions;
- resource setup and semantic configuration;
- SDK or Kitchensink participants and their pre-programmed behavior;
- Temporal API and other runtime actions, including typed references to values such as run IDs that
  become known only during execution;
- fault actions and their activation points;
- ordering, concurrency, and causal constraints;
- observation instructions and compiled expectations derived from properties;
- convergence and termination conditions; and
- cleanup obligations.

The trace remains parameterized by operational bindings that do not change its meaning, such as
endpoints, credentials, namespaces, resource limits, and granted authority. Each trace is logically
self-contained and independently executable. A runner may safely reuse a test cluster or participant
process as an optimization only when it preserves the trace's declared isolation.

## Verify the model

`umpire-check-model` is a thin Go CLI that invokes model-declared Lean verification checks and
assembles a provenance-rich verification receipt.

```text
umpire-check-model
umpire-check-model --profile nightly
umpire-check-model --check <name>
umpire-check-model list
umpire-check-model explain <name>
```

The model declares at least a bounded per-commit profile and a more expensive nightly profile. The
per-commit profile is the default, so CI does not need to know or assemble its individual checks.
Named checks are available for focused diagnosis. A receipt states exactly which proofs or bounded
checks ran, their bounds and trust modes, and whether they established, refuted, or could not
establish their claims; the command must not claim that the model is universally correct.

## Generate portable tests

`umpire-gen-tests` is a Lean executable owned by `model/`. It translates a named regression,
model-defined test set, or model-selected batch into canonical versioned JSON.

```text
umpire-gen-tests list
umpire-gen-tests explain <test-set>
umpire-gen-tests <test-set> --output <directory>
```

The output is a manifest plus complete test traces. Identical model sources, declaration, parameters,
bounds, and seed produce byte-identical output. JSON is the portable source artifact for persistence,
inspection, replay, code generation, test environments, and downstream canary tooling.

The JSON carries the actions and compiled expectation program, property and semantic identities,
required capabilities, bounds, omissions, and provenance. A runtime interpreter may evaluate this
closed expectation language without shipping Lean. It must reject unknown instructions and must not
reimplement Temporal product semantics independently of the generated program.

## Generate ordinary Go tests

`umpire-gen-tests-go` converts generated JSON traces into readable, strongly typed Go test code.

```text
umpire-gen-tests-go <manifest> --output <package>
go test ./<package>/... -run <test>
```

The generated source exposes setup, participant programs, actions, late-bound references, faults,
ordering, observations, expectations, and cleanup as typed trace literals. This lets Go developers
inspect the complete test, use normal test discovery and filtering, set breakpoints, and step through
the runner in their IDE. The source is a deterministic, digest-bound projection of the JSON and does
not become a second editable source of behavioral meaning.

There is no separate public `umpire-run-tests` command. Generated tests call a reusable Go runner
library that interprets traces, realizes actions through registered adapters, collects observations,
evaluates the compiled expectation program, enforces cleanup, and returns qualified results.

## Fuzz a real environment

`umpire-fuzz` is a time-bounded random runtime explorer over a named Lean exploration.

```text
umpire-fuzz list
umpire-fuzz explain <exploration>
umpire-fuzz <exploration> \
  --environment <profile> \
  --time 30m \
  --parallelism 8 \
  --state <directory> \
  --seed <seed>
```

The exploration declaration owns the space to search, required trace contents, structured mutation
operators, properties, semantic coverage features, candidate scoring, and selection strategy. It may
require a feature such as at least one Nexus operation start or may leave the bounded space broadly
open. `umpire-fuzz` never performs exhaustive exploration and never claims completeness.

Lean exposes three exploration operations with serializable state:

```text
initialize exploration seed -> initial batch + exploration state
nextBatch exploration state -> next batch + reserved semantic identities
observe exploration state results -> updated corpus, coverage, priorities, and state
```

The Go CLI executes each batch in parallel through the shared runner, returns normalized results to
Lean, persists the opaque exploration state and corpus, and stops when its operational time budget is
exhausted. Lean can begin with seeded random selection and later use coverage-guided mutation,
rarity-based prioritization, bandit-style learning, or other model-defined techniques without
changing the Go orchestration protocol. Semantic trace identities prevent repetition within a
campaign, including traces currently leased to parallel workers. Persisted state permits resume and
records enough seed, selection, and result history to reproduce how a trace was chosen.

Coverage may include model states and transitions, actions and outcomes, property branches,
variation and fault combinations, causal relationships, convergence paths, black-box observation
signatures, and implementation coverage supplied by an authorized white-box adapter. The model
decides which features are meaningful and which traces remain in the corpus; the CLI only transports
and stores them.

Model-only exhaustive verification remains an `umpire-check-model` concern. Runtime execution of a
finite complete set uses `umpire-gen-tests` followed by the generated Go tests.

## Downstream environment tools

Umpire does not provide a canary CLI or own production policy. It provides the trace and result
schemas, Go runner, expectation interpreter, exploration protocol, and adapter interfaces. Temporal
builds its canary and other deployment-specific tools around those primitives.

The Lean model defines the canary test subset and verifies that it requires only the allowed
black-box actions and observations. A Temporal canary tool must enforce that eligibility, provide the
concrete environment bindings, and own credentials, authorization, rate limits, concurrency,
cleanup, rollout policy, and blast-radius controls. Convergence is a model-defined property that the
same generated tests, fuzzing loop, and downstream canary tools can evaluate; it is not a separate
Umpire CLI.
