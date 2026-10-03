---
satisfies: [R3]
---
# fn-113-gomad-reduce-version-pin-maintenance.2 Regenerate adapter anchors for a new module version behind an approval digest

## Description
One governed `gomadtool` command that re-derives an adapter's rewrite and digest anchors for a new exact module version (R3).

**Size:** M
**Files:** new subcommand file under `tools/gomad3/cmd/gomadtool/`, `tools/gomad3/deterministicio/adapter_rewrite.go`, `adapter_copy.go`, the regenerated `*_adapter.go` and `toolchain/version/version.json`, fixtures
**Touches:** [tools/gomad3/cmd/gomadtool/**, tools/gomad3/deterministicio/**, tools/gomad3/toolchain/version/**, tools/gomad3/internal/compatibilitypack/**]

### Approach
- Extend the existing rewrite helpers; do not add a second rewrite engine.
- Dry run: fetch the new exact version into a private cache, apply each rewrite by its existing exact-occurrence anchor, and print the changed upstream source for the rewritten files plus the proposed new anchors and an approval digest over both.
- Fail without writing when an anchor matches zero or more than one time, or a rewritten file no longer exists upstream.
- Apply: with the matching approval digest, build the complete output set in a scratch copy first: adapter constants, the version descriptor entry, pinned-digest test data, and every file `make generate` derives from them. Verify the staged set, then publish it.
- The transaction boundary includes the generated outputs. Publication takes an exclusive lock, revalidates that the checkout files it read are unchanged since staging, and records a marker so an interrupted publication is completed or rolled back by the next run. A generation failure in the scratch copy publishes nothing.
- Report libc-bound packs the change leaves stale; the pack refresh in task 3 repairs them.
- Regenerate one real adapter: the first adapted module the root `go.mod` has moved past when the task starts. If none has moved, state that and rely on the fixture.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/deterministicio/adapter_rewrite.go:17-118` — `sourceRewrite`, `anchorRewrite`, `prepareRewrittenModule`, `rewriteAdapterSource`
- `tools/gomad3/deterministicio/adapter_copy.go:144` — source inventory digest
- `tools/gomad3/deterministicio/grpc_adapter.go` — the adapter with the most anchors
- `tools/gomad3/internal/compatibilitypack/authoring/generate.go` — approval-digest pattern to mirror

**Optional** (reference as needed):
- `.flow/memory/bug/integration/profile-adapter-changes-leave-libc-2026-10-01.md`
- `tools/gomad3/deterministicio/adapter_rewrite_test.go:239-279` — module download in tests

### Key context
- fn-109 tasks 10 and 11 change the registry and the source-inventory owner (spec Open Questions 1). Check their state and build on whichever landed.
- fn-112 task 9 consolidates the adapter test family; coordinate edits to those tests.
- Changing an adapter changes target identity; `.bin/gomad` must be rebuilt before qualification.
## Acceptance
- [ ] A dry run prints changed upstream source, proposed anchors, and an approval digest, and writes nothing
- [ ] Apply with the matching digest updates constants, descriptor, and pinned test data together; a wrong digest writes nothing
- [ ] Negative fixtures: moved anchor, anchor matching twice, rewritten file deleted upstream, generation failure in staging, interrupted publication, checkout changed between staging and publication, and two competing apply operations
- [ ] After any failed or interrupted apply, the checkout holds either the old complete set or the new complete set, including generated outputs
- [ ] One real adapter is regenerated across a version bump and its workload qualifies, or the done summary states that no adapted module had moved
- [ ] Stale libc-bound packs are reported
- [ ] `make -C tools/gomad3 validate` and the `deterministicio` tests pass
## Done summary
Delivered `gomadtool adapter-regenerate` for all 15 adapter families using the existing exact-occurrence rewrite engine. Dry run downloads into a private cache and prints readable upstream changes, proposed anchors, and their approval digest without checkout writes. Approved apply stages constants, descriptor, pinned fixtures and every generated output, including the root qualification manifest, and verifies them before publication.

Publication uses an exclusive lock, exact old-byte checks and re-enumeration of the three copied input trees with the same traversal exclusions as staging. A durable old/new transaction marker recovers interrupted source and generated-output publication. Round-one review found newly added input files escaped drift validation; the regression reproduced that failure, and additions in all three trees now reject before marker/output writes. Cache-only additions still allow valid publication. Anchor zero/two matches, missing files, wrong approval, generation failure, interruptions, drift, competing applies and stale libc pack reporting have retained test evidence in `handover.json` and `fix-round-1-evidence.json`.

No adapted root dependency had moved: nine selected versions/sums match, six adapters are absent (`root-adapter-versions.json`). The authorized controlled Sprig v3.3.0 to v3.2.3 fixture exercised the real command, complete staged apply and patched-runtime qualification with two exact replays (`sprig-fixture-evidence.json`, `fixture-qualification-evidence.json`). Production adapter pins were not changed.

Focused command, deterministicio and compatibility-pack packages, validate, scoped vet, formatting and diff checks passed. Root lint exit2 is retained: its checker cannot load nested-module packages from the root. Nested lint reports pre-existing findings outside the changed adapter files and zero findings in those files; global lint is not claimed green. Independent same-family codex:gpt-6-sol:high re-review returned SHIP with R3 met and no findings; current nine source hashes and patch digest match the receipt scope. R4/R5/R6 and both-platform full qualification remain later tasks.

No files staged or committed: the user owns commits.
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (sync active=false).
## Evidence
- Commits:
- Tests: make -C tools/gomad3 validate, go -C tools/gomad3 test -tags test_dep -count=1 ./cmd/gomadtool ./deterministicio ./internal/compatibilitypack/..., go -C tools/gomad3 vet -tags test_dep ./cmd/gomadtool ./deterministicio, go -C tools/gomad3 test -tags test_dep -count=1 -run ^TestAdapterPublication ./cmd/gomadtool, Sprig controlled fixture: adapter-regenerate dryrun/apply, make validate, Gomad qualification seed7 repeat2 choices and two exact success replays, codex impl-review round2: SHIP, working-tree patch/source binding verified, git diff --check; gofmt changed adapter files: clean, root lint-code-fast: exit2 nested-module loading limitation; nested lint: pre-existing findings, zero changed adapter findings
- PRs: