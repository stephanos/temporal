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
Selected additive install flags --temporal-checkout and --temporal-package. After completion review found an R3 gap, ba57de4e5b made prototype, installation, inspection and qualification explicit complete-policy inputs to wasi.TemporalConfig. The shared production derivation owns every network/environment/entropy/argv/writable/output/transcript field; callers apply no posthoc overrides. Command and fixture derivation compare entire final serialized configurations for equivalent inputs without normalization, with an independent historical full-policy matrix. Fresh command/installation/CLI suites and focused WASI/backend/qualification policy tests exit 0 on the integrated tree. A fresh installation manifest compares byte-identical to the retained 29f49b17bc successful explore/replay input (SHA256 34096860a2f8783e4dfbaa7052dcad7b7b8ec6fe647666fdc57073582eaabff2). Independent source re-review SHIP after ba57de4e5b; original completion review will be resumed. Defaults: 256MiB output, 1GiB transcript, 2^28 calls, 4096 descriptors, 100000 files, 512MiB filesystem, 4096 pending events, 128 listeners, 4096 connections, 128MiB network total, 4MiB/connection, 10s stall and 2^20 transitions; memory 4GiB, fuel 2^50. Evidence: .flow/artifacts/fn220-setup-proof-2026-10-20.md. Full WASM qualification remains separately owned by fn-219.4/fn-222.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fb8044eca5, 7c8ef42504, ff719ea3b4, 8896829710, fc04600872, 93b12381b7, ba57de4e5b, c2386bd479
- Tests: integrated go test -p=1 -tags test_dep ./internal/backends/wasm/command ./internal/backends/wasm/installation ./cmd/gomad/internal/cli (exit 0), integrated policy/config tests in WASI, backend and qualification (exit 0), complete final command/fixture JSON equality without normalization and independent historical policy matrix (exit 0), cmp policy-installation.json integrated-installation.json and SHA256 34096860a2f8783e4dfbaa7052dcad7b7b8ec6fe647666fdc57073582eaabff2 (exit 0), compiler-backed TestBuildSelectionChangesInvalidateCompilationCache (exit 0), network connection-limit mutation 4096 to 4095: command+inspection tests exit 1; restored exit 0, derived install and discover on clean 29f49b17bc (exit 0), explore TestFrontendSystemInfo retained success on clean 29f49b17bc (exit 0), fresh gomad replay retained success: reproduced=true choice-replay=exact (exit 0), make lint-code-fast GOLANGCI_LINT_FIX=false (exit 0), independent source review after ba57de4e5b: SHIP
- PRs: