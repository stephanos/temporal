---
satisfies: [R8]
---
# fn-195-modelsuite-a-models-test-file-holds-its.5 DSL completion: the test file in SEMANTICS and the README

## Description
**DSL completion phase (owner, 2026-10-17).** Add a SEMANTICS section for the test file: the five-kind table (examples, claims, explorations; conformance, sampled conformance), the three sections, the `test` builder, `expect:`, spaces, `oneOf` enumerates and `any` searches, Case names, the defaults and refusals. Rewrite §9 Scenarios and Queries to say the Model asks nothing of itself and point at the new section; update §19 Realizations for `Realizes` without `runs` and the binding rule. Update the README's test guidance. Note that lifting the sections and intermediate Contract assertions are the IR phase.

**Size:** S

**Files:** `model/framework/SEMANTICS.md`, `model/framework/README.md`.

### Verification
- Docs match the DSL as landed; SEMANTICS source links point at a sample suite.

## Acceptance
- [ ] SEMANTICS test-file section with the five-kind table; §9 and §19 rewritten
- [ ] README test guidance updated

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
