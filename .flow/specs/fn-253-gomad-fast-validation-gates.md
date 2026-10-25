# Gomad: fast validation gates

## Problem

The feedback loop is too slow to steer by.
- **The full cycle has never completed inside the hour.** Cycle 8 (2026-10-24) used 3,554 s, ran 25 of 96 stages, and failed on `rust-diagnostic-test` about 50 minutes in. The stage estimates add up to about 8,300 s.
- **The everyday Rust loop is slow too.** After touching one file, `cargo test --release -p gomad --no-run` takes 3 min 11 s (22 CPU-minutes). `make lint-rust` runs 16 release clippy builds one after another at 12–18 s each. Only `cargo check` (4 s) is fast.

The overhead inside the cycle driver (per-stage custody hashing, single-job stages, repeated compiles) is proposed as an fn-242 amendment: `.flow/artifacts/fn242-gate-speed-amendment-draft-2026-10-25.md`, pending owner approval. This spec covers the rest.

## Goal

Three loops, each fast enough for its job:
1. **Inner loop (seconds):** after an edit, check, lint and the changed crate's tests in under about 30 s of incremental time on this 12-core host.
2. **Pre-merge gate (minutes):** the release build, the full lint feature matrix and the Rust and Go test suites. Their failures surface before any long-running stage starts.
3. **Full cycle (at most one hour):** cheap stages fail first, and unchanged stages can be reused between development cycles. Final qualification (fn-246) stays fresh.

## Constraints

- **Priority (owner, 2026-10-25).** Highest priority outside fn-242; see MILESTONES "Fast validation gates first". fn-248 and fn-249 depend on this spec at spec level. Tasks .1, .4, .5 and .6 start now, in parallel with fn-242; .2 and .3 start right after the fn-242 cutover.

- **Gates get faster, never weaker.** No stage, assertion or bound is dropped to save time. Reuse is keyed on content, not on time or trust.
- **Python driver freeze.** Changes to the cycle land in the Rust cycle verb (fn-242 R2), not in `tools/gomad/rust/driver/`. Tasks .2 and .3 therefore wait for the verb to be cut over (house convention: cross-spec ordering is recorded in the task). This spec has no spec-level dependency on fn-242, so that .1 can start now.
- **Perf measurement stays deferred.** Wall-time targets are measured on the host as it is and recorded. They are not binding until the quiet-host window, like the existing one-hour limit.
- **Testing:** golden and e2e fixtures come first; a unit test must name its invariant.

## Acceptance Criteria

- **R1:** A `check-fast` entrypoint (a mise task under R5) runs `cargo check`, one clippy pass with all features, `cargo fmt --check`, and the unit plus golden tests of the crates changed since a base revision. Incremental wall time after a one-file edit is recorded before and after. Errors: a target that skips a crate with changed sources is a finding.
- **R2:** Development builds optimise dependencies (`[profile.dev.package."*"] opt-level = 3`) and leave our own crates unoptimised, so guests run at usable speed in the inner loop. The pre-merge gate and the cycle keep `--release`. Errors: a test that passes only under one profile is a finding, to be fixed rather than skipped.
- **R3:** The cycle schedules cheap stages first. Each stage has a measured cost class, and no stage of class "long" (over about 300 s) is admitted while a "cheap" stage (under about 60 s) is still pending, unless the cheap stage depends on the long one. Errors: a failure that a cheap stage would have caught, surfacing only after a long stage ran, is a finding.
- **R4:** Development cycles can reuse a stage whose receipt came from a green run and whose full input closure (sources, toolchain, config and upstream artifacts, hashed) is unchanged. Reuse is never available in fn-246 final qualification or in any cycle that claims closure. Errors: a reused stage whose closure misses an input that changes its outcome is a finding, shown by a mutation of that input.

- **R5:** `tools/gomad` is runnable on its own. Every gomad gate (generate, validate, build, lint, test, check-fast, qualify) runs from `tools/gomad` with tools pinned in `tools/gomad/mise.toml`, with no call into `temporal/`'s Makefile, `cmd/tools/lintcode` or `.github/.golangci.yml`. `temporal/` code that gomad exercises (slim-refusal tests, the target tree) is reached only as gomad's subject, through an explicit target-repository path. `temporal/`'s Makefile has no gomad targets. Errors: a gomad gate that fails, or silently behaves differently, when `temporal/`'s Makefile and lint tooling are absent is a finding.
- **R6:** Qualification evidence is stored by content hash and kept under a retention rule. Executables, modules and preparation caches are stored once and shared between runs. Only the newest green run and the newest failed run of each lane keep full payloads; older runs keep receipts, digests and the logs of failing stages. Errors: a reproducible proof that no longer verifies after pruning, or a payload duplicated across runs, is a finding.

- **R7:** Rust tests run under cargo-nextest (pinned in `tools/gomad/mise.toml`, configured in `tools/gomad/rust/.config/nextest.toml`) in the inner loop and the pre-merge gate. Tests that need single-threaded execution are placed in a nextest test group with `max-threads = 1`, each with its reason. The cycle's test stages switch only once the fn-242 leaf-argv pin allows it. Errors: a test that `cargo test` runs but nextest does not list is a finding.

## Decision Context

Owner, 2026-10-25: "one of the most important things is the validation gates; and making sure they are fast. we need a fast feedback loop." Measurements and analysis were taken from cycle 8 evidence and from timing the gomad crates on 2026-10-25. fn-178 once had stage reuse (`run-qualification-fast.sh --reuse`, keyed on git pathspecs). R4 brings it back keyed on content hashes, because a pathspec misses any input it doesn't name.

Owner, 2026-10-25 (same day): "we also want to move the gomad specific makefile targets out of there and into cargo/mise where appropriate inside tools/gomad. they need to work standalone. temporal/ doesn't care/know about them." This became R5. The same day's disk audit found ≈79 GiB of evidence (624k files), ≈34 GiB in 93 worktrees and 36 GiB in `~/.cache/codex-build`, with ≈30 GiB of payloads duplicated across runs and no retention rule; that became R6. Owner decided the same day to adopt cargo-nextest without a measurement first (R7): it builds tests once into a reusable archive, runs them in parallel with serial groups for single-thread tests, and reports slow and hung tests (input for R3).
