# Gomad

## Read before work

- Read [README.md](README.md), [SPEC.md](SPEC.md), and [ARCHITECTURE.md](ARCHITECTURE.md) for the current contracts.
- For backend changes, also read the [WASM README](internal/backends/wasm/README.md) and [qualification contract](internal/backends/wasm/QUALIFICATION.md).

## Rules

### Runtime

- Run guest code inside WASM.
- The same candidate and inputs produce the same guest behavior.
- Make memory, fuel, time, and output bounds explicit.
- Preserve typed failure kinds across boundaries.

### Data and evidence

- Verify provenance and contents before trusting artifacts.
- Use structs/enums and shared validation code.
- Use pinned handles at filesystem trust boundaries.
- Treat missing, empty, and invalid data as distinct states.

### Delivery

- Keep preparation independently testable.
- Get one end-to-end path working early.
- Exercise small, real boundaries before expensive runs.
- Reuse evidence when relevant inputs are unchanged.
- Claim only what the required acceptance checks actually prove.
