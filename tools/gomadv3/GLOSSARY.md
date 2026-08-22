# Gomad v3 Glossary

## How it works

The **Gomad Toolchain** runs a **Target** under a patched Go runtime. The
**Runner** prepares that Target once, then a **Campaign** launches a fresh
**Execution** for each selected **Seed** or **Choice Replay Plan**. Each
Execution produces a **Choice Trace**, runs under a **Deterministic I/O
Contract**, and records explicit external events from the **World Model**.

The Runner validates those observations into an **Execution Record** and
publishes an immutable **Artifact**. **Replay** verifies the same behavior, the
**Corpus** retains useful verified Artifacts for future Campaigns, and a
**Qualification Set** measures what a specific toolchain and platform support.

## Language

**Gomad Toolchain** — The pinned Go toolchain whose runtime patch and source
overlay make supported runtime choices, time, and host I/O reproducible.

**Runner** — The host orchestrator that prepares Targets, supervises Executions,
enforces wall limits, and publishes evidence.

**Target** — One reviewed, immutable executable and its identity and provenance.
It is prepared once and may be executed many times.

**Campaign** — One immutable exploration plan and all Executions selected by it.
This replaces “batch.”

**Execution** — One launch of a prepared Target with one Seed or replay plan.
This replaces “run.”

**Seed** — A stable input that selects runtime Choices for an otherwise
unchanged Target, toolchain, and configuration.

**Choice** — One runtime-controlled selection among eligible alternatives, such
as a runnable goroutine or `select` candidate.

**Choice Trace** — The bounded, validated sequence of Choices observed during an
Execution.

**Choice Replay Plan** — An identity-bound sequence of expected Choices that is
validated before each recorded selection is applied.

**Deterministic I/O Contract** — The versioned identity, supported operations,
adapters, limits, and transcript rules for transparent deterministic I/O.

**Captured Read-Only Input** — Host file or directory data imported on demand,
stored in the Artifact, and reused by Replay without reopening the host path.

**World Model** — The pure in-memory model for explicit external events,
ordering, logical time, snapshots, and replay.

**Cluster** — The application-facing contract for starting, waiting for,
stopping, crashing, and restarting configured Nodes. SIM-0 defines this seam;
SIM-1 will provide its first execution backend.

**Node** — One stable logical service identity, configured address, boot
identity, and volume attachments within a Cluster.

**Incarnation** — One specific execution of a Node. Restart preserves the Node
identity and creates a monotonically increasing incarnation handle so stale
work can be rejected.

**Boot Identity** — A stable identifier registered to an application boot
function. Specs record the identity rather than a process-local function
pointer.

**Simulation Backend** — The execution mechanism behind the Cluster contract.
The declared values are `in_process` and `process`; neither is implemented by
SIM-0.

**Simulation Model Fidelity** — A claim that detached modeled transitions and
outcomes satisfy the declared simulation contract. It is not a hard cleanup or
fresh-global claim.

**Hard Isolation Fidelity** — A process-backend-only claim that an incarnation
has process-level cleanup and fresh runtime state.

**Parity Case** — One named v2-derived expected v3 behavior with exact source
references, an explicit preserve-or-replace decision, delivery stage, and
required backend/fidelity evidence.

**Execution Record** — The canonical, versioned evidence describing one
Execution, its identities, inputs, observations, and outcome.

**Artifact** — An immutable, content-addressed directory containing one Execution
Record and its bounded replay payloads.

**Replay** — Validation and re-execution of an Artifact against its recorded
Target, Choices, deterministic I/O, and World inputs.

**Corpus** — The bounded set of verified, interesting Artifacts used to guide
later Campaigns toward semantically novel behavior.

**Qualification Set** — A versioned set of workloads, expectations, and
limits used to measure support for one exact toolchain and platform.
