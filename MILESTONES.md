# Milestones

The owner's overview of what is open, in what order, and why. One page, describing the present.

As of 2026-10-20.

## Keeping this page current

- This page describes the present. Rewrite a status in place; do not append dated entries.
- List each open spec's remaining tasks in a `Name | Status | What` table: the task ID, its status
  (🔄 in progress, ⬜ todo, ⏸️ deferred) and a brief description. Remove a task's row once it is done,
  and the spec once it is closed. Flow and git keep the history.
- Only the root agent edits this page. Workers report status in their handoffs.
- Update the "As of" date with every edit.
- Task/spec branch integration follows `AGENTS.md`'s Local milestone execution policy.

## Direction

On 2026-10-18 the owner paused everything else for one question: should Umpire be built in Rust?
The Scala framework is frozen under `model/archived/` for reference and is not expected to build.
Earlier Scala phases and the one-language series remain deferred in Flow and retain paths relative
to the archived tree. The Rust follow-ups below use the owner's priority order; their individual
readiness gates and dependencies still apply.

Execution resumed under the owner's active milestone-completion goal (2026-10-20).
fn-213, fn-215, fn-223 and fn-224 are complete locally with their completion reviews.
fn-225 is complete locally: its single task passed implementation review, correctness and
standards audits, and default/required-TLC gates with byte-identical goldens. fn-226 is also
complete locally with all three search requirements covered, reviews passed and model gates green.
fn-218 is complete locally with all five requirements covered, real-model expansions,
default/required-TLC gates and the completion review passed.
fn-229 is complete locally with both tasks, all three requirements, implementation
and completion reviews, correctness/standards audits and integrated/live gates passed.
TLC is implemented for its admitted source subset; unsupported source,
including the current Phased/reject activity Model, is explicitly refused.

Umpire in Rust lives in `temporal/model`, a Cargo workspace pinned through `mise.toml`
(`rust = "1.99.1"`), with crates under `model/umpire/`: `dsl` (with `dsl/macros`), `checker`,
`testing` (the dev-only fixture runner), `tests` (the shared end-to-end fixtures), `wasm`
(the generated-protobuf guest), and `lift` plus `backends/tlc` (the independent source backend);
the `samples` crate has been removed. The original Rust verdict was scoped to what
`model/archived/framework/samples/Kettle.scala` uses. The verdict is one rule (R24): if the checker
runs as wasm in a Go host through wazero (pure Go, no cgo) on the stock toolchain, with Kettle's report matching
the Rust goldens byte for byte, Rust wins. Everything else in `model/DESIGN.md` is information. fn-196.8, the Scala checker speed fix, stays
deferred, since speed no longer decides. The decisive slice passed in fn-210.9: the recommendation
is Rust. The guest now directly exports `umpire_buffer_alloc`, `umpire_buffer_release` and
`umpire_call`, with no custom linker or export rewriting (fn-210.16).

| Phase | Specs and tasks | Scope |
| --- | --- | --- |
| After the Rust verdict | fn-217 → fn-230 → fn-231 → fn-227 → fn-211 → fn-228 | Owner priority order (fn-231 after fn-217 and fn-229, before fn-228.2 if possible; fn-229 is complete; fn-230 has no dependency and must land before fn-228.4; fn-227 placed before fn-211 on 2026-10-20 so Apalache arrives to a parameterized Model; fn-228 depends on fn-227 only, its place relative to fn-211 is the owner's). fn-210, fn-212, fn-213, fn-214, fn-215, fn-216, fn-218, fn-221, fn-223, fn-224, fn-225, fn-226 and fn-229 are complete locally with their scoped reviews and gates passed; fn-225 and fn-226's single-task reviews cover their full specs. fn-216's broad umbrella retains seven documented pre-existing failures. The recommendation is Rust under stock WASM/Go parity alone. Preserve each spec's prerequisites; priority does not add dependency edges or mark deferred work ready. fn-212 supplies lifting and TLC for its admitted source subset; fn-211 adds Apalache over its emitter. The former "Case slice" is filed as fn-228 (Runner and Run): the Model as umpire, one Model per Run, Propose on the protocol, one Go host loop, Driver and Shim, observations plus judgment and the never-overwrite publish store. Targets are at most 5,000 generic-runtime and 11,000 total non-test Testpilot Go lines. Standalone extraction remains a plan in `MIGRATE.md`. |

## Specs

### fn-217: Annotation catalog over the Temporal API: kinds, tags, rules, expected errors and gap analysis (model, temporal)

Captured on 2026-10-19, after fn-216. A typed sidecar catalog over the upstream `temporal.api`
messages (one kind per field, any tags with consequences, JSON Schema presence rules, AIP
`field_behavior` roles, protovalidate constraint names), exhaustive by construction at four levels
(fields, enum values, RPC methods, Model actions) so an `api.binpb` bump fails the build until
classified; every request-side rule maps to the server's expected error and a probe produces
`gaps.txt` (`Confirmed`/`Mismatch`/`Accepted`/`Unverifiable`) plus reverse coverage of the server's
validators. Generic vocabulary in `umpire-annotations`, Temporal tags in `model/temporal`.
[Spec](.flow/specs/fn-217-annotation-catalog-over-the-temporal.md).

| Name | Status | What |
| --- | --- | --- |
| fn-217.3 | 🔄 in progress | Mini activity catalog with expected errors; actual SDK transport `fill`/`validate` in the slice test (R3, R5) |
| fn-217.4 | ⬜ todo | After .3: gap probe and reverse coverage; `gaps.txt` golden (R3, R4) |
| fn-217.5 | ⬜ todo | After .4: DESIGN.md write-up (R5) |

### fn-230: Reproducible guest builds: a system-agnostic wasm build step, the committed Temporal guest, and the freshness gate (model, temporal)

Captured on 2026-10-20 from the owner's decision that the built guest is checked in so test runners
need Go and no Rust. Umpire gains a `guest` Cargo profile and `guest.sh` producing byte-identical
wasm from any checkout, proven on two; the Temporal guest is committed beside its Go host module and
embedded with `go:embed`; `make model-check` rebuilds and byte-compares it like the protobuf
freshness checks, and `make umpire-guest` refreshes it. Umpire's own guests stay uncommitted.
[Spec](.flow/specs/fn-230-reproducible-guest-builds-a-system.md).

| Name | Status | What |
| --- | --- | --- |
| fn-230.1 | 🔄 in progress | Linux source checkpoint integrated and combined model gate passed; two-checkout proof verified; actual unwinding and macOS/cross-OS proof remain open (R1) |
| fn-230.2 | ⬜ todo | After .1: committed Temporal guest with `go:embed`; freshness step; `umpire-guest`; docs (R2, R3, R4) |

### fn-231: Declarative Realizations: the #[realization] attribute surface and the coverage golden (model, temporal)

Captured on 2026-10-20 after fn-217 and fn-229. A Realization becomes annotated handlers,
`#[observe(Service::Method, reads(..), ignores(..))]` and `#[answer(..)]` under `#[realization(activity)]`,
with dispatch, unpacking and `Unsupported` refusals generated over fn-229's helpers; `#[propose]` is
reserved for fn-228.2. Coverage is a golden, `realization.snap.txt`, classifying every catalog field of each
handled RPC as mapped, ignored with reason or unsupported, plus unhandled RPCs, checked against fn-217's
catalog at build time. Attributes live on the Realization only; the Model names nothing from Temporal.
[Spec](.flow/specs/fn-231-declarative-realizations-the.md).

| Name | Status | What |
| --- | --- | --- |
| fn-231.1 | ⬜ todo | Realization macro crate, `Step`/`Reply`, generated `Realize`; activity rewritten with expansion snapshot (R1, R3, R4) |
| fn-231.2 | ⬜ todo | After .1: coverage golden against the catalog; unknown paths refused; DESIGN.md (R2, R4) |

### fn-227: Parameterized Models: declared worlds, action inputs, and the activity Model with retries and timeouts (model, temporal)

Captured on 2026-10-20, after fn-218 and fn-217 and before fn-211. Closes the DSL gap DESIGN.md's
readiness table leaves open: `#[derive(World)]` with `&World` access on handlers and claims, a
`#[starts]` block for computed starts, `Finite` and action inputs with `#[domain]`, and lifting of
worlds as `CONSTANT`s and inputs as finite choice. Delivered through the activity Model extended
in place: `maximum_attempts` and `RespondFailed { retryable }`, then the four optional timeouts,
`Heartbeat`, `Tick` and asynchronous cancel with progress and bound claims; the `maximum_attempts = 1`
world reproduces the six-phase goldens. The realization reads the world from the observed start
request and the live slice proves one retry then completion. The lifter admits `impl Phased` and
`reject!` so TLC checks the activity Model.
[Spec](.flow/specs/fn-227-parameterized-models-declared-worlds.md).

| Name | Status | What |
| --- | --- | --- |
| fn-227.1 | ⬜ todo | Declared World, `&World` access, `#[starts]` block; `parameterized_world` fixture with TLC (R1, R3) |
| fn-227.2 | ⬜ todo | After .1: `Finite` and action inputs; `action_inputs` fixture with TLC (R2, R3) |
| fn-227.3 | ⬜ todo | After .2: lifter admits `Phased`/`reject!`; activity `maximum_attempts` and retryable failure (R3, R4) |
| fn-227.4 | ⬜ todo | After .3: timeouts, Heartbeat, Tick, asynchronous cancel; progress and bound claims (R5) |
| fn-227.5 | ⬜ todo | After .4: world from observed start options; Answer retryable; live retry proof (R6) |
| fn-227.6 | ⬜ todo | After .5: DESIGN.md write-up and readiness table closure (R7) |

### fn-211: Apalache backend for Umpire in Rust (model)

Captured on 2026-10-18 as a follow-up, after fn-210, fn-212 and (since 2026-10-20) fn-227, and only if their verdicts are to continue in
Rust. Adds Apalache as a second TLA+ backend over the same lifted tree and emitter: Snowcat type
annotations in the emitter, a pinned Apalache release run on the mise JDK with one bounded check
per invariant at the native depth plus one, ITF counterexamples decoded through `Spell` and
replayed through the Model, and an Apalache consumer selected by `expected/apalache.txt`. No
liveness through Apalache; TLC owns progress claims.
[Spec](.flow/specs/fn-211-apalache-backend-for-umpire-in-rust.md).

| Name | Status | What |
| --- | --- | --- |
| fn-211.1 | ⬜ todo | Snowcat type annotations in the TLA+ emitter; TLC results unchanged (R1) |
| fn-211.2 | ⬜ todo | After .1: `umpire-apalache` in `umpire/backends/apalache/`, pinned release, bounded check per invariant, deterministic lifecycle (R2) |
| fn-211.3 | ⬜ todo | After .2: ITF counterexamples decoded through `Spell` and replayed through the Model (R3) |
| fn-211.4 | ⬜ todo | After .3: Apalache consumer; Kettle's `apalache.txt` agreement golden (R4, R5) |

### fn-228: Runner and Run: the one Go host loop over the Model, Propose on the protocol, and the Testpilot shrink (model, temporal)

Captured on 2026-10-20, after fn-227. Files the post-verdict obligations R33 to R40 as one spec under
the owner's direction that Go is dumb and the Model holds the smarts, and retires the "Case slice"
name. Names settled: `umpire-wasm` becomes `umpire-runtime`, `umpire-tests` becomes `umpire-fixtures`,
Realization is the one word for a binding; new `Propose` on `umpire.model.v1`; Go package `runner`
with `Runner`, `Run`, `Driver` and `Shim`. The Model is the umpire; the Runtime hosts one Model; a
Realization binds it to one system; the Runner drives a Driver, asks the Model to Propose, Judge
and Answer, and records a Run. Deletes verification, evaluation, CEL, contract, casefile, campaign
and replay; one `umpire` command with five modes.
[Spec](.flow/specs/fn-228-runner-and-run-the-one-go-host-loop.md).

| Name | Status | What |
| --- | --- | --- |
| fn-228.1 | ⬜ todo | Renames alone: `umpire-runtime`, `umpire-fixtures`, Realization in prose (R1) |
| fn-228.2 | ⬜ todo | After .1: `Propose` on protocol, Runtime, `Realize`, `Host`; activity Realization proposes (R2) |
| fn-228.3 | ⬜ todo | After .2: Go `runner`: `Runner`, `Run`, `Driver`, five modes, one `umpire` command, publish (R3) |
| fn-228.4 | ⬜ todo | After .3: Temporal Driver as act-and-observe; live slice through `umpire explore` (R4) |
| fn-228.5 | ⬜ todo | After .4: Shim endpoint for non-Go SDK workers (R5) |
| fn-228.6 | ⬜ todo | After .4: delete and shrink Testpilot packages; sizes reported (R6) |
| fn-228.7 | ⬜ todo | After .5 and .6: DESIGN.md Runner and Run write-up; R33 to R40 discharged (R7) |

## Open for the owner

- fn-230's caught-panic requirement conflicts with the pinned stock `wasm32-wasip1`
  standard library: the compiler selects `panic=abort` and rejects forced unwinding.
  Approval to preserve existing trap-and-invalidate semantics is requested; the
  requirement has not been waived.
- fn-230's Linux/macOS byte-identity proof needs a reachable macOS executor.
  Two Linux checkouts prove path independence, not cross-OS identity.
