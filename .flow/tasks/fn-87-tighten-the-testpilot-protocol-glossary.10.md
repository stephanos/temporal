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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

