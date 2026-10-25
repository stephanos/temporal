# Task 5 source lane

Guest callers use native xtask for compiler cache hashing, atomic executable launcher creation, Cargo artifact selection, build publication and read-only freshness. The shared `model/scripts/xtask.sh` prints its absolute native binary after an incremental host build. Shell entrypoints disable mise installation before bootstrap and preserve caller-relative output paths. Rust admits pins, installed-tool metadata, executable readiness, rustup, rust-src and wasm std on every freshness call.

The source lane began at `2c3c308ee27d2920063dcd29777c5ef084484381` in `work/fn2475-port`. No Flow CLI ran because the conductor owns shared lifecycle state and review. The pre-edit `cargo test -p xtask --locked` baseline passed 25 identity controls. The task's new integration target did not yet exist. Canonical guest build/check and complete Model validation belong to the joined conductor proof.

## Controls and migration

| Retired source responsibility | Native control |
| --- | --- |
| test_guest.py custom library artifact, build/fresh/stale/missing output, compiler and metadata failures | `guest_pipeline::guest_build_check_and_artifact_refusals`; native JSON refuses zero, multiple and malformed candidates, check preserves output, compiler status 37 and native status 1 checked |
| test_guest.py wrapper/cache changes, tool and checkout whitespace, custom cache | `shell_caller_relative_paths_and_changed_inputs_select_new_cache`; native binary and lockfile hashes select cache; lock comment control changes cache |
| test_guest.py macOS launcher loader environment | Native executable copy launcher invoked by fake Cargo with a loader marker; inherited raw loader bytes also covered by `guest_identity::compiler::previous_wrapper_and_loader_environment_keep_raw_os_bytes` |
| test_guest_freshness.py cold and cached admission, local/CI policy | `cold_shell_bootstrap_and_cached_native_validation`, `cold_bootstrap_failures_never_build_or_publish`; bootstrap/compiler failures return 1, missing runtime Go fails with CI absent, empty and nonempty |
| test_guest_freshness.py pins, installed metadata and probe errors, missing executables, Rust sources and wasm target | `native_prerequisite_failures_stop_before_guest_build`, `rust_toolchain_metadata_and_rustup_failures_refuse`; these go through the actual cold shell bootstrap |
| test_guest_freshness.py real Make stale propagation | `actual_make_freshness_recipe_propagates_stale_failure` extracts the current recipe verbatim into a minimal Make target; stale remains visible and asset stays unchanged |
| test_guest_freshness.py missing go:embed file | `absent_embedded_guest_fails_go_compilation` uses actual pinned Go, an isolated module and task-owned GOCACHE |
| guest-lock.py and guest-rustc.py controls | Completed task 2's 25 `guest_identity` controls, including lock graph/ambiguity/malformed input, native macro collisions, real compiler sidecars, actual jobserver descriptor inheritance, prior wrapper and raw OS arguments, status/signal/launch failures and sidecar cleanup |

The old Python executable whitespace restriction, full Python TOML/JSON quirks, exact diagnostic text and byte parity, Python identity records and local/CI missing-tool skip matrix are retired under the owner amendment. Native schema 2 and current-input reproducibility replace those compatibility checks. No scientific golden, frozen original or application semantics changed.

## Commands and observed statuses

All Cargo commands ran in this lane's `model/` with `CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2475-target` and a 600-second foreground timeout. Every started process was joined.

- Baseline `mise exec -- cargo test -p xtask --locked` exited 0, 25 identity tests passed. See `baseline.log`.
- New-target red `mise exec -- cargo test -p xtask --test guest_pipeline --locked` exited 101 before implementation. Native guest commands and the shell freshness entrypoint were missing. See `pipeline-red.log`.
- Final `mise exec -- cargo test -p xtask --locked` exited 0, 25 identity and 8 pipeline controls passed. See `verified.log`.
- `mise exec -- cargo clippy -p xtask --all-targets --locked -- -D warnings` exited 0. See `clippy.log`.
- `mise exec -- cargo fmt -p xtask --check`, `bash -n model/scripts/guest.sh model/scripts/guest-freshness.sh model/scripts/xtask.sh model/scripts/tests/build-guests.sh` and `git diff --check` exited 0.

The host injects a Bash startup file that changes child cwd. The fixture removes `BASH_ENV` from its child environment so caller-relative path checks observe the requested working directory. The initial control exposed this test-host behavior; the final control checks the intended caller path.

## Integration and deferred evidence

The conductor must merge dispatcher, library and Make edits from tasks 5, 6 and 7, retaining every command and dependency. Task 7 can resolve `helper=$(/bin/bash "$workspace/scripts/xtask.sh")` once before its pipelines. Task 6 can resolve the same helper before its installer invocation. The seam prints `<normalized CARGO_TARGET_DIR>/xtask-bootstrap/debug/xtask` and builds before returning; callers execute that binary directly.

`joined-proof.md` gives the exact final publication/reproducibility/freshness sequence. The canonical Temporal guest remains unchanged at SHA-256 `eaf6efd637e0e33b1a9f1e02b24cd8595c1f73706b5695fcde86151e884515f0`. This is a historical predecessor identity, not a byte acceptance oracle. The lane does not claim fresh committed bytes, two-clean-build agreement, final Go parity, Darwin execution, joined dependency accounting or full Model validation. The conductor runs scoped Temporal Go parity after publication; final task 3 owns the single required full gate and its complete host suites.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)
