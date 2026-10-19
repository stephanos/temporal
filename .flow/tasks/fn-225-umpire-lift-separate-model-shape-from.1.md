---
satisfies: [R1]
---
# fn-225-umpire-lift-separate-model-shape-from.1 Split umpire-lift into shape and body lowering; tree into declarations and expressions

## Description
Pure refactor of model/umpire/lift/src: a shape module (single #[machine] module, imports, attributes, fields, starts, actions, claim/assumption/progress declarations and order) that never lowers statements or expressions, and a body/ module (expr, effects, vector, claim and progress bodies incl. path!, typer) that never walks items, taking a typed environment plus a syn node and returning Expr or Vec<Branch>. Split tree.rs into declarations and expr.rs. Public API, diagnostics text and locations unchanged; every golden byte-identical. A few lines on the seam in model/README.md and model/DESIGN.md. Verify make model-check, UMPIRE_BACKENDS=require and make model-time. Starts after fn-224 is closed in umpire's history.

## Acceptance
- [ ] Shape module never lowers bodies; body module never walks items
- [ ] tree.rs split into declarations and expressions
- [ ] All goldens byte-identical; diagnostics unchanged
- [ ] make model-check and UMPIRE_BACKENDS=require pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
