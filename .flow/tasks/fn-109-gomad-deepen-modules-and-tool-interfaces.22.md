---
satisfies: [R1, R18]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.22 Repair the simulation-exploration target path so a real campaign completes

## Description
Prerequisite found by fn-109.1 on 2026-10-01: no fixture can complete a simulation-exploration campaign, locally or through the isolated coordinator, so R1's simulation leg cannot be proven. Three defects, all outside fn-109.1's Touches (evidence: .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task1-evidence.txt and task1-review.md):

1. `tools/gomad3/target/capability.go` `builtInSimulationLinknames` is stale against `tools/gomad3sim`: five SHA pins differ, `gomadProcessVolumeOperation` is unpinned, and `runtime_time_toolchain.go` has no entry.
2. The `gomad3sim` test variant imports `syscall`, which closure mode rejects.
3. With the pins corrected in a scratch overlay, the target rejects the Runner's plan: "simulation exploration candidate identity does not match".

Establish for each whether it is drift since the simulation track last qualified (find the last passing evidence and the commit that broke it) or a never-working path, fix the cause at its owner, and add a retained simulation-capable fixture target under the existing test tiers that completes a simulation-exploration campaign with committed logical executions and exact replay. Preserve exact first-party simulation pins and allowed bridge directives: re-pin only to the reviewed current sources, never loosen the policy or admit `syscall` generically. Then finish the simulation leg of the fn-109.1 transport test (tools/gomad3/runner/coordinator_transport_test.go) so it requires a completing campaign with the supplied bounds through real isolated execution.

**Touches:** tools/gomad3/target/capability.go and its tests, tools/gomad3sim/** (test variant only where the syscall import lives), tools/gomad3/runner/simulation_exploration*.go and the exploration-plan identity owner, tools/gomad3/runner/coordinator_transport_test.go, tools/gomad3/runner/testdata/**, a fixture under the existing conformance/testdata tree.

Quick: go test -count=1 ./runner/... ./target/... (in tools/gomad3); go test -count=1 -tags test_dep ./tools/gomad3sim/... (repo root); make -C tools/gomad3 validate test-host; one built-CLI `gomad explore` simulation-exploration run that completes.

## Acceptance
- Each of the three defects has an identified cause and owner; the fix is at the owner, with a regression test that fails before and passes after.
- A retained simulation-capable fixture completes a simulation-exploration campaign locally and through the isolated coordinator with distinguishable nonzero limits, committed logical executions and exact replay of a retained artifact.
- The fn-109.1 transport test's simulation leg requires that completing campaign.
- Simulation pins stay exact and policy is not widened; closure-mode analysis of the fixture reports supported.
- darwin/arm64 gates pass; linux/amd64 is recorded as not run. No commits, staging or worktrees.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
