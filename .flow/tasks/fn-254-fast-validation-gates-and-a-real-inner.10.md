---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.10 Move the framework gate into model/umpire; root Makefile keeps Temporal steps only


## Description
Compose the framework gate and Temporal delegation on .13's independently buildable workspace/tool roots (spec R7). Workspace separation, pins/cache-root migration and guest identity primitives belong to .13; this task owns stage orchestration, mixed controls and generated producer/consumer wiring.

**Cross-spec gate:** coordinate with in-flight fn-246 lanes; workspace and lockfile moves touch every crate's build. Land at an integration boundary when no lane has uncommitted Cargo changes.

**Size:** M
**Files:** framework entry point and moved tooling controls, core/Kettle protobuf and guest orchestration, root Makefile's Temporal stages, host parity controls and current generated assets.
**Touches:** [model/umpire/dev/**, model/umpire/protocol/**, model/umpire/hosts/go/**, model/umpire/reference/kettle/**, model/dev/xtask/**, model/scripts/**, model/temporal/hosts/go/**, model/temporal/guest/**, Makefile, model/README.md, model/umpire/README.md, model/GUEST_BUILDS.md]

### Approach
- Consume .13's root resolution, tool/pin/cache migration and guest identity contract; do not repeat that migration.
- Framework protobuf generation and checks use framework-owned pinned generators, not the root `.bin/`.
- Root `model-check`/`model-validate` delegate framework work and retain only Temporal guest freshness, source boundary, Temporal protobuf/API prerequisite, Temporal Go host and Temporal crate checks. Expose stage dependency information so .5 may overlap independent work; do not mandate all framework work precede all Temporal work. Full gates override inherited quick-cache enablement.
- Split mixed guest/protobuf/retired-name controls: stock/control/Kettle/panic/overflow and core/Kettle stay framework-owned; Temporal guest/protobuf/API and deny-vocabulary policy stay Temporal-owned. Preserve all old logical tests exactly once. Regenerate affected current guest bytes through .13's identity primitives and prove source freshness/native-Wasm-Host parity, including possible Temporal embedded guest publication.
- Stage names, fail-fast behaviour and `UMPIRE_TIME_STEPS` timing are preserved across both halves.
- **Equivalence pin:** the combined stage list and the full test list (with ignored tests) are identical before and after; the TLC-required run passes.

### Investigation targets
**Required:**
- `model/design/framework-gate.md` — layout and decisions
- `model/umpire/backends/process/src/support.rs`, `model/umpire/backends/apalache/src/lib.rs:120-190` — cache roots
- `model/scripts/proto-go.sh` — core/kettle/temporal modes
- `Makefile:1270-1315`

## Owner validation cadence (2026-10-25)

Keep changed framework/root entry-point wiring, both-workspace coverage accounting, modes, cache bypass and named failure controls now; unrelated unfiltered combined gates wait for .8. Deferred checks remain final .8 requirements and are never reported green. This owner instruction supersedes earlier broad per-task verification commands.

## Acceptance
- [ ] The framework entry point runs every framework stage from `model/umpire`; the root `Makefile` contains no framework stage.
- [ ] .13's standalone path/tool contract remains intact; every split mixed control and generated producer/consumer has one owner. All affected guests reproduce, are fresh and pass Host/native-Wasm parity.
- [ ] Stage/test coverage is preserved in sum, and changed entry-point/mode/cache/failure controls pass. Unrelated unfiltered `model-check`/TLC-required `model-validate` execution is deferred to .8 under the owner cadence amendment.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
