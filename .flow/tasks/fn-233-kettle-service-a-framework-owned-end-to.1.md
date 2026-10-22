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
Catalog authors can now supply their own context, API root, service identity and request roots through shared catalog/descriptor implementations. Shared realization and annotation/probe transport replaces the former system-owned envelopes; Observation.request retains tag 4 and absent-request bytes.

R1 is implemented. The no-Temporal source gate passes, and all 361 canonical Model/semantic files compare byte-for-byte with base 8715c740adb9819d47b3ed9b927d5972ce766049. Compiler expansion pins change only generated runtime/envelope paths and their formatting; one compiler diagnostic pin follows the moved views.rs path. DESIGN records the retained API policy, probe controls and realization-specific source analysis. The independent Kettle catalog fixture and dependency graph contain no extension crate dependency. The live Kettle service/catalog remains assigned to later tasks.

baseline: green (make model-check and make umpire-check-model-slice, run before edits).

Validation passed: make model-check; make umpire-check-model-slice; make umpire-check-model-gaps (365.641s); make lint-code-fast GOLANGCI_LINT_BASE_REV=8715c740adb9819d47b3ed9b927d5972ce766049. Focused controls cover omitted descriptor fields/enums/methods, rustc action/method exhaustiveness, both catalog consumers, context/API-root selection, ID fill, unchanged response bytes, malformed input, bounded probe recipes, wrong descriptor-kind refusal, wire identities and request tag 4. Regression evidence is under .flow/tmp/fn-233 and the two lane handovers. The first full post-change run failed on prettyplease formatting of the shortened Callback assignment; its generated pin was corrected before the final green runs. A native parser regression also proved and closed silent context/API parameters inside supplemental field intents.

GATE_SKIPPED:unittest:green-receipt aa7df937 - final verification reused prior post-gate pass
GATE_SKIPPED:smoke:green-receipt aa7df937 - final verification reused prior post-gate pass

stage: impl-review - ran [2026-10-22T17:49:18.866350+00:00..2026-10-22T17:53:57.627497+00:00] SHIP; correctness/contracts/integration all SHIP with zero findings. Reviewer gpt-6.1-sol at high uses the same family as the writer under the explicit repository override. Reviewers performed source checks; their read-only sandboxes/toolchain defaults prevented independent test execution. The worker ran the green gates above. Durable receipt: .flow/artifacts/fn-233-kettle-service-a-framework-owned-end-to/task1-review.json.

Tier: session (jev-unavailable(no_key)); repository implementer override gpt-6.1-sol at medium.

Two disjoint implementation helpers handled envelopes/transport and descriptor generators; the worker integrated, verified and committed their changes. The full evidence range also contains the conductor's Flow-only fn-238/fn-239 administration commits 93ef3f5290 and 1554d4c422; task code is carried by ddf3cc32c5, 09245933d6 and aa7df93711.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 93ef3f52903e30b78d9a060fd796d0b6b681a5c6, 1554d4c42293daa139821caec11ea476e1e09454, ddf3cc32c58ee525594bb140f637d95f24edb726, 09245933d6197bd172f96226db34cb3898e4558f, aa7df93711d5df93582312779bac3714a1d5d90a
- Tests: make model-check, make umpire-check-model-slice, make umpire-check-model-gaps, make lint-code-fast GOLANGCI_LINT_BASE_REV=8715c740adb9819d47b3ed9b927d5972ce766049, cd model && mise exec -- cargo test -p umpire-catalog-build -p umpire-temporal-api --tests, cd model && mise exec -- cargo test -p umpire-catalog --test kettle, cd model && mise exec -- cargo test -p umpire-catalog-syntax context_and_api_parameters, GATE_SKIPPED:unittest:green-receipt aa7df937 - final verification reused prior post-gate pass, GATE_SKIPPED:smoke:green-receipt aa7df937 - final verification reused prior post-gate pass
- PRs: