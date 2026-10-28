---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.15 Port remaining model shell scripts to xtask/Makefile; only the bootstrap stays shell

## Description
Owner request (2026-10-25): "is any existing task cleaning up temporal/model/scripts ie making them Rust / Makefile scripts?" then "file a task after fn-254.13 and .10". fn-247 deliberately kept Bash as orchestration. fn-254.13 and .10 move and split the scripts but keep them as shell. This task finishes the one-language rule for model tooling: logic goes into Rust (`xtask` / `umpire-guest-driver` subcommands), thin wrappers become Makefile recipes, and the only shell left is the minimal xtask bootstrap.

Runs after .13 (framework copies in `model/umpire/scripts/`, Temporal copies in `model/scripts/`), .10 (`cargo xtask gate` composition) and .14 (mise checks already gone, so the ports do not bring them back). Start from whatever script set those tasks leave. The current executable scripts are fetch-tlc, guest, guest-freshness, proto-go, proto-tools, retired-names, time-gate and xtask. The current control/build scripts across the two roots are test-capabilities, test-time-gate, test-model-validate, test-model-wiring, tests/bootstrap, tests/proto-tools, tests/proto-go, tests/build-guests, tests/build-guests-controls and tests/guest-roots. Refresh this inventory after .17/.16 and map each actual script and its behavioral controls to its replacement or an explicit obsolete-case reason.

**Size:** M
**Files:** `model/umpire/scripts/**`, `model/scripts/**`, `model/umpire/dev/xtask/**`, `model/umpire/dev/guest-driver/**`, `model/dev/xtask/**` (Temporal extension), root `Makefile`, the framework entry point from .10, and READMEs or `GUEST_BUILDS.md` that name the scripts.
**Touches:** [model/umpire/scripts/**, model/scripts/**, model/umpire/dev/xtask/**, model/umpire/dev/guest-driver/**, model/dev/xtask/**, Makefile, model/umpire/Makefile, model/Cargo.toml, model/Cargo.lock, model/umpire/Cargo.toml, model/umpire/Cargo.lock, model/README.md, model/DESIGN.md, model/GUEST_BUILDS.md, model/umpire/**/README.md, .plans/UMPIRE_MODULES.md, model/umpire/backends/tlc/src/runner.rs, model/umpire/hosts/go/runner/fake_test.go, model/umpire/reference/kettle/kettle_test.go]

### Approach
- **Into Rust (xtask subcommands, framework or Temporal side to match .9's classification):** `fetch-tlc` (call the mise provider from .16 and retain runtime JAR admission; no native downloader/extractor), `proto-go`/`proto-tools` (run `protoc` and the pinned generators through `mise exec`), `time-gate` (the timing and status mechanism belongs to `cargo xtask gate`, decided in .9), and `retired-names`. Owner allows breaking changes, so rewrite the retired-name patterns in Rust `regex` syntax. This reverses fn-247's keep-`grep -E` decision; flag it in the done summary.
- **Into the Makefile or direct invocation:** `guest.sh` and `guest-freshness.sh` if they are still thin wrappers around `umpire-guest-driver`. Callers run the driver directly.
- **Shell controls into Rust integration tests:** all behavioral controls from the refreshed inventory above, including bootstrap, proto-tools, build-guests-controls, guest-roots and model-wiring controls. Keep only the cases that check real behavior: refusals, status propagation, cleanup and concurrency. Do not rebuild fake-`mise` stubs for checks that .14 removed.
- **Stays shell:** `xtask.sh` bootstrap only (fn-247: build xtask before guest wrapper flags are set). Keep it as small as possible.

### Constraints
- No new Cargo dependencies beyond fn-210 R18 admission. Hash admission reuses existing support; provisioning belongs to mise under .16. Do not retain `tar`/`flate2` solely for a removed installer; no clap.
- Framework xtask stays free of Temporal names (fn-254 R7 standalone control still passes).
- Byte-for-byte compatibility with the old script output is not required.


### Source readiness after .17 closure (2026-10-27)

The integrated tree still has29active shell paths. Committed source snapshot
`7cf05d1c53` reduces them to six: two bootstraps and four deliberately retained
control shells. This is preparation, not the final two-bootstrap acceptance.
Exact source object IDs, current/prepared inventories and remaining adaptation
gaps are recorded in `task15/readiness-20261027.json` under this spec's artifacts.
Use committed objects selectively; dirty source-assembly is not an input.

Preserve .17's owned timing/cleanup/lease/count contracts and authoritative
`test-run --package` route; prepared timing code still has raw spawn/status paths.
Preserve final .16 provider interfaces and runtime admission. Native Make controls
must cover current prerequisites/bootstrap/quick/CI-floor cases, and all four
remaining shell controls need native coverage before deletion. Task execution
continues to require committed .16 closure; this read-only report advances no dependency.

The detailed `task15/script-control-map-60f1c762.json` pins all29script blobs,
caller references, behavioral vectors and31case mappings for the four remaining
controls to the frozen .16 candidate. It identifies adaptation gaps and retains
`native_preparation_validated=false` and `dependency_advancement=false`; compile,
runtime validation and final acceptance mapping still follow committed .16 closure.

### Joined-source caller scope (2026-10-28)

Workspace manifests and locks register the already-admitted fn-210 R18 Rust regex dependency. The module-map xtask ownership row and maintained design documentation describe the native entry points. TLC and Go changes outside xtask are deleted-script diagnostic string replacements only. These paths complete the original caller-cleanup acceptance; they add no Model or Go behavior.
## Acceptance
- [x] `find model -name '*.sh' -not -path '*/target/*' -not -path '*/archived/*' -not -path '*/design/*'` lists only the xtask bootstrap script(s); the archived and design probes are out of scope.
- [x] Every removed script's job is reached through `cargo xtask <cmd>`, a guest-driver subcommand or a Makefile recipe. Root `Makefile`, the framework gate and the docs call the new forms, and an `rg` check finds no references to deleted scripts in active source callers, maintained tests or maintained user/developer documentation. Preserve historical specs, completion records, review artifacts and archived/design evidence that name the former scripts.
- [x] A complete current script-to-replacement mapping accounts for the behavioral controls the shell tests covered. Their Rust replacements pass; any control dropped as moot is listed with its reason.
- [x] The retired-name check runs in Rust and still catches each retired name with a planted file.
- [x] The framework standalone control (.11) and the focused xtask, guest-driver and TLC tests pass. The full gate runs once at the end, as usual for fn-254.
## Throughput graph consumer

Run after .17 and .16. Port leaf implementations behind .17's established native stage interfaces; keep one authoritative scheduler, stage identities, cleanup/resource contracts and framework-only task resolution. Do not rebuild a separate shell or Rust task graph during the port.

## Done summary
Ported all29 baseline shell paths:27 removed, only the framework xtask bootstrap and system forwarding bootstrap remain. Framework/Temporal native commands, owned timing, guest routes/builds, protobuf provisioning/generation/freshness, capability and Make controls replace every removed job. Maintained callers/docs use native routes; historical/deferred migration records remain preserved. Retired-name matching deliberately changes from external grep/POSIX ERE to Rust byte regex under owner approval.

Validation: current affected package runs pass159 framework xtask,38guest-driver and39Temporal xtask tests (four ignored exact Temporal child entries); TLC34 passed on unchanged TLC implementation. Both affected workspaces pass strict all-target/all-feature Clippy and formatting. Each demonstrated review regression was reproduced before its fix: empty framework target, SIGKILL during actual timing benchmark, and wrong provisioning hint. Full source-only standalone proof ran once on exact source9babff6c815beb7cff92d7001f647ddca19663bf:17 stages,938discovered/935executed/3ignored including four passed and one ignored doctest; parent counts are not double charged, source seals match and all3800observed owned PIDs are gone. Exact909prior framework identities are retained through four documented renames; all101mapped controls compile. Later review fixes carry current focused package/regression/lint evidence, not relabelled full-gate counts.

Review: initial correctness/contracts returned SHIP; integration hit the600-second outer limit and was explicitly recorded failed. Supported finalization retained those completed reviews without inventing a third verdict. After fixes, fresh integration review returned SHIP on89c99a82c0523e0598d88d47a38685aaffbd85b5 (ClaudeFable5.1 high, different family from the Codex writer), confirming R7 met and accepting the scoped proof/follow-up evidence. Both finalized rounds and raw reports are committed. Optional helper consolidation and preexisting empty-target consistency are follow-ups. Repeated-control cost and interruption-safe realistic Model-content-edit/final warm measurements remain explicit task8 obligations; its description now carries those requirements. No broad combined Model gate or Darwin acceptance is claimed; migration stays deferred.

Proof custody: CAS manifest68208a5aa0e5df79783d97b740a07da3f0808dce4935aa076450630a5183b610 independently verifies4801members, including exact original parent binaries before later rebuild and both recorded/private driver identities. No cache cleanup is claimed by that receipt. All completed helper worktrees/branches were retired normally; preserved unfinished donor branches and shared caches remain protected. Source and closure integrate directly on umpire; no remote writes.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 642aab42dddc650ce2ac634728a1ec3a28c64af8, 2e85512bbb66ceb01ff9671494b11fa36904259a, 63f6ae8f8c732e3419a1247727dffa1f3118a1ed, 0d3efede39bcf7b0ea8da45c4db37e930f35b874, 6d299d6f3496457e9bd20473140d6a34062dac9d, 60ca64abac57dd7bf99f42390571b8be871a3973, 8c1f6b908b1c24a9757aeb4f074701f994bcc710, b3f1611a5a3714d7eabd63b6e6bfde1affd5e769, 3b61dd55e4b96a2d543fe5fd8aa5d1885d6c1a7d, adc8cd50600a9a8bf074471605628a5302179ee4, 31b37384e906556dcc2fc7f1ce3be569e7b182c9, 86a81cb393406e1d47b70d4041f5cb852d6db9b1, d6055e0235e7dd5fa4ebea99d031e1a513ea7148, 2ec41cad138afb7fa35e2e3f5205238325c628aa, ce6fe1f1f42520a7bc27e3b7a96f70dac49a9425, ff8b3169323074ccaf843ef3da7b566a6ddafd8f, 5342e3e27058a95826e0a4e01da2e8f06eb71fb0, 4a570dd4e6431139295fd03e4b92576db3805011, adb403a07eaa37d7fd75c2f2d75f4fe68078ce86, ef047f09067242421dc3a63d7e950cdaf4c17a52, 25901aad57c8f9332b5aec6a2e9e2dcbf5a2cae6, 4fd176c2561fa968baa1dd2ce652714e67e64617, 4d980c9acdb6654b7d460e0aaf87b92aa14709a4, 9babff6c815beb7cff92d7001f647ddca19663bf, c178882dd0a492f61d0c1c7c1a873d803c6b85cd, e0a29b99da3127f5973cd7af5d23744428e7ee33, 900dbe12cccd29e59172c343413491549a858a90, 3ca2900f67140552c1fc466949449bba0a9bd91a, 4f1a47b03b5398178684b5f7975e75aef7d4d9bf, d7fb8f760de82d6a15435e5a7c9b89be24cfb82f, 6d2512d5c13e15d565cb6da86ffc69b2c42f91c9, 0aa9dfe9f32a0535f4d8969513ecdd06d9526d3f, 903cda34c6270da6339183ecaee748c4fe58e6c8, 89c99a82c0523e0598d88d47a38685aaffbd85b5, 4708ded1586bba99da782b019b4418e2dc0f5417
- Tests: Framework xtask test-run --package xtask:159 passed, Framework xtask test-run --package umpire-guest-driver:38 passed, Framework xtask test-run --package temporal-xtask in model workspace:39 passed,4ignored helper entries, Framework xtask test-run --package umpire-tlc:34 passed on unchanged TLC source, Both workspace affected packages cargo clippy --all-targets --all-features -- -D warnings; both cargo fmt --all -- --check, Native timing SIGKILL / empty framework target / generator hint RED then GREEN controls, Guarded standalone9bab:17 stages,938discovered935executed3ignored, source/PID/count checks accepted, Exact baseline identity retention and current scoped inventories;101mapped controls, Cross-family reviews:correctness SHIP,contracts SHIP,integration follow-up SHIP, Proof custody4801members independently verified before and after parent rebuild
- PRs: