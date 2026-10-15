# Compile-time reification instead of the post-compile lifter: spike and recommendation

Status: research spike with a recommendation, 2026-10-13. Scala 3.9.0, scala-cli 1.17.1, Temurin 27,
macOS. No source was changed. Paths are relative to the repository root unless they start with
`/tmp`; a bare `dsl/`, `export/`, `inspect/` or `lint/` path is under `model/framework/`. Line numbers
and line counts are from the tree this was written against. Figures marked
"measured" come from the spike's outputs under `/tmp`, so they can be rechecked only on the machine
that ran it; everything else is checkable in the repository.

The question: should the post-compile lifter (`model/framework/export`, which reads TASTy in its own
JVM after compilation) be replaced by Scala 3 compile-time metaprogramming? Under that approach, step
functions, guards, properties and helpers are reified to IR while the Model compiles, and refusals
become compile errors.

The prototype lives at **`/tmp/umpire-macro-spike/`**, outside the repository (§9 describes its layout
and the commands to rerun it). It is kept for review. `/tmp` does not survive a reboot.

Follow-up, 2026-10-14: fn-174's first implementation attempt blocked on two conflicts the macro host
cannot resolve (stored helper IR invisible to callers; reference spellings lost across helpers). A
compiler-plugin prototype solved both; see `compiler-plugin-prototype.md`. The plugin host that §7
names as the fallback is now the spec's host.

## 0. Answer in brief

- **Feasible, with byte-identical output.** A 375-line macro lifter, written against the same
  `quotes.reflect` API the lifter already uses, reified Kettle's four rules (each with its guard),
  four effects and the `temperature` helper. Its output is **byte-identical** to today's lift of
  those 9 functions (sha1 `714b5bd2…`, written through the lifter's own ProtoJSON writer). It
  expands the `enter` sugar generically from its `@sugar` definition. Native `Map`/`Set` operations
  reify to the proposed `Collection` ops in the same way.
- **The rule "no inline, no macro" rests on two technical claims that are false on Scala 3.9**
  (measured; §1). A macro sees `match`/`CaseDef` intact. With `-Yretain-trees`, `Symbol.tree`
  returns bodies of defs in the same file, in other files of the same run, and in a jar's TASTy.
  One claim is still true: a macro cannot *evaluate* code of its own run. So declarations must still
  come from running the Model (fn-141 Part B), and macros do not replace that.
- **Macros help the function level, not the declaration level.** Function bodies and types are
  about 2,500 of the lifter's 12,541 non-test lines. The bulk of the shrink is still fn-141 Part B.
  On top of fn-141, macros remove the TASTy inspector run and fn-141.5's "find the function at a
  recorded span". Net code effect: roughly −300 to −700 lines beyond fn-141 (estimate; §5).
- **The slow loop is mostly toolchain.** Measured on Kettle, today's pipeline takes 28.0 s under
  `--server=false` and **5.1 s** on a warm Bloop daemon (5.6 s after a one-line edit). The
  prototype's one-line-edit loop is **1.8 s** on Bloop and 7.2 s without it. fn-172 already owns the
  toolchain fixes. Architecture saves about 3 s per lift: one lifter JVM run (2.96 s) against
  a 0.49 s exporter run.
- **Two macro-specific defects were found** (§6):
  1. **Stale IR under incremental compilation (measured).** A macro that bakes in the IR of a
     helper defined in another file does not recompile when only that helper's body changes.
  2. **Errors land at the capture site for reached defs.** A refusal inside a separately defined
     def, such as an `effects` member, is reported at the capture call (`on(...)`), with the true
     line only in the "inline stack trace". A refusal inside the macro's own argument lands on the
     exact line and column. A compiler-plugin host reports at the exact line everywhere (measured).
- **Recommendation (§7): adopt in a narrow form, after fn-172, at fn-141.5.** Make fn-141's capture
  points *thick*: the capture macro lifts the function body it already holds, instead of recording
  a span for a later TASTy lookup. Share one body-lifter module between the macro and the TASTy
  lifter, and run it in shadow mode first. Adopt only once the staleness defect has a fix. Do not
  try to reify declarations with macros.
- **A further step (§8), outside the question asked.** Once declarations are exported from the
  running Model and function bodies are lifted at capture points, the exporter itself can go: each
  DSL type builds its own `ir` message, the capture point returns an `ir.Function`, and lift is
  "construct, then print". It is the only shape with one description per construct. Its gating
  items are the same two as the narrow form, staleness and declaration capture, plus schema coupling.

## 1. Where "no inline, no macro, no runtime semantics" comes from, and whether it holds

**Origin.**

- **`.plans/DSL_OPERATORS.md`.**
  - Line 8: "the lifter reads TASTy and matches calls by method name … so an operator is a plain
    `def` or `extension` whose tree shape the lifter recognises; no runtime semantics, no macros,
    no library (fn-113 R25)".
  - Rule 5 (line 242): "No `inline`, `transparent inline` or macro on the author surface", because
    the lifter matches the unexpanded call by name.
  - This is a consequence of the TASTy design. It is not an independent principle.
- **`.plans/SCALA.md:108-114`** (option C, macro lifting) lists two caveats. First, a macro sees only
  a reference to a separately defined def; reading its body "needs `-Yretain-trees` or TASTy".
  Second, "a Scala macro cannot evaluate code compiled in the same run".
- **`model/README.md:2021-2025`** and **`model/framework/export/Lift.scala:8-10`** give the reason
  the lifter is not a macro: "a macro sees a function's body only for definitions of its own
  compilation run, and there only after pattern matching has been compiled away".
- **fn-113 R15** retired the native evaluator ("no runtime semantics"). **fn-113 R25** weighs
  libraries; it names neither macros nor inline. fn-141 already records all of this (Implementation
  Tradeoffs).

**Does it hold?**

| Claim | Status | Evidence |
| --- | --- | --- |
| A macro sees bodies only after pattern matching is compiled away | **False** (3.9.0) | `exp1` probe: a macro argument holds `Match`, `CaseDef` and `If` trees. Pattern matching is compiled in a much later phase than macro expansion (inlining runs right after pickling). |
| A macro sees a def's body only within its own run | **False with `-Yretain-trees`** | `exp1`: without the flag, `Symbol.tree` gives a `DefDef` with no right-hand side, in every case. With it, the body is there for a def in the same file, in another file of the same run, and in a separately packaged jar (`kit.jar`, read from its TASTy). |
| A macro cannot evaluate code of its own run | **True** | Still the reason declarations (Queries, builders, helper defs that declare) cannot be computed at compile time. `.plans/archive/cmp/scala/README.md:56` hit this. fn-141 Part B solves it by running the Model. |
| TASTy is pickled before inlining | True | It stays relevant only for the TASTy lifter. |
| "No inline on the author surface" | Already has exceptions | Rule 5 itself sanctions one: `Rules.on` is `inline` with `codeOf` (`dsl/Syntax.scala:664-806`). Composition `records` is a second `inline` capture point (`dsl/Syntax.scala:241`, SEMANTICS §21) that rule 5 does not list. fn-141.5 (activated 2026-10-13) adds capture macros, and fn-141 R15 rewrites rule 5. |
| "No runtime semantics" | Partly reversed, then narrowing again | fn-126 R16 added the run-time overlap check. reachable-exploration D4 removes Scala-side checks again (Go only). fn-141 Part B needs run-time *construction*, not run-time semantics. |

**Conclusion.** The rule no longer holds as written. Its technical basis is false except for "cannot
evaluate its own run". A defensible replacement rule:

- Models and sugar use no `inline` or macro.
- The framework uses macros only at its capture points.
- No Scala code evaluates Umpire semantics.
- Declarations come from constructing the Model.

This is fn-141 R15's wording, extended to allow capture points that lift.

## 2. What the prototype shows

All paths are under `/tmp/umpire-macro-spike/`. The repository copy used for "today" is
`repo/` (rsync of `model/framework/{dsl,realize,samples,export,inspect,lint,check}`, without the
build caches or `testdata`). Nothing under the repository was modified.

### 2.1 Reifying a step function and a guard (task 1)

- **Capture point.**
  - `proto/model/framework/dsl/Syntax.scala` is the DSL copy. Its only edit is in `Rules.on`
    (5 lines):
    - `cases` becomes an `inline` parameter;
    - `reified += framework.reify.Reify.rule[Firing[S, O, F, I] ?=> Unit](a)(cases)` is added.
  - The macro returns the IR JSON as a string constant at the call site, so Kettle is unchanged.
- **Macro lifter.** `proto/reify/Lifter.scala` (375 lines) handles the following.
  - **Rule lowering:** each `where(g) ~> effect` arm becomes `if g(s) then effect(s) else …`,
    ending in an empty list at the `on` position. The rule is named `<machine>.rules.<action>`; the
    machine name comes from `Symbol.spliceOwner`.
  - **Body lifting:** literals, `var`, field, call, `copy` with default getters, `if`, `match`,
    `let`, binary and unary operators, `List`/`Nil`/varargs, `Step` with default arguments, `UpTo`
    erasure, enum literals, and non-local `val`s inlined at their definition position.
  - **Generic `@sugar` expansion:** the call's arguments are bound to the definition's parameters,
    and the definition body is lifted at the call position (fn-141 R4). This gives `enter` and
    `ok.outcome` through the resolved `given Ok`.
- **Exporter.** `proto/export/Export.scala` constructs `samples.Kettle.Model.rules` and parses each
  JSON array into the generated `ir.Function`. This checks that the JSON fits the schema. It then
  writes the result with the lifter's printer (`Printer` + the same `Pretty` class).
- **Result.**
  - All 9 functions are equal to the lifter's as messages and **byte-identical**:
    - `model.rules.{pressButton,failover,heat,depressButton}`;
    - `samples.Kettle$.Model$.effects$.{…}`;
    - `samples.Kettle$.temperature`.
  - Output: `out/final-proto/reified.json` and `out/final-proto/lifted-subset.json`. The baseline
    is a fresh lift of the current Kettle, `out/today0/Kettle.json`, identical to
    `/tmp/kettle-check/out/Kettle.json`.
  - Positions, the given's position for the outcome literal (line 35) and the
    call/argument position split of the expanded `enter` all match without special-casing.

### 2.2 Compile-time refusals at the source position (task 2)

- **`var` and `while` in `effects` defs, through `on`** (`broken/Broken.scala`;
  `out/broken-errors.txt`):
  - Both are refused with the lifter's text ("`var next` has no IR form: …", "a loop has no IR
    form …").
  - Each is reported at the **capture site**: `Broken.scala:33:20`, the `on(hand.press)(…)` line.
    The true line, `Broken.scala:25`, appears only in the inline stack trace.
- **The same constructs through a direct macro call** (`broken2/Direct.scala`;
  `out/direct-errors.txt`):
  - When the refused code is *inside the macro argument* (a lambda), the error is at the exact
    position: `Direct.scala:24:4`, the `while`.
  - When it is in a def the macro reaches by `Symbol.tree`, the error is anchored at the call site
    (`Direct.scala:22:46`), with the trace naming line 14.
- **Compiler-plugin host** (`plugin/Plugin.scala`, 44 lines; `out/plugin-errors.txt`). "Plugin"
  here and below means a Scala compiler plugin, not the lifter's own plug-in seam
  `framework.export.Plugin` (fn-169.1), which is unrelated.
  - The *same* `Lifter` class runs in a standard plugin phase between `pickler` and `inlining`. It
    gets its `Quotes` from `QuotesImpl()`, which is public in bytecode.
  - It reports `Broken.scala:25:12` (`var`) and `Broken.scala:29:8` (`while`), the exact positions.
  - On Kettle it lifts all four effects identically to the lifter.
  - It loads through scala-cli as `-O -Xplugin:plugin.jar:reify.jar`.

### 2.3 Declarations (task 3)

- **Recommendation: read the constructed objects at run time (fn-141 Part B), not macros.**
- **Why not macros.**
  - A macro cannot evaluate same-run code (§1). So reifying `val queries = Vector(query(...) …)`,
    a Query comprehension, or a helper def that declares would mean re-implementing the lifter's
    `fold` evaluator inside a macro. That is the very duplication fn-141 removes.
  - The run-time route makes any declaration-level Scala legal, which is the owner's goal (fn-141
    R13).
- **What macros do add.**
  - **Names and IDs:** `Symbol.spliceOwner` names the enclosing `val` or object at compile time, so
    a macro can supply the naming context fn-141 R7 wants. This is the `sourcecode` question;
    fn-141.5 weighs it.
  - **Function IR at the capture point:** the exporter merges the baked-in function IR with the
    declaration IR it reads from the constructed objects. The prototype does exactly this for
    `Rules`: `reified` is a field of the constructed `rules` object.

### 2.4 Native collections (task 4)

`collections/Ledger.scala` declares `Books(balances: Map[Acct, Int], open: Set[Acct])`. Two
`Reify.function` capture points reify the following (positions stripped, from `out/ledger.jsonl`):

```text
deposit: if (a OP_CONTAINS s.open)
           then s.copy(balances = Collection{OP_UPDATED, s.balances, [a, OP_ADD(Collection{OP_GET_OR, s.balances, [a, 0]}, 1)]})
           else s
close:   let open$1 = Collection{OP_EXCL, s.open, [a]}; let balances$1 = Collection{OP_REMOVED, s.balances, [a]};
         s.copy(balances = balances$1, open = open$1)
```

- **Recognition** is by receiver type (`immutable.Map`/`Set` base classes) and method name. This is
  the table of reachable-exploration §6.5. The `Collection` node is the *proposed* §6.1 schema; it
  is not in today's `expression.proto`, so the exporter cannot parse it into `ir` classes yet.
- **Out-of-order named arguments.** `close` shows a tree quirk the lifter already handles
  (`Expressions.scala` `arguments`/`synthetic`): they arrive as synthetic vals. The spike does not
  port that handling, so they appear as `let`. The macro sees the same trees as TASTy, so the
  existing code ports unchanged.
- **Unsupported operations.** Any `scala.collection` call without a row is refused at compile time
  ("outside the liftable subset: IterableOnceOps.sum" at `Ledger.scala:24:69`;
  `proto/collections-reject/`, `out/ledger-reject.txt`).

## 3. Measurements

**Method.**

- Wall time, 3 runs per scenario, median reported. Script: `scripts/measure.py`; raw data:
  `out/measure/timings.json` and `out/measure/measure.log`.
- Kettle only. "Today" mirrors `framework/check/Samples.scala`: `package` + `compile
  --print-class-path` + `run framework.export.lift`, on the `repo/` copy.
- "Prototype" is one `scala-cli run` that compiles dsl + realize + Kettle (with the capture macros)
  and runs the exporter.
- **Bloop.** Server runs use a *private* daemon (`--bloop-daemon-dir /tmp/umpire-macro-spike/bloop-daemon`),
  because another process on the machine was using the shared one. "Daemon-cold" means the private
  daemon was stopped and the build caches deleted before each run. "Clean build" means the caches
  were deleted and the daemon kept warm.
- **Noise.** Other builds (Bloop, Metals) were running on the machine; the spread within each
  scenario was under 0.5 s except where noted.

| Scenario | Today: package | classpath | lift | **Today total** | **Prototype total** |
| --- | ---: | ---: | ---: | ---: | ---: |
| `--server=false`, caches deleted ("cold") | 8.3 | 7.6 | 12.9 | **28.7** | **6.8** |
| `--server=false`, warm | 7.9 | 7.4 | 12.7 | **28.0** | **6.8** |
| `--server=false`, one-line edit to `heat` | 8.1 | 7.5 | 12.9 | **28.6** | **7.2** |
| Bloop, daemon-cold | 10.4 | 0.6 | 10.3 | **22.6** | **8.5** |
| Bloop, clean build on a warm daemon | 4.6 | 0.5 | 8.2 | **13.3** | **3.6** |
| Bloop, warm, nothing changed | 1.0 | 0.5 | 3.6 | **5.1** | **1.5** |
| Bloop, one-line edit to `heat` | 1.6 | 0.5 | 3.5 | **5.6** | **1.8** |
| Bloop, edit to the macro implementation (reifier jar repackaged; recompiles every Model) | – | – | – | – | **3.4** |
| Lift alone, prebuilt lifter, plain `java` (the fn-172 R4 shape) | – | – | 2.96 | – | exporter alone: **0.49** |

**Reading.**

- **`--server=false` recompiles everything on every call.** Cold equals warm, the lifter is
  recompiled inside every lift, and the Models are compiled twice. scala-cli documents
  `--server=false` as "no caching or incremental compiler". This is fn-172 R1/R2/R4 territory and
  holds for either architecture.
- **Earlier figures.** The figures quoted at the start of this work (package 32.6 s, classpath
  12.4 s, lift 30–40 s) were taken on the repository under load. On the `/tmp` copy, with the same
  commands, they are 8 s, 7.5 s and 13 s.
- **Architecture vs toolchain.**
  - With a warm server on both sides, an edit costs 5.6 s today against 1.8 s for the prototype.
  - Of the 3.8 s difference, about 2.5–3 s is the lifter JVM (2.96 s against 0.49 s for an
    exporter), and the rest is the second and third scala-cli invocations.
  - fn-172 R2–R4 (compile once, a prebuilt lifter, all samples in one lifter run) would bring today
    to roughly 1.6 s + 3.0 s per edit, with the lift amortized over samples. This is an estimate,
    not measured end to end.
- **Macro expansion cost is negligible.**
  - Each compile has 4 expansions (one per `on` block), costing 13–18 ms in total. Most of that is
    the first expansion's class loading (6–14 ms); later ones take 0.4–1.4 ms
    (`out/macro-times.log`).
  - The expansion is separable and measured directly. `-Yretain-trees` overhead is folded into the
    compile time; on Kettle it is not visible against today's `package`. **Unverified for
    activity-sized Models.**
- **Incremental.** With Bloop, the edit recompiles only the edited file, and its 4 rules are
  re-expanded. An edit to the macro implementation changes the classpath and recompiles every Model
  (3.4 s for Kettle). That is the same as a `framework/dsl` edit today.

## 4. Feasibility per lifter responsibility

This uses fn-141's table. Today's line counts are for `model/framework/export` without tests or
`project.scala` (12,541 lines, 619 `fail(`/`LiftError(` sites). They are not comparable to fn-141's 2026-10-06 figure of
8,242 lines: the realization, capability and gap code has grown since.

| Responsibility (fn-141 share) | Where it goes | Notes |
| --- | --- | --- |
| **Refusals and validation** (27%) | Split. **Function-level refusals move to macros**, as compile errors with today's text. "This spelling cannot be read" refusals **disappear with running the Model** (Part B). Construction checks **stay** in factories. The order, structure and marker lints (`lint/`, 2,264 lines) **stay**. | Verified: `var`/loop refusals keep their text. Position caveat: §2.2. The initialization-cycle lint is whole-program; a macro cannot do it, and Part B makes it more important (fn-141 R14). `-Wsafe-init` does not check objects (`lint/Order.scala:43`). |
| **Recognizing which DSL call a tree is** (15%) | Mostly **gone by running the Model**: a capture point *is* the DSL call, so nothing is recognized by name at the declaration level. Body-level forms (`Step`, `choose`, `because`, channel ops, `hole`, `through`, roles) **move to macros** unchanged. | The prototype recognizes `where`/`always`/`~>` structurally inside `on`. With constructed `Rule` values (Part B / fn-141.10) even that goes. |
| **Assembling IR messages** (11%) | Declarations: **running the Model** (the exporter). Functions: **macros**, emitting ProtoJSON-shaped strings that the exporter parses into `ir` classes. | Byte-identical through the lifter's printer (§2.1). The alternative, ScalaPB classes inside the macro, puts the generated IR jar on every Model's compile classpath through the macro module. Not tried. The DSL already depends on the ScalaPB runtime (`dsl/project.scala`) and the Temporal Models already compile against `build/api-scalapb.jar` (`model/project.scala`), so this is a new jar, not a new kind of dependency; §8 takes it. |
| **Lifting function bodies and types** (10%; now `Expressions` 1,043 + `Types` 202) | **Moves to macros nearly verbatim.** | Both hosts use `quotes.reflect`. The only seams are "definition of a symbol" (`Index.defs` against `Symbol.tree`) and "refuse" (`LiftError` against `report.error`). |
| **Lowering sugar** (9%; `Expansion` 508, `Syntax` 1,163) | Body sugar: the generic `@sugar` expansion **moves to macros** (the prototype does `enter` in about 15 lines through `Symbol.tree`). Declaration sugar: **running the Model** (fn-141 Part A/B). | The remaining owners are listed in SEMANTICS §21 (fn-141.8–.13). |
| **Names, IDs and positions** (7%; `Context` 455) | Function names and positions: **macros**, with compiler positions read directly. Declaration names and IDs: **running the Model**, with capture context (fn-141 R7). A macro can supply it through `Symbol.spliceOwner`. | Positions improve. Today `Index.sourceLine` re-reads source files relative to the working directory, and `Context.written` swallows a failure. A lift run from another directory silently writes `"position": { }` and exits 0 (observed: `out/today0/K1.json`). Byte identity needs the same `-sourceroot`-relative paths and `@gap`-aware start lines (`Index.startPastGaps`). |
| **Evaluating Scala at lift time** (7%; `Constants`, `fold`) | Declaration `fold`: **gone by running the Model**. Body-level constant `val`s: **macros** (lifted at their definition position, verified with `boilingTemperature` and `floor`). | |
| **Plumbing** (15%; `Lift` 419, `Lifting` 237, `inspect/Index` 103, prefixes, `--ir`) | The TASTy inspector driver, jar and prefix handling and the separate JVM are **removed**. IR-file assembly **stays**, in the exporter Part B already needs. | The lints in `lint/` still run on the TASTy inspector unless they move to a plugin phase. So the inspector JVM survives as a lint step (§7). |

## 5. Code-size estimate (unverified; from the inventory above and the prototype)

- **Ported, about net zero.** The function-level code moves: `Expressions`, `Types`, `Expansion`,
  `Roles`, `Equality`, `Constants` and part of `Context`, about 2,300–2,500 lines. Its `Index`
  lookups become `Symbol.tree`. A 5–10% shrink is plausible; the prototype covers Kettle's subset
  plus collections in 375 lines.
- **Removed beyond fn-141.**
  - TASTy inspector plumbing for the IR: roughly 600–800 lines (`Lift.scala`'s inspector and
    targets, `Lifting` entry, `inspect/Index` source-line reading, prefix handling).
  - fn-141.5's span lookup and binding, which is not written yet: an estimated 200–400 lines
    avoided.
- **Added.**
  - Capture-point macros, about 10–12 kinds (rules: `on` arities and `Class`; `effect`/`is`
    blocks; `holds`/`holdsAcross`; monitor `next`/`after`; `leadsTo` `from`/`to`; `end`;
    refinement maps; evidence; capability function fields), 15–30 lines each: 200–350 lines.
  - JSON emission: about 150 lines.
  - A staleness guard (§6): 50–100 lines.
  - Total: about 450–650 lines.
- **Net beyond fn-141: about −300 to −700 lines.** The large reduction is fn-141 Part B's: its
  projection was a non-lint lifter of 3,000–3,500 lines. Macros neither enable nor block that.
- **Tests.** The 331 fixture files (16,447 lines) stay as sources. Refusal fixtures can become
  in-process checks: `scala.compiletime.testing.typeCheckErrors` returns the macro's refusal text
  and column for both plain and `transparent inline` capture points (verified, `proto/tce`). Today
  each needs a separate build and lift.

## 6. Risks

- **Incremental staleness (measured; the main defect).**
  - Experiment (`scripts/stale.sh`, `out/stale-server.txt`): `temperature` was moved to its own
    file, and only its body was edited (`+ 1`).
  - Under Bloop, the reified IR stayed at the old body. Zinc tracks the API of non-inline defs, not
    their bodies, so the file that baked in the helper's IR is not recompiled.
  - With `--server=false` the IR was correct (`out/stale-noserver.txt`). The TASTy lifter cannot
    have this defect, since it rereads all TASTy on every run.
  - Possible fixes:
    - (a) Bake each def's IR where the def is defined, with capture at the definition site, so the
      def's own file recompiles. That needs a wrapper or a macro annotation; SIP-63 annotations are
      experimental in 3.9 (`.plans/DSL_SIMPLIFICATION.md:64`).
    - (b) A plugin that writes one IR fragment per compilation unit, which the exporter merges.
    - (c) The macro records a hash of each reached def's source span, and the exporter refuses or
      forces a rebuild when a hash no longer matches.
    - (d) Generate IR only from full builds.
  - (c) is the cheapest guard. (a) is the cleanest. **Owner decision (2026-10-13): (a)**, with the
    rule that a capture reads its own compilation unit only; a def reached across files carries a
    macro annotation that lifts it where it is defined (fn-174 R2).
- **Error positions for reached defs (measured).** These are reported at the capture call, with the
  true line in the inline trace (§2.2). Metals shows the primary position, so the squiggle sits on
  `on(...)`, not on the `var`; this effect is **unverified in Metals**. A def reached from several
  captures reports once per capture. The fix is the same as staleness (a): capture where the
  function is written. A plugin host avoids the issue (measured).
- **Macro hygiene.** Low. The macro only reads trees and returns a string constant, generating no
  code that refers to author symbols.
  - Real concern 1: it reads bodies outside its argument, which is non-local and causes the
    staleness defect.
  - Real concern 2: `cases` becomes an `inline` parameter used twice (by `block` and by the macro).
    This is harmless while the macro only reads it.
  - **Unverified:** how nested inline calls inside a captured body look to the macro, for example
    the composition `records` capture inside a Property. The TASTy lifter sees them unexpanded and
    refuses expansions loudly (DSL_OPERATORS rule 5).
- **API stability.**
  - `quotes.reflect` is the public, versioned metaprogramming API. It is also what the lifter and
    the TASTy inspector use today, so the exposure is about the same.
  - `-Yretain-trees` is a private `-Y` flag, and the `Symbol.tree` documentation warns that trees
    may be missing. The approach depends on it.
  - Macros compiled with 3.9 are expected to run on later 3.x compilers. **Unverified here.**
  - The plugin route uses internal compiler API (`dotty.tools.dotc.plugins`,
    `scala.quoted.runtime.impl.QuotesImpl`). That is less stable, much as the inspector already
    requires a compiler at least as new as the Models'.
- **IDE experience (Metals).** Bloop diagnostics carry macro errors, with the anchoring caveat
  above.
  - `typeCheckErrors` (typer plus inlining on a snippet) reports refusals for both plain and
    `transparent` capture points.
  - Whether Metals' presentation compiler shows them *as you type*, or only on save, is
    **unverified**. `transparent inline` capture points are the likeliest to show while typing.
- **Cross-module inline limits.**
  - The macro implementation must be compiled before its use, so it needs its own module and jar
    (here `reify.jar`), built once like the lifter. Scala 3 can suspend same-run units, which was
    not measured.
  - Defs in other jars are readable only if the jar has TASTy and the Models compile with
    `-Yretain-trees` (verified for a jar).
  - A function-valued *parameter* of a helper is opaque to the macro, exactly as fn-141 R8 states
    for the run-time capture.
- **Incremental invalidation.** An edit to the macro implementation recompiles every Model (3.4 s
  for Kettle; the same as a `framework/dsl` edit today). Otherwise only the edited files recompile,
  subject to the staleness defect.
- **Testing macros.** Body-lifting logic can be unit-tested in-process through `typeCheckErrors`
  for refusals and constructed capture values for IR. That is cheaper than today's
  separate-build-per-fixture.
  - Shared-module option: the same lifter class can still be driven by the TASTy inspector in
    tests, as the plugin probe shows with a plugin host.
- **Byte identity (fn-141's contract).** Achieved on 9 Kettle functions through the lifter's
  writer. Remaining risks:
  - file paths: the spike cuts at `/model/`, where the real code needs `-sourceroot` relativization;
  - `@gap`-aware start lines;
  - synthetic-parameter renaming and synthetic-val substitution: the same trees, so port the same
    code;
  - JVM class-file string constants are limited to 64 KB. The largest function in today's IR is
    27 KB compact (`activitySystem.rules.reset`), but a capture that bundles reached functions can
    exceed the limit. Emit one constant per function, or chunk.

**Against the friction seen this session.**

- **Liftable-subset surprises** (`Vector`, tuple patterns, `.copy` on a call result, `reject`
  overloads): macros do not widen the subset; the same code decides. They make the refusal arrive
  at compile time. `Vector` in `queries` is declaration-level and disappears with Part B.
- **Initialization-cycle lints:** not helped by macros.
- **The `State`/`Outcome` type-member clash:** orthogonal; a DSL naming issue.
- **Refusals far from the source:** fixed for code inside capture arguments. For reached defs,
  they arrive at compile time, anchored at the capture.
- **The slow loop:** mostly toolchain (§3).

## 7. Recommendation

**Do not replace the lifter wholesale with macros.** The declaration level, about 80% of the code,
must come from running the Model whichever way bodies are lifted.

**Adopt macros narrowly**, as *thick capture points* inside fn-141.5's capture design. Each capture
point already holds the function tree, so it lifts the body to IR at compile time. This removes the
recorded-span lookup in TASTy and the lifter JVM.

**Conditions for adoption.**

1. fn-172's toolchain fixes land first. They give most of the speedup for either architecture, and
   the comparison should then be redone on activity_mini.
2. The staleness defect has a tested fix (§6 (a) or (c)), with an incremental-edit regression test
   under Bloop.
3. Capture-site error anchoring is accepted, or capture moves to where functions are written.
4. Shadow mode (phase 2 below) shows byte identity on every lifter fixture and on activity_mini.
5. `-Yretain-trees` costs under about 10% compile time on activity_mini (unmeasured).
6. 64 KB constant chunking is in place.

**Owner decision (2026-10-13), after this recommendation was read:** adopt the further step of §8,
not the narrow form. Captured as Flow spec fn-174 (`.flow/specs/fn-174-the-model-constructs-its-own-ir-dsl.md`),
which supersedes fn-141 Part B's mechanism and runs after fn-172. Conditions (2) and (3) are met by
definition-site capture (§6 fix (a), macro annotations, `-experimental` accepted); condition (4),
byte identity, is dropped for the migration, with the Go checks' answers as the oracle; declaration
positions may be empty; names stay. The phased plan below is superseded by fn-174's tasks and is
kept for the record.

If (2) or (3) cannot be met cheaply, use the **plugin host** instead. It is measured to give exact
positions and to run the same `Lifter` class. Weigh its internal-API exposure against the TASTy
inspector's, and give it per-unit IR fragments to avoid staleness.

**Phased migration that keeps the gate green.**

0. **fn-172:** a warm server where it is safe, compile once, a prebuilt lifter, one lifter run for
   all samples. Measured headroom: 28 s to about 5 s per Kettle cycle.
1. **Extract the body lifter** (`Expressions`, `Types`, `Expansion`, `Roles`, `Constants`, the
   position and name parts of `Context`) into a module generic over `Quotes`, with two seams:
   "definition of a symbol" and "refuse". The TASTy lifter uses it unchanged. No IR or refusal
   changes.
2. **Shadow capture.** `Rules.on`, `effect`/`is`, `holds` and the other capture points bake IR
   through the shared module. The lifter's fixture test and the gate compare baked IR with
   tree-lifted IR for every function, byte for byte, and fail on difference. Add the staleness
   guard here. Output is unchanged.
3. **Switch functions, kind by kind,** inside fn-141 Part B (R11's order). The exporter takes
   function IR from captures and declaration IR from constructed objects. A function kind leaves
   the TASTy path only when its shadow comparison has been clean on every fixture.
4. **Retire the inspector for IR.** Keep it, or a plugin phase, for the order, structure and marker
   lints. Update `.plans/DSL_OPERATORS.md` rule 5, `model/README.md:2021-2025` and
   `export/Lift.scala:8-10`, whose stated reasons §1 shows are false.

Each phase leaves the checked-in IR and Cases byte-identical, so a byte difference is a defect of
the step, as fn-141 already requires.

## 8. A further step: the Model constructs its own IR

This section goes beyond the question asked. It records the shape that removes the second
description of every construct, which §7's narrow form does not, and what it would cost. It is not
part of the recommendation; it is the end state the recommendation is compatible with.

### 8.1 The idea

Today every construct is described twice. When a Model runs, `action(this)` builds an `ActionDecl`,
`object rules` fills its buffer of `Rule` values (`dsl/Syntax.scala:981-995`), and the gate's
`IrFile.construct` instantiates every reachable machine and binds its rules. Then the lifter's JVM
reads TASTy, recognises the same `action(...)` and `on(...)` calls by name, rebuilds the
declarations from trees, lifts the bodies, assembles an `ir.Model` (`export/Lift.scala:170`) and
prints it (`export/Lift.scala:362-374`). fn-141 Part B removes the tree-side description of
declarations by adding an exporter that walks the constructed objects. The further step removes the
exporter as well, in three moves.

1. **Declarations construct their own message.** `ActionDecl` (`dsl/Action.scala:34`) already has
   nearly the fields of `ir.Action` (`machine.proto`): name, actor, on, creates, inputs, results,
   examples, timer, internal, delivers, loses. It lacks the ID, the position and the schemas. So
   `action(this)` builds the `ir.Action` where it runs, with name and position from the capture
   context fn-141 R7 asks for. Each DSL type gets a short conversion to its message, written beside
   the type: machines, channels, monitors, assumptions, holes, compositions, Properties, Scenarios,
   Queries, progress claims and realizations. fn-141.8 already describes realizations as "one
   generic mapping from a constructed value to the IR message of its name"; the realization reader
   today builds messages through ScalaPB descriptors (`export/Realizations.scala:55-70`), so the
   generic shape exists on the tree side and ports.
2. **Capture points return the function IR.** This is §7's thick capture point taken one step
   further: `on(...)` lifts the body at compile time, as the spike does, but returns an
   `ir.Function` (or its ProtoJSON, parsed once at the capture) instead of a string a later exporter
   parses. The rule holds its function; `StepBinding` is three fields away. Guards, effects, Property
   and progress predicates, monitor functions, `end`, refinement maps, evidence and capability
   function fields have the same shape (fn-141 R8's list).
3. **Lift is "construct, then print".** Run the gate's root construction, collect the messages the
   objects now hold into an `ir.Model`, and write it with the printer lines at the end of
   `export/Lift.scala`. The lifter JVM, the TASTy inspector and the tree readers for declarations,
   claims, compositions, realizations and capabilities go. Type lifting may need no trees either:
   each machine holds its `Finite` instances for state, outcome and fact (`dsl/Machine.scala:400`),
   which can describe their types at run time.

**What stays.** The body lifter (`Expressions`, `Types`, `Expansion`, `Roles`, `Constants`, about
2,300 lines) moves into the capture macro as in §7. The order, structure and marker lints keep
reading TASTy, or move to a plugin phase. Capability expansions and gap resolution (`Capabilities`
1,308, `Gaps` 678) are computations, not data: they stay as logic that builds messages from values
rather than from trees, which is a rewrite, not a deletion.

**Size.** The declaration-side tree readers, about 8,000 lines, become a few hundred lines of
conversions plus a driver of under 100 lines. Against fn-141's projection of 3,000 to 3,500 lines
including a separate exporter, this is the smaller end state. The number is an estimate from the
inventory in §4, not measured.

### 8.2 Pros

- **One description per construct, held by the type checker.** A DSL type that builds its own
  message cannot drift from the schema. Today five tree readers must be kept in step by hand.
- **The largest deletion available.** fn-141 Part B still leaves an exporter; this leaves none.
  Tasks fn-141.8 to .13 become "add a conversion beside each type".
- **Any declaration-level Scala works.** Comprehensions, helper defs and loops that build Queries or
  realizations are just code. The "this spelling cannot be read" refusals, about a third of today's,
  disappear (fn-141 R13).
- **The shortest loop.** One compile and one short run: 0.49 s measured for the exporter against
  2.96 s for the lifter JVM (§3).
- **Function-body refusals at compile time**, at the exact position when the code is inside the
  capture argument (§2.2).
- **Fits the gate.** `IrFile.construct` already instantiates every machine and binds its rules; the
  driver extends what runs today.

### 8.3 Cons

- **The IR schema leaks into the author-facing framework.** The generated IR jar and the macro
  module sit on every Model's classpath. A schema change recompiles the DSL and every Model, where
  today it recompiles the lifter only. There is precedent (`build/api-scalapb.jar` on the Temporal
  Models), but the coupling is new.
- **Byte identity gets harder.** Repeated-field order, derived IDs and positions come from
  construction order and from what each object holds. fn-141 R10 forbids reading object identity,
  hash order or initialization order; every conversion must be audited for it, where a tree reader
  is deterministic by construction.
- **Declaration names and positions need their own capture.** A constructed `ActionDecl` does not
  know its line or the `val` that names it. fn-141 R7 covers this, but it is a capture on every
  declaring call, not only on function captures.
- **The staleness defect stands** (§6). Function IR baked into a class file does not refresh when a
  helper in another file changes under incremental compilation. The §6 fixes must land first.
- **Computations do not shrink.** About 2,000 lines of expansion and gap logic are rewritten from
  trees to values, not removed.
- **Error anchoring for reached defs** is at the capture call (§2.2) unless the plugin host is used.
- **The whole-program view moves to run time.** A half-built object exported silently gives wrong IR
  rather than a refusal, so the initialization-order lint matters more (fn-141 R14).
- **The test suite changes shape.** The 331 fixtures compare lifted IR to expected IR; here a fixture
  is a Model that compiles and runs, and a refusal fixture is a compile-error check
  (`typeCheckErrors`, §5). Cheaper per fixture, but a migration.
- **It cuts across fn-141's phased plan** in form, not in substance: Part B's one-kind-at-a-time
  order and byte-identity gate are the same work. The saving is in the final size, not in the number
  of steps.

### 8.4 Reading

**Status (owner, 2026-10-13): adopted.** This is now the plan, as Flow spec fn-174, with three
decisions: staleness is removed at the definition site with macro annotations (§6 (a)); byte-for-byte
identity is not required during the migration, the Go checks' receipts, Cases and backend answers
are the oracle; declaration positions may be empty and names stay. fn-141 Part B's obligations carry
over (its .6 and .7 are prerequisites); its mechanism, an exporter per kind plus span lookup, is
superseded. The reading below was written before the decision.

The pros are structural and permanent; the cons are migration cost plus two design risks, schema
coupling and staleness. If fn-141 Part B happens, the marginal cost of putting each conversion on
the DSL type instead of in a separate exporter is small, and it is the only option that removes the
second description entirely. The staleness fix and declaration capture gate the schedule either
way; this choice does not.

**Not taken: an embedded expression DSL.** Writing step bodies as typed combinators over an
expression type would make function bodies constructed values too, so nothing would read trees and
no macro would exist. The project rejected this on purpose: native `match`, `if` and `copy` on plain
values is the point of the DSL (`model/README.md:2021-2025`), and fn-141 R13 forbids spelling
changes. It is named here so a reader does not wonder why the only route to zero lifter is absent.

## 9. The spike: layout and how to rerun it

Archived in `.plans/archive/scala-macros-spike/` (sources, scripts and the small outputs; the jars,
build state, the copied `repo/`, `qi/` and the large measurement outputs were left out). The scripts
still name the original `/tmp/umpire-macro-spike/` paths; rerunning needs that layout, or edits to
`scripts/`:

| Path | What |
| --- | --- |
| `scripts/sc` | scala-cli 1.17.1 with the mise Temurin 27 |
| `scripts/today.sh <out> [server opts]` | today's package + classpath + lift of Kettle on `repo/` |
| `scripts/proto.sh <out> [server opts]` | the prototype: compile with capture macros, then export and compare |
| `scripts/measure.py [scenario…]` | the timing table (§3); results in `out/measure/` |
| `scripts/stale.sh server\|noserver` | the incremental staleness probe (§6) |
| `exp1/` | the `Match` and `Symbol.tree` probe (§1); run with `-O -Yretain-trees` and without |
| `proto/reify/` | the macro library: `Reify.scala` entry points, `Lifter.scala` body lifter, `Json.scala`; packaged to `proto/reify.jar` |
| `proto/model/` | DSL copy with the `Rules.on` capture point, Kettle unchanged, `-Yretain-trees` |
| `proto/export/Export.scala` | the run-time exporter and comparator |
| `proto/plugin/` | the compiler-plugin host of the same `Lifter` (§2.2) |
| `proto/{broken,broken2,collections,collections-reject,tce}/` | refusal, direct-call, collection and `typeCheckErrors` probes |
| `stale/` | Kettle with `temperature` in its own file |
| `out/` | every output cited above |

To rerun the core result: run `scripts/proto.sh /tmp/x`, then `cmp /tmp/x/reified.json
/tmp/x/lifted-subset.json`. To rebuild the reifier after editing it: `cd proto/reify &&
../../scripts/sc --power package --server=false --library . -f -o ../reify.jar`.
