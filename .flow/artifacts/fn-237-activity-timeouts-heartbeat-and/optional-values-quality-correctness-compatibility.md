AXIS: correctness

- Scope: 2 dirty files · Remaining findings: 0 · New findings: 0 · Critical 0 · Should Fix 0 · Consider 0 · Ship: ✅ Ship for this compatibility repair.

**walked:** With no bounded declarations, optional parsing now falls through to existing scalar/enum resolution, restoring ordinary `Some`/`None` constants and fieldless `Option` enums. With bounded declarations present, canonical-path, shadowing, payload, arity and encoding checks remain unchanged. Unsupported primitive options still refuse through existing type/expression admission.

The two new tests assert successful admission; existing refusal assertions remain intact. No builds, tests, Flow calls or edits were performed during this audit. Final gate results remain pending.
