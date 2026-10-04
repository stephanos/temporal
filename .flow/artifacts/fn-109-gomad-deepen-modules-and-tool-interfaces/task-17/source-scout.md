# Network handle source scout

Read-only investigation of the current task-14 typed-command candidate at
HEAD `0dd05b313acd0986312da7fd3159520e6a21f1bf`. No implementation, tests,
generation, Flow mutation, bridge or native qualification was performed.
Dispatch: Codex thinking scout, requested `gpt-6.1-sol` at high effort.
Tier: session (jev-unavailable(no_key)); explicit routing remains authoritative.

## Creation-time selection

Keep exported `Listener` and `Conn` wrappers, each holding one private interface.
Listener operations: Accept, Close, Address, SetDeadline. Connection operations:
Read, Write, Close, CloseRead, CloseWrite, LocalAddress, RemoteAddress,
SetDeadline, SetReadDeadline and SetWriteDeadline. Exported methods delegate;
there is no optional-backend union or generic registry. Preserve read/write
serialization around the whole operation, either in the wrapper or each owner.

Three concrete implementations own their valid state:

- Standalone: local listener queue/deadline/mutex/change channel and paired
  connection stream state, addresses, pending read bytes and close-once.
- Simulation: corresponding local state plus mandatory simulation model and
  owner endpoint; connection identity and target endpoint are also mandatory.
- Process: remote handle and returned addresses only, with no local queue,
  stream state, endpoint, model or close-once.

Share backend-neutral paired stream types and timer/signal mechanics, without
copying the simulation domain model or relocating backend conditionals into
another optional-state object.

## Source map

- `network.go:102,145`: ListenTCP/DialTCP select process role 2, current
  simulation domain, then standalone in the existing validation order.
- `network.go:139,195`: standalone construction.
- `simulation_network.go:391,490`: simulation construction; listener
  maps/history migrate to concrete simulation listeners, not exported unions.
- `process_network.go:38,108`: process listener/connection construction;
  Accept creates the process connection from its typed result.
- `network.go:213-301,327-614`: listener and connection operation bodies.
- `process_network.go:53-105`: existing process typed-command operations.
- `process_network.go:174,257`: host registration uses ordinary address
  methods instead of private address fields. Host resources retain wrappers.
- New overlay filenames require the descriptor inventory and generation.
  Adapt task-14 test-only process constructors, not its literal wire vectors.

## Preservation traps

Lock ordering remains global/model network before listener/shared-stream locks;
direction locks precede model locks. Release locks before blocking and retain
close-and-replace notifications. Simulation transition validation precedes queue
removal, identity increments, delivery mutation and close state.

Accept drains pending connections before local closed/deadline checks. Reads
prioritize reset/read-closed, buffered bytes, incoming delivery, peer EOF, then
deadline. Writes check closed/reset, then available capacity, then deadline.
Do not introduce uniform early deadline checks.

Preserve backend-specific outcomes:

- Process empty Read/Write bypass exchange and recording. Local empty Read
  records success before endpoint validation; simulation empty Write validates
  its endpoint and then records.
- Process successful Close removes its host resource; repeated Close and
  subsequent Accept on a closed listener return ESTALE. Local repeated Close
  returns ErrClosed and queued Accept can still drain after close.
- Process Write chunks into 64-KiB host operations. Preserve partial counts,
  count/data consistency, EIO/short-write classification and transcript granularity.
- Graceful revocation yields peer EOF; crash resets both sides and removes
  delayed deliveries. Preserve partial buffered data, incarnation checks and
  non-consuming capacity/transition failures.

## Test inventory and additions

Existing Runner network/bind tests cover standalone pending-before-close and
deadline semantics, half-close, interrupted reads, cancellation, port exhaustion,
bind/rebind and transcripts. `TestModelConformanceTCP` covers five fixed seeded
64-operation sequences. Root `network_toolchain_test.go` covers in-process
partition/heal, delay replay, topology mismatch, restart, EOF/reset and capacity.
Root cluster tests cover process isolation, host-model TCP/listen routing and
inflight crash draining. The existing detached-model comparison uses boots with
no TCP operations and is not handle-operation parity evidence.

Add one operation table across standalone, in-process and real process runners:
address operations, duplicate bind/rebind, short reads across chunk boundaries,
empty I/O, half-close, final bytes before EOF, deadline changes/clearing, and
partial Write followed by deadline/closure. Encode existing backend distinctions.
Exercise simulation capacity, delayed delivery removal, graceful/crash outcomes
and exact replay on both simulation backends. Add stale endpoint/incarnation and
host resource revocation/wrong-domain/wrong-kind checks. Operation replay
mismatches must reject before mutation. Real process cases belong in the Runner
root integration selector; direct root tests can skip unavailable transport, so
stock-runtime stand-ins do not prove process coverage.

The model-delay watchdog disposition stays unchanged. Native patched-runtime
and process qualification remains required on darwin/arm64 and linux/amd64.
