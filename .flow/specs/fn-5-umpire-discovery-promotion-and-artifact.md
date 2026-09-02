# Umpire discovery promotion and artifact

> HTML render lens (local): open `.flow/artifacts/fn-5-umpire-discovery-promotion-and-artifact/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Umpire4 architecture reconciliation

This stable Flow handle is retained so completed prerequisite and downstream references do not move.
The implementation scope is the two capabilities below and nothing implied by the historical title.

## Overview
<!-- scope: both -->

Give the current Nexus model a small, concrete discovery surface and give the one duplicate-delivery
failure a checked path to a Lean source proposal for human review. The discovery rows are a closed
projection of already checked Nexus values; the promotion path is a closed binding for one already
known failure, not a new semantic authority.

## Goal & Context
<!-- scope: business -->

Contributors can currently inspect one scenario when they already know its identity, but cannot ask
which retained Nexus examples exist or explain the checked declarations that make up one example.
The later fn-22 workflow also needs a static, compile-checked proposal target after it has reproduced,
minimized, and Exactly Replayed the duplicate-delivery failure. This spec supplies those two seams
without reversing the dependency: fn-22 consumes fn-5.

## Architecture & Data Models
<!-- scope: technical -->

`Temporal.Tool.NexusDiscovery` owns one closed `NexusDiscoveryEntry` list. It contains exactly these
four current examples, in canonical query-identity order:

1. `temporal.nexus.basic-lifecycle.query.async-start`
2. `temporal.nexus.basic-lifecycle.query.cancellation`
3. `temporal.nexus.basic-lifecycle.query.successful-completion`
4. `workflow-nexus.query.exact-action-caller-closure`

Each entry is constructed from its existing checked Property, Behavior, Query, and planned
`ExperimentSpec`; it carries their canonical identities and source locations rather than copied
semantic prose. `list` projects deterministic summaries from all four entries. `explain` performs
an exact query-identity lookup and projects the same summary plus the checked declaration and plan
lineage for that one example. Neither command infers entries by scanning imports or source text.

`Umpire.Promotion` owns a sealed `CompiledPromotionSource` and the smallest checker needed to prove
that a proposal uses the unchanged base checked Query, its target-owned `.found` expected count-one
trace and base planned `ExperimentSpec`, fresh fixed promoted Behavior/Query identities, fixed
imports, deterministic source bytes, and successful clean elaboration.
`Temporal.Tool.PromotionBinding` owns exactly one static
`PromotionCandidateBinding`,
`temporal.nexus.caller-closure.promotion.cancel-unique-regression`. The binding keeps two distinct
lineages: the unchanged `exactActionQuery`/PlannerRun/base `ExperimentSpec` supplies the checked
expected count-one trace, while the selected duplicate-delivery Space point supplies the separate
fault-bearing `ExperimentSpec` identity/checksum later reproduced and minimized by fn-22. The fault
intent and observed count-two result are never treated as target-owned Query output.

The fn-5 executable retains the closed downstream contract
`temporal-model-promote temporal.nexus.caller-closure.promotion.cancel-unique-regression` and emits
one inert canonical `umpire-promotion-proposal/v2` envelope plus one LF. Direct invocation proves
only that the fixed expected-trace source is checked, deterministic, and elaborates; it makes no
claim that any runtime failure is reproduced, minimized, or accepted by Exact Replay. Fn-22 owns
those eligibility gates and separately cross-binds the selected fault-bearing `ExperimentSpec`,
minimized Result, Violation Signature, and inert fn-5 proposal before it writes a review artifact.
Fn-5 does not import fn-22, consume runtime evidence, or install source.

```mermaid
flowchart LR
  N[Four checked Nexus examples] --> L[list]
  N --> E[explain exact query identity]
  Q[Base Query + expected count-one plan] --> B[One static promotion binding]
  D[Fault-bearing ExperimentSpec identity] --> F[fn-22 eligibility + cross-binding]
  B --> C[Elaborated CompiledPromotionSource]
  C --> P[Review-only proposal envelope]
  P --> F
```

## API Contracts
<!-- scope: technical -->

- `temporal-model-inspect list` emits one canonical `umpire-nexus-discovery/v1` JSON value containing
  the four summaries in exact query-identity order, followed by one LF and empty stderr.
- `temporal-model-inspect explain <query-id>` accepts exactly one canonical query identity from the
  closed list and emits one canonical `umpire-nexus-explanation/v1` JSON value followed by one LF.
  Unknown, case-shifted, ambiguous, or extra selectors emit empty stdout, one structured diagnostic
  plus one LF on stderr, and status 1. Existing positional scenario inspection remains unchanged.
- Discovery rows expose only existing checked identities, kind labels, source locations, Behavior
  Fingerprints, and planned `ExperimentSpec` identity. Output ordering is independent of authoring
  order and repeated calls are byte-identical.
- `temporal-model-promote temporal.nexus.caller-closure.promotion.cancel-unique-regression` accepts
  only that exact candidate identity and no source path, executable path, import, promoted identity,
  trace, or output override. Success emits exactly one canonical `umpire-promotion-proposal/v2`
  value plus one LF; unknown candidate, validation, elaboration, argument, or serialization failure
  emits no partial stdout.
- The inert promotion envelope separately binds the base Query/PlannerRun/base-`ExperimentSpec`
  identities, the selected fault-bearing `ExperimentSpec` identity/checksum, fixed promoted
  identities, source identity, SHA-256, and exact source bytes. It contains no runtime eligibility
  claim and is never written into a Lean package by the command.

## Edge Cases & Constraints
<!-- scope: technical -->

- The four-entry Nexus inventory is explicit and checked for duplicate identities, wrong kinds,
  missing source/fingerprint fields, crossed Property/Behavior/Query ownership, missing plans, and
  nondeterministic order before either discovery command can succeed.
- `explain` is exact: it does not case-fold, prefix-match, alias, or silently redirect selectors.
- Promotion rejects non-`.found` base planning, base Query/PlannerRun/`ExperimentSpec` drift,
  fault-bearing `ExperimentSpec` drift, crossed base/fault lineage, observed-trace substitution,
  reused promoted identities, missing imports, nondeterministic rendering, digest drift, or source
  that does not elaborate in a clean focused Lake build.
- Direct fn-5 invocation never establishes runtime eligibility. Fn-22 must reject non-reproduction,
  incomplete reduction, missing Exact Replay, receipt-digest mismatch, or crossed runtime lineage
  before cross-binding and writing the inert proposal as a review artifact.
- Existing comments and current single-scenario inspector behavior are preserved.

## Quick commands

```sh
cd model && mise exec -- lake build Temporal.Tool.NexusDiscoveryTests Temporal.Tool.PromotionBindingTests TemporalExperimentalTests temporal-model-inspect temporal-model-promote
make umpire-check-regression
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `temporal-model-inspect list` and `explain` expose one coherent, deterministic, exact view
  of the four retained Nexus examples and their existing checked Property, Behavior, Query, source,
  fingerprint, and plan identities. Invalid inventory state or selectors fail structurally without
  partial output, while existing positional inspection remains byte-compatible.
- **R2:** Exactly one checked binding can compile the unchanged base Query's target-owned expected
  count-one trace into deterministic Lean source and a canonical inert review-only proposal for the
  separately identified duplicate-delivery Space-point `ExperimentSpec`. The fixed candidate-argument
  command makes no runtime eligibility claim. Fn-22 alone admits its output after runtime
  reproduction, complete minimized-or-irreducible reduction, and Exact Replay, then cross-binds the
  fault-bearing `ExperimentSpec`, Result, Violation Signature, and proposal before publication.
  Observed count-two substitution, conflated base/fault identities, drift, unelaborated source, or
  any override produces no proposal and nothing is installed automatically.

## Early proof point

Task `.1` must prove the four current examples form one deterministic closed inventory without
copying their checked meaning. Task `.4` must prove a proposal cannot substitute the observed
count-two trace for the target-owned expected count-one trace. If either proof fails, revise the
concrete adapter or sealed-source boundary before wiring a command.

## Non-goals
<!-- scope: business -->

- A generic semantic graph or reusable reference graph.
- A generated glossary or machine index.
- A broad stable regression set or generalized regression suite.
- General artifact evolution, schema migration, or a new persisted artifact family.
- Source scanning, automatic proposal installation, runtime replay, minimization orchestration,
  SDK history replay, campaign management, remote execution, or CI/release Claim Assessment.

## Decision Context
<!-- scope: both -->

### Motivation
<!-- scope: business -->

The existing inspector and checked Nexus values already provide the authority needed for the
retained discovery surface. Extending that concrete tool keeps the user experience coherent and
avoids introducing a second vocabulary.

### Implementation Tradeoffs
<!-- scope: technical -->

A single static promotion binding preserves the useful checked-source seam required by fn-22 while
keeping runtime evidence and review policy outside the reusable Lean source type. The closed
four-entry adapter is deliberately concrete: adding an entry requires a reviewed code change, but
that cost makes current scope and ownership obvious.

## Requirement coverage

| Requirement | Tasks | Notes |
|---|---|---|
| R1 | `.1`, `.2`, `.3`, `.7` | Closed Nexus inventory plus exact list/explain and focused integration checks |
| R2 | `.4`, `.5`, `.6`, `.7` | Sealed source, one duplicate-delivery binding, elaboration, and review-only command checks |
