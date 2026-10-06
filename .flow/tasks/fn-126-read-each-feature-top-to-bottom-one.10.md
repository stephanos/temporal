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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
