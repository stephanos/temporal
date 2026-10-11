---
satisfies: [R2]
---
# fn-163-system-capabilities-distributed-systems.1 Capabilities split into capabilities/system and capabilities/product

## Description
Move Closable, Pausable, Pollable, Terminable and Describable to `temporal.capabilities.product`, Retryable and Expirable to `temporal.capabilities.system`; update imports, lint and docs. IR: capability origin names change (rename map).

## Acceptance
- Two packages, each capability in the level it describes
- Models, tests and the Describable lint exception follow
- Rename map lists the moved origins

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
