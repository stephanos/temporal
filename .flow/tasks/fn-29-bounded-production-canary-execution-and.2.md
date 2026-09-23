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
- [x] Lean and Go agree on the canary Case bytes and identity, and the policy pins the same identity.
- [x] A changed Case, a crossed Profile name, another catalog or an unregistered canary Case rejects before authority acquisition.
- [x] The Contract reaches its Verdict only from the public server observations it names; the canary Case carries no Known Gap.

## Done summary
Canary Cases are recorded in their own Lean registry, and `umpire-case --render-canary <case-id>` renders only a registered one (an unregistered ID refuses by name). `make canary-gen-case` writes the pinned `nexusCallerCanary.syncCompletion` Case to `tools/canary/casebinding/testdata/`, `canary-check-case` diffs a fresh render and runs in `umpire-check-regression`, and the policy pins its identity. `tools/canary/casebinding` embeds the Case, checks its identity against the policy's, and `Bind` prepares it under the hand-authored `ProfileSpec`, held equal to `DeriveProfile`'s output by test. Tests pin the Case to the two public WorkflowService methods and the history and scheduled-source correlations, with no Known Gap, and pin that a changed Case, a crossed Profile name, another catalog or other bindings refuse before any Driver validation or open. Implementation review: SHIP on round three; its three P3 notes applied.
## Evidence
- Commits: 14de78e3c18d9c25b002b619d923a0046716c464, 09420f6d6cdf818244e6b5c7d9e67aa898ca6553, e648abf86e2f94eaca5facbee56477677a95c2b0, 8856a29ce15cf6b4910344b7b6f84c850b04c43b
- Tests: cd model && lake build umpire-case, make canary-check-case, go test -count=1 -tags test_dep ./tools/canary/..., GOLANGCI_LINT_BASE_REV=HEAD make lint-code-fast
- PRs: