---
satisfies: [R2]
---
# fn-163-system-capabilities-distributed-systems.1 Capabilities split into capabilities/system and capabilities/product

## Description
**Note 2026-10-18 (one-language series):** Sweep proofs use the in-process IR (fn-201), fn-200's native lint and fn-203's Cases, not `model/ir` or `.lint.json`.

Move Closable, Pausable, Pollable, Terminable and Describable to `temporal.capabilities.product`, Retryable and Expirable to `temporal.capabilities.system`; update imports, lint and docs. IR: capability origin names change (rename map).
## Acceptance
- Two packages, each capability in the level it describes
- Models, tests and the Describable lint exception follow
- Rename map lists the moved origins

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: