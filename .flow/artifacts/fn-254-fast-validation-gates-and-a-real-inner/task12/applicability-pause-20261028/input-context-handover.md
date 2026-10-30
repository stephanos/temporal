# fn254.12 exact-input custody source handover

Owned source: model/umpire/dev/xtask/src/gate/budget_context.rs only.
Branch: work/fn25412-input-context. Base: 5c1ee94045.
RED checkpoint: d9457aedab. Implementation checkpoint: 95c843f9fa.
Parent observed RED in .flow/tmp/fn25412-context-red.log (7 fixtures, 6 expected custody-not-implemented failures; validation-only fixture already failed closed).
Worker ran no builds, tests, actual tools, environment mutation, push, or rebase.
Formatting: /home/agent/.rustup/toolchains/1.99.1-aarch64-unknown-linux-gnu/bin/rustfmt --config-path model/umpire/rustfmt.toml model/umpire/dev/xtask/src/gate/budget_context.rs
Static check: git diff --check HEAD~2..HEAD passed.

## API

SourceRoot { label: String, root: PathBuf, excluded: Vec<PathBuf> }.
Inputs::capture(&[SourceRoot], &[PathBuf], &impl Serialize) -> io::Result<Inputs>.
Inputs::{verify_unchanged() -> io::Result<()>, digest() -> &str, authority() -> &Authority}.
Authority implements Serialize and contains version, labeled relative source files/exclusions, ordered canonical raw tool path bytes/content hashes/executable bit masks, and frozen semantic JSON.
PinnedImage::{digest() -> &str, verify_unchanged() -> io::Result<()>, copy_to(&Path) -> io::Result<()>}.
pin_current_image() -> io::Result<PinnedImage>: Linux opens /proc/self/exe, other OS Unsupported.

Caller supplies command/argv, explicit env, mode/tier, budgets, selected work, compiled observer capability and any other behavior-affecting semantics. Captured semantic JSON is immutable; each fresh invocation recaptures current semantic inputs. Completion checks verify_unchanged; retry must additionally compare fresh invocation digest with admitted digest. Root owns factory, source-root coverage and narrow exclusion policy. No checked-in source allowlist.

Canonical absolute source roots required; exclusions relative to each source root with literal normal components only. Defaults root-only .git, target, build, .bin. Missing excluded outputs allowed, existing symlink exclusion or symlink ancestor refused. Source inventory intentionally local because boundary::files accepts only one target exclusion and has no identity guards.

Sorted raw relative file paths, bytes SHA256, exact executable masks and labeled exclusions enter digest; absolute source root locations and inode identities do not. Equivalent copied roots with equal labels/exclusions hash equally. Source/tool symlinks and special files refused. Traversal maximum 1,000,000 entries, 2 GiB/file, 64 GiB source bytes; streaming 64 KiB hashing, bounded directory membership collection.

File identity guards catch source replacement even with identical bytes. Directory guards retain device/inode/mode but omit timestamp/size so excluded output creation does not invalidate admission. Directory metadata checked before/after each scan and full source re-inventory occurs on capture completion and completion verification. External tools retain open File plus original path identity, rehash both and reject path alias/replacement even when byte-equivalent. Pinned running image intentionally survives pathname replacement; retained bytes and inode/device/mode still verified. copy_to uses create_new mode0700, checks copied digest and retained image again; caller supplies protected private destination and sibling driver locators.

## Remaining parent work

Integrate implementation, run focused GREEN fixtures, clippy and relevant tests under parent-owned runtime lease. Register module (parent reports already done). Connect actual input factory and main-before-Cargo running-image pin. Exact macOS image pin unsupported/deferred, not a claimed guarantee.

## Parent compiler feedback corrections

612d45272e replaces unsupported sha2 0.11 Array LowerHex formatting with existing standalone-style explicit hex rendering for both byte and streaming hashes.
38521ac380 uses safe OpenOptionsExt O_NOFOLLOW|O_NONBLOCK for source/tool opens, preventing symlink/FIFO substitution races from following symlinks or blocking before descriptor type validation.
Both source-only checkpoints formatted with pinned rustfmt 1.99.1 and passed git diff --check. Worker did not run builds/tests. Parent-owned GREEN rerun remains required.
