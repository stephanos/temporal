# The conformance descriptor ships with the Case

## Conversation Evidence

> user (2026-10-18): "I thought the testpilot IR Cases are self-contained ie contain all the checks needed that the model expects. why would we need to pull in the model again? it's supposed to be a standalone export that runs anywhere without having the original model"
> user (2026-10-18): "conformance for models allows for two strategies; one is sending traffic and observing … the other one is passively observing behavior and feeding that into the model to verify it holds … does that require the full model ie JVM instead of just a Go program?"

## Goal & Context

A Case is self-contained for its own verdict: the Contract carries every check the witness path needs. The second judgment, whether everything the server did is explained by some path of the Model and what every Property and monitor says on the executions that explain it, today loads the Model through the Go reader at run time. That is the only reason a live test or an offline assessment needs a Model. This spec makes that judgment an artifact: the Scala producer emits a conformance descriptor (the reachable slice as rows with claim readings folded in, monitors as automata) alongside the Program and Contract, and the Go runtime interprets it with no Model in sight. Active conformance (send traffic, observe) and bounded passive conformance (feed observed events, check they are explained) then both run in `tools/testpilot` without a JVM. Unbounded passive observation, which can leave any finite slice, is a later JVM mode and out of scope.

## Architecture & Data Models

- Descriptor contents (schema in fn-202): interned states; rows keyed by state and class with ordered results (target, outcome, facts, choice); hole marks per pair; per-result Property bits (about, holds); monitor automata (states, next over (monitor state, result), read set, violated set); progress and end marks; fairness classes; abstract evidence roles (correlation keys, ordinals, closing events).
- The product with monitors is never materialized; the interpreter forms it lazily along the executions the evidence picks out.
- Scala emission: the same row enumeration fn-203 exposes, cut from the Query's start under `Limits`, serialized once per Case or once per Query start when several Cases share one.
- Go interpretation: the existing bitset search over partially ordered evidence reads rows and bits instead of `check.Realizer`; evidence reading maps observed events to roles through the Temporal driver's concrete mapping.
- Consumers: the live tests' in-process assessment, `umpire-run --model` and `umpire-assess --model` all take the descriptor; pinned runs are re-recorded against descriptor-based receipts.

## API Contracts

- `Cases.lower` attaches a `ConformanceDescriptor` to each Case when the Query's Model declares claims; a Case without one is judged by its Contract only.
- Go `conformance.Prepare(descriptor, case, limits)` replaces `Prepare(model, claimKey, case, limits)`; `AssessmentFactory` and `Assessor` keep their shapes.
- Assessment outcomes keep their vocabulary: Conformant, Nonconformant, Inconclusive, with claim conclusions Satisfied, Violated, Inconclusive and the reason IDs; a new reason "outside descriptor bound" marks evidence the slice cannot explain.
- Size: the descriptor for each sample's Query slice is measured and recorded; a slice above an owner-set ceiling is refused with the size at lowering time.

## Edge Cases & Constraints

- Evidence outside the slice is "outside descriptor bound", never nonconformant.
- A hole pair in the slice keeps its unknown standing; a claim read through a hole is inconclusive.
- Monitors run unbounded on their own automata; only machine-state claims are bound by the slice.
- The descriptor is derived data; a mismatch between a Case's Contract and its descriptor is a producer defect and fails the fixture check.

## Acceptance Criteria

- **R1:** The Scala producer emits a descriptor per Case from the native graph with rows, interned states, hole marks, Property bits, monitor automata, progress and end marks, fairness classes and evidence roles; the samples' Cases carry one and its size is recorded. Errors: a slice above the ceiling refuses at lowering with the size.
- **R2:** Go conformance assesses from the descriptor alone, with no IR, interpreter or checker import, and reproduces every pinned run's recorded verdict after re-recording. Errors: evidence outside the bound yields the new reason, never nonconformant.
- **R3:** The live tests' in-process assessment, `umpire-run --model` and `umpire-assess --model` take the descriptor; the Go `conformance` package imports nothing from the Model-reading packages. No error surface beyond R2.
- **R4:** Bounded passive observation works: an event stream from a recorded Run or an external source, mapped through the evidence roles, is judged by the same interpreter. Errors: an event with no role mapping is reported, not dropped.
- **R5:** Monitor-only passive checks run without a machine slice and are unbounded. No error surface beyond R4.

## Parked unknowns

- Whether a GraalVM native-image build of the Scala checker is viable (Scala 3 macros, retained trees, ScalaPB, discovery). On yes, a native oracle sidecar beside the Driver would make the Contract's verification clauses (fn-203 R3) and this descriptor redundant for every mode, including unbounded passive observation; the Case would keep only the Program and execution-control predicates, at the cost of no longer being self-contained. Resolved by a spike (proposed fn-209) that re-cuts fn-203.3 and this spec on go. Raised by the owner 2026-10-18.

## Boundaries

- Unbounded passive observation against the full Model (a JVM mode over `Checker.replay`) is out of scope; recorded as later work.
- No change to the Contract's semantics.
- No Temporal Model migration.

## Decision Context

2026-10-18, phase renumbering: the owner inserted Phase 3 "Temporal Models native" (re-enable and check `model/temporal` against the final DSL) between DSL completion and the IR batch; the IR batch is Phase 4 and the one-language series (fn-199 through fn-209) is Phase 5. This spec runs in Phase 5 unless its Decision Context says otherwise.

The alternatives were to keep a Model reader in Go (what we are retiring), to assess offline only in Scala after the run (loses mid-run nonconformance and makes the live tests JVM-dependent), or to ship the judgment with the Case. Shipping matches the P pattern (spec monitors compiled and run by the harness) and matches what the Go conformance search already consumes: it never interprets a Model, it walks a bound table and reads claim readings. Folding readings into rows rather than materializing products keeps the descriptor linear in pairs and lets the interpreter pay only for the executions evidence selects. Depends on fn-202 (schema) and fn-203 (row enumeration and producer). Filed 2026-10-18 as part of the one-language series (fn-199 through fn-208).

2026-10-18, one-language series: absorbs by reference fn-128.7/.8 (evidence classification and Profile binding in conformance): both become descriptor evidence roles and the Go interpreter's reading of them (.2). fn-196 R5's clause that existing replay, binding and table guarantees remain until replaced now lives on R2.

2026-10-18, deterministic simulation (owner): the descriptor is the deterministic runtime's bounded-exploration oracle until fn-209 decides otherwise; it is a table, so it forks with the process and needs no RPC. Keep the descriptor's slice selectable per exploration (per Query start under Limits) so a schedule-exploring runtime can ask for the slice it needs.

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.

