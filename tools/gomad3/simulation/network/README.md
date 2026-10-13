# Native network host projection

`go generate ./simulation/network` projects the authoritative `gomadio`
simulation into a nonblocking host package. The generator selects declarations,
removes native diagnostic transport calls, supplies explicit domains and logical
time, and replaces unlocked waits with immediate pending results. Its checked
transform counts and preservation tests reject native drift. Native sources stay
unchanged.

`Model.Dispatch` consumes at most one 64 KiB write chunk. A pending result owns
its request until `poll` consumes completion or cancellation. `Ready` and
`Advance` settle requests in request-ID order without blocking host goroutines.
The execution owner advances time only at global guest quiescence; clock reads
and polling never advance it. Listener, connection, retained-byte, transition
and pending-request allocations have declared limits. Native timers and the
existing host `NextDeadline`/`Advance` path drive stall expiry even with no
blocked application operation. Process execution keeps its existing time/progress
arbiter.
The request transport serializes a closed model-owned error vocabulary without
calling error-interface callbacks or host syscall formatters; its stale-handle
message is the portable `stale file handle`. Structured native bridge errors
retain their detached message.

The following policies describe the approved
[fn-154 migration](../../../../.flow/specs/fn-154-gomad-simpler-virtual-network-with.md).
Pending-dial/history source, final generated identities, integrated review and
execution gates remain open; this document does not qualify the checkpoint.
The stream profile and affected config/record/model/transition identities must
rotate together. Old identities reject before activation through the single
generated semantic codec. See the
[network contract](../../README.md#network-streams-partitions-and-replay).

| Action | Existing connection | New dial |
| --- | --- | --- |
| Delay | Already queued bytes retain their readiness; future writes use the new delay. | Native connection creation remains immediate. |
| Partition | Holds every undelivered suffix, including pre-partition traffic. Already-received bytes remain readable. Writes accept bytes until the local byte bound blocks them. | Waits for heal, caller deadline/context or independent strict stall expiry. |
| Heal | Effective heal releases the held FIFO after normal delay; fresh writes cannot overtake it. | Ends the stall interval and permits ordinary listener/connection admission. |
| Reset | Discards retained bytes and resets both directions. Stall expiry retains a persistent timeout reason. | A later dial can connect to the existing listener. |
| Incarnation revoke | Native graceful stop or crash policy; stale handles and pending requests fail. | Requires a new registered incarnation. |

Each direction charges held bytes, received unread bytes and short-read tails
until consumption or discard. Local fullness blocks; global retained-byte
exhaustion returns resource-exhausted, with any accepted prefix. Explicit positive
`NetworkConnectionBytes` and `NetworkStallNanos` limits use `DefaultLimits()`'s
4 MiB and 10 virtual seconds; omitted/zero values reject. Listener, connection,
history, request and frame ceilings remain independently bounded. Chunk and
delivery counts cease to admit payload. Empty I/O creates no queued byte range.

A continuous interval starts when positive undelivered occupancy encounters a
disabled link, with first expiry at `start + limit + 1ns`. Either stalled
direction resets the pair even after its writer returns; idle connections do
not expire. Effective heal ends the interval. Repartition holds the undelivered
suffix and the next effective heal applies a fresh delay. Redundant heal adds
no delay; redundant disable does not restart the interval. A pending dial has
its own interval and reservation, separate from caller deadlines. It allocates
no connection merely to wait. Process cancellation keeps its existing transport
capabilities.

Admission atomically reserves one terminal history slot per stalled pair or
pending dial within the transition ceiling. Expiry consumes it; heal or cleanup
releases unused capacity. Invalid topology, insufficient reservation or duration
overflow reject before mutation. Stale epochs/incarnations have no effect and
cannot release replacement reservations. Close, stop, crash and restart revoke
obsolete work. Listener identity survives connection reset.

For nonempty I/O, terminal reads precede readable bytes, EOF and caller deadline;
writes check terminal state, available capacity, then caller deadline. Caller
deadlines remain recoverable, while clearing them cannot undo timeout reset. Full close discards
local unread and held outbound bytes; received peer data drains normally to EOF.
Read half-close discards local unread bytes; write half-close keeps accepted
outbound bytes for draining before EOF, including held bytes after heal. Graceful
stop discards queued/held bytes and exposes EOF; crash discards them and resets.
Restart never restores old bytes or descriptors.

History keeps operations, held-write outcomes, topology/fault changes and timeout
resets. Delivery/release steps, delivery IDs/counts/inventories, `partition_drop`
and network payload hashes disappear. Snapshots retain bounded endpoint byte
occupancy and metadata. Replay derives delivery from reexecuted writes, links
and time, retaining structural/order/terminal validation before mutation, full
consumption, preflight and nested corruption checks. Explicit `deliver` fault
matching retains its occurrence semantics. Volume hashes and controller/artifact
identities remain separate. Direct network-only replay loses guaranteed detection
of same-length payload changes; composed Runner replay preserves bounded generic
I/O content checks and its first mismatch ordinal. No replacement network hash
is introduced.

Scheduled faults at one timestamp run in declared plan order before that
timestamp's stream readiness is settled. This defines the reset/deadline tie
policy and is retained with the fault plan and World evidence. At the exact stall
limit expiry is not due. At the first due instant, heal committed first invalidates
the epoch and expiry committed first is terminal, under the existing model order.

The one-guest adapter maps guest loopback listeners and callers into distinct
server and client domains (`127.0.0.1` and `127.0.0.2`), because the native model
forbids self links and duplicate node addresses. Guest-facing addresses remain
loopback. A broker attaches its own node/incarnation domains and address mapping
through the same model API.

The retained WASM v1 network model and its count-only glue keep their previous
semantics. V2 has a separate configuration, import and implementation identity;
mixing both configurations is rejected.
