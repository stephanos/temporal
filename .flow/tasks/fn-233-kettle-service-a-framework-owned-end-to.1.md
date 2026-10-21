---
satisfies: [R1]
---
# fn-233-kettle-service-a-framework-owned-end-to.1 Extract generic catalog tooling and migrate realization envelopes

## Description
Move the three envelopes (Observation, Callback, Reply) from `model/temporal/proto` to `model/umpire/proto` under a generic package, and make the descriptor coverage build check (`temporal/api/build.rs`) framework-owned under `umpire/`, parameterised by descriptor path. No validator generator or Go source walk moves: the catalog names no server internals (owner, 2026-10-21). The Temporal slice consumes the moved pieces and keeps byte-identical Model/semantic goldens with the specified Any identity migration.

**Size:** M
**Touches:** `model/umpire/proto/**`, `model/temporal/proto/**`, `model/temporal/api/build.rs`, `model/umpire/hosts/go/**`, `model/temporal/realization/README.md` (the "exactly three envelopes" sentence), `model/README.md`

### Approach
Move first, generalise second: each piece lands under `umpire/` with the same behaviour and a parameter where a Temporal path or package name was hard-coded. Run the full Temporal gate after each move. Anything that cannot move without a Temporal name is listed in DESIGN.md as a finding, not worked around.

### Required shared extraction and migration

Follow the parent implementation contracts for generic tooling and envelope migration. In addition to the existing Touches, own `model/temporal/annotate/{syntax,macros,src}/**`, `model/temporal/api/{services,wire_schema,codegen}.rs`, `model/temporal/realization/src/**`, both Go modules' generated code/consumers, `model/scripts/proto-*.sh`, generic annotation/protocol crates, Cargo manifests and Makefile. Parameterize context type, service/module paths and request roots; move generic Annotate/probe transport with its implementation. Preserve fn-227.12's request tag 4. This task follows fn-227.12 and migrates its consumers. Inspect fn-228's rename state before editing shared paths; overlapping work is serial.

Quick checks: descriptor/syntax/macro fixtures for both parameterized consumers, protocol generation freshness and Go wire tests. Final task boundary: make model-check and make umpire-check-model-slice; compare Model/semantic goldens separately from expected Any identity changes. Missing generic pieces must be implemented here; a DESIGN finding alone does not satisfy R1/R3.

## Acceptance
- Envelopes live under `model/umpire/proto` with a generic package; `temporal/proto` keeps only Temporal-specific messages (R1).
- Descriptor coverage check runs from `umpire/` with the Temporal slice as their first caller; no Temporal name under `umpire/` (gate check) (R1).
- All Temporal Model/semantic goldens byte-identical; only specified envelope Any type-URL expectations migrate; `make model-check` green.
- DESIGN.md records any piece that could not move and why.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

