---
satisfies: [R2, R4]
---
# fn-224-gomad-ringpop-membership-on-wasm-within.2 Run the ringpop unit tests under the cooperative profile and classify every outcome

## Description
Prepare common/membership/ringpop as a go-test target with the Temporal guest build tags under the cooperative profile. Early proof point: one test at one seed; stop and report if preparation refuses an import. Inventory and reconcile the actual leaf methods in factory_test.go and monitor_test.go (currently nine under TestRpoSuite and TestRingpopSuite), then run every selected leaf over at least three seeds, classify each outcome as pass, refused import, capacity, timeout or guest failure, replay every pass exactly, and diagnose any stall to a named site. Resolve stalls in the diagnosed test harness; preserve NetworkV2 frozen-clock admission with ClockPolicy.ReadStepNanos=0 and record the fix. Nonzero read-step is refused and is not a workaround. Tolerate tchannel's 'Could not get SyscallConn' log line in expected output. Retain commands, seeds, identities and the classification under .flow/artifacts/.

**Size:** M
**Files:** .flow/artifacts/fn-ringpop-*/, common/membership/ringpop/*_test.go only if a stall needs a harness change

**Touches:** .flow/artifacts/fn-ringpop-*/, common/membership/ringpop/*_test.go only if a stall needs a harness change

**Blocker (2026-10-21, resolved by `go-host-v4-compat` `a1b28c4aa3`; see the fn-224 artifact, "Tip rerun"):** On the integrated evidence candidate 39694cd12b (gomad 92c69594f2 + fn-224), all nine leaves pass on seeds 0, 1 and 2: 27/27 retained successes with the leaf's own PASS line and no SKIP, recorded with complete choice evidence. None of the 27 standalone replays reproduces. They fail with `choice trace malformed`, the Go-host replay attestation regression from dd00c57f70, which the fn-232 replay owner owns. The gomad tip Go host also cannot record, because it refuses the runtime decision header v2 from 38572d7e49. Exact replay (27/27) is proven only on the fn-221 sweep build 1d7b37f00a. Resume once standalone replay works on the integrated candidate (Rust replay plus an installer, or a Go-host fix), then rerun /home/agent/wt/evidence/fn224/candidate3/run.sh's ringpop block and retain passing replays. Evidence: .flow/artifacts/fn224-ringpop-scenario-2026-10-21.md, "Merge with gomad".

## Acceptance
- [x] Every inventoried leaf classified over at least three seeds with retained exact suite/method selectors, commands, identities and authenticated selected-method verdicts; suite wrapper PASS and skipped/unrun methods do not count as leaf passes
- [x] Every pass is retained under explicit bounded success retention with complete bounded choice recording and replays exactly; retention/choice overflow remains a classified unresolved result
- [x] Any stall diagnosed to a named site and resolved outside gomad's clock, choice recorded
- [x] No change to gomad models or config

## Done summary
On the gomad tip with the Go host v4 repair (`a1b28c4aa3`), all nine reconciled ringpop leaves pass the cooperative profile with `--choices`. That is `TestRpoSuite` ×4 and `TestRingpopSuite` ×5, over seeds 0, 1 and 2 with exact `^Suite$/^Leaf$` selectors: 27/27 successes. Each retains the leaf's own `--- PASS` line with no SKIP and no FAIL, and records a complete bounded `gomad-choice-trace/v4` profile. Every pass replays exactly: 27/27 `reproduced=true choice-replay=exact`, over empty tapes and over non-empty tapes of up to 4,194 decisions. No stall occurred, and gomad's clock, models and config are unchanged. The earlier standalone-replay blocker was the Go-host header v2 / trace v4 / tape-attestation regression, repaired in `go-host-v4-compat` (fn-230 R12, fn-228 R4/R9). Evidence: `.flow/artifacts/fn224-ringpop-scenario-2026-10-21.md`, "Tip rerun after the Go-host v4 repair". Raw data: `/home/agent/wt/evidence/go-host-v4-compat/ringpop-9x3`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: acc4b06853, 051ef6e504, 29503bcbf4, a1b28c4aa3
- Tests: /home/agent/wt/evidence/go-host-v4-compat/ringpop.sh ringpop-9x3 0,1,2 <9 leaves>, 27/27 standalone replay reproduced=true choice-replay=exact
- PRs: