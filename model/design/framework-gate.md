# Standalone framework gate

Decision for fn-254.9, 2026-10-25. Inventory source: `3d54e19b1bfbce799ada92e4d5ffc7e8db636797`;
all existing file:line references below describe that checkpoint, not the future layout.
This is a design and source inventory; it runs no production gate and claims no timing result.
Historical costs remain in [gate-speed.md](gate-speed.md). The
[prior workspace design](../../.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/planning/20261025/workspace-design.md)
is retained evidence; this note settles its choices.

## Selected layout

Use `cargo xtask gate` from `model/umpire`, implemented by the host-only Rust crate
`model/umpire/dev/xtask`. Give `model/umpire` its own `Cargo.toml` workspace and
`Cargo.lock`. Keep `model/Cargo.toml` as the Temporal workspace, with `exclude = ["umpire"]`
and Temporal members depending on framework crates by ordinary path dependencies.
Explicit framework membership must include the renamed-facade consumer and every guest/control
package currently listed in `model/Cargo.toml:7-33`. No framework manifest inherits policy,
version, dependencies or profiles from above `model/umpire`.

A local Makefile would add a second orchestration language without solving the root paths.
Package selection in the shared workspace would still require its parent manifest/lockfile,
so it cannot pass the directory-only control. The independent workspace is the smallest
layout that proves the requirement. Preserve currently resolved dependency versions/checksums,
R18 admission and guest closure restrictions; this decision admits no new dependency.

| Destination relative to `model/umpire/` | Owner and contract |
| --- | --- |
| `Cargo.toml`, `Cargo.lock` | Framework membership, admitted dependencies, package/lint/profile defaults and locked resolution. Temporal keeps its own equivalent declarations and lockfile. |
| `.cargo/config.toml` | Native dev-profile `xtask` alias; framework-local fmt/clippy/doc aliases. No parent-relative tool include. |
| `rust-toolchain.toml`, `rustfmt.toml`, `clippy.toml`, `deny.toml` | Copy current `model/` policy; preserve audited crate-local capability policies and process unsafe boundary. |
| `mise.toml` | Local Rust 1.99.1, Go 1.27.0, protoc 29.5, Java temurin-27 pins, copied from root `mise.toml:1-6`; omit archived Scala. .3 adds the admitted nextest pin here. |
| `dev/xtask/` | Thin general command front door, installer/retired-name commands and native gate orchestration. Guest operations execute the independent driver. |
| `dev/guest-driver/` | Dedicated `umpire-guest-driver` package/executable with guest build, compiler wrapper, lock admission and support. Its library owns admitted tool/workspace context and raw-OS remap/environment encoding; it has no dependency on the general CLI/gate. |
| `scripts/` | Framework bootstrap and thin tool wrappers relocated from `model/scripts`; no semantic parser duplicated in Bash. |
| `dev/gate/policy/framework-boundary.txt` | Exact declarative gate-only name policy exception described below. |
| `.bin/protoc-gen-go-v1.36.6`, `.bin/protoc-gen-go-grpc-v1.3.0` | Framework-owned pinned generators, provisioned without root Make. |
| `build/tlc/`, `build/apalache/`, `build/evidence/` | Local pinned backend caches/private scratch and backend-control diagnostic output. Preserve symlink refusal, hash admission, locks and cleanup. |
| `target/` | Default framework Cargo/bootstrap/guest outputs; explicit target overrides remain supported. |
| `.config/nextest.toml` | .3's framework runner configuration and bounded JVM group. Temporal gets its own local configuration if its tests use nextest. |
| `dev/gate/budgets.toml` | .12's single framework stage/tier budget table; Temporal stages use `model/dev/gate/budgets.toml`. |

Tiny configuration duplication is intentional: each workspace owns its effective local
files, without symlinks or parent includes. .13 copies the current lint and arithmetic policies,
release/guest profiles and tool pins coherently; subsequent intentional shared-policy changes
update both copies and verify equality of the shared settings. Workspace-specific members,
path dependencies, runner selections and budgets may differ. Root `mise.toml` remains
Temporal/repository-owned; it is not the framework's authority.

## Native tool ownership amendment

2026-10-25, owner requested fixes for the three confirmed accidental-complexity sources in the existing tasks. The source investigation is retained in `.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/planning/xtask-complexity-20261025/investigation.md`. It establishes cache/root/protocol coupling, not a measured performance improvement.

Task .13 separates the dedicated guest-driver package from the general xtask/gate implementation. Move guest orchestration, compiler wrapping, lock admission and shared support into `dev/guest-driver`; installer, retired-name and gate commands stay in `dev/xtask`. The dependency direction is from general xtask to driver context utilities only. Guest operations execute the driver, so the executable copied as the compiler wrapper and hashed for the guest cache is the driver itself. The driver never depends on general CLI/gate code.

One admitted context in the driver library owns framework tool/pin roots, caller compilation workspace, caller-relative output/target paths, sysroot/Cargo roots and canonical longest-root remaps. General commands consume its admitted roots rather than rediscovering repository ancestors. Guest orchestration and compiler wrapping share its raw-OS environment codec and remap list. Preserve native process dispatch, loader/jobserver inheritance, source/lock ambiguity refusal, per-output leases and atomic publication.

Keep the exact driver binary hash, full current caller lockfile hash and compiler input in the guest cache key. Keep existing portable sidecar and per-package lock-closure semantics. Dropping identity inputs or hashing only a version string would weaken the contract. The full lockfile remains a deliberately coarse invalidation input; this amendment isolates unrelated compiled CLI/gate changes, not arbitrary lockfile changes.

Task .13 proves focused driver/key independence and actual-driver invalidation alongside its existing guest reproduction checks. Task .10 adds gate orchestration and a discriminating compiled CLI-only mutation control. CLI bytes must change while driver bytes and the guest cache namespace stay fixed for identical guest/driver/lock/compiler inputs, and a representative guest must reuse admitted artifacts. A data edit that leaves the executable unchanged proves nothing about this coupling. Reuse required reproduction runs where their inputs match instead of adding a timing-baseline campaign. Task .11's directory-only gate builds and invokes both local tools, strips parent prebuilt-tool overrides and refuses missing local configuration instead of finding the parent checkout.

These changes extend existing tasks .13/.10/.11. Unrelated broad validation remains deferred to .8. Existing source/jar/tool admission and Model semantics remain unchanged.

## Gate stage inventory

F = framework, T = Temporal, M = mixed today and split by .10. These are **all 19 timed
`model-check` stages** in `Makefile:1280-1298`, plus its untimed prerequisites/mode setup.
A framework gate must own the F half of every M row. Final root Make owns T stages and
one framework delegation; it must not retain a second framework stage registry.

| Current stage / source line in Makefile | Class | Destination / retained work |
| --- | --- | --- |
| `model-check` prerequisites, 1276 | M | Framework provisions its own two Go generators; root retains Temporal generators and `$(API_BINPB)`. |
| TLC/Apalache mode setup, 1278-1279 | M | Framework backend admission; root propagates effective modes to both workspaces. Apalache stays opt-in until fn-211 resumes. |
| `temporal-guest-freshness`, 1280 | T | Compare committed Temporal guest from the caller workspace. |
| `retired-names`, 1281 | M | Framework filesystem hygiene; root retains repository/archive/Temporal exclusions. |
| `timing-controls`, 1282 | M | Framework timer/status controls; root Make/combined-wrapper controls remain T. |
| `capability-controls`, 1283 | F | Both core disallowed APIs/types, forbid-unsafe roots and audited process policy. |
| `source-boundary`, 1284 | T | Existing Temporal policy tests; add standalone framework gate policy enforcement separately. |
| `protobuf-controls`, 1285 | M | Core/Kettle selector/pin/freshness/error controls F; Temporal selector/handwritten host controls T. |
| `protobuf-freshness`, 1286 | F | Shared Model/realization/annotation/run protobufs. |
| `kettle-protobuf-freshness`, 1287 | F | Reference Kettle protobufs and gRPC output. |
| `temporal-protobuf-freshness`, 1288 | T | Temporal activity/probe protobufs. |
| `fmt`, 1289 | M | All members of each independent workspace. |
| `clippy`, 1290 | M | All members, all targets/features, warnings denied, in each workspace. |
| `docs`, 1291 | M | All members, no deps, rustdoc warnings denied, in each workspace. |
| `tests`, 1292 | M | All member tests and explicit doctests in each workspace; preserve feature/backend modes and ignored identities. |
| `wasm32-wasip1`, 1293 | F | `umpire-dsl`, `umpire-checker` target builds. |
| `core-wasm-clippy`, 1294 | F | Same two packages, library target, locked, warnings denied. |
| `wasm-guest`, 1295 | M | Stock/Kettle/stateful/caught-panic/overflow guests and native reports F; Temporal guest T. |
| `go-host`, 1296 | F | Entire generic Go host module, CGO disabled, `test_dep`, count=1. |
| `kettle-go-host`, 1297 | F | Entire Kettle Go module under the same mode. |
| `temporal-go-host`, 1298 | T | Entire Temporal Go module under the same mode. |

`model-fmt` (`Makefile:1300`), `model-update` (`Makefile:1303`), `model-time` (`Makefile:1307`)
and `model-validate` (`Makefile:1311`) are mixed entry points today. .10 delegates their
framework work and keeps Temporal orchestration at root. `model-validate` still forces
TLC through conflicting environment/Make assignments, runs one full combined gate and
then its benchmark; a failed gate prevents the benchmark. Backend modes retain their
current meaning, and every full gate disables quick-cache reads. .6 adds `model-quick` as the
combined caller, with its generic affected-crate/cache implementation under framework xtask.
No `model-check-changed` definition exists in this source checkpoint; .5 verifies absence
and removes any remaining active references rather than inventing a replacement.

## Every current script

Paths in this table are relative to `model/scripts/`. Relocation is .13's responsibility;
splitting responsibilities and final delegation are .10's.

| Script | Class | Disposition |
| --- | --- | --- |
| `xtask.sh` | F | Move under framework scripts; bootstrap only the framework native tool. |
| `fetch-tlc.sh` | F | Move; local `build/tlc` and pinned hash/version, no repository root. |
| `guest.sh` | F | Move generic wrapper; preserve caller output resolution and explicit compilation workspace. |
| `guest-freshness.sh` | F | Move generic admission/comparison wrapper; Temporal invocation stays root-owned. |
| `proto-go.sh` | M | Framework core/Kettle generation with local plugins; Temporal wrapper owns its schemas/output and passes explicit framework includes/tools. No Temporal selector in framework source. |
| `retired-names.sh` | M | Move framework filesystem half; retain root historical Git exclusions separately. |
| `time-gate.sh` | M | Generic timing/status mechanism F; root combined gate and activity benchmark selection T. |
| `test-time-gate.sh` | M | Framework timing controls F; stop calling root-specific controls from the standalone tree. |
| `test-model-validate.sh` | T | Shipping root Make/combined gate mode and error propagation controls. |
| `test-capabilities.sh` | F | Move; replace old workspace-relative safe-root/xtask paths. |
| `tests/build-guests.sh` | M | Five framework guests and two native report binaries F; Temporal guest build stays T. |
| `tests/proto-go.sh` | M | Split core/Kettle vs Temporal controls, retaining each refusal/status/cleanup case. |
| `tests/.gitignore` | F | Ignore legacy Python bytecode scratch; remove this obsolete entry when relocating the native-only controls. |

## Every current xtask source module

All nine files in `model/dev/xtask/src/` are F. Under the amendment above, `guest.rs`, `compiler.rs`, `lock.rs` and their support move to `dev/guest-driver`; the general dispatcher, installer, pins and retired-name command move to `dev/xtask`. The following table records their source-checkpoint responsibilities. A generic tool can
compile an external consumer without owning that consumer's policies or inputs.

| Module | Contract and coupling to resolve |
| --- | --- |
| `main.rs` | Raw OS argv/status dispatcher; `fetch-apalache` ancestor discovery at :24-29 must stop searching for `model/Cargo.toml`. Add gate dispatch here in .10. |
| `lib.rs` | Host-only modules and native error/result contract. |
| `guest.rs` | Tool admission, caller metadata/lock, guest build and publication; separate tool pins from compilation root (:78-81, :224-255). |
| `compiler.rs` | Portable source/compiler/dependency identities, sidecars and direct child execution; preserve outside-root refusal (:122-155). |
| `lock.rs` | Portable non-development graph closure from caller metadata and lockfile (:225); preserve build/platform edges, duplicate/ambiguous rejection. |
| `support.rs` | Schema 2, SHA, canonical JSON, atomic publication and per-output leases; no root dependency. |
| `fetch_apalache.rs` | Pinned locked/hash-checked installer; replace repository `model/build` construction (:139-143) with local `build`. |
| `apalache_pins.rs` | One release/archive/jar identity shared by installer/backend; fix the backend include when moved. |
| `retired_names.rs` | Raw byte record filter and exact facade-path exception; caller supplies POSIX ERE. Keep grammar/status contracts, independent of the name-boundary scanner. |

## Caches, pins and generators

`model/build/` is absent in this clean worktree. The inventory below classifies declared
cache owners, not an assertion that local cache bytes exist or are reusable.

| Current input/output and evidence | Class | Target |
| --- | --- | --- |
| `model/build/tlc/tla2tools-1.7.4.jar`, hash in `model/scripts/fetch-tlc.sh:8-10` and `model/umpire/backends/tlc/src/runner.rs:11-16` | F | Framework `build/tlc`, same pin/hash; update both producer and admission consumer. |
| TLC private `run-*` scratch, `model/umpire/backends/process/src/support.rs:52` | F | Framework `build/tlc`, retain private mode and explicit cleanup. |
| `model/build/apalache/apalache-0.62.4/lib/apalache.jar`, install lock/staging, `model/dev/xtask/src/fetch_apalache.rs:139-164` | F | Framework `build/apalache`; pins remain shared with backend. |
| Apalache process scratch, `model/umpire/backends/apalache/src/lib.rs:126,182` | F | Framework `build/apalache`; preserve containment/version checks. |
| Synthetic `model/build/tool`, `model/umpire/backends/process/src/jar.rs:106-112` | F | Test-only local synthetic layout; keep cache-escape controls. |
| `.flow/tmp/{tlc-controls,apalache-controls}`, e.g. `model/umpire/backends/tlc/tests/scaling.rs:28`, `model/umpire/backends/apalache/tests/controls.rs:20` | F | Framework `build/evidence`, without requiring `.flow` or Git. |
| `<target>/xtask-bootstrap`, `model/scripts/xtask.sh:11-15` | F | Framework target/bootstrap; clear compiler wrappers/guest flags before native bootstrap. |
| `<target>/{guest-cache,guest-wrappers}/<key>`, `model/dev/xtask/src/guest.rs:243-255` | F mechanism, M caller artifacts | Keyed by native tool bytes, caller lock and compiler; framework/Temporal targets stay separately selectable. Keep persistent sidecar locks and atomic publication. |
| Workspace Cargo artifacts and `OUT_DIR` prost/catalog/fixture generation | M | Each workspace owns its target and locked graph; no committed build outputs. |
| Root `.bin/protoc-gen-go-v1.36.6`, `.bin/protoc-gen-go-grpc-v1.3.0`, `model/scripts/proto-go.sh:27-46` | M today | Framework owns local copies for core/Kettle; root retains its Temporal generator provisioning. Keep exact version probes and lock/symlink-safe publication. |
| Shared prost schemas, `model/umpire/protocol/runtime/build.rs:4-33`; Kettle prost/catalog, `model/umpire/reference/kettle/build.rs:15-34`; fixtures, `model/umpire/dev/fixtures/build.rs:12-25` | F | Remain framework-local; protoc comes from framework pins; generated Rust stays in OUT_DIR. |
| API descriptor/prost, `Makefile:127,1276`, `model/temporal/api/build.rs:28-32` | T | Keep root `$(API_BINPB)` prerequisite and Temporal API generation. Framework must not provision/read it. |
| Committed `temporal/hosts/go/umpire_temporal_guest.wasm` and generic rebuild comparison | T asset, F builder | .13 owns changed-input reproduction/publication after workspace remapping, with Temporal host parity; .10 owns only final freshness orchestration. |
| `umpire/hosts/go/{go.mod,go.sum}` and `umpire/reference/kettle/{go.mod,go.sum}` | F | Remain local; Kettle's `replace ... => ../../hosts/go` is already contained. Temporal Go closure stays T. |
| Root `mise.toml`, model Rust configs and shared Cargo.lock | M today | Split ownership as selected above. Preserve host-only xtask dependencies outside guest closure; no compiler/dependency upgrade. |

## Repository-root coupling inventory

Exactly 45 occurrences of `join("../../../..")` occur under `model/umpire` in this
checkpoint. Each resolves from a crate manifest to the repository root. All belong to
framework backend/tool tests or shared fixtures and must resolve the framework root instead.
Numbers below enumerate **all 45 sites**, grouped only to keep the inventory readable.
Paths in this table are relative to `model/umpire/`.

| File | Lines |
| --- | --- |
| `backends/apalache/src/lib.rs` | 367, 399 |
| `backends/apalache/tests/controls.rs` | 20, 41, 81, 185, 216, 231, 295, 331, 364, 413, 470, 498, 558, 596, 777 |
| `backends/process/src/jar.rs` | 104, 164 |
| `backends/process/tests/process.rs` | 21 |
| `backends/tla/tests/emission.rs` | 18, 71, 121 |
| `backends/tla/tests/narrow_remainder.rs` | 25 |
| `backends/tla/tests/optional_ticks.rs` | 275 |
| `backends/tla/tests/phased.rs` | 191 |
| `backends/tla/tests/sequential_checks.rs` | 23 |
| `backends/tla/tests/shadowing.rs` | 36 |
| `backends/tla/tests/vector.rs` | 101, 269 |
| `backends/tla/tests/world.rs` | 358 |
| `backends/tlc/src/runner.rs` | 750, 784 |
| `backends/tlc/tests/controls.rs` | 170, 220, 241, 272 |
| `backends/tlc/tests/scaling.rs` | 25 |
| `dev/fixtures/fixtures/action_inputs/tests.rs` | 169 |
| `dev/fixtures/fixtures/phased_rejection/tests.rs` | 103, 142, 189 |
| `dev/fixtures/fixtures/world_fairness/tests.rs` | 267 |
| `dev/fixtures/interpreted.rs` | 246 |
| `dev/testing/src/backend_tests.rs` | 442 |

The following additional couplings are not part of that 45-site count. Internal relative
fixture includes stay internal; do not mechanically shorten every `../` string.

| Current file:line | Coupling and selected change |
| --- | --- |
| `model/umpire/backends/process/src/support.rs:52,120-127` | Repository-shaped TLC scratch plus root mise Java lookup → local framework `build/tlc` and local pins. |
| `model/umpire/backends/tlc/src/runner.rs:16,757-766,791-796` | Cache constant and synthetic `model/build`/`model/scripts` escape controls → framework layout, preserving actual refusal tests. |
| `model/umpire/backends/apalache/src/lib.rs:16,126,182` | External pins include and repository-shaped cache → moved local `dev/xtask/src/apalache_pins.rs` and `build/apalache`. |
| `model/umpire/backends/apalache/tests/controls.rs:252` | Pins source include → moved framework-local source, without a duplicate pin. |
| `model/umpire/dev/testing/src/backend.rs:108-111` | Fixture ancestors select first mise root → explicit framework tool root, independent of fixture location. |
| `model/umpire/author/dsl/tests/readme.rs:10` | Includes `model/README.md`, outside standalone copy → framework README owns the compiled example and matching test. |
| `model/umpire/hosts/go/host/host_test.go:580`, `model/umpire/hosts/go/host/stateful_test.go:23`, `model/umpire/hosts/go/host/current_contract_test.go:21`, `model/umpire/hosts/go/runner/fake_test.go:31` | Default `../../../../target` → framework target fallback; preserve explicit artifact/target overrides. |
| `model/umpire/hosts/go/README.md:13-16,27,35-38`, `model/umpire/reference/kettle/README.md:20` | Parent workspace target/build guide and consumer-specific guest prose → local guide, move Temporal-only guidance outside framework. |
| `model/umpire/reference/kettle/kettle_test.go:73`, `model/umpire/reference/kettle/snapshots_test.go:32` | Old script/workspace diagnostics → framework-local instructions; keep ignored live snapshot semantics. |
| `model/dev/xtask/src/guest.rs:79-81,224-255,353-357` | Parent pins, caller current directory/lock/metadata and portable remaps → distinct tool and compilation roots. |
| `model/dev/xtask/src/main.rs:24-29`, `model/dev/xtask/src/fetch_apalache.rs:139-143` | Discover repository via `model/Cargo.toml`, construct model/build → framework-owned root/cache. |
| `model/scripts/proto-go.sh:27-46`, `Makefile:305-317,1276` | Root generator provisioning and prerequisite dependency → local framework tools, retain root Temporal generator rules. |
| `model/scripts/retired-names.sh:4-18,33-35` | Git root, historical paths and Temporal compatibility exclusion → separate framework filesystem and repository policy owners. |
| `model/scripts/time-gate.sh:5,39-43`, `model/scripts/test-model-validate.sh:4-18` | Parent Make invocation/control extraction → framework timer mechanism versus combined root wrapper. |
| `model/scripts/tests/build-guests.sh:4-16`, `model/scripts/tests/proto-go.sh:6,46-71` | Shared workspace and Temporal selector/payload → split callers and controls. |
| `model/Cargo.toml:5-149`, `model/Cargo.lock`, `model/.cargo/config.toml:1-5` | Shared membership, dependency/profile inheritance, aliases/resolution → two independently closed workspaces. |

## Standalone name policy and dependency independence

The standalone gate itself rejects planted Temporal names and dependencies. The existing
Temporal test remains an independently owned external policy at
`model/temporal/testing/boundary.rs:8-25`; its rs/proto/go/toml scope and exact GitHub
organization exception are unchanged. The reusable scanner
`model/umpire/dev/testing/src/boundary.rs:25-30` remains vocabulary-agnostic, as do all
Model/compiler/runtime libraries. No core conformance policy changes. Retired-name filtering is a different policy and does
not discharge this name ban.

Select one **narrow gate-data exception**: `model/umpire/dev/gate/policy/framework-boundary.txt`.
This exact regular file contains declarative forbidden fragments, the current occurrence-level
organization-path allowance and control payload data. It can use a versioned TOML document
inside the `.txt` file, parsed with the already-admitted parser. It is not Rust source or
runtime configuration. The extension keeps the existing Temporal scan unchanged; the new
gate explicitly exempts only this file, not all txt files or its containing directory.
Reject missing/malformed/empty policy, unknown fields and symlink substitution. No string
obfuscation, generated forbidden spellings in Rust, or executable policy callbacks specific
to Temporal. This is development policy ownership, not knowledge of system behavior.

The gate's filesystem inventory includes source, tests, generated committed inputs, prose,
scripts, manifest/lock/config files and extensionless files. Scan path names and file bytes
for supplied fragments, with deterministic relative path/line diagnostics; a NUL or non-UTF-8
file is not an exemption. Only the exact policy data and named owned output trees (`target`,
`build`, `.bin`, including an admitted explicit target override) are excluded from name
contents; dependency containment and policy validation still apply. Git administration
metadata is not an input. Do not follow source symlinks; refuse them. Outputs must be
separate from source so an override cannot hide source by declaring the workspace itself
as its target. The isolated copy uses the same current-input inventory and exclusions.

Preserve only the existing organization allowance, at the exact matched occurrence in
`github.com/temporalio/<nonempty-module>`; a second forbidden occurrence in that token/line
still fails. Move the current Temporal-specific README paragraph outside the framework.
All ordinary README/tests/bootstrap files remain scanned. Plant control payloads from the
policy data in a disposable copy, rather than storing the forbidden names in generic test source.

Check both declarations and resolution for dependency independence. Enumerate local paths
in normal/build/dev/target dependencies, workspace dependency entries, patches/replacements,
and Go local replacements, including unused declarations; resolve relative to the declaring
manifest and require canonical containment inside the framework root. Reject missing paths
and escaping symlinks explicitly. Also check every resolved Cargo local package manifest
(`source == null`) and its graph edges under locked metadata; registry cache packages are
legitimate external toolchain inputs, not local-path escapes. A renamed dependency or an
inactive target edge cannot evade the declaration check. The control includes an existing
benign crate outside the root as well as an absent outside path, so an incidental Cargo
resolution failure cannot masquerade as a proved containment check.

Source-path checks resolve literal Rust `path`/`include*` references from their source file
and manifest-root joins from their actual crate manifest, then require containment. The
planted manifest-root `../../../..` escape must fail by this structural check. Do not ban
the spelling globally: several deeper fixture includes with four or more parents remain
inside the framework. Dynamic backend/cache paths still pass the existing runtime root
admission, including isolated synthetic roots used by negative controls. .13 changes the
45 known root joins; .10/.11 enforce the source-path guard and its planted control.

This is the smallest explicit exception reconciling R7's local name ban with its agnostic
libraries: operator policy data may name the banned system; ordinary framework files may
not. The conductor must record this exact exception with the selections in Decision Context
before .10/.11 implement it. The ban is not transferred to the combined Temporal gate.

## Migration boundary and verification

**.13 owns workspace/tools/identity.** Move manifests, lockfiles, xtask and framework helpers;
localize all roots/pins/config/generators above; update affected test paths and documentation.
Preserve the existing root entry points while migrating. In particular, the **intermediate
root gate must run fmt/clippy/docs/tests against both workspaces**, plus every existing
non-Cargo stage; changing just `model/Cargo.toml` silently drops framework coverage.
Keep temporary thin root helper delegation until .10 removes the replaced paths.

Tool root and guest compilation workspace are different. Bootstrap/admit tools from the
framework's local pins; run guest Cargo metadata/build with the caller's manifest/workspace,
lockfile and output/target context. Resolve relative outputs/target overrides before a chdir.
An extension guest's metadata includes its framework path dependencies, and its caller lock
is authoritative for that entire resolved guest graph. Add the declared framework source
root to portable compiler remaps when needed, alongside caller workspace, sysroot, registry
and build outputs; longest-root matching must distinguish nested framework sources from
caller sources. Keep all undeclared outside-root sources refused. A standalone framework
uses only its own root, never a fabricated parent repository.

Moves change native tool bytes and workspace paths and may change guest identities. .13
owns invalidation, source remapping/lock-closure controls, current-input reproducibility,
recorded old/new hashes, refreshed committed Temporal guest bytes and host freshness/parity
on the joined inputs. Historical cache hits/old bytes are not acceptance. No need to admit
new Rust dependencies merely to implement these filesystem/config changes.

The conductor must align .13's declared touch set before dispatch with these concrete
path-sensitive changes: framework scripts, DSL README control, Go target fallbacks and
consumer documentation are not all named in its current touch list. This note authorizes
no worker to edit outside its assigned touch set. Final proto/Host integration remains .10.

**.10 owns gate composition.** Add the framework Rust stage registry/executor and final root
delegation; split mixed protobuf/guest/hygiene/timer controls; preserve stage diagnostics,
statuses, backend modes and dependency order. Minimal bootstrap disables mise auto-install
before its first probe, requires installed pinned Rust/Cargo, clears guest wrappers/flags
and builds the locked native xtask; no Git or parent Makefile is needed. Rust owns semantic
admission. Framework hygiene enumerates current files itself and feeds equivalent raw
`path NUL line-number NUL bytes LF` records to the existing retired-name filter with POSIX
ERE matching. Retain malformed-stream/SIGPIPE/error propagation controls. Root may keep
its separate Git historical exclusions. No root `git grep` fallback is permitted standalone.

**.3/.5/.6 use this layout.** .3 puts nextest pin/config and JVM concurrency group under the
framework root, retains explicit doctests and maps full logical identities (package, target,
name, ignored state, features/backend modes). .5 overlaps non-Cargo work with Rust without
losing failures and keeps Apalache out of routine runs. .6 puts generic affected-crate,
reverse-dependency and TLC cache work in framework xtask; the root caller selects the
Temporal activity. Native/interpreter checks always run; full validation never reads that
cache. .12 adds the budget table/registry checks in the selected paths; adding any stage
requires its budget entry, hard work counts and the specified two-attempt time rule.

Future proof obligations, not results of this note:

- .13: locked metadata/build closure from both workspaces; guest_pipeline/guest_identity,
  installer/retired-name and backend cache/root-refusal controls after moves; deterministic
  guest reproduction/freshness/parity; intermediate dual-workspace coverage inventory.
- .10: shipping combined mode/failure controls (`model/scripts/test-model-validate.sh:70-92`), named
  planted failure for every stage, and complete before/after stage/test identity accounting.
  Keep explicit core wasm packages, all five framework guests, both native report binaries,
  core/Kettle protobufs and both entire Go modules; doctests need their own invocation when
  nextest replaces cargo test. Logical test counts alone do not prove preservation.
- .11: run the real framework entry point on a filtered copy of **current inputs** without
  Temporal, parent Make/.bin, or `.git`. Suppress only recursive execution of this same
  copy-control stage. Plant name, foreign dependency and root-escape failures, including an
  existing outside dependency; require named failure reasons. Do not substitute `git archive
  HEAD`, which omits current uncommitted inputs, or fake tool success for this final proof.

## Decision Context entry for conductor integration

2026-10-25, fn-254.9 delegated selection: adopt framework-owned Rust `cargo xtask gate`
at `model/umpire/dev/xtask` and independent `model/umpire/Cargo.toml`/`Cargo.lock`;
Temporal remains at `model/` with `exclude = ["umpire"]` and path dependencies inward.
The complete stage/script/module/cache inventory and all 45 root joins are recorded in
`model/design/framework-gate.md` at the inventory checkpoint named there. Copy tiny local
policy/tool configs; framework pins/tools/caches/nextest/budgets have no parent dependency.
Keep framework tool root distinct from caller guest workspace/lock/output; .13 owns remaps,
locked closure, current-input reproduction and refreshed Temporal guest publication. .13's
intermediate root gate checks both workspaces; .10 owns final orchestration and mixed-control
splits. Preserve R7's standalone planted-name/dependency rejection using exactly
`dev/gate/policy/framework-boundary.txt` as declarative gate-policy data exempted from its
own name scan, with no directory/source-code exemption. The existing Temporal vocabulary
policy stays Temporal-owned; generic libraries/scanners/compiler/runtime remain agnostic.
Filesystem hygiene and the standalone copy need no Git; use declared current inputs and
preserve path/symlink, raw-byte/status and dependency-containment checks. No Model or
conformance policy changes are authorized by this design.

## Task .13 source checkpoint

The framework now owns its independent 28-member Cargo workspace, lockfile, tool
pins and native tools. The seven-member outer workspace retains Temporal inputs
and a thin `temporal-xtask` command bridge. Root guest scripts and `cargo xtask`
select the framework workspace for its five declared guests and the outer workspace
for extension guests, resolving caller-relative output and target paths first.
The dedicated guest driver owns compiler wrapping and admitted context; the general
CLI can use its context library but cannot enter a guest runtime closure.

The durable task13 proof index is
[task13/current-index.json](../../.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task13/current-index.json).
It records the source identities, 925 preserved logical tests plus five new native
controls, all 563 preserved snapshot/UI hashes, unchanged registry resolutions,
both locked build/test-compilation closures, an isolated framework build, and
current-input guest reproduction/freshness and Go host results. Earlier failed or
inconclusive attempts remain identified in the same evidence.

This checkpoint does not execute the combined full gate. Task .10 still owns final
gate composition, protobuf/host orchestration and renewal of driver independence
with a compiled general-CLI mutation after gate code is added. Task .11 owns the
real standalone full-gate control; .8 retains unrelated workspace execution and
combined validation. Their deferred checks are not task .13 results.

Review corrections preserve timer caller directories in step mode and align all
root Go guest/native paths with the builder's target. Cargo home resolves before
commands change directories, with existing symlinks canonicalized consistently
for tool execution and source remaps. The optional-eager TLC control creates its
evidence parent when absent. The canary policy pins the exact committed Temporal
guest; all other policy fields and refusal checks are preserved. The current proof
index retains the initial checkpoint and trial failures as historical evidence.
