---
satisfies: [R9]
---
# fn-85-model-side-effects-as-typed-actions-and.1 Early proof point: Umpire records for entities, actions, machines and observations; a path-driven Producer reproduces the async-Nexus Case

## Description
Prove the party-to-entrypoint design before any syntax exists (Early proof point, R9). Add the Umpire records the layers diagram names (`Entity`, `Action`, `InputField`, `Example`, `Observation`, `Machine` rows with guards and evidence, `Timer`, `SetupParameter`) as plain Lean structures under `Umpire.Command`, replace `Umpire.Case.Producer.Realization`'s whole-Program function with a binding from action classes to instructions, RPCs and entrypoints, assemble the Program and Contract from a Query's path in order, and hand-build Query 2 (schedule, async reply, succeeded completion) over those records with a Temporal realization. Its Case must equal the checked-in async-Nexus fixture with identities masked. Stop if the assembly needs a Nexus-specific branch in the Producer or cannot reproduce the template's edges from path order.

**Size:** M
**Files:** `common/testing/testpilot/internal/protocolmigration/**` and `.gitignore` and `tools/umpire/internal/retiredvocabulary/check.go` (the oracle retirement, first commit), `model/Umpire/Command/Records.lean` (new: the records; no syntax), `model/Umpire/Case/Producer.lean` (path-driven assembly; `Realization` rebinding; `Input` grows from one `operationRole` to entity instances), `model/Umpire/Case/Tests/Producer.lean`, `model/Temporal/Case/Realization/Nexus.lean` (new: the Nexus realization value replacing the `nexusOperation` template for this proof), `model/Temporal/Case/Tests/ProofPoint.lean` (new: masks identities and compares against the fixture bytes), `model/Temporal/Case/Template/NexusOperation.lean` (read only; deleted in task .11), `model/Umpire/ARCHITECTURE.md` (Producer paragraph)
**Touches:** [model/Umpire/Command/**, model/Umpire/Case/**, model/Temporal/Case/**, model/Umpire/ARCHITECTURE.md, common/testing/testpilot/internal/protocolmigration/**, tools/umpire/internal/retiredvocabulary/check.go, .gitignore]

### Approach
- **First, retire the protocol-migration oracle** (plan review round 1, blocker B1). It pairs each
  frozen pre-fn-87 baseline fixture with its regenerated counterpart one to one and fails on any
  difference no declared step explains, on an undeclared addition, and on a deleted baseline fixture,
  for which it has no removal list (`equivalence.go:578-623`); CI runs it
  (`.github/workflows/umpire.yml`). Its declared subject is fn-87 — "fn-87 changes the Testpilot
  protocol without changing any Verdict. This package proves it" — and fn-87's completion review has
  discharged that. Tasks .4, .8, .9, .10 and .11 all change the wire or the fixture set, and .10
  deletes a baseline fixture, which the package cannot express. Delete the package and its baseline in
  a separate commit before anything else, so the later tasks are not each carrying a mapping to a
  baseline nobody reads. The conformance `expected.json` pins stay: they, not the oracle, are the
  Verdict net.
- Read `.plans/LEAN_GUIDELINES.md` first. Start from the fn-87 shapes (order by default: an instruction depends on the previous one in its entrypoint; `after` for the rest; no environment list, reservations or outcome fields in the Case).
- Records are Temporal-free (`Umpire.*` names no RPC, instruction or event); the realization is the only Temporal-owned value. Bind per action class, not per party: today's `complete` is a handler-party action realized as a controller instruction over a handle slot, so a per-party binding is wrong by construction.
- Assembly: walk the Query's path; each `driven` action becomes the instruction its class binds, appended to its entrypoint; observed actions and `system` rows become Contract expectations through the existing projection lowering; evidence resolves through `Temporal.Case.EventKind`. Keep `Umpire.Case.Compiler` and the correlated lowering as they are.
- Masking: Case ID, Program and Contract IDs, run scope and provenance rows are replaced by placeholders on both sides; everything else must be byte-equal. Record the diff in the receipt if the comparison needs more masks and explain each.
- Do not touch `Umpire.Command.Syntax`; the `case` command and the templates keep working for the other fixtures until task .11.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Case/Producer.lean:86-101,170-192,347-427` — `Input`, `Realization`, `produce`
- `model/Temporal/Case/Template/NexusOperation.lean:99-137,139-218,219-253` — the nodes, the two hand-written Programs, the template's `Realization`
- `model/Temporal/Case/EventKind.lean:23-73` — the observation catalog
- `model/Temporal/Feature/Nexus/Success/Model.lean:100-105` — the Case this proof reproduces
- `tests/testcore/testpilot/testdata/async-nexus-case.json` — the fixture (post-fn-87 shape)

**Optional:**
- `model/Umpire/Command/Authoring.lean:138-168,242-323` — `DeclaredModel`, `declareModel` (the records the new ones sit beside)
- `model/Temporal/Feature/Nexus/DESIGN.md` section 3 — the specimen and the realization sketch

### Key context
- fn-84 .3's decision: behavior-neutral refactors must not strengthen validation; this task adds records and an assembly path and removes nothing.
- `Umpire.Command.Authoring` already imports `Umpire.Case.Producer`, so the command layer and the Producer are one import component; keep MOD-01 and SCP-02 green (`make lint-model`, LEAN_NUM_THREADS=1).

## Acceptance
- [ ] `Umpire.Command` declares the entity, action, input-field, example, observation, machine-row, timer and setup-parameter records with no Temporal name; `make lint-model` green
- [ ] `Umpire.Case.Producer` assembles Program and Contract from a path and a realization binding action classes to instructions, RPCs and entrypoints; no Nexus-specific branch exists in it, and non-test Lean under `model/Umpire` names no RPC, instruction or event kind (the three pre-existing matches are prose in `Exploration/Engine.lean` and `ARCHITECTURE.md` and fixture-identity test data in `Case/Tests/Producer.lean`)
- [ ] the hand-built Query 2 Case equals the async-Nexus fixture with identities masked, pinned by a Lean test that prints the first differing path on failure
- [ ] the existing fixtures still regenerate byte-identical through the templates (`make umpire-check-case-runtime-conformance`); focused `lake build Umpire.Case.Tests Temporal.Case.Tests` green
- [ ] the protocol-migration oracle is retired in its own commit: `common/testing/testpilot/internal/protocolmigration` and its frozen baseline tree are deleted, with the `.gitignore` negation for that tree and the `protocolMigrationBaseline` exemption in `tools/umpire/internal/retiredvocabulary/check.go` removed; `go test -count=1 -tags test_dep ./common/testing/testpilot/...` green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
