---
satisfies: [R2, R5]
---
# fn-114-gomad-correct-search-path-defects-and.3 Bind environment and tick policy into the corpus identity and reject coverage-instrumented provenance

## Description
The two small identity corrections (R2 for C1, R5 for C4). They share no file and are combined because each is a single check with its tests.

**Size:** M
**Files:** `tools/gomad3/runner/internal/corpus/model.go`, `corpus.go`, `corpus_test.go`, `guide_test.go`, `tools/gomad3/runner/guidance.go`, `tools/gomad3/runner/runner_test.go`, `tools/gomad3/target/target.go`, `target_test.go`
**Touches:** [tools/gomad3/runner/internal/corpus/**, tools/gomad3/runner/guidance.go, tools/gomad3/runner/runner_test.go, tools/gomad3/target/target.go, tools/gomad3/target/target_test.go, tools/gomad3/runner/replay_operation.go]

### Approach
- C1: add the recorded target environment to the corpus identity as a digest of a canonical projection. `Identity` is compared with `!=`, so it must stay a comparable struct; a slice field will not compile at `corpus.go:284`.
- Project the same entries Campaign and Artifact identity bind, and leave out the per-execution entries (seed, I/O profile, choice profile) so the identity is equal across seeds. The clock-tick policy is carried as an environment entry: strict is the absence of the entry and forward is its presence, so the projection distinguishes them without a separate field.
- `openGuidance` already receives the base environment; pass it into both identity constructors.
- Raise the corpus schema string. A corpus written under the previous schema, or under a later one, is rejected at open with the schema error, before the identity comparison.
- C4: add a coverage check to `validateDeterministicBuildInfo` with its own error text, and a row in the existing table test. Confirm with a binary built by the pinned toolchain which build settings `-cover` and `go test -cover` record; if a coverage build can omit the setting, detect it another way and say how.
- Check whether the artifact-side build-info validation needs the same rejection, and add it if a replayed or minimized target could otherwise bypass the check.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/runner/internal/corpus/model.go:15`, `:37-47`, `:98-147` — schema constant, identity, projection
- `tools/gomad3/runner/internal/corpus/corpus.go:259`, `:284-285` — schema check and changed-identity error
- `tools/gomad3/runner/guidance.go:26-35` — identity construction with the environment in scope
- `tools/gomad3/record/identity.go:9-37`, `:229-241` — how Campaign and Artifact identity project the environment
- `tools/gomad3/record/validation.go:510-556` — tick-policy entry and reserved environment entries
- `tools/gomad3/target/target.go:812`, `:828-851` — provenance validation
- `tools/gomad3/target/target_test.go:774-810` — negative table to extend

**Optional** (reference as needed):
- `tools/gomad3/runner/internal/corpus/corpus_test.go:59-93`, `:227` — identity-change test and helper to mirror
- `tools/gomad3/runner/replay_operation.go:571` — artifact-side build-info validation
- `tools/gomad3/runner/runner_test.go:351` — guided run test that opens a corpus

### Key context
- fn-105 task 30 (target `--env`) is done, so C1 is reachable today.
- fn-109 task 12 edits `corpus.go` and tasks 9 and 10 edit `target/target.go`; rebase onto whichever landed.
- Documentation for both changes is written in task 14.

## Acceptance
- [ ] Opening a corpus with a different `--env` entry fails with the changed-identity error
- [ ] Opening a corpus recorded under strict ticks with the forward policy fails with the changed-identity error, and the reverse
- [ ] The identity is equal across seeds of one campaign configuration
- [ ] A corpus with the previous schema string and one with an unknown later schema string are each rejected by schema
- [ ] `exec` provenance for a coverage-instrumented binary is rejected with a coverage-specific error; a table row covers it and a build without coverage still passes
- [ ] The build settings a coverage build records under the pinned toolchain are stated in the test or its comment
- [ ] `go -C tools/gomad3 test -tags test_dep ./runner/... ./target/...` and `make -C tools/gomad3 validate` pass


## Done summary
Implemented fn-114 task 3 (R2/C1 and R5/C4) on darwin/arm64, leaving all changes uncommitted. Corpus schema is now `gomad3.guide-corpus/v2`. Both constructors and retained-case validation bind a canonical environment digest, excluding seed/I/O/choice entries while preserving user environment, UTC, tick policy, and diagnostic profile. Runner normalizes base environment with its existing `environmentForSeed` contract. Existing value equality fixes the reproduced same-configuration choice corpus reopen defect; `Identity` remains comparable. Previous/future schemas are rejected before typed canonical decoding, and current-schema malformed, noncanonical, unknown-field, and duplicate-key inputs still fail closed.

Coverage-instrumented exec provenance and retained replay/minimization targets now fail with coverage-specific errors through one shared build-info predicate. Actual pinned `go build -cover` and `go test -c -cover` binaries both record `-cover=true`; the ordinary build omits the entry. All three binaries are retained, along with standalone covered artifacts and verify-only/minimization rejection receipts. No toolchain/runtime source changes, CLI/record contract changes, dependencies, commits, staging, pushes, cache deletion, or rebuilds were made. Documentation is owned by task 14.

Final frozen verification passes: complete Runner/target suites (125.95s), `make -C tools/gomad3 validate`, standard host gate (45 fresh packages, 127.79s, without `GOMAD3_STOCK_GO` override), focused vet, formatting, and whitespace. The host gate includes the existing diagnostic/watchdog replay suites. Root lint remains unavailable: its default revision `main` is absent; the HEAD workaround exits through nested-module discovery, including the retained fixture module. Linux remains unverified.

Meaningful red regressions are retained in `regression-red`, `additional-red`, and `actual-layout-schema-red`. `focused-extended` first failed on an unsorted admission fixture; `case-admission-green` despite its historical filename failed on a mistaken expectation that admission validates before replay. Both fixture expectations were corrected, with no production change for them. `retain-covered-artifact` first failed to compile its standalone evidence probe because it used the wrong record-hash field, then passed after that probe repair. The initial 115.91s Runner/target run overlapped schema source/test edits and is informational only; it does not support final acceptance. Original freeze/patch, overlap note, and logs are preserved.

`task-only.patch` reconstructs all 11 owned files from actual before copies. The declared replay overlap adds only coverage validation to the prior diagnostic replay implementation; all other 69 protected source files are unchanged. Parent reconstruction receipt `parent-patch-validation.json` is bound below. Final sources remained unchanged throughout every final gate. Parent owns independent review and Flow completion; no work beyond task 3 is claimed.

Independent implementation review: SHIP, gpt-6-astra high, session 01a0fcf7-4b05-7d83-b588-6920d3dc6af1, 2026-10-02T14:16:56.134866Z. R2 and R5 met; no introduced or pre-existing findings. Parent verified all 103 evidence bindings, 21 binary/artifact bindings, 11 frozen sources, and exact task-patch reconstruction.

stage: impl-review - SHIP (codex:gpt-6-astra:high, round 1)
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: wave - sequential(shared dirty checkout; user forbids worktrees)
Tracker sync: n/a (bridge inactive). User retains commit ownership.
## Evidence
- Commits:
- Tests: tools/gomad3/.toolchain/bin/go -C tools/gomad3 test -tags test_dep -count=1 ./runner/... ./target/..., make -C tools/gomad3 validate, make -C tools/gomad3 test-host, tools/gomad3/.toolchain/bin/go -C tools/gomad3 vet -tags test_dep ./runner/... ./target/..., /Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/gofmt -l tools/gomad3/runner/internal/corpus/model.go tools/gomad3/runner/internal/corpus/corpus.go tools/gomad3/runner/internal/corpus/corpus_test.go tools/gomad3/runner/internal/corpus/guide_test.go tools/gomad3/runner/guidance.go tools/gomad3/target/target.go tools/gomad3/target/target_test.go tools/gomad3/runner/replay_operation.go tools/gomad3/runner/guidance_identity_test.go tools/gomad3/runner/coverage_replay_test.go tools/gomad3/target/coverage_test.go, git diff --check
- PRs: