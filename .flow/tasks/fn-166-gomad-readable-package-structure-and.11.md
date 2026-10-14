---
satisfies: [R11]
---
# fn-166-gomad-readable-package-structure-and.11 Rename the gomad3 module and tree to gomad

## Description
After the identity-rotating structure batch settles the final tree, rename tools/gomad3 to tools/gomad and change the nested Go module/import path from go.temporal.io/server/tools/gomad3 to go.temporal.io/server/tools/gomad. Update workspace references, commands, scripts, CI routing, generated/source identities, fixtures, and documentation in one atomic move. Preserve the gomad CLI name and behavior; do not retain a gomad3 compatibility tree or import shim.

## Acceptance

Current scope (owner decision, 2026-10-14): apply [native hibernation](../../MILESTONES.md#native-backend-hibernation); native execution obligations transfer to fn-128/fn-149, superseding earlier native completion blockers in this task. All retained source/portable and actual WASM criteria remain required; this amendment supplies no native pass.

The repository has no surviving tools/gomad3 path or go.temporal.io/server/tools/gomad3 import/module reference outside intentional historical records. All workspace, build, generator, architecture, lint, WASM, qualification, documentation, and fixture references use tools/gomad. The rename is atomic, identities and generated outputs are rotated from the final source closure, and the full post-rotation gate set passes without compatibility shims.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
