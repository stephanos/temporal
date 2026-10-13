---
satisfies: [R1, R2]
---
# fn-175-gomad-filesystem-namespace-in-the-rust.3 Rust namespace data ops, fd 1/2 output accounting and fd-subscription polling

## Description
Port the data-path operations and fd-subscription polling into the Rust namespace (R1, R2): reads, writes, seeks, sizes, sync, append, fd 1/2 routing with output accounting, and `poll_oneoff` fd subscriptions. These are the hot calls (≈ 6.7k `fd_pwrite`, ≈ 390 `fd_pread` per activity run).

**Size:** M
**Files:** `tools/gomad3/backends/wasm/wasmhost/src/namespace.rs`, `tools/gomad3/backends/wasm/wasi/testdata/namespace-conformance/` (extra scripts), `tools/gomad3/backends/wasm/wasi/namespace_conformance_test.go`
**Touches:** [tools/gomad3/backends/wasm/wasmhost/src/namespace.rs, tools/gomad3/backends/wasm/wasi/namespace_conformance_test.go, tools/gomad3/backends/wasm/wasi/testdata/namespace-conformance/**]

### Approach
- Port `fd_read`/`fd_pread` (pread also needs Seek; `Isdir`; `Inval` when offset > MaxInt64; only `fd_read` advances the offset).
- Port `fd_write`/`fd_pwrite` with Go's exact order (namespace.go:481):
  1. `capacity("write-buffer")` above 4 MiB
  2. fd 1/2 offered-output charge
  3. getFD (Write; pwrite also Seek)
  4. `Isdir`
  5. `Rofs`
  6. append-flag offset
  7. growth via `resize` with `capacity("filesystem-bytes")`
  8. `modified = now` when data is non-empty
- fd 1/2 bytes go to stdout/stderr buffers owned by the namespace. Keep Go's offered-output semantics: rejected writes are still charged, as pinned by `TestOfferedOutputChargesRejectedWrites…`, environment_test.go:488.
  - Decision: the Go namespace is the reference for offered output. The ABI-level `output_bytes` reservation/refund in `abi.rs` stays the transport bound. fn-175.5 reconciles the terminal `output_bytes` check (process.go:791) so both counts are defined and asserted.
- Port `fd_filestat_set_size`, `fd_seek` (whence and over/underflow rules, `Inval` on dirs and std fds) and `fd_sync` (no-op without storage, Sync right).
- Port fd-subscription polling from `clockCall` (wasi/clock.go):
  - type 1/2 subscriptions need Poll|Read or Poll|Write; their errno goes into the event
  - a directory gives event `Inval`
  - a read on a regular file has nbytes = size − offset, or the hangup flag at EOF
  - a write has nbytes 0
  - fd subscriptions complete at the current `now` and advance no time
- Mixed clock and fd subscriptions must produce the same event order and clock advance as Go. Expose an fd-readiness helper that the existing local clock poll (model.rs:792) can combine with, rather than a second poll implementation.

### Investigation targets
**Required:**
- `tools/gomad3/backends/wasm/wasi/namespace.go:340-661` (data ops, resize)
- `tools/gomad3/backends/wasm/wasi/clock.go:28-150` (poll subscription handling)
- `tools/gomad3/backends/wasm/wasi/environment_test.go:410-535` (capacity, offered output, captured stdin)
- `tools/gomad3/backends/wasm/wasmhost/src/model.rs:792-848` (clock poll to combine with)

## Acceptance
- [ ] The full conformance corpus passes: all ops, including mixed clock and fd `poll_oneoff` scripts and every write/read capacity and errno script.
- [ ] Rejected fd 1/2 writes are charged identically on both sides, and positional writes to fd 1/2 follow Go's routing.
- [ ] Seeded generated sequences (≥ 10k ops across ≥ 100 seeds) run clean in `make -C tools/gomad3/backends/wasm test-conformance`.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
