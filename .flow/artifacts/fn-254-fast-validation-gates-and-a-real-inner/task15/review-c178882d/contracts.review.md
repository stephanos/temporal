**Scope read:** task spec `.flow/tasks/fn-254-fast-validation-gates-and-a-real-inner.15.md`, the full diff (source portion plus the deleted shell bodies for behavioral comparison), the retained bootstraps, the Temporal gate helpers, framework `main.rs`, `test_run`, `boundary::files`, the process backend's stop/kill paths, and the committed control map and caller scan.

**Overall:** The port is complete and consistent. All 27 scripts map to native commands or Rust controls, callers (root Makefile, both gates, CI workflow, READMEs, TLC/Go diagnostics) reach actual native code, the retired-name scan is in-process Rust regex with the planted-name control, the Temporal protobuf selector composes `proto-go-spec` with the right includes/inputs/subtree, symlink refusal and check-mode cleanup are covered, and the standalone stage no longer needs the bootstrap target. No Model or Go behavior changes. Three small concrete regressions below; none block.

## Findings

- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/dev/xtask/src/gate/timing.rs:11` (struct), `:115` (begin), `:120` (restore)
- **R-IDs**: []
- **Problem**: The inner-loop benchmark now *mutates* the source file (appends `// Inner-loop timing edit.`) and restores it only via normal return or `Drop`. The deleted `time-gate.sh` used `touch`, which never altered content. `make model-time` runs under `supervise` → guardian → `__owned-command time`; on Ctrl-C or any supervisor stop the guardian sends SIGINT/SIGKILL to the child (`backends/process/src/syscall.rs:730`), which has no handler, so the destructor never runs and `umpire/backends/native/src/lib.rs` is left with the appended line in the working tree. README (`model/README.md:179`) promises "restores the source bytes". Secondary: `UMPIRE_INNER_SOURCE` is documented as selecting any source, but a `//` line is only a comment in Rust.
- **Suggestion**: Keep the touch semantics: bump mtime with `File::set_modified(SystemTime::now())` (stable std) instead of rewriting bytes, which removes both the interruption hazard and the comment-syntax assumption. If a content change is genuinely required by the test-run cache, say so in the README and register a stop handler that restores before exit.

- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/xtask/src/build_guests.rs:18`
- **R-IDs**: []
- **Problem**: `CARGO_TARGET_DIR=""` is treated as a relative path and becomes the caller's cwd (`caller.join("")`), then canonicalized and passed to the driver and `cargo run`, so `cargo xtask build-guests` from the framework root writes `wasm32-wasip1/guest/*.wasm`, `panic-native/` and `current-contract-native/` into the source tree. The deleted `tests/build-guests.sh` treated `""` as unset, and the Temporal port explicitly filters empty (`model/dev/xtask/src/build_guests.rs:13`); the checkpoint records that correction on the Temporal side only.
- **Suggestion**: Add the same `.filter(|value| !value.is_empty())` as the Temporal leaf (and in `proto.rs` `Context::new` for consistency).

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/xtask/src/proto.rs:85`
- **R-IDs**: []
- **Problem**: The missing-generator diagnostic says "run cargo xtask proto-tools" for every spec. For the Temporal selector `plugin_root` is the repository root, whose `.bin/protoc-gen-go-v1.36.6` is provisioned by the Temporal Makefile (`model-prerequisites`), not by the framework's `proto-tools` (which installs into `model/umpire/.bin`). The deleted `model/scripts/proto-go.sh` pointed at the Make target.
- **Suggestion**: Make the hint generic ("provision the pinned generator for this plugin root") or let the Temporal bridge map the refusal to its Make target.

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/xtask/src/gate.rs:159`, `model/dev/xtask/src/gate.rs:186`
- **R-IDs**: []
- **Problem**: The new control tests are ordinary non-ignored workspace tests, so every gate runs them twice: once in the `tests` stage (`nextest run --workspace`) and again in the `timing-controls`/`capability-controls`/`protobuf-controls` stages. The capability control alone spawns 10 compiler invocations; the Temporal recipe control runs ~30 `make` children. The shell versions ran once. Not a correctness issue, but it works against this epic's gate-time goal.
- **Suggestion**: Defer to .8 measurement; if it matters, exclude the control identities from the workspace run with a `not test(...)` filter rather than `#[ignore]` (which would change the inventory counts).

## FYI (non-blocking)
- `MIGRATE.md:227,326,327` still name deleted scripts; the file is owner-deferred and the caller scan records it as historical. Acceptable under the acceptance text.
- Task acceptance checkboxes remain unchecked and the done summary is TBD; the spec requires flagging the `grep -E` → Rust regex reversal in the done summary. Process only.
- Nested `supervise` for `timing-controls`/`proto-controls` stage children inherits the JVM lease; the single mise task scheduler and inventory authority are preserved per the committed 17-stage proof.

## Requirements coverage
| R-ID | Status | Evidence |
| --- | --- | --- |
| R7 | met | `gate/standalone.rs` runs `target/debug/xtask fetch-tlc` without the bootstrap target; committed `standalone-9babff6c/summary.json` reports all 17 stages passing, 938/935/3 counts and 909 identities retained. |

Unaddressed R-IDs: []

Suppressed findings: 2 (error-masking `?` after lock/tempdir cleanup, confidence 25)
Classification counts: 4 introduced, 0 pre_existing.

```json
{"suppressed_count":{"25":2},"classification_counts":{"introduced":4,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>