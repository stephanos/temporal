---
satisfies: [R1, R10]
---
# fn-52-caller-neutral-grpc-portable-test-plans.1 Reconcile model and portable-plan authority

## Description
Define the normative authority and claim-scope vocabulary required by R1 before introducing the wire format. Preserve every existing rule ID and distinguish the existing Lean-derived Portable Evaluation Contract from caller-authored PortableTestPlan conformance.

**Size:** S
**Files:** `.plans/UMPIRE4_SPEC.md`
**Touches:** [.plans/UMPIRE4_SPEC.md]

### Approach
- Add new stable rule IDs instead of renumbering or repurposing existing rules.
- Keep SEM-10 and AUT-07 authoritative for model-bound Tests and Model Definitions; add explicit plan-local authority and model-provenance rules for the new portable plan.
- Define how external verification obligations limit local and model-bound claims.
- Preserve all existing comments and surrounding terminology.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_SPEC.md:1-18` — governance and stable-rule-ID requirements
- `.plans/UMPIRE4_SPEC.md:65-83` — current model authority and portable interpreter seam
- `.plans/UMPIRE4_SPEC.md:166-179` — current public Lean authoring rules
- `.plans/UMPIRE4_SPEC.md:199-258` — Test Plan, Artifact, and portable contract terms
- `.plans/UMPIRE4_SPEC.md:287-323` — local decision and thin-runtime rules

### Acceptance
- [ ] New terms and rule IDs distinguish Behavior Model authority, plan authority, plan-local conformance, validated model-bound scope, and external obligations.
- [ ] Existing rules remain stable and coherent; non-Lean plans are not presented as Model Definitions or Umpire Properties.
- [ ] Invalid provenance cannot produce model-bound scope or silently downgrade to plan-local.
- [ ] Existing comments remain intact.
- [ ] `make lint-model` and `make umpire-check-regression` pass.

## Acceptance
- [ ] R1 is fully specified with stable rule IDs and no contradictions.
- [ ] Existing comments are preserved.
- [ ] Model lint and regression gates pass.

## Done summary
Defined Portable Plan authority, Plan-local Conformance, Validated Model Provenance, Model-bound Scope, and External Verification Obligations with stable rules SEM-11 through SEM-15. Reconciled admitted Portable Plans with Execution, Run Evaluation, and Local Canary Decision vocabulary; kept external plans outside Behavior Model authority; and made Lean the first model compiler without making it the exclusive plan author. Aligned fn-52 and the roadmap around generated-client disposable-local proof while leaving production canary consumption to fn-29.

stage: impl-review - ran [2026-09-03T17:32:48Z..2026-09-03T17:43:32Z] (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0502b1b120f0cac710c366ca6753f4291ead4015, 5ed70183a16ae65ba464d2b81d33572fe1b099cb
- Tests: make lint-model, make umpire-check-regression, flowctl validate --spec fn-52-caller-neutral-grpc-portable-test-plans --json, make umpire-check-plan-index, stable rule-ID audit: 82 unique IDs, no duplicates, git diff --check
- PRs: