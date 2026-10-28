---
satisfies: [R8]
---
# fn-254-fast-validation-gates-and-a-real-inner.12 Per-stage budgets: committed table, hard work counts, wall time with tolerance


## Description
Make the fast-gate goal enforceable for every current and future stage (spec R8): a committed budget table, hard work-count checks, and wall-time budgets with a tolerance. fn-255's stages (and anything added later) must declare entries here.

**Size:** M
Run after .15's native timer/control port, which also incorporates .16's provider changes. Enforce budgets against those final stage interfaces.

**Files:** `model/umpire/dev/gate/budgets.toml` and `model/dev/gate/budgets.toml` (the two locations selected by .9), the framework/Temporal native gate timing and receipts from .17/.15, and their Rust controls.
**Touches:** [model/umpire/dev/gate/budgets.toml, model/dev/gate/budgets.toml, model/umpire/dev/xtask/**, model/dev/xtask/**, model/umpire/backends/process/src/work.rs, model/umpire/backends/process/src/support.rs, model/umpire/backends/process/src/lib.rs, model/umpire/backends/tlc/src/runner.rs, model/umpire/backends/tlc/src/runner/**, model/umpire/README.md, model/README.md, model/umpire/dev/gate/BUDGETS.md, model/dev/gate/BUDGETS.md, .plans/UMPIRE_MODULES.md, Makefile]

### Approach
- Use .1's historical timing references as initial estimates, then calibrate against the optimized stages and actual counts produced by .2/.3/.4/.5/.6/.11. Historical host/workload differences remain explicit; no fresh pre-optimization timing campaign is required. Count budgets use actual discovered/executed inventories and launch counts, not historical estimates.
- Enforcement: count overruns and unknown/omitted stage registrations fail immediately. Use a 25% warm-time tolerance by default. After an overrun, drain competing owned work and retry that stage once immediately with identical source/tool/budget/tier inputs. A second overrun fails; second-fast passes with the first overrun reported. Interrupted/incomparable retries are incomplete evidence. Include both attempts in tier totals; use invocation-local records so concurrent gates cannot reset one another's state.
- Counts distinguish discovered/executed/ignored/skipped tests and actual TLC launches/configurations. Consume .2/.3/.4/.5/.6/.11's final framework stage/count inventory. Later .7 and fn-255 stage changes update these entries as part of their implementation; .12 does not wait for deferred Temporal consolidation.
- Output: one line per stage with actual vs budget, plus tier totals against fn-254 R6's targets.
- Changing a budget is a reviewed table edit with a reason; record that rule in the README.
- Controls cover over-budget counts, unbudgeted/omitted stages, second-slow failure, first-slow/second-fast success, interrupted retry and independent concurrent invocation records. Each failure names the stage, and no backend refusal is converted to a timing-only warning.

### Investigation targets
**Required:**
- `model/umpire/dev/xtask/src/gate.rs`, `model/umpire/dev/xtask/src/gate/**`, `model/dev/xtask/src/gate.rs` and their Rust timing/receipt controls after .15
- `model/umpire/dev/gate/budgets.toml` and `model/dev/gate/budgets.toml`
- `model/design/gate-speed.md` (.1 baseline)
- `model/design/framework-gate.md` (.9 layout)

### Owner validation cadence (2026-10-25)

Keep stage registration/count/timing/retry controls now. Derive actual count bounds and preserve deferred full-gate/final timing obligations for .8. Deferred checks remain final .8 requirements and are never reported green. This owner instruction supersedes earlier broad per-task verification commands.

### Producer ownership decision (2026-10-28)

Keep the existing process supervisor and stage DAG. Add observation instrumentation at the existing process/TLC invocation boundary, and closed typed protocol-fixture provenance using the existing clean spawning path. Legacy unconfirmed starts remain unknown. Preserve all negative controls and Model semantics. The canonical six nullable WorkCounts fields remain owned by the gate collector; explicit applicability evidence distinguishes completed native dispatch branches from missing observers. Only actual producers and verified fixture provenance may support admitted bounds.
## Owner validation cadence (2026-10-25)

Keep stage registration/count/timing/retry controls now. Derive actual count bounds and preserve deferred full-gate/final timing obligations for .8. Deferred checks remain final .8 requirements and are never reported green. This owner instruction supersedes earlier broad per-task verification commands.

## Acceptance
- [ ] A committed budget table covers every current stage, framework and Temporal.
- [ ] Work-count overruns and unbudgeted stages fail; wall-time overruns fail only on repeat; all listed registration/count/timing/retry controls pass.
- [ ] README documents how to add a stage and how to change a budget.

## Final producer graph

Consume .17's authoritative stage DAG, uniform runner identities and invocation-local timing/count receipts after .5/.6/.11 and the .16/.15 provider and native timer/control ports. Budget internal/child costs without double-charging inclusive parent elapsed time. Provider/port changes must preserve or explicitly update these registrations; no second scheduler or budget mechanism belongs here.

### Remaining count modalities before admission

The .17 producer checkpoint supplies structured stage receipts for actual Rust
nextest/doctest counts and confirmed TLC launches/configurations. Go-host test
counts and other unadapted modalities remain explicitly unobserved (`null`).
Complete their actual producers where a stage has work counts before admitting
its budget table. A missing observation is neither zero nor proof that the metric
is inapplicable. The real mixed-control stage also records unconfirmed TLC
starts from negative controls; its launch/configuration totals remain unknown.
Resolve that measurement gap before admitting a launch/configuration budget,
preserving those controls. Preserve current caller/output contracts, invocation ownership,
child authority and the single stage DAG while adding these adapters.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
