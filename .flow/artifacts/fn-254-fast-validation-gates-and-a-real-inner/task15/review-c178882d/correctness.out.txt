Caller scan is clean: the only remaining `.sh` references outside excluded roots are the owner-deferred `MIGRATE.md`, a design doc, and a generic plugin template unrelated to these scripts. I have what I need for the verdict.

## Review of fn-254.15 (a82eb65..c178882)

Scope read: task spec and R7, the full diff of the Temporal and framework xtask sources, the 27 deleted scripts, the untouched code the new paths depend on (`ToolRoot::local`, `supervise`, `supervised_inner`, `JvmLease::inherited`, `admission::configure`, `run_selected_observed_with`, Makefile model recipes), and the committed proof artifacts.

Verified by reading: Temporal stages still run with `model/` as cwd after the timing refactor (`stage_with_inventory` sets it at `model/dev/xtask/src/gate.rs:59`); the bootstrap builds `xtask` and `umpire-guest-driver` side by side so the `with_file_name` driver lookup is sound; `ToolRoot::local` falls back to the compile-time framework root, so dropping the shell wrappers' `UMPIRE_TOOL_ROOT` export does not break guest or fetch commands; the nested gate under the supervised `time` command inherits the JVM lease through the fixed descriptor and env locator; the two retained-name regexes are valid Rust byte-regex syntax with ASCII classes and the planted-name tests cover each pattern; the Temporal protobuf composition passes the three inputs, two includes, repository plugin root and `umpire` check subtree without gRPC; the standalone gate no longer bootstraps a second xtask and uses the already-built `target/debug/xtask fetch-tlc`.

### Findings

- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/dev/xtask/src/gate/timing.rs:115`
- **R-IDs**: []
- **Problem**: The inner-loop benchmark now rewrites a tracked source file (default `umpire/backends/native/src/lib.rs`) by appending `// Inner-loop timing edit.` and relies on `SourceEdit::restore` or `Drop` to put the bytes back. The edit runs inside the `__owned-command time` child under `supervise`. If the developer interrupts `make model-time` or `make model-validate` with Ctrl-C during the inner loop, or the supervisor's owned-process cleanup kills the child with a signal, no destructor runs and the tracked file is left modified in the working tree. The replaced `time-gate.sh` only did `touch`, so this is a new way to dirty the checkout. The existing control `source_edit_restores_original_content_after_failed_child` covers only the failed-child path, not process termination.
- **Suggestion**: Either keep the mtime-only bump until .8 owns the realistic edit, or make restoration survive termination: perform the edit and restore in the supervising parent around `scheduler::supervised` rather than in the owned child, and persist the original bytes to a sidecar in the receipts directory so a later run can detect and restore an interrupted edit.

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/xtask/src/proto.rs:85`
- **R-IDs**: []
- **Problem**: `regular_executable` reports "missing pinned generator ...; run cargo xtask proto-tools" for any plugin root. For the Temporal selector the plugin root is the repository, whose `.bin/protoc-gen-go-v1.36.6` is provisioned by `make model-prerequisites`, not by `proto-tools` (which installs into the framework `.bin`). On a fresh checkout, `cd model && mise exec -- cargo xtask proto-go temporal` now gives a remedy that cannot fix the problem. The deleted Temporal script named the Make target.
- **Suggestion**: Let the Temporal bridge map this refusal to the Make target, or add an optional remedy string to `GoSpec` that the consumer bridge fills in.

**FYI (not affecting the verdict):**
- Every framework `time --step` call now goes through `supervise` (tempdir, lease, owned child) because `time` sits in the supervise list in `model/umpire/dev/xtask/src/main.rs:121`. It works and the standalone proof passed, but it is heavier than the former bash step. Optional: route `--step` around supervision.
- `build_guests::path_controls::isolated_build_child` in the framework is a non-ignored test that returns early without the env var, so it is counted as an executed, passing identity in the inventory. The Temporal twin is `#[ignore]`. Consider aligning so count authority is not padded.
- `warnings()` in `model/dev/xtask/src/gate/timing.rs:49` silently falls back to the default budget when `UMPIRE_GATE_BUDGET` is non-numeric while still printing the raw string as the soft target.
- Process observation: the committed proof is the framework 17-stage standalone plus focused suites. I found no committed log of a real `make model-time` or `make model-validate` on the new source, which is the only end-to-end exercise of supervised `time`, the nested gate's inherited lease and the edit/restore path. Fake-child controls cover the Make wiring; the maintainer decides whether the "full gate runs once at the end" acceptance line is satisfied.
- Code smells, judgement only: `model_controls.rs` is a 524-line fixture module with long methods (`Fixture::new`, `timer_controls`); `go_spec` hand-parses arguments; `(root, driver, target, home)` is a recurring data clump in `build_guests` worth a small struct.

### Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R7 | met | Framework gate stages are native (`model/umpire/dev/xtask/src/gate.rs` `native`/`controls`), root Makefile calls only the native entry points, standalone proof accepted in `task15/standalone-9babff6c/summary.json`, source-boundary control still refuses Temporal names |

Unaddressed R-IDs: []

Suppressed findings: 2
Classification counts: 2 introduced, 0 pre_existing.

```json
{"suppressed_count":{"50":1,"25":1},"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>