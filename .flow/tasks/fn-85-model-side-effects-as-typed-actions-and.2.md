---
satisfies: [R1, R2]
---
# fn-85-model-side-effects-as-typed-actions-and.2 The entity, action and observation commands; enum constructors with finite fields; schema checks

## Description
Give the Model file the `entity`, `action` and `observation` commands (R1, R2): entities with `refer:` and `key:`; actions with `party:`, `on:`/`creates:`, optional `schema:`, `input:` fields over finite enums whose constructors may carry finite fields, optional `results:` and `examples:`; declared derived observations with `on:`/`read:`; evidence names resolving against the realization's catalog or a declaration. Every rejection the two requirements list is a located error pinned by `#guard_msgs`.

**Size:** M
**Files:** `model/Umpire/Command/Syntax.lean` (three new commands; the `enum` macro accepts constructor fields `| handlerError (retryable : Bool)`; `domainConstructors` admits constructors with finite fields and enumerates their members), `model/Umpire/Command/Authoring.lean` (elaboration into the task .1 records; `Origin`-derived Definition IDs), `model/Umpire/Command/Registry.lean` (entity, action, observation entries), `model/Temporal/Case/Schema.lean` (new: resolve a `schema:` message name against the generated API and check class members and examples against its fields at elaboration; store only the name), `model/Temporal/Feature/Nexus/Success/Tests.lean` or a new `model/Temporal/Feature/Nexus/Tests/Commands.lean` (the `#guard_msgs` specimens), `model/Umpire/Command/Tests/Authoring.lean`
**Touches:** [model/Umpire/Command/**, model/Temporal/Case/Schema.lean, model/Temporal/Feature/Nexus/**]

### Approach
- Reading rule from fn-83 .15: column-0 declaration kind, indented `word:` key, everything else an author name or value; follow the `elab "property"` shape for located diagnostics and `addConstInfo` hovers.
- `enum` with fields: the macro today parses only `| ident`; extend it to `| ident (binders)` and derive the finite member set as the product of constructor fields, each field an enum or `Bool`; a non-finite field rejects in place.
- `schema:` is Temporal's: `Umpire.Command` stores the message name as a string and an opaque check hook; `Temporal.Case.Schema` resolves it through the generated `Temporal.API` schema nodes the way `EventKind.attributeFields` walks a response, checks each class member and example against the message's fields, and discards the descriptor (the memory entry on canonical identity embedding a 27 MB schema is the reason to keep the name only).
- Parties are strings declared by use; `system` is reserved and rejects as an action's party.
- Rejections to pin: undeclared entity reference, duplicate key name, instance bound of zero (R1); `system` party, non-finite input or constructor field, unresolvable schema, member or example outside the schema, example matching no class, evidence name neither catalogued nor declared (R2).

### Investigation targets
**Required:**
- `model/Umpire/Command/Syntax.lean:34-46,116-134,361-427` — `enum`, `domainConstructors`, `resolveDeclared`, the `property` elab to mirror
- `model/Umpire/Command/Authoring.lean:31-77,91-98` — `Origin`, `DeclaredNames`
- `model/Temporal/Case/EventKind.lean:23-73` — the schema walk to reuse for `schema:`
- `model/Temporal/Feature/Nexus/Success/Tests.lean:593-760` — the `#guard_msgs (error) in` specimen style

**Optional:**
- `model/Temporal/API.lean:1688,4082,4190` — the three Nexus messages the examples name

### Key context
- AUT-09: enum constructors in declaration order are the ordered domain; keep that for constructors with fields (order by constructor, then by field member order).

## Acceptance
- [x] `entity`, `action`, `observation` elaborate into the task .1 records with Definition IDs derived by `Origin`; a Model file declaring the DESIGN.md section 3 entities, actions and `pendingAttempts` compiles
- [x] an `enum` constructor with finite fields is a class whose members enumerate; `handlerError (retryable := true)` resolves as a pattern
- [x] `schema:` resolves its message names at elaboration and stores only them (`Tests/Commands.lean` asserts each stored entry is shorter than the descriptor of the message it names). **Checking class members and examples against the message moved to `.8`:** the members the design's own examples name are values of `temporal.api.nexus.v1.HandlerError.error_type`, which the generated schema types as a `string`, so the descriptor carries nothing to check them against until a payload declares typed fields
- [x] every R1 and R2 rejection this task's syntax can produce has a `#guard_msgs` specimen at the offending line; `lake build TemporalModelTests` green; `make lint-model` green at the 163 baseline; `make umpire-check-regression` exit 0. **Two rejections have no syntax to fire on here and are recorded where they do:** an instance bound of zero on `.4`, which declares instance bounds, and an evidence name neither catalogued nor declared on `.14`, which introduces `evidence:`


## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: