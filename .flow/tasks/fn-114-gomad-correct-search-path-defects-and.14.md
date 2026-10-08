---
satisfies: [R12]
---
# fn-114-gomad-correct-search-path-defects-and.14 Qualify the combined toolchain and Runner on Darwin and update the docs

## Description

Review correction (2026-10-08). First formal round retained one introduced P3/R12: source case clauses were confused with polled non-nil cases. Current correction context is [review-fix handover](../artifacts/fn-114-gomad-correct-search-path-defects-and/task-14/source-acceptance-20261008/review-fix-1/handover.md) and [current evidence](../artifacts/fn-114-gomad-correct-search-path-defects-and/task-14/source-acceptance-20261008/review-fix-1/evidence.json). The nil-channel fixture has three clauses but two polled non-nil cases; all seven allowed shapes use PolledCases 2. Current documentation and milestone wording are bound to the correction snapshot; original reports/commands retain the checkpoint meaning at 1fcef5ea4bfbf3ec5f7279a8a6c67070d5159a9d. No code, manifests, qualification, policy or acceptance scope changed. Re-review resumes the same canonical receipt; the original NEEDS_WORK round is not refunded or reset.

Current source acceptance (2026-10-08). Mandatory review context is [handover](../artifacts/fn-114-gomad-correct-search-path-defects-and/task-14/source-acceptance-20261008/handover.md), [evidence](../artifacts/fn-114-gomad-correct-search-path-defects-and/task-14/source-acceptance-20261008/evidence.json), the six changed product documents and MILESTONES finding rows. Review all ten source dispositions and their precise implementation/test links, actual parser option/error contracts, separate campaign/corpus/merge accounting, seven-shape select proof boundary and per-parent minimizer state. The frozen documentation audit and conductor commands bind current inputs; older whole-module snapshots do not cover these documentation changes. The missing patched-driver coverage test remains an inconclusive native observation, not a portable failure waiver or native pass. Reused scoped Go/runtime standards keep their original filters and the fn-109 aggregate lint remains open. Native manifests/dispositions remain byte-identical. Root owns formal review and completion; no native qualification or publication action is authorized.

Current source-acceptance admission follows fn-114.13 source completion at ea2c16ae62. Its five source gates, exact preservation chain and fresh SHIP review remain reusable only under matching inputs. Conductor admits task-local source acceptance evidence to Touches. Complete the retained documentation/flag/disposition reconciliation without native execution, a new qualification pass, report generation or any PR/push/CI action. The worker owns admitted product documentation and task-local evidence; root owns Flow, lifecycle milestones, review, completion and Git. Milestone content corrections are returned as recommendations for root to apply. Native build-key/measurement/full-host/core/smoke/representative evidence stays with the mapped native owners.

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implemented scheduler/search behavior, current-source Darwin runtime/full/core/smoke/representative exact replay, measurements, docs and review. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

R12 here: one Darwin qualification of the candidate that holds every delivered fn-114 change, and the documentation of the delivered behavior. Linux qualification, reports and documentation belong to fn-128.1, fn-128.4 and fn-128.7.

**Size:** M
**Files:** `tools/gomad3/README.md`, `CLI.md`, `ARCHITECTURE.md`, `SPEC.md`, `MILESTONES.md`, `.plans/GOMAD_NEXT.md`, qualification reports retained under `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/qualification/`
**Touches:** [tools/gomad3/README.md, tools/gomad3/CLI.md, tools/gomad3/ARCHITECTURE.md, tools/gomad3/SPEC.md, tools/gomad3/TUTORIAL.md, MILESTONES.md, .plans/GOMAD_NEXT.md, .plans/GOMAD_CMP.md, tools/gomad3/qualification/**, tools/gomad3integration/qualification/**, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/qualification/**, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/retained-bytes.md, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/select-reduction/**, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-14/source-acceptance-20261008/**]

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


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] The build key under test and the runtime edits it contains are recorded
- [ ] `make -C tools/gomad3 validate` and `test` pass on darwin/arm64
- [ ] The core, smoke, and representative sets pass on darwin/arm64 with exact replay where required, with reports retained
- [ ] The corresponding native linux/amd64 gates and retained reports are explicitly assigned to fn-128.1, fn-128.4 and fn-128.7; missing transferred Linux evidence does not keep source-owned R12 incomplete here
- [ ] No disposition or manifest was weakened; any changed disposition has a recorded cause
- [ ] README, CLI guide, SPEC, ARCHITECTURE, milestones, roadmap, and assessment describe the delivered behavior, and every documented flag is accepted by the CLI
- [ ] Each of the ten findings is marked delivered, refuted, or open in the assessment and the milestones
## Done summary
# fn-114.14 source acceptance

Delivered R12 documentation/source acceptance under the scoped native ownership amendments. The five Gomad guides, assessment and milestone rows now describe all ten source dispositions, CLI options/errors, per-parent minimizer state, standalone campaign charging versus corpus/shared/merged accounting, and seven proven select shapes with two polled non-nil cases. Nil-channel source clauses are not counted as polled cases. Typed scenario shrinking remains open; no runtime code or qualification manifest changed.

Fresh conductor verification executed 16 portable contract tests with zero failures/skips, generated-source validation and whitespace successfully. The final documentation audit covers 491 flag occurrences, 131 examples, 38 registrations, 93 local destinations, six unchanged qualification manifests, 87 precise preservation inputs and the retained 72-file raw chain. Root independently verified 26 evidence references, 89 frozen fix inputs, the three-clause/two-polled-case fixture and unchanged executable inputs. Historical full-module snapshots and pre-fix commands retain their original meanings, not blanket current-documentation credit.

Formal review reached SHIP on the second delivered round after correcting the first round's introduced P3/R12 wording finding. The original NEEDS_WORK and all three first-round reports remain retained; no round was reset or refunded. Reviewer selected codex:gpt-6.1-sol:high, same GPT family as the writer; executing-model metadata was not independently observed. Final R12 is met within retained source scope with no unaddressed R-ID.

The mixed selection's absent patched-driver coverage test remains an inconclusive native observation, not a portable failure waiver or native pass. Current runtime/full-host/core/smoke/representative/replay/measurement/soak obligations stay deferred and unverified under fn-149/fn-128. Parent fn-109 aggregate lint remains open; reused scoped Go/runtime standards retain their recorded filters and limits. No native qualification, PR, push or CI action occurred.

Evidence: handover.md, evidence.json, review-fix-1/{handover.md,evidence.json}, conductor-checks.json and source-review.json in this directory. Product checkpoints: 1fcef5ea4bfbf3ec5f7279a8a6c67070d5159a9d and 0f61fa18b8242be96ad4c51c7f2c86a95fc631ca; source review range ea2c16ae62165222d92838540e3f2e08060e6ebf..0f61fa18b8242be96ad4c51c7f2c86a95fc631ca.

stage: wave-dispatch - ran (model: explicitly selected gpt-6.1-sol at high; executing metadata unavailable)
stage: impl-review - ran (two delivered rounds; selected codex:gpt-6.1-sol:high; final SHIP)
stage: plan-sync - skipped(config: planSync.enabled is false)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits: 1fcef5ea4bfbf3ec5f7279a8a6c67070d5159a9d, 0f61fa18b8242be96ad4c51c7f2c86a95fc631ca
- Tests: '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go' '-C' 'tools/gomad3' 'test' '-tags' 'test_dep' '-count=1' '-json' '-run' '^(TestCharacterize(InputRejection|ByteSizeAndFlagValueRejection|ExploreRequestDefaultsAndWiring)|TestWorkspace(Resume.*|KeepsStatePerParentArtifact|RejectsSymbolicLinkStateRoot|StateGatesInitialRunsAndResumes)|TestIdentityForProjectsCanonicalEnvironmentAcrossSeedsAndProfiles|TestCorpusRejects(IdentityChangesAndNonMatchingReplay|PreviousAndFutureSchemaBeforeChangedIdentity|CaseWithChangedEnvironment)|TestValidateProvenanceRejectsUnsupportedBuildModes|TestReplayBuildInfoRejectsMatchingCoverageInstrumentation)$' './cmd/gomad/internal/cli' './runner/internal/minimizer' './runner/internal/corpus' './target' './runner', 'make' '-C' 'tools/gomad3' 'validate', 'git' 'diff' '--check', '/usr/bin/python3' '/Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-14/source-acceptance-20261008/review-fix-1/audit.py', 'git' 'diff' '--check'
- PRs:
## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implemented scheduler/search behavior, current-source Darwin runtime/full/core/smoke/representative exact replay, measurements, docs and review. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
