---
satisfies: [R2, R7, R15, R22]
---
# fn-115-make-the-scala-model-the-model-and.11 Migrate live functional fixtures and the pinned canary Case to Scala production

## Description
Migrate live functional fixtures and the pinned canary Case to Scala production. Implements R2, R7, R15, R22 using the reviewed parent contracts.

**Size:** M
**Files:** tests/testcore/testpilot fixture generation/consumers; tools/canary/casebinding and policy; affected recorded-run compatibility tests
**Touches:** [tests/testcore/testpilot/**, tests/testpilot*test.go, tools/canary/**, tools/umpire/**, model/temporal/**, model/cases/**, .plans/umpire-migration-*.json]

### Approach
- Execute the audited R22 replacement inventory through the existing checked lowering and transactional managed-tree writer. Bind every live fixture and the canary pin to a named Scala Query, or record the explicit unsupported primitive and retained-old-bytes exception.
- Treat replacement Program/Contract/provenance/Case identities as an explicit consumer artifact change, separate from invariant interpreter goldens. Change policy and binding references atomically and run preflight/admission before target effects.
- Retain historical recorded Runs unchanged; verify each still receives its established acceptance/rejection and record newly captured executions under their real identities. Do not rewrite recordings to match replacement Cases.
- Preserve generic generated live runner behavior and functional/canary exact-byte sharing where required. Test capability rejection and concurrent independent bindings, then execute the affected live demonstrations.

### Investigation targets
**Required** (current paths at planning time; follow the recorded move map after relocation):
- `tools/canary/casebinding/casebinding.go:26`
- `tests/testpilot_scala_canary_test.go`
- `tests/testcore/testpilot/scala_fixture.go:87`
- `model/scalav2/goir/testpilot/generated.go:219`
- `tools/canary/preflight`
- `tests/testpilot_scala_generated_test.go`

### Quick commands
CC=/usr/bin/clang mise exec -- go test -tags 'test_dep canary_harness' ./tools/canary/... ./tests/testcore/testpilot/...; the renamed generated/canary Testpilot live tests with -tags 'test_dep integration canary_harness'; make lint-code-fast

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.

## Acceptance
- [ ] Each live fixture has a Scala producer/Query or the R22 exception names its missing primitive and continuing consumer.
- [ ] Pinned canary, policy and binding identities agree; new Runs are recorded honestly and historical Runs remain byte-unchanged with established compatibility decisions.
- [ ] Complete-tree generation is deterministic, capability rejection precedes I/O, and affected functional/canary live gates pass.
- [ ] Semantic goldens remain unchanged except the separately recorded fixture migration.

## Done summary
Executed the reviewed R22 inventory of 47 Case fixtures. The eight replaceable functional Cases and the canary's pinned Case are now lowered from the Scala model through `umpire-gen-cases --kind functional|canary` into managed trees (`tests/testcore/testpilot/testdata/generated`, `tools/canary/casebinding/testdata`); each pinned Case is its `model/cases` file byte for byte. 16 Cases were already generated and 22 exceptions keep their exact bytes with the missing declaration named. New targets: `umpire-gen-fixtures`, `umpire-check-fixtures`, `canary-gen-case`, `canary-check-case`; CI's canary job runs both checks.

Both historical companion Cases were copied first with their recorded hashes; historical Runs, receipts and proposals are byte-unchanged and keep their decisions. The canary policy `caseIdentity` follows the new pin with no Profile change, and a new canary Run was recorded on the in-process cluster under its real identity. `model/`, checked IR and the 1,411 goldens are unchanged.

Full Go suite (48 packages), the three check targets, the live selection (26 tests; one assertion follows the Contract's rule count) and `lint-code-fast` pass. Recorded R22 gap: a Property phase clause lowers to no STATE rule, so the legacy control rule `state-succeeded` has no lowered counterpart. Independent review (Claude Fable, fresh context) returned SHIP in round 1; its P3 follow-ups are applied. Known defects outside this task, left for task 12: a data race in `TestWorkflowStartCaseSequentialAndConcurrentRunIsolation`, the undefined `bridgeExecutable` under the `integration` tag in `common/testing/testpilot/campaign`, and the policy `workflowPath` naming the deleted workflow. Handover: .flow/tmp/fn115-11-summary.md; evidence: .flow/tmp/fn115-11-evidence.json; review: .flow/tmp/fn115-11-review/round1-review.md. No agent commits.
## Evidence
- Commits:
- Tests: see commands, Independent review round 1 SHIP (claude-fable-5-1); .flow/tmp/fn115-11-review/round1-review.md
- PRs: