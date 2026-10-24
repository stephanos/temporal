# One language for the framework: a Rust process supervisor and an xtask crate replace the Python helpers (model)

## Conversation Evidence

> user (2026-10-24): "why is there backends/tla and backends/tlc ?" then, on moving process supervision out of the emitter: "yes"
> user (2026-10-24): "can w delete all __pycache__ folders?"
> user (2026-10-24): "why do we have python scripts at all?"
> user (2026-10-24): "yes file"

## Goal & Context

The Rust workspace ships with 1,645 tracked lines of Python: an embedded supervisor (`backends/tla/src/process.py`) that `process.rs` runs through `python3 -c` to own every TLC and Apalache child process, five build helpers under `model/scripts/` from fn-230 (guest compiler-input normalization, lockfile closure digests, guest freshness, Apalache admission, a grep filter), a hash check in the activity compatibility checkpoint, and their `unittest` suites, which the Makefile runs as a gate step. Python 3.11 is a documented prerequisite for building the guest. The supervisor exists because owning a child process properly (subreaper, new session, process-group kill, reaping grandchildren) needs `libc`, and the dependency policy kept it out; the scripts exist because build glue was written outside Cargo.

Umpire is one language for the Model and the framework, with Go only where the host demands it. This spec removes the third language: process supervision becomes a Rust crate, the build helpers become a Cargo `xtask` binary, Python leaves the prerequisites, and fn-244's open item (process supervision does not belong in the TLA+ emitter) is discharged on the way. No behaviour change; goldens and guest bytes identical.

## Architecture & Data Models

- **`umpire-backend-process`** at `model/umpire/backends/process`: `Limits`, `Refusal`, `Scratch`, `java_command`, `pinned_java`, `run_process`, `run_process_cancelled`, `bounded_text` move from `umpire-tla::process`; the supervisor is reimplemented in Rust. On Linux: `prctl(PR_SET_CHILD_SUBREAPER)`, child in a new session (`setsid`), kill by process group, reap descendants through `waitpid(-1)` and `/proc/self/task/<pid>/children` as the Python does today. On macOS: new session, process-group kill, and the `RLIMIT_NPROC` descendant restriction in the child's pre-exec hook, refusing root as today. Other platforms refuse at admission. `libc` 0.2.191, already in the lockfile transitively, is promoted to a direct dependency under R18, the same move fn-211.3 made for `serde`; the unsafe surface is one module with the syscalls above and nothing else. Output capture, size limits, timeouts, cancellation and dump files stay in safe Rust as now.
- **Dependencies.** `umpire-tla`, `umpire-tlc` and `umpire-apalache` depend on `umpire-backend-process`; `umpire-tla` keeps binding, emission, Snowcat and typed value admission only; `umpire-tlc`'s `pub use umpire_tla::{Limits, Refusal, values::*}` splits so `Limits` and `Refusal` come from the process crate. fn-240's rIC3 runner depends on the process crate when revived, never on the emitter.
- **`xtask`** at `model/dev/xtask`, a workspace binary crate run as `cargo run -p xtask -- <command>` (with a `cargo xtask` alias in `.cargo/config.toml`): subcommands `guest-rustc`, `guest-lock`, `guest-freshness`, `fetch-apalache` and `retired-names` port `model/scripts/*.py` one for one, keeping their arguments, exit codes and output bytes so `guest.sh`, `retired-names.sh` and the Makefile change only the command they call. The guest launcher that `guest.sh` writes into the target directory becomes a small shell or Rust shim that execs the xtask binary with the same arguments, so the cache key and compiler-input normalization are unchanged. `verify.py` becomes part of the Rust compatibility test that already calls it; `retired-name-matches.py` becomes a few lines of the `retired-names` subcommand.
- **Tests.** `model/scripts/tests/test_*.py` port to `cargo test -p xtask` and `-p umpire-backend-process`, case for case (wrong version, failed launch, timeout, child cleanup including a grandchild in another process group, symlink escape, concurrent installation, freshness policy). The Makefile's `python3 -m unittest discover` step becomes the Cargo test invocation; the gate stays the same set of checks.
- **Prerequisites.** `GUEST_BUILDS.md`, `model/README.md` and the Makefile preflight stop requiring Python; `__pycache__` can no longer appear in crate sources. The `.gitignore` entry stays harmless.

## API Contracts

- `umpire-backend-process`: the moved items with their current signatures; `Supervisor` internals private. No change to `umpire-tlc` or `umpire-apalache` public APIs beyond the import source of `Limits` and `Refusal`.
- `xtask` subcommands: same argument lists, exit codes and stdout bytes as the scripts they replace; `guest.sh` and `retired-names.sh` call them.
- No protocol, Model, DSL, IR or Go change.

## Edge Cases & Constraints

- Child ownership must survive the parent being killed: a TLC or Apalache JVM left behind after a cancelled gate is the failure the Python supervisor was written for, and the Rust control reproduces it (spawn a child that spawns a grandchild in a new process group, cancel, assert both gone).
- Cache identity: the guest wrapper's cache key is computed over its own bytes today; porting the wrapper changes the key once and must leave the built guest bytes identical, proved by `make umpire-guest` and the freshness check against the committed guest.
- Byte-identical goldens, guest bytes and TLC/Apalache output across the move; the required gate (`UMPIRE_BACKENDS=require`) and the Apalache controls (`UMPIRE_APALACHE=require`) are the proof.
- R18: one promotion (`libc`), no new package in the lockfile; `cargo deny` stays green.
- The xtask binary builds on the stock toolchain for the dev profile only; it is not part of the guest and never on the wasm target.

## Acceptance Criteria

- **R1:** `umpire-backend-process` exists with the moved items and a Rust supervisor; `umpire-tla` contains no process code and no Python; `tla`, `tlc` and `apalache` depend on the process crate; `apalache` does not depend on `tlc`. Errors: an unsupported platform refuses at admission naming it; a cancelled or timed-out run leaves no child or grandchild (control).
- **R2:** `xtask` replaces every script under `model/scripts/` with identical arguments, exit codes and output bytes; `guest.sh`, `retired-names.sh` and the Makefile call it; the guest rebuilt through it is byte-identical to the committed guest and the freshness check passes. Errors: a drifted output byte fails the port's golden test.
- **R3:** Every `unittest` case has a Cargo test counterpart; the Makefile runs no Python; `git ls-files '*.py'` under `model/` is empty. No error surface beyond the gate.
- **R4:** `GUEST_BUILDS.md`, `model/README.md` crate map and the Makefile preflight no longer list Python; `model/DESIGN.md` records the one-language rule and the `libc` promotion under R18; `make model-time` pasted. No error surface beyond the gate.

## Boundaries

- No behaviour change to TLC, Apalache, guest builds or any Model; no new third-party package; no change to the Flow tooling's Python (`tools/flow`, outside `model/`).
- No rewrite of `guest.sh` beyond the commands it calls; Bash stays where it is.
- No Windows support claim; the platform set is Linux and macOS as today.

## Decision Context

2026-10-24, owner: process supervision leaves the TLA+ emitter ("yes" to the follow-up); fn-244 had closed minutes earlier, so the move is filed here instead of folded into fn-244.1. Then, asked why Python exists at all, the owner chose to remove it ("yes file") over keeping the scripts and moving only the supervisor. Rationale recorded: Python was a workaround for the dependency policy (the supervisor) and the absence of a Cargo home for build glue (the scripts); Umpire is one language plus Go where the host demands it. `libc` promotion follows the fn-211.3 precedent for `serde`. Placement: before fn-246.1, since fn-246 and fn-240 build on the process crate; it touches only the back-end support crates, scripts and Makefile, so it does not overlap fn-237 or fn-243 lanes.
