# fn-172 guest-side cost experiments: guest flame attribution (R1) and A/B variants (R2), Linux/arm64, 2026-10-15 UTC

Diagnostic only: never qualification evidence, and no diagnostic default changes. Measured serially on the quiet
Linux/arm64 host (`Linux 7.0.14 aarch64`, 12 CPUs, cpuinfo CPU architecture 8, implementer 0x61, boot
`ad1ec4fe-2bb8-4c96-bf7f-a714910117e6`, go1.27.1, `required_host_measured=true` in both reports).

## Helpers (merged-tree rebuild, perf-RUNBOOK A1-A3)

| helper | features | sha256 | admission |
| --- | --- | --- | --- |
| diagnostic-profile (A1) | `diagnostic-profile` | `8638b64376f158662e870e9fdff915f312ec83565d1a7b5c8c1c21f607215f96` | `/home/agent/wt/perf-evidence/helpers/diagnostic-profile/release-helper-2416475343/admitted-helper-1363781343/helper-admission.json` |
| guest-profile (A2) | `diagnostic-profile,guest-profile` | `28ca463e93368a6f1d6df8afe9eca7d2ae20e02667d709d0f20462260a385d68` | `/home/agent/wt/perf-evidence/helpers/guest-profile/release-helper-3755332094/admitted-helper-2859023309/helper-admission.json` |
| fuel-off (A3) | `diagnostic-fuel-off,diagnostic-profile` | `a58350ae05b519bf81905b8cba24093853b4e449fa4e2c70fa6d9ca8438f1b63` | `/home/agent/wt/perf-evidence/helpers/fuel-off/release-helper-3858144041/admitted-helper-1341971232/helper-admission.json` |

Engine revisions: `cranelift-fuel-nan-canonical-nounwind-v2` (+ `-guest-profile` suffix for the sampler build).
Model `sha256:42c1f68b…6e46`, runtime identity `1d17508f…7f7e`.

## R1 (fn-172.2): guest flame profile and attribution

Sampler at 1 ms, max 10 000 samples (none truncated). Module `./tests` sha256 `21043d63…b6f`. Every sampled run's
evidence equals its unsampled pair (`evidence_identical=true`). Self samples are bucketed by the first matching frame
from leaf to root (third-party frames such as sqlite defer to their caller's bucket; "other" means no frame matched).

Source commit `6e5dec62d3dad3078737a93aa9a02aaaf6fd57ae` (dirty=False), helper admission `/home/agent/wt/perf-evidence/helpers/guest-profile/release-helper-3755332094/admitted-helper-2859023309/helper-admission.json`.

| selection | seed | samples (truncated) | gc | scheduler | allocation | test | server | runtime | stdlib | other | evidence identical | flame sha256 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| workflow | 0 | 312 (False) | 54 (17.3%) | 8 (2.6%) | 25 (8.0%) | 0 (0.0%) | 67 (21.5%) | 91 (29.2%) | 67 (21.5%) | 0 (0.0%) | True | `dc0b9d8fbaf2e645…` |
| workflow | 1000003 | 326 (False) | 43 (13.2%) | 12 (3.7%) | 31 (9.5%) | 0 (0.0%) | 71 (21.8%) | 109 (33.4%) | 60 (18.4%) | 0 (0.0%) | True | `51e83f7e3e9b1e96…` |
| workflow | 2000006 | 293 (False) | 34 (11.6%) | 12 (4.1%) | 25 (8.5%) | 1 (0.3%) | 71 (24.2%) | 92 (31.4%) | 58 (19.8%) | 0 (0.0%) | True | `3063c69ce6462246…` |
| activity | 0 | 1012 (False) | 73 (7.2%) | 52 (5.1%) | 75 (7.4%) | 0 (0.0%) | 299 (29.5%) | 312 (30.8%) | 201 (19.9%) | 0 (0.0%) | True | `2fa8230ff3faf686…` |
| activity | 1000003 | 1060 (False) | 107 (10.1%) | 53 (5.0%) | 79 (7.5%) | 6 (0.6%) | 301 (28.4%) | 299 (28.2%) | 215 (20.3%) | 0 (0.0%) | True | `068f4a5553956af3…` |
| activity | 2000006 | 1410 (False) | 109 (7.7%) | 87 (6.2%) | 118 (8.4%) | 2 (0.1%) | 410 (29.1%) | 429 (30.4%) | 255 (18.1%) | 0 (0.0%) | True | `008d075102da6c0c…` |

- workflow seed 0: `/home/agent/wt/perf-evidence/fn172/guest-profile/guest-profile-1751299915/evidence/workflow-seed0-guest.firefox.json` sha256 `dc0b9d8fbaf2e645e48b4b8bcde65780ae891334b91aac6bd3156c2932681e1c`; helper `28ca463e93368a6f1d6df8afe9eca7d2ae20e02667d709d0f20462260a385d68`, module `21043d63aedc1741dfe07a75b2271d71be4d31fe2b07d565170551fbc3c31b6f`, engine `cranelift-fuel-nan-canonical-nounwind-v2-guest-profile`
- workflow seed 1000003: `/home/agent/wt/perf-evidence/fn172/guest-profile/guest-profile-1751299915/evidence/workflow-seed1000003-guest.firefox.json` sha256 `51e83f7e3e9b1e96b131801d78a550db9766c22649b651e7620b067fff719944`; helper `28ca463e93368a6f1d6df8afe9eca7d2ae20e02667d709d0f20462260a385d68`, module `21043d63aedc1741dfe07a75b2271d71be4d31fe2b07d565170551fbc3c31b6f`, engine `cranelift-fuel-nan-canonical-nounwind-v2-guest-profile`
- workflow seed 2000006: `/home/agent/wt/perf-evidence/fn172/guest-profile/guest-profile-1751299915/evidence/workflow-seed2000006-guest.firefox.json` sha256 `3063c69ce6462246eca603d867ace1d7cf84034d68b9ccb811fbbcf420397391`; helper `28ca463e93368a6f1d6df8afe9eca7d2ae20e02667d709d0f20462260a385d68`, module `21043d63aedc1741dfe07a75b2271d71be4d31fe2b07d565170551fbc3c31b6f`, engine `cranelift-fuel-nan-canonical-nounwind-v2-guest-profile`
- activity seed 0: `/home/agent/wt/perf-evidence/fn172/guest-profile/guest-profile-1751299915/evidence/activity-seed0-guest.firefox.json` sha256 `2fa8230ff3faf6867bedfe9417e3ecf471e2e2493c05803b1cb589f6eff51e5e`; helper `28ca463e93368a6f1d6df8afe9eca7d2ae20e02667d709d0f20462260a385d68`, module `21043d63aedc1741dfe07a75b2271d71be4d31fe2b07d565170551fbc3c31b6f`, engine `cranelift-fuel-nan-canonical-nounwind-v2-guest-profile`
- activity seed 1000003: `/home/agent/wt/perf-evidence/fn172/guest-profile/guest-profile-1751299915/evidence/activity-seed1000003-guest.firefox.json` sha256 `068f4a5553956af39a8c0c23b49772d54d74097d680c20b9c064d72ddc353ffb`; helper `28ca463e93368a6f1d6df8afe9eca7d2ae20e02667d709d0f20462260a385d68`, module `21043d63aedc1741dfe07a75b2271d71be4d31fe2b07d565170551fbc3c31b6f`, engine `cranelift-fuel-nan-canonical-nounwind-v2-guest-profile`
- activity seed 2000006: `/home/agent/wt/perf-evidence/fn172/guest-profile/guest-profile-1751299915/evidence/activity-seed2000006-guest.firefox.json` sha256 `008d075102da6c0c2bfcec82e303da7dddb19ad4fc8e14818d7ad1740dde35bb`; helper `28ca463e93368a6f1d6df8afe9eca7d2ae20e02667d709d0f20462260a385d68`, module `21043d63aedc1741dfe07a75b2271d71be4d31fe2b07d565170551fbc3c31b6f`, engine `cranelift-fuel-nan-canonical-nounwind-v2-guest-profile`

Scope: weighted guest stack samples; epoch function-entry/loop bias; host-call ticks excluded; no timestamp-gap charge; sampled wall times are not baselines

Limits: scheduler-frame share does not quantify all unwind/rewind work; fuel checks are distributed across ordinary guest frames

Rerun (cwd `/home/agent/wt/merge/tools/gomad`): `GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go GOMAD_WASM_GUEST_HELPER_ADMISSION=/home/agent/wt/perf-evidence/helpers/guest-profile/release-helper-3755332094/admitted-helper-2859023309/helper-admission.json GOMAD_WASM_GUEST_PROFILE_ROOT=/home/agent/wt/perf-evidence/fn172/guest-profile GOMAD_WASM_TEST_HELPER=/home/agent/wt/perf-evidence/helpers/guest-profile/release-helper-3755332094/admitted-helper-2859023309/gomad3-wasmhost env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 60m -run '^TestDiagnosticGuestProfile$' ./internal/backends/wasm/wasi`

**Activity selection reading:** Temporal server code ≈ 28-30 % of guest self samples, Go runtime (non-GC, non-scheduler,
non-allocation: memmove, maps, hashing, pcvalue/stack walks) ≈ 28-31 %, stdlib ≈ 20 %, GC mark/sweep ≈ 7-10 %,
allocation ≈ 7-8 %, scheduler ≈ 5 %, test package < 1 %. The flame files open at profiler.firefox.com.

Classifier fix (commit `6e5dec62d3`): the first C1 run (commit `15b491281c`, kept as
`/home/agent/wt/perf-evidence/fn172/guest-profile.interrupted-20261014T180721/guest-profile-2726234009`)
reported 0 % server/test, because the real wasm name section spells `go.temporal.io_server_…` and Go 1.27 spells
`runtime.scanObject`. The table above is the rerun after the fix, with the same helper and module.

## R2 (fn-172.3): GC, GOWASM and fuel-off A/B

Seeds 0, 1000003, 2000006 × 5 warm repetitions, unsampled diagnostic-profile helper, baseline first and candidate order
rotated. Modules: baseline `21043d63aedc1741dfe07a75b2271d71be4d31fe2b07d565170551fbc3c31b6f`, `GOWASM=satconv,signext`
`ea256df1b72665ee2749c76a4415d73a7093ab775a14ebd7cde100c0fd0bb9af` (`GOEXPERIMENT=nogreenteagc` for both). Variants:
`GOGC=200`, `GOGC=400`, `GOGC=off` + `GOMEMLIMIT` 75 % of memory, GOWASM, fuel-off helper.

| selection | variant | seed | execute ms median (min-max) | residual ms median (min-max) | reps | evidence identical | failed |
| --- | --- | --- | --- | --- | --- | --- | --- |
| workflow | baseline | 0 | 409.8 (397.0-628.0) | 377.4 (365.7-562.2) | 5/5 | True | False |
| workflow | gc200 | 0 | 372.9 (362.0-585.4) | 342.3 (330.8-456.7) | 5/5 | False | False |
| workflow | gc400 | 0 | 345.3 (318.8-363.8) | 315.8 (289.8-332.9) | 5/5 | False | False |
| workflow | gc-off | 0 | 308.0 (302.2-321.6) | 278.1 (273.1-290.7) | 5/5 | False | False |
| workflow | gowasm | 0 | 404.7 (379.1-429.7) | 373.1 (349.0-394.8) | 5/5 | False | False |
| workflow | fuel-off | 0 | 369.0 (356.7-412.6) | 337.6 (325.3-372.9) | 5/5 | True | False |
| workflow | baseline | 1000003 | 388.9 (363.1-625.5) | 357.5 (331.8-581.7) | 5/5 | True | False |
| workflow | gc200 | 1000003 | 373.0 (366.4-461.6) | 339.5 (335.5-424.5) | 5/5 | False | False |
| workflow | gc400 | 1000003 | 334.8 (314.7-409.4) | 304.4 (285.2-371.9) | 5/5 | False | False |
| workflow | gc-off | 1000003 | 303.8 (299.6-389.5) | 273.6 (269.6-355.3) | 5/5 | False | False |
| workflow | gowasm | 1000003 | 404.0 (392.2-417.1) | 372.4 (361.0-381.1) | 5/5 | False | False |
| workflow | fuel-off | 1000003 | 330.5 (322.0-344.9) | 299.5 (291.4-313.1) | 5/5 | True | False |
| workflow | baseline | 2000006 | 391.7 (368.9-867.2) | 360.5 (339.2-756.7) | 5/5 | True | False |
| workflow | gc200 | 2000006 | 404.6 (376.4-420.4) | 372.3 (344.8-386.6) | 5/5 | False | False |
| workflow | gc400 | 2000006 | 366.4 (331.9-480.3) | 328.1 (301.7-435.0) | 5/5 | False | False |
| workflow | gc-off | 2000006 | 325.1 (304.9-530.5) | 294.4 (274.4-489.4) | 5/5 | False | False |
| workflow | gowasm | 2000006 | 436.7 (395.5-455.6) | 403.5 (362.9-415.0) | 5/5 | False | False |
| workflow | fuel-off | 2000006 | 342.2 (322.2-366.7) | 308.3 (290.4-325.7) | 5/5 | True | False |
| activity | baseline | 0 | 1521.2 (1476.7-2013.2) | 1337.8 (1290.2-1746.5) | 5/5 | True | False |
| activity | gc200 | 0 | 1478.6 (1354.3-1611.4) | 1290.7 (1183.1-1412.8) | 5/5 | False | False |
| activity | gc400 | 0 | 1435.2 (1293.7-1553.0) | 1256.2 (1126.0-1358.6) | 5/5 | False | False |
| activity | gc-off | 0 | 1266.9 (1236.6-1859.9) | 1094.2 (1069.5-1642.8) | 5/5 | False | False |
| activity | gowasm | 0 | 1515.4 (1397.3-1598.1) | 1329.1 (1214.7-1402.5) | 5/5 | False | False |
| activity | fuel-off | 0 | 1445.6 (1223.9-1515.9) | 1263.7 (1046.5-1324.2) | 5/5 | True | False |
| activity | baseline | 1000003 | 1711.9 (1545.8-2019.1) | 1520.1 (1363.1-1800.7) | 5/5 | True | False |
| activity | gc200 | 1000003 | 1537.2 (1337.3-1613.9) | 1354.9 (1168.7-1416.7) | 5/5 | False | False |
| activity | gc400 | 1000003 | 1598.3 (1281.0-1920.0) | 1385.8 (1105.4-1713.4) | 5/5 | False | False |
| activity | gc-off | 1000003 | 1436.2 (1230.8-1684.1) | 1252.1 (1058.3-1487.5) | 5/5 | False | False |
| activity | gowasm | 1000003 | 1554.2 (1398.6-1598.1) | 1372.5 (1208.2-1413.9) | 5/5 | False | False |
| activity | fuel-off | 1000003 | 1451.1 (1410.1-1505.9) | 1249.2 (1218.2-1319.2) | 5/5 | True | False |
| activity | baseline | 2000006 | 1603.3 (1470.7-1929.3) | 1413.8 (1292.3-1718.5) | 5/5 | True | False |
| activity | gc200 | 2000006 | 1467.1 (1345.4-1998.2) | 1277.1 (1173.5-1753.2) | 5/5 | False | False |
| activity | gc400 | 2000006 | 1547.6 (1266.9-1602.6) | 1357.1 (1097.4-1402.0) | 5/5 | False | False |
| activity | gc-off | 2000006 | 1347.3 (1219.5-1522.6) | 1155.1 (1043.2-1303.5) | 5/5 | False | False |
| activity | gowasm | 2000006 | 1521.9 (1443.0-1939.1) | 1330.6 (1270.3-1696.8) | 5/5 | False | False |
| activity | fuel-off | 2000006 | 1443.6 (1301.6-1619.9) | 1247.1 (1121.8-1406.4) | 5/5 | True | False |

| selection | variant | recommendation | follow-up |
| --- | --- | --- | --- |
| workflow | baseline | **control** | none; diagnostic control |
| workflow | gc200 | **reject** | explicit GC environment and memory policy decision; requalify affected selections |
| workflow | gc400 | **reject** | explicit GC environment and memory policy decision; requalify affected selections |
| workflow | gc-off | **reject** | bounded GOMEMLIMIT policy and GC decision; requalify affected selections |
| workflow | gowasm | **reject** | production preparation currently forces GOWASM=; adopting requires preparation and build identity changes plus requalification |
| workflow | fuel-off | **investigate** | explicit fuel/resource containment and engine identity decision; diagnostic helper is not a release configuration |
| activity | baseline | **control** | none; diagnostic control |
| activity | gc200 | **reject** | explicit GC environment and memory policy decision; requalify affected selections |
| activity | gc400 | **reject** | explicit GC environment and memory policy decision; requalify affected selections |
| activity | gc-off | **reject** | bounded GOMEMLIMIT policy and GC decision; requalify affected selections |
| activity | gowasm | **reject** | production preparation currently forces GOWASM=; adopting requires preparation and build identity changes plus requalification |
| activity | fuel-off | **investigate** | explicit fuel/resource containment and engine identity decision; diagnostic helper is not a release configuration |

Decision rule: adopt only when all seeds have identical successful evidence and median execute gain strictly exceeds both baseline and candidate run-to-run ranges; adoption remains a follow-up decision

Equality: "exact choice, WASI, network, stdout/stderr, outcome and virtual time; only GC environ reply outputs masked; module/environment/helper/engine identities separate"

Build settings: {"GOARCH": "wasm", "GOEXPERIMENT": "nogreenteagc", "GOOS": "wasip1", "baseline_GOWASM": "", "variant_GOWASM": "satconv,signext"}

Scope: unsampled diagnostic-profile timing; full transcript retention for exact behavior comparisons; residual includes guest plus Wasmtime/trampoline work, not guest-only CPU

### Written recommendation per variant

- **GOGC=200 / GOGC=400 / GOGC=off: reject (as an automatic change).** Median execute moves by -10..+3 % (GOGC=200), -4..-16 % (GOGC=400) and
  -16..-25 % (gc-off: workflow -17..-25 %, activity -16..-17 %), but evidence is **not identical**. The choice trace changes (e.g. workflow
  seed 0: 554 976 → 545 280 bytes at GOGC=200), so GC pacing changes guest scheduling decisions. That is a behaviour
  change, not noise. Follow-up decision: an explicit GC environment and memory policy (with a bounded `GOMEMLIMIT` for gc-off),
  accepting new choice traces and requalifying the affected selections.
- **GOWASM=satconv,signext: reject.** No gain (workflow -1..+11 %, activity -9..0 %, within spread), and the WASI transcript
  differs (the choice trace is identical). Production preparation forces `GOWASM=` (`backend/provider.go`), so adopting it
  would also need a preparation and build-identity change plus requalification.
- **Fuel-off: investigate.** Evidence is identical; median execute is 5-15 % lower (workflow 369/331/342 vs 410/389/392 ms;
  activity 1446/1451/1444 vs 1521/1712/1603 ms). The gain does not exceed the run-to-run range on every seed, so it is not
  adoptable under the decision rule. Follow-up: an explicit fuel/resource-containment and engine-identity decision. The
  diagnostic helper is not a release configuration.

## Evidence paths and rerun

- Guest profile: `/home/agent/wt/perf-evidence/fn172/guest-profile/guest-profile-1751299915/guest-attribution-report.json`, flame files and per-run attribution under `/home/agent/wt/perf-evidence/fn172/guest-profile/guest-profile-1751299915/evidence/`.
- Variants: `/home/agent/wt/perf-evidence/fn172/variants/guest-variants-2013340609/guest-variants-report.json`, raw per-run evidence under `/home/agent/wt/perf-evidence/fn172/variants/guest-variants-2013340609/variants/`.
- Logs: `/home/agent/wt/measure-logs/fn172-C1-guest-profile.log`, `fn172-C2-variants.log`.
- Rerun (cwd `/home/agent/wt/merge/tools/gomad`, perf-RUNBOOK C1/C2), or
  `RERUN="fn172-C1-guest-profile fn172-C2-variants" ONLY="fn172-C1-guest-profile fn172-C2-variants" /home/agent/wt/measure-all.sh`:
  - C1: `GOMAD_WASM_GUEST_PROFILE_ROOT=/home/agent/wt/perf-evidence/fn172/guest-profile GOMAD_WASM_TEST_HELPER=<A2>/gomad3-wasmhost GOMAD_WASM_GUEST_HELPER_ADMISSION=<A2>/helper-admission.json env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 120m -run '^TestDiagnosticGuestProfile$' -v ./internal/backends/wasm/wasi`
  - C2: `GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION=/home/agent/wt/perf-evidence/helpers/diagnostic-profile/release-helper-2416475343/admitted-helper-1363781343/helper-admission.json GOMAD_WASM_FUEL_OFF_HELPER=/home/agent/wt/perf-evidence/helpers/fuel-off/release-helper-3858144041/admitted-helper-1341971232/gomad3-wasmhost GOMAD_WASM_FUEL_OFF_HELPER_ADMISSION=/home/agent/wt/perf-evidence/helpers/fuel-off/release-helper-3858144041/admitted-helper-1341971232/helper-admission.json GOMAD_WASM_GUEST_PROFILE_ROOT=/home/agent/wt/perf-evidence/fn172/variants GOMAD_WASM_TEST_HELPER=/home/agent/wt/perf-evidence/helpers/diagnostic-profile/release-helper-2416475343/admitted-helper-1363781343/gomad3-wasmhost env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 180m -run '^TestDiagnosticGuestVariants$' ./internal/backends/wasm/wasi`

## fn-172.1 helper gates (2026-10-15, `measure` `6e5dec62d3`, `CARGO_BUILD_JOBS=4`)

`cargo +1.94.1 fmt --check`; `test --release --locked` and `clippy --release --locked --all-targets -- -D warnings` for
the default, `guest-profile` and `diagnostic-fuel-off` builds (separate target dirs): all exit 0, 676 Rust tests passed,
0 failed (logs `/home/agent/wt/measure-logs/rust-gates/`). `cargo tree -e normal` for the release build lists no
`profiling`/`ittapi` crate. Relevant tests: `release_refuses_guest_profile_before_loading_or_execution`,
`guest_profile_is_symbolized_bounded_and_preserves_evidence_on_write_failure`,
`guest_profile_drops_epochs_elapsed_inside_host_calls`, `guest_profile_rejects_invalid_bounds_and_unknown_fields`,
`diagnostic_fuel_off_reports_distinct_revision_and_preserves_offered_fuel` (`wasmhost/tests/engine.rs`). Real-host use: the
A2 guest-profile helper wrote symbolized Firefox profiles for all six C1 runs with evidence identical to unsampled runs,
and the A3 fuel-off helper ran all 30 fuel-off runs to `exit` with identical evidence.
