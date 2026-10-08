---
satisfies: [R5]
---
# fn-129-activity-coverage.5 Close: new Cases, live run, MILESTONES

## Description

Close only at the conjunctive shared fn-128.6/fn-129.5 boundary after .4 integration and joined fn-128.7/.8 focused corrections. This task shares one production generation/full-gate/review/live invocation with fn-128.6, not a second suite. Source-only completion is insufficient.

**Size:** M, bounded orchestration and closure using the existing gates/runner; source corrections return to their owning task/conductor.
**Files:** shared generated Model IR/Cases and exact managed fixture closure, final inventory/evidence and MILESTONES entry; canonical task/spec lifecycle through Flow only.
**Touches:** [model/ir/**, model/cases/**, tests/testcore/testpilot/testdata/generated/**, tests/testcore/testpilot/testdata/generated-case-names.txt, tools/canary/casebinding/testdata/**, .flow/tmp/activity-batch/**, MILESTONES.md]

## Approach

- Before generation, verify .1-.4 integrated original commits and focused evidence, then join parallel fn-128.7/.8 after their rescan of newly introduced reset/by-ID rows, strict fatal classification and exact final Profile charges. The conductor records one final source identity and shared gate ownership. Do not wait for fn-128/fn-138 spec closure; do not reopen the sealed fn-138 comparison or replace its baseline with fn-129 output.
- Run ONE shared production Model generation and managed fixture refresh from that source; review exact changed closure and expected Case deltas against sealed fn-138 plus .1-.4 scratch inventories. Then run repository-defined complete generation/lint/test/umpire gates and independent implementation review. Necessary corrections get their own original commits and invalidate affected evidence; do not hide them by editing expected outcomes, waivers or generated outputs directly.
- Map each R1-R4 requirement and each new behavior to an authored realized Query and admitted Case/Program/Contract identity; list all new Cases, pinned/exploring decisions and expected statuses/reasons. A heartbeat invocation is not server receipt; reset's raw SDK count is not an ordinal; an empty worker is not absent. Missing realization or unproved chronology is a blocker, not satisfied by a Model-only test.
- Execute ONE shared generated-Case live suite invocation for fn-128.6/fn-129.5 after live preflight and bounds/cleanup checks. Run the repository's bounded offline replay on the resulting recorded Runs. Assert disposition, cleanup, Contract, conformance, Property and every reason by equality live and replayed. Preserve old retry/retryAfterTimeout/retryExhaustion Property-only explanationsDisagree values exactly, fatal SATISFIED/satisfied Contracts, conformant retry paths and bounded pauseResume. Only the named ShutdownWorker race permits its known extra inconclusive. Retained cancellation-requested retry mismatch is context, not new live evidence; any actual conformance disagreement returns to human judgment under AGENTS.
- Require separate independent spec-completion reviews for fn-128 and fn-129 against this shared generation/gate/live/replay receipt and exact requirement maps. The shared conductor also owns a separate fn-138 completion review/closure only after its independent R3 seal and these shared final gates; no fn-138 task-body rewrite or reopened baseline. Lifecycle changes happen only after all three acceptance sets and reviews pass. Remove fn-129's now-completed MILESTONES entry and close it through official Flow completion; record the shared receipt rather than rerunning live for an individual close. Preserve downstream fn-142 -> fn-143 -> fn-140 -> fn-123 -> fn-145 -> fn-146 -> fn-147 -> fn-148 -> fn-141 last; unrelated captured/deferred work stays out of scope.
- Retain final source/generated commits, commands/exits/counts/bounds, full gate logs, actual review receipts, live Run IDs/log paths, replay comparison, cleanup result and R1-R5/new-Case inventory. Failed cleanup/resource limit/incomplete or unexpected status/reason blocks closure. This plan refresh itself performs none of these lifecycle or source actions.

## Investigation targets

**Required** (read before executing):
- `AGENTS.md`, `model/README.md`, `model/SEMANTICS.md` and `.plans/UMPIRE4_SPEC.md` current gate/conformance rules.
- `Makefile` existing production generation, full gate and generated-Case live/replay targets; owning Testpilot/Umpire runner docs.
- fn-128.6/.7/.8 and fn-129.1-.4 persisted plans, final original commits, scratch receipts and strict expectation inventory.
- Retained activity-batch `live-preflight.md`, conductor receipt and separately sealed fn-138 original/adopted proof.
- Current `MILESTONES.md` and official Flow requirement/spec-completion validation.

## Quick commands

```bash
python3 /home/agent/.codex/scripts/flowctl.py validate --spec fn-129 --coverage --json
git diff --check
make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks
make umpire-gen-fixtures canary-gen-case
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks
make umpire-check-testpilot-protocol umpire-check-cases umpire-check-fixtures canary-check-case umpire-check-lint umpire-check-exploration-bridge umpire-check-replay-bridge umpire-check-backends
make lint-model lint-code
/usr/bin/time -p mise exec -- go test -json -tags test_dep -p 2 -timeout 30m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...
UMPIRE_REPEAT_RUN_DIR=/absolute/shared-boundary/runs mise exec -- go test -v -count=1 -timeout 30m -tags 'test_dep integration' ./tests -run '^TestTestpilotGeneratedCases$'
```

These are existing shared-boundary commands, not authorization to run them during this planning refresh or .1-.4. With Go run separately, generation/model gate use MODEL_GATE_ARGS=--skip-go-checks, and the already-required full Go run retains -json, exit status and elapsed walltime under MILESTONES verification instructions. The conductor substitutes a real absolute run directory and records exact selectors/commands/exits/counts before execution; serialize heavy work with the shared flock and retain measured JSON/time logs without piping away the test exit. `tests/testpilot_generated_test.go:99` performs bounded offline `AssessedCase.Evaluate` with exact Verdict/Assessment equality in this same live invocation. Retain that comparison and independently reassess each captured Run with the existing umpire-assess CLI when needed; do not use the violated-Run reducer as a substitute for successful-Run semantic replay. No duplicate live suite is authorized by this task.

## Acceptance

- [ ] Conjunctive source gate includes .4 and both fn-128.7/.8 correction receipts on one final source. One shared production generation/managed fixture closure, complete gates and actual independent implementation review pass; sealed fn-138 evidence is intact.
- [ ] R1-R4 and every new behavior map to admitted realized Queries and live Cases; all new Cases and exact identities/expected outcomes/reasons are listed. Exploring/pinned R4 mapping is explicit and source-only rows do not supply live credit.
- [ ] One shared generated-Case live invocation and bounded offline replay pass exact disposition/cleanup/Contract/conformance/Property/reason assertions, including preserved retry exceptions, strict fatal SATISFIED and bounded pauseResume. Actual logs, Run IDs, replay equality and cleanup/bounds are retained; no unknown inconclusive is accepted.
- [ ] Independent fn-128 and fn-129 spec-completion reviews accept the shared evidence before lifecycle mutation; the conductor's separate fn-138 review also accepts its independently sealed R3 proof plus final shared gates before that spec closes. Final done summary records the shared receipt and new Case list, MILESTONES removes fn-129 and official Flow closure succeeds; unrelated specs/activation/order are unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
