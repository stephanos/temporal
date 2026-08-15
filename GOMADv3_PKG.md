# Gomad v3 package structure review

## Scope

This review covers `tools/gomadv3` package seams, dependency direction, exported
interfaces, file organization, and terminology. It excludes the patched Go
overlay's internal standard-library layout except where host packages expose
that implementation.

The goal is not to maximize the number of packages. It is to create deep
modules: narrow interfaces that hide substantial behavior, keep changes local,
and use the domain's language in package, file, and type names.

## Summary

The current layout has a sound ownership skeleton, but several owner packages
became broad aggregation packages rather than deep modules. The most important
problems are:

1. Almost the entire host implementation is a public Go interface even though
   the repository has no Go importers outside `tools/gomadv3`.
2. `runner`, `qualification`, `deterministicio`, and `evidence` each expose
   several independent interfaces through one package.
3. The ubiquitous language says **Campaign** and **Execution**, while code,
   schemas, output, and documentation still heavily use `batch` and `run`.
4. One Execution has several similarly named representations whose conversion
   logic is spread across Runner.
5. Test adapters and process-launch details are fields in production request
   types.
6. Several generic names (`evidence`, `campaignstore`, `packdev`, `generate`,
   `host`) conceal the domain concept actually owned by the code.

The recommended direction is:

- keep only deliberately target-facing packages public;
- retain the useful owner-level dependency direction, but move host owners
  below module-wide `internal/`;
- extract only genuine deep modules from broad owner packages;
- make a single-Execution module return one validated Observation;
- give Execution Record, Artifact, Campaign, Qualification Suite, and
  Compatibility Pack their own named modules; and
- quarantine old `batch`/`run` names in compatibility codecs instead of using
  them throughout current code.

## What is already working well

- `architecture_test.go` makes package ownership and allowed dependency
  direction executable.
- Raw choice and deterministic-I/O framing is hidden in `internal/wire`, and an
  architecture test explicitly prevents domain packages from re-exporting it.
- `runner/internal/corpus` and `runner/internal/frontier` contain substantial
  policy and state behind domain-named modules.
- `runner/internal/execution` owns process containment and bounded output rather
  than leaving those mechanics in the CLI.
- `world.Model` is a genuinely deep module: ordering, capacity, replay,
  snapshots, and mutation invariants sit behind a small operational interface.
- `evidence.Store` contains real durability and safe-publication behavior; it is
  not merely a filesystem wrapper.
- `toolchain.Build` hides a meaningful build pipeline, cache key, publication,
  and locking protocol.
- Schemas and templates are generally stored close to the runtime domain that
  consumes their generated output.

These should be preserved. The goal is to narrow and rename their seams, not to
split every large file into a package.

## Current interface pressure

The following counts are for non-test Go files in each root package. Exported
counts include constants, types, and functions; methods are listed separately.
They are not quality scores, but they show where a caller must learn several
interfaces at once.

| Package | Production lines | Exported top-level declarations | Exported methods | Distinct responsibilities |
| --- | ---: | ---: | ---: | --- |
| `runner` | 4,811 | 51 | 9 | explore, resume, replay, inspect, coordinator, qualification evidence |
| `qualification` | 3,360 | 78 | 4 | analysis, repeated qualification, suite execution, support comparison |
| `deterministicio` | 2,587 | 62 | 25 | contract, adapters, target requirements, transcript, mount broker, captured inputs, probes |
| `toolchain` | 2,495 | 31 | 3 | build, source, patch, installation, upgrade dossier |
| `evidence` | 2,386 | 62 | 7 | canonical codec, identity, Execution Record, Artifact store and reader |
| `world` | 2,218 | 49 | 24 | event model, snapshots, replay, recording, transport codecs |
| `target` | 1,981 | 36 | 6 | preparation, capability review, provenance, toolchain identity |
| `choice` | 1,499 | 45 | 9 | trace, replay plan, feature projection, process session |

Specific interface leaks include:

- `runner.CampaignSpec` has 37 fields. It combines user policy, installation
  state, process commands, resume state, progress output, and three test
  adapters (`Preparer`, `Executor`, and `Replayer`).
- `runner.Executor` accepts `runner/internal/execution.Spec`. External callers
  cannot name that type, so this is not a usable public seam; it is a test seam
  exposed through production Runner configuration.
- `qualification.WorkloadSpec` carries three injected functions, and
  `qualification.SuiteSpec` carries an execution function.
- `toolchain.BuildSpec` exposes test-only failure injection and asks its caller
  to supply canonical build environment policy.
- `target.Spec` exposes build overlays, modfiles, and adapter replacements that
  should only be produced by a reviewed deterministic-I/O preparation path.
- `deterministicio.Spec` is an opaque singleton, but exposes contract identity,
  target preparation, adapter verification, requirements, bootstrap framing,
  and validation through one method set.

## Ubiquitous language

### Resolve the current contradictions

`GLOSSARY.md` says **Campaign** replaces “batch” and **Execution** replaces
“run”. Current code and documentation still contain hundreds of `batch` names
and thousands of `run` names. Some are verbs or compatibility schema names, but
many identify current domain objects:

- `batch.json`, `gomadv3.batch/v2`, `batch_path`, and `BatchPath` coexist with
  `Campaign`, `CampaignPath`, and `CampaignID`.
- `runs.jsonl`, `runCompletion`, `runSeed`, and “batch run” coexist with
  `ExecutionRecord`, `ExecutionJournal`, and `ExecutionEvidence`.
- `CampaignRecord.CampaignID` is serialized as `run_id`.
- `evidence.ExecutionRecord.CampaignID` is serialized as `batch_id`.
- CLI inspection reports `Kind: "batch"` while its Go value is
  `CampaignInspection`.
- `ARCHITECTURE.md` describes a `BatchJournal` even though the implementation
  type is `CampaignJournal`.

Choose the following current language and reserve the old words for explicit
compatibility adapters:

| Canonical term | Use for | Legacy term allowed only in |
| --- | --- | --- |
| **Campaign** | immutable selection plan and its Executions | old schema names, JSON fields, filenames read by compatibility codecs |
| **Execution** | one target process for a Seed or Choice Replay Plan | `run` as a verb, OS/process method names, old schema fields |
| **Execution Summary** | one compact entry in a Campaign journal | current `campaignstore.ExecutionRecord` |
| **Execution Observation** | validated result of supervising one process | current `execution.Result` plus World/I/O/Choice validation |
| **Execution Record** | canonical replay and identity record | current `evidence.ExecutionRecord` |
| **Artifact** | immutable directory containing a Record and payloads | never “evidence store” when Artifact is meant |
| **Guidance** | policy that selects Seeds from a Corpus | define this term in the glossary |
| **Corpus** | bounded retained set used by Guidance | not the static qualification fixture module |
| **Qualification Suite** | manifest and workload set measuring support | current `qualification/corpus` should not be called Corpus |
| **Capability Review** | reviewed Target closure and blockers | add to the glossary if it remains a supported interface |
| **World Session** | target-side connection between a Target and its World Model | current `world/target.Session` |
| **Captured Read-Only Input** | captured mount mappings, snapshot, broker, and Artifact payload | current `readonly_*` cluster |
| **Compatibility Pack** | exact reviewed exception and its authoring workflow | replace `packdev` and generic `compatibility` names |

Compatibility names should be visibly quarantined in files such as
`legacy_batch_v2.go` or private wire structs. Current Go names, error messages,
variables, documentation, and CLI text should use Campaign and Execution.

### Fix documentation/code drift

- `ARCHITECTURE.md` and `README.md` say a target takes the session-owned World
  from `Session.World`; the implementation exposes `Session.Model()`.
- Architecture names **Guide** as an owner, while the glossary defines Corpus
  and the code mixes `guide`, `guidance`, and `corpus`. Define Guidance and say
  that it reads and updates a Corpus.
- `qualification/corpus` is a static fixture module named
  `gomadv3.core.corpus`, but Corpus already has a precise runtime meaning.
  Rename this to a Core Qualification Suite or qualification fixtures.
- `evidence` is not a glossary term, while Execution Record and Artifact are.
- `world/target` and the host-side `target` package force frequent aliases and
  use the same name for different roles.

## Recommended public surface

The current repository has no Go imports of host packages from outside
`tools/gomadv3`. Make the host implementation private by default.

Keep public only:

```text
world                 # World Model and adapter-facing domain values
world/mailbox         # explicit target-facing adapter
world/session         # target-side World Session; rename world/target
```

If external trusted build tooling is a committed supported use case, add one
small public package for that use case, such as `provenance` or
`target/provenance`. It should expose a reviewed operation such as
`ReviewAndWrite`, not the complete mutable `target.Spec`, capability closure,
adapter-replacement, and preparation implementation.

Move `choice`, `deterministicio`, `evidence`, `qualification`, `runner`,
host-side `target`, and `toolchain` below module-wide `internal/`. This prevents
accidental external dependencies while the internal interfaces are deepened.
It also makes the supported public contract obvious from the directory tree.

## Recommended module topology

The following is a target shape, not a request to create every directory in one
change:

```text
tools/gomadv3/
  cmd/
    gomad/
    gomadtool/                    # move from toolchain/cmd/gomadtool

  world/                          # deliberately public
    model.go
    replay.go
    snapshot.go
    mailbox/
    session/                      # target-side session; current world/target

  internal/
    runner/                       # narrow host-orchestrator facade
      campaign/                   # plan, control state, journal, resume
      execution/                  # one supervised and validated Execution
      corpus/                     # retained Corpus and admission
      frontier/                   # Choice Frontier algorithm/state

    record/                       # Execution Record schema, identity, validation
    artifact/                     # immutable publication/open/payload access

    target/                       # reviewed Target preparation
    capabilityanalysis/           # gomad analyze workflow and report

    deterministicio/
      contract/                   # current contract, adapters, requirements
      transcript/                 # host transcript session and projection
      capturedinput/              # Captured Read-Only Input broker/replay/artifact
      boundary/                   # boundary inventory, probes, generation support

    qualification/                # repeat and compare one workload
    qualificationsuite/           # manifest, orchestration, report, legacy readers
    supportcomparison/            # compare two Qualification Suite reports

    toolchain/                     # build pipeline; source/patch/key stay private here
    installation/                  # resolve an installed Gomad Toolchain
    upgrade/                       # upgrade dossier workflow

    compatibilitypack/            # policy, schema, checked data
      authoring/                   # discover, review, approve, generate, check

    worldtransport/               # private host/target configuration framing
    canonicaljson/                # private shared codec implementation
    hostexec/
    hostfs/
```

This tree deliberately keeps the deep algorithms intact. It does not split
World queueing from replay, Toolchain source acquisition from Build, or Choice
Trace from Choice Replay Plan merely because those implementations occupy
different files.

## Package-specific recommendations

### Runner and Campaign

Keep `runner` as the host-orchestrator facade, but remove infrastructure and
test seams from per-Campaign requests.

1. Construct Runner once with installation-owned details: Runner identity,
   supervisor/coordinator commands, target preparer, process executor, and
   Artifact replayer. Production construction can remain private; tests can use
   internal adapters.
2. Give Runner separate `Explore`, `Resume`, and `Replay` operations. Remove
   `ResumeBatch` from `CampaignSpec`; `Explore` should not be a tagged union with
   resume behavior.
3. Replace the flat 37-field request with domain groups:
   `Selection`, `ExecutionPolicy`, `RetentionPolicy`, `Guidance`, `Target`, and
   `ArtifactLocation`.
4. Remove `GuideSnapshotSHA256`, `CollectRunEvidence`, process commands, and
   injected adapters from the Campaign interface. They are implementation or
   workflow state.
5. Move the pure seed Campaign state in `runner/campaign.go` together with
   Campaign planning/journaling under `runner/campaign`. Expose operations such
   as create/open/resume/record/publish, not raw file transitions.
6. Move `runner/internal/campaignstore/artifact.go` to the Artifact module.
   Publishing one Artifact is not Campaign-journal behavior.
7. Let the Campaign module own crash-safe state transitions. Runner should not
   have to call `BeginPreparation`, `CompletePreparation`, `StartExecutions`,
   several `ExecutionJournal.Transition` values, `AppendExecution`, and
   `Publish` in the correct order.

`runner/runner.go` is 1,611 lines and contains public declarations, config
validation, target preparation, scheduling, completion processing, Artifact
publication, retention, guidance, and final publication. After the deeper seams
above exist, split the remaining file by domain behavior (`explore.go`,
`campaign_config.go`, `execution_completion.go`, `environment.go`). Do not
create forwarding packages solely to reduce file length.

### One deep Execution module

`runner/internal/execution.Run` currently returns a raw process `Result`.
Runner then decodes World recording, validates Choice Trace, summarizes semantic
coverage, classifies the outcome, constructs captured input payloads, and builds
multiple record projections.

Deepen this seam to one operation conceptually shaped as:

```go
Execute(context.Context, Plan) (Observation, error)
```

`Observation` should contain already validated and classified World,
deterministic-I/O, Choice, stream, containment, and termination facts. The
implementation should hide:

- inherited descriptor layout and `*os.File` session handles;
- supervisor/bootstrap request framing;
- transcript terminal frames;
- World recording decoding and composition;
- Choice session collection and divergence projection; and
- process-level error translation.

Keep a process adapter seam inside this module for tests. Remove the exported
`runner.Executor` whose method mentions an internal type. Tests of Runner should
provide completed Observations; tests of Execution should exercise production
process containment through its own interface.

This concentrates one-Execution invariants and removes a large amount of
branching from the Campaign loop.

### Execution Record and Artifact

Split `evidence` into the two modules already named separately by
`ARCHITECTURE.md`:

- **Execution Record** owns schema types, canonical encoding, validation,
  record identity, and failure identity.
- **Artifact** owns safe publication, immutable opening, bounded payload access,
  and no-replace filesystem behavior.

Avoid returning mutable wire structs as the main interface. Prefer opaque
validated values:

```go
record.Build(Context, execution.Observation) (record.Record, error)
record.Decode([]byte) (record.Record, error)
artifact.Publish(Store, record.Record, Payloads) (artifact.Artifact, error)
artifact.Open(path string) (artifact.Artifact, error)
```

Provide explicit read projections needed by inspection, replay, and
qualification rather than letting every caller mutate `ExecutionRecord` fields
or call generic canonical JSON helpers.

Move `CanonicalJSON`, `StrictDecode`, JSON-lines helpers, and raw hash helpers
to private implementation modules where possible. Domain modules should expose
`Encode`/`Decode` for their own validated values. A generic public codec makes
it easy for callers to construct schema-shaped but semantically invalid data.

### Execution representations

Name the existing representations by role and centralize their projections:

| Current type | Recommended role/name | Owner |
| --- | --- | --- |
| `execution.Result` | `ExecutionObservation` after validation | Execution |
| `runner.ExecutionEvidence` | `QualificationEvidence` or eliminate in favor of a Record projection | Qualification |
| `evidence.ExecutionRecord` | `ExecutionRecord` | Record |
| `campaignstore.ExecutionRecord` | `ExecutionSummary` | Campaign |

`runner/evidence.go` currently owns a qualification schema, while
`runner/runner.go:manifestForRun` separately builds the replay Record and
Campaign code builds another summary. Record and Qualification modules should
own these projections from the same Observation. Runner should orchestrate
them, not define their schemas.

### Qualification

`qualification` is currently an umbrella for four user-visible workflows. Split
it along those workflows because each has its own input, report schema,
validation, caller, and change axis:

- `capabilityanalysis`: `Analyze` and its report formatting/codec;
- `qualification`: repeat/replay one workload and publish its report;
- `qualificationsuite`: load/run/open a versioned suite;
- `supportcomparison`: compare two validated suite reports.

These are deep modules, not wrappers: each already contains hundreds of lines
of validation and policy behind one primary operation.

Move `PreparedCapabilityReview` out of Qualification. Preparing a reviewed
Target closure and deterministic-I/O adapters belongs to Target review or
Capability Analysis. Its current location makes Compatibility Pack authoring
depend on Qualification merely to obtain a Target review.

Move function fields such as `Explore`, `ReplayArtifact`, `Write`, and
`SuiteSpec.Execute` into unexported module dependencies. A workload request
should describe a workload, not how tests replace every implementation step.

Rename and colocate the core fixture set:

```text
qualification/core/
  manifest.json
  go.mod
  workloads/
    concurrency/
    filesystem/
    network/
    persistence/
```

Use **Core Qualification Suite** or **Qualification Fixtures**, not Corpus.

### Deterministic I/O Contract

The domain ownership is valid, but one package currently presents at least
three different interfaces to different callers:

1. contract identity, target adapters, requirements, and bootstrap;
2. transcript process session and semantic probes; and
3. Captured Read-Only Input capture, replay, wire serving, and Artifact
   persistence.

Keep a small `deterministicio` facade if useful, but place these implementations
in named internal modules. In particular:

- replace generic `readonly_*` filenames with a `capturedinput` module whose
  files are `mapping.go`, `capture.go`, `replay.go`, `broker.go`, and
  `artifact.go`;
- move `SessionFiles` and raw mount request/response functions below the
  Execution seam;
- let the contract expose a high-level Target review/preparation operation
  rather than a singleton with many unrelated methods; and
- keep semantic probe identity with the boundary inventory that defines it.

The existing test that forbids raw wire names is useful, but it only guards a
list of symbols. The stronger design is for those packages to be internal and
for process framing to be unreachable through the contract interface.

### Target and Capability Review

Separate caller input from reviewed build state.

The caller-facing Target request should contain kind, source, arguments, build
tags, working directory, and provenance. Preparation root, toolchain location,
build overlay, modfile, and adapter replacements are infrastructure or reviewed
state and should not be freely mutable fields in the same type.

Make `Prepared` opaque or at least immutable outside Target. It should expose
identity, argv, verification, and explicit Record projections without allowing
callers to change adapter or compatibility evidence after preparation.

Keep capability closure discovery and Target preparation together if
preparation must always enforce the review. Expose a second read-only Review
operation for `gomad analyze`. Do not split source walking, build-info reading,
and compatibility policy into shallow packages unless they vary independently.

Rename `target/packdev` to `compatibilitypack/authoring`; `packdev` is an
abbreviation and does not surface the domain. Rename generic
`target/internal/compatibility` to `compatibilitypack` so reports, requests,
approved packs, schema, and policy are found under one concept.

### World

Preserve the event model, replay validation, queueing, and snapshot invariants
in one `world` module. Splitting these would force internal state and ordering
rules across package seams.

Narrow the public surface around that model:

- move recording headers/codecs and host/target config framing to private
  `worldtransport` and World Record implementations;
- remove `world/host`, which is a 20-line pass-through that copies
  `transport.Config` into `host.SessionSpec` and calls `transport.Encode`;
- rename `world/target` to `world/session` and `child.go` to `session.go`; and
- let the session create or restore the World it owns instead of accepting a
  `*world.Model` that replay may discard.

A simpler target-facing interface is:

```go
session, err := worldsession.Open(world.Config{Limits: limits})
world := session.World()
// modeled work
err = session.Finish()
```

This matches the architecture text, removes the `Model()`/`World()` mismatch,
and makes session ownership explicit.

Snapshot value types may remain public because adapters genuinely need them.
Raw recording and process transport types do not need to be public.

### Choice

Keep Choice Trace, Choice Replay Plan, canonical Decisions, and feature
projection together; they share validation and identity invariants.

Move the host process `Session`, backing files, and terminal framing below the
Execution implementation. These are an adapter at the process seam, not part of
the Choice domain interface. Internalize the package unless target authors are
expected to manipulate Choice Traces directly.

### Toolchain

Keep source acquisition, patch materialization, build-key derivation, locking,
and publication behind `Toolchain.Build`; they form one deep build pipeline.

Move installation resolution and the upgrade dossier to separate modules:

- Installation is used by the user CLI and changes with bundle layout.
- Upgrade is a high-level workflow that imports Qualification. Keeping it in
  `toolchain` makes the foundational Toolchain owner depend upward on Runner and
  Qualification through `toolchain -> qualification -> runner`.

An `upgrade` workflow should import Toolchain and Qualification, not the other
way around.

Deepen `Build` by resolving canonical PATH, Bash identity, timeouts, and failure
injection internally. Its normal request should not expose `Testing`,
`FailurePhase`, `BuildBashVersion`, or canonical path policy. Test failure
injection belongs to an internal adapter or test-only constructor.

Move `toolchain/cmd/gomadtool` to `cmd/gomadtool`. Commands are easier to find
when command packages share one root.

`toolchain/internal/generate` is organized around the verb “generate” rather
than the generated domain. Move boundary manifest discovery/qualification next
to deterministic-I/O boundary ownership, and move protocol generation next to
the Choice and deterministic-I/O schemas. Share a private generator core only
where the schema formats and invariants are genuinely the same.

### CLI file organization

Keep `cmd/gomad/internal/cli` as an adapter. Make `cli.go` contain dispatch and
shared parsing only. It currently hides several commands in a 699-line generic
file.

Move command behavior into files named after commands:

```text
dispatch.go
explore.go
resume.go
replay.go
inspect.go
doctor.go
analyze.go
qualify.go
qualify_set.go
compare_support.go
target_flags.go
output.go
```

In particular, move `runDoctor` beside the doctor model/checks, and give Explore,
Replay, and Inspect their own files. This is a file-level organization change,
not a new package seam.

## Dependency direction

The current owner test is useful but too coarse for the proposed modules. It
permits any package within an owner and preserves the upward
`toolchain -> qualification` dependency.

Target dependency direction:

```text
command adapters
  -> Runner / Qualification / Capability Analysis / Upgrade workflows

Runner
  -> Campaign
  -> Execution
  -> Target
  -> Record
  -> Artifact
  -> Corpus
  -> Choice Frontier

Qualification -> Runner + Record
Qualification Suite -> Qualification + Capability Analysis
Support Comparison -> Qualification Suite reports
Upgrade -> Toolchain + Qualification Suite

Execution -> Choice + Deterministic I/O + World Record + hostexec
Artifact -> Record + hostfs
Target -> Toolchain identity + Compatibility Pack policy

public World adapters -> World
World Session -> World + private worldtransport
```

Enforce exact allowed edges for these modules. Also add gates that:

- enumerate deliberately public packages;
- reject imports of host implementation from an external consumer fixture;
- reject production request structs containing test-only adapters;
- keep wire/schema compatibility code in explicitly named legacy files; and
- prevent CLI packages from becoming dependencies of domain modules.

## Suggested migration sequence

### Phase 1: language and supported surface

1. Decide whether trusted external provenance tooling is a supported Go
   consumer.
2. Expand `GLOSSARY.md` with Guidance, Capability Review, World Session, and
   Compatibility Pack.
3. Add a naming map for legacy `batch`/`run` schema fields and stop introducing
   new current-code uses.
4. Move host packages below module-wide `internal/` without changing behavior.
5. Add package documentation for every enduring module. At present only the
   World family has package comments.

### Phase 2: deep Execution and durable evidence

1. Define the validated Execution Observation seam.
2. Move process adapters and choice/I/O sessions behind it.
3. Split Execution Record from Artifact and make both validated values opaque.
4. Centralize Observation-to-Record, Observation-to-Qualification, and
   Observation-to-Execution-Summary projections.
5. Replace Runner's public injected interfaces with internal test adapters.

### Phase 3: workflows and ownership

1. Move Campaign state, plan, journal, and resume under one Campaign module;
   move Artifact publication out of `campaignstore`.
2. Split Qualification by its four workflows.
3. Move Capability Review preparation out of Qualification.
4. Extract Upgrade from Toolchain and remove the upward dependency.
5. Split Deterministic I/O by contract, transcript, and Captured Read-Only
   Input interfaces.

### Phase 4: names and file locality

1. Rename current Go types, files, errors, and CLI output from batch/run to
   Campaign/Execution while retaining legacy wire readers.
2. Rename Compatibility Pack and World Session packages.
3. Rename the Core Qualification Suite fixture tree.
4. Split generic large command/source files along domain concepts.
5. Update architecture tests to encode the final exact graph.

Each phase should be behavior-preserving and leave old Artifact and report
schemas readable. Avoid a single tree-wide move combined with schema changes.

## Verification strategy

- Keep golden tests for all existing Artifact, Campaign, qualification, and
  replay schemas.
- Add interface-level tests for Execution, Record, Artifact, Campaign, and
  Qualification. Tests should assert observable results across the same seam
  callers use.
- Once the deeper interface tests exist, remove tests that only preserve old
  shallow forwarding modules.
- Run focused package tests with `-tags test_dep`, then the Gomad v3 package
  architecture test and `make lint-code` from the repository workflow.
- Run replay and interrupted-Campaign recovery tests after every move involving
  Record, Artifact, or Campaign journal ownership.

## Trade-offs and failure modes

- **Complexity:** too many packages can recreate the former shallow-module
  problem. Extract only the modules above whose behavior, callers, and change
  axes already differ. Prefer file organization inside a package for the rest.
- **Performance:** package moves should be allocation-neutral. Opaque Record and
  Artifact interfaces must retain streaming/bounded payload access rather than
  copying transcripts, snapshots, or outputs into multiple projections.
- **Scalability:** the deeper Execution and Campaign seams should keep all
  existing byte, count, concurrency, and deadline bounds explicit. At 10x more
  Executions, they must not retain every Observation in memory; Campaign should
  continue to journal and reduce results incrementally.
- **Crash safety:** Campaign must remain the sole owner of lifecycle and resume
  state, and Artifact must remain the sole owner of immutable publication. Do
  not move sync/rename ordering into Runner or CLI while reorganizing packages.
- **Security:** internal packages and opaque reviewed values reduce ways to
  bypass provenance, adapter, record, and payload validation. Preserve the
  current fail-closed behavior at every new seam.
- **Compatibility:** old `batch` and `run` wire names are durable data, not a
  reason to keep those names in the current domain model. Decode through
  explicit legacy adapters and project immediately into canonical Campaign and
  Execution values.

## Completion criteria

The reorganization is successful when:

- the public directory tree lists only deliberate target-facing interfaces;
- a new reader can locate Campaign, Execution Record, Artifact, Corpus,
  Qualification Suite, and Compatibility Pack by package/file name;
- Runner configuration contains domain policy, not process commands, resume
  internals, or test adapters;
- one Execution crosses one small seam and yields one validated Observation;
- Record, Artifact, Campaign, and Qualification own their own projections and
  schemas;
- current Go names and documentation consistently use Campaign and Execution;
- legacy schema compatibility remains explicit and tested; and
- the architecture graph has no upward `toolchain -> qualification` dependency.
