# Native cross-OS guest diagnosis

Both full guests can match across native Linux and macOS. A controlled macOS
compiler-input replay reproduced the exact original Linux stock and Temporal
artifacts at source `8d429e97409b1537ecb347c359129bffafa63f78`. The production
builder remains unchanged, so the original certification failure remains open.

## The two changes

1. For each target compilation unit, replace Cargo's `-Cmetadata` argument with
   the value recorded for the corresponding Linux unit. The experiment matches
   units by crate, normalized source location, target, cfgs and profile flags.
   Host programs still compile and execute natively on macOS.
2. Add a specific rustc `--remap-path-prefix` for each generated `OUT_DIR`, using
   the corresponding Linux path after its target root was remapped to `/target`.
   The existing broad target-root remap leaves Cargo's hashed build-directory
   component visible; this extra mapping controls that component too.

These wrappers change compiler inputs. They perform no post-link Wasm rewriting,
copy no compiled Linux dependency into the macOS build, and change no source,
dependency version, panic behavior, target, ABI or protocol.

## Measured stages

| Artifact | Bytes | SHA-256 |
| --- | ---: | --- |
| Original Linux stock | 192255 | `5ec57e2e5371723adf318665c1048b5a4021e9f4203ed56884c5f849e7a2dfc1` |
| Original macOS stock | 192263 | `ea826b9088f9fae68943b149289d5646e349fc8f2751f0bab422312fe74e2e54` |
| macOS stock, metadata replay only | 192255 | `e07278b323286617d3ea0a03be79b2b6d91a8f374e82984d38a8e901895b6049` |
| macOS stock, metadata and path replay | 192255 | `5ec57e2e5371723adf318665c1048b5a4021e9f4203ed56884c5f849e7a2dfc1` |
| Original Linux Temporal | 3329990 | `10724102d03e1659f28b07c3a0c848d2cac5876668f3be9261d0a3b65023cf06` |
| Original macOS Temporal | 3329996 | `791b33b3db2e8d57464d94fcdfb8b95594d15b434a6fb7e1c24905a5e4069547` |
| macOS Temporal, metadata and path replay | 3329990 | `10724102d03e1659f28b07c3a0c848d2cac5876668f3be9261d0a3b65023cf06` |

The diagnostic baseline uses a logging-only compiler wrapper. Both baseline
guests on each OS compare identically with their original certification outputs,
so the logging wrapper preserves the failure. Both builds use the same pinned
Rust/Cargo 1.99.1, Go 1.27.0 and protoc 29.5 toolchains as the original proof.

Metadata replay alone made every stock section except the data section identical,
including the complete code section. Exactly 32 bytes still differed. They were
the two 16-character Cargo build-directory hash components in generated-source
paths for `google.protobuf.rs` and `umpire.model.v1.rs`.

For example, the remaining stock data differed between these path components:

```text
/target/wasm32-wasip1/guest/build/umpire-wasm-fca8c8d50ef33d7f/out/
/target/wasm32-wasip1/guest/build/umpire-wasm-884de41c772b4995/out/
```

The additional `OUT_DIR` remap removed those differences. Actual `cmp` calls then
returned 0 for both guests. [results.json](diagnostics/results.json) retains the
artifact hashes, comparison commands, exit codes and section comparisons.

The macOS stock/control host suite and Temporal host suite also passed with the
matching artifacts. The generic suite loaded the experimental stock output; the
Temporal suite loaded its committed embedded asset after an explicit byte
comparison with the experimental Temporal output. The Temporal suite took
12.945 s. [Host verification transcript](diagnostics/macos-host-parity.log)
records the actual commands and final clean-source check.

## Why Cargo's inputs differed

The captured arguments show the divergence before LLVM or the Wasm linker runs:

| Target crate | Linux `-Cmetadata` | macOS `-Cmetadata` |
| --- | --- | --- |
| core | `f3f7e499bcdb2f82` | `f3f7e499bcdb2f82` |
| compiler_builtins | `df23f9fde27e28c5` | `311adcc37356df73` |
| std | `b78b6a30636ce6d6` | `39322beb15281227` |
| umpire_stock_guest | `912655849e7cae9c` | `72b4c8eb515d1463` |

The pinned Cargo implementation omits the compiler host from direct cross-target
version hashing, but retains it for host units and incorporates dependency
metadata recursively. This provides a source-level explanation for host build
dependencies propagating different hashes into target crates.
[Pinned Cargo implementation](https://github.com/rust-lang/cargo/blob/5f94df4789f005f9a352888e8355ffc645b7ed0e/src/compiler/build_runner/compilation_files.rs#L668-L865)

`core` has matching metadata; `compiler_builtins`, which has a native build script,
already differs. Rebuilt std and downstream target crates inherit differences.
The staged replay demonstrates that these target metadata values account for the
stock code-layout differences. No compiler/backend defect is needed to explain
the observed artifacts.

Additional checks found identical std and libc build-script stdout. The
compiler_builtins stdout differed only in its installed compiler-rt source path.
Four generated Rust files matched directly; the two generated fixture registries
matched after normalizing their source-root paths. Their raw path differences
did not prevent the complete controlled output comparison from passing.

## Evidence and replay constraints

The [diagnostics directory](diagnostics/) retains the compiler argument traces,
selected environment values, shell transcripts, unit-matching helper, replay
wrappers and Linux reference maps. Complete Wasm outputs and compiled targets
remain under `.flow/tmp/fn232-diagnosis/`. Original certification artifacts remain
under `.flow/tmp/fn232-certification-{linux,macos}/`.

The baseline command is:

```sh
bash .flow/tmp/fn232-diagnosis/run.sh "$PWD" macos
```

Linux executes the same script through the existing `sbx exec dev-build` executor.
The metadata-only replay is in `macos-replay.log`; the fresh metadata/path replay
is in `macos-replay-paths-fresh.log`. Temporal baseline logs are
`linux-temporal.log` and `macos-temporal.log`; its successful replay is in
`macos-temporal-replay.log`. Each transcript includes the actual environment and
builder invocation. The quick result check is:

```sh
python3 .flow/tmp/fn232-diagnosis/summarize.py
```

That check expects the original comparisons and metadata-only stock comparison
to fail, and both complete replays to pass. It also checks the Linux reference
hashes and the metadata-only stock code-section equality.

An attempted clean of the owned metadata-replay cache was rejected by Cargo
because it lacked `CACHEDIR.TAG`. No files were removed. The path-remap experiment
instead used a fresh target; the failed cleanup transcript is retained.

## Permanent fix still required

The experiment reads metadata values and output paths recorded from Linux. This
is a diagnostic replay, not a self-contained reproducible build recipe. A durable
fix must derive matching, collision-safe crate identities independently on both
OSes and normalize generated-source paths without a reference Linux build.
Hardcoding the recorded hashes would conceal the general problem.

That fix belongs in a separately scoped task under fn-232's existing boundary.
After it, certification must repeat the unchanged four-clean-root comparisons
and host suites using the new source pin. Matching bytes in this experiment do
not close the original recipe's certification. No commits were made by this
investigation, and other specs were untouched.
