---
satisfies: [R1]
---
# fn-211-apalache-backend-for-umpire-in-rust.1 Snowcat type annotations in the TLA+ emitter

## Description
Extend the TLA+ emitter in `umpire-lift` to emit `\* @type: ...;` annotations for every variable, constant and operator, derived from the Model's Rust types (integers, booleans, enums as strings or variants, records, later imbl maps and sets as functions and sets). Re-run TLC on every fixture with a TLC golden and require identical results; update the `.tla` goldens.

**Files:** model/umpire/lift/src/ (emitter, type mapping), TLA goldens under the fixture folders.

**Quick commands:** `make model-check`; `UMPIRE_BACKENDS=require make model-check`

## Acceptance
- Every emitted module is fully annotated; TLC results unchanged.
- A type with no mapping refuses emission naming it (golden).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
