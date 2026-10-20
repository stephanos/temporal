# Gomad: qualify the one-box cell and define the per-test support claim

## Conversation Evidence

> user: "our goal is definitely to run arbitrary functional tests with gomad (wasm)."
> user: "yes file these specs"

## Goal & Context

`internal/backends/wasm/QUALIFICATION.md` lists four workload cells. The one-box smoke cell still needs unchanged source preparation, storage/transport artifact qualification and repeated measured pass/replay on the current candidate. The matrix supplies neither an arbitrary Temporal support claim nor a cross-host guarantee.

The existing mandatory one-box cell is `TestActualOneboxQualification`, ID `unchanged-workflow-type-smoke`: unchanged `./tests` selection `^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$` plus its invalid-regexp canary. It is not `TestFrontendSystemInfo`. The latter is a separate one-box family cell, `TestActualOneboxFrontendSystemInfoQualification`, ID `frontend-system-info`, running unchanged `./tests/gomadfunctional`. This spec requires both selections for the first support list; the frontend cell remains supplementary to the mandatory corpus, not optional for this spec's acceptance.

## Architecture & Data Models

**One-box qualification.** Use the existing qualification owner (`internal/evidence/qualification`, wrapped by `internal/backends/wasm/qualification`) and the exact cooperative guest configuration derived through fn-220's shared `wasi.TemporalConfig` qualification policy. Build with `backend.TemporalGuestBuildTags()`. Retain numeric exits, host identity, helper, compiler, runtime overlay, module, engine and model identities for the frozen candidate.

The commands and environment are defined in `QUALIFICATION.md#one-box-cell-gates`. Run prepare-only `TestActualOneboxQualification` first, then its success and canary modes. Preparation excludes `GOMAD_WASM_QUALIFY_WORKLOADS`; measured smoke modes require it. Preserve the preparation receipt before later modes rewrite it and verify the same module identity in both final Results. Validate each Result and PublicationReceipt, and run the `onebox` selection of `TestTemporalInspectionRetainedFailure` for storage/transport inspection. Run the frontend cell's own preparation, success and canary modes in its own root without `GOMAD_WASM_QUALIFY_WORKLOADS`; it shares the inspection gate but its Results are independently validated.

**Per-test support list.** The additive `gomad.wasm-support-list/v1` document is produced by `BuildSupportList` through the existing qualification owner. It accepts only registered cells' final success/canary Result and PublicationReceipt pairs from one candidate, plus recorded zero exits for every gate. Authenticate retained success stdout and require an actual named PASS, not a wrapper-only pass, skip or no-tests execution. Require the exact canonical argument vector; flags that suppress execution cannot establish support. The list records candidate identities and one row per supported selection with package, function, module, gates, exits and retained artifact references. Unlisted tests are unsupported on that candidate. A sweep pass alone cannot add a row.

## API Contracts

- The one-box cell row changes from pending to qualified only with a retained checkpoint covering every required gate.
- Support lists are candidate-bound; changed identities start an empty list. Historical lists retain their original meaning.
- Registered smoke and frontend cells keep distinct qualification roots. A mixed mandatory-corpus root is not a support-list input.
- Qualification settings, capacity budgets and canonical evidence formats remain unchanged.

## Edge Cases & Constraints

- A missing or nonzero gate leaves the relevant row pending, with the failing command and exit retained. No partial qualification claim.
- Raising a capacity or measured-cost limit requires a recorded decision; this correction grants none.
- The first accepted list must contain both `./tests` `TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds` and `./tests/gomadfunctional` `TestFrontendSystemInfo`. A smoke-only interim list may be retained as partial evidence but does not complete R2.
- Freeze the support-list candidate after identity-changing goal-path source work is integrated. Earlier qualification may satisfy its own frozen candidate's gate, but cannot qualify later helper/model/module identities.
- No native execution, new guest capability, slim build-tag change, CI run, PR or push.

## Acceptance Criteria

- **R1:** The mandatory one-box smoke cell is qualified on the current candidate: every gate named in QUALIFICATION.md has a numeric exit, identities and host in a retained checkpoint under `.flow/artifacts`, and the cell row references it. Missing gates or exits leave the row pending.
- **R2:** The candidate-bound per-test support-list shape is documented in QUALIFICATION.md and the first list exists for the current candidate, containing exactly the gate-passing registered selections and at minimum both the smoke selection and `TestFrontendSystemInfo` above. Every row has all gates and authenticated actual test execution. A sweep pass, unsupported registry cell, suppressed/no-tests execution or claim covering unlisted tests is a finding.

## Decision Context

The owner chose a per-test, candidate-bound support claim, not a package-level or suite-level claim. The 2026-10-20 source correction distinguishes the existing mandatory workflow-type smoke cell from the supplementary frontend cell without dropping the original minimum frontend requirement or weakening mandatory corpus coverage. fn-220's complete shared qualification policy now supplies the configuration; no caller-specific overrides are introduced.

## Early proof point

Task .1 runs unchanged source preparation first. A preparation failure is retained and reported before measured gates are attempted. The frontend preparation is also required before its measured modes.

## Quick commands

```sh
cd tools/gomad && go test -tags test_dep ./internal/backends/wasm/qualification ./internal/evidence/qualification
# Exact cell commands and environment: internal/backends/wasm/QUALIFICATION.md#one-box-cell-gates
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Mandatory one-box cell qualified with retained checkpoint. | fn-222-gomad-qualify-the-one-box-cell-and.1 | — |
| R2 | Candidate-bound support list, including smoke and frontend selections. | fn-222-gomad-qualify-the-one-box-cell-and.2 | — |
