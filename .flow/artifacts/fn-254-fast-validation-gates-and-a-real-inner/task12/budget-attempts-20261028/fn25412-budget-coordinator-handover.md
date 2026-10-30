# fn254.12 budget coordinator source handover

Source tip `fc03cba590`, RED checkpoint `1511c3c83c`, based on `55c51d397b`. Root confirmed the intentional RED failure; GREEN compile/tests remain pending. This lane executed no Cargo, JVM or test runtime. Acceptance is false.

Only `model/umpire/dev/xtask/src/gate/budget_coordinator.rs` is owned. API visibility is `pub(super)`; root owns registration and integration. The module accepts the actual selected scoped stage inventory and reviewed finite drain Duration. It admits payloads beneath the unchanged mise graph, performs no callback/graph/process/lease work, and holds its stable file lock only around bounded strict JSON transitions. The manifest binds admitted context, private directory identity and stable lock identity. State updates replace `state.json` atomically and sync; unknown/partial files or invalid state fail closed.

First slow completion removes active ownership atomically, consumes the sole retry and queues it. Retry requests outrank queued exclusive-first/shared admissions. Both exclusive-first and retry drain active payloads and then block all outer payload admission. Queued callbacks never count as active. Normal finish consumes its opaque owner token. Dropped permits/tokens abort without clearing stale active entries or inferring PID absence.

Internal incomplete failures never replace the first actual provided FailureCause. A later actual37/143 supersedes earlier internal incomplete1; subsequent provided failures preserve the first actual cause. `Error::original_failure` exposes it, and `Coordinator::failure` supports root supervisor polling. Root still owns actual status propagation, same-image/context checks, count admission, callback replay, attempt evidence and inclusive tier timing.

Nine source controls cover concurrent slow drains, retry priority, statuses37/143, second-slow/drop refusal, duplicate/crossed/partial state, independent invocations, missing owner finite drain, lock/symlink/partial/unknown/duplicate integrity, and invalid failure/stage refusal. Pinned rustfmt and diff checks pass. JSON companion records exact API, schema, invariants and remaining runtime limits. Worktree is retained until root integrates and authorizes retirement.

## Fixture setup correction

Root's first GREEN attempt failed all nine controls at the privacy guard. Pinned Rust tempfile3.27.0 directories use default permissions, unlike Python mkdtemp. `3989d3f2d1` makes all fixture roots explicitly0700 under canonical temp base, separates directory/mode/canonical diagnostics, removes unnecessary fs qualification, and adds a tenth control proving755 input is rejected without repair or writes. Production admission policy remains unchanged. Runtime rerun is root-owned and pending.
