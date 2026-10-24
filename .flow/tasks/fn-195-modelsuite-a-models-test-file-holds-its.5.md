---
satisfies: [R8]
---
# fn-195-modelsuite-a-models-test-file-holds-its.5 DSL completion: the test file in SEMANTICS and the README

## Description
**DSL completion phase (owner, 2026-10-17).** Add a SEMANTICS section for the test file: the five-kind table (examples, claims, explorations; conformance, sampled conformance), the three sections, the `test` builder, `expect:`, spaces, `oneOf` enumerates and `any` searches, Case names, the defaults and refusals. Rewrite §9 Scenarios and Queries to say the Model asks nothing of itself and point at the new section; update §19 Realizations for `Realizes` without `runs` and the binding rule. Update the README's test guidance. Note that lifting the sections and intermediate Contract assertions are the IR phase.

After fn-195.6, show primary machines as `class Model`, including parameterless models, and bind a fresh stable-named instance in each suite. Document that singleton primary Machine declarations are no longer supported; shared constants, actors and namespaces remain objects. Keep the user-facing reference consistent without references to Scala source files.

Consume the committed native waited-class/checked-coverage interface and the expectation convenience/admission validator. Reconcile the interim SEMANTICS and REFERENCE sections with the final ModelSuite integration. Explain unsupported-kit refusal, performed versus awaited exact input coverage, independent nine-field defaults and raw/copied expectation validation at conformance admission. Keep incomplete capability/discovery and held system-kit limitations visible until their actual integration proves them. No Scala source-file references belong in REFERENCE.

**Size:** M, three documentation files and their final cross-guide consistency checks.

**Files:** `model/framework/SEMANTICS.md`, `model/framework/README.md`, `model/framework/REFERENCE.md` (primary-class convention and suite-facing reference).

**Touches:** [model/framework/SEMANTICS.md, model/framework/README.md, model/framework/REFERENCE.md]

### Verification
- Docs match the DSL as landed; SEMANTICS source links point at a sample suite.
## Acceptance
- [ ] SEMANTICS test-file section with the five-kind table; §9 and §19 rewritten
- [ ] README test guidance updated

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: