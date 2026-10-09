# fn-142-split-modeltemporalshared-into.1 Move shared/ to foundations/ and actors/, Bounds.scala to the root; regenerate; ID-mapping proof; docs
# fn-142-rename-modeltemporalshared-to.1 Move shared/ to foundations/ and Bounds.scala to the root; regenerate; paths-only proof; docs

## Description
**Size:** S
**Touches:** [model/temporal/**, model/irgen/**, model/ir/**, model/cases/**, model/check/**, tools/umpire/**, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, MILESTONES.md]

`git mv` `shared/taskqueue` to `foundations/taskqueue`, `shared/worker` to `actors/worker`, `Client.scala` to `actors/client/Client.scala` and `shared/Bounds.scala` to `Bounds.scala`; update packages and imports, the structure lint's folder classes and their fixtures, and the docs. Regenerate once, as one batch, and prove the IR and Case diff is only the package mapping of Definition IDs (`temporal.shared.taskqueue` → `temporal.foundations.taskqueue`, `temporal.shared.worker` → `temporal.actors.worker`), source paths and positions (R3).
## Acceptance
- [ ] R1 and R2: the folders, packages and imports follow the new layout, and no `temporal.shared` or `shared/` Model path remains outside dated history.
- [ ] R3: one regeneration batch; Definition IDs change exactly by the package mapping; Query answers and receipts are unchanged apart from those renamed IDs; a diff check shows only that mapping, source paths and positions.
- [ ] The structure lint, the module map and the docs name the new layout; `make lint-model`, `make umpire-check-model` and the Go tooling suite pass.
## Done summary
Moved the Temporal task queue, worker, client and shared bounds into the `foundations`, `actors` and root packages, with lifter fixtures, layout enforcement and architecture documentation aligned. Scratch equivalence passed across 11 artifacts, 7 Models, 386 Check receipts and 340 Query answers; the Batch 1 contract assigns production IR/Case regeneration, the full Go suite and integrated batch review to fn-145.4.

Focused verification passed with `make fmt-model`, the IR generator fixture suite, `temporal.IrFilesTest`, the three Go layout tests and the scratch equivalence checker. The spec defines no Quick commands, so the baseline is `none`. `make lint-model` still reports inherited violations in untouched activity regression tests, and the broader artifact-backed Go packages still read the deliberately stale checked-in IR; neither result is recorded as green.

stage: impl-review - ran [2026-10-09T02:32:13Z..2026-10-09T02:42:02Z]
Tier: session (jev-unavailable(no_key))

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8e9e15eab08c99b2741bf3251471d44319da5092, 326988f050482326b092b31ce46be0bbe603be61
- Tests: make fmt-model, UMPIRE_LIFTER_UPDATE=1 timeout 600s mise exec -- scala-cli test model/irgen --suppress-outdated-dependency-warning, mise exec -- scala-cli test --server=false model/project.scala model/umpire model/temporal --test-only temporal.IrFilesTest, go test -count=1 -tags test_dep ./tools/umpire/ir -run '^(TestFoundationsAndActorsLayout|TestRetiredModelPathsStayRetired|TestRetiredModelMentionsAreFound)$', mise exec -- go run .flow/tmp/fn1421-current/equivalence.go .flow/tmp/fn1421-current/before .flow/tmp/fn1421-current/after, git diff --check 59cb017d308a87483081d24d24b10c9733c2e8cf..HEAD
- PRs: