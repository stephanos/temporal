# Next gate correction at a6a28720af

Recommend one corrective implementation task under fn-109 R18/R19 for the remaining gomadtool stdout-report failure class. This is a command-contract defect and 17 actual errcheck findings, with public portable regression routes. It does not resolve the complete lint or native gate.

Read-only research used the project gpt-6-astra/high route at `a6a28720af58fe65cb6fd3bc618ef8102765b50b`. AGENTS, flowctl usage/brief, original ownership and current sources were checked. README and MILESTONES are byte-unchanged from the preceding full read. No Go, lint, source/test edits, task-state or Git mutations were performed by this audit. This receipt is its only write.

## Ownership and current evidence

Fn-109 spec lines 361-376 require ordinary CLI, error-precedence and transaction preservation under R18 and retained complete gates under R19. Task21 Description explicitly implements nothing and sends source gaps to implementation owners. Task23 Touches cover lint routing, not these commands. Admit one corrective R18/R19 task, following the existing corrective-task pattern, and link its evidence into task21. Fn-113.3 keeps authoring, approval and publication ownership; this task changes only the CLI's unchecked report boundary. No duplicate pack-policy owner or new milestone feature is needed.

The conductor's fresh `.flow/tmp/lint-gate-a6a28720af/full-lint.log` reports 302 findings across 55 host packages, with 244 errcheck, 2 exhaustive, 11 forbidigo and 45 staticcheck. The command exited 2 and never reached errortype. The log SHA-256 is `2391ed2c21e151396d6762fb97e7defacc88933ffc1c3ec9496c47411e936251`. Lines 253-598 independently identify the 17 stdout findings below; the larger diagnostic class must remain visible.

## Exact source surface

| Source under tools/gomad3/cmd/gomadtool | Unchecked stdout lines | Contract boundary |
| --- | --- | --- |
| main.go | 178, 193, 206, 237, 318, 360, 406, 408 | Patch/script validation, materialize/regenerate, build key, test success, completed toolchain build |
| compatibility_pack.go | 95, 128, 157, 176, 215, 286, 311 | Discovery, review, regeneration/approval, check, aggregate and per-request qualification |
| boundary.go | 39 | Source-discovered candidates |
| upgrade.go | 67 | Final path after successful dossier publication |

Each currently ignores a real stdout write failure and can return success. CLI.md lines 449-456 assigns output-publication failure status 3. Existing `diagnostic.go:49-52`, `compatibility_pack_refresh.go:125-155`, pin-impact and adapter-regenerate already check output errors. Apply the same documented status at the unchecked boundaries. Preserve format strings, arguments, healthy-output bytes, flag grammar and successful publication.

## Preservation constraints

Terminal success writes can return 3 on error without changing earlier operation statuses. Do not add a stderr fallback, roll back files, remove approvals or alter underlying execution. Both stdout and stderr failing still returns the existing primary status when one exists, otherwise 3 from the new output failure. Existing diagnostic-only stderr sites remain outside this task; checking them merely to discard their error is not a repair.

Two sequences need explicit handling. `qualifyAllCompatibilityPacks:277` currently continues to later requests after an ignored per-request output error. An immediate 3 from `qualifyCompatibilityPackRequest:311` would stop later qualification and could hide its original status 1/2. Keep qualification status separate from report error through this private call boundary. In --all, retain the first report error, continue the original request order and stop only for the original operational failure; that failure retains precedence. After all requests succeed, check the aggregate summary write and return 3 if any report failed. A private two-result helper is sufficient; no new callback or production test seam is needed.

`boundary.go:37-54` prints after discovery and checks its error afterward. Preserve that ordering and primary discovery status; collect output failure separately. Current DiscoverCandidates returns nil candidates on error (`boundary_discovery.go:48-53`), so simultaneous partial-discovery evidence is not currently reachable and must not be fabricated. Preserve both completed toolchain-build report attempts at main.go:405-408; no build operation follows either write.

## Regression plan

Use public `run` and the existing real read-only-file writer at `compatibility_pack_refresh_output_test.go:16-49`. Assert actual EBADF execution before asserting status, pair each with exact healthy bytes, and retain no-output invalid-input/status-2 and operational-failure/status-1 controls. No syscall privilege, descriptor theft or patched-toolchain spoof is required.

Eight sites have direct portable routes: existing build-key, patch-validate and script-validate fixtures (`main_test.go:11,74,85`); stock-source boundary discovery; and pack review, generate-all, generate-approved and check using the real temporary authoring state already assembled in `compatibility_pack_refresh_test.go:100-191`. For the four authoring operations, verify the published requests, reports, packs, approval and generation manifest remain exactly what the successful operation produced despite lost stdout. Materialize/regenerate can additionally adapt the real tiny patch/archive fixtures from `toolchain/patch_test.go:128,196,572,626` without changing production source or pins.

Toolchain/test, capability discovery/qualification and final dossier-success paths require their genuine successful operation inputs. Do not add hooks or weaken host guards to cover those report lines. Record each unexecuted path explicitly; a lint reduction is not end-to-end behavioral proof. Multi-request qualification precedence requires native or otherwise legitimate existing operation evidence before claiming that path verified.

After admission and RED, use named new `TestRunMaintainerOutput...` controls with `go -C tools/gomad3 test -tags test_dep -count=1 ./cmd/gomadtool -run 'TestRunMaintainerOutput|TestRunBuildKey|TestRunPatchValidate|TestRunScriptValidate|TestRunCompatibilityPackRefresh|TestCompatibilityPackPaths'`, plus the conductor's retained portable CLI/authoring families. Compare unfiltered configured lint blocks, expecting only the admitted stdout findings to disappear unless separately explained; rerun required architecture, vet/errortype, check-only validation and fast lint. Preserve every original native/full/formal gate and fn-128 deferral. This recommendation grants no formal SHIP or task completion.
