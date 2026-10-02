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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
