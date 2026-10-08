# Safety and liveness groups for object properties

## Conversation Evidence

> user (turn 1): "> object properties:"
> user (turn 1): "in umpire; what if we separated clearly into safety and liveness properties on that level? I think it would be a great help"
> user (turn 2): "let's write a flow next spec (or just todo?) to address this; add it to MILESTONES.md"

## Goal & Context

<!-- Source: [paraphrase] -->
Model authors and reviewers should see safety and liveness separately at the object-properties
level. The distinction makes it easier to review both what a design must prevent and what it
must eventually accomplish.

## Architecture & Data Models

<!-- Source: [inferred] -->
Each machine or composition groups its authored claims under properties.safety and
properties.liveness. Safety contains same-step and transition assertions and references to
safety monitors. Liveness contains the existing bounded progress claims. Shared capability
laws follow the same classification without duplicating their definitions or registration.
Queries continue to select how and where a claim is checked.

## API Contracts

- The authoring checker enforces the group against the claim's declaration kind. It does not
  attempt to infer arbitrary temporal meaning from predicate bodies or declaration names. [inferred]
- Existing diagnostics and claim reports expose the classification and retain the claim's
  source location, bound and assumptions where applicable. [inferred]

## Edge Cases & Constraints

- A postcondition saying that a successful worker response leaves an activity completed is
  safety. It does not require the worker to respond. An existential witness is likewise no
  liveness proof. [inferred]
- Liveness currently means bounded progress. Preserve explicit bounds and fairness or recovery
  assumptions; exhausted exploration or missing evidence cannot establish progress. [inferred]
- Helpers are not claims. Empty groups may be omitted, and regrouping must not register a shared
  law or monitor twice. Preserve monitor attachment and Query selection semantics. [inferred]
- Preserve modeled behavior, check results and generated Case meaning. Review and account for
  any source-derived identity or provenance changes caused by moving declarations. [inferred]

## Acceptance Criteria

- **R1:** Authored object properties have distinct safety and liveness groups, visible at the
  declaration site. No error surface beyond the declaration checks in R2. [paraphrase]
- **R2:** The authoring gate rejects a progress declaration in safety and a safety assertion in
  liveness, naming the declaration and expected group. Valid groups and omitted empty groups
  pass; helpers and shared laws do not create duplicate claims. [inferred]
- **R3:** Liveness declarations retain explicit progress bounds and assumptions. Negative
  fixtures cover a missing or invalid bound and a progress counterexample; incomplete checks
  remain incomplete. No unbounded eventuality claim is introduced. [inferred]
- **R4:** Existing claim diagnostics and reports distinguish safety from bounded liveness and
  retain source attribution. A missing progress prerequisite never suppresses an independently
  established safety violation. Unsupported checks remain explicitly unsupported. [inferred]
- **R5:** Existing Models, shared capability laws, references and authoring examples use the
  grouping consistently. Regression checks preserve behavior and verdicts, including negative
  controls; generated identity and provenance changes are explicitly accounted for. [inferred]
- **R6:** Author documentation explains the two groups with a safety postcondition and a bounded
  progress claim, including their different treatment of assumptions. It explicitly distinguishes
  finding a witness from proving progress. No error surface beyond R2 and R3. [inferred]

## Boundaries

- Unbounded temporal operators, new fairness semantics and a new progress-checking algorithm are
  outside this change. [inferred]
- Classification does not change the machine transition relation or create new product promises.
  It does not turn witness Queries into universal claims. [inferred]
- No independent reporting application or new runtime verification protocol is required. [inferred]

## Decision Context

- Keep the two groups inside properties so a reviewer finds all promises in one place. Use a
  checked distinction so headings and evaluated claim kinds cannot disagree. [inferred]
- Capture this as one cohesive spec. Validation, reporting and migration make a bare reminder
  insufficient to preserve the intended distinction. [inferred]
- Coordinate implementation with fn-140's property/Query authoring changes and fn-141's declaration
  lifting changes. Re-anchor to the completed vocabulary and schema before implementation; this
  spec does not change the approved delivery order. [inferred]
