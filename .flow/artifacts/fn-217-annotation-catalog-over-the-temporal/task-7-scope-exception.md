# Task 7 scope authorization

The conductor authorized four additional paths on 2026-10-21 so the directory rename covers every active Rust root and documentation link found during investigation.

- `model/temporal/testing/build.rs` discovers and source-lifts the feature fixtures.
- `model/temporal/testing/fixtures.rs` compares the feature fixture goldens.
- `model/temporal/api/tests/build_inputs.rs` verifies the catalog build inputs.
- `model/design/second-opinion.md` links the activity Model and its report.

These changes only replace the moved directory's path. Historical Flow records, `.plans` architecture and migration records, and archived Scala paths and provenance in existing Go tests and JSON artifacts retain their original spelling. The active Rust reference check excludes those historical records explicitly.

`MODEL_FEATURE_ROOT` resolves to `model/archived/temporal/features` because `MODEL_ROOT` is `model/archived`. Its lint/lift targets and `temporal.features.` package prefix belong to the frozen Scala tree and remain unchanged.
