---
satisfies: [R3, R4, R5, R16]
---
# fn-112-make-the-standalone-activity-scala.4 Add typed composition members, sync references and reusable claims

Touches: [model/umpire/Compose.scala, model/umpire/Claims.scala, model/umpire/Machine.scala, model/umpire/Syntax.scala, model/lifter/Compositions.scala, model/lifter/Claims.scala, model/lifter/Expressions.scala, model/lifter/Syntax.scala, model/lifter/test/**, model/lifter/testdata/**]

## Description
Replace composition/action/fact string keys with typed selectors, add the level-1 claim patterns, bind function-valued def arguments at lift time, and support one claim definition across member and composed states through a typed supertype.

**Size:** M
**Files:** model/umpire/Compose.scala, Claims.scala and Machine.scala (core: selectors, `Declares[S]`); `model/umpire/Syntax.scala` (the claim patterns and the composition `records` form, sugar); model/lifter/Compositions.scala, Claims.scala (fold), Expressions.scala (`callee`) and `model/lifter/Syntax.scala` (pattern matching); focused fixtures (`lifts/Patterns.scala` + `expected/patterns.json`, `Rejects.scala` lines).

### Approach
- Add typed member selectors for composition declaration, own, synced, records and withMember. Use `c.synced(_.member -> action)` as the disambiguated scenario form. `->` is Scala's own key/value arrow: left is always the member or key, right its value; it never means "transition to". `synced`, `own` and `withMember` are core (they name a sync or a member the IR records).
- `after.records(_.member, fact)` is the composition form of the Step helper `records(fact)` fn-112.3 defines (spec R5): the same word, dotted, lowered to `OP_CONTAINS(fact, field(member step, facts))` over the selected member's facts, so a Property reads the same on a machine and on a composition. It is sugar and lives in `model/umpire/Syntax.scala` beside the machine form. Do not introduce a second spelling (`contains`, `recorded`, infix `records`).
- Preserve sync/member order. `withMember` recomputes the selected member's `replaces` target from the new machine's direct refinement instead of copying the base target; reject missing/non-field selectors, incompatible members/refinements and zero/multiple sync matches at the selector.
- **`Declares[S]`** (spec API Contracts "Shared claims"): `property` and `scenario` are two separate extensions today (`model/umpire/Claims.scala:80-86`, one on `Machine[S, O, F]`, one on `Composition[S]`) and `trait Model` has no type parameter, so one def cannot declare on both. Add the typed supertype `Declares[S]` in `Machine.scala`, extended by `Machine[S, ?, ?]` and `Composition[S]`, move `property`/`scenario` onto it, and let `fold`'s `property`/`scenario` cases recognize `umpire.Declares` as they recognize `umpire.Machine` and `umpire.Composition` (`isNamed`). Core, not sugar.
- **Claim patterns** (spec API Contracts "Claim patterns"; `.plans/TEMPORAL_PATTERNS.md` section 2): add `once(over).keeps(projection)`, `never(to)`, `never(to).from(before)`, `stays(p)` and `stays(p).unless(release)` on `PropertyBuilder` as plain `def`s (words, dotted, no `inline`, no new symbol) in `model/umpire/Syntax.scala`, each documented with the `holdsAcross`/`holds` lambda it stands for; `Claims.scala` imports nothing from it. Match them in `model/lifter/Syntax.scala`, reached from `fold` through one hook beside `holds`/`holdsAcross` (`model/lifter/Claims.scala:179`), by lifting the author's lambdas with `stepFunction` and synthesizing the Property function from existing `Expr` nodes: `once/keeps` to `transition = true`, `or(not(over(before)), eq(field(after.state, x), field(before, x)))`; `never/from` to `transition = true`, `or(not(from(before)), not(to(after)))`; `never(p)` alone to the same-step `holds(not p)`; `stays/unless` to `transition = true`, `or(not(p(before)), or(p(after.state), release(after)))`, `stays(p)` alone without the last disjunct. No new IR node. The `Patterns.scala` fixture declares each pattern beside its lambda spelling and the expected JSON proves the trees equal. Refuse at its line a pattern chained after `when` (a transition Property takes no `when`) and a `keeps` projection that is not a field path. The monitor pattern `sticky` is fn-114's (both uses are in the close policy); do not add it here.
- **Function-valued arguments** (spec API Contracts "Shared claims"; `.plans/SEMANTIC_PROTOCOLS.md` section 2 "What is missing (a)"): `fold`'s `Apply(fn, args) if isFunction` case binds a def's arguments by folding them, so `terminal(before)` inside a def whose `terminal: S => Boolean` is a parameter is refused today (`callee` resolves only top-level `DefDef`s, `Expressions.scala:132`). Bind an argument that is a reference to a lifted top-level def (or a def on a vocabulary object) into the fold env as a function reference, and let `callee` resolve a call of such a parameter through the env to that def's lifted function. Refuse a lambda literal in that position at its line, naming the def to write. One lifting fixture, one refusal fixture. This is core lifter machinery, not sugar matching.
- **R4 shared claims as parameterized defs** (spec R4): write `notAdmittedWhilePaused`, `atMostOneActive` and `terminalStays` as top-level defs over `Declares[S]` taking the state-dependent parts as function parameters (`paused`, `running`, `terminal: S => Boolean`), not closures over `AdmissionState`, with bodies `never(s => running(s.state)).from(paused)`, `never(_.state.active == Active.two)` and `once(terminal).keeps(_.phase)`; one definition declares the Property on a machine and on a composition over the member projection. Task 7 applies them to the feature; this task proves the shape on fixtures. The explicit-name form keeps each instance's frozen name.
- Cover separator-bearing names and identically spelled domains so member/action identity is injective rather than string-joined.
## Acceptance
- [ ] All typed composition operations lower to existing member/sync/action keys with original order and identities.
- [ ] withMember fixtures cover ordinary providers replacing dispatchQueue and the lossy provider replacing dispatchQueueUnderStorageLoss, with exact original metadata for both.
- [ ] Ambiguous syncs require member qualification and every invalid selector/replacement has a located refusal fixture.
- [ ] `Declares[S]` is the supertype `property` and `scenario` are declared on, recognized by the lifter; one property definition over it produces the original machine and composition Property rows without string keys.
- [ ] `after.records(_.member, fact)` lifts to the same `OP_CONTAINS` over the member's facts that the machine form `records(fact)` uses, with one lifting fixture and one refusal fixture (selector naming no member); no other fact-recorded spelling exists in `umpire`.
- [ ] `once`, `keeps`, `never`, `from`, `stays` and `unless` live in `model/umpire/Syntax.scala` with their lambda form documented and are matched in `model/lifter/Syntax.scala`; they lower to the existing Property IR (`transition` set for the two-state forms, `never(p)` alone same-step) with bodies built only from `or`, `not`, `==`, `field` and calls; the `Patterns.scala` fixture declares each form once on a machine and once on a composition over a member projection beside its lambda spelling, including `never(p)` without `from` and `stays(p)` without `unless`, and the expected JSON proves IR equality; `Rejects.scala` covers a pattern after `when` and a non-field-path `keeps` projection.
- [ ] A function-valued argument referencing a lifted def binds at lift time and its call inside the def body lifts to that def's function (fixture); a lambda literal in that position is refused at its line (fixture).
- [ ] The three R4 shared claims exist as parameterized top-level defs over `Declares[S]` written with the patterns and produce the original Property rows on a machine and on a composition fixture.
- [ ] Core files import no `Syntax.scala`; task-1 equivalence and focused composition/claim tests pass; tables, IDs, fingerprints, answers and Case bytes equal the task-1 baseline under the R1 projection.
## Done summary
Added typed composition selectors, `Declares[S]`, function-valued argument binding, the level-1 claim patterns and the composition form of `records`. No production Model changed. model/ir, model/cases and every lifts/expected/*.json are byte-identical; rejects.txt gains the new refusals.

**What changed**
- **Core DSL:**
  - `Compose.scala`:
    - typed `compose[S](_.a -> m, ...)` (`@targetName`), typed `sync` and `replaces`;
    - `withMember(_.f -> m)(using Family)`;
    - `c.synced(_.m -> a)` and `c.own(_.m, a)`, both returning `Composed`.
  - `Claims.scala`: `actions((ClassRef | Composed)*)` and `whenAction(Composed)`. `Property` is no longer final.
  - `Machine.scala`: `trait Declares[S]` (type members `Outcome`/`Fact`) carries `property`/`scenario`; `Machine` and `Composition` extend it. The string forms stay, because nexuscaller and the feature still use them until task 7.
- **Sugar (`umpire/Syntax.scala`):**
  - `once(...).keeps(...)`, `never(...)`, `never(...).from(...)`, `stays(...)` and `stays(...).unless(...)` as plain defs on `PropertyBuilder`, plus the classes `Once`, `Never` and `Stays`;
  - `after.records(_.member, fact)`.
  - Each is documented with its `Core form:`.
- **Lifter:**
  - The fold recognises `Declares`.
  - `declaring`/`boundDef` bind function-valued arguments (to a def, or to an eta-expanded or forwarding lambda) and type arguments (`boundFunctions`/`boundTypes`). `typeRef`, `callee` and `stepFunction` resolve them, and one `forwardedDef` extractor serves all three. A lambda literal passed for such a parameter is refused, naming the def to write.
  - `Compositions.scala` lifts the typed forms, `withMember` and the composed keys. Members and actions are resolved by field and Definition ID, not by string.
  - `lifter/Syntax.scala` lowers the patterns through one fold hook, building each body from `or`, `not`, `==`, `field` and calls of the author's predicates.
- **Fixtures:**
  - `Declarations.scala` and `Captured.scala` use the typed forms; their IR is unchanged.
  - `Members.scala` holds typed twins of all seven production compositions (five via `withMember`), the 18 keyed Scenarios and a `whenAction`. They are exactly equal to production; lossy replaces `dispatchQueueUnderStorageLoss`, the others `dispatchQueue`. It also covers separator-bearing names and actions spelled alike.
  - `Patterns.scala` pairs each pattern (machine and composition, def and inline) with its lambda spelling, plus the three R4 shared defs on both.
  - 25 new refusals in `Rejects.scala`.
  - README documents all of this, core and sugar apart.

**Decisions (taken autonomously)**
- **R4 bodies:** over an abstract `S`, the spec's `never(_.state.active == Active.two)` and `keeps(_.phase)` cannot type-check. `atMostOneActive` takes `twoActive: S => Boolean`, and `terminalStays[S, P]` takes `phase: S => P`. `keeps` accepts a field path, or a def whose body is one.
- **`records(_.member, fact)`** lowers to `OP_CONTAINS(text("<field>_<factKey>"), field(after, facts))`, the composed key a composition step records. This is what the current string form lifts to, so task 7's migration keeps the IR. The spec's "field(member step, facts)" does not exist in the IR.
- **`stays(p).unless(r)`** is left-nested, matching how its documented core lambda parses. It means the same as the spec's shape.
- **Patterns after `when`:** every pattern refuses a preceding `when`, including `never(p)` alone.
- **Typed `whenAction(c.synced/own)`** was added; task 7 needs it to remove its `whenAction` strings.
- **Fixtures, not a rewritten Admission fixture:**
  - The R4 shapes are proven by pair tests, not by rewriting lifts/Admission.scala. TestMigrationGoldens, fn-115's migration golden in tools/umpire/{model,lower}, still compares function bodies of model/ir and lifts/expected. So any change to a function body there fails, even though the fn-112 original baseline projects bodies out.
  - **Task 6/7 must extend that projection, or re-capture it, before changing production function bodies.**
- **No `expected/patterns.json`:** the inventory is closed, so the pairs are assertion tests, as in fn-112.2/.3.
- **Derived compositions:** `withMember` lifts its base composition into the same IR, as derived machines do.
- **Parallel work:** two opus subagents built the composition surface and the pattern surface; I built the DSL, the binding core and the integration.

**Review:** claude-opus-5-5 high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus).
- Round 1: SHIP with 4 P3s. Three were fixed in 2fbcea6c88: the parameter refusal is limited to function types, the forwarded-def match is shared, and the missing refusal fixtures and inline twins were added.
- Round 2: SHIP.

**Deferred P3/FYI:**
- `pattern` and `composedKey` are long methods.
- `composedFact` checks every lifted composition of the state type, not just the Property's own.
- Composed keys can still collide as strings. Go's validator refuses shared keys.
- `synced` cannot name a sync whose other side takes inputs.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0fde1e207e, 2fbcea6c88
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), scala-cli test model/lifter (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|MigrationGoldens|Admission' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), make lint-model (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs: