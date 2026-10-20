# fn-188.6 Rust cluster test helper feature gates — 2026-10-20

## Scope and cause

This is a separate acceptance-gate repair following the fn-227 network-pin checkpoint `1ae035f31f`, owned by fn-188.6. It does not reopen closed fn-187 or complete any Flow task.

`wasmhost/tests/cluster.rs` already excluded its two cluster scenarios when either `diagnostic-fuel-off` or `guest-profile` was enabled. The helper refuses cluster execution in both modes (`src/cluster/execution/mod.rs`). The private fixtures and imported inspection decoder nevertheless remained compiled, causing guest-profile all-target Clippy to reject 19 unused items. An existing crate-level diagnostic-fuel-off `allow(dead_code, unused_imports)` concealed the same ownership mismatch for that mode.

The repair moves the exact existing two-feature condition to the test crate root, removes the two redundant per-test conditions, and removes the blanket lint allowance. No production source, expected result, assertion or lint exclusion is changed. No feature condition is added beyond those already applied to the two cluster scenarios. The shared `profile_stderr` self-test in this crate follows the crate gate too; the common helper remains used by other test crates.

## RED evidence

Before this edit, the following command exited **2**:

```sh
cd /home/agent/wt/fn227-map-order && env CARGO_BUILD_JOBS=2 CARGO_TARGET_DIR=/home/agent/.cache/fn219-target/default make -C tools/gomad/internal/backends/wasm lint-rust
```

Formatting and the default, conformance, diagnostic-profile and diagnostic-fuel-off all-target Clippy modes passed. The guest-profile mode failed with dead-code errors for these 19 items:

- `tests/cluster.rs`: `GUEST`, `model`, `model_inputs`, `module`, `drive`, `drive_raw`, `broker_config`, `network_evidence`, `result_of`, `order_guest`, `order_guest_terminal`, `write_module`, `world_evidence`.
- `tests/common/inspection.rs`, as imported by `tests/cluster.rs`: `take`, `word`, `number`, `handle`, `witness`, `boundaries`.

## GREEN evidence

After the edit, the exact command above exited **0**. It ran pinned rustfmt in check mode and the Makefile's complete Clippy matrix, each with `--release --locked --all-targets --no-deps` and warnings denied:

| Feature selection | Result |
| --- | --- |
| default (empty feature string) | Exit 0 |
| `conformance` | Exit 0 |
| `diagnostic-profile` | Exit 0 |
| `diagnostic-fuel-off` | Exit 0 |
| `guest-profile` | Exit 0 |
| `guest-profile,diagnostic-fuel-off` | Exit 0 |
| `stress-jitter` | Exit 0 |

The exact expanded loop remains in `tools/gomad/internal/backends/wasm/Makefile`; no mode, target or warning class was removed.

The supported default cluster integration suite also exited **0**:

```sh
cd /home/agent/wt/fn227-map-order && env CARGO_BUILD_JOBS=2 CARGO_TARGET_DIR=/home/agent/.cache/fn219-target/default /home/agent/.cargo/bin/cargo +1.99.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --test cluster
```

It reported three tests passed, zero failed: `profile_stderr::llvm_profile_diagnostics_are_rejected`, `manual_inspection_reaps_waiting_peer_and_preserves_never_started_node`, and `cluster_broker_runs_recorded_go_orders_with_central_faults`. Build time was `14.30s`; test execution was `0.02s`.

`git -C /home/agent/wt/fn227-map-order diff --check` exited **0**. Evidence consists of command-tool output in the worker session; no separate raw log files were retained.

## Identity and handover

Only `wasmhost/tests/cluster.rs` changes outside this evidence artifact. This edit does not change the Go model digest or Rust production `src` manifest from the network-pin checkpoint, but it does change tracked helper test source and therefore the conductor's full source manifest must be refreshed before qualification.

After the default cluster test command completed, `sha256sum /home/agent/.cache/fn219-target/default/release/gomad-wasmhost` exited **0** and reported `ae4c7f9e2b7cc07d8d2490d3da755436a53f5b39f7d7213cee7cd6665695462f`. This supersedes the earlier local executable observation `50ccc4f2bd274ea327b0795d1a0c66f20ce83aa240f366334f7985aff0c37349` from the preceding release build in the fn-227 source artifact; the cargo test command rebuilt that output path. These are local build observations, not qualification receipts.

The conductor separately owns the unchanged-source Go helper corpus guard mismatch and its regression, integrated all-corpus execution, independent review, current-candidate qualification and Flow completion. The worker does not claim that this worktree's broader Go helper corpus suite passes before that separate fix is integrated.
