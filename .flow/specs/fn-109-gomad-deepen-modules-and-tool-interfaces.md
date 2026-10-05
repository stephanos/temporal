# Gomad deep modules and tool interfaces

**Plan date:** 2026-09-30

## Goal & Context

Make Gomad easier to extend and use by placing campaign intent, execution
assessment, retention, target preparation, process progress and resource lifetime
behind small, testable interfaces. Address every finding from the September 30
architecture analysis while preserving shipped capabilities and replay contracts.

The user asked to write a new Flow spec addressing all findings. This scope
includes the eleven ranked findings and all five secondary design opportunities.
The earlier suggestions to defer simulation and handle restructuring are now
selected work. Comparing alternative interfaces is part of delivery; a design
note or renewed deferral alone cannot fulfill an implementation requirement.

The [architecture assessment](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/architecture-assessment.md) supplies source
context, including a source-confirmed coordinator omission. This spec states
its own contracts because the assessment can be revised and source locations
will change during refactoring. Re-anchor findings before task breakdown and
reuse verified work that lands after the assessment; already-correct code or
documentation needs evidence, not a second implementation. No runtime
reproduction or new qualification was performed while authoring the assessment.

### Relationship to existing work

- [fn-108](fn-108-gomad-reduce-code-size-without-removing.md) owns common
  assessment and retention implementation, fulfilling R2/R3 here through its
  R6/R7. Reuse or transfer fn-105 D1/D2 obligations once at task breakdown;
  retain fn-108's separate code-size and preservation requirements.
- [fn-105](fn-105-gomad-follow-ups-deferred-scope.md) D3/D4/D5 are revived by
  this request and fulfilled through R5/R8/R9 here. Reuse or transfer their
  existing tasks; do not create parallel implementation owners.
- Public Go interface changes selected here remain outside fn-108. Complete
  and verify its shared policy extraction before migrating overlapping public
  interfaces, preserving its historical acceptance baseline.
- [fn-107](fn-107-gomad-finish-downstream-cell.md) retains downstream service
  integration and qualification ownership. Preparation changes must preserve
  its external-module path and qualified configuration.
- fn-105 D12/D14 retain Linux and Darwin replay-divergence ownership. This spec
  neither fixes those channels by assertion nor changes their qualification
  dispositions. Affected final verification must identify failures attributable
  to those owners and retain actual evidence.

Authoring creates a spec only. Task breakdown must resolve ownership and real
ordering before implementation; it does not start tasks or create commits.

## Architecture & Data Models

### Campaign intent and application construction

Use one serializable campaign-options owner across local and isolated execution.
Group target intent, search settings, resource limits, observation and retention
by their invariants. Keep private dependencies, callbacks, resolved child
commands and host construction outside serialized intent. Keep immutable replay
records and campaign plans as separate versioned contracts.

Local and isolated execution receive the same normalized semantic settings.
The CLI retains flag-presence validation and presentation. Shared semantic
normalization belongs to the operation owner. Plan and explore share parsing
directly, and each invokes its own operation. Resolve installation, Runner
identity and private modes through one application construction path.

### Assessment and retention

Use fn-108's private common assessment for World validation, common coverage
and choice-feature projection, and outcome interpretation. Keep filesystem
verification/capture, cancellation, counters and journal transitions in their
current effectful owners. Preserve staged interpretation where different
strategies require different error/effect order.

Use one common retention decision and artifact-input composition owner.
Assessment supplies detached validated evidence. Strategy transactions supply
publication context and advance novelty/budget state only at the existing
successful durable commit point. Seed ordinal publication, atomic exploration
rounds and replay-verified corpus admission retain separate transactions.

### Preparation and host commands

Compose complete preparation above target and deterministic-I/O adapter
implementation, respecting the existing directional dependencies. Offer complete
target preparation and capability inspection as separate operations. Hide
adapter selection, replacement workspaces, generated module/overlay inputs,
metadata assembly and validation ordering from ordinary callers. Return complete
validated target or inspection evidence; callers do not attach adapter identity
after preparation.

Preparation owns implementation-only workspace lifetime. Campaign and portable
bundle owners retain their durable destinations and lifecycle journal updates.
Existing build caches, provenance and exact compatibility policy remain
implementation owners. Qualification repetitions still prepare independently.
Closure inspection does not compile; linked inspection compiles without
launching the target; guarded execution preserves its existing policy.

A private Go-command adapter uses existing host process primitives for context,
time bounds, process-group cleanup and bounded diagnostics. Structured listing
requires complete bounded output and rejects overflow. Diagnostic capture may
retain bounded head/tail with full hashes. Share command mechanics without
merging their different output/error contracts.

A validated installation description owns toolchain identity and location
knowledge. Ordinary consumers use that value instead of re-deriving build,
cache and adapter paths. Preserve stable replacement locations recorded in
binary build information.

### Simulation progress, protocols and handles

Generate host and allocation-free runtime simulation-time projections from one
versioned definition. Keep descriptor I/O, native timers and quiescence hooks
in runtime. Preserve bytes, reserved fields, correlation and monotonicity.
Completed bootstrap generation remains in place.

Choose an operation-lifecycle interface for simulation progress after comparing
at least two designs against current ordering guarantees. The selected module
owns admission, forwarding, delivered-but-unconsumed work, abandonment and
participant removal. Callers no longer sequence several accounting updates or
maintain a parallel response-barrier state machine. Blocking transport,
process lifetime and domain mutation retain their separate owners. Preserve
concurrent blocking operations and avoid using host IPC arrival order as replay
identity.

Select backend-specific listener, connection, filesystem-handle and mapping
implementations at creation. Each implementation owns its valid state shape,
while callers use ordinary operations. Preserve standalone, in-process and
process behavior, local-model lock ownership, stale-incarnation rejection,
deadlines and mapping capabilities. Share domain models without duplicating
semantics or introducing a generic backend registry.

Typed domain commands own model-operation arguments and response interpretation.
One translation owner maps each command to the existing compact envelope,
hiding generic string/integer slot meanings. Wire framing and domain semantics
retain their own owners. Accepted inputs and canonical bytes remain unchanged;
stronger shape rejection requires a separately specified contract change.

### Resource lifetime and pure transitions

Separate detached Artifact references from owned opened handles. Opened handles
own the pinned directory and private validated manifest; detached snapshots do
not carry live resources or mutable aliases into the handle. Payload access
continues to validate inventory, mode, size and hashes.

Give the seed controller one completion transition that updates attempt and
classified outcome together, with distinct-failure information explicit. Keep
scheduling, failure policy and counters pure.

Separate capability evidence collection, pure policy evaluation and linked
projection internally behind the same usable review interface. Reuse existing
compatibility-pack policy. Give adapter source-inventory hashing one neutral
private owner consumed by target and adapter preparation. Preserve exact
first-party simulation pins and allowed bridge directives.

## API Contracts

Preserve CLI commands, flags, argv boundaries, defaults, explicit-flag rejection,
stdout/stderr routing, JSON schemas, classifications and exit statuses. The
coordinator correction makes an already valid simulation-exploration request
reach execution with its supplied bounds; it does not add a new mode.

Keep artifact, record, choice, transcript, campaign-plan, journal, provenance,
qualification and compact model-wire formats compatible. For fixed supplied
identities, canonical projections and payload bytes match the prior behavior.
Actual rebuilt Runner/toolchain identities legitimately change. Replay across
changed tool or platform identities is not promised.

This spec deliberately permits migration of unusable public executor injection
into private dependencies and separation of Artifact reference/open-handle
types. Inventory consumers and record the exact public Go changes, replacement
construction and caller migrations before implementing them. Preserve usable
preparation/replay substitution and all other supported public capabilities.
Ordinary consumers can construct requests and invoke operations without naming
nested internal execution types or building descriptor layouts. Add a public
tool constructor only if the consumer inventory establishes a need; otherwise
centralize private application construction.

R8's actual-source checks also admit the corrective migrations inventoried in
`go-interface-changes.md`: detached public Runner capacity and target pack-evidence
graphs replace retained private identities, and public pack-directory intent
replaces pinimpact's inaccessible loader callback while preserving its authoring
override. Preserve complete reports, canonical bytes, nil/empty distinctions,
field order and validation/load precedence.

World terminal reporting reconciles a real purity conflict rather than exempting
arbitrary callbacks. Add a detached Recorder.FinishTerminal(Terminal) boundary;
retain FinishError(error) for original World sentinels and owned typed/classified
errors without arbitrary method dispatch. The effectful process Session reporting
seam preserves general Error-before-Is normalization and established validation,
classification and cleanup ordering. Direct recorder custom/external-wrapper
inputs migrate to caller-side projection or process reporting; their former
callback-derived acceptance/messages are deliberately not preserved inside the
pure model. Immutable internal sentinel identities/messages replace behavior
dependent on rebinding exported sentinel variables. Preserve original default
sentinels, typed cause relationships, all identified callers and known-error
record/snapshot/digest bytes. This bounded intentional behavior/API migration is
recorded before implementation in `task-19/world-terminal-design-decision.md`
and the interface inventory; it permits no other World semantic or policy change.

Return existing typed/classified failures at established seams. Preserve error
precedence, including simultaneous evidence errors, watchdog/cancellation,
cleanup failure and publication failure. A wrapper that only forwards another
interface does not fulfill a depth requirement.

## Edge Cases & Constraints

- Preserve existing comments with their owning code. Simplification cannot
  depend on comment removal, compressed formatting or lost regression coverage.
- Preserve fresh execution processes, single-P assumptions, native timer
  ownership, World purity, detached model state, separate evidence identities,
  explicit capacities, controlled environments and exact compatibility pins.
- Validate replay before activation and model mutation; require complete tape
  consumption and preserve terminal state, output and outcome checks.
- Keep seeded scheduling separate from host deadlines and output timing.
  Preserve committed versus uncommitted model-operation outcomes during death,
  cancellation, late responses and restart.
- Preserve retained-success count/byte accounting, failure deduplication,
  novelty order, interrupted-round archival, resume and corpus single-writer
  publication. A failed transaction cannot claim uncommitted coverage.
- Keep strict structured-output overflow distinct from diagnostic truncation.
  Missing installation, invalid packs/sums, changed binary/source/cache inputs
  and cleanup failures remain visible failures.
- Keep writable mapping behavior and backend fidelity explicit. In-process
  restart cannot claim fresh globals or hard cleanup supplied by process nodes.
- Generated runtime code preserves allocation, stack, dependency and nosplit
  constraints. Register changed schema/template/generated sources in existing
  overlay and version inventories; validation must detect drift.
- Cover both qualified platform source sets. A host effect in a pure module
  must be caught even when imported from the standard library or a dependency.
  Mixed effectful/pure packages need a targeted rule, not a blanket prohibition.
- Follow the milestone constraints. Add no third-party dependencies, use no
  worktrees, leave commits to the user and retain native/default behavior.
- At ten times campaign work, keep policy state bounded by declared frontier,
  parallelism, retention and evidence limits. Avoid total-selection-sized state
  or extra complete-payload copies introduced solely by extraction.

## Acceptance Criteria

- **R1:** Local and isolated campaigns use one normalized options owner.
  `MaxForcedDecisions`, `MaxExplorationResultBytes` and every
  `SimulationDimensionLimits` field survive coordinator transport. Tests
  exercise seed, choice and simulation strategies through actual isolated
  execution with distinguishable nonzero limits. Errors: invalid/missing bounds,
  incompatible strategy settings, malformed requests and unknown fields remain
  rejected; local fake-executor tests alone do not fulfill transport coverage.
  [paraphrase]
- **R2:** Seed, choice and simulation completion use fn-108's shared private
  computational assessment. Fixed-input projections and classifications match
  the pre-extraction behavior. Errors: malformed/missing World evidence, wrong
  seed, coverage/choice errors, watchdog, cancellation and simultaneous faults
  preserve precedence and strategy-specific effects. Fulfilled through fn-108
  R6 with linked evidence. [paraphrase]
- **R3:** Shared retention and artifact composition hide novelty, capacities
  and payload assembly while preserving seed, round and corpus transactions.
  Errors: duplicate failures, incomplete transcripts, count/byte exhaustion,
  publication failure, cancelled/interrupted commits and failed corpus replay
  preserve bounded evidence and committed policy state. Fulfilled through
  fn-108 R7 with linked evidence. [paraphrase]
- **R4:** Explore, portable planning, analysis and compatibility review use one
  complete preparation owner returning validated target/review evidence with
  adapter identities attached. Workspace cleanup is owned explicitly. Tests
  cover fresh/cache builds, external modules/local replacements, custom
  preparers and independent qualification preparations. Errors: invalid sums,
  replacement conflicts, unsupported closure, malformed linked records,
  changed binaries and cleanup failures retain classifications. Closure
  inspection does not compile or execute. [paraphrase]
- **R5:** Public executor injection moves behind private dependencies, with all
  repository consumers migrated and supported external use compiled from
  outside the Runner subtree. Keep usable preparation/replay seams and fake
  failure coverage through private adapters. Errors: inaccessible internal
  types, global mutable test hooks and exposed descriptor mechanics fail
  acceptance; execution failures retain existing semantics. Fulfills D3.
  [paraphrase]
- **R6:** CLI operation construction resolves installation/private modes once
  and shares semantic normalization with Runner. Plan and explore share parsing
  directly. Tests cover documented command grammar, explicit zero/irrelevant
  flags, environment/tags/argv, text/JSON output and exit statuses. Errors:
  malformed input and output-writer failures retain existing behavior; a
  changed default or hidden plan-only argument route fails acceptance.
  [paraphrase]
- **R7:** Simulation-time layout and codecs have one generated definition with
  host and runtime-safe consumers. Cross-consumer vectors preserve bytes and
  exercise actual runtime consumption on both platforms. Errors: truncated
  frames, wrong magic/kind, nonzero reserved bytes, generation mismatch and
  time regression fail as before. Generated-output drift fails validation.
  [paraphrase]
- **R8:** Architecture checks discover every host package on both qualified
  source sets and enforce targeted ownership, host-effect and public-signature
  visibility rules. Negative fixtures demonstrate rejection of an ownerless
  new root, forbidden effect/import edge and inaccessible public type. Valid
  platform files and explicit overlay/fixture exclusions remain accepted.
  Errors: silently uninspected packages and checks based only on filename
  presence fail acceptance. Fulfills D4. [paraphrase]
- **R9:** Current architectural guidance describes both supported platforms,
  implemented choice replay/exploration and both backends, using existing
  requirement IDs. Separate capability support, repeatability, exact replay
  and expectation matching. Document current residual findings and intentional
  Go interface changes. Errors: unmeasured support or performance claims,
  obsolete delivery-state claims and failures described as qualification
  success fail acceptance. Fulfills D5. [paraphrase]
- **R10:** Target compilation, Go identity queries and listing use a coherent
  private host-command seam with contextual lifetime and output limits. Tests
  cover command cancellation/termination, long diagnostics and cache-lock
  release. Errors: structured-output overflow, malformed listing, unsupported
  capability, timeout and cleanup failure stay distinct; truncated structured
  data cannot be accepted. [paraphrase]
- **R11:** Compare at least two simulation-progress interfaces and implement
  the chosen lifecycle owner, removing caller-owned multi-counter sequencing
  and duplicate response-barrier bookkeeping. Tests cover forwarding,
  acknowledged arrivals, delivered-but-unconsumed work, arrival at quiescence,
  cancellation with late committed response, death and restart. Errors:
  unknown acknowledgement, abandoned response and stale incarnation fail
  before invalid progress or model mutation; concurrent blocking operations
  remain possible. Design evidence alone leaves this requirement unmet.
  [paraphrase]
- **R12:** Network and filesystem factories select backend-specific handles
  once, with private implementations owning valid local/process state.
  Shared operation tests cover standalone and both simulation backends;
  backend-specific tests retain hard-isolation and mapping distinctions.
  Errors: duplicate bind, deadline, close/reset, partial I/O, capacity, stale
  incarnation and replay divergence preserve behavior and validation-before-
  mutation. Moving dispatch into another forwarding helper alone does not
  fulfill this requirement. [paraphrase]
- **R13:** Detached Artifact references and owned opened handles have distinct
  types and lifetime contracts. Opened manifest state is private and snapshots
  cannot mutate it. Tests cover published/detached references, open/close,
  directory replacement and valid payload access. Errors: use after close,
  unlisted payload, wrong mode/size/hash and symlink/path substitution remain
  rejected; detached values never transfer live resource ownership. [paraphrase]
- **R14:** Typed network and volume commands own argument/response semantics;
  one translation owner hides generic model-wire field slots. Fixed vectors
  retain existing bytes and supported operation results. Errors: partial I/O
  with errors, invalid handles, capacities and unavailable backend operations
  preserve domain information. Previously accepted wire shapes are not
  silently rejected by this refactor. [paraphrase]
- **R15:** A validated installation description supplies pinned identity and
  all owned build/cache/adapter locations to consumers. Tests cover each
  resolution source and retained stable replacement locations. Errors:
  malformed manifests, missing/stale builds, invalid roots and identity
  mismatches fail closed with existing repair guidance. Re-deriving the tree
  in ordinary consumers or changing path-stamped identity fails acceptance.
  [paraphrase]
- **R16:** One pure seed-controller completion transition updates attempted,
  active and classified counters with failure-policy stopping atomically.
  Tests cover success, cancellation, watchdog, distinct/duplicate failure,
  resume counters and every failure policy. Errors: completion without active
  work is rejected as an invariant violation; partial counter updates and
  changed ordinal scheduling fail acceptance. [paraphrase]
- **R17:** Capability collection, pure evaluation and linked projection have
  separate private ownership behind the existing review contract, reusing
  compatibility policy. Adapter source-inventory hashing has one neutral
  owner consumed by target and adapter preparation. Tests preserve canonical
  ordered findings, live/eliminated blockers and inventories. Errors: source
  drift, invalid overlays/replacements, unsafe bridge directives, capacity
  exhaustion and malformed linked evidence remain fail-closed. [paraphrase]
- **R18:** Every shipped capability, recorded format, ordinary CLI behavior and
  qualified workload remains available. Preserve existing comments, fixed-
  identity canonical bytes, native defaults, independent replay identities,
  error precedence and transaction guarantees. Inventory and migrate only the
  intentional Go interface changes described above. Errors: feature removal,
  accidental public breakage, weakened assertions/expectations or a new generic
  host-I/O grant fails acceptance. [inferred]
- **R19:** Retain baseline revision/inputs, interface decisions, consumer
  migrations, commands and platform-specific results. Focused tests, generator
  validation, architecture checks, complete Gomad gates on both platforms,
  native/default integration, functional smoke and affected qualification
  suites pass against unchanged dispositions. Exercise 10-job/100-job bounded
  control cases without new selection-sized policy storage or full-payload
  copies. Errors: unavailable required hosts, unexplained regressions or
  unmeasured gains leave the corresponding acceptance incomplete; record
  separately owned D12/D14 evidence without weakening a gate. [inferred]
- **R20:** A completion matrix maps every F1-F11 and secondary S1-S5 opportunity
  to its R-ID and implementation/verification evidence. Link shared fulfillment
  from fn-108 and transferred/reused D1-D5 obligations exactly once. Errors:
  duplicate task owners, unmapped findings or optional items closed only by
  renewed deferral leave this spec incomplete. No additional runtime error
  surface beyond R1-R19. [inferred]

## Boundaries

- New features, dependency upgrades, platforms, multi-P execution, runtime
  scheduler/GC redesign and native timer replacement are outside this spec.
- World, network, persistence, fault and corpus semantics retain their existing
  owners; a universal scheduler or generic strategy/plugin framework is excluded.
- Artifact/schema migration, source translation, rewriting existing test
  assertions and changing production Temporal behavior are excluded.
- Downstream service integration, new clock policies, dynamic Linux clock audit,
  trace-capacity extensions and unresolved runtime divergence stay with their
  existing specs. Preserve their contracts during refactoring.
- This request authors a spec. Task creation/start, implementation, commits,
  pushes and deployment are separate operations.

## Decision Context

A deep module removes caller knowledge about order, state and validation.
Moving files or adding a constructor around the same obligations does not
satisfy the goal. Use concrete private functions for pure computation and
interfaces only for actual backend/host-test variation. Retain independent
physical execution, logical policy and durable publication.

Preparation composition must sit above target and adapter implementation to
avoid a cycle. Capability extraction and source inventories remain exact and
bounded. Stable adapter replacement paths enter binary build information, so
path centralization must preserve locations and identities.

Simulation progress and handle changes carry more risk than shared Runner
policy. Compare alternatives and migrate one handle family at a time, with
state-machine and process conformance evidence. Preserve concurrent blocking
work and late-response accounting. Native/default and backend-specific tests
remain necessary because detached model agreement cannot prove hard isolation.

At task breakdown, schedule transport correction and characterization first.
Reuse fn-108 for assessment/retention. Sequence options/private construction,
preparation/commands/installations and capability ownership by their actual
shared files and outputs. Public migrations overlapping fn-108 follow its
completed extraction. Then migrate Artifact lifetime, generated/typed protocols,
simulation progress and backend handles. Integrate architectural checks and
current documentation with the changed owners, then run final qualification.
Parallel tests that mutate shared toolchain/cache/qualification resources must
be serialized or use independent resources without worktrees.

Task 21's retained lint failure has a separate corrective implementation owner,
task 23. It repairs root-versus-nested module routing and supplies reproducible
ordinary host-lint gates under the existing R19 qualification obligation.
Evidence, compiler-negative fixtures and runtime overlays retain their explicit
owners; ordinary nested source cannot be silently exempted. The original
criteria, lint rules and native qualification requirements remain unchanged.

Task 23's source-progress checkpoint is ce80d2425cf34da103939b5aa23f90bde1c2092f.
Actual lint remains red. Task 24 separately owns the demonstrated inherited
configuration path-base defect: restore the existing repository-relative
exclusion intent with behavioral positive/negative tests, without adding
suppressions or changing enabled rules, pins or comparisons. Task 24's pre-edit
parsed-regex controls and byte inspection refute the earlier escaping
hypothesis; the working expressions remain unchanged.
Its reviewed one-line gitroot repair and actual-tool controls establish source
progress only. Root fast still fails on the controller lifecycle switch and
ordinary Gomad on 419 inventoried findings; neither receives a suppression or
acceptance waiver. Task 21 remains the final verification consumer and original
R18/R19, formal review and both native gates stay open. Task 24's handover and
independent review remain source-bound pre-commit snapshots.

Task 25's independently reviewed source progress repairs root fast's exhaustive
lifecycle finding with a two-branch distinction inside the unchanged outer
lifecycle case. Literal before/after characterization preserves all 36
state/operation cases, targets, incarnation, identity and outer controls; actual
unfiltered package lint reproduces the original defect and passes after repair.
Root fast now completes ordinary root and tagged integration lint/vet before its
automatic nested scope fails on the exact retained 419 findings across 31 owners.
Nested vet and later scopes remain unreached. Task 25 stays blocked on original
qualification, not source correctness; task 21 depends on the correction. Its
source-progress review is not formal SHIP and changes no original R18/R19,
workload/default, reporting or native acceptance requirement. Future campaign
repairs must preserve the characterized completion invariant panics unless a
separate completion-error redesign is admitted.

Task 26's independently reviewed source progress restores Runner semantic
ownership in actual CLI callers at the original validation points. Typed error
translation, enabled-zero/presence checks, first-error presentation and writer
routing remain characterized. Current and saved base CLI pass the same 33
behavioral tests; the meaningful ownership regression adds the 34th final test.
Runner production and the eight existing public seams remain unchanged.
Architecture and both external-consumer compilation checks pass. Actual affected
CLI lint remains red on the same 54 findings with zero introduced/resolved;
complete CLI and expanded portable-plan tests retain unsupported-host/missing
patched-toolchain failures, and Darwin identity proof remains skipped.
Task 26 stays blocked on qualification. Task 21 consumes this source correction;
original task 5/predecessors, R6/R18/R19, full/native/formal and fixed-identity
requirements remain unchanged and open. Its handover and review are immutable
pre-commit source snapshots. Commit verified progress before another source task.

Task 27 retains independently source-reviewed documentary progress for Choice
Trace v2 refusal, controller-v2 journal refusal and retirement of the previously
selected v041 fixture. Current derived guidance and rootfast wording reflect the
source-bound migrations, repaired routing and red419 qualification. All 25 authored
links, five fragments, 49 protected files and three protected sections pass fresh
checks against the worker freeze and BASE. Historical reports, first-task
baseline inputs and original API/acceptance/boundaries remain unchanged.
Disclosure does not restore format/workload availability, prove fixed-identity
equivalence or close original R18/R19/task21/full/native/formal qualification.
Task 27 stays blocked on qualification; task 21 consumes the reviewed supplement.
Root commits verified progress before the next writer.

Task 28's independently reviewed campaign source correction checks thirteen
cleanup returns and replaces two policy switches while preserving invariant
panics, first-only cancellation, budget drain, original canonical/error vectors
and cleanup lifetimes. Conditional root cleanup preserves nil-close error
identity and joins a nonnil close error after the operation error. Actual
unfiltered package lint changes from 17 to 2 unchanged invariant findings:
15 mapped findings resolved, none introduced. Whole ordinary package, focused,
boundary, errortype and generator checks pass on developmental linux/arm64;
fresh independent review finds no actionable introduced defect.
Task 28 stays blocked on qualification; task 21 consumes this owner's evidence.
No real nonnil-root-close execution is demonstrated. The broader 419-finding
receipt remains historical; package progress supplies no new whole-scope count.
Original R16/R18/R19, task 3/predecessors, fixed-identity, full/native/formal
requirements remain unchanged. Root commits verified progress before next writer.

Task 29's independently reviewed private artifact payload repair checks seven
copyPayload/writePayload cleanup returns without changing primary error identity
when cleanup succeeds, output-before-input release or successful Sync/Close/
metadata ordering. Ownership retires before explicit Close attempts; the old
second input Close no longer risks successful publication. Ordinary package,
focused real-file, boundary, errortype and generator checks pass on developmental
linux/arm64. Actual unfiltered artifact lint falls from 28 to 21 residual findings,
with exactly seven mapped repairs and no introduced diagnostics. No first-Close
fault execution or new whole-Gomad count is supplied. Public CopyPayload, directory
sync and shared verification retain separate owners. Task 29 stays blocked;
task 21 consumes its source evidence. Original R13/R18/R19, task 12/predecessors,
matched first-baseline fixed identities and complete/full/formal/both-native
acceptance remain open. Root commits reviewed source progress before the next writer.

Task 30's independently reviewed public CopyPayload repair checks six cleanup
sites and five original test-handle cleanup sites. Payload descriptors close once,
destination before source. Nil Close preserves exact primary errors, and a sole
cleanup failure retains its raw error. Source validation, exclusive destination
creation, pinned source/private manifest, modes and copy/hash/count/EOF/Sync
retain their original operation body. Real-file controls pass against baseline
and final source; nil-wrap and second-close mutants fail for their intended reasons.
Ordinary package, focused, boundary, errortype and generator checks pass on
developmental linux/arm64. Actual unfiltered artifact lint falls from 21 to 10,
with eleven mapped repairs and no introduced diagnostics. Genuine first-Close
and simultaneous cleanup-failure execution remain unproved. Directory sync,
shared verification, reflection and publication/pool retain separate owners.
Task 30 stays blocked; task 21 consumes its source evidence. Original R13/R18/R19,
task 12/predecessors, matched first-baseline fixed identities and complete/full/
formal/both-native qualification remain open. Root commits reviewed progress
before the next source writer.

Task 31's directory/verifier cleanup candidate checks three production and three
test-handle cleanup returns at their original lifetimes. Directory operation/
post-context precedence remains primary and the inner Close precedes the post
check. Verifier cleanup releases file before root and clears metadata on a
genuine cleanup failure. Original helper operations, assertions, transaction
owners and public/private payload APIs retain their bytes. Developmental
package, focused, boundaries, errortype and generator checks pass. Actual
unfiltered artifact lint falls from 10 to four unchanged findings, with six
repairs and none introduced. Fresh independent source review permits only a
source-progress commit. Genuine first-Close and post-Sync cancellation timing
proof remain unexecuted. Original R13/R18/R19, task 12/predecessors, task 21,
matched first-baseline fixed identities and complete/full/formal/both-native
qualification remain open. Root commits progress before another source writer.

Task 32's independently reviewed error-provenance candidate preserves named-slice
single-error Unwrap traversal, destination concrete conversion callbacks and fmt
writer-error results. Six fresh allocation alias families retain shared existing
storage; conversion-local array/struct copies isolate value cells while nested
references retain their callback path. All 39 earlier fixture bodies remain an
exact prefix of the final 67 stock-host fixtures and 134 supported-metadata
observations. Whole architecture, focused, five actual boundaries, broader
purity/edges/host-vet, gomadtool consumer, errortype, source-scoped static and generator checks
pass on developmental linux/arm64. Actual unfiltered architecture lint retains
four byte-identical inherited findings, with none introduced or resolved.
The full staged diff-check retains one historical archived handover EOF blank;
source and newly written root-document checks are separately clean.
Fresh corrective review and root read-only audit permit SOURCE_PROGRESS_COMMIT_ONLY,
with no actionable introduced defects. Historical failures and inconclusive
probes remain disclosed; this is no universal copy, length/capacity or formatting
completeness claim. Task 21 consumes the evidence. Original R8/R18/R19,
task 19/fn-105 D4, predecessors, matched first-baseline identities and complete/
full/formal/both-native/affected-consumer qualification remain open. Root commits
reviewed progress before another source writer.

Task 33's independently reviewed reflection-helper repair explicitly handles
all 27 Kind values in both test helpers while preserving their existing branches,
initialized values and every original assertion. Arrays recurse through indexed
elements; invalid/unsupported shapes reject without skipping typed nils. Three
controls characterize supported array/reference isolation, scalar and nil/empty
preservation, and eight unchanged production-guard cases. The controls passed
before helper changes; the meaningful RED is the actual exhaustive analyzer.
Developmental artifact 49/49, focused 23/23, five actual boundaries, errortype,
source/static and generator checks pass. Actual unfiltered artifact lint drops
from four to two byte-identical invariant-panic/uppercase-World findings, with
two resolved and none introduced. Helper Fatalf and deliberately shared-array
rejection remain unexecuted, with source inspection supplying only bounded
coverage. Production and 1,044 protected inputs remain unchanged. Source review
permits only SOURCE_PROGRESS_COMMIT_ONLY. Task 21 consumes this evidence; original
R13/R18/R19, task 12/predecessors, matched first-baseline identities and complete/
full/formal/both-native/affected-consumer qualification stay required and open.
Root commits reviewed progress before another source writer.

Task 34's independently reviewed private-mode fixture cleanup restores stdin
before checking its existing single pipe-reader Close, reports failure through
nonfatal testing.T, and preserves every other fixture byte and assertion.
Baseline and final private-mode 1/1, portable CLI 34/34, five actual boundaries,
errortype and formatting pass on developmental linux/arm64. Actual unfiltered
CLI lint falls from 54 to 53 byte-identical production findings, with exactly
one fixture diagnostic resolved and none introduced. All 1,044 protected inputs
remain unchanged; generator inputs are unaffected. Fresh source review permits
SOURCE_PROGRESS_COMMIT_ONLY. Genuine Close and simultaneous primary/cleanup
failure execution remain unproved. Task 21 consumes the evidence; original
R6/R18/R19, task 4/task 5/predecessors, matched first-baseline identities and
complete/full/formal/both-native/affected-consumer qualification remain required
and open. Root commits reviewed source progress before another source writer.

Task 35's independently reviewed corpus cleanup checks the two original production
releases and four fixture releases at their existing lifetime boundaries. Nil
Close preserves original results and concrete errors; a genuine cleanup failure
returns zero result with the raw sole error or primary-first joined errors and
prevents validation-dependent publication. Original operations, comments and fixture assertions stay intact outside the
admitted signatures and cleanup wrappers; 1,043 protected inputs remain unchanged. Four real-file controls retain exact
error order, publication state and a nonempty BASE canonical snapshot vector.
Corpus package 24/24, focused 14/14, five actual boundaries, errortype and
formatting pass on developmental linux/arm64. Actual unfiltered corpus lint
falls from six findings to zero with none introduced. Fresh source review
permits SOURCE_PROGRESS_COMMIT_ONLY. Genuine first-Close and simultaneous
operation/cleanup failure execution remain unproved. Task 21 consumes the
evidence; original R3/R13/R18/R19, shared fn108 assessment/retention,
task12/relevant predecessors, matched first-baseline identities and complete/
full/completion/formal/both-native/affected-consumer qualification remain
required and open. Root commits reviewed source progress before another writer.
Product/document diff checks pass; full staged diff-check retains only the raw
Go-env log's empty-GOFLAGS EOF warning (exit 2). Its archived bytes stay intact.

Task 36's reviewed Choice Exploration repair replaces only the failure-policy
switch with its two stopping predicates. Original statements, comments and
test bodies remain intact; 1,044 protected inputs match admission. Eight literal
BASE state/segment vectors, twelve trailing-sibling error controls and four
constructor cases preserve stopping, distinct-budget accounting, bounded
children, validation and input immutability. Developmental package 16/16,
focused 8/8, five actual boundaries, errortype and formatting pass; actual
configured choice lint falls from one exhaustive finding to zero. Review permits
source-progress commit only. Original R18/R19, round/search/replay, shared fn108,
predecessors/task21, matched first-baseline identities and full/completion/formal/
affected-consumer/both-native gates stay required/open. Product/document checks
pass; full staged diff retains only the immutable empty-GOFLAGS archive EOF
warning. Root commits before another source writer
([choice-policy acceptance](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-36/acceptance-open.md)).

Task 21's additive R18 accountability supplement binds the eight named campaign
helper/type seams to task 5/R6 through an exact retained-patch, WIP and checkpoint
source-block comparison. The WIP commit remains their first Git introduction;
whole-file/checkpoint equivalence is not established. The interface inventory
already lists the set, and task 26 repaired its actual CLI consumers. Task 20's
nine-flag correction and the separately owned fn112/fn113/fn114 migrations are
linked without repeating implementation or changing their contracts. Historical
qualification, audit, baseline and measurement records remain immutable. This
closes only the named inventory/ownership evidence gap. Choice Trace v2,
controller-v2 journals, selected v041 availability, changed guidance defaults,
matched first-baseline bytes and all original full/formal/affected-consumer/
both-native requirements remain open
([accountability supplement](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/r18-accountability-2026-10-04/accountability.md)).

Task 37's qualification-report cleanup checks the five retained staging Remove,
early writer Close and deferred reader Close results at their original lifetime
boundaries. Nil cleanup preserves primary error identity; genuine failures join
primary first, reader failures clear the report, and published paths stay intact.
All original production/test bodies and 1,045 protected inputs are preserved.
Saved BASE and final real-file controls bind literal report/evidence bytes,
private modes, validation and staging-name behavior. Developmental focused 8/8,
package 23/23, consumers 60/60, five boundaries, purity/edges, errortype and
formatting pass. Actual unfiltered package lint falls from six findings to the
one unchanged diagnostics import-order finding; five errcheck findings resolve
with none introduced. Real first-Close, multiple cleanup, Remove and post-rename
directory faults remain unexecuted. Root commits independently reviewed source
progress separately; original R18/R19/R20, task21/predecessor/shared-fn108,
matched first-baseline identities and complete/full/completion/formal/
affected-consumer/native-default/both-native acceptance remains required/open
([qualification-cleanup acceptance](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-37/acceptance-open.md)).

## Finding coverage

Secondary IDs identify the five opportunities in the assessment's order.

| Finding | Required result | Acceptance |
| --- | --- | --- |
| F1 | Campaign options and coordinator parity | R1 |
| F2 | Completed-execution assessment | R2, shared with fn-108 R6 |
| F3 | Retention and Artifact composition | R3, shared with fn-108 R7 |
| F4 | Complete preparation | R4 |
| F5 | Private execution injection and operation construction | R5, R6 |
| F6 | Generated simulation-time protocol | R7 |
| F7 | Architecture fitness checks | R8 |
| F8 | Current contract guidance | R9 |
| F9 | Coherent target command execution | R10 |
| F10 | Simulation operation lifecycle | R11 |
| F11 | Backend-specific handles | R12 |
| S1 | Artifact reference and owned handle | R13 |
| S2 | Typed model commands | R14 |
| S3 | Validated installation description | R15 |
| S4 | Atomic seed completion | R16 |
| S5 | Internal capability and source-inventory ownership | R17 |

R18-R20 apply across all findings. This coverage is contractual, not task status
or implementation evidence.
