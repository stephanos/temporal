---
satisfies: [R5, R7]
---
# fn-81-delete-the-pre-testpilot-go-generations.5 Reconcile docs, index, roadmap, and flow records, then run final gates

## Description
Implements R7 and R5 (spec §Commit order 9). Adds historical notes where retained documents link into deleted trees, relabels `.plans/index.json`, adds the roadmap entry, syncs the fn-80, fn-30, and fn-14 records that reference the retired gate list or deleted workflow, and runs the full gate set with before/after measurements for the receipt.

**Size:** M
**Files:** `docs/research/agentworkflow-oss-landscape.md`, eight `docs/superpowers/specs/*.md` (gomad2 entrypoint, gomad3 protobuf IPC, umpire follow-up observability, three agentworkflow designs, genleanmodeldescriptors migration, shared Lean library migration), `model/Temporal/Feature/Nexus/Experimental/AutoClose.lean` (line 173 comment), `tools/umpire/CLEANUP_INVENTORY.md` (planindex sentence), `.plans/index.json`, `.plans/UMPIRE4_ORDER.md`, `.flow/tasks/fn-80-close-the-model-to-case-seam-and-harden.4.md`, `.flow/tasks/fn-80-close-the-model-to-case-seam-and-harden.8.md`, `.flow/specs/fn-80-close-the-model-to-case-seam-and-harden.md`, `.flow/tasks/fn-30-release-evidence-graph-and-manual.6.md`, `.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md`
**Touches:** [docs/research/agentworkflow-oss-landscape.md, docs/superpowers/specs/*.md, model/Temporal/Feature/Nexus/Experimental/AutoClose.lean, tools/umpire/CLEANUP_INVENTORY.md, .plans/index.json, .plans/UMPIRE4_ORDER.md, .flow/tasks/fn-80-*.md, .flow/specs/fn-80-*.md, .flow/tasks/fn-30-*.6.md, .flow/specs/fn-14-*.md]

### Approach
- Docs: one-line status banner at the top of each design record matching the wording two of them already carry ("Historical design: the implementation was removed by fn-81"); de-link the fifteen `../../tools/agentworkflow/...` paths in the landscape doc; rewrite `AutoClose.lean:173` to name the removed file as provenance without a path; rewrite `CLEANUP_INVENTORY.md:58-59` to say planindex was retained by fn-66 and revalidated by fn-81.
- `.plans/index.json`: set `lifecycle: historical` on the six GOMAD3 documents and `AGENT_PLAN.md`; remove the entry for the nonexistent `UMPIRE_DSL_EVOLUTION_SPEC.md`; `go run ./tools/planindex` must pass.
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
