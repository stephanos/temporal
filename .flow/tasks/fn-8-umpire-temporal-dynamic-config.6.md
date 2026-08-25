---
satisfies: [R8, R9]
---
# fn-8-umpire-temporal-dynamic-config.6 Prove callback admission against immutable snapshots

## Description
Add the bounded callback-admission consumer required by R8 and run the final generation/model gates from R9. Keep it pure and isolated from live Temporal environments.

**Size:** M
**Files:** `model/Temporal/Experiment/Config.lean`, `model/Temporal/Experiment/ConfigTests.lean`, `model/Temporal/ExperimentTests.lean`
**Touches:** [model/Temporal/Experiment/Config.lean, model/Temporal/Experiment/ConfigTests.lean, model/Temporal/ExperimentTests.lean]

### Approach
- Project one resolved view into a compact callback domain configuration containing entity-creation CHASM enablement, per-request maximum count, interpreted address rules, and destination-specific timeout. Reuse the existing pure target/setup patterns instead of passing raw settings into transitions.
- Implement the parent spec's binding callback transition table exactly: disabled-at-creation selects legacy HSM routing; count equality admits and only overflow rejects; special/wildcard/insecure address cases follow the authored interpretation; missing destination and malformed canonical rules fail before transitions; elapsed time equal to or above the timeout (and non-positive timeouts) times out.
- Model bounded attachment/admission and dispatch timing only. Keep HSM-versus-CHASM route selection distinct from address/count admission so disabling CHASM never masquerades as callback rejection.
- Create paired snapshots/contexts whose intended differences alter route, admission, or dispatch outcomes, then prove each trace stays pinned to its starting projection even if another snapshot exists.
- Keep sampling/change metadata descriptive: no restart simulation, config-change action, mutable view, environment preset, YAML path, or public CLI.
- Run focused Go generation tests, regenerate once, build the full Lean model, format imports, lint code, and confirm no drift-check/CI surface appeared.

### Investigation targets
**Required** (read before coding):
- `service/history/workflow/mutable_state_impl.go:697-705` — CHASM callback selection at entity creation
- `service/history/workflow/mutable_state_impl.go:3388-3410` — disabled selection falls back to HSM attachment
- `service/history/workflow/mutable_state_impl.go:7197-7208` — existing CHASM callbacks after disablement
- `chasm/lib/workflow/workflow.go:124-142` — maximum-count equality/overflow behavior
- `chasm/lib/callback/config.go:71-180` — special URLs, wildcard, malformed-rule, and insecure-address behavior

**Optional** (reference as needed):
- `components/callbacks/executors.go:112-132` — destination timeout application
- `model/Temporal/Experiment/Semantics.lean:433-620` — existing pure semantic composition patterns

### Key context
Production disables CHASM by selecting legacy HSM attachment, not by rejecting callbacks, and continues processing already-created CHASM callbacks. The authored Lean interpreter intentionally rejects malformed canonical rules even though the raw Go converter silently ignores malformed raw entries; raw conversion is outside the model boundary.

### Quick commands
```bash
go test -count=1 -tags test_dep ./common/dynamicconfig ./cmd/tools/genleandynamicconfig
make umpire-gen-dynamic-config
cd model && mise exec -- lake build
make fmt-imports
make lint-code
```
## Acceptance
- [ ] The callback consumer reads only a typed projection of a validated immutable view and uses exactly the four scoped setting meanings.
- [ ] Disabled-at-creation selects legacy HSM routing, enabled-at-creation selects CHASM, and the captured route remains stable for the trace.
- [ ] Count equality is admitted and only overflow rejects; exact special URLs, wildcard matches, secure defaults, and explicitly permitted insecure HTTP follow the binding transition table.
- [ ] Missing destination and malformed canonical rules fail before execution; elapsed time equal to or greater than timeout and non-positive timeout produce the timeout outcome.
- [ ] Paired snapshots can change route/admission/dispatch outcomes across experiments, while every trace remains pinned and exposes no mid-trace update/restart behavior.
- [ ] No live server, callback execution, YAML parser, environment preset, public config CLI, drift-check target, or CI workflow is added.
- [ ] Focused Go tests, generation, full Lean build, formatting, and linting pass with existing comments preserved.
## Done summary
Implemented the pure callback-admission consumer for R8 over a private projection of exactly four typed values from an immutable `ConfigView`. The model captures legacy-HSM versus CHASM routing, admits count equality and rejects only overflow, applies exact special-URL and whole-host address rules, rejects request/address failures before dispatch, models zero additions as a no-op, and enforces positive/equal/greater/non-positive timeout boundaries while paired snapshots prove route, admission, dispatch, and within-trace immutability.

Focused coverage exercises zero additions below and above the maximum, count equality/overflow, the two exact Temporal URLs plus path/query/fragment variants, wildcard and insecure HTTP exactness, unknown/missing/unmatched/insecure addresses, missing destination, malformed canonical rules, timeout boundaries, paired outcomes, and CHASM-route capture after a disabled snapshot exists. Verification passed `go test -count=1 -tags test_dep ./common/dynamicconfig ./cmd/tools/genleandynamicconfig`, `make umpire-gen-dynamic-config`, `cd model && mise exec -- lake build Temporal.Experiment.ConfigTests`, `cd model && mise exec -- lake build ExperimentTests`, `cd model && mise exec -- lake build`, and `make fmt-imports`; `make lint-code` remains inherited red at the identical pre-edit and terminal count of 1828 pre-existing Go findings, its formatting side effects were restored, and the task has no Go diff. Memory capture was attempted after the review fix but the repository memory store is not initialized.

baseline: red (`make lint-code` failed pre-edit with 1828 inherited pre-existing findings; all other Quick commands green)

stage: impl-review - ran [2026-08-25T18:24:27Z..2026-08-25T18:31:01Z]
## Evidence
- Commits: c248289b106103ba6c1d558aef7825d9e304a80f, c208294d0c61e539cab573a3f9bcae0e8aa1a8d8
- Tests: go test -count=1 -tags test_dep ./common/dynamicconfig ./cmd/tools/genleandynamicconfig, make umpire-gen-dynamic-config, cd model && mise exec -- lake build Temporal.Experiment.ConfigTests, cd model && mise exec -- lake build ExperimentTests, cd model && mise exec -- lake build, make fmt-imports, make lint-code (inherited baseline and terminal red: identical 1828 pre-existing Go findings; no task Go diff)
- PRs: