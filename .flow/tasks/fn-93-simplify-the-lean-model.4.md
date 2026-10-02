---
satisfies: [R4]
---
# fn-93-simplify-the-lean-model.4 WireName derivation: settle placement and convert the core-side enums (A2, proof point)

## Description
Lane A2, first task and the spec's early proof point. Build the `WireName` derivation and prove where it can live under `semanticModelIsolation` before converting anything else. Then convert the enums outside the semantic roots whose modules lane B keeps.

**Size:** M
**Files:** new `model/Umpire/WireName.lean` (handler or macro; name may differ), `model/Umpire/Tests/WireName.lean` (new), `model/Umpire/Core.lean`, `model/Umpire/KnownGap.lean`, `model/Umpire/Command/Records.lean` (`PartyBinding`, `SetPurpose`, `CoverageGoal` names), `model/Umpire/Inventory/Types.lean`, `model/Umpire/Replay.lean`, `model/Umpire/Evaluation.lean`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/WireName.lean, model/Umpire/Tests/**, model/Umpire/Core.lean, model/Umpire/KnownGap.lean, model/Umpire/Command/Records.lean, model/Umpire/Inventory/Types.lean, model/Umpire/Replay.lean, model/Umpire/Evaluation.lean, model/UmpireTests.lean, model/ModelLint/ImportGraph.lean]

### Approach
1. **Placement probe first (DG4).** Write the handler modelled on `model/Umpire/Command/Finite.lean:290-307` (register in `initialize`; generate top-level `T.name`, `T.ofName?`, `T.all` as structural `match` defs via `mkIdent`). Import it from one throwaway semantic-root module (e.g. `Umpire.Query`) and run `LEAN_NUM_THREADS=1 make lint-model`. If `semanticModelIsolation` fires (the check follows external `Lean.*` edges), switch to the recommended fallback: an `Init`-only command macro that declares the enum, its overrides and the three functions together (macros need no `Lean.Elab` import at the use site). Record the chosen form and the lint output in the receipt; later A2 tasks use it unchanged.
2. Overrides: `attribute [wire_name "…"] T.ctor` after the declaration, then `deriving instance WireName for T` (Lean rejects attributes on constructors). In the macro form, the override is part of the macro syntax.
3. Types with payload constructors: `name` ignores fields; `ofName?`/`all` only for field-less types; refuse otherwise with a message naming the type.
4. Before converting, check each hand-written parser (`ofName?`/`parse?` in `Fingerprint.lean`, `KnownGap.lean`) for aliases or case folding the derived parser would drop; such a type keeps its parser.
5. Convert the listed modules' name functions. For each: add an equivalence check for every constructor (kernel theorem `cases c <;> rfl` where proofs use the function, `#guard` table otherwise), build, delete old function and check in the same commit.
6. `constructorClassifiers` keeps its descriptions where they are (they render into INVENTORY.md's Meaning column); only the `name` field comes from the derived name.

### Investigation targets
**Required:**
- `model/Umpire/Command/Finite.lean:290-307` — deriving handler pattern
- `model/ModelLint/ImportGraph.lean:182-199,400-415` — `semanticRoots` and the isolation rule
- `model/Tools/LeanImportGraph.lean:55-70` — external-edge traversal
- `model/Umpire/OutcomeClassification.lean` — classifier type (Init-only module)
**Optional:**
- `model/Umpire/Search.lean:582,631` — `constructorClassifiers_exactlyOne` proof shape (converted in the next task)

### Key context
- A derived name must stay definitionally usable where proofs unfold the old function; if a proof breaks, add the API lemma (LEAN_GUIDELINES §2) instead of unfolding generated code.

### Quick commands
```sh
cd model && lake build Umpire UmpireTests
LEAN_NUM_THREADS=1 make lint-model
make umpire-check-goldens && make umpire-check-inventory
```

## Acceptance
- [ ] Receipt records the placement decision (handler vs `Init`-only macro) with the lint output that forced it
- [ ] Derivation tests cover kebab default, override, payload constructor refusal, `ofName?` round trip, `all` order = constructor order
- [ ] Listed modules' name functions derived; equivalence checks passed before deletion (commit shows both)
- [ ] Goldens, INVENTORY.md, Fingerprints byte-identical; `lint-model` green


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
