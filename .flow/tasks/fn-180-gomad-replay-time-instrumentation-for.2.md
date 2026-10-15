---
satisfies: [R1, R4]
---
# fn-180-gomad-replay-time-instrumentation-for.2 Stop-and-inspect: debug goroutine, stop decision and built-in stack/heap dumps

## Description
Implement the R1/R4 stop-boundary contract using identical dormant infrastructure in every recording/replay incarnation, within the coupled .1–.3 batch. At `runtime_wasm.go.txt`'s `gomadChoiceDecision`/`gomadWasmStartUserCode` and `overlay.go`'s runq/select hooks validate a retained decision, preserve it in preallocated pending state and hand off to the precreated debug goroutine before applying it. Run-queue hooks execute on g0: never invoke callbacks there or return an invalid runnable rank. A user select hook parks its application goroutine and reaches the same scheduler path. Support every tape decision kind without conflating trace observations, and resolve time/failure using exact recorded shared order.

Seal and verify the selected global prefix, all executed incarnation cuts and compulsory runtime-state witnesses before inspection work. Park application goroutines, timer/finalizer/cleanup user callbacks and all other cluster guests; permit necessary runtime GC workers/assists for post-stop allocations, without World/time/network/lifecycle advancement or simulation resumption. End through the dedicated inspection terminal; never run normal full-tape finish against a deliberately unconsumed suffix.

Built-in stack/frame and memory-profile primitives live in runtime, which cannot import runtime/pprof. Emit structured frames/samples to the host for standard heap pprof encoding and validate it with `go tool pprof`; preserve existing sampling policy. User callbacks enter through .3's explicit guest init link seam. Bound output/cancellation, isolate per-dump panic and oversize and abort inspection on nonyielding timeout without reusing a cancelled store. Requalify the integrated runtime/model/helper candidate once after identity rotation.
## Acceptance
- [ ] Choice, time and failure stops produce selected-guest stacks/heap profile and requested/resolved boundary reports
- [ ] Fixtures distinguish equal local ordinals on two nodes and across a restart, and resolve equal-time boundaries by recorded order
- [ ] Terminated-incarnation replay inspects only an available prior boundary; no-boundary failure refused; validated stop not reached reports divergence
- [ ] Verify all executed cluster evidence prefixes and shared World order; perturb a nonselected node/order to prove it cannot escape verification
- [ ] Other guests stay parked during dumps; replay ends afterward
- [ ] Full WASM requalification gates pass on the integrated candidate
- [ ] Every admitted decision kind supports the exact predecision stop, including g0 runq and user select hooks; select observations do not shift tape ordinals
- [ ] All application execution stays parked while runtime GC can support allocating dumps; no dump callback executes on g0 or under an application lock
- [ ] Runtime stack/memory-profile primitives avoid the pprof import cycle; emitted heap profile is readable by `go tool pprof`
- [ ] Dedicated inspected-prefix terminal neither consumes the suffix nor claims full reproduction; normal replay still rejects missing/extra decisions
- [ ] Current-candidate mandatory witness and prefix comparisons catch output, storage, World and nonselected-incarnation mutations before success; optional diagnostics off is covered
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
