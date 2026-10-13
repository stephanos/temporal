---
satisfies: [R7]
---
# fn-156-wasm-gomad-productionization.7 Optimize supported profiles and deliver backend documentation

## Description
Integrated MAIN CLI verification (2026-10-11). The focused shared CLI selection passed30 top-level cases (203 including subcases), the WASM entrypoint selection passed3 (9 including subcases), and both package vets passed with all processes reaped. /tmp/fn156-main-minimization-cli-evidence.json SHA2564620b0472648f6ebf8510803e6b48e3262fc51571afc44fd709e3829687ceeca binds commands and raw logs. Clean-installed minimization preparation selects an unchanged default cooperative configuration and private DEFAULT60e65; it will first derive an actual unforced-success seed, then discover the forced failure, detach only private source/compiler/runtime/preparation-cache inputs, minimize and freshly replay. That preparation has not executed a guest and supplies no installed or final-cost acceptance.

Original fn-151 Phase 10: re-measure the corrected main-based artifact, admit tested provider exclusions and immutable engine-code caching, document install/discover/replay/support and measured costs. Native replacement remains a separate owner decision.

The provisional delivery surface adds an opt-in WASM command adapter through the existing Gomad command and Runner owners, preserving native defaults and grammar. Installation/discovery verifies pinned helper/compiler/runtime/model inputs; retained replay must run without original sources or compiler access. Advertised profiles remain bounded by actual integrated qualification; network/storage and cost claims stay pending until their owning gates pass. Do not infer a checkpoint or provider exclusion without its explicit safety/identity/native-default gate.

Source-support repair (2026-10-11): installation validation now rejects unqualified `NetworkV2` alongside v1 network and storage. A red regression demonstrated that Build, Write and Provider admitted a valid cooperative V2 configuration, and Resolve reached helper-file access before qualification rejection. The repaired installation source suite passes four top-level cases and three model-admission subcases; focused vet and formatting/diff checks pass. The tests also preserve valid cooperative runtime identity, path-error precedence and trailing-JSON rejection. Logs are `/tmp/fn156-installation-{baseline,red,green}.log`; isolated source candidate was integrated byte-identically from `/tmp/gomad-fn156-installation.fUJzqi`. No real helper, guest or operator qualification was run by this repair. Task status remains todo until `.6` and the original clean-install/replay/cost/documentation gates are verified.

Compatible minimization CLI source (2026-10-11). The shared external dispatcher now admits `minimize`, forwarding the installed backend to existing Runner minimization with unchanged parser, attempt/byte/output/resume defaults and native characterization. Explicit native-only --toolchain-root, including an empty value, rejects before installation resolution. The isolated CLI batch passed focused shared/operator tests and both vets, and its six owned files were integrated byte-identically. README and CLI guidance advertise only complete standalone cooperative exact target-failure choice-prefix minimization, fixed inputs and fresh full-tape replay; stock observed/cluster/crash-prefix modes remain rejected. Grammar/replay-preflight status2 versus other minimizer/infrastructure status3 is preserved. The positive actual public API gate passed separately, but integrated source checks and clean-installed positive minimization/detached replay remain open; no installed qualification or task completion follows. Handovers `/tmp/fn156-minimization-cli-{summary.md,evidence.json}` retain exact commands and patch SHA256 `25e8b84095a11b0a1813b8fda873a702289eb79b9b3d9769b8ccb381f1a190a5`.

**Touches:** `tools/gomad3/cmd/gomad/**`, `tools/gomad_wasm/cmd/**`, `tools/gomad_wasm/installation/**`, `tools/gomad_wasm/CLI.md`

README/Makefile integration is serialized with the helper owner. Final admission follows `.6`. Gates include installation/path/identity negative tests, native command characterization/architecture checks, an actual clean-install guest failure discovery/inspect/fresh replay, and final measured support/cost documentation.
## Acceptance
Original fn-151 Phase 10 exit gate.

## Done summary
Supported profiles, immutable compile caching and installed discovery/replay are documented (CLI.md, README.md, QUALIFICATION.md) and match behavior; the installed operator gate passes on 9d6012b52b.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9d6012b52b, 7cf3da2110, 77517b9b01
- Tests: installed operator gate, make -C tools/gomad3 validate, evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: