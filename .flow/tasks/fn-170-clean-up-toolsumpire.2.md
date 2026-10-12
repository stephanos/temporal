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
5. Prove no behaviour change: the Go tests of the moved packages pass; lint, lowering and Case generation outputs are byte-identical on the checked-in IR.

## Acceptance
- `tools/umpire/framework/` and `tools/umpire/temporal/` exist; no Go package remains at the old flat paths except as recorded.
- The boundary test passes, and a deliberate violation fails it.
- `export/quint` holds the Quint backend; agreement receipts identical.
- Outputs byte-identical; importers and docs updated.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
