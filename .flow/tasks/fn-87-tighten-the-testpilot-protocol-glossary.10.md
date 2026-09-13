---
satisfies: [R12]
---
# fn-87-tighten-the-testpilot-protocol-glossary.10 Resource ceilings move to the Profile; SEM-16 amendment drafted

## Description
Move node, edge, byte, work, capture, depth, duration and count ceilings out of the Case and into the Profile (R12, spec "Resource ceilings in the Profile"). A Case keeps only bounds that carry meaning: instruction timeout and attempts, Contract deadlines, and correlated windows. Runtime and admission read ceilings from the prepared Profile snapshot. Draft the SEM-16 amendment under GOV-02.

**Size:** M
**Files:** `proto/.../v1/{program,instruction,contract,correlated}.proto`, `api/testpilot/v1/*`, `common/testing/testpilot/profile.go`, `common/testing/testpilot/contract/profile.go` (if ProfileSpec lives there), `common/testing/testpilot/internal/execution/{prepare.go,program.go,dataflow.go,values.go,projection.go}`, `common/testing/testpilot/internal/verification/{prepare.go,captures.go,correlated_prepare.go}`, `common/testing/testpilot/temporal/{profile.go,driver.go,server/session.go,worker/session.go,worker/routing.go}`, `common/testing/testpilot/conformance_test.go:160-172`, `tests/testcore/testpilot/profile.go:47`, `tools/umpire/cmd/umpire-run/*` (Profile construction), `model/Testpilot/Authoring.lean` (`Program.limits`, `Contract.limits`, `instructionLimits`), `model/Temporal/Testpilot/CaseSupport.lean:29,82-85`, `model/Umpire/Case/{Compiler,Producer,Correlated}.lean`, typed Producers, fixtures, mapping, `.plans/UMPIRE4_SPEC.md` (SEM-16 draft), `common/testing/testpilot/internal/verification/README.md:14-17,41-44`, `common/testing/testpilot/temporal/README.md:42-51`, `tools/umpire/CONTEXT.md:95`, `model/ARCHITECTURE.md:123-138`
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, common/testing/testpilot/**, tests/testcore/testpilot/**, tests/testpilot_*_test.go, tools/umpire/**, model/Testpilot/**, model/Temporal/**, model/Umpire/Case/**, model/ARCHITECTURE.md, .plans/UMPIRE4_SPEC.md]

### Approach
- Which bounds stay: the spec lists what a Case keeps (instruction timeout and attempts, deadlines, correlated windows), so every other bound moves, including `max_total_duration_milliseconds`, `max_cleanup_duration_milliseconds`, `ProgramLimits.max_attempts`, `max_run_events`, `max_activations`, `max_entrypoints` and every `CorrelatedLimits` field. Per Edge Cases "Ceilings", a bound that carried meaning stays in the Case: if moving the total or cleanup duration changes any Verdict, disposition or cleanup status in a conformance class or live test, keep that bound in the Case as a behavior bound and record why.
- Case after this task: `InstructionNode.limits` holds only `timeout_milliseconds` and `max_attempts`; `Program.limits`, `Contract.limits` and `CorrelatedContract.limits` are removed from the Case; `CorrelatedRule.bound` and `Deadline` stay. `ProgramLimits`, `ContractLimits`, `CorrelatedLimits` remain as Profile ceiling messages (not referenced from `Case`), and the per-instruction ceilings move into `ProgramLimits` as new fields `max_instruction_emitted_events` and `max_instruction_response_bytes`; the existing Program-wide `ProgramLimits.max_response_bytes` keeps its meaning (`correlated_prepare.go:81` still bounds `max_event_bytes` by it), so no admission check changes which value it compares against.
- Profile: `ProfileSpec` gains `CorrelatedLimits` (today correlated limits are only checked positive, `verification/correlated_prepare.go:59-77`); `hardLimits()` (`execution/program.go:156`, `verification/prepare.go:71`) still bounds the Profile. Admission checks the Case's remaining bounds against Profile ceilings exactly as today (`execution/prepare.go:107-131`, `verification/prepare.go:74-114`, `dataflow.go:123-133`).
- Runtime reads: replace every read of Case limits with the prepared Profile snapshot (`execution/values.go:61,108,135`, `projection.go:41,161`, `verification/captures.go`, `temporal/worker/session.go:66-95,322`, `temporal/worker/routing.go:178`, `temporal/server/session.go:62,92`, `temporal/driver.go:206,220`). Prepared state is immutable (ART-10); Drivers get ceilings through prepared metadata, not the source Case.
- Profile construction: `temporal.DeriveProfile` (`temporal/profile.go:53-54`), `conformance_test.go:170-171` and `tests/testcore/testpilot/profile.go:47` copy Case limits today. Add one Temporal default ceiling set (for example `temporal.DefaultCeilings()`), valued at the maximum each ceiling takes across today's checked-in Cases, so no Case is refused; DeriveProfile, the facade test Profile and the testcore Profile use it. Per Edge Cases, list every bound that loosens (Case value lower than the new Profile ceiling) per fixture in the done summary; none of them is a timeout, attempts, deadline or window, which stay in the Case.
- Lean: `Testpilot.Authoring` drops `Program.limits`/`Contract.limits`/correlated limits constructors and `instructionLimits` takes two values; `CaseSupport.bounds`/`programLimits`/`contractLimits` and `Umpire.Case.Compiler`'s `contractLimits` input and `Producer.Realization.contractLimits` go (the Lean unbounded-number narrowing checks for removed fields go with them; keep them for timeout and attempts).
- SEM-16 draft: under SEM-16 in `.plans/UMPIRE4_SPEC.md`, add a restatement marked `*(drafted by fn-87; awaiting GOV-02 approval.)*` (the MOD-14 restatement is the pattern): a Case is authoritative for its Program, Contract and the bounds that carry behavior (instruction timeouts and attempts, deadlines, correlated windows); resource ceilings belong to the Profile, which admission checks every Case against. Do not edit ART-09 or the Profile glossary entry (boundary); list in the done summary that ART-09's "independent limits" wording needs a GOV-02 follow-up.
- Pinned tests: `preparation_error_test.go:81` uses a `max_rules` Case limit; move it to a Profile ceiling case.
- Mapping: drop the removed limit fields; a step table records per fixture which bounds moved to the Profile. Retired tokens: none needed unless a message is renamed.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/profile.go:60-120`
- `common/testing/testpilot/internal/execution/prepare.go:100-135`, `program.go:150-170`
- `common/testing/testpilot/temporal/profile.go:20-60`
- `model/Temporal/Testpilot/CaseSupport.lean:20-90`
- `.plans/UMPIRE4_SPEC.md` SEM-16 and the MOD-14 restatement (drafted-amendment format)

**Optional:**
- `common/testing/testpilot/temporal/worker/session.go:60-100,315-330`
- `tools/umpire/cmd/umpire-run/` — how the CLI builds a Profile

### Key context
- Live tests and `umpire-run` construct Profiles; every construction site must get ceilings or preparation will reject for a missing ceiling (`profile.go:97` already requires Program limits when bindings exist).

## Acceptance
- [ ] a Case declares only instruction timeout and attempts, deadlines and correlated windows; Program, Contract and correlated limits are Profile ceilings
- [ ] a Case bound above the Profile ceiling still rejects at preparation (unit test per remaining bound kind); runtime reads ceilings only from the prepared Profile
- [ ] one Temporal default ceiling set feeds `DeriveProfile`, the facade test and the testcore Profile; loosened bounds listed per fixture in the done summary
- [ ] SEM-16 restatement drafted under GOV-02 in `.plans/UMPIRE4_SPEC.md`; ART-09 follow-up noted, not edited
- [ ] equivalence test passes with the declared limit-removal step; Verdict pins unchanged; `make umpire-check-regression` exit 0 with nine live identities; `make lint-model` 163; `make lint-code` no new issues


## Done summary
Resource ceilings now live in the Profile. A Case keeps only the bounds that carry behavior: instruction timeout and attempts, Contract deadlines, and correlated windows. Admission and runtime read every other ceiling from the prepared Profile snapshot, and the SEM-16 restatement is drafted under GOV-02.

**Protocol and Go**
- `Program.limits`, `Contract.limits` and `CorrelatedContract.limits` are removed, and `InstructionLimits` keeps `timeout_milliseconds` and `max_attempts`.
- `ProgramLimits` gains `max_instruction_emitted_events` and `max_instruction_response_bytes`; `ProfileSpec` gains `CorrelatedLimits`, required only to admit a correlated contract.
- Execution, verification and the correlated monitor read `PreparedProgram.limits` and `PreparedContract.limits` / `correlatedLimits`. Drivers read `testpilot.PreparedProgram.Limits()`: the worker's `programDefinition.limits`, and the server's own Profile for the instruction response size.
- Worker and server Profile validation admit the two new fields.
- `temporal.DefaultCeilings()` feeds `DeriveProfile`, and through it `umpire-run`, plus the testcore `caseProfile`. The exported testcore Profile helpers lost their now-unused `source` parameter.

**Decisions** (recorded in the fn-87 Planning decisions as "decided in .10")
- **Every bound moved, durations included.** The full regression passed on its first run with all nine live identities, and no fixture's `expected.json` or correlated `expected` changed. So no bound had to stay in the Case as a behavior bound.
- **Remaining Case bounds are checked as before.** An instruction timeout must fit the total or cleanup duration, and its attempts the Profile's attempts. Because there is now one attempts ceiling, an instruction may no longer declare more attempts than the Program-wide ceiling. Deadlines and correlated windows get no new ceiling check, since none bounded them before.
- **Profile consistency.** The two per-instruction ceilings must fit `max_run_events` and `max_response_bytes`; this replaces the old per-Case check. A correlated contract reserves the capture ceiling in its capture budget only when it declares captures.
- **Facade conformance test (deviation).** The spec wanted it to use `temporal.DefaultCeilings`, but the enforced gate `TestTestpilotOwnsCaseProtocolAndRuntime` forbids generic Testpilot files from importing `common/testing/testpilot/temporal/**`. The test therefore spells the same ceiling set, and the Driver's new `TestDefaultCeilingsAdmitTheConformanceCorpus` prepares that corpus under `DefaultCeilings`.
- **Separate test Profiles.** The synthetic Case and the correlated corpus keep their former bounds as their own Profiles. The corpus's runnable Cases need 256 nodes and 2048 Run Events.
- **Lean.**
  - `Testpilot.Authoring.instructionLimits` takes two values, and the `Program.limits` / `Contract.limits` constructors are removed along with the limits parameters of `Program.make`, `Contract.contract` and `Contract.correlated`.
  - `CaseSupport.bounds` / `programLimits` / `contractLimits`, `Compiler.Input.contractLimits`, `Producer.Realization.contractLimits` and the TypedNexus ceiling defs are gone.
  - `Testpilot.Correlated.decode` takes `(limits : CorrelatedLimits)`, and `Umpire.Case.Correlated.Lowered` carries the `limits` it decoded under, so the checked agreement and theorems are unchanged.
  - The Lean narrowing check for correlated limits stays, because the lowering still computes them.

**Tests**
- `TestPrepareRejectsCaseBoundsAboveProfileCeilings`: one row per remaining bound kind (ordinary timeout, cleanup timeout, attempts), each admitted at the ceiling and rejected with `limit_exceeded` at the node path one above it.
- `TestPreparationErrorCase`: the `max_rules` and `max_nodes` Case-limit rows became the Profile ceiling rows "program ceiling" and "contract ceiling".
- `TestPreparationErrorCorrelatedLimits` now covers a missing correlated ceiling (malformed) and a non-positive one (limit_exceeded).
- A new correlated "missing Profile correlated limits" test.
- `TestDefaultCeilingsAdmitTheConformanceCorpus`.
- `TestDeclaredCeilingStepAdmitsOnlyBoundsWithinTheirProfile`.
- Reservation tests updated for the single attempts ceiling, with a new "local above global" rejection row.

**Fixtures and oracle**
- Fixtures were regenerated through the generators; the diff is limit deletions only (1516 lines across 14 files).
- The oracle's new R12 step drops each limit block or bound only when it lies within its fixture's declared Profile ceiling (`protocolmigration/ceilings.go`). Lowering the Temporal `maxRunEvents` ceiling to 256 made the oracle fail on worker-outage.
- Retired tokens: none. No message was renamed.

**Loosened bounds per fixture** (baseline value -> Profile ceiling)
- The five conformance Cases (satisfied, violated, inconclusive, cross-run-isolation, static-preparation-rejection), plus cleanup-failure-after-proved-violation and get-system-info, all loosen the same way:
  - program: cleanupDuration 5000->20000, responseBytes 4096->8192, runEvents 256->512
  - every instruction: emittedEvents 8->128, responseBytes 4096->8192
  - contract: captureBytes 8192->65536, captures 4->64, transitions 16->64, workPerEvent 100000->4000000
- **typed-unary:** the same program and contract loosening. Instructions: responseBytes 4096->8192 on all; emittedEvents 8->128 on all except history.
- **async-nexus:**
  - program: the same loosening as above
  - instructions: responseBytes 4096->8192 on all; emittedEvents 8->128 on all except history and start-workflow
  - contract: transitions 16->64, workPerEvent 100000->4000000
  - correlated: events 32->64, buffered 16->32, support 128->256, semanticTransitions 16->32, captures 0->16, correlationDepth 0->2
- **typed-nexus:** program runEvents 256->512; every short instruction emittedEvents 8->128 and responseBytes 4096->8192. Its contract and correlated bounds were already at the ceiling.
- **worker-outage:**
  - program: cleanupDuration 5000->20000
  - instructions: emittedEvents 8->128 on the short ones and 64->128 on history; responseBytes 4096->8192 on the short ones
  - contract: the same as the conformance Cases
- **synthetic:** none.
- **correlated.json, every entry:** `case` program activations, attempts, edges and nodes 16->256, pathFanout 8->256, runEvents 32->2048; `runnableCase` none.

**Follow-ups**
- ART-09's "independent limits" wording, and the Profile glossary entry's "independent Program and Contract ceilings", need a GOV-02 restatement. Neither was edited (.13 drafts ART-09).

**Outside the declared Touches:** none. Only the `.flow` spec decision and its review JSON.

**Gates**
- Baseline was green: the oracle ran, and the regression receipt b32e9867 was honored.
- Oracle and the focused Lean, protocol, conformance and vocabulary make targets: green.
- `make umpire-check-regression`: run 1 at b90e5aab exited 0 with 9 live identities; no flake occurred. Receipt b90e5aab written.
- `lint-code` after `go clean -cache` at 5a6a26c4 reported 163 issues, including 2 gci formatting hits in my two verification test files. Both were fixed by gofmt in b90e5aab. The remaining 161 is the baseline; I did not rerun lint after the fix.
- `lint-model`: 163, the baseline.

**Review** (Claude impl-review: SHIP on round 1). Two P3 findings were not applied:
- (1) The facade test duplicates the ceiling numbers. The reviewer believed the import was allowed, but the regression gate forbids it, and equality cannot be pinned across the boundary without a new exported helper. The Driver-side admission test is the substitute.
- (2) The server `open(..., limits)` parameter is only nil-checked. It keeps the prepared-Program guard; the server reads ceilings from its own Profile, whose identity preparation already matched.

stage: impl-review - ran (claude backend, SHIP on first round)
## Evidence
- Commits: 5a6a26c44e8f5779c1c2bcb1c6662bb8078e4b65, b90e5aab73942133a50a4063c66697bb8d0d4d48
- Tests: go test -count=1 -tags test_dep ./common/testing/testpilot/internal/protocolmigration/, make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance umpire-check-retired-vocabulary, CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression (run 1 at b90e5aab: exit 0, 9 live identities), go clean -cache && make lint-code GOLANGCI_LINT_FIX=false at 5a6a26c4: 163 issues, 2 of them gci formatting in changed test files; both gofmt-fixed in b90e5aab (lint not rerun after that; 163 - 2 = baseline 161), make lint-model (163 = baseline), go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/...
- PRs: