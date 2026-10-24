---
satisfies: [R2, R5]
---
# fn-243-programmed-workers-the-model-issues.5 Restore unheld cancellation admission after the independent cooperative Model

## Description
Complete the deferred unheld-cancellation portion of R2 and its R5 documentation after fn-237.3/.4 deliver the independently specified cooperative Model and binding. This task consumes those semantics; it does not author or fit Model behavior to server observations.

**Size:** M.
**Prerequisites:** Native dependency on fn-243.4. Keep this task blocked until fn-237.4 is done and its implementation and Flow closure commits are ancestors of `umpire`; the conductor verifies those facts and resets the block before admission. The CLI cannot encode cross-spec task dependencies.
**Files:** `model/temporal/feature/activity/temporal.rs`, `model/temporal/feature/activity/phase.rs`, realization proposal/history tests, live CLI/reconciliation tests and the existing worker design/support documentation.
**Touches:** [model/temporal/feature/activity/temporal.rs, model/temporal/feature/activity/phase.rs, model/temporal/realization/tests/**, model/temporal/hosts/go/**, tests/testpilot_model*test.go, model/design/programmed-workers.md, model/temporal/realization/README.md, model/README.md, model/DESIGN.md, tools/canary/policy/production-canary.json]

### Approach
- Reanchor the refusal in the current proposal owner and consume the committed native/source cooperative contract and actual protocol evidence from fn-237.3/.4.
- Restore unheld open-state cancellation only for the supported binding and World. Retain explicit refusal before effects for unsupported combinations; do not silently replace a Cancel with Read or remap real statuses.
- Reuse the ordinary Runner, executor identity, actual acknowledgments, Describe projection, compound evidence and replay controls. Use deterministic transport witnesses that demonstrate the intended interleavings rather than assuming a seed exercised them.
- Prove successful unheld Cancel while Started, pending cancellation, completion/failure races, worker acknowledgment and repeated cancellation classification under the independent contract. A server disagreement remains a finding, not a reason to rewrite the Model.
- Preserve the original atomic-Model counterexample and exact guest/hash replay as historical evidence. New supported Runs use their own Model identity and are a separate proof.
- Update the temporary scope documentation, DESIGN R40 final status and fn-228 R7 evidence pointer only after the new proof passes. Keep malformed/cross-identity evidence, wrong-result/failure mutants, unprogrammed refusal and existing supported slices intact.

### Investigation targets
Required before coding:
- `model/temporal/feature/activity/model.rs` and the fn-237.3/.4 Flow records for the independently delivered contract.
- `model/temporal/feature/activity/temporal.rs` and `phase.rs` for proposal admission and server-history projection.
- `model/temporal/realization/tests/program_history.rs` and `proposal.rs` for no-pending-residue, publication and scope controls.
- `tests/testpilot_model_slice_umpire_test.go` and `tests/testpilot_model_reconcile_test.go` for actual Run/parity/evidence witnesses.
- `.flow/artifacts/fn-243-programmed-workers-the-model-issues/counterexamples/` for the original Run and guest recovery pointer.
- `model/DESIGN.md` and the existing programmed-worker design/support docs for the current admission boundary.

### Verification
Focused native proposal/history tests and actual committed-guest controls must pass before live checks. Run both executor slices and deterministic no-hold race controls, retained mutants, strict scope checks and server-free replay. Refresh changed guest bytes through the normal build and repin only the canary artifact hash. Run `make model-validate` once for the required full model gate and one timing benchmark, plus affected Go/lint/dependency checks and fresh implementation review. Record actual soft-target misses and source identities.

### Quick commands
```bash
cd model && mise exec -- cargo test -p umpire-temporal --test program_history --test proposal
```

### Boundaries
No change to frozen original compatibility artifacts or Model policy. No new program language, response kind or envelope work beyond what fn-237.4 already supplies. No reinterpretation of the original failing Run as passing.

### Ordering note (2026-10-24)

fn-246 (Realization as a checked refinement) is placed before fn-237.4 and rewrites the activity binding this task touches (`feature/activity/realization.rs` replaces `temporal.rs` and `phase.rs`; the entity lens and Reach replace the hand-written proposal correlation and projection). Write this task against that shape: the restored unheld admission is a proposal-policy change plus evidence predicates for the cooperative rows, not an edit to the retired hand projection. Reanchor the Files and Touches lists at execution.
### Verification adoption

Use `make model-validate` for remaining final validation. It runs one complete gate with TLC required and one inner-loop benchmark. Apalache remains independently opt-in; none of these prior plans required it. Keep all standalone focused controls, actual raw/SDK live slice/gaps/replay and SUT proofs, and preserved legacy counterexample proof. Label new gate timing as required-mode and historical default-mode timing as historical; retain actual command provenance. Legacy commands remain available. No TLC batching, caching or reduced case coverage is introduced.

## Acceptance
- [ ] Committed fn-237.3/.4 independent semantics and binding justify restored unheld open-state admission; unsupported combinations remain explicitly refused before Cancel RPC.
- [ ] Deterministic raw and SDK witnesses record actual Cancel requests/ACKs, pending state, completion/failure races, worker acknowledgment and repeated-request classification; judgments follow the independent contract and do not predict worker outcomes or remap code 9.
- [ ] Original atomic counterexample/guest replay remains exact, new supported Runs replay without a server, and all existing held/terminal, scope, malformed-evidence, program-mutant and unprogrammed controls remain intact.
- [ ] Current support docs and final DESIGN R40/fn-228 R7 pointers reflect the committed proof; neither parent closes before it. Required gates, actual timing, guest/program identities and fresh review are recorded.


## Done summary
Blocked:
Waiting for committed fn-237-activity-timeouts-heartbeat-and.4.

The conductor may reset this block only after fn-237.4 is done and its implementation and Flow closure commits are ancestors of `umpire`. Its independent cooperative Model and live binding are required before restoring unheld cancellation admission. Native fn-243.4 remains a separate prerequisite. The CLI cannot encode cross-spec task dependencies. Do not clear this block merely because native readiness succeeds.
## Evidence
- Commits:
- Tests:
- PRs:
