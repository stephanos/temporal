# Network handles: selected creation-time ownership

This design implements the existing task 17/R12 requirements after the task-16 source candidate is independently reviewed. It adds no backend, registry or public API. Native acceptance remains separate. The user's standing instruction to choose grounded recommendations replaces further design questions; Flow remains the sole task tracker and the user owns commits.

## Alternatives and decision

1. Select private listener/connection interfaces at construction, with standalone, simulation and process concrete owners. This removes invalid combinations while preserving public methods, the shared simulation domain model and real backend differences. Selected.
2. Move backend conditionals to a common optional-state helper. Fewer changed lines, but it preserves the invalid state space and distributed choice that R12 expressly removes. Rejected.
3. Introduce a backend registry or duplicate model engines per backend. More abstraction and maintenance, without a required extension use case; duplicate models threaten equivalence. Rejected.

Exported Listener and Conn become thin facades. Read/write direction serialization still covers the entire operation. Standalone owns local queue/stream/address/deadline state. Simulation owns that state plus mandatory model/endpoint/incarnation/connection identity. Process owns remote handles and returned addresses. Host process registration retains facade values and uses address methods rather than their old private fields. Simulation listener indexes retain concrete simulation state, not invalid facade field assumptions. Shared connState/connShared and timer/signal mechanics may remain backend-neutral; the simulation model remains one domain owner used by both simulation backends.

## Preserved behavior and errors

Construction keeps existing validation and backend-selection precedence. Methods preserve lock order, deadline precedence, pending/data-before-close behavior, half-close, partial counts, capacity failures, transition-before-mutation, stale incarnations and replay rejection. No wrapper-level uniform recording is added. Process empty I/O does no exchange/record; successful remote close removes the resource, unlike local repeated-close ErrClosed and queued-accept behavior. Process writes retain 64-KiB host chunks and transcript granularity. Graceful revocation yields EOF and crash yields reset/removes delayed deliveries. The source scout records the complete migration and validation map.

## Tests and delivery

Baseline the canonical Quick commands and capture old source before editing. Preserve task-14 literal wire vectors and typed-command ownership. Establish an architecture RED against real old production AST optional-state/backend-branch violations before production migration, not a compile error or grep check. Add shared operation cases before migration with explicit backend expectations; cases must exercise actual standalone, in-process and process routes. Include the real process cases in Runner's integration selector. Stock/scratch shim coverage is developmental only and cannot prove IPC, isolation, native timers or exact replay.

Serialize descriptor/generator/overlay edits, use the existing generator and validate exact inventories. Preserve runtime protocol bytes and existing comments. Freeze source, retain focused final commands/exits/timings/source identities and no-live-command handover; conductor verification/review precede the next source admission. No git mutation, Flow completion or native waiver is delegated.

The conductor clarified task 17's Touches through flowctl for its required test delivery: the nested-module architecture guard and execution-package test/integration selector. Acceptance and delivery order are unchanged. Writing-for-agents influenced this clarification by co-locating the actual process-selection obligation with its allowed test surface.
