---
satisfies: [R1]
---
# fn-222-gomad-qualify-the-one-box-cell-and.1 Qualify the one-box cell on the current candidate

## Description
Run the existing mandatory one-box smoke cell `TestActualOneboxQualification` (ID `unchanged-workflow-type-smoke`, unchanged `./tests` selection `^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$`) through every gate in `internal/backends/wasm/QUALIFICATION.md#one-box-cell-gates`. Use fn-220's shared qualification policy with no caller-specific overrides. Run prepare-only first, retaining its module receipt; if it fails, report that result before measured gates. Then run success/canary with the required workload-intent environment, validate both Results and PublicationReceipts, and run the onebox storage/transport inspection cell. Record commands, numeric exits, host, compiler, runtime overlay, helper, module, engine and model identities in a retained `.flow/artifacts` checkpoint. Update the row only when all gates pass on the frozen current candidate; otherwise retain pending and the failing gate. `TestFrontendSystemInfo` is a separate supplementary cell required by task .2, not a substitute for this mandatory cell.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/QUALIFICATION.md, .flow/artifacts/fn222-*.md
**Ordering (owner decision 2026-10-20, Rust host rewrite):** a qualification record produced by the Go binary is Go-lineage evidence that the Rust binary refuses, and fn-232 R12 requires a full cycle on the Rust binary. This task runs after fn-232.10 (qualify in Rust) on the Rust binary; fn-222.2 proceeds now on the Go binary. Flow task dependencies cannot cross specs, so this ordering is recorded here.

## Acceptance
- [ ] Every cell gate has a recorded numeric exit and identities in the retained checkpoint
- [ ] QUALIFICATION.md one-box row references the checkpoint, or stays pending with the failing gate recorded
- [ ] No capacity limit raised without a recorded decision

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
