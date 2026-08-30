---
satisfies: [R1, R5]
---
# fn-51-shorten-ordinary-model-authoring.1 Add the ModelValue.named constructor

## Description
Introduce and directly test the transparent Core constructor (R1, R5).

**Size:** S
**Files:** `model/Umpire/Core.lean`, `model/Umpire/CoreTests.lean`, `model/Umpire/ImportTests.lean`
**Touches:** [model/Umpire/Core.lean, model/Umpire/CoreTests.lean, model/Umpire/ImportTests.lean]

### Approach
- Place the constructor beside `ModelValue` and return the exact existing two-field record.
- Add direct equality, invalid-raw-input transparency, documentation, and public visibility checks.
- Do not move validation or identity lookup into Core.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:102-105` — current Model Value definition
- `model/Umpire/CoreTests.lean` — aggregate Core test root
- `model/Umpire/ImportTests.lean` — public facade contract

**Optional** (reference as needed):
- `model/Umpire/Examples/Switch.lean:57-63` — representative literal use

## Acceptance
- [ ] `ModelValue.named` is documented, publicly reachable as intended, and exactly equal to the corresponding record literal.
- [ ] The constructor performs no validation, lookup, inference, registration, or canonicalization.
- [ ] Direct Core and import tests pass.
- [ ] Existing Core comments remain intact.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
