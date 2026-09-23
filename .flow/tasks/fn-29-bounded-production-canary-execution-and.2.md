---
satisfies: [R2, R6, R10]
---

# fn-29-bounded-production-canary-execution-and.2 Pin the fixed canary Case the Lean model produces

## Description
Record canary Cases in a registry of their own (`Registry.recordCanary` in the canary branch of `Temporal.Case`'s `case` block, beside the white-box-gap check) update the comments that said canary Cases are registered nowhere (the Caller Model, its tests and `Temporal.Case.Syntax`), and add `umpire-case --render-canary <case-id>`, which renders only a registered canary Case. Add `make canary-gen-case` writing `umpire-case --render-canary`'s canonical compact output to `tools/canary/casebinding/testdata/nexusCallerCanary-syncCompletion-case.json`, which `casebinding` embeds, and `canary-check-case` comparing a fresh render with a plain `diff`, and `canary-check-case` diffing a fresh render, added to `umpire-check-regression`. Replace the policy's all-zero Case identity with the pinned Case's. Add `tools/canary/casebinding`: the pinned Case's identity (`recordedrun.CaseIdentity`, exported in .1) must equal the policy's, and `Bind(policy, environment testpilotdriver.Environment)` prepares it with `testpilot.Prepare` under the canary's hand-authored `ProfileSpec` (fn-80's decision: explicit literal roles, methods, opcodes, command types, binding IDs, Program, Contract and correlated limits and instruction defaults, the environment's coordinates filled in), with no connection; a test holds that spec equal to `testpilotdriver.DeriveProfile`'s output for the pinned Case under the same environment, so drift on either side fails review (.2 tests it with test names; .3 gives it the protected environment's coordinates, and its `PreparedCase` is the one the controller runs); a test pins that its Contract names only public server observations (no Known Gap, no instruction outside the Program's public vocabulary) and that a changed Case, a crossed Profile name or another catalog rejects before any authority is acquired.

### Quick commands
`cd model && lake build umpire-case && cd .. && make canary-check-case && go test -count=1 -tags test_dep ./tools/canary/casebinding/`

**Files:** `model/Temporal/Case/Syntax.lean`, `model/Temporal/Feature/Nexus/Caller/Model.lean`, `model/Temporal/Feature/Nexus/Caller/Tests.lean`, `model/Temporal/Case/Registry.lean`, `model/Temporal/Tool/Testpilot.lean`, `Makefile`, `tools/canary/casebinding/**`, `tools/canary/policy/**`
**Touches:** `model/Temporal/Case/Syntax.lean`, `model/Temporal/Feature/Nexus/Caller/Model.lean`, `model/Temporal/Feature/Nexus/Caller/Tests.lean`, `model/Temporal/Case/Registry.lean`, `model/Temporal/Tool/Testpilot.lean`, `Makefile`, `tools/canary/casebinding/**`, `tools/canary/policy/**`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Lean and Go agree on the canary Case bytes and identity, and the policy pins the same identity.
- [ ] A changed Case, a crossed Profile name, another catalog or an unregistered canary Case rejects before authority acquisition.
- [ ] The Contract reaches its Verdict only from the public server observations it names; the canary Case carries no Known Gap.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
