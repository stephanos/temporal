---
satisfies: [R2, R3]
---
# fn-108-gomad-reduce-code-size-without-removing.2 Remove unused internal helpers, decimal/decoder, minimizer serialization and duplicate canonical validation

## Description
Stage 1a (R2, R3): delete the enumerated unreachable private code and the second canonical re-encoding in two readers. All deletions; no new helpers. Planning re-anchored every candidate at `d4d800fb47`: each symbol exists and has no caller in source, tests, templates, generators, specs or plans under `tools/gomad3`, `tools/gomad3sim`, `tools/gomad3integration`, `tests/gomadfunctional`, `.plans`. Repeat that check yourself before each deletion (R2 requires it) and keep anything that turns out to be live.

**Size:** M
**Files:** `tools/gomad3/target/capability.go`, `tools/gomad3/qualification/set/set.go`, `tools/gomad3/runner/deterministicio.go`, `tools/gomad3/runner/campaign.go`, `tools/gomad3/runner/internal/campaign/campaign_journal.go`, `tools/gomad3/deterministicio/domain.go`, `tools/gomad3/runner/internal/minimizer/minimizer.go`, `tools/gomad3/runner/internal/minimizer/minimizer_test.go`, `tools/gomad3/runner/portable_plan.go`, `tools/gomad3/runner/internal/campaign/merge.go`
**Touches:** [tools/gomad3/target/capability.go, tools/gomad3/qualification/set/set.go, tools/gomad3/runner/deterministicio.go, tools/gomad3/runner/campaign.go, tools/gomad3/runner/portable_plan.go, tools/gomad3/runner/internal/campaign/campaign_journal.go, tools/gomad3/runner/internal/campaign/merge.go, tools/gomad3/runner/internal/minimizer/**, tools/gomad3/deterministicio/domain.go]

### Approach

Unused helpers (delete the function only; keep the named live neighbour):

| Symbol | Location | Keep |
| --- | --- | --- |
| `validateGoCapabilityClosure` | `target/capability.go:184` | `reviewGoCapabilityReview`, `validateCapabilityReview` (`:477`), `ReviewCapabilityClosure` (`:195`) |
| `matchesExpectation`, `firstReplay` | `qualification/set/set.go:902`, `:924` | `matchesSupportedExpectation` (`execution.go:131`), `matchesUnsupportedAnalysis` (`execution.go:105`) |
| `deterministicCapturedInputs` | `runner/deterministicio.go:42` | `recordedCapturedInputs`, `recordedCapturedInputLimits`, `deterministicCapturedInputLimits` (three live callers: `resume.go:69`, `campaign_shard_execution.go:57`, `portable_plan.go:279`) |
| `orderRunCompletions` | `runner/campaign.go:5` | `orderShardRunCompletions` (`:9`) |
| `removeCompletedPartial` | `runner/internal/campaign/campaign_journal.go:583` | `removeCompletedPartialContext` (`:587`) |
| `decimal` type + `MarshalJSON`/`UnmarshalJSON`, `decodeCanonicalJSON` | `deterministicio/domain.go:42-70`, `:85-108` | `canonicalJSON` (`:72`, used by `bootstrap.go:30` and `profile.go:190`) and `validateCanonicalStrings` (`:110`). Do not replace `canonicalJSON` with the shared encoder: key order differs (spec "Edge Cases"). |
| minimizer `Encode`, `Decode` | `runner/internal/minimizer/minimizer.go:121`, `:128` | `Validate`, `seal`, `stateIdentity`. In `minimizer_test.go:72-82` remove only the encode/decode round-trip lines; the budget/stop-reason assertions above them stay. |

Remove imports that become unused. `firstReplay` is used only by `matchesExpectation`.

Duplicate canonical validation (R3): `canonicaljson.DecodeCanonicalJSON` (`internal/canonicaljson/canonical.go:62`) already strict-decodes, re-encodes and compares bytes. Two readers repeat that immediately afterwards:
- `runner/portable_plan.go:220-223` — delete the `CanonicalJSON(document)` + `bytes.Equal` block; the schema/mapping/strategy/guidance/shard check at `:224` and everything after it stays.
- `runner/internal/campaign/merge.go:445-446` — drop only the `canonical`/`bytes.Equal` terms; the schema, schema-version, plan-hash, selection and count terms stay in the same condition with the same error.

In both, the first decode already rejected noncanonical bytes, so the removed branch was unreachable and the error a caller sees is unchanged. These were the only two sites a search for "DecodeCanonicalJSON followed by CanonicalJSON+bytes.Equal" found; do not touch other readers.

### Investigation targets

**Required:**
- each location in the table above
- `tools/gomad3/internal/canonicaljson/canonical.go:62-74`
- `tools/gomad3/runner/portable_plan_test.go`, `tools/gomad3/runner/internal/campaign/merge_capacity_test.go` — existing reader coverage; add a noncanonical-input rejection case for either reader only if none exists

### Quick commands

```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go build ./...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go vet -tags test_dep ./target ./qualification/set ./runner/... ./deterministicio
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep . ./target ./qualification/set ./runner/... ./deterministicio ./internal/canonicaljson
```

### Key context

`go vet`/the compiler will not flag unused unexported functions; the caller check is a repository search across Go, templates (`*.tmpl`), generators under `internal/gomadtool/generation`, `SPEC.md`, `ARCHITECTURE.md`, and `.plans/`.

### Standing constraints (every fn-108 task)

- The user owns commits: no `git commit`, `git add`, `git stash`, and no worktrees. Leave changes in the working tree and report the paths.
- No new dependencies (Go modules or external tools). `tools/gomad3` is a nested module pinned to go1.27.1; `tools/gomad3sim` and `tools/gomad3integration` belong to the root module.
- Preserve existing comments: keep them with the logic they describe when code moves, and delete a comment only together with the dead code it documents. Do not compress formatting.
- Public Go names/signatures/fields/defaults, CLI commands/flags/exit statuses, schemas, canonical bytes, `HostError.Reason` values and failure precedence stay unchanged (spec "API Contracts", R8).
- Host is darwin/arm64. linux/amd64 gates cannot run here: list them as "not run (no host)" in the evidence, never as passed.
- Focused tests run from `tools/gomad3` as `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep <packages>`. `.toolchain/bin/go` is the patched toolchain; `make -C tools/gomad3 toolchain` rebuilds it (needs go.dev access). New test assertions use `require` with whole-value equality.
- Evidence (commands, platform, results, remaining failures) goes under `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/`. A defect found on the way is recorded for its existing owner, not fixed here.

## Acceptance
- [ ] Each listed symbol is removed only after a recorded repository-wide caller search (source, tests, build tags, templates, generators, specs); any symbol found live is kept and reported instead.
- [ ] `canonicalJSON`/`validateCanonicalStrings` in `deterministicio/domain.go`, and minimizer `Validate`/`seal`/`stateIdentity` with their budget and stop-reason assertions, are unchanged.
- [ ] Portable-plan and merged-campaign readers call the canonical decoder once and keep every schema/identity/mapping/count/capacity check; noncanonical, malformed, trailing-data and unknown-field inputs and invalid protocol identities are still rejected with the same error classification.
- [ ] Focused build, vet and tests above pass on darwin/arm64; `TestPackageArchitecture` and `TestCanonicalJSONHasOnePrivateOwner` pass.
- [ ] Evidence note lists removed symbols with line counts; nothing staged or committed.


## Done summary
Removed the enumerated unreachable private code and the second canonical re-encoding in the portable-plan and merged-campaign readers. Nothing is staged or committed; the change is the working-tree diff recorded as `task2.diff` under `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/`, with the full record in `task2-evidence.md`.

Removed after a whole-repository consumer search (`task2-consumer-search.txt`): `validateGoCapabilityClosure`, `matchesExpectation`, `firstReplay`, `deterministicCapturedInputs`, `orderRunCompletions`, `removeCompletedPartial`, the deterministic-I/O `decimal` type with its two methods, `decodeCanonicalJSON`, and minimizer `Encode`/`Decode` with the round-trip lines of their test. No symbol had a live consumer, so none was kept. `canonicalJSON`, `validateCanonicalStrings`, minimizer `Validate`/`seal`/`stateIdentity` and the budget and stop-reason assertions are unchanged.

Readers: `openCampaignPlan` and `OpenMergedCampaign` now validate canonical bytes once, through `canonicaljson.DecodeCanonicalJSON`. The removed re-check was unreachable because no reachable type has a pointer-receiver marshal method. Every schema, identity, mapping, count and capacity check stays. `OpenMergedCampaign` returns `errors.New("merged campaign record is invalid")` where it returned `errors.Join` of the same text and an always-nil error; message and `errors.Is`/`errors.As` results are the same.

Tests added outside the Touches list, as the task's investigation targets direct because neither reader had a noncanonical-input case: `TestOpenCampaignPlanRejectsNoncanonicalAndInvalidDocuments` (`runner/portable_plan_test.go`) and `TestOpenMergedCampaignRejectsNoncanonicalAndInvalidRecords` (`runner/internal/campaign/merge_capacity_test.go`). Each covers noncanonical, trailing-data, unknown-field, malformed and invalid-identity input (R3 error cases) with exact error text. Replacing the remaining decoder call with `StrictDecode` turned the noncanonical cases red; the mutation was reverted.

Size (rule v2, this task's files): production Go -139 physical / -127 code lines / -4110 code bytes; test Go +61 code lines. Overlay, generated, protocol-input and other classes unchanged; `size-compare.sh` exits 0. Public `go doc` and CLI help capture diff against `api-baseline/`: empty.

Gates on darwin/arm64, all exit 0: focused vet, focused tests (14 packages), `TestPackageArchitecture`, `TestCanonicalJSONHasOnePrivateOwner`, `make -C tools/gomad3 validate`, `make -C tools/gomad3 test-host` (45 packages ok, run once with fn-108.4's in-progress edits present). linux/amd64: not run (no host).

baseline: red (`go build ./...` failed pre-edit). `./...` matches `toolchain/runtime/overlay`, which builds only inside the patched GOROOT; the failure is identical after the change. The host package set builds with exit 0. Follow-up for the fn-108 task text owner: replace that Quick command in the remaining tasks.

Follow-up, not done here: `TestStateStopsAtAttemptBudgetAndRoundTrips` keeps its name although it no longer round-trips, because the task removes only those lines and the baseline disposition table is keyed by test name.

stage: impl-review - ran (raw codex bridge on working-tree diff; commits forbidden) (model: gpt-5.6-sol) [round 1 NEEDS_WORK 1 finding (evidence note absent), round 2 SHIP with 1 NIT applied unreviewed; record in task2-review.md]
stage: plan-sync - skipped(config: planSync.enabled != true)

GATE_SKIPPED lines: none.
## Evidence
- Commits:
- Tests: baseline: red (go build ./... failed pre-edit; ./... matches toolchain/runtime/overlay; same failure post-edit), darwin/arm64: .toolchain/bin/go build <host package set> exit 0, darwin/arm64: .toolchain/bin/go vet -tags test_dep ./target ./qualification/set ./runner/... ./deterministicio exit 0, darwin/arm64: .toolchain/bin/go test -tags test_dep . ./target ./qualification/set ./runner/... ./deterministicio ./internal/canonicaljson exit 0 (14 packages ok), darwin/arm64: go test -run '^(TestPackageArchitecture|TestCanonicalJSONHasOnePrivateOwner)$' . exit 0, darwin/arm64: make -C tools/gomad3 validate exit 0, darwin/arm64: make -C tools/gomad3 test-host exit 0 (45 packages ok), size-compare.sh exit 0 (this task: production-go -127 code lines), api-capture.sh diff -r against api-baseline: empty, linux/amd64: all gates not run (no host)
- PRs: