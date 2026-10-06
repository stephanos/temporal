---
satisfies: [R3]
---
# fn-132-group-the-nexus-and-activity-models-by.6 General activity declarations in activity/Activity.scala

## Description
**Size:** M
**Touches:** [model/temporal/features/activity/**, model/irgen/**, model/ir/**, model/cases/**, tools/umpire/**, common/testing/testpilot/**, tools/canary/**, tests/*.go, tests/testcore/testpilot/**, Makefile, model/README.md, .plans/UMPIRE_MODULES.md, MILESTONES.md]

**Required investigation:** current `activity/standalone` header/product/System/Realization; task 4 binding decision; exact projection helper/type ledger; scoped action/entity typing and lifting tests. Task 2 must precede this extraction; task 5 is logically independent but cannot run concurrently while both regenerate shared artifacts.

Part C. Move into `features/activity/Activity.scala`: `Timeout`, `TimeoutType`, `AttemptResult`, the worker's `poll` and `respond` on an activity (with task 4's entity binding), `timers` and `deadline`. The standalone form reads them from there.

`ActivityProduct`, `ActivitySystem`, `Control`, `client` and the `activity` entity stay in `standalone/`; the worker actions move as R3 requires, with task 4's proved binding. Include System-owned `TimeoutType` in the exact type identity map. Keep `product.{Phase,State,Fact}` and `system.{Phase,State,Fact}` in their levels, and the System realizations under `standalone/system/`.

Independent of task 5.

## Acceptance
- [ ] `features/activity/Activity.scala` declares every listed type, worker action, timer and deadline; no duplicate remains in `standalone/`. Types-only extraction does not discharge R3.
- [ ] A before/after projection differs only in the paths of moved declarations.
- [ ] The spec's Verification gates pass.

## Done summary
`features/activity/Activity.scala` now declares Timeout, TimeoutType (it used to be in the System level), AttemptResult, the `result` input, the worker's unbound `poll`/`respond`, `timers` and `deadline`. The standalone form binds the two worker actions to its own `activity` Entity with immutable aliases, as task 4 proved. It reads everything else from the kind through chained package clauses, the same way the Nexus forms do. Nothing listed is still declared in `standalone/`. Entity, Control, client, Outcome, Product, System and Realizations stay with the form.

Tier: implementer (AGENTS.md model routing)
stage: impl-review - ran [first round, single claude dispatch] - SHIP (claude:high). Receipt `/tmp/impl-review-receipt-8f37faba39e2-fn-132-group-the-nexus-and-activity-models-by.6.json`. The one P3 finding, a comment naming a binding spelling a form cannot write, is fixed in fcb8c74151. That fix is comment-only with an unchanged line count, so it does not affect the IR.
stage: memory - skipped(clean first-pass SHIP)
baseline: green. At base 5240b0383b, `make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model` regenerated the committed IR with no diff (exit 0, 602 s).

### Exact extraction proof
`.flow/tmp/fn-132.6/prove.py <base>` derives the expected output from the base artifacts and compares it field by field with the regenerated tree. Its finite ledger, `extraction-ledger.json`, is:
- 11 qualified identities, old to new;
- each changed source file's line map, taken from its own diff against base;
- the moved declarations' new lines, matched by text in Activity.scala;
- the IR's canonical re-sort of top-level lists.

Result: EXACT PASS over 58 artifacts (17 IR/lint/laws documents, 14 lift expectations, all Case trees and the manifest). Five seeded-mutation controls are refused. Every behavior, property and projection fingerprint is unchanged, and the only Case delta is the Realization wait-source line moving 226 to 227. No golden was recaptured. The Go position pins that moved (hints 7 lines, lint api 56→57, the lower grouping-spike wait map now 227/228→230/231) were cross-checked against the derived line map.

### Gates (logs under `.flow/tmp/fn-132.6/`, `verification-ledger.md`)
- `make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model` 0 (321 s); `make MODEL_GATE_ARGS=--skip-go-checks umpire-check-model` 0 (337 s).
- `make umpire-gen-cases umpire-gen-fixtures canary-gen-case` 0; `make umpire-check-cases umpire-check-fixtures canary-check-case` 0.
- `make lint-model`: the format phase passed. The host's /usr/bin/make 3.81 rejects `--output-sync`, so the four MODEL_LINTS ran serially and `lint-model-syntax` ran separately; both exit 0.
- `lint-code-fast`, read-only against batch base 21b9964965: 0 issues. `.bin/` holds linux-aarch64 golangci-lint/errortype binaries, so darwin builds of the same pinned versions were passed in through GOLANGCI_LINT/ERRORTYPE. `.bin` was not modified.
- `go test -json -count=1 -tags test_dep -p 2 -timeout 30m ./tools/umpire/...`: exit 1, with 2950 pass, 8 skip and 1 failure, all inherited (see below).
- flock is not installed on this host, so heavy suites ran unlocked and one at a time.

### Inherited, not fixed (needs owner or conductor action)
`tools/umpire/cmd/umpire-assess` TestAssessWithAModelReproducesTheLiveAssessment fails with "crossed: the recorded Run is not of the current generated control Case". Task 5's 296f75a1b1 regenerated `model/cases/nexus-workflow-control-forgedCompletion-case.json`. The recorded live Run `common/testing/testpilot/replay/testdata/nexus-workflow-control-forgedCompletion-run.json` was not re-recorded, and task 5's affected-Go runs never ran this package. Fixing it means re-recording a live Run. Editing the recorded identity would fake evidence, so I left it alone. This range does not touch either input.

### Inherited, fixed in its own commit
5a4b2ca271 updates two interp pins from `Presence.scala:75` to `:73`. The comment conversion 0e78bfb051 had moved the line, and task 5's coordinate refresh missed these two.

### Notes
- 76689b5618's `git add -A` swept in the owner's concurrently written `.flow/specs/fn-140-one-sentence-witness-queries-with.{json,md}`. Their content is unchanged. Untracking them again was denied by the permission classifier, so they stay in that commit for the owner to decide. Later commits staged explicit paths only, so the owner's uncommitted MILESTONES.md edit, the fn-141 files and the conductor's fn-132 spec JSON change stay uncommitted.
- Choices I made where the task left them open: chained package clauses for every standalone source, matching the Nexus forms; no local timers/deadline alias objects, because the form reads the kind's directly; the two lift fixtures (Hints, ScriptRejects) import Timeout/deadline from the kind, which adds one line each.
- Follow-up for task 7 and the DSL batch: re-record the nexus-workflow-control replay Run, and either restore darwin binaries in `.bin` or keep `.bin` platform-specific.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 76689b5618fdad212ae5d9e8c8df945f26d20a08, 5a4b2ca271800d1e2466754ddaf437df8be490e2, fcb8c741511d39457120436ef8d4e1906549ba32
- Tests: make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model, make MODEL_GATE_ARGS=--skip-go-checks umpire-check-model, make umpire-check-cases umpire-check-fixtures canary-check-case, make lint-model (format) + serial MODEL_LINTS + lint-model-syntax, lint-code-fast (read-only, base 21b9964965, darwin tool builds), go test -json -count=1 -tags test_dep -p 2 -timeout 30m ./tools/umpire/... (1 inherited umpire-assess failure), python3 -I .flow/tmp/fn-132.6/prove.py 5240b0383b (EXACT PASS, 58 artifacts)
- PRs: