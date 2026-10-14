I can't write that file from here, because this side channel has no tools. Here is the full content, ready to paste into `.plans/archive/lean/UMPIRE_LEAN_SLIM.md`. You can also ask in the main conversation to have it written and registered in `.plans/index.json`.

```markdown
# Slimming the Lean model

Assessment note, 2026-09-29. It records where the Lean model's size comes from and three
successive cuts that shrink it without losing what only Lean provides. It changes no rule and
approves no design. Line counts were measured on this date with `.lake/` excluded. Companion notes:
[UMPIRE_OUTSIDE_THE_BOX](../../lean/UMPIRE_OUTSIDE_THE_BOX.md) and [UMPIRE_GO](../UMPIRE_GO.md).

## 1. Where the lines are

`model/` holds 138k Lean lines, of which 40.7k are generated (`Temporal/API.lean`,
`Temporal/API/Types.lean`, `Temporal/DynamicConfig/Settings.lean`). The 97.6k authored lines:

| Part | Lines | Share |
| --- | --- | --- |
| Tests (`*Tests*` paths) | ~40,700 | 42% |
| `Umpire` production, the reusable toolchain | 42,800 | 44% |
| `Temporal` production | 9,500 | 10% |
| Feature models proper | ~1,500 | 1.5% |

`Umpire` production by subsystem: Command 8.1k (`Syntax.lean` alone 3.9k), Property 5.3k, Case
4.2k, Evidence 4.1k, Search 4.1k, Artifact 2.7k, ImplementationLink 2.6k, Variations 2.0k, Model
1.8k, Scenario 1.3k, then Value, Query, Exploration, Operation, Inventory, Promotion, Replay.
Declaration counts: 6,606 `def`s, 817 structures, 352 theorems, 1,484 `#guard`s, 910 `example`s,
788 `native_decide`s. The model is a compiler and checker written in Lean with proofs at a few
seams, not a proof development.

## 2. Why it grew

- **Lean charges for what Go derives.** fn-93 counted about 100 hand-written `X.name`
  functions, 13 copies of one registry pattern, 952 lines of hand-escaped JSON, and five identical
  error records. Every `enum` needs `BEq`, `DecidableEq`, `Repr`, `Finite`.
- **Per-behaviour testing in Lean.** The 120-line Nexus `Success` specimen carries a 1,890-line
  test file. The Nexus family is 7.8k lines, of which the product models are 1.2k.
- **Architecture churn.** Six weeks, 94 specs, 650 commits; 236k authored lines added and 139k
  deleted. fn-93 names about 15k surviving lines reached only from tests or facades.
- **The protocol is implemented twice by design.** SEM-18 makes Lean produce Cases, so `Shared`,
  `Testpilot.Authoring` and `Testpilot.Correlated` mirror Go, and `ModelLint` enforces module
  rules in Lean.
- **Every Feature module imports the 33k generated API lines** so `schema:` and `relates:` resolve
  at compile time. Unmeasured, but the first suspect for the ~15-minute cold build and the
  3–5 minute per-edit loop.

## 3. Cut one: move the back half to Go

Lean stops emitting Cases and emits one versioned proto export per Query (finite table, catalog,
witness, lowered clauses, bindings), rendered by the existing canonical ProtoJSON encoder and
decoded strictly in Go. Search and exploration candidate enumeration stay in Lean, because
witnesses remain Lean values replayed through the kernel (fn-88's rationale).

| Leaves Lean (production) | Goes to |
| --- | --- |
| `Umpire.Case` (4.2k) | Go Case compiler; fixture goldens prove parity |
| Realizations (1.2k), `Testpilot.Authoring` (0.8k) | Go realizations |
| `Umpire.Artifact` (2.7k) | The export; Go `internal/artifactv2` (1k) deleted |
| `Umpire.Evidence` (4.1k), `ImplementationLink.Application` (1.0k) | Nothing; no production consumer |
| `Umpire.Variations` (2.0k) | Delete; unreachable from every production root |
| `Umpire.Replay` and its bridge (0.9k) | Go replay |
| `Umpire.Promotion` (0.45k) | Go proposal renderer |
| Inventory, Evaluation Profiles, goldens, inspect, renderer (1.7k) | Go generators |
| `ModelLint` (1.4k) | Go lint |
| `Shared` (0.6k) | Go only |

About 24k production and 12–15k test lines leave Lean; the model lands near 60k. `ProtoJSON`,
`Carried` and `Protocol` stay for the export. SEM-18 is amended: Lean produces the export, Go is
the Producer.

## 4. Cut two: trim to what one feature family uses

- **Property language.** Feature models use `when:` 18 times, `holds:` 17 times, `relates:` twice.
  Keep Bool predicates, `when:`, and equality relations; drop operators, cardinality, presence,
  captures. About 4k production and 3k tests.
- **Retire the frozen reference search backend** once monitor lowering covers every used form:
  `Search.lean`, `Branches`, the 840-line differential test. The kernel replay gate stays. About
  2.5k production and 1k tests.
- **Prune single-use commands.** `from/restrict/extend` serves only the negative control. Keep
  `compose` (the fault story) and `examples:` (class splitting in exploration).
- **Drop or park the configuration models.** `System.Configuration`, `Callback`, `Matching` have
  no non-test consumer. About 1.7k plus 0.6k.
- **Change test style.** 258 `#guard_msgs` pins and row-by-row table pins become one fixture diff
  per machine and one rejection pin per error kind. About 8–10k across the tree.

Result: roughly 30–35k lines, about 18k of it production.

## 5. Cut three: a structure-only front end

If Lean's job narrows to parsing, name and dependency resolution, and structural invariants, with
no table enumeration, no search, and no Case production, it stops being the engine and becomes a
front end that emits an IR Go consumes. The floor for a Lean-hosted DSL at that point is roughly:

| Part | Lines |
| --- | --- |
| Command elaborators with curated errors, about ten commands | 4–5k |
| Finite deriving and key spelling | 1.5k |
| Definition IDs and fingerprints | 1k |
| Proofs kept (refinement, composition agreement) | 1–2k |
| IR exporter | 0.3k |

About 10k of core, plus 300–800 lines per feature, plus tests. The edit loop could drop from
minutes to seconds because nothing enumerates at elaboration. Whether Lean is still the right host
at that size is the language question in UMPIRE_OUTSIDE_THE_BOX; the cut is worth making either way.

## 6. What each cut loses

Three losses are real and should be decided, not absorbed:

1. **Evaluator agreement becomes a test.** Today the search monitor, Contract lowering, and runtime
   monitor share Lean semantics with agreement proofs. After cut one it is a differential test
   against the Go evaluator.
2. **White-box scaffolding goes.** The Evidence chain and `ImplementationLink.Application` are the
   path from internal records to a model verdict. The mode has been off since fn-81; reviving it
   means rebuilding that path in Go.
3. **Exploration narrows if class claims go.** Variations can go; `examples:` should stay, or
   exploration only walks rows and stops testing the author's groupings.

Everything else is unused today, redundant with Go, or a test-style preference.

## 7. What to keep regardless

- The refinement check. The ten-language comparison found it caught real spec bugs twice in one
  afternoon.
- `compose`, as the only way to state cross-entity and fault claims.
- `examples:` class claims.
- Deterministic Definition IDs, fingerprints, and fixtures.

## 8. Order

1. Measure which modules dominate build time before touching the generated-API imports.
2. Delete unreachable subsystems (Variations, Evidence chain, `ImplementationLink.Application`,
   retired artifact families): no behaviour change.
3. Add the proto export beside today's Case output; move Case production to Go behind
   byte-identical fixtures (UMPIRE_GO T8 is the proving ground).
4. Trim the property language and retire the reference backend.
5. Change test style.
6. Decide on cut three after UMPIRE_GO reports.
```
