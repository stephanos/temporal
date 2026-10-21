---
satisfies: [R1, R3, R4]
---
# fn-231-declarative-realizations-the.1 Realization macro crate: #[realization], #[observe], #[answer]; Step and Reply; generated Realize impl; activity rewritten with expansion snapshot

## Description
Add a proc-macro sub-crate under model/temporal providing #[realization(<module>)], #[observe(Service::Method, reads(..), ignores(..))] and #[answer(Task, reads(..), ignores(..))] plus the reserved #[propose]; return-type classification (state-valued, Step<A>, Projection); generated Realize impl dispatching on method path through the fn-229 helpers with Unsupported for unlisted methods. Add Step and Reply to the realization crate. Rewrite the activity Realization on the attributes; all activity goldens, wire goldens, slice and cover tests byte-identical. trybuild pins duplicate handler, unclassifiable return, wildcard enum arm; insta expansion snapshot via prettyplease. Boundary scan passes. Gate and model-time pasted.

### Execution scope and verification

**Touches:** `model/Cargo.toml`, `model/Cargo.lock`, `model/temporal/realization/**`, `model/temporal/feature/activity/temporal.rs`, `model/temporal/annotate/syntax/**`, `model/temporal/api/**`, `model/temporal/testing/**`, `model/temporal/hosts/go/umpire_temporal_guest.wasm`, `model/temporal/realization/README.md`, `model/README.md`
**Quick:** focused macro parser/expansion and trybuild controls; actual realization/helper tests with refusal-order, unknown Cancel code, Current/one-step/unexplained projection, callback and partial-reply controls; native activity fixtures and existing fn-218 cover controls; generic source boundary; `make umpire-guest` followed by actual `make umpire-check-model-slice` and `make umpire-check-model-gaps`, retaining all cases/privacy/goldens and existing operation/aggregate limits.
**Full gates:** `make model-check`; `UMPIRE_BACKENDS=require make model-check`; `make model-time`.

Use the integrated source, not the old `features/activity` spelling or five-message
examples. The read-only seam research is `.flow/tmp/fn231-realization-seam-research.md`.
It was re-anchored before fn-230.2; recheck current source rather than assuming its
line numbers or the former test-build exclusion still apply. The current catalog
has sixteen WorkflowService request/response messages plus nested declarations.
Reuse the admitted shared parser and descriptor graph with an acyclic dependency
boundary. No second parser for Catalog or generated API imports in its syntax crate.

Preserve the exact existing observable behavior. Select method before decoding;
Cancel's unknown raw i32 code yields empty candidates, and Cancel/Describe retain
their distinct decode refusal mappings. State conversion is explicitly owned by
the Realization; Started from Describe must not invent Poll. Exhaustiveness checks
target admitted generated-enum matches, preserving legitimate byte-policy fallbacks.
Retain existing helper, atomicity, rollback, external-payload and cover controls.
Partial callbacks and outgoing SDK replies are not complete catalog-valid requests;
do not newly require SDK-filled task tokens or double-wrap the Reply envelope.
Document the `Reply.request` packing contract and pin actual decoded outgoing bytes.

Unknown/overlapping path and empty-ignore admission must follow the parent contract.
Keep incoming callback, observed response and outgoing request contexts distinct.
Reasoned ignores preserve currently accepted unmodeled fields; unsupported checks
use decoded protobuf optional/oneof presence, never opaque IgnoredField presence.
Do not defer a runtime safety/admission obligation merely because the full coverage
golden belongs to task .2. Any shared metadata seam required now must support that
successor without a duplicate declaration list. The runtime owns state freezing,
transition judgment and transactional publication; do not invent new quiescent or
pending semantics absent from its existing trait. Propose remains reserved only.

No Model edits, phase-mapping changes, fitted judgments, report/rows/states/wire or
live-gap golden updates, ABI/protocol changes, dependency/version additions, Go/SDK
panic changes, timeout extensions or unrelated refactors. Existing workspace test
dependencies may be reused. Refresh committed guest bytes after Rust changes with
the admitted generic builder. The separate fn-232 macOS proof remains open and
does not block this task. Task .2 owns realization.snap.txt and DESIGN.md.
## Acceptance
- [ ] Macro crate with the three attributes and reserved propose; Step and Reply
- [ ] Activity Realization on attributes; goldens, slice and cover tests byte-identical
- [ ] trybuild refusals and expansion snapshot pinned; boundary scan passes

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
