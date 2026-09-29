---
satisfies: [R3, R4]
---
# fn-101-gomad-f7-any-functional-test-and-ci.6 Make core-linux pass: platform-neutral required probes, measured linux expectations, platform-aware host tier

## Description
From fn-101.5's fork CI findings (run 36485805062; per-seed linux reports in run 36470738437): (1) remove `stdlib.os.getwd` from the functional suites' required_probes in temporal.json and tests.generator.json (it fires only from darwin's os init, not from anything the suites depend on); keep only probes for modeled operations the suites actually use on both platforms, verified against both platforms' reports; (2) set frontend-system-info's linux/amd64 expectation to `intermittent` with a finding naming F3 and the observed nondeterministic seed; (3) make the linux host tier's darwin-assuming tests (TestGenerateRendersDescriptorConsumers, upgrade Run* tests) select or skip by platform correctly (no weakening on darwin); (4) push, dispatch the fork workflow, and from the linux Temporal requalification tighten every linux expectation that qualified on both seeds in the run (e.g. user-timers, activity-batch-cancel, the F6 slice) from `intermittent` to `qualified` only where the evidence shows both seeds qualified; anything still diverging on linux stays with a finding and is reported to the conductor as a determinism finding; (5) iterate until core-linux passes.

## Acceptance
- fork gomad3 workflow: all four jobs pass
- every linux expectation is either qualified (measured) or intermittent/unrepeatable with a finding

## Done summary
core-linux now passes, and so do the other three gomad3 jobs. Fork run 36497204705 at adb0b1b136 succeeded on host-tools-linux, core, core-linux, and temporal-integration. The earlier fork run 36493869196 at cfba28756a also passed all four jobs; its linux report is the basis for the tightened expectations.

Changes:
- Dropped `stdlib.os.getwd` from the ten functional suites' required_probes in temporal.json and tests.generator.json, and regenerated tests.json. Only darwin's os package init calls Getwd. The remaining probes (net.interfaces, os.newfile, os.openfile) appeared in every linux report of run 36470738437 and in every darwin run.
- frontend-system-info on linux/amd64 is now `intermittent` with the F3 finding. Seed 17 was `nondeterministic` in run 36466396209, and both seeds qualified in 36476712810, 36485805062, 36493869196, and 36497204705.
- Host tier: the upgrade test fixture now declares the toolchain's qualified platforms (`gomadversion.SupportedPlatforms`), not darwin only. TestGenerateRendersDescriptorConsumers runs make with `--no-print-directory`. It failed only because GNU make's `-C` exports `-w`, which reproduces on darwin with MAKEFLAGS=w (red before the fix, green after). The linux host tier now gates again: `continue-on-error` is removed.
- Tightened linux/amd64 from `intermittent` to `qualified` for user-timers, activity-batch-cancel, and the ten F6 `functional-*` suites. Run 36493869196 qualified all of them on both seeds with exact replay: 18 supported, 0 failed, 36 replays, 0 diverged. The linux jq gate now requires those suites to be qualified. It allows `(supported + failed) == 18` with `supported >= 17`, so only the intermittent frontend probe may fail.
- Recorded linux per-suite metrics (seconds, Campaign MB, trace MB, decisions) for F3, F5, and F6 in GOMAD_MILESTONES.md F7, and updated the integration README.

Determinism findings for the conductor: no linux divergence remains in these runs. The only open item is the historic frontend seed-17 nondeterminism from run 36466396209, which the frontend's F3 expectation still covers.

stage: impl-review - ran [codex fan-out, 3 draws, all SHIP -> merged SHIP]
## Evidence
- Commits: cfba28756a8ef008cf9e5c7875ef9a2e74dbfb0d, adb0b1b13676e6af7c20b01a49e5eb3ed13e3aec
- Tests: baseline: none (spec defines no Quick commands), make -C tools/gomad3 validate, go test -count=1 -tags test_dep ./upgrade ./toolchain/version ./qualification/set/... (tools/gomad3), MAKEFLAGS=w go test -run TestGenerateRendersDescriptorConsumers ./toolchain/version (red before fix, green after), make gomad3-integration-test, fork run 36493869196 at cfba28756a: all four jobs success (linux report basis for tightening), fork run 36497204705 at adb0b1b136: all four jobs success with tightened linux expectations
- PRs: