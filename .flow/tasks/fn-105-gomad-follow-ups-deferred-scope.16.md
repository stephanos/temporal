---
satisfies: [R16]
---
# fn-105-gomad-follow-ups-deferred-scope.16 D16: investigate the forward-clock gRPC poll deadline mismatch

## Description
Decision on 2026-09-30: investigate TestStandaloneActivityTestSuite/TestStartDelay/UpdateWhilePaused_AfterWindow_ExtendsDispatch. The recorded seed-11 failure is a client deadline error instead of a successful empty three-second poll; it does not establish early activity dispatch. Under clock_tick forward the effective server deadline is recorded as a few microseconds later than the client deadline. Determine whether the correction belongs in Gomad clock handling, deadline propagation, or the test timeout expectation. Investigation is approved; the specific fix and any accepted limitation remain decisions to make from its evidence.

## Acceptance
- Retain a minimal reproducer and compare native Go with Gomad on seeds 11 and 17, including forward and baseline clock policies where applicable. Record platform, target/profile identity, commands, and outcomes.
- Trace the client deadline, server deadline, virtual time.Now readings, timer-clock progression, cancellation, and empty-response ordering sufficiently to identify the causal path.
- Establish whether the observed behavior follows ordinary gRPC/context semantics or violates Gomad's supported clock/deadline contract. Preserve the test's assertion that the activity does not dispatch before its extended delay.
- Record the diagnosed cause, correction owner, proposed next action, and regression/qualification criteria in fn-105. If a fix is needed, retain it as explicit open work for the next decision; a classification alone does not resolve the skipped test.
- Retain the skip until evidence supports its removal. Preserve the forward policy's timestamp improvements; no Gomad-only test rewrite, disabled assertion, or widened deadline may substitute for the investigation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
