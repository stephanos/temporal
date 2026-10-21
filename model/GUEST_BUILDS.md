# Guest builds: reproduction and diagnostics

`bash model/scripts/guest.sh <crate> <output> [--check]` builds any cdylib with
the mise-pinned Rust 1.99.1 compiler, `--locked`, `wasm32-wasip1` and `profile.guest`.
It discovers the output through Cargo's artifact messages, including custom library
names. It requires Bash, Python 3, the pinned toolchain and its `rust-src` component.
Exit 0 means built or fresh, 1 means build/tool/output failure, and 2 means missing
or stale output. Staleness reports existing/built sizes and the first differing
zero-based byte offset, including EOF; check mode never updates the output.
Relative output paths are relative to the caller. `CARGO_TARGET_DIR` selects the
build cache, defaulting to `model/target`.

The profile inherits release, strips symbols, disables debug information and uses
fat LTO with one codegen unit and checked overflow. The checker's release profile
keeps its profiling symbols. Workspace, Cargo registry, toolchain sysroot and target
paths are remapped by rustc, including generated protobuf locations under `OUT_DIR`.
No wasm bytes are rewritten after linking. The Temporal API generator sorts merged
descriptor files before prost generation; otherwise Go map iteration could change
merged Rust declaration order and the resulting guest bytes.

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
python3 -m unittest discover -s model/scripts/tests -p 'test_*.py'
bash -n model/scripts/guest.sh model/scripts/tests/build-guests.sh
CARGO_TARGET_DIR=/absolute/task-owned-target bash model/scripts/tests/build-guests.sh
UMPIRE_BACKENDS=require CARGO_TARGET_DIR=/absolute/task-owned-target make model-check
CARGO_TARGET_DIR=/absolute/task-owned-target make model-time
```

`model-time` executes the actual default gate and an edited-checker inner loop.
The 30 s full-gate and 10 s inner-loop budgets are soft targets; observed overruns
must be reported. Embedding and committed-artifact freshness remain fn-230.2's work.

## Linux source-path independence

The current source pin, matching hashes/sizes, platform and toolchain are recorded
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
That separate certification remains unverified under fn-232. Historical aborting
artifacts and earlier smaller Temporal closures do not certify the current unwind
build. The original engine feasibility proof is retained under
`.flow/tmp/fn230-eh-proof-20261021T032505Z/` with its sources and negative controls.

The executed integrated unwind proof used source
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
Go 1.27.0 and protoc 29.5. DESIGN records the resulting bytes and hashes.

## Verification and measured gate cost

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
