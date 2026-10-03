---
satisfies: [R2, R6, R13, R14, R15, R23, R24, R25]
---
# fn-115-make-the-scala-model-the-model-and.13 Document the live system and validate the complete migration

## Description
Document the live system and validate the complete migration. Implements R2, R6, R13, R14, R15, R23, R24, R25 using the reviewed parent contracts.

**Size:** M
**Files:** model and module READMEs; AGENTS.md; active plans/docs; archive indexes; all open Flow spec path references; MILESTONES.md
**Touches:** [model/**/*.md, tools/umpire/**/*.md, common/testing/testpilot/README.md, tests/testcore/testpilot/README.md, tools/canary/README.md, AGENTS.md, .plans/**, .flow/specs/**, MILESTONES.md]

### Approach
- Write the newcomer README around one real Query and captured Run, defining each term at first use and distinguishing Contract Verdict from model assessment. Show authoring, lifting, both IRs, checking, lowering, runtime and export in a small Mermaid diagram with working commands.
- Update module docs, runtime/fixture/canary docs, active plans and source anchors. Preserve dated evidence as dated evidence. Move the four active legacy-guidance documents into the existing archive directory without overwriting its sixteen frozen documents; update only archive metadata/indexes.
- Update all open downstream specs, including fn118–120, to the map's paths and baseline names via Flow; closed specs/tasks and historical .turbo/docs-superpowers remain untouched. Preserve user model-routing pins in AGENTS.
- Obtain a fresh-context README-only reader audit answering purpose, two IRs and gate invocation; fix failures and record answers.
- Run the complete final model/generation/lint/build/vet/Go/live matrix sequentially, monitor disk, and prove archive hashes, golden equivalence, no extra generated drift and no active stale paths. Conductor performs final independent spec completion review; no commits or staging.

### Investigation targets
**Required** (current paths at planning time; follow the recorded move map after relocation):
- `model/scalav2/README.md:178`
- `model/scalav2/SEMANTICS.md:540`
- `common/testing/testpilot/README.md`
- `tests/testcore/testpilot/README.md`
- `.plans/archive/README.md:18`
- `AGENTS.md`
- `MILESTONES.md`

### Quick commands
The final model gate and deterministic --update comparison; make lint-model; make lint-code; CC=/usr/bin/clang mise exec -- go build ./...; CC=/usr/bin/clang mise exec -- go vet ./...; CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...; the complete previously live Testpilot set with -tags 'test_dep integration canary_harness'

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.
## Acceptance
- [ ] README-only independent reader correctly answers purpose, both IRs and how to run the gate; the worked Query reaches a real recorded Verdict with honest support limits.
- [ ] Active docs and open specs resolve to the current map; archived and closed history is preserved, existing archive-document hashes match, and user model pins remain unchanged.
- [ ] All required broad and previously live gates pass, with exact commands, nonempty test selection, deterministic artifacts and known fixture exceptions recorded.
- [ ] Golden semantics, frozen archive bytes and no-extra-drift checks pass; complete R1–R25 evidence is ready for independent completion review.

## Done summary
Rewrote `model/README.md` for a newcomer: purpose first, every project term defined at first use, a Mermaid diagram of the layers and both IRs, a layer/module/command table, and one real Query (`nexusProtocol/syncCompletion`) followed from its Scala lines through the IR, the generated Case and a recorded Run to `VERDICT_STATUS_SATISFIED`, with the Contract Verdict kept apart from the model assessment and honest support limits (16 of 262 Queries lower to a Case). Two fresh README-only readers answered purpose, the two IRs, how to run the gate and the worked example correctly; their answers are in `.flow/tmp/fn115-13/audit/`. Added `tools/umpire/README.md`; updated the export, Testpilot, Temporal Driver, functional-fixture and canary READMEs, the canary runbook, `SEMANTICS.md`, specimens docs, active plans, the archive index and `AGENTS.md` (Lean mandate replaced by a Model mandate; Flow-Next and model-routing blocks byte-identical). Path references in the seven open specs that had them follow the module map. Stale names in code comments are fixed (comment-only; `ir.pb.go` regenerated for its comment lines).

Final matrix on the final tree: full `make umpire-check-model` (rerun after a load-induced timeout), `umpire-gen-model` with no drift, the three Case checks, `lint-model`, `lint-code`, `lint-api`, build, vet with live tags, the Go suite (3,083 tests), `umpire-check-live-tests` (81 identities), 843/843 archive originals and 1,411/1,411 goldens unchanged. Plain `go vet ./...` keeps 11 findings in three files identical to `main`; `umpire-check-backends` is deferred by the owner.

Independent review (Claude Fable, fresh context): NEEDS_WORK in round 1 (three README paths, fixed by the conductor; Scala-qualified test names, moved to fn-115.15), SHIP in round 2. Handover: .flow/tmp/fn115-13-summary.md; evidence: .flow/tmp/fn115-13-evidence.json; reviews: .flow/tmp/fn115-13-review/. No agent commits.
## Evidence
- Commits:
- Tests: {"command": "make umpire-check-model", "exit": 2, "wall_seconds": 812, "log": ".flow/tmp/fn115-13/gate-check-model.log", "note": "go test timed out (10m default) in export/lower/model under load from another workspace; superseded by run 2"}, {"command": "make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks", "exit": 0, "wall_seconds": 129, "log": ".flow/tmp/fn115-13/gate-gen-model.log", "note": "no artifact or status change"}, {"command": "make umpire-check-cases", "exit": 0, "wall_seconds": 18, "log": ".flow/tmp/fn115-13/check-cases.log"}, {"command": "make umpire-check-fixtures", "exit": 0, "wall_seconds": 19, "log": ".flow/tmp/fn115-13/check-fixtures.log"}, {"command": "make canary-check-case", "exit": 0, "wall_seconds": 18, "log": ".flow/tmp/fn115-13/canary-check-case.log"}, {"command": "mise exec -- make lint-model", "exit": 0, "wall_seconds": 12, "log": ".flow/tmp/fn115-13/lint-model.log"}, {"command": "GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code", "exit": 0, "wall_seconds": 238, "log": ".flow/tmp/fn115-13/lint-code.log"}, {"command": "make lint-api", "exit": 0, "wall_seconds": 3, "log": ".flow/tmp/fn115-13/lint-api.log"}, {"command": "CC=/usr/bin/clang mise exec -- go build ./...", "exit": 0, "wall_seconds": 118, "log": ".flow/tmp/fn115-13/go-build.log"}, {"command": "go vet -tags \"test_dep integration canary_harness\" ./tests/... ./common/testing/testpilot/... ./tools/canary/... ./tools/umpire/...", "exit": 0, "wall_seconds": 52, "log": ".flow/tmp/fn115-13/go-vet-live-tags.log"}, {"command": "go vet -tags test_dep ./...", "exit": 1, "wall_seconds": 56, "log": ".flow/tmp/fn115-13/go-vet-test_dep.log", "note": "11 known findings in 3 files identical to main"}, {"command": "go test -count=1 -json -tags 'test_dep canary_harness' ./common/testing/testpilot/... ./tools/canary/... ./tests/testcore/testpilot/...", "exit": 0, "wall_seconds": 44, "log": ".flow/tmp/fn115-13/full-go.jsonl", "note": "32 pkgs pass, 2 no tests; 3083 tests pass, 10 skip"}, {"command": "make umpire-check-live-tests", "exit": 0, "wall_seconds": 507, "log": ".flow/tmp/fn115-13/live-tests.log", "note": "81 passing identities"}, {"command": "make umpire-check-model", "exit": 0, "wall_seconds": 486, "log": ".flow/tmp/fn115-13/gate-check-model-2.log", "note": "final tree; includes go vet + go test -tags test_dep ./tools/umpire/... (16 packages ok)"}, {"command": "GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code", "exit": 0, "wall_seconds": 223, "log": ".flow/tmp/fn115-13/lint-code-2.log", "note": "after two comment-only edits"}, Independent review round 2 SHIP (claude-fable-5-1); .flow/tmp/fn115-13-review/round2-review.md
- PRs: