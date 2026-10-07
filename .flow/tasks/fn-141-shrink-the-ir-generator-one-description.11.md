---
satisfies: [R5, R6, R11, R12]
---
# fn-141-shrink-the-ir-generator-one-description.11 Export compositions and syncs

## Description
`sync` and `replaces` are no-ops at run time today; only the lifter knows what they pair. Make them record it, then export compositions.

**Size:** M
**Files:** `model/umpire/Compose.scala`, `model/irgen/Compositions.scala`
**Touches:** [model/umpire/**, model/irgen/**, model/temporal/**]
**Deferred:** see MILESTONES.md, Deferred, fn-141. Planned 2026-10-06 against the tree before the DSL batch. The files and names below are that tree's and carry no line numbers on purpose: when the spec is revived, re-read them and recount before starting, since the DSL batch and fn-140 rewrite them.

### Approach
- `sync` and `replaces` record their pairs and names in the `Syncs` object (R5).
- A member selector (`_.order -> Machine`) yields its field name at run time. The composition already probes the selector for the member; extend the probe to the field.
- `synced` and `own` export their composed keys with today's spelling.
- Derived compositions (`withMember`) export from the built value.
- Semantic rules stay, per the ledger: a member's state type, an action no sync pairs, a duplicate sync name.

### Investigation targets
**Required:**
- `model/irgen/Compositions.scala` (`compositionOf`, `sync`, `replaces`, `withMember`, `composedKey`)
- `model/umpire/Compose.scala` (`Composition`, `Shape`, `selected`, `Syncs`, `Composed`)

## Acceptance
- [ ] Compositions in the IR come from the exporter; `model/irgen/Compositions.scala` is deleted.
- [ ] A test reads back what each `sync` and `replaces` paired.
- [ ] The ledger's composition rows have their outcomes.
- [ ] `make umpire-gen-model` leaves `model/ir`, `model/cases` and the lifter fixtures' expected IR byte-identical (R11); any difference stops the task until it is traced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
