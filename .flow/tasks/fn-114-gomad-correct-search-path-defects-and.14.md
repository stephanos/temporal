---
satisfies: [R12]
---
# fn-114-gomad-correct-search-path-defects-and.14 Qualify the combined toolchain and Runner on Darwin and update the docs

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implemented scheduler/search behavior, current-source Darwin runtime/full/core/smoke/representative exact replay, measurements, docs and review. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

R12 here: one Darwin qualification of the candidate that holds every delivered fn-114 change, and the documentation of the delivered behavior. Linux qualification, reports and documentation belong to fn-128.1, fn-128.4 and fn-128.7.

**Size:** M
**Files:** `tools/gomad3/README.md`, `CLI.md`, `ARCHITECTURE.md`, `SPEC.md`, `MILESTONES.md`, `.plans/GOMAD_NEXT.md`, qualification reports retained under `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/qualification/`
**Touches:** [tools/gomad3/README.md, tools/gomad3/CLI.md, tools/gomad3/ARCHITECTURE.md, tools/gomad3/SPEC.md, tools/gomad3/TUTORIAL.md, MILESTONES.md, .plans/GOMAD_NEXT.md, .plans/GOMAD_CMP.md, tools/gomad3/qualification/**, tools/gomad3integration/qualification/**, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/qualification/**, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/retained-bytes.md, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/select-reduction/**]

### Approach
- Record the toolchain build key under test and which runtime edits it contains (tasks 5, 11, 13, and any fn-110, fn-112, or fn-109 runtime task that landed). If another spec's runtime change is about to land, agree one candidate with its owner and qualify once.
- Run on darwin/arm64: `validate`, `test`, the core set, the smoke set, and the representative Temporal set, with exact replay where the manifests require it.
- The corresponding native linux/amd64 gates and reports belong to fn-128.1, fn-128.4 and fn-128.7. Those owners retain Linux as unverified until native evidence exists; missing transferred Linux evidence does not keep source-owned R12 incomplete here.
- A workload whose disposition changes is investigated. Dispositions and manifests are not weakened to obtain a pass.
- Take the after measurements that need this run: representative retained bytes (task 10) and the Signal suite counts (task 12), if those tasks deferred them.
- Docs, each to the delivered state and nothing planned:
  - README: corpus identity now binds environment and tick policy; guided selection, the regression mode, and the new-execution count; the exploration start ordinal and how to find it; minimizer resume; the retained-size statement; the scheduling paragraph on runtime-owned goroutines and select decisions; the provenance rejection list.
  - CLI guide: the three new options and their errors.
  - SPEC: provenance, guidance, choice frontier, artifact, and minimization clauses. ARCHITECTURE: choice traces and exploration, artifact layout, guide identity.
  - Milestones: the fn-114 row and section. Roadmap: BUG-5 keeps typed shrinking, resume is delivered. Assessment: mark each finding delivered, refuted, or open.
- Check the docs against the parsers: every flag named in the docs is accepted by the CLI and every new flag is documented.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/Makefile:115-121`, `:138`, `:175` — core qualification, validate, and test targets
- root `Makefile:192-207` — representative and smoke qualification targets
- `tools/gomad3/README.md:108`, `:143-160`, `:205-222`, `:334`, `:741-757` — sections to update
- `tools/gomad3/SPEC.md:208-228`, `:340-344`, `:358`, `:370` — clauses to update
- `tools/gomad3/ARCHITECTURE.md:252-271`, `:355-381`, `:418-433` — sections to update
- `tools/gomad3/CLI.md:170-187`, `:255-260`, `:414`, `:591` — option documentation
- `MILESTONES.md:42` — the fn-114 row

**Optional** (reference as needed):
- `.plans/GOMAD_NEXT.md:37-45` — BUG-5
- `.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/` — the retained qualification baseline to compare against
- `tools/gomad3integration/qualification/temporal.json`, `smoke.json` — manifests

### Key context
- fn-110 task 5, fn-112 task 10, and fn-109 tasks 20 and 21 also qualify a candidate and edit the same documents. Check their state, rebase onto whichever landed, and share one qualified identity where the changes land together.
- The full `./tests` set is not a gate (milestone constraint on validation scope).
- A finding closed as refuted is documented as refuted, with its evidence, and changes no behavior text.
## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] The build key under test and the runtime edits it contains are recorded
- [ ] `make -C tools/gomad3 validate` and `test` pass on darwin/arm64
- [ ] The core, smoke, and representative sets pass on darwin/arm64 with exact replay where required, with reports retained
- [ ] The corresponding native linux/amd64 gates and retained reports are explicitly assigned to fn-128.1, fn-128.4 and fn-128.7; missing transferred Linux evidence does not keep source-owned R12 incomplete here
- [ ] No disposition or manifest was weakened; any changed disposition has a recorded cause
- [ ] README, CLI guide, SPEC, ARCHITECTURE, milestones, roadmap, and assessment describe the delivered behavior, and every documented flag is accepted by the CLI
- [ ] Each of the ten findings is marked delivered, refuted, or open in the assessment and the milestones
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implemented scheduler/search behavior, current-source Darwin runtime/full/core/smoke/representative exact replay, measurements, docs and review. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
