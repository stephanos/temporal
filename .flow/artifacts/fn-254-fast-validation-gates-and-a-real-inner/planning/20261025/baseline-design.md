# fn254.1 measurement handoff

Read-only investigation completed 2026-10-25. No builds, tests, profiling, source/Flow metadata changes or commits were performed. This scratch handoff is the only authorized write.

## Ownership and fixed measurement conditions

The conductor admits the quiet measurement window, owns the root target lease and freezes the source boundary. The task1 worker owns `model/design/gate-speed.md` and its task-owned probe scratch. Other source owners must reach a safe boundary before measurement. No new tool installation or ownership of other caches is implied.

**Use only the conductor's explicit root cache:** `/home/agent/.cache/codex-build/umpire-milestones-20261024/fn24612`. The inherited `/home/agent/.cache/codex-build/cargo-target` belongs to shared/other ownership and must not be used, modified or cleaned for this work. My initial discovery reported that inherited value; it was not a target selection and is superseded here.

Every admitted measurement command needs `login:false`, `unset BASH_ENV`, and an explicit `cd /Users/stephan/Workspace/skunkworks/umpire/temporal` (or its `model` directory). Preserve these conductor conditions explicitly:

```sh
export CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-milestones-20261024/fn24612
export GOCACHE="$CARGO_TARGET_DIR/go"
export CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0
export CARGO_PROFILE_DEV_INCREMENTAL=0 CARGO_PROFILE_TEST_INCREMENTAL=0
export CARGO_INCREMENTAL=0 CARGO_BUILD_JOBS=2
export GOMAXPROCS=2 GOFLAGS=-p=2 CGO_ENABLED=0
```

Record the actual source/tree, dirty source, host cores/memory/free disk, tool versions, target, profile flags, feature set and backend modes. Record competing process identities before/after and periodically during measurements. An admission statement or initial empty process list alone does not prove the complete interval stayed quiet.

## Reuse existing evidence before scheduling work

`task8/joined-program-correction/required-validation-attempt1.json` under the fn246 artifact root records source `f85d9399c48c09cc2dc8db08ee3cbe1fae071b9d`, warm root target, **415.12 s**, exit **2**. It fails inside workspace tests at `temporal/realization/tests/program.rs:22`, test `configured_reset_issues_a_deterministic_program_and_unconfigured_reset_does_not`. Twelve successful stage timers are available: temporal-guest-freshness 39.864; retired-names 0.881; timing-controls 0.335; capability-controls 0.299; source-boundary 9.263; protobuf-controls 0.241; protobuf-freshness 0.170; kettle-protobuf-freshness 0.090; temporal-protobuf-freshness 0.055; fmt 0.797; clippy 28.688; docs 39.702 seconds. There is no completed tests-stage timer, later stages, inner benchmark or quiet-host proof. The failed total is never a passing full gate or quiet baseline.

**Attempt2 was running at the last read.** `.flow/tmp/milestones-resume-20261024/fn246-joined-required-validation-attempt2.json` records source `8dab804688abb1c2d07a208fefd79799a641d06e`, model tree `0ccc7c4c5a0f15b7bbe3c32e4c18734b3a1404a6`, start 2026-10-25T18:55:05.840159+00:00, PID 3718270 and warm root target after the 116-test Temporal suite with no other build owners admitted. Its adjacent log identifies TLC=require, Apalache=skip. Inspect its eventual terminal receipt before admitting another required gate. Reuse any equivalent successful stage/full-gate measurements with their exact source, environment, warm state and observed interference. Missing conditions remain unknown. Do not repeat a broad gate solely to duplicate equivalent evidence.

fn247 task3 `validation.json` is historical evidence with Apalache=require, gate 2081.715 s and outer 2092.924 s, exit 0. Its source and mode differ from the routine baseline.

## Least-work collection plan

1. At the admitted source boundary, collect binaries once with `mise exec -- cargo test --workspace --locked --no-run --message-format=json --timings` from `model`, using the root overrides above. Retain compiler-artifact package_id, complete target, executable, features and fresh flag, and copy the timing HTML immediately. Stable Cargo timings give compiler-unit total and some codegen subdivisions. Separate linker time remains unknown. A fresh artifact records reuse; never divide aggregate build time into invented per-binary compile costs or force a clean target merely to populate cells.
2. Collect the missing canonical stage evidence only. Default command is `UMPIRE_BACKENDS=skip UMPIRE_APALACHE=skip UMPIRE_TIME_STEPS=1 make model-check`. Required command is `UMPIRE_BACKENDS=require UMPIRE_APALACHE=skip make model-validate`, only if attempt2 cannot supply equivalent measurements. Use monotonic outer timing and preserve failures. `model-validate` includes the existing checker-source mtime touch and benchmark after the gate; report gate, benchmark and outer total separately.
3. A scratch Cargo target runner can time actual executable start/exit in the canonical run without recompilation. Local Cargo supports `CARGO_TARGET_AARCH64_UNKNOWN_LINUX_GNU_RUNNER`; resolve the host triple afresh. Preserve harness concurrency, cwd, arguments, test status and backend mode. Identify the separate source-boundary stage, workspace tests and checker benchmark. Observer overhead makes the observed timing an instrumented measurement and must be declared.
4. Discover every binary with `--list --format=pretty`, then `--list --ignored --format=pretty`. Store package, target kind/name/src_path, full test name, ignored status, features, backend mode and binary hash. Inventory doctests separately with `cargo test --workspace --locked --doc -- --list --format=pretty` and the corresponding `--ignored` command, preserving package boundaries and exact source/line names. These commands have their own elapsed cost. Inventory UI cases from actual trybuild registrations/glob expansion and expected diagnostic hashes. A source file search found 128 UI/admission-ui Rust files; the spec's historical 121 is not a current measured inventory.

## Local harness support and per-test limits

Pinned rustc is 1.99.1. Existing binary help plus the installed `library/test/src/cli.rs` establish that stable `--list`, `--ignored`, `--format=pretty`, `--exact` and `--test-threads` work. `--report-time`, JSON and JUnit require unstable options despite appearing in help. No cargo-nextest, strace or bpftrace command was present; no installation is authorized by this investigation.

Stable per-test timing needs a separately labelled serial observer run. Pass `--format=pretty --test-threads=1` and timestamp partial bytes around `test NAME ... ` and the result. Local `pretty.rs` flushes the start before execution and the result afterward; waiting for newline loses the start. Preserve capture and execute each selected binary once. These are serial observer wall times including harness overhead; canonical parallel per-test times remain unknown. Combine this run with TLC observation instead of making another suite pass.

An alternative scratch measurement can enable libtest's runtime `RUSTC_BOOTSTRAP=1 -Z unstable-options --format=json --report-time`. This must be an explicit instrumented mode. Bootstrap inherited by nested Cargo/trybuild can alter their behavior, so it cannot silently become canonical evidence. Compile/link subdivisions and ignored-test execution times may remain unknown with reasons.

## Actual TLC launch observer

A plain PATH Java wrapper misses production launches. `pinned_java()` asks `mise which java` for an absolute path; `java_command()` removes Java option env variables. For the separately labelled observer run, a scratch mise dispatcher can intercept exactly `mise which java`, return an absolute scratch Java wrapper and delegate all other calls to the saved real mise. The Java wrapper must delegate to the saved real pinned Temurin 27 executable and preserve admission/version checks, process exit status, stdout/stderr and process-group timeout behavior. This changes the launch path and adds overhead; label it as instrumentation and verify the dispatcher actually receives resolution before claiming launch completeness.

Before each launch, atomically record binary/test context, monotonic start, argv, cwd, emitted module bytes/hash, Run.cfg bytes/hash and dump label. Record completion/status/duration afterward. Distinguish Java -version, `tlc2.TLC -h` admission and actual `tlc2.TLC -tool` checks. Interrupted launches retain starts with unknown/incomplete durations.

Map actual module/config bytes to fixture snapshots and source registrations. Module names alone are insufficient because several fixtures use Graph. Preserve ambiguous mappings explicitly. Shared fixture consumers run twice for determinism (`dev/testing/src/runner.rs`), and `check_module` adds progress/negative configurations. Count every actual invocation; one-plus-property-count is structural expectation only. Mean launch duration is not fixed startup cost. Fixed cost stays unknown unless a separately labelled trivial-model probe uses identical flags.

The final report should rank five measured costs, separate historical/current/instrumented conditions, and label expected task savings as estimates. No old receipt establishes new semantics, and no pending or failed run is a pass.
