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
Wrote the capability laws as plain Property-returning defs with `Law` values (server citations, `promises`, `doesNotPromise`), the catalog as a Scala value with its two-entity test, and the inventory of every claim under `model/temporal/**`. fn-112's shared defs are now instances of the laws. Tables, Definition IDs, Property rows, Query answers and totals are unchanged, and model/cases is byte-identical. The original baseline passes with `original.json` untouched.

**What changed**
- `model/umpire/laws/Laws.scala`:
  - `terminalStatesAreFinal[S, P](m)(status, terminal)` and `closedIsRejectedUniformly[S, P](m)(status, terminal, rejected)`, each with its `Law`.
  - `Catalog.scala`: `Capability` (Closable, Terminable, Pausable, Cancelable, Pollable, Describable), `Law(name, statement, cites, promises, doesNotPromise)` with the def by reference, `Brought`, `Catalog` (`single`, `pair`, `++`, `laws(declared)`, which finds pairs among the declared set), and `entityNeutral`.
- `model/temporal/laws`:
  - `Terminate.scala` `terminateSettles(m)(terminate, settled)`, `Cancel.scala` `cancelIsRequested(m)(requestCancel, requested)`, `Pause.scala` `pausedIsNotDispatched(m)(paused, running)` (Pausable × Pollable), each with its `Law`.
  - `Catalog.scala`: the one `given catalog: Catalog`.
  - `Catalog.test.scala`: the two-entity rule over the planned declarations, failing by law name. It also checks that a pair is found without being listed and that law names are unique.
- Instances, named by their vals:
  - `terminalIsFinal = terminalStatesAreFinal(activityProduct)(Product.phase, Product.terminal)`;
  - `pausedIsNotDispatched = temporal.laws.pausedIsNotDispatched(activityProduct)(…)`;
  - `notAdmittedWhilePaused`/`terminalStays` as local vals in `admissionClaims`, `overQueueClaims` and `overMatchingClaims`.
  - The `notAdmittedWhilePaused` and `terminalStays` defs are gone.
- Vocabulary:
  - `Product.terminal` and `Admission.terminal` take the phase, as `Protocol.terminal` already did, so they are the law's `terminal: P => Boolean`.
  - New: `Product.phase`, and `Product.ends` (state-typed, keeping `ends`' inline expression).
  - The `OverQueue`/`OverMatching.terminal` forwarders are gone.
- Lifter (two small changes, both needed by the early proof point):
  - `Lift.scala` reads `umpire/laws` TASTy. The framework's TASTy was excluded entirely, so no entity-neutral law could fold.
  - `Claims.declaring` threads `named`: a def body's final declaration takes the name of the val that declares the call.
- `.plans/SEMANTIC_PROTOCOLS.md` §3 holds the inventory: 76 claims, 8 law instances, 68 feature-specific (each with its reason), none unclassified. It also lists the planned declarations, the borderline calls for the owner, and the findings.
- `model/README.md` documents the naming rule and the law sources.
- `golden/config.json` gains two `source_path_merges`, laws files compared as `model/temporal/standaloneactivity`.

**Decisions (own)**
- **No function-name substitution.** Function names follow `<machine>.property.<name>`, so moving the defs moved only positions. fn-115's by-file positions needed the two merges instead.
- **`atMostOneActive` stays in the feature.** Boundaries say one instantiating machine. "The three R4 defs move" is read as the two laws.
- **Core form in umpire.** The sugar rule forbids pattern names in model/umpire, so the entity-neutral bodies are the core form of `once(...).keeps(...)` / a `holdsAcross`. The Temporal laws use the patterns.
- **Catalog keys.** It is keyed by a `Capability` enum, since task 2 introduces the capability types. `Law.statement` holds the def by reference.
- **Planned set includes task 4.** The planned declaration set includes task 4's `nexusOperation`. Without it, Terminable and Cancelable would have one machine.
- **No Close/Poll/Describe Temporal files.** Close's laws are entity-neutral, Pollable alone has no two-machine law, and Describable's field is a realization table (see the findings).

**Findings:** the bodies of `closedIsRejectedUniformly`, `terminateSettles` and `cancelIsRequested` need value-argument binding (outcome, class, fact), which the fold lacks. A throwaway probe showed each refusal (probe.md). This is task 2's, with the capability fields. A Describe law over a describe action would need a transition Property with `when`.

**Shared files for the conductor's merge with fn-114.1:**
- `model/lifter/Claims.scala` (`declaring` threads `named`)
- `model/lifter/Lift.scala` (`lifted` filter)
- `model/temporal/standaloneactivity/{Model,Properties}.scala`, `admission/{Model,Properties}.scala` and `compositions/{Model,Properties}.scala`
- `tools/umpire/internal/golden/config.json` (two merges)
- `model/ir/activity*.json` and `model/lifter/testdata/lifts/expected/{admission,realizations}.json`: regenerate after the merge.
- `tools/umpire/model/activity_parity_test.go` and `tools/umpire/export/quint_test.go` (comment).
- `model/README.md`
- `original.json` is untouched.

**Gates:** all pass (evidence.md):
- `umpire-gen-model`, `umpire-check-model` and `lint-model`;
- the OriginalBaseline and Migration goldens;
- the full Go tooling suite: one failure on the first run, `TestActivityEveryClaimDeclarationIsLifted`, which scans source for claim vals; fixed in 225cf49ed4 and the model and export packages re-run green;
- `lint-code-fast` (0 issues).

**Review:** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Round 1 was SHIP with 3 P3s:
- P3 1 (record the IR-shape delta) is fixed in dfd36fdef9.
- P3 3 (subtotal label) is fixed in dfd36fdef9. The count was right, so it was relabelled.
- P3 2 (`Law.statement` is untyped and unread, and its name is repeated) is deferred to task 2, which reads the catalog. It can derive the name from the reference or check it there. The reference is kept because the spec names law defs "by reference".

**Deferred P3/FYI:**
- The law-file `source_path_merges` map to the activity's directory. Revisit once the Nexus operation instantiates a law, at task 4.
- The parity test lists the law files by name.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3aad890a05, 225cf49ed4, dfd36fdef9
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), make lint-model (exit 0), scala-cli test model/project.scala model/umpire model/temporal incl. temporal.laws.CatalogTest (exit 0), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|Migration' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0 after merges), go test -json -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 1: TestActivityEveryClaimDeclarationIsLifted, fixed in 225cf49ed4), go test -json -tags test_dep -count=1 -p 2 ./tools/umpire/model ./tools/umpire/export (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), flowctl claude impl-review --spec claude:claude-opus-5-5:high (SHIP, round 1)
- PRs: