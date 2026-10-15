# A compiler plugin for fn-174's two blockers: prototype and verdict

Status: viability prototype with a verdict, 2026-10-13. Scala 3.9.0, scala-cli 1.17.1, Temurin 27,
Linux sandbox. No production source was changed. The prototype is kept at
`.flow/tmp/fn174-plugin-prototype/` (its `README.md` lists the layout and `run.sh` reruns it); move
it if `.flow/tmp` is cleaned. "Plugin" means a Scala compiler plugin (`dotty.tools.dotc.plugins`),
not the lifter's own plug-in seam `framework.export.Plugin`.

fn-174's two implementation attempts stopped on design conflicts recorded in
`.flow/tmp/fn174-boundary-conflict.md` and `.flow/tmp/fn1742-name-conflict.md`:

1. A caller's `choose` must refuse an alternative that calls a helper giving two steps. The
   helper's type cannot say how many steps it gives, so the caller needs the helper's lifted body.
   A macro annotation stores it after `pickler`, where no caller macro can see it (eight probes).
2. `scripted(original)` and `scripted(alias)` construct the same value, but today's exporter names
   their commands `original` and `alias` by following the helper's parameter to the caller's
   reference. Constructed values alone cannot recover the spelling.

The spec excluded a compiler plugin. This prototype tests whether a plugin supplies exactly the two
missing pieces, with everything else (Model syntax, names, compile-time refusals, no non-local body
reads) kept.

## 0. Answer in brief

**Both blockers are solved by a 166-line standard plugin, on the pinned compiler, with no `-experimental`
and no macro annotation.** Measured, with logs under the prototype's `logs/`:

- **Stored helper IR crosses the compilation boundary.** A plugin phase after `posttyper` and before
  `pickler` lifts each `@lifted` def's body at its definition and stores the result as an ordinary
  `@liftedIr(steps, at, body)` annotation on the def's symbol. Because the phase runs before
  `pickler`, the annotation is in the TASTy. A caller compiled later against the helper's jar reads it
  in a plugin phase **and** in a macro through the supported `Symbol.annotations` API. The same-run
  case (helper and caller in one compilation) works in any file order, since the store phase finishes
  for all units before the check phase starts. A `@lifted` def that calls another `@lifted` def reads
  the callee's stored count, not its body.
- **The refusal is a compile error at the caller's alternative with today's text.**
  `Refused.scala:14:34`: "the alternative outer2 of a choose calls helper, which gives 2 steps at
  …/Helper.scala:9: a function an alternative calls gives no step or one step written out in each
  branch". The helper bound directly as a rule stays accepted.
- **Staleness is solved without a hash guard, by zinc.** Under a warm Bloop server, editing only the
  helper's body from one step to two recompiled `Helper.scala` alone in zinc's first cycle, then
  invalidated and recompiled `Refused.scala` in a second cycle, which failed at 14:22. The stored
  annotation is part of the definition's API hash, so a body change that changes the IR propagates
  to every dependent unit. This is fn-174 R2's regression test, passing.
- **Reference names survive helpers and aliases.** A third phase rewrites, at every call, an argument
  that is a reference to a `val` of a nameable type into `ref.named("<spelling>")`, and leaves
  parameters alone. `scripted(original)`, `scripted(alias)`, a two-level `twice(alias)` and a direct
  `everyCase(original)` give ids `original, alias, alias, original`; `original eq alias` stays true.
  This needs a slot on the DSL value to carry the spelling (§2.2), which the real `Instruction` enum
  does not have today.
- **Costs are the known ones:** internal compiler API, a plugin jar built before the Models, and
  `-Xplugin` on every Model build. Three things were not proved here: the 2,300-line body lifter
  running inside the plugin (the earlier spike ran the same `Lifter` class in a plugin phase,
  `scala-macros-export.md` §2.2), whether `typeCheckErrors` runs plugin phases for refusal tests,
  and Metals behaviour.

**Verdict: viable. Recommend lifting the spec's plugin exclusion for a bounded implementation** (§4).

## 1. What was built

`plugin/LiftedPlugin.scala`, a `StandardPlugin` with three `PluginPhase`s. The scheduler accepted
all three after `posttyper` and before `pickler`; `-Ycheck:all` passes after each
(`logs/helper-build.log` shows `checking … after phase liftedStore`, `liftedCheck`, `liftedNames`,
then `pickler`). `-Vphases` lists only the base plan; plugin phases do not appear in it.

| Phase | Where | What it does |
| --- | --- | --- |
| `liftedStore` | after `posttyper`, before `sbt-api` and `pickler` | For a def with the author's `@liftlib.lifted` marker (a plain `StaticAnnotation`), lift the body and add `@liftlib.liftedIr(steps, "file:line", body)` to the symbol with `sym.addAnnotation(Annotation(cls, args, span))`. |
| `liftedCheck` | after `liftedStore`, before `pickler` | At each `framework.dsl.choose`, for each alternative, find called defs with a stored `@liftedIr` and `report.error` at the alternative when `steps != 1`. |
| `liftedNames` | after `posttyper`, before `pickler` | At each `Apply`, replace an `Ident`/`Select` argument that is a `val` (not a `Param`, not a method) of a type deriving from `liftlib.realize.Named`, passed to a parameter whose declared type is not `Any`/`AnyRef`/`Matchable`/`AnyVal`, by `arg.named("<name>")`. |

The lift is a stand-in: `StepCount` recognises `choose` (counts alternatives), `stay`/`enter` (one),
`if`/`match` (max over branches), `Block`/`Typed`/`Inlined`, and a call of another stored def. The
real lifter is the 2,300 lines fn-174 moves into `reify`; what matters here is that it runs over
the typed tree at the definition site and that its result reaches callers.

`lib/` is the carrier library both sides depend on: the two annotations, a `Capture.annotationsOf`
macro that prints what a macro sees on a reached symbol, and a stand-in `realize` with
`Instruction(target, ref: Option[String]) extends Named`, `everyCase` (reads `ref`, throws if
absent) and `script`. The blocker-1 fixtures use the real framework DSL from
`model/build/model-scala.jar` (`choose`, `stay`, `enter`, `Machine`, `Bindings`), unchanged.

## 2. What was measured

### 2.1 Blocker 1: stored helper IR

| Run | Command shape | Result |
| --- | --- | --- |
| Helper to jar | `package --library helper -o out/helper.jar` with `-Xplugin` | exit 0. `STORE boundary.Helper$.helper steps=2 at=…Helper.scala:9`, `single steps=1`, `viaSingle steps=1` (`logs/helper-build2.log`). |
| Allowed caller against the jar | `run --jar out/helper.jar caller-allowed` | exit 0. `CHECK … outer calls Helper$.single steps=Some(1)`, `other calls Helper$.viaSingle steps=Some(1)`. The macro prints `MACRO-SEES boundary.Helper$.helper: new liftlib.liftedIr(2, "…Helper.scala:9", "framework.dsl.choose[…")` (`logs/allowed2.log`). |
| Refused caller against the jar | `compile --jar out/helper.jar caller-refused` | exit 1, one error at `Refused.scala:14:34` with the text above (`logs/refused.log`). |
| Helper and both callers in one compilation | `compile warm/src` (no helper jar) | exit 1 with the same error at `warm/src/Refused.scala:14:34`; `ChosenSingle` accepted (`logs/single-run.log`). |

The macro observation is the one the earlier probes could not get: a generated *definition* is
invisible across the boundary, but an *annotation* added before `pickler` is visible through
`Symbol.annotations`, a supported Quotes API. So the body lifter could stay a macro and only the
carrier would be the plugin's, or the plugin could host everything; §4 weighs this.

### 2.2 Blocker 2: reference names

`naming/Probe.scala` (`logs/naming2.log`, exit 0):

```text
NAME Probe.scala: scripted(original) tagged
NAME Probe.scala: scripted(alias) tagged
NAME Probe.scala: twice(alias) tagged
NAME Probe.scala: everyCase(original) tagged
original-eq-alias=true
ids=Vector(original, alias, alias, original)
```

The first version of the rule tagged every `val` argument of a nameable type, including the
`alias` in `original eq alias`, and printed `original-eq-alias=false` (`logs/naming.log`). The rule
now requires the parameter's declared type not to be a top type, so `eq`, `==`, `println` and
collections of `Any` leave the reference alone. This is the lesson of the probe: the rewrite must
be a deliberate contract ("a declaration passed where a nameable value is expected carries its
spelling"), not "tag everything".

What this costs the real DSL: a value must be able to carry the spelling. `Command("", instruction)`
already has an id slot; `Instruction` is an enum of cases without one. Either every nameable type
gets a `ref` field (the prototype's shape), or the DSL gets one wrapper type and the parameters of
`everyCase`, `onPath`, `perform`, `aliasOf` and author helpers accept it. Both are Model-visible
only in that `.named(...)` is inserted by the plugin, never written by an author; no spelling
changes. The exporter's rule for a non-reference argument (an `Instruction.Hold("x")` written out
in a call) is not handled by the prototype; `everyCase` throws at construction. The plugin can
refuse that at compile time instead, since it sees the argument is not a reference.

### 2.3 Staleness under a warm server

`warm/incr/` holds `Helper.scala` (helper giving one step) and `Refused.scala` (an alternative
calls it). Private Bloop daemon, `--bloop-daemon-dir` under the prototype, mode 0700.

| Step | Edit | Result (`logs/warm-*.log`) |
| --- | --- | --- |
| 1 | none | Compiles all three units. `STORE … steps=1`, `CHECK … steps=Some(1)`. exit 0. |
| 2 | only `Helper.scala`: body `stay(s)` → `choose(first -> stay(s), second -> enter(!s))` | Cycle 1 types `Helper.scala` only, `STORE … steps=2`, "Compiled". Cycle 2 types `Refused.scala` only, error at `Refused.scala:14:22` with today's text. exit 1. |
| 3 | `Helper.scala` back to `stay(s)` | exit 0. In the first session the daemon had been stopped after step 2, so this was a fresh-server control; the `run.sh` rerun kept one daemon through steps 1 to 3 and step 3 was green on it. |
| 4 | none | exit 0, no output. |

Step 2 is the measured proof that zinc invalidates a dependent unit on an IR-only change: in 3.9.0
`sbt-api` runs after `pickler`, and the annotation added before `pickler` is in the API zinc hashes
(annotations are hashed by their tree's text). No guard, no `-Yretain-trees` and no non-local body
read were used. The daemon was stopped with `bloop exit --bloop-daemon-dir …`.

## 3. What was not proved

- **The real body lifter in a plugin phase.** The stand-in counts steps. The spike ran the same
  `Lifter` class in a plugin phase through `QuotesImpl()` (`scala-macros-export.md` §2.2), so the
  port is believed possible, but the 2,300-line move and its sample/mini equivalence are fn-174.1's
  work, not this prototype's.
- **Stored IR size.** The annotation argument here is a string constant. An `ir.Function` can be
  27 KB compact today; JVM constants are limited to 64 KB, and an annotation tree can hold a list
  of chunks or bytes. Not measured.
- **Refusal tests.** `scala.compiletime.testing.typeCheckErrors` runs typer and inlining on a
  snippet; whether plugin phases run inside it is unverified. fn-174 R8 already allows "a compile
  of the fixture" as the alternative.
- **Metals.** Whether plugin diagnostics show in the editor as you type is unverified; Bloop
  carries them (step 2 above).
- **Build integration beyond scala-cli.** `-Xplugin:<absolute path>` works from the directives and
  through Bloop. The jar must be built before any Model compiles, like `reify.jar` would be.
- **Compiler coupling.** The plugin uses `Annotation.apply`, `Symbol.addAnnotation`,
  `PluginPhase`, `MethodType.paramInfos`, `report`, `cpy.Apply`: internal API that may move between
  minor versions. The TASTy inspector already requires a compiler at least as new as the Models'.
- **Naming beyond the probe.** Default arguments, named arguments, varargs of nameable values, and
  constructor arguments of case classes (`Realization(scripts = Vector(scripted(alias)))`) were not
  exercised. The rule as written applies to `Apply` only.

## 4. Recommendation

Authorize the plugin for a bounded implementation, replacing the two mechanisms that failed:

1. **Carrier.** `@lifted` becomes a plain marker annotation; a plugin phase before `pickler` lifts
   the body and stores the `ir.Function` as an annotation on the def. This removes SIP-63 macro
   annotations and the `-experimental` flag from the design, and gives R2's staleness proof for
   free through zinc.
2. **Names.** A plugin phase inserts the caller's spelling at call sites of nameable parameters.
   The DSL gains the carrier slot (§2.2); authors write nothing new. The owner decides between a
   field on each nameable type and one wrapper type.
3. **Host for the rest.** Two options, to decide in fn-174.1 with a fixture:
   - *Hybrid:* capture points stay macros (`Reify.function` over `Quotes`, as the spec has it) and
     read stored IR through `Symbol.annotations`. Smallest change to the approved design; two
     compile-time mechanisms.
   - *Plugin only:* the capture points' lifting and the `choose` check also run in plugin phases,
     over `QuotesImpl` so the moved lifter code stays written against `Quotes`. One mechanism,
     exact positions everywhere (spike §2.2), no `-Yretain-trees`; more internal API.
   I lean to plugin-only, because the prototype's hardest property, "what one unit stores is what
   every other unit reads", is simplest when one phase owns both sides.

Proofs still owed before the port is accepted, in the spec's terms: R1 equivalence on the six
samples and `activity_mini`; R2's warm test on a real `@lifted` helper with the real lifter; R4
names on the realization fixtures, including the non-reference refusal; the 64 KB question; and a
`typeCheckErrors` or compile-the-fixture decision for R8's refusal fixtures.

## 5. Rerun

```sh
cd temporal && .flow/tmp/fn174-plugin-prototype/run.sh
```

The script builds `out/plugin.jar`, `out/lib.jar` and `out/helper.jar`, then runs the allowed,
refused, single-run and naming probes and the warm sequence under a private Bloop daemon, writing
`logs/`. Expected exits: allowed 0, refused 1, single-run 1, naming 0, warm 0/1/0.
