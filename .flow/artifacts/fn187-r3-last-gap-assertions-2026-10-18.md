# fn187 R3: final mapped-gap assertion replacements

Source review, 2026-10-18. Worker root: `/home/agent/wt/wasm-fn187-functional`.
Paths below are relative to `tools/gomad/internal/backends/wasm` in that worker.
All three original roots remain intact. This ledger does not authorize their
deletion or replace the required complete coverage/feature/review gates.

## No-snapshot replay

Original: `wasmhost/src/model.rs`,
`model::tests::dormant_predecision_leaves_tape_mismatch_to_ordinary_runtime`.
Parent SHA256 `4b12b639edc06e954829350e8ea403757685a97c880e56de4db81ece2431148d`.

Replacement: `wasmhost/tests/testdata/model_default_cut/dormant-predecision-before-any-snapshot.json`,
SHA256 `3adb310ceabea393331f4819195983c5da83190167198d9676e71b525d8cb4ec`.
Consumer: `model::tests::default_cut_corpus::rust_model_default_cut_corpus` in
`wasmhost/src/model/tests/default_cut_corpus.rs`, SHA256
`386f2ecd1e5160669cc8fd640f3b22b228920205bced01c4a363b7ad8eb3a2ce`.

The consumer uses the original `model()`, `bootstrap`, `debug_predecision()`
and `decision()` helpers. The fixture binds mode2, `Some(empty)` typed tape
and actions bootstrap/debug/runtime, with no snapshot. It assigns the fixture's
actual tape to the model, not a separately hardcoded substitute.

| Original check | Exact fixture field | Consumer check |
| --- | --- | --- |
| debug exchange succeeds (`unwrap`) | `expected.debug_success=true` | Immediate equality of actual debug result |
| no divergence after debug | `expected.divergence_after_debug=false` | Immediate observation before runtime |
| runtime decision returns error | `expected.runtime_error=true` | Actual original runtime call's error state |
| divergence after runtime | `expected.divergence_after_runtime=true` | Immediate post-runtime observation |

The complete four-field Expected value is also compared. The 20-control guard
rejects altered setup, action order, tape, stage outcomes, unknown fields and
trailing JSON. Actual intended guard RED101, restored GREEN guard/corpus/original
all pass. The genuine corpus profile hits inspection.rs532–538, count1 each;
owning Recorder::boundary and default-Cut closure both have positive FNDA.
Ordinary helper execution snapshots before debug and is not an equivalent
replacement for this precise test.

## Typed storage-error mapping

Original: `wasmhost/src/storage_transport.rs`,
`storage_transport::tests::crash_image_failures_are_classified_with_their_message`.
Parent SHA256 `b9fa7a0e1bcd3d969107cfa57c66fe8b1852186204d32d7d19afc2fe28183fcd`.

Replacement: `wasmhost/tests/testdata/storage_crash_failure/original-error-mapping.json`,
SHA256 `b997bea7b0ddb25ff11ac66b9aa725346a8f5e17ba49d4d13de7ec5f1e0f3d52`.
Consumer: `storage_transport::crash_failure_corpus::rust_storage_crash_failure_corpus`,
`wasmhost/src/storage_transport/crash_failure_corpus.rs`, SHA256
`012f3a810fcbc5b6f657b039c6a8aa67461eec44cc7ed308c9d44b4e59769efa`.

| Exact original input | Named case | Complete expected tuple |
| --- | --- | --- |
| Capacity(resource=image-bytes, required3, maximum2) | image-capacity-preserves-classification | capacity; WASI capacity: storage-image-bytes |
| Invalid(crash selection contains an unknown operation) | unknown-operation-preserves-infrastructure-message | infrastructure; crash selection contains an unknown operation |

The strict literal binding validates the input strings before constructing
the original static-string error variants. The consumer calls the actual
`crash_failure` and compares its complete tuple for each case. Its 19-control
guard includes schema-valid variant swaps, changed tuple values and roster
changes. Actual intended guard RED101, restored GREEN guard/corpus/original
all pass. Isolated genuine-corpus coverage hits infrastructure mapper491
(historical488 shifted only by the three-line test hook), count1; crash_failure
FNDA2. This is a Rust-only typed-error corpus, not a Go oracle or a claim that
the defensive Invalid error is reachable through a valid helper request.

## Go pending-open absence

Original: `wasi/volume_test.go`, `TestStorageCrashFaultAndModelLimits`.
Parent SHA256 `fd8ad3626decd47f18a2627e1d422d5f8a8839b1018b12add7767457b494fa97`.

Replacement: `wasi/testdata/volume-conformance/vectors/pending-open-namespace-absence.json`,
SHA256 `0953f3bea55166b2721d064fc7ea4b8641187f49e8c08c8a541f4473327a9271`.
Consumer: `TestVolumePendingOpenNamespaceAbsenceCorpus`,
`wasi/volume_pending_absence_test.go`, SHA256
`a9ec831af1e2fd1c2190c0be855e02405da4df4d5193034cc1ffc439ef018a49`.

The consumer uses original `storageConfig()`, PendingOperations1, FD4, path db,
create flag1 and rightsAll. It executes the genuine volume-conformance driver
and observes the namespace immediately after the sole operation.

| Original condition | Exact replacing observation |
| --- | --- |
| error is BoundaryError | genuine Handle error classification serialized as boundary reply by existing driver |
| boundary.Kind equals capacity | exact reply `{"boundary":"capacity:storage-pending_operations"}` |
| lookup(/workspace/db) is nil | `probe_path=/workspace/db`, `absent_after_operation=[true]`, direct same lookup after operation |

The existing RawMessage-aware complete corpus comparison additionally checks
all observations, not only the newly added absence field. Twelve negative
controls reject changed path/rights/limit/op/reply/absence and malformed inputs.
Actual intended guard RED1, restored GREEN, two byte-identical external
generations, canonical three-root replay and isolated corpus-only coverage pass.
The isolated profile hits namespace.go247.4–248.1 count1. This vector stays in
a separate Go-only subdirectory; no shared Rust JSON schema is expanded.

## Evidence and limits

Receipts under `/home/agent/wt/evidence/fn187/r3-current`:

- `model-default-cut-proof-2026-10-18/evidence.json`, SHA256
  `be2302b296bef4870ee43b8d61b0799e94e2bcb1122fecc4b6fbe2cf9d589808`.
- `storage-crash-mapper-proof-2026-10-18/evidence.json`, SHA256
  `b02aa9555f72c70db8e1fa365236d8743c42788e8b4f8deaeb6b0ed42ac3c204`.
- `lookupnil-go-vector-runtime-2026-10-18/evidence.json`, SHA256
  `f07f05d37283f80c1120730f7317d8db301885e1bf9628314f587f2ee81f0852`.
- `default-cut-mapper-corpus-lines-2026-10-18/evidence.json`, SHA256
  `cf7faab53ef2182a89209fdd6656001a32d5b0bf1eccbf6f1ed54b1b3cba13c8`.

The last receipt is single-object focused evidence, not a common-source complete
suite/union certificate. Root read the original bodies, full new consumers and
JSONs, actual selected test logs and target coverage rows. The frozen d42f
binary and both raw pools are hash-bound; its model pool contains only the
genuine model corpus run, and its mapper pool only the genuine mapper run.
No counters from original tests or guard tests supply these focused line hits.
