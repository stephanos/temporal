---
satisfies: [R5]
---
# fn-201-the-ir-is-internal-the-lift-module-no.5 Numeric contract decision recorded in SEMANTICS and the builtins table

## Description
**Owner decision 2026-10-18:** host Int (32-bit wrapping, division by zero throws). Write it into SEMANTICS and the builtins table, retire the int64 wording, and note that emitters state the bound; the unbounded-arithmetic lint kind is fn-200.7.

Present the choice (host `Int` semantics versus a checked bound) with the measurement note's Bank evidence, obtain the owner's decision, write it into SEMANTICS and the builtins table, and remove the IR's int64 wording.

**Files:** model/framework/SEMANTICS.md, model/builtins.json, .flow/specs/fn-201-*.md.

**Quick commands:** `grep -rn int64 model/framework model/builtins.json` returns nothing.
## Acceptance
- Dated decision in Decision Context; docs updated; grep clean.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: