# Umpire components and delivery status

Status: reconciled 2026-08-26 against the current `model/` tree, its generators, and the existing Go
Umpire implementations. The component boundaries were originally extracted from the
[Inspect Umpire Branch](https://chatgpt.com/share/6a8b71cb-74e4-83e8-947a-c2f6d595fefc)
design conversation. This remains an architectural inventory, not an approved implementation plan.
Where it conflicts with `UMPIRE_LEAN.md`, the active roadmap takes precedence.

Component status in this document means integration status for the current `model/` semantic
pipeline. Functionality in `common/testing/umpire`, `tools/umpire2`, or `tools/umpire3` is called out
as an existing baseline, but is not considered integrated until it consumes the current
`umpire-experiment/v1` artifact and preserves its semantic identities.

## TL;DR

Status terms: **built** means implemented in the current `model/` pipeline; **partial** means only
part of the component contract is built; **separate** means a working implementation exists in
another Umpire tree but does not consume current-model artifacts; **missing** means no corresponding
implementation was found.

| Component | Status | Summary |
| --- | --- | --- |
| C1 API importer | Partial | Deterministic full-descriptor Lean projection is built. A standalone catalog, complete field dispositions, drift report, explain surface, and shared bounded-selection policy are missing. |
| C2 config importer | Partial | The initialized production registry, typed generated settings, identities, and Go-produced resolution fixtures are built. Product classifications currently cover selected Callback and Matching uses; no standalone explain/check surface exists. |
| C3 Lean authoring DSL | Partial | Checked targets, properties, behaviors, queries, exact traces, ordering, bounds, and model-only examples are built. Variation axes, authored faults, coverage goals, a persisted registry, and list/explain UX are missing. |
| C4 ExperimentSpec compiler | Partial | Pure planning emits one canonical `umpire-experiment/v1` for a selected trace, and the inspector exposes two scenarios. Batch compilation, persisted JSON decoding/validation, migrations, explicit runtime config, and runtime consumption are missing. |
| C5 Go/docs generator | Partial | One stable caller-closure regression has deterministic checked-in Go and Markdown projections; broader catalog and promotion surfaces remain. |
| C6 execution runtime | Separate | Go runtimes drive Temporal and retain cleanup/run data, but none decodes or executes `umpire-experiment/v1`. |
| C7 evidence/conformance | Partial + separate | Current Lean checks properties over model traces and exports observation requirements. Go/Umpire3 evidence systems exist separately; no current-model observation program or live qualified result exists. |
| C8 exploration | Partial | Deterministic finite exhaustive/shortest-style planning is built. The named coverage-guided policy only seed-rotates enumeration; it does not consume coverage state. Broader campaign exploration exists separately. |
| C9 SDK participant | Separate | Umpire3 has participant protocols and a Temporal SDK adapter; the current model emits no participant program or binding. |
| C10 replay/promote | Separate | Go campaign pipelines minimize, replay, and propose regressions, but cannot read the current artifact or generate a Lean regression from it. |
| C11 formal checks | Partial + separate | Current Lean provides proofs, finite completeness, bounded verification, and model counterexample search. Umpire3 has mature receipts and optional family-scoped Veil support; the current model has neither an explicit first-order checker view nor a Veil binding. |
| C12 qualification | Separate | Environment, canary, and release-qualification foundations exist in Go/Umpire3. No local, CI, remote, or canary qualification path accepts the current artifact. |

| Milestone | Status | Summary |
| --- | --- | --- |
| A: describe real tests | In progress | The model-only C1-C4 slice, shared basic Nexus teaching target, and one-regression C5 projection pilot work; the two walkthroughs, combinatorial authoring, broader projection catalog and promotion, and usability evidence remain. |
| B: check real Temporal | Not integrated | No current-artifact decoder, local driver binding, live evidence interpreter, or qualified result path exists. |
| C: find things | Foundations only | Finite planning and separate campaign tooling exist, but discovery, minimization, replay, and promotion are not composed end to end. |
| D: reuse the investment | Separately advanced | Formal, optional Veil, and qualification machinery are substantial in Umpire3, but current-model checker-view binding and the active R6/R7 qualification work remain open. |

### Active Flow delivery plans

Planning status is tracked separately from implementation status above. A reviewed plan does not
make its component built.

| Slice | Flow spec | Plan status | Bounded ownership |
| --- | --- | --- | --- |
| C7 offline observation | `fn-4-umpire-observation-and-semantic-verdicts` | Reviewed: Ship; six tasks ready | Closed reusable mapping expressions, bounded synthetic evidence qualification, coordinate-complete derivations, unchanged pure Property evaluation, and one Temporal-owned synthetic profile. Live collection and conformance remain separate. |
| C3/C5 catalog and exact promotion | `fn-5-umpire-discovery-promotion-and-artifact` | Reviewed: Ship; seven tasks ready; dependency synchronized | Checked eight-query plus one authored-space semantic catalog closure, generated glossary/index and list/explain UX, lineage-linked exact in-memory promotion with compiled Temporal source bindings, and a two-regression thin projection set. Persisted readers/migrations remain wholly owned by fn-18; live replay/minimization remain separate. |
| Milestone A pilot gate | `fn-14-milestone-a-pilot-baseline-and-lean` | Reviewed: Ship; six tasks ready | Frozen eight-defect/twelve-mutation baseline, provider-free cost and coverage measurement, exactly three fresh Agentworkflow authoring trial slots, and a strict `LEAN_FIRST_GO` / `FACADE_FOLLOW_UP` / `NO_GO` / `INCONCLUSIVE` receipt. No facade or live-runtime implementation. |
| C1/C2 standalone input catalogs | `fn-15-standalone-api-and-config-input-catalogs` | Reviewed: Ship; seven tasks ready | Complete mechanical API facts, exact bounded current-model selection, all generated settings plus six owned semantic uses, and one shared exact list/explain/check engine. Drift verification, live lookup, runtime, and conformance remain separate. |
| C3/C4 authored variation spaces | `fn-16-authored-variation-spaces-and` | Reviewed: Ship; six tasks ready | Checked finite axes/choices, request-only faults, seek-only coverage goals, canonical metadata for fn-5, dependent-kernel point lowering, and atomic bounded batch compilation. Coverage scoring, artifact reading, runtime, and conformance remain separate. |
| C8 bounded semantic exploration | `fn-17-bounded-semantic-exploration-and` | Reviewed: Ship; seven tasks ready | Pure selection over fn-16's atomic bounded universe with exhaustive, pair/t-wise, seeded-random, proof-reduced symmetry, real coverage-guided state/report/resume, and pinned precedence. Runtime, evidence, persisted readers, replay, minimization, and promotion remain separate. |
| C4/C7/C8 versioned artifact boundary | `fn-18-versioned-umpire-artifact-boundary` | Reviewed: Ship; eleven tasks ready | Exact Lean-owned v1 wire schemas, strict bounded Go admission, embedded semantic-reference resolution, compatible ExperimentSpec reading, runtime/run/evidence/result/coverage transports, complete-set validation, honest named migrations, and immutable atomic publication. Execution, evidence collection/interpretation, coverage scoring, replay, promotion, and qualification remain separate. |
| C6/C9 bounded local execution | `fn-19-bounded-local-temporal-execution-and` | Reviewed: Ship; nine tasks ready | One fn-18-admitted caller-closure input set, domain-neutral five-phase runtime/participant boundary, production-safe `temporaltest` lifecycle, closed ephemeral authority, exact Nexus SDK force-close participant, four-source raw evidence, and an admitted/published run set. Semantic interpretation, Result/conformance, replay, promotion, remote/CI/canary, and qualification remain separate. |
| C7 local semantic conformance | `fn-20-local-execution-semantic-conformance` | Reviewed: Ship; seven tasks ready | One exact four-member local run set passes through a private bounded Go-to-Lean checker bridge, the existing pure Observation/Property authorities, independent operational/qualification/semantic statuses, fail-closed evidence mutations, and an admitted/published six-member SemanticEvidence/Result set. Replay, promotion, formal checking, and non-local/release qualification remain separate. |
| C4/C6/C7/C9 deterministic Nexus negative control | `fn-21-nexus-duplicate-observation-control` | Reviewed: Ship; seven tasks ready | One exact request-only duplicate-delivery fault lowers to a second closed local program/configuration, drives one normal requested/completed cancellation lifecycle, records mechanical callback count one plus one labeled synthetic contribution, and lets the existing conformance authority report a qualified uniqueness-only violation. Generic fault infrastructure, a new artifact family, replay, promotion, and non-local qualification remain separate. |
| C10 deterministic replay, minimization, and reviewed promotion | `fn-22-deterministic-replay-semantic` | Reviewed: Ship; eight tasks ready | The exact fn-21 six-member violation set is admitted, reproduced twice, compared by a closed normalized violation signature, reduced only through checked action/order/fault edits, explained by a non-destructive evidence core, and bound to one statically elaborated fn-5 proposal candidate. One bounded local command emits an exact report/progress contract and review-only Lean source; no new persisted artifact family, dynamic campaign surface, automatic regression installation, or non-local qualification is introduced. |
| C11 Veil compatibility and adoption gate | `fn-23-veil-toolchain-compatibility-and` | Reviewed: Ship; six tasks ready | Two exact pinned Veil candidates are evaluated under Lean 4.33.1 only through an opt-in Linux/aarch64 diagnostic with frozen Git/npm/solver/Lean/Zig/Node/runtime closures, no-network execution, honest solver trust, candidate-scoped measurements, and one canonical adopt/defer/inconclusive receipt. No Veil dependency, semantic claim, default build integration, generated source, or production surface is introduced. |
| C11 Lean-native receipts and canonical replay | `fn-24-lean-native-verification-receipts-and` | Reviewed: Ship; six tasks ready | A reusable Temporal-free formal module runs the exact checked Query/dependent kernel itself, emits honest bounded-search receipts, and admits a violation only after the candidate replays through the canonical kernel, Behavior, and pure Property evaluator. One caller-closure verify receipt and one family-test negative control prove both paths; Veil, ExperimentSpec, runtime, promotion, and qualification remain separate. |
| C11 optional CallerClosure checker binding | `fn-25-optional-callerclosure-veil-binding-and` | Reviewed: Ship; six tasks ready; depends on fn-23 and fn-24 | One completed compatibility receipt freezes an adopt or defer branch. Adoption adds one family-owned finite view with bidirectional correspondence, one exact optional handwritten declaration, a non-breaking external receipt v2, and mandatory canonical replay; defer/inconclusive adds no dependency, source, command, or placeholder claim. The reusable Umpire package remains Temporal/Nexus/checker-name free. |
| C12 local qualification receipt | `fn-26-local-qualification-receipts-and-staged` | Reviewed: Ship; six tasks ready; depends on fn-14, fn-18, fn-19, and fn-20 | One offline local-only profile admits the exact six-member conformance set plus the strict pilot decision, preserves operational/evidence/semantic status and cleanup independently, and publishes one environment-scoped QualificationReceipt in a non-breaking ArtifactSet v2. It acquires no authority and reserves no CI/remote/canary vocabulary; those profiles require reviewed schema versions. |
| C12 hermetic CI execution and qualification | `fn-27-hermetic-ci-execution-and-qualification` | Reviewed: Ship; nine tasks ready; depends on fn-14, fn-18, fn-19, fn-20, and fn-26 | One exact CI RuntimeConfiguration reuses the disposable loopback runtime and canonical conformance authority for the same ExperimentSpec, then publishes a versioned CI receipt/set with two-stage isolation evidence and bounded self-reported GitHub Actions provenance. One manual least-privilege workflow is isolated from default/release CI; remote, canary, authenticated provenance, and release eligibility remain separate. |
| C12 authorized remote staging black-box qualification | `fn-28-authorized-remote-staging-black-box` | Reviewed: Ship; eleven tasks ready; depends on fn-14, fn-18, fn-19, fn-20, fn-26, and fn-27 | One fixed protected environment supplies an in-memory mTLS authority for a preallocated staging namespace and Nexus endpoint. A server-enforced lease, idempotent one-mutation participant, public-only evidence mapping, cleanup-before-conformance ordering, ephemeral recovery/reconcile path, and versioned remote receipt/set preserve the same ExperimentSpec without target selection or internal evidence. The claim is staging-scoped and never canary or release eligible. |
| C12 bounded production canary qualification | `fn-29-bounded-production-canary-execution-and` | Reviewed: Ship; thirteen tasks ready; depends on fn-14, fn-18, fn-19, fn-20, fn-26, fn-27, and fn-28 | One trusted-ref-gated protected workflow runs the same ExperimentSpec against a fixed, dedicated production-canary namespace and Nexus route with no customer traffic, faults, deployment, configuration, or routing mutation. Exact lease reuse/fencing, idempotent mutation, public evidence, cleanup-reserved RPC accounting, versioned recovery, and strict receipt/set successors keep every canary receipt non-release-eligible. Receipt bytes are inspectable but not self-authenticating; release aggregation and its trusted artifact channel remain separate. |
| C12 release evidence graph and manual authorization | `fn-30-release-evidence-graph-and-manual` | Reviewed: Ship; seven tasks ready; depends on fn-14, fn-18, and fn-26 through fn-29 | One signed seven-slot evidence index authenticates complete or explicitly held inputs, externally signed build/deployment attestations bind one immutable server candidate, and one fixed Lean policy produces an inspectable qualified/held/rejected graph. Separate protected owner roles create append-only approve/deny/revoke decisions; neither qualification nor authorization can deploy, promote, route, configure, or roll back. |

Bottom line: the component decomposition still holds, but the current implementation is not an
end-to-end Umpire pipeline. It is a model-only slice ending at deterministic artifact inspection.
The next runtime seam is a strict decoder/validator and local execution adapter for
`umpire-experiment/v1`. Veil belongs at a different seam inside C11: an optional, family-owned
Lean checker target bound by proof to an explicit canonical view, never a generated backend or a
second semantic authority.

## 1. Organizing principle

Umpire should be a collection of independently useful tools connected by explicit, versioned
artifacts. Each component should have:

- one narrow responsibility;
- a stable input and output contract;
- a public API or coherent `umpire` CLI surface;
- deterministic, inspectable artifacts;
- fixtures that allow it to be developed without the full system; and
- no independent copy of Lean-owned behavioral semantics.

The target pipeline is:

```text
Protobuf descriptors --------+
Dynamic-config declarations -+
                              v
                    semantic catalogs
                              |
                              v
                  Lean model and specs
                              |
                              v
               regression / scenario space
                              |
                              v
                    ExperimentSpec
                    +---------+---------+
                    |                   |
                    v                   v
             model exploration      execution
                                        |
                                        v
                                ExperimentRun
                                        |
                                        v
                                  raw evidence
                                        |
                                        v
                                semantic evidence
                                        |
                                        v
                                      Result
                                        |
                                        v
                                  replay bundle
```

The solid current `model/` path ends at deterministic `ExperimentSpec` inspection. Runtime
execution, evidence qualification, replay, and promotion remain separate implementations. The
single public command can eventually remain `umpire`; the decomposition is an internal and
artifact-level boundary, not a requirement to ship many unrelated binaries.

## 2. Artifact contracts

These artifacts are the seams between components:

| Artifact | Purpose | Current state |
| --- | --- | --- |
| API catalog | Mechanical Protobuf schema knowledge and field dispositions | Generated Lean structure exists in `model/Temporal/API.lean` and `model/Temporal/API/`; a separate catalog, field-disposition artifact, and drift report do not. |
| Config catalog | Keys, types, defaults, precedence, scope, and declared classification | The complete initialized registry snapshot and resolution fixtures exist in `model/Temporal/DynamicConfig/`; handwritten classifications and typed uses exist for selected Callback and Matching settings. |
| Semantic catalog | Lean-owned resources, actions, properties, observations, targets, and hashes | Checked declarations and canonical projections exist in `model/Umpire/`; there is no consolidated persisted catalog or list/explain interface. |
| Regression/space | Named regressions and exploration spaces over Lean semantics | Checked `Property`, `Behavior`, and `Query` values exist, with reusable Switch and Temporal caller-closure examples; there is no persisted regression catalog. |
| ExperimentSpec | Environment-independent executable specification with a drive plan, properties, requirements, bounds, omissions, provenance, and semantic hashes | `umpire-experiment/v1` and `umpire-drive-plan/v1` are implemented and deterministically inspected. Choices, variants, and faults are represented but not yet authored or populated. Runtime reading and migrations are explicit omissions. |
| ExperimentRun | One realized execution binding an `ExperimentSpec` to an environment, seed, controls, receipts, and cleanup outcome | Run/artifact concepts exist in the Go Umpire implementations, but not for the current `model/` `ExperimentSpec`. |
| Raw evidence | Typed implementation facts, receipts, omissions, source positions, and causal references | Implemented in the Go Umpire baselines; not connected to the current `model/` artifact. |
| Semantic evidence | Lean-defined interpretation of raw facts | Implemented in Umpire3's model/runtime path and partially mirrored by generic Go interpretation; not generated from the current `model/` declarations. |
| Result | Qualified established, violated, unknown, conflict, or unsupported claims | Qualified result models exist in the Go Umpire baselines; no current-model conformance result is produced. |
| Replay bundle | ExperimentSpec, ExperimentRun, evidence, result, bounds, and provenance | Campaign/replay artifacts exist in the Go Umpire baselines; they do not accept `umpire-experiment/v1`. |
| Veil binding | Source-bound correspondence between an explicit canonical first-order view and family-authored Veil declarations | Umpire3 exports `umpire3/veil-binding/v1` with declaration mappings, source/view digests, backend revision, semantic relation, and trust data. The current model has no corresponding checker view or binding. |
| Verification receipt | Checker target, bounds, trust mode, proof/counterexample, and provenance | Umpire3 has checker receipts and release evidence. The current `model/` produces `PlannerRun` results, not a verification receipt. |

Every persisted artifact should carry a format version. Semantic artifacts should additionally carry
source and semantic digests, declared omissions, and enough provenance to reject incompatible or
stale consumers. The current `ExperimentSpec` and `DrivePlan` meet this baseline; the missing
reader/migration/runtime boundary must preserve it.

## 3. Components

### C1. Protobuf API importer

**Responsibility:** mechanically project Temporal API structure from descriptor sets without
inventing product meaning.

**Status: implemented projection, incomplete component surface.** `umpire-gen-lean-api` merges the
public, server API, internal, and CHASM descriptor sets and deterministically owns
`model/Temporal/API.lean` plus the complete `model/Temporal/API/` directory. It generates typed
messages, enums, maps, `oneof`s, presence, recursion links, and service methods, with Go fixture and
publication tests. It does not emit a separate API catalog, field-disposition file, drift report, or
explain interface.

```text
descriptor set -> generated Lean wire declarations + API catalog + drift report
```

Current interface:

```text
make umpire-gen-lean-api
go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api
make umpire-build-model
```

The importer owns messages, fields, enums, presence, `oneof`s, maps, nested types, digest-and-size
byte values, depth-bounded recursive references, supported well-known types, RPC shape, streaming,
and method deprecation. Semantic identity, ordering, completion, and absence remain handwritten
Lean interpretations. The current `model/` generator imports all declarations in its supplied
descriptor sets. `UMPIRE_LEAN.md` separately requires bounded selected wire surfaces for Umpire3,
so selection policy is still an explicit convergence decision rather than a settled shared
contract.

### C2. Dynamic-config importer

**Responsibility:** generate typed configuration knowledge without deciding product semantics.

**Status: implemented catalog and selected semantic use.**
`umpire-gen-lean-dynamic-config-catalog` discovers registration sites, snapshots the initialized
production registry, and deterministically owns
`model/Temporal/DynamicConfig.lean` plus the complete `model/Temporal/DynamicConfig/` directory.
The generated catalog includes normalized keys, schemas/codecs, defaults, precedence policies,
constraints, fixtures, and a catalog identity. `model/Temporal/System/Configuration/` implements
typed resolution and validation; Callback and Matching modules demonstrate selected handwritten
classifications, sampling points, and change effects.

```text
Temporal dynamic-config declarations -> generated ConfigKey declarations + config catalog
```

Current interface:

```text
make umpire-gen-lean-dynamic-config-catalog
go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-dynamic-config-catalog
make umpire-build-model
```

The catalog records key, type, default, precedence/scope, and description. Handwritten classification
separates feature, validation, semantic, timing, topology, performance, and observability impacts and
records whether a value is sampled live, at creation, per request, per task, or after restart.
Cross-language resolution fixtures are implemented. A standalone catalog artifact and
sync/check/explain command surface are not.

### C3. Lean authoring DSL

**Responsibility:** provide approachable declarations for regressions and exploration over
Lean-owned models.

**Status: implemented v1 core, with authoring gaps.** Reusable, Temporal-independent modules under
`model/Umpire/` now provide checked `Property`, `Behavior`, and `Query` declarations over composed
targets and finite kernels. They cover setup, capabilities, actions, exact traces, occurrence
bounds, partial ordering, sequences, adjacency, properties, query bounds, and deterministic search
policy. The synthetic Switch and Temporal Workflow-Nexus caller-closure scenarios exercise the
full authored-to-checked lifecycle without a running server.

```text
.lean specifications -> checked regression/space catalog
```

Current interface:

```text
make umpire-build-model
make umpire-check-regression
```

The remaining language scope is authored variation axes, faults, explicit coverage goals, and a
catalog/list/explain surface. Outcomes deliberately remain target-owned rather than author-owned.
Regression, witness, counterexample, verification, and exploratory selection already share the
same checked query path.

Lean-first behavioral authoring is now the implemented choice. A generated Go facade remains
conditional on usability evidence; in either case, Lean owns the resulting semantic object.

### C4. Semantic ExperimentSpec compiler

**Responsibility:** turn a checked regression or selected point in a scenario space into one or more
bounded `ExperimentSpec`s without requiring Temporal.

**Status: implemented for one selected model trace.** `Umpire.Planning.plan` consumes a checked
query and proof-carrying incremental kernel, distinguishes verified, selected, absent,
budget-exhausted, unsatisfiable, and invalid outcomes, and may produce a canonical
`umpire-experiment/v1`. `Temporal.Tool.Inspect` exposes checked-in Switch and caller-closure
scenarios and emits deterministic JSON with structured diagnostics.

```text
checked spec + model target + bounds -> ExperimentSpec[]
```

Current interface:

```text
make umpire-inspect SCENARIO=workflow-nexus.query.exact-action-caller-closure
make umpire-inspect SCENARIO=switch.query.exact-action
```

The current `ExperimentSpec` records bindings and symbolic roles, preconditions, requested actions,
model-owned outcomes and resulting states, a linear extension, checkpoints, properties, observation
requirements, bounds, omissions, provenance, and semantic hashes. Compilation distinguishes a
requested action from its model-owned outcome. Choices, variants, and faults are reserved fields
currently emitted empty; configuration is present only through semantic setup/state rather than a
separate runtime configuration contract. Artifact reading, migrations, and runtime consumption are
not implemented.

### C5. Go test and documentation generator

**Responsibility:** expose stable Lean regressions through familiar, deterministic projections.

**Status: implemented for one stable regression.** The production inspector and canonical fixture
for `workflow-nexus.query.exact-action-caller-closure` generate one ordinary Go test and one readable
Markdown projection. The checked-in outputs are
`tools/umpire/regression/catalog_generated_test.go` and
`model/Temporal/Tool/Generated/Regressions.md`.

```text
one stable inspected regression -> thin _test.go wrapper + readable documentation
```

Repository interface:

```text
make umpire-gen-regression-projections
make umpire-check-regression-projections
make umpire-check-regression
```

The generation target publishes both outputs transactionally. The focused check regenerates them
into an isolated root, byte-compares the complete pair, and runs the generator and fixture-backed
verifier tests; the stable regression gate includes that check. The generated test calls only
`RequireProjection`, and both outputs carry the Lean-owned identity, provenance, fixture, property
and observation-requirement identities, and semantic fingerprint without copying behavioral
procedure.

This remains a projection-only pilot: it neither executes Temporal nor supplies execution evidence
or conformance. The manifest is closed to one stable regression. The reviewed
`fn-5-umpire-discovery-promotion-and-artifact` plan owns the broader checked catalog, list/explain,
exact in-memory promotion, and two-regression projection slice, but none of that plan is implemented
yet. Persisted readers and migrations remain a separate fn-18 boundary.

### C6. Execution runtime

**Responsibility:** realize one `ExperimentSpec` against an environment and return an
`ExperimentRun` plus raw evidence without deciding whether Temporal was correct.

**Status: existing Go baseline, not integrated with the current model.**
`common/testing/umpire`, `tools/umpire2`, and `tools/umpire3/execution` already implement bounded
preparation, realization, observation, isolation, cleanup, and run artifacts for their own
contracts. None currently reads `model/Umpire.Artifact.ExperimentSpec`; the current artifact
therefore lists execution evidence and runtime order among its omissions.

```text
ExperimentSpec + Environment -> ExperimentRun + raw evidence
```

Proposed CLI:

```text
umpire run <experiment-or-regression> --environment <profile>
```

The runtime owns preparation, bounded action realization, evidence capture, control receipts,
source closure, cleanup, persistence, and isolation. Environment adapters may include local
in-process, test cluster, gRPC, remote, and authorized canary profiles, but only the local adapter is
needed initially.

### C7. Evidence interpretation and conformance

**Responsibility:** interpret raw implementation facts through Lean-owned observation programs and
compare them with the `ExperimentSpec` requirements and realized `ExperimentRun`.

**Status: existing baselines, not integrated with the current model; the offline semantic boundary
has a reviewed implementation plan.** Generic facts, lifecycle and
relation state, evidence profiles, qualified claims, and fail-closed handling exist under
`common/testing/umpire`; Umpire3 also has Lean-owned observation programs and a live conformance
path. The current `model/` exports property observation requirements, but does not export an
observation program or consume facts to produce a result.

`fn-4-umpire-observation-and-semantic-verdicts` now plans the independently implementable offline
slice: a reusable checked Observation mapping language, bounded synthetic evidence qualification,
coordinate-keyed derivations, semantic verdicts through the unchanged Property evaluator, and one
synthetic Temporal profile. It deliberately does not collect live evidence or claim live
conformance; those remain downstream work.

```text
ExperimentSpec + ExperimentRun + raw evidence -> semantic evidence -> qualified Result
```

Proposed CLI:

```text
umpire evidence normalize <raw-evidence>
umpire result check <experiment> <evidence>
umpire result explain <result>
```

This component owns evidence identity, causal matching, source-local ordering, closure, ambiguity,
conflicts, omissions, and diagnostic obligations. Missing or ambiguous evidence fails closed. It
must remain separate from execution so fixtures can test output, history, link, log, metric, and span
interpretation without a running server.

### C8. Exploration engine

**Responsibility:** select useful `ExperimentSpec`s from a Lean-owned scenario space under a strategy
and budget.

**Status: finite model planning implemented; broader exploration is partial.** The current Lean
planner supports exhaustive, breadth-first, and shortest policies with separate behavior and
candidate-evaluation bounds, deterministic tie-breaking, completeness evidence, and
instrumentation. Its policy vocabulary also names `coverage-guided`, but that branch currently
uses the seed only to rotate canonical setup, action, and outcome enumeration; it has no coverage
state, scoring, or coverage report. The Go campaign baseline separately supports bounded
matrix/lifecycle selection, sampling, semantic coverage, and pinned regressions. These paths do
not yet share the current `ExperimentSpec`, and pairwise, t-wise, seeded-random, symmetry-reduced,
and campaign-wide coverage selection are not all implemented in the current Lean planner.

```text
scenario space + strategy + bounds + coverage state -> selected ExperimentSpecs + coverage report
```

Proposed CLI:

```text
umpire explore <space> --strategy <exhaustive|pairwise|t-wise|random|coverage> --budget <n>
```

The target engine adds pairwise, t-wise, seeded random, transition/relation/outcome coverage,
symmetry reduction, and campaign-wide coverage-guided selection to the implemented finite planner.
It can operate entirely on the model before composing with C6 for live execution. Pinned
regressions always run independently of the exploration budget.

### C9. SDK participant runtime

**Responsibility:** realize semantic participant commands through Kitchensink or an SDK and return
structured observations.

**Status: implemented in Umpire3, not integrated with the current model.** Umpire3 has a participant
protocol and Temporal SDK participant adapter. Umpire2 also has Go action realizers. The current
`model/` does not emit a participant program or bind one into `umpire-experiment/v1`.

```text
ParticipantProgram -> SDK behavior + raw observations
```

Proposed CLI:

```text
umpire participant run <program>
```

The participant protocol should be small and language-neutral enough to add Go, Java, TypeScript, or
other SDK realizers without changing the semantic model or core execution runtime.

### C10. Replay, minimization, and promotion

**Responsibility:** reproduce a failure, reduce it to the smallest relevant semantic coordinates,
and turn a stable discovery into a reviewed regression.

**Status: existing Go baselines, not integrated with the current model.**
`common/testing/umpire/campaign` implements bounded selection, monotonic reduction, replay, and
deterministic regression candidates; Umpire3 has its own campaign/replay/promotion pipeline. The
current `ExperimentSpec` explicitly omits artifact reading and promotion, and no path proposes or
generates a Lean regression from one of its failures.

```text
ReplayBundle -> replayed result -> minimized bundle -> Lean regression proposal
```

Proposed CLI:

```text
umpire replay <bundle>
umpire minimize <bundle>
umpire promote <bundle>
```

This component consumes artifacts rather than runtime internals. It should minimize actions,
ordering constraints, faults, configuration dimensions, and evidence while preserving the same
qualified violation.

### C11. Formal checker integrations

**Responsibility:** run an explicitly declared model target under bounds and return a
provenance-rich receipt or counterexample.

**Status: model-local foundation plus a separate mature Umpire3 implementation.** The current
`model/` has Lean proofs, checked target composition, finite completeness evidence, exhaustive
verification, witness/counterexample search, and canonical planner results. It does not yet produce
a verification receipt or replayable checker counterexample. Umpire3 already provides exact and
native checked certificates, Lean temporal checking, optional embedded Veil declarations, receipts,
mutation gates, and release evidence; those artifacts are not bound to the current model's semantic
digests.

```text
model target + bounds + declared view -> verification receipt or counterexample
```

Proposed CLI:

```text
umpire verify <target>
```

Lean-native checking is the default. Counterexamples must replay through canonical Lean semantics
before promotion. Although the chat describes pluggable backends, the active roadmap requires Veil
to remain a Lean library and embedded DSL, not a second semantic authority or generated target.

The approved current-model Veil direction is detailed in [UMPIRE4_DSL.md](UMPIRE4_DSL.md#optional-veil-checking). Veil is
optional per model family and property. Its handwritten declarations live beside the owning family
in the primary Lake project, behind a focused import and test aggregate. A Lean-owned binding must
relate initial states, actions, transitions, and the checked property to an explicit canonical
first-order view. Go may isolate the checker and transport strict, digest-bound artifacts; it must
not generate Veil source or interpret Veil semantics. Results preserve testing, trusted-SMT,
reconstructed-proof, and kernel trust as different claim classes. Every counterexample must replay
through the canonical Umpire transition kernel before it can support violation or promotion.

The first adoption gate is toolchain compatibility: the current model uses Lean 4.33.1 while the
existing Umpire3 Veil project uses Lean 4.28.0. After a pinned Veil revision builds behind an
optional import, one Nexus family may add a meaningful binding, nearby semantic mutation, strict
receipt, and canonical replay. Veil does not enter `ExperimentSpec`, runtime execution, production
binaries, or the default regression gate until its deterministic build and developer-cost budgets
are accepted.

### C12. Deployment and qualification

**Responsibility:** realize the same `ExperimentSpec` under authorized remote profiles and assemble
qualified release claims without erasing environment-specific trust and omissions.

**Status: generic and Umpire3 foundations exist; current-model qualification is not implemented.**
The Go Umpire trees include environment profiles, guarded canary controls, qualification models, and
release evidence. `UMPIRE_LEAN.md` R6 still tracks independently retained local, CI, remote, and
public-gRPC qualification. The current `umpire-experiment/v1` is not accepted by these paths.

```text
ExperimentSpec + deployment profile + authority -> ExperimentRun + qualified Result / release graph
```

Proposed CLI:

```text
umpire qualify <release-or-result-set>
```

CI, staging, cloud, black-box, and canary execution reuse the same semantic `ExperimentSpec` but have
different evidence capabilities, credentials, cleanup policies, blast-radius controls, and claim
strength. This component is downstream of successful local execution and conformance.

## 4. Component dependency map

```text
C1 API importer --------+
                        |
C2 config importer -----+----> semantic catalogs
                                  |
                                  v
                          C3 authoring DSL
                                  |
                                  v
                         C4 experiment compiler ----> C5 Go/docs generator
                                  |
                     +------------+-------------+
                     |                          |
                     v                          v
              C8 exploration              C11 formal checks
                     |
                     v
              ExperimentSpec
                     |
             +-------+--------+
             |                |
             v                v
       C6 execution      C9 SDK participant
             |                |
             +-------+--------+
                     v
        ExperimentRun + raw evidence
                     |
                     v
              C7 conformance
                     |
                     v
                   Result
                     |
             +-------+--------+
             |                |
             v                v
      C10 replay/promote  C12 qualification
```

C1, C2, C3, C5, C8, C9, C10, and C11 can be developed largely with fixtures or synthetic models.
C4 and C7 are the principal semantic integration points. C6 is the principal Temporal integration
point. C12 depends on the local path being credible first.

The current implementation cut is:

```text
C1/C2 generated structure
        -> C3 checked target/property/behavior/query
        -> C8 finite planning
        -> C4 umpire-experiment/v1 inspection
        -> [missing current-model runtime adapter]
        -> C6-C12 baselines in other Umpire trees
```

C5's broader catalog and exact-promotion slice now has a reviewed Flow plan, while its implementation
remains open. The most important runtime architectural gap is still a versioned adapter that can
decode and validate `umpire-experiment/v1` without restating its Lean-owned semantics, then return
run/evidence/result artifacts bound to the same semantic identities.

## 5. Delivery milestones

### Milestone A: Lean can describe real Temporal tests

Components: C1-C5, limited to the API/config/model material needed by the pilot.

**Status: in progress.** C1-C4 have a usable model-only vertical slice, and C5 has a checked-in
one-regression projection pilot. The implemented DSL does not yet cover combinatorial axes or
authored faults, and broader projection catalog and promotion work remains.

Deliverable:

```text
concise Lean regression
        -> semantic validation
        -> explained ExperimentSpec
        -> ordinary generated Go test
```

Current evidence:

- a generated full Temporal Protobuf structural projection;
- a generated full initialized dynamic-config catalog with cross-language resolution fixtures;
- a reusable Temporal-independent DSL tested with a synthetic Switch target;
- a shared basic Nexus lifecycle teaching target with checked composition and finite planning;
- a Workflow-Nexus caller-closure model with target-owned cancellation, output, and ownership
  observations; and
- deterministic `umpire-experiment/v1` inspection, golden fixtures, structured diagnostics, and a
  focused regression gate.

Remaining exit evidence:

- one simple Nexus regression;
- one combinatorial exploration declaration;
- deterministic Go wrapper generation; and
- ordinary Go engineers can modify the examples after a short orientation, or the evidence selects
  a generated Go authoring facade.

No new runtime is required if the compiled `ExperimentSpec` can initially use existing test
machinery.

### Milestone B: The model checks real Temporal

Components: C6, C7, C9, and one local environment adapter.

**Status: not started for the current model artifact.** Relevant runtime, participant, and evidence
capabilities exist elsewhere, but there is no decoder/validator or execution binding from
`umpire-experiment/v1` to those implementations.

Deliverable:

```text
the same checked semantic test
        -> local Temporal execution
        -> ExperimentRun
        -> raw causal evidence
        -> qualified state and output result
```

Exit evidence:

- one Nexus cancellation or dispatch-failure path executes end to end;
- one SDK participant realizes part of the scenario;
- one API/history output property and one diagnostic obligation are checked;
- missing, corrupted, duplicated, or causally unrelated evidence fails closed; and
- cleanup and evidence-source closure are explicit in the Result.

This is the first major proof-of-value milestone.

### Milestone C: The model finds things

Components: C8 and C10, composed with Milestone B.

**Status: partial foundations only.** Pure finite planning exists in the current model and campaign,
minimization, replay, and promotion exist in separate Go implementations. They are not composed
through the current artifact or a live Milestone B path.

Deliverable:

```text
scenario space
        -> bounded exploration
        -> previously unwritten interaction
        -> minimized replay
        -> promoted permanent regression
```

Exit evidence:

- pinned historical bugs and realistic mutations fail for the intended semantic reason;
- exploration reports semantic rather than raw case-count coverage;
- at least one discovered interaction is minimized and replayed deterministically; and
- promotion generates a stable Lean regression and thin Go wrapper.

An unknown product defect is a high-value outcome, not a prerequisite for completing the milestone.

### Milestone D: Reuse the investment

Components: C11, C12, and additional execution/control profiles justified by observed needs.

**Status: separately advanced, not converged.** Umpire3 contains most formal-checking and release
qualification machinery, while generic Go Umpire includes environment and canary controls. Current
model digest binding and independently retained deployment qualification remain open; the active
details live in `UMPIRE_LEAN.md` R6 and R7.

Deliverable:

```text
the same model and ExperimentSpec
        -> stronger formal checks
        -> CI/remote/black-box/canary profiles
        -> environment-specific ExperimentRuns
        -> qualified release evidence
```

Exit evidence:

- every claim exposes checker trust, bounds, omissions, environment, and evidence profile;
- remote results remain bound to the same semantic hashes as local results;
- counterexamples replay through canonical Lean and, where applicable, real execution; and
- every Veil-owning family has a source-bound canonical-view relation, mutation evidence, explicit
  trust and axiom inventory, and a canonical replay gate;
- operational authority, cleanup, and blast-radius controls are explicit.

Additional current-model formal integration, production-canary binding, and deterministic Go
scheduling remain downstream of a credible local slice.

## 6. Pilot sequence

The Nexus caller-closure behavior, reusable model DSL, generated structures, and v1
`ExperimentSpec` are now implemented. The remaining bounded proof-of-value sequence is:

1. Finish the simple Nexus authoring examples and assess whether Lean-first authoring is usable
   enough to defer C5's generated Go facade.
2. Record historical bugs, realistic mutations, current hand-authored coverage, comparison costs,
   time budget, and stop/go thresholds before beginning live integration.
3. Define the minimal versioned decoder and compatibility checks for `umpire-experiment/v1`, plus
   the `ExperimentRun`, raw-evidence, semantic-evidence, and `Result` schemas needed by caller
   closure. Do not introduce another semantic IR.
4. Bind one local Temporal environment and one SDK participant to the selected scenario, preserving
   requested attempts, model-owned outcomes, semantic hashes, evidence closure, and cleanup.
5. Run evidence corruption and ambiguity fixtures before treating a green live run as conformance.
6. Compose the current finite planner with campaign minimization, replay, and reviewed promotion for
   the same scenario.
7. Measure defect detection, feedback latency, semantic interaction coverage, marginal authoring
   cost, maintenance cost, evidence quality, execution cost, and usability by another engineer.
8. Continue current-model qualification work only if the evidence meets the predeclared thresholds;
   follow `UMPIRE_LEAN.md` for the separate Umpire3 R6/R7 release path.

## 7. Deferred work

These remain architectural possibilities, not prerequisites for the current-model proof:

- replacing the current full-descriptor API projection with a bounded selection policy shared with
  Umpire3;
- standalone API/config catalog, explain, and drift-report command surfaces;
- production canary binding for `umpire-experiment/v1`;
- generalized remote qualification for the current model;
- additional current-model formal checker integrations not required by the pilot;
- Veil adoption beyond one compatibility spike and one meaningful family binding;
- exhaustive internal tracepoints;
- deterministic Go scheduler control or full DST; and
- low-level lock and scheduler modeling in feature semantics.

Generated Veil source, a second Lake project, a checker-neutral semantic IR, blanket Veil adoption,
and Veil in production runtime paths are rejected designs, not deferred work.

The detailed rationale and the other two design conversations are synthesized in
[UMPIRE_CHATS.md](UMPIRE_CHATS.md).
