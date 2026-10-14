---
satisfies: [R2, R3, R9]
---
# fn-107-scala-umpire-prototype-for-standalone.2 Extend finite model IR and TASTy lifting

Touches: [proto/internal/temporal/server/api/modelir/v1/**, api/modelir/v1/**, model/scalav2/lifter/**, model/scalav2/SEMANTICS.md, model/scalav2/gen.sh, model/scalav2/run.sh, model/scalav2/ir/nexus-caller.json, model/scala/umpire/**]

## Description
Extend model admission and lifting for the reviewed bounded contracts, passive monitors, composition, refinement interfaces, and explicit support gaps. Scenario execution lowering stays in its own task.

**Size:** M
**Files:** modelir ir.proto and generated outputs; Lift.scala; native framework declarations only where the reviewed declaration inventory requires extension; SEMANTICS.md; lifter diagnostic fixtures; existing generation/lift gates where required.

### Approach
- Inventory source constructs needed by both specimens before extending schema. Preserve stable definition identity independently of source locations. The task-1 handoff identifies Action/Domain/Refine and small new framework declaration types as possible seams; the framework Touches includes them so no parallel declaration layer is needed.
- Express first-class passive monitor state, typed channels, visible-result projection, assumptions, and explicit holes in the portable finite subset. Retain native functions and guarded transitions.
- Preserve disabled actions and existing Nexus artifact semantics. Add reference/type/version/bounds rejection fixtures through ordinary lifting/admission. Regenerate the existing Nexus IR only for the source provenance correction removing the leaked CLI warning flag; compare all remaining declarations and derived tables/IDs/fingerprints unchanged.
- Regenerate Go and JVM protocol classes with existing gates. Use existing checked Nexus IR as the behavior pin for unaffected declarations.

### Investigation targets
**Required:** proto/internal/temporal/server/api/modelir/v1/ir.proto:19; model/scalav2/lifter/Lift.scala:392; model/scalav2/gen.sh; model/scalav2/goir/diagnostics_test.go; model/scala/umpire/Claims.scala.
**Optional:** model/scalav2/ir/nexus-caller.json; model/scalav2/run.sh.

### Quick commands
`make protoc`; `make umpire-gen-scala`; `make umpire-check-scala`; `make lint-scala`.

## Acceptance
- [ ] Both reviewed model/monitor surfaces lift without handwritten IR construction.
- [ ] Valid union/presence and bounded channel cases round-trip; rejection fixtures identify source declarations.
- [ ] Existing Nexus tables, identities, and supported fingerprints retain their declared comparison behavior.
- [ ] Required protocol generation and focused Scala gates pass.

## Done summary
Extended the existing finite modelir/native Scala framework and TASTy lifter for the reviewed bounded model/monitor surfaces: Option/presence, typed channels, passive monitors, Properties/Queries, composition/provider interfaces, fact/outcome visibility, assumptions/progress/holes and named bounds. Both surfaces compile/lift from source, with no handwritten feature IR: Admission has 3 machines/2 monitors/14 Queries; CloseReset has 3 designs/2 monitors/21 Queries. Existing activity Product and Protocol source now lift. Runtime/DAG/descriptor execution and production CallerClosePolicy remain outside this task.

Tier: session (jev-unavailable(no_key)).
Stage: implement - ran (requested opus/high via worker-owned foreground Claude CLI; actual claude-opus-5-5; delegated: 0; session 8ceed6d1-8f5d-489d-8886-0079fb3181d1). Three bridge runs exited 0: 6114.401s, 704.817s, 626.679s; prompts, commands, outputs and metadata are task2-{bridge,followup,reviewfix}-*.
Stage: impl-review - ran (actual codex:gpt-5.6-sol:high SHIP at 2026-09-30T21:02:53.559556Z, same session 01a0f40a-b51c-7041-995a-afecfaa5e7a0). Three introduced findings were reproduced/fixed: visible stutter outcomes, finite channel message catalogs, and computed order/loss coercion. Native receipt marks all three fixed; no surviving findings.

Final validation (all rc 0; exact commands/durations/logs in task2-gates.jsonl and task2-logs/):
- make protoc with pinned Darwin LOCALBIN: 72.79s; all 338 captured API/CHASM bytes identical afterward, no new files.
- GOFLAGS=-tags=test_dep make umpire-gen-scala: 55.13s; make umpire-check-scala: 65.65s; default make lint-scala: 48.11s, no exclusion override.
- mise exec -- model/scala/run.sh --no-prove: 20.38s; 68 tests/9 suites. Pinned mise PATH fixes the host protoc32.1 versus protobuf4.29.5 mismatch without source edits.
- Focused Go tests (-tags test_dep,-count=1) and vet: 6.14s; actual scoped make lint-code(api/modelir/v1), pinned Darwin linters: 34.97s.
- Six source-generated JSON/binary wire roundtrips; 16 refusals (15 source-located), 342 stored source positions, line-independent identities/unknown-root mutation controls, and original-comment audit pass. Negative fixtures remain readable .scala.fixture text, materialized and cleaned by run.sh.
- Nexus changes only source provenance, removing the leaked CLI flag. All other IR fields, four supported tables, IDs, refinement/evidence and target fingerprints compare strictly equal (followup-nexus-semantic-compare.log).
- Global lint-code-fast remains red on unrelated preexisting deletion of common/schedules Go files; not claimed passing. Scoped lint passes. Other initial tooling/gate failures and their recovery remain recorded, not erased.

Review and preservation: original base bbe765d70a708b6776c298aa5c0418bbd514048c; user externally committed WIP c5646fe69e36afaa08189c854e775dae462b0d1a during implementation. Agent commits []; no staging/commits/worktrees/reset/stash/rebase/checkout/revert. Original source-before snapshots retain preexisting edits; separate external-head/index guards preserve the user commit/current index. Owner transcript audit finds zero Git mutations/delegations. Every worker-owned foreground command exited; no active owner/review/gate runner (task2-command-closure.json).

Immutable final review artifact: task2-review-r2/{manifest.json,task.diff,before,after},38 extant task files and 2 fixture renames/deletions; diff SHA256 bed1e7b2540d432b6cdff7e7b2af83dba7172d2809f9c6586aea4b446878aa47. The in-memory compatibility adapter supplies only the task's saved-before/current diff, including externally committed task bytes while excluding unrelated user/task1 work. Native backend, models, reservations, verdicts/receipts/counters remain untouched. Source/diff/adapter hashes, current HEAD and raw index were checked after SHIP and immediately before done.
Actual receipt: /tmp/impl-review-receipt-8f37faba39e2-fn-107-scala-umpire-prototype-for-standalone.2.json, copied verbatim to task2-final-review-receipt.json; SHIP reservation 893551bbea96420b91c41eca2cc033e5; native sourceReceiptId review-7a54e4b0bdd53f32c32c7c9af8960c1e1c60f14cb07581bcfe0253655f32ac31. Final guards: task2-post-ship-guards.json. Memory captured in .flow/memory/bug/integration/channel-catalogs-and-visible-results-2026-09-30.md.

Backend handoff (.14/.15/.3): source-lifter admission is implemented; arbitrary-IR version/type/duplicate/bounds validation and new-declaration evaluation remain backend work. Existing Go Validate checks names/arity/bindings only; new monitor/assumption/composition/claim/progress fields are parsed but not evaluated yet, channel/hole expressions fail closed. Frontend refusal does not certify raw IR. Native tables refuse reached holes and bound channel delivery/loss; native Search refuses monitor-bearing Queries and supplies no assumption/progress semantics. Int channel messages admit only declared Finite.upTo ranges; nonrepresentable Int/list catalogs and computed policies are located errors. Runtime observations, neutral descriptors/DAG lowerings and exporters remain later tasks. Existing staleAdmission refinement counterexample remains expected.

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: GOFLAGS=-tags=test_dep make protoc LOCALBIN=.flow/tmp/fn-107/tool-prep/bin (rc 0,72.79s; .flow/tmp/fn-107/task2-logs/host-make-protoc-r2.log), GOFLAGS=-tags=test_dep make umpire-gen-scala (rc 0,55.13s; .flow/tmp/fn-107/task2-logs/reviewfix-umpire-gen-scala.log), GOFLAGS=-tags=test_dep make umpire-check-scala (rc 0,65.65s; .flow/tmp/fn-107/task2-logs/reviewfix-umpire-check-scala.log), make lint-scala (rc 0,48.11s; .flow/tmp/fn-107/task2-logs/reviewfix-lint-scala.log), mise exec -- model/scala/run.sh --no-prove (rc 0,20.38s; .flow/tmp/fn-107/task2-logs/host-native-proto-r2.log), mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./api/modelir/... && mise exec -- go vet -tags test_dep ./api/modelir/... ./model/scalav2/... (rc 0,6.14s; .flow/tmp/fn-107/task2-logs/reviewfix-go-tests.log), GOFLAGS=-tags=test_dep make lint-code LINT_CODE_TARGETS=./api/modelir/v1 GOLANGCI_LINT=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-107/task2-tools/golangci-lint-v2.13.1 ERRORTYPE=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-107/task2-host-tools/errortype GOLANGCI_LINT_BASE_REV=bbe765d70a708b6776c298aa5c0418bbd514048c (rc 0,34.97s; .flow/tmp/fn-107/task2-logs/host-lint-code-r2.log), GOFLAGS=-tags=test_dep mise exec -- go run .flow/tmp/fn-107/task2-wirecheck/main.go model/scalav2/lifter/testdata/lifts/expected/*.json model/scalav2/ir/nexus-caller.json (rc 0,0.87s; .flow/tmp/fn-107/task2-logs/reviewfix-wire-roundtrip.log), python3 .flow/tmp/fn-107/task2-comment-audit.py (rc 0,0.1s; .flow/tmp/fn-107/task2-logs/reviewfix-comment-audit.log), Baseline Scala rc0/35.04s and Go rc0/0.854s: task2-baseline/gates.json, Strict Nexus tables/IDs/refinement/evidence/fingerprints: followup-nexus-semantic-compare.log (rc0), 342 IR positions and15 refusal lines resolve: reviewfix-positions.log, Global lint-code-fast rc2 inherited no-Go-files common/schedules: lint-code-fast-darwin.log; scoped lint rc0, Source/diff/adapter/currentHEAD/raw-index guards after SHIP and before done: task2-post-ship-guards.json, All worker-owned foreground commands exited: task2-command-closure.json
- PRs: