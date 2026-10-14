---
satisfies: [R1, R2]
---
# fn-175-gomad-filesystem-namespace-in-the-rust.2 Rust namespace core: tree, descriptors, path resolution and metadata/path ops

## Description
Port the Go namespace's structure and metadata/path operations into a pure Rust module, with no Wasmtime or protocol types, driven by the fn-175.1 harness (R1, R2). Data-path operations follow in fn-175.3. The split keeps each change reviewable against one half of `namespaceCall`.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/namespace.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/main.rs` (mod), `tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs` (wire the `namespace` model)
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/namespace.rs, tools/gomad/internal/backends/wasm/wasmhost/src/main.rs, tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs]

### Approach
- Mirror the Go types: `node` (ino, directory, readonly, data, children ordered by name, modified), `descriptor` (node, offset, rights, inheriting, flags, preopen, std) and `namespace` (root, fd table, `nextFD` from 3, `nextInode` from 1, files, bytes). Use ordered containers (`BTreeMap`) everywhere iteration order can reach replies or digests.
- Port `newNamespace` construction from a pre-validated, flattened input (the fn-175.4 image shape: directories, read-only files with bytes, writable dirs, cwd, stdin, limits). Include preopen ordering, std fd rights and the `capacity("preopen-descriptors")` and `capacity("initial-namespace")` checks.
- Port `resolvePath` (namespace.go:259) exactly: stack of nodes, `..` containment, trailing-slash rule, `Notcapable` for absolute, NUL, >4096 bytes or escape.
- Port these operations with Go's check order and errnos: `fd_prestat_get`, `fd_prestat_dir_name`, `fd_fdstat_get`, `fd_filestat_get`, `fd_close`, `fd_fdstat_set_flags` (unsupported message for flags outside `1|4`), `fd_readdir` (sorted entries, 24-byte dirents, cookie rules), `path_open` (the 11-step check order at namespace.go:674), `path_create_directory`, `path_remove_directory`, `path_unlink_file`, `path_filestat_get`, `path_rename` (`containsNode` cycle check, same-name no-op).
- Unsupported operations (`path_readlink`, `path_symlink`, `sock_*`) return the Go unsupported message verbatim.
- Times: `atim`/`mtim`/`ctim` = epoch + `node.modified`; the caller supplies `now` per operation. Do not read any host clock.
- Implement the fn-175.1 state digest on the Rust side.
- Typed API, roughly one method per op returning `(errno, typed output)` or `Err(Boundary{kind, dimension})`. fn-175.5 adapts it to fn-174's typed imports.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/namespace.go:107-340` (types, construction, lookup, getFD, resolvePath, stat)
- `tools/gomad/internal/backends/wasm/wasi/namespace.go:674-971` (open, pathOperation, readdir, rename, containsNode)
- `tools/gomad/internal/backends/wasm/wasi/environment_test.go:39-239` (rights/offset/cookie/rename/captured cases)

**Optional:**
- `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs:792-848` (existing local poll as Rust style reference)

### Key context
- The Go `BoundaryError` aborts the run and is not a reply; keep that distinction in the Rust result type.
- Storage-related branches (`storagePath`, `storage` handles) are out of scope; fn-176 adds them.

## Acceptance
- [ ] The conformance corpus for all metadata/path ops (scripted, generated and recorded subsets restricted to these ops) passes with identical replies, errnos and digests.
- [ ] `cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked namespace` passes (unit tests for resolvePath containment, open check order and readdir cookies).
- [ ] No Wasmtime, serde_json or protocol types in `namespace.rs`'s public API.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
