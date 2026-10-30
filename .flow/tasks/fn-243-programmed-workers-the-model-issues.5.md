---
satisfies: [R2, R5]
---
# fn-243-programmed-workers-the-model-issues.5 Restore unheld cancellation admission after the independent cooperative Model

## Description
Complete the deferred unheld-cancellation portion of R2 and its R5 documentation after committed fn-237.4 delivers the live binding over the independently authored cooperative Model. Consume fn-246.15's timeout-retry correction and fn-246.13's explicitly approved pending-cancellation worker-failure/local-timeout Failed/TimedOut correction and renewed native/TLC proofs. Scheduled cancellation and explicit worker acknowledgement remain Canceled. This task consumes that committed contract and authors no further Model policy or fitted server behavior.

**Size:** M.
**Prerequisites:** Native dependency on fn-243.4. Keep this task blocked until fn-237.4 is done and its implementation and Flow closure commits are ancestors of `umpire`; the conductor verifies those facts, fn-237.4's consumption of committed whole fn-246 (including .13's approved correction and renewed proof) and whole fn-247, then resets the block before admission. The CLI cannot encode cross-spec task dependencies.
**Files:** Provisional post-fn-246 owners are `model/temporal/feature/activity/realization.rs`, relevant shared lens/registry/projection modules, realization proposal/history tests, delivered worker/ActivityEvidence consumers, live CLI/reconciliation tests, `Makefile` required-PASS checks and existing worker design/support documentation. Reanchor these against committed fn-237.4 before implementation; `temporal.rs` and `phase.rs` are historical investigation anchors only.
**Touches:** [model/temporal/feature/activity/realization.rs, model/temporal/shared/{catalog,lens,probes,support}/**, model/temporal/realization/tests/**, model/temporal/hosts/go/**, common/testing/testpilot/temporal/act/**, tests/testpilot_model*test.go, Makefile, model/design/programmed-workers.md, model/temporal/realization/README.md, model/README.md, model/DESIGN.md, tools/canary/policy/production-canary.json]

Touches are provisional consumed-owner anchors. Replace them with actual post-fn-237.4 owners before admission; source/generated ownership remains serial at the prerequisite boundaries.

### Approach
- Reanchor the refusal in the current proposal owner and consume the committed fn-237.4 binding, independent cooperative native/source contract and actual protocol evidence, including fn-246.15 and the approved fn-246.13 correction and renewed proof. Preserve the original atomic checkpoint and historical chosen-policy evidence.
- Restore unheld open-state cancellation only for the supported binding and World. Retain explicit refusal before effects for unsupported combinations; do not silently replace a Cancel with Read or remap real statuses.
- Reuse the ordinary Runner, executor identity, actual acknowledgments, delivered feature-local projection, whole-prefix witness contract, ActivityEvidence journal/publication controls and server-free replay. Describe supplies phase/outcome/last-failure evidence; correlated actual heartbeat responses and worker terminal RPC acknowledgements establish notification and explicit acknowledgement. Prove actual native-SDK cancellation delivery through its supported transport path. Use deterministic transport witnesses that demonstrate the intended interleavings rather than assuming a seed exercised them.
- Update the actual live harness and `Makefile` required-PASS set to require the restored admitted raw/SDK no-hold cases. Retain before-effect refusal controls and their forged-evidence mutants for unsupported combinations. The old blanket `UnheldCancelScope` refusal cannot earn credit for newly admitted cooperative races.
- Prove successful unheld Cancel while Started, pending cancellation, completion/failure races, worker acknowledgment and repeated cancellation classification under the independent contract. A server disagreement remains a finding, not a reason to rewrite the Model.
- Preserve the original atomic-Model counterexample's semantic finding and source/guest provenance as historical evidence, without requiring current readers to load its old bytes. New supported Runs use their own Model identity and are a separate proof.
- Update the temporary scope documentation, DESIGN R40 final status and fn-228 R7 evidence pointer only after the new proof passes. Keep malformed/cross-identity evidence, wrong-result/failure mutants, unprogrammed refusal and existing supported slices intact.

### Investigation targets
Required before coding:
- `model/temporal/feature/activity/model.rs`, fn-237.3/.4 and fn-246.13/.15 Flow records, and `.flow/artifacts/fn-246-realization-as-a-checked-refinement-a/task13/planning/approved-api-correction.json` for the independently delivered corrected contract and proof provenance.
- Actual post-fn-246 `model/temporal/feature/activity/realization.rs` and shared lens/registry/projection owners for proposal admission and evidence predicates. `temporal.rs` and `phase.rs` retain historical context only.
- `model/temporal/realization/tests/program_history.rs` and `proposal.rs` for no-pending-residue, publication and scope controls.
- `tests/testpilot_model_slice_umpire_test.go`, `tests/testpilot_model_reconcile_test.go`, delivered ActivityEvidence/transport consumers and `Makefile` for actual Run/parity/evidence witnesses, unsupported-scope controls and required-PASS enforcement.
- `.flow/artifacts/fn-243-programmed-workers-the-model-issues/counterexamples/` for the original Run and guest recovery pointer.
- `model/DESIGN.md` and the existing programmed-worker design/support docs for the current admission boundary.

### Verification
Focused native proposal/history tests and actual committed-guest controls must pass before live checks. Run both executor slices and deterministic no-hold race controls with actual journal evidence and native-SDK cancellation delivery, retained same-Describe notification/ACK mutants, strict unsupported-scope checks and server-free replay. Run `make umpire-check-model-slice` and `make umpire-check-model-gaps` with the required-PASS set updated for restored admitted cases; refusal earns no admitted-race credit. Refresh changed guest bytes through the normal build and repin only the canary artifact hash. Run `make model-validate` once for the required full model gate and one timing benchmark, plus affected Go/lint/dependency checks and fresh implementation review. Record actual soft-target misses and source identities.

### Quick commands
```bash
cd model && mise exec -- cargo test -p umpire-temporal --test program_history --test proposal
```

### Boundaries
Preserve original lifecycle/counterexample evidence and the committed corrected independent Model policy semantically; current formats/goldens may regenerate. The explicit fn-246.13 correction supersedes only the approved pending-cancellation outcome cells and preserves the historical evidence. This task introduces no additional Model-policy changes. No new program language, response kind or envelope work beyond what fn-237.4 already supplies. No reinterpretation of the original failing Run as passing.

### Ordering note (2026-10-24)

fn-246 (Realization as a checked refinement) is placed before fn-237.4 and rewrites the activity binding this task touches (`feature/activity/realization.rs` replaces `temporal.rs` and `phase.rs`; the entity lens and Reach replace the hand-written proposal correlation and projection). Write this task against that shape: the restored unheld admission is a proposal-policy change plus evidence predicates for the cooperative rows, not an edit to the retired hand projection. Files and Touches above use provisional future owners; reanchor them against committed fn-237.4 before implementation. Consume its delivered ActivityEvidence and current whole-prefix projection contracts.
### Verification adoption

Use `make model-validate` for remaining final validation. It runs one complete gate with TLC required and one inner-loop benchmark. Apalache remains independently opt-in; none of these prior plans required it. Keep all standalone focused controls, actual raw/SDK live slice/gaps/replay and SUT proofs, and preserved original semantic counterexample evidence. Label new gate timing as required-mode and historical default-mode timing as historical; retain actual command provenance. Current focused commands remain available. No TLC batching, caching or reduced case coverage is introduced.

## Acceptance
- [ ] Committed fn-237.4 consumes whole fn-246/fn-247, including fn-246.15 and fn-246.13's approved cancellation correction and renewed native/TLC proofs. That corrected independent contract and live binding justify restored unheld open-state admission; unsupported combinations remain explicitly refused before Cancel RPC.
- [ ] Deterministic raw and native-SDK witnesses retain actual Cancel requests/ACKs, pending state, completion/failure races, correlated heartbeat notification, explicit worker terminal RPC acknowledgement and distinct repeated-request classification in Run-owned ActivityEvidence. SDK cancellation delivery, same-Describe notification/ACK mutants and server-free replay discriminate actual evidence. Judgments follow the corrected independent contract and do not predict worker outcomes or remap code 9.
- [ ] The live harness and Makefile required-PASS set require restored admitted no-hold cases for both executors while retaining explicit unsupported-scope and forged-evidence controls. A blanket open-state refusal supplies no admitted-race proof.
- [ ] Original atomic counterexample retains its scientific meaning/provenance, new supported current-format Runs replay without a server, and all existing held/terminal, scope, malformed-evidence, program-mutant and unprogrammed controls remain intact.
- [ ] Current support docs and final DESIGN R40/fn-228 R7 pointers reflect the committed proof; neither parent closes before it. Required gates, actual timing, guest/program identities and fresh review are recorded.


## Done summary
Blocked:
Waiting for committed fn-237-activity-timeouts-heartbeat-and.4.

The conductor may reset this block only after fn-237.4 is done and its implementation and Flow closure commits are ancestors of `umpire`. Its independent cooperative Model and live binding are required before restoring unheld cancellation admission. Native fn-243.4 remains a separate prerequisite. The CLI cannot encode cross-spec task dependencies. Do not clear this block merely because native readiness succeeds.
## Evidence
- Commits:
- Tests:
- PRs:
