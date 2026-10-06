---
satisfies: [R2]
---
# fn-132-group-the-nexus-and-activity-models-by.5 One NexusProduct that both Nexus forms refine

## Description
**Size:** M
**Touches:** [model/temporal/features/nexus/**, model/irgen/**, model/ir/**, model/cases/**, tools/umpire/**, common/testing/testpilot/**, tools/canary/**, tests/*.go, tests/testcore/testpilot/**, Makefile, model/README.md, .plans/UMPIRE_MODULES.md, MILESTONES.md]

**Required investigation:** task 4 executable binding/outcome decision; both Nexus forms and local levels; current kind-level admission; exact projection helper and move ledger; product-property/refinement regressions in `tools/umpire/check`.
Declare the exact allowed identity/path/fact/catalog/refinement delta before regeneration. Unsupported Reply members remain in the union and are disabled per form; prove a step with no product carrier fails. Move the product-used network/timeout signature closure into the kind core so Product imports no form. Place the standalone Nexus machine and vocabulary in its target System level under the explicit canonical declaration map, preserving its realization's form-root location. Reuse task 4's positive/negative carrier/catalog fixtures and actual history/Describe evidence reads.

Part B. Move `NexusProduct` from `features/nexus/workflow/product/` to `features/nexus/product/Product.scala`, and `Reply`, `Resolution` and the handler actor to `features/nexus/Nexus.scala`, with the entity binding task 4 chose. `Reply` takes both forms' members (`handlerError(retryable)` included); a form whose handler cannot answer a member disables it in its rules.

`NexusSystem` (workflow) refines the shared product as before. The standalone machine gains `object refinement extends Refinement(NexusProduct)`: `unstarted` reads as `scheduled`, and `terminated` and the outcomes follow task 4's decisions. Product Properties (`terminalIsFinal` and the rest) are checked on both forms through their refinements.

No behaviour is added: no cancel in the workflow form, no retries or deadlines in the standalone form.

## Acceptance
- [ ] `NexusProduct`, `Reply`, `Resolution` and the handler's actions are declared once under `features/nexus/`, and no copy remains in either form.
- [ ] Both forms refine `NexusProduct`; the gate checks each product Property on both.
- [ ] Exact table comparison proves every previously enabled source transition, existing Query answer, verdict and realization read unchanged under the finite map. New unavailable Reply classes are disabled in each affected form; source graphs gain no new reachable behavior. Product changes are only the proved required carriers/state/catalog delta.
- [ ] Fingerprints are freshly derived from the declared augmented baseline and generated Case files match the complete expectation, including every changed fingerprint field. Do not drop fingerprints, broadly normalize bytes or accept any unspecified difference. Kind Product/signature depend on neither form; all moved declarations and fact identities are in the ledger.
- [ ] The spec's Verification gates pass.

## Done summary
Both Nexus forms now refine one kind-owned NexusProduct and share its complete signature. Standalone Nexus vocabulary and machine live under system/, with immutable canonical bindings, terminal carriers and disabled unavailable Reply classes; existing enabled behavior and realizations remain preserved.

Tier: session (jev-unavailable(no_key))

Acceptance evidence: [finite extraction contract](/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn132-5/delta-ledger.md), [ownership and independent reconstruction](/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn132-5/resume-ledger.md), [integrated committed-input ledger](/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn132-5/integrated-verification-ledger.md), and [complete proof](/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn132-5/integrated-complete-proof.stdout). All187101 previous source rows/all170 complete Query answers, all63 artifacts, every Case/manifest/pin byte and fingerprint, and all-four complete table/Report/lint packets match independent expectations. Original user mirrors remain in resume-capture/; root preserved overlapping work in stash933b5ec7fe3cdb840cb05f8055b75cec5e373bd0. Both intervening user commits and all unrelated planning/prototypes are preserved.

Current receipts under .flow/tmp/fn132-5: independent expected derivation exit0/wall91; canonical unfiltered GEN exit0/wall413; actual packet derivation exit0/wall39; canonical publication exit0/wall64; complete proof exit0/wall0; model lint exit0/wall20; shared affected7 exit0/wall71 (1829pass events/10skip/0fail); all canary consumers exit0/wall5 (396pass/0fail); shared publication check exit0/wall77; read-only Go lint exit0/wall61/zeroissues. Separate stdout/stderr/exit/wall logs retain actual commands and input ledgers. Same-config negative lint probe proves syntax and semantic rules run despite caught JVM diagnostics. Initial construction failures/refuted historical raw-Run trial remain diagnostic evidence, not green claims. The historical Case/Run bytes match immutable6cd blobs; crossing uses the original encoded record.

stage: impl-review - ran [2026-10-06T18:11:15Z..2026-10-06T18:15:07Z], SHIP. Three actual fresh codex:gpt-6.1-sol:high axes, same-family per AGENTS, zero introduced/pre-existing findings. Receipt /tmp/impl-review-receipt-79fe9d8761f4-fn-132-group-the-nexus-and-activity-models-by.5.json; reports /tmp/fn132-5-integrated.2HmwQ6/.flow/review-fanout/e644a2f775f7418b91117909daf736c2/. Reviewers' attempted Go reruns were sandbox-blocked/inconclusive; no fresh reviewer test pass is claimed.

Review base9ae8d94810bca6f2a0c693b0fc3d9aa325070ce7 through a054fcf7cb3c7bb724c044feae7e247c2ef1f2c8. Original6cd and isolated0e audit receipts remain immutable. Exact task5 review-ledger diff is .flow/tmp/fn132-5/review-ledger.diff. All attributable command handles are terminal. Remaining full PartB/C and live/backend boundary stays task6-owned/deferred, with the inherited matching worker-stop exception retained. No full-Go/live/backend green is claimed. Stop after task5; no next task admitted.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 557f769355a170ea175f715ab95050a8e01d46f0, 40a5216e148a7e9d4460f5213bffcc98752be590, 296f75a1b1d1d094fabe42fa8ce886198972cbc6, a054fcf7cb3c7bb724c044feae7e247c2ef1f2c8
- Tests: mise exec -- make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model (clean committed integrated inputs; integrated-gen.* exit0 wall413), mise exec -- go run .flow/tmp/fn132-5/derive.go (independent expected input SHA161febba...; integrated-expected.* exit0 wall91,187101sourceRows/170completeQueryAnswers), mise exec -- go run .flow/tmp/fn132-5/derive.go actual (integrated-actual.* exit0 wall39), python3 .flow/tmp/fn132-5/verify-integrated.py (all63 complete artifacts/all-four exact canonical packets; integrated-complete-proof.* exit0 wall0), mise exec -- make umpire-gen-fixtures canary-gen-case umpire-check-cases umpire-check-fixtures canary-check-case (integrated-publication.* exit0 wall64), mise exec -- make lint-model (integrated-model-lint.* exit0 wall20; same-config negative lint canary exit96 proves rules execute), mise exec -- go test -json -count=1 -tags test_dep -p 2 -timeout 30m ./tools/umpire/check ./tools/umpire/lower ./tools/umpire/ir ./tools/umpire/lint ./common/testing/testpilot ./tools/canary/casebinding ./tests/testcore/testpilot (shared-affected-go.* exit0 wall71,1829pass/10skip/0fail), mise exec -- go test -json -count=1 -tags test_dep -p 2 -timeout 30m ./tools/canary/... (shared-canary-consumers.* exit0 wall5,396pass/0fail), mise exec -- make umpire-check-cases umpire-check-fixtures canary-check-case (shared-publication.* exit0 wall77), mise exec -- env GOLANGCI_LINT_BASE_REV=21b9964965c8f6e383ee0a751a5fc3cb32d172db GOLANGCI_LINT_FIX=false make lint-code-fast with existing pinned absolute tool paths/-o preventing shared tool rebuild (integrated-go-lint.* exit0 wall61,0issues), git diff --check (exit0); remaining full PartB/C/live boundary deferred to task6 by parent Verification/MILESTONES
- PRs: