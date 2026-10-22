# Rust host foundation

This workspace links the existing `gomad-wasmhost` library in-process. The helper
binary still uses that same library for its existing transport protocol. Each
linked execution creates a fresh engine and store, with fuel, memory, output,
evidence and a guest epoch watchdog. The public seam is
`gomad_wasmhost::linked::{Request, Limits, ModelInput, Execution}`; modeled guests
use the existing sealed namespace and model implementations. Replay bytes enter
the helper's existing preflight validator, never a host-projected decision tape.
`Prepared::network` attaches the standalone stream-network owner before execution,
including the retained node-local partition, heal, delay and named reset schedule.
Each prepared execution owns fresh network and World state; network evidence is
bounded standalone JSON, while clusters keep their shared owner and binary evidence.

The watchdog starts before compilation and refuses guest instantiation after an
expired compilation, but pinned Wasmtime's synchronous compiler cannot be
preempted or reclaimed in-process. The old Go process watchdog also bounded
compilation and teardown. That existing hard-wall guarantee is therefore an open
fn232.1 acceptance gap, not a completed bound or an authorized contract change.
No detached compiler thread or per-execution subprocess is used.

The public `inspect` verb reads Rust-lineage artifacts through the shared
authenticated opener. Its JSON shape and mandatory fresh Go/Rust comparison
are documented in [the maintainer CLI guide](../CLI.md#rust-artifact-inspection).
The public `replay` verb uses helper-owned semantic tape admission and typed
standalone/cluster execution. Its inspection and compiled diagnostic-role mapping
are documented in [the maintainer CLI guide](../CLI.md#rust-retained-replay).
Seed `explore`, `resume`, `execute-shard` and `merge` use Rust-owned preparation,
ordered campaign policy, authenticated journals and retained target pools. Their
source preparation and differential lanes are documented in
[the maintainer CLI guide](../CLI.md#rust-seed-campaigns).
The public `minimize` verb reduces retained standalone failures with bounded
attempts, authenticated checkpoints and `--resume`. Its public flags are unchanged;
the JSON result and retained failure-signature semantics are documented in
[the maintainer CLI guide](../CLI.md#rust-minimization).
`qualify --spec` and installation-free `report --result` qualify one WASM cell
on the linked host and reopen its retained Result. Their spec mapping,
linked-process accounting and lanes are documented in
[the maintainer CLI guide](../CLI.md#rust-qualification).
Other public host verbs remain with their later fn-232 tasks. The
`determinism-fixture` feature exposes the foundation, inspect, replay and campaign test drivers;
their commands and fault controls are absent from ordinary builds. Run from the
repository root:

```sh
cargo +1.99.1 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml --features determinism-fixture
cargo +1.99.1 build --release --locked --manifest-path tools/gomad/rust/Cargo.toml --features determinism-fixture
python3 tools/gomad/testdata/parity/foundation-check.py --binary /absolute/target/release/gomad --work /absolute/new-check-root
```

The shared corpus supplies actual WAT execution fixtures and the semantic
comparator examples. Reviewed `goldens-foundation` retain exact canonical records,
identities, hashes and ledger bytes for the pinned Linux/aarch64 build. Golden
regeneration requires `GOMAD_GOLDEN_REGENERATE=1` and first authenticates two fresh
runs. Run regeneration twice and compare every resulting file byte-for-byte.
Other platforms/build configurations need their own reviewed identity lane;
identity fields are never normalized to make a comparison pass.

Records use `gomad.execution-record/v2`. Canonical bytes have sorted keys and
native integer widths, no host paths/timestamps, and a nullable unsigned guest
exit status. Failure grouping hashes the complete record with only its seed set
to zero. Execution identity binds target, toolchain, platform, implementation and
configuration digests. Build identities hash actual source/build inputs, compiler,
target and enabled features; linked records bind both host and helper identities.

Artifacts use a separate storage lineage and refuse historical Go manifests
before decoding. The opener pins private directories, freezes the manifest and
authenticates bounded payload reads on their opened handles. A detached single-link
copy works without source or installation metadata. Store-owned artifacts derive
their module-pool capability from the private parent store's fixed `.store.json`
layout and `.targets` handle, not a manifest path or new public CLI flag. Multi-link
modules must match the contained pool's authenticated content-addressed inode.
Storage envelopes must match the fixed canonical v2 bytes exactly; alternate JSON
encodings refuse, so retained-byte accounting includes their exact on-disk size.

Publication uses real OS locks and handle-relative exclusive directory rename:
Linux `renameat2(RENAME_NOREPLACE)`, Darwin `renameatx_np(RENAME_EXCL)`. Unsupported
exclusive publication fails closed. Payloads, manifests, stage directories and
the store are synced before any reference is returned, including collision reuse.
Crash fixtures terminate the process at the tested boundaries without unwinding;
unreferenced staging directories may remain, but no partial artifact is returned.

The baseline comparator under `testdata/parity/reference` is a separate Go-only
driver built against the authenticated frozen source. It is not guest isolation
for the Rust host. `foundation-reference-check.py` authenticates that baseline
bundle and each fresh raw Go result, independently authenticates the Rust bundle,
then compares concrete termination and guest exit status. Stock-WASI WAT cases
have no cooperative choice imports: baseline seeds label two fresh repetitions,
while the Rust concurrent ledger carries the requested execution seeds.

The replay parity lane uses its frozen Go launcher, matching helper and archived
runtime source for standalone recordings, including all four network cases.
Registered inspection and cluster cases use the separately declared current
producer pair for intentional World/lifecycle changes. These references are not
interchangeable: the newer Go-owned network replay path can refuse input-tape
attestation that the frozen launcher does not require. Its retained status-3
receipts describe that newer-producer regression, not frozen-baseline parity.
