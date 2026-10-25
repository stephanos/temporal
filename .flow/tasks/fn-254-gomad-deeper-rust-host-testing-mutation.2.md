---
satisfies: [R2]
---
# fn-254-gomad-deeper-rust-host-testing-mutation.2 Deep tier entrypoint and cargo-mutants with triaged survivors

## Description
R2, plus the deep-tier entrypoint.

1. Add `mise run deep`, alongside the fn-253.4 entrypoints. It runs the scheduled tier: mutants, fuzzing for a time budget, shuttle/loom, large-count proptest and Miri. Each tool also gets its own `mise run deep:<tool>` entry.
2. Pin cargo-mutants and configure `tools/gomad/rust/.cargo/mutants.toml` with the survey's exclusions, using nextest as the test runner (fn-253.6).
3. Provide `mise run deep:mutants --in-diff <base>` for on-demand checks of changed code.
4. Run the first full workspace pass and triage every survivor: add a catching test (golden or named-invariant) or record the mutant as equivalent with its reason in `.flow/artifacts/fn-254/mutants-triage.md`.

Ordering: after fn-253.4 (entrypoints) and fn-253.6 (nextest); cross-spec order recorded here.

## Acceptance
- `mise run deep` and `mise run deep:mutants [--in-diff <base>]` exist and use nextest.
- The first full run's survivors are all triaged in `mutants-triage.md`; every added test fails against its mutant (recorded).
- A seeded known-bad change (e.g. deleting a refusal check) is reported as a caught mutant, or as a survivor that the triage then fixes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
