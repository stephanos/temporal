# Gomad

## Read before work

- Read [README.md](README.md), [SPEC.md](SPEC.md), and [ARCHITECTURE.md](ARCHITECTURE.md) for the current contracts.
- For backend changes, also read the [WASM README](internal/backends/wasm/README.md) and [qualification contract](internal/backends/wasm/QUALIFICATION.md).

## Rules

- WASM: run guest code inside WASM.
- Determinism: the same candidate and inputs produce the same guest behavior.
- Evidence: verify provenance and contents before trusting artifacts.
- Limits: make memory, fuel, time, and output bounds explicit.
- Types and validators: use structs/enums and shared validation code.
- Setup: keep preparation independently testable.
- Delivery: get one end-to-end path working early.
- Tests: exercise small, real boundaries before expensive runs.
- Verification: reuse evidence when relevant inputs are unchanged.
- Completion: claim only what the required acceptance checks actually prove.
