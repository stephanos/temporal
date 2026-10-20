---
satisfies: [R2]
---
# fn-222-gomad-qualify-the-one-box-cell-and.2 Define the per-test support list and produce the first one

## Description
Document and produce the candidate-bound `gomad.wasm-support-list/v1` through the existing qualification owner. Require registered cells' final success/canary Results and PublicationReceipts, a zero exit for every cell gate, exact canonical argument vectors and authenticated stdout proving the named test actually passed without skip or no-tests suppression. The first accepted list must include both the mandatory smoke selection `./tests` `TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds` and the supplementary `./tests/gomadfunctional` `TestFrontendSystemInfo` cell. Run the latter's preparation and measured modes under `TestActualOneboxFrontendSystemInfoQualification` in its own root, without `GOMAD_WASM_QUALIFY_WORKLOADS`, and validate its Results independently. Retain any failing gate; a smoke-only interim list does not complete this task. Record candidate/module identities, gate commands/exits and retained references per row, explicitly state unlisted tests are unsupported, and store JSON/Markdown unchanged next to checkpoint records with a QUALIFICATION.md reference. Sweep passes alone never add rows. Changed candidate identities must re-earn every row.

**Size:** S
**Files:** tools/gomad/internal/backends/wasm/qualification/**, tools/gomad/internal/backends/wasm/QUALIFICATION.md, .flow/artifacts/fn222-*.md
## Acceptance
- [ ] Support list shape documented in QUALIFICATION.md with per-test, candidate-bound wording
- [ ] First list for the current candidate contains exactly the gate-passing tests
- [ ] A sweep pass alone cannot add a row (documented and enforced by the producer)

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
