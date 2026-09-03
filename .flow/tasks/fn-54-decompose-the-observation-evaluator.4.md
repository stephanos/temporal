---
satisfies: [R1, R2, R5, R6]
---
# fn-54-decompose-the-observation-evaluator.4 Extract accepted-trace admission and lock the Evaluation facades

## Description
Move the opaque accepted trace, its private constructor, Observation results, provenance reconstruction, and accepted validation together into Admission. Reduce the stable evaluator to child imports, result mapping, and `evaluateEvidence` orchestration, then prove facade and direct-consumer compatibility.

**Size:** M
**Files:** `model/Umpire/Observation/Evaluation/Admission.lean`, `model/Umpire/Observation/Evaluation.lean`, `model/Umpire/Observation/ImportTests.lean`, `model/Umpire/Observation/Tests/EvidenceLink.lean`, `model/Umpire/Observation/Tests/Mutations.lean`, `model/Umpire/Observation/Tests/Disposition.lean`
**Touches:** [model/Umpire/Observation/Evaluation/Admission.lean, model/Umpire/Observation/Evaluation.lean, model/Umpire/Observation/ImportTests.lean, model/Umpire/Observation/Tests/EvidenceLink.lean, model/Umpire/Observation/Tests/Mutations.lean, model/Umpire/Observation/Tests/Disposition.lean]

### Approach
- Move `EvidenceBackedTrace`, its private construction helper, `ObservationResult`, provenance reconstruction, and `validateEvidenceBackedTrace` together into Admission.
- Keep the unchecked carrier in Types and retain negative fixture construction through that carrier only.
- Reduce `Evaluation.lean` to its preserved module documentation, ordered child imports, diagnostic-to-result mapping, and `evaluateEvidence` orchestration.
- Strengthen facade tests for validation, unchecked fixture access, results, entry points, derived instances, and inaccessible accepted record construction/update.
- Add an accepted-envelope mutation matrix for every R5 category with complete expected diagnostics and combined mutations that pin admission precedence and prove no accepted value is constructed.
- Build all direct evaluator consumers without changing their imports.
- Review module docs and architecture text; update public docs only if the stated facade or internal ownership became false.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Evaluation.lean:273-334` — unchecked and accepted carriers plus results
- `model/Umpire/Observation/Evaluation.lean:1610-2124` — accepted provenance validation and orchestration
- `model/Umpire/Observation/Tests/EvidenceLink.lean:9-230` — accepted construction and admission failures
- `model/Umpire/Observation/Tests/Mutations.lean:200-270` — forged accepted-envelope regressions
- `model/Umpire/Observation/ImportTests.lean:13-45` — facade accessibility contract
- `model/Umpire/ImplementationLink/Application.lean:1-120` — direct accepted-trace consumer

**Optional** (reference as needed):
- `model/Umpire/ARCHITECTURE.md:249-285` — documented accepted-trace and Evaluation ownership

### Key context
- The opaque accepted type and private construction helper must remain co-located.
- Do not move consumer imports to child modules or let an admission failure surface as a later-stage status.

## Acceptance
- [ ] R5 is satisfied by co-located opaque accepted construction and complete admission validation with no public or proof-only bypass.
- [ ] Plan identity, bounds, coordinates, link metadata, identity coverage, order/closure support, record support, field retention, digest, expression, and trace-identity mutations retain exact diagnostics and precedence.
- [ ] The mutation matrix covers noncanonical plan identity, individual link-metadata drift, malformed record-support variants, trace-identity-only drift, and competing mutations with complete diagnostics and no accepted value.
- [ ] `evaluateEvidence` returns the same accepted semantic value or Observation non-success, and no failed admission reaches Property, Implementation Link, Run Evaluation, or artifacts.
- [ ] The stable Evaluation and Observation facades expose the same root names, constructors, projections, instances, and entry points; direct consumers retain their imports.
- [ ] Existing accepted traces, Evidence Links, fingerprints, artifact and persisted bytes, comments, trust dependencies, and traversal counts remain unchanged.
- [ ] `cd model && mise exec -- lake build Umpire.Observation.Tests.EvidenceLink Umpire.Observation.Tests.Mutations Umpire.Observation.Tests.Disposition Umpire.Observation.ImportTests` passes.
- [ ] `cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.ImplementationLink.Tests UmpireTests TemporalModelTests` passes.
- [ ] `make umpire-build-model`, `make umpire-check-regression`, `make lint-model`, and `make lint-code GOLANGCI_LINT_FIX=false` pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
