---
satisfies: [R1, R3]
---
# fn-220-gomad-one-command-temporal-test-setup.1 Factor the fixture configuration derivation into shared code and choose the verb

## Description
Extract the Temporal guest configuration capture from internal/backends/wasm/wasi/measure_integration_test.go and internal/backends/wasm/backend/temporal_inspection_integration_test.go into shared production code under the WASM installation/configuration owners, so the fixtures call it. Decide whether the consumer path is a new gomad verb or an extension of install/discover, record the decision and the default file/descriptor/call/network limits taken from the fixtures. Early proof point: derive the configuration for tests/gomadfunctional behind a temporary flag and show gomad discover accepting it; stop if the fixtures cannot be factored without changing configuration bytes.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/installation/**, tools/gomad/internal/backends/wasm/wasi/**, tools/gomad/internal/backends/wasm/backend/**, the two integration fixtures

## Acceptance
- [ ] Both integration fixtures call the shared derivation code
- [ ] Configuration bytes derived by the shared code equal the fixtures' previous bytes for the same inputs
- [ ] Verb or install/discover extension decision and the default limits recorded in the task summary
- [ ] gomad discover accepts the derived installation for tests/gomadfunctional

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
