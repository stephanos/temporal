---
satisfies: [R2, R3, R4, R5, R6, R7]
---
# fn-153-checked-property-examples-pilot.6 Assess the pilot and close its behavior and identity compatibility proof

## Description
Write the evidence-limited pilot assessment, document authoring and check workflows, and join the original-to-final compatibility proof and gates. The conductor owns review, Flow closure and MILESTONES.

**Size:** M
**Files:** `model/README.md`, `model/SEMANTICS.md`, `tools/umpire/README.md`, new `model/docs/property-illustrations-assessment.md`, `.flow/tmp/fn-153/` evidence
**Touches:** [model/README.md, model/SEMANTICS.md, tools/umpire/README.md, model/docs/property-illustrations-assessment.md]

### Approach
- Document the actual typed attachment, classification/error contract, managed document/gate workflow and hypothetical-versus-found distinction. Use the vocabulary/authoring and Claims/Results/Admission reader sections identified below; do not retell unrelated model semantics.
- Assess all three authored/rendered examples. Record state setup size and duplicated detail, misleading edge cases, mutation/drift evidence and the recommendation to expand, revise or stop. Evaluate the concrete output; explicitly record absent developer feedback and any missing evidence. Mechanical checks cannot establish comprehension.
- Compare the final joined source/IR/checker/producer inputs with .1's independently frozen baseline. Require complete transition tables and behavioral check answers, Definition IDs, Behavior Fingerprints and all managed Case bytes unchanged. Explain every allowed illustration-carrier or whole-artifact digest change separately; no normalization may erase predicate, position or Case changes. Test explanation-only edits and empty metadata independence.
- Run required canonical gates once against final relevant inputs under the shared lock. Preserve Go `-json -tags test_dep -p 2 -timeout 30m` output/exit/wall time, the model gate with `MODEL_GATE_ARGS=--skip-go-checks`, Case/fixture checks, model/Go lint and dependency ownership checks. Reuse unchanged passing input-bound receipts only under the milestone rule. Follow its one-hour deferral rule honestly; never report a deferred validation as passed.
- Return concise task-unique summary/evidence linking assessment, baseline/final pins, managed diff, original predicate restoration and all gate results. Independent implementation and whole-spec review remain conductor-owned. No new live generation or coverage credit is claimed.

### Investigation targets
**Required:**
- `model/README.md:18` and `:413` - vocabulary and authoring
- `model/SEMANTICS.md:434` and `:791` - Claims and Admission
- `tools/umpire/README.md:40` - command/check workflow
- `.flow/tmp/fn-153/` - .1's immutable baseline and task-local receipts
- `MILESTONES.md` - canonical commands, one-hour rule and root ownership
**Optional:**
- `model/docs/property-illustrations.md` - all three checked explanations

### Quick commands
```bash
go test -tags test_dep ./tools/umpire/ir -run 'Property|Illustration|Origin|Identity'
go test -tags test_dep ./tools/umpire/check -run 'Property|Illustration'
go test -tags test_dep ./tools/umpire/cmd/umpire-illustrations
make lint-model
```

## Acceptance
- [ ] Assessment covers all three authoring examples and rendered explanations with measured setup/duplication, misleading cases, drift evidence and a grounded recommendation; missing developer feedback is explicit.
- [ ] Author and checker docs describe actual APIs, tri-state/error behavior, hypothetical reachability limits and the deterministic document workflow.
- [ ] Complete original-to-final input-bound comparisons prove unchanged tables, behavioral answers, Definition IDs, Behavior Fingerprints and all managed Cases; every allowed carrier/digest delta is separately accounted for.
- [ ] Required model/Go/Case/fixture/lint/dependency gates have retained results; deferred or failing checks grant no pass or live credit and keep their follow-up obligations.
- [ ] Concise handoff supplies the conductor the evidence for all R-IDs, independent review and closure without claiming developer comprehension or executing new live Cases.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
