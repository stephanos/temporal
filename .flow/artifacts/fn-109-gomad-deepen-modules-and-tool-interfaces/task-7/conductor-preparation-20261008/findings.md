# fn-109.7 source-acceptance preparation

Read-only scout: `/root/prep_fn1097_source`, returned 2026-10-08. Base:
`ca6fd855868fac364b69cb31394c87ad2912e623`. Requested research model:
gpt-6-astra/high; no actual-model metadata was supplied. Tier selector was
unavailable (`no_key`). No gates, edits or lifecycle changes were authorized.
This is dispatch preparation, not source review or acceptance. fn-113.2–.4
still precede fn-109.7.

## Current binding

The scout verified five of eight files still match task-7 `final-source.json`:
preparation_test.go, preparation_equivalence_legacy_test.go,
preparation_owner_test.go, deterministicio.go and portable_plan.go. The three
changed files are preparation.go (task-8 StageReview addition), runner.go
(subsequent executor/completion/artifact work) and architecture_test.go
(extracted ownership and both-source-set checks). Downstream installation,
capability-review and adapter-cache dependencies changed too. Rebind the actual
candidate at implementation; historical matching caller files alone do not
prove current end-to-end equivalence.

## Gaps that change dispatch

- `internal/preparation/preparation.go:54` owns selection, preparation,
  adapter attachment and validation. Custom preparation bypasses selection
  and receives an empty nonnil adapter slice. `runner/runner.go:547` and
  `runner/portable_plan.go:123` call the owner. Retain those behaviors.
- The ordinary owner tests do not skip on Linux/arm64: profile validation
  rejects before several success, adapter-sum/conflict, validation and cleanup
  assertions. Missing driver is a separate later dependency. Test presence
  is not execution evidence. Add portable interface coverage using supported
  source profiles and controlled build inputs while retaining production host
  refusal and real identity validation. Any necessary private composition seam
  needs exact admission; no public host override or global-hook mutation.
- Cleanup owns only its unique `.adapter-work-*` directory. Add adapter-error
  cleanup, caller-owned sibling preservation and two preparations sharing one
  durable root. Existing independence tests use separate roots.
- Fresh/cache labels lack explicit empty-cache/build-count checks. Retain
  actual cache-state evidence for those claims.
- Current Runner failure selector is
  `TestRunPreparationFailureLeavesClassifiedPartial` at runner_test.go:1508;
  custom-preparer errors precede profile validation and are portable. Mutation
  coverage is `TestRunRejectsPreparedTargetMutationBeforeFailurePublication`.
- Lower-level registry sum/replacement controls support, but do not replace,
  preparation-interface coverage. Inspect unfiltered affected-package lint,
  architecture/exact edges/public signatures/external consumers, both supported
  static source sets and generated validation on the eventual frozen candidate.

## Preserve the first baseline

All following artifacts are in the parent task-7 artifact directory. The scout
verified the four original preimages still match `preimages.json` and
`task-only.patch` still matches its manifest. Both
`equivalence/{before,after}/snapshot.json` and
`equivalence-adapter/{before,after}/snapshot.json` remain identical within each
pair. They are historical Darwin bindings, normalizing only Prepared.Path.

The baseline probe spells the original four-stage protocol; it does not invoke
original Explore. `legacy-portable-overlay.log` separately records an actual
old-caller missing-root failure, with obsolete absolute overlay paths. Preserve
original bodies and snapshots. Add matched original/current caller controls
with identical fixed inputs and explicit cache evidence; recover compatible
original dependencies if necessary rather than rewriting a weaker baseline.
Keep protocol equivalence, caller behavior and native execution distinct.

The retained full-host log has 46 package-ok lines but no captured make exit;
the initial watchdog failure is transcript-only. Neither is a current pass.
Current independent source review remains required. Native qualification stays
deferred and unverified under fn-149/fn-128.
