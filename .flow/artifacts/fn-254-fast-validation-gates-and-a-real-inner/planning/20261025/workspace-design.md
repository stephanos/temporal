# fn-254 workspace and framework-gate design

Read-only design for tasks .9/.13/.10/.11. This scratch report is the only write made by this scout. No source, Flow metadata, commits, builds or tests were changed or run. Implementation admission remains after the authorized fn-246 drain and committed closures.

## Recommended decisions for .9

Use a framework-owned xtask at `model/umpire/dev/xtask`, exposed as `cargo xtask gate` from `model/umpire`. Make `model/umpire` an independent Cargo workspace with its own Cargo.lock. Keep the Temporal workspace at `model/`, with Temporal members, `exclude = ["umpire"]`, and ordinary path dependencies into the framework. This is stronger standalone evidence than package selection from the current shared workspace.

The current shared members, dependency declarations, lint policy and profiles are at `model/Cargo.toml:5-149`. Preserve versions and checksums where still resolved; preserve the current lint settings, all checked-arithmetic settings and the guest/release profiles. Copy the small policy/toolchain files into the framework rather than having its manifests or configuration include files above its root. The Temporal workspace retains its own corresponding policy. The framework owns `.cargo/config.toml`, rust-toolchain.toml, rustfmt.toml, clippy.toml, deny.toml, mise.toml and the later nextest configuration. Current tool pins are actually in repository-root `mise.toml:1-6`, not model/mise.toml. Framework pins need Rust, Go, protoc and Java; .3 adds nextest. Scala need not become a framework requirement.

## Preserve R7's standalone name ban

R7 requires the isolated framework gate itself to reject a planted Temporal name. Do not transfer that obligation to the combined Temporal gate or silently weaken it.

The existing scanner is appropriately generic: `model/umpire/dev/testing/src/boundary.rs:23-30` accepts caller-owned extensions, forbidden vocabulary and an exception callback. The current system-specific vocabulary and exact GitHub organization exception are deliberately outside the framework at `model/temporal/testing/boundary.rs:8-25`. Retired-name filtering is a separate policy and is not a substitute for this check.

.9 must explicitly decide how a framework-owned development gate obtains its deny vocabulary without teaching any generic library, compiler, runtime or Model core about Temporal. Recommended decision: a narrow, documented exception for one exact gate policy-data file, for example `dev/gate/policy/framework-boundary.txt`. That data contains the deny vocabulary and exact existing organization-path allowance; generic gate/scanner code consumes the data. The policy-data exception is distinct from dependency independence: it authorizes only this declarative data, never a Temporal crate, schema, root path, tool or API dependency.

Document the exact exemption and its rationale in the design note and R7 acceptance before implementation. No directory-wide exemption, broad ignored glob, or exception for arbitrary Rust/test/README files is authorized. A planted Temporal identifier anywhere in the admitted scanned tree outside that exact policy-data location must fail the isolated gate with a named reason. Controls must also reject a planted foreign path dependency and an escaping repository-root path. Place control payload vocabulary in the same admitted data mechanism or construct generic fixtures from supplied policy input; do not introduce new Temporal literals into generic test source.

The current source scanner covers rs/proto/go/toml. .9 must state the intended scope for new gate files and prose explicitly, rather than letting a new extension evade enforcement. Existing framework prose already names Temporal at `model/umpire/hosts/go/README.md:35-36`; move that consumer-specific guidance to Temporal-owned documentation if the final ban includes prose. Preserve only explicitly admitted upstream organization URL exceptions. These are policy-data and scan-scope decisions, not a waiver of R7.

## .13: independent workspaces, owned tools, roots and identity

.13 owns the new framework manifests/lockfile and the Temporal manifest/lockfile adjustment; moving `model/dev/xtask` to `model/umpire/dev/xtask`; local tool/config files; helper relocation and tool provisioning; framework cache/root resolution; and tests whose paths or fixtures change. This includes backend process/TLC/Apalache roots, TLA world/test root callers, the Apalache pins include, generic fixture backend root resolution, protobuf generator discovery, and the Go tests' target fallbacks. Keep semantic runner optimization and test consolidation in .2/.4.

Keep tool root separate from compilation workspace. Guest orchestration currently uses current_dir for its compilation workspace and reads that workspace's Cargo.lock/metadata (`model/dev/xtask/src/guest.rs:224-255`), which is correct for a generic caller compiling an extension. Its prerequisite check currently reads a parent mise.toml (`:79-81`), which must change to framework-owned tool admission. The Temporal wrapper may select its own compilation workspace while invoking the framework tool. Do not make the framework tool know Temporal names. Preserve caller-relative output handling and document target-root selection. Do not infer the framework tool root from a Model fixture's ancestors (`model/umpire/dev/testing/src/backend.rs:108-111`).

Native xtask bytes and the compilation-workspace lockfile participate in guest cache identity (`guest.rs:243-255`); workspace remapping participates in portable identities (`:353-357`). A move or split can invalidate caches and change guest bytes. Preserve identity admission and reproducibility, not historical cache hits. .13 owns current-input reproduction/freshness and any necessary joined Temporal guest publication at `model/temporal/hosts/go/umpire_temporal_guest.wasm`, with parity checks. The framework tool must continue to admit sources within declared roots; do not accept arbitrary outside-root sources merely to accommodate the split.

Important intermediate checkpoint: the existing Makefile's cargo --workspace stages would silently stop covering framework crates after the split. .13 must make those existing Rust stages check both workspaces, and keep helper compatibility bridges working, until .10 replaces them with final delegation. Both framework and Temporal coverage must exist at .13 closure.

## .10: framework gate and root delegation

.10 owns the framework gate's stage registry and execution, the final root Makefile delegation, timing/environment/failure propagation, the separation of mixed scripts and controls, and framework-local hygiene enumeration. It does not reopen .13's workspace or tool-root design.

Separate framework stock/control/Kettle/panic/overflow builds from the Temporal guest (`model/scripts/tests/build-guests.sh:9-16`), core/Kettle protobuf checks from Temporal checks (`model/scripts/tests/proto-go.sh:46-71`), and framework name policy from repository historical exclusions (`model/scripts/retired-names.sh:9-18,33-35`). The Temporal side keeps API_BINPB prerequisites, Temporal guest/protobuf/host stages, Temporal tests and its external consumer policy.

No .git bootstrap is necessary. The standalone framework hygiene stage should enumerate declared current source roots with filesystem operations and feed the existing native retired-name filter with equivalent records, preserving its raw-byte and exit-status contracts. The root repository may retain Git-based historical exclusions. Do not substitute the UTF-8 lexical boundary scanner for the different retired-name byte-stream contract. Exclude only explicitly owned build/cache outputs from source enumeration.

## Ownership and future verification

.13 and .10 serialize shared Makefile/manifests/tool-path changes. .2 and .4 have no new semantic prerequisite merely because they edit shared files; serialize/rebase those exact paths and preserve their separate acceptance. .3/.5/.6 target the selected framework layout and include framework xtask paths in their ownership.

Future .13 evidence: locked metadata/build closure from each independent workspace; moved xtask guest_pipeline, guest_identity, apalache_install and retired_names controls; process/TLC/Apalache root admission controls; current-input guest reproduction/freshness and host parity; complete dual-workspace root coverage.

Future .10/.11 evidence: combined stage inventory and full test inventory preserved exactly once; required-TLC propagation under conflicting environment and Make assignments (existing controls at `model/scripts/test-model-validate.sh:70-92`); named per-stage planted failures; framework gate passing on a filtered copy of current inputs with no root Makefile, root .bin, Temporal tree or .git; planted name/dependency/root-escape failures. The child run must suppress only its own standalone-copy stage to prevent recursion. Do not use git archive HEAD to stand in for current uncommitted inputs.

Documentation ownership includes model/README.md, framework README and moved xtask README, model/GUEST_BUILDS.md, current module-map/AGENTS workspace guidance, and the design note. The root conductor alone owns MILESTONES.md.
