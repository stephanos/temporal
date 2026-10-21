---
satisfies: [R2, R4]
---
# fn-224-gomad-ringpop-membership-on-wasm-within.2 Run the ringpop unit tests under the cooperative profile and classify every outcome

## Description
Prepare common/membership/ringpop as a go-test target with the Temporal guest build tags under the cooperative profile. Early proof point: one test at one seed; stop and report if preparation refuses an import. Inventory and reconcile the actual leaf methods in factory_test.go and monitor_test.go (currently nine under TestRpoSuite and TestRingpopSuite), then run every selected leaf over at least three seeds, classify each outcome as pass, refused import, capacity, timeout or guest failure, replay every pass exactly, and diagnose any stall to a named site. Resolve stalls in the diagnosed test harness; preserve NetworkV2 frozen-clock admission with ClockPolicy.ReadStepNanos=0 and record the fix. Nonzero read-step is refused and is not a workaround. Tolerate tchannel's 'Could not get SyscallConn' log line in expected output. Retain commands, seeds, identities and the classification under .flow/artifacts/.

**Size:** M
**Files:** .flow/artifacts/fn-ringpop-*/, common/membership/ringpop/*_test.go only if a stall needs a harness change

**Touches:** .flow/artifacts/fn-ringpop-*/, common/membership/ringpop/*_test.go only if a stall needs a harness change

## Acceptance
- [ ] Every inventoried leaf classified over at least three seeds with retained exact suite/method selectors, commands, identities and authenticated selected-method verdicts; suite wrapper PASS and skipped/unrun methods do not count as leaf passes
- [ ] Every pass is retained under explicit bounded success retention with complete bounded choice recording and replays exactly; retention/choice overflow remains a classified unresolved result
- [ ] Any stall diagnosed to a named site and resolved outside gomad's clock, choice recorded
- [ ] No change to gomad models or config

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
