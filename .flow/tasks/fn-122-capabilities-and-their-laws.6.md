---
satisfies: [R8, R9, R11]
---
# fn-122-capabilities-and-their-laws.6 Document capabilities and laws, count authored versus generated claims and close fn-122

## Description
Docs, the table view's `promises`/`doesNotPromise` column read from the sidecar, the authored-versus-generated counts per Model, the vision's acceptance-test evidence, and the full gates once.

**Size:** S
**Files:** `model/README.md` (capabilities, laws, `except`, `overriding`, how a new entity gets its laws, the sidecar, the core/sugar rule for this surface); the generated table view (the law's `promises`/`doesNotPromise` beside its Properties, from the sidecar); `.plans/UMPIRE_MODULES.md` (ownership of `umpire/laws`, `temporal/laws`, `temporal/nexusoperation`, the sidecars); `.flow/tmp/fn122-6/**`.
**Touches:** [model/README.md, .plans/UMPIRE_MODULES.md, tools/umpire/model/**, .flow/tmp/fn122-6/**]

### Approach
- README: one worked example (the activity's two declarations) and the rules that a law enters the catalog with two instantiating machines and that sugar lives in `Syntax.scala` files.
- Modality rendering (R8): on fn-120.3's per-operation table, render each law from the sidecar as the modality it pins on the cells of its capability's action (`pausedIsNotDispatched` on the `paused` cells of the dispatching action as MUST NOT, `closedIsRejectedUniformly` on the `terminal` cells), and list the cells of a capability's action no law pins; a product law the protocol has not been checked against is marked inherited. Work in `tools/umpire/model/**` beside the `promises`/`doesNotPromise` column, reusing fn-120.3's view code.
- Counts: Properties and Queries authored vs generated, before (fn-112 closing counts) and after, per Model, by one reproducible command stored with its output.
- Evidence for R11 (the vision's #PROTOCOLS acceptance test) collected from tasks 3 and 4: both entities' generated laws and Cases, the unlisted interaction law, the recorded override, the violating fixture's rejection naming law and binding.
- Run the full model gate, `make lint-model`, the Umpire Go tests with `-json` timing per MILESTONES verification instructions, and `make lint-code-fast` once; record commands, results and log paths.

### Quick commands
```bash
make umpire-check-model && make lint-model && make lint-code-fast
go test -count=1 -json -tags test_dep ./tools/umpire/... > .flow/tmp/fn122-6/go-test.json
```
## Acceptance
- [ ] README and module map describe capabilities, laws, waivers, the sidecar, the two-entity rule and the core/sugar file rule; the table view shows each law's `promises`/`doesNotPromise`, renders each law as the modality it pins on the per-operation table's cells, lists unpinned cells of a capability's action and marks inherited product laws (fixture on the activity IR).
- [ ] Authored vs generated Property and Query counts per Model, before and after, are in the done summary with their command.
- [ ] The done summary links the evidence for each clause of the vision's #PROTOCOLS acceptance test (R11).
- [ ] Model gate, lint-model, Umpire Go tests and lint-code-fast pass in full.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
