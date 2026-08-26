---
satisfies: [R3]
---
# fn-27-hermetic-ci-execution-and-qualification.3 Admit CI evidence through the canonical conformance authority

## Description
Implement R3 by adding one fixed CI profile branch around the unchanged semantic checker.

**Size:** M
**Files:** `model/Temporal/Tool/Conformance/**`, `model/Temporal/Tool/ConformanceTests.lean`, `tools/umpire/conformance/**`
**Touches:** [model/Temporal/Tool/Conformance/**, model/Temporal/Tool/ConformanceTests.lean, tools/umpire/conformance/**]

### Approach

- Replace the local-only preflight equality with a closed two-entry runtime-profile admission whose branches share the exact experiment, observation program, mapping, query, Property set, evidence profile, and source schemas.
- Keep the private request/response wire versions, checker identity/digest, 30-second/32-MiB limits, sibling resolution, and Lean mapping/evaluation code unchanged.
- Ensure the CI four-member set produces the ordinary six-member v1 set with byte-identical inputs and independently validated Result/qualified-outcome bindings.
- Add paired local/CI fixtures proving equal qualified semantic meaning for equivalent facts while configuration, run, artifact, and manifest identities remain distinct.
- Cover wrong-profile, mixed-profile, crossed-response, incomplete partition, and all existing ambiguity/contradiction cases.

### Investigation targets

**Required** (read before coding):
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — checker ownership, protocol, status, and limits
- `.flow/tasks/fn-20-local-execution-semantic-conformance.2.md` — fixed Lean adapter
- `.flow/tasks/fn-20-local-execution-semantic-conformance.3.md` — sibling protocol enforcement
- `.flow/tasks/fn-20-local-execution-semantic-conformance.4.md` — controller and Result construction
- `.flow/tasks/fn-20-local-execution-semantic-conformance.5.md` — mutation matrix

### Acceptance

- [ ] Only the two compiled runtime-profile identities reach the same semantic authority; no arbitrary selector or second evaluator exists.
- [ ] CI input yields a valid ordinary six-member set with complete status/verdict/omission bindings.
- [ ] Equivalent local/CI evidence has the same semantic outcome and qualified-outcome identity under the parent exclusion rules.
- [ ] The local protocol, command bytes/statuses, and corruption behavior remain unchanged.

## Acceptance
- [ ] R3 closed CI admission through the canonical checker is complete.
- [ ] Focused Lean/Go protocol, mutation, paired-profile, and local regression suites pass.
- [ ] Existing semantic comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
