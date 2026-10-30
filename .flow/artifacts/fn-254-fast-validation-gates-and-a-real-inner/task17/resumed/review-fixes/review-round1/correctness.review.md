I'm requesting changes for one introduced P1: with Apalache opted in, the combined `make model-check` never downloads the pinned Apalache jar. There is also one P2 (`make model-fmt` now waits on the host JVM lease). This session had no shell, so I ran no tests; the findings come from reading the diff and the current source.

## Findings

### 1. Combined gate no longer fetches Apalache when `UMPIRE_APALACHE=require`
- **Severity:** P1
- **Confidence:** 75
- **Classification:** introduced
- **File:Line:** `model/dev/xtask/src/gate.rs:741` (Temporal `run`); `model/umpire/dev/xtask/src/gate.rs:284-291`
- **R-IDs:** [R4, R7]
- **Problem:**
  - Before this change, `make model-check` ran the framework `gate` first. That goes through `Gate::full_inner`, which runs `fetch-apalache` when `UMPIRE_APALACHE=require`.
  - Now `model-check` only enters the Temporal gate. That runs `model:all` and launches each framework stage directly with `gate --scheduled-stage X`, so `full_inner` and its Apalache fetch never run.
  - The framework `tests` stage still runs the Apalache controls. Those call `Apalache::discover`, which goes through `PinnedJar::admit(root, "build/apalache", …)`. That function refuses when the jar is missing and does not download it.
  - **Failing scenario:** on a clean checkout or CI runner without `model/umpire/build/apalache`, `make UMPIRE_APALACHE=require model-check` fails in `framework/tests` with a missing-jar refusal. Previously it fetched the pinned jar and ran.
  - The task requires "explicit optional Apalache behavior" to be kept, and both the Makefile and `model/README.md` advertise this opt-in.
  - `test-model-wiring.sh` only checks the Make call count against a fake child, so this path is untested.
- **Suggestion:** Add an Apalache-fetch step to the shared graph (or to native setup before `scheduler::run`) that `gate:tests` depends on, and that records a receipt marked skipped when Apalache is not required. Add a wiring control for the combined required-Apalache path.

### 2. `cargo xtask fmt` (`make model-fmt`) now blocks on the host JVM lease
- **Severity:** P2
- **Confidence:** 100
- **Classification:** introduced
- **File:Line:** `model/umpire/dev/xtask/src/main.rs:437435-437441` (diff); `model/umpire/dev/xtask/src/gate.rs:400-412`
- **R-IDs:** []
- **Problem:**
  - `fmt` is in the list of commands routed through `supervise`. That calls `scheduler::supervised`, which calls `JvmLease::inherited_or_acquire()`.
  - The lease is one exclusive lock per user under `/tmp`, shared across checkouts and repositories.
  - **Failing scenario:** with a gate or quick run in progress in another terminal, `make model-fmt` (which only runs `cargo fmt --all`) waits silently until that run finishes, which can be 15 minutes or more.
  - The task limits mutual exclusion to JVM-bearing work: framework tests, Temporal tests, the standalone child and quick TLC.
- **Suggestion:** Remove `fmt` from the supervised/leased command list on both the framework and Temporal sides, or route it through the owned-process path without taking the lease.

### Code-smell notes (FYI, not blocking)
- **Middle Man:** `scheduler.rs` stacks one-line wrappers: `execute` → `execute_with_inventory` → `execute_inner` → `execute_captured`; `supervised` / `supervised_clean` → `supervised_inner`; `scheduled` / `skipped` → `scheduled_with`. Several layers only forward arguments.
- **Duplicated Code:** Temporal `gate.rs` turns a relative `CARGO_TARGET_DIR` into an absolute path three separate times (`admit_command`, `mise`, `run`).
- **Long Method:** `work_counts::validate_receipt` / `collect` and `scheduler::collect` each mix parsing, validation and policy in one function.

## Pre-existing issues (not blocking this verdict)
None filed.

Suppressed findings: 2 (one at 25: doctest-summary parsing could misread a doctest's own stdout under `--nocapture`, though gate stages use `--quiet`; one at 50: the shared `$TMPDIR/umpire-command` parent is created with default permissions, which could refuse a second user on a multi-user host).
Classification counts: 2 introduced, 0 pre_existing.

```json
{"suppressed_count":{"50":1,"25":1},"classification_counts":{"introduced":2,"pre_existing":0}}
```

<verdict>NEEDS_WORK</verdict>