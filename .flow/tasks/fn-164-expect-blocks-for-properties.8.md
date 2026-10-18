---
satisfies: [R4, R5, R6, R7]
---
# fn-164-expect-blocks-for-properties.8 Remove framework lambda finishers and document the final expect surface

## Description
**Superseded by fn-190.4 (owner phasing 2026-10-17).** The native batch already removed the public lambda finishers; fn-190.4 retires `expect` itself and documents the final claim surface.

Framework-only final public surface, independent of disabled Temporal callers. Touches framework DSL Claims/Syntax, claim/pattern/witness tests, generic capture/export seams, syntax lint and framework-owned author guidance. Remove public holds and holdsAcross together after .7. Internal sugar/witness construction uses direct PropertyDecl or a private core factory; core stored predicate functions and fields keep their meaning. Do not expose a renamed/deprecated compatibility finisher and do not make core files import sugar. Preserve direct ordinary-core oracle fixtures and generic constructed-predicate consumption. External author-package compile negatives prove old forms inaccessible; internal package tests alone are insufficient. Update Core form docs and final framework examples. Exact inventories/IR parity prove no lost declarations, wrapper functions or changed parameter identities. Use make check-framework. Temporal migration and original full-scope .6/R6 remain held; that hold does not block this framework removal.
## Acceptance
Public framework holds/holdsAcross and compatibility routes are absent; all executable framework callers use expect or explicitly inventoried core constructors; external old-spelling compile refusals, full exact generic IR comparisons, framework tests and final author guidance agree; no Temporal source is read/restored; full Temporal migration remains unclaimed.

## Done summary
Already superseded by fn-190; status hygiene 2026-10-18

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: