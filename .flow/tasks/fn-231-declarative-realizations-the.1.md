---
satisfies: [R1, R3, R4]
---
# fn-231-declarative-realizations-the.1 Realization macro crate: #[realization], #[observe], #[answer]; Step and Reply; generated Realize impl; activity rewritten with expansion snapshot

## Description
Add a proc-macro sub-crate under model/temporal providing #[realization(<module>)], #[observe(Service::Method, reads(..), ignores(..))] and #[answer(Task, reads(..), ignores(..))] plus the reserved #[propose]; return-type classification (state-valued, Step<A>, Projection); generated Realize impl dispatching on method path through the fn-229 helpers with Unsupported for unlisted methods. Add Step and Reply to the realization crate. Rewrite the activity Realization on the attributes; all activity goldens, wire goldens, slice and cover tests byte-identical. trybuild pins duplicate handler, unclassifiable return, wildcard enum arm; insta expansion snapshot via prettyplease. Boundary scan passes. Gate and model-time pasted.

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
