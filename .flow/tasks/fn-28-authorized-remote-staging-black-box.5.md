---
satisfies: [R2, R6]
---
# fn-28-authorized-remote-staging-black-box.5 Admit remote evidence through canonical semantic conformance

## Description
Implement R2/R6 by adding one fixed remote input branch around the unchanged Lean semantic authority.

**Size:** M
**Files:** `model/Temporal/Tool/Conformance/**`, `model/Temporal/Tool/ConformanceTests.lean`, `tools/umpire/conformance/**`, `tools/umpire/conformance/testdata/**`
**Touches:** [model/Temporal/Tool/Conformance/**, model/Temporal/Tool/ConformanceTests.lean, tools/umpire/conformance/**, tools/umpire/conformance/testdata/**]

### Approach
- Extend closed runtime/evidence profile admission with the exact remote pair while retaining the private request/response protocol, checker identity, child limits, and local/CI bytes.
- Project only admitted public participant/history/control/cleanup facts through Task `.2`'s fixed mapping; keep authority/target/lease/cleanup qualification provenance outside semantic evaluation.
- Produce the ordinary six-member v1 conformance set with byte-identical inputs and complete ExperimentSpec/configuration/run/program/mapping/query/Property/qualified-outcome bindings.
- Add paired local/CI/remote literal fixtures and an independent mutation oracle for missing, ambiguous, conflicting, unsupported, reordered, crossed, stale, internal-only, payload-derived, and response-drift cases.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — canonical checker protocol and status authority
- `.flow/tasks/fn-20-local-execution-semantic-conformance.3.md` — fixed process bridge and strict response binding
- `.flow/tasks/fn-27-hermetic-ci-execution-and-qualification.3.md` — closed multi-profile admission pattern
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.2.md` — exact remote mapping and configuration
- `tools/umpire/regression/projection.go:201-225` — existing strict JSON/trailing-data precedent

### Key context
Target stability and cleanup can downgrade qualification but cannot rewrite a semantic Result. The semantic checker sees only admitted evidence facts and identity bindings.

### Acceptance
- [ ] Only the three compiled profile branches reach the same checker and semantic evaluator.
- [ ] Qualified equivalent facts share semantic outcome identity while all environment/run identities remain distinct.
- [ ] Every insufficient, ambiguous, conflicting, unsupported, crossed, internal, or payload-derived evidence mutation fails closed or yields the exact non-satisfied status.
- [ ] Local/CI protocols, fixtures, commands, and semantic outputs remain unchanged.

## Acceptance
- [ ] R2/R6 remote conformance admission and status preservation are complete.
- [ ] Focused Lean/Go protocol, paired-profile, corruption, and race suites pass.
- [ ] Existing checker comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
