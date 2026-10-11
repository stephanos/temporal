# Porting the standalone activity model to Rust

Assessment date: 2026-10-10. Scope: the complete standalone activity model,
including its shared dependencies, properties, queries, compositions, executable
realizations and regression coverage. This is a source and artifact assessment;
no Rust prototype, benchmarks or model-gate run were performed. Source counts
include comments and blank lines. The working tree contains an existing change
to `system/Heartbeat.scala`; source observations include it, while artifact counts
describe the stored IR and manifest, whose freshness was not established here.

## Assessment

**Yes: a complete port is technically realistic. The strongest approach is a
Rust authoring frontend that emits the existing Umpire IR and continues to use
the Go checker, lowering, conformance and Testpilot runtime.** Budget for a small
compiler and DSL project, rather than a direct translation of one state machine.

The behavior is a good fit for Rust: finite enums and records, bounded counters,
pure transitions, explicit alternatives and predicates. There is no language-level
obstacle in the inspected model. The difficulty lies in preserving the authoring
and verification contracts around that behavior. Confidence is high in behavioral
expressibility, moderate in achieving equally readable authoring without a trial,
and low in any performance or delivery estimate before a prototype.

My planning judgment is **roughly 10–20 engineer-weeks for the complete standalone
frontend port**, assuming one engineer already comfortable with Rust and this
IR, stable requirements and continued use of the Go pipeline. This is an order
of magnitude, not a measured estimate. A full replacement within a few weeks is
not a credible commitment. A bounded feasibility trial is worthwhile before
committing to migration.

If the motivation is faster model checking, changing authoring language alone
will not deliver it: the evaluator already runs in Go. If the motivation is
removing JVM/compiler coupling, Rust team familiarity, or distributing a native
authoring tool, the port has a clearer potential payoff.

## What “the whole model” includes

The folder is substantially more than its main lifecycle. Its
[export declarations](../temporal/model/temporal/features/activity/standalone/Standalone.scala)
root three independent IR artifacts:

| Stored artifact | Machine declarations | Compositions | Properties | Queries | Realizations | Lowered Cases |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `activity-standalone.json` | 20 | 1 | 65 | 65 | 15 | 18 |
| `activity-standalone-record.json` | 13 | 7 | 41 | 87 | 0 | 0 |
| `activity-standalone-race.json` | 3 | 0 | 2 | 2 | 2 | 2 |

Counts come from the [IR directory](../temporal/model/ir/) and
[Case manifest](../temporal/model/cases/manifest.json). Declarations repeated
across artifacts are counted in each artifact; these are not distinct-definition
totals. The manifest has 154 standalone query entries: 20 `lowered`, 98
`nothing-to-realize`, 33 `no-realization` and 3 `unsupported`. Complete migration
must account for all of them, including deliberate failures and unsupported
paths. The existing unsupported Cases concern attempt-record ordering and an
unanswered attempt; porting authoring does not fix those runtime limitations.

The behavioral scope includes:

- The public Product and detailed System, with their state/fact projection and
  stuttering refinement.
- Completion, failure retry, exhaustion, cancellation, pause/resume, start delay,
  retry backoff, four deadline kinds, heartbeat, By-ID settlement and deferred
  reset, including retained pause and control priority.
- Durable dispatch admission, failed commits, stale delivery, duplicate delivery
  and lost admission responses, plus monitors and deliberately faulty controls.
- Worker synchronization and task-queue compositions, including opaque and
  detailed providers, provider replacement, storage-loss assumptions and invalid
  providers that must remain rejected.
- Realizations: typed RPC declarations, payloads, controller and worker scripts,
  learned execution IDs, evidence correlation, occurrence numbering, delivery
  controls, API visibility, timer bounds and cleanup.

The dependency closure leaves the folder. Relevant source sizes are:

| Area | Non-test Scala files | Lines | Why it matters |
| --- | ---: | ---: | --- |
| Standalone feature | 18 | 3,888 | Behavior and executable declarations |
| Its local tests | 12 test files | 4,021 | Finite row pins, truth sets, refinement and realization regression checks |
| Generic framework | 19 | 3,477 | Finite catalogs, rules, derivation, composition, capabilities and realization typing |
| Shared capability declarations | 7 | 354 | Shared laws whose expansion must remain equivalent |
| Temporal realization kit | 6 | 1,197 | Common roles, behavior hints, typed scripts and settlement helpers |
| Task-queue foundation | 3 | 510 | Providers and assumptions used by dispatch compositions |
| Current Scala IR generator | 20 | 11,337 | Evidence of the compiler responsibilities being replaced |

The activity-kind declarations, client/worker actors and bounds also belong to
the closure. These sizes are context, not additive estimates of Rust code: the
frontend need only implement constructs used by the port, and the current lifter
also serves other features. The [module map](../temporal/.plans/UMPIRE_MODULES.md)
describes the ownership boundaries.

If “whole model” means **all of `temporal/model`**, rather than this feature and
its dependencies, the visible authoring/framework sources span 68 non-test files
and 12,419 lines, plus 33 test files and 7,164 lines, excluding the lifter, gate
and refusal fixtures. Nexus forms and their additional claims/realizations widen
the work. I would treat that as a subsequent migration, provisionally several
additional months, with its estimate revisited after the standalone trial.

## The existing architecture gives us a useful boundary

The [model README](../temporal/model/README.md#the-layers) establishes this flow:

```text
Scala declarations -> TASTy lifter -> Umpire IR -> Go interpretation/checking
                                             -> Go lowering -> Case -> Testpilot
                                             -> Go conformance and Quint export
```

A Rust port can replace the first two stages:

```text
Rust declarations -> Rust frontend -> the same Umpire IR -> existing Go pipeline
```

The IR represents finite types and symbolic functions/expressions, not native
callbacks. Its meaning is defined in
[SEMANTICS.md](../temporal/model/SEMANTICS.md), independently of the Scala
lifter. A complete standalone model can therefore be authored in Rust while
checking and live execution remain in Go. No Rust Temporal SDK, async runtime,
FFI bridge or Temporal server rewrite is required for that scope.

The [shared specification](../temporal/.plans/UMPIRE4_SPEC.md) already requires
serializable cross-language declarations (`AUT-05`) and producer-neutral Cases
(`SEM-18`). Some newer language-neutral frontend wording is explicitly marked
as awaiting approval. That is architectural support for investigating a second
frontend, not evidence that a multi-frontend implementation already exists or
that all migration policies have been settled.

## How well the code maps to Rust

| Current construct | Plausible Rust representation | Assessment |
| --- | --- | --- |
| Finite enums, including facts with payloads | Enums with unit and data variants | Straightforward; derive/catalog code must preserve declared order |
| Case-class state and `.copy(...)` | Structs with functional update syntax | Straightforward; scalar lifecycle state can be copied by value |
| `UpTo[2]` and saturating count | Private bounded newtype, checked constructors and explicit saturating successor | Straightforward; ordinary integer storage alone loses the finite-domain guarantee |
| Structural equality | Derived `Eq`/`PartialEq` | Straightforward for model values |
| Rules, guards and effects | Typed descriptors plus a restricted macro or expression builders | Feasible, but substantial frontend work |
| Named `choose` alternatives | Ordered alternatives carrying stable names | Straightforward representation; preserve every result rather than sample one |
| Roles on individual phase cases | Enum-variant role metadata and generated membership predicates | Requires a different encoding; Rust traits apply to types, not individual variants |
| Derived machines and compositions | Declarative overrides, member projections and typed sync descriptors | Feasible; inherited ownership and replacement laws require careful preservation |
| Shared capability laws and waivers | Generic declarations expanded under explicit bindings | Feasible; preserve generated properties, queries, origins and waiver metadata |
| Typed protobuf fields and RPCs | Generated message/method/field descriptors with typed selector handles | Feasible; generated messages alone do not provide the current selector DSL |

The [System state](../temporal/model/temporal/features/activity/standalone/system/System.scala)
has eight finite fields and 5,616 catalog states:
13 phases × 3 dispatch values × 3 attempt counts × 16 deadline combinations ×
3 policies. Its 119 action classes imply 668,304 catalog state/action pairs
before reachability filtering. Rust can represent the state compactly, but does
not reduce those combinations. The separate
[Product](../temporal/model/temporal/features/activity/standalone/product/Product.scala)
has nine states and intentionally admits nondeterministic alternatives.

Rust's [enum support](https://doc.rust-lang.org/reference/items/enumerations.html)
and [struct update syntax](https://doc.rust-lang.org/reference/expressions/struct-expr.html)
cover the basic representations. Persistent collections are not needed for this
scalar lifecycle state. Private mutation of a fresh transition draft is fine if
the input and sibling alternatives remain unchanged. Sharing larger data can be
considered after measurement; ownership does not itself prove behavioral laws.

## The main cost: a new authoring frontend

### Extracting symbolic behavior

The [current lifter](../temporal/model/irgen/Lift.scala) reads typed TASTy trees,
resolved symbols, helper calls and protobuf selections. Translating Scala `def`s
into ordinary Rust functions leaves no exporter that recovers their symbolic
meaning. Stable [Rust procedural macros](https://doc.rust-lang.org/reference/procedural-macros.html)
consume token streams, rather than providing the typed-tree inspection interface
used here. A parser can recognize syntax; it cannot assume resolved types or
symbols across arbitrary helper functions and crates.

Two credible designs are:

1. **Typed expression builders with small derives/macros.** Build symbolic
   `Expr<T>`, field handles, actions and model descriptors directly. This bounds
   compiler complexity and supports isolated testing, but may be visibly more
   verbose than today's models.
2. **A restricted model macro.** Support a documented subset of Rust-like
   matches, conditions, updates and helper declarations, generating symbolic IR
   and type-checkable declarations from the same source. This can improve
   readability, but entails name resolution, diagnostics, helper restrictions,
   source mapping and a larger compiler test suite.

I would prototype builders first, then add syntax only where actual feature code
needs it. Avoid a general Rust compiler integration as the initial dependency.
Do not maintain separate handwritten native and symbolic transition definitions.

An ordinary `fn step(...) -> Vec<Step>` with a Rust model checker is enough for a
behavioral prototype. Executing those functions and dumping tables is not a
drop-in Umpire frontend: the current IR also carries expressions, functions and
positions used for interpretation, diagnostics and Quint translation.

### Preserving checks outside the evaluator

The frontend must retain finite enumeration order, action-input typing,
disjoint-rule checks, expression-level restrictions, phase-role metadata,
closedness checks, declaration/export discovery, source locations and refusals
for unsupported constructs. Some of these are enforced by Scala types or its
authoring framework rather than the IR reader. For example,
[RoleRefinements.test.scala](../temporal/model/temporal/features/activity/standalone/RoleRefinements.test.scala)
checks closedness that the role-free IR cannot reconstruct.

### Typed realizations and protobuf compatibility

[Realization.scala](../temporal/model/temporal/features/activity/standalone/system/Realization.scala)
is 1,064 lines, over a quarter of the standalone non-test source. Its field
selections encode request/response types, optional messages, evidence identities,
script dependencies and learned values. Replacing those selections with strings
would move errors later and weaken authoring guarantees. Generate typed field
and method handles from the repository's exact descriptors, including the linked
Temporal API and Testpilot schema.

[Prost](https://github.com/tokio-rs/prost) supplies protobuf message generation;
[pbjson-build](https://docs.rs/pbjson-build/latest/pbjson_build/) and
[prost-reflect](https://docs.rs/prost-reflect/latest/prost_reflect/) offer routes
to ProtoJSON and descriptors. These are candidate dependencies, not choices
validated by an implementation here. Plain Serde derives are insufficient:
[ProtoJSON](https://protobuf.dev/programming-guides/json/) has specific enum,
64-bit integer, bytes, presence and well-known-type rules. Preserve the existing
nine-file Umpire schema closure and present empty oneof marker arms.

### Identity and artifact migration

Rust module names must not silently replace public model IDs. Preserve semantic
family, owner, action, field and choice names explicitly, even where Rust source
uses different identifier conventions. Match catalog order and ordered results;
set-equivalent transitions alone are insufficient for deterministic witnesses.
The [fingerprint implementation](../temporal/tools/umpire/internal/engine/canonical.go)
documents frozen canonical encodings carried by existing Cases.

Truthful Rust source positions will differ from Scala positions. Equal behavior
does not guarantee equal IR digests or complete Case bytes, which can include
producer/source metadata. Decide the identity/provenance policy before migration,
compare behavioral equivalence separately, and review any deterministic fixture
migration explicitly. Do not hide differences with generic normalization or
ignore lists: the specification requires exact generated Case fixtures
(`ART-11`). Subsequent Rust regeneration must reproduce its approved fixtures
byte-for-byte.

## Pros and cons

| Potential advantage | Practical value here |
| --- | --- |
| Remove JVM/TASTy coupling from migrated authoring | A native frontend can simplify distribution and reduce dependence on Scala compiler internals |
| Strong enums, typed descriptors and explicit ownership | Good fit for finite state and isolated transitions; benefits depend on DSL quality |
| Familiar tooling for a Rust-oriented team | Cargo, formatting, linting and editor support may improve daily work if that is the team's preference |
| Opportunity to simplify frontend boundaries | Catalog derivation, symbolic expressions, typed realizations and export admission can become separately testable modules |
| Reuse the current checking investment | Emitting the same IR keeps Go checks, lowering, live execution and independent export available |

| Cost or disadvantage | Practical impact here |
| --- | --- |
| Build a compiler/DSL, not merely translate states | The biggest cost and source of migration defects |
| Lose Scala's existing DSL ergonomics | Variant roles, contextual typing, helper expansion and typed selectors require replacement designs; Rust may be more verbose |
| Duplicate shared declarations during migration | Activity depends on laws and kits also used by Scala models; temporary dual authoring needs parity checks and a retirement plan |
| Maintain two toolchains during coexistence | Adding Rust initially increases the build matrix; Scala/JDK cannot disappear while other models still need them |
| Risk behavioral and identity drift | A subtly changed guard, result ordering, evidence mapping or catalog can change witnesses and Cases |
| Limited direct performance payoff | Go still evaluates the IR; state explosion and real-server waits remain |
| Opportunity cost | Several months spent on language migration cannot also expand model coverage or fix current realization gaps |

Rust's memory-safety guarantees are valuable when implementing a checker, but
the current authoring layer already uses managed Scala and substantial type
checks. They are not a strong standalone reason to rewrite this declarative
model. Native startup and memory improvements are plausible for authoring;
compile time and end-to-end speedups require measurement.

## Alternatives and recommendation

| Scope | Realistic outcome | Recommendation |
| --- | --- | --- |
| Native Rust lifecycle prototype | Demonstrates states and transitions; omits much of the whole model | Useful trial, insufficient migration acceptance |
| Complete standalone Rust frontend emitting Umpire IR | Preserves the full model and existing evaluation/execution pipeline | Preferred port boundary |
| All Scala models and shared authoring replaced | Removes Scala from authoring after the last dependency is migrated | Consider after standalone parity and ergonomics are demonstrated |
| Checker, lowering, conformance and Testpilot also rewritten | Rebuilds most of Umpire and its Temporal execution integration | Separate, much larger project; requires a concrete benefit and new estimate |

[Stateright](https://github.com/stateright/stateright) is a credible Rust-native
finite-state checker to investigate for an independent backend, but its README
marks liveness as experimental/incomplete. It does not supply this project's
realization-aware Case pipeline. [Proptest](https://proptest-rs.github.io/proptest/intro.html)
can help differential testing and shrinking; it does not establish exhaustive
catalog coverage. [Kani](https://github.com/model-checking/kani) can check bounded
Rust code, but is a different verification harness rather than an implementation
of Umpire queries, refinements and evidence semantics. None is a turnkey
replacement for the whole model.

The current [Quint agreement](../temporal/tools/umpire/export/README.md) remains a
useful independent check after a frontend port. Its coverage is limited: it does
not independently validate all queries, refinements or progress claims. It cannot
be the sole migration acceptance criterion.

## A practical feasibility trial

Allocate approximately **2–3 engineer-weeks** to answer the uncertain questions:

1. Build a small frontend that emits symbolic IR for finite state, bounded
   counters, guards, record updates and facts. Port completion and one retry
   path through the Product/System projection; have Go admit/check the IR and
   lower an equivalent executable Case.
2. Port a difficult slice: named admission commit/failure alternatives and a
   queue composition/provider replacement, plus a heartbeat or deferred-reset
   realization with nested typed fields, learned execution identity and attempt
   evidence. Include an expected negative result.
3. Compare authored readability, compiler diagnostics, cold/warm build cost,
   generated declarations, query receipts and Case differences. Document the
   provenance policy and remaining frontend constructs.

Continue only if the trial preserves semantics without growing a broad Rust
compiler dependency, produces readable subject models, retains useful typing
and diagnostics, and demonstrates a concrete toolchain benefit. The lifecycle
alone is too easy a trial to answer the whole-model question.

For production, isolate the frontend behind a narrow interface:
`export(declarations) -> admitted-version IR or located errors`. Keep generic
finite-domain/expression support, descriptor-derived realization typing and
Temporal-specific declarations separate. The frontend writes declarations;
the Go pipeline remains responsible for authoritative evaluation.

## What would establish completion

Preserve the existing comments and implementation-source explanations when
translating declarations. Establish a reviewed source/IR baseline first, including
the current uncommitted edit. Then require:

- Complete export and manifest accounting for all three standalone artifacts.
- Exact finite catalogs and state/action results, including disabled pairs,
  ordered facts, outcomes, named alternatives and explanations. Reuse the
  [Scala table pins](../temporal/model/temporal/features/activity/standalone/StandaloneActivityPins.test.scala)
  and [Go activity checks](../temporal/tools/umpire/check/activity_pins_test.go)
  as migration evidence.
- Equal property truth sets, refinement results, closedness laws, monitors,
  assumptions and bounded query results/witnesses under the same limits.
  Negative controls must still fail; invalid providers must remain rejected.
- Complete capability expansion, origins/waivers and unchanged declared known
  gaps or unsupported standings unless separately reviewed.
- Valid descriptor/ProtoJSON round trips, source diagnostics, stable public
  identities and reviewed artifact differences. Regeneration must be deterministic.
- Equivalent generated Programs/Contracts for all 20 currently lowered Cases,
  followed by the existing live and replay checks of Verdict, conformance,
  property status and cleanup. Equal tables alone do not validate realizations.

Relevant existing validation paths are `make umpire-check-model`,
`go test -tags test_dep ./tools/umpire/...`, `make umpire-check-backends` and
`make umpire-check-live-tests`, run from `temporal/`. The model gate currently
builds Scala, so a Rust frontend needs gate integration before that command
validates both languages. These commands are proposed migration checks, not
checks run for this assessment.

**Recommendation:** pursue the bounded trial if native authoring or Rust
maintainability is an explicit goal. A full standalone port is achievable with
the existing IR boundary; its value depends on replacing compiler/toolchain
complexity with a simpler, usable frontend. Keep the checker/runtime rewrite as
a separate decision supported by measured bottlenecks.
