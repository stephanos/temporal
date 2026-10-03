---
satisfies: [R1, R16, R17, R18, R19, R20]
---
# fn-112-make-the-standalone-activity-scala.10 Close fn112 with refusal coverage, source metrics and full frozen-output gates

Touches: [model/lifter/test/**, model/lifter/testdata/**, model/gate/**, model/temporal/standaloneactivity/**, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, .flow/tmp/fn112-10/**]

## Description
Audit every added construct once, finish docs/metrics and run the complete closure gates against the original baseline.

**Size:** M
**Files:** lifter/gate fixtures and docs; standaloneactivity final sources; .flow/tmp/fn112-10 evidence.

### Approach
- Complete the R16 matrix, accepting compiler refusals only for forms that cannot produce TASTy and requiring located lifter refusals for every reachable misuse.
- Run the original-baseline comparison across all IR/fixtures/manifests/Cases and ordinary Go admission, then the full model gate, lint-model, Umpire Go tests, lint-code-fast and installed export checks once.
- Recount lines/literals with task 1's method, classify every retained literal and reduce the feature to R17/R18 limits. Report both standalone-only and combined standalone-plus-queue metrics. Verify all authored totals and shared queue entity reuse with the precise R1 metadata deltas.
- Update README, SEMANTICS and module ownership only for final syntax and unchanged meaning; preserve fn114/fn118/fn120/fn119 boundaries.

## Acceptance
- [ ] Every R2/R3/R4/R5/R6/R7/R9 construct, including each claim pattern (`once/keeps`, `never/from`, `stays/unless`) and the function-argument binding, has positive and invalid coverage at the correct compiler/lifter layer with located diagnostics.
- [ ] All six original-baseline IRs, positive fixtures, manifests, rejects, IDs/tables/fingerprints/answers and Case bytes pass the strict equivalence harness under only the exact R1 metadata deltas.
- [ ] Full model gate, lint-model, complete tagged Umpire tooling tests, lint-code-fast, Cases/fixtures/canary and installed exports pass with exact commands/versions/times recorded.
- [ ] Standalone activity is at most 1,600 lines and 60 literals; every retained literal belongs to an allowed category and docs match the final surface.
- [ ] All R1-R20 criteria have direct evidence, including total arithmetic/diagnostics and independent shared-queue reuse. Only fn-120's named-choice foundation was brought forward; fn-118 hint-driven waiting, fn-120's strict refusal/tools and fn-122's capabilities remain in their later phases; the done summary says fn-122.1 and fn-114.1 may start.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
