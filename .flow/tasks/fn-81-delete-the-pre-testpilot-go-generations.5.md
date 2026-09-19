---
satisfies: [R5, R7]
---
# fn-81-delete-the-pre-testpilot-go-generations.5 Reconcile docs, index, roadmap, and flow records, then run final gates

## Description
Implements R7 and R5 (spec §Commit order 9). Adds historical notes where retained documents link into deleted trees, relabels `.plans/index.json`, adds the roadmap entry, syncs the fn-80, fn-30, and fn-14 records that reference the retired gate list or deleted workflow, and runs the full gate set with before/after measurements for the receipt.

**Size:** M
**Files:** `docs/research/agentworkflow-oss-landscape.md`, seven `docs/superpowers/specs/*.md` (gomad2 entrypoint, umpire follow-up observability, three agentworkflow designs, genleanmodeldescriptors migration, shared Lean library migration), `model/Temporal/Feature/Nexus/Experimental/AutoClose.lean` (line 173 comment), `tools/umpire/CLEANUP_INVENTORY.md` (planindex sentence), `.plans/index.json`, `.plans/UMPIRE4_ORDER.md`, `.flow/tasks/fn-80-close-the-model-to-case-seam-and-harden.4.md`, `.flow/tasks/fn-80-close-the-model-to-case-seam-and-harden.8.md`, `.flow/specs/fn-80-close-the-model-to-case-seam-and-harden.md`, `.flow/tasks/fn-30-release-evidence-graph-and-manual.6.md`, `.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md`
**Touches:** [docs/research/agentworkflow-oss-landscape.md, docs/superpowers/specs/*.md, model/Temporal/Feature/Nexus/Experimental/AutoClose.lean, tools/umpire/CLEANUP_INVENTORY.md, .plans/index.json, .plans/UMPIRE4_ORDER.md, .flow/tasks/fn-80-*.md, .flow/specs/fn-80-*.md, .flow/tasks/fn-30-*.6.md, .flow/specs/fn-14-*.md]

### Approach
- Docs: one-line status banner at the top of each design record matching the wording two of them already carry ("Historical design: the implementation was removed by fn-81"); de-link the fifteen `../../tools/agentworkflow/...` paths in the landscape doc; rewrite `AutoClose.lean:173` to name the removed file as provenance without a path; rewrite `CLEANUP_INVENTORY.md:58-59` to say planindex was retained by fn-66 and revalidated by fn-81.
- `.plans/index.json`: set `lifecycle: historical` on `GOMAD_CMP.md` and `AGENT_PLAN.md`; the GOMAD3 documents and `GOMAD_MILESTONES.md` keep their lifecycle because Gomad v3 is retained; remove the entry for the nonexistent `UMPIRE_DSL_EVOLUTION_SPEC.md`; `go run ./tools/planindex` must pass.
- `.plans/UMPIRE4_ORDER.md`: `### 3. Delete the pre-Testpilot Go generations — fn-81` under Current work after the fn-80 block, noting it removed the pinned live-failure list and the legacy white-box seam.
- Flow records via `flowctl task set-spec` and `flowctl spec set-plan`: fn-80 tasks .4 and .8 acceptance lines "expected-failure list unchanged" become "`make umpire-check-live-tests` passes"; fn-80 spec Edge Cases sentence likewise; fn-30 task .6 investigation anchor moves from `umpire-model-verification.yml` to `.github/workflows/umpire.yml`; fn-14 spec gains a closure note that its agentworkflow Touches point at removed files.
- Final gates: `make lint-code`, `make lint-model`, `make umpire-check-regression`, `go run ./tools/planindex`, the Quick commands test set, and the Quick commands grep (only annotated notes may remain).
- Receipt measurements: `git diff --stat <spec-start-commit>..HEAD | tail -1`, `du -sh tools/umpire3 model/.lake` before (from the ledger) and after, `git ls-files | wc -l` before and after.

### Investigation targets
**Required** (read before coding):
- `.plans/index.json` — lifecycle fields and the stale entry
- `.plans/UMPIRE4_ORDER.md:1-61` — Current work section shape
- `docs/superpowers/specs/2026-08-23-agentworkflow-yaml-workflows-design.md:1-5` — existing banner wording to copy

**Optional** (reference as needed):
- `tools/umpire/internal/retiredvocabulary/check.go:264` — banned tokens new prose must avoid

## Acceptance
- [ ] Every listed doc carries a historical banner or rewritten reference; the Quick commands grep returns only those annotated lines
- [ ] `go run ./tools/planindex` passes with the relabeled entries and no stale entry
- [ ] UMPIRE4_ORDER has the fn-81 entry; fn-80 .4/.8 and spec, fn-30 .6, and fn-14 records are updated via flowctl and `flowctl validate --all` passes
- [ ] `make lint-code`, `make lint-model`, `make umpire-check-regression` pass
- [ ] Receipt records line-count delta, `du -sh` before/after for `tools/umpire3` and `model/.lake`, and tracked-file counts before/after

## Done summary
Reconciled every record that pointed into a deleted tree, then ran the final gate set.

Documentation: historical banners on the four `.plans` documents whose subjects were removed
(`UMPIRE2.md`, `UMPIRE3.md`, `AGENT_PLAN.md`, `GOMAD_CMP.md`), on the two research records that
linked into `tools/umpire3`, and on every tracked `docs/superpowers` design record for a removed
implementation. The sixteen `tools/agentworkflow` source links in the research landscape are
unlinked and its "implemented in the current working tree" claim is past-tensed. `AutoClose.lean`
names the Umpire v2 model as provenance instead of carrying a dangling path. `LEAN_GUIDELINES`
drops its mathlib claim about `tools/umpire3/model`, `GOMAD_MILESTONES` strikes its now-done umpire3
closure item, and `UMPIRE4_COMPONENTS` warns that its superseded inventory's present-tense package
references describe deleted trees.

Index: `.plans/index.json` relabels `AGENT_PLAN.md`, `GOMAD_CMP.md`, and `UMPIRE2.md` to
`historical` lifecycle and authority, drops the stale `UMPIRE_DSL_EVOLUTION_SPEC.md` entry, and
allowlists with reasons the four links into deleted trees that the sweep left dangling.

Roadmap: the fn-81 entry in `.plans/UMPIRE4_ORDER.md` was rewritten from the planned description to
the landed outcome, with the realized per-tree line counts, the Gomad v3 retention, the two
unpredicted consumers, and the gate redesign. The concurrent fn-77 move to `## Completed cutovers`
and the fn-80 re-plan were merged rather than overwritten; only section 2 changed.

Flow records: fn-80 tasks .4 and .8 and its spec no longer assert an unchanged expected-failure
list, since fn-81 retired it; they now require `make umpire-check-live-tests` to pass. fn-30 task .6
points at the retained `umpire.yml`. fn-14 gains a closure note that `tools/agentworkflow` is gone,
so its trials cannot be revived as written. The 2026-09-04 memory entry records the gate redesign
and the new rule that an empty inherited failure set needs a passing-identity floor.

Gates. `make umpire-check-regression` passes end to end, live gate included. Three gates are
inherited-red and are recorded as such rather than as green: `make lint-model` is byte-identical to
its 169-error baseline because fn-81 deleted no Lean; `make lint-code` fell from 1,284 issues to 128
with none in a file this spec edited; `go run ./tools/planindex` is back to its 44-finding baseline
after the sweep pushed it to 49, with all five fn-81-owned findings resolved and none added. The
acceptance lines that read "`make lint-code` ... pass" and "`go run ./tools/planindex` passes" were
not achievable at any point in this spec, and the ledger recorded that at task .1.

Two findings the review raised are recorded rather than fixed. Ten of the twenty-five
`docs/superpowers/specs` records — including six the reviewer named — are excluded from this
repository by the user's global gitignore (`docs/superpowers/**`) and are not tracked, so R7 cannot
reach them; the annotations were applied locally and cannot be committed. And fn-81 is still
unregistered in the index's `flowSpecs`, along with roughly thirty other specs; registering them is
the separate reconciliation the ledger scoped out of this spec's boundaries.

stage: impl-review - ran (backend claude, model claude-fable-5-1, effort high, 3 rounds: NEEDS_WORK
on an index entry still claiming architecture authority for a deleted tree, NEEDS_WORK on an
un-annotated genmodels reference and a premature "all five tasks are done" roadmap claim, then SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4e60340045e2ca2d81ea8690d00b25eafe5916a4, 00352315e87c8820c2d082de8f3fb2b2c1489ae0, 04ecff91a8d64f8ef4f69fc40a8e632591d0ceef, e56f9b6f355d0d68828646d70c0baea1aac75ad4
- Tests: make lint-model: 'Found 169 errors in 23389 declarations ... in Temporal.Lint with 8 linters' - byte-identical to the task-start baseline (167 simpNF in the generated Temporal.API.Types, 2 unusedArguments in the generated Temporal.API.Proto). Shared and Umpire.Lint pass. fn-81 deleted no Lean, so this gate did not move. Exit 2, inherited-red., make umpire-check-regression: rc=0 end to end, including the redesigned live gate ('Live Testpilot failure identities match the empty expected set across 4 passing identities.') and the full tagged package run., make lint-code GOLANGCI_LINT_FIX=false: 128 issues (errcheck 1, govet 4, revive 106, staticcheck 17), down from the 1,284-issue task-start baseline; exhaustive 5->0, forbidigo 209->0, goimports 1->0, testifylint 1->0. None of the 128 is in a file this spec edited. Exit 2, inherited-red., go run ./tools/planindex: 44 findings, from 44 at task start and 49 after the deletion sweep. The five fn-81 owned it (four dangling .plans links plus the stale UMPIRE_DSL_EVOLUTION_SPEC.md entry) are resolved and no new finding appeared. Exit 1, inherited-red on the ~40 unregistered .plans documents and flow specs the ledger scoped out., go build -tags 'test_dep integration' ./... (rc=0), go vet -tags test_dep ./... (rc=1, still exactly the 15 inherited diagnostics), CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/... ./common/testing/testpilot/... ./tests/testcore/... : 34 packages ok; the two inherited-red packages are tools/planindex (the .plans registration drift above) and tools/tests (no local Cassandra on 127.0.0.1:9042)., go run ./tools/umpire/cmd/umpire-check-retired-vocabulary (rc=0), Quick-commands grep over the tree excluding .plans/.flow/.turbo/docs: only the retained gomad3 artifact-store paths (.gitignore:4, Makefile:867) and the fn-81 ledger's own annotated rows., flowctl validate --all: 82 specs, 503 tasks, 0 errors., receipt: git diff --stat 5b9cc0dc9..HEAD = 1525 files changed, 1576 insertions, 386985 deletions; tracked files 7785 -> 6345; du -sh tools/umpire3 = 5.1M before, gone after; du -sh model/.lake = 2.8G before and after (untouched, retained workspace).
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
