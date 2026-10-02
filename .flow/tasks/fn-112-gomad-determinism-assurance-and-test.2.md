---
satisfies: [R2]
---
# fn-112-gomad-determinism-assurance-and-test.2 Run the orphaned simulation, overlay, and choice-replay tests in a gate

## Description
Make tests that exist but no target runs execute in `make -C tools/gomad3 test` and CI (R2). Depends on task 1 so newly failing tests are attributable.

**Size:** M
**Files:** `tools/gomad3/Makefile`, `tools/gomad3/architecture_test.go`, `tools/gomad3/internal/gomadtool/conformance/runtime_campaign.go` and the `runtime_*.go` file that owns replay behavior, `.github/workflows/gomad3.yml`
**Touches:** [tools/gomad3/Makefile, tools/gomad3/architecture_test.go, tools/gomad3/internal/gomadtool/conformance/runtime_campaign.go, tools/gomad3/internal/gomadtool/conformance/runtime_choice.go, .github/workflows/gomad3.yml, tools/gomad3sim/**]

### Approach
- Add a Make target that runs `tools/gomad3sim` with the `gomad3_toolchain` tag under the patched toolchain, and add it to `test`. Decide whether the `integration`-tagged `simulation_root_integration_test.go` joins it or is superseded by it.
- Extend `overlay-test` with `internal/gomadsim`, `internal/gomadmodelwire`, `internal/gomadio`, `os`, and `cmd/internal/gomadcap`.
- Execute the `choice-replay` fixture that `runtime_campaign.go` builds, in the conformance file that owns replay behavior.
- Run `./toolchain` in `test-toolchain` (patched) and `test-builder` (stock) only; drop it from `test-host`.
- `TestMakeTargetsMatchTheirOwnership` constrains new Make targets; update its table.
- A test that fails once it runs is fixed or recorded as a finding; deletion needs a stated reason.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/Makefile:127-133` — `test-host` and `overlay-test` package lists
- `tools/gomad3/internal/gomadtool/conformance/runtime_campaign.go:299` — `choice-replay` build with no executor
- `tools/gomad3/architecture_test.go:138` — Make target ownership check
- `tools/gomad3/runner/internal/execution/simulation_root_integration_test.go:1-19` — the only current executor of the simulation tests

**Optional** (reference as needed):
- `Makefile:189` — root `gomad3-integration-test`
- `tools/gomad3/architecture_test.go:290` — conformance grouping rule

### Key context
- `tools/gomad3sim` is on the list of features whose fate is undecided (spec Open Questions 4). Running its tests does not settle that.
## Acceptance
- [ ] The six `tools/gomad3sim/*_toolchain_test.go` files run in a Make target included in `test` and in CI
- [ ] `overlay-test` covers the five previously omitted overlay packages
- [ ] The `choice-replay` fixture is executed and asserted
- [ ] `./toolchain` runs once per toolchain kind
- [ ] Every newly running test passes, or is recorded as a finding with its failure
- [ ] The new gate's name and scope are in the done summary for task 10 to document
- [ ] `make -C tools/gomad3 test` passes on darwin/arm64; linux status recorded
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
