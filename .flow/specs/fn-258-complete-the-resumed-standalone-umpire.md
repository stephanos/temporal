# Complete the resumed standalone Umpire migration

## Goal & Context

The owner renewed the goal on 2026-10-27: complete migration, then work on MILESTONES.md, using independent judgment and parallel agents. This replaces retired fn-257 without changing its completed export evidence or accepting its unfinished tasks. The stop/validation hold is superseded by the current instruction. Preserve the original inventories and recovery bundles.

The current integrated Temporal framework is newer than standalone f7b42ce. Freeze a new source inventory at ac60daaf47 before transferring its integrated delta. Retain and adapt consumer 91547f57ec. Do not substitute old candidates for current source.

## Acceptance Criteria

- **R1:** Record the exact source revision, complete selected-file byte/mode/symlink
  inventory, archive identity, history boundaries, destination probe result, and
  retained-candidate ownership/transfer ledger. Errors: an unknown or inaccessible
  destination stays unknown; dirty and unintegrated work is retained, never silently
  incorporated, discarded or declared transferred. Freeze newer source checkpoints
  explicitly rather than mutating an earlier inventory.
- **R2:** Produce a local export only in an isolated clone, retaining Rust lineage
  through all earlier framework/host/tool paths, relevant authors/dates/messages,
  and an old-to-new commit map. The extracted checkpoint exactly matches R1's
  selected source bytes, modes and symlink targets. Errors: revision qualification
  excludes Scala that formerly occupied the same paths; both sibling archive-move
  histories are included; destination collisions, omitted fixtures and mismatched
  trees fail the inventory comparison. Do not filter the working repository or its
  shared object directory.
- **R3:** A framework-only checkout owns all required support, focused documentation,
  attribution, agent instructions, native command aliases and CI. Canonicalize both
  Go module identities and regenerate the corresponding descriptors together with
  imports and schema options. Preserve protocol package names, field numbers, ABI,
  guest ownership and all semantic golden payloads. Errors: no build depends on a
  Temporal/Scala sibling, personal absolute path, parent generator, or previously
  populated target; existing caller-context and source-boundary controls remain.
- **R4:** Actual Temporal Rust and Go consumers resolve the local framework source
  through sibling Cargo paths and a parent Go workspace; existing model commands,
  guest freshness and CI use that source explicitly. Errors: missing checkout fails
  with actionable setup instructions; no remote/cache fallback replaces local source;
  existing parent workspace entries are preserved; generated files follow their
  owning schema and the committed Temporal guest remains reproducible. Consumer
  finalization consumes R3's actual exported paths, Go identities and forwarding
  commands; provisional preparation may proceed earlier.
- **R5:** When execution resumes, verify the exact current standalone/consumer pair,
  including an isolated fresh-target framework gate, protobuf freshness, release
  tests, real host/guest parity and required paired gates in fresh paired checkouts.
  Require generated-code cleanliness, unchanged fixture metadata/payloads and
  archive/cache identity before and after cutover. Disposable uncommitted
  Rust and Go edits each cause recognizable Temporal consumer failures, then green
  after exact restoration, without commits/publication between each pair. Retain
  revision-bound commands/results and final timings. Errors: stale guests, omitted
  checks, historical proofs from different sources and pre-existing failures never
  establish success; preserve the existing performance obligations and report misses.
- **R6:** Inspect the real destination and preserve its history during publication.
  Cut over only after R5 and a complete candidate-transfer disposition naming each
  repository, recovery ref, tracker record and cross-repository prerequisite; update both
  repositories' current documentation, CI, agent and Flow/milestone references.
  Errors: publication and source removal remain separately authorized boundaries;
  absent access, unresolved destination conflicts or validation hold prevent those
  steps. Retain the source checkpoint, all unfinished branches/evidence and the
  unchanged Scala archive/caches; transferred obligations remain open.


## Architecture & Boundaries

Use the reviewed architecture and complete acceptance contract in MIGRATE.md and fn-257. Keep relevant exported history, separate framework and consumer workspaces, actual sibling Cargo dependencies and parent Go workspace, canonical Go identities and unchanged semantic payloads/ABI. Preserve all unintegrated candidates and Scala archive/caches. Remote publication requires working destination access and authorization; no remote writes or source removal during preparation. Do not close outstanding fn-254/fn-246 or live-evidence obligations through extraction.

## Execution

Renew current inventory and candidate-transfer ownership. Repeat the independently verified history projection at ac60daaf47 in a disposable clone, extend the original old-to-new map through every available source commit, and prove each retained snapshot/author/date/message/parent plus every pruned commit's projection neutrality. Join this verified history to the portable standalone without rewriting either branch or changing its admitted active tree; an aggregate source patch alone does not satisfy R2. Transfer the integrated source delta and finish standalone portability; adapt actual consumers and source quick selection; prove exact pair with fresh-target standalone, generation, release/parity/required paired gates and disposable uncommitted Rust/Go red-green controls; then review cutover/publication and update current docs and milestones. Final proof and timings remain mandatory; historical receipts are provenance only.

## Verification

Use MIGRATE.md Verification evidence in full, with exact source revisions, commands and results. Scope focused checks to changed seams before broad final proof. Preserve fixed inventories and report performance misses. Each closure requires fresh review, committed evidence and validation-cost assessment under AGENTS.md.

## Conversation Evidence

"complete the migration and then work on MILESTONES.md; when in doubt follow your own recommendations instead of asking for input. use sub agents to parellize work."

## Interim ownership, proof expiry and publication

Temporal's in-tree framework remains authoritative until destination access, publication, current-pair proof and final cutover are established. The refreshed standalone and consumer lane are recoverable source candidates; keep the consumer branch unmerged while a fetchable destination/CI pin is unavailable. Task 3 produces a reviewed, bundled consumer candidate and local-resolution evidence, not CI activation or root-source cutover. Commit its closure/evidence on `umpire`; retain its source branch for the final task's serial integration. Task 4 verifies that exact isolated pair before integration. Task 5 retains R4/R6 ownership of actual CI activation and source cutover; access failure leaves task 5 open and the migration incomplete.

Any subsequent `model/umpire` product change expires the proof pair. Freeze a new immutable source manifest, renew candidate/ref observations, extend and independently verify the filtered history/map, repeat the task 2 adaptation inventory, adapt consumer interfaces and rerun every affected focused/final proof before cutover. Metadata-only Flow/doc changes do not alter source identity, but verify the framework tree matches the frozen manifest before reusing proof. Do not silently resume framework milestone edits against an expired migration checkpoint.

## Decision Context

Maintainability (plan review): duplication - canonical Go identity rewrites must be inventoried by an explicit per-path adaptation rule, regenerated descriptors and repeated freshness checks rather than guessed manual merges; structure - none identified.
