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
Selected additive install flags --temporal-checkout and --temporal-package; factored schema capture, guest directories, environment and capacities into wasi.TemporalConfig plus the bounded TemporalNetwork policy. Both named integration fixtures call the shared owner and explicitly retain their historical profile overrides. Independent full-byte tests cover canonical prototype and final inspection configurations; a network-limit mutation fails both command and inspection tests. Derived install and discover exit 0 on clean integrated 29f49b17bc. Defaults: 256MiB output, 1GiB transcript, 2^28 calls, 4096 descriptors, 100000 files, 512MiB filesystem, 4096 pending events, 128 listeners, 4096 connections, 128MiB network total, 4MiB/connection, 10s stall and 2^20 transitions; memory 4GiB, fuel 2^50. Independent review SHIP after 93b12381b7. Full evidence/commands/identities: .flow/artifacts/fn220-setup-proof-2026-10-20.md. This consumer proof is separate from full WASM qualification owned by fn-219.4/fn-222.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs:
