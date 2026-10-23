# Rust host

This single workspace contains `gomad` and the linked `gomad-wasmhost` library,
with one `Cargo.lock` and the shared `rust/target` output directory (or explicit
`CARGO_TARGET_DIR`). Only `gomad` is a default workspace member. The standalone
protocol binary requires `reference-helper` and exists for oracle fixtures; it
is never the installed host or a qualification producer. Each
linked execution creates a fresh engine and store, with fuel, memory, output,
evidence and a guest epoch watchdog. The public seam is
`gomad_wasmhost::linked::{Request, Limits, ModelInput, Execution}`; modeled guests
use the existing sealed namespace and model implementations. Replay bytes enter
the helper's existing preflight validator, never a host-projected decision tape.
`Prepared::network` attaches the standalone stream-network owner before execution,
including the retained node-local partition, heal, delay and named reset schedule.
Each prepared execution owns fresh network and World state; network evidence is
bounded standalone JSON, while clusters keep their shared owner and binary evidence.

The process-wide deadline backstop is armed before compilation and stays armed
through teardown. Cooperative cancellation first stops work; if compilation or
cleanup cannot honor the bound, the backstop exits the CLI with the infrastructure
classification. It covers all active slots without per-execution subprocesses.

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
Installation, discovery, prepare-only planning and journal recovery use the Rust
owners documented in [the maintainer CLI guide](../CLI.md#rust-installation-and-discovery).
The
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

Fresh same-seed host perturbations use the explicitly admitted [stress producer](STRESS.md). Its full frozen-candidate matrix remains a separate acceptance gate.
## Native measurement preservation

`gomad-measure` retains the measurement input catalogue and explicit `--repo`,
per-producer input flags, `--keep` and rerun records. Build it with the ordinary
workspace toolchain and select `--list` to inspect the declared inputs. Producers
prepare and execute Rust-lineage modules; they refuse historical Go host artifacts
and receipts. Diagnostic, sampled-guest and fuel-off variants require their
actual Cargo features and independently admitted executable identities.

The internal `--linked-run REQUEST OUTPUT` command runs an explicit standalone or
cluster request and retains its modules, actual terminal outcomes and raw evidence.
`--retained-linked-run REQUEST ROOT [--keep]` manages the same execution beneath
`ROOT/.gomad-measure/linked-profile`. Success publication is atomic; failed runs
retain diagnostics without advancing the latest successful run. Input leases and
explicit keep markers prevent collection of evidence that is still in use.

These native fixtures and source ports preserve behavior for fn-232.13. They do
not establish the complete original test-wasm predicate coverage or final frozen
runtime/workload/replay acceptance. Actual fn-185.2/.3 profiles follow fn-232 in
the milestone order. The separate stress spec interface is documented in
[STRESS.md](STRESS.md); catalogue-style stress preparation remains pending.

## Linked host attribution (fn-185.1)

`gomad-profile` is a separate diagnostic producer, built with
`--features helper-diagnostic-profile --bin gomad-profile`. The ordinary `gomad`
entry point refuses commands in that feature build, including qualification.
Empty-feature builds omit `gomad-profile` and the diagnostic collector.
The ordinary `gomad-measure` retains native execution evidence without enabling
diagnostic counters or sampling.

```sh
CARGO_TARGET_DIR=/absolute/private/profile-target cargo +1.99.1 build \
  --manifest-path tools/gomad/rust/Cargo.toml --locked --release \
  -p gomad --features helper-diagnostic-profile --bin gomad-profile
/absolute/private/profile-target/release/gomad-profile request.json /absolute/new-run
/absolute/private/profile-target/release/gomad-profile check /absolute/new-run/profile.json
```

Input is explicit JSON with `kind: "standalone"`, `guest` and optional `network`,
or `kind: "cluster"`
with nodes, broker, network, incarnations, steps, boundaries, crash groups,
`wall_millis` and `evidence_bytes`. Each guest supplies an absolute module path,
its actual SHA-256, the ordinary execution limits, and optional model, namespace
and storage inputs. A model supplies configuration, arguments, environment and
optional replay bytes. Cluster incarnations bind a handle, guest and optional
network model. The executable corpus in `gomad/tests/profile_corpus.rs` constructs
both input forms; production selections use their actual guest, model and original
bounds, without fixture controls or environment switches.

The output directory must be new. It retains the request, terminal outcomes,
actual prefix streams and hashes, raw bounded profile, host/linked implementation
identities and validation result. Cluster runs additionally retain actual steps,
World/network/inspection streams, virtual cutoff and per-incarnation outcomes and
streams. Diagnostic cluster watchdogs also retain the actual bounded World history;
the existing World and aggregate evidence limits still apply. A missing World
cutoff or exhausted bound is a refusal, leaving prefix proof incomplete.
Fuel, capacity, traps and watchdog terminal outcomes finalize the same
capture; status 0 means the capture and collector completed, with guest termination
recorded separately. A failed collector returns status 3 after retaining the profile
and report.
An abrupt process kill or deadline-backstop exit cannot finalize a capture and
leaves incomplete evidence, which does not qualify a run.

Each capture owns its sink. Cluster store threads explicitly join that capture;
concurrent requests share no capture counters. `abi_ops` and `owners` retain exact
operations, distributions and exclusive wall, and the Rust collector independently
checks per-store and aggregate reconciliation, exact field inventories, CPU sample
bounds and checked legacy ABI totals. Raw check input rejects duplicate JSON keys
and reports at or above 48 KiB. Legacy residuals still subtract only the named ABI
buckets; `abi_excluded` participates in op-total equality while remaining excluded
from the original residual equation. Reconciliation uses separately timed guest
gaps. A bounded thread-row overflow is
retained as `threads_omitted` and fails validation. Report-byte overflow drops
thread rows, then operation rows, retaining `report_rows_dropped`; any drop fails
validation. `elapsed_wall_nanos` is one raw
linked capture interval through admission, compilation, execution and teardown;
request/module file reads and output publication are outside it. Cluster execute/owner spans are
aggregate store thread-time and can overlap; they do not supply elapsed shares or
an Amdahl bound. `instrumentation` and `handoff_wait` remain separate from reducible
host work. The retained request and invocation supply the exact run inputs; use a
fresh output directory for each repetition.

These diagnostics do not establish release preservation. The single-guest activity
release-witness proof and fn-185 R4 remain pending actual identity-bound receiving
evidence from fn-232.13. CPU sampling, sweep measurements and the optimization
verdict belong to fn-185.2/.3.
