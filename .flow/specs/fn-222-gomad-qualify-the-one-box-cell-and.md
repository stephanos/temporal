# Gomad: qualify the one-box cell and define the per-test support claim

## Conversation Evidence

> user: "our goal is definitely to run arbitrary functional tests with gomad (wasm)."
> user: "yes file these specs"

## Goal & Context

`internal/backends/wasm/QUALIFICATION.md` lists four workload cells. The one-box smoke cell still needs unchanged source preparation, storage/transport artifact qualification and repeated measured pass/replay on the current candidate. The matrix supplies neither an arbitrary Temporal support claim nor a cross-host guarantee.

The existing mandatory one-box cell is `TestActualOneboxQualification`, ID `unchanged-workflow-type-smoke`: unchanged `./tests` selection `^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$` plus its invalid-regexp canary. It is not `TestFrontendSystemInfo`. The latter is a separate one-box family cell, `TestActualOneboxFrontendSystemInfoQualification`, ID `frontend-system-info`, running unchanged `./tests/gomadfunctional`. This spec requires both selections for the first support list; the frontend cell remains supplementary to the mandatory corpus, not optional for this spec's acceptance.

## Architecture & Data Models

**One-box qualification.** Task .1 runs on the Rust host after fn-232.10, using the preserved cell/configuration semantics below. The existing Go owner/test names are semantic references for that Rust port, not instructions to rerun the whole Go-host ladder. Use the existing qualification owner (`internal/evidence/qualification`, wrapped by `internal/backends/wasm/qualification`) and the exact cooperative guest configuration derived through fn-220's shared `wasi.TemporalConfig` qualification policy. Build with `backend.TemporalGuestBuildTags()`. Retain numeric exits, host identity, helper, compiler, runtime overlay, module, engine and model identities for the frozen candidate.

The commands and environment are defined in `QUALIFICATION.md#one-box-cell-gates`. Run prepare-only `TestActualOneboxQualification` first, then its success and canary modes. Preparation excludes `GOMAD_WASM_QUALIFY_WORKLOADS`; measured smoke modes require it. Preserve the preparation receipt before later modes rewrite it and verify the same module identity in both final Results. Validate each Result and PublicationReceipt, and run the `onebox` selection of `TestTemporalInspectionRetainedFailure` for storage/transport inspection. Run the frontend cell's own preparation, success and canary modes in its own root without `GOMAD_WASM_QUALIFY_WORKLOADS`; it shares the inspection gate but its Results are independently validated.

**Per-test support list.** The additive `gomad.wasm-support-list/v1` document is produced by `BuildSupportList` through the existing qualification owner. It accepts only registered cells' final success/canary Result and PublicationReceipt pairs from one candidate, plus recorded zero exits for every gate. Authenticate retained success stdout and require an actual named PASS, not a wrapper-only pass, skip or no-tests execution. Require the exact canonical argument vector; flags that suppress execution cannot establish support. The list records candidate identities and one row per supported selection with package, function, module, gates, exits and retained artifact references. Unlisted tests are unsupported on that candidate. A sweep pass alone cannot add a row.

## API Contracts

- The one-box cell row changes from pending to qualified only with a retained checkpoint covering every required gate.
- Support lists are candidate-bound; changed identities start an empty list. Historical lists retain their original meaning. Task .2 may now implement/document the producer and retain Go-candidate guest observations or pending dispositions. Its phase does not claim a first accepted list without both cells' complete evidence. Task .1 owns final Rust qualification and production of the first fully gate-passing Rust list under R2.
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
- **R2:** Task .2 documents/enforces the candidate-bound per-test support-list shape and records truthful Go-candidate dispositions now. After fn-232.10, task .1 produces the first accepted list for the frozen Rust candidate, containing exactly the gate-passing registered selections and at minimum both the smoke selection and `TestFrontendSystemInfo` above. Every row has all gates and authenticated actual test execution. A sweep pass, unsupported registry cell, suppressed/no-tests execution or claim covering unlisted tests is a finding.

## Decision Context

Owner ordering amendment (2026-10-20): task .2 proceeds now for guest knowledge and the support-list contract on the Go candidate; absent or failed gates produce a pending disposition, never qualified rows. Task .1 later owns R1 and the final R2 Rust list after fn-232.10. The old .2-to-.1 dependency is superseded for this preparatory phase; supported force-start may bypass only that obsolete edge after confirming .2 is todo and unclaimed. No .1 completion is inferred.


The owner chose a per-test, candidate-bound support claim, not a package-level or suite-level claim. The 2026-10-20 source correction distinguishes the existing mandatory workflow-type smoke cell from the supplementary frontend cell without dropping the original minimum frontend requirement or weakening mandatory corpus coverage. fn-220's complete shared qualification policy now supplies the configuration; no caller-specific overrides are introduced.

## Early proof point

Task .1 runs unchanged source preparation first. A preparation failure is retained and reported before measured gates are attempted. The frontend preparation is also required before its measured modes.

## Quick commands

```sh
cargo test --manifest-path tools/gomad/rust/Cargo.toml -p gomad qualification
python3 -m unittest discover -s tools/gomad/rust/driver/tests -p 'test_onebox*'
# Exact cell commands and environment: tools/gomad/docs/wasm/QUALIFICATION.md#one-box-cell-gates
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Mandatory one-box cell qualified with retained checkpoint. | fn-222-gomad-qualify-the-one-box-cell-and.1 | — |
| R2 | Candidate-bound support list, including smoke and frontend selections. | fn-222-gomad-qualify-the-one-box-cell-and.2 | — |

## Amendments (2026-10-24, conductor)

1. **Owners are Rust.** `internal/evidence/qualification`, `internal/backends/wasm/qualification` and `backend.TemporalGuestBuildTags()` named above were retired with the Go host. The owner is `tools/gomad/rust/gomad/src/qualification/**`; the support-list producer and validator are the `fn222-support` tooling and `rust/driver/onebox_receiving.py`; the guest tags come from the installation's Temporal guest-tag policy in QUALIFICATION.md. The Go names stay as semantic references only.
2. **Runs inside the cycle.** MILESTONES ruling 12: the one-box fn-222 group runs alongside the inspection stage's own onebox inspection inside the 96-stage cycle, not after it. Task .1 closes on that cycle's gate exits and the two-cell support list (`fn222/two-cell-ordered-support-list`).
3. **No freeze; provisional claims.** "Freeze the support-list candidate" in Edge Cases is superseded by the 2026-10-24 opportunistic-merge decision. The list binds to the commit the cycle ran on. Support rows published from that cycle are labeled provisional until fn-242's requalification under the Rust cycle verb; the label is part of the published document, not a side note.
4. **Mandatory corpus unchanged.** Both selections (`unchanged-workflow-type-smoke` and `frontend-system-info`) remain required for the first accepted list. Nothing in these amendments adds or removes a gate.
