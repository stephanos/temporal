# Gomad: syscall-level I/O boundary, from spike to decision

## Conversation Evidence

> user (turn 17): "why do we need things like /Users/stephan/Workspace/skunkworks/gomad/temporal_wasm/tools/gomad3/deterministicio/grpc_adapter.go ?"
> user (turn 17): "if we replaced the lower level APIs, that wouldn't be needed?"
> user (turn 18): "isn't gosim doing that ie replacing syscalls?"
> user (turn 19): "let's spike it directly; feel free to use a separate git worktree"
> user (turn 19b): "can't we do the spike on darwin/arm64 too?"
> user (turn 20): "write a flow spec to impl those steps"
> user (turn 23, selected): "1 platform is sufficient; use the one you're on at the time"
> user (turn 23, selected): "Add opt-out"
> user (turn 23, selected): "Include multi-node"
> user (turn 23, selected): "Gate the three"

The "steps" in turn 20 are the spike's next steps as the agent reported them:
(1) port the patch onto the Gomad toolchain and run the determinism soak on the
demo (same seed, same transcript); (2) sort the 15 deterministic-I/O adapters
into networking-only, filesystem/`os`, and semantic; (3) decide the boundary
for files and DNS before fn-109 .13–.18 and fn-110 restructure the overlay.
Turn 23 answers the planning questions: platform scope, a per-run adapter
opt-out, multi-node coverage, and which fn-109/fn-110 tasks wait for the
decision.

## Goal & Context

<!-- Goal & Context: 50% [paraphrase], 50% [inferred] -->

Gomad fakes I/O at the level of standard-library calls (`net`, `os`, `time`).
Libraries that reach below that level (raw descriptors, `golang.org/x/sys/unix`)
escape it, so each needs a version-pinned source adapter, and the adapters carry
a large pinning and regeneration toolchain. gosim instead fakes I/O at the
syscall layer, where those libraries are caught without adapters.

A spike (branch `gomad-syscall-spike-20261009`) patched stock Go 1.27.0 with an
in-memory TCP kernel below the `syscall` package and a small runtime poller
hook. Unmodified gRPC v1.83.2 and `net/http` ran over it on darwin/arm64 and on
linux/arm64 (Docker, networking disabled): 50/50 runs each, race-clean, no host
sockets, no gRPC adapter. It did not test determinism, because stock Go has no
seeded scheduler.

This spec turns the spike into a go/no-go decision. It ports the boundary onto
the Gomad toolchain, proves or disproves determinism and replay with it for
single-process programs and multi-node simulations, classifies which adapters
it would retire, and records the decision for files and DNS before overlay
restructuring in fn-109 and fn-110 builds on the current boundary.

## Architecture & Data Models

<!-- Architecture & Data Models: 30% [paraphrase], 70% [inferred] -->

- **Descriptor layer over the existing network models.** Virtual descriptors
  map to connections and listeners of Gomad's existing network models: the
  standalone model for single-process programs, the simulation network for
  multi-node runs (node addresses, incarnations, revocation), and the process
  backend's forwarding to the coordinator. The boundary adds descriptors,
  non-blocking semantics and readiness on top; it does not add a second
  connection model, so fn-154's simplification applies to both boundaries.
- **Syscall edge.** The `syscall` package's typed socket functions and its
  generic entry points route virtual descriptors to the descriptor layer before
  any existing capability guard runs. On linux the generic path dispatches on
  the syscall number; on darwin it identifies the libc function behind a
  trampoline.
- **Runtime hook.** Virtual descriptors skip the host poller. Gomad treats
  host netpoll results as host timing (unseeded batch order, and quiescence
  refuses to advance virtual time while netpoll waiters exist), so virtual
  readiness must never go through a netpoll batch: the goroutine that changes a
  descriptor's state unblocks its waiters directly on the seeded path, and
  waiters on virtual descriptors do not count as netpoll waiters for
  quiescence.
- **Capability admission.** Gomad's compile-time guard denies every exported
  `syscall` and `golang.org/x/sys` entry point in guarded mode, and its closure
  policy rejects third-party packages that import them; together they stop
  unadapted code before it reaches the syscall edge. Under the boundary profile
  only, the socket entry points the edge models are admitted by both layers,
  bound into the profile's identity; everything else stays denied, and with the
  boundary off nothing changes. R8 must weigh whether this admission brings
  back per-library review in another form.
- **Run selection and identity.** The boundary is a per-run choice carried like
  the clock-tick policy (command-line option, recorded environment, workload
  field). It selects a distinct deterministic-I/O profile, so artifact identity
  binds the boundary and replay refuses a mismatch. With the boundary selected,
  the standard-library-level TCP hooks stand aside for both standalone and
  simulation networks, owned by one switch; resolver and interface hooks stay
  at the current level, since DNS is out of scope.
- **Adapter opt-out.** A per-run option excludes chosen adapters from
  preparation, so a workload can be proven without them.

## API Contracts

<!-- API Contracts: 100% [inferred] -->

- The syscall boundary and the adapter opt-out are opt-in per run; with neither
  selected, Gomad behaves as before.
- The recorded artifact identifies the I/O boundary and the excluded adapters;
  replay refuses an artifact whose boundary or adapter set differs.

## Edge Cases & Constraints

<!-- Edge Cases & Constraints: 100% [inferred] -->

- Generic data operations on virtual descriptors must be safe when the caller's
  buffer lives on a goroutine stack that moves; the spike's version was not.
- Generic hooks must not misroute calls whose first argument merely falls in the
  virtual descriptor range, and the range must stay disjoint from Gomad's
  reserved and inherited descriptors.
- Existing capability guards (for example on writes to non-standard
  descriptors) must recognize virtual descriptors first, or every virtual write
  would be denied.
- Non-stream sockets (UDP) and other host sockets under the boundary are refused,
  never silently passed to the host; each refusal is recorded as a transcript
  event and fails the operation with a clear error.
- Calls that bypass the `syscall` package (assembly issuing raw syscalls,
  `x/sys/unix` no-error variants on linux, cgo) remain escapes; a static scan of
  the evaluated workloads' closures lists them in the adapter classification
  report, which R8 cites.
- Linux syscall-number dispatch tests compile on the platform in use but run
  only on linux/amd64; their runtime coverage is deferred to fn-128 when the
  platform in use is darwin/arm64.
- Descriptor-layer state uses deterministic iteration orders and no host time.
- The toolchain must keep building and passing its tiers on both qualified
  platforms (darwin/arm64, linux/amd64); determinism proof is required on one,
  the platform in use. Native proof on the other stays with fn-128 or fn-149.
- Restarted or crashed nodes revoke their virtual descriptors with the same
  semantics as the current simulation network.

## Acceptance Criteria

The count trips the split check, but every criterion serves one outcome, a
go/no-go decision on the syscall boundary, so this stays one spec.

- **R1:** The virtual descriptor layer, syscall edge and runtime hook are carried by the Gomad toolchain for its pinned Go version, opt-in per run; with the boundary off, all existing Gomad test tiers pass unchanged on the platform in use, and the toolchain builds for both qualified platforms. Errors: selecting the boundary where the generic dispatch is unsupported fails the run at startup with a clear error. [paraphrase]
- **R2:** Under Gomad's seeded runtime with the boundary on, a gRPC and HTTP workload with its network adapters excluded (gRPC and the network adapters its dependencies pull in, such as `x/net` and `sockaddr`) produces identical choice traces and I/O transcripts for the same seed across repeated runs, over the existing determinism soak's seeds and repetitions, on one qualified platform (the one in use). Errors: any divergence fails the soak and is reported with the first differing event. [user]
- **R3:** A run recorded with the boundary on replays exactly, and replay rejects an artifact recorded with a different I/O boundary or adapter set. Errors: mismatch → replay refused before execution. [paraphrase]
- **R4:** Generic data operations on virtual descriptors are safe when buffers live on a moving goroutine stack; calls whose first argument is not a virtual descriptor are never routed to the descriptor layer; Gomad's reserved descriptors are never virtual; capability guards admit virtual descriptors. Each is covered by tests. Errors: no error surface beyond R1. [paraphrase]
- **R5:** Each of the 15 deterministic-I/O adapters is classified as networking-only, filesystem/`os`, or semantic, with evidence; for each networking-only adapter, its covered workload runs deterministically with the adapter excluded and the boundary on. Errors: an adapter whose exclusion breaks determinism is reclassified with the observed failure. [paraphrase]
- **R6:** The gRPC adapter's remaining unexercised paths (channelz socket options, the DNS resolver rewrite, disconnect errno classification) are each exercised under the boundary, classified as still needing an adapter, or recorded as not reachable on the platform in use (deferred to fn-128 or fn-149). Errors: no error surface beyond R5. [paraphrase]
- **R7:** The runtime-patch and overlay size with the boundary is measured against the fn-110.1 baseline with its method, and reported. Errors: no error surface. [paraphrase]
- **R8:** A recorded decision states whether Gomad moves files and DNS to the syscall boundary, keeps them at the current level, or abandons the syscall boundary, citing R2–R7 and R10; fn-109.17, fn-109.18 and fn-110.3 wait for it, and fn-109 .13–.16, fn-110 .4–.5 and fn-154 are annotated with its effect. Errors: no error surface. [user]
- **R9:** A per-run option excludes named adapters from preparation; the exclusion is bound into artifact identity and the recorded run. Errors: excluding an unknown adapter → rejected before preparation; a workload that needs an excluded adapter and is not covered by the boundary's capability admission (R11) fails with the existing closure error rather than running unadapted host code. [user]
- **R10:** With the boundary on, multi-node simulations in both the in-process and process backends serve their TCP traffic through virtual descriptors with the existing node addresses and restart semantics, and a multi-node workload passes the same-seed determinism soak of R2. Errors: a descriptor of a crashed or stopped node incarnation fails with the existing revocation outcome; cross-node routing to an unknown address is refused. [user]
- **R11:** Under the boundary profile only, the compile-time capability guard and the closure policy admit the `syscall` and `golang.org/x/sys` socket entry points the syscall edge models, for the standard library and third-party packages; the admission is bound into the profile's identity. Errors: with the boundary off, denial is unchanged; under the boundary, entry points the edge does not model still fail with the existing capability error. [inferred]

## Early proof point

Task fn-155-gomad-syscall-level-io-boundary-from.3 validates the core approach
(a single-process gRPC workload over virtual descriptors, with its network
adapters excluded and capability admission in place, produces byte-identical
transcripts for the same seed on the Gomad runtime, and replays exactly). If it
fails, re-evaluate whether readiness wakeups or capability admission can be
made deterministic and contained before building multi-node support or the
soak.

## Boundaries

- Implementing the file-system or DNS side of the syscall boundary is out of scope; only the decision is in scope. [paraphrase]
- Deleting adapters from the main toolchain is out of scope; R9's opt-out excludes them only per run. [paraphrase]
- No change to the network fault model; fn-154 owns it. [paraphrase]
- Native determinism proof on a second platform is out of scope; it stays with fn-128 (linux/amd64) or fn-149 (darwin/arm64). [user]

## Decision Context

<!-- Decision Context: 60% [paraphrase], 40% [inferred] -->

### Motivation

Per-library adapters exist because Gomad's boundary sits above where libraries
reach the kernel; faking the lower level removes the reason for them.
[paraphrase] gosim demonstrates that design, but it is experimental and built
on source translation, so Gomad adopts the idea, not the code. [paraphrase] The
spike proved networking works without adapters on both darwin and linux; the
open question is determinism, which only the Gomad runtime can answer.
[paraphrase] Overlay restructuring in fn-109 and runtime-patch minimization in
fn-110 would partly be discarded if the boundary moves, so the decision comes
first; only the three tasks that rewrite the same code wait for it. [user]
Building the descriptor layer over Gomad's existing network models, rather than
porting the spike's separate kernel, gives multi-node support (node addresses,
restarts, process backend) without a second connection model. [inferred]
Plan review found that capability guards, not only descriptor routing, decide
whether unadapted libraries can reach the edge, and that host netpoll is
host-timed in Gomad; R11 and the runtime-hook rule above come from that review.
[inferred]
Maintainability (plan review): duplication - the switch that makes std-level network hooks stand aside was decided twice (standalone in .2, simulation in .4) without one owner; structure - `gomadio.NetworkEnabled()` absorbs boundary branching on every net hook. Resolved by giving .2 sole ownership of a split predicate (TCP vs resolver/interfaces) that .4 consumes.

## Quick commands

```bash
make -C tools/gomad3 overlay-test test-toolchain test-simulation
make -C tools/gomad3 test-host
make gomad3-soak
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | The virtual descriptor layer, syscall edge and runtime hook are carried by the Gomad toolchain for its pinned Go version, opt-in per run; with the boundary off, all existing Gomad test tiers pass unchanged on the platform in use, and the toolchain builds for both qualified platforms. Errors: selecting the boundary where the generic dispatch is unsupported fails the run at startup with a clear error. | fn-155-gomad-syscall-level-io-boundary-from.1, fn-155-gomad-syscall-level-io-boundary-from.2 | — |
| R2 | Under Gomad's seeded runtime with the boundary on, a gRPC and HTTP workload with its network adapters excluded (gRPC and the network adapters its dependencies pull in, such as `x/net` and `sockaddr`) produces identical choice traces and I/O transcripts for the same seed across repeated runs, over the existing determinism soak's seeds and repetitions, on one qualified platform (the one in use). Errors: any divergence fails the soak and is reported with the first differing event. | fn-155-gomad-syscall-level-io-boundary-from.3, fn-155-gomad-syscall-level-io-boundary-from.5 | — |
| R3 | A run recorded with the boundary on replays exactly, and replay rejects an artifact recorded with a different I/O boundary or adapter set. Errors: mismatch → replay refused before execution. | fn-155-gomad-syscall-level-io-boundary-from.2, fn-155-gomad-syscall-level-io-boundary-from.3 | — |
| R4 | Generic data operations on virtual descriptors are safe when buffers live on a moving goroutine stack; calls whose first argument is not a virtual descriptor are never routed to the descriptor layer; Gomad's reserved descriptors are never virtual; capability guards admit virtual descriptors. Each is covered by tests. Errors: no error surface beyond R1. | fn-155-gomad-syscall-level-io-boundary-from.1 | — |
| R5 | Each of the 15 deterministic-I/O adapters is classified as networking-only, filesystem/`os`, or semantic, with evidence; for each networking-only adapter, its covered workload runs deterministically with the adapter excluded and the boundary on. Errors: an adapter whose exclusion breaks determinism is reclassified with the observed failure. | fn-155-gomad-syscall-level-io-boundary-from.6 | — |
| R6 | The gRPC adapter's remaining unexercised paths (channelz socket options, the DNS resolver rewrite, disconnect errno classification) are each exercised under the boundary, classified as still needing an adapter, or recorded as not reachable on the platform in use (deferred to fn-128 or fn-149). Errors: no error surface beyond R5. | fn-155-gomad-syscall-level-io-boundary-from.6 | — |
| R7 | The runtime-patch and overlay size with the boundary is measured against the fn-110.1 baseline with its method, and reported. Errors: no error surface. | fn-155-gomad-syscall-level-io-boundary-from.7 | — |
| R8 | A recorded decision states whether Gomad moves files and DNS to the syscall boundary, keeps them at the current level, or abandons the syscall boundary, citing R2–R7 and R10; fn-109.17, fn-109.18 and fn-110.3 wait for it, and fn-109 .13–.16, fn-110 .4–.5 and fn-154 are annotated with its effect. Errors: no error surface. | fn-155-gomad-syscall-level-io-boundary-from.7 | — |
| R9 | A per-run option excludes named adapters from preparation; the exclusion is bound into artifact identity and the recorded run. Errors: excluding an unknown adapter → rejected before preparation; a workload that needs an excluded adapter and is not covered by the boundary's capability admission (R11) fails with the existing closure error rather than running unadapted host code. | fn-155-gomad-syscall-level-io-boundary-from.2 | — |
| R10 | With the boundary on, multi-node simulations in both the in-process and process backends serve their TCP traffic through virtual descriptors with the existing node addresses and restart semantics, and a multi-node workload passes the same-seed determinism soak of R2. Errors: a descriptor of a crashed or stopped node incarnation fails with the existing revocation outcome; cross-node routing to an unknown address is refused. | fn-155-gomad-syscall-level-io-boundary-from.4, fn-155-gomad-syscall-level-io-boundary-from.5 | — |
| R11 | Under the boundary profile only, the compile-time capability guard and the closure policy admit the `syscall` and `golang.org/x/sys` socket entry points the syscall edge models, for the standard library and third-party packages; the admission is bound into the profile's identity. Errors: with the boundary off, denial is unchanged; under the boundary, entry points the edge does not model still fail with the existing capability error. | fn-155-gomad-syscall-level-io-boundary-from.8 | — |
