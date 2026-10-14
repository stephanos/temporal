# Volume conformance corpus

These fixtures run the generated Go `internal/model/volume` projection through
the real WASI storage environment. `TestVolumeConformanceCorpus` replays the
scripted, seeded and recorded-operation cases and rejects the first drift with
seed and step. `TestVolumeConformance` uses the existing fn-175 line transport
to compare the same script with Rust. The Rust selector currently returns
`unsupported`; the Go differential explicitly skips, or fails when
`GOMAD_WASM_VOLUME_CONFORMANCE_REQUIRED=1`. Neither is an acceptance pass.
`wasmhost/tests/volume_corpus.rs` is ignored until the volume owner is connected
to the actual namespace and fault routing.

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

`image_identity` is `gomad3.volume-recovery-image/v1`; `image_base64` retains the
entire self-contained binary image and `image_digest` is its SHA-256. The
test-only Go encoder reads the real full state through the locked, deeply copied
`volume.FS.ConformanceState` accessor, which is available only under `test_dep`.
This replaces the incomplete snapshot/occurrence JSON identity in this oracle;
production Go storage image handling is unchanged until its owning migration.
The namespace digest remains fn-175 v1. Pure Rust model and image tests do not
qualify the still-unsupported namespace/fault differential.

Image v1 starts with `GMVOLIMG`, little-endian u32 version 1 and section count 3.
Each section has a little-endian u32 tag and u64-length-prefixed bytes, in order:
model identity (tag 1), sorted fault occurrence key/count pairs (tag 2), and
complete filesystem recovery state (tag 3). State carries nextInode, generation,
and volumes sorted by ID: mount/capacity, persisted objects/paths/owned inodes,
pending and history operations with full effects, nextOperation, and sorted last
store key/operation pairs. Scalars use u64 little-endian; bytes/UTF-8 strings and
lists have u64 lengths/counts; signed times retain their two's-complement bits.
Object presence is u64 0/1. Every orphan object and effect payload is retained.
Namespace effect, dependency, store, pending and history sequence order is kept.

`vectors/core.json` additionally retains 39 pure Go API steps covering signed
metadata times, rename dependencies, detached handles, seeded selection,
file/directory sync, clone/crash/graceful lifecycle counters and two full images.
The nested partial-persistence vector intentionally retains `/dir/db` without
`/dir`: a valid dependency-closed selection need not persist the parent
namespace operation. Rust must preserve that flat durable state, not reject it
as a malformed tree.

The handwritten `test_dep` accessor and these encoder/vector tests are new
source-identity inputs for a future qualification. They do not change generated
projection bytes, native overlays, or production-tag APIs.

Regenerate into an existing external directory for review:

```sh
go test -tags test_dep -p 1 -run '^TestVolumeConformanceCorpus$' ./internal/backends/wasm/wasi -args -volume-conformance-output=/absolute/review/corpus
```

Copy reviewed fixture changes into this tree through the normal patch workflow.
The source-only preparation does not qualify Rust, native execution, or fn-176
acceptance, and does not complete its fn-175 prerequisite.
