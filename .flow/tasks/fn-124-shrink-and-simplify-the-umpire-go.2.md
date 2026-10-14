---
satisfies: [R2]
---
# fn-124-shrink-and-simplify-the-umpire-go.2 Remove the duplicate refinement implementation and production APIs only tests call

## Description
Implements R2. Remove model/machine.go:690-851 (Interpreter.refinement, readsAs, seen, noStutter, carrierOf, sameNamedKey, allIn) and the RefineTables-vs-Build comparison in checking.go:569-607, plus Machine.Refinement/Rejected/Transitions plumbing only that comparison reads; keep one behaviour test of the refinement rule against checker/refine.go. Remove production APIs with only test callers: checker QueryCanonical, ComposedStep.Moves/MemberMove, Table.Stuck, Realizer.ClassKey, QueryTotal, explore.RenderTrace, and testpilot PackCaseProtoJSON, Bundle.Handles, replay NewBridge/EvidenceCore/OutsideCore, campaign RunCandidate, evaluation ProfileNames, except where an open task will consume one (model/laws.go ReadLawSidecar/LawViolations is planned for fn-122.5's law lint: keep it and say so). Also drop the stale 'Package testpilot' comment in tools/umpire/lower/lower.go:1 and ownership_test's reference to the missing export/export.go. Gates as in the spec.

Kept, with the reason (recorded during implementation):
- `Machine.Transitions`: production readers besides the comparison (`claims.go`, `export/slice.go`, `export/p.go`).
- `Realizer.ClassKey`: production caller `tools/umpire/lower/lower.go:759` (method value passed to `newAdapter`); the list above was wrong to name it.
- `explore.RenderTrace`: open task fn-119.6 (todo) consumes it; its description names `tools/umpire/explore/trace.go:35` as the renderer of the page's Run history.
- `ReadLawSidecar`/`LawViolations` (model/laws.go): planned for fn-122.5's law lint.
## Acceptance
- [ ] TBD

## Done summary
Implemented R2 (Go only, behaviour-neutral). Production Go: +46/-422 lines; tests +254/-309 (base bc6da50312).

**Refinement:** Build no longer derives a refinement of its own. Removed: `Interpreter.refinement`/`readsAs`/`seen`/`noStutter`/`carrierOf`/`sameNamedKey`/`allIn` (machine.go, 210 lines), `Machine.Refinement`/`Rejected`, `interpretation.unrefined`, `binding.unevaluated`, and the RefineTables-vs-Build comparison in `checker.refinement`. `Machine.Transitions` stays because claims.go and export/ read it. Tests now read refinements through the generic check via a new `refinementOf` helper. The behaviour tests kept are the nexus and activity refinement pins, the rejected refinement, visible projection and stutters, plus internal/checker/refinement_test.go. `TestBuildAndRefineTablesMustAgree` is deleted.
- **Equivalence evidence:** before the removal, I ran a throwaway test with the pre-change sources via `go test -overlay`. Over 75 refinements in 60 Models (current IR, frozen migration inputs, and the original-baseline baselines/expected/current), the generic rows and the accept/reject result equalled Build's (25 rejected). Log: `.flow/tmp/fn124-2/equivalence.log`.
- **Goldens:** the 12 migration `semantics/*` snapshots and the 12 `semantics/*` digests in internal/golden/testdata/original/model.json were recaptured. Decoded, they are identical to the frozen ones once `Rejected` is dropped (`semantics-diff.txt`). The receipts already carry the rejection text. Declarations and refined-properties are unchanged.
- **Build behaviour change:** Build no longer fails on a refinement whose map or visible function errors. Check still reports it. Build's only production caller, `export.OpenWithin`, reads no refinement.

**Removed test-only APIs:**
- checker: `Query.QueryCanonical`, `ComposedStep.Moves` and `MemberMove`, `Table.Stuck` (test helpers compute the stuck state now)
- model: `QueryTotal` (external tests use `WithTotals`)
- testpilot: `PackCaseProtoJSON`, `Bundle.Handles` (with `cleanupBundle`), `replay.NewBridge`, `EvidenceCore`, `OutsideCore` (replay/core.go deleted), `campaign.RunCandidate` (the integration test now uses `campaign.Drive`), `evaluation.ProfileNames`

**Kept, with the reason:**
- `Realizer.ClassKey`: production caller at lower.go:759
- `explore.RenderTrace`: open task fn-119.6 consumes it
- `ReadLawSidecar`/`LawViolations`: planned for fn-122.5

**Also fixed:** the `lower.go` package comment, the `ownership_test` paths for the missing export/export.go and export_test.go, and the lifter comment naming `QueryTotal`.

**Review:** claude-opus-5-5 at high, the same family as the writer (Opus 5.5). Round 1 was NEEDS_WORK: an unrecorded keep of RenderTrace, and a typo. Round 2 was SHIP.

**Lost coverage (P3):**
- The explore-level unreproduced-reduction test became an admit test; replay's tests still cover the rule.
- The composed-step test can no longer see which members moved.
- The live control test no longer checks the evidence core.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e27f98dd8b, 5c2992e53a, 1d63723464, 2253babc40, 4fc2c3c4eb
- Tests: MODEL_GATE_ARGS=--skip-go-checks make umpire-check-model (exit 0, at 1d63723464), go test -tags test_dep -count=1 -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 372 s), make umpire-check-cases (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), go test -tags test_dep,integration -run 'TestTestpilotExplorationDiscoversUnpinnedExecution|TestTestpilotNexusControlForgedCompletionIsViolated|TestTestpilotNexusControlReplaysThroughTheCommand' ./tests/ (exit 0), pre-change overlay refinement equivalence, 75 refinements over 60 Models (exit 0)
- PRs: