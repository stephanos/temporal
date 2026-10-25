# Gomad rules

## Runtime

- Run guest code inside WASM.
- The same candidate and inputs produce the same guest behavior.
- Make memory, fuel, time, and output bounds explicit.
- Preserve typed failure kinds across boundaries.

## Rust 

- Write all Gomad-owned code, tooling, and scripts in Rust.
- Return typed errors for invalid input.
- Keep crate APIs small and dependencies one-way.
- Reuse existing crates before adding new ones.

## Tooling

- Assume mise is installed. Call it directly; if it is missing, fail.
- Don't probe for mise or the tools it pins, and don't add fallbacks for running without it.

## Data and evidence

- Verify provenance and contents before trusting artifacts.
- Use structs/enums and shared validation code.
- Use pinned handles at filesystem trust boundaries.
- Treat missing, empty, and invalid data as distinct states.

## Testing

- Prove the intended failure and restore valid behavior.
- Isolate process-wide state between tests.

## Delivery

- During each task, validate changed gates and affected behavior.
- Run broad validation on the integrated spec candidate before closing the spec.
- Repeat passed checks only when later changes affect what they prove.
- Keep preparation independently testable.
- Get one end-to-end path working early.
- Exercise small, real boundaries before expensive runs.
- Reuse evidence when relevant inputs are unchanged.
- Claim only what the required acceptance checks actually prove.
