---
satisfies: [R1, R2]
---
# fn-222-gomad-qualify-the-one-box-cell-and.1 Qualify the one-box cell on the current candidate

## Description
**Touches:** tools/gomad/rust/gomad/src/qualification/**, tools/gomad/docs/wasm/QUALIFICATION.md, .flow/artifacts/fn222-*.json and .flow/artifacts/fn222-*.md. Reconcile shared documentation with completed task .2 before publishing the final Rust support list.

After fn-232.10, run the Rust qualification equivalent of the existing mandatory one-box smoke cell `TestActualOneboxQualification` (ID `unchanged-workflow-type-smoke`, unchanged `./tests` selection `^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$`) through every gate in `internal/backends/wasm/QUALIFICATION.md#one-box-cell-gates`. Use fn-220's shared qualification policy with no caller-specific overrides. Run prepare-only first, retaining its module receipt; if it fails, report that result before measured gates. Then run success/canary with the required workload-intent environment, validate both Results and PublicationReceipts, and run the onebox storage/transport inspection cell. Record commands, numeric exits, host, compiler, runtime overlay, helper, module, engine and model identities in a retained `.flow/artifacts` checkpoint. Update the row only when all gates pass on the frozen current candidate; otherwise retain pending and the failing gate. `TestFrontendSystemInfo` is a separate supplementary cell, not a substitute for this mandatory cell. Carry task .2's support-list contract into this Rust lineage, earn both cells' full gates and produce the first accepted Rust list under R2. Earlier Go observations and partial lists do not satisfy those gates.

**Size:** M
**Files:** tools/gomad/docs/wasm/QUALIFICATION.md, .flow/artifacts/fn222-*.md
**Ordering (owner decision 2026-10-20, Rust host rewrite):** a qualification record produced by the Go binary is Go-lineage evidence that the Rust binary refuses, and fn-232 R12 requires a full cycle on the Rust binary. This task runs after fn-232.10 (qualify in Rust) on the Rust binary; fn-222.2 proceeds now on the Go binary. Flow task dependencies cannot cross specs, so this ordering is recorded here.

## Acceptance
- [ ] Every cell gate has a recorded numeric exit and identities in the retained checkpoint
- [ ] QUALIFICATION.md one-box row references the checkpoint, or stays pending with the failing gate recorded
- [ ] No capacity limit raised without a recorded decision
- [ ] Final R2 Rust support list contains both required selections with complete matching-candidate success/canary receipts, named PASS evidence and zero exits for every required gate

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

### Rust gate attempt, 2026-10-22 (conductor note)
Stopped at gate 1: the Rust binary has no prepare-only freeze that retains the module receipt, so no measured gate was run. The row stays pending. Checkpoint: `.flow/artifacts/fn222-rust-onebox-gates-2026-10-22.md`.

Additional ordering: this task also needs four pieces of Rust tooling.
- **Install for the `./tests` namespace** (fn-232.11). Current test tooling hard-codes `./tests/gomadfunctional`.
- **Prepare-only freeze** with the module receipt (gate 1).
- **Onebox storage/transport inspection cell equivalent.**
- **Support-list producer and validator** (this task's own `rust/gomad/src/qualification/**`).

Resume after fn-232.11. The last three items are this task's own work or belong to fn-232.10/.11, whichever owns the verb.
