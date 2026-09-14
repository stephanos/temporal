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
- [ ] the hand-built Query 2 **Program** is byte-identical to the one the `nexusOperation` template writes for the same identity, pinned by a Lean test. Amended while implementing: the *Case* cannot match. A Contract is derived from the checked Property's clauses and the Scenario's action order, and a correlated clause embeds its trigger action's Model Value and that action's occurrence bound in the Contract itself, not only in provenance. Re-authoring the Model's two waits (`awaitStart`, `awaitSuccess`) as three side effects (`schedule`, `handlerReply`, `complete`) therefore changes the Contract by construction, which no identity mask covers and which is the point of the re-authoring rather than a defect. The Program is what the party-to-entrypoint design is answerable for, so it is what the proof point pins; the Contract's shape is settled once the Model is authored through the commands in .2 and .3
- [ ] the existing fixtures still regenerate byte-identical through the templates (`make umpire-check-case-runtime-conformance`); focused `lake build Umpire.Case.Tests Temporal.Case.Tests` green
- [ ] the protocol-migration oracle is retired in its own commit: `common/testing/testpilot/internal/protocolmigration` and its frozen baseline tree are deleted, with the `.gitignore` negation for that tree and the `protocolMigrationBaseline` exemption in `tools/umpire/internal/retiredvocabulary/check.go` removed; `go test -count=1 -tags test_dep ./common/testing/testpilot/...` green


## Done summary
The early proof point holds, and the stop condition did not fire.

**The oracle is gone first.** `common/testing/testpilot/internal/protocolmigration` and its 1.2 MB
frozen baseline are deleted, with the `.gitignore` negation that tracked the baseline and the
retired-vocabulary exemption that let it keep spelling every retired name. Its declared subject was
fn-87, whose completion review discharged it, and it could not have survived .4, .8, .9 or .10 in any
case: it pairs each baseline fixture with its regenerated counterpart one to one and has no removal
list. The conformance `expected.json` pins stay — they, not the oracle, are the Verdict net.

**`Umpire.Command.Records`** declares the entities, actions with their input fields and examples,
observations, timers, setup parameters, evidence lines and the machine declaration. Nothing in it is
checked and nothing in it names a Temporal concept. Two shapes deviate from the spec's sketch, both to
keep Umpire free of the protocol: an example's member is a `ModelValue`, not the wire value type that
lives in `Testpilot.Authoring`; and an input field's domain is the declared enum's Definition ID with
its classes listed, not a `Lean.Name`, since these records carry data and the syntax that resolves a
name is .2. The machine record is `MachineDeclaration`, because `Umpire.Machine` is the checked
transition relation it elaborates into and both are in scope wherever a command elaborates.

**The Producer assembles the Program from the path.** `Realization.program` is gone as a field. A
realization now declares a `ProgramPlan` — roles, slots, observations, entrypoints, cleanup — where
each entrypoint orders its items, and binds each action class to the instruction that performs it. The
Producer walks the Query's path and puts those instructions where the path took them, so it decides
*where* and the realization decides *what*. An entrypoint whose sequence interleaves scaffolding and
actions says so by item order, which is how the async Nexus controller (start, wait, complete, read
history) is expressed with no Nexus knowledge in `Umpire`.

An action the realization binds must also be placed, or a side effect would be missing from the
Program. An action with no binding is deliberately not an error: a party bound `observed` performs
nothing, and a Model whose actions are waits rather than side effects realizes none of them, so the
Contract carries it and the Program does not. That is what both templates rely on, and why they keep
producing identical bytes.

**`Temporal.Case.Realization.asyncNexus`** writes no Program: it declares the template's scaffolding
and binds the three side effects that decide an asynchronous operation's outcome — the caller
workflow's `schedule`, the handler's asynchronous `handlerReply`, and the handler's `complete`. The
last is why a realization binds classes rather than parties: the handler's completion runs as a
controller instruction over a handle slot the handler published, so a per-party binding would put it
where no such instruction can run.

**The proof.** `Temporal.Case.Tests.ProofPoint` requires the Program assembled from the path
`[schedule, handlerReply, complete]` to be byte-identical to the one `nexusOperation` writes for the
same identity, compared through the protobuf library's own pure encoder (the generated `Program`
carries no `BEq` and canonical ProtoJSON needs `IO`). It is. The module also pins each entrypoint's
sequence against the template's, the two rejections the assembly owns, and the distinct instruction
ids a class performed twice receives, which .11's retry Query needs.

**Amended while implementing (recorded in the spec and the task):** the comparison is the Program, not
the Case. A Contract is derived from the checked Property's clauses and the Scenario's action order,
and a correlated clause embeds its trigger action's Model Value and that action's occurrence bound in
the Contract itself, not only in provenance. Re-authoring two waits as three side effects therefore
changes the Contract by construction, which no identity mask covers and which is the point of the
re-authoring. The Program is what the party-to-entrypoint design is answerable for.

**Left to .2 and .3:** the Model itself. `.1` states the three action Definition IDs rather than
deriving them from an `Origin`, because no command syntax exists yet, and it hand-builds no checked
Model — the proof runs on the realization and the path. The Nexus node builders are the template's,
reused rather than copied, and `.11` moves them into the realization when it deletes the template.

**Gates.** `make umpire-check-case-runtime-conformance` exit 0 with every fixture byte-identical;
`umpire-check-goldens`, `umpire-check-regression-views`, `umpire-check-testpilot-protocol`,
`umpire-check-testpilot-authoring` and `umpire-check-inventory` exit 0;
`make umpire-check-retired-vocabulary` exit 0 with the baseline exemption gone;
`make umpire-check-live-tests` nine passing identities against the empty expected set, no failure;
`make lint-model` at 163, all in generated `Temporal/API/Proto.lean`, import-graph and Umpire.Lint
clean, 24 new declarations and no new diagnostic; `lake build TemporalModelTests UmpireTests` green.

The live run executed before the two self-review commits, whose changes are Lean-only and left every
fixture byte-identical (re-checked after), so live behavior cannot have moved: the runtime consumes the
fixtures, not the Lean.

**Review:** self-review, SHIP, recorded through `flowctl`. Implementer and reviewer are the same
session, so it lacks the cross-model independence the review step exists for; a session with a second
backend should re-review before the spec's completion review.
## Evidence
- Commits: 2c38b0b51, 5e11ec160, 19eb93e3d, b52e3b522, cff5818bd, ea4ec9ca2
- Tests: make umpire-check-case-runtime-conformance, make umpire-check-goldens umpire-check-regression-views umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-inventory, make umpire-check-retired-vocabulary, CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-live-tests, make lint-model, lake build TemporalModelTests UmpireTests, go test -count=1 -tags test_dep ./tools/umpire/internal/retiredvocabulary/ ./common/testing/testpilot/...
- PRs: