# Spike: the Scala checker as a portable binary (GraalVM native-image, Scala Native, WebAssembly)

## Conversation Evidence

> user (2026-10-18): "if we had a GraalVM native image; would we still need to have any model evaluation/verification (via CEL etc) in the testpilot IR at all? we could just run the model itself, no?"
> user (2026-10-18): "for the graal-spike; can we also include scala-native if that makes sense?"
> user (2026-10-18): "is WASM another potential option to make the model portable? should we add it to the spike?"
> user (2026-10-18): "the simulation itself will be in webassembly" (host: wasmtime / wasmtime-go)
> user (2026-10-18): "if we have a sidecar, why wouldn't we want the Go part to be as small as possible; really just an adapter; that feeds all the observations into the model? that would allow for maximum re-use and abstraction, no?"

## Goal & Context

If the Scala checker runs as a native binary beside the Testpilot Driver, the Driver can ask it directly whether an observed step is explained and what the claims say. The Case then needs no verification content at all: no Contract clauses lowered from Properties (fn-203 R3), no conformance descriptor (fn-204), and no bound on passive observation. The cost is that a Case is no longer self-contained; running it needs the oracle binary of the matching Model version. Whether that trade is available depends on one engineering question: can the checker be built as a native binary at all. This spike answers it for three targets, GraalVM native-image, Scala Native and WebAssembly, with a go/no-go that re-cuts fn-203.3 and fn-204 on go. It is filed before fn-203 so the hard work follows the decision.

## Architecture & Data Models

- The known blocker on both targets is the claim-section rewriter, which rewrites bytecode at class-load time to give claim blocks their sectioned evaluation. GraalVM forbids load-time rewriting unless it moves to build time; Scala Native has no bytecode. The first task asks whether that rewriting can be a compile-time macro or a plugin phase, which would open both targets.
- GraalVM route: build the checker plus one sample suite with native-image, supply reachability metadata for any `java.lang.reflect` use, and measure binary size, startup and a Bank `verifyAll` against the JVM numbers in the measurement note.
- Scala Native route: cross-build the checker and one sample with Scala Native, with ScalaPB runtime's native artifact, replacing any reflection or bytecode dependency; same measurements.
- WebAssembly route: build the checker and one sample with the Scala.js WebAssembly backend (WasmGC plus exception handling), with ScalaPB's Scala.js artifact, and load the module from a Go host (wasmtime-go; wazero once it supports the GC proposal) answering `rows(state)` for one sample. This is the only target that keeps a Case self-contained: `model.wasm` ships beside the Case and `tools/testpilot` embeds it, with no sidecar process and no bridge. TeaVM over JVM bytecode is a fallback probe only; Scala Native to wasm32 is not attempted.
- Target end state on go (owner, 2026-10-18): Go is act-and-observe only, Scala is decide-and-judge. The adapter protocol is the fuzz bridge generalized to every run: the Model side issues one instruction at a time (`next`), the Go driver executes it against the real system and returns observations with their correlation keys (`observe`), and the Model side judges. No Program interpreter, Contract evaluation, evidence policy or assessment remains in Go; `common/testing/testpilot` shrinks to the driver and the adapter, and another system is a new driver. The Case survives as a frozen export for CI and regression. The spike measures each target against this end state, including the per-step round-trip cost (sidecar) versus in-process calls (wasm), how the live Go tests and the canary launch and pin the oracle, and where the driver's read-completion predicates evaluate without a round trip.
- Go/no-go: targets rank on one axis, embedded (wasm) over sidecar (Scala Native or native-image) over descriptor (status quo). Go on any target means fn-203.3 is closed as unnecessary, fn-203 shrinks to the Program export and the realization adapter, fn-204 is recut to the adapter protocol with the descriptor route recorded as the rejected alternative, and fn-202's generic/Temporal split becomes adapter protocol plus Temporal driver. No-go on both means the descriptor route stands and this spec closes with the evidence.

## API Contracts

- A design note, `model/design/native-checker-spike.md`, records per target: build recipe, blockers found and how each was resolved or not, binary size, startup time, Bank `verifyAll` wall time, and the recommendation.
- The spike produces no production code; a macro or plugin replacement for the rewriter, if proven, is filed as its own task on fn-196 or fn-199 before any target is adopted.

## Edge Cases & Constraints

- Measurements use the measurement note's worlds and flags so they compare with the JVM rows.
- A target that builds but refuses a sample Model is a no-go for that target, with the refusal recorded.
- Nothing is downloaded into the user's home; toolchains are pinned through mise or the framework build directory.

## Acceptance Criteria

- **R1:** The rewriter question is answered with evidence: either a working compile-time replacement for the claim-section rewriting on one sample, or a recorded reason it cannot be done. Errors: none beyond the recorded outcome.
- **R2:** A GraalVM native-image build of the checker with one sample either runs `verifyAll` with the JVM's verdicts and recorded measurements, or the blocking failure is recorded. No error surface beyond that.
- **R3:** A Scala Native build of the checker with one sample either runs `verifyAll` with the JVM's verdicts and recorded measurements, or the blocking failure is recorded. No error surface beyond that.
- **R3a:** A WebAssembly build of the checker with one sample either answers `rows(state)` from a Go host with the JVM's verdicts and recorded module size, load time and call latency, or the blocking failure (host shim, GC support, rewriter) is recorded. No error surface beyond that.
- **R3b:** (2026-10-18, the simulation is WebAssembly) The module loads in wasmtime via wasmtime-go beside a second module in the same store, with the GC and exception-handling proposals enabled; its state snapshots and restores with the store (memory and GC heap) and answers identically after restore; per-call latency is measured at exploration scale (10^5 calls). Errors: a host that cannot enable GC or exceptions is recorded as the blocking failure, never worked around with a non-Scala build.
- **R4:** The design note states go or no-go per target, ranked embedded over sidecar over descriptor, and one recommendation; on go, fn-203.3 and fn-204 are recut as described and MILESTONES' Phase 5 row names the oracle-sidecar route. No error surface.

## Boundaries

- No oracle protocol implementation; the sketch is recorded only.
- No change to fn-203 or fn-204 before the decision.
- No production build target is added to the framework check.

## Decision Context

2026-10-18, phase renumbering: the owner inserted Phase 3 "Temporal Models native" (re-enable and check `model/temporal` against the final DSL) between DSL completion and the IR batch; the IR batch is Phase 4 and the one-language series (fn-199 through fn-209) is Phase 5. This spec runs in Phase 5 unless its Decision Context says otherwise.

The descriptor route (fn-204) was designed so that conformance runs without a JVM. A native binary would make the Model itself available without a JVM, which removes two of the hardest pieces of the series and the passive-observation bound at once. The three targets are tried together because they fail on different things: native-image on load-time rewriting and reflection, Scala Native on the absence of bytecode and reflection altogether, WebAssembly on the host runtime's GC support and the Scala.js host shim. WebAssembly is the highest payoff, since an embedded module restores self-contained Cases, and the lowest prior of success; the owner added it on 2026-10-18. If the rewriter can become a macro, Scala Native is the better sidecar (smaller, faster start, no closed-world configuration); if not, native-image with a build-time rewrite is the only option. Filed 2026-10-18 at the owner's request, before fn-203, so the decision precedes the work. Depends on fn-196's native checker as it stands; fn-203 and fn-204 depend on this spec.

2026-10-18, naming (owner question): Testpilot stays the runtime's name while the Case is the artifact and the Go side executes a Program; `umpire.testpilot.v1` already states it is a component of Umpire. On a go here, where Go becomes act-and-observe and the Model decides each step, the name is revisited in .4 with "driver" as the candidate (`tools/driver`, `common/testing/driver`: the adapter protocol plus one driver per system), since the Go side would no longer pilot anything.

2026-10-18, deterministic simulation (owner): the deterministic Go runtime (SCALA.md: one Go process holding runtime, Temporal and model state so a fork() checkpoint captures everything, no cross-language RPC in the exploration loop) is a future consumer of the oracle. Its needs are the spike's ranking stated precisely: the Model must be callable in-process at exploration rates and its state must snapshot and restore with the runtime's checkpoint. The WebAssembly target satisfies both (linear memory copies with the process); a sidecar process satisfies neither in a fork()-based loop; the descriptor satisfies both for bounded exploration. Add to R3a's measurements: snapshot/restore of the module's state and per-call latency at exploration scale.

2026-10-18, the simulation is WebAssembly (owner; host wasmtime / wasmtime-go): WebAssembly is no longer the highest-payoff candidate but the required target, because the Model must live in the same wasmtime store as the simulated system to snapshot together and be called in-host. The wasm task (.5) runs first after the rewriter question (.1); native-image (.2) and Scala Native (.3) are fallbacks for the live-deployment sidecar only and no longer gate the decision. The host binds the toolchain: the Scala.js WebAssembly backend emits WasmGC with exception handling, which wasmtime implements and wazero does not; the simulation's host choice is therefore also the Model's. The adapter protocol stays act-and-observe over pure data and becomes an in-host interface (host functions or the component model) in the simulation, NDJSON only for the live Testpilot runtime.

