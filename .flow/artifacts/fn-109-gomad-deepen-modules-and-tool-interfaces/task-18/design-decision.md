# Filesystem handles: creation-time ownership

Design preparation for task 18/R12. Implementation admission waits for the
integrated task-17 source candidate's independent review and feasible checks.
Task status and acceptance remain governed by Flow; this document grants no
native qualification. The user's instruction to choose grounded recommendations
replaces further design questions. The user owns commits.

## Alternatives and decision

1. Give Handle and Mapping one private implementation selected at creation.
   Local and process concrete owners preserve the two existing representations;
   standalone and in-process share the local filesystem model. Selected: this
   removes invalid combinations without inventing a backend or public API.
2. Relocate the optional local/process fields and dispatch into a common helper.
   Smaller movement, but retains the invalid state space R12 removes. Rejected.
3. Introduce a registry or a separate simulation filesystem implementation.
   Adds an extension mechanism or duplicate semantics without a use case.
   Rejected.

## Ownership and operation flow

Mirror task 17's final reviewed facade shape, not its intermediate sources.
Handle delegates its fifteen existing operations through a private interface;
Mapping delegates Bytes and Close. FS.Open and local/process Map construction
choose the concrete owner once. Patched os and libc callers keep their current
operations and signatures. FS path-operation selection remains separately
scoped; volume persistence, journaling, replay and crash semantics stay shared.

Local owners retain their required FS/node identities, offsets, access and
append flags, generation, revocation and closed state. FS handle/mapping indexes
track local implementation pointers, avoiding backend assertions in lifecycle
helpers. Process owners retain remote IDs, handle path or mapping cache, and
closed state; they use task 14's typed commands. Host process resources may
retain facade values and release the registry lock before invoking operations.

Preserve FS.mu ownership, observer-before-journal mutation and existing lock
ordering. Unlinked nodes live until their final handle closes. Identical local
mapping regions share one buffer and byte charge; closing the charged owner
transfers accounting to a surviving alias. Writes, truncation and flushing
retain their existing mapping visibility and volume behavior.

## Backend distinctions and error contracts

Preserve local error precedence: unavailable filesystem, stale generation or
revocation, then closed. Retain exported ErrClosed identity and process partial
result validation. Successful remote close removes its resource; failed close
does not. Domain/incarnation validation and replay rejection precede mutation.

Readonly mounts remain immutable and the standalone loader remains separate
from NewSimulation. Local writable maps require a readable, writable volatile
file; process writable maps return ENOTSUP before closed/access/bounds checks.
Process mapping bytes are copied and cached; a cached read does not newly
validate its resource. Local alias/write-through semantics are not imposed on
that cache. Process ReadDir's empty-slice normalization differs from local EOF
results. Keep existing local capacity accounting, process registry capacity and
frame/data bounds explicit rather than harmonizing their outcomes.

## Test and evidence design

Before production migration, retain a meaningful architecture RED against the
old production Handle/Mapping optional-state ASTs, and baseline behavioral
cases against captured old source. A compile failure or source-text search is
not that RED. Test operation results through real implementations.

Reuse task 17's final shared-operation fixture shape across standalone,
in-process and actual process modes. Cover partial ReadAt/EOF, offsets, append
and WriteAt rejection, truncate/metadata, incremental sorted directories,
chdir, unlink survival, close and ErrClosed identity. Encode known backend
differences in the expectations. Real-process cases use separately selected
top-level tests, node-owned state, bounded admission/cleanup and explicit
successful terminal-state assertions; add them to Runner's integration
selector. The existing oneNodeVolumeSpec helper is in-process only.
Ensure CI's canonical Makefile simulation filter selects those cases too,
extending task 17's gate-selection regression and preserving its existing
strict-delay exclusion and separate forward-mode check.

Retain local mapping alias charge transfer, capacity/overlap precedence,
survival after file close, stale generation and zeroing. Cover process copied
cache, writable-map precedence, malformed partial results, domain mismatch,
revocation and successful-close registry removal. Reuse task 14's literal
wire vectors and existing actual-process volume restart/hard-isolation tests.
Developmental stock/scratch tests cannot prove IPC, isolation or native replay.

Serialize overlay descriptor edits and generation; validate exact inventories.
Freeze source before conductor checks and independent review, retaining command
results, source identities and no-live-command handover. Canonical native
Quick commands remain required on qualified hosts; absent toolchain and
unsupported-host results leave acceptance open. No Flow completion or git
mutation is delegated.

The existing task 18 is the implementation plan; writing-plans is not available
in this session. Writing-for-agents shaped this record by grouping ownership,
backend distinctions and their evidence requirements, with the source scout as
the detailed migration reference. Inline design review found no placeholders,
new API or conflicting backend requirements. Reconcile this design with task
17's frozen reviewed implementation before admission.
