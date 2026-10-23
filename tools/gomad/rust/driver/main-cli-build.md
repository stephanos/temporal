# Main CLI build custody

`main_cli_build.py build --tools-plan PLAN --gnu-plan GNU_PLAN --root FRESH
--role release` provisions the ordinary final `gomad` executable. The separate
closed `host-fixture` role adds only `determinism-fixture`; its linked helper
remains ordinary. The two roles have distinct host identities and cannot be
substituted for one another. Neither role is a measurement-helper build.

The fixed Rust 1.99.1 command is `cargo build --release --locked --offline
--no-default-features -p gomad --bin gomad --message-format=json --manifest-path
REPO/tools/gomad/rust/Cargo.toml`. Only the fixture recipe adds its one fixed
feature. The producer uses the existing GNU envelope, exact private environment,
one build job, fresh owned target, and actual compiler/version journals. It
never invokes a nonexistent main-CLI identity command.

The output `FRESH/main-build.json` has schema `gomad.driver-main-build/v1`.
`binary` names the single-link published copy `FRESH/admitted/gomad`.
`cargo_artifact` names the actual Cargo executable. The receipt retains the
complete known regular tracked source/toolchain/cache closure, raw Cargo and process journals,
actual source manifests from both build scripts, and derived host/helper/runtime
implementation identities. The reader reconstructs the fixed artifact and
library source, profile and feature predicates. Host Cargo package identity uses
its actual `#0.1.0` fragment; the differently named helper package retains
`#gomad-wasmhost@0.1.0`.

`read_build(receipt, frozen=None, expected_role=None)` independently reopens all
of this custody using files and fixed Git inventory queries; it invokes no
compiler, Cargo, main CLI or guest. A frozen binding additionally
checks the ordinary candidate or separately selected fixture role and the full
input table. `input_paths(receipt)` supplies the actual completed build's files
to the freezer. A caller-authored legacy wrapper receipt is refused.

`source_sha256` has the explicit contract `regular-tracked-non-flow/v1`; it
is not a digest of every Git tree object. Tracked symlinks are retained in a
separate literal table, authenticated against the actual committed Git blobs and
rechecked after compilation. Frozen admission binds both the HEAD and complete
symlink table in addition to the regular-source digest. Coherently changing a
working symlink and caller table cannot replace the unchanged Git blob.

These are prefreeze source/build inputs. No later installation, preparation,
guest, qualification or final-section output is predicted or hashed. Actual
installation discovery must independently match the compiled identities. A
source-only test or successful build does not qualify a runtime or original root.
