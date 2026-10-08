---
satisfies: [R8, R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.32 Preserve concrete error callback provenance in architecture checks

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Bounded R8/R18/R19 repair after task31 source checkpoint c0e21c9cbb54a9d2909d0d47da081fdd44c94d6c and actual-commit evidence follow-up f931c9879e3017b346562667b9f6fbcc4db458ec. Task19/fn105 D4 remains the original acceptance owner, task21 consumes final evidence, and root owns Flow/parent/MILESTONES/review/Git/index/commits. Continue source implementation under MILESTONES delivery item4 without marking old predecessor/native requirements done.

**Size:** M
**Touches:** [tools/gomad3/internal/gomadtool/architecture/effects.go, tools/gomad3/internal/gomadtool/architecture/standard.go, tools/gomad3/internal/gomadtool/architecture/error_provenance_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-32/**]

Read AGENTS.md matching Codex routing, Gomad README/MILESTONES, original fn109 R8/R18/R19/task19 and .flow/tmp/architecture-error-provenance-owner-plan.md. That source-only note predicts three surviving omissions, supplies isolated fixture recipes and changes no acceptance criterion. Its original R13 mention is cross-scope context; this owner advances R8/R18/R19. Current effect/program/test source hashes must match the note before edits. The earlier %w/struct/interface-return and graph-key repairs already exist and are not to be redone.

Use test-driven-development/writing-good-tests, systematic-debugging, code-style and verification-before-completion. Baseline whole architecture package and actual pinned unfiltered package lint/errortype first. Add error_provenance_test.go using real fresh temporary modules and existing range/initialization fixture patterns. Keep only record.Check as pure production root; all errors, wrappers, conversions and writers live outside pure roots in allowed internal/canonicaljson helper package. No clock in initializers, eager pure closures or pure factories. Stock-host fixture runs assert literal dirty callback count1 and clean/nil/unreachable0. Analyzer Load inspects both supported metadata source sets, with empty PackageEdges and a host-effect finding containing record.Check, the specific leaf callback and time.Now on one causal path. A compile/loading/pin error or incidental/unresolved finding is not causal RED. Stock linux/arm64 execution and cross-platform metadata analysis are developmental, not qualified native execution.

A. Prove errors.Is/As lose a named slice error returned through Unwrap() error because child.elements is mistaken for multi-error containment. Dirty Leaf callback, pure Error and outer wrapper, direct/joined/%w compositions, As companion and true Unwrap() []error control. Clean callback, nil/empty children and unreachable leaf controls remain accepted. Correct signature distinction only after the corresponding causal RED. Keep errors.Unwrap's single-only behavior and original recursive wrappers/pins.

B. Prove typed parameter n int converted to helper.Leaf(n), returned as error and later Error invoked loses destination concrete methods. Add Supplier interface-return and local-replacement external-source parity controls, retaining literal callback count. Clean returned error and noninvoked dirty return remain accepted; include conversion-to-interface retaining dynamic type and callback/alias conversion controls needed by the minimal repair. A direct untyped Leaf(1) alone does not isolate the typed-source loss. Preserve destination concrete type and relevant payload/function/alias provenance without mutating unrelated aliased caller values or replacing all conversions with blank abstractions.

C. Prove fmt.Fprint discards concrete Writer.Write error return, making later Error callback escape. Include Fprintf/Fprintln same-summary variants, nil/clean/noninvoked returned-error controls and wrapped errors.Is composition with pure Error where useful. Preserve concrete writer error in existing fmt summary result as JSON summaries already do, along with result count/tuple shape and nil/error semantics. Unknown callback-return provenance must remain fail-closed; no blanket builtin-error or nil-wrapper waiver.

Before changing each production obligation retain its actual prechange causal failure with stable source and successful stock-host callback counter. If an alleged case already passes, do not modify production for it; retain evidence and refine only an actual diagnosed defect within scope. Minimal code in effects.go/standard.go only, new regression file only; existing effects/range/mutation/initialization/source tests byte-identical. No Program/public API, package classification/exclusions, graph-key/memo/fixedpoint/style consolidation, pins, dependencies, runtime/overlay, config, waived diagnostics, fixture goldens or identity changes. Preserve alias/capture/recursion and report ordering. No broader unknown-wrapper theory without established causal evidence.

Use cached stock Go1.27.1 linux/arm64 with GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOFLAGS='' and both seeds absent; tests -count=1 -tags test_dep. Go /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go; pinned lint/errortype /tmp/fn109-lint-tools.ZdNe1t50/{golangci-lint-v2.13.0,errortype}. Final whole architecture, focused new causal tests and existing summary/initialization/mutation/range/control regressions, five actual root boundaries and affected gomadtool architecture command consumer checks. Inspect generator input lists and make validate without regeneration. Actual unfiltered lint run --config absolute unchanged .github/.golangci.yml --build-tags test_dep --fix=false ./internal/gomadtool/architecture; errortype -tags test_dep same package. Record actual baseline/final diagnostics and classify every introduced/resolved/residual finding; no presumed green/historical419-as-current. Formal review only on qualified green tree; root conducts bounded fresh source review.

Retain exact command/env/cwd/start/end/elapsed/timeout/childexit/raw-log/tool/config identities and complete architecture-source pre/post map per gate. New test absent at baseline is an explicit inventory difference; exact test-first production and intermediate obligations must be reconstructible in memory from final source, or source snapshots retained once in bounded task32 artifacts, not multiple full trees. Root source-admission compact protected tracked inputs excludes only two existing production paths and the new test; all artifact task31/product/contracts and previous evidence remain unchanged. Preserve all original test/assertion bytes and production outside minimal causal changes. Lean handover.md/evidence.json plus reproducible read-only audit, no audit that rewrites immutable evidence on rerun.

Original R8/R18/R19/task19/fn105D4/predecessors/task21, full/completion/formal, Darwin patched-native and affected-consumer acceptance remain open wherever unproved; original first-baseline fixed-identity requirements stay intact. Linux native execution, pack/report/replay and qualification documentation belong to fn-128.1, fn-128.4 and fn-128.7; missing transferred Linux evidence does not block this task. One source writer current branch gomad; root alone stage/commit/Flow/MILE/review. Worker owns only admitted three source/test paths and task32 artifacts excluding root-owned admission/checkpoint/review/lifecycle. Use absolute apply_patch paths. No worktrees/stash/push/history rewrite/bridge/download/newplatform or unrelated writes. Do not flowctl done or claim formal SHIP. Root reviews and commits verified source progress before another source writer. Return only after all owned commands/delegates terminal with lean artifacts, command statuses, scope delta and explicit limits.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Each surviving omission has an actual prechange causal host-effect RED, stock-host literal callback evidence and final GREEN; clean/nil/unreachable controls remain accepted without incidental pure roots.
- [ ] Single versus multi-error Unwrap traversal, typed conversion/interface/alias provenance and fmt writer-error results retain concrete reachable callback effects with original recursive/JSON/summary behavior.
- [ ] Existing tests, package/pin/classification boundaries, public APIs, fixed-input bytes/identities, runtime/config/dependencies and unrelated source remain unchanged.
- [ ] Stable source-bound baseline/red/intermediate/final package/consumer/boundary/lint/errortype/generator receipts retain actual commands/exits/logs/tools and every residual/introduced finding.
- [ ] Fresh independent source review has no actionable introduced defect and root commits verified progress; original R8/R18/R19/task19/fn105D4/predecessors/task21/full/native/formal acceptance remains open.


## Done summary
Verified source progress only; original acceptance remains open. Task is blocked,
not done. Exact single/multi Unwrap, concrete conversions and fmt writer errors
now retain reachable callbacks. Six admitted fresh allocation alias pairs pass;
array/struct conversion copies isolate value cells and nested references still
reach Dirty -> time.Now. No Program, graph/memo, public API, pin, dependency or
runtime change; old tracked tests and 1,042 protected inputs remain unchanged.

Final worker and fresh corrective reviewer whole architecture 26 top-level tests,
focused 16, 67 stock-host fixtures / 134 metadata observations each, five actual
root boundaries, three broader purity/edge/host-vet boundaries and gomadtool
consumer 30 all pass. Source-scoped static, errortype and make validate pass without regeneration.
Actual unfiltered lint exits 1 with four byte-identical inherited findings;
introduced 0, resolved 0. Full staged diff-check exits 2 solely for the frozen
historical handover EOF blank; source and newly written root documents have clean
scoped checks. The archive stays immutable; no passing full-index quality claim.
Stock linux/arm64 is developmental; supported-platform
Load/vet is not qualified native execution.

Same original corrective reviewer: SOURCE_PROGRESS_COMMIT_ONLY, no actionable
introduced defects. Root fresh read-only review audit session87340 exited 0 at
2026-10-04 20:02:54 UTC against frozen source. The separate authorized AGENTS
research-routing delta is now committed as e95d9fa1b61951388818715677e5c42b0c03ee0f;
it changes no product source. Archived admissions/auditors remain immutable and
the review explicitly binds the historical and current document views.

The invalid unused-import RED and earlier zero-selection probes are inconclusive.
Prior concurrent-audit failure/stable retry and two writer audit-construction
failures remain retained, not relabeled as product successes. Seven supplementary
inherited controls were red on BASE; final candidate resolves them without a
whole-baseline-green claim. Inherited zero-variable, map-key formatting and
general assignment/parameter-copy limitations remain; no universal completeness.

Original R8/R18/R19, task19/fn105D4, predecessors/task21, fixed first-baseline
identities, complete/full/completion/formal and both patched-native/affected-
consumer qualification stay open. Verified source-progress checkpoint: e429f0a7230ea2d1c56c60fb49cc5cf4c81fa02d. Original acceptance remains open.

Tier: session (jev-unavailable(no_key)).
Requested writer/reviewer: gpt-6.1-sol at high; same configured Codex family.
Actual execution model metadata unknown.
stage: impl-review - skipped(policy: conductor-deferred; fresh corrective source review passed, full/native qualification remains red)
stage: plan-sync - skipped(config: disabled; task remains blocked rather than done)

Blocked:
# Task 32 acceptance remains open

The reviewed source candidate preserves concrete error callbacks through exact
single/multi Unwrap traversal, named conversions and fmt writer-error results.
Six admitted fresh-allocation alias families now keep existing shared storage;
array/struct value conversions copy value cells while nested references stay
shared. The final regression file retains all 39 prior fixture bodies as an exact
prefix and executes 67 stock-host fixtures with 134 metadata observations.

Worker and fresh corrective reviewer whole architecture, focused, actual root
public boundaries, broader purity/edges/host-vet, gomadtool consumer, errortype,
source-scoped static and generator checks pass. Actual pinned unfiltered architecture lint
remains exit 1 with four byte-identical inherited findings, zero introduced and
zero resolved. No new whole-Gomad lint count is supplied.

The stock-valid RED records twelve introduced alias failures; Git BASE passes
those aliases and fails seven supplementary inherited controls. The origin-stage
repair leaves two value-copy failures; the final candidate resolves them.
The invalid unused-import fixture and earlier zero-selection commands remain
inconclusive. Prior concurrency-audit failure plus stable retry and the two
writer auditor-construction failures remain retained. Archived evidence and
auditor bytes were not rewritten.

Fresh corrective review permits SOURCE_PROGRESS_COMMIT_ONLY with no actionable
introduced defects. Root's independent read-only audit session87340 exited 0
before the separate AGENTS-only research-routing commit e95d9fa1b6. Its narrowly
counted historical AGENTS view and exact current/index delta are explicit in
review checks; no product or original requirement changed.

Task 32 remains blocked on original R8/R18/R19, task19/fn105D4,
predecessors/task21, fixed first-baseline identities, affected consumer/integration,
complete/full/completion/formal and both patched-native qualification. Stock
Go1.27.1 linux/arm64 and supported-platform Load/vet are developmental evidence,
not native acceptance or formal SHIP. Existing zero-variable, map-key formatting
and general assignment/parameter-copy limits remain. There is no universal
length/capacity, formatting or copy completeness claim.

Evidence: [handover](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-32/allocation-repair-handover.md),
[worker](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-32/allocation-repair-evidence.json), [review](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-32/allocation-source-review.md),
[review checks](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-32/allocation-source-review-checks.json) and
[root bindings](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-32/root-source-checks.json).
## Evidence
- Commits: source-progress e429f0a7230ea2d1c56c60fb49cc5cf4c81fa02d; AGENTS-only e95d9fa1b61951388818715677e5c42b0c03ee0f is separate.
- Tests: See task-32/allocation-repair-evidence.json and allocation-source-review-checks.json for exact commands, exits, logs, tools and source bindings.
- PRs: none; no push authorized.

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
