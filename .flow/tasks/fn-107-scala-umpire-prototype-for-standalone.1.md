---
satisfies: [R1, R4, R5, R6]
---
# fn-107-scala-umpire-prototype-for-standalone.1 Review Scala specimen contracts and expected traces

Touches: [model/scalav2/README.md, model/scalav2/SEMANTICS.md, model/scalav2/specimens/**]

## Description
Review the authoring surface and executable milestones for R1 before extending framework code. Record two sketches and trace oracles in the prototype's specimen directory.

**Size:** M
**Files:** model/scalav2/README.md, model/scalav2/SEMANTICS.md, proposed model/scalav2/specimens/activity.md and nexus.md.

### Approach
- Inventory the existing vocabulary before adding declarations. Sketch signatures and source shapes without building a second DSL.
- Pin activity bad/good traces, opaque-provider replacement boundaries, and the two Nexus bad/good traces. Give each trace explicit observations and property evaluation points.
- Define the bounded domain manifest and monitor obligations, including candidate evidence ambiguity.
- Identify Testpilot primitives already available and specific missing primitives. Assign each proposed evolution to the later runtime tasks. Record feature source size and compile/lift diagnostic timing.

### Investigation targets
**Required:** model/scala/umpire/Claims.scala:22; model/scala/umpire/Machine.scala:49; model/scalav2/SEMANTICS.md; model/scala/temporal/standaloneactivity/Claims.scala; model/scala/temporal/nexuscaller/Realization.scala:68.
**Optional:** common/testing/testpilot/README.md:14; .plans/lean/UMPIRE_OUTSIDE_THE_BOX.md:237.

### Quick commands
`make umpire-check-scala` establishes the existing baseline; it does not verify new sketches.

## Acceptance
- [ ] The two sketches use existing native Scala conventions and identify any required extension.
- [ ] Trace oracles distinguish all negative controls from their corrected counterparts and expose required commitments.
- [ ] The domain manifest, passive monitor state, refinement provider interface, and evidence/capability matrix are written.
- [ ] Testpilot reuse/evolution points and authoring baseline measurements are recorded without reporting sketch review as proof.

## Done summary
Implemented the task1 Scala specimens, contracts, trace-oracle tables, finite-domain manifests, native declaration inventory, and authoring measurements in model/scalav2/specimens/{README,activity,nexus}.md, with additive links/status notes in model/scalav2/{README,SEMANTICS}.md. Existing documentation content/comments remain preserved. No framework implementation or proof claim is included. Proposed E1–E12 extensions are explicitly unsupported sketches; native supported blocks received scratch sanity checks.

Tier: session (jev-unavailable(no_key)).
Stage: implement - ran (requested opus at high via foreground Claude CLI; actual model claude-opus-5-5; delegated: 1 read-only Explore inventory). Bridge command, prompt, output, metadata and stderr are retained as task1-bridge-* under .flow/tmp/fn-107/.
Stage: impl-review - ran (codex:gpt-5.6-sol:high; actual SHIP at 2026-09-30T18:11:34.762729Z; session 01a0f36a-9da5-7572-ad66-95e41c3a1263). Five introduced findings were fixed and independently rereviewed in the same native session: finite redelivery budget, controlled versus uncontrolled evidence, honest semantic-build outcomes, explicit recovery commitments, and both accepted/retained acknowledgement obligations. Provider loss now requires discarded durable sender custody before receiver/admission custody; retained acknowledgement requires transferable delivery obligation, not storage alone.

Validation:
- Baseline make umpire-check-scala: rc0, 45.270s, baseline-scala.log. Baseline focused Go: rc0, 6.471s, baseline-go.log.
- GOFLAGS=-tags=test_dep make umpire-check-scala: rc0, 38.332s, verify-scala.log and verify-metadata.json.
- mise exec -- go test -tags test_dep ./model/scalav2/... ./model/go/...: rc0, 1.342s (cached), verify-go.log and verify-go-metadata.json.
- Exact native Scala blocks extracted to .flow/tmp/fn-107/sketch/{Admission,CloseReset}.scala. Final scratch check-r2-final.log records compilation/package, five lifts/load checks, four successful semantic builds and staleAdmission's expected refinement error, 35 query answers, and focused third-delivery/acknowledgement-custody controls. Current native blocks measure 173/119 and 209/144 total/code lines.
- task1-doc-check.json: 13 local links/anchors, balanced fences, preservation of all original README/SEMANTICS lines, 35 query observations and task diff whitespace passed. Global gate classification was full because it sees the extensive preexisting checkout changes; no docs-only skip receipt was claimed.

Review delivery and provenance:
- HEAD/base remains bbe765d70a708b6776c298aa5c0418bbd514048c; commits []. Never staged, committed, switched branch, created worktrees, reset, stashed or reverted edits.
- Review used task1-review-adapter.py, an in-memory compatibility adapter for installed commits-only Flow review. Plugin sources, Git history/index and native model/backend/reservation/verdict handling remain unchanged. It supplies immutable task-only pre-edit/current-file diff and current-file hashes, explicitly tells the reviewer the committed range is empty, and rejects scope/hash drift.
- Final immutable artifact: .flow/tmp/fn-107/task1-review-r2/{manifest.json,task.diff,before,after}; diff SHA256 5a7a8d96278a3572127f8211d50baf1e3703bbdbe2d0e53bedc1f7bcf64a4313. Native artifact identity 3f618d202df4e6fa04f4b86fc2524cea4a41e8b937bca03adcb0a12e590cf860. Exact five-file scope/current hashes are in manifest.json. Current file/diff hashes were checked unchanged after SHIP and immediately before done.
- Actual receipt: /tmp/impl-review-receipt-8f37faba39e2-fn-107-scala-umpire-prototype-for-standalone.1.json; copied verbatim to .flow/tmp/fn-107/task1-final-review-receipt.json. Native output: task1-review-r2-output.json. SHIP reservation d6950aa3774742b88408ad1e235d0064.
- task1-review-attempts.json retains the native journal: first outer-wrapper timeout at600s was a transport failure, correctly refunded (ef2e7e645da7454580ea25363a80f2d8, no verdict/round consumed); 1800s native-bound retry produced NEEDS_WORK, followed by actual same-session SHIP. No self-verdict/counter reset or overlapping review. R1 artifacts, actual outputs, merge plan and stderr remain retained.
- Pre-edit anchors: preedit-{README.md,SEMANTICS.md,staged.txt,unstaged.txt,status.txt}; baseline-metadata.json. The task-file staged diff still matches preedit-staged.txt byte-for-byte.
- Flow fix-loop memory captured: .flow/memory/bug/integration/realized-controls-constrain-admission-2026-09-30.md; task1-memory-result.json.

Concrete handoff for task2:
- Reuse the existing declarations in model/scala/umpire/ and native specimen blocks; do not build a parallel DSL. The specimens README maps E1–E12 to their smallest declaration/IR/checker needs and inventories Testpilot Case/Program/Contract/Profile/Driver/Prepare/Run/Evaluate.
- activityProduct already lifts. activityProtocol fails at existing model/scala/temporal/standaloneactivity/Model.scala:263 because ProtocolFact* produces scala.collection.immutable.Seq without a finite catalog. Preserve Product semantics while extending Protocol lifting for explicitly bounded collections; unsupported/undeclared finite bounds must diagnose before Go checking. Retain diagnostic/log in measure/measure.log. Named because arguments, inferred min, local pattern-bound identifiers and Claims lifting are concrete additional gaps.
- Existing Nexus lift byte-matches checked-in IR. Existing activityProtocol refinement has 240 public-visible stutters of552; extensions must state projection rules rather than silently break this baseline. The lifter currently handles temporal/ only and returns an empty IR for a nonexistent root; Int range bounds apply to all record fields.
- F1: Scala CLI1.17.1 with -Werror emitted a missing-exhaustiveness error yet returned0 and packaged a jar; ordinary type error returned1. run.sh is outside task1 Touches; parent must assign the appropriate later owner for compiler diagnostic validation. The specimen records the limitation honestly.
- Future runtime tasks: SDK activation task13, neutral optional assessment seam task12, adapters task7, delivery/admission evidence task10, lowerers task6. CallerClosePolicy runtime remains design-only; existing Nexus runtime controls supply the eventual replay loop.

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: make umpire-check-scala (baseline, rc0), mise exec -- go test -tags test_dep ./model/scalav2/... ./model/go/... (baseline, rc0), GOFLAGS=-tags=test_dep make umpire-check-scala (post-change, rc0), mise exec -- go test -tags test_dep ./model/scalav2/... ./model/go/... (post-change, rc0), Native Scala scratch compilation/lifting and 35 query checks: .flow/tmp/fn-107/sketch/check-r2-final.log (rc0, expected staleAdmission refinement failure), python3 .flow/tmp/fn-107/task1-doc-check.py (passed)
- PRs: