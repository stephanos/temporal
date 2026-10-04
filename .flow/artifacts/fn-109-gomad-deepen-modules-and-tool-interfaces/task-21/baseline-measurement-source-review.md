# Baseline measurement source/method review

Fresh-context early review by `r19_measurement_method_review`; not a formal
implementation verdict, current comparison, R19 acceptance, or native qualification.
Requested reviewer routing was `gpt-6.1-sol/high`, the writer's model family.
Actual execution metadata was unavailable.

The reviewer reported no Critical or Important findings. Pair gates enforce
snapshots at `(returned, committed, active)` equal to `(0,0,2)`, `(N-2,N-2,2)`,
and `(N,N,0)`. `KeepAlive` retains both active payloads through profiling, and
two GC cycles address delayed profile accounting. Executor and preparer retain
no request/result history. The developmental platform overlay and guarded
descriptor substitutions are scratch-only; transport-call counters remain zero.

Two Minor provenance gaps require hardening for comparative reruns:

1. Bind compiler/runtime settings before execution. The historical driver
   inherited settings and retained explicit overrides only; post-run environment
   observations cannot prove historical settings.
2. Compare complete pre/post source path sets as well as hashes. The historical
   driver checked initially enumerated files but could miss unexpected additions.
   The two known supplemental tests were added after its campaign postcheck.

Attribution must distinguish named logical bytes from allocator storage,
retained evidence, and selection-derived aggregate capacities. Normalize
producer/assessment allocations by executions, publication allocations by
publications. Raw stack origins distinguish transcript production, semantic
decoding, World encoding, and publication buffers. Alias checks and live
snapshots do not independently exclude transient copies. Completed snapshots
prove sampled payload release after campaign policy objects have returned.

The reviewer checked five selected file hashes before/after and matched scratch
test/guard/helper hashes to retained copies. It performed only reads and hashes.
The explicitly bound rerun assignment addresses these findings without rewriting
the historical evidence or inferring its environment.
