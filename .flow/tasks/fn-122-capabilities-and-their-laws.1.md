---
satisfies: [R1, R4]
---
# fn-122-capabilities-and-their-laws.1 Write the law bodies and the catalog table as plain defs with server citations

## Description
Create the entity-neutral and Temporal law defs with their `Law` values, the catalog as a Scala value, and the inventory that classifies every existing claim; move fn-112 R4's three parameterized shared-claim defs under the laws without changing their meaning, so `terminalIsFinal`/`terminalStays` are the first instances of `terminalStatesAreFinal` and `pausedIsNotDispatched`/`notAdmittedWhilePaused` of `pausedIsNotDispatched` (`Pausable x Pollable`). This is the early proof point: laws must lift as plain Property-returning defs through the existing fold.

**Cross-spec entry gate (flowctl tracks only same-spec deps):** start only after fn-112.10 is done (claim patterns, function-argument binding, DefinitionScope and the structural Case-byte freeze exist). Verify with `flowctl show fn-112-make-the-standalone-activity-scala.10` before claiming. fn-114 may be running; this task adds files and edits the activity's `Properties.scala` files only to re-import the moved defs (fn-114.1/.7 may touch them too; whichever lands second rebases).

**Size:** M
**Files:** `model/umpire/laws/Catalog.scala` (the `Catalog` type, `Law` value type, entity-neutral entries); `model/umpire/laws/Laws.scala` (`terminalStatesAreFinal`, `closedIsRejectedUniformly`); `model/temporal/laws/{Close,Terminate,Pause,Cancel,Poll,Describe}.scala` (one def per law with its `Law` value) and `model/temporal/laws/Catalog.scala` (the `given Catalog` adding the Temporal laws); `model/temporal/standaloneactivity/Properties.scala` (the three fn-112 R4 defs move out, call sites re-import); `.plans/SEMANTIC_PROTOCOLS.md` section 3 table updated as the inventory; `.flow/tmp/fn122-1/**`.
**Touches:** [model/umpire/laws/**, model/temporal/laws/**, model/temporal/standaloneactivity/Properties.scala, model/temporal/standaloneactivity/admission/Properties.scala, model/temporal/standaloneactivity/compositions/Properties.scala, tools/umpire/internal/golden/config.json, model/ir/activity*.json, .plans/SEMANTIC_PROTOCOLS.md, .flow/tmp/fn122-1/**]

### Approach
- A law takes the model and the capability's fields and returns a `Property` (spec API Contracts): `def terminalStatesAreFinal[S, P](m: Declares[S])(status: S => P, terminal: P => Boolean): Property[S] = m.property(…).once(s => terminal(status(s))).keeps(status)`; `pausedIsNotDispatched(m)(paused, running)` as `never(s => running(s.state)).from(paused)`. This is exactly fn-112 R4's def shape and `Declares[S]` is the typed supertype fn-112.4 introduces; a law written as a predicate cannot use the patterns, so none is. No transition Property with `when`.
- `Law(cites, promises, doesNotPromise)` is plain data beside each def (a small case class in `umpire/laws`), not a docstring: the lifter reads values, not comments. Citations are file paths, e.g. `chasm/lib/activity/handler.go` for terminate, `chasm/lib/nexusoperation/operation.go` for `ErrOperationAlreadyCompleted`, `chasm/lib/scheduler/scheduler.go` for the schedule's missing reason.
- The catalog: `Catalog` maps a capability type, and an unordered pair, to law defs by reference; `umpire/laws` holds the type and the entity-neutral entries, `temporal/laws` the one `given Catalog` the Models use. A catalog test enforces the two-entity rule with the spec's one definition (a machine with its own state type that declares the capability) over the checked-in Models' declarations, failing by law name; until task 3 declares capabilities the test reads the inventory's classification as the planned declaration set and says so.
- Inventory: classify every Property, monitor and progress claim under `model/temporal/**` as single-capability law, interaction law or feature-specific (R1), extending `.plans/SEMANTIC_PROTOCOLS.md` section 3's table; a claim that fits no class is listed for the owner. This decides task 3's retirement list and the Boundaries' "stay authored" list.
- Moving the fn-112 R4 defs changes only their function symbols; record them as `function_name_substitutions` entries in the golden config, as fn-114's file moves do. Tables, IDs, fingerprints, answers and Case bytes stay exact (fn-112.1 equivalence).
- Core and sugar (spec R12): everything this task writes is core; a convenience spelling, if one is wanted, goes to `model/temporal/laws/Syntax.scala` with its core form documented, and its lifter matching to task 2's `model/lifter/Syntax.scala`.

### Investigation targets
**Required:**
- `.plans/SEMANTIC_PROTOCOLS.md` sections 1-3 - the law families, U/P/X legend and instance table
- `.plans/UMPIRE4_VISION.md` "Reusable behavioral protocols (#PROTOCOLS)" - the acceptance test this spec anchors on
- `model/temporal/standaloneactivity/Properties.scala` (post fn-112.8) - the three parameterized R4 defs and `Declares[S]`
- `model/lifter/Claims.scala` `fold` - how a def's arguments are bound (the `Apply(fn, args) if isFunction` case) and fn-112.4's function-argument binding
- `model/SEMANTICS.md` Claims section - what a Property may say
**Optional:**
- `chasm/lib/activity/handler.go`, `chasm/lib/nexusoperation/operation.go`, `chasm/lib/scheduler/scheduler.go` - citations

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir model/cases
make umpire-check-model && make lint-model
```

### Execution constraints
- No IR schema change; no new Property, Query or Case in this task (the laws are not yet instantiated by a capability declaration); lifter fixtures are task 2's.
- Behavior-neutral: do not add validation while moving code.
## Acceptance
- [ ] `umpire/laws` and `temporal/laws` hold one Property-returning def per law with its `Law` value (citation, `promises`, `doesNotPromise`), written with the claim patterns where one fits; `terminalStatesAreFinal` and `pausedIsNotDispatched` are the generalizations of fn-112's R4 defs.
- [ ] The catalog is a Scala value (`umpire/laws` type and entries, `temporal/laws` given) and its test fails by law name for fewer than two instantiating machines, using the spec's one definition.
- [ ] Every existing Property, monitor and progress claim under `model/temporal/**` is classified (single-capability law, interaction law, feature-specific) in `.plans/SEMANTIC_PROTOCOLS.md` with the law or reason; unclassifiable claims are listed for the owner.
- [ ] fn-112 R4's three defs live under the laws, their instances keep their names, and the only IR delta is the recorded function-symbol substitutions.
- [ ] fn-112.1 equivalence, model gate and lint-model pass; evidence under `.flow/tmp/fn122-1/`.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
