---
satisfies: [R3, R4, R11]
---
# fn-92-compose-entity-machines-into-one-system.2 The compose command: generated unions, synchronization, key resolution, reachable literal table

## Description
Add `compose` to `Umpire.Command` with `for:`, `state:`, `members:`, `sync:`, `starts:`, `ends:`; generate tagged-union Action, Outcome, and Fact types; key unsynchronized actions `<field>.<member key>` and synchronized ones by their `sync:` name; extend Scenario and Property key resolution; build the reachable literal table by a frontier walk; elaborate to a `CheckedModel`; reject a `set` over a composition. Proofs and the per-composition check are task .3.

**Size:** M
**Files:** `model/Umpire/Command/Compose.lean` (new), `model/Umpire/Command/Syntax.lean` (keyword, keys, elaborator dispatch, Scenario `actions:` and `starts:` resolution, Property `when:` resolution, `set` rejection), `model/Umpire/Command/Registry.lean` (record compositions), `model/Umpire/Command/Tests/Compose.lean` (new), `model/Umpire/Command/Tests.lean` (register)
**Touches:** [model/Umpire/Command/Compose.lean, model/Umpire/Command/Syntax.lean, model/Umpire/Command/Registry.lean, model/Umpire/Command/Tests/Compose.lean, model/Umpire/Command/Tests.lean]

### Approach
- Keyword via `declarationKeyword` (`Syntax.lean:374-384`); `compose` is not the reserved party spelling (`:1522`).
- Generate `<name>.Action`, `<name>.Outcome`, `<name>.Fact` inductives with one constructor per member field wrapping the member's type; catalog keys `<field>.<member key>`; member Definition IDs kept inside constructors as `instances` keeps them (`Instances.lean:72-190`); synchronized action constructor and key named by the `sync:` line.
- Frontier BFS from composed start states over member tables: unsynchronized action steps its owner; synchronized step enabled only when every participant has a row, results the ordered product sorted by `stepOrderKey`, refused above 16; classless participants match every class; timers per member; catalog = reachable states sorted by `modelValueOrderKey`; emit a literal `FiniteTable`; bound check reachable states × actions vs `enumerationBound` (`Finite.lean:123`); drop never-enabled actions with a located warning.
- Key resolution: the Scenario elaborator keeps only the last dotted component (`Syntax.lean:595`) and splits `starts:` on `-` (`:561-563`); extend both, and Property `when:`, to resolve `operation.handlerReply (async)` and `sync:` names against the composed catalog. Every path that reads `members (α := State)` for a machine reads the catalog for a composition (`declareModel`, `Registry.recordModel.states`, Scenario `starts:`).
- Located errors pinned with `#guard_msgs`: unsynchronized shared name, participant input mismatch, product above 16, bound exceeded with counts, `set` over a composition (`Syntax.lean:2700-2722`).
- Fingerprint test: reorder `members:` and `sync:` lines in a fixture composition and assert equal fingerprints.

### Investigation targets
**Required:**
- `model/Umpire/Command/Instances.lean:63-190`
- `model/Umpire/Command/Syntax.lean:548-620, 561-595, 1656-1700, 1866-2000, 2157-2170, 2379-2492, 2700-2722`
- `model/Umpire/Command/Authoring.lean:170-300`
- `model/Umpire/Command/Finite.lean:82-170`

### Key context
- The author's state structure derives no `Finite`; the reachable list is the catalog.

## Acceptance
- [ ] `compose` elaborates a fixture composition to a `CheckedModel` usable by `property`, `scenario`, `limits`, `query`, with Scenario `actions:` and Property `when:` resolving composed and synchronized keys
- [ ] Generated unions with member-prefixed keys; no catalog collision for two members with `accepted`
- [ ] Reachable literal table; fingerprint unchanged under `members:`/`sync:` reordering (pinned)
- [ ] Every listed located error pinned; `set` over a composition rejected
- [ ] `make lint-model` passes

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
