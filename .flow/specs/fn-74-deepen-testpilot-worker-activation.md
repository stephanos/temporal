# fn-74-deepen-testpilot-worker-activation Deepen Testpilot worker activation semantics and preparation diagnostics

## Goal & Context
<!-- scope: business -->

Make the Temporal Driver implementer's semantic responsibilities small enough to test without an SDK runtime. Today the workflow and Nexus-handler interpreters assemble activation-owned reference maps, expression evaluation, work subtraction, validation, and immutable outcome storage around Testpilot's prepared instruction methods. Those methods already provide shared typed semantics, but their correct composition remains the adapter's responsibility. A private activation module will own that composition, while the adapter retains SDK commands, futures, cancellation and delivery authority.

Separately, callers of public admission functions receive structured errors whose concrete type lives in Testpilot's inaccessible `internal/ir` package. Provide a stable public preparation diagnostic so callers can classify and report rejection without parsing messages.

This implements architecture-review finding 5 after fn-72-extract-the-reusable-temporal-testpilot establishes the shared Temporal Driver. It is an independent improvement to correctness and implementer effort, not a prerequisite for the first canary. It preserves UMPIRE4 ART-09/10, EVD-11, EVD-15/16/17, and the private execution/evaluation authority split.

## Architecture & Data Models
<!-- scope: technical -->

**Dependency:** fn-72-extract-the-reusable-temporal-testpilot must establish `common/testing/temporaltestpilot` (Go package `temporaltestpilot`) with its composed server/worker Driver and private delivery implementation. This spec consumes that location; it does not perform the relocation. No dependency on fn-71-standalone-lean-testpilot-protocol's Lean protocol extraction, fn-73-explicit-environment-binding-for's environment binding, canary implementation, or model-to-Case lowering is needed.

**Ownership decision:** place the new activation module inside the reusable Temporal Driver's private implementation, accessible to its worker adapter and unit tests. It depends on public Testpilot prepared plans and protocol value types. It must not import Testpilot's inaccessible internal packages. Workflow and Nexus-handler execution are two concrete paths in the same adapter, not evidence for a new public adapter framework. Therefore no public activation API is added. Existing public low-level plan methods remain compatible.

The module owns one fresh mutable state per actual worker interpretation, containing its immutable entrypoint plan, cached instruction plans, activation-local admitted outcome fields, instruction evaluation/admission state, and remaining work. Prepared plans contain no mutable activation state. Worker replay recreates this state from the prepared plan and the SDK's replayed results. Delivery/reservation identity checks and redelivery caching remain with the existing Driver; activation state neither authorizes delivery nor survives as a shared cross-Run cache.

The small interface has three operations: construct from one prepared worker entrypoint; evaluate the guard and optional input of an identified instruction; admit that instruction's primitive outcome. Evaluation resolves references exclusively against previously admitted, local fields. Outcome admission uses existing `InstructionPlan.ValidateOutcome`, privately owns the resulting snapshots, and publishes all declared fields together only after complete validation. Work charges returned by existing `EvaluateInput` and `ValidateOutcome` are consumed internally on both success and failure. Existing IR expression and outcome validation remain the semantic authority; the extraction must not recreate them.

The SDK adapter still selects instructions using the existing prepared DAG order, executes supported opcodes, constructs primitive success/failure outcomes, holds Nexus futures, waits with SDK timeouts, and handles SDK cancellation and terminal responses. The new module selects no instruction, schedules no work, starts no goroutine, creates no timer, consults no wall clock, performs no I/O, and calls no SDK API. Controller Slots, attempts, recorder publication, projections, Monitor state, Run-level limits and controller scheduling remain private to Testpilot execution.

The public preparation diagnostic belongs to the `testpilot` facade, independently of the private activation helper. Map admission errors at public `NewCatalog` and `Prepare` boundaries; preserve internal error representation for internal consumers. This includes Program and Contract preparation and facade preconditions.

## API Contracts
<!-- scope: technical -->

The private activation interface takes a prepared worker entrypoint at construction and an instruction index relative to that entrypoint for evaluation/admission. Exact private Go names are implementation details. Its complete operation contract is:

- Construct: prepared entrypoint → fresh activation state or error. Accept currently supported workflow and Nexus-handler contexts; reject zero/invalid plans and controller contexts. Derive the production work ceiling from the prepared entrypoint; no adapter-supplied larger budget or reset operation exists.
- Evaluate: context, instruction index → independent optional input value, enabled boolean, error. Resolve and charge the guard first. False yields `(nil, false, nil)` and does not evaluate the input or create outcome fields. An enabled instruction with no authored input returns `(nil, true, nil)`. A missing required read is an error, not a disabled instruction.
- Admit: context, instruction index, primitive `InstructionOutcome` → error. Require an enabled evaluation of that instruction, validate once, and atomically retain private snapshots of its declared fields. Admission before evaluation, after a skipped guard, or repeated admission rejects without replacing any fields. No caller receives the private reference map, resolver, snapshot map or mutable work counter.

One activation instance is used serially by its interpreter. Independent instances can run concurrently against the same prepared plan. The interface does not promise concurrent calls on one instance and must not introduce locks or Go channels into workflow interpretation. Evaluation may not reset an instruction's completed/skipped state or its budget. Failed evaluation/admission publishes no outcome; the adapter propagates the failure and abandons that interpretation rather than retrying or refreshing its allowance. Unknown indices and invalid contexts return errors rather than panic. Constructing a fresh state for a separate interpretation cannot expose another state's values, even when entrypoint and instruction IDs match.

The public diagnostic shape is:

```go
type PreparationErrorCategory string

const (
    PreparationMalformed     PreparationErrorCategory = "malformed"
    PreparationUnknown       PreparationErrorCategory = "unknown"
    PreparationTypeMismatch  PreparationErrorCategory = "type_mismatch"
    PreparationUnavailable   PreparationErrorCategory = "unavailable"
    PreparationUnsupported   PreparationErrorCategory = "unsupported"
    PreparationLimitExceeded PreparationErrorCategory = "limit_exceeded"
)

type PreparationError struct {
    Category PreparationErrorCategory
    Path     string
    Detail   string
}
```

`*PreparationError` implements `error`. These are the complete exported fields and categories. Internal cause retention and `Unwrap() error` may be implemented without adding exported fields. `errors.As` must recover this public type through ordinary wrapping. Category is a stable machine-readable classification; Path identifies the rejected input using the existing admission path vocabulary; Detail is human-readable, not a stable string API. Preserve existing admission messages where present and retain the existing bounded path behavior. Invalid/missing Profile and Catalog preconditions use `malformed` with meaningful Profile/Catalog paths. Successful calls return no diagnostic. No JSON diagnostic wire format is introduced. Decode-only ProtoJSON errors and runtime Run/Driver failures are outside this new classification contract.

## Edge Cases & Constraints
<!-- scope: technical -->

- Preserve presence checks and boolean short-circuit behavior. A false success guard must not access the absent result of a failed SDK operation; a true success guard with an absent required result must fail. Skipped instructions supply no fabricated success outcomes.
- Preserve type, byte, depth, fanout and outcome-status validation, including rejection of undeclared/wrong typed payloads and prohibition on storing StartNexusOperation futures as VALUE. Opaque completion capabilities never enter value expressions or this module.
- Use the current execution expression accounting, including ownership-copy charges, rather than Contract evaluator accounting. One activation's allowance decreases across evaluation and admission; exact available work succeeds and one-less-than-required work fails. Failed operations retain consumed charges; zero remaining work cannot underflow, wrap, reset, or admit more positive-cost work. Keep the prepared ceiling derivation and existing accounting units unchanged.
- Snapshot ownership must survive mutation of raw SDK outcomes, evaluated inputs and any existing public plan snapshots. Stored values are never handed to an adapter for mutation. Additional copying must remain bounded; avoid copying an entire prepared instruction slice on every operation.
- Context cancellation propagates through pure operations. Workflow code continues using replay-compatible SDK context checks and bounded pure evaluation, without replacing them with wall-clock Go deadlines. Existing SDK cancellation, stop races, late publication rejection and quarantine behavior remain unchanged.
- This state is ephemeral. A process failure creates no new persistence/recovery responsibility; existing SDK replay and Driver lifecycle behavior remain authoritative. At ten times the admitted activations, state remains independent and bounded per activation, with overall admission limits enforced by existing Driver and executor owners; no new global registry is introduced.
- Preserve existing comments relating to changed code and preserve unrelated working-tree changes. No new third-party dependency is required.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A private pure activation module is owned by the reusable Temporal Driver established by fn-72-extract-the-reusable-temporal-testpilot and has construction, evaluation and admission as its only semantic operations. It reuses existing compiled expression/outcome validation through public Testpilot plans, and exposes no public activation or scheduling API. Errors: invalid/zero plan, unsupported context and invalid instruction index reject without panic or SDK/I/O work.
- **R2:** Both real workflow and Nexus-handler interpreters use the module for reference resolution, input/guard evaluation, outcome storage and cumulative work accounting. Their instruction execution paths no longer assemble reference maps, subtract returned work, or invoke outcome validation and storage as separate steps. Errors: evaluation/admission failure reaches the existing adapter error path and prevents the dependent SDK instruction or terminal response; no error surface beyond the new module and existing SDK adapter errors.
- **R3:** Unit tests through the smaller interface demonstrate guard-first evaluation, false-guard short-circuiting over unavailable inputs, presence checks, successful guarded reads, and failure for missing required values after a true guard. Errors: skipped nodes admit no outcome, malformed/out-of-order/duplicate admission rejects atomically, and unknown references remain absent rather than resolving from another owner.
- **R4:** Tests demonstrate cumulative work across multiple evaluations and admissions, exact-bound success, one-unit-short rejection, error-path charges, and isolation of one activation's exhaustion from another. Preserve current runtime ceiling derivation and expression/validation work units. Errors: nonpositive remaining allowance, overflow/underflow hazards and over-budget work fail closed without partial field publication or allowance reset; tight budgets may be injected by private module tests without exposing a production override API.
- **R5:** Tests demonstrate immutable raw-outcome and evaluated-input ownership, no values on failed validation, and distinct sequential/concurrent activations with identical entrypoint/instruction names sharing one unchanged prepared plan. Errors: invalid outcome status/type/size, undeclared values, nil outcomes and canceled contexts reject through existing validation semantics; failed/foreign data cannot replace an admitted snapshot.
- **R6:** SDK workflow Start/Await/Finish and Nexus Respond paths retain their results, timeout behavior, deterministic replay, delivery/redelivery isolation, cancellation and late-publication behavior. Existing focused SDK and session regressions pass after adaptation, including replay and Stop-race tests; controller scheduler and recorder authority are unchanged. Errors: absent futures, SDK failure/timeout/cancellation, delayed or foreign delivery, and post-close publication retain their existing adapter handling.
- **R7:** Public `NewCatalog` and `Prepare` rejection paths return publicly nameable `*PreparationError` with stable category, meaningful path and human detail. External-package tests use only public imports and `errors.As`, including through wrapping, to classify malformed input/profile, unknown declaration, type mismatch, unavailable reference, unsupported capability and exceeded limits across applicable catalog, Program and Contract cases. Errors: facade nil/typed-nil preconditions are classified; successful preparation yields no error; internal type imports and error-string parsing are unnecessary.
- **R8:** Existing public Driver/Session/PreparedCase signatures, low-level plan method behavior, Case/protocol bytes, Contract interpretation and Run disposition/cleanup/Verdict separation remain compatible. Focused pure-module tests, public facade tests and the reusable Driver's SDK/session suites pass with `-tags test_dep`; run `make lint-code` as the repository standards gate. Errors: any environmental inability to complete a gate is reported explicitly rather than marked passing; no new error surface beyond R1–R7.

## Boundaries
<!-- scope: business -->

- fn-72-extract-the-reusable-temporal-testpilot owns the Driver relocation and import migration; this spec builds on that completed seam.
- No new generic plugin framework, exported scheduler, alternate executor, public IR extraction or public reservation ledger.
- No consolidation of controller Run/Slot state with worker replay state and no controller scheduler rewrite.
- No new SDK instruction, retry policy, activity interpreter, event stream or outcome kind.
- No Lean protocol, model lowering, environment-binding, canary deployment, authorization policy or fixture-production changes.
- No removal or mandatory migration of existing public low-level plan APIs; the smaller interface is internal until a concrete independent adapter needs it.
- No universal error taxonomy for runtime, transport or ProtoJSON decoding failures.
- This improvement does not gate first-canary delivery absent a separately demonstrated correctness requirement.

## Decision Context
<!-- scope: both -->

The review recommends initially private ownership. Inspection supports that choice: the current worker's `evaluateInstruction` and `validateAndStore` functions already delegate semantic evaluation/validation but manually compose budgeting and `activationValues`; workflow and Nexus paths repeat that composition. The controller has a richer private store with Slot/projection/recorder atomicity and per-attempt ownership. Merging those stores would couple replay execution to controller synchronization for no demonstrated consumer benefit.

A small private state owner removes concrete responsibilities from both worker paths without publishing a second facade over the same scheduler. Instruction metadata and traversal remain in the adapter because dispatch and SDK scheduling are its authority. Sharing the existing evaluator and validator preserves semantic parity and avoids a second implementation of guard or outcome rules. The public diagnostic is justified independently by existing public admission calls exposing an otherwise inaccessible structured error.

The performance tradeoff is bounded snapshot ownership in exchange for mutation isolation; caching instruction plans avoids repeated whole-entrypoint copies. Per-activation state scales with admitted instructions and payload bounds, without shared mutable registries. The complexity tradeoff deliberately leaves the existing low-level public methods available rather than mixing interface removal into a correctness extraction. The security boundary is unchanged: pure values carry no SDK futures, credentials, clients or completion capabilities. A general adapter framework, public activation service, global budget manager and new recovery layer are rejected because none has an additional concrete consumer here.
