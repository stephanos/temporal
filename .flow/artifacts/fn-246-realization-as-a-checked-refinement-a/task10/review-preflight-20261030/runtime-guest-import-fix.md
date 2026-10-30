# Runtime guest import correction

The aggregate retained-history guard introduced two `std::collections::HashSet::new()` values. Their default `RandomState` requested WASI entropy. Final guest admission reproducibly rejected the resulting control and Kettle modules with `model import forbidden: wasi_snapshot_preview1.random_get [127 127] -> [127]`.

The checker already uses seedless hashing for full-state identity (`backends/native/src/visited.rs`). The runtime fix uses the standard library's `BuildHasherDefault<DefaultHasher>` for both pointer-identity sets, avoiding an additional dependency. Fallible reservations, unique retained edge/call counting and host import policy are unchanged.

`tests-import-inventories.log` preserves the exact comparison. The rejected control module imported `random_get`, `environ_get`, `environ_sizes_get`, `fd_close`, `fd_seek`, `fd_write`, and `proc_exit`. Newly built control and Kettle modules import only the latter six functions. The rejected module remains in `tests-rejected-random-import-control.wasm`; the original failure is preserved in `tests-go-frontier-final.log`.

Post-fix actual Go host/runner frontier, whole-prefix and native/guest parity passed (`tests-go-frontier-deterministic.log`). Fresh native runtime 72 tests, shared traversal 4 tests, Kettle 16 tests and strict runtime/checker all-target clippy passed (`runtime-{tests,checker-search,kettle,clippy}-deterministic.log`). No host capabilities were added.
