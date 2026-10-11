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
and polling never advance it. All listener, connection, delivery, pending-byte,
transition and request allocations have declared limits.
The request transport serializes a closed model-owned error vocabulary without
calling error-interface callbacks or host syscall formatters; its stale-handle
message is the portable `stale file handle`. Structured native bridge errors
retain their detached message.

The `gomad3.simulation-network-stream/v2` profile retains the native policies:

| Action | Existing connection | New dial |
| --- | --- | --- |
| Delay | Already queued bytes retain their readiness; future writes use the new delay. | Native connection creation remains immediate. |
| Partition | Queued bytes drain. Future writes succeed with a recorded `partition_drop`; this is the native lossy stream policy, not TCP packet-loss simulation. | Blocks until heal, cancellation or deadline. |
| Heal | Future writes deliver again; it does not reconstruct dropped bytes. | Pending dials become ready. |
| Reset | Discards queued deliveries and resets both directions. | A later dial can connect to the existing listener. |
| Incarnation revoke | Native graceful stop or crash policy; stale handles and pending requests fail. | Requires a new registered incarnation. |

Connection reset is an explicit v2 extension mechanically extracted from the
native revoke body. Listener identity survives this operation. Native fault
recording and the generated drop-delivery owner remain authoritative.
Scheduled faults at one timestamp run in declared plan order before that
timestamp's stream readiness is settled. This defines the reset/deadline tie
policy and is retained with the fault plan and World evidence.

The one-guest adapter maps guest loopback listeners and callers into distinct
server and client domains (`127.0.0.1` and `127.0.0.2`), because the native model
forbids self links and duplicate node addresses. Guest-facing addresses remain
loopback. A broker attaches its own node/incarnation domains and address mapping
through the same model API.

The retained WASM v1 network model and its count-only glue keep their previous
semantics. V2 has a separate configuration, import and implementation identity;
mixing both configurations is rejected.
