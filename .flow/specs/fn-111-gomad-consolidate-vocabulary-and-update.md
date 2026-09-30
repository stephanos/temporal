# Gomad: consolidate vocabulary and update documentation

**Plan date:** 2026-09-30

## Goal & Context

Give Gomad readers one canonical vocabulary and consistent descriptions of
implemented behavior across the product specification, architecture, CLI guide,
and tutorial. Merge the useful glossary definitions into
`tools/gomad3/SPEC.md` and delete `tools/gomad3/GLOSSARY.md`, as selected by the
user. Preserve each existing product requirement identifier and the distinctions
between target selection, prepared execution, observations, replay controls,
backend mechanism, and fidelity claims.

This spec captures `.turbo/plans/gomad3-glossary-update.md`. The terminology
assessment is retained below so acceptance does not depend on an ignored local
plan file. Documentation edits already exist in the working tree; the spec
records their contract and remaining reconciliation and acceptance rather than
requiring a second implementation. It remains open until acceptance evidence
has been recorded through the normal Flow lifecycle.

### Relationship to existing work

- [fn-109 R9](fn-109-gomad-deepen-modules-and-tool-interfaces.md) and fn-105 D5
  retain ownership of architecture documentation associated with the broader
  interface and ownership changes. Reuse this spec's verified documentation
  evidence where it satisfies those requirements; do not duplicate tasks or
  declare future interface documentation complete from today's edits.
- fn-105 D12/D14 retain replay-divergence fixes and qualification restoration.
  This work describes those limitations and does not alter their dispositions.
- fn-105 D13/D15 retain tracing-policy and capacity work. Describe the implemented
  policy at acceptance time without claiming the proposed changes have shipped.

## Documentation Ownership

- `tools/gomad3/SPEC.md` owns canonical product vocabulary, requirements, and
  stable semantic identifiers. Keep definitions abstract; implementation types,
  schema versions, and package layout remain with their existing owners.
- `tools/gomad3/ARCHITECTURE.md` explains ownership boundaries and design rationale.
- `tools/gomad3/CLI.md` explains user and maintainer commands against current parsers.
- `tools/gomad3/TUTORIAL.md` explains the execution and reproduction journey.
- `tools/gomad3/README.md` anchors installation, development, supported behavior,
  and historical simulation narrative; link its vocabulary entry to SPEC.

Product prose may use shorter canonical terms with explicit aliases for existing
names. Public Go symbols, flags, defaults, serialized fields, canonical bytes,
replay identities, and existing comments retain their implementation contracts.

## Terminology Decisions

| Original term | Preferred term and action | Assessment and alternatives |
| --- | --- | --- |
| Gomad Toolchain | Keep **Gomad Toolchain** | Identifies the pinned patched Go toolchain. “Deterministic Toolchain” would imply a broader guarantee than the qualified boundary provides. |
| Runner | Keep **Runner** | Covers preparation, supervision, limits, and evidence. “Executor” would describe only its launch responsibility. |
| Target | Split **Target** and **Prepared Target** | Target names the user's selected program/test/executable; Prepared Target names the reviewed immutable executable and bound inputs. “Binary” loses provenance and execution inputs. |
| Campaign | Keep **Campaign**; correct definition | Names the bounded exploration effort, selected Executions, policy, and evidence. The immutable plan is a separate object. “Batch” obscures adaptive exploration and resume. |
| Execution | Keep **Execution**; broaden controls | One isolated attempt with a Seed and applicable runtime/model controls. “Run” is overloaded across command, Campaign, and Node lifetime. |
| Seed | Keep **Seed** | Names the reproducible root value selecting deterministic alternatives. “Random Seed” would blur runtime scheduling and separately derived application entropy. |
| Choice | Keep **Choice** | Names one eligible runtime selection. “Scheduling Decision” excludes ready `select` alternatives; use “Decision” for the encoded tape entry where needed. |
| Choice Trace | Keep **Choice Trace**; correct contents | Observed bounded runtime evidence includes observations and nonbranching decisions. “Decision Tape” is a different projection used for exact replay. |
| Choice Replay Plan | Keep **Choice Replay Plan** for full/prefix controls; use **Decision Tape** for exact replay | Exact tapes omit observations and nonbranching entries; forced prefixes support exploration before seeded execution continues. “Replay Plan” alone conflicts with simulation and portable campaign plans. |
| Deterministic I/O Contract | Keep **Deterministic I/O Contract** | Names the transparent I/O support, identity, adapters, limits, and transcript rules. **Interaction Boundary** is the broader reviewed capability contract; treating them as identical hides their scope. |
| Captured Read-Only Input | Keep **Captured Read-Only Input** | Preserves host origin, capture, immutability, and host-independent replay. “Captured Input” is shorter but loses the read-only mount distinction; “Snapshot” could imply eager capture of an entire directory. |
| World Model | Prefer **World**; retain alias | Matches the specification and explicit-event API. “World Model” adds a redundant suffix and can imply ownership of application state or all transparent I/O. |
| Cluster | Keep **Cluster**; expand definition | Application-facing simulation control includes lifecycle, topology, faults, histories, observations, and oracles. “Simulation Harness” describes the surrounding facility, not this contract alone. |
| Node | Keep **Node** | Stable simulated participant across restarts. “Service” excludes clients and other participants; “Process” excludes in-process nodes. |
| Incarnation | Keep **Incarnation** | Names one monotonically identified Node lifetime. “Instance” can mean the stable Node; “Execution” already names the outer isolated attempt. |
| Boot Identity | Keep **Boot Identity** | Names the stable identity of a registered boot function. “Boot ID” is suitable for a symbol, but can be mistaken for an Incarnation identifier in prose. |
| Simulation Backend | Prefer **Backend** in simulation context; retain **Simulation Backend** as qualifier | Names the execution mechanism. Both in-process and process mechanisms exist; runtime/coordinator availability still applies. “Fidelity” names a separate claim. |
| Simulation Model Fidelity | Prefer **Model Fidelity**; retain alias | Removes a redundant qualifier while preserving the detached model-behavior claim. “Model Correctness” would imply correctness beyond the declared contract. |
| Hard Isolation Fidelity | Prefer **Hard Isolation**; retain alias | Directly names fresh runtime state and process cleanup available with the process backend. “Process Backend” names a mechanism; the claim remains independently recorded. |
| Parity Case | Retire from current vocabulary; keep historical narrative | Names a removed v2-to-v3 manifest concept. **Conformance Case** would suit a current check, but renaming historical parity cases would imply a surviving current entity. |
| Execution Record | Prefer **Record** in product prose; retain **Execution Record** as qualifier | Aligns with `SPEC.md`; `record.ExecutionRecord` remains the implementation name. “Result” blurs the canonical evidence object with an Outcome or command return value. |
| Artifact | Keep **Artifact** | Names the immutable validated content-addressed package carrying a Record and replay payloads. “Bundle” already has installation/portable-plan uses. |
| Replay | Keep **Replay**; distinguish verification-only and **Exact Replay** | Validation precedes optional re-execution of the stored target. Exact Replay forces and consumes recorded decisions. “Reproduction” omits integrity checks and verification-only operation. |
| Corpus | Keep **Corpus**; replace “interesting” with semantic novelty | Established collection term for bounded replay-verified cases guiding later Campaigns. “Seed Bank” omits retained artifacts, captured inputs, and coverage evidence. |
| Qualification Set | Keep **Qualification Set** | Versioned workloads, expectations, and bounds reported as one support claim. “Test Suite” omits capability analysis, platform identity, and unsupported dispositions. |

## Other vocabulary corrections

| Term | Recommendation | Reason |
| --- | --- | --- |
| Prepared Target | Add to the merged vocabulary | Separates selection from reviewed executable identity; preparation happens once per Campaign. |
| Decision Tape | Retain and clarify | Names exact branching runtime decisions, independently of the richer Choice Trace. |
| Fidelity | Add as a shared concept | Backend and Fidelity are separate fields; the process backend also accepts model fidelity. |
| Choice Exploration | Define as an activity | The existing definition describes pending work rather than the exploration effort. “Runtime Choice Exploration” is a useful qualifier when comparing dimensions. |
| Combined ChoiceExploration | Rename prose to **Combined Exploration** | Fixes the joined word and covers runtime, scenario, network, storage, fault, and crash dimensions. Preserve `CAMPAIGN.COMBINED.FRONTIER` and existing strategy flags. |
| Frontier | Add for remaining work | Gives the bounded set of unvisited alternatives its own name instead of overloading “Exploration.” |
| Interaction Boundary | Retain | Names the reviewed external capability boundary; document its relationship to the narrower Deterministic I/O Contract. |
| Simulation | Retain | Covers nodes, modeled network/storage, scenarios, faults, observations, and oracles; Cluster is its application control contract. |
| Evidence, Outcome, Observation, Transcript | Retain their distinct definitions | Respectively name support for claims, semantic classification, detached behavior, and ordered modeled interactions. None can replace Record or Artifact. |
| Identity, Provenance, Coverage, Probe | Retain their distinct definitions | Separate sameness, trusted build origin, observed semantic/choice features, and named semantic events. |
| Scenario, Fault Plan, Oracle | Retain their distinct definitions | Separate composed actions, deterministic fault selection, and correctness evaluation. |
| Portable Plan, Shard, Aggregate | Retain their distinct definitions | Separate portable work identity, disjoint selected work, and a validated merged result. |
| Recovery, Resume | Retain both | Recovery repairs publication state; Resume may execute unfinished work. |
| Qualification, Qualified Platform | Retain both | Separate repeated evidence comparison from a platform support claim. |
| Capability Review, Supported Target, Unsupported Target | Retain | Distinguish reviewed capability support from an application's success or failure. |
| Support Comparison, Compatibility Pack, Conformance Tier, Release Gate, Upgrade Dossier | Retain | Name different support/maintenance products; simpler generic “report,” “adapter,” or “test” labels would erase their contracts. |
| Exact Replay, Replay Divergence | Retain | Separate complete forced replay from its first mismatch; ordinary seeded replay need not have a runtime tape. |
| Target Failure, Watchdog Observation, Infrastructure Failure, Capacity Exhaustion | Retain | Preserve classification boundaries; wall-time expiration does not establish deterministic deadlock. |

## Acceptance Criteria

- **R1:** Assess every one of the 25 original glossary entries, retaining the
  recommendation, alternatives, and rationale above. All 24 current concepts
  survive in SPEC as definitions or explicit aliases. Parity Case has an explicit
  historical disposition in the README narrative. Errors: an unassessed term,
  lost current concept, or historical entity presented as current leaves
  acceptance incomplete.

- **R2:** `PRODUCT.VOCABULARY` in SPEC is the single canonical vocabulary;
  GLOSSARY.md is deleted and current guides link to SPEC. Preserve every original
  semantic identifier, including command-table identifiers, in its original
  order; add only unique identifiers if a new requirement needs one. Errors:
  broken definitions, duplicate vocabulary owners, or renamed/reused identifiers
  fail acceptance.

- **R3:** Definitions distinguish Target from Prepared Target, Campaign from its
  immutable plan, Choice Trace from the branching Decision Tape, and complete
  exact replay from finite forced prefixes. Backend describes the mechanism;
  Fidelity describes the claim, with Model Fidelity on both backends and Hard
  Isolation limited to the process backend. World remains separate from host
  I/O and application state. Errors: conflated concepts or unsupported isolation
  and replay guarantees fail acceptance.

- **R4:** ARCHITECTURE describes implemented tracing, exact replay, bounded Choice
  Exploration and Combined Exploration, both simulation backends, and the
  supported darwin/arm64 and linux/amd64 platform bundles. Distinguish platform
  capability from workload qualification and record current residual findings.
  Describe strict/forward time reads, the native timer clock, and the influence
  of time.Now-derived deadlines accurately. Errors: obsolete delivery claims,
  invented guarantees, or qualification stated beyond evidence fail acceptance.

- **R5:** CLI command names, flag spelling and placement, examples, defaults,
  resource bounds, exit classifications, capability modes, and user/maintainer
  workflows agree with implemented parsers and tests. Preserve literal flags and
  strategy values while aligning prose terms. Errors: nonexistent commands,
  invalid examples, changed defaults presented as current, or conflation of
  watchdog observation with target failure fail acceptance.

- **R6:** TUTORIAL uses the merged vocabulary for preparation, isolated
  Executions, optional recording, replay, World, Simulation, retention,
  exploration, and minimization. Separate seeded repeatability from verified
  runtime Choice replay and Backend from Fidelity. Preserve the existing
  GOMAD_NEXT roadmap update. Errors: blanket determinism claims, unrecorded exact
  replay claims, or superseded platform/gap statements fail acceptance.

- **R7:** All concrete local Markdown links and fragments resolve, code fences
  balance, and active guides contain no reference to the deleted glossary or
  malformed Combined ChoiceExploration vocabulary. Preserve historical task and
  artifact references as evidence. The README links SPEC and avoids a stale
  simulation schema number. Errors: broken navigation or erasing historical
  evidence fails acceptance.

- **R8:** Retain a bounded acceptance summary identifying the source revision,
  files checked, term/identifier comparisons, command/flag cross-checks, link and
  fence validation, and whitespace results. Map evidence to R1-R7 and the reused
  portion of fn-109 R9/D5; broader interface documentation remains with fn-109.
  Errors: passing claims without actual checks, reliance on the ignored plan
  alone, or marking unrelated qualification work complete fails acceptance.

## Verification

This scope changes documentation and Flow tracking only. Runtime tests are
required only if executable code changes under a separately authorized scope.

- Recover the pre-consolidation glossary and SPEC from the recorded source
  revision. Count only the glossary's Language section; expect 25 assessments
  and 24 current definitions or aliases.
- Compare all semantic identifiers, including table entries, against that
  revision. The initial documentation update preserved 123 identifiers; verify
  against the actual recorded revision before making an acceptance claim.
- Cross-check commands and flag names with `cmd/gomad/internal/cli` and
  `cmd/gomadtool`, and values/behavior with relevant parser and regression tests.
  The initial update checked 29 command-index entries and 67 documented flags;
  recheck if later edits change the inventory.
- Validate local links, fragments, code fences, term coverage, and whitespace
  across SPEC, ARCHITECTURE, CLI, TUTORIAL, and README.
- Compare platform claims with `toolchain/version/version.json` and
  `deterministicio/boundary/manifest.json`; use milestones and qualification
  reports for residual workload dispositions.
- Inspect the scoped diff and run `git diff --check`. Preserve other working-tree
  edits; no commit or history operation is part of spec creation.

## Boundaries

- No API, command, flag, default, schema, runtime, protocol, dependency,
  qualification-expectation, or capability-boundary changes.
- No new glossary or parallel architecture task owner. Future public-symbol
  renames require a separate compatibility decision and consumer benefit.
- No speculative platform, performance, isolation, or determinism claims.
- Creating this spec and indexing it in milestones starts no implementation
  tasks and does not mark the existing working-tree edits accepted.

## Context Files

- `tools/gomad3/SPEC.md`, `ARCHITECTURE.md`, `CLI.md`, `TUTORIAL.md`, and `README.md`.
- The pre-consolidation `tools/gomad3/GLOSSARY.md` and SPEC in Git history.
- `tools/gomad3/target/target.go`, `choice/trace.go`, and `choice/tape.go`.
- `tools/gomad3sim/types.go`, `spec.go`, and `cluster.go`.
- `tools/gomad3/toolchain/version/version.json` and
  `tools/gomad3/deterministicio/boundary/manifest.json`.
- `.plans/GOMAD_MILESTONES.md`, fn-105 D5/D12/D13/D14/D15, and fn-109 R9.
