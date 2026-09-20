# Umpire 4 delivery order

Build on the completed Testpilot and model authoring cutovers. Flow owns task status;
this document records delivery order. Architecture and terminology live in the
[Umpire 4 specification](UMPIRE4_SPEC.md).

## Current work

### 1. Finish the Model command surface — fn-83

[fn-83 — Author a live Case from a Model file](../.flow/specs/fn-83-author-a-live-case-from-a-model-file.md),
from the 2026-09-09 gap analysis of [UMPIRE4_VISION](UMPIRE4_VISION.md). Its goal was that a
Model file plus one `case` block is a running, checked-in, deterministic live test. On 2026-09-10
the user replaced the `case` direction (see fn-85 below), so fn-83 now finishes the command surface
and hands the rest to fn-85.

**Landed.** A generic `Umpire.Case.Producer` (.1); the `nexusOperation` and `workflow` realization
templates (.2); the `case` command, the Lean Case registry and `umpire-case --list/--render` (.3);
the generator reading that registry with a live test named by its fixture alone, which landed
before .4 was blocked; a provisioning package and the `umpire-run` CLI (.7); every generated
Testpilot JSON fixture stored indented for review (.9); the five Model commands moved into
`Umpire.Command`, with AUT-07a drafted (.10); Query-declared Known Gaps replacing the hard-coded
ones (.11); the `enum` command and a generated Setup (.12).

**Also landed (2026-09-10).** .13 made Facts optional so a Model stops restating its states; .14
reports authoring mistakes in `property`, `scenario` and `query` at their line while the Model file
compiles; .15 respelled the commands under one reading rule (column-0 declaration, indented
`word:` key, derived labels). .14 and .15 skip the `case` command. **Eleven of seventeen tasks are
done and nothing is open** — the six below are blocked, so fn-83 is paused on fn-85, not stalled.

**Blocked 2026-09-10, superseded by fn-85.** .4's remainder, .5 (fault lines against template
hooks), .6 (sync Nexus translation), .8 (tutorial), .16 (fixture name from the `case` name) and .17
(template arguments). fn-85's final task closes them and names where each concern went, so fn-83
itself closes only after fn-85. Its former spec-level dependents were re-anchored on 2026-09-10:
fn-33, fn-29, fn-70 and fn-79 now depend on fn-85, whose sets and Realization replace the `case`
block they consumed; fn-84 and fn-46 dropped the dependency. Their tasks that touched fn-83's open
work carried a dated note to start after fn-83 .15, because Flow cannot record a cross-spec task
dependency; .15 is done, so those notes are satisfied.

AUT-07a (.10) and the AUT-09 amendment .8 would have drafted are the GOV-02 items fn-83 leaves; the
amendment moves to fn-85's rule drafts.

**Closed 2026-09-20 with fn-85 .13.** The six blocked tasks are closed as superseded, each record
naming where its concern went: .4 to fn-85 .7 (the generator reads `umpire-case --list`, the
artifact test is table-driven, `bindCase`/`runCase` derive the binding, the live helpers are
shared); .5's fault grammar to fn-85's actions (a fault is an action of a declared party, realized
by an `ActionBinding` that injects it) and the outage Model with its outage-order rule to fn-86 R4;
.6 to fn-85 .10's Query 1 with its `COVERAGE.md`; .8 to fn-85 .13 (`model/AUTHORING.md`, the drift
test, the concept entries and rule drafts); .16 to fn-85 .7's derived identities; .17 to the
realization's keyed bindings and production-time evidence rejections (fn-85 .10, .11). The AUT-09
amendment is drafted in fn-85 .13. The spec's own `spec close`, like fn-85's and fn-87's, needs
runtime task state a fresh clone does not carry, so the records carry the closure and Flow's status
follows in a clone that has it.

### 2. Deepen five shallow module clusters in Umpire and Testpilot — fn-84

[fn-84 — Deepen five shallow module clusters in Umpire and Testpilot](../.flow/specs/fn-84-deepen-five-shallow-module-clusters-in.md),
from the 2026-09-09 architecture review of `model/Umpire`, `model/Testpilot`, `common/testing/testpilot`,
`tests/testcore/testpilot` and `tools/umpire`. Five independent scans, one per area, each nominated
one deepening: a shallow module cluster where one decision is spelled out in two to six places and
kept in agreement by hand becomes one deep module with a small interface that callers and tests both
cross. The spec carries exactly those five, one task each, in the order the scans recommend:

| Task | Deepening | Pin |
| ---- | --------- | --- |
| .1 | one `outage` module owned by the worker registry; `Validate` and `Open` share one `OutagePlan` | fault suite and live worker-outage tests, unchanged Verdicts |
| .2 | one Driver-contract leaf package replacing the 14-declaration facade mirror and its four adapters | conformance and facade tests compile unedited |
| .3 | `Umpire.Search.admit` owning the Property, Scenario, Query, view and search chain | `PlanResult` bytes, goldens and fingerprints unchanged |
| .4 | the offline Evidence structure module returns a verdict per audience instead of findings two callers re-judge | mutation suite diagnostics byte-identical |
| .5 | `Umpire.Case.Projection.lower` derives the Contract rule from the checked field Property | Case fixtures byte-identical under ART-11 |

No behavior changes; every task records an equivalence pin before it moves code and closes with
`make umpire-check-regression`. The plan review is SHIP after one fix round.

**Done and closed in Flow 2026-09-12.** All five tasks landed serially with byte-identical pins: the worker `outage`
module with one `OutagePlan` (live outage Verdicts and Run Events unchanged); the `contract` leaf
package with facade aliases (conformance and facade tests unedited; MOD-14 restatement drafted,
pending GOV-02); `Umpire.Search.admit` and `AdmittedQuery` (query ids, fingerprints and `PlanResult`
bytes unchanged); per-audience Evidence structure verdicts over one `Shared` reachability walker
(36,000 recorded diagnostics unchanged); and `Umpire.Case.Projection.lower` deriving both typed
Producers' monitor rules, the capture rule included (Case fixtures unchanged). Every task closed with
`make umpire-check-regression` exit 0 and nine live identities, `make lint-model` at 163 and `make
lint-code` at 161; each impl-review and the completion review were SHIP. Deferred: the typed unary and
typed Nexus Producers stay off `admit` because they never search; the Race unions shrink to their
Model-stage arms plus an `AdmissionDiagnostic` carrier; the three Operations modules still build their
view by hand, a candidate for fn-85; and a failed outage stop or resume still records no Driver
invariant diagnostic, a pre-existing gap now under `CONSIDER(umpire)` in the worker Session.

Boundaries: no renames beyond what the new modules need, no change to the evaluation budget
(`CONSIDER(umpire)` on the cubic reservation stays separate), to delivery routing, to the
generators' publication tooling, to Property evaluation combinators, or to `Umpire.Json` sealing.
EVD-20 is approved (2026-09-10, GOV-02); the MOD-14 restatement .2 drafts stays pending.
Candidates the scans surfaced and the spec declined are listed in its Decision Context.

### 3. Tighten the Testpilot protocol — fn-87

[fn-87 — Tighten the Testpilot protocol: glossary names, one expression language, structure by concept](../.flow/specs/fn-87-tighten-the-testpilot-protocol-glossary.md),
from the 2026-09-11 review of the Testpilot protocol (855 lines in seven files). The review found
thirty issues: three expression languages for the same operators (fourteen duplicated Program and
Contract messages plus a correlated predicate language), names that break SEM-19 (a status enum for
Run disposition, correlated `clauses` of type `CorrelatedRule`, "projection" and "capability" each
naming two concepts), per-kind Run Event fields for faults, duplicate encodings (opaque handles,
`natural`, capture types, three binding shapes, a nested `version`), an enum no wire message
references, a layout that spreads the correlated capability over three files, and no written
checklist for extending the protocol.

The spec renames to glossary words, restructures into one file per concept with every message
documented, replaces the three expression languages with one `Expression` checked per context at
preparation, gives Run Events a payload oneof, removes the duplicates, and adds an extension
checklist.

On 2026-09-11, with breaking changes allowed, it also took on the defaults and readability the
review of `typed-nexus-case.json` called for. That fixture is 2,025 lines, and 77% of its bytes are
21 parameterized model values of about 11,000 characters each; every instruction guard in the six
checked-in Cases is "every dependency succeeded"; each Case writes 24 to 83 limit fields. So
instructions run in entrypoint order unless `after:` or a guard says otherwise; the environment
bindings, activation reservations and outcome fields are derived; resource ceilings move to the
Profile, keeping only meaningful bounds in the Case (an SEM-16 amendment); provenance becomes
structured rows, with Case-local names and short model values mapped to Definition IDs and
fingerprints there; fixtures print fields in declaration order with string field paths and named
enums; a comparison with an absent operand is false, so presence checks disappear; and Run-only
messages leave the Case's import closure.

Verdicts do not change, except where the absent-operand rule is checked against every conformance
class and live test: R1 and R2 land first as mechanical changes, proven by an equivalence test that
maps every pre-migration fixture to the new protocol under a declared mapping. The wire has no
compatibility promise (`buf` breaking ignores the package), so there is no `v2` and no shim. New
capabilities stay with their owners: typed worker instructions and per-Case observation declarations
with fn-85 R10, cancel with fn-79, correlated transitions over structured machine state with fn-85.

**All 17 tasks done and the completion review SHIP, 2026-09-13.** The plan review was SHIP on its
second round, and every task landed serially in this checkout with its own review SHIP:

- **Landed:** the equivalence harness over a frozen descriptor snapshot (59 declared mapping steps);
  glossary renames; one file per concept with documented messages and a closure test; one
  `Expression`, which correlated conditions and evidence-lift guards also use; a Run Event payload
  oneof; one opaque-handle encoding, unsigned integers instead of `natural`, and the wire
  `EntrypointKind` removed in favour of a Go classification in the Driver-contract leaf; resource
  ceilings in the Profile; entrypoint order by default with `after`;
  derived bindings, reservations and outcome fields; structured provenance with Case-local names and
  short model values; declaration-order ProtoJSON with string field paths and named enums; the
  absent-operand rule; and the extension checklist in `common/testing/testpilot/README.md`.
- **Gates at .17:** `make umpire-check-regression` exit 0 with nine live identities, `make lint-code`
  at 161 after `go clean -cache`, `make lint-model` at 163, and the protocol, authoring, conformance
  and retired-vocabulary checks passing. Every conformance `expected.json` stayed byte-identical and
  no live Verdict moved. `typed-nexus-case.json` shrank from 315,914 to 43,215 bytes.
- **Decisions taken during delivery**, recorded in the spec's Planning decisions:
  - R9 amended: `after:` names dependencies within one entrypoint only. Preparation already rejected
    cross-entrypoint dependencies, no Case used one, and entrypoints coordinate through Temporal.
  - Every comparison with an absent operand is false, `NOT_EQUAL` included, so a Producer keeps the
    presence check beside a negated comparison.
  - `Reference` gained a correlated step and a projected-value arm.
  - Presence stays single-arm oneofs, because the pinned `protoc-gen-go-helpers` rejects proto3
    `optional`.
  - The wire scalar kinds stay, because admission compares a slot's kind to its field's.
  - An instruction may not declare more attempts than the Profile's ceiling.
  - `cmd/tools/protogen` rewrites cross-file enum references.
- **Rule drafts pending GOV-02** in `UMPIRE4_SPEC.md`: SEM-16, ART-09 and ART-13 restatements and the
  Case, Provenance and Profile glossary entries.
- **Completion review, 2026-09-13:** SHIP, recorded in
  [`.flow/artifacts/fn-87-tighten-the-testpilot-protocol-glossary/completion-review.md`](../.flow/artifacts/fn-87-tighten-the-testpilot-protocol-glossary/completion-review.md).
  Every requirement R1 to R15 is met, with R5's presence deviation and R9's amendment recorded in the
  spec. The review is recorded through `flowctl`, so the spec carries the receipt and
  `completion_review_status: ship`, with the backend recorded as `claude` because the reviewer was the
  delivering session rather than a separate model. **The spec is not closed**: `flowctl spec close`
  refuses because runtime task state lives in the clone's `.git` common-dir and a fresh cloud clone has
  none, so all 17 tasks read `todo` from the committed snapshot — the shape every historical spec in
  this store has. Closing is one command in a clone that carries the runtime state.
- **The four deferred follow-ups landed with the review.** `ir` calls a path read a read (`ReadPath`,
  `readPath`, `readPayloadPath`, `readOperandPath`, `pathReadWork`, and a diagnostic that says "path
  read"); hand-written Go no longer calls an Opcode a capability (`InstructionOpcode`, `opcode`
  locals, opcode diagnostics, and the retired facade name held by the vocabulary gate); the frozen
  baseline tree is a `.gitignore` negation beside the other testpilot ones rather than a `git add -f`;
  and `cmd/tools/protogen` sorts its rewrite offsets explicitly and skips a selector's `Sel`. fn-85
  .1 then removes that tree and its negation together, since the oracle's subject is discharged; the
  negation is what the tree needs for as long as it is tracked, not churn.
- **Left open:**
  - regenerating the spec's local HTML lens (no spec in the repository has one; markdown is the
    record);
  - the Driver contract's hand-written Go still says capability in the effect-handle sense
    (`CapabilityEffect`, `CapabilityBridge` and the server Session's slots and claims). The rename
    table retires those names on the wire and says the runtime concepts say effect handle, but fn-87
    carved the Driver seam out; renaming it touches the server, worker and composite Sessions, every
    test Session, the conformance corpus and Umpire's lowering, so it wants its own change.
- **Intermittent live failures:** three tests also fail at base commits in about one or two runs in
  ten. They are the umpire-run namespace-delete timeout, the typed Nexus evidence-ordering mismatch,
  and an async-Nexus Run ending INCONCLUSIVE.

**fn-85 depends on it** (recorded in Flow), and fn-22 and fn-26 now depend on it too, so fn-85's
instructions and fn-86's migrated Cases are authored on the final shapes.

### 4. Model side effects as typed actions and run query sets — fn-85

[fn-85 — Model side effects as typed actions and run query sets](../.flow/specs/fn-85-model-side-effects-as-typed-actions-and.md),
from the 2026-09-10 design session recorded in
[the Nexus design specimen](../model/Temporal/Feature/Nexus/DESIGN.md). The `case` block kept every
side effect (the RPC, its request fields, the handler's reply, timeouts, cancel) in a hand-written
Program template, so a Model could not tell a retryable handler error from a non-retryable one and a
Property could not read the fields that make the difference. The design surveyed the server's Nexus
operation behavior and all 16 Nexus functional test files, which the Model must eventually express.

The spec makes three things true. Side effects are part of the Model: entities with identity,
actions a party performs with typed inputs grouped into classes and an example per class, machines
that keep each entity's state and rows, and observations that confirm each row. A Temporal-owned
realization in `Temporal.Case` binds them to RPCs, Testpilot instructions, history events and
dynamic config. Queries are grouped into sets by purpose: a set binds each party to `driven` or
`observed`, and a functional set compiles to one Case per Query. And the Nexus caller-side operation
runs this way end to end: a product machine and a protocol machine that refines it, with a
functional set of seven Queries translated from the Nexus functional tests, each run under both the
HSM and CHASM implementations.

The early proof point rebuilds today's async-Nexus Case from the new abstractions and must match
its fixture with identities masked before any Testpilot protocol change. Worker instructions then
carry the Temporal API messages the actions' schemas name (the Nexus schedule command's attributes
with its three timeouts, a `StartOperationResponse` or `HandlerError` reply, a completion payload or
failure) instead of a Testpilot field per server option (the old `StartNexusOperation`,
`RespondNexus` and `NexusResponseKind` stay until fn-86 .3 migrates the last Producer that emits
them), and a Case declares each observation once
for both its Program and its Contract, which gives the retry Query's attempt count its read source.
Canary and exploratory sets are admitted with their
coverage targets enumerated; running them stays in fn-70, fn-29 and fn-33. Whole-Program templates
and the `case` command are removed.

**Done 2026-09-20: all sixteen tasks landed and the receipts below record each.** The spec's
`spec close` waits on a clone with runtime task state, as fn-87's does. **Task .1 is done,
2026-09-14, and the early proof point holds.** The Program the
Producer assembles from the path `[schedule, handlerReply, complete]` is byte-identical to the one the
`nexusOperation` template writes by hand, so the party-to-entrypoint design carries what the template
stated; the spec's stop condition did not fire. What landed: the protocol-migration oracle retired;
`Umpire.Command.Records` for entities, actions, observations, timers, setup parameters, evidence lines
and the machine declaration; a `ProgramPlan` and per-action-class bindings replacing
`Realization.program`, with the Producer placing instructions from the path; and
`Temporal.Case.Realization.asyncNexus` with the proof in a proof-point test module (`.11` retired it with the templates). Its review
is a self-review, so a session with a second backend should re-review before the completion review.

**The proof point's acceptance was amended during delivery:** the comparison is the Program, not the
Case. A correlated clause embeds its trigger action's Model Value and that action's occurrence bound in
the Contract itself, not only in provenance, so re-authoring two waits as three side effects changes
the Contract by construction -- which is the point of the re-authoring and which no identity mask
covers. `.2` and `.3` settle the Contract's shape once the Model is authored through the commands.

**Task .3 is done, 2026-09-14, and the authoring form holds.** A step function over a structure of
finite fields enumerates to exactly the rows an author would have written -- same rows, same order,
same keys -- so the two forms fill the same table and fingerprint the same value, and the row grammar
stays unbuilt. `Umpire.Command.Finite` carries the domain, its deriving handler refuses a non-finite
field by naming it, and `enumerateBounded` refuses a domain past `elaborationBound` with both factors
rather than truncating. Elaborating the prototype measured inside the noise of its own import.

**Task .2 is done, 2026-09-15.** `entity`, `action` and `observation` elaborate into the task .1
records, and `DESIGN.md` section 3's entities, input domains, actions and derived observation are the
specimen they are checked against, machines and the fn-79 cancel actions excepted. `schema:` stores
the message names and asks a platform whether they resolve; `Temporal.Case.Schema` answers from the
descriptor closures of the RPCs those messages travel on, and a test says the stored entry is shorter
than the descriptor of the message it names.

Three things were decided during delivery.

- **The three command words are not reserved tokens.** `entity`, `action` and `observation` are also
  field names in the records the commands build (`EntityReference.entity`, `FiniteTransitionRow.action`),
  so reserving them the way `model` and `property` are reserved would make those fields unwritable.
  A plain `&"entity"` does not work either: Lean indexes a non-reserved symbol under its own token
  while a command beginning with a bare identifier dispatches under `ident`, so the parser was only
  reachable behind a doc comment. `nonReservedSymbolNoAntiquot word (includeIdent := true)` indexes it
  under both, which is what makes a column-0 `entity` a command without taking the word away from the
  tree.
- **The schema check decides whether a name resolves, and not whether a class member is in it.** The
  members the design's own examples name -- `BadRequest`, `Internal` -- are values of
  `temporal.api.nexus.v1.HandlerError.error_type`, which the generated schema types as a `string`, so
  the descriptor carries nothing to check them against. R2's "a class member or example outside the
  schema" moved to `.8`, which gives an action's payload the typed fields a member can be checked
  against, and is recorded on that task.
- **Two R1 and R2 rejections have no syntax to fire on in `.2`.** An instance bound of zero was already
  `.4`'s, where instance bounds are declared. An evidence name that is neither catalogued nor declared
  moved to `.14`, which introduces `evidence:`; it is recorded there.
- **A class is a member of an input domain, not a constructor.** `handlerError (retryable : Bool)` is
  one constructor and two classes, which is the granularity `DESIGN.md` section 2.2 writes an example
  at and the only reading under which the design's own two example lines are both legal. It follows
  that every class has exactly one member and nothing in a Model counts the realized values a class
  covers, so R8's abstraction claim is triggered by the presence of an `examples:` line rather than by
  a member count. That amendment is recorded on `.7`.

**Seven review rounds; six NEEDS_WORK, all fixed, round seven SHIP.** Round one found that a declaration's rules were
being applied against the whole import closure rather than the file -- the second feature Model to
declare `entity workflow` would have been rejected by the first -- and that a reference rebuilt the
referenced declaration's Definition ID from the *referring* file's `Origin`, so an entity, enum or
observation named across files pointed at a name in the wrong family. Round two found the second fix
half done: an entity's id was stored, but an enum's was still rebuilt at the reference, reading the
referring file's `model_conventions` visibility. A declaration now records its own id where it is
written -- `Registry.EntityEntry.id` and the new `Registry.DomainEntry` -- and every reference emits
that. `Temporal.Feature.Nexus.Tests.SecondModel` is the second Model file both defects needed to
surface.

Round three found the walk could still exhaust the stack: a failed `deriving` is logged rather than
thrown, so a domain whose members do not enumerate was recorded anyway, and the bound is a width
rather than a depth. A domain is recorded only once its `Finite` instance exists. Round four found
that flattening a class to a string to compare it deleted the parentheses separating a constructor's
name from its first field's, so an example of `a (bc := false)` was stored, silently, against
`ab (c := false)`; a class is now a tree compared structurally. Round five found a written
constructor's qualification read and discarded, which `DESIGN.md` section 3 reaches in the feature's
own vocabulary -- `Reply` and `CancelReply` both declare `handlerError (retryable : Bool)`. Round six
found `refer:` had no uniqueness rule on its field names, and three sites still asserting the
superseded reading, `DESIGN.md` among them.

Gates on the closing tree: `lake build` green, `make lint-model` at 163, `make
umpire-check-regression` exit 0 with nine live identities.

Its review is a self-review: no second backend is reachable in a cloud session, so `codex exec`,
`cursor-agent` and `grok` are not installed and all fall back to the session model. `.1`, `.2` and
`.3` each owe a cross-model re-review before the completion review.

**Tasks .14 and .16 are done, 2026-09-18 and 2026-09-19.** `machine` is a command over step
functions, `model` is retired, every Model in the tree is a machine, and the design's protocol
machine elaborates (192 states over 23 action classes, 1152 rows, about forty seconds, once the
canonical-table law stopped comparing a table against itself quadratically). The receipts on the
two tasks carry the decisions.

**Task .15 is done, 2026-09-19.** A `property` names a machine and an ordinary Lean predicate --
`Step → Bool` under `when:` for a same-step claim, `Step → Step → Bool` for a transition claim over
the step before and the step after -- and the command enumerates it over the machine's table into
the clause records a Property has always carried. The reading is the one the keyed `require:` block
spelled out: the predicate fixes a state, an outcome or a fact when every accepted step carries it
and changing it is rejected, probed on the predicate itself rather than read off the table's
coincidences; the fixed values must carry the predicate exactly, and a disjunction across fields is
refused with the step the clauses cannot tell apart. Every migrated Property carries the fingerprint
its keyed block had, pinned. The keyed form is rejected at its key naming `holds:`, and `DESIGN.md`
section 3 carries the amendment. Its review is a self-review.

**Task .4 is done, 2026-09-19.** Its first half (2026-09-19, earlier session) carried a machine's
state fields on the Model and on the wire and made both evaluators read them. Its second half makes
the fields readable on the Model side -- the evaluator reads a state and every field it holds, and a
machine's capability means its fields, which moved every Property-over-a-machine fingerprint once --
adds a structured two-operation Target to the conformance corpus whose rule reads the `attempts`
field and whose two operations are tracked apart by both evaluators, and lets a Scenario run over
`instances:` of one entity: the Search walks the product of that many copies of the machine under
the machine's own law, one instance's Property is read on the acting slot, and the Producer reads
the first instance back with every instance's actions as the Program's path. Rejections are pinned
where they are written. Its review is a self-review.

**Task .6 is done, 2026-09-19.** A machine declares `refines:` and `map:` -- the product machine and
a Lean function from its state to the product's -- and the command walks every row through the map:
a row whose mapped states are a product step is that step, a row whose mapped states are equal is a
stutter, and any other row rejects at the `map:` line naming the row, both readings and what the
product lacks. Outcomes and facts read by name, a fact's constructor covering its members; a fact
the product does not name is hidden, an outcome it does not name rejects, and a product step may
record less than the protocol step it carries, never more. The derived step mapping is read back as
data and pinned; the witness is `Umpire.ImplementationLink.Refinement`'s stuttering forward
simulation, decided by the kernel over the two tables, with `traceForward` carrying every admitted
protocol trace to an admitted product trace. A Property on the product machine is read on the
protocol machine's paths through a state field named after the product machine, and a Query over a
protocol Scenario may `find:` or `verify:` it. The product machine gained a `timeout` timer, the
Scenario grammar took classed actions and a phase for `starts:`, a setup constructor is named by
the start phase rather than the punctuated key, and the `model`-era refusal of a step out of an end
state is retired because the design writes such steps. Its review is a self-review.

**Task .5 is done, 2026-09-19.** A machine's `setup:` parameters travel with the declared Model by
name and definition, the realization binds each to a dynamic-config key of the generated catalog,
and one it leaves unbound is an `input` Known Gap of the Case naming the parameter. The Profile
records the configuration the environment ran under, in the catalog's spelling and as part of its
binding fingerprint, so the same Case bytes run under two switch values under two Profiles. The
Nexus realization declares the `implementation` switch (`hsm`, `chasm`) with the three settings the
upstream suites set, resolved by name for `.7`'s `repeat:`; the live harness runs the async-Nexus
Case once per value under a dedicated environment constructed with the value's settings and fails
on a divergence naming the switch, both values and both Verdicts, with the divergence check pinned
by a unit test. `atConcurrencyLimit` is not bound: the limit exists, one key per implementation,
but binding it needs a setup-varying table, a value per key and a key per switch value, which the
research spike of 2026-09-19 (`UMPIRE4_RESEARCH_NEXUS_MODEL.md`) recommends `.10` resolve by
dropping the parameter. Its review is a self-review.

**Task .7 is done, 2026-09-19.** The `set` command groups Queries by purpose and binds every party
except `system`, rejecting in place an unbound party, a bound `system`, a stray party, an
`observed` party's action on a functional path, a `verify` Query, `repeat:` outside a functional set
or naming a switch no realization registered, an exploratory set without a goal or budget, and the
wrong keys for a purpose. The Temporal `case` block over a functional set realizes each Query under
`temporal.case.<set>.<query>` and `<set>-<query>-case.json`, registered like any Case, so
`umpire-case --list` and the list-driven generator carry it; the success slice's
`nexusSuccessTests` produces `nexusSuccessTests-completion-case.json`. A Case records an abstraction
claim row -- action, field, class, example -- for every class with an `examples:` line its Program
performs, as the protocol's ninth provenance field. Its review is a self-review.

**Task .8 is done, 2026-09-19.** The protocol carries the Temporal API messages the design named:
a workflow command carrying a `Command`, a handler reply carrying a `StartOperationResponse` or a
`HandlerError`, and a completion carrying a `Payload` or a `Failure`, imported from `proto/api.binpb`
and compiled once into Lean by `Testpilot/Carried.lean` so the protocol module's own rebuild keeps
its cost. Preparation admits each message against a Driver-reach table naming the fields the Driver
realizes, so an unsettable field, an invalid or over-ceiling duration, a reply the activation does
not admit and a command type the Profile does not admit reject in the existing categories at the
field's path, pinned by unit tests and four conformance corpus variants; the Profile admits commands
per command type. The worker Driver maps each message to the SDK call that produces it, with a
Driver test per carried message. The Nexus realization binds the schedule, every handler-reply class
and both completion classes, the asynchronous `case` form produces through it, and the two Query 2
fixtures regenerate on the typed instructions, differing in exactly the three bound instructions.
The checklist missed five places, now listed. Its review is a self-review.

**Task .9 is done, 2026-09-19.** A Program declares each kind of correlated evidence once
(`Program.evidence`): its source as a history event arm, a Run Event kind or a unary read, its scope,
key path and fields. A history read's lift rule names the declaration and spells nothing else, a
`ReadEvidence` controller instruction polls a read declaration through the new `Session.PollRPC`
until an element satisfies its condition and lifts what the condition selects, a Run Event kind is
lifted by the scheduler as the event is recorded, and the correlated Contract's projection rules
resolve against the declarations; the Case compiler localizes the new names with the rest. The
undeclared reference and the duplicate source-and-key rejections land in `unknown` and `malformed`,
pinned by two corpus variants beside three accepted ones, one per source. `pendingAttempts` is the
read catalog's one binding (`Temporal.Case.ReadKind`: `DescribeWorkflowExecution`,
`pending_nexus_operations`, key `scheduled_event_id`, field `attempts`), the third answer an
`evidence:` line resolves against, and the Nexus template carries it beside its two history kinds.
The two Query 2 fixtures regenerate on by-name rules with their two declarations. The `case` block's
evidence lines stay for `.11`. Its review is a self-review.

**Task .10 is done, 2026-09-19.** The Caller Model (`Temporal.Feature.Nexus.Caller`, family
`temporal.nexus.caller`) promotes the product and protocol machines out of the test specimens with
six predicate Properties, four Scenarios, Queries 1 to 4 and the functional set `nexusCallerTests`
over the HSM and CHASM switch; `nexusProtocol` takes no `setup:` parameter, because
`atConcurrencyLimit` has no dynamic-config key. The set is realized by a `realized by asyncNexus`
arm that writes no evidence lines: the mapping derives from the machine's `evidence:` catalog per
Query path, the realization's bindings are keyed by the classed member a Scenario action resolves
to, and `whenOnPath` places `await-completion-authority` on the completion paths only. A Contract
now carries only the projection rows its rules can reach (the protocol machine's whole table made a
2.6 MB fixture), and an instructed handler error completes the handler activation instead of
stopping the Run. Four fixtures regenerate (`nexusCallerTests-{syncCompletion,asyncCompletion,
asyncFailure,handlerError}-case.json`); `async-nexus-case.json` is gone, and Query 2's fixture
differs from it by the controller's `await-close` read, the finish literal, one correlated rule for
two, three declarations for two, five transitions for two and no known gaps. `COVERAGE.md` maps the
four upstream tests; DESIGN.md section 3 is the cancel-free specimen with a `.10` amendment. `make
umpire-check-regression` exit 0 with **20 passing live identities** (eleven before). `lint-model`
adds two unused-binder warnings from the `enum` command's generated binders, the pattern
`Tests/Commands.lean` carries. Its review is a self-review.

**Task .11 is done, 2026-09-19.** The Caller Model adds Queries 5 to 7: a retryable handler error
then sync success after one backoff, with `pendingAttempts` polled until `attempt == 1`; a
schedule-to-start timeout after `workerStop` stops the handler's worker on its own task queue; an
async reply then a start-to-close timeout. Timers are `TimerBinding`s on the Realization (2000 ms
each) passed to the schedule command and observed through the timed-out event, so no
wait-for-duration instruction was needed. The Producer folds a silent step into the next confirmed
rule and carries it as a capability Known Gap (`backoff.unobserved`, `workerStop.unobserved`);
every controller path first reads the scheduled event (`await-scheduled`), the runtime chains one
operation's evidence across sources by a parent, admits a canceled reservation of an entrypoint
that performs nothing, and the worker keeps a retryable handler activation open for the retried
start. The whole-Program templates, the `fixture`-named `case` form, the template grammar of the `as` clause,
`ProofPoint` and the `Success` set are gone and retired; `case … realizes <set> as <Realization>`
is the one Case-producing command. Seven fixtures, `COVERAGE.md` mapping all seven upstream
tests, timer stability three runs under both values (4.2 to 4.8 s per Query), `make
umpire-check-regression` exit 0 with **29 passing live identities** (20 before). `lint-model` at
the `.10` baseline. Its review is a self-review.

**Task .12 is done, 2026-09-19.** The `case … realizes <set>` block admits a canary set: each
Query's Case is produced under the realization, registered nowhere, and read for a white-box Known
Gap (`capability`, `interpretation`), which rejects naming the Query and the gap;
`nexusCallerCanary` over Queries 1 and 2 with `handler: observed` admits, and a canary over the
retry Query rejects on `backoff.unobserved`, pinned by `#guard_msgs`. An exploratory set names
`machine:`, and its `budget:` must be a `limits` declaration; `Umpire.Command.Coverage` enumerates
its targets from the declared Model (rows within the budget's steps of a start, the results they
reach, the claims their actions make, cut at the search count) and `nexusCallerExploration` over
`nexusProtocol` under `four` lists 885 of 1152 rows, two results and two class members in
`Caller/Fixtures/CallerExploratoryCoverage.json`, rendered byte-identically twice and checked by
`umpire-check-goldens`. `make umpire-check-regression` exit 0 with **29 passing live
identities** (29 before). Its review is a self-review.

**Task .13 is done, 2026-09-20, and fn-85 is complete.** `model/AUTHORING.md` walks the Caller
Model from an empty file to a green live test in thirteen steps, quoting every marked region of
`Caller/Model.lean` (a `header` marker joined the twelve .10 placed); `tools/umpire/authoring`
checks each quoted block against its region byte for byte and fails on a block naming a missing
marker, a marker the Model carries twice, a drifted region, an unquoted region and an unfenced or
repeated block, each pinned by a planted test; the file is in the vocabulary gate's required
files. `UMPIRE4_SPEC.md` gains Entity, Party, Refinement, Set, Realization and Abstraction Claim
and amends Action, Observation and Machine, with the AUT-07a (`set`, `register_switch`, the
platform's Case-producing block), MOD-02 (the realization in `Temporal.Case`) and AUT-09 (what
the commands derive) amendments drafted under GOV-02; the MOD-15 name gate is green. DESIGN.md
points at the spec and the Model and its needs table records what each need received; the two
architecture documents, the model README, the testcore README and `tools/umpire/CONTEXT.md`
(eight glossary entries with their `_Avoid_` lists) follow. fn-83's six blocked tasks are closed
as superseded above. Gates: `make umpire-check-regression` exit 0 with **29 passing
live identities**; `lint-model` the `.11` baseline: two errors in generated `Temporal/API/Proto.lean` and 41 warnings (generated binders, deprecations, the two `enum` binders in `Caller/Model.lean`), none new; `lint-code` 0 issues over the changed packages (`GOLANGCI_LINT_BASE_REV=39a61b4 make lint-code-fast`); the full `make lint-code` is not measurable in a shallow clone with no `main` merge base, as the 2026-09-13 row records. Its review is a
self-review.

**`flowctl ready` reports fn-85 blocked by fn-87**, because fn-87's spec is still `open`: `spec close`
needs runtime task state, which lives in a clone's `.git` and which a fresh cloud clone does not have.
It is bookkeeping, not a dependency -- fn-87's `completion_review_status` is `ship` -- and
`flowctl start` is unaffected. **R3 authoring form decided 2026-09-12:**
machines are Lean step functions over a structure of finite fields, enumerated at elaboration into the
same finite table, per [the FizzBee comparison](UMPIRE_CMP_FIZZBEE.md) section 4.1; the row grammar is
not built.

**Broken into 13 tasks on 2026-09-12. Plan review round 1, 2026-09-13: NEEDS_WORK**, recorded in
[`.flow/artifacts/fn-85-model-side-effects-as-typed-actions-and/plan-review.md`](../.flow/artifacts/fn-85-model-side-effects-as-typed-actions-and/plan-review.md).
Every requirement has a task and the ordering holds; the revisions are one blocker and four smaller
findings, none of them a design change:

- **Blocker: no task owns the protocol-migration oracle.**
  `common/testing/testpilot/internal/protocolmigration` pairs each frozen pre-fn-87 baseline fixture
  with its regenerated counterpart one to one, fails on any difference no declared step explains, on
  an undeclared addition, and on a deleted baseline fixture — and it has no removal list. `.4`, `.8`
  and `.9` change the wire and rewrite every fixture; `.10` deletes `async-nexus-case.json` and adds
  four fixtures; `.11` adds three more. CI runs the oracle
  (`.github/workflows/umpire.yml`), so this is a gate. Decide before `.4`: retire the oracle, whose
  declared subject fn-87's completion review has discharged, or extend it with declared steps and a
  removal list. Retiring is the smaller change.
- `.1`'s "a grep for `nexus` in `Umpire/` is empty" cannot pass — three files already match, two in
  prose and one as fixture-identity test data. Restate it as no Nexus-specific branch in the Producer.
- `.3` is over-sized: split its three planned commits into three tasks, or at least move the
  `property`-predicate change out, so the step-function prototype's fallback stays actionable.
- The `property`-predicate decision has no requirement row, so the coverage table cannot fail on it.
- `.13` closes fn-83's six blocked tasks "through `flowctl`", which needs runtime state a fresh clone
  does not have; the fallback is named.

The split moved dependencies with it: `.4` and `.6` need the `machine` command, so they depend on
`.14` rather than on `.3` alone, and `.10` writes its Properties as predicates, so it depends on `.15`.
`flowctl ready` now offers `.1` and `.3`, and blocks the rest correctly.

Flow records its dependencies on fn-84 and fn-87, and fn-83 and fn-22 now depend on it; the fn-83
tasks it builds on (.13, .14, .15) are done. It closes fn-83's six blocked tasks as superseded.

**Cancellation stays deferred.** The design's cancel Query and its Testpilot instructions overlap
fn-79, which resumes only on an explicit user request, so on 2026-09-10 they moved out of fn-85 into
fn-79's re-planning note.

Boundaries: no composition (update-, query- or activity-backed handlers, several callers), no
reset, no visibility or standalone operations, no metrics or spans as observations, no endpoint
registry, matching or cross-cluster topology, no HTTP transport fault kind, no schema interface, no
change to the hand-written Models (fn-86 retires them).

### 5. Retire hand-written Models: one authoring path through the commands — fn-86

[fn-86 — Retire hand-written Models: one authoring path through the commands](../.flow/specs/fn-86-retire-hand-written-models-one.md),
from the 2026-09-10 decision to align completely on the developer-facing commands. After fn-85 the
tree still carries about 7,000 lines that build Umpire records or Testpilot Cases directly in Lean:
the two typed Nexus examples, the Race, Lifecycle, Operations and Experimental Nexus models, the
worker-outage and get-system-info Cases with no Model, and Umpire's Switch example. AUT-08 still
names that path as an expert alternative.

The spec makes the commands the only authoring path for feature Models and gives every piece of
covered behavior a destination before anything is deleted. An inventory lists each hand-written
module with the Properties, goldens, fixtures, live tests and tools that read it. The commands gain
field relations (a Property comparing action input, action result and observation fields through
their schemas), and the typed examples migrate onto them with their crossed-pairing mutations
intact. The worker-outage and get-system-info Cases become command Models, the Switch example is
re-authored, and the Race and Experimental models are deleted with their behavior written into fn-79
and fn-33. A `lint-model` import rule then keeps feature Models from importing Umpire's authoring
owners except through `Umpire.Command`, and GOV-02 drafts remove AUT-08's expert alternative.

Two decisions frame it: the typed examples migrate rather than being deleted first, because they
carry the only field-level Property coverage; and `Temporal.System.Nexus` stays as the single
exception, because it is the spec's only Feature-to-System Implementation Link (SEM-08, MOD-04).
Testpilot conformance and synthetic Cases stay, since they test the runtime.

**Depends on fn-85** (recorded in Flow). The early proof point expresses the typed unary Property as
a field relation that lowers to the same Contract field reads before any deletion. The order stays
fn-87, fn-85, fn-86: fn-87 does not touch the Race, Lifecycle, Operations or Experimental models, so
deleting them earlier buys nothing, and Lifecycle cannot go before fn-85 because the kept
Implementation Link imports it until fn-86 .4 re-anchors it.

**Broken into 9 tasks on 2026-09-12. Plan review round 1, 2026-09-13: NEEDS_WORK**, recorded in
[`.flow/artifacts/fn-86-retire-hand-written-models-one/plan-review.md`](../.flow/artifacts/fn-86-retire-hand-written-models-one/plan-review.md).
No blocker: inventory first, `.4` before `.5`, and a direct-import lint rule are all right. Five
findings, applied:

- `.1`'s typed-unary Contract baseline is a scaffold and `.2` deletes it, rather than leaving a
  generated Contract no generator writes.
- `.5` left `umpire-inspect`, `umpire-list` and `umpire-explain` an open either/or inside a deletion
  task. Decided: re-point the inspector's registry at the Caller Model's Queries, keep Switch, keep the
  three targets. `umpire-case --list/--render` renders Cases, not Plans, so it replaces neither
  `inspect` nor `explain`, and all three are documented.
- `.7` checks the Definition IDs the command path produces before touching a Switch golden: the
  goldens carry `switch.*` ids and three fingerprints, and the commands derive ids from `Origin`, so
  the compatibility-family pin is part of the task.
- R1 gained "keep with the reason" as a destination, which the two `Temporal.Testpilot` runtime-testing
  modules and the kept Implementation Link need.
- `.1` re-reads the tree as fn-85 left it and corrects the file lists of `.2`, `.3` and `.6` first;
  every fn-86 task was written against the pre-fn-85 tree.

**Task .1 is done, 2026-09-20.** `model/HANDWRITTEN_INVENTORY.md` freezes the tree before anything
moves: the eleven production modules under `Temporal.Feature`, `Temporal.Testpilot` and
`Umpire.Examples` that import an authoring owner directly (the two typed examples, the two
Model-less Cases, `CaseSupport`, `Lifecycle.Model`, the four `Operations` modules and
`Race.Authoring`), the modules the spec's table names beside them (`Observation`, `Experimental`,
the `Success` specimen, `Umpire.Examples.Switch`), what is kept with the reason (the Implementation
Link with its two Feature imports, the realization, `Conformance`) and every tool, target,
compatibility family and document that reads them, each with a destination. `lint-model` reads
the ledger and reports `hand-written module not inventoried` for a production module under those
roots that imports an owner and is missing from it, pinned by a planted module in
`ImportGraphTests`; the typed-unary Contract baseline is a scaffold under `tests/testcore/testpilot/baseline/`
(beside `testdata/`, which the conformance gate keeps generator-owned) that `.2` compares against
and deletes. `.2`, `.3` and `.6` were re-read against the fn-85 tree.
Gates: `lint-model` at the fn-85 baseline: the import-graph and Batteries steps pass with the ledger read, and the `lake lint` step reports the same two generated `Proto.lean` errors and 41 pre-existing warnings, none new; `make umpire-check-regression` exit 0 with 29 passing live
identities. Its review is a self-review.

**Task .2 is done, 2026-09-20.** A field relation is a `relates:` line of `property` beside
`holds:` (`<action>.input.<path>` or `<action>.result.<path>` of the `when:` action, or
`<kind>.<path>` of a recorded event kind the machine's `evidence:` names; `=`, `≠`, `present`),
resolved while the file compiles through a new platform hook
(`Umpire.Command.installFieldResolver`, answered by `Temporal.Case.FieldPath` off the generated
descriptors) into `PropertyFieldPath` steps, presence reads and a scalar type, and lowered by the
Producer through `Umpire.Case.Projection.lower` to the monitor rule `<property>.relation`, with
the literal read from the realization's action binding. The typed unary example is
`Temporal/Feature/Workflow/Start/Model.lean` with one such line, produced through
`Temporal.Case.Realization.workflowStart` as `workflowStartTests-started-case.json`; its rule's
reads equal the `.1` baseline's up to the rule name and the identity-derived literal, pinned by a
Go test that names the first differing read, and the hand-written module, its tests, fixture and
the baseline scaffold are gone. Nine rejections and the crossed-pairing pins (`some false`,
missing evidence `none`) are in the Model's Tests. Gates: `lint-model` at the `.1` baseline;
`make umpire-check-regression` exit 0 with 29 passing live identities. Its review is
a self-review.

**Follow-up after fn-86, not yet a spec:** one Contract monitor declared per entity and instantiated
per instance, replacing the per-instance rule copies Producers emit today (the typed Nexus Case
carries its operation rules twice). It changes how the runtime evaluates rules, so it gets its own
spec once every Case comes from the commands.

### Additional open specs

These remain open in Flow and are outside the first-canary critical path.

| Spec                                                                                              | Dependencies | Next action                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------- | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [fn-46 — Lean module impact index](../.flow/specs/fn-46-export-lean-model-module-impact-index.md) | fn-45 | Refreshed three-task plan is SHIP against current model owners. fn-83 .15 is done, so task .1 is free to start; .2 and .3 start after fn-86 R6, because their root list pins `TemporalExperimentalTests`, which fn-86 deletes; refresh the module rows for `Umpire.Command` (fn-83 .10) and, once planned, fn-85's new modules. Deliver the shared loader, pure dependency/facade/test index, and opt-in export/check commands. |

## Gate baselines

Re-measured 2026-09-20 on a four-core, 16 GB cloud session at the fn-85 closeout:

| Gate | This session |
| ---- | ------------ |
| `make umpire-check-regression` | exit 0 — 614 Lean jobs, the offline checks, **29 passing live identities** (nine at the fn-87 closeout: the Caller Model's seven Queries under two switch values replaced the async-Nexus Case) |
| `make lint-model` | the `.11` baseline: two errors in generated `Temporal/API/Proto.lean` and 41 warnings (generated binders, deprecations, the two `enum` binders in `Caller/Model.lean`), none new |
| `make lint-code` | 0 issues over the changed packages (`GOLANGCI_LINT_BASE_REV=39a61b4 make lint-code-fast`); the full `make lint-code` is not measurable in a shallow clone with no `main` merge base, as the 2026-09-13 row records |

Re-measured 2026-09-13 on a four-core, 16 GB cloud session at the fn-87 closeout, with the
pre-installed toolchain (`/opt/temporal-toolchain`) on `PATH`:

| Gate | This session |
| ---- | ------------ |
| every `umpire-check-regression` constituent, run individually | exit 0 — model build 595 Lean jobs, eight offline checks, **nine passing live identities** and no intermittent failure on the first run |
| `make umpire-check-retired-vocabulary` | exit 0, and about twenty minutes: 353 compiled rules against every line of every scanned tree, the frozen baseline fixtures included. It is the slowest offline gate by an order of magnitude |
| `make umpire-check-regression-views` | exit 0 only after the `go list` stderr fix below |
| `make lint-model` | 163 in generated `Temporal/API/Proto.lean`, `Shared` and `Umpire.Lint` clean — the baseline exactly |
| `make lint-code GOLANGCI_LINT_FIX=false` | **not measurable as cloned**: the gate passes `--new-from-rev=main`, the clone is shallow and carries only `umpire` and the working branch, and `git merge-base HEAD main` has no answer even after fetching `main`, so golangci-lint reports the whole tree (7,576 pre-existing diagnostics) instead of 161. Deepen the clone until `main` shares an ancestor to reproduce the number. What answers the gate's question meanwhile is the same config and build tags over the changed package trees with `--new-from-rev` at the pre-change commit: **0 issues** |

**`flowctl` is not in a cloud session's image.** Install it from GitHub through the plugin
marketplace — `claude plugin marketplace add gmickel/flow-next`, then
`claude plugin install flow-next@flow-next` — which puts `scripts/flowctl` under
`~/.claude/plugins/cache/flow-next/flow-next/<version>/`. The published 5.2.2 carries the same store
`SCHEMA_VERSION` (3) as this repository's `.flow`, so it reads and writes the store without migrating
it. What a fresh clone still cannot do is change task status: runtime state lives in the clone's `.git`
common-dir, so every task reads `todo` from the committed snapshot and `start`, `done` and
`spec close` refuse. Reviews, dependencies, spec status and validation all work.

Three environment notes for a cloud session: `mise` is a passthrough shim, so `/opt/temporal-toolchain/*/bin`
must be on `PATH` before any `make` target that uses `lake` or `protoc`; and warm the Go module cache
(`go mod download`) before the first gate, then discard the `go.sum` hashes that download adds; and
`git fetch origin main` gives the branch a name but not a merge base, because the clone is shallow.

`tools/umpire/regression/ci_workflow_test.go` decoded `go list -deps -test -json` from
`CombinedOutput`, so a cold module cache's download progress landed in the JSON stream and
`TestTestpilotOwnsCaseProtocolAndRuntime` failed with `invalid character 'g' looking for beginning of
value` on a green tree. It now reads stdout alone.

Measured 2026-09-10 after the upstream merge, on a host with disk headroom:

| Gate | Baseline |
| ---- | -------- |
| `make umpire-check-regression` | exit 0 — 571 Lean jobs, eight conformance checks, nine passing live identities |
| `make lint-model` | 163, all in generated `Temporal/API/Proto.lean`; `Umpire.Lint` and `Shared` clean |
| `make lint-code GOLANGCI_LINT_FIX=false` | **161** (exit 2, inherited red) |
| `go vet -tags test_dep ./...` | 15 pre-existing diagnostics |
| `go run ./tools/planindex` | ~49, pre-existing `.plans` registration drift |

**`make lint-code` under-reports when the disk is low.** Earlier runs recorded 128 and were believed
for several specs. golangci-lint aborts with `no space left on device (typecheck)` and still exits
with a count — `Issues before processing: 11800, after processing: 1` in the failing case against
`14507 -> 161` in a healthy one. Run `go clean -cache` before trusting this gate, and treat any
number below 161 as a truncated run rather than an improvement.

Live tests need `CC=/usr/bin/cc` (mise's lean4 clang shadows the toolchain; cgo fails with
`stddef.h not found`) and a physical `TMPDIR` (`TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P)`; the
default macOS path traverses the `/var` symlink). `go vet -tags test_dep` is NOT the gate — the live
suite builds with `-tags 'test_dep integration'`, which compiles strictly more files.

## Downstream delivery

Each spec needs a fresh Testpilot plan review before implementation. Prior reviews of the
retired execution architecture do not approve the rewritten plans. Completed dependencies do not
block replanning or execution.

| Spec                                                                                              | Dependencies                       | Next action                                                                                                                                                                                                                                   |
| ------------------------------------------------------------------------------------------------- | ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [fn-33 — Bounded exploration](../.flow/specs/fn-33-run-serial-bounded-semantic-exploration.md)    | fn-40, fn-64, fn-69, fn-84, and **fn-85** | Re-plan on fn-85: an exploratory set's coverage goal (rows, result classes, members of claimed input classes) is the candidate space, and a divergent class member is the counterexample Promotion keeps. Then review whole-Case candidates, serial coordination through Testpilot, lost iterations, semantic coverage, and bounded 10x behavior. |
| [fn-22 — Replay and reduction](../.flow/specs/fn-22-deterministic-replay-semantic.md)             | fn-5, fn-64, fn-69, **fn-85** and **fn-87** | Resolve **MAJOR_RETHINK** before implementation: separate exact candidate identity from Contract-relative violation equivalence, prove the negative Case before reduction, and retain explicit offline semantic replay and checked promotion. |
| [fn-26 — Qualification receipts](../.flow/specs/fn-26-local-qualification-receipts-and-staged.md) | fn-48, fn-64, fn-69 and **fn-87**  | Review offline Testpilot Case/Profile/Run/Verdict admission, receipt multiplicity, and idempotent publication. Assessment must never create or replay a Run.                                                                                  |
| [fn-29 — Production canary](../.flow/specs/fn-29-bounded-production-canary-execution-and.md)      | **fn-26**; fn-48, fn-64, fn-69, fn-83 .7, and fn-85 | After fn-26 ships, review external policy and credentials, serial Testpilot Runs, leases, lost Runs, reconciliation without redispatch, and publication. It runs an fn-85 canary set and consumes fn-83's provisioning package. |

All runtime work uses `testpilot.Prepare(case, profile)` → `PreparedCase.Run(ctx, driver)` and the
server/worker authority split. New scenarios remain Case data; canary policy, credentials,
leases, recovery, and publication stay outside Testpilot and Umpire.

## Deferred and superseded

**fn-79 — Nexus operation cancellation:** [spec](../.flow/specs/fn-79-deferred-nexus-operation-cancellation.md).
Includes former fn-78.5/.8/.9 cancellation scope and fn-77’s cancellation qualification. Resume only
on an explicit user request; autonomous delivery approval does not override this deferral. Generic
fn-78 syntax/monitoring/qualification and fn-70 remain deliverable without it. Existing shutdown
and bounded cleanup cancellation behavior stays in scope. On resume it re-plans on fn-85
(entities, actions, sets) and takes the cancel Query and the Testpilot cancel instructions that
fn-85 left out; its re-planning note in Flow lists them.


**fn-70 — Scheduled canary proof of concept:** deferred by user decision; it was previously
queued after fn-78 as the second model consumer. Resume on an explicit user request. Nothing in
the delivery queue depends on it. On resume it inherits fn-80's `temporal.DeriveProfile`, so a
second model consumer no longer hand-writes a `ProfileSpec`. The bind-and-run helpers landed as
test-local `bindCase`/`runCase` in `tests/testpilot_run_case_test.go` rather than exported from the
fixture package, because exporting them would compile the whole server into a Quick command; a
canary test under `tests/` reuses them where they are. fn-83 extracts the provisioning that forced that
placement into `common/testing/testpilot/temporal/provision` and adds `umpire-run` (both landed);
on resume fn-70 consumes both and re-anchors its catalog entry on an fn-85 canary set, since fn-85
removes the `case` block that replaced the fn-68 Producer.

[Spec](../.flow/specs/fn-70-scheduled-canary-proof-of-concept-as-a.md).
Retained scope on resume: fn-78 first, with fn-68, fn-71, fn-72, and fn-73 as transitive
prerequisites.
Nine tasks cover all ten requirements, with a SHIP plan review. Implementation follows fn-77
to serialize shared Producer edits; fn-77 is not a semantic prerequisite.

Implement the second consumer under `tools/canary`: manual check selection, a fresh scheduled
Workflow each minute, Activity-owned Testpilot execution, bounded results, and isolated repeated
measurements. Consume the Driver and binding interfaces delivered above; do not repeat their
implementation work. Retain the cross-consumer proof; fn-73 already owns the live proof that one
Case byte sequence runs against two environment bindings.

This is a local/development prototype. It does not depend on fn-26 or fn-29 and does not authorize
production deployment or replace fn-29's separately scoped production-canary design.


These entries are outside the delivery queue and are not prerequisites for it.

| Deferred spec                                                             | Revisit when                                                                                                   |
| ------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| [fn-60](../.flow/specs/fn-60-deepen-authored-lean-canonical-json.md)      | Optional handwritten canonical JSON consolidation becomes worth prioritizing; it has no downstream dependency. |
| [fn-15](../.flow/specs/fn-15-standalone-api-and-config-input-catalogs.md) | Platform completeness is needed beyond the proven model family.                                                |
| [fn-23](../.flow/specs/fn-23-veil-toolchain-compatibility-and.md)         | Optional checker adoption becomes valuable.                                                                    |
| [fn-24](../.flow/specs/fn-24-lean-native-verification-receipts-and.md)    | A verification receipt/profile platform is justified.                                                          |
| [fn-25](../.flow/specs/fn-25-optional-callerclosure-veil-binding-and.md)  | A second verification backend is justified; caller closure remains historical.                                 |
| [fn-30](../.flow/specs/fn-30-release-evidence-graph-and-manual.md)        | Real Claim Assessment evidence supports release governance.                                                    |

[fn-14](../.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md) is historical;
[fn-61](../.flow/specs/fn-61-simplify-the-umpire-go-execution-surface.md) and
[fn-63](../.flow/specs/fn-63-consolidate-umpire-go-tests-into-golden.md) are superseded by fn-64.
Any remaining `todo` children do not reactivate them. Broader test consolidation needs a new
Testpilot proposal with an independent oracle.
