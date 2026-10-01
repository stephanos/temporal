---
satisfies: [R1]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.2 Give local and isolated campaigns one normalized options owner

## Description
Stage 1, second half of R1 (F1 options ownership). After task 1 the transport is correct but still a hand-maintained copy of 36 fields. Give the serializable campaign intent one owner that `runLocal` and the coordinator envelope both consume, so a new option cannot be dropped again.

**External ordering:** start only after the fn-108 R6/R7 tasks are done and verified: `fn-108-gomad-reduce-code-size-without-removing.5` (shared assessment, R6) and `.6` (retention and artifact-input composition, R7) in `tools/gomad3/runner`. Re-anchor the line references below against the post-fn-108 source first. flowctl cannot record a cross-spec task edge, so check `flowctl tasks --spec fn-108-gomad-reduce-code-size-without-removing` before `flowctl start`.

**Size:** M
**Files:** `tools/gomad3/runner/runner.go` (`CampaignSpec` `:126-177`, `validateConfig` `:1194-1404`), `runner/coordinator.go`, `runner/resume.go:91-110`, `runner/campaign_plan.go:40-50`, `runner/campaign_shard_execution.go:80-95`, a new private options file and its test.
**Touches:** [tools/gomad3/runner/*.go]

### Approach
- One private serializable options value grouped by invariant (target intent, search settings, resource limits, observation, retention). Callbacks (`Progress`), injected dependencies (`Preparer`, `Executor`, `Replayer`), resolved child commands (`SupervisorCommand`, `CoordinatorCommand`), `RunnerBuild` and private resume state stay outside it.
- One normalization function produces it from `CampaignSpec`; `runLocal` and `runIsolated` both use its output, and the coordinator envelope is its serialization. The two conversion functions from task 1 disappear. Strategy defaulting (`runner.go:1211-1214`), shard normalization (`:1215`) and `resumeRequestDefaults` (`resume.go`) are the normalization inputs to unify.
- `record` campaign plans (`campaign_plan.go:47-48`) and resume plans (`resume.go:103-104`) remain separate versioned contracts. Map to and from them; never make the mutable request type the recorded schema.
- Keep the exported `CampaignSpec` field set source-compatible in this task. Regrouping exported fields is a public Go change that needs the consumer inventory owned by the executor-injection task; record any regrouping wish in `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md` instead of doing it here.
- Equivalence harness: before editing, capture for a table of specs (each strategy, guided, sharded, resume, every `validateConfig` rejection) the returned `SeedSelection`, base environment, error text and the coordinator request bytes; the same table must produce identical values afterwards. Request bytes may change only by field grouping, and the test then pins the new bytes with decode equality.

### Investigation targets
**Required:**
- `tools/gomad3/runner/runner.go:126-177,362-400,1194-1450`
- `tools/gomad3/runner/coordinator.go` (post task 1)
- `tools/gomad3/runner/resume.go`, `runner/campaign_plan.go`, `runner/campaign_shard_execution.go:40-95`
- `tools/gomad3/runner/coordinator_transport_test.go` (task 1 tests; they must keep passing unchanged in intent)

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner -run 'Coordinator|IsolatedRunner|ValidateConfig|Resume|CampaignPlan|Shard'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./cmd/gomad/... ./qualification/...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep . -run 'TestPackageArchitecture|TestExactModuleEdges'
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance
- [ ] Local and isolated execution consume the same normalized options value; no second hand-written field list exists for coordinator transport.
- [ ] Callbacks, injected dependencies, resolved child commands and Runner identity are outside the serialized intent; campaign-plan and resume records keep their own schemas and bytes.
- [ ] The characterization table yields identical selections, environments, error messages and precedence before and after; task 1's real isolated-execution tests still pass for all three strategies.
- [ ] Unknown fields, trailing data and malformed coordinator requests remain rejected.
- [ ] Exported `CampaignSpec` fields are unchanged, or every change is recorded in `go-interface-changes.md` with its consumer migration.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
