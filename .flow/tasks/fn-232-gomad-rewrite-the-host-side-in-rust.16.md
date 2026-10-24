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
Absent executed-divergence declarations now remain explicit UNEXECUTED, uncredited reach gaps under the owner's report-only policy. Broad diagnostics and the final reach reader independently preserve those facts. Enabling reach completeness refuses absence. Declared malformed, failed, corrupted or mismatched bindings remain strict refusals.

Declared child execution now receives explicit configuration through the existing verified loader. Dry-run roles without verified configuration refuse; ordinary frozen-role behavior remains supported. The synchronized broad-build boundary from .17 is unchanged.

The retained regressions failed before implementation. All 71 focused checks pass, including independent diagnostics/credit tampering and real child-CLI configuration admission. The mandatory combined gate passed on b15d9b5e0eb0c044612c8fb28db9a701aeea25de, with all 583 Python driver tests passing. Log: /home/agent/wt/evidence/fn232-split-closure/counterfactual-integration-gate.log.

stage: wave-dispatch - ran (model: gpt-6.1-sol)
stage: impl-review - ran: SHIP across correctness, contracts and integration (model: gpt-6.1-sol)
stage: plan-sync - skipped(config: disabled)

Review receipt: /home/agent/wt/evidence/fn232-split-closure/counterfactual-review.json. Source and review commits remain reachable from gomad. The actual 96-stage cycle remains owned by .13; remaining reach ports and strict completeness restoration remain owned by .14. Tracker sync: n/a (bridge inactive). No push or PR.
## Evidence
- Commits: 645e00f0613d5bf7e5fe35005b2b02fd95588e8e, b15d9b5e0eb0c044612c8fb28db9a701aeea25de, 7b5556210f
- Tests: timeout 600 env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/gomad/rust/driver/tests -p test_executed_divergence.py, timeout 600 env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/gomad/rust/driver/tests -p test_broad_stages.py, timeout 600 env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/gomad/rust/driver/tests -p test_live.py, timeout 600 env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/gomad/rust/driver/tests -p test_broad_harness.py, timeout 600 env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/gomad/rust/driver/tests -p test_dry_run.py, timeout 600 env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/gomad/rust/driver/tests -p test_dry_run_readiness.py, timeout 600 env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/gomad/rust/driver/tests -p test_live_backend_roles.py, git diff --check, PATH=<stock-Go1.27.1>/bin:$PATH GOCACHE=/home/agent/.cache/go-build-fn232-corpus CARGO_TARGET_DIR=<workspace>/.build-cache/fn232-composition-target CARGO_BUILD_JOBS=2 make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism; exit0 onb15d9b5e0eb0c044612c8fb28db9a701aeea25de; 583 driver tests pass
- PRs: