---
satisfies: [R7]
---
# fn-253-gomad-fast-validation-gates.6 Adopt cargo-nextest for gomad Rust tests

## Description
R7: adopt cargo-nextest. Owner decision 2026-10-25: adopt without a measurement first.

1. Pin `cargo-nextest` in `tools/gomad/mise.toml`.
2. Add `tools/gomad/rust/.config/nextest.toml` with a `default` profile, a `ci` profile (JUnit output, fail-fast off, slow-test reporting) and test groups for tests that need serial execution (today the cycle forces `RUST_TEST_THREADS=1`). Each grouped test carries its reason as a comment.
3. The three `harness = false` targets (`cycle_process_owner`, `cycle_journal_owner`, `cycle_custody_owner`) must answer nextest's `--list --format terse` and `--exact` protocol. Either implement that in their `main` or convert them to libtest harness targets; don't leave them run separately.
4. The inner loop (fn-253.1 `check-fast`), the pre-merge test entrypoint and CI use `cargo nextest run`, plus `cargo test --doc`, since nextest does not run doctests.
5. The cycle's Rust test stages switch to `cargo nextest run --archive-file` (build once, run each feature mode) only when the fn-242 leaf-argv pin allows it. Record that as a follow-up rather than editing the Python driver.

## Acceptance
- `cargo nextest list --workspace` lists every test that `cargo test --workspace -- --list` lists (a census diff that is empty, ignoring doctests); the `harness = false` targets are included.
- `mise run test` and `check-fast` use nextest and run doctests separately; CI uses the `ci` profile and uploads JUnit.
- Serial-group membership is explicit, with a reason per test.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

### Focused verification — 2026-10-25

Frozen integration candidate `aeb813d6ec9c0ebeca98e12e846a955658b5a09c`:

- Default Cargo/Nextest census: 2,480 tests in 27 suites, 55 ignored; zero differences
  in package/target/test identities, ignored flags and executable paths. The three
  custom harnesses list 24 process cases, three journal cases and five custody cases.
- Focused native execution: 28 passed, four explicitly ignored because they require
  operator inputs. Saved JUnit contains 32 cases, four skipped and zero failures.
- Raw Cargo output, direct binary listings, comparison and execution records:
  `/home/agent/wt/evidence/fn253-nextest/verification-aeb813d6ec/default/`.

The reference-helper census is still pending. Broad workspace test execution and
doctests remain deferred to the integrated spec gate under the owner's validation
cadence. These focused checks do not resume fn-242 or establish qualification.
