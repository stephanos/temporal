# fn-170-clean-up-toolsumpire.2 Layout: tools/umpire/framework (Temporal-free) and tools/umpire/temporal (adapter), mirroring model/

## Description
Owner decision, 2026-10-12: `tools/umpire` mirrors `model/`. Generic, Temporal-free code goes under `tools/umpire/framework/`, and the Temporal adapter under `tools/umpire/temporal/`. Runs before fn-170.1, which builds the CLI in these homes. The research pass on `tools/umpire` may refine the package split inside each side.

Target:
```
tools/umpire/framework/   ir, interp, engine, check, explore, lint, conformance, lower and realization cores,
                          export (backend-neutral) and export/quint (Quint is a checker backend, not Temporal),
                          internal/cli, runtimecel
tools/umpire/temporal/    activity/nexus lowering and realization (lower/activity_*.go, lower/realization.go,
                          realization/activity_*.go, realization/carriers.go), Temporal lint binding kinds,
                          conformance evidence/guard specifics, API descriptor registration
```

Do:
1. Move the generic packages under `framework/` and the Temporal-specific code under `temporal/`. Split mixed packages at a seam: the generic side defines an interface or registry that the Temporal side fills, as in the Scala fn-169.2 plug-in seam.
2. Split `export` (package `backends`): the backend-neutral parts (`slice.go`, `checked.go`, `tool.go`, `composed.go`) stay in `framework/export`. The Quint parts (`quint.go`, `quint.sh`, `agreement.go`, `verify.go`, `itf.go`, `dump.go`, `raw.go`) go to `framework/export/quint` with their tests. Update `make umpire-check-backends`, its README and `ir/ownership_test.go`; agreement receipts stay identical.
3. Add a dependency-boundary test: nothing under `tools/umpire/framework/` imports `go.temporal.io/api`, `go.temporal.io/sdk`, `service/...`, `tests/...` or `tools/umpire/temporal/...`.
4. Update every importer (Testpilot, `tools/canary`, `tests/`, Makefile, docs, `.plans/UMPIRE_MODULES.md`).
5. **Research mapping (2026-10-12).** Re-check each item against the tree after fn-170.3/.4.
   - Temporal side, whole files: `realization/activity_external.go`, `activity_reset.go` and `carriers.go` (which hard-codes `temporal.api.workflowservice.v1.*` names); `lower/realization.go`, `lower/activity_external.go` and `lower/activity_reset.go`; and the model-pinning tests, about 7.9k lines. Those tests are `check/{activity_*,nexus_*,taskqueue,retries,deadline,dispatch_progress}_test.go`, `lower/{activity_*,heartbeat_*}_test.go`, `conformance/{activity,nexus}_test.go` and `realization/activity_*_test.go`.
   - Seams in mixed code:
     - `realization/validate_realization.go` (~lines 655–760): replace the switch over `Script_Workflow`/`Script_Activity`/`Script_NexusHandler` and `Command_WorkflowCommand`/`AwaitActivityPublication` with a registry of script-activation and settlement kinds that the Temporal side registers.
     - `lower/lower.go`: separate generic standing and ordering from the accounting that reads Temporal evidence kinds. Check that `lower/waits.go` is really generic, as its header says.
     - `conformance/evidence.go` (activity attempt/run ids), `guard.go` (`GetSdkAttempt`) and `outcome.go` (the opt-in rejection-code check).
     - `lint` kinds `explicit-wait`, `unmodeled-api-value`, `silent-rejection` and `unreachable-binding`, plus `lint/bindings.go` and `kinds.go`, which read `GetActivity().GetStarts()`.
     - Literals: the `"temporal.case"` identity root (`lower/generated.go`, `explore/explore.go`) becomes the adapter's.
   - `export`: the research places `agreement.go`, `itf.go`, `tool.go` and the receipts generic, and `quint.go`, `composed.go`, `verify.go`, `checked.go` and `quint.sh` Quint-specific. That differs from item 2 above; decide file by file from the code.
   - The IR schema itself mixes the two: `script.proto` holds Workflow/Activity/Nexus activations and commands, and `realization.proto` holds the activity settlements. Until a held schema split (see fn-169.3) gives them generic activation/settlement shapes with Temporal arms, the Go framework handles those oneof arms only through the registered adapter.
6. Prove no behaviour change: the Go tests of the moved packages pass; lint, lowering and Case generation outputs are byte-identical on the checked-in IR.
## Acceptance
- `tools/umpire/framework/` and `tools/umpire/temporal/` exist; no Go package remains at the old flat paths except as recorded.
- The boundary test passes, and a deliberate violation fails it.
- `export/quint` holds the Quint backend; agreement receipts identical.
- Outputs byte-identical; importers and docs updated.


## Done summary
Superseded by fn-202/fn-208 (one-language series, 2026-10-18): Go runtime split generic/temporal; tools/umpire renamed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: