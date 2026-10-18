---
satisfies: [R6, R7, R8]
---
# fn-107-scala-umpire-prototype-for-standalone.20 Generic Testpilot evidence sources: single message, Run Events under a guard, kinds sharing a source

## Description
**Touches:** [common/testing/testpilot/internal/execution/**, common/testing/testpilot/internal/ir/**, common/testing/testpilot/internal/verification/**, common/testing/testpilot/contract/**, common/testing/testpilot/*.go, common/testing/testpilot/README.md, common/testing/testpilot/temporal/**, proto/internal/temporal/server/api/testpilot/v1/**, api/testpilot/v1/**, tests/testcore/testpilot/**, tools/umpire/**/testdata/**, tools/canary/**/testdata/**]

Give Testpilot the three generic evidence primitives the Scala activity and Nexus realizations need. Each is named by the task 19 review (`.flow/tmp/fn-107/task19/t19-r1.md`, judgments c and d) and by task 19's handover; none is feature policy.

**Size:** L

### Approach
- **Single-message read source.** Today `ReadEvidence` lifts evidence only from a repeated field (`internal/execution/evidence.go` ~:121), and every standalone activity status is `DescribeActivityExecution`'s single `info` message. Add a distinct single-message source (or an explicit cardinality on the source), preserving method authorization, descriptor checking, limits, polling semantics and dense-ordinal behaviour. Existing repeated-field reads stay byte- and behaviour-compatible.
- **Run Event evidence with a payload guard.** Attempt start and attempt identity are stable only in the typed `InstructionOutcome.activity_attempt` Run fact (task 13); a Describe poll can miss the transient scheduled and started states. Let an evidence declaration take its occurrences from Run Events under a generic payload predicate (for example `sdk_attempt > 0` and a nonempty delivery, which excludes `NOT_NEEDED` records), keyed by the typed identities. `response` is the worker's offer and proves no server acceptance; the primitive must not suggest otherwise.
- **Distinct kinds in one dense source.** Testpilot rejects two evidence declarations with the same source and operation path, so the five Nexus history kinds, which one read lifts from one history, cannot all be declared. Permit distinct kinds lifted by one instruction from one shared dense source, with one ordinal space, keeping unique-kind admission.
- **A canceled answer for an activity attempt.** The Scala vocabulary declares `Instruction.AttemptCanceled` (task 19) and no Testpilot instruction lowers it: beside `Finish` and `activity_attempt_failure`, an activity script needs an arm by which the worker answers the attempt as canceled, recorded as a typed `activity_attempt` response (an offer, like the others) through the closed `reservationOutcomes` table and the attempt lifecycle task 13 documented. An attempt that deliberately gives no answer (for a start-to-close timeout) is NOT in scope; it stays a recorded limit.
- One additive protocol change for all four if the descriptor must change. Every Testpilot protocol file is inside the Driver catalog identity: measure the blast radius as task 13 did (guarded `tools/umpire` and `tools/canary` baseline before and after), prove every checked-in Case and the lowered Cases keep their identity, and re-record the pinned Runs once with `PATH=/tmp/umpire-no-lean:$PATH make umpire-rerecord-pinned-runs`.
- Test-first. No feature-specific code: the tests use neutral fixtures.

### Investigation targets
**Required:** `.flow/tmp/fn-107/handover/task19-summary.md`, `task13-summary.md` (protocol procedure, re-recording, the three lake-stub tests that are the tools baseline); `common/testing/testpilot/internal/execution/{evidence,dataflow,scheduler}.go`; `internal/verification/correlated.go`; `proto/internal/temporal/server/api/testpilot/v1/{case,program,instruction,run}.proto`.

### Quick commands
`CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./model/scalav2/...` (and `-race` over `./common/testing/testpilot/...`); `PATH=/tmp/umpire-no-lean:$PATH CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./tools/umpire/... ./tools/canary/...` (baseline: only three lake-stub tests fail); `PATH=/tmp/umpire-no-lean:$PATH GOFLAGS=-tags=test_dep mise exec -- make protoc`; scoped `make lint-code` with `GOLANGCI_LINT_FIX=false`. Nothing that calls `lake`.

## Acceptance
- [ ] An evidence declaration can lift from a single message of a response, with method authorization, descriptor checks, limits, polling and dense ordinals as for a repeated field; existing repeated-field Cases are byte- and behaviour-identical.
- [ ] An evidence declaration can take its occurrences from Run Events under a generic payload predicate and typed identity key; a `NOT_NEEDED` or undelivered attempt record is not an occurrence; nothing treats the worker's offered response as server acceptance.
- [ ] Distinct evidence kinds can share one dense source lifted by one instruction; two declarations of one kind are still refused.
- [ ] An activity script can answer an attempt as canceled; the answer is a typed attempt response accepted only for an activity entrypoint, and the lifecycle table test covers it.
- [ ] Every checked-in Case file and every lowered Case keeps its identity; pinned Runs are re-recorded if the descriptor changed; the guarded tools run is at its baseline; existing Testpilot consumers pass.
- [ ] No feature policy and no dependency from Testpilot on `model/`.


## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: