---
satisfies: [R2, R4]
---
# fn-224-gomad-ringpop-membership-on-wasm-within.2 Run the ringpop unit tests under the cooperative profile and classify every outcome

## Description
Prepare common/membership/ringpop as a go-test target with the Temporal guest build tags under the cooperative profile. Early proof point: one test at one seed; stop and report if preparation refuses an import. Then run all eleven tests in factory_test.go and monitor_test.go over at least three seeds, classify each outcome as pass, refused import, capacity, timeout or guest failure, replay every pass exactly, and diagnose any stall to a named site. Resolve stalls in the test harness or with a declared ClockPolicy.ReadStepNanos, never by changing gomad's clock; record the choice. Tolerate tchannel's 'Could not get SyscallConn' log line in expected output. Retain commands, seeds, identities and the classification under .flow/artifacts/.

**Size:** M
**Files:** .flow/artifacts/fn-ringpop-*/, common/membership/ringpop/*_test.go only if a stall needs a harness change

## Acceptance
- [ ] All eleven tests classified over at least three seeds with retained commands and identities
- [ ] Every pass replays exactly
- [ ] Any stall diagnosed to a named site and resolved outside gomad's clock, choice recorded
- [ ] No change to gomad models or config

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
