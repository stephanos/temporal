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
Blocked:
Owner override. Native Umpire framework DSL authoring, typing, runtime, caller/API retirement and native guidance come first. This original mixed/integration task remains open and held while native fn164.2/.10, fn165.7-.9, fn173.10/.11, fn123.9/.10 and fn153.7 complete. Consume native results later without repeating or claiming their deferred IR/Go/backend proofs. Retain all original exact IR, metadata, schema, compatibility, consumer and integration requirements. Full spec closure is not a native prerequisite. Ignore model/temporal entirely; no task resumes Temporal work without new owner authority. After native DSL completion, re-anchor this task to the final API before release. This is a deliberate priority/scope hold, not evidence of completion or a technical failure.
## Evidence
- Commits:
- Tests:
- PRs:
