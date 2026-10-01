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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
