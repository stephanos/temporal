---
satisfies: [R2, R3, R18, R19, R20]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.21 Run final qualification and retain the finding completion matrix

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.
Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Stage 6, R18-R20, plus the evidence links for R2/R3. Run the complete gates once against the finished tree, compare with the baseline, and write the matrix that maps every finding to its requirement and evidence. This task implements nothing; a gap it finds goes back to the owning task.

**Size:** M
**Files:** `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/completion-matrix.md`, `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/qualification-evidence.md`, retained command output under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/evidence/`.
**Touches:** [.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/**, MILESTONES.md]

### Approach
- Baseline: record the pre-fn-109 revision (planning anchored at `d4d800fb47`; use the actual first-task base), toolchain build key, and the current qualification dispositions (`tools/gomad3integration/qualification/*.json`, the core set report) before judging regressions.
- R2/R3: no implementation here. Link the evidence of `fn-108-gomad-reduce-code-size-without-removing.5` (R6), `.6` (R7) and `.7` (equivalence and gate evidence). If either is not done and verified, R2/R3 stay open and this task reports the spec as incomplete.
- Completion matrix: one row per finding F1-F11 and S1-S5 with R-ID, owning task, implementation evidence (files and symbols) and verification evidence (command and result). D1/D2 map to fn-108 once; D3/D4/D5 map to tasks 6, 19 and 20 once, with fn-105.3/.4/.5 closed by reference. No finding may be closed by deferral, and no obligation may have two owners.
- R18 preservation audit: exported-surface diff of `runner`, `artifact`, `target`, `deterministicio`, `record`, `choice`, `world`, `qualification` against the baseline (only the changes in `go-interface-changes.md` may appear); CLI command and flag inventory against `CLI.md`; comment preservation spot-check on moved code (`git diff` for deleted comment lines); fixed-identity canonical projections; no new generic host-I/O grant in the boundary manifest or compatibility packs.
- Source-owned R19 gates run on native `darwin/arm64`, serialized (they share `.toolchain` and qualification output directories): generator validation, the full Gomad test tiers, native/default integration, functional smoke, the core qualification set, and the suites affected by overlay changes. Failures attributable to D12/D14 are recorded with their evidence under those owners, not waived.
- Bounded control cases: run a seed campaign at 10 and at 100 jobs with fixed parallelism and compare policy-state size and payload copies (heap profile or explicit counters) to show no selection-sized state or extra full-payload copy was introduced by the extractions. Record the numbers; an unmeasured claim is not evidence.
- Measurement preparation is retained in `task-21/bounded-campaign-measurement-source-scout.md`. Use an isolated scratch-only same-package verification harness, retained under this task's artifacts, rather than production hooks or shipped source changes. Reconstruct the actual first-task dirty baseline before comparison; run baseline/current subcases separately after the final source freezes. Distinguish bounded policy/live payload state from selection-derived journal capacities and retained evidence. Profiles need site-specific attribution and normalized per-execution bytes; constructor alias checks alone cannot exclude transient copies. The existing 10/100 active-width/capacity test is a behavioral control, not the missing memory/copy measurements. Any production gap returns to its owning task.
- The actual first-task dirty nested-module baseline has been reconstructed under `task-21/baseline-reconstruction/`. Read `reconstruction.md` and `conductor-verification.md`, then freshly verify its input/source manifests before measurement. The complete 670-file reconstruction independently matches the nested-module Git blobs and executable modes at `38957053f1ce342a8797af1803f5f8f6bb53fcad`; that later equivalent tree corroborates the original baseline without replacing it or identifying the whole historical repository. No measurements or qualification were performed by the reconstruction.
- The environment-bound developmental baseline campaigns are retained in `task-21/bound-baseline-measurement/measurement.md`, with complete pre/post source inventories, explicit pre-launch environment bindings, per-case commands and allocation-site attribution. Read `task-21/conductor-bound-baseline-verification.md` and the independent checkpoint source review before the matched current-tree run. Preserve the historical manifests and their disclosed task-description-only metadata drift. These four baseline campaigns do not complete the current comparison, task admission, R19 or either native platform's gates.
- For unavailable native darwin/arm64 execution, list every source-owned gate as incomplete with its exact command (CI workflow `.github/workflows/gomad3.yml`). Source-owned R19 acceptance remains unmet until Darwin has current-source-bound evidence; the current unsupported host supplies none. Native linux/amd64 gates and their evidence ledger belong to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md), and missing transferred Linux evidence does not block this task. A historical pre-integration result or developmental run cannot satisfy native qualification on either platform.

### Investigation targets
**Required:**
- the spec's "Finding coverage" table and "Acceptance Criteria" R18-R20
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md`, `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/simulation-progress-design.md`, `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/documentation-evidence.md`
- `.github/workflows/gomad3.yml`, `.github/workflows/gomad3-smoke.yml` (gate definitions per platform)
- `tools/gomad3/Makefile` and the root `Makefile:165-235`
- `MILESTONES.md` sections "Open findings" and "Constraints"

### Quick commands
```bash
cd tools/gomad3
make validate
make test
cd ../..
make gomad3-runner
make gomad3-integration-test
make gomad3-smoke-qualification
make -C tools/gomad3 compatibility-pack-qualification core-qualification-set
tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim
make lint-code-fast
flowctl validate --spec fn-109-gomad-deepen-modules-and-tool-interfaces
```

### Constraints
- Follow `MILESTONES.md`: commit this task's verified progress separately, including its implementation, tests, documentation and Flow records. Root is the sole committer; preserve unrelated changes. Record unavailable source-owned Darwin gates as incomplete and keep Darwin acceptance open; retain transferred Linux obligations under fn-128.1/.4/.7. This supersedes older user-only commit instructions. Push, stash, worktree creation and history rewrites require separate authorization.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- Recheck the actual execution host before gates. This development session is `linux/arm64`, with the patched toolchain absent; neither native `darwin/arm64` nor native `linux/amd64` qualification is available here. Cross-platform source/type/vet checks and stock-host tests are developmental evidence only. Keep each required Darwin gate incomplete here until a current-source-bound result exists on darwin/arm64; fn-128.1/.4/.7 retain native Linux gates as unverified until their own source-bound linux/amd64 results exist.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.

### Canonical JSON correction evidence owner, 2026-10-05

Task41 owns one grouped terminal case covering the canonical JSON validator's 20 previously omitted reflect.Kind members, preserving its original tagged switch and seven existing branches, plus literal BASE/final characterization. The earlier expressionless-switch candidate was rejected by actual QF1002 lint; its evidence is historical RED, not a final pass. Consume the revised candidate's source, consumer, generator and integrated-lint evidence through the direct acceptance dependency. The existing visited-slice key omits length; task41 preserves and discloses that behavior rather than claiming complete UTF-8 rejection. The attempted Runner-fixture baseline at source a683e64af560322014e14f3a1ef3953b27cad96a failed before fixture execution under the linux/arm64 preparation guard, so those fixtures remain unchanged and unadmitted. Preserve every original qualification and dependency requirement.

### Compatibility-pack exhaustive correction evidence owner, 2026-10-05

Task42 owns the single FactKind exhaustive finding in compatibilitypack Selection.Evaluate under R18/R19; task11 keeps sole R17 semantic ownership. Consume its direct dependency's literal matching-pack BASE/final decisions, preserved source/pins, generator controls and actual unfiltered/integrated lint reduction. The seven inherited package findings remain outside this correction. Source admission from reviewed integrated commit 7750f4f57b84289545f6e4972cf8b6fe85c92eb3 does not imply completion of this task or any original qualification gate. Preserve every original dependency and acceptance requirement; Linux remains transferred and nonblocking under fn128.

### Target lint corrective ownership

Task43 supplies R21's separately admitted five-import policy correction. Consume
its direct dependency's rejection regressions, unchanged valid-pack decisions,
pre-publication refusal, generator and consumer evidence. Reconcile only the two
formerly erroneous plugin/cgo admissions and their loader expectations under
R21's explicit preservation exception. Task42's historical source/BASE evidence
remains valid for its former candidate. All original qualification, dependencies
and non-Linux acceptance remain in force; Linux remains under fn128.


Tasks 38 and 39 own the eight target digest/import findings and nine target cleanup findings respectively. Retain their independent source evidence in the final R18/R19 matrix without treating committed source progress as qualification. Both are direct acceptance dependencies of this final gate; neither requires task21 completion for source admission. The original final-gate requirements and historical evidence remain unchanged.
### Adapter command correction evidence owner, 2026-10-05

Task40 supplies the bounded shared-command compatibility mechanism and actual BASE process controls for task9's omitted adapter listing. Task9 retains helper integration and its task8/task40 dependencies; task21 consumes both evidence sets through the existing chain. Preserve original predecessor and qualification requirements. The 29-case stock-Go BASE probe establishes ordinary error/cancellation/cwd observations only; overflow, descendants, nonempty platform source selection, pin reproduction and all original native/full/formal/matched-first-baseline gates remain required where unproved. Historical task9 checked acceptance is not current complete R10 coverage.

### R18 cross-owner contract audit, 2026-10-05

The [recorded-format and selected-workload audit](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/r18-contract-conflict-20261005.md) distinguishes intentional v2 wire refusal under fn114.11, recorded-controller resume/refusal under fn114.12, and the genuinely selected v041 fixture retired under fn113.3. Reconcile their explicit owner contracts before commissioning contradictory restoration or counting these migrations as unconditional R18 preservation. This task still implements nothing; existing owners retain any correction. No requirement, test expectation, approval, qualification disposition or preservation waiver changes. Task20 retains the stale complete-v2 replay guidance correction. Linux remains transferred and nonblocking under fn128.

### Maintainer stdout correction evidence owner, 2026-10-07

Task46 owns the 17 unchecked maintainer-command stdout reports identified at `a6a28720af58fe65cb6fd3bc618ef8102765b50b`. Consume its direct dependency's public-command writer failures, unchanged healthy bytes and publication snapshots, deferred qualification-output error precedence, and actual lint delta. Fn-113.3 retains approval and publication ownership. This task still implements nothing. Keep every unexecuted conformance/build/discovery/qualification/dossier success sequence, original full/native-Darwin/formal/affected and matched-first-baseline requirement open. Linux remains deferred and unverified under fn128. The [source audit](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/next-gate-source-audit-a6a28720af.md) records the 302-finding full-gate baseline; correction evidence belongs under task-46.

### Source archive correction evidence owner, 2026-10-07

Task47 owns the eleven source-archive cleanup omissions and redundant legacy tar selector identified at `4a5f2cf2f84332727b44ddbc14b4844490ea9ef8`. Consume its direct dependency's real cleanup errors, primary-error/retry/publication preservation, raw legacy-header controls and actual lint delta under task-47. This task still implements nothing. Keep original native Darwin/full/formal/affected, matched-first-baseline, bounded-measurement and predecessor acceptance unchanged and open wherever unproved; Linux remains deferred and unverified under fn128. The [source audit](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/next-gate-source-audit-4a5f2cf2f8.md) records the bounded implementation recommendation.

### Patch regeneration correction evidence owner, 2026-10-07

Task48 owns the seven unchecked patch-regeneration cleanup sites identified at `101b14f882195422c31f35afc259cb25050b31e3`. Consume its direct dependency's public failure controls, retained output and callback lifetimes, and actual lint delta under task-48. This task still implements nothing. Keep every original qualification, preservation, measurement and predecessor requirement open wherever unproved; Linux remains deferred and unverified under fn128. Fn110 retains patch minimization and native regeneration qualification ownership.

### Adapter cache correction evidence owner, 2026-10-07

Task49 owns the cached-adapter preparation, publication and reuse cleanup omission identified at `8486dcb98d2b15e1f985d5bb1e79ae7da81a0d5a`. Consume its direct dependency's genuine cleanup errors, withheld modfiles/evidence, stable cache publication and retry controls, and actual lint delta under task-49. This task still implements nothing. Preserve all original predecessor, preservation, qualification and measurement requirements; Linux remains deferred and unverified under fn128. The [source audit](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/next-gate-source-audit-8486dcb98d.md) records the complete owner and rejected alternatives.
## Acceptance

Owner decision (2026-10-07). R18 recognizes only the already-approved fn-114.11 Choice Trace v3/v2-refusal and fn-114.12 controller/select-poll and controller-v2-journal-refusal migrations, verified against their actual owner contracts and retained refusal/identity controls. All unaffected matched-first-baseline/fixed-identity/default/error/feature/API/CLI preservation remains required. No new migration, golden rewrite, decoder restoration or blanket waiver is authorized; reconcile restored selected v041 separately. See [the bounded decisions](../artifacts/source-unblocking-20261007/owner-decisions.md).


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.
Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] `completion-matrix.md` maps every F1-F11 and S1-S5 to its R-ID, owning task, implementation evidence and verification evidence, with fn-108 R6/R7 linked for R2/R3 and D1-D5 each mapped exactly once.
- [ ] The preservation audit shows no feature removal, no public Go change beyond `go-interface-changes.md`, unchanged CLI inventory, preserved comments and unchanged fixed-identity canonical bytes.
- [ ] Baseline revision and inputs, commands and per-platform results are retained; darwin/arm64 gates (validation, full Gomad tiers, integration, smoke, core qualification, affected suites) pass against unchanged dispositions or each failure is attributed with evidence.
- [ ] 10-job and 100-job control cases are measured and show bounded policy state and no added full-payload copy.
- [ ] Unfinished source-owned fn-108 evidence and any D12/D14-owned failure are listed as incomplete acceptance with the command to run. Transferred linux/amd64 gates are referenced under [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md), with Linux qualification remaining unverified until native evidence exists; missing transferred Linux evidence does not keep this task's acceptance open. Nothing is reported as passing that was not run.
## Done summary
Blocked on R18 preservation reconciliation, required lint and both native platform gates.

The matched developmental 10/100 campaigns, sixteen-finding matrix and exact native-command ledger are frozen against source candidate 8604c07def0f97b63cbca3864b4c286d6803c4b1. Fresh read-only integrity checks passed. Independent evidence review found no introduced findings and recommends committing verified progress; it is not a formal SHIP or qualification verdict.

[Checkpoint and remaining acceptance](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/acceptance-open.md) retains the scope, checks and owner handbacks. [Independent review](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/current-checkpoint-independent-review.md) retains actual raw-profile and gate-ledger verification. The frozen worker handover records its earlier in-progress snapshot; current Flow state is blocked. Original acceptance criteria, native expectations and D3-D5 closure remain unchanged.

The [additive R18 accountability supplement](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/r18-accountability-2026-10-04/accountability.md) binds the named campaign helper/type set to task 5/R6 and its WIP introduction, with task 26's real CLI consumer correction. It links task 20's nine-flag correction and separately owned API/CLI changes. This supersedes only those historical inventory/ownership and missing-flag status statements at checkpoint 0988ab1041c5580b2d02763088d5048d6ee70586; the frozen audit, qualification evidence, measurements and original blocked checkpoint remain unchanged. Format/workload availability, changed guidance defaults, matched first-baseline identities, full/formal/affected-consumer and both native acceptance gates remain required and open.

stage: impl-review - deferred(policy: required lint is red; native qualification and preservation acceptance are incomplete)

stage: plan-sync - skipped(config: disabled; task remains blocked)

## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.


## Generator stderr correction owner — 2026-10-09

Task50 is a direct dependency and supplies five terminal generator stderr checked-result corrections plus source-bound preservation/analyzer evidence. Task21 consumes that evidence and implements nothing. Task46 retains its separate stdout scope. Original dependency, first-baseline, full/default/affected/functional, formal-review and native-transfer obligations remain unchanged. No source-progress receipt alone completes this task.


## Usage-status reconciliation consumer — 2026-10-09

Task21 directly depends on task51’s bounded test-only correction from reviewed source-progress commit e09187751326abf393011052dd08fdfc9af61900. The parent and task-51/usage-status-20261009/admission.md admit only the later fixture’s compatibility-pack invalid-input failed-stderr expected status 2→1; task8’s original production status is preserved. Task21 implements nothing. All original acceptance, dependencies, histories, red gates and native transfers remain unchanged.

### Remaining diagnostic correction — 2026-10-09

Task52 owns the 59 remaining unchecked maintainer stderr writes retained by fn112.10 at `eb82ea59a4`. It preserves primary outcomes, report ordering and publications and supplies its own source checks and independent review. Task21 consumes that correction and implements nothing; all other acceptance, original-base source lint, preservation and transferred native obligations remain unchanged.


## Format-compatibility amendment (2026-10-09)

Per the spec's 2026-10-09 amendment, byte-for-byte and format compatibility is no longer required. Drop the R18 audit items for unchanged fixed-identity canonical bytes, recorded-format preservation and the fn-114.11/.12 refusal recognitions as byte-preservation evidence. fn-109.35 and fn-109.41 are superseded by fn-152 and fn-153 and are not required for completion. Now also depends on fn-109.63.
