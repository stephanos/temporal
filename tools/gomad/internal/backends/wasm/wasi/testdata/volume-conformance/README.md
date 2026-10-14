# Volume conformance corpus

These fixtures run the generated Go `internal/model/volume` projection through
the real WASI storage environment. `TestVolumeConformanceCorpus` replays the
scripted, seeded and recorded-operation cases and rejects the first drift with
seed and step. `TestVolumeConformance` uses the existing fn-175 line transport
to compare the same script with Rust. The Rust selector currently returns
`unsupported`; the Go differential explicitly skips, or fails when
`GOMAD_WASM_VOLUME_CONFORMANCE_REQUIRED=1`. Neither is an acceptance pass.
`wasmhost/tests/volume_corpus.rs` is ignored until fn-176.2 implements the owner.

The session has `model: "internal/model/volume"`, the fn-175 `image`, and
`storage` using the Go `StorageModel` fields and JSON spellings. Every operation
has `op`, `input`, and explicit virtual monotonic `now`, used before any effect.
Filesystem records reuse fn-175 WASI inputs. Storage controls are:

| Operation | Input |
| --- | --- |
| `enumerate` | `volume`, Go `CrashEnumerationLimits` as `limits`, optional saved `cursor` name, `save` name |
| `select` | `seed`, `save` name for the selected per-volume operation lists |
| `crash` | explicit `selections`, or saved `selection`, or `page` with zero-based `state`; `image` name |
| `recover` | previously saved `image` name; fresh environment with original session configuration |

Every successful session starts with `{}`; rejected configurations return the
Go boundary record. A control operation's reply is `{}` or an error record.
Filesystem replies use fn-175's exact errno/output/unsupported/message encoding.
Modeled crashes use `storage_crash`; classified boundaries use `boundary`;
projection errors retain `error`. Each subsequent driver reply is one complete
observation, with exact reply bytes and appended storage JSON lines stored as
base64, snapshots, operations sorted by volume ID, full enumeration (including
state identities, pending digests, completion/capacity, and exploration cursor),
selected operations, image digest and namespace digest. Recovery resets the
event stream and descriptor incarnation while continuing fault occurrences.

`image_identity` is explicitly
`gomad3.go-storage-image-snapshots-occurrences/v1`: the current Go identity,
SHA-256 of the Go snapshot/occurrence JSON. It is provisional until fn-176.3's
self-contained binary recovery image encoder; it does not claim to authenticate
every recovery-relevant field. The namespace digest remains fn-175 v1.

Regenerate into an existing external directory for review:

```sh
go test -tags test_dep -p 1 -run '^TestVolumeConformanceCorpus$' ./internal/backends/wasm/wasi -args -volume-conformance-output=/absolute/review/corpus
```

Copy reviewed fixture changes into this tree through the normal patch workflow.
The source-only preparation does not qualify Rust, native execution, or fn-176
acceptance, and does not complete its fn-175 prerequisite.
