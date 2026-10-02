# Gomad: reduce code size without removing features

**Plan date:** 2026-09-30

## Goal & Context

Reduce Gomad's maintained production code by deleting unreachable internal code
and giving repeated implementation one owner. Preserve every shipped capability,
public API, command, compatibility format, and qualified workload.

The September 30 cleanup analysis identified about 200 lines of localized
production cleanup, followed by repeated completed-execution assessment and
retained-evidence composition across seed, choice, and simulation strategies.
The [assessment](../../.turbo/technical-debt.md) and
[previous architecture assessment](../artifacts/fn-102-gomad-architecture-consolidate/architecture-assessment.md)
provide context. This spec states its own scope and acceptance criteria because
the shared assessment report can be revised.

The user's request to reduce code size while retaining all features authorizes
the assessment and retention consolidation deferred as D1/D2 in
[fn-105](fn-105-gomad-follow-ups-deferred-scope.md), originating in
[fn-102](fn-102-gomad-architecture-consolidate.md) R2/R3. At task breakdown,
reuse or transfer those existing obligations instead of creating duplicate
implementation tasks. D3's public executor migration remains outside this
spec and is selected under
[fn-109](fn-109-gomad-deepen-modules-and-tool-interfaces.md), which reuses
this spec's R6/R7 evidence. Complete and verify the shared policy extraction
before overlapping public migrations; preserve this spec's acceptance baseline.

Authoring this spec does not implement changes, start tasks, or claim new
qualification evidence.

## Architecture & Data Models

### Local cleanup

Recheck callers before deleting the following private implementation:

| Candidate | Disposition |
| --- | --- |
| `validateGoCapabilityClosure` | Remove the unused capability-validation wrapper; retain current capability review. |
| `matchesExpectation` and `firstReplay` | Remove the unused expectation matcher and its exclusively used helper; retain current supported/unsupported expectation matching. |
| `deterministicCapturedInputs` | Remove the unused conversion; retain active mount conversions. |
| `orderRunCompletions` | Remove the unused wrapper; retain shard-aware ordering. |
| `removeCompletedPartial` | Remove the unused wrapper; retain context-aware cleanup. |
| Private `decimal` and `decodeCanonicalJSON` | Remove the unused type/methods and decoder; retain the live deterministic-I/O inventory encoder. |
| Internal minimizer `Encode`/`Decode` | Remove serialization used only by a round-trip test; retain budget assertions, validation, and identity hashing. |

Remove the second canonical re-encoding/comparison where a reader immediately
follows `DecodeCanonicalJSON` with the same work. Keep every independent schema,
identity, mapping, count, mode, inventory, and capacity check.

Represent the modernc memory adapter's existing anchored edits using the
existing rewritten-module implementation. Preserve exact input/output pins,
replacement content, inventory, source-set identity, and build evidence. Keep
the adapter itself and all supported versions/platforms.

Publish upgrade dossiers through the existing host filesystem replacement
primitive. Preserve pretty-printed payload bytes, trailing newline, mode,
destination, replacement semantics, and publication after a failed gate.
The shared primitive checks cleanup and synchronizes files/directories; document
and verify its stronger error handling without changing dossier classification.
Artifact no-replace publication retains its separate owner.

### Shared execution assessment

Introduce one private assessment owner in Runner for common interpretation of
a completed execution. It consumes captured evidence and recorded limits and
returns detached validated evidence, common coverage projections, and outcome
information. It owns common World decoding/seed validation, semantic coverage
decoding, choice-feature projection, and shared outcome interpretation.

Use concrete private types/functions. Keep preparation, process launches,
filesystem input capture, progress, counters, cancellation, and journal
publication outside the computational assessment. Keep strategy-specific
choice-tape projection, candidate expansion, simulation records, and forced
prefixes with their existing owners where their contracts differ.

Migrate seed, choice, and simulation completion paths to the same common
implementation. Preserve their existing validation order, diagnostic reasons,
and failure handling. A staged assessment interface is acceptable when callers
need to preserve different effect/failure ordering; a strategy-flag matrix is
not required.

### Shared retention and artifact composition

Give common novelty predicates, capacity calculations, and artifact-input
composition one private owner. Assessment supplies validated detached evidence;
strategy callers supply explicit publication context and commit policy state
only after their current successful commit point.

Seed campaigns retain ordinal commits. Exploration campaigns retain atomic
round commits, hash-linked journals, interruption archival, and full interrupted
round reruns. Corpus admission retains mandatory publication and exact replay
before advancing its index. These durable protocols remain separate.

Extract shared computation without introducing a generic strategy framework,
new package-forwarding layer, persistent entity, or another replay controller.
An extraction must reduce net authored production code after its types and
wrappers are counted.

## API Contracts

Keep all exported Go names, signatures, request fields, default values, and
usable construction patterns of public packages unchanged. Only the enumerated
unused internal implementation is removable. Public `Executor`/`ReplayExecutor`
interfaces and injection fields remain available despite their existing
visibility limitations.

Keep CLI commands, flags, argument boundaries, presence-sensitive validation,
stdout/stderr routing, JSON event/report schemas, classifications, and exit
statuses unchanged. This includes `checked-run`, `exec --provenance`,
portable campaign plans/shards/merge, guidance, inspection, recovery/resume,
both exploration strategies, and simulation minimization.

Keep compatibility packs and adapters, choice-v1 inspection, exact choice-v2
replay, artifact/record/plan/journal formats, capacity identities, and
qualification manifests supported. Canonical bytes and semantic identities
must match when supplied inputs/identities are held fixed. Rebuilding Runner
legitimately changes its build identity; this spec does not promise replay of
artifacts bound to another Runner build.

## Edge Cases & Constraints

- Preserve existing comments when removing surrounding obsolete code or moving
  live code. Preserve meaningful comments with their owning logic. Code-size
  accounting must not reward comment deletion or compressed formatting.
- Preserve fresh processes, deterministic I/O, capability governance, native
  timer ownership, independent replay identities, and replay validation before
  activation or model mutation.
- The inventory encoder and shared canonical-JSON encoder order keys
  differently. Replacing the live encoder is outside this cleanup.
- Preserve malformed, incomplete, overflowing, cancelled, watchdog, unsupported,
  and divergent results, including existing failure precedence.
- Preserve success/failure retention limits, byte accounting, novelty ordering,
  deduplication, and deterministic publication order under out-of-order host
  completion. Failed publication or interrupted rounds cannot advance common
  novelty state or silently drop evidence.
- Keep World, simulation backends, network/volume/fault models, guided corpus,
  target provenance, caches, and both sibling packages. Their implementation
  may shrink only through behavior-preserving reuse.
- Follow the [milestone constraints](../../MILESTONES.md#constraints).
  Add no dependencies. Use no worktrees. Leave commits to the user.
- Existing Linux/Darwin replay findings remain separately owned. Preserve their
  current dispositions and baseline evidence; do not relax a gate or suppress
  a failure to obtain a passing refactor.

## Acceptance Criteria

- **R1:** Final authored production Go line count is lower than the recorded
  implementation baseline across Gomad and both sibling packages, with new
  helpers, types, and wrappers included. Report production/test/generated
  counts separately, using the same tracked-file inventory and counting rule.
  Runtime patch/schema/template changes are also reported so moving code into
  generated inputs cannot create a false reduction. Comments, formatting
  compression, feature deletion, and movement across directories do not count
  as simplification. Errors: inability to establish comparable counts or a net
  production increase leaves this requirement unmet.

- **R2:** The local unused-code candidates above are deleted only after
  repository-wide checks of source, tests, build tags, generators, templates,
  and specifications establish no supported consumer. Test-only minimizer
  serialization is removed while its live budget/state assertions remain.
  Current capability, expectation, mount, ordering, and cleanup behavior stays
  intact. Errors: a newly discovered live consumer keeps its implementation;
  report the correction rather than remove the consumer.

- **R3:** Portable-plan and merged-campaign readers perform canonical byte
  validation once through the existing decoder and retain every independent
  validation. Valid fixtures retain the same decoded values. Errors:
  noncanonical/malformed/trailing/unknown-field input and invalid protocol
  identities remain rejected with the same public classification.

- **R4:** Memory adapter preparation uses the existing rewrite owner and
  produces identical replacement bytes, inventory/source-set identities, and
  evidence for each supported platform. Errors: changed versions/sums,
  source drift, missing/duplicate anchors, invalid source files, and changed
  replacement inventories remain preparation failures.

- **R5:** Upgrade dossier publication uses the existing replacement owner and
  retains payload formatting, mode, path, replacement, and failed-gate report
  behavior. Errors: write/close/rename/cleanup/sync failures remain visible
  infrastructure errors; partial output cannot replace the prior complete
  dossier. Document the shared primitive's stronger cleanup/sync reporting.

- **R6:** Seed, choice, and simulation completion paths use one private owner
  for common evidence assessment. For fixed supplied identities, equivalent
  captured evidence produces the same detached projections, classifications,
  and canonical record inputs as before extraction. Errors: malformed World,
  seed mismatch, malformed coverage/choices, missing terminal evidence,
  watchdog, cancellation, and multiple simultaneous faults preserve existing
  precedence and strategy-specific publication/recovery behavior.

- **R7:** Common retention decisions and artifact-input composition are shared
  without merging durable strategy transactions. Preserve all/novel/discard
  success modes, failure deduplication, guided corpus admission, and exact
  replay evidence. Errors: count/byte exhaustion fails visibly; publication
  failure, cancellation, or interrupted round commits leave novelty and
  counters at the existing committed state. Resume/recover retain their
  behavior and deterministic ordering.

- **R8:** Every shipped feature, public Go/CLI surface, compatibility
  version/pack, qualified workload, and replay protocol remains available.
  Existing comments and behavioral assertions are preserved except assertions
  exercising demonstrably unused internal serialization. Errors: an API,
  schema, feature, default, failure classification, or compatibility removal
  fails acceptance even if line counts improve.

- **R9:** Focused tests and required final checks pass against the same
  baseline dispositions. Exercise ordinary seed, guided, choice-exploration,
  simulation-exploration, retention, interruption/resume, and minimization
  paths with fixed-input regression evidence. Run generated-source validation,
  architecture checks, full Gomad gates on both supported platforms, Temporal
  integration/smoke, and affected qualification suites. Errors: missing-host
  validation, unexplained baseline regression, changed canonical projections,
  or weakened expectations is recorded as incomplete acceptance.

## Boundaries

- Feature retirement, package deletion, compatibility retirement, and public
  executor API changes are excluded.
- Runtime scheduler/GC/time redesign, overlay/protocol changes, dependency
  upgrades, and new capabilities are excluded.
- Broad target-preparation/options redesign and unrelated bug fixes are
  excluded. Record discovered defects with their existing owner.
- Removing architecture assertions or meaningful regression coverage to
  reduce test size is excluded.
- This spec does not implement or break work into tasks. Task breakdown must
  reuse or transfer fn-105 D1/D2 obligations.

## Decision Context

The user explicitly retained every feature, so the larger deletion candidates
in the analysis do not belong in this work. The first proof point is local
cleanup with existing focused tests; its approximate 200-line estimate is
guidance rather than a quota. Then migrate common assessment and retention one
strategy at a time, comparing fixed-input projections and failure order.

Existing rewrite, host filesystem, Record, Artifact, World, and frontier
modules already hide substantial behavior behind small interfaces. Reusing
them and extracting narrow private computation removes duplicate knowledge.
Moving folders, adding generic strategy abstractions, or changing exported
request shapes would increase migration cost without satisfying the requested
preservation contract.

Use focused tests for each change, then the final gates once. Characterization
tests cover observable equivalence and failure precedence rather than pinning
helper names or duplicating the implementation. Use `test_dep` for tests and
`require` with whole-value/proto equality where appropriate. Artifact tests
use fixed supplied identities; actual rebuilt tool identities are reported
separately. Keep implementation evidence with the Flow spec, including the
baseline revision, size accounting, commands, platform, and remaining failures.
