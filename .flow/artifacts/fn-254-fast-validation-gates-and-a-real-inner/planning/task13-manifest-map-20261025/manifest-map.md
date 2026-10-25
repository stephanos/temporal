# fn-254.13 manifest inheritance map

Source checkpoint `9eefdd2f94cf890cfc7ad37b677bd5abcbb94e1f`. Investigation only. No Cargo metadata, compilation, tests, benchmarks, source changes or Flow lifecycle changes were performed. The pending `be919e0999` TLC manifest was inspected separately; its `strategy-equivalence` feature must survive integration.

The split requires two root manifests and lockfiles, plus relocation of the xtask package. All 33 existing member manifests can preserve their text for inheritance and dependency paths alone. Crate source/test path changes are separately owned by the existing design inventory.

## Membership and inheritance

Current member declarations are `model/Cargo.toml:5`. Framework receives 27 members, including relocated xtask; Temporal keeps six. The following paths are current package roots relative to `model/`; the reference is the exact current membership line.

| Root | Package | Owner | Reference |
| --- | --- | --- | --- |
| `dev/xtask` | `xtask` | Framework | `model/Cargo.toml:6` |
| `umpire/author/umpire` | `umpire` | Framework | `model/Cargo.toml:7` |
| `umpire/author/umpire/tests/consumer` | `umpire-author-consumer` | Framework | `model/Cargo.toml:8` |
| `umpire/author/dsl` | `umpire-dsl` | Framework | `model/Cargo.toml:9` |
| `umpire/author/dsl/macros` | `umpire-dsl-macros` | Framework | `model/Cargo.toml:10` |
| `umpire/author/suite` | `umpire-suite` | Framework | `model/Cargo.toml:11` |
| `umpire/front` | `umpire-front` | Framework | `model/Cargo.toml:12` |
| `umpire/ir` | `umpire-ir` | Framework | `model/Cargo.toml:13` |
| `umpire/ir/derive` | `umpire-ir-derive` | Framework | `model/Cargo.toml:14` |
| `umpire/backends/native` | `umpire-checker` | Framework | `model/Cargo.toml:15` |
| `umpire/backends/process` | `umpire-backend-process` | Framework | `model/Cargo.toml:16` |
| `umpire/backends/tla` | `umpire-tla` | Framework | `model/Cargo.toml:17` |
| `umpire/backends/tlc` | `umpire-tlc` | Framework | `model/Cargo.toml:18` |
| `umpire/backends/apalache` | `umpire-apalache` | Framework | `model/Cargo.toml:19` |
| `umpire/protocol/runtime` | `umpire-runtime` | Framework | `model/Cargo.toml:20` |
| `umpire/protocol/runtime/guests/stock` | `umpire-stock-guest` | Framework | `model/Cargo.toml:21` |
| `umpire/protocol/runtime/guests/control` | `umpire-control-guest` | Framework | `model/Cargo.toml:22` |
| `umpire/protocol/runtime/guests/caught-panic` | `umpire-caught-panic-guest` | Framework | `model/Cargo.toml:23` |
| `umpire/protocol/runtime/guests/overflow` | `umpire-overflow-guest` | Framework | `model/Cargo.toml:24` |
| `umpire/catalog` | `umpire-catalog` | Framework | `model/Cargo.toml:25` |
| `umpire/catalog/syntax` | `umpire-catalog-syntax` | Framework | `model/Cargo.toml:26` |
| `umpire/catalog/macros` | `umpire-catalog-macros` | Framework | `model/Cargo.toml:27` |
| `umpire/catalog/build` | `umpire-catalog-build` | Framework | `model/Cargo.toml:28` |
| `umpire/reference/kettle` | `umpire-reference-kettle` | Framework | `model/Cargo.toml:29` |
| `umpire/reference/kettle/guest` | `umpire-kettle-guest` | Framework | `model/Cargo.toml:30` |
| `umpire/dev/testing` | `umpire-testing` | Framework | `model/Cargo.toml:31` |
| `umpire/dev/fixtures` | `umpire-fixtures` | Framework | `model/Cargo.toml:32` |
| `temporal/api` | `umpire-temporal-api` | Temporal | `model/Cargo.toml:33` |
| `temporal/annotate/syntax` | `umpire-temporal-annotate-syntax` | Temporal | `model/Cargo.toml:34` |
| `temporal/realization` | `umpire-temporal` | Temporal | `model/Cargo.toml:35` |
| `temporal/realization/macros` | `umpire-temporal-realization-macros` | Temporal | `model/Cargo.toml:36` |
| `temporal/guest` | `umpire-temporal-guest` | Temporal | `model/Cargo.toml:37` |
| `temporal/testing` | `umpire-temporal-tests` | Temporal | `model/Cargo.toml:38` |

For the framework member paths, remove the initial `umpire/`; xtask becomes `dev/xtask` after its physical move to `model/umpire/dev/xtask`. The outer member paths remain unchanged. Explicitly retain the consumer and five framework guests even when no other package depends on them. Outer `workspace.exclude` names `umpire`, per `model/design/framework-gate.md:15`. Neither root currently has `default-members`, patches or replacements.

Member `package.version`, `edition`, `rust-version`, `license`, inherited `publish`, dependency `workspace = true`, and inherited lints find the enclosing workspace. There is no explicit `package.workspace` pointer in any member to rewrite. Framework dependencies used by Temporal still inherit from their own framework root, not from the outer caller. The moved xtask manifest has no relative dependency path; its explicit bin path remains relative to its moved package (`model/dev/xtask/Cargo.toml:24`).

The local lint exceptions remain literal. `model/umpire/backends/process/Cargo.toml:22` and `model/umpire/protocol/runtime/Cargo.toml:28` carry complete `[lints.rust]`/`[lints.clippy]` blocks. All other member manifests inherit lints. Facade, consumer and the two test-only fixture packages explicitly set `publish = false`; retain those local values.

## Explicit tests and independent nested packages

| Manifest reference | Keys or targets to retain | Membership consequence |
| --- | --- | --- |
| `model/umpire/dev/fixtures/Cargo.toml:9` | `autotests = false`; `[[test]]` fixtures at line 11 and interpreted at line 15, paths `fixtures.rs` and `interpreted.rs` | Explicit framework member; root-level test files would be lost by generic tests-directory discovery. |
| `model/temporal/testing/Cargo.toml:9` | `autotests = false`; fixtures, snapshot_ownership, boundary at lines 11, 15, 19 | Explicit Temporal member. |
| `model/temporal/annotate/syntax/Cargo.toml:9` | `autotests = false`; realize at line 11 | Explicit Temporal member. |
| `model/temporal/api/Cargo.toml:27` | Explicit descriptor test; autotests remains enabled | Temporal member, retain both explicit and automatic targets. |
| `model/umpire/author/umpire/tests/consumer/Cargo.toml:14` | Renamed `author-kit`, `package = "umpire"`, relative `path = "../.."` | Explicit framework member, no nested workspace. Path stays valid. |
| `model/temporal/realization/macros/tests/renamed/Cargo.toml:7` | Independent `[workspace]`; `invalid-enum` at line 10; renamed author-kit, temporal-kit, api-kit paths at lines 13-15 | Remains outside both member lists. All three paths stay valid; its own Cargo.lock remains authoritative. |
| `model/design/native-symbolic/Cargo.toml:8` | Independent `[workspace]`; four local paths at lines 11-14; own dev overflow profile at line 16 | Remains outside both member lists, with its existing Cargo.lock. All paths stay valid. |

The renamed realization control actually executes its nested workspace with offline locked Cargo (`model/temporal/realization/macros/tests/renamed.rs:13`), so checking only the two new main workspace locks misses this consumer. Independent nested workspaces do not inherit either new root profile/lint policy; preserve their current local declarations. No other checked-in Cargo manifests were found under model.

## Workspace dependency declarations

Framework retains all 19 framework path declarations below, with unchanged package versions `0.0.1`. Remove the initial `umpire/` from each path relative to its new declaring root. Outer direct-use requirements are marked in the last column. Outer retained paths keep their present spelling.

| Key | Current path | Declaration | Directly inherited by Temporal member |
| --- | --- | --- | --- |
| `umpire` | `umpire/author/umpire` | `model/Cargo.toml:49` | No |
| `umpire-reference-kettle` | `umpire/reference/kettle` | `model/Cargo.toml:50` | No |
| `umpire-catalog` | `umpire/catalog` | `model/Cargo.toml:51` | Yes |
| `umpire-catalog-macros` | `umpire/catalog/macros` | `model/Cargo.toml:52` | No |
| `umpire-catalog-syntax` | `umpire/catalog/syntax` | `model/Cargo.toml:53` | Yes |
| `umpire-catalog-build` | `umpire/catalog/build` | `model/Cargo.toml:54` | Yes |
| `umpire-dsl` | `umpire/author/dsl` | `model/Cargo.toml:55` | Yes |
| `umpire-dsl-macros` | `umpire/author/dsl/macros` | `model/Cargo.toml:56` | No |
| `umpire-checker` | `umpire/backends/native` | `model/Cargo.toml:57` | Yes |
| `umpire-testing` | `umpire/dev/testing` | `model/Cargo.toml:58` | Yes |
| `umpire-suite` | `umpire/author/suite` | `model/Cargo.toml:59` | Yes |
| `umpire-ir-derive` | `umpire/ir/derive` | `model/Cargo.toml:60` | No |
| `umpire-ir` | `umpire/ir` | `model/Cargo.toml:61` | No |
| `umpire-front` | `umpire/front` | `model/Cargo.toml:62` | Yes |
| `umpire-backend-process` | `umpire/backends/process` | `model/Cargo.toml:63` | No |
| `umpire-tla` | `umpire/backends/tla` | `model/Cargo.toml:64` | Yes |
| `umpire-apalache` | `umpire/backends/apalache` | `model/Cargo.toml:65` | No |
| `umpire-tlc` | `umpire/backends/tlc` | `model/Cargo.toml:66` | Yes |
| `umpire-runtime` | `umpire/protocol/runtime` | `model/Cargo.toml:67` | Yes |

Framework omits all four Temporal-only keys at `model/Cargo.toml:68` (umpire-temporal-api), `:69` (umpire-temporal-annotate-syntax), `:70` (umpire-temporal), and `:71` (umpire-temporal-realization-macros). Outer retains these four unchanged. Copying their unused declarations into the framework would still create escaping or missing paths and violate declaration containment.

Outer members directly inherit 11 framework keys in the table and 12 registry keys below. Framework dependencies resolve their own additional workspace declarations; the outer root does not need duplicate entries solely for those transitive edges. Keeping additional existing framework declarations in the outer root is unnecessary but does not change a dependency graph by itself. `umpire-apalache` is currently an unused workspace dependency declaration, yet its package is an explicit framework member.

| Registry key | Declaration | Versions present in current lock | Outer direct use |
| --- | --- | --- | --- |
| `indexmap` | `model/Cargo.toml:74` | 2.14.2 (`model/Cargo.lock:373`) | No |
| `rustc-hash` | `model/Cargo.toml:75` | 2.1.3 (`model/Cargo.lock:710`) | No |
| `smallvec` | `model/Cargo.toml:76` | 1.16.3 (`model/Cargo.lock:825`) | No |
| `petgraph` | `model/Cargo.toml:77` | 0.8.3 (`model/Cargo.lock:488`) | No |
| `fixedbitset` | `model/Cargo.toml:78` | 0.5.7 (`model/Cargo.lock:270`) | No |
| `imbl` | `model/Cargo.toml:79` | 7.0.2 (`model/Cargo.lock:352`) | No |
| `thiserror` | `model/Cargo.toml:80` | 2.0.21 (`model/Cargo.lock:891`) | No |
| `libc` | `model/Cargo.toml:81` | 0.2.191 (`model/Cargo.lock:410`) | No |
| `rustix` | `model/Cargo.toml:82` | 1.1.5 (`model/Cargo.lock:716`) | No |
| `tempfile` | `model/Cargo.toml:83` | 3.27.0 (`model/Cargo.lock:869`) | No |
| `serde` | `model/Cargo.toml:85` | 1.0.229 (`model/Cargo.lock:750`) | No |
| `serde_json` | `model/Cargo.toml:86` | 1.0.154 (`model/Cargo.lock:780`) | No |
| `toml` | `model/Cargo.toml:87` | 1.1.8+spec-1.1.0 (`model/Cargo.lock:911`) | No |
| `sha2` | `model/Cargo.toml:88` | 0.11.1 (`model/Cargo.lock:802`) | No |
| `tar` | `model/Cargo.toml:89` | 0.4.46 (`model/Cargo.lock:853`) | No |
| `flate2` | `model/Cargo.toml:90` | 1.1.10 (`model/Cargo.lock:276`) | No |
| `prost` | `model/Cargo.toml:92` | 0.14.4 (`model/Cargo.lock:569`) | Yes |
| `prost-types` | `model/Cargo.toml:93` | 0.14.4 (`model/Cargo.lock:611`) | Yes |
| `prost-build` | `model/Cargo.toml:94` | 0.14.4 (`model/Cargo.lock:579`) | Yes |
| `heck` | `model/Cargo.toml:96` | 0.5.0 (`model/Cargo.lock:331`) | Yes |
| `darling` | `model/Cargo.toml:97` | 0.24.1 (`model/Cargo.lock:182`) | Yes |
| `proc-macro-crate` | `model/Cargo.toml:98` | 3.5.0 (`model/Cargo.lock:520`) | Yes |
| `syn` | `model/Cargo.toml:99` | 2.0.119, 3.0.9 (`model/Cargo.lock:831`) | Yes |
| `quote` | `model/Cargo.toml:100` | 1.0.47 (`model/Cargo.lock:626`) | Yes |
| `proc-macro2` | `model/Cargo.toml:101` | 1.0.107 (`model/Cargo.lock:540`) | Yes |
| `manyhow` | `model/Cargo.toml:102` | 0.14.1 (`model/Cargo.lock:428`) | No |
| `prettyplease` | `model/Cargo.toml:103` | 0.2.37, 0.3.0 (`model/Cargo.lock:500`) | Yes |
| `insta` | `model/Cargo.toml:105` | 1.49.0 (`model/Cargo.lock:383`) | Yes |
| `trybuild` | `model/Cargo.toml:106` | 1.0.122 (`model/Cargo.lock:962`) | Yes |
| `proptest` | `model/Cargo.toml:107` | 1.12.0 (`model/Cargo.lock:549`) | No |

Every one of the 30 registry declarations is directly inherited somewhere in the framework member set, including xtask and dev dependencies. Preserve their exact version requirements, default-features settings and feature arrays. The outer direct set is darling, heck, insta, prettyplease, proc-macro-crate, proc-macro2, prost, prost-build, prost-types, quote, syn and trybuild. This is a declaration minimum, not the outer resolved registry closure.

## Policies, lockfiles and feature boundaries

Both root manifests need resolver 3 (`model/Cargo.toml:4`), package defaults version 0.0.1, edition 2024, rust-version 1.99, publish false and MIT (`:41`), and the complete workspace Rust/Clippy policies (`:110`, `:117`). Both need dev overflow checks (`:133`), release overflow checks, debug 1, thin LTO and one codegen unit (`:136`), and guest inheriting release with debug false, symbol stripping, fat LTO, one codegen unit, overflow checks and unwind panic (`:142`). No explicit test/build-override profile exists today. Test/bench defaults derive from Cargo profiles; do not invent replacement settings.

Compilation uses the invoking workspace root profiles, including for framework dependencies compiled from Temporal. A matching framework guest profile alone does not preserve Temporal guest arithmetic or panic behavior. The alias at `model/.cargo/config.toml:5` explicitly runs xtask in dev; moving xtask also requires its framework alias/bootstrap ownership, already specified by the design.

The current lock is format 4 (`model/Cargo.lock:3`). Preserve package name/version/source/checksum tuples and their dependency edges when creating the two roots. Seed each graph from the existing lock rather than resolving broad requirements afresh. Removing unreachable workspace packages may prune entries, so identical lock bytes are not the required result. Both new locks must ultimately pass locked resolution; this report did not execute that proof. Compare surviving tuples including both syn majors and both prettyplease versions listed above. A path package has no registry checksum; package identity must continue to point to the same source tree.

Temporal Cargo resolves framework path packages into the Temporal lock; the framework lock is not a transitive authority for that invocation. Standalone framework builds use its own lock. Nested renamed/native-symbolic workspaces use their own locks as well. Those two existing fixture locks may need reconciliation if Cargo considers their graphs stale after integration, but no dependency/path rewrite is inherently required. Preserve their versions rather than replacing their locks indiscriminately.

Features remain invocation-dependent even with identical versions. Preserve `smallvec` union/const_generics, petgraph graphmap/stable_graph with defaults off, rustix process, TOML std/parse/serde with defaults off, tar defaults off, flate2 rust_backend with defaults off, darling defaults off, syn full/visit/extra-traits and proc-macro2 span-locations (`model/Cargo.toml:76`, `:77`, `:82`, `:87`, `:89`, `:90`, `:97`, `:99`, `:101`). DSL macros additionally request syn visit-mut (`model/umpire/author/dsl/macros/Cargo.toml:20`); Apalache and xtask request serde derive (`model/umpire/backends/apalache/Cargo.toml:18`, `model/dev/xtask/Cargo.toml:14`).

Runtime testing is an explicit feature (`model/umpire/protocol/runtime/Cargo.toml:13`) enabled by its own dev dependency (`:17`), Kettle dev dependency (`model/umpire/reference/kettle/Cargo.toml:23`) and Temporal realization dev dependency (`model/temporal/realization/Cargo.toml:20`). Preserve all three edges. Resolver 3 retains host/build/target separation; splitting workspace-wide selection changes which member features unify. An outer all-features invocation no longer selects all features of framework dependencies as members. The separate framework all-features invocation must retain that coverage.

Pending `be919e0999:model/umpire/backends/tlc/Cargo.toml:13` introduces `[features]` and `strategy-equivalence = []` at line 15. Preserve the integrated declaration and feature identity; do not overwrite the manifest from this older checkpoint. This report makes no assumptions about pending TLC source changes.

The DSL excludes umpire-ir and imbl from wasm target dependencies (`model/umpire/author/dsl/Cargo.toml:18`); moving declarations must preserve that target table. xtask host dependencies must remain outside guest package edges even though they share a workspace. Guest cache keys hash the caller Cargo.lock (`model/dev/xtask/src/guest.rs:243`); metadata and lock publication also use that caller (`:251`). Portable closure publication discards dev-only edges (`model/dev/xtask/src/lock.rs:121`). New lock bytes and moved native tools can invalidate cache identity without a dependency upgrade; preserving versions does not establish guest byte freshness.

The report establishes declaration inventory and source-based inheritance consequences. Resolved membership, selected feature graphs, locked closure acceptance and guest reproducibility remain the implementation proof obligations in fn-254.13.
