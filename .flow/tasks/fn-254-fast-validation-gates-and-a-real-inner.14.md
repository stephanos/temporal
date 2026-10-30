---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.14 Assume mise is installed: drop tool-presence and pin probes, fail hard

## Description
Owner decision (2026-10-25): "we don't need to check for mise everywhere all the time, we can assume its installed - or fail hard". Assume mise and the pinned tools in `mise.toml` are installed. Do not probe for them first; run `mise exec -- <tool>` and let a missing tool fail the command. This replaces fn-247 R2's missing-tool and probe-failed-tool admission and negative controls.

Runs after .13, on the files where .13 left them (framework copies under `model/umpire/`, plus any root-side copies).

**Size:** S
**Files:** `model/umpire/scripts/xtask.sh`, `model/umpire/scripts/proto-go.sh`, `model/umpire/dev/guest-driver/src/guest.rs`, `model/umpire/backends/process/src/{support,jar,tests}.rs`, `model/umpire/dev/guest-driver/tests/**` (the guest pipeline controls), `model/umpire/scripts/tests/proto-go.sh`, the matching root-side scripts if any are left, and READMEs that document the prerequisite checks.
**Touches:** [model/umpire/scripts/xtask.sh, model/umpire/scripts/tests/bootstrap.sh, model/umpire/scripts/proto-go.sh, model/umpire/scripts/tests/proto-go.sh, model/scripts/proto-go.sh, model/scripts/tests/proto-go.sh, model/umpire/dev/guest-driver/src/guest.rs, model/umpire/dev/guest-driver/tests/guest_pipeline.rs, model/umpire/dev/guest-driver/tests/pipeline/support.rs, model/umpire/backends/process/src/support.rs, model/umpire/backends/process/src/tests.rs, model/umpire/backends/process/src/jar.rs, model/umpire/dev/guest-driver/README.md, model/README.md, model/GUEST_BUILDS.md, model/temporal/hosts/go/umpire_temporal_guest.wasm, tools/canary/policy/production-canary.json]

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
Removed repeated mise/tool/sysroot readiness probes, Java pin/version checks and protoc version comparisons while preserving explicit tool execution, disabled auto-install, integrity hashes and guest cache identity. Obsolete controls were removed; actual missing-mise bootstrap failure is covered. All 118 focused Rust tests, three shell suites, clippy/fmt/syntax checks and a required real TLC counterexample control pass. Two fresh driver/guest builds reproduce the committed guest bytes exactly, so no WASM/policy publication was needed. Shared Make tool entry now disables auto-install before its first mise invocation (task6 producer); same-primary-session re-review SHIP verifies the fail-hard contract. Unrelated broad final gates remain task8.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5ba482b5b73ca4d85ab758568d9a25ec4c4126b4, 773fb58152
- Tests: cd model/umpire && mise exec -- cargo test --locked -p umpire-guest-driver -p umpire-backend-process -p xtask (110 tests passed), cd model/umpire && mise exec -- cargo test --locked -p umpire-tlc --lib runner::tests (7 passed), cd model/umpire && mise exec -- cargo clippy --locked -p umpire-guest-driver -p umpire-backend-process -p xtask --all-targets -- -D warnings, bash model/umpire/scripts/tests/bootstrap.sh, bash model/umpire/scripts/tests/proto-go.sh, bash model/scripts/tests/proto-go.sh, bash -n on all six edited shell files; rustfmt --check on all five edited Rust files; git diff --check, bash ROOT/.flow/tmp/fn254-task14/run-guest-proof.sh (dual fresh target build; exact cmp with each other and committed WASM), cd model/umpire && UMPIRE_BACKENDS=require mise exec -- cargo test --locked -p umpire-tlc --test main real_tlc_initial_invariant_has_a_typed_counterexample -- --exact controls::real_tlc_initial_invariant_has_a_typed_counterexample (1 real TLC test passed), Actual shipping Make initial-mise inherited-auto-install RED/GREEN and same-primary-session re-review SHIP; all25 current task14 source-manifest entries plus retained proof hashes verified.
- PRs:
## Parallel ownership refinement

Own the exact listed files, including the missing-mise bootstrap failure control. Task .6 owns only backend-process fingerprint.rs and lib.rs; do not edit either. Preserve all existing task requirements. Guest-driver input changes require exact guest reproduction and old/new hashes; publish a changed Temporal guest only if reproduction changes committed bytes, and amend only its canary modelHash if required. Heavy JVM/guest proofs queue behind active standalone validation, while source and focused controls proceed independently. Any additional file need requires honest scope amendment and admission before editing.
