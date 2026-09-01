# Umpire components and historical delivery inventory

Inventory snapshot: reconciled 2026-08-31 against the current `model/` tree, its generators, and
the existing Go Umpire implementations. The component boundaries were originally extracted from the
[Inspect Umpire Branch](https://chatgpt.com/share/6a8b71cb-74e4-83e8-947a-c2f6d595fefc)
design conversation. This remains an architectural inventory, not an approved implementation plan.
[`UMPIRE4_SPEC.md`](UMPIRE4_SPEC.md) owns normative rules,
[`UMPIRE4_ORDER.md`](UMPIRE4_ORDER.md) owns remaining delivery order, and
[`UMPIRE4_SPEC_COMPS.md`](UMPIRE4_SPEC_COMPS.md) owns the current component architecture.

Component status in this document means integration status for the current `model/` semantic
pipeline. Functionality in `common/testing/umpire`, `tools/umpire2`, or `tools/umpire3` is called out
as an existing baseline, but is not considered integrated until it consumes the current
`umpire-experiment/v2` artifact and preserves its Behavior Fingerprints.

## TL;DR

Status terms: **built** means implemented in the current `model/` pipeline; **partial** means only
part of the component contract is built; **separate** means a working implementation exists in
another Umpire tree but does not consume current-model artifacts; **missing** means no corresponding
implementation was found.

| Component | Status | Summary |
| --- | --- | --- |
| C1 API importer | Partial | Deterministic full-descriptor Lean projection is built. A standalone catalog, complete field dispositions, drift report, explain surface, and shared bounded-selection policy are missing. |
| C2 config importer | Partial | The initialized production registry, typed generated settings, identities, and Go-produced resolution fixtures are built. Product classifications currently cover selected Callback and Matching uses; no standalone explain/check surface exists. |
| C3 Lean authoring DSL | Partial | Checked targets, Properties, Behaviors, Queries, and finite authored Spaces are built, including canonical axes/choices, request-only faults, seek-only coverage goals, and checked metadata. Fn-5 catalog/list/explain aggregation and usability evidence remain. |
| C4 ExperimentSpec compiler | Partial | Pure planning and `umpire-gen-tests` retain canonical `umpire-experiment/v2` artifacts for one selected trace or an atomic bounded Space batch. Strict persisted admission, the inert RuntimeConfiguration contract, exact sets, immutable publication, and the paired normal/request-only-fault local consumers are built; broader runtime bindings remain. |
| C5 Go/docs generator | Partial | One stable caller-closure regression has deterministic checked-in Go and Markdown Generated Views, and one generation-only Go seam emits an ordinary digest-bound local execution test; broader catalog and promotion surfaces remain. |
| C6 execution runtime | Partial | Paired exact admitted normal and duplicate-delivery caller-closure sets run through the same domain-neutral bounded runner and invocation-local Temporal adapter, returning admitted Run/RawEvidence in memory with complete cleanup and source closure. Additional profiles remain downstream. |
| C7 Evidence/Run Evaluation | One paired bounded local slice | The fixed caller-closure checker consumes both real fn-19 RawEvidence shapes through checked Observation Evaluation, Implementation Link, and unchanged Property authorities, then publishes admitted six-member satisfied or uniqueness-violated Evidence/Result sets. Other profiles and non-local evaluation remain open. |
| C8 exploration | Partial | Deterministic finite exhaustive/shortest-style planning is built. The named coverage-guided policy only seed-rotates enumeration; it does not consume coverage state. Broader campaign exploration exists separately. |
| C9 SDK participant | Partial | The System-owned caller-closure participant and exact duplicate-delivery variant bind the same Go SDK adapter; the latter records one real callback plus one labeled test-owned synthetic contribution. Additional programs and SDKs remain downstream. |
| C10 replay/promote | Separate | Go campaign pipelines minimize, replay, and propose regressions, but cannot read the current artifact or generate a Lean regression from it. |
| C11 formal checks | Partial + separate | Current Lean provides proofs, finite completeness, bounded verification, and model counterexample search. Umpire3 has mature receipts and optional family-scoped Veil support; the current model has neither an explicit first-order checker view nor a Veil binding. |
| C12 Claim Assessment | Separate | Environment, canary, and release-Claim Assessment foundations exist in Go/Umpire3. No local, CI, remote, or canary Claim Assessment path accepts the current artifact. |

| Milestone | Status | Summary |
| --- | --- | --- |
| A: describe real tests | In progress | The model-only C1-C4 slice now includes ordinary Nexus Lifecycle/Operations walkthroughs and an experimental two-by-two authored Space proof; broader Generated View catalog/promotion and usability evidence remain. |
| B: check real Temporal | One paired bounded local slice | Exact normal and request-only-fault Artifacts drive local caller-closure runs, close raw evidence and cleanup, and produce accepted six-member Run Evaluation sets with satisfied and uniqueness-only-violated outcomes. The negative control's second contribution is synthetic; broader profiles, remote execution, replay, and qualification remain open. |
| C: find things | Foundations only | Finite planning and separate campaign tooling exist, but discovery, minimization, replay, and promotion are not composed end to end. |
| D: reuse the investment | Separately advanced | Formal, optional Veil, and Claim Assessment machinery are substantial in Umpire3, but current-model checker-view binding and the active R6/R7 Claim Assessment work remain open. |

### Historical Flow plan snapshots

The table below preserves the plan descriptions recorded by the 2026-08-31 reconciliation. Its
status phrases do not declare current readiness or delivery order; use Flow and `UMPIRE4_ORDER.md`
for those decisions. A reviewed plan does not make its component built.

| Slice | Flow spec | Reconciliation snapshot | Bounded ownership |
| --- | --- | --- | --- |
| C7 offline observation | `fn-4-umpire-observation-and-semantic-verdicts` | Reviewed: Ship; six tasks ready | Closed reusable mapping expressions, bounded synthetic Observation Evaluation, coordinate-complete Evidence Links, unchanged pure Property evaluation, and one Temporal-owned synthetic profile. Live collection and Run Evaluation remain separate. |
| C3/C5 catalog and exact promotion | `fn-5-umpire-discovery-promotion-and-artifact` | Reviewed: Ship; seven tasks ready; dependency synchronized | Checked eight-query plus one authored-space semantic catalog closure, generated glossary/index and list/explain UX, lineage-linked exact in-memory promotion with compiled Temporal source bindings, and a two-regression thin Generated View set. Fn-18 now owns sole-v2 admission and persistence; post-v2 evolution and live replay/minimization remain separate. |
| Milestone A pilot gate | `fn-14-milestone-a-pilot-baseline-and-lean` | Reviewed: Ship; six tasks ready | Frozen eight-defect/twelve-mutation baseline, provider-free cost and coverage measurement, exactly three fresh Agentworkflow authoring trial slots, and a strict `LEAN_FIRST_GO` / `FACADE_FOLLOW_UP` / `NO_GO` / `INCONCLUSIVE` receipt. No facade or live-runtime implementation. |
| C1/C2 standalone input catalogs | `fn-15-standalone-api-and-config-input-catalogs` | Reviewed: Ship; seven tasks ready | Complete mechanical API facts, exact bounded current-model selection, all generated settings plus six owned semantic uses, and one shared exact list/explain/check engine. Drift verification, live lookup, runtime, and Run Evaluation remain separate. |
| C3/C4 authored variation spaces | `fn-16-authored-variation-spaces-and` | Implemented and verified | Checked finite axes/choices, request-only faults, seek-only coverage goals, canonical metadata for fn-5, dependent-kernel point lowering, atomic bounded batch compilation, and byte-identical v2 output through `umpire-gen-tests`. RuntimeConfiguration and Artifact admission are now separate implemented fn-18 contracts; coverage scoring, runtime, and Run Evaluation remain separate. |
| C8 bounded semantic exploration | `fn-17-bounded-semantic-exploration-and` | Reviewed: Ship; seven tasks ready | Pure selection over fn-16's atomic bounded universe with exhaustive, pair/t-wise, seeded-random, proof-reduced symmetry, real coverage-guided state/report/resume, and pinned precedence. Runtime, evidence, persisted readers, replay, minimization, and promotion remain separate. |
| C4/C7/C8 versioned artifact boundary | `fn-18-versioned-umpire-artifact-boundary` | Implemented and verified | The current v2 Artifact is the sole baseline. Strict bounded Go admission preserves Definition IDs, Behavior Fingerprints, Artifact Checksums, Limits, Known Gaps, exact reference closure, and immutable atomic publication. Read-only checks expose one Artifact or complete set without publishing. Only future post-v2 successors may add migrations. Execution, Evidence collection/interpretation, replay, promotion, and Claim Assessment remain separate. |
| C6/C9 bounded local execution | `fn-19-bounded-local-temporal-execution-and` | Implemented and verified | One fn-18-admitted caller-closure input set, domain-neutral five-phase runtime/participant boundary, production-safe `temporaltest` lifecycle, closed ephemeral authority, exact Nexus SDK force-close participant, four-source raw evidence, and an admitted in-memory run set exercised by a deterministic ordinary generated Go test. Its one fixed semantic consumer is integrated by fn-20; replay, promotion, remote/CI/canary, and Claim Assessment remain separate. |
| C7 local Run Evaluation | `fn-20-local-execution-semantic-conformance` | Implemented and verified | One exact four-member local run set passes through a private bounded Go-to-Lean checker bridge and the existing Observation Evaluation, Implementation Link, and Property authorities. Operational, Observation Evaluation, Implementation Link, Property, and tooling outcomes remain independent in the admitted six-member Evidence/Result set. Replay, promotion, formal checking, non-local evaluation, and Claim Assessment remain separate. |
| C4/C6/C7/C9 deterministic Nexus negative control | `fn-21-nexus-duplicate-observation-control` | Implementation complete; final parent gate pending | One exact request-only duplicate-delivery fault lowers to a second closed local program/configuration and is live-paired with the normal control through execution, Run Evaluation, immutable publication, and reopen. Its real requested/completed lifecycle and callback count one plus one labeled test-owned synthetic contribution produce an accepted uniqueness-only violation with status 2. Generic fault infrastructure, a new Artifact family, replay, promotion, and non-local Claim Assessment remain separate. |
| C10 deterministic replay, minimization, and reviewed promotion | `fn-22-deterministic-replay-semantic` | Reviewed: Ship; eight tasks ready | The exact fn-21 six-member violation set is admitted, reproduced twice, compared by a closed normalized violation signature, reduced only through checked action/order/fault edits, explained by a non-destructive evidence core, and bound to one statically elaborated fn-5 proposal candidate. One bounded local command emits an exact report/progress contract and review-only Lean source; no new persisted artifact family, dynamic campaign surface, automatic regression installation, or non-local Claim Assessment is introduced. |
| C11 Veil compatibility and adoption gate | `fn-23-veil-toolchain-compatibility-and` | Reviewed: Ship; six tasks ready | Two exact pinned Veil candidates are evaluated under Lean 4.33.1 only through an opt-in Linux/aarch64 diagnostic with frozen Git/npm/solver/Lean/Zig/Node/runtime closures, no-network execution, honest solver trust, candidate-scoped measurements, and one canonical adopt/defer/inconclusive receipt. No Veil dependency, semantic claim, default build integration, generated source, or production surface is introduced. |
| C11 Lean-native receipts and canonical replay | `fn-24-lean-native-verification-receipts-and` | Reviewed: Ship; six tasks ready | A reusable Temporal-free formal module runs the exact checked Query/dependent kernel itself, emits honest bounded-search receipts, and admits a violation only after the candidate replays through the canonical kernel, Behavior, and pure Property evaluator. One caller-closure verify receipt and one family-test negative control prove both paths; Veil, ExperimentSpec, runtime, promotion, and Claim Assessment remain separate. |
| C11 optional CallerClosure checker binding | `fn-25-optional-callerclosure-veil-binding-and` | Reviewed: Ship; six tasks ready; depends on fn-23 and fn-24 | One completed compatibility receipt freezes an adopt or defer branch. Adoption adds one family-owned finite view with bidirectional correspondence, one exact optional handwritten declaration, a non-breaking external receipt v2, and mandatory canonical replay; defer/inconclusive adds no dependency, source, command, or placeholder claim. The reusable Umpire package remains Temporal/Nexus/checker-name free. |
| C12 local Evaluation Receipt | `fn-26-local-qualification-receipts-and-staged` | Reviewed: Ship; six tasks ready; depends on fn-18, fn-19, and fn-20 | One offline local Evaluation Profile admits the exact six-member v2 Run Evaluation set plus optional admitted verification Evidence, preserves every phase status and Known Gap independently, and publishes one environment-scoped Evaluation Receipt. It acquires no Execution authority and accepts no pilot or pre-v2 compatibility input. |
| C12 hermetic CI execution | `fn-27-hermetic-ci-execution-and-qualification` | Implemented and verified | One ordinary generated Go test consumes the byte-identical local v2 `ExperimentSpec`, reuses the disposable loopback runner and canonical Run Evaluation authority, and proves Artifact Checksum, Behavior Fingerprint, and stable typed semantic parity while allowing fresh transport identities. It adds no CI Evaluation Profile, Evaluation Receipt, provenance schema, new Artifact-set version, or Claim Assessment path. |
| C12 authorized remote staging black-box Claim Assessment | `fn-28-authorized-remote-staging-black-box` | Reviewed: Ship; eleven tasks ready; depends on fn-14, fn-18, fn-19, fn-20, and fn-27 | One fixed protected environment supplies an in-memory mTLS authority for a preallocated staging namespace and Nexus endpoint. A server-enforced lease, idempotent one-mutation participant, public-only Evidence mapping, cleanup-before-Run Evaluation ordering, ephemeral recovery/reconcile path, and versioned remote Evaluation Receipt preserve the same `ExperimentSpec` without target selection or internal Evidence. The claim is staging-scoped and never canary or release eligible. |
| C12 bounded production canary Claim Assessment | `fn-29-bounded-production-canary-execution-and` | Reviewed: Ship; thirteen tasks ready; depends on fn-14, fn-18, fn-19, fn-20, fn-26, fn-27, and fn-28 | One trusted-ref-gated protected workflow runs the same ExperimentSpec against a fixed, dedicated production-canary namespace and Nexus route with no customer traffic, faults, deployment, configuration, or routing mutation. Exact lease reuse/fencing, idempotent mutation, public evidence, cleanup-reserved RPC accounting, versioned recovery, and strict receipt/set successors keep every canary receipt non-release-eligible. Receipt bytes are inspectable but not self-authenticating; release aggregation and its trusted artifact channel remain separate. |
| C12 release evidence graph and manual authorization | `fn-30-release-evidence-graph-and-manual` | Reviewed: Ship; seven tasks ready; depends on fn-14, fn-18, and fn-26 through fn-29 | One signed seven-slot evidence index authenticates complete or explicitly held inputs, externally signed build/deployment attestations bind one immutable server candidate, and one fixed Lean policy produces an inspectable accepted/held/rejected graph. Separate protected owner roles create append-only approve/deny/revoke decisions; neither Claim Assessment nor authorization can deploy, promote, route, configure, or roll back. |

Bottom line: the component decomposition still holds, but the current implementation is not a
general end-to-end Umpire pipeline. It now reaches one paired bounded local mini-demo: exact normal
and request-only-fault `umpire-experiment/v2` sets plus their RuntimeConfigurations execute through
the closed caller-closure adapter, return admitted Run/RawEvidence, and publish/reopen checked
six-member satisfied and uniqueness-only-violated Evidence/Result extensions. Replay/promotion,
additional profiles, formal checker integration, and non-local Claim Assessment are the next seams. Veil belongs at a different seam
inside C11: an optional, family-owned Lean checker target bound by proof to an explicit canonical
view, never a generated backend or a second semantic authority.

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

The current bounded path now continues from deterministic `ExperimentSpec` inspection through one
local caller-closure execution and checked Run Evaluation. Replay, promotion, other profiles, and
non-local Observation Evaluation remain separate implementations. The decomposition is an internal
and artifact-level boundary, not a requirement to ship many unrelated binaries.

## 2. Artifact contracts

These artifacts are the seams between components:

| Artifact | Purpose | Current state |
| --- | --- | --- |
| API catalog | Mechanical Protobuf schema knowledge and field dispositions | Generated Lean structure exists in `model/Temporal/API.lean` and `model/Temporal/API/`; a separate catalog, field-disposition artifact, and drift report do not. |
| Config catalog | Keys, types, defaults, precedence, scope, and declared classification | The complete initialized registry snapshot and resolution fixtures exist in `model/Temporal/DynamicConfig/`; handwritten classifications and typed uses exist for selected Callback and Matching settings. |
| Semantic catalog | Lean-owned resources, actions, properties, observations, targets, and hashes | Checked declarations and canonical projections exist in `model/Umpire/`, including `CheckedSpaceMetadata`; fn-5 still owns consolidated catalog aggregation and list/explain. |
| Regression/space | Named regressions and exploration spaces over Lean semantics | Checked `Property`, `Behavior`, `Query`, and Space values exist, with reusable Switch and Temporal examples; there is no persisted regression catalog. |
| ExperimentSpec | Environment-independent executable specification with a drive plan, properties, requirements, Limits, Known Gaps, provenance, and Behavior Fingerprints | `umpire-experiment/v2` is the sole byte-identical planning Artifact emitted by planning and `umpire-gen-tests`; its nested plan is exactly `umpire-drive-plan/v2`. Strict Go admission and complete-set closure are implemented. Runtime bindings belong to the separate admitted `umpire-runtime-configuration/v2`; runtime consumption remains a Known Gap. |
| ExperimentRun | One realized execution binding an `ExperimentSpec` to an environment, seed, controls, receipts, and cleanup outcome | Exact `umpire-experiment-run/v2` Lean/Go codecs, admission, closure, and fixtures are implemented; one bounded local caller-closure runtime produces it. |
| Raw evidence | Typed implementation facts, receipts, Known Gaps, source positions, and causal references | Exact bounded `umpire-raw-evidence/v2` Lean/Go codecs, admission, closure, and fixtures are implemented; one local adapter collects four closed sources. |
| Semantic evidence | Lean-defined interpretation of raw facts | Exact `umpire-evidence/v2` transport mirrors the current Observation Evaluation types, Evidence-backed Model Trace, and Evidence Links; one fixed live caller-closure Run Evaluation produces it. |
| Result | Accepted, violated, unknown, conflict, or unsupported Run Evaluation outcomes | Exact `umpire-result/v2` transport preserves operational, Observation Evaluation, Implementation Link, Property, Limit, Known Gap, cleanup, and semantic statuses independently; one fixed local evaluator produces it. |
| Replay bundle | ExperimentSpec, ExperimentRun, evidence, result, Limits, and provenance | Campaign/replay artifacts exist in the Go Umpire baselines; they do not accept `umpire-experiment/v2`. |
| Veil binding | Source-bound correspondence between an explicit canonical first-order view and family-authored Veil declarations | Umpire3 exports `umpire3/veil-binding/v1` with declaration mappings, source/view digests, backend revision, semantic relation, and trust data. The current model has no corresponding checker view or binding. |
| Verification receipt | Checker target, Limits, trust mode, proof/counterexample, and provenance | Umpire3 has checker receipts and release evidence. The current `model/` produces `PlannerRun` results, not a verification receipt. |

The retained boundary is exactly embedded `umpire-drive-plan/v2` plus persisted
`umpire-experiment/v2`, `umpire-runtime-configuration/v2`, `umpire-experiment-run/v2`,
`umpire-raw-evidence/v2`, `umpire-evidence/v2`, and `umpire-result/v2`. Every document uses
fixed-order two-space pretty JSON, stable escaping and canonical numbers, no trailing spaces, and
exactly one terminal LF. Artifact Checksum input is `domain + "\n" + preimage`, where the preimage
is those exact pretty bytes with only that document's own checksum omitted and one LF. Behavior
Fingerprints identify checked meaning; provenance and Artifact checksums independently bind
provenance and complete content.

Exact executable, execution, and evaluation closures contain two, four, and six members
respectively. Their manifest and members preserve every Limit and Known Gap and reject incompatible
references or stale checksums. `PublishSet` validates and privately stages a complete set, then
installs one immutable manifest-digest directory with a single rename; readers observe absence or a
fully revalidated set. Read-only checks never call publication. This baseline supersedes fn-37's
compact spelling in place. Compact/alternate whitespace, other families, schema migrations, generic
envelopes, platform orchestration, and mutating artifact-management commands have no normalization,
alias, or fallback here.

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
and method deprecation. Behavior Fingerprint, ordering, completion, and absence remain handwritten
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

**Status: implemented prototype core, with catalog and usability gaps.** Reusable,
Temporal-independent modules under `model/Umpire/` provide checked `Property`, `Behavior`, `Query`,
and finite Space declarations over composed targets and finite kernels. They cover setup,
capabilities, actions, exact traces, occurrence Limits, partial ordering, sequences, adjacency,
properties, query Limits, deterministic search policy, canonical axes and choices, request-only
faults, seek-only coverage goals, and canonical checked Space metadata. The synthetic Switch and
Temporal Nexus examples exercise the authored-to-checked lifecycle without a running server.

```text
.lean specifications -> checked regression/space catalog
```

Current interface:

```text
make umpire-build-model
make umpire-check-regression
```

Fn-5 still owns catalog aggregation and list/explain generation from `CheckedSpaceMetadata`; no
persisted registry exists here. Later C8 work consumes `lowerSpacePoint` and checked coverage goals
for exploration policy and coverage state. Outcomes deliberately remain target-owned rather than
author-owned. Regression, witness, counterexample, verification, and exploratory selection share
the same checked Query path.

Lean-first behavioral authoring is now the implemented choice. A generated Go facade remains
conditional on usability evidence; in either case, Lean owns the resulting semantic object.

### C4. Semantic ExperimentSpec compiler

**Responsibility:** turn a checked regression or selected point in a scenario space into one or more
bounded `ExperimentSpec`s without requiring Temporal.

**Status: implemented for one selected Model Trace and one atomic bounded Space batch.**
`Umpire.Planning.plan` consumes a checked Query and proof-carrying incremental kernel, distinguishes
verified, selected, absent, budget-exhausted, unsatisfiable, and invalid outcomes, and may produce a
canonical `umpire-experiment/v2`. `Umpire.Space.compileBatch` canonically lowers every point through
that same target-indexed kernel and returns the complete batch or one typed error with no partial
list. `Temporal.Tool.Inspect` still exposes only the checked-in Switch and caller-closure scenarios
and emits deterministic JSON with structured diagnostics.

```text
checked spec + model target + Limits -> ExperimentSpec[]
```

Current interface:

```text
make umpire-inspect SCENARIO=workflow-nexus.query.exact-action-caller-closure
make umpire-inspect SCENARIO=switch.query.exact-action
```

The current planning `ExperimentSpec` records bindings and symbolic roles, preconditions, requested actions,
model-owned outcomes and resulting states, a linear extension, checkpoints, properties, observation
requirements, Limits, Known Gaps, provenance, and Behavior Fingerprints. Compilation distinguishes a
requested action from its model-owned outcome. Ordinary planning leaves choices, variants, and
faults empty; checked Space intent populates those same fields without changing the artifact format.
`umpire-gen-tests` emits those same canonical v2 planning Artifacts without adding participant,
setup, ordering, termination, cleanup, or other runtime bindings. Those bindings belong to the
separate `umpire-runtime-configuration/v2` contract. Fn-18 now supplies strict read-only admission,
exact executable/execution/evaluation set closure, and immutable publication for that boundary.
Runtime consumption is not implemented.

### C5. Go test and documentation generator

**Responsibility:** expose stable Lean regressions through familiar, deterministic Generated Views.

**Status: implemented for one stable model view and one operational Generated View.** The production
inspector and canonical fixture for `workflow-nexus.query.exact-action-caller-closure` generate one
ordinary model-view Go test and one readable Markdown Generated View. The checked-in outputs are
`tools/umpire/regression/catalog_generated_test.go` and
`model/Temporal/Tool/Generated/Regressions.md`.

```text
one stable inspected regression -> thin _test.go wrapper + readable documentation
```

Repository interfaces:

```text
make umpire-gen-tests ARGS='<selection> --output <directory>'
make umpire-gen-regression-views
make umpire-check-regression-views
make umpire-check-regression
go run ./tools/umpire/cmd/umpire-gen-tests-go \
  tools/umpire/temporal/nexus/testdata/caller-closure-input-set/manifest.json \
  --output tools/umpire/temporal/nexus
```

The regression-view generation target publishes both outputs transactionally. The focused check regenerates them
into an isolated root, byte-compares the complete pair, and runs the generator and fixture-backed
verifier tests; the stable regression gate includes that check. The generated test calls only
`RequireGeneratedView`, and both outputs carry the Lean-owned identity, provenance, fixture, property
and observation-requirement identities, and semantic fingerprint without copying behavioral
procedure.

The generation-only `umpire-gen-tests-go` seam separately admits the exact System-composed two-member
caller-closure set and deterministically renders
`tools/umpire/temporal/nexus/caller_closure_runner_generated_test.go`. That ordinary test retains
literal set/member digest binding and calls `tools/umpire/runner` directly. It does not reconstruct
behavioral intent, publish output, or evaluate semantic meaning.

The model-view path remains a Generated View-only pilot. The separate operational Generated View now
executes one closed local slice and returns RawEvidence; the fixed downstream checker proves one
Run Evaluation without changing the Generated View. Both paths are closed to one stable regression. The reviewed
`fn-5-umpire-discovery-promotion-and-artifact` plan owns the broader checked catalog, list/explain,
exact in-memory promotion, and two-regression Generated View slice, but none of that plan is implemented
yet. Sole-v2 persisted admission is now implemented by fn-18; future post-v2 migration remains a
separate concern.

### C6. Execution runtime

**Responsibility:** realize one `ExperimentSpec` against an environment and return an
`ExperimentRun` plus raw evidence without deciding whether Temporal was correct.

**Status: one current-model local operational slice is integrated.** `tools/umpire/runtime` owns the
domain-neutral checked five-phase engine, while `tools/umpire/runner` verifies generated digest
binding and composes it with one closed adapter. The first ordinary generated Go test consumes the
exact fn-18-admitted caller-closure ExperimentSpec/RuntimeConfiguration set and returns an admitted
four-member set containing ExperimentRun and RawEvidence. Existing baselines under
`common/testing/umpire`, `tools/umpire2`, and `tools/umpire3/execution` remain separate.

```text
ExperimentSpec + Environment -> ExperimentRun + raw evidence
```

Current public execution surface:

```text
go test ./tools/umpire/temporal/nexus/... -run <generated-test>
```

The runtime owns preparation, bounded action realization, evidence capture, control receipts,
source closure, cleanup, and isolation. The runner returns admitted output in memory and does not
publish. The only current authority is invocation-owned loopback local execution; there is no public
local-run/run-tests command, profile selector, remote adapter, or ambient authority.

### C7. Evidence interpretation and Run Evaluation

**Responsibility:** interpret raw implementation facts through Lean-owned observation programs and
compare them with the `ExperimentSpec` requirements and realized `ExperimentRun`.

**Status: implemented for one bounded local caller-closure scenario.** The fn-19 four-member
execution set now passes through the fixed private Go/Lean sibling pair, checked System Observation,
checked Implementation Link, unchanged Feature Property, and strict Query summary. The sole
publisher preserves the four input members and adds Evidence and Result as one admitted six-member
set. Repeated checking publishes the same manifest-digest destination.

The live proof keeps operational completion, Observation acceptance, Implementation Link
application, Property satisfaction, and cleanup independently inspectable. It runs corruption and
ambiguity controls first, reopens the published set, and independently checks the zero-handle/four-
source closure plus the API/history-backed caller-closure clauses. Generic profiles, remote
Observation Evaluation, replay, promotion, formal checking, and Claim Assessment remain downstream.

```text
ExperimentSpec + ExperimentRun + raw evidence -> semantic evidence -> evaluated Result
```

Current fixed offline CLI:

```text
umpire-local-run-evaluation --set <four-member-directory> --output-root <directory>
make umpire-check-local-run-evaluation SET=<four-member-directory> OUTPUT_ROOT=<directory>
```

This component owns evidence identity, causal matching, source-local ordering, closure, ambiguity,
conflicts, Known Gaps, and diagnostic obligations. Missing or ambiguous evidence fails closed. It
remains separate from execution: the command consumes an already-admitted four-member set, performs
no Temporal or network operation, and publishes only after complete checking.

### C8. Exploration engine

**Responsibility:** select useful `ExperimentSpec`s from a Lean-owned scenario space under a strategy
and budget.

**Status: finite model planning implemented; broader exploration is partial.** The current Lean
planner supports exhaustive, breadth-first, and shortest policies with separate behavior and
candidate-evaluation Limits, deterministic tie-breaking, completeness evidence, and
instrumentation. Its policy vocabulary also names `coverage-guided`, but that branch currently
uses the seed only to rotate canonical setup, action, and outcome enumeration; it has no coverage
state, scoring, or coverage report. The Go campaign baseline separately supports bounded
matrix/lifecycle selection, sampling, semantic coverage, and pinned regressions. These paths do
not yet share the current `ExperimentSpec`, and pairwise, t-wise, seeded-random, symmetry-reduced,
and campaign-wide coverage selection are not all implemented in the current Lean planner.

```text
scenario space + strategy + Limits + coverage state -> selected ExperimentSpecs + coverage report
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

**Status: one current-model SDK participant is integrated.** `Temporal.System.Execution` owns the
exact caller-closure participant program, portable RuntimeConfiguration, local profile, and evidence
source meaning. The Go Nexus adapter realizes that one checked program through the SDK and returns
mechanical receipts and facts; the domain-neutral runner and participant contracts retain no
Temporal/Nexus vocabulary. Umpire2/Umpire3 realizers remain separate.

```text
ParticipantProgram -> SDK behavior + raw observations
```

Current invocation:

```text
generated Go test -> tools/umpire/runner -> closed Nexus Binding
```

The participant protocol remains small and domain-neutral enough for later SDK realizers without
changing the semantic model or core execution runtime. No standalone participant CLI exists.

### C10. Replay, minimization, and promotion

**Responsibility:** reproduce a failure, reduce it to the smallest relevant Model Coordinates,
and turn a stable discovery into a reviewed regression.

**Status: existing Go baselines, not integrated with the current model.**
`common/testing/umpire/campaign` implements bounded selection, monotonic reduction, replay, and
deterministic regression candidates; Umpire3 has its own campaign/replay/promotion pipeline. The
current Artifact boundary supports strict reading, but no current path replays a Run, minimizes a
failure, or proposes and generates a Lean regression from one.

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
evaluated violation.

### C11. Formal checker integrations

**Responsibility:** run an explicitly declared model target under Limits and return a
provenance-rich receipt or counterexample.

**Status: model-local foundation plus a separate mature Umpire3 implementation.** The current
`model/` has Lean proofs, checked target composition, finite completeness evidence, exhaustive
verification, witness/counterexample search, and canonical planner results. It does not yet produce
a verification receipt or replayable checker counterexample. Umpire3 already provides exact and
native checked certificates, Lean temporal checking, optional embedded Veil declarations, receipts,
mutation gates, and release evidence; those artifacts are not bound to the current model's semantic
digests.

```text
model target + Limits + declared view -> verification receipt or counterexample
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

### C12. Deployment portability and Claim Assessment

**Responsibility:** realize the same `ExperimentSpec` under authorized remote profiles and assemble
assessed release claims without erasing environment-specific trust and Known Gaps.

**Status: bounded hermetic CI portability is implemented; current-model Claim Assessment is not.**
The ordinary CI test executes the byte-identical canonical v2 `ExperimentSpec` from the local path
through the same invocation-owned loopback runner and Run Evaluation authority:

```sh
mise exec -- go test -count=1 -tags test_dep ./tools/umpire/temporal/nexus/... -run '^TestHermeticCIPortability$'
```

`make umpire-check-regression` is the aggregate repository gate and invokes the same test. The proof
requires equal Artifact Checksum and Behavior Fingerprints, then compares stable typed semantic
meaning across operational, Observation Evaluation, Implementation Link, Property and clause,
Limit, Known Gap, and cleanup outcomes. Fresh executions may have different runtime-scoped
transport identities; workflow YAML cannot construct semantic declarations or reinterpret Evidence.
Evaluation Profiles, Evaluation Receipts, provenance schemas, new artifact-set versions, Claim
Assessment, remote, canary, and release work are excluded from this bounded hermetic proof.

The Go Umpire trees also contain older environment profiles, guarded canary controls, Claim
Assessment models, and release evidence. `UMPIRE_LEAN.md` R6 still tracks independently retained
local, CI, remote, and public-gRPC Claim Assessment. The current `umpire-experiment/v2` is not
accepted by those paths.

```text
ExperimentSpec + deployment profile + authority -> ExperimentRun + evaluated Result / release graph
```

Proposed CLI:

```text
umpire qualify <release-or-result-set>
```

CI, staging, cloud, black-box, and canary execution reuse the same semantic `ExperimentSpec` but have
different evidence capabilities, credentials, cleanup policies, blast-radius controls, and claim
strength. This component is downstream of successful local execution and Run Evaluation.

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
              C7 Run Evaluation
                     |
                     v
                   Result
                     |
             +-------+--------+
             |                |
             v                v
      C10 replay/promote  C12 Claim Assessment
```

C1, C2, C3, C5, C8, C9, C10, and C11 can be developed largely with fixtures or synthetic models.
C4 and C7 are the principal semantic integration points. C6 is the principal Temporal integration
point. C12 depends on the local path being credible first.

The current implementation cut is:

```text
C1/C2 generated structure
        -> C3 checked target/property/behavior/query/space
        -> C8 finite planning
        -> C4 umpire-experiment/v2 inspection
        -> strict v2 admission/set closure/immutable persistence
        -> deterministic ordinary generated Go test
        -> bounded local C6/C9 execution + admitted Run/RawEvidence
        -> fixed C7 Observation/Implementation Link/Property Run Evaluation
        -> admitted six-member Evidence/Result publication
        -> hermetic CI execution with stable typed semantic parity
        -> remaining remote profiles and C10-C12 Claim Assessment integrations
```

C5's broader catalog and exact-promotion slice now has a reviewed Flow plan, while its implementation
remains open. The first exact admitted `umpire-experiment/v2` and RuntimeConfiguration now execute
without restating Lean-owned semantics and return Run/RawEvidence bound to the same Behavior
Fingerprints. The fixed downstream Lean authority maps that live evidence and produces the
Evidence/Result Run Evaluation for this one scenario. The same Artifact and authority now support
the bounded hermetic CI proof. Broader remote profiles, replay, promotion, and Claim Assessment
remain integration gaps.

## 5. Delivery milestones

### Milestone A: Lean can describe real Temporal tests

Components: C1-C5, limited to the API/config/model material needed by the pilot.

**Status: in progress.** C1-C4 have a usable model-only vertical slice, including ordinary Nexus
Lifecycle/Operations walkthroughs and an experimental two-by-two Space whose four points compile
atomically. C5 has a checked-in one-regression Generated View pilot. Broader Generated View catalog
and promotion work remains.

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
- a shared basic Nexus Lifecycle teaching target and focused Operations walkthroughs with checked
  composition and finite planning;
- a reusable finite Space package plus one experimental Nexus two-by-two declaration whose four
  canonical specs retain target-owned outcomes;
- a Workflow-Nexus caller-closure model with target-owned cancellation, output, and ownership
  observations; and
- deterministic `umpire-experiment/v2` inspection, golden fixtures, structured diagnostics, and a
  focused regression gate.

Remaining exit evidence:

- broader deterministic Go wrapper generation beyond the one operational caller-closure test; and
- ordinary Go engineers can modify the model after a short orientation, or the evidence selects
  a generated Go authoring facade.

The first compiled `ExperimentSpec` now uses the bounded local runner through ordinary Go test
machinery; broader generated-test coverage remains part of C5.

### Milestone B: The model checks real Temporal

Components: C6, C7, C9, and one local environment adapter.

**Status: complete for one bounded local caller-closure scenario.** The exact current
ExperimentSpec/RuntimeConfiguration set drives the invocation-local Temporal/Nexus adapter, returns
an admitted ExperimentRun plus closed four-source RawEvidence, and passes that output through the
fixed checked Observation, Implementation Link, Property, and Result path. This does not complete
other profiles, remote execution, CI/staging/canary qualification, or release evidence.

Deliverable:

```text
the same checked semantic test
        -> local Temporal execution
        -> ExperimentRun
        -> raw causal evidence
        -> evaluated state and output result
```

Exit evidence:

- one Nexus cancellation path executes operationally end to end;
- one SDK participant realizes the planned force-close action;
- one API/history output property and one diagnostic obligation are checked;
- missing, corrupted, duplicated, or causally unrelated evidence fails closed; and
- cleanup and evidence-source closure are explicit in the Result.

All five bullets are now exercised by the bounded live regression for this one scenario. That proof
does not generalize Milestone B to another profile or environment.

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
Claim Assessment machinery, while generic Go Umpire includes environment and canary controls. Current
model digest binding and independently retained deployment Claim Assessment remain open; the active
details live in `UMPIRE_LEAN.md` R6 and R7.

Deliverable:

```text
the same model and ExperimentSpec
        -> stronger formal checks
        -> CI/remote/black-box/canary profiles
        -> environment-specific ExperimentRuns
        -> assessed release evidence
```

Exit evidence:

- every claim exposes checker trust, Limits, Known Gaps, environment, and evidence profile;
- remote results remain bound to the same Behavior Fingerprints as local results;
- counterexamples replay through canonical Lean and, where applicable, real execution; and
- every Veil-owning family has a source-bound canonical-view relation, mutation evidence, explicit
  trust and axiom inventory, and a canonical replay gate;
- operational authority, cleanup, and blast-radius controls are explicit.

Additional current-model formal integration, production-canary binding, and deterministic Go
scheduling remain downstream of a credible local slice.

## 6. Pilot sequence

The Nexus caller-closure Behavior, reusable model DSL, Generated Views, v2 `ExperimentSpec`, exact
RuntimeConfiguration, bounded local runner, SDK participant, operational generated Go test, and one
checked local Run Evaluation are now implemented. Steps 3 and 4 below are complete for that one
scenario; step 5 remains the next integration boundary:

1. Finish the simple Nexus authoring examples and assess whether Lean-first authoring is usable
   enough to defer C5's generated Go facade.
2. Record historical bugs, realistic mutations, current hand-authored coverage, comparison costs,
   time budget, and stop/go thresholds before beginning live integration.
3. **Complete for one local scenario:** carry admitted ExperimentRun/RawEvidence through Observation
   Evaluation, Implementation Link checking, Property evaluation, and Result without another
   semantic IR.
4. **Complete for the same scenario:** run evidence corruption and ambiguity fixtures before the
   green operational control and independently verify cleanup/source closure in the Result.
5. Compose the current finite planner with campaign minimization, replay, and reviewed promotion for
   the same scenario.
6. Measure defect detection, feedback latency, semantic interaction coverage, marginal authoring
   cost, maintenance cost, evidence quality, execution cost, and usability by another engineer.
7. Continue current-model Claim Assessment work only if the evidence meets the predeclared thresholds;
   follow `UMPIRE_LEAN.md` for the separate Umpire3 R6/R7 release path.

## 7. Deferred work

These remain architectural possibilities, not prerequisites for the current-model proof:

- replacing the current full-descriptor API projection with a bounded selection policy shared with
  Umpire3;
- standalone API/config catalog, explain, and drift-report command surfaces;
- production canary binding for `umpire-experiment/v2`;
- generalized remote Claim Assessment for the current model;
- additional current-model formal checker integrations not required by the pilot;
- Veil adoption beyond one compatibility spike and one meaningful family binding;
- exhaustive internal tracepoints;
- deterministic Go scheduler control or full DST; and
- low-level lock and scheduler modeling in feature semantics.

Generated Veil source, a second Lake project, a checker-neutral semantic IR, blanket Veil adoption,
and Veil in production runtime paths are rejected designs, not deferred work.

The detailed rationale and the other two design conversations are synthesized in
[UMPIRE_CHATS.md](UMPIRE_CHATS.md).
