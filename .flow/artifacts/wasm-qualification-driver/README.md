# WASM frozen-qualification drivers

Snapshot of the drivers used for fn-176/fn-178 frozen qualification (2026-10-15).

- `run-qualification.sh`: the serial reference driver (fn-176 84 stages plus the fn-178 cluster stages; 89 total).
- `run-qualification-fast.sh`: owner-approved fast variant (2026-10-15). Same stage set, commands and post-checks, reordered into a fail-fast static phase with a tool preflight, the helper-build gate, three parallel functional lanes (heavy Temporal guests share one token), a serial selection/catalogue phase and an exclusive phase for the busy12 stress and timing-sensitive stages. At startup it asserts its stage set equals the reference driver's (`stage-list.py`) and aborts on any difference. `--plan` prints lanes and reuse keys without running.
- `--reuse <prior QUAL_ROOT>` (repeatable, off by default) copies a prior stage pass only when its key matches exactly: stage label, command, helper and compiler identities, and the stage's declared input digest. Reused stages are logged as `REUSED <label> <root>` and never count as fresh passes.
- `quiet-window-fast.fn178.sh`: the fn-178 measurement window (fast driver, concurrent measurement prebuilds, then baseline/candidate/baton timing alone). Its workspace paths are specific to the fn-178 run.

Scratch must live on native disk, not virtiofs (the driver refuses virtiofs/fuse scratch). Default paths name the sandbox layout; pass explicit arguments elsewhere.
