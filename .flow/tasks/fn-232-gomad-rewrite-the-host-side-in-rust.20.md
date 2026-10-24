---
satisfies: [R11, R12]
---

# fn-232-gomad-rewrite-the-host-side-in-rust.20 Restore strict Rust lint across qualification build modes
# Restore strict Rust lint across qualification build modes

## Description
Size: S

**Touches:** `tools/gomad/rust/gomad/src/measurement/module.rs`, `tools/gomad/rust/gomad/src/profile.rs`, `tools/gomad/rust/gomad/src/qualify/support/tests.rs`

Cycle4 on ea26efc078 passed seven static stages including the repaired Go lint gates. Rust lint then failed with actual cargo-clippy exit101: duration_suboptimal_units at measurement/module.rs:134 for Duration::from_secs(1800). Formatting passed. Keep raw failure receipts/logs unchanged at resume-closure-cycle4/cycle/root/stages/rust-lint/static/command-0001.* and producer.stderr. Repair this demonstrated style finding without changing the 1800-second bound or behavior, adding suppressions, weakening warnings, changing features/dependencies or making unrelated edits. Existing actual strict Clippy failure is the regression; no test that mirrors a mechanical time-unit expression is needed.

Read surrounding measurement module callers and existing Duration conventions. Run the exact complete rust-lint recipe matrix from static_stages.recipe for all thirteen declared build modes with pinned Rust1.99.1, locked/offline dependencies and task-owned CARGO_TARGET_DIR. No main/active-target mutation: current cycle/source remain held fixed until its processes are terminal. Notify root if further matrix findings require other files. Preserve all actual exits/logs, compile context and source diff. Root owns independent review, full combined mandatory gate, fresh source-bound role/admission preparation, and the full cycle.

Changing Rust host source bytes invalidates host implementation identities even for this behavior-preserving expression. Genuine current role/admission receipts must be produced as required by the readers; do not rewrite historical receipts or claim old binaries current. The separately investigated default-build byte mismatch is a target-dependent artifact-path issue; root will align the next actual role build and qualification target/context while preserving its byte guard. No driver guard change belongs to this task.

The exact matrix after the duration edit exposed five further strict test-only lint findings: profile.rs attribution_requires_every_import_and_reconciliation has116lines and a complex 23-case function-pointer tuple type, plus missing trailing semicolons in two mutation closures; qualify/support/tests.rs declares local Change after statements. These reported sites are in scope. Preserve every existing mutation/case, assertion, rejection and public test coverage; no suppressions, deleted cases or relaxed checks. Use existing test conventions and meaningful small helpers or coherent grouping where needed for readability. Move type declarations before statements. Keep the existing attribution test root and all 23 negative mutations covered. Further files still require evidence and root scope update before editing.

## Acceptance
- [ ] Strict Clippy finding is fixed with exactly the original 1800-second duration/behavior and no warning suppression or feature/dependency changes.
- [ ] Pinned formatter and all thirteen declared qualification Clippy modes pass; every command/actual exit is retained.
- [ ] Changes remain within the three declared source files unless root explicitly expands scope based on new evidence.
- [ ] Failed cycle4 evidence remains unchanged. Task .13 retains actual current source-bound builds/admissions, complete96stagecycle and all receiving/identity/size/retirement obligations.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
