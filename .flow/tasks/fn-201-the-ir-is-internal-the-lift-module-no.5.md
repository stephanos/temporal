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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
