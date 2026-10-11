---
satisfies: [R9]
---
# fn-166-mini-activity-a-small-complete-model-to.6 Separate lowered Cases into generated/cases/

## Description
Separate lowered Cases from IR artifacts without changing Model or runtime semantics.

**Size:** S
**Touches:** [model/temporal/features/**/generated/**, model/OutputPaths.scala, model/OutputPaths.test.scala, model/framework/IrFile.test.scala, model/check/*.scala, model/check/test/*.scala, tools/umpire/ir/**/*.go, tools/umpire/cmd/umpire-gen-cases/*.go, tools/umpire/cmd/umpire-repeat/*.go, tools/umpire/lower/*test.go, tools/canary/**/*test.go, tests/**/*test.go, Makefile, model/README.md, .plans/UMPIRE_MODULES.md]

After task 1's lossless colocation, move active `manifest.json` and `<query>.case.json` files into each feature's `generated/cases/`. Keep `ir.json` and `ir.lint.json` directly under `generated/`. Update the existing shared Go and Scala path helpers and their readers, generator publication/check paths, repeat runner, tests and documentation; do not add selectors or duplicate path rules.

Keep transitional historical IR/lint payloads under `generated/legacy/`. Move their associated Case files and split manifests losslessly under `generated/cases/legacy/`, preserving historical filenames, logical IR/Case IDs, query expectations and payload bytes until fn-160 performs its separate merge. Do not touch real Model sources, table pins, equivalence baselines or recorded Runs.

Case publication owns only its Case subtree: scoped generation may not rewrite IR, lint acceptances, legacy payloads or another feature's outputs. Prove preservation with original payload hashes and manifest-row comparison modulo approved file paths, including error cases for missing references and duplicate destination names. Run focused Go/Scala path and generator tests before review. Lint checks and manual acceptance work are set aside for this run by the owner (2026-10-11); preserve the existing lint files and do not change project-wide lint policy.

## Acceptance
- [ ] Each feature's active Case manifest and Case files live under `generated/cases/`; IR and lint companion paths stay unchanged.
- [ ] Historical IR/lint payloads remain in `generated/legacy/`; historical Cases and manifests move byte-preservingly to `generated/cases/legacy/`, with no semantic merge, logical-ID changes or pin/baseline/recorded-Run changes.
- [ ] Every Case reader, generator kind, repeat route and gate resolves the new paths through the existing shared helpers; documentation matches the layout.
- [ ] All original Case payloads remain byte-identical at unique destinations; all manifest query metadata is unchanged except approved file paths, and every Case reference resolves.
- [ ] A scoped generation run changes only the selected Case subtree, not IR/lint/legacy payloads or sibling features; tests cover this and missing-reference/duplicate-name failures.
- [ ] Focused Go/Scala path and generator tests pass; lint checks are explicitly recorded as not run under the owner's instruction, not credited as passes; the task is reviewed and closed through Flow.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
