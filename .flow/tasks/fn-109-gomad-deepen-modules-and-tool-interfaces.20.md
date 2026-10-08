---
satisfies: [R9]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.20 Reconcile architectural guidance with the delivered owners and interfaces (fulfils fn-105.5 D5)

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Stage 6, R9 (F8). Most of the original F8 premise is already fixed: verify and cite that, then document what this spec changed. This task is the single owner of fn-105.5 (D5, origin brief `.flow/tasks/fn-102-gomad-architecture-consolidate.6.md`); close fn-105.5 by reference afterwards.

**Size:** S/M
**Files:** `tools/gomad3/ARCHITECTURE.md`, `tools/gomad3/SPEC.md`, `tools/gomad3/README.md`, `tools/gomad3/CLI.md`, `tools/gomad3/TUTORIAL.md`, `MILESTONES.md` (status lines only), `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/documentation-evidence.md`.
**Touches:** [tools/gomad3/*.md, MILESTONES.md, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/documentation-evidence.md, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-20/trace-version-correction-20261005/**]

### Approach
- Already fixed since the assessment; needs evidence, not rewriting: `SPEC.md:176` names both `darwin/arm64` and `linux/amd64`; `ARCHITECTURE.md:17-23` and `:659-662` state both platforms; the closing paragraph (`:687-691`) no longer classifies choice tracing as research; `GLOSSARY.md` was deleted by fn-111 and its terms live in SPEC. Reuse fn-111's retained evidence (`flowctl show fn-111-gomad-consolidate-vocabulary-and-update`, its historical tasks .1/.2 and current task .3 evidence under `.flow/artifacts/fn-111-gomad-consolidate-vocabulary-and-update/`) rather than re-deriving it. If fn-111 is still open, cite its state and do not edit the same passages concurrently.
- New documentation owed by this spec: the options owner and coordinator envelope; the preparation owner with its two operations; the Go-command seam and its two output contracts; the installation description; private executor injection and the Artifact reference/handle split, as intentional Go interface changes sourced from `go-interface-changes.md`; the generated simulation-time protocol under "Binary protocol ownership"; the simulation progress lifecycle owner under "Process arbitration and model evidence"; backend-specific handles; the architecture checks under "Maintenance gates".
- Reconcile task 19's inventoried public report graphs, pack-directory intent and World detached-terminal/closed-error migration after its source freezes. Reuse its relevant boundary guidance and preservation evidence; document the actual final names and reporting-versus-model ownership, rather than repeating implementation or claiming unchanged direct custom-error/sentinel-rebinding behavior. The retained task-20 source scouts cover tasks 2–18 and must be reconciled with the final task-19 candidate.
- Keep four claims separate everywhere: capability support, same-seed repeatability, exact replay, and CI expectation matching. Use existing SPEC requirement IDs; add none.
- Preserve current residual dispositions: Linux replay divergence (D12), suites without exact replay and host-clock escapes such as `MemStats.LastGC` stay open. D14's Darwin correction remains recorded as done with its source-bound evidence; historical summaries saying D14 is open do not reopen it. A structural refactor neither fixes D12 nor qualifies the final integrated candidate from D14's earlier native result.
- No performance or support claim without a measurement retained under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`. Describe each change in the present tense as current behaviour; delivery history belongs in the milestones file.
- Update only this spec's status rows in `MILESTONES.md` (work-tracking row and the fn-109 section status); leave other specs' text alone.

### Investigation targets
**Required:**
- `tools/gomad3/ARCHITECTURE.md` (whole), `tools/gomad3/SPEC.md:140-240`
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md`, `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/simulation-progress-design.md`
- `.flow/tasks/fn-102-gomad-architecture-consolidate.6.md`
- `MILESTONES.md` sections "Open findings", "Deep modules and tool interfaces (fn-109)", "Vocabulary and documentation (fn-111)"
- `tools/gomad3/toolchain/version/version.json` (supported platforms as generated fact)

### Quick commands
```bash
cd tools/gomad3
grep -n 'darwin/arm64\|linux/amd64' SPEC.md ARCHITECTURE.md README.md
GOWORK=off go test -count=1 -tags test_dep . -run 'TestCurrentVocabularyHasNoLegacyCampaignBoundary|TestMakeTargetsMatchTheirOwnership'
make validate
cd ../.. && flowctl show fn-105-gomad-follow-ups-deferred-scope.5 && flowctl tasks --spec fn-111-gomad-consolidate-vocabulary-and-update
```

### Constraints
- Follow MILESTONES verification instruction 5: commit each verified task separately before starting the next task, including implementation, tests, documentation and Flow records. Root is the sole committer; preserve unrelated changes and push only when authorized. Keep unavailable native gates incomplete and acceptance open. This supersedes older user-only commit instructions; stash, worktree creation and history rewrites require separate authorization.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- Recheck the actual execution host before gates. This development session is `linux/arm64`, with the patched toolchain absent; neither native `darwin/arm64` nor native `linux/amd64` qualification is available here. Cross-platform source/type/vet checks and stock-host tests are developmental evidence only. Keep each required native gate incomplete until a source-bound result exists on its qualified platform.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.

### Documentation source checkpoint (2026-10-04)

The five guides and permitted fn-109 milestone status lines have passed the focused Quick commands, document checks and independent corrective source review. The generated host-codec path finding is corrected. See `task-20/source-checkpoint.md`, the immutable original and corrective review reports, and the worker handover/evidence for the frozen hashes and commands. Root will commit this verified source progress before successor source work. Formal task-20 review, task-19 formal/native gates, acceptance and fn-105.5 closure remain open; task 21 remains unadmitted. No acceptance criteria or dependency is waived.

### Source-progress revival (2026-10-05): current choice trace guidance

Correct the six stale v2 Choice Trace/replay claims in README.md, ARCHITECTURE.md and TUTORIAL.md to the current v3 reader and projection contract. Explain select-result readiness projection onto corresponding poll decisions, unknown readiness when no result names a decision, explicit v2 refusal, and legacy-v1 decoding for inspection without a replay tape. Legacy records can retain decision flags; inspection-only support does not mean every record is an observation. Preserve unrelated artifact/pack/schema-v2 references, all commands, flags, qualification dispositions and production source. This satisfies part of R9; it neither implements legacy-format support nor resolves the separate R18 preservation reconciliation.

The source scout identified choice/trace.go DecodeStoredTrace and choice/tape.go ProjectReplayPlan/projectSelectReadiness as the current authoritative behavior, with existing reader-version/refusal and readiness tests. fn-114.11 owns the version migration and its retained evidence; fn-114.14 consumes final qualification. Task 20 remains the guidance owner. Keep dependency .19, every original Acceptance item, fn-105.5 closure, formal review and required Darwin/full/affected gates open where unproved. Linux execution stays nonblocking under fn-128. No new requirement or waiver is introduced.

Baseline and rerun the existing task Quick checks, focused choice-reader/readiness controls and check-only generator validation on frozen source. Retain a six-row before/after claim mapping, links/fences/whitespace checks and proof that every other product source byte and unrelated v2 reference is unchanged. Documentation-only scope needs no broad Go lint rerun; retain the actual inherited 317-diagnostic lint result and unreached integrated errortype as historical/current-code residuals, not a new pass. A fresh independent source-progress review precedes the separate progress commit. Root owns Flow lifecycle, MILESTONES status and commit; the worker owns only the three named guides and this unique evidence directory. Previous task work is terminal and committed. External research and formal plan review are skipped for this bounded factual correction; source review gates the checkpoint and original formal acceptance remains required.

### Current trace guidance checkpoint (2026-10-05)

The three guides now name current v3 Choice Trace/replay evidence in all six selected passages and explain readiness, stored-v2 refusal and legacy-v1 inspection without replay. [The checkpoint](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-20/trace-version-correction-20261005/progress.md) links the exact six-row source mapping, meaningful baseline claim failures, all 25 final document checks, fresh BASE/FINAL/conductor tests and generator validation, 1,108 Git-base bindings and 1,105 unchanged paths. Commands, other schemas, runtime source and qualification dispositions retain their bytes.

The actual gate classifier returns FULL for the three guides and two unrelated user scratch files; no docs-only tier-B pass or full-gate pass is claimed. The inherited 317-diagnostic lint result remains historical evidence against unchanged executable source, with integrated errortype unreached. Original task .19 dependency, full R9/R18 reconciliation, formal/native/full/affected gates and fn-105.5 closure remain open. This checkpoint changes guidance only; it adds no legacy-format support or native replay proof. Linux execution remains nonblocking under fn-128.

stage: impl-review - skipped(policy: original full/native/R18 gates remain open; separate source-progress review recorded)
stage: plan-sync - skipped(config: disabled; no task completion)
## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] `documentation-evidence.md` cites, with file and line, the already-correct platform, choice replay/exploration and backend statements, and links fn-111's evidence instead of duplicating it.
- [ ] Architecture guidance describes the delivered owners (options, preparation, command seam, installation description, generated simulation-time protocol, progress lifecycle, backend handles, architecture checks) using existing requirement IDs.
- [ ] Every intentional Go interface/behavior migration in go-interface-changes.md, including executor injection, Artifact reference/handle and task 19's public reports, pack-directory intent and detached World terminal boundary, is documented with its actual replacement and caller migration.
- [ ] Capability support, repeatability, exact replay and expectation matching are stated as separate claims. D12, suites without exact replay and known clock escapes remain open; D14 retains its recorded Darwin fix and source-bound native evidence without treating it as qualification of the final integrated candidate.
- [ ] No unmeasured support or performance claim, obsolete delivery-state claim, or failure described as qualification success is present; local links and code fences resolve.
- [ ] fn-105.5 is closed by reference to this task (one owner).
## Done summary
Blocked on inherited D5 and final qualification; nine-flag CLI documentation correction source-reviewed.

The earlier three-draw formal SHIP covers its recorded source checkpoint only. The current correction passed independent source review and a bounded corrective review with zero remaining findings. Focused tests, make validate and all 19 document/preservation checks passed; unavailable native gates remain incomplete.

[Current correction and open acceptance](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-20/cli-inventory-correction/acceptance-open.md) links the exact reviewed hashes and immutable prior captures. Task 21's developmental 10/100 controls are retained, but R18 reconciliation, both native gates, task 19 formal review and fn-105.5 closure remain open. Original acceptance and dependencies are unchanged.

stage: impl-review - skipped(policy: red root lint and incomplete qualification; source-progress checkpoint only)

Blocked:
ORIGINAL_QUALIFICATION_OPEN. Six stale Choice Trace/replay version claims now match current v3 readers. The three guides explain readiness projection, unknown readiness, explicit v2 refusal and legacy-v1 inspection without a replay tape. Fresh BASE, FINAL and conductor checks pass the two vocabulary/Make tests, three reader/readiness/legacy controls and generator validation. All 25 final document checks pass; 1,105 other inventoried paths retain their BASE bytes.

This documentation checkpoint implements no legacy-format support and proves no current native replay or full R9/R18 reconciliation. The actual classifier returns FULL for the three guides and two unrelated user scratch files; it is not a docs-only tier-B pass. Original full/native/affected gates remain open. The prior actual lint receipt retains 317 diagnostics and integrated errortype UNREACHED; no fresh lint run or green result is claimed.

Keep dependency .19, original task acceptance, formal review, required Darwin/full/affected qualification and fn-105.5 D5 closure open where unproved. Transferred Linux execution remains nonblocking under fn-128. Historical completion/evidence stays historical and unchanged.
## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
