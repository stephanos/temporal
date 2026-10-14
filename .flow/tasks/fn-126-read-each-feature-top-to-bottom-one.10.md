---
satisfies: [R1, R11, R20]
---
# fn-126-read-each-feature-top-to-bottom-one.10 Let each Product and System file own Phase, State, and Fact

## Description
Amend decision 16 so each level owns its vocabulary. In standaloneactivity and nexuscaller, move ProductPhase, ProductState, and ProductFact out of the root feature file and into product/Product.scala as Phase, State, and Fact; move the corresponding System Phase, SystemState, and SystemFact into system/System.scala as Phase, State, and Fact. Keep Product and System machine names unchanged. Refinements and System zoom-ins must cross level and subject boundaries explicitly through product.* and system.* qualification (or unambiguous local aliases), while the root feature file retains only genuinely shared types and the shared signature. This is an intentional type-identity migration, not a projection-preserving file move.

## Acceptance
- Standalone activity and Nexus caller `product/Product.scala` files each own `product.Phase`, `product.State`, and `product.Fact`; no `ProductPhase`, `ProductState`, or `ProductFact` remains.
- Their `system/System.scala` files each own `system.Phase`, `system.State`, and `system.Fact`; the former root-level System `Phase`, `SystemState`, and `SystemFact` names are retired.
- Refinements, record/close-policy zoom-ins, compositions, realizations, tests, generated IR/Cases, and pinned runs use explicit level qualification or unambiguous aliases; every machine object's inherited `State` continues to mean that machine's own state type.
- The root feature files retain only genuinely shared types and the shared signature. The task queue's shared `QueueView`/`QueueDetail` vocabulary is unchanged because it does not use the Product/System-prefixed pattern and is consumed across its zoom-ins.
- The layout template, structure lint, and README teach that level-owned types live in their level file; negative fixtures reject level-owned Product/System types stranded at the root.
- A machine-readable rename ledger and semantic-equivalence proof account for every intentional fully-qualified type/name change and show no behavior/table/query/claim drift beyond those renames and regenerated source positions.
- Focused red/green tests cover ownership and qualification; `make umpire-check-model`, `make lint-model`, the task-base read-only Go lint, and the relevant Go/model smoke suites pass at the batch boundary.

## Done summary
Standalone activity and Nexus caller now keep Product/System `Phase`, `State`, and `Fact` in their canonical level files, with System-only timer/composition types moved out of the roots. The structure lint and template enforce that ownership while preserving only the named taskqueue shared-vocabulary exception; generated artifacts remain behaviorally identical under the 17-entry identity map.

The machine-readable ledger is `.flow/tmp/fn-126/task10/renames.json`; the final proof is `.flow/tmp/fn-126/task10/review-prove.log` and ends `RESULT: OK` for projection, casegen, lint, laws, IR, 21 Cases, 9 generated fixtures, 2 canary fixtures, and case names. Focused Scala/Go checks, model generation, case checks, formatting, and pinned-run checks passed. Per the user-approved batching policy, the authoritative `make umpire-check-model && make lint-model`, task-base read-only Go lint, full Go/model suite, and smoke checks are deferred to dependent task `fn-126-read-each-feature-top-to-bottom-one.11`, the closing batch boundary.

Baseline: green via `.flow/tmp/handovers/fn-126.8-evidence.json`; all paths after its verified commit and before task base `e1dd0212f58d9e8d9a39e6071797da8390a81cc0` were `.flow/` only.

stage: impl-review - ran (codex:gpt-5.6-sol:xhigh; NEEDS_WORK -> fixes -> SHIP; receipt `/tmp/impl-review-receipt-8f37faba39e2-fn-126-read-each-feature-top-to-bottom-one.10.json`)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8555cb1fdf4ecc7c2aa2f78f7e3257467cf12810, 32c10699833e5f887956f7c4b0209737618bba22, ea4c124dab9ebc5b189ff139a3242337f85611b7, fe64e63ca8f7927309f92bdcfdfd4609bb7a8a22, d96e905560a3f4e2883a600eb604397c4ff4389a, 638eb4eed67e6dd38b9a735338f5c2ff2233ad6c, 0a7ffc19c6c4b9cd9a421872f107a4feecc6e5f9, 6e813bae8c1721599a3612910f262c738be20938
- Tests: BASELINE_REUSED:.flow/tmp/handovers/fn-126.8-evidence.json (only .flow paths changed before task base), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen --test-only 'umpire.irgen.Fixtures' -- '*root-owned*', make model/build/model-scala.jar, mise exec -- scala-cli test --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only temporal.capabilities.CatalogTest, make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model, make umpire-gen-cases umpire-gen-fixtures canary-gen-case, make umpire-rerecord-pinned-runs, bash .flow/tmp/fn-126/tools/prove.sh --renames .flow/tmp/fn-126/task10/renames.json --repo "$PWD" .flow/tmp/fn-126/task10/before "$PWD" .flow/tmp/fn-126/task10/prove, make fmt-model, make umpire-check-cases, mise exec -- go test -count=1 -tags test_dep ./tools/umpire/model -run '^(TestActivityPropertyRowsCatchWhatThePathsMiss|TestNexusProductPropertyOverEveryTraceWithinFour|TestNexusProductPropertyOnAMissingAction|TestTheActivityModelsTotalsAreStatesTimesSlots)$', mise exec -- go test -count=1 -tags test_dep ./tools/umpire/lint -run '^(TestDisabledByDefault|TestTablesAreWrittenByMachineAndClass)$', BATCH_GATE_DEFERRED_TO:fn-126-read-each-feature-top-to-bottom-one.11 (make umpire-check-model && make lint-model; task-base read-only Go lint; full Go/model suite; smoke checks)
- PRs: