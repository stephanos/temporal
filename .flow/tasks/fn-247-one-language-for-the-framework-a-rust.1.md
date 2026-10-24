---
satisfies: [R1]
---
# fn-247-one-language-for-the-framework-a-rust.1 umpire-backend-process: move process support out of umpire-tla and reimplement the supervisor in Rust with libc promoted under R18

## Description
Create `model/umpire/backends/process` (`umpire-backend-process`), move `Limits`, `Refusal`, `Scratch`, `java_command`, `pinned_java`, `run_process`, `run_process_cancelled` and `bounded_text` from `model/umpire/backends/tla/src/process.rs`, and replace the embedded `process.py` with a Rust supervisor: Linux subreaper (`prctl`), new session, process-group kill, descendant reaping via `waitpid(-1)` and `/proc/self/task/<pid>/children`; macOS new session, process-group kill, `RLIMIT_NPROC` pre-exec restriction, root refused; other platforms refuse at admission. Promote `libc` 0.2.191 (already in the lockfile) to a direct dependency; one unsafe module; `cargo deny` green. `umpire-tla`, `umpire-tlc` (`runner.rs`) and `umpire-apalache` (`lib.rs`) depend on the new crate; `umpire-tlc`'s re-export of `Limits`/`Refusal` comes from it; `umpire-tla` has no process code. Port `model/scripts/tests/test_backend_process.py` to Cargo tests and add the orphan-grandchild control (spawn a child that spawns into a new process group, cancel, assert both gone). Gates: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `UMPIRE_APALACHE=require` controls; TLC and Apalache outputs byte-identical. Update the `model/README.md` crate map rows for `tla` and `process` so each names one job.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
