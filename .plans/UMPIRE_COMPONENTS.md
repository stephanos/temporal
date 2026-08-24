# Umpire proposed components and milestones

Status: extracted from the
[Inspect Umpire Branch](https://chatgpt.com/share/6a8b71cb-74e4-83e8-947a-c2f6d595fefc)
design conversation. This is a proposed delivery decomposition, not an approved implementation plan.
Where it conflicts with `UMPIRE_LEAN.md`, the active roadmap takes precedence.

## 1. Organizing principle

Umpire should be a collection of independently useful tools connected by explicit, versioned
artifacts. Each component should have:

- one narrow responsibility;
- a stable input and output contract;
- a public API or coherent `umpire` CLI surface;
- deterministic, inspectable artifacts;
- fixtures that allow it to be developed without the full system; and
- no independent copy of Lean-owned behavioral semantics.

The proposed pipeline is:

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

The single public command can remain `umpire`; the decomposition is an internal and artifact-level
boundary, not a requirement to ship many unrelated binaries.

## 2. Artifact contracts

These artifacts are the proposed seams between components:

| Artifact | Purpose | Produced by | Consumed by |
| --- | --- | --- | --- |
| API catalog | Mechanical Protobuf schema knowledge and field dispositions | API importer | Lean models, generators, evidence interpretation |
| Config catalog | Keys, types, defaults, precedence, scope, and declared classification | Config importer | Lean models, experiment compiler, execution profiles |
| Semantic catalog | Lean-owned resources, actions, properties, observations, targets, and hashes | Lean model export | Authoring, compiler, generators, checkers |
| Regression/space | Named regressions and exploration spaces over Lean semantics | Authoring DSL | ExperimentSpec compiler, exploration, Go generator |
| ExperimentSpec | Environment-independent executable specification with resources, actions, ordering, faults, config, properties, expectations, and model hash | Compiler or exploration | Runtime, checker, replay |
| ExperimentRun | One realized execution binding an `ExperimentSpec` to an environment, seed, controls, receipts, and cleanup outcome | Runtime | Evidence interpreter, replay, qualification |
| Raw evidence | Typed implementation facts, receipts, omissions, source positions, and causal references | Runtime and participants | Evidence interpreter |
| Semantic evidence | Lean-defined interpretation of raw facts | Evidence interpreter | Conformance checker |
| Result | Qualified established, violated, unknown, conflict, or unsupported claims | Conformance checker | CI, replay, qualification, reporting |
| Replay bundle | ExperimentSpec, ExperimentRun, evidence, result, bounds, and provenance | Runtime/result pipeline | Replay, minimization, promotion |
| Verification receipt | Checker target, bounds, trust mode, proof/counterexample, and provenance | Formal checker integration | Qualification and reporting |

Every persisted artifact should carry a format version. Semantic artifacts should additionally carry
source and semantic digests, declared omissions, and enough provenance to reject incompatible or
stale consumers.

## 3. Components

### C1. Protobuf API importer

**Responsibility:** mechanically project Temporal API structure from descriptor sets without
inventing product meaning.

```text
descriptor set -> generated Lean wire declarations + API catalog + drift report
```

Proposed CLI:

```text
umpire api sync
umpire api check
umpire api explain <message-or-rpc>
```

The importer owns messages, fields, enums, presence, `oneof`s, maps, nested types, supported
well-known types, RPC shape, and annotations. Semantic identity, ordering, completion, and absence
remain handwritten Lean interpretations. The chat proposes complete schema import; the active
roadmap currently requires a bounded selected wire surface, so scope expansion is a decision gate.

### C2. Dynamic-config importer

**Responsibility:** generate typed configuration knowledge without deciding product semantics.

```text
Temporal dynamic-config declarations -> generated ConfigKey declarations + config catalog
```

Proposed CLI:

```text
umpire config sync
umpire config check
umpire config explain <key>
```

The catalog records key, type, default, precedence/scope, and description. Handwritten classification
separates feature, validation, semantic, timing, topology, performance, and observability impacts and
records whether a value is sampled live, at creation, per request, per task, or after restart.
Cross-language fixtures must show that Lean resolution agrees with Temporal's Go implementation.

### C3. Lean authoring DSL

**Responsibility:** provide approachable declarations for regressions and exploration over
Lean-owned models.

```text
.lean specifications -> checked regression/space catalog
```

Proposed CLI:

```text
umpire spec check [file]
umpire spec list
umpire spec explain <id>
```

The small language should cover setup, actions, expected outcomes, properties, variation axes,
partial orders, interleaving, faults, coverage goals, and budgets. Regression and exploration are
two selections from the same semantic space. The DSL should be testable against small synthetic
models before it depends on Nexus or a running server.

Authoring language is a decision gate: the chat's final proposal is Lean-first behavioral authoring,
with a generated Go facade only if usability testing shows it is needed. In either case, Lean owns
the resulting semantic object.

### C4. Semantic ExperimentSpec compiler

**Responsibility:** turn a checked regression or selected point in a scenario space into one or more
bounded `ExperimentSpec`s without requiring Temporal.

```text
checked spec + model target + bounds -> ExperimentSpec[]
```

Proposed CLI:

```text
umpire compile <spec>
umpire explain <spec-or-experiment>
```

An `ExperimentSpec` records resources, actions, ordering constraints, configuration, faults, expected
observations, properties, bounds, omissions, and model hash. Compilation must distinguish requested
action attempts from successful semantic transitions and bind the executable projection to the
Lean-owned declaration.

### C5. Go test and documentation generator

**Responsibility:** expose stable Lean regressions through familiar, deterministic projections.

```text
regression catalog -> thin _test.go wrappers + readable documentation
```

Proposed CLI:

```text
umpire gen go-tests
umpire gen docs
```

Generated tests should call a stable regression identity, cite the Lean source and semantic hash,
and optionally render a Given/When/Then summary. They must not reproduce procedural semantics.
Stable regressions may be checked in and verified by a clean regeneration diff; ephemeral
exploration candidates are generated at runtime and are checked in only after promotion.

### C6. Execution runtime

**Responsibility:** realize one `ExperimentSpec` against an environment and return an
`ExperimentRun` plus raw evidence without deciding whether Temporal was correct.

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

```text
scenario space + strategy + bounds + coverage state -> selected ExperimentSpecs + coverage report
```

Proposed CLI:

```text
umpire explore <space> --strategy <exhaustive|pairwise|t-wise|random|coverage> --budget <n>
```

The engine supports exhaustive, pairwise, t-wise, seeded random, transition/relation/outcome
coverage, symmetry reduction, and coverage-guided selection. It can initially operate entirely on
the model and later compose with C6 for live execution. Pinned regressions always run independently
of the exploration budget.

### C9. SDK participant runtime

**Responsibility:** realize semantic participant commands through Kitchensink or an SDK and return
structured observations.

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

### C12. Deployment and qualification

**Responsibility:** realize the same `ExperimentSpec` under authorized remote profiles and assemble
qualified release claims without erasing environment-specific trust and omissions.

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

## 5. Delivery milestones

### Milestone A: Lean can describe real Temporal tests

Components: C1-C5, limited to the API/config/model material needed by the pilot.

Deliverable:

```text
concise Lean regression
        -> semantic validation
        -> explained ExperimentSpec
        -> ordinary generated Go test
```

Exit evidence:

- one simple Nexus regression;
- one combinatorial exploration declaration;
- one Workflow/Nexus output property;
- deterministic catalogs, ExperimentSpec output, and Go wrapper generation; and
- ordinary Go engineers can modify the examples after a short orientation, or the evidence selects
  a generated Go authoring facade.

No new runtime is required if the compiled `ExperimentSpec` can initially use existing test
machinery.

### Milestone B: The model checks real Temporal

Components: C6, C7, C9, and one local environment adapter.

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
- operational authority, cleanup, and blast-radius controls are explicit.

Additional formal work, production canaries, and deterministic Go scheduling are deferred until the
local slice demonstrates value or a concrete failure requires them.

## 6. Pilot sequence

Before Milestone A, define a bounded Nexus proof-of-value charter:

1. Select Nexus cancellation/retry plus caller closure or callback arrival as the narrow behavior.
2. Record historical bugs, realistic mutations, current hand-authored coverage, comparison costs,
   time budget, and stop/go thresholds before implementation.
3. Stabilize only the semantic catalog, regression/space, ExperimentSpec, ExperimentRun, evidence,
   and Result schemas needed for the slice.
4. Import only required Protobuf/config material until the schema-scope decision is resolved.
5. Implement Milestones A-C, changing shared infrastructure only when the experiment exposes a
   concrete blocker.
6. Measure defect detection, feedback latency, semantic interaction coverage, marginal authoring
   cost, maintenance cost, evidence quality, execution cost, and usability by another engineer.
7. Continue to Milestone D only if the evidence meets the predeclared thresholds.

## 7. Deferred work

These remain architectural possibilities, not prerequisites for the first proof:

- complete preemptive import of every Temporal Protobuf and dynamic config;
- production canary orchestration;
- generalized remote qualification;
- additional formal checker integrations not required by the pilot;
- exhaustive internal tracepoints;
- deterministic Go scheduler control or full DST; and
- low-level lock and scheduler modeling in feature semantics.

The detailed rationale and the other two design conversations are synthesized in
[UMPIRE_CHATS.md](UMPIRE_CHATS.md).
