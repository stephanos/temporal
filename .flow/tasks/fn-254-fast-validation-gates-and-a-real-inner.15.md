---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.15 Port remaining model shell scripts to xtask/Makefile; only the bootstrap stays shell

## Description
Owner request (2026-10-25): "is any existing task cleaning up temporal/model/scripts ie making them Rust / Makefile scripts?" then "file a task after fn-254.13 and .10". fn-247 deliberately kept Bash as orchestration. fn-254.13 and .10 move and split the scripts but keep them as shell. This task finishes the one-language rule for model tooling: logic goes into Rust (`xtask` / `umpire-guest-driver` subcommands), thin wrappers become Makefile recipes, and the only shell left is the minimal xtask bootstrap.

Runs after .13 (framework copies in `model/umpire/scripts/`, Temporal copies in `model/scripts/`), .10 (`cargo xtask gate` composition) and .14 (mise checks already gone, so the ports do not bring them back). Start from whatever script set those tasks leave. The current executable scripts are fetch-tlc, guest, guest-freshness, proto-go, proto-tools, retired-names, time-gate and xtask. The current control/build scripts across the two roots are test-capabilities, test-time-gate, test-model-validate, test-model-wiring, tests/bootstrap, tests/proto-tools, tests/proto-go, tests/build-guests, tests/build-guests-controls and tests/guest-roots. Refresh this inventory after .17/.16 and map each actual script and its behavioral controls to its replacement or an explicit obsolete-case reason.

**Size:** M
**Files:** `model/umpire/scripts/**`, `model/scripts/**`, `model/umpire/dev/xtask/**`, `model/umpire/dev/guest-driver/**`, `model/dev/xtask/**` (Temporal extension), root `Makefile`, the framework entry point from .10, and READMEs or `GUEST_BUILDS.md` that name the scripts.
**Touches:** [model/umpire/scripts/**, model/scripts/**, model/umpire/dev/xtask/**, model/umpire/dev/guest-driver/**, model/dev/xtask/**, Makefile, model/umpire/Makefile, model/README.md, model/GUEST_BUILDS.md, model/umpire/**/README.md]

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
## Acceptance
- [ ] `find model -name '*.sh' -not -path '*/target/*' -not -path '*/archived/*' -not -path '*/design/*'` lists only the xtask bootstrap script(s); the archived and design probes are out of scope.
- [ ] Every removed script's job is reached through `cargo xtask <cmd>`, a guest-driver subcommand or a Makefile recipe. Root `Makefile`, the framework gate and the docs call the new forms, and an `rg` check finds no references to deleted scripts in active source callers, maintained tests or maintained user/developer documentation. Preserve historical specs, completion records, review artifacts and archived/design evidence that name the former scripts.
- [ ] A complete current script-to-replacement mapping accounts for the behavioral controls the shell tests covered. Their Rust replacements pass; any control dropped as moot is listed with its reason.
- [ ] The retired-name check runs in Rust and still catches each retired name with a planted file.
- [ ] The framework standalone control (.11) and the focused xtask, guest-driver and TLC tests pass. The full gate runs once at the end, as usual for fn-254.


## Throughput graph consumer

Run after .17 and .16. Port leaf implementations behind .17's established native stage interfaces; keep one authoritative scheduler, stage identities, cleanup/resource contracts and framework-only task resolution. Do not rebuild a separate shell or Rust task graph during the port.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
