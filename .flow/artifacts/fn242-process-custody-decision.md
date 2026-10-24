# fn-242 process custody correction — 2026-10-24

Status: conductor-selected architecture for the active .13 review fix; implementation and re-review remain pending. This is not a qualification pass.

Review round 1 retained three introduced findings: configured command timeouts above 3600 wrongly refused original 5400/7200 bounds; supervisor death leaked descendants; blocking stream/registration reopening could hang on FIFO replacement. Raw review: `/home/agent/wt/fn242-process-review/.flow/review-fanout/fa63d5ac3ede474597e48c40e2bb72db/correctness.review.md`. Same-receipt fix loop: `/tmp/impl-review-receipt-7697af133a77-fn-242-gomad-shrink-the-qualification.13.json`.

Actual host probes retained at `/home/agent/wt/evidence/fn242-cycle-process/review-fix/namespace-probes.json`: UID/GID1000 with supplementary groups27/1000/1001; PID namespace alone exit1 EPERM; user+PID namespace exit0 but changes UID/GID to0 and supplementary groups to65534. User namespace wrapping changes producer permission semantics. Another ordinary guardian only moves the outer supervisor failure.

The dedicated cycle CLI process becomes the explicit trusted process owner. It claims subreaper status once before children/threads, requiring empty per-thread child inventories and unclaimed subreaper state. A private owner-PID token gates every Runner. Never initialize implicitly in a generic Runner, restore the role mid-lifetime, or initialize unrelated measurement/other CLI commands.

After claim, all process-domain children belong to the cycle. Spawn/registration/gate release/reaping synchronize with one fatal-drain barrier. Supervisor death or malformed/incomplete observation aborts the entire domain, rejects admission, kills every direct/adopted child, and reaps to kernel ECHILD before returning error. No per-command attribution is asserted after adoption. Normal resources still come from exactly one actual leaf wait4; lost accounting remains inconclusive failure even when cleanup succeeds. Failure to drain remains a failure and cannot be called a clean handoff.

The owner's own unexpected SIGKILL lies outside its lifetime guarantee, as for the original harness. This is process custody, not a hostile-code security sandbox. Do not claim finite cleanup for indefinitely uninterruptible kernel tasks.

.13 proves the same strict initializer and production implementation using a standalone single-threaded harness=false test target (additive Cargo.toml scope). No test-only relaxed initializer or ELF pre-main hook. B15 must separately wire actual cycle startup before children/threads and prove the real route; its original acceptance remains unchanged. Current production CLI wiring is absent.

The owner authorized autonomous implementation decisions and batching; this correction follows those instructions. It changes the earlier internal isolated-supervisor design while preserving original acceptance, exact leaf argv/environment/cwd/credentials and all proof ownership. No full qualification, enrollment, reservation or allowance was used. The fixed operational journal binding remains absent; 0/2 is activity/policy accounting rather than verified durable enrollment.
