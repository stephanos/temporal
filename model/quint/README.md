# model/quint

An experimental [Quint](https://quint.sh) version of the Nexus caller, standalone activity,
and worker models in [model/go](../go/) and [model/scala](../scala/). It starts from the
[archived Quint sketch](../../.plans/archive/cmp/quint/), corrects its executable behavior,
and adds checks against the existing models.

[RESULTS.md](RESULTS.md) records the verified tables, checker results, and findings.

This is a semantic experiment. It does not yet implement the full
[Umpire specification](../../.plans/UMPIRE4_SPEC.md) or generate Model IR or Testpilot Cases.
The design questions it exercises are in
[UMPIRE4_TLA_COMPAT.md](../../.plans/UMPIRE4_TLA_COMPAT.md).

## Run

Run from the Temporal repository root:

```sh
model/quint/run.sh                 # typecheck, regression tests, table agreement, simulation
model/quint/run.sh --tla           # also export the four machines to TLA+
model/quint/run.sh --tla --verify  # also check their complete finite graphs with TLC
```

The runner uses Node/npm, Python 3, and the repository's Go toolchain. `quint.sh` invokes
Quint **0.33.0** through npm's cache; its first use downloads that exact version. The
TypeScript evaluator runs the tests and simulations, so the default checks need no Rust
evaluator binary. TLA+ export and TLC verification also need Java and download the CLI's
Apalache distribution. Generated data and traces go into `.out/`, which is ignored by Git.

The default checks include:

- Pure framework regressions, named scenarios, late-request behavior, and repeated event
  occurrences.
- Exhaustive checks of the two state-map refinements over their finite tables.
- Worker composition scenarios, including disabled replies and dispatch after worker stop.
- Agreement with freshly computed Go tables for all five base machines, and with the
  committed Lean tables for Nexus, the Activity product, and Worker.
- Seeded simulations of four machines and two compositions: 100 samples of at most 12
  steps each, with their declared safety invariants.

The agreement tests compare complete state and action catalogs, starts, ends, reachable
states, and every state/action pair. They check disabled pairs as well as enabled results,
including outcomes and ordered facts. Catalog order is compared as sets; result and fact
order is retained. The Activity protocol has no committed Lean table because its 288
states exceed that experiment's bound, so its agreement oracle is Go.

Tests end in `Test`, which Quint discovers by default. The model files also contain `run`
declarations for scenarios and queries. To execute one directly, select its name:

```sh
model/quint/quint.sh test model/quint/nexus_protocol.qnt \
  --main nexusProtocol --match '^retry$' --backend typescript --seed 0x1
model/quint/quint.sh run model/quint/activity_protocol.qnt \
  --main activityProtocol --invariant properties --backend typescript \
  --seed 0x1 --max-samples 100 --max-steps 12
```

`quint run` is sampled exploration. The pure table checks are exhaustive over their
declared finite domains. `--verify` runs TLC to completion over the finite machines,
including auxiliary trace state; it checks safety under the abstract model. These checks
make no claim about implementation conformance or unbounded progress. The optional checker
fails the runner on errors or invariant violations.

## Layout

| Path | Purpose |
| --- | --- |
| `umpire.qnt` | Generic Step, last-step records, claims, finite table walkers, refinement predicates |
| `worker.qnt` | Worker state machine, instantiated by task queue |
| `nexus_caller_vocabulary.qnt`, `activity_vocabulary.qnt` | Feature inputs and descriptive catalogs |
| `*_product_table.qnt`, `*_protocol_table.qnt` | Pure domains, transitions, and explicit abstraction maps |
| `*_product.qnt`, `*_protocol.qnt` | Stateful execution, properties, scenarios, and queries |
| `nexus_caller.qnt`, `standalone_activity.qnt` | Synchronized worker compositions |
| `pins.qnt`, `*_tests.qnt` | Exact checks and regression scenarios |
| `parity/main.go`, `parity.py` | Read reference tables and generate temporary Quint expectations |
| `quint.sh`, `run.sh` | Pinned CLI and repeatable checks |

The product tables expose abstract state values and `productAllows(before, after)`.
Protocol modules import those definitions explicitly. This keeps the abstraction map
visible and avoids a Quint 0.33.0 flattening failure encountered with aliased imports of
product tables. Their internal declarations have unique names: compilation also exposed
private-name collisions between the product and protocol tables. Public aliases preserve
the table API used by the agreement tests. Each model module lives in its own file so
compilation loads its required dependencies.

## Semantic choices

Step functions return `List[Step]`; `[]` disables an action. A stateful action chooses
nondeterministically from the complete set of returned alternatives and commits the new
state, outcome, and facts atomically. Exploratory steps choose from enabled classes.
Unsynchronized composition steps explicitly preserve the other member's state.

Each machine keeps `state`, `last`, and a Boolean `occurrence`. `last` holds the action
class, previous state, outcome, and ordered facts. `occurrence` toggles on every real action,
so repeated identical steps remain distinguishable from synthetic stuttering without an
unbounded event counter. It is auxiliary trace state, excluded from base-table agreement.
The experiment has no deadline monitor that counts synthetic stuttering as progress.

Refinement checks mapped state transitions and allows mapped stuttering. They do not assert
that all facts correspond across abstraction levels. Fact lists and outcomes have their
own agreement checks against the reference tables.

Find-query claims keep their scenario meaning. For example, a successful Nexus completion
query looks for a completion fact. Whole-machine invariants also cover late requests:
terminal completions and Activity controls must return `notFound`, preserve state, and
emit no facts. Success assertions apply to accepted requests.

## Current boundaries

Schema, party, example, set, and realization catalogs are descriptive data. Their strings
do not constitute checked Umpire bindings. There is no Case producer, ITF-to-Case adapter,
Definition ID or Behavior Fingerprint parity, or connection to a running Temporal system.

The models reproduce the Go/Scala baseline: Nexus cancellation and Activity reset and
heartbeat timeout remain outside it. Channels, history-sensitive monitors, bounded-progress
queries, fairness, holes, and known-bug warning/fix reporting are future work. Known-bug
semantics are tracked in [KNOWN_BUG.md](../scalav2/specs/KNOWN_BUG.md); model properties remain
independent of any future reporting policy.

Quint's [CLI manual](https://quint.sh/docs/quint) describes simulation, compilation, and
checking. Its [language manual](https://quint.sh/docs/lang) defines atomic updates,
nondeterminism, and temporal operators. Export uses Quint's own compiler; this experiment
does not implement the proposed Umpire IR exporter.
