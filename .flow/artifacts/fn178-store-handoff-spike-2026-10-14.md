# fn-178.1 store handoff spike (2026-10-14)

Throwaway spike for the fn-178 threading decision. Source:
`.flow/artifacts/fn178-store-handoff-spike/` (`Cargo.toml`, `main.rs`). It is an
out-of-tree crate because variant (b) needs the wasmtime `async` feature, which
the helper does not enable. Built with `cargo +1.94.1 build --release --offline`
against the helper's `Cargo.lock` (wasmtime 47.0.3, same engine configuration as
the helper: Cranelift, fuel, canonical NaNs, no native unwind info, relaxed
SIMD/memory64/multi-memory off). Binary SHA-256
`9b32924146f59d25c7142f155ccf6b3838ec957e72f1060604b49e00d8960515`.

## Guest and method

Guest: `(func (export "run") (param $n i32) (loop call $hop ... br_if))`, one
host call per iteration, 200,000 iterations per store, two stores. Every host
call hands control to the other store. Each variant runs 7 rounds, interleaved
so host-load drift hits all variants alike; the reported value is the median.

- (a) one OS thread per store (8 MiB host stack), stores built from one shared
  `InstancePre`, typed sync `func_wrap` import that passes a baton:
  `a_condvar` uses `Mutex<turn>` plus one `Condvar` per store; `a_park` uses an
  atomic turn plus `thread::park`/`unpark`.
- (b) both stores on one thread, `func_wrap_async` import whose future returns
  `Pending` once, `call_async`, and a round-robin poll loop.
- Import cost without handoff: a typed sync import from a sync store and from an
  async (fiber) store; an async import that is immediately ready.

## Host

Owner's Linux/arm64 host (`Linux 7.0.14 aarch64`, 12 vCPU, 15.8 GiB). Not a
quiet window: another worker was running heavy WASM gates (load averages below).
Treat the absolute thread numbers as an upper-side indication; the a/b ratio is
the decision input.

## Results (ns, median of 7)

| Measure | Run 1 (load 17.3 → 7.4) | Run 2 (load 3.1 → 4.2) |
| --- | --- | --- |
| (a) threads, Mutex+Condvar, per switch | 13,557.4 | 12,827.9 |
| (a) threads, park/unpark, per switch | 12,514.1 | 12,300.3 |
| (b) async fibers, per switch | 157.2 | 161.0 |
| typed sync import, sync store, per call | 2.0 | 2.0 |
| typed sync import, async store, per call | 2.0 | 2.0 |
| async import, never yields, per call | 22.0 | 22.1 |

Raw rounds (run 1):
`a_condvar` 13798.1, 12279.3, 12077.8, 14221.7, 17565.0, 10661.2, 13557.4;
`a_park` 12514.1, 11308.8, 11555.2, 19376.2, 16574.5, 12973.1, 9049.9;
`b_fibers` 157.8, 157.2, 160.7, 155.1, 153.5, 154.3, 378.5;
`sync_plain` 1.90, 2.94, 1.99, 1.98, 1.98, 2.03, 2.07;
`sync_fiber` 2.08, 2.10, 2.00, 2.01, 2.03, 2.12, 1.99;
`async_ready` 22.30, 21.96, 21.98, 21.26, 21.73, 23.29, 21.94.

Raw rounds (run 2):
`a_condvar` 12534.4, 6318.0, 12827.9, 19590.0, 12499.0, 22934.4, 34023.5;
`a_park` 10085.8, 12300.3, 21733.8, 11597.8, 15342.7, 18793.5, 2871.8;
`b_fibers` 459.1, 147.8, 161.0, 151.7, 149.3, 322.1, 172.8;
`sync_plain` 2.01, 1.89, 3.00, 1.91, 1.97, 1.98, 1.97;
`sync_fiber` 2.05, 1.94, 2.00, 2.07, 2.18, 2.07, 1.99;
`async_ready` 33.45, 21.83, 22.09, 21.60, 22.95, 116.30, 21.65.

## Reading

- A thread baton switch costs about 12–14 µs here, dominated by futex wake and
  reschedule latency in the VM; fibers switch in about 0.16 µs (about 80x less).
- Typed sync imports are unaffected by the store kind (2.0 ns either way).
  Under (b) only imports that may hand off must become async; those cost about
  20 ns more per call plus a boxed future.
- Enabling wasmtime `async` in the helper adds four `Cargo.lock` packages
  (`wasmtime-internal-component-macro`, `wasmtime-internal-wit-bindgen`,
  `wit-parser`, `id-arena`); `wasmtime-internal-fiber` and `async-trait` are
  already locked.
- Handoffs in the Go broker happen only at `runtime_idle` (`prepareIdle`), so the
  cluster's handoff count is the guests' idle count; today each such handoff
  already costs at least one pipe round trip and a process wake.

Decision (recorded in the fn-178 spec's design record): (a) behind the scheduler
seam, no dependency change; fn-178.10 revisits fibers with measured idle counts.
