# Gomad architecture: consolidate execution policy and bootstrap ownership

## Goal & Context

Gomad's system architecture is sound: the patched runtime owns scheduling and native virtual time, Runner owns process lifetime, World owns modeled external events, and Record/Artifact own evidence and persistence. Preserve these seams. The remaining architectural cost is duplicated execution-result policy, a hand-maintained runtime bootstrap endpoint, public test seams exposing inaccessible implementation types, and incomplete enforcement/documentation of the intended package structure.

This is a maintenance spec following F7, dependent on `fn-101-gomad-f7-any-functional-test-and-ci`. F4–F7 remain the delivery priority. Capability closure, observed same-seed/replay divergences, general deterministic-GC research, and Linux clock auditing retain their existing owners. This spec must not relax qualification expectations to make refactoring pass.

Users retain the same CLI, artifact formats, execution bounds, and failure classifications. Maintainers gain fewer places to change protocol and evidence rules. The explicit source-level exception is removal of unusable public executor-injection fields and interfaces; usable public preparation and artifact-replay seams remain.

## Architecture & Data Models

Retain the existing process-per-execution architecture and single-P runtime, generated host/overlay wire codecs, pure World and exploration engines, campaign controller, launch-resource plan, read-only-mount module, artifact store, and capability-review policy.

The runtime's early bootstrap reader becomes another generated consumer of the existing I/O schema. Generation owns frame constants, header recognition, and seed decoding. Runtime retains descriptor reads, early activation, termination, and the allocation-free startup contract. Full checksum/identity validation stays with the existing later validator; this change must not move validation phases or invent a second schema.

Inside Runner, one private completed-execution assessor consumes bounded captured results and explicit policy inputs, returning detached validated World/coverage/evidence and outcome data. It does not receive the complete campaign configuration, launch processes, verify mutable filesystem state, update counters, publish files, or decide whether an exploration candidate is expandable. A private same-package module is preferred where a subpackage would introduce cycles or public conversion types.

One private retention-policy owner handles shared eligibility, novelty, capacity decisions, and artifact-input composition. Existing Artifact publication primitives remain the durability owner. Seed orchestration keeps selection-ordinal publication; choice and simulation orchestration keep staged round transactions, hash links, frontier expansion, interruption recovery, and commit ordering. Guided corpus retention stays independent of campaign success retention.

Executor injection moves behind private runner dependencies used by internal tests and production defaults. Public campaign, replay, resume, shard-execution, and minimization requests describe user policy without signatures requiring imports from Runner's internal execution package. Do not introduce a new public executor abstraction or expose descriptor/wire internals.

Architecture tests discover all host packages, validate ownership and selected internal edges, enforce targeted host-effect restrictions on pure modules, and reject public signatures requiring inaccessible nested-internal types. Overlay sources and independent fixture modules receive explicit exclusions. Test the checker with violations, rather than only asserting today's filenames.

## API Contracts

- CLI flags, JSON schemas, exit codes, `HostError.Reason` values, error precedence, and replay verdicts remain unchanged.
- For identical supplied inputs and identities, canonical evidence, artifact manifests, journal records, and wire bytes remain unchanged. Rebuilding Runner or the runtime legitimately changes implementation identities; maintain existing replay compatibility checks and never promise replay across changed toolchains or platforms.
- Remove public `Executor` and `ReplayExecutor` injection contracts and corresponding request fields after confirming all repository consumers. Migrate in-repository test callers atomically. Preserve `Preparer`, `ArtifactReplayer`, and all other usable public contracts. Document this source-level API change explicitly; do not hide it behind forwarding aliases.
- Private assessment returns validated data or the existing typed/classified error. Strategy-specific scheduling, state transitions, and durable commit remain outside its interface.

## Edge Cases & Constraints

Success, target failure, World failure, watchdog, cancellation, incomplete/malformed recordings, trace overflow, missing required probes, and infrastructure failure remain distinct. Preserve which error wins when evidence and publication both fail. No artifact may become eligible for exact replay on partial or unvalidated evidence.

Retention checks precede publication under the existing count/byte bounds. Preserve first-novel selection, failure deduplication, independent corpus quotas, and publish-before-journal/index ordering. Artifact deduplication must never prune a distinct forced prefix. An interrupted exploration round is rerun according to current recovery rules; a seed campaign resumes according to committed ordinals.

Increasing campaign size tenfold must not introduce new state proportional to total selected seeds or new full-payload copies in the extracted policy. Validate the preserved control bounds with deterministic fake-executor campaigns of 10 and 100 jobs at parallelism 2, identical bounded payloads and fixed retention limits: maximum active executions stays at most 2, and count/byte exhaustion preserves its existing classification. Review the extracted data flow for new total-seed-sized collections and payload copies. This maintenance spec makes no wall-time, RSS, throughput or allocation-count performance claim; it does not add a benchmark project.

Early runtime bootstrap processing must remain allocation-free and must not import host or ordinary standard-library facilities unavailable during runtime startup. Disabled execution and direct seed activation stay unchanged. Unsupported platforms stay unsupported. Preserve existing code comments and avoid third-party dependencies.

## Scope cut (2026-09-29)
<!-- scope: both -->

Only R1 (the generated runtime bootstrap consumer) remains in this spec. R2–R6 moved to
`fn-105-gomad-follow-ups-deferred-scope` as D1–D5: they are maintenance with no behavior change
and no consumer waiting on them. R1's task now also carries the full-gate qualification that R6
used to own.

## Acceptance Criteria

- **R1:** Runtime bootstrap layout, header recognition, and seed projection are generated from the existing schema and exercised through the actual runtime consumer on both qualified platforms. Valid, empty, truncated, wrong-magic/version/kind, and seed-boundary vectors preserve current early behavior; malformed checksum/identity remains rejected by the existing later phase before workload execution. Disabled and direct-seed execution remain unchanged.
- **R2:** *Moved to `fn-105` on 2026-09-29 (scope cut; see [deferred follow-ups](../../.plans/GOMAD_MILESTONES.md#f10-follow-ups-deferred-scope)).* Seed, choice, and simulation exploration use one private assessment owner for their shared World, coverage, evidence, and outcome rules. Characterization tests preserve fixed-identity output and classification for success, target/World failure, malformed or missing required World data, trace overflow, missing probes, watchdog, cancellation, and infrastructure failure, including error precedence. Strategy-only checks remain outside.
- **R3:** *Moved to `fn-105` on 2026-09-29 (scope cut; see [deferred follow-ups](../../.plans/GOMAD_MILESTONES.md#f10-follow-ups-deferred-scope)).* Shared retention decisions and artifact-input composition have one owner while strategy-specific publication transactions remain intact. Tests cover novelty, duplicate failures, count/byte exhaustion, incomplete transcripts, publication failure, out-of-order completions, cancellation, interrupted round resume, and guided-corpus independence; manifests and committed journal projections match the prior behavior for fixed identities. The 10-job/100-job control-bound cases preserve parallelism and retention limits without adding total-seed-sized policy state or full-payload copies.
- **R4:** *Moved to `fn-105` on 2026-09-29 (scope cut; see [deferred follow-ups](../../.plans/GOMAD_MILESTONES.md#f10-follow-ups-deferred-scope)).* Intended public Runner use compiles from a consumer outside the Runner subtree without referencing its internal execution types. Public executor injection is removed and repository callers migrated; usable preparation/replay interfaces and CLI behavior remain. Existing fake-executor failure tests work through private dependencies, with no new global mutable hooks. No error surface beyond the preserved execution errors in R2/R3.
- **R5:** *Moved to `fn-105` on 2026-09-29 (scope cut; see [deferred follow-ups](../../.plans/GOMAD_MILESTONES.md#f10-follow-ups-deferred-scope)).* Architecture checks enumerate every host package in both qualified platform source sets, reject ownerless packages and selected forbidden host-effect/import edges, and reject inaccessible nested-internal types in public Runner signatures. Negative fixtures demonstrate each failure; legitimate platform-specific files and explicitly excluded overlays/fixture modules remain accepted. No blanket standard-library allowlist is introduced.
- **R6:** *Moved to `fn-105` on 2026-09-29 (scope cut; see [deferred follow-ups](../../.plans/GOMAD_MILESTONES.md#f10-follow-ups-deferred-scope)).* Current architecture, platform, and determinism documentation distinguishes capability support, workload repeatability, exact replay, and expectation matching; it describes shipped choice exploration and both simulation backends without obsolete future-tense claims. Stable requirement IDs remain intact. Focused tests, generated-source checks, full Gomad gates on both qualified platforms, and project lint pass or report an explicit environment blocker; a blocker is not a passing qualification. No new runtime behavior or support claims are introduced by documentation.

## Boundaries

No generic strategy/plugin framework, runtime scheduler redesign, multi-P support, deterministic-GC implementation, new fault model, new CLI/configuration, artifact schema migration, or qualification expansion. Do not relocate the already-deep Record, Artifact, World, launch-resource, or target-preparation modules. Capability-facade decomposition is deferred until repeated policy changes justify it; size alone is not a defect. No commits are part of this planning request.

## Decision Context

Prefer focused ownership changes over package proliferation: existing architecture already provides useful deep modules. Share completion policy because three real strategies consume it; retain distinct transaction semantics because seed journals and exploration rounds differ. Generate the early bootstrap endpoint because the same protocol is already generated for other consumers. Make purity and public visibility checks enforce architectural effects, while retaining existing owner/import tests.

The tradeoff is migration and regression-testing effort now for better locality later. Private modules avoid a wider public interface. Runtime bootstrap remains allocation-free; host policy preserves existing bounded resource ownership. A quantitative performance gate is intentionally excluded because this change makes no performance claim; deterministic bound tests and data-flow review verify the scalability contract. Public executor removal is a deliberate source-level cleanup, not a wire compatibility change. Qualifying current functionality takes precedence, hence the F7 dependency.

## Early proof point

Task 2 characterizes existing completed-execution behavior before extraction and proves that shared validation can return detached data without owning strategy state. If equivalence requires a strategy flag matrix or changes failure precedence, narrow the seam before task 3. Task 1 independently proves the runtime-safe generated bootstrap endpoint.

## Requirement coverage

| Requirement | Task |
| --- | --- |
| R1 | 1 |
| R2 | 2 |
| R3 | 3 |
| R4 | 4 |
| R5 | 5 |
| R6 | 6 |
