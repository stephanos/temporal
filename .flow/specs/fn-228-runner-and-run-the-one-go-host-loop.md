# Runner and Run: the one Go host loop over the Model, Propose on the protocol, and the Testpilot shrink (model, temporal)

## Conversation Evidence

> user (2026-10-20): "tell me more here. I thought with the model shipping as WASM, we could instead consult the model from Go and make that dumb to contain all the smarts in the model."
> assistant: that is the recorded plan (R33 to R40): the Model judges every observation and answers every callback; Go keeps transport, mechanics, recording and storage; the Temporal knowledge sits in the Rust realization inside the guest. One gap: the protocol has no "what should I send next" question, so Go still chooses actions.
> user: "is Case slice still the right name?"
> assistant: no; Case is Testpilot's authored artifact and the thing this work deletes. The deliverable is the loop (R38 "host loop") and its output is a Run (R39).
> user: "what are the names for the packages/itnerfaces we're building. those are impprtant to get right"
> assistant: keep the `umpire-*` crates, `umpire.model.v1`, Explore/Rows/Reset/Judge/Answer, Observation/Callback/Reply, `Host`, guest/host. Rename `umpire-wasm` to `umpire-runtime` and `umpire-tests` to `umpire-fixtures`. One word for the binding: Realize/Realization. New: `Propose` on the protocol; Go package `runner` with `Runner` and `Run`; `Driver` (kept from Testpilot) and `Shim` (R40). Vocabulary: the Model is the umpire; the Runtime hosts one Model; a Realization binds it to one system; the Runner drives a Driver, asks the Model to Propose, Judge and Answer, and records a Run.
> user: "okay write that flow spec"

## Goal & Context

The Model ships as a wasm guest and Go already consults it through `umpire.model.v1` for judgment (Judge) and SDK callbacks (Answer). Testpilot, the Go test runtime, still carries roughly 20,000 generic and 29,000 total non-test lines of its own smarts: CEL predicates, contracts, verification, evaluation profiles, an IR interpreter, replay, campaigns, and generated Case trees. fn-216 proved the dumb-host shape for one activity, bespoke, outside Testpilot. This spec makes that shape the only one: Go becomes a Runner that drives a Driver, asks the Model what to send, hands every observation and callback to the Model, and records a Run. Everything that decides meaning moves into, or already lives in, the Model and its Realization inside the guest.

It discharges the post-verdict obligations R33 to R40 recorded in DESIGN.md's "Recommendation and next slices" table and retires the "Case slice" name. It follows fn-227, because proposing a request needs a world and action inputs to fill it from.

## Architecture & Data Models

- **Names.** Crate `umpire-wasm` becomes `umpire-runtime` (the export-free Model runtime: state, replay, freezing, Reset, Realize, Runtime); `umpire-tests` becomes `umpire-fixtures`. "Realization" is the one word for a Model-to-system binding; README and DESIGN prose stop saying "binding". Go: package `runner` with types `Runner`, `Run`, `Driver`; the wasm transport stays `Host`; the out-of-process worker forwarder is `Shim`.
- **Propose.** Method `METHOD_PROPOSE = 6` in `umpire.model.v1`. `ProposeRequest { uint64 seed = 1; }` and `ProposeResponse { google.protobuf.Any request = 1; string action = 2; }`; a Model with no enabled external action answers `Refusal` with a new `REFUSAL_CODE_QUIESCENT = 9`. `Realize` gains `fn propose(&self, state: &M::State, seed: u64) -> Result<Proposal<M::Action>, Refusal>` where `Proposal { action, request: Any }`; the Runtime checks the action is enabled in the current state and records it as pending until the Driver's Observation is Judged. Internal actions and Deadline ticks are never proposed. Determinism: the same state and seed yield the same proposal.
- **Runner (Go, `umpire.local/model/umpire/runner`).** One loop: Reset; repeat { Propose; Driver.Act(request) gives Observation(s); Judge each; Answer any Callback the Driver surfaces } until the Model is quiescent, the step budget is spent, or a judgment is Unexplained. Exactly one `Host` (one Model instance) per Run; callbacks are serialized into it. Modes are arguments to the one loop: `question` (one Propose or Judge over a given state, no Driver), `explore` (seeds 1..n), `reduce` (shortest seed prefix reproducing an Unexplained), `replay` (a recorded Run's observations re-judged, no Driver), `repeat` (n Runs, same seed set). One command `umpire` with those modes replaces run, fuzz, replay, assess and repeat.
- **Run.** `umpire.run.v1.Run { model_hash, protocol_version, world, seed, repeated Event events, Verdict verdict }` where an Event is one of Proposal, Observation plus JudgeResponse, Callback plus AnswerResponse, Refusal, or DriverFault (unreachable endpoint, callback timeout, read timeout without Model acceptance). The verdict is Explained, Unexplained(at event i) or Refused. No profiles, receipts, provenance, contracts or evaluation documents. `publish` stays a never-overwrite store of Runs keyed by model hash plus seed; the canary keeps only its lease and policy.
- **Driver (Go interface).** `Act(ctx, request Any) ([]Observation, error)` and `Callbacks() <-chan Callback`; plus the mechanics the Model does not own: poll interval, read timeout and backoff as instruction fields on the Observation request, capture and redaction as field paths, RPC completion and evidence well-formedness checks. The Temporal Driver is the retained `temporal/*` driver code reduced to act-and-observe over the generated API; it never parses a Temporal message for meaning.
- **Shim.** For non-Go SDK workers: a local gRPC endpoint on the Runner speaking `umpire.model.v1` Answer with a protocol version field; a thin per-SDK forwarder returns result, failure, heartbeat or delay. Unknown version is a Refusal. In-process Go workers call Answer directly.
- **Deletion.** `internal/verification`, `evaluation`, `cel`, `predicate`, the IR CEL bridge, `contract`, `casefile`, `campaign` and `replay` are removed; `internal/ir` shrinks to instruction decoding, `internal/execution` to act-and-observe; generated Case trees and the seven legacy failing tests go with them, replaced by Runner questions over the activity Model.

## API Contracts

- `umpire.model.v1`: new `Method`, `ProposeRequest`, `ProposeResponse`, `Response.propose`, `REFUSAL_CODE_QUIESCENT`; all existing messages and codes unchanged; wire goldens for the new method.
- `umpire.run.v1.Run` as shown; it is the only artifact a Run writes.
- `Realize::propose` as shown; existing `observe`, `matches`, `answer` unchanged.
- Go `runner.Runner` with `Run(ctx, mode, opts) (*Run, error)`; `runner.Driver` as shown; `host.Host` gains `Propose`.
- `umpire` command: `umpire <question|explore|reduce|replay|repeat> --model <guest.wasm> [--seed n] [--budget n] [--run <path>]`; exit code 0 Explained, 1 Unexplained, 2 Refused or DriverFault.

## Edge Cases & Constraints

- Every existing golden, wire golden and the fn-216/fn-227 live slice keep passing across the renames; the renames land alone before any behaviour change.
- A Propose while a proposal is pending is a `REFUSAL_CODE_STATE`; a Judge that explains the pending proposal clears it; an Unexplained freezes the run until Reset, as today.
- A Driver fault is recorded as an Event and ends the Run with Refused; it is never a Go verdict about the system.
- Reduce is bounded by the original seed count and reports the shortest reproduction it found, not a minimum.
- Size: at most 5,000 generic-runtime and 11,000 total non-test Go lines in Testpilot after this spec, reported by package in DESIGN.md; the generated `v1` package is excluded from both counts.
- R30 budgets hold for `make model-check`; the Runner's Go tests join the existing Go host targets, not the Rust gate.
- No new Rust crates beyond the renames; Go stays `CGO_ENABLED=0` on wazero.

## Acceptance Criteria

- **R1:** `umpire-wasm` is `umpire-runtime` and `umpire-tests` is `umpire-fixtures` everywhere (Cargo, Makefile, scripts, README, DESIGN, MIGRATE); prose uses Realization, not binding. Errors: the gate and `UMPIRE_BACKENDS=require` pass byte-identically; a leftover old name fails a grep check in the gate.
- **R2:** Propose is on the protocol, the Runtime, `Realize` and `Host`; the activity Realization proposes start, cancel and worker responses from enabled external actions with inputs filled from the world; same state and seed give the same proposal; wire goldens pin the method. Errors: quiescent Model refuses with `QUIESCENT`; a pending proposal refuses a second Propose with `STATE`; a proposal naming a disabled action is a Runtime defect caught by a negative control.
- **R3:** The Go `runner` package runs the one loop with the five modes and writes a `Run`; one Host per Run; callbacks serialized; the `umpire` command replaces run, fuzz, replay, assess and repeat and those commands no longer exist. Errors: Unexplained ends the Run with the event index; DriverFault and Refusal end it Refused; exit codes as specified.
- **R4:** The Temporal Driver acts and observes over the generated API with poll, timeout, backoff, capture and redaction as mechanics; the live activity slice runs through `umpire explore` end to end (start, retry, completion, cancellation, host refusal) with the Runner's Run as the only artifact. Errors: a read timeout without Model acceptance is a DriverFault event judged Refused; malformed evidence is refused before Judge.
- **R5:** A Shim endpoint on the Runner forwards Answer to one non-Go SDK worker with protocol versioning; a control proves result, failure, heartbeat and delay forwarding and the unknown-version refusal. Errors: unreachable endpoint and callback timeout are DriverFault events.
- **R6:** The listed packages are deleted, `ir` and `execution` shrunk, generated Case trees and the seven legacy failing tests removed; the Testpilot umbrella is green; sizes reported by package meet at most 5,000 generic and 11,000 total. Errors: a count over target fails the task unless the owner grants an exception in writing.
- **R7:** DESIGN.md gains "Runner and Run" recording the vocabulary sentence, the protocol addition, the Run schema, the per-package line counts before and after, and timings; the R33 to R40 table rows become discharged with pointers. No error surface beyond the gate.

## Boundaries

- No new Model, no Model semantics change, no DSL change beyond what fn-227 delivered.
- No Component Model, no wasmtime for Go, no cgo.
- No remote or multi-host Runner; one process, one Host per Run.
- No Workflow, Nexus or Schedule Drivers; the Temporal Driver covers the activity slice only. Other Temporal primitives arrive with their Models.
- No distribution of prebuilt wasm and no extraction per MIGRATE.md; MIGRATE.md's path table is updated for the renames only.
- No reintroduction of Case identity, Program, Contract, profiles, receipts or provenance under another name.

## Decision Context

2026-10-20, owner: the Go side should be dumb and the Model should contain the smarts, consulting the wasm Model from Go. That is R33 to R40 as recorded; this spec files them as one spec, adds the missing Propose question, and names the pieces. The owner asked whether "Case slice" is still the right name; it is not, since Case is the artifact this work deletes, so the spec is named after the Runner it builds and the Run it produces. The fn-210 open owner item "replace generated Case trees with Model questions" is decided here in the affirmative.

Names, settled with the owner on 2026-10-20: keep the `umpire-*` crates, `umpire.model.v1`, the five existing verbs and their nouns, `Host`, guest and host; rename `umpire-wasm` to `umpire-runtime` and `umpire-tests` to `umpire-fixtures`; Realization is the one word for a binding; `Propose`, `runner`, `Runner`, `Run`, `Driver`, `Shim`. Vocabulary: the Model is the umpire; the Runtime hosts one Model; a Realization binds it to one system; the Runner drives a Driver, asks the Model to Propose, Judge and Answer, and records a Run.

Placement: after fn-227 (Propose fills requests from the world and action inputs). The order relative to fn-211 is the owner's; nothing here depends on Apalache.

Rejected: keeping Go choosers for actions (the Model would be an oracle with no voice in exploration, and Testpilot's IR interpreter would survive); a `Next` or `Drive` verb (Propose matches the Model's role of proposing, Go's of acting); a separate Rust host-loop crate compiled to wasm (the loop owns sockets, clocks and SDK workers, which belong in Go); keeping `campaign` and `replay` as packages under new names (both become modes of one loop).
