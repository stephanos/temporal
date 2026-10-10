# Gomad: simpler virtual network with stalling partitions

## Conversation Evidence

> user (turn 13): "what's the beneift of simulating TCP so detailed? wou;dn't we just test the gRPC and HTTP libraries isntead of our code? what's the point?"
> user (turn 14): "yes A"
> user (turn 15): "should we generally simplify that fake TCP impl? it doesn't need to mimic TCP so closly. I'm sure we can stll simulate all faults we need?"
> user (turn 15, selected): "Widen this spec"

Option A held partitioned writes and in-flight bytes, released them in order after heal, and reset both endpoints after a virtual stall timeout. Writes filled a bounded buffer before blocking. The owner then widened the work to simplify the connection model while retaining Gomad's existing faults.

## Goal & Context

Temporal workloads should observe complete ordered streams after heal or a timeout reset. The current model accepts writes across disabled links while discarding their bytes, and lets earlier queued traffic cross a later partition. That behavior can produce stream gaps and framing failures instead of the requested stall-or-reset outcomes.

The rewrite also removes duplicated local connection mechanics, chunk-count admission, delivery-count admission, delivery history and network payload hashes. Both simulation backends use the same local model. Process IPC and virtual descriptors remain adapters. Existing fault selection, bounded resource failures and separate replay evidence retain their owners.

The target applications are unmodified. Developers gain one local read/write/deadline owner. Simulation users gain explicit byte and stall limits, a versioned network-record migration, and a documented reduction in direct network-only payload checking. No production Temporal configuration changes.

## Architecture & Data Models

One local connection module owns each endpoint pair's FIFO byte storage, read/write/deadline/close behavior and nonblocking primitives. Standalone loopback and multi-node execution supply their existing domain checks, topology and recording effects at that seam. The process adapter forwards bounded commands to that owner. The descriptor adapter consumes its nonblocking primitives and readiness notifications. Complete-operation read/write locks, half-close and partial-result semantics remain.

Each direction distinguishes already-received readable bytes from undelivered bytes. A disabled link holds every undelivered suffix, including traffic queued before partition. An elapsed ready time alone does not prove receipt. Already-received bytes remain readable. An enabled reverse link can make progress until the connection pair becomes terminal.

Effective heal ends that direction's continuous stalled interval. Its held FIFO becomes eligible after the link's normal delay, and fresh writes cannot overtake it. Repartition holds any still-undelivered suffix; the next effective heal applies a fresh delay. Redundant heal does not add delay. Redundant disable does not restart a continuous stalled interval.

An interval starts when positive undelivered occupancy first encounters a disabled link and ends at effective heal or removal of all stalled bytes. Idle connections create no stall expiry. The first expiry instant is strictly after the configured limit, at nanosecond resolution `stallStart + stallLimit + 1ns`. Either stalled direction can reset the whole connection pair. Reset frees retained bytes once, wakes both endpoint waiters, and stores a persistent timeout reason. Close, stop, crash, restart and heal invalidate obsolete expiry epochs and incarnations.

Expiry is virtual-time work even after a successful writer has returned and no application operation remains blocked. Native timers and the existing process time/progress arbiter retain ownership. This spec adds neither a scheduler nor a second process response barrier.

## API Contracts

Explicit limits include `NetworkConnectionBytes` per direction and `NetworkStallNanos`. `DefaultLimits()` supplies 4 MiB per direction and 10 virtual seconds. The 4 MiB bound preserves the former 64 times 64 KiB bound in each direction. Ten seconds is a planning choice, not a measured production or TCP timeout. Regression cases use explicit shorter values. The global network-byte default remains unchanged.

Omitted and zero values reject under the new strict configuration format. Reject negative representations, out-of-range values and duration/deadline arithmetic overflow before activation or the affected model mutation. A per-direction byte limit must be positive and no greater than the existing maximum network-byte bound. The stall duration must permit positive signed-duration conversion and the strict one-nanosecond threshold. Admission checks each actual expiry addition too.

Local fullness blocks the writer. Global retained-byte exhaustion remains an immediate resource-exhausted failure. Both budgets charge held data, received unread data and short-read tails until actual consumption or discard. A failed later admission returns the accepted prefix and its error. Listener, connection, history/transition and IPC-frame bounds remain; chunk/delivery counts cease to control payload admission. Empty I/O cannot create an unbounded segment inventory. Queue metadata and expiry work stay bounded by retained positive byte ranges, connection state and existing pending-operation bounds.

The timeout reset has a stable error identity and timeout classification through local calls, net wrappers, process transport and selected descriptors. Clearing an operation deadline cannot undo it. Caller deadline errors remain recoverable. Unknown error-code behavior and other wire errors retain existing precedence.

A pending dial starts a continuous interval on first observing its directed link disabled. Heal ends that interval; a later effective repartition starts another. Dial stall expiry uses the same strict threshold as established connections, independently of the caller deadline/context. It does not create an established connection or a TCP handshake model. Preserve the cancellation capabilities actually supplied by process transport; no new post-IPC cancellation command is implied.

## Ordering, replay and identities

Equal-time actions use existing seeded execution and serialized model order. A heal committed before due expiry invalidates that epoch; expiry committed first permanently resets the pair. At exactly `start + limit`, expiry is not yet due. Test both deterministic event orders at the first due instant.

Preserve operation precedence. Read checks terminal state before readable data, then EOF, then caller deadline. Write checks terminal state before available-capacity admission, then caller deadline. A committed timeout reset wins; existing readable data or available write space can produce progress before a deadline error. Dial checks completed caller context before topology/listener admission.

Network history records operations, held-write outcomes, topology/fault changes and timeout resets. Heal records the semantic topology cause once. Releases create no per-release or per-delivery entries. Remove `partition_drop`, delivery identifiers/counters and network payload-hash fields. Snapshot inventories retain byte occupancy and bounded metadata needed for identities and validation, without reviving delivery objects as history.

Bounded history admission reserves terminal capacity before accepting work that can stall. Each connection pair with a live data-stall interval reserves one timeout-reset slot, shared across its stalled directions. Each pending disabled-link dial reserves one timeout-outcome slot. Committed transitions plus live reservations never exceed the existing transition ceiling. A held write must atomically admit its ordinary write record and any new reservation before bytes enter the queue. An effective partition that stalls existing queued traffic must atomically admit its topology record and all newly needed pair reservations before any group mutation. Insufficient capacity rejects with the existing resource error and changes no bytes, topology, timer or identity. Ordinary competing operations cannot consume a reserved slot.

Expiry consumes its reserved slot for the recorded terminal outcome within that ceiling. Effective heal or removal of the last stalled data direction releases the unused pair reservation; stale callbacks cannot release a replacement reservation. Dial completion/caller failure releases or atomically consumes its pending reservation according to the existing dial-record path. Close, stop and crash release obsolete reservations as part of their ordinary atomic cleanup. A replay mismatch remains an observable replay failure before the affected mutation; it cannot silently strand successful work. Both-backend controls fill all unreserved history capacity after a held writer returns, then prove that its one recorded reset still occurs without active application I/O.

Replay reexecutes target writes using actual bytes. The network record checks representable structure, operation length/order/outcome, topology, faults and terminal behavior before the relevant mutation. FIFO delivery derives from those writes, link state and virtual time. Preserve full record consumption, identity preflight and nested controller/fault corruption checks. Volume hashes, artifact authentication and metadata identity digests remain.

Direct network-only simulation without the deterministic-I/O profile deliberately loses same-length payload divergence detection from network history. Other observations may still detect a difference; no universal rejection is promised. Composed Runner replay retains its separate bounded generic I/O content checks and first mismatch ordinal. Keep a composed-profile payload mismatch negative and a direct network-only waiver control. This spec does not add a replacement replay layer or remove generic I/O hashes.

Change the affected network model, transition, config/record wire, snapshot and simulation schema identities coherently. Reject old network/config identities before model activation. Use the single generated semantic codec admitted by the predecessor; no legacy decoder or broad unrelated execution-format bump. Final-input generation and paired host/overlay validation bind the accepted candidate. Intermediate task checkpoints are not released or qualified formats.

Admit additive held/timeout vocabulary, endpoint validation and matching host/overlay lanes before runtime producers emit it. Keep currently produced delivery shapes valid until the destructive history migration. Each checkpoint must encode and validate its own run results; an unreleased checkpoint is not permission to fail its acceptance tests. Owners of new both-backend regressions register and execute their process cases before completing; the later process audit checks the complete selection.

## Edge Cases & Constraints

Both graceful stop and crash retain their existing queued-byte discard behavior. Graceful stop exposes the established EOF outcome; crash exposes reset. Held bytes do not drain through a stopped incarnation. Restart cannot revive old bytes, descriptors or expiry work. Stale/cancelled expiry work produces no timeout transition.

Asymmetric and grouped faults preserve their existing directed topology and atomic validation. Refused dials and resource exhaustion remain model outcomes. Caller-supplied fault matching, including the explicit `deliver` operation string and occurrence semantics, remains valid even when automatic delivery history disappears.

Admission consumes fn-109.14's source-accepted single generated network codec and its existing time-codec prerequisite. The tracker cannot express cross-spec task edges, so the codec-consuming task records this narrow admission gate. Do not replace it with a whole-fn-109 dependency. Consume the fn-109 progress interface and fn-155's selected descriptor shape; any concrete interface change needs an explicit owner handoff before editing it. Fn-109 retains backend factories and progress acceptance. Fn-152/153 retain storage and JSON/publication work; refresh affected identities at their final inputs without inventing spec-wide predecessors.

Fn-155 remains first for implementation under the operative delivery order. Planning here does not admit dependent implementation or satisfy fn-155's first supported-platform execution. Serialize overlapping overlay, generator, toolchain and Runner identity work at integration and gates.

Inherited native qualification remains deferred under its existing owners. Newly introduced tests remain with this spec unless the owner explicitly transfers that exact gate. Required supported-platform execution, source review, ordinary host-source tests, lint, generated checks, preservation and both-source-set static checks remain open until proved. No CI, PR, push or native-owner revival is authorized.

## Acceptance Criteria

- **R1:** Partitioned writes succeed for accepted bytes while local capacity remains, then block until consumption, caller deadline or reset. Errors: caller deadline returns the accepted prefix with recoverable deadline error; terminal stall reset returns the prefix with persistent timeout; global-byte admission failure retains the resource-exhausted outcome.
- **R2:** No undelivered bytes cross a disabled directional link, including queued pre-partition bytes. Already-received bytes remain readable, and an enabled reverse link remains usable until pair reset. Errors: read deadline remains recoverable; terminal reset follows R4.
- **R3:** Effective heal delivers held bytes complete and FIFO after normal delay. Fresh writes never overtake them; repeated partition/heal never loses, reorders or duplicates bytes. Errors: terminal failures follow R4/R12; invalid or rejected topology changes mutate nothing.
- **R4:** Positive undelivered occupancy held continuously longer than the configured virtual stall limit resets both endpoints permanently with timeout classification, even without active application I/O or unreserved history capacity. Idle connections survive indefinite partition. Errors: insufficient terminal reservation rejects admission atomically; stale epoch/incarnation events do nothing; close/stop/crash invalidate expiry; overflow rejects before mutation; exact-limit and same-time boundaries follow Ordering.
- **R5:** A partitioned dial waits for effective heal, caller deadline/context or strict stall expiry. Errors: completed caller context precedes admission; deadline remains a caller error; stall expiry is a timeout; refusal, stale endpoints and capacity errors retain existing behavior; no unsupported process cancellation guarantee is added.
- **R6:** Per-simulation stall configuration has the explicit named 10-second default supplied by `DefaultLimits()`, identical across both backends. Errors: omitted, zero, invalid, out-of-range or overflowing values reject before activation; old config identities reject rather than silently default.
- **R7:** `partition_drop` disappears. Held-write outcomes, topology heal and timeout reset are recorded and replayed in both backends; FIFO release adds no delivery/release step. Committed history plus live terminal reservations stays within the existing transition ceiling. Errors: representable replay mismatch fails before the relevant mutation; stale expiry adds no reset step; competing operations cannot consume reserved terminal capacity.
- **R8:** Length-framed streams in both backends deliver every frame intact after a short partition, or fail with timeout after a long partition, without a stream-gap framing error. Errors: caller deadlines, explicit terminal faults and capacity outcomes follow R1-R5/R12 rather than inventing framing failures.
- **R9:** Standalone and multi-node domains share one local connection owner for read/write/deadline/close and nonblocking behavior. Process RPC and descriptors remain adapters. Errors: preserve half-close, empty I/O, partial-result, readiness-generation and existing local-domain error precedence through the shared owner.
- **R10:** Network history contains neither per-delivery steps nor network payload hashes. Replay derives delivery during reexecution and retains structural/topology/terminal/fault checks, full consumption and preflight. Errors: representable mismatches retain divergence failures; direct network-only same-length payload checking is deliberately reduced; composed-profile generic I/O checks remain and reject payload mismatch at their existing ordinal/bound.
- **R11:** Stream capacity uses bytes only, with an explicit default 4 MiB per direction and the existing global byte ceiling. Every unread retained byte remains charged. Errors: local fullness blocks under R1, global exhaustion returns resource-exhausted, invalid limits reject; metadata/history/frame bounds remain, while chunk/delivery-count admission disappears.
- **R12:** Existing partition/heal/delay, crash reset, graceful EOF/discard, refused dial and capacity outcomes remain available through their established fault-plan/scenario or model interfaces, each covered in both backends. Errors: retain atomic rejected fault actions, nested corruption, extra/unused/reordered actions, revocation and fresh-process isolation controls; explicit `deliver` fault matching survives.

## Early proof point

Task fn-154-gomad-simpler-virtual-network-with.4 proves local-domain and descriptor equivalence through the existing public interface before adding partition timing. If the equivalence pin fails, revisit the connection seam before building held-data and expiry behavior on it.

## Boundaries

- No additional fault kinds, lossy-link mode or reset-on-partition mode.
- No packet stack, retransmission, congestion/window model or keepalive behavior.
- No changes to fault/scenario selection or scheduling.
- No replacement clock, deterministic scheduler, response barrier or replay framework.
- No deletion of process IPC, virtual descriptors, volume hashes or generic I/O content checks.
- No unrelated source cleanup, production Temporal behavior changes or third-party libraries.

## Decision Context

The owner chose stall with timeout and widened the work to connection simplification. Reset-on-partition omits the requested hang-until-deadline case; a gRPC interceptor gives up transparent targets and other stream clients. Both alternatives remain outside this spec.

Source review confirmed the need for explicit defaults and differentiated capacity outcomes. The inferred dial stall bound remains a deliberate new feature under the autonomous instruction. R11 now separates local backpressure from preserved global resource failure. These are planning decisions rather than claims that the behavior already exists.

Ten virtual seconds keeps the representative 1 ms caller deadline independent and falls below the 20 s virtual process-test harness timeout. It is not a production calibration. Retaining byte FIFO state and existing domain adapters avoids a separate network package with a hypothetical seam.

The payload-free history change accepts reduced direct network-only same-length payload checking. The conditional generic I/O transcript cannot prove unchanged detection for every direct simulation. Tests and documentation must state that division rather than preserve an obsolete payload-field assertion.

## Quick commands

```bash
go test -tags test_dep ./tools/gomad3sim -run 'Test(ValidateSpec|DecodeSpec|SpecJSONFieldNames|ClusterRecord)'
make -C tools/gomad3 validate
make -C tools/gomad3 overlay-test test-simulation
make lint-code-fast
```

Patched-runtime commands require a supported darwin/arm64 or linux/amd64 candidate. Ordinary portable source checks do not establish those passes.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Partitioned writes succeed for accepted bytes while local capacity remains, then block until consumption, caller deadline or reset. Errors: caller deadline returns the accepted prefix with recoverable deadline error; terminal stall reset returns the prefix with persistent timeout; global-byte admission failure retains the resource-exhausted outcome. | fn-154-gomad-simpler-virtual-network-with.10, fn-154-gomad-simpler-virtual-network-with.12, fn-154-gomad-simpler-virtual-network-with.4, fn-154-gomad-simpler-virtual-network-with.5, fn-154-gomad-simpler-virtual-network-with.9 | — |
| R2 | No undelivered bytes cross a disabled directional link, including queued pre-partition bytes. Already-received bytes remain readable, and an enabled reverse link remains usable until pair reset. Errors: read deadline remains recoverable; terminal reset follows R4. | fn-154-gomad-simpler-virtual-network-with.10, fn-154-gomad-simpler-virtual-network-with.12, fn-154-gomad-simpler-virtual-network-with.5, fn-154-gomad-simpler-virtual-network-with.9 | — |
| R3 | Effective heal delivers held bytes complete and FIFO after normal delay. Fresh writes never overtake them; repeated partition/heal never loses, reorders or duplicates bytes. Errors: terminal failures follow R4/R12; invalid or rejected topology changes mutate nothing. | fn-154-gomad-simpler-virtual-network-with.10, fn-154-gomad-simpler-virtual-network-with.12, fn-154-gomad-simpler-virtual-network-with.5, fn-154-gomad-simpler-virtual-network-with.9 | — |
| R4 | Positive undelivered occupancy held continuously longer than the configured virtual stall limit resets both endpoints permanently with timeout classification, even without active application I/O or unreserved history capacity. Idle connections survive indefinite partition. Errors: insufficient terminal reservation rejects admission atomically; stale epoch/incarnation events do nothing; close/stop/crash invalidate expiry; overflow rejects before mutation; exact-limit and same-time boundaries follow Ordering. | fn-154-gomad-simpler-virtual-network-with.10, fn-154-gomad-simpler-virtual-network-with.12, fn-154-gomad-simpler-virtual-network-with.3, fn-154-gomad-simpler-virtual-network-with.4, fn-154-gomad-simpler-virtual-network-with.6, fn-154-gomad-simpler-virtual-network-with.9 | — |
| R5 | A partitioned dial waits for effective heal, caller deadline/context or strict stall expiry. Errors: completed caller context precedes admission; deadline remains a caller error; stall expiry is a timeout; refusal, stale endpoints and capacity errors retain existing behavior; no unsupported process cancellation guarantee is added. | fn-154-gomad-simpler-virtual-network-with.10, fn-154-gomad-simpler-virtual-network-with.12, fn-154-gomad-simpler-virtual-network-with.3, fn-154-gomad-simpler-virtual-network-with.7 | — |
| R6 | Per-simulation stall configuration has the explicit named 10-second default supplied by `DefaultLimits()`, identical across both backends. Errors: omitted, zero, invalid, out-of-range or overflowing values reject before activation; old config identities reject rather than silently default. | fn-154-gomad-simpler-virtual-network-with.1, fn-154-gomad-simpler-virtual-network-with.10, fn-154-gomad-simpler-virtual-network-with.12, fn-154-gomad-simpler-virtual-network-with.2 | — |
| R7 | `partition_drop` disappears. Held-write outcomes, topology heal and timeout reset are recorded and replayed in both backends; FIFO release adds no delivery/release step. Committed history plus live terminal reservations stays within the existing transition ceiling. Errors: representable replay mismatch fails before the relevant mutation; stale expiry adds no reset step; competing operations cannot consume reserved terminal capacity. | fn-154-gomad-simpler-virtual-network-with.10, fn-154-gomad-simpler-virtual-network-with.12, fn-154-gomad-simpler-virtual-network-with.2, fn-154-gomad-simpler-virtual-network-with.5, fn-154-gomad-simpler-virtual-network-with.6, fn-154-gomad-simpler-virtual-network-with.8 | — |
| R8 | Length-framed streams in both backends deliver every frame intact after a short partition, or fail with timeout after a long partition, without a stream-gap framing error. Errors: caller deadlines, explicit terminal faults and capacity outcomes follow R1-R5/R12 rather than inventing framing failures. | fn-154-gomad-simpler-virtual-network-with.11, fn-154-gomad-simpler-virtual-network-with.12 | — |
| R9 | Standalone and multi-node domains share one local connection owner for read/write/deadline/close and nonblocking behavior. Process RPC and descriptors remain adapters. Errors: preserve half-close, empty I/O, partial-result, readiness-generation and existing local-domain error precedence through the shared owner. | fn-154-gomad-simpler-virtual-network-with.12, fn-154-gomad-simpler-virtual-network-with.4, fn-154-gomad-simpler-virtual-network-with.9 | — |
| R10 | Network history contains neither per-delivery steps nor network payload hashes. Replay derives delivery during reexecution and retains structural/topology/terminal/fault checks, full consumption and preflight. Errors: representable mismatches retain divergence failures; direct network-only same-length payload checking is deliberately reduced; composed-profile generic I/O checks remain and reject payload mismatch at their existing ordinal/bound. | fn-154-gomad-simpler-virtual-network-with.11, fn-154-gomad-simpler-virtual-network-with.12, fn-154-gomad-simpler-virtual-network-with.8 | — |
| R11 | Stream capacity uses bytes only, with an explicit default 4 MiB per direction and the existing global byte ceiling. Every unread retained byte remains charged. Errors: local fullness blocks under R1, global exhaustion returns resource-exhausted, invalid limits reject; metadata/history/frame bounds remain, while chunk/delivery-count admission disappears. | fn-154-gomad-simpler-virtual-network-with.1, fn-154-gomad-simpler-virtual-network-with.10, fn-154-gomad-simpler-virtual-network-with.12, fn-154-gomad-simpler-virtual-network-with.2, fn-154-gomad-simpler-virtual-network-with.4, fn-154-gomad-simpler-virtual-network-with.8, fn-154-gomad-simpler-virtual-network-with.9 | — |
| R12 | Existing partition/heal/delay, crash reset, graceful EOF/discard, refused dial and capacity outcomes remain available through their established fault-plan/scenario or model interfaces, each covered in both backends. Errors: retain atomic rejected fault actions, nested corruption, extra/unused/reordered actions, revocation and fresh-process isolation controls; explicit `deliver` fault matching survives. | fn-154-gomad-simpler-virtual-network-with.10, fn-154-gomad-simpler-virtual-network-with.11, fn-154-gomad-simpler-virtual-network-with.12, fn-154-gomad-simpler-virtual-network-with.9 | — |
