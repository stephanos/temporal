---
satisfies: [R1]
---
# fn-245-finite-collections-sets-maps-and.1 IR types, canonical values and expression nodes for sets, maps and list queue operations

## Description

Extend `umpire-ir` (`model/umpire/ir/src/{lib,literal,value,expr,print}.rs`) with `Type::Set(Box<Type>)`, `Type::Map(Box<Type>, Box<Type>)`, canonical `Value::Set` and `Value::Map`, and matching `Literal` variants. Add their one typed admission through `Literal::admit`, reusing scalar admission for every element/key/value and refusing duplicates after typed identity conversion. Canonical order follows enum declaration position, bool false/true and numeric order. Equality, hash and Display remain structural.

Add the spec's collection expression nodes, extending Length. Quantifiers carry a typed `Elements`/`Keys`/`Values` projection so same-typed map keys and values remain distinct; values iterate present entries in key order, including repeated values. Front carries its fallback, MapGetOr carries key and fallback, and PopFront returns the successor sequence. Extend `ExprKind::children()`/`children_mut()` in evaluation order for every new child, including eager defaults and predicate expressions; ensure binder scoping and the existing `Expr::infallible` logic recognize short-circuit and fallible forms rather than assuming new nodes cannot fail. Keep recursion-only consumers on these helpers; per-variant semantic consumers gain the required arms.

Starts only after committed fn-237 as a whole and fn-250, following fn-248.3's traversal helpers. These are the existing spec dependencies and owner sequencing; this plan does not edit metadata. No implementation is admitted by the review. Existing Models keep their checking meaning; reviewed goldens may change.

## Acceptance

- [ ] All new types, literals, values and nodes have structural equality/hash/Display and canonical type-aware ordering; shared Literal admission rejects duplicate elements/keys, wrong nominal identity and out-of-domain members.
- [ ] Each new expression child is visited once by both shared traversal helpers; binder/projected type and scoped predicate survive renaming, priming and rewriting.
- [ ] IR unit controls cover empty/nonempty values, canonicalization, typed duplicates, all three quantifier projections, same-typed map keys/values and eager-default child order.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
