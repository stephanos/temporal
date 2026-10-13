---
satisfies: [R7, R6]
---
# fn-166-gomad-readable-package-structure-and.7 Group shared packages under evidence and model

## Description
Move runner, backends, record/artifact/captured/qualification (as evidence), choice/world/simulation clock and network/volume/iowire (as model), target (into backends), hostfs+hostexec (as host) and puritycheck (into archcheck) under internal/; delete the external Runner consumer test, its fixture module and the public-signature and type-alias rules. Update archcheck owner, edge and pure-set tables, generators, Make targets and documentation. Land in the same integrated candidate as the R5 renames, fn-153 and fn-154.

Carried from .4 and .9 (2026-10-13): move inspection to `runner/inspect` (needs plan reading moved into `runner/internal/campaign` first so `Inspect` uses no Runner-private projections and Runner tests keep their fakes without an import cycle; 36 Runner identifiers named from `backends/wasm` and the 7 `SimulationClock` methods in identity-bound `runner/simulation_broker.go` must move with it to halve Runner's exports), and remove the remaining `tools/gomad3integration` module (`TestPublicWrappers`, `testdata/tagged`) together with the library-contract checks; update the `soak.json` description path when its manifest digest is rotated anyway.

### Final six-group binding

Keep target as internal/backends/target to avoid Provider → artifact → target import cycles. Place shared strictjson and runtimesource under internal/model, and semantic generator definitions under internal/model/schema; no extra immediate internal owner remains. This realizes the spec's 2026-10-13 final rotation integration bindings. Complete inspection extraction and export reduction, not merely import/path replacement.

### Inspection source-lane admission (2026-10-13)

The approved inspection extraction may proceed independently while the helper and fn154 conformance owners finish. This lane changes no helper, broker, network, generated source, Make target, or architecture inventory. It extracts inspection into runner/inspect, moves plan reading/admission into runner/internal/campaign, and places necessary shared operational projections below both users. Update exact Runner/CLI inspection consumers and tests without compatibility forwarding aliases. Preserve all behavior, refusals, output and tests. Record pre/post exported inventories; full reduction also depends on the later clock/backend extraction. Other package moves and task acceptance remain with the final rotation batch. Root integrates in serialized order and retains Flow completion, review and final qualification.

**Inspection lane Touches:** [tools/gomad3/runner/inspect*.go, tools/gomad3/runner/inspect/**, tools/gomad3/runner/portable_plan.go, tools/gomad3/runner/portable_plan_test.go, tools/gomad3/runner/portable_plan_mounts.go, tools/gomad3/runner/recovery.go, tools/gomad3/runner/runner.go, tools/gomad3/runner/campaign_merge.go, tools/gomad3/runner/campaign_shard_execution.go, tools/gomad3/runner/exploration_resume_test.go, tools/gomad3/runner/seed_resume_test.go, tools/gomad3/runner/choice_exploration_campaign_test.go, tools/gomad3/runner/guidance_test.go, tools/gomad3/runner/preparation_dependencies_test.go, tools/gomad3/runner/backend_test.go, tools/gomad3/runner/internal/campaign/**, tools/gomad3/runner/internal/inspection/**, tools/gomad3/cmd/gomad/internal/cli/inspect*.go, tools/gomad3/cmd/gomad/internal/cli/cli.go, tools/gomad3/cmd/gomad/internal/cli/cli_test.go]

Stop and request conductor scope binding before modifying a surviving consumer outside this declaration. In particular do not edit runner/internal/exploration/integration_test.go, currently owned by fn154 conformance.

The listed recovery/Runner/campaign/portable-mount and six exact test consumers were admitted after the worker's dependency inventory. Preserve their operational behavior while relocating shared inspection projections and plan admission; no unbounded cross-owner consumer permission is implied.

**Mechanical inspection consumer Touches:** [tools/gomad3/backends/wasm/backend/integration_test.go, tools/gomad3/backends/wasm/command/kill_resume_linux_test.go, tools/gomad3/cmd/gomad/integration_test.go, tools/gomad3/qualification/qualification.go, tools/gomad3/qualification/qualification_test.go, tools/gomad3/backends/native/qualification/soak/soak.go, tools/gomad3/backends/native/qualification/soak/soak_test.go, tools/gomad3/backends/native/conformance/testdata/runner_external/consumer.go, tools/gomad3/runner/characterization_test.go, tools/gomad3/runner/coverage_test.go, tools/gomad3/runner/diagnostics_test.go, tools/gomad3/runner/minimize_operation_test.go, tools/gomad3/runner/replay_operation_test.go]

These additional exact consumers may receive import/type/call migration only, without changing their semantics, expectations or qualification requirements. Preserve the external fixture until its separately authorized removal. No changes to helper transport or network conformance are admitted. Shared detached inspection projections belong beneath both inspection and operational Runner; preserve operational result types without public forwarding aliases.

**Move-bound lint policy Touches:** [.github/.golangci.yml, cmd/tools/lintcode/lint_policy_test.go]

The preserved impossible validated-choice-kind panic may receive one new exclusion at its relocated exact runner/inspect/inspect.go path, matching only the existing source line and forbidigo message. Bind the path/source and lookalike/mutated-message negatives in lint policy tests; no generic panic exception is approved. Existing rule inventory changes only for this one relocation.

### Public-library retirement source lane (2026-10-13)

Revival: complete the approved R7/R9 tool-only end state alongside fn153 audit corrections. This lane retires public signature/type-alias/forwarding-alias and external-consumer contract checks, preserving actual ownership, import, effect/purity, CI and selection guards. Remove obsolete gomad3integration wrappers but move tagged qualification fixtures to the native qualification owner before removal. No replacement public facades. Full six-group relocation and task acceptance remain in the rotation batch.

**Library-retirement Touches:** [tools/gomad3/internal/archcheck/architecture_test.go, tools/gomad3/internal/puritycheck/signatures.go, tools/gomad3/internal/puritycheck/check_test.go, tools/gomad3/runner/integration_test.go, tools/gomad3/runner/testdata/runnerconsumer/**, tools/gomad3/backends/native/conformance/testdata/runner_external/**, tools/gomad3/backends/native/qualification/**, tools/gomad3integration/**, Makefile, tools/gomad3/Makefile]

Root owns Flow, MILESTONES, review and integration. fn153.21 owns archcheck edges.go/inventory.go; this lane must not edit them. Inventory native external fixture users before removal; preserve unrelated fixture assertions. Ask for exact additional consumer paths before edits. Intermediate architecture public-signature red is expected until approved retirement; no ownership/purity rule weakening is authorized.

**Tagged-relocation consumer Touches:** [cmd/tools/lintcode/main.go, cmd/tools/lintcode/main_test.go, .github/workflows/gomad3-smoke.yml]
Limit changes to removing obsolete integration-module routing and preserving relocated tagged qualification selection; default lint behavior and actual qualification gates stay unchanged. Root coordinates the exact native qualification testdata fixture prefix with fn153.21's inventory.go owner.

**Surviving-check consumer Touches:** [tools/gomad3/backends/wasm/architecture_test.go, .github/workflows/gomad3.yml]
Remove only the WASM public-signature branch and obsolete gomad3integration path triggers; retain actual WASM purity assertions and qualification selections. The replacement native qualification paths are already covered by tools/gomad3 triggers.
## Acceptance
The module top level holds only cmd, internal and testdata, and internal/ holds runner, backends, evidence, model, host and archcheck; Git renames preserve history; full WASM gate set passes on the integrated candidate.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
