# Guest builds: reproduction and diagnostics

`bash model/scripts/guest.sh <crate> <output> [--check]` builds any cdylib with
the mise-pinned Rust 1.99.1 compiler, `--locked`, `wasm32-wasip1` and `profile.guest`.
It discovers the output through Cargo's artifact messages, including custom library
names. It requires Bash, mise, installed pinned Rust/Cargo, and the `rust-src` component;
active guest/build helpers require no Python.
Current guests declare WHOLE_PREFIX_V1 in their ordinary Reset response. All current
native and wasm producers use the same runtime contract; Host refuses a missing or
unknown declaration before Driver creation. This wire-format change requires rebuilding
guests and current records together; historical guest bytes are not preserved.
The model hash binds an owned copy of the exact loaded guest bytes independently of
the Reset declaration.
`xtask.sh` builds the current native front door and dedicated guest driver in `<target>/xtask-bootstrap` before
installing guest wrappers, flags or cache settings. It prints the absolute binary
path for shell callers. This incremental build also prevents stale cached tools
from bypassing source changes. Whitespace in checkout and tool paths is supported.
The atomic `guest-rustc` launcher is an executable copy of the dedicated driver binary,
selected by its executable name; it preserves macOS loader variables and inherited
jobserver handles without a shebang, `/usr/bin/env` or recursive Cargo invocation.
Exit 0 means built or fresh, 1 means native tool/input/output failure, and 2 means
missing or stale output. Guest compiler exit codes pass through (signals use 128 +
signal); their diagnostics remain on stderr. Staleness reports existing/built sizes and the first differing
zero-based byte offset, including EOF; check mode never updates the output.
Relative output paths are relative to the caller. `CARGO_TARGET_DIR` selects the
build cache, defaulting to `model/target`. Guest compilation uses a versioned
`guest-cache/<identity>` subdirectory beneath it; artifact destinations are unchanged.

The profile inherits release, strips symbols, disables debug information and uses
fat LTO with one codegen unit and checked overflow. The checker's release profile
keeps its profiling symbols. Workspace, Cargo registry, toolchain sysroot and target
paths are remapped by rustc, including generated protobuf locations under `OUT_DIR`.
No wasm bytes are rewritten after linking. The Temporal API generator sorts merged
descriptor files before prost generation; otherwise Go map iteration could change
merged Rust declaration order and the resulting guest bytes.

The private native compiler wrapper derives target symbol identities from the pinned compiler, the
crate's own lockfile closure, normalized package/source paths, crate configuration
and dependency identities. Before building, `xtask guest-lock` digests, for every
workspace-resolved package, the lockfile coordinates and checksums of its non-dev
dependency closure across all platforms; path packages carry no location. The map
is written as `umpire-lock-<content hash>.json` in the guest cache. Adding an
unrelated workspace crate, or a dev-dependency, changes no guest bytes unless it
changes the resolved dependencies of a package in a guest's closure. `cargo metadata`
resolves optional dependencies across the whole workspace, so enabling an optional
dependency or feature of a shared crate still changes the guest. This over-hashes but
never under-hashes; the build's own `--unit-graph` would be an exact per-unit
refinement. Packages that share a name and version across sources, or lack a closure,
refuse the build, as does a registry package without a lockfile entry. Build-std units
carry Cargo's std marker and come from the sysroot or from registry crates in the
sysroot's own `library/Cargo.lock`; they take no workspace closure, since the pinned
compiler fixes theirs.
Cargo retains its own filenames and fingerprints. Atomic per-artifact `.umpire-id`
files carry the portable identities; missing or malformed records fail the build.
Records appear before rustc starts because Cargo pipelines dependent compilations
after metadata emission, and compiler failures remove them. Host proc macros compile
unchanged; their portable package/feature/profile identities include collision
witnesses that refuse conflicting native variants in one target dependency graph.
Each target's `OUT_DIR` remaps to `/generated/<portable identity>`, removing Cargo's
host-specific directory hash from generated-source locations. Native tool binary,
lockfile or compiler changes select a new cache so older artifacts cannot bypass this
evidence; a lockfile change outside every guest closure rebuilds identical bytes.
Existing `RUSTC_WRAPPER` executables are chained. These inputs support the pinned
workspace and toolchain. Changed compiler/dependency inputs require current-input
reproduction, freshness and existing host controls. Platform evidence is explicit;
historical cross-OS certification is not a compatibility gate for the native driver.


Native guest commands execute with `model/` as their working directory:
`xtask guest <crate> <output> [--check]` and
`xtask guest-freshness <crate> <output>`. Shell entrypoints preserve caller-relative
output paths; relative target/cache paths retain their model-relative meaning.
Cargo artifact input is UTF-8 newline-delimited JSON with a string `reason`;
`compiler-artifact` records require a typed `target.crate_types` array and
`filenames` array. Exactly one cdylib `.wasm` path is accepted, including custom
library names. Missing, multiple, malformed or unreadable artifacts fail before
publication. Successful builds publish by sibling rename; checks never publish.
The native v2 closure/sidecar schemas intentionally invalidate Python records.
JSON whitespace, Python TOML quirks, exact old diagnostics and historical guest
bytes are not compatibility contracts. `RUSTFLAGS` accepts whitespace-separated
UTF-8 tokens; `CARGO_ENCODED_RUSTFLAGS` preserves raw OS tokens separated by 0x1f.

## Unwinding on the pinned compiler

The shipped wasm32-wasip1 standard library aborts panics even when a manifest asks
for unwinding. The guest builder rebuilds `std` and `panic_unwind` from the pinned
compiler's installed sources, with command-scoped `RUSTC_BOOTSTRAP=1`,
`-Zbuild-std=std,panic_unwind`, `-Cpanic=unwind`,
`-Ctarget-feature=+exception-handling` and
`-Cllvm-args=-wasm-use-legacy-eh=false`.
This deliberately uses unstable compiler interfaces on a pinned stable compiler.
Rust prints an unstable target-feature warning. No nightly, compiler/std patch,
global bootstrap setting or extra dependency version is introduced. A toolchain
upgrade must repeat the caught-panic and reproducibility controls.

The transport host uses wazero 1.12's existing interpreter with final WebAssembly
exception handling enabled. The same caught-panic bytes failed under its ARM64
compiler in the retained diagnostic; the interpreter passed. Memory bounds,
context cancellation, call serialization, import/export admission and invalidation
after an uncaught trap remain in force. Checker-caught panics refute their step
and leave the instance usable. Go/SDK panic handling is unchanged.

Rebuilt std imports two additional WASI names. Only exact signatures for
`fd_close(i32)->i32` and `fd_seek(i32,i64,i32,i32)->i32` are admitted. Both always
return BADF (8), including standard descriptors and arbitrary offsets, and do not
access descriptors or mutate output memory. The host installs no general WASI
module. Permanent registered-import controls verify denial and unchanged memory;
wrong signatures, namespaces and other imports remain refused.

## Permanent guest-profile controls

`make model-check` builds stock, stateful, Temporal and caught-panic/overflow
control guests through the same builder. Its native control executable evaluates
the existing checker Tiny EffectPanic/Overflow fixtures and checks their existing
report goldens. The production Go host compares full native and wasm responses,
repeats calls and resets each instance. Existing stock report/ordered-row goldens
and uncaught trap/invalidation controls run separately. This tests actual unwind
artifacts, rather than inferring unwind from the profile declaration.

From the repository root:

```sh
cd model/umpire && mise exec -- cargo test -p xtask -p umpire-guest-driver --locked
bash -n model/scripts/guest.sh model/scripts/tests/build-guests.sh
CARGO_TARGET_DIR=/absolute/task-owned-target bash model/scripts/tests/build-guests.sh
CARGO_TARGET_DIR=/absolute/task-owned-target make model-validate
```

`model-validate` executes one TLC-required full gate and one edited-checker inner loop.
It forces TLC-required mode through conflicting environment and Make assignments; add
`UMPIRE_APALACHE=require` to require Apalache independently in the same run. Required timings
are separate from historical default-mode receipts below. Legacy `model-check` (gate only) and
`model-time` (gate plus benchmark) retain default/skip and explicit required-mode selection.
The 60 s full-gate (raised from 30 s on 2026-10-21) and 10 s inner-loop budgets are soft targets; observed overruns
must be reported. Temporal embeds the committed `temporal/hosts/go/umpire_temporal_guest.wasm`
through `temporal.Guest()`. Its Go consumers use no Rust or guest-path environment override.

`make umpire-guest` refreshes only that asset through `guest.sh`. `make model-check`
runs the same builder with `--check` before the other gate steps. Its internal
`guest-freshness.sh` bootstraps the native tool with `MISE_AUTO_INSTALL=false` and
`MISE_EXEC_AUTO_INSTALL=false` before the first tool probe. Rust then validates
TOML Rust-pin consistency, typed installed-tool JSON, installed mise-pinned
Rust/Cargo, Go and protoc executables, rustup toolchain readiness when applicable,
Rust sources and the wasm target. Missing bootstrap tools/compiler failures and
runtime prerequisites fail for every `CI` value; there is no local skip policy.
The cached tool follows the same validation path. Bash only orchestrates bootstrap;
metadata and pin semantics live in Rust.
The full Rust authoring gate still requires its toolchain. Slice and gaps use the
embedded asset directly; the umbrella's separate legacy IR bridge remains source-built.
Only the Temporal guest is committed, as a binary Git file without LFS; generic
stock/control guests are built for the gate and remain uncommitted.

## Native tool migration and current artifact (fn-247)

Native Rust commands own the former Python lock/compiler, guest/freshness, installer
and retired-name responsibilities. The workspace `cargo test --workspace --locked`
stage owns their unit/integration controls and the process controls once; no Python
unittest or extra package-specific Cargo stage is required. Bash timing, capabilities,
protobuf and guest-entry scripts remain orchestration. Native TOML/JSON schemas,
arguments and errors intentionally replace Python quirks; Model meaning, typed
refusals, lifecycle/mutation controls, raw compiler argv, jobserver/loader inheritance
and current artifact integrity remain required.

The [joined proof](../.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task5/joined/proof-report.md)
built stock and Temporal guests at source `523d131d2cf992b04a43b4d595ea0f1a434f44b6`
in two clean Linux/aarch64 clones with different paths and independent empty targets.
Both pairs agreed. Native identity schema 2 caused the necessary Temporal regeneration;
the [publication receipt](../.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task5/joined/canonical-publication.json)
records the old hash, publication and two successful checks. That publication was
3507265 bytes, SHA-256 `bc27af7d5b6b1ed8fa5c524015046f3a8b7c12d2d488e9ce7cb3df3ea9217872`;
fn-248.7 superseded it (below).
The component proof includes actual cold/cached freshness and stale status 2, not
the semantic Go suites or a complete Model gate. Final integrated validation uses
one `UMPIRE_APALACHE=require make model-validate`, which covers both required backends
and existing host suites, then benchmarks once. Native macOS/cross-OS certification
is not claimed for this migration. The earlier proofs below retain their own pins.

## Joined consolidation regeneration (fn-248.7)

fn-248's six lanes changed sources and the lockfile inside the Temporal guest closure
(`thiserror`, `prost-types` WKTs, `quote`/`darling` catalog generation, IR traversal;
enum-iterator removed), so the committed asset went stale. `make umpire-guest` on the
joined inputs, including the completion-review fixes that fully qualify the generated
`AnyMessage` calls, published 3516063 bytes, SHA-256
`11c953c35455d99d003aa774c75b11ffe90239e52acaaad3ba04a42afad8f68a`. A clean clone at a
different path with an independent empty target rebuilt identical bytes and passed the
freshness check. The fn-250.4/.5 back-end changes are outside the guest closure.
Linux/aarch64 only; no macOS certification is claimed.

## Consolidation lanes regeneration (fn-250.1/.2, fn-251.1/.2, fn-252.1/.2)

`Path` holding only its `Pattern` (umpire-dsl) and the removed `umpire-temporal-api`
build-dependency (it changes the generated-file path hash embedded in panic locations)
changed the Temporal guest: 3516063 B `11c953c3…` → 3516114 B
`b10b813547924162c9c195abc3a85d303d838b8194303a97639135580319542a`. A clean clone at a
different path with an empty target rebuilt identical bytes. Linux/aarch64 only.

## Linux source-path independence

The fn-230 source pin, matching hashes/sizes, platform and toolchain are recorded
in [DESIGN.md](DESIGN.md). Each proof uses two clean source clones at different
absolute paths with initially absent independent build targets. Downloaded
dependencies may be shared; compiled artifacts are not. From the repository root,
substitute the recorded source commit for `PIN` and fresh absolute paths:

```sh
git clone --shared --no-checkout . /absolute/proof/source-a
git clone --shared --no-checkout . /absolute/proof/different-source-b
git -C /absolute/proof/source-a checkout --detach PIN
git -C /absolute/proof/different-source-b checkout --detach PIN
test -z "$(git -C /absolute/proof/source-a status --porcelain)"
test -z "$(git -C /absolute/proof/different-source-b status --porcelain)"
CARGO_TARGET_DIR=/absolute/independent-target-a bash /absolute/proof/source-a/model/scripts/guest.sh umpire-stock-guest /absolute/proof/stock-a.wasm
CARGO_TARGET_DIR=/absolute/independent-target-a bash /absolute/proof/source-a/model/scripts/guest.sh umpire-temporal-guest /absolute/proof/temporal-a.wasm
CARGO_TARGET_DIR=/absolute/independent-target-b bash /absolute/proof/different-source-b/model/scripts/guest.sh umpire-stock-guest /absolute/proof/stock-b.wasm
CARGO_TARGET_DIR=/absolute/independent-target-b bash /absolute/proof/different-source-b/model/scripts/guest.sh umpire-temporal-guest /absolute/proof/temporal-b.wasm
cmp /absolute/proof/stock-a.wasm /absolute/proof/stock-b.wasm
cmp /absolute/proof/temporal-a.wasm /absolute/proof/temporal-b.wasm
sha256sum /absolute/proof/stock-a.wasm /absolute/proof/stock-b.wasm /absolute/proof/temporal-a.wasm /absolute/proof/temporal-b.wasm
wc -c /absolute/proof/stock-a.wasm /absolute/proof/temporal-a.wasm
test -z "$(git -C /absolute/proof/source-a status --porcelain)"
test -z "$(git -C /absolute/proof/different-source-b status --porcelain)"
```

Linux directory independence does not establish macOS or cross-OS byte identity.
The fn-232 attempt below failed cross-OS identity. Historical aborting
artifacts and earlier smaller Temporal closures do not certify the current unwind
build. The original engine feasibility proof is retained under
`.flow/tmp/fn230-eh-proof-20261021T032505Z/` with its sources and negative controls.

The earlier task-1 integrated unwind proof used source
`ab9c8edb38318d497e4ce06cd6548fd96b8772c5` from
`/Users/stephan/Workspace/skunkworks/umpire/temporal`.
Its exact runner is `.flow/tmp/fn2301-integrated-artifacts/run-proof.sh`, with output in
`.flow/tmp/fn2301-integrated-artifacts/proof.log`. It ran through
`timeout 600s env -u RUSTFLAGS -u CARGO_ENCODED_RUSTFLAGS bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2301-integrated-artifacts/run-proof.sh`
and exited 0. A task-specific `mktemp -d` created
`.flow/tmp/fn2301-integrated-proof.IJl2tf/`, containing source clones `source-a` and
`different-source-b` and initially absent targets `independent-target-a` and
`independent-target-b`. Target identities were `58:453312922` and `58:453329518`.
The proof ran from 06:45:02 to 06:47:30 UTC on 2026-10-21 (148 s).
The runner checked both source statuses before and after building, compared both pairs with `cmp`,
and recorded SHA-256 and byte counts. Each build used `CARGO_BUILD_JOBS=4`.
The observed platform was Linux 7.0.14/aarch64 with Rust 1.99.1
(`58cf8f9c35b355199727627d07ae34a4e2846d67`, LLVM 23.1.1), Cargo 1.99.1,
Go 1.27.0 and protoc 29.5. Its Temporal output was 3330375 bytes, SHA-256
`0d4e1b112097ba62f62f53c6a95ecc7776bfbf907df8594592586d3fe767ece0`.

The current proof follows the request-independent probe metadata reuse change at
source `594c13b237b3e3e5defcd67e7457257fa1db21ae`. Its runner is
`.flow/tmp/fn2302/run-reproduction-proof.sh`, executed as
`timeout 600s env -u RUSTFLAGS -u CARGO_ENCODED_RUSTFLAGS bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2302/run-reproduction-proof.sh 594c13b237b3e3e5defcd67e7457257fa1db21ae`.
It exited 0, from 08:36:24 to 08:38:13 UTC on 2026-10-21 (109 s), on the same
Linux platform and pinned toolchain above. Unique directory
`.flow/tmp/fn2302-reproduction-proof.JrgaIS/` contains clean clones `source-a` and
`different-source-b` and initially absent targets `independent-target-a`
(`58:453392182`) and `independent-target-b` (`58:453408168`). Each Temporal output
matches its own clone's committed embedded asset and the working checkout's asset.
Both stock and Temporal output pairs compare byte-identically; DESIGN records the
current sizes and hashes. The full transcript and exit status are
`.flow/tmp/fn2302/reproduction-proof.log` and `reproduction-proof.rc`.

The metadata cells publish only successful immutable inventories and lazy canonical
expectations. Every PREPARE still materializes and validates the current request;
request bytes, recipes, live slots and observations are never cached. The matched
no-server control retained all twelve response hashes and sizes across the change.
Warm valid-start preparation changed from roughly 1.05 s to 50 ms, and selected
generic preparation to 1.28–5.14 ms, without a timing assertion. Both independent
hosts retained 52 memory pages across repeated contexts and Reset, below the
unchanged 64 MiB limit. The full Temporal Go module passed with Rust unavailable
on PATH. Native controls cover failed/partial/panicked initialization retry,
concurrent publication, lazy canonical reuse and all 2238 generic cases across
two contexts. Evidence is in `.flow/tmp/fn2302/` and its `metadata-lane-evidence/`.

## Linux/macOS certification attempt (fn-232)

Certification failed on 2026-10-21 at source
`8d429e97409b1537ecb347c359129bffafa63f78`. Linux 7.0.14/aarch64 and macOS
25.6.0/arm64 each built both guests from two clean clones with initially absent,
independent target directories. Both path comparisons passed on each OS. Every
cross-OS comparison failed. The first differing offsets are 443 for stock and
635 for Temporal, zero-based. These results remain separate from the historical
fn-230 Linux receipts above.

Both executors used Rust 1.99.1, commit
`58cf8f9c35b355199727627d07ae34a4e2846d67`, LLVM 23.1.1, Cargo 1.99.1
(`5f94df478`), Go 1.27.0 and protoc 29.5. The Rust hosts were
`aarch64-unknown-linux-gnu` and `aarch64-apple-darwin`; both outputs targeted
`wasm32-wasip1` using the unchanged unwind builder and two Cargo jobs.

| OS | Guest | Bytes in each path | SHA-256 in each path |
| --- | --- | ---: | --- |
| Linux | Stock | 192255 | `5ec57e2e5371723adf318665c1048b5a4021e9f4203ed56884c5f849e7a2dfc1` |
| macOS | Stock | 192263 | `ea826b9088f9fae68943b149289d5646e349fc8f2751f0bab422312fe74e2e54` |
| Linux | Temporal | 3329990 | `10724102d03e1659f28b07c3a0c848d2cac5876668f3be9261d0a3b65023cf06` |
| macOS | Temporal | 3329996 | `791b33b3db2e8d57464d94fcdfb8b95594d15b434a6fb7e1c24905a5e4069547` |

The retained runner, platform transcripts, target identities, source-status checks
and all twelve actual `cmp` results are under
[fn-232 evidence](../.flow/artifacts/fn-232-linux-macos-guest-byte-identity/proof.md).
That report records exact invocations and the separate host-parity results.
The stock WAT diff changes the element table, function bodies and data layout.
The subsequent [diagnosis](../.flow/artifacts/fn-232-linux-macos-guest-byte-identity/diagnosis.md)
isolated Cargo metadata and generated-output path hashes; a controlled macOS
replay matched both full Linux guests. The separately tracked fn-236 implementation now
normalizes those compiler inputs without a recorded Linux hash map.

## Host-independent compiler inputs (fn-236)

Four frozen source snapshots at base
`77d2e4e90eeea14711ebfd050f41f0ef3b1d0b1e` plus the scoped build fix produced
identical guests from two independent paths on each native OS. Every target was
initially absent. All twelve actual pairwise comparisons passed, as did both
stock/control and Temporal host suites on Linux and macOS. The refreshed embedded
Temporal asset matches all four builds.

| Guest | Bytes in each build | SHA-256 in each build |
| --- | ---: | --- |
| Stock | 192271 | `f3da11cbe34e75e24ec698979058a6f300aad016e5215a63fe2d392951341219` |
| Temporal | 3330006 | `5bd641822d7a124f10b247d022d8f6d296fe13632f418acfa23bd85152a6285e` |

The [fn-236 proof](../.flow/artifacts/fn-236-host-independent-guest-compiler-inputs/proof.md)
retains source/overlay manifests, toolchains, exact commands, comparisons, host
logs and independent review. Sixteen focused script tests and the full
`UMPIRE_BACKENDS=require make model-check` gate passed on macOS. The gate's final
stock and Temporal outputs match the frozen proof and refreshed asset. These frozen
snapshots preceded the committed clean-checkout certification below. The earlier
fn-232 failure remains above.

## Committed Linux/macOS certification (fn-232)

Certification passed at `3c314fb5b85427471a1525d6f190fe5d1fb9f3b2` on 2026-10-21.
Linux 7.0.14/aarch64 and macOS 25.6.0/arm64 each used two clean detached clones,
initially absent independent targets and the same pinned toolchain. All twelve
actual byte comparisons passed. Stock and Temporal have the sizes and hashes in
the fn-236 table above; each Temporal output also matches the committed asset.
Both native stock/control and embedded Temporal host suites passed, with clean
source checks repeated after the suites.

Every clone used Rust 1.99.1 (`58cf8f9c35b355199727627d07ae34a4e2846d67`), LLVM
23.1.1, Cargo 1.99.1 (`5f94df478`), Go 1.27.0 and protoc 29.5. From the repository
root, the exact retained invocations were:

```sh
sbx exec -w "$PWD" dev-build bash -c 'bash .flow/artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/run-proof.sh "$PWD" "$PWD/.flow/tmp/fn232-certified-linux-3c314fb5b8"'
bash .flow/artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/run-proof.sh "$PWD" "$PWD/.flow/tmp/fn232-certified-macos-3c314fb5b8"
python3 .flow/artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/compare.py "$PWD"
```

The [committed certification](../.flow/artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/proof.md)
records the full source pin, exact invocations, platform/toolchain transcripts,
source and target guards, artifact manifests, actual comparisons and host logs.
This completes fn-232's historical committed-source requirement. Its retained `compare.py`
command is reproduction evidence outside the active Model toolchain, not a current Python
prerequisite. The frozen fn-236 validation
and historical failed fn-232/Linux-only proofs remain separate records.

## Lockfile closure identities (fn-238.8)

At source `107285ac64`, and again after the review follow-ups at `9df904d985`, two clean clones at different absolute paths with initially
absent independent targets built identical guests on Linux 7.0.14/aarch64 with the
pinned toolchain, four Cargo jobs and clean source checks before and after. A third
clone added an unrelated workspace crate and a dev-dependency on it from
`umpire-dsl`, which both guests depend on; its `Cargo.lock` changed, and both guests
still matched byte for byte. The same edit at the pre-change base `ccacbf4b37`
changed the Temporal guest. The refreshed embedded asset matches all three builds.

| Guest | Bytes | SHA-256 |
| --- | ---: | --- |
| Stock | 192489 | `4fc5c19ff069c954c4aaef3d97d014f8dc029951d8401b50d501987fa2a41b61` |
| Temporal | 3332314 | `0248382ec34f616078a97190dd0ae49544851800e54b5ecb47aaa772d056cae0` |

## Renewed committed Linux/macOS certification (fn-239)

Certification passed at `a96a4c6550b468c5d7c3011a7961d4e118dacd93` on 2026-10-22,
after fn-238.8 and its front-end integration. Linux 7.0.14/aarch64 and macOS
25.6.0/arm64 each used two clean detached clones with initially absent independent
targets. All twelve actual byte comparisons passed, both stock/control and Temporal
host suites passed on each OS, and the Temporal outputs match the committed asset.
The precheck and all four builds produced identical lockfile closure maps.

| Guest | Bytes in every clone | SHA-256 in every clone |
| --- | ---: | --- |
| Stock | 192489 | `bae203f82ecea5183d592232c62b746f6ccabda9944df53e46dd442482d1bc63` |
| Temporal | 3332314 | `d8127a2bf8c72de75cfafde655e35b5b1aa6355a023628708dd8ec6d22f83370` |

Every clone used Rust 1.99.1 (`58cf8f9c35b355199727627d07ae34a4e2846d67`), LLVM
23.1.1, Cargo 1.99.1 (`5f94df478`), Go 1.27.0 and protoc 29.5. The retained
fn-232 runner and comparator changed only their source pin and task output paths.
The [fn-239 proof](../.flow/artifacts/fn-239-renew-the-linuxmacos-guest-byte/certification-a96a4c6550/proof.md)
retains exact commands, toolchain and host transcripts, source/target guards,
manifests, closure-map checks and all twelve comparison results. This renews the
certification for the per-crate closure identity input. Earlier proofs retain
their historical source pins and hashes.

## Historical fn-230 task-2 verification and measured gate cost

Task 2 passed both complete embedded live targets, including the exact existing
gap report golden, all 2238 generic cases and the strict/scope controls. The slice
package took 1.111 s and gaps 355.473 s. The pre-edit full gaps baseline had timed
out at the unchanged 20-minute deadline; no timeout, corpus or golden was changed.
Default and required-backend `model-check` passed again after the final fixture
correction. Current stale-asset control failed at the real freshness gate with
3326027-byte sizes and offset 8; `umpire-guest` restored the exact committed asset.

With the same caches and four Cargo jobs documented below, `make model-time`
passed: full gate 51.936 s, above its unchanged 30 s soft target; edited-checker
inner loop 1.607 s, below 10 s. Native tests took 24.041 s, guest builds 14.353 s
and Temporal Go tests 9.862 s. The first timing run failed because its timing flag
hid the nested fixture's stale diagnostic. The fixture now removes only that flag
from its own Make subprocess, retaining the assertions and production timing path.
The failed run, focused red/green and successful retry remain in
`.flow/tmp/fn2302/`; the owned `task-2-verification.md` artifact records exact logs.
Gate-built stock and Temporal bytes still match the renewed clean-clone proof.

## Historical fn-230 task-1 verification and measured gate cost

The default and `UMPIRE_BACKENDS=require make model-check` gates passed on the
integrated source. Required mode exercised the existing TLC agreement and negative
controls with the existing checksum-pinned 1.7.4 jar in the checkout's ignored cache.
The Temporal source shape retains its existing TLC refusals. Both Go host modules
also passed `go vet -tags test_dep ./...`. A verbose guest-profile parity run passed
the stock goldens, caught panic/overflow, repeated calls/Reset, uncaught traps and
exact descriptor-denial/signature controls. Gate-built stock and Temporal artifacts
compared byte-identically with the clean-clone proof after both gates and timing.

With `CARGO_BUILD_JOBS=4`,
`CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2301-unwind/cargo` and
`GOCACHE=/home/agent/.cache/codex-build/fn2301-unwind/go`, `make model-time`
exited 0. Its full gate took 163.764 s, exceeding the 30 s soft target. Native
tests took 98.880 s and guest builds 55.257 s. The edited-checker inner loop took
3.295 s, within its 10 s soft target. These are individual observations on the shared
executor. The larger unwind artifacts and this gate overrun remain visible costs
of preserving caught-panic judgments.

Integrated logs are `.flow/tmp/fn2301-integrated-artifacts/model-check.log`,
`model-check-required.log`, `profile-parity.log` and `model-time.log` in that directory.
Earlier failures remain in `.flow/tmp/fn2301-unwind-artifacts/`. The first gate exposed a relative child
script invocation under this host's directory-resetting shell startup; the builder
now invokes an absolute script path. The first timing run exposed its wrapper's
different working directory; Python test discovery now also uses an absolute path.
Neither correction changes test assertions or guest code generation.

The independent framework workspace at `model/umpire` owns the native tools and
pins. `xtask` executes the dedicated `umpire-guest-driver`; guest caches hash that
driver's complete executable, the caller workspace's complete lockfile and compiler
input. Framework and caller workspace roots are admitted separately. Their canonical
longest-root remaps distinguish framework sources from caller sources, including when
the framework is nested inside the caller. Root shell entry points preserve the existing
Temporal guest publication paths while framework-local scripts work without the parent.
