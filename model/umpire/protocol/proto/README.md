# Model ABI v1

The generated protobuf messages in `umpire/model/v1/model.proto` are the wire contract.
Method enum numbers are stable dispatch IDs. `umpire_call` always returns a `Response` envelope.
Explore replaces the current exploration only on success; Rows needs a successful Explore.
Each rows string is one complete native golden line, including its newline. States and actions
use Display; domain outcomes and facts use Debug.

Reset (3), Judge (4) and Answer (5) carry the core run protocol. Judge accepts an opaque
observation Any and returns either a present Explanation (including zero steps) or
Unexplained with expected actions. Answer accepts an opaque callback Any and returns an
opaque reply Any. `umpire.realization.v1` supplies Observation, Error, Callback and Reply envelopes.
Observation retains optional request Any at tag 4 and unchanged absent-request bytes.
System payload registries own type-URL admission; the old namespace
is not an alias for the new one. The Kettle guest currently refuses these three operations
with Unsupported (8). The export-free `umpire-runtime` library supplies the transactional
runtime and ABI; system guests supply a `Machine` and `Realize` realization. Reset restores
the unique start and clears frozen evidence. Judge replays actual Model steps; the
Explanation lists Debug-formatted outcomes in parallel with its steps, including `()`
for unit outcomes. The realization matches the final replayed outcome; a zero-step current
check has no outcome. The first Unexplained freezes advancement until Reset. Answer validates
delivery and its enabled response action before committing. Failed response publication commits nothing.
`Any::pack`/`unpack` use generated `prost::Name` canonical URLs and typed Refusal
constructors; the registry remains system-owned. Existing method IDs and old wire bytes
remain pinned by host tests.

Annotate (7) carries an opaque input Any and returns an opaque output Any at
Response tag 8. Shared `umpire.annotations.v1` messages carry annotation requests,
results and probe metadata; system probe controls remain system-owned.
`AnnotationRuntime` dispatches this stateless operation independently of Judge's frozen run.
Response tag 7 stays reserved; Propose uses fresh tag 9.

The wasm32-wasip1 guest directly exports application functions
`umpire_buffer_alloc(u32) -> u32`, `umpire_buffer_release(u32, u32)` and
`umpire_call(u32, u32, u32) -> u64`, plus memory and toolchain lifecycle exports.
The packed result is `(length << 32) | pointer`. `umpire_buffer_alloc(0)` returns zero;
`umpire_buffer_release(0, 0)` succeeds. Other allocations are nonzero, initialized,
guest-owned buffers.
The guest bounds individual buffers to 1 MiB and total live storage to 16 MiB.
Allocation exhaustion returns zero; invalid frees trap without dereferencing the supplied pointer.
`umpire_call` accepts only an exact live allocation range, with `(0, 0)` representing an empty request.
It retains the request and returns a separately owned response allocation, or zero on exhaustion.
Unknown methods, malformed requests and invalid checker queries return typed Refusal envelopes.
`REFUSAL_CODE_SCOPE_ACTION` (7) covers invalid action admission, including unknown catalog
actions and explicit restrictions that admit no actions. The message distinguishes these refusals.
`REFUSAL_CODE_MALFORMED_REQUEST` (2) also covers invalid progress bounds and native registered
progress query misuse. Diagnostics distinguish a zero bound, an unknown or ambiguous retained
registration, and an action catalog whose ordered values differ from exploration. These are
invalid checker requests, distinct from action-admission refusals (7) and state budgets (3).
This mapping adds no wire method; registered progress queries remain a native checker operation.
The host copies response bytes before freeing either allocation, then frees both exactly once.
Access to one instance is serialized. An unexpected trap invalidates the host instance.

Rust generates into Cargo's OUT_DIR with prost-build 0.14 and mise's protoc 29.5.
Standard well-known types are not generated: prost-build's default mapping uses `prost-types`,
which `umpire_runtime::prost_types` re-exports for realizations and generated code. Strict
`Any` admission stays in `umpire_runtime::AnyMessage`.
Go uses the repository's pinned protoc-gen-go v1.36.6 and protobuf runtime v1.36.11.
The separate Go host module and generated Go messages live at `../hosts/go/`;
regenerate from the repository root with `bash model/scripts/proto-go.sh`.
Use `--check` for byte-for-byte freshness; `make model-check` provisions the pinned
generator and runs freshness, native controls, both guests and both Go modules.
The system extension and its reproduction commands are documented in `model/README.md`
relative to the server root; its upstream schemas never become generic protocol inputs.

The sibling `umpire.realization.v1` package owns Observation, Error, Callback and
Reply. Observation carries its RPC method, one response or error, and an optional
actual-request Any at tag 4; absent requests retain their original encoding.
Callback carries a task Any and Reply a request Any. `umpire.annotations.v1`
owns stateless annotation and generic probe metadata, results and recipe transport.
System-specific controls remain in their owner's protocol. Regenerate these Go
messages together with the core contract using `bash model/scripts/proto-go.sh`.

Propose (6) returns a deterministic fully filled request and its typed kind at Response
tag 9: ACTION (1) names an enabled external Model action; BOOTSTRAP (2) has an empty
action and admits a fresh run outside the transition graph; READ (3) has an empty action
and observes worker progress without proposing a Model action. Consumers refuse unspecified
or unknown kinds. QUIESCENT (9) means there is neither a read nor an enabled external action;
rejected self-loops are still enabled actions. Propose never steps the Model. A second
proposal while pending refuses STATE. Internal actions cannot be proposed.

Realize defaults to unsupported programs, answers and proposals, no bootstrap and final observations. Opted-in
realizations correlate actual observations with the pending request; a final Explained
clears pending, an intermediate Explained publishes continue_polling (Judge tag 3), and
Unexplained freezes without continuing. The continuation hook sees the prospective state
and typed pending action. The continuation hook decides when a correlated Read finishes;
intermediate progress commits while the Read remains pending. Callbacks may commit delivery between polls without clearing it.
Bootstrap requires a positively matched current start, never a replayed transition.
All pending/admission/state changes share response-publication transactions. Reset clears
pending and frozen state and, for bootstrap realizations, admission.

ResetRequest.configuration (Any, tag 1) is an optional system-owned configuration handoff;
absent fields preserve the old empty Reset wire bytes. The core runtime refuses unknown
configuration; composing system dispatchers own its validation and transactional install.
ResetResponse.program (Any, tag 1) optionally carries the realization's World-bound worker
program; Unsupported from the program hook means no program, while other refusals prevent Reset.
The host retains this opaque value without decoding its system-owned contents.
Replay must retain this exact Reset call and response. Existing wire vectors remain unchanged. The new
Judge response with both a judgment and continue_polling has separate literal Rust/Go
vectors: protobuf permits field ordering differences (prost writes the oneof first; Go
writes the scalar first). Both decode to the same message.

`umpire.run.v1.Run` (`umpire/run/v1/run.proto`) is the Go Runner's record of one Run: model
hash, protocol version, world, seed, ordered events and verdict. Each event embeds the exact
`umpire.model.v1` request and response, or the refused request bytes and Refusal, so a Run
replays against a fresh instance without a Driver. It is generated for Go only; no guest
reads it. Its canonical bytes are Go's deterministic encoding, and its store key carries
their SHA-256 outside the payload.
