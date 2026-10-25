Localized backend/fixture roots, cache and evidence paths, Go target fallbacks, and standalone consumer documentation. The compiled README example now reads the exact local framework block.

Scope: path-consumer lane of fn-254-fast-validation-gates-and-a-real-inner.13; task remains in_progress under the conductor.

Checks: scoped rustfmt --check and git diff --check exited 0; 50 old root joins removed, 154 test names/attributes unchanged, README fixture byte parity checked; no snapshot/stderr/wasm changes. No Cargo/Go build or runtime gate was run in this lane. Parent must run joined affected controls after workspace/tools/scripts integration.

Scope admission followup: commit 23c3d48819 updates exactly seven direct Temporal TLC roots through shared tests::framework_root(), with exact replacement-only static parity across the three admitted files and all 36 test declarations retained. Baseline/post scoped rustfmt and diff checks exited 0. Joined compilation remains with parent.

stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)
