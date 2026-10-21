---
satisfies: [R3]
---
# fn-211-apalache-backend-for-umpire-in-rust.3 ITF counterexamples decoded through typed Schema/Value decoding and replayed through the Model

## Description
**Placement (owner 2026-10-18):** `model/umpire/backends/apalache/`.

Decode Apalache's ITF JSON trace into states and actions via the canonical `typed Schema/Value decoding` form (the same decoder rules as the TLC dump), then replay the path step by step through the Model's `step`; a mismatch is witness-rejected. Write `expected/apalache/paths/<invariant>.txt` for violations.

**Files:** model/umpire/backends/apalache/src/itf.rs, replay helper in the checker or testing crate.

**Quick commands:** `cd model && UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-apalache`
### Required review corrections

Use the parent typed trace contract: ITF to typed values to exact same-World checked_step replay. Display/Debug are output only; no Spell API or formatting lookup is introduced. Validate starts, concrete inputs, choices, outcomes and violated claim; strip only declared history/stutter fields. Arithmetic-error witnesses replay a valid prefix and confirm native refusal rather than fabricating an accepted edge. Tamper each boundary in negative tests.

## Acceptance
- Kettle's `!beeping` negative control yields a replayed violation path golden.
- A tampered trace is witness-rejected.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
