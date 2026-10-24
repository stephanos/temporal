# fn247.2 native guest identity research

The existing lock and compiler helpers expose a small port surface. Nine lock tests and twelve wrapper tests already specify most identity behavior. Their assertions can become native Rust controls without retaining Python serialization or parsing. The remaining proof work concentrates on Unix argument bytes, process inheritance, failure cleanup, atomic publication and the actual compiler build.

This is read-only research, not an implementation decision or admission receipt. Source was inspected on 2026-10-24 while the root worker owned the checkout. The authorized fn247 and .2 briefs in `.flow/tmp/drop-byte-compat-final-apply/` supplied scope. No Cargo, Flow, build, test or installation command ran. Only this report was written. The conductor still needs fresh primary review/admission evidence before implementation. The supplied .4 closure and .9 status were not independently revalidated.

## Existing controls and reuse points

All paths below are relative to `temporal/`.

| Responsibility | Implementation | Existing controls and native meaning |
| --- | --- | --- |
| Portable lock coordinates | `model/scripts/guest-lock.py:16` | `test_guest_lock.py:57,74` exclude unrelated packages/dev-only edges and remove checkout paths. Coordinate identity uses name/version/source, not Cargo's path-bearing package ID. |
| Closure propagation | `guest-lock.py:34` | Tests at 66,78,86,94 cover workspace feature-unified optional edges, checksum propagation to every ancestor, all-platform build dependencies and edge topology. Keep graph edges as well as members. A shared crate gaining an optional edge intentionally affects dependents. |
| Ambiguous/missing closure | `guest-lock.py:49` | Tests at 100,115 mark duplicate name/version coordinates ambiguous and refuse a missing sourced lock entry. Git packages may legitimately lack a checksum. Native structured errors can replace exception strings. |
| Content-addressed publication | `guest-lock.py:56` | Test at 126 verifies the map content and content-derived filename, malformed JSON failure and no success stdout. Exact filename length/JSON bytes may change with the native schema. |
| Compiler option extraction | `model/scripts/guest-rustc.py:14` | Handles separate and joined values, repeated cfg/extern/codegen options. Preserve the compiler's untouched argv except intended target metadata replacement and OUT_DIR remapping. Add focused malformed-option and raw-byte controls. |
| Compiler identity | `guest-rustc.py:56` | `test_guest_rustc.py:55,67,72` cover checkout/host metadata independence, dependency propagation and package/version/features/profile/compiler distinctions. The identity excludes Cargo metadata, extra-filename and incremental flags; target metadata is replaced while Cargo artifact names remain intact. |
| Build-std exemption | `guest-rustc.py:42` | Test at 96 requires wasm target plus Cargo's unstable marker and either remapped sysroot sources or a matching package in rust-src's library lockfile. Normal workspace crates cannot claim the exemption. |
| Dependency sidecars | `guest-rustc.py:35,87` | Tests at 117 and 142 reject missing/corrupt identity, invalid hash/schema and conflicting transitive native macro variants. Preserve alias-to-digest dependency pairs. |
| Host proc macros | `guest-rustc.py:104` | Test at 129 preserves original host compiler argv, omits native cfg differences from portable identity, and records Cargo metadata variants for collision detection. This is safety behavior, not a Python quirk. |
| Probe passthrough | `guest-rustc.py:60` | Test at 153 runs compiler probes without requiring guest environment. Ordinary non-guest host compilations likewise pass through. |
| Pipelining and failure cleanup | `guest-rustc.py:147` | Tests at 157 and 168 require the sidecar before rustc starts and its removal after compiler failure. Launch failure and signal outcomes exist in implementation at 159-167 but lack explicit tests. |
| Jobserver and previous wrapper | `guest-rustc.py:150,159` | Test at 181 checks an inherited descriptor reaches the compiler. Previous-wrapper execution exists but has no direct control in this file. |
| Darwin loader inheritance | `model/scripts/guest.sh:44` | `test_guest.py:43` checks `DYLD_FALLBACK_LIBRARY_PATH` through the wrapper on Darwin only. The comment explains why `/usr/bin/env` launchers are unsuitable. Native direct execution removes the interpreter launcher requirement but must retain this process behavior. |

The lock corpus has no parser-quirk suite to recreate. Retire Python `json.dumps` spacing/escaping, Python exception text/types, Python import fixtures, interpreter shebang copying and the Python executable whitespace restriction. Retain paths containing spaces as an ordinary valid-input control. `test_guest.py`'s exact first-difference offset strings and cache-key formula need not survive; fresh/stale/failure distinctions, cache invalidation and never-publish checks remain useful. No Python TOML/BigInt/date/NaN/recursion/digit emulation is needed.

## Small native interface candidates

These are bounded surfaces inferred from current callers, not a settled module design.

- A lock operation consumes Cargo metadata JSON plus Cargo.lock and produces a structured closure map. Its CLI reads stdin and publishes a content-addressed map in a supplied directory. Keep graph computation independent from publication so most controls need no child process.
- A compiler preparation operation consumes `Vec<OsString>` and the needed environment values and returns the prepared command plus an optional sidecar path/record. A thin runner publishes the record, invokes the prior wrapper if present, waits, and removes the record on launch/nonzero failure. Waiting rather than replacing the process is necessary for cleanup.
- Shared support only needs deterministic JSON bytes, SHA-256 formatting, schema/version constants and atomic file publication. A sibling temporary file opened with `create_new`, complete write, close and rename avoids partial-reader visibility; error paths must remove owned temporaries. No installer archive parser, generic process supervision layer or Python-compatible serializer belongs here.

Decode command names and known textual fields explicitly. Keep paths, untouched rustc options and environment as OS strings. Unix `OsStrExt`/`OsStringExt` allow byte inspection/reassembly without unsafe code. Never derive identity inputs through `to_string_lossy()`. A native schema can encode non-UTF-8 identity-bearing path/option bytes explicitly, for example byte arrays or hex, while keeping remapped logical roots stable. Use lexical path normalization with root-boundary checks. Filesystem canonicalization would require existing source paths and could introduce symlink/host differences that the current lexical normalization does not have.

Raw bytes matter inside joined arguments too, including `--extern=alias=<path>` and `--remap-path-prefix=<path>=...`. ASCII option-prefix parsing must preserve the remainder. Inherit unmodified environment, including non-UTF-8 values. `std::process::Command` and Unix exit-status inspection fit the existing foreground child contract; verify inherited jobserver descriptors with a native helper process on the available Linux host. Do not route compiler children through the checker supervisor, whose descriptor/session policy serves a different use case.

## Foundation and independent lanes

The central owner first needs to settle xtask package wiring, dispatcher, shared schema/hash/publication interfaces and test fixture access. Existing patterns are `model/Cargo.toml:42` (workspace metadata), `:74` (dependencies), `:102` (workspace lint policy), `model/.cargo/config.toml:1` (aliases), and `model/umpire/backends/apalache/Cargo.toml:1` (package inheritance and dependency opt-in).

After those interfaces exist, the lock implementation and its fixtures can occupy a separate file/test lane from compiler preparation and sidecar/process controls. The latter owns raw OS argument tests and a small native test-child fixture. Cargo.toml/Cargo.lock, alias, shared support and dispatcher remain centrally owned. Tests must have either a single owner per file or distinct integration files to avoid parallel edits. The authorized `guest_identity` target can aggregate separate test modules. Python source stays available to current shell callers until the successor switches orchestration, as the .2 brief explicitly requires.

Installed primary source confirms `serde_json` 1.0.154 uses BTreeMap by default (`src/map.rs:3,39`), with `from_slice` in `src/de.rs:2667` and `to_vec` in `src/ser.rs:2213`. Current model manifests have no `preserve_order` request. This is a source observation, not a resolved feature inventory. Serialization through `serde_json::Value` preserves sorted object keys; typed structs serialize declaration order, so select and document one native identity representation deliberately. Arrays representing sets still require explicit sorting.

The locally cached `toml` 1.1.8+spec-1.1.0 has `toml::from_str` at `src/de/mod.rs:72`; its approved parse/serde/std features are in Cargo.toml:109-122. The workspace lock already records the authorized checksum at `model/Cargo.lock:776`. Only sha2 0.10.9 was locally cached during inspection; sha2 0.11 was absent from both this lockfile and the cache. Its exact resolved API/transitives remain for implementation-time verification under the existing admission. This report does not substitute the old cached version. No external documentation was necessary.

## Caller and artifact update surface

`model/scripts/guest.sh:20-61` owns wrapper selection, cache derivation, launcher publication, metadata piping and previous-wrapper capture. Its successor must bootstrap xtask before installing guest flags/wrappers and directly invoke the native wrapper. Cache invalidation must cover the native implementation/schema; otherwise old `.umpire-id` records can outlive a format change. `guest.sh:80` owns artifact selection and build/check publication; its current ordinary copy is not an atomic-publication example to reuse.

`model/scripts/tests/test_guest.py:81` copies Python helpers into a fake workspace and mutates their contents to test cache invalidation. Those fixtures need coordinated replacement when orchestration moves. `guest-freshness.py:88`, Makefile:467/1278 and `scripts/tests/build-guests.sh:10-16` are live callers. The latter builds stock, Kettle, control, Temporal, caught-panic and overflow guests. `Makefile:1281` currently discovers Python controls, and later workspace test stages at 1288-1291 can naturally execute xtask tests once included.

The committed Temporal asset is `model/temporal/hosts/go/umpire_temporal_guest.wasm`, embedded by `guest.go:5`. Compiler metadata changes may change its bytes even with unchanged Model semantics. The Go runner hashes actual wasm at `model/umpire/hosts/go/runner/runner.go:133`, writes that identity into Runs at `session.go:38`, checks it at `replay.go:17`, and publishes under model-hash directories at `publish.go:71`. Preserve those integrity checks when regenerating any affected fixtures. Focused searches found no literal Temporal guest hash pin in its Go sources; this does not prove no pin exists elsewhere. Search the actual old hash repo-wide at regeneration time, and update only current artifacts with a demonstrated dependency. The compatibility manifest's hashes identify frozen originals and are separately slated for removal, not guest repinning.

## Focused verification surfaces

No command below was executed by this research lane.

The authorized .2 brief already names the proposed target `cd model && mise exec -- cargo test -p xtask --test guest_identity --locked` and `cd model && mise exec -- cargo clippy -p xtask --all-targets -- -D warnings`. The target and package do not exist yet. Proposed test modules should cover the table above, deterministic repeat/shuffled-order/relocated-input results, malformed native JSON/TOML, publication failure/concurrency, raw Unix bytes, previous-wrapper ordering and status/signal propagation. Small native fixture executables can check argv/environment/FD contents and sidecar availability without Python.

Existing post-switch integration surfaces are `bash model/scripts/guest.sh umpire-temporal-guest <candidate>`, its `--check` mode, `model/scripts/tests/build-guests.sh` and `model/scripts/tests/guest-parity.sh`. They require actual tools and artifacts; they are not unit-test shortcuts. The existing Makefile Go stages provide the exact environment for runner, Kettle and Temporal controls. Keep final integrated TLC-plus-Apalache validation in the already established fn247.3 single pipeline; this report proposes no additional full gate.

## Bounded risks requiring actual proof

- New direct dependencies can alter shared feature-unified metadata edges despite being host-only. Compare actual guest package/feature closures before and after; a workspace-level lock diff alone cannot prove isolation.
- Stable new identities require a consistent OS-byte representation and complete option normalization. Existing tests do not cover arbitrary byte paths or every rustc joined-option spelling.
- Atomic visibility does not prove concurrent cleanup ownership. Exercise overlapping same-sidecar writes/failures and decide the needed behavior from actual Cargo/cache concurrency.
- Linux FD success cannot establish macOS loader inheritance. Source inspection and environment fixture controls can retain the intended behavior; actual Darwin execution remains unproven here. The latest owner instruction explicitly drops macOS receipts, so this is no hardware or admission gate.
- Passing synthetic identities does not establish reproducible wasm. The eventual integrated regeneration must record the old/new guest hashes, preserve semantic controls and reproduce the newly committed asset. None of these risks requires restoring old Python bytes or diagnostics.
