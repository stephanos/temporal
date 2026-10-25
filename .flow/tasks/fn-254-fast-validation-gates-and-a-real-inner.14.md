---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.14 Assume mise is installed: drop tool-presence and pin probes, fail hard

## Description
Owner decision (2026-10-25): "we don't need to check for mise everywhere all the time, we can assume its installed - or fail hard". Assume mise and the pinned tools in `mise.toml` are installed. Do not probe for them first; run `mise exec -- <tool>` and let a missing tool fail the command. This replaces fn-247 R2's missing-tool and probe-failed-tool admission and negative controls.

Runs after .13, on the files where .13 left them (framework copies under `model/umpire/`, plus any root-side copies).

**Size:** S
**Files:** `model/umpire/scripts/xtask.sh`, `model/umpire/scripts/proto-go.sh`, `model/umpire/dev/guest-driver/src/guest.rs`, `model/umpire/backends/process/src/{support,jar,tests}.rs`, `model/umpire/dev/guest-driver/tests/**` (the guest pipeline controls), `model/umpire/scripts/tests/proto-go.sh`, the matching root-side scripts if any are left, and READMEs that document the prerequisite checks.
**Touches:** [model/umpire/scripts/xtask.sh, model/umpire/scripts/proto-go.sh, model/umpire/scripts/tests/proto-go.sh, model/umpire/dev/guest-driver/**, model/umpire/backends/process/**, model/scripts/**, model/README.md, model/GUEST_BUILDS.md, model/umpire/**/README.md]

### Remove
- `xtask.sh`: the `command -v mise` guard. A failed `mise exec -- cargo build` already exits non-zero with the bootstrap failure message.
- Guest driver `prerequisites()`, which runs on every freshness call: the `mise ls --installed --json` probe for each tool, the executable lookup under each install path, the `rustup toolchain list` probe and the rust-src and wasm32-wasip1 sysroot checks. `rust-toolchain.toml` already declares those components and targets.
- `pinned_java`: the text match on `java = "temurin-27"` in `mise.toml` and the `java -version` Temurin 27 check (`validate_java`). Keep `mise which java` to find the path, and fail hard if it fails or prints no absolute path.
- `proto-go.sh`: the `protoc --version` comparison.
- The negative controls and fake-`mise` stubs that only test those checks: `missing-mise` in the guest pipeline test, the `temurin-21` refusal in `jar.rs`, the `validate_java` tests, and the wrong-protoc case in the proto-go control.

### Keep
- `MISE_AUTO_INSTALL=false` and `MISE_EXEC_AUTO_INSTALL=false`. These make a missing tool fail instead of installing silently.
- Every place that runs a tool through mise (`mise exec`, `mise which`).
- The check that `mise.toml` and `rust-toolchain.toml` pin the same Rust version. It compares two files and does not probe mise. Delete it too if it only feeds the removed checks.
- The archive and jar SHA-256 checks and guest cache identity. Neither checks that a tool is installed.

## Acceptance
- [ ] No `command -v mise`, no `mise ls`, no `rustup toolchain list` probe, no sysroot readiness check, no Java pin text match or `-version` check, and no protoc version comparison remain in framework or Temporal tooling. Verify with `rg -n 'command -v mise|mise ls|toolchain list|validate_java|temurin-27|libprotoc' model --glob '!**/target/**'`, which may still list `mise.toml` itself.
- [ ] The negative controls for those checks are deleted, not left skipped; the remaining xtask, guest-driver, process and proto-go controls pass.
- [ ] Missing mise fails hard: with `PATH` lacking mise, `xtask.sh` exits non-zero with a clear message and builds and publishes nothing (one focused control).
- [ ] Guest bytes are regenerated only if the guest driver's identity changes; record old and new hashes as usual.
- [ ] Focused checks: guest-driver and xtask tests, `umpire-backend-process` tests, TLC runner tests and the proto-go control. The full gate is not required.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
