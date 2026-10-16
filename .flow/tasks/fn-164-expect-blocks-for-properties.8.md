---
satisfies: [R4, R5, R6, R7]
---
# fn-164-expect-blocks-for-properties.8 Remove framework lambda finishers and document the final expect surface

## Description
Framework-only final public surface, independent of disabled Temporal callers. Touches framework DSL Claims/Syntax, claim/pattern/witness tests, generic capture/export seams, syntax lint and framework-owned author guidance. Remove public holds and holdsAcross together after .7. Internal sugar/witness construction uses direct PropertyDecl or a private core factory; core stored predicate functions and fields keep their meaning. Do not expose a renamed/deprecated compatibility finisher and do not make core files import sugar. Preserve direct ordinary-core oracle fixtures and generic constructed-predicate consumption. External author-package compile negatives prove old forms inaccessible; internal package tests alone are insufficient. Update Core form docs and final framework examples. Exact inventories/IR parity prove no lost declarations, wrapper functions or changed parameter identities. Use make check-framework. Temporal migration and original full-scope .6/R6 remain held; that hold does not block this framework removal.

## Acceptance
Public framework holds/holdsAcross and compatibility routes are absent; all executable framework callers use expect or explicitly inventoried core constructors; external old-spelling compile refusals, full exact generic IR comparisons, framework tests and final author guidance agree; no Temporal source is read/restored; full Temporal migration remains unclaimed.

## Done summary
Blocked:
Owner override. Native Umpire framework DSL authoring, typing, runtime, caller/API retirement and native guidance come first. This original mixed/integration task remains open and held while native fn164.2/.10, fn165.7-.9, fn173.10/.11, fn123.9/.10 and fn153.7 complete. Consume native results later without repeating or claiming their deferred IR/Go/backend proofs. Retain all original exact IR, metadata, schema, compatibility, consumer and integration requirements. Full spec closure is not a native prerequisite. Ignore model/temporal entirely; no task resumes Temporal work without new owner authority. After native DSL completion, re-anchor this task to the final API before release. This is a deliberate priority/scope hold, not evidence of completion or a technical failure.
## Evidence
- Commits:
- Tests:
- PRs:
