# Gomad: finish downstream cell integration and qualification

**Plan date:** 2026-09-30

## Goal & Context

Run a Walker-backed Temporal cell in one process under Gomad, with matching
same-seed executions and exact replay on `darwin/arm64` and `linux/amd64`.
This spec owns the remaining implementation and qualification work described
in [GOMAD_CLOUD.md](../../.plans/GOMAD_CLOUD.md).

[fn-104](fn-104-gomad-run-a-downstream-cell-under-the.md) delivered downstream
module preparation, external compatibility-pack loading, an address-library
adapter, and boundary measurements. Its September 30 linked-mode experiment
still found 44 live blockers after loading an external pack. It closed through
the acceptance branch that permits classified blockers. This spec requires
the downstream smoke test to pass and replay; recording an unsupported outcome
does not satisfy its completion criteria.

Implementation spans the Temporal Gomad repository and the consuming
`saas-temporal` repository, including its nested `walker` module. Flow tracking
lives here; downstream source, configuration, workloads, and dependency packs
belong in the consuming repository. Implementation was authorized by the request to work through fn-107.

The scope also includes the three downstream follow-ups in
[fn-105](fn-105-gomad-follow-ups-deferred-scope.md): D8 closure-mode support,
D9 Linux packs and qualification, and D10 the downstream seam guide. The
request to finish this consumer's remaining work supplies the revival reason.
When breaking this spec into tasks, reuse or transfer those existing task
obligations instead of creating duplicate implementation tasks.

## Architecture & Data Models

### Downstream execution profile

Provide an opt-in configuration for the localcell harness that runs Temporal,
Walker datanodes, walkabouts, bimmers, and the smoke workload in one process.
Use the existing localcluster lifecycle, storage interfaces, and configuration
injection points. A small profile constructor should supply the filesystem,
membership transport, capacity policy, providers, and explicit addresses;
those implementations must be testable independently of cluster startup.
Keep defaults and production behavior unchanged and preserve existing comments.

Select the smallest configuration that exercises Walker-backed execution and
history persistence. Add a dedicated downstream smoke entry point if the
existing `TestLocalCell_Walker` requires unrelated external services. Reuse
the harness and real storage/control-plane services rather than duplicating
their implementation or substituting a success-only fake for Walker.
Document every external dependency the selected configuration removes or
substitutes. Any required substitute is in-process and supplied through an
explicit downstream interface; the support claim covers only that configuration.

### Resolved persistence composition

The dedicated `saas-temporal/localcell/gomad` package uses the actual Walker
`localcluster` services and Temporal `testcore` cluster factory. It avoids the
root localcell chaos and server-tooling closure. Supply CDS's existing
`WedgeShardController` and `ExecutionStoreWrapper` with the real per-shard
Walker provider; direct Walker leaf stores do not implement the OSS execution
store contract. Expose existing conversion/interceptor wiring through a small
constructor rather than duplicating it.

Inject process-local CDS shard, metadata, watermark and WAL dependencies with
stateful records, ownership/CAS, stream reads and writes. SQL supplies OSS
namespace, cluster metadata, matching task/fair task, queue/QueueV2 and Nexus
stores. CDS supplies OSS shard ownership and execution/history stores so
acquisition activates and recovers Walker-backed shards. The SQL backing must
be the same one seeded by testcore before its additional server options are
applied. The profile owns these resources and closes them with the cluster.
The injected per-history factory borrows testcore's SQL factory. It owns its
CDS controller and real namespace registry backed by a distinct SQL metadata
store handle, and closes those idempotently without closing the borrowed SQL
factory. Testcore retains that base factory's lifecycle. Profile cleanup is a
backstop for owned resources, including constructor failures. Cassandra,
etcd/SMS, BOSS WAL proxies and cloud object stores are excluded from
this support profile. A negative workload must show that denying or stopping
Walker prevents workflow persistence; a SQL-only workflow cannot qualify.

### Task ownership

Tasks fn-107.1 and fn-107.2 implement storage and profile/membership.
The user authorized isolated worktrees for parallel execution: fn-107.6 supplies
stateful bounded CDS auxiliary stores/WAL and fn-107.7 supplies the injectable
factory and host source seams, independently of fn-107.2. fn-107.3 integrates
these three surfaces, then fn-107.4 supplies the dedicated workflow smoke. Existing
fn-105.8 (D8), fn-105.9 (D9) and fn-105.10 (D10) supply closure preparation,
Linux pack/qualification and generic seam guidance. fn-107.5 consumes their
evidence and performs final requirement reconciliation. Flow dependencies are
spec-local: the conductor must enforce the external ordering manually and
cannot finish fn-107.5 before these three obligations pass. Their revival
reason is this spec's dual-platform closure-mode qualification gate.

### Requirements from the boundary table

| Source finding | Required implementation | Owner |
| --- | --- | --- |
| UDP membership | Inject a TCP-only transport while retaining membership discovery and updates | Downstream |
| Advisory locks, `chown`, `link`, raw descriptors, file deadlines, `ReadFrom`/`WriteTo` | Inject the storage engine's filesystem through its existing interface, including `VolumeOptions.WithFS`; audit every engine/replica path for fallback to host I/O | Downstream |
| `statfs` and volume discovery | Supply deterministic capacity from configuration and avoid host volume-discovery subprocesses | Downstream |
| Interface enumeration and address resolution | Configure bind, advertise, and peer addresses as IP literals; use loopback for the qualified configuration | Downstream |
| Process metrics | Exclude host process collectors under `gomad`, retaining application metrics | Downstream |
| Subprocesses and signals | Isolate volume discovery, repository-root lookup, cluster subprocess mode, certificate/port-forward proxies, and signal registration behind source seams or dependency injection | Downstream |
| CLI and cloud credentials | Remove CLI-only providers and live credential discovery from the in-process closure; explicitly supply the needed local providers | Downstream |
| Signal-handling third-party metrics library | Add an exact adapter if the final closure still reaches the library; prove elimination otherwise | Gomad, D8 |
| Foreign dependency facts and versions | Retain reproducible external pack inputs, review platform-specific facts, and qualify the selected packs on both hosts | Downstream and Gomad, D9 |
| Test driver and instructions | Use the Gomad CLI/manifest without `-race`, publish the seam and qualification guide | Downstream and Gomad, D10 |

All-interface TCP binds, concrete `*net.TCPListener`/`*net.TCPAddr` values, port
probing and rebinding, and native readiness timers are already modeled. Reuse
these capabilities and test their composition with the consumer. DNS support
remains limited to `localhost`; configured IP literals avoid other lookups.

### Dependency preparation and qualification

The dedicated smoke package owns an opt-in `localcell/gomad/go.mod`. Its main
module replacements link the consuming root and Walker modules to these
checkouts and link the server to the sibling Gomad checkout. Mirror required
fork/pin replacements from the consuming root because replacements in dependency
modules do not propagate. This keeps the published dependency defaults in the
root and Walker modules intact. The native repository test driver already
discovers arbitrary nested modules; invoke it from the repository root. The
Gomad CLI uses the smoke module as its working directory with `GOWORK=off`.
The smoke uses the existing Walker `hashicorpmetrics` build tag in both native
and Gomad runs. Record it with `test_dep`, `integration` and, for Gomad,
`gomad`; do not combine it with `armonmetrics`. Fresh closure evidence determines
which metrics library needs an exact signal-refusal adapter.

Prepare the dedicated smoke module through `--working-dir`. Its checked main
`go.mod` owns the effective local replacements for the Gomad server checkout
and the consuming root and Walker modules. A workspace file is not a replacement
for that graph because the Runner uses `GOWORK=off`. Keep configuration/schema inputs in recorded read-only mounts or
deterministically initialized model state. Export required private-module
settings for host-side preparation.

Retain downstream pack requests, reviewed facts, generation inputs, and packs
in the downstream repository. Use the existing `--compatibility-root` flow
and `GOMAD3_COMPATIBILITY_PACKS` loading. Adapter and pack identities bind the
exact module versions, sums, source digests, platform, and preparation inputs.
Local replacement findings require source seams or injection; an exact pack
cannot authorize them.

A checked downstream manifest selects the smoke target, build tags, seeds,
clock policy, mounted inputs, required probes, choice/transcript capacities,
wall watchdogs, and artifact retention bounds. It requires successful exact
replay in closure mode on both platforms. Linked analysis remains a diagnostic
and must also report supported. Every run retains machine-readable reports
with the actual platform, target and tool identities.

## API Contracts

Reuse existing typed filesystem, membership, provider, and cluster options.
Add only the injection points needed to keep the qualified configuration
independent of host operations. Profile construction rejects missing providers,
invalid capacity, unsupported execution modes, and incompatible address
configuration before starting services. The default construction path retains
its current behavior.

The public execution interface remains the Gomad CLI and qualification
manifest. Preserve existing artifact schemas, exact identity checks, and
failure classifications. Invalid preparation inputs remain invalid input;
unsupported operations, execution failures, watchdogs, and replay divergence
remain visible failures and cannot satisfy a `qualified` expectation.

## Edge Cases & Constraints

- Apply the [milestone constraints](../../.plans/GOMAD_MILESTONES.md#constraints)
  and current Gomad [specification](../../tools/gomad3/SPEC.md). Keep the
  boundary fail-closed. Generic `syscall`, `x/sys`, subprocess, signal, or host
  I/O grants are excluded.
- Use `gomad`/`!gomad` seams for host-dependent implementations and dependency
  injection for modeled resources. Select source at build time where closure
  analysis would otherwise retain a forbidden import despite a runtime branch.
- Scope filesystem state to the profile/cluster so separate tests cannot share
  stale volumes or lock state. Preserve storage write/read, lock ownership,
  reopen, and error semantics in isolated tests; an in-memory filesystem is
  not evidence of real disk durability.
- Configure capacity with finite valid values and deterministic failures. Avoid
  reporting host free space or silently ignoring a restore-capacity failure.
- Readiness and shutdown use virtual deadlines plus host watchdogs. Keep
  `strict` or `forward` clock policy explicit and recorded; neither policy
  supplies missing I/O readiness. Do not weaken assertions or shorten logical
  deadlines simply to make a stalled cluster pass.
- Adapters remain exact and single-version. Dependency drift must fail
  preparation until the changed inputs are reviewed and qualified. Provide
  negative evidence for unavailable/changed external packs and adapter pins.
- The Linux replay channel tracked by fn-105 D12 and the Darwin residual in
  D14 remain separately owned. If they affect this target, retain evidence and
  record the dependency; this spec stays incomplete until its required runs
  replay exactly. An intermittent expectation is not a substitute.
- Preserve native integration testing alongside Gomad. The qualified smoke
  configuration makes no claim about production throughput, host scheduling,
  signals, real disk behavior, or external-service semantics.

## Acceptance Criteria

- **R1:** A reproducible downstream smoke target runs the actual Temporal and
  Walker services in-process, starts and completes a workflow through
  Walker-backed execution/history persistence, and verifies the result and
  recorded history. Its configuration names every external dependency removed
  or substituted. Errors: required external services, missing providers,
  subprocess mode, and invalid configuration fail visibly; boot-only or
  storage-bypassing workloads do not count. [inferred]
- **R2:** Source seams or injected implementations remove all live downstream
  subprocess/signal sites, repository-root and volume-discovery commands,
  certificate/port-forward proxies, CLI-only providers, and cloud credential
  discovery from the qualified closure. Errors: selecting one of these host
  paths under `gomad` is rejected rather than falling back to host execution.
  [paraphrase]
- **R3:** Every engine, replica, restore, and local blob path used by the smoke
  target uses an injected in-memory filesystem or an already modeled boundary.
  Isolated tests cover write/read, close/reopen, per-cluster isolation and lock
  lifecycle. Errors: missing files, failed writes and lock conflicts propagate;
  raw descriptors and unsupported file operations cannot escape to the host.
  [paraphrase]
- **R4:** Storage and restore capacity comes from validated deterministic
  configuration, eliminating `statfs` and volume-discovery host dependencies.
  Errors: invalid capacity and insufficient configured capacity are tested and
  fail predictably without querying host disk state. [paraphrase]
- **R5:** Membership uses an injected TCP-only transport and explicit IP
  bind/advertise/peer addresses. Tests cover member discovery and updates,
  unspecified loopback-normalized binds, concrete listener/address types,
  duplicate binds, and close/rebind. Errors: UDP, interface enumeration and
  non-localhost DNS remain denied; invalid peer configuration fails before
  startup. [paraphrase]
- **R6:** The qualified profile excludes host process collectors while retaining
  application metrics and deterministic service startup/readiness/shutdown.
  Errors: an unhealthy cluster or leaked service fails under explicit logical
  deadlines and wall watchdogs; metrics collection never reads host process
  state. [paraphrase]
- **R7:** Closure-mode preparation either eliminates the signal-handling
  third-party metrics library or uses an exact, platform-qualified adapter
  that excludes host signal registration. Errors: changed source digests or
  incompatible module versions fail preparation; generic signal admission is
  rejected. This fulfills fn-105 D8. [paraphrase]
- **R8:** The downstream owns reproducible pack authoring inputs and reviewed
  packs for the final target on `darwin/arm64` and `linux/amd64`, using the
  existing external-pack flow. Errors: changed or missing pack identities,
  wrong platform/version, and attempted admission of forbidden imports or local
  replacement findings fail closed. This fulfills fn-105 D9's pack work.
  [paraphrase]
- **R9:** Fresh `gomad analyze` reports `supported` with no live blockers for
  the final downstream smoke target in both closure and linked modes on each
  platform. Reports identify the exact target, tags, replacements, packs and
  adapters. Errors: unsupported analysis or preparation failures leave this
  criterion unmet; documented blockers alone are insufficient. [inferred]
- **R10:** A checked downstream qualification manifest requires `qualified`
  outcomes on both platforms for seeds 11 and 17, two fresh same-seed
  executions each, and exact replay of every retained success with runtime
  choice and I/O evidence. Execute it through the Gomad CLI without `-race`;
  record explicit resource bounds, clock policy and inputs. Errors: test
  failures, skips replacing the smoke assertions, missing evidence, capacity
  overflow, watchdogs and replay divergence fail the gate. Reports and
  replayable artifacts are retained within declared bounds. This fulfills
  fn-105 D9's qualification work. [inferred]
- **R11:** Relevant native tests/builds with `gomad` absent verify unchanged
  defaults, and isolated profile tests verify the added injection points and
  negative cases. Use `test_dep`, the integration tag for integration tests,
  and the repository's native test entrypoints. Errors: native regressions
  remain failures; Gomad support cannot depend on changing production behavior
  or weakening existing test assertions. [paraphrase]
- **R12:** A generic Gomad-side seam guide, without downstream repository or
  component names, documents the tag pairing, default-build
  invariant, injected resources, closure-versus-linked analysis, exact dependency
  policy, module replacements, driver choice, qualification/replay commands and
  external-service limits. Downstream instructions name the concrete profile,
  target, packs, configuration and commands. Update `GOMAD_CLOUD.md` and milestone tracking with
  measured support and the remaining findings, and link fn-105 D8/D9/D10 to
  their fulfillment here. Errors: a command depending on unrecorded local edits
  or reporting a classified failure as passing is not acceptable. This fulfills
  fn-105 D10. [paraphrase]

## Boundaries

- General subprocess, signal, UDP, non-loopback network, or external-service
  models are outside this spec. The supported target uses explicit in-process
  dependencies. External base stores, log proxies, coordination services and
  object stores remain outside its support claim.
- Mixedbrain, the subprocess localcell chaos catalog, SIGKILL/SIGTERM behavior,
  and real resource-constrained Docker execution remain native integration work.
- No source translation, Gomad-specific rewriting of existing test assertions,
  race-detector profile, multi-P execution, multi-version adapter system, or
  new platform beyond the two qualified hosts.
- Existing modeled TCP/time operations and completed fn-104 preparation work
  are reused. A new Gomad modeled operation requires its own contract, resource
  bound, transcript coverage, exact replay, and negative evidence under COMPAT-5.
- Implementation and verification are authorized. The user retains commit
  ownership; this work creates no commits, pushes, or deployments. The user
  subsequently authorized isolated worktrees for parallel implementation.

## Decision Context

The storage and membership layers already expose injection interfaces. Using
them keeps host-specific behavior out of a small profile and permits meaningful
isolated tests without expanding Gomad's generic boundary. The qualified
configuration trades real disk and UDP behavior for reproducible in-process
execution; native integration tests retain coverage of those production paths.

Closure analysis makes the downstream manifest usable with the same policy as
Temporal's checked functional workloads. Linked analysis helps identify which
dependencies the linker already removes. Both platforms have separate reviewed
packs and evidence, and artifacts replay only on their producing platform.

The prior spec's classified-blocker completion was appropriate for capability
discovery. This follow-up has a named consumer and requires successful exact
replay. D8/D9/D10 supply existing obligations; broader runtime determinism
investigations remain with their current owners.
