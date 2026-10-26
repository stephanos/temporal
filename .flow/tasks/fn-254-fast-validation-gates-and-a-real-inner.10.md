---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.10 Move the framework gate into model/umpire; root Makefile keeps Temporal steps only


## Description
Compose the framework gate and Temporal delegation on .13's independently buildable workspace/tool roots (spec R7). Workspace separation, pins/cache-root migration and guest identity primitives belong to .13; this task owns stage orchestration, mixed controls and generated producer/consumer wiring.

**Cross-spec gate:** coordinate with in-flight fn-246 lanes; workspace and lockfile moves touch every crate's build. Land at an integration boundary when no lane has uncommitted Cargo changes.

**Size:** M
**Files:** framework entry point and moved tooling controls, core/Kettle protobuf and guest orchestration, root Makefile's Temporal stages, host parity controls and current generated assets.
**Touches:** [.github/workflows/umpire.yml, model/umpire/scripts/**, model/umpire/dev/**, model/umpire/protocol/**, model/umpire/hosts/go/**, model/umpire/reference/kettle/**, model/dev/xtask/**, model/scripts/**, model/temporal/hosts/go/**, model/temporal/guest/**, Makefile, model/README.md, model/umpire/README.md, model/GUEST_BUILDS.md]

### Immediate framework-only checkpoint and working seams
- Start after committed .13/.9, using .13's existing `fetch-tlc.sh` and `fetch-apalache [framework-root]` public provisioning commands. .14's probe removal and .16's provider implementation replacement preserve those public command/status contracts; neither is a producer prerequisite for gate composition. Renew affected caller checks when they later change inputs. Prioritize this independent framework entry point so framework iteration can omit Temporal checks now.
- Keep real framework gate orchestration in `model/umpire/dev/xtask/src/gate.rs` and its gate-local submodules, exported and called by the real dispatcher. Keep both workspaces' existing real test execution in their respective `src/test_run.rs` leaves, already called by their actual gates. Initially preserve the existing Cargo tests/doctests exactly once. Task .3 changes only those working leaves/config/pins; no dormant quick/standalone module is introduced here.
- Full-gate entry already must override inherited quick-cache enablement. Establish its authoritative disable policy before any stage child is launched, exercise real propagation controls and record the exact policy for .6. .6 adds the cache then proves the populated-cache bypass without rewriting gate orchestration.
- These working boundaries permit honest disjoint Touches: .3 runner leaves/configs/pins; .6 quick command/CLI/Make/cache; .11 gate-local standalone stage and integration controls. .5/.12/.15 remain serialized whenever their actual ownership overlaps. Record realized paths before lane admission; do not claim nonexistent files or an unregistered command are delivered.

### Approach
- Consume .13's dedicated guest driver and admitted context for root resolution, tool/pin/cache ownership and guest identity; do not repeat that migration. Keep gate orchestration in the general xtask package, with no reverse dependency from the guest driver. Guest commands cross the dedicated executable interface rather than invoking guest orchestration inside the general CLI process.
- Add a discriminating native-build control: change only retained general-CLI/gate executable code, verify that CLI bytes change while driver bytes and guest cache namespace remain unchanged for identical driver/lock/compiler/guest inputs, and show a representative existing guest build still reuses admitted artifacts. A budget data edit that leaves executable bytes unchanged is insufficient. Preserve driver/lock/compiler invalidation controls and current-input freshness; never remove identity inputs merely to make a cache hit. Reuse .13 evidence only for unchanged inputs.
- Framework protobuf generation and checks use framework-owned pinned generators, not the root `.bin/`.
- Root `model-check`/`model-validate` delegate framework work and retain only Temporal guest freshness, source boundary, Temporal protobuf/API prerequisite, Temporal Go host and Temporal crate checks. Expose stage dependency information so .5 may overlap independent work; do not mandate all framework work precede all Temporal work. Full gates override inherited quick-cache enablement.
- Split mixed guest/protobuf/retired-name controls: stock/control/Kettle/panic/overflow and core/Kettle stay framework-owned; Temporal guest/protobuf/API and deny-vocabulary policy stay Temporal-owned. Preserve all old logical tests exactly once. Regenerate affected current guest bytes through .13's identity primitives and prove source freshness/native-Wasm-Host parity, including possible Temporal embedded guest publication.
- Stage names, fail-fast behaviour and `UMPIRE_TIME_STEPS` timing are preserved across both halves.
- **Equivalence pin:** the existing combined stage/test coverage (with ignored tests) is preserved exactly once under the checked migration mapping; newly required controls have a separate explicit inventory; the TLC-required run passes.

### Investigation targets
**Required:**
- `model/design/framework-gate.md` — layout and decisions
- `model/umpire/backends/process/src/support.rs`, `model/umpire/backends/apalache/src/lib.rs:120-190` — cache roots
- `model/scripts/proto-go.sh` — core/kettle/temporal modes
- `Makefile:1270-1315`

#
- Owner request (2026-10-25): add the framework gate and the root `model-check` to `.github/workflows/umpire.yml`. fn-256.1 removes the dead `tools/umpire` reader tests that the CI job runs now. While rewriting the `model-check` recipe, factor out the repeated `cd model && target=… case …` prelude (now Makefile lines 1296–1297).
## Owner validation cadence (2026-10-25)

Keep changed framework/root entry-point wiring, both-workspace coverage accounting, modes, cache bypass and named failure controls now; unrelated unfiltered combined gates wait for .8. Deferred checks remain final .8 requirements and are never reported green. This owner instruction supersedes earlier broad per-task verification commands.


### Mise provider prerequisite

Task .13 already supplies the stable TLC and optional Apalache public provisioning entry points. Compose against those existing contracts and preserve routine Apalache exclusion; do not reimplement downloads in the gate. Task .16 replaces their provider internals after .14 and preserves public behavior. It need not block this framework-only entry point; changed provider paths/admission inputs require renewed focused caller checks later.

## Acceptance
- [ ] A real compiled CLI/gate-only change preserves guest-driver bytes and guest cache/artifact reuse; actual driver/lock/compiler changes still invalidate. Gate/installer/guest callers consume .13's shared context without duplicated root or remap policy.
- [ ] The framework entry point runs every framework stage from `model/umpire`; the root `Makefile` contains no framework stage.
- [ ] .13's standalone path/tool contract remains intact; every split mixed control and generated producer/consumer has one owner. All affected guests reproduce, are fresh and pass Host/native-Wasm parity.
- [ ] Stage/test coverage is preserved in sum, and changed entry-point/mode/cache/failure controls pass. Unrelated unfiltered `model-check`/TLC-required `model-validate` execution is deferred to .8 under the owner cadence amendment.

## Done summary
All four formal task10 findings are corrected at `0e7981836def935d3554feb62b8f03581320722d`.

Make/bootstrap and both gate halves preserve caller-relative Cargo paths; the timing benchmark does too. The Rust boundary lexer recognizes character and byte-character literals, and the organization exception now uses the exact original occurrence/token rule. `proto-tools` is a real listed registry node executed once. Gate tests and snapshot updates consume the live native `test-run` route; task3 can sequence nextest and doctests solely in the existing runner leaf.

All13 focused groups passed: 31 framework native tests,10 outer native tests including real shipping Make/bootstrap with relative Cargo paths, shell controls, both clippies and formats. Real compiled boundary controls pass the current tree,17 negative/2 positive policy cases and5 escaping-include cases. The actual retired-name stage passed. A renewed retained CLI-code mutation changed CLI bytes while exact driver bytes, guest namespace and real stock Wasm artifacts were reused unchanged.

Accounting preserves930 baseline Rust identities (4 ignored),563 snapshot/UI hashes,119 registry tuples,23 old stages through27 ownership partitions and152 shell control leaves. The current24 added Rust identities include19 prior controls plus5 new revision regressions. This is focused execution plus exact source accounting, not a new whole-workspace campaign.

The historical791 index/proofs remain byte-identical. Failed RED/fixture/cleanup attempts are explicit in the index. Full combined/default/TLC-required gates remain task8 obligations; the real directory-only complete proof remains task11. No driver, guest, lock, generated host, Model or dependency changes occurred.

Three fix worktrees and their solely owned build caches were removed; their refs and parent proof cache remain. Coordinator owns same-session re-review, Flow closure and root integration.

Handoff branch `work/fn25410-gate`, tip `a6e7aededaf7a5d1058b920b5f51f714c05f1355`. Current pointer `.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task10/current-index.json`. No Flowdone/root integration performed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a6e7aededaf7a5d1058b920b5f51f714c05f1355, 0e7981836def935d3554feb62b8f03581320722d, ff9b9930b6, 5ce89a3350, e31af6b6f2
- Tests: framework-proto-controls: exit 0, outer-proto-controls: exit 0, framework-timer-controls: exit 0, framework-proto-tools-controls: exit 0, framework-bootstrap-controls: exit 0, framework-build-guest-controls: exit 0, root-timer-controls: exit 0, framework-native-tests: exit 0, outer-native-tests: exit 0, framework-native-clippy: exit 0, outer-native-clippy: exit 0, framework-fmt: exit 0, outer-fmt: exit 0, actual-boundary: exit 0, actual-retired: exit 0, actual-stage-list: exit 0, cli-initial-build: exit 0, cli-initial-guest: exit 0, cli-mutated-build: exit 0, cli-mutated-guest: exit 0, cli-restored-build: exit 0, Conductor hash audit:29 current source/47 current proof/54 historical proof hashes match; same-session Codex6.1high re-review SHIP allfour findingsfixed.
- PRs: