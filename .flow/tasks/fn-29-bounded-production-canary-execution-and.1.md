---
satisfies: [R1, R10]
---

# fn-29-bounded-production-canary-execution-and.1 Declare the canary Evaluation Profile and the canary policy outside Umpire

## Description
Add `Temporal.Evaluation.Canary` declaring the `production-canary` Evaluation Profile with `Umpire.Evaluation`: claim text naming one serial Run of the pinned canary Case against the dedicated production-canary namespace and route, trust `dedicated-production-canary`, blocking Known Gap kinds `capability`, `input`, `interpretation` and `claim`, and `local-ephemeral`'s table in its order except that `unsupported-rule` forces `rejected`; its identity pinned in `Temporal/Evaluation/CanaryTests.lean`; and a `canary-harness` Profile beside it (trust `test-cluster-harness`, the same table), rendered only into `tools/canary/testharness/profiles/` and embedded only by the harness build (.10). Change `umpire-evaluation-profiles` to take `--local-dir <dir> --canary-dir <dir> --harness-dir <dir>` and render `Temporal.Evaluation.Local.declared`, `production-canary` and `canary-harness` into them; `umpire-gen-evaluation-profiles`/`umpire-check-evaluation-profiles` pass `tools/umpire/evaluation/profiles`, `tools/canary/assessment/profiles` and `tools/canary/testharness/profiles`. Export `evaluation.ParseProfile` (the existing strict parse and validation), and export `tools/umpire/internal/recordedrun` as `tools/umpire/recordedrun` (the internal path aliases it for its current importers), so the canary computes a Case's identity with `recordedrun.CaseIdentity` rather than re-implementing the canonical form. Add `tools/canary/policy`: `production-canary.json` embedded and decoded strictly into `Policy` (canary Case identity, Case Profile name, Evaluation Profile name, authority class, SHA-256 digests of the gRPC host name, namespace, task queue, handler queue and Nexus endpoint, the lease's workflow ID, type and task queue (`umpire-canary-lease`), trusted ref and workflow path, and the Limits: iterations 2, invocation 10m, cleanup reserve 2m, lease run timeout 24h, progress 64 KiB), rejecting unknown, repeated or case-folded keys, a missing field, another version, a zero or negative limit, a lease run timeout no longer than the invocation limit plus the reserve, and a non-digest; and `tools/canary/assessment/profile.go` loading the embedded canary Profiles through `evaluation.ParseProfile`. The untagged build's policy source returns only the embedded policy, which names `production-canary`; a `canary_harness`-tagged source (added in .10) is the only other. Extend the dependency rule in `tools/umpire/regression` so no package under `tools/umpire`, `common/testing/testpilot` or `model` tooling imports `tools/canary`.

### Quick commands
`cd model && lake build Temporal.Evaluation.CanaryTests umpire-evaluation-profiles && cd .. && make umpire-check-evaluation-profiles && go test -count=1 -tags test_dep ./tools/canary/... ./tools/umpire/evaluation/ ./tools/umpire/regression/`

**Files:** `model/Temporal/Evaluation/Canary.lean`, `model/Temporal/Evaluation/CanaryTests.lean`, `model/Temporal.lean`, `model/TemporalModelTests.lean`, `model/Temporal/Tool/EvaluationProfiles.lean`, `Makefile`, `tools/canary/assessment/profiles/**`, `tools/canary/testharness/profiles/**`, `tools/canary/assessment/profile.go`, `tools/canary/assessment/profile_test.go`, `tools/canary/policy/**`, `tools/umpire/evaluation/profile.go`, `tools/umpire/recordedrun/**`, `tools/umpire/internal/recordedrun/**`, `tools/umpire/replay/recorded.go`, `tools/umpire/regression/ci_workflow_test.go`
**Touches:** `model/Temporal/Evaluation/Canary.lean`, `model/Temporal/Evaluation/CanaryTests.lean`, `model/Temporal.lean`, `model/TemporalModelTests.lean`, `model/Temporal/Tool/EvaluationProfiles.lean`, `Makefile`, `tools/canary/assessment/profiles/**`, `tools/canary/testharness/profiles/**`, `tools/canary/assessment/profile.go`, `tools/canary/assessment/profile_test.go`, `tools/canary/policy/**`, `tools/umpire/evaluation/profile.go`, `tools/umpire/recordedrun/**`, `tools/umpire/internal/recordedrun/**`, `tools/umpire/replay/recorded.go`, `tools/umpire/regression/ci_workflow_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] The canary Profile declares and renders only into the canary directory; `umpire-assess` cannot select it, and its identity is pinned in Lean and matched in Go.
- [ ] An unknown, repeated or case-folded policy key, a missing field, another version, a non-positive limit or a non-digest coordinate rejects, each by name.
- [ ] No Umpire, Testpilot or model-tooling package imports `tools/canary`, pinned by the dependency rule; the canary Profile and policy carry no credential or raw coordinate.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
