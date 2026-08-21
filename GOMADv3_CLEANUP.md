# Gomad v3 cleanup findings

Date: 2026-08-21

Scope: `tools/gomadv3` and the Gomad v3 documents that describe its package model. This is an analysis-only cleanup backlog; it does not prescribe a flag day or classify generated/compatibility material as stale merely because it is old.

## Executive summary

The main problem is not the number of packages. It is that a few broad packages own several independently changing use cases while a few other packages are only aliases, transport fragments, or misplaced tooling. The highest-leverage cleanup is to deepen the execution, artifact, qualification, and deterministic-I/O boundaries, then make the architecture test enforce those boundaries.

The current tree has 514 tracked files and about 113,700 lines. The largest structural pressure is in `runner` (roughly 7,600 production lines), followed by `qualification`, `deterministicio`, `evidence`, `target`, and `toolchain`. `runner/runner.go` alone is 1,836 lines, and `runLocal` occupies roughly 800 of them. These are symptoms of mixed ownership, not reasons to split files arbitrarily.

The recommended direction is:

```text
cmd/
  gomad/                  user CLI
  gomadtool/              developer/generation/upgrade CLI

runner/                   small campaign facade and stable public DTOs
  internal/execution/     one execution lifecycle
  internal/campaign/      campaign orchestration and recovery
  internal/exploration/   shared engine with choice and simulation adapters

record/                   execution record schema, validation, identity
artifact/                 artifact publication, storage, opening, payloads

qualification/
  analysis/               static capability analysis
  comparison/             support comparison
  set/                    checkpointed qualification-set orchestration

deterministicio/
  readonlymount/          captured read-only input subsystem
  ...                     contract, transcript, and session internals

target/                   deep public preparation/review facade
  internal/build/
  internal/capabilityreview/
  internal/provenance/
```

This is a direction, not a requirement to expose every directory above. Prefer private packages or files until an interface is both meaningful and independently testable.

## Ranked backlog

| ID | Finding | Impact | Effort | Priority |
|---|---|---:|---:|---:|
| C1 | Separate execution, campaign orchestration, and inspection in `runner` | High | High | P1 |
| C2 | Split execution records from artifact storage/publication | High | High | P1 |
| C3 | Consolidate the duplicated choice and simulation exploration engines | High | High | P1 |
| C4 | Split qualification by use case and remove the `toolchain -> qualification` dependency | High | High | P1 |
| C5 | Extract the cohesive read-only-mount subsystem from `deterministicio` | High | Medium | P1 |
| C6 | Centralize process-group lifecycle and checked host execution | High | Medium | P1 |
| C7 | Keep `target` deep while moving build, review, and provenance mechanics behind it | High | High | P2 |
| C8 | Align package and API vocabulary with Campaign, Execution, Qualification Set, and Gomad | Medium | High | P2 |
| C9 | Move developer tooling and generators to their actual owners | Medium/High | Medium/High | P2 |
| C10 | Collapse shallow World process-session packages | Medium | Medium | P2 |
| C11 | Replace coarse owner-level architecture checks with module-edge rules | High | Medium | Continuous |
| C12 | Remove or assign owners to confirmed stale and orphaned artifacts | Medium | Low | P0 |

## Findings

### C1. `runner` owns too many use cases

Evidence:

- `runner/runner.go:123` exposes a `CampaignSpec` of roughly 42 fields, including preparation, execution, replay, process commands, storage, resume state, and evidence policy.
- `runner/runner.go:323-1127` implements most of a campaign in `runLocal`.
- The root package also owns resume, process coordination, inspection, replay command projection, frontier campaigns, simulation campaigns, and public evidence DTOs.
- `runner/runner_test.go` is 2,481 lines, tracking the same breadth.

Recommendation:

- Keep `runner` as the stable campaign entry point and public result/configuration surface.
- Put the single-run state machine, supervisor protocol, deterministic-I/O session, and process lifecycle behind one deep execution module.
- Put campaign scheduling, recovery, journaling, and result aggregation behind a campaign module.
- Keep inspection as a structured data API. Move human/CLI command rendering to `cmd/gomad`.
- Do not expose aliases for internal seams merely to preserve the current package shape.

Migration constraint: campaign directories, JSON tags, replay identities, journal formats, and domain hashes are compatibility boundaries. Refactor behind existing codecs first; change stored schemas only through explicit versioned migration.

### C2. `evidence` conflates records, codecs, and artifact storage

`evidence` currently owns three different responsibilities:

- canonical JSON and strict decoding in `evidence/canonical.go`;
- execution-record schema, validation, and identity in `evidence/types.go`, `validation.go`, and `identity.go`;
- filesystem artifact publication/opening/payload handling in `evidence/store.go` and `open.go`.

Artifact publication is also defined inside `runner/internal/campaignstore/artifact.go`, causing corpus admission and other non-campaign code to depend on a campaign store. The architecture document already treats Record and Artifact as distinct concepts.

Recommendation:

- Make `record` own the immutable execution record, validation, identity, and compatibility decoding.
- Make `artifact` own durable publication, atomicity, opening, payload access, and layout validation.
- Move generic JSON mechanics to a private shared module only if both Record and World delegate to it; do not create a public utility package.
- Move `PublishArtifact` out of `campaignstore`. Campaigns should consume the artifact interface rather than own it.

This boundary should be implemented before large runner moves because it removes an important dependency tangle.

### C3. Choice and simulation exploration have parallel engines

The following pairs encode the same broad queue/round/journal lifecycle with different domains:

- `runner/internal/frontier` and `runner/internal/combinedfrontier`;
- `runner/frontier_campaign.go` and `runner/combined_frontier_campaign.go`;
- campaign-store frontier and combined-frontier journals.

`combinedfrontier` is also relational vocabulary: it does not say that the extra dimensions are runtime, scenario, network, storage, fault, and crash decisions.

Recommendation: extract one private exploration engine with explicit adapters for choice exploration and controlled simulation. The engine should own scheduling, bounds, recovery, and journal transitions; adapters should own domain hashing, candidate expansion, and result projection. Prefer `exploration/choice` and `exploration/simulation` over `frontier` and `combinedfrontier` as package vocabulary.

Persisted journal codecs must remain separate until old campaigns can still resume. Consolidating the in-memory engine does not require immediately merging stored formats.

### C4. Qualification and toolchain dependencies follow the wrong use cases

`qualification` currently combines:

- static analysis (`analysis.go`);
- support comparison (`comparison.go`);
- single-workload execution (`workload.go`);
- checkpointed set orchestration, persistence, aggregation, and publication (`suite.go`).

`RunSuite` in `qualification/suite.go:326-507` performs input normalization, executable validation, module identity, analysis, execution, checkpoints, aggregation, publication, and expectation handling. Meanwhile `toolchain/upgrade.go` imports `qualification`, creating an upward `toolchain -> qualification -> runner` path merely because upgrade orchestration lives in the toolchain package.

Recommendation:

- Split `qualification` into use-case modules: analysis, comparison, and qualification-set orchestration. Keep workload result vocabulary at the smallest common layer.
- Move upgrade orchestration out of foundational toolchain construction. The dependency should be `upgrade -> toolchain + qualification`, not `toolchain -> qualification`.
- Keep `toolchain.Build` as a deep module owning source acquisition, build-key derivation, locking, patch materialization, and publication.
- Separate installation resolution if it remains independently used by the CLI.

The persisted schema already says `qualification-set`; the Go API should converge on `SetManifest`, `SetReport`, `RunSet`, and `Workloads`. Preserve legacy JSON names through compatibility structs rather than carrying two current vocabularies.

### C5. `deterministicio` contains a cohesive filesystem subsystem

The `readonly_*` files implement a complete read-only-mount subsystem: mapping parsing, capture/replay, filesystem policy, inventory, persistence, a broker, wire types, and platform-specific hard-link behavior. At the package root, names such as `Kind`, `Limits`, `Child`, `Entry`, `Snapshot`, `Broker`, and `Prepare` are ambiguous because their read-only-mount domain is missing.

Move this cluster to `deterministicio/readonlymount` (or a similarly explicit private name). Keep transcript/session behavior and the target-facing deterministic-I/O contract separate. Test the extracted module through capture/replay and persistence round trips, not file-by-file helpers.

Do not split `deterministicio` into many schema-sized packages. The goal is three deep concepts—contract, transcript/session, and captured read-only input—not one package per file.

### C6. Process termination and host execution policy are duplicated

Unix process-group termination, probing, escalation, and bounded reaping are independently implemented in:

- `internal/hostexec/command_unix.go:146-271`;
- `runner/internal/execution/supervisor_unix.go:413-543`;
- `runner/coordinator_process_unix.go:18-93`.

Callers in toolchain build, patch, conformance, and upgrade code also repeatedly reconstruct timeout, cancellation, signal, exit, and output classification. Environment sanitization is repeated as well.

Recommendation: deepen `internal/hostexec` around a tested process-group lifecycle, checked completion policy, and environment sanitization. Keep raw results available for conformance tests that intentionally expect failures and timeouts. Keep coordinator/supervisor protocol decisions local, but delegate operating-system lifecycle mechanics.

This is a correctness and leak-prevention cleanup, not just deduplication. Add tests for parent exit before child, ignored TERM, deadline races, cancellation, already-reaped groups, and bounded KILL/reap.

### C7. Keep `target` as a deep facade

`target.Spec` mixes caller intent (`Kind`, `Source`, `Args`, build tags, working directory) with prepared infrastructure (`PreparationRoot`, `ToolchainRoot`, overlay/modfile paths, adapter replacements, capability mode). `target.go` owns preparation, provenance I/O, execution preparation, package discovery, and Go builds; `capability.go` owns compatibility policy, linked-manifest projection, and adapter inventory.

Recommendation:

- Preserve `target.Prepare` and capability review as the public facade.
- Separate input requests from immutable prepared results.
- Move build, provenance, and capability-review mechanics into private modules with compact request/result types.
- Replace boolean build policy such as `rejectUnsupported` with an explicit policy or caller decision at the facade boundary.

Do not turn the internal modules into public forwarding packages. Their value is hiding mechanics behind the existing target abstraction.

### C8. Vocabulary drift obscures current concepts

The glossary establishes Campaign and Execution as the current terms, replacing batch and run. Current APIs still leak the old vocabulary:

- `CampaignPath` serializes as `BatchPath` in `runner/runner.go:86` and `:172`;
- `CampaignSpec.ResumeBatch` is current Go API vocabulary;
- `ExecutionEvidence` serializes as `RunEvidence`;
- campaign-store filenames and CLI output still use batch/run terms.

This should be handled in layers:

1. Use Campaign and Execution in current Go APIs, logs, documentation, and CLI output.
2. Quarantine Batch and Run inside legacy wire/storage codecs where compatibility requires the old field or filename.
3. Version any persisted-schema change; do not silently rename JSON fields or campaign files.

Other vocabulary cleanup:

- standardize Qualification Set instead of mixing Set and Suite;
- replace `combinedfrontier` with simulation/controlled-exploration vocabulary;
- replace abbreviated `packdev` with `compatibilitypack` or `packreview` if that workflow moves;
- avoid `world/target`, which conflicts with top-level target preparation while actually describing the child process;
- make `gomadtool` user-facing usage consistent instead of advertising `hosttool` in subcommands.

### C9. Developer tooling and generators are misplaced

`toolchain/cmd/gomadtool` owns protocol generation, compatibility-pack authoring, conformance, and upgrade orchestration—not only toolchain construction. Its compatibility-pack import requires a special exception in `architecture_test.go`.

Move the binary to `cmd/gomadtool` and make developer tooling an explicit owner. Then split `toolchain/internal/generate`, which is named after a verb, by produced domain:

- boundary discovery/qualification and compiler fixtures;
- cross-endpoint protocol generation, organized by deterministic I/O, Choice, simulation, and live capability while retaining a small shared template core.

`target/packdev` belongs to compatibility-pack authoring, not target runtime behavior. A move may require extracting shared compatibility schema/policy from `target/internal/compatibility`; do this only when the ownership benefit justifies changing that deliberate internal boundary.

`toolchain/internal/conformance/runtime.go` should remain one package and one campaign interface, but its 1,256 lines should be grouped into behavior-local files (compatibility, clocks, linking, scheduling, maps/load, repeatability). Fixture directories can likewise be grouped by runtime, I/O, and compiler use case without pretending each fixture is a module.

### C10. World process-session packages are shallow and ambiguously named

`world/host` is effectively a small forwarding package over `world/internal/transport`. `world/target` implements the other side but calls its role the child, and its name collides with top-level `target`.

Prefer one deep `world/process` or `world/session` module that owns host specification encoding, transport, and child opening. A lower-risk first step is to rename the child side and eliminate the one-function host forwarding layer.

This is a design candidate rather than an immediate move: the split appears intentional and should be checked against external imports. The core `world` model, snapshots, recording, restoration, and replay are cohesive and should not be fragmented. `world/mailbox` is a legitimate adapter seam.

### C11. Architecture checks are too coarse

`architecture_test.go:165-220` groups the entire runner subtree under one owner and permits all same-owner imports. It also contains explicit exceptions such as `toolchain/cmd/gomadtool -> target/packdev`. The test therefore documents broad ownership but cannot prevent dependency inversions inside an owner.

Retain owners for reporting, but add exact rules for the new deep modules. In particular, enforce:

- `upgrade -> toolchain + qualification`, never the reverse;
- campaign orchestration -> execution and artifact interfaces;
- artifact and record do not depend on runner;
- target facade -> private build/review/provenance modules;
- CLI presentation depends on structured inspection, not the reverse.

Update the architecture gate in the same change as every package move; otherwise the cleanup will decay back into broad same-owner coupling.

### C12. Confirmed stale, orphaned, and local artifacts

#### Safe first-pass cleanup

- `target/internal/compatibility/migration_baseline.json` has no repository reference to its filename or schema. Delete it if the migration is complete; otherwise wire it into a named regression test and record its owner and retirement condition.
- `runner/internal/execution/output.go` is only an alias/constructor forwarding layer over `internal/hostexec`, with substantially duplicated tests. Remove the layer and test the owning module.
- `runner.Open` simply calls `Inspect(path, InspectOptions{})` and has no production repository caller. Remove it after confirming it is not an intended external API.
- `runner/inspect.go` and `cmd/gomad/internal/cli/quote.go` contain duplicate `quoteArgument` implementations. Inspection should return data; the CLI should render replay commands.
- `GOMADv3_PKG.md` is an earlier package cleanup review whose measurements and recommendations predate substantial runner/simulation growth. This report supersedes it; archive or delete it after any still-relevant notes are transferred.
- Fix confirmed documentation drift: `README.md` refers to `Session.World` while the implementation exposes `Session.Model()`; `TUTORIAL.md` names nonexistent `world/child`; `ARCHITECTURE.md` names nonexistent `protocol/iowire.json` and `upgradegen`; several Tutorial links are invalid relative paths.
- Remove or correct unused/incomplete Make metadata: `VERSION_OUTPUTS` and `COMPATIBILITY_OUTPUTS` are unused, and `IOWIRE_INPUTS/OUTPUTS` no longer describe all protocol generator inputs and outputs. Either use one generator-owned manifest or rely on the existing phony generation plus content validation.
- Rename or narrow misleading Make targets: `patch-test` runs all of `./toolchain`, `runner-test` covers almost every host package and repeats toolchain tests, and `toolchain: validate` pulls higher-level compatibility-pack validation into foundational builds.

#### Needs an ownership or compatibility decision

- `simulation/parity` has tests and an embedded manifest but no production importer. Because the README calls it the canonical SIM-0 contract, either consume it from validation/generation or demote it to documentation/testdata; do not silently delete it.
- `build.sh`, several `test.sh` modes, and a few generated version accessors have no repository caller. They may be external interfaces, so establish compatibility policy before removal.
- Root conformance scripts can move closer to their fixtures, except stable paths such as `exec.sh` that the repository build consumes.

#### Ignored local state

The current checkout contains about 6.6 GB under ignored `tools/gomadv3/.toolchain` and 12 MB under `.bin`. The largest entries are builds (2.9 GB), Temporal qualification output (1.7 GB), generator cache (1.4 GB), core qualification output (279 MB), and a 179 MB panic-patch candidate. There are also hundreds of zero-byte files. The Makefile has no clean/prune target.

Add separate, explicit operations:

- `clean`: remove reproducible short-lived outputs;
- `prune-cache`: evict build/generator caches by age or size;
- `clean-qualifications`: remove large campaign artifacts only when explicitly requested.

Keep these targets narrow and recoverable in intent. Do not make normal validation delete investigation or qualification evidence.

## Preserve: not cleanup targets

The following looked suspicious during inventory but have active owners or compatibility roles:

- `choice/legacy_v1.go` is used by current trace decoding.
- qualification legacy/previous readers are used by suite/set migrations.
- generated wire code and runtime overlay mirrors are active cross-endpoint contracts.
- compatibility requests, reports, packs, and generation state are generator-owned inputs/outputs.
- the versioned Go patch, boundary reports, expected-intercept inventory, and `version_generated.mk` are tracked build inputs or verified generated artifacts.
- `world`, `choice`, `minimizer`, and the toolchain build pipeline are deep modules despite containing large files; do not split them by file count alone.
- `internal/hostfs`, `internal/hostexec`, `world/mailbox`, and `qualification/corpus` are meaningful seams, not shallow-package candidates.

## Recommended sequence

1. **Contract inventory and P0 cleanup.** Record intended external Go import paths and persisted formats; fix docs, stale metadata, and confirmed orphaned files. Add cache cleanup targets.
2. **Shared correctness seams.** Centralize canonical JSON mechanics and process-group/host-execution policy with focused compatibility tests.
3. **Record/artifact separation.** Extract storage/publication from campaign ownership without changing serialized formats.
4. **Use-case boundaries.** Split qualification and toolchain upgrade/installation responsibilities; extract the deterministic-I/O read-only-mount subsystem.
5. **Runner/exploration.** Introduce the execution and campaign interfaces, then consolidate exploration engines behind adapters while preserving old journal codecs.
6. **Target and World internals.** Deepen private implementation seams and collapse shallow process-session packages after import compatibility is decided.
7. **Enforcement.** Tighten architecture edges as each seam lands; remove transitional forwarding aliases once callers migrate.

Avoid combining package moves with schema changes. Mechanical import moves are easy to review; persistence migration, vocabulary migration, and behavior changes need separate commits and tests.

## Verification expectations

Every structural change should retain or add focused tests for:

- old campaign inspection/resume and journal recovery;
- record/artifact canonical bytes, identity, publication atomicity, and payload opening;
- choice and simulation exploration parity across resume/checkpoint boundaries;
- qualification-set legacy decoding, checkpoints, and aggregation;
- deterministic-I/O capture/replay and read-only-mount persistence;
- process cancellation, timeout, escalation, child leakage, and bounded reaping;
- generated host/overlay wire compatibility;
- exact architecture dependency rules.

Use repository commands with the required test tag, starting with affected packages, then the Gomad validation gates, and finally project linting. Representative commands are:

```sh
go -C tools/gomadv3 test -tags test_dep ./...
make -C tools/gomadv3 validate
make -C tools/gomadv3 test
make lint-code
```

Confirm the exact package command from the relevant Make target before using a narrower substitute. Integration-only tests should additionally use the `integration` tag.
