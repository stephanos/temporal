---
satisfies: [R5]
---
# fn-253-gomad-fast-validation-gates.7 Replace publish-executable.sh with a Rust publish step

## Description
R5: remove the last shell helper from the gomad build. `tools/gomad/rust/publish-executable.sh` (17 lines of POSIX sh) is called by the `build` and `wasm-test-helper` Make targets. It copies a Cargo output binary to `rust/target/release/<name>`. Owner request 2026-10-25: replace it. No other spec plans a replacement; fn-240.13 only moves it to `rust/tools/`.

Contract to keep (from the script):
- Exactly two arguments, SOURCE and DESTINATION; usage error exits 2.
- SOURCE must be a regular file, not a symlink. An existing DESTINATION must be a regular file, not a symlink. Refusal exits 1.
- It works when SOURCE and DESTINATION are the same path (the default `CARGO_TARGET_DIR` case). Plain `install`/`cp` refuse or truncate here.
- It publishes a new private inode at mode 0755 through a temp file in the destination directory plus an atomic rename, so readers never see a partial binary. The temp file is removed on failure.
- The destination path stays unchanged; fn-240 requires the helper artifact path to stay the same.

Approach: put the step into the fn-253.4 mise build tasks, as a small Rust bin or a subcommand of an existing build-tool bin in the workspace (no new Python, no new shell). Repoint the callers and delete the script. If this lands before fn-240.13, drop the "move publish-executable.sh to rust/tools/" item from fn-240.13 and its spec layout (house convention: cross-spec ordering is recorded in the task). Depends on fn-253.4, because the callers move from the Makefile into mise tasks there.

## Acceptance
- `publish-executable.sh` is deleted. `rg -n 'publish-executable' tools/gomad .flow/specs` finds only historical notes, and the build and test-helper gates call the replacement.
- Golden or e2e fixtures cover: same-path publish succeeds, symlink source refused, symlink destination refused, non-regular destination refused, a published file is mode 0755 with a new inode, and a failed copy leaves no temp file and an unchanged destination.
- `gomad` and `gomad-wasmhost` (with reference-helper) are still published at the same paths, and `validate-model-corpora` consumers still find them.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
