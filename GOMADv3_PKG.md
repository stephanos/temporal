# Gomad v3 package architecture

## Decision

Organize Gomad v3 by architectural ownership, not by command, technical layer, or
visibility. The directory tree should explain the system before a reader opens a
file.

The target architecture has eight domain modules:

- `runner` owns exploration campaigns, replay, resume, and target-process
  supervision.
- `qualification` owns support analysis, qualification suites, reports, and
  comparisons.
- `target` owns target preparation, provenance, and capability review.
- `evidence` owns canonical execution records and immutable replay artifacts.
- `choice` owns runtime-choice observation and exact choice replay.
- `deterministicio` owns the deterministic-I/O contract and captured inputs.
- `world` owns the explicit external-event model.
- `toolchain` owns the patched Go toolchain and its build, validation, and
  upgrade machinery.

The production CLI is a thin adapter over those modules. Root `internal/` is
restricted to two genuinely shared host primitives; it is not the default home
for new packages.

This proposal changes package ownership and Go vocabulary. It does not change
commands, wire formats, JSON schemas, runtime semantics, or support claims.

## Why the current structure is hard to read

`tools/gomadv3/internal/` currently contains 44 peer packages. Their location
communicates only that they are private to the Go module; it does not communicate
which architectural module owns them.

Several concrete problems follow:

- Packages used by only one owner appear globally shared. Examples include
  `doctor`, `inspect`, `installation`, `guide`, `supportcompare`, and
  `worldrecord`.
- Small implementation details look like architectural modules. Examples
  include `commandline`, `outcome`, `outputcapture`, `testtier`, and
  `worldpipe`.
- `artifact` mixes immutable execution artifacts, campaign plans, mutable batch
  journals, resume locks, retained-evidence lookup, and the choice-frontier
  journal.
- `choicewire` mixes generated binary framing, trace decoding, semantic
  projection, choice-replay plans, and divergence reporting.
- `ioprofile`, `iowire`, and `romount` split one deterministic-I/O contract
  across unrelated-looking peers.
- Toolchain construction is spread across `buildkey`, `boundary`,
  `boundarygen`, `hosttool`, `patchset`, `protocolgen`, `scriptpolicy`,
  `sourcearchive`, `testdriver`, `testtier`, `toolchainbuild`, `upgrade`,
  `upgradegen`, `version`, and `versiongen`.
- The names `record`, `artifact`, `process`, `outcome`, `guide`, `profile`, and
  `wire` require callers to understand implementation history before they can
  understand the domain.

The package graph therefore exposes more interface than the architecture
requires. Related changes spread across peer packages, and a reader cannot infer
valid dependency direction from paths.

## Design rules

### Organize by owner

A module used only by `runner` belongs below `runner/`. A module used only by the
toolchain belongs below `toolchain/`. Go's nested `internal/` rule should enforce
that ownership where useful.

Top-level packages are reserved for domain modules used across architectural
owners or directly by target applications.

### Prefer deep modules

Each domain module presents one small interface while hiding validation,
ordering, persistence, framing, and platform details. Callers should exchange
validated domain values rather than reconstruct implementation steps.

Generated frames, file layouts, descriptor assignments, lock files, and
temporary paths are implementation. They do not belong in a domain module's
interface.

### Name domain concepts, not implementation mechanisms

Do not introduce top-level packages named `common`, `core`, `util`, `manager`,
`service`, `models`, or `wire`. These names describe neither an owner nor a
domain. Owner-local `internal/wire` packages are permitted for generated binary
framing that callers cannot import directly.

Use singular Go package names. Prefer a longer complete term such as
`deterministicio` over an unexplained abbreviation.

### Keep dependencies one-way

Low-level domain modules do not import orchestration modules. In particular:

- `evidence` does not import `runner`, `qualification`, `choice`,
  `deterministicio`, or `world`.
- `choice` does not import `runner` or `evidence`.
- `world` does not import `runner` or `evidence`.
- `target` does not import `runner` or `qualification`.
- Storage accepts detached, already-validated payloads instead of importing
  every payload owner.

Adapters that translate choice, I/O, or World results into execution evidence
belong to `runner`, which is the module composing those domains.

### Do not preserve old packages with forwarding wrappers

Compatibility forwarding packages would retain the confusing graph and create
shallow interfaces. Move each ownership slice completely and update its callers.
Existing artifact compatibility is handled by stable schemas and matching old
toolchain bundles, not by keeping obsolete Go import paths alive.

## Domain vocabulary

Go names should use the following vocabulary. Stable schema strings and existing
CLI flags remain unchanged unless a separate compatibility change explicitly
updates them.

| Current term | Proposed term | Meaning |
| --- | --- | --- |
| batch | campaign | One immutable exploration plan and all executions selected by it |
| run | execution | One prepared target launched once with one seed or replay plan |
| `runner.Config` | `runner.CampaignSpec` | Complete request for a new exploration campaign |
| `runner.Summary` | `runner.CampaignResult` | Terminal campaign state and aggregate counts |
| `runner.Progress` | `runner.CampaignEvent` | Typed progress emitted while a campaign runs |
| `record.Manifest` | `evidence.ExecutionRecord` | Canonical evidence for one completed execution |
| `artifact.Artifact` | `evidence.Artifact` | Validated handle to stored immutable evidence |
| `artifact.Store` | `evidence.Store` | Publisher and opener of immutable artifacts |
| `process.Request` | `execution.Spec` | Private runner specification for one child execution |
| `process.Result` | `execution.Result` | Detached result from one child execution |
| `choicewire.Trace` | `choice.Trace` | Validated observations of runtime choices |
| `choicewire.Tape` | `choice.ReplayPlan` | Identity-bound decisions supplied for exact replay |
| `ioprofile.ProfileSpec` | `deterministicio.Spec` | Requested deterministic-I/O configuration |
| `ioprofile.Identity` | `deterministicio.Contract` | Resolved immutable I/O identity and requirements |
| read-only mount | captured read-only input | Host data imported once and replayed without reopening the host path |
| `world.World` | `world.Model` | Pure in-memory external-event state machine |
| qualification set | qualification suite | Versioned workloads, expectations, and limits qualified together |
| guide | corpus | Durable semantic corpus used to select and retain useful executions |

Keep the existing terms `runner`, `target`, `qualification`, `toolchain`, and
`world` at the module level. They already match the architecture and are clearer
than replacements such as “engine,” “controller,” or “platform manager.”

## Target structure

```text
tools/gomadv3/
├── cmd/
│   └── gomad/
│       ├── main.go
│       └── internal/
│           └── cli/
│               ├── cli.go
│               ├── explore.go
│               ├── replay.go
│               ├── qualification.go
│               ├── doctor.go
│               └── inspect.go
│
├── runner/
│   ├── campaign.go
│   ├── replay.go
│   ├── resume.go
│   ├── inspect.go
│   ├── events.go
│   └── internal/
│       ├── campaignstore/
│       ├── execution/
│       ├── frontier/
│       └── corpus/
│
├── qualification/
│   ├── analysis.go
│   ├── qualification.go
│   ├── suite.go
│   ├── report.go
│   ├── comparison.go
│   └── corpus/
│
├── target/
│   ├── prepare.go
│   ├── provenance.go
│   ├── capabilities.go
│   └── internal/
│       └── compatibility/
│           └── packs/
│
├── evidence/
│   ├── record.go
│   ├── validation.go
│   ├── identity.go
│   ├── artifact.go
│   ├── storage.go
│   └── canonical.go
│
├── choice/
│   ├── trace.go
│   ├── replay.go
│   ├── controller.go
│   ├── schema/
│   └── internal/
│       └── wire/
│
├── deterministicio/
│   ├── contract.go
│   ├── session.go
│   ├── transcript.go
│   ├── readonly_inputs.go
│   ├── adapters.go
│   ├── boundary/
│   ├── schema/
│   └── internal/
│       └── wire/
│
├── world/
│   ├── model.go
│   ├── types.go
│   ├── recording.go
│   ├── replay.go
│   ├── snapshot.go
│   ├── target/
│   ├── host/
│   ├── mailbox/
│   └── internal/
│       └── transport/
│
├── toolchain/
│   ├── build.go
│   ├── buildkey.go
│   ├── patch.go
│   ├── source.go
│   ├── upgrade.go
│   ├── installation.go
│   ├── version/
│   ├── runtime/
│   │   ├── go1.26.4.patch
│   │   └── overlay/
│   ├── cmd/
│   │   └── gomadtool/
│   ├── internal/
│   │   ├── generate/
│   │   ├── conformance/
│   │   │   └── testdata/
│   │   └── validation/
│
├── internal/
│   ├── hostexec/
│   └── hostfs/
│
├── docs/
├── Makefile
├── README.md
└── ARCHITECTURE.md
```

The filenames are illustrative. A package may use more files where that improves
locality; additional directories require an ownership reason.

## Module interfaces and ownership

### CLI

`cmd/gomad/main.go` should contain only process-level setup:

```go
func main() {
    os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
```

`cmd/gomad/internal/cli` owns flag parsing, status-code projection, human output,
JSON output selection, installation flags, and command help. It absorbs the
current CLI handlers plus `commandline`, `doctor`, and the presentation part of
`inspect`.

The CLI may call domain interfaces. It must not import an owner's nested
`internal/` package or decode domain storage itself.

### Runner

`runner` is the host orchestrator. Its principal interface is:

- `Explore(ctx, CampaignSpec) (CampaignResult, error)`
- `Resume(ctx, ResumeSpec) (CampaignResult, error)`
- `Replay(ctx, ReplaySpec) (ReplayResult, error)`
- `Inspect(path, InspectOptions) (Inspection, error)`

It also owns one narrowly documented entry point for the executable's private
coordinator, supervisor, and bootstrap modes. Those modes remain implementation,
not user commands.

The runner's private modules are:

- `campaignstore`: campaign plan, append-only execution journal, resume lock,
  recovery, publication, and inspection. This is the immutable-plan and
  crash-consistent-storage seam. Existing schema names such as
  `gomadv3.batch-plan/v1` remain stable even when Go types use “campaign.”
- `execution`: target launch, descriptor plan, bootstrap, supervision, process
  group termination, output capture, choice/I/O/World sessions, and detached
  outcome classification. It absorbs `process`, `outcome`, and the runner-facing
  parts of `worldrecord`.
- `frontier`: bounded alternative-prefix search, canonical frontier state,
  checkpointing, and resume. It absorbs `choicefrontier` and the frontier
  journal currently stored in `artifact`.
- `corpus`: semantic features, seed selection, admission, locking, and durable
  corpus updates. It absorbs `guide`.

`replay` becomes a runner operation because it uses the same execution,
containment, target, and evidence machinery. Keeping a separate top-level replay
package would expose runner implementation.

### Qualification

`qualification` owns every operation whose result is a support or qualification
claim:

- `Analyze` produces capability and blocker evidence.
- `RunWorkload` qualifies one workload repeatedly.
- `RunSuite` evaluates a versioned suite.
- `Compare` compares validated suite reports.
- `OpenReport` validates a stored report.

It absorbs `capabilityanalysis`, `qualify`, `qualificationset`, and
`supportcompare`. These are files and domain concepts inside one deep package,
not four peer packages with overlapping report types.

Use explicit names such as `AnalysisReport`, `QualificationReport`,
`SuiteReport`, and `Comparison` instead of several unrelated `Report` and
`Result` types.

The qualification corpus remains data under `qualification/corpus/`. It is not
part of the runner's adaptive semantic corpus.

### Target

`target` owns the complete transition from a target specification to a reviewed,
immutable executable:

- `Prepare` builds or validates the executable.
- `ReviewCapabilities` computes the capability closure and compatibility
  decision.
- `WriteProvenance` publishes trusted provenance.
- `ReadProvenance` validates it.

Compatibility-pack matching moves to `target/internal/compatibility`.
Qualification consumes the public target review result and does not import
compatibility policy directly. This keeps policy enforcement at the preparation
seam rather than allowing reporting code to reinterpret it.

### Evidence

`evidence` merges the current `record` package with only the immutable-artifact
part of `artifact`.

Its interface owns:

- canonical encoding and strict decoding;
- execution-record validation and identity;
- immutable artifact publication;
- validated artifact opening;
- bounded payload access; and
- legacy record decoding that is still intentionally supported.

Use explicit entry points such as `DecodeExecutionRecord`, `OpenArtifact`, and
`PublishArtifact`. Avoid ambiguous package-level names such as `Open` or
`Write` when more than one evidence form exists.

`evidence.Store` accepts an `ExecutionRecord` plus detached payloads. It does not
know choice frames, I/O transcripts, World transitions, campaign scheduling, or
corpus policy. The runner asks the owning domain modules to validate those
payloads, constructs the record, then publishes it.

Campaign journals, resume state, and frontier checkpoints are not artifacts and
move to the runner. This gives “artifact” one meaning: immutable evidence for
one replayable execution.

### Choice

`choice` is the single deep module for runtime-controlled decisions. It owns:

- trace framing and validation;
- semantic decision projection;
- implementation identity;
- bounded recording state;
- `ReplayPlan` construction and validation;
- prefix derivation; and
- stable replay divergence.

Callers use `Trace`, `Summary`, `ReplayPlan`, and `Divergence`. Raw headers,
records, terminal frames, offsets, and publish ordering stay in
`choice/internal/wire`.

Source schemas and templates move from the shared `protocol/` directory to
`choice/schema/`. The toolchain generator emits both host code and the matching
runtime-overlay implementation from that source.

### Deterministic I/O

`deterministicio` merges `ioprofile`, `iowire`, and `romount` behind one
contract. Its principal concepts are:

- `Spec`: requested configuration;
- `Contract`: immutable identity, requirements, adapters, and limits;
- `Session`: resources for one live or replay execution;
- `Transcript`: validated observed I/O; and
- `CapturedInputs`: read-only host inputs stored for replay.

The module owns bootstrap framing, adapter selection, transcript validation,
semantic coverage, read-only input capture, replay lookup, and capacity
accounting. The runner asks it to prepare a session and consumes a detached
result; it does not construct pipes or parse I/O records.

Binary protocol details live in `deterministicio/internal/wire`. Protocol source
moves to `deterministicio/schema/`. The reviewed interception manifest and its
platform reports move to `deterministicio/boundary/` because they define this
module's supported behavior, even though toolchain generation maintains them.

“Captured read-only input” replaces “read-only mount” in Go vocabulary because
the feature imports deterministic data; it does not provide a live OS mount.
Existing CLI flags and record fields remain compatible.

### World

`world` remains a dependency-light, pure in-memory event model. Rename
`world.World` to `world.Model` to remove stutter while preserving the established
module name.

The transport becomes two explicit adapters:

- `world/target` attaches a model inside the target process and replaces
  `world/child`.
- `world/host` prepares and collects the runner side.

Both adapters share `world/internal/transport`. The core `world` package does not
import evidence types, host filesystem code, or runner code.

Conversion from a World recording to execution evidence belongs to
`runner/internal/execution`. `world/mailbox` remains an explicit modeled adapter.

### Toolchain

`toolchain` owns toolchain construction as one deep module. Merge
`toolchainbuild`, `buildkey`, `patchset`, and `sourcearchive` into files in the
root package so callers learn one `Build` interface rather than a pipeline of
implementation packages.

- `toolchain/version` is a small leaf package because the deterministic-I/O
  contract, target review, builder, and generators all consume the pinned
  version manifest without importing the complete toolchain implementation.
- `toolchain/internal/generate` owns version, choice, I/O, and boundary
  generation.
- `toolchain/internal/conformance` owns the current test driver and tier
  registry.
- `toolchain/internal/validation` owns script policy, patch validation, and
  generated-output validation.
- `toolchain/cmd/gomadtool` replaces `hosttool` and the four small generator
  executables with one private tool and explicit subcommands.
- `toolchain/runtime` owns the Go patch and overlay because only the toolchain
  builder consumes those source assets.
- `toolchain/internal/conformance/testdata` owns the current root `testdata/`
  fixtures.

Installation resolution is toolchain domain logic and moves into
`toolchain/installation.go`. The CLI retains only its flags and presentation.

### Shared host primitives

Root `internal/` contains only:

- `hostexec`: bounded host-command execution and stream capture shared by
  qualification, runner execution, and toolchain construction.
- `hostfs`: no-follow open, atomic replacement, no-replace publication, link
  checks, and advisory locking shared by evidence, target, runner storage, and
  toolchain construction.

These packages contain no Gomad domain types. A new root-internal package
requires at least three architectural owners and a cohesive interface. Otherwise
the code belongs to its sole owner.

## Dependency direction

Arrows mean “may import.”

```text
cmd/gomad/internal/cli
    ├──> runner
    ├──> qualification
    ├──> target
    ├──> evidence
    ├──> deterministicio
    └──> toolchain

qualification
    ├──> runner
    ├──> target
    ├──> evidence
    ├──> choice
    └──> deterministicio

runner
    ├──> target
    ├──> evidence
    ├──> choice
    ├──> deterministicio
    └──> world

deterministicio
    ├──> target
    └──> toolchain/version

target
    ├──> evidence
    └──> toolchain/version

toolchain
    ├──> toolchain/version
    └──> qualification       # upgrade dossier only

evidence ──> internal/hostfs
choice   ──> standard library
world    ──> standard library
```

`toolchain/version` is deliberately separate from `toolchain` so consumers of
the pinned manifest do not create a cycle through the builder or upgrade
dossier.

`hostexec` and `hostfs` are leaves. They never import domain modules.

## Current-to-target package mapping

| Current package | Target owner |
| --- | --- |
| `cmd/gomad` | Thin `cmd/gomad` plus `cmd/gomad/internal/cli` |
| `internal/artifact` | Split: immutable storage to `evidence`; campaign state to `runner/internal/campaignstore`; frontier journal to `runner/internal/frontier` |
| `internal/boundary` | Generation logic to `toolchain/internal/generate`; manifest data to `deterministicio/boundary` |
| `internal/boundarygen` | `toolchain/cmd/gomadtool` and `toolchain/internal/generate` |
| `internal/buildkey` | `toolchain/buildkey.go` |
| `internal/capabilityanalysis` | `qualification/analysis.go` |
| `internal/choicefrontier` | `runner/internal/frontier` |
| `internal/choicewire` | `choice` and `choice/internal/wire` |
| `internal/commandline` | `cmd/gomad/internal/cli` |
| `internal/commandrun` | `internal/hostexec` |
| `internal/compatibility` | `target/internal/compatibility` |
| `internal/doctor` | `cmd/gomad/internal/cli` |
| `internal/filelock` | `internal/hostfs` |
| `internal/guide` | `runner/internal/corpus` |
| `internal/hosttool` | `toolchain/cmd/gomadtool` |
| `internal/inspect` | Validation to `runner` and `evidence`; formatting to `cmd/gomad/internal/cli` |
| `internal/installation` | `toolchain/installation.go` |
| `internal/ioprofile` | `deterministicio` |
| `internal/iowire` | `deterministicio/internal/wire` |
| `internal/outcome` | `runner/internal/execution` |
| `internal/outputcapture` | `internal/hostexec` |
| `internal/patchset` | `toolchain/patch.go` |
| `internal/process` | `runner/internal/execution` |
| `internal/protocolgen` | `toolchain/internal/generate` |
| `internal/qualificationgen` | Delete; the directory is empty |
| `internal/qualificationset` | `qualification/suite.go` |
| `internal/qualify` | `qualification/qualification.go` and `qualification/report.go` |
| `internal/record` | `evidence` |
| `internal/replay` | `runner/replay.go` |
| `internal/romount` | `deterministicio/readonly_inputs.go` |
| `internal/runner` | `runner`, `runner/internal/campaignstore`, and `runner/internal/execution` |
| `internal/safefile` | `internal/hostfs` |
| `internal/scriptpolicy` | `toolchain/internal/validation` |
| `internal/sourcearchive` | `toolchain/source.go` |
| `internal/supportcompare` | `qualification/comparison.go` |
| `internal/target` | `target` |
| `internal/testdriver` | `toolchain/internal/conformance` |
| `internal/testtier` | `toolchain/internal/conformance` |
| `internal/toolchainbuild` | `toolchain/build.go` |
| `internal/upgrade` | `toolchain/upgrade.go` |
| `internal/upgradegen` | `toolchain/cmd/gomadtool` |
| `internal/version` | `toolchain/version` |
| `internal/versiongen` | `toolchain/cmd/gomadtool` and `toolchain/internal/generate` |
| `internal/worldpipe` | `world/host`, `world/target`, and `world/internal/transport` |
| `internal/worldrecord` | `runner/internal/execution` |
| `world` | `world` with `world.Model` |
| `world/child` | `world/target` |
| `world/mailbox` | `world/mailbox` |

Non-package assets move with their owner:

| Current path | Target path |
| --- | --- |
| `protocol/choicewire*` | `choice/schema/` |
| `protocol/iowire*` | `deterministicio/schema/` |
| `boundary/*` | `deterministicio/boundary/` |
| `go1.26.4.patch` | `toolchain/runtime/go1.26.4.patch` |
| `overlay/` | `toolchain/runtime/overlay/` |
| `version.json` and generated Go descriptor | `toolchain/version/` |
| `testdata/` | `toolchain/internal/conformance/testdata/` |
| `qualification/core.json` | Remains a suite manifest beside the `qualification` package |
| `qualification/corpus/` | Remains the self-contained qualification fixture module |

## Migration strategy

Perform the refactor in ownership slices. Do not combine it with behavior
changes.

### Phase 0: establish the baseline

1. Record the current package dependency graph.
2. Run the existing targeted and full Gomad v3 test gates.
3. Add an architecture test that enumerates the explicit Go roots rather than
   `./...`, because the overlay intentionally contains GOROOT-internal imports.
4. Preserve all existing comments when moving code.

### Phase 1: move shared leaves

1. Merge `filelock` and `safefile` into `internal/hostfs`.
2. Merge `commandrun` and `outputcapture` into `internal/hostexec`.
3. Keep their behavior and tests unchanged.

These leaves reduce repeated import rewrites in later phases.

### Phase 2: deepen evidence

1. Move `record` into `evidence`.
2. Move immutable artifact open/publication code into `evidence`.
3. Leave campaign and frontier files temporarily in place.
4. Change `evidence.Store` to accept detached payloads so it does not import
   domain owners.
5. Verify canonical bytes and hashes against fixtures before renaming Go types.

### Phase 3: deepen choice and deterministic I/O

1. Move semantic choice code into `choice` and generated framing into
   `choice/internal/wire`.
2. Move I/O contract, transcript, adapter, and captured-input code into
   `deterministicio`.
3. Move source schemas and boundary data beside their owners.
4. Replace raw frame operations at call sites with validated sessions and
   detached results.
5. Regenerate and prove host/runtime generated outputs agree.

### Phase 4: isolate target and World adapters

1. Move target preparation and provenance into `target`.
2. Hide compatibility selection below `target/internal/compatibility`.
3. Rename `world.World` to `world.Model`.
4. Replace `world/child` and `worldpipe` with `world/target`,
   `world/host`, and private transport.
5. Keep World free of evidence dependencies.

### Phase 5: consolidate runner ownership

1. Move campaign journals and recovery into `runner/internal/campaignstore`.
2. Move process supervision and outcome conversion into
   `runner/internal/execution`.
3. Move guided corpus logic into `runner/internal/corpus`.
4. Move frontier state and persistence into `runner/internal/frontier`.
5. Merge replay into `runner`.
6. Rename batch/run Go entities to campaign/execution after the structural move
   is green.

### Phase 6: consolidate qualification and CLI

1. Merge analysis, qualification, suite, and comparison into
   `qualification`.
2. Move CLI-only code below `cmd/gomad/internal/cli`.
3. Move installation resolution to `toolchain`.
4. Make `cmd/gomad/main.go` a thin process adapter.

### Phase 7: consolidate toolchain ownership

1. Merge build, build-key, patch, and source acquisition into `toolchain`.
2. Consolidate generation, conformance, and validation internals.
3. Replace generator mains and `hosttool` with `gomadtool` subcommands.
4. Move patch, overlay, version, boundary, schema, and conformance assets.
5. Update Makefile inputs only after every new command path exists.

### Phase 8: remove the old graph

1. Delete emptied old directories; do not leave forwarding packages.
2. Add an import-architecture test for the dependency rules in this document.
3. Update README and ARCHITECTURE vocabulary.
4. Run generation, focused tests, the complete Gomad v3 test gate, repository
   formatting, and lint.

For reviewability, keep structural moves and vocabulary renames in separate
commits when this plan is implemented. Each commit must build and test on its
own.

## Compatibility and identity

Moving Go packages changes source paths, binary build information, and derived
implementation identities. A no-behavior-change refactor therefore still
creates a deliberate runner/toolchain identity epoch.

The migration must follow these rules:

- Do not change JSON field names, schema constants, protocol values, canonical
  ordering, or digest domains merely to match new Go names.
- Existing artifacts remain inspectable through retained legacy decoders.
- Exact replay still requires the matching historical runner and toolchain
  bundle. A new binary must fail preflight rather than claim compatibility with
  an old implementation identity.
- Generated boundary and protocol bytes should remain identical unless their
  identity intentionally includes relocated source paths. Any difference must
  be explained and reviewed.
- Compatibility-pack identities and qualification baselines must be
  regenerated only after the package move is complete and the new identity is
  stable.

Keeping old import paths through wrappers would not preserve binary or source
identity and would undermine the architecture. Do not do it.

## Error handling and failure modes

### Import cycles

The most likely structural failure is a cycle through evidence or toolchain
version data. Prevent it by keeping evidence domain-neutral and keeping
`toolchain/version` independent of the toolchain builder.

### Private subprocess modes

Coordinator, supervisor, and bootstrap dispatch currently enters runner/process
implementation directly from the CLI. The runner must expose one narrow dispatch
entry point before `process` becomes private. End-to-end tests must cover every
private mode and exit classification.

### Generated-code drift

Moving schemas, overlay sources, and generators can make host and runtime
protocol implementations diverge. Generation checks must compare both outputs
from one source schema and fail closed on stale files.

### Storage recovery

Moving campaign journals must not change fsync order, no-replace publication,
lock behavior, recovery validation, or partial-directory cleanup. Injected
failures at every mutation should recover the same prior committed state.

### Artifact compatibility

Type renames must not leak into canonical JSON. Golden byte tests should compare
records and artifacts before and after the move. Legacy data that lacks a
currently required identity remains rejected exactly as it is today.

### Unsupported hosts

Package movement must not widen platform claims. Darwin/arm64 remains the only
fully qualified runner platform until separate evidence changes that claim.

### Crashes and ten-times load

The refactor changes locality, not runtime behavior. Process crashes remain
contained by the runner, artifact publication remains atomic, and campaign
recovery remains journal-based.

At 10× executions, output, frontier state, or corpus size, the package structure
must not require materializing new global collections. `campaignstore`,
`evidence`, `frontier`, and `corpus` retain their explicit byte/count limits and
streaming or append-only behavior. Future sharding and segmented journals deepen
those owner-local modules instead of adding cross-cutting storage packages.

## Verification plan

### Architecture checks

Add a standard-library-only architecture test that:

1. runs `go list` over the explicit host package roots;
2. rejects imports that violate the dependency direction above;
3. rejects any root `internal/` package other than `hostexec` and `hostfs`;
4. rejects imports of another owner's nested `internal/` code;
5. rejects reintroduction of old package paths; and
6. permits the special GOROOT overlay only through toolchain conformance tests.

### Behavioral tests

Preserve and move existing tests with their implementation. Add or retain cases
for:

- canonical execution-record bytes and identities;
- immutable artifact publication, reopen, and corruption rejection;
- campaign interruption, recovery, repeated resume, and final publication;
- choice trace validation, replay-plan derivation, prefix replay, divergence,
  and capacity;
- deterministic-I/O live capture, exact replay, malformed frames, overflow,
  and captured read-only inputs;
- World record, restore, target/host transport, and replay;
- target capability review and exact compatibility-pack selection;
- qualification analysis, suite expectations, support comparison, and report
  compatibility;
- toolchain generation, patch validation, build cache identity, and upgrade
  dossier; and
- every CLI command and private subprocess mode.

Tests at a deep module's interface replace tests of obsolete forwarding
packages. Internal state is tested only where a private state machine has its
own meaningful invariants.

### Commands

During implementation, use the repository's existing gates and always include
`-tags test_dep` for Go tests. From `tools/gomadv3`, test explicit host roots so
the GOROOT overlay is not treated as an ordinary module package:

```sh
GOWORK=off go test -tags test_dep \
  ./cmd/... ./runner/... ./qualification/... ./target/... \
  ./evidence/... ./choice/... ./deterministicio/... \
  ./world/... ./toolchain ./toolchain/version \
  ./toolchain/cmd/... ./toolchain/internal/... ./internal/...
```

From the repository root, run:

```sh
make -C tools/gomadv3 validate
make -C tools/gomadv3 runner-test
make -C tools/gomadv3 world-test
make -C tools/gomadv3 test
make fmt-imports
make lint-code
```

Run the smaller affected-module test first after each phase, then the complete
gate before declaring the phase complete.

## Trade-offs

### Performance

Package boundaries add no intended runtime cost. Deep choice and I/O sessions
may remove repeated validation and assembly from callers. The refactor must not
add serialization, copies, locks, or goroutines merely to enforce source
ownership.

### Scalability

Owner-local campaign, frontier, corpus, and evidence modules provide clear
places for future segmentation, sharding, and streaming. They also prevent a
generic storage layer from coupling unrelated limits and recovery rules.

### Complexity

The target has more top-level domain packages than a monolith but substantially
fewer peer implementation packages. Nested internal packages are used only
where they enforce real ownership. Merging qualification and evidence increases
their file counts, but callers learn fewer interfaces and changes remain local.

### Security

Owner-local internals reduce accidental access to process launch, filesystem
publication, compatibility policy, and raw wire parsing. This improves
reviewability but does not turn Gomad into an OS sandbox. Existing fail-closed
capability and host-I/O rules remain authoritative.

## Acceptance criteria

The package refactor is complete when:

- the top-level tree exposes the eight domain modules in this document;
- root `internal/` contains only `hostexec` and `hostfs`;
- every former `internal/*` peer has a documented owner or has been removed;
- `artifact` means one immutable execution artifact, never a campaign journal;
- raw wire types are private to `choice`, `deterministicio`, or `world`;
- the CLI does not decode domain storage or import owner internals;
- evidence, choice, and World remain independent of the runner;
- no compatibility forwarding packages remain;
- canonical schemas and protocol behavior are unchanged;
- expected implementation-identity rotation is explicit and tested;
- package, generation, conformance, full Gomad v3, formatting, and lint gates
  pass; and
- existing code comments have been preserved through moves and renames.
