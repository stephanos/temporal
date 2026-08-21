# Umpire3 cleanup proposal

Status: proposed, based on the tree at 2026-08-21.

## Outcome

Make `tests/umpire3` navigable by use case, with one canonical model family per target, a small number
of deep Go modules, explicit ownership of every tracked artifact, and no historical status encoded in
package names.

This is a structural cleanup. It must preserve semantic behavior, existing comments, strict decoding,
failure classifications, security constraints, and the independence of Umpire3 from earlier Umpires.
It should not introduce another semantic representation, checker abstraction, or third-party
dependency.

## Why the tree is confusing

The problem is not merely its size. The current tree has 504 tracked files, including 247 Go files,
184 Lean files, 49 JSON files, 46 Go packages, 29 top-level directories, 14 commands, and 33 Lean
files directly under `model/`. It is organized simultaneously by layer, tool, historical migration,
artifact kind, and operational workflow. None of those axes consistently wins.

The most important examples are:

| Finding | Evidence | Consequence |
| --- | --- | --- |
| One target has competing model identities. | The catalog and checked Experiment define `nexus-cancellation` with `Temporal.Product.Nexus`, `Temporal.System.NexusTasks`, and `Temporal.Refinement.NexusTasks`. The first-order, native, and Veil artifacts for that target name `Umpire3.Temporal.System.NexusCancellationFencing.behavior` and carry a different semantic hash. | A reader cannot tell which model family is authoritative, and a family-scoped check can select sources from one family while consuming retained checker evidence from another. |
| Historical migration is represented as architecture. | `Temporal/System/MigratedFamilies.lean` contains 11 namespaces in 746 lines; `Temporal/Refinement/MigratedFamilies.lean` contains their refinements in 1,261 lines. | Unrelated families change together, and “migrated” says where code came from rather than what it means. |
| Old and new model shapes coexist. | Twelve files under `Temporal/System` import `Temporal.Product`; many embed a product `visible` state or a `productRun`. Independent `Feature`, `System`, and `Refinement` modules also exist for newer work. | The tree preserves two answers to how a family is modeled. Some older system models violate the roadmap rule that system behavior be independently defined. |
| `protocol` is a catch-all dependency. | It has about 8,000 production lines across 30 files and is imported by 33 Umpire3 packages. It owns wire documents, generated vocabulary, embedded defaults, first-order execution, outcome classification, release signing, and trace conversion. | Almost every use case depends on one broad interface, so unrelated changes have a large reasoning and rebuild radius. |
| Command packages contain implementation. | `cmd/umpire3-export` has about 1,960 production lines; `main.go` alone has about 1,100. `cmd/umpire3-api` has about 800. | Generation logic is tested through `package main`, is hard to reuse, and mixes CLI parsing with source discovery, Lean execution, validation, and artifact construction. |
| Shallow packages occupy the top level. | `clockskew`, `developerux`, `familycheck`, `mutationaudit`, and `resilience` each have one or two consumers and mostly exist to feed release assurance or build tooling. | The top level overstates their architectural importance and obscures the main authoring and execution flows. |
| Generated, retained, and hand-authored data use the same visual language. | Data is spread across `generated`, `results`, `bindings`, `testdata`, and a bare `ledger.json`. | A reader cannot tell whether a file is reproducible output, a reviewed evidence snapshot, a fixture, or source input. |
| Documentation and compatibility state have drifted. | `GENERATION_GATE.md` is not referenced and is omitted from `documents.go`; the nested Veil README is outside that audit. The README advertises the unified command while the release manifest still emits `cmd/umpire3-qualify` commands. | The documented surface and the machine-readable surface disagree. |
| Local tool caches dominate the directory. | The two ignored `.lake` directories currently occupy about 3.9 GB, versus roughly 4 MB of tracked Umpire3 files. The isolated Lentil cache alone is about 507 MB. | Disk usage reinforces the impression that experiments and caches are product source. |

## Canonical vocabulary

Choose vocabulary before moving files. Apply these names to packages, types, commands, docs, format
descriptions, and generated identifiers. Preserve existing serialized format strings until their next
explicit version bump.

| Current term | Use instead | Reason |
| --- | --- | --- |
| `Product` and `Feature` for the same role | `Feature` | The roadmap's authoritative family shape uses Feature for user-visible behavior. Do not keep two names for that role. |
| `MigratedFamilies` | The domain family name | Migration status belongs in the migration ledger, not a permanent namespace. |
| `nexus.Regression(...)` / `workflow.Regression(...)` returning `scenario.Scenario` | `nexus.Scenario(...)` / `workflow.Scenario(...)` | A Regression is the executed test; a Scenario is authored intent. |
| generated `<Family>Regression` constructors returning Scenario values | `<Family>Scenario` | Same distinction, enforced in the generated facade. |
| `umpire3test` | `regression` | This module compiles and executes a Scenario as a Regression; the current name describes its repository location rather than its responsibility. |
| runtime limit/result names in Go | execution limit/result | Execution is the canonical operation. Keep `umpire3/runtime-result/v3` readable for compatibility, then rename the wire format only in a versioned revision. |
| `profile` package | `deployment` | The domain term is Deployment profile; `profile.Profile` is both redundant and vague. |
| `campaign` | `mutation` or `mutationcampaign` | The package only runs mutation discovery, minimization, replay, and promotion. |
| `process` | `internal/subprocess` | It supervises OS child processes, not a Umpire domain process. |
| checker `canonical` | `leanreplay` | It invokes the canonical Lean replay checkers; “canonical” alone does not identify the use case. |
| checker `native` | `finite` or `nativefinite` | It produces finite-state certificates with a native Go searcher. The current name hides both facts. |
| `wirecase` | `conformance/wire` | It drives typed protobuf conformance cases, not generic “cases.” |
| generic `results` directories | `retained` | These are reviewed evidence snapshots, not arbitrary run output. |

Also fix local vocabulary mismatches such as `RequireRegression` reporting that it needs an
“environment profile” when its interface actually requires an `execution.Factory`, and aliases named
`umpire3runtime` that refer to the `execution` package.

## Recommended module map

The target shape should make the main use cases visible and group supporting implementations beneath
them. Exact package moves can be adjusted to avoid cycles, but the ownership should be stable.

```text
tests/umpire3/
  README.md
  docs/
    architecture.md
    authoring.md
    generation.md
    operations.md
    security.md
    recovery.md
    support.md

  model/                         Lean semantic authority
  protocol/                      strict versioned transport subtree only
    catalog/                     vocabulary, IDs, values, catalog document
    experiment/                  Experiment document and strict codec
    checker/                     checker views, certificates, receipts, traces
    monitor/                     generated monitor document types
    release/                     release and qualification document types

  scenario/                      author intent and compiler
    nexus/
    workflow/
  regression/                    author-facing test facade

  execution/                     Experiment execution engine
    evidence/
    fault/
    observation/
    participant/
  adapter/
    temporal/                    Temporal Environment adapter
      internalhistory/           explicitly privileged history adapter

  exploration/                   bounded semantic discovery
  replay/                        replay bundle lifecycle and drift
  mutation/                      mutation campaign, minimization, promotion

  checker/
    leanreplay/
    finite/
    veil/

  deployment/
    canary/
  assurance/
    release/                     candidate assembly, qualify, promote, validate
    migration/
    audit/
      clockskew/
      developerexperience/
      mutation/
      resilience/
      documentation/
  conformance/
    wire/

  internal/
    artifactio/                  atomic publication and bounded reads
    subprocess/                  child lifecycle and resource limits
    generate/                    generation registry and implementations
    command/                     unified command implementation

  cmd/
    umpire3/                     supported user command
    umpire3-canary/              operator controller, if still independently deployed
    umpire3-canary-worker/       required killable process seam
    umpire3-participant/         required participant process seam, if externally invoked
```

The intended dependency direction is:

```text
protocol/catalog
      ↓
protocol/experiment ← scenario ← regression
      ↓                  ↓
execution ← adapter/temporal
      ↓
replay ← mutation

protocol/checker ← checker/*

protocol/release ← assurance/release
                         ↑
             deployment + assurance/audit/*
```

`scenario`, `execution`, the Temporal adapter, and the regression facade already have useful depth.
Keep those seams. Do not recreate the deleted `compiler`, `environment`, `runner`, or `runtime`
packages as pass-through layers.

## Lean model cleanup

### 1. Converge each target before rearranging files

For every catalog target, write down exactly one Feature, one independently defined System, one
Refinement, its Observation, and its checker Targets. Start with `nexus-cancellation` because it has
the concrete split described above.

The catalog, generated Experiment, proof manifest, first-order/temporal views, checker coverage,
family dependency graph, and retained evidence must all name and hash the same family. Add a gate
that rejects a checker entry whose canonical model or semantic hash is not reachable from the
catalog target's declared family.

Only after that gate passes should the superseded Nexus/Product/System path be deleted. Moving both
families into prettier directories without choosing one would preserve the ambiguity.

### 2. Replace horizontal layers with family locality

The existing `Product`, `System`, `Refinement`, `Targets`, `Experiments`, `Mutations`, and
`Observation` directories spread one change across the tree. Prefer:

```text
model/Temporal/Families/NexusCancellation/
  Feature.lean
  System.lean
  Refinement.lean
  Observation.lean
  Experiment.lean
  Mutation.lean
  Targets/
    Attempt.lean
    FirstOrder.lean
    Veil.lean

model/Temporal/Families/UpdateLifecycle/
  Feature.lean
  System.lean
  Refinement.lean
  Observation.lean
  Experiment.lean
  Targets/
```

Physical locality must not weaken logical independence: `System.lean` must not import `Feature.lean`
or carry Feature state, runs, or proofs. `Refinement.lean` is the first module allowed to import both.
Genuinely shared mechanics such as Task Delivery belong under `Temporal/Mechanisms`, not under an
arbitrary family.

Split `System/MigratedFamilies.lean` and `Refinement/MigratedFamilies.lean` by domain immediately.
Then compare each split model with its same-named older System/Refinement pair and retain only the
family that satisfies the independence rule and is referenced by the canonical target.

### 3. Move entry points out of the model root

The 33 root Lean files are mostly one-function exporter or executable wrappers. Group them under
`Umpire3/Command/Export`, `Umpire3/Command/Replay`, and `Umpire3/Command/Check`, and update Lake roots
accordingly. Keep only the real library roots (`Umpire3.lean`, `Temporal.lean`, and test aggregators)
at `model/`.

### 4. Mirror family layout in Lean tests

Replace broad files such as `Umpire3Tests/TemporalWorkflowEvidence.lean` with tests next to the
family vocabulary, for example `Umpire3Tests/Families/WorkflowLineage.lean`. A family check should
discover tests from the family manifest rather than scanning every file in one flat directory.

## Go cleanup

### Keep and deepen the primary modules

- `scenario` should continue to own Scenario normalization, enumeration, compilation, and
  explanation behind `Compile` and `Explain`. Rename only the misleading Regression constructors.
- `execution` should continue to own `Run(ctx, Request) Result` and the Environment seam. Its current
  1,375-line `run.go` should be split by responsibility inside the package before extracting any new
  interface: preparation, action realization, checkpoint evaluation, fault lifecycle, cleanup, and
  result finalization.
- `adapter/temporal` should remain an adapter to the Environment seam. Split the 847-line
  `participant_sdk.go` and the large SDK/evidence files by mechanism, without exposing more of their
  implementation.
- `regression` should remain the small author-facing interface that compiles and executes all
  Experiments. Its depth comes from hiding the compiler, execution, failure formatting, and optional
  Replay corpus.
- Checker implementations should stay separate because native finite search, Veil, and canonical
  Lean replay are real adapters with different trust and failure modes. Group them under `checker`;
  do not invent a backend-neutral semantic IR.

### Reduce `protocol` to transport

Preserve `protocol` as the strict transport seam required by the roadmap, but make it a subtree of
cohesive document packages. Move implementation algorithms out:

- first-order state execution and replay move to `checker/finite`;
- outcome classification moves to `execution`;
- semantic-trace construction from backend-specific results moves to checker adapters;
- embedded default artifact lookup moves to a generated-artifact registry;
- release candidate assembly, signing workflow, qualification, and promotion move to
  `assurance/release` while their wire document types remain in `protocol/release`.

The protocol packages should expose document construction, strict decode, validation, digest, and
versioning. They should not execute a model or orchestrate a use case.

### Group shallow modules by their only use case

- Move `clockskew`, `developerux`, `mutationaudit`, `resilience`, and the root documentation audit
  beneath `assurance/audit`. A single assurance builder should call them; callers should not need to
  understand each audit's storage details.
- Merge `release` and `qualification` orchestration into one deep `assurance/release` module with a
  small interface such as `BuildCandidate`, `Qualify`, `Promote`, and `Validate`.
- Move `familycheck` into `internal/generate` or `internal/check`; it exists only for build tooling.
- Move `process` and `internal/artifact` to the more precise `internal/subprocess` and
  `internal/artifactio` names.
- Nest evidence, faults, observations, and participant programs under execution. They remain
  importable by the Temporal adapter and replay, but no longer compete with primary use cases at the
  root.
- Move `wirecase` under conformance. It is test support for a specific transport seam, not a primary
  domain module.

### Extract generator implementation from commands

Create an `internal/generate` module with an interface resembling:

```go
type Request struct {
    Kind    Kind
    Variant string
    Inputs  Inputs
}

type Generator struct {
    Lean LeanRunner
}

func (g *Generator) Generate(context.Context, Request) (Artifact, error)
func (g *Generator) Check(context.Context, Request, []byte) error
```

The module owns the export registry, source dependency closure, Lean invocation, validation, and
deterministic encoding. The CLI parses flags and maps errors to exit codes; it does not contain
artifact construction. Keep the real Lean runner and an in-memory test adapter as the two adapters
at that seam.

Fold manifest, migration, family, API projection, native, and Veil build drivers into one internal
developer command or generated Make recipes. Keep separate commands only when a process isolation
or deployment seam requires a separate executable.

Delete the `umpire3-run` and `umpire3-qualify` pass-through commands after updating the release
manifest and documentation and announcing one compatibility cutoff. Do not replace them with alias
packages; the supported `umpire3` subcommands already provide the behavior.

## Artifacts and stale material

Every tracked non-source file should have exactly one class and owner:

| Class | Meaning | Location convention | Required gate |
| --- | --- | --- | --- |
| generated | Deterministic output reproducible byte-for-byte from checked source. | Owner's `testdata/generated/`, except generated Go/Lean source that must compile in place. | Regenerate to a temporary path and compare. |
| retained | Reviewed evidence that is intentionally not compared on nondeterministic measurements. | Owner's `testdata/retained/`. | Validate provenance, source digest, format, bounds, and release reachability. |
| fixture | Hand-authored bounded test input. | Owner's `testdata/fixtures/`. | Strict decode and use by at least one named test. |
| experiment | Quarantined, non-qualifying research code or output. | `experiments/<name>/`. | Explicit owner and opt-in command; excluded from default discovery and release. |
| cache | Rebuildable local tool state. | Tool-standard ignored directory. | Scoped clean command; never tracked. |

Add one machine-readable artifact manifest containing path, class, owning module, source command,
format version, and retention reason. Fail generation if a tracked JSON/result/binding is absent from
the manifest or if a manifest entry has no file. This replaces inference from directory names.

Specific cleanup candidates:

1. Delete `GENERATION_GATE.md` after merging its useful content into `docs/generation.md`; it is an
   unreferenced peer of audited docs today.
2. Delete the quarantined TLA/TLC/Apalache Go package, command, generated modules, retained results,
   and Make targets. The roadmap explicitly excludes this work and permits deletion when quarantine
   costs maintenance. If an owner still needs it, move the entire subtree to `experiments/tla` in one
   commit and keep it out of normal Go package discovery.
3. Delete the isolated Lentil project unless it gains an owner and an acceptance criterion. Its only
   Make target is not part of `umpire3-check`, and the required temporal proof already lives in the
   primary Lean project. The fallback is `experiments/lentil`, not `model-checkers`.
4. Remove compatibility commands after their declared cutoff and regenerate the release candidate so
   no machine-readable command points at them.
5. Delete superseded Product-carrying System modules only after their independent family replacements
   own the catalog, Experiments, Targets, monitors, and retained evidence.
6. Add `make umpire3-clean` that removes only the two resolved `.lake` directories and other explicit
   Umpire3 caches. It must not accept an arbitrary root, glob, or unresolved environment variable.

Do not delete native benchmark, certificate, Veil, mutation, resilience, or release snapshots merely
because they are under `results` or `testdata`; they are current assurance inputs until the artifact
manifest and a regenerated release prove otherwise.

## Documentation cleanup

Keep the root README as a short orientation and command index. Move detailed documents into `docs/`
by audience:

- architecture: domain language, model shape, trust model, and Go/Lean seam;
- authoring: Scenario and Regression workflows;
- generation: source ownership, artifact classes, regeneration, and review;
- operations: execution, qualification, promotion, and retained evidence;
- security and recovery: separate because they have distinct reviewers and failure procedures;
- support: supported formats, commands, and compatibility dates.

Replace the hard-coded `documentationNames` list with an embedded, sorted walk of `README.md` and
`docs/*.md`, or a checked documentation manifest. The release assurance digest must cover every
published document in that set. Nested checker notes should either join the published set or be
clearly marked implementation-local.

## Migration sequence

### Phase 0: add invariants before moves

- Add a target-family consistency test covering catalog modules, Experiment model modules, proof
  manifests, checker canonical model, semantic hash, and family dependency sources.
- Add Lean import guards: System cannot import Feature/Product; Refinement may import both.
- Replace exact-directory assertions in `layout_test.go` with dependency and ownership invariants.
- Add artifact-manifest completeness and documentation-coverage tests.

Expected failures are useful here: they identify the competing Nexus families, Product-carrying
System modules, and orphaned documents before anything is renamed.

### Phase 1: remove or quarantine dead ends

- Merge and remove `GENERATION_GATE.md`.
- Remove TLA and Lentil experiments, or move them under `experiments/` with explicit owners.
- Add the scoped clean target.

This reduces package, command, artifact, and cache noise without changing the supported model.

### Phase 2: converge model families

- Choose the canonical Nexus cancellation family.
- Rebind its catalog, Experiment, proof, checker views, Veil declarations, monitors, and evidence.
- Split `MigratedFamilies` by domain and converge each duplicate family.
- Regenerate all derived artifacts and create a new candidate release revision because Lean module
  paths and source digests are release inputs.

Do not carry old qualification receipts across this phase; their source and release bindings are no
longer exact.

### Phase 3: reorganize Lean by family

- Move one complete family at a time, including its tests and export roots.
- Update family dependency generation to read explicit family ownership instead of path heuristics.
- Move exporter and runner wrappers under `Umpire3/Command`.

Run the affected family gate after each family and the complete generated gate after all source-path
changes.

### Phase 4: narrow Go modules

- Split protocol documents in dependency order: catalog, Experiment, checker, monitor, release.
- Move algorithms to execution/checker/assurance owners.
- Group execution support, checker adapters, audits, deployment, and internal utilities.
- Rename the author-facing facade and vocabulary in one atomic import update.

Avoid long-lived compatibility packages. All known Go consumers are in this repository, so an atomic
move has lower complexity than maintaining pass-through interfaces.

### Phase 5: simplify tooling and artifacts

- Extract `internal/generate` and make command packages thin.
- Consolidate build-only commands and update Make targets.
- Classify and relocate tracked artifacts, then generate the artifact manifest.
- Remove compatibility commands at the announced cutoff.
- Update README, docs, CI paths, and release commands.

### Phase 6: final verification

Run, in increasing cost order:

```sh
go test -count=1 -tags test_dep ./tests/umpire3/<changed-module>/...
make umpire3-check-family FAMILY=<changed-family>
make umpire3-check-generated
make umpire3-check
make umpire3-integration
make lint-code
```

Also verify that a clean checkout plus `make umpire3-check` produces no untracked artifact outside
ignored caches and no tracked diff.

## Failure handling and trade-offs

- **Semantic drift:** Lean moves change source identities even when theorem bodies do not. Treat the
  family move as a new candidate release, regenerate dependency digests, and reject old receipts.
- **Partial generation:** The generator returns errors rather than calling `os.Exit`, validates all
  outputs in memory, and publishes only through atomic artifact I/O. A crash must leave either the
  previous complete artifact or the new complete artifact, never a partial sibling.
- **Import cycles:** Split `protocol` from foundational documents upward. `protocol/catalog` cannot
  import Experiment, execution, checker implementations, or assurance. Compile after every package
  move rather than moving the whole graph before feedback.
- **Performance:** More cohesive packages may add import paths but should reduce rebuild radius.
  Family-local Lean modules and an explicit family manifest are important at 10 times the current
  family count; global scans of `Umpire3Tests/*.lean` and one giant protocol/generator package will
  scale poorly.
- **Load:** At 10 times the execution volume, bounded execution, evidence, subprocess, and artifact
  limits remain at the execution and deployment seams. The cleanup must not turn them into caller
  conventions or duplicate them across adapters.
- **Complexity:** A single large rename is easy to review mechanically but hard to diagnose. The
  phased order pays the semantic-convergence cost once, then makes package moves mostly mechanical.
- **Security:** Preserve strict unknown-field rejection, credential isolation, least-authority
  deployment profiles, redaction, signature verification, and hard worker termination. Moving code
  under `internal` must not broaden accepted flags, formats, or environment variables.
- **Cleanup safety:** The clean target resolves and checks exact Umpire3 cache paths before removal.
  It never recursively removes a repository root or path supplied by an unresolved variable.

## Completion criteria

The cleanup is complete when:

- each target resolves to one family and one semantic identity across model, Experiment, checker,
  observation, and release artifacts;
- no System model imports or embeds its Feature model;
- `MigratedFamilies` and historical package names are gone;
- the top-level Go tree exposes use cases rather than audits and utilities;
- protocol packages contain transport behavior only;
- commands are thin adapters over testable modules;
- every tracked artifact and published document has one declared owner and lifecycle;
- quarantined experiments are deleted or visibly separated from supported checkers;
- compatibility commands have a documented removal date and no current release depends on removed
  entry points;
- the focused, generated, full, integration, and lint gates pass from a clean checkout.
