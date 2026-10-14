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
`entity`, `action` and `observation` are commands. Each elaborates into the plain record
`Umpire.Command.Records` owns, records a registry entry a later command resolves against, and carries
a Definition ID computed from the namespace that declared it. `DESIGN.md` section 3's entities, input
domains, actions and derived observation are the specimen they are checked against
(`Temporal/Feature/Nexus/Tests/Commands.lean`) — the machines (task `.14`) and the actions the design
marks `fn-79` excepted. `Temporal/Feature/Nexus/Tests/SecondModel.lean` is a second Model file
declaring the same names and referring across the boundary, which is what tells a file-scoped rule
from an import-closure-scoped one.

An `enum` constructor may carry finite fields. A **class** is a member of the domain, not a
constructor: `handlerError (retryable : Bool)` is one constructor and two classes, which is the
granularity `DESIGN.md` writes an example at. Classes are built as trees and compared structurally,
so a reordered binding is the same class, a qualification is checked rather than discarded, and two
constructors whose names would run together in one string stay two classes.

`schema:` stores the message names and asks a platform whether they resolve. `Umpire.Command.Schema`
holds the hook; `Temporal.Case.Schema` installs Temporal's answer at import, resolving against the
descriptor closures of the four RPCs a Model's messages travel on. Only names survive elaboration: a
guard says each stored entry is shorter than the descriptor of the message it names.

### Decisions

- **The three command words are not reserved tokens.** They are also field names in the records the
  commands build (`EntityReference.entity`, `FiniteTransitionRow.action`), so reserving them would
  make those fields unwritable; a plain `&"entity"` does not dispatch on a bare identifier.
  `nonReservedSymbolNoAntiquot word (includeIdent := true)` indexes under both.
- **A class is a member of its domain.** `DESIGN.md` section 2.2 said "each constructor is a class"
  and then, one sentence later, called a fully applied constructor a class. The paragraph is amended
  with a dated note; the implementation follows the members reading. It follows that nothing in a
  Model counts the realized values a class covers, so R8's abstraction claim is triggered by the
  presence of an example — recorded on `.7`.
- **The schema check decides whether a name resolves, not whether a class member is in it.** The
  members the design's own examples name are values of `temporal.api.nexus.v1.HandlerError.error_type`,
  a protobuf `string`. R2's member check moved to `.8`.
- **Two R1/R2 rejections have no syntax to fire on here.** An instance bound of zero was already
  `.4`'s; an evidence name neither catalogued nor declared moved to `.14`.

### Rejections pinned by `#guard_msgs`, each at the offending line

An undeclared entity reference; a duplicate key name; a second `refer:` by one field; `system` as a
party; `on:` and `creates:` together; a duplicate key; a missing `party:`; a missing `read:`; a
non-finite input field; a non-finite constructor field; an unresolvable `schema:`; an example
matching no class; an example binding a field the constructor does not carry; a bare constructor that
carries fields; a positional argument; another domain's class; two examples for one class; a finite
domain that is not an `enum`.

### Review

Seven rounds, same reviewer, fresh context each round; rounds one through six each found something
real and were fixed, round seven is SHIP. What they found, in order: an import-closure-scoped
duplicate-key rule; cross-file Definition IDs rebuilt in the referring file's family; the same defect
left half-fixed for enums; a re-derived member walk that could exhaust the stack with no location; a
string flattening that silently stored an example against a class its author did not write; a
discarded constructor qualification, reachable in the feature's own vocabulary; and a missing
uniqueness rule on `refer:` fields with three stale-prose sites.

Implementer and reviewer are the same session — `codex`, `cursor-agent` and `grok` are not installed
in a cloud session, so every backend falls back to the session model. `.1`, `.2` and `.3` each owe a
cross-model re-review before the spec's completion review.
## Evidence
- Commits: 2532f92bb, 494cffbf0, 976d9afd3, 58520e1f3, 860b57af0, d57f9c1f7, df61b4ce8, dbd862d1b, 3f21cd33b, e81a39471, a7c4df58f, 33c0b887a, 0b1905849
- Tests: cd model && lake build, LEAN_NUM_THREADS=1 make lint-model, make umpire-check-regression
- PRs: