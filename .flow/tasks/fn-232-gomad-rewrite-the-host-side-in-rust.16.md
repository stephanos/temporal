---
satisfies: [R12, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.16 Report absent counterfactual reach without weakening declared bindings

## Description
Size: M

**Touches:** `tools/gomad/rust/driver/broad_stages.py`, `tools/gomad/rust/driver/live.py`, `tools/gomad/rust/driver/counterfactual_receiving.py`, `tools/gomad/rust/driver/executed_divergence.py`, `tools/gomad/rust/driver/tests/test_broad_stages.py`, `tools/gomad/rust/driver/tests/test_live.py`, `tools/gomad/rust/driver/tests/test_executed_divergence.py`

### Files / Touches
- tools/gomad/rust/driver/broad_stages.py
- tools/gomad/rust/driver/live.py
- tools/gomad/rust/driver/counterfactual_receiving.py
- tools/gomad/rust/driver/executed_divergence.py
- tools/gomad/rust/driver/tests/test_broad_stages.py
- tools/gomad/rust/driver/tests/test_live.py
- tools/gomad/rust/driver/tests/test_executed_divergence.py

### Approach
Use one consistent absent-versus-declared binding contract across broad run/collect and final reach reading. C4 has no executed_divergence inputs; currently the cluster-functional stage unconditionally runs and reads that endpoint. With REACH_GAPS_BLOCK=false, report absence as explicit uncredited reach owned by .14, rather than executing an unavailable endpoint or manufacturing PASS facts. Reenabling completeness must refuse the gap. A declared endpoint remains strict for malformed inputs, failed processes and unauthenticated receipts. For declared dry-run role records, pass the sealed configuration explicitly to the executed_divergence child and load/verify it before accepting those records. Preserve normal frozen-role behavior and source/config/role custody. This task follows the shared-readiness task because both touch broad_stages.py.

### Bounded required reads
Read this spec R12/R14, .13 owner amendment and .14 scope; broad_stages.py run/collect/closure, live.py reach and final-reader paths, counterfactual_receiving.py selection/argv/inputs/observations/final_read, executed_divergence.py roles/produce/main and existing focused tests. Read only directly referenced configuration/dry-run loader helpers needed to retain their verification contract.

### Focused tests
Run the existing broad-stage, live reach, counterfactual receiving and executed-divergence test suites with their established invocation. Cover absent versus invalid declarations in both producer and independent final-reader paths and explicit verified child configuration. No reach-port implementation, full cycle or binary rebuild belongs here.
## Acceptance
- [ ] Absent executed_divergence neither launches its producer nor demands its receipts; broad and final readers retain an explicit unexecuted, uncredited gap when REACH_GAPS_BLOCK=false.
- [ ] The same absence refuses when reach completeness is enabled; unrelated strict stage obligations remain enforced.
- [ ] Declared malformed, failed or unauthenticated divergence inputs remain refusals, including source/config/role mismatches; none is downgraded to optional absence.
- [ ] Declared dry-run roles receive explicit child configuration authenticated by the existing verified loader, while normal frozen-role validation remains strict.
- [ ] Focused producer/reader regressions pass; .14 retains reach closure and .13 retains the actual 96-stage cycle/evidence obligation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
