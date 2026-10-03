# Should the Umpire IR's expressions become CEL? (fn-116 spike)

## Answer

**Keep the IR's own expressions.** CEL can express every expression in the checked-in Models, and
both CEL libraries agree with the reader on every call that was measured. The trade still loses:

- **No code is saved.** Adoption would delete about 340 lines of Go evaluator and up to 240 lines
  of validator, and replace the lifter's 450-line `Expressions.scala`. It would add about 960 lines
  of Go (descriptors, value conversion, the cel-go environment and dispatch) and about 1,000 lines
  of Scala translation rules in the lifter. The prototype itself is about 1,770 lines of Go plus
  about 200 lines of dispatch, because it does the translation in Go.
- **The IR gets larger and the reader gets slower.** CEL has no matches, user functions or record
  update, so step functions have to be inlined.
  - **Size.** With positions on both sides (`Expr` positions against CEL's `SourceInfo`), the
    largest step function becomes 1.8–2.3× its `Expr` form plus callees, and 3.7× in the worst case
    (nexus-control). Without positions, which measures the expression structure alone, the ratios
    are 3.4–4.3× and 7.8×.
  - **Time.** Deriving tables with cel-go takes 4.6–6.3× as long cold, and 3.8–5.0× with compiled
    programs reused. Treat this as an upper bound for an adopted design: about a third of
    cel-go-mode time is converting the reader's values to protobuf messages, which an adoption with
    protobuf-native values might avoid. On nexus-close the reader derives its goldens in 44 s, and
    cel-go did not finish in 300 s.
- **`SEMANTICS.md` still owns the hard parts.** Holes, preconditions, step records, catalogs and
  keys stay Umpire's to define. CEL replaces only the 35-line Expressions section, and adds rules
  for its own error handling in `&&`/`||`, lazy bindings and 64-bit overflow.

A subset (Properties and monitor verdicts only) deletes nothing, because step functions would still
need the reader's evaluator, and it adds a second expression language. It also needs most of the
machinery: of the 208 Property, monitor and progress roots in the six Models, 206 use the
enum-to-oneof rule and need the generated descriptors. In nexus-close, 81 of its 138 roots also
need inlined calls (see "The subset option"). Realization operands are a different case: `Operand` is already a small
boolean language that fits CEL as it is. Its consumers are the lowerer and the Case runner, so it
belongs to the parked question about the Case's expression language, not to this spike.

What the spike does show is that **CEL works as an export target**. Use it the way the Quint and P
exports work, not as the IR's own form. The translator here is a working, verified
`Expr`-to-CEL export. If a consumer without the Go reader ever needs to evaluate a Model, that
export is a cheap follow-up and leaves the IR unchanged.

## What was built

All prototype code is in the git-ignored `.flow/tmp/fn116-spike/`. Nothing in the main tree changed
except this report. `git status --short go.mod go.sum` shows no change.

- `go/` is a separate Go module (`fn116spike`). It points at the server module with
  `replace go.temporal.io/server => ../../../..` and adds `cel.dev/cel-go v0.32.0`. From v0.32.0
  on, cel-go's module path is `cel.dev/cel-go`, not `github.com/google/cel-go`.
  - `go/celir/` holds the translator (`translate.go`), the descriptors and value conversion
    (`schema.go`), and the cel-go environment, host functions and evaluation (`eval.go`).
  - `go/tools/umpire/model/` is a copy of the reader, including `internal/checker`, its testdata
    and `tools/umpire/internal/golden`. Import paths are rewritten.
- **How the CEL evaluator sits behind the reader's interface.** The public API exposes no way to
  replace the evaluator, and `internal/` packages cannot be imported from another module, so the
  copy is the workaround:
  - The copy's `eval.go` changes in 5 lines. The reader's `Call`, `Eval` and `apply` become
    `callExpr`, `evalExpr` and `applyExpr`.
  - A new `eval_cel.go` puts all three entry points behind `UMPIRE_EVAL=expr|cel|both`.
  - In `both` mode every call is evaluated by both evaluators, the reader's result is returned, and
    the two results are compared.
  - Everything else in the reader is untouched: `Build`, `Check`, refinement and claims.
  - Three repository-scanning test files were left out: ownership, oracle-boundary and schema.
    (`isolation_test.go` was also left out at first, then copied back because other tests use a
    helper defined in it.)
  - The reader was copied from the working tree at commit `90fb38e2e7`, which had uncommitted
    changes. The copy matches the reader's current files apart from import paths, the 5-line change
    and one comment in `fixtures_test.go` that a later path rename changed.
- `jvm/CelJavaHarness.java` evaluates the exported checked trees with `dev.cel:cel:0.14.0`, using
  the standard runtime and the planner runtime. It loads cel-go's checked trees directly into
  cel-java's runtime, so cel-java's parser and checker were never exercised.
- Model files: Nexus caller is `model/ir/nexus-caller.json`. Standalone activity is
  `model/ir/activity.json`, with the related `activity-system.json` and `activity-race.json`
  measured as well. `nexus-control.json`, `nexus-close.json` and the six lifter fixtures were
  covered as far as stated below.

### Translation rules as implemented

| IR construct | CEL form in the prototype |
| --- | --- |
| Record type | A message. One file, package `m`, names sanitized (`$` becomes `_`). |
| Enum type | A message with one oneof. A case without fields is a `bool` member set to `true`. A case with fields is a nested `Case_<name>` message. |
| Enum literal or constructor | `m.T{c: true}`, or `m.T{c: m.T.Case_c{...}}` |
| Field of an enum case | `x.c.f`. The case comes from static types the translator tracks. |
| `match` | Nested `_?_:_` over `has(x.c)` and equality tests, with bindings as selections of the scrutinee. An exhaustive match drops its last test. A non-exhaustive match ends in the host function `umpire_nomatch`. |
| Call | Inlined by default. Parameters are bound with the `cel.bind` comprehension, or substituted when the argument is a name, a constant or a selection. The callee's precondition becomes `req ? body : umpire_precondition(...)`. Optionally the call is a host function per Model function instead. |
| `copy` | The whole message constructed again, with untouched fields selected from the base. |
| `let` | `cel.bind`, or substitution for a simple value. |
| Hole | The host function `umpire_hole`. |
| Precondition of the evaluated function | A separate checked CEL expression, evaluated first. |
| Lambda (machine and composition `ends`) | A separate expression over its parameters as variables. |
| Step record `{outcome,state,facts,because}` | A CEL map. A `umpire.Step` parameter is `map(string, dyn)`. |
| Channel contents and inbox | A repeated `Delivery_<channel>` field. Empty and full become `size(...)`. FIFO send appends. **Send to an unordered channel has no translation** (see Differences). |

Every translated root passes cel-go's type checker against the generated descriptors.

## Measurements

### Expressions and translation (`out/translation.json`, `logs/translate.log`)

Counts are of IR `Expr` nodes, covering every function body and precondition, start, `ends`,
monitor initial state and Scenario start. "No table rule" means the node is plain CEL: an operator,
conditional, literal, list, record, or a `let` through `cel.bind`. Step records as a map are counted
as plain CEL as well; that row is not in the spec's table.

| Model | Exprs | No table rule | enum→oneof | call | match | copy | lambda | Roots translated | Untranslated |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| nexus-caller | 542 | 346 | 149 | 33 | 7 | 3 | 4 | 49/49 | none |
| activity | 726 | 474 | 197 | 36 | 10 | 4 | 5 | 56/56 | none |
| activity-system | 1,816 | 1,116 | 530 | 102 | 28 | 21 | 19 | 214/214 | none |
| activity-race | 435 | 279 | 109 | 25 | 13 | 6 | 3 | 44/44 | none |
| nexus-close | 2,180 | 1,094 | 932 | 110 | 18 | 17 | 9 | 266/266 | none |
| nexus-control | 303 | 203 | 68 | 25 | 3 | 3 | 1 | 21/21 | none |

- Nexus caller has 3 functions with a precondition. Two of them are also inlined at call sites as
  conditionals. The standalone activity Models have none.
- No Model has a hole, a channel or a non-exhaustive match. That is why `umpire_nomatch` and
  `umpire_hole` were exercised only by the synthetic edge cases.

### Equality with the reader's evaluator (cel-go)

**Calls during `Build` and `Check`.** These ran in `both` mode with inlined calls and CEL's own
`&&`/`||` (`out/<model>/compare.json`, `logs/compare-*.log`).

| Model | States × classes | Calls compared | Distinct calls | Mismatches |
| --- | --- | --- | --- | --- |
| nexus-caller | 4,498 | 23,412 | 4,911 | 0 |
| activity | 6,445 | 35,166 | 7,546 | 0 |
| activity-system | 9,325 | 64,252 | 13,064 | 0 |
| activity-race | 495 | 2,841 | 693 | 0 |
| nexus-control | 4,608 | 14,435 | 4,811 | 0 |
| nexus-close (tables only, `SPIKE_CHECK=0`) | 624,960 | 685,575 | 349,567 | 0 |

- **What the calls cover.** `Build` calls each bound step function once per state and class, and
  each start, `ends`, evidence and refinement function. `Check` adds every Property, monitor,
  Scenario start and refinement read made by every Query, with each witness replayed.
- **The other two translation variants also match exactly** on nexus-caller, activity,
  activity-race and nexus-control: `&&`/`||` as conditionals, and calls as host functions
  (`compare-strict.json`, `compare-host.json`).
- **Negative control.** The comparison detects differences when they exist: on the channels fixture
  it reports 756 mismatches, where the prototype has no translation for one construct
  (`logs/compare-channels.log`).
- **Functions that `Build` and `Check` never call** were evaluated on every assignment of their
  parameter catalogs, with 0 mismatches (`logs/uncovered.log`). They are `admissionEnds` and
  `queueEnds` in activity-system, and `admissionEnds` and `atMostOneActiveAttempt.violated` in
  activity-race.

**Derived artifacts from cel-go alone (`UMPIRE_EVAL=cel`).** These ran against the fn-115 reader
goldens in `tools/umpire/model/testdata/migration`, compared byte for byte by `TestSpikeGoldens`,
one input per run (`out/goldens/*.json`, `logs/goldens-*.log`). The goldens cover tables, evidence,
state fields, Definition IDs, refinement rows, Property rows on every row, and every receipt (Query
answers with witnesses).

- **Default translation (inlined calls, CEL's own logic).** 11 of the 12 golden inputs ran: 44
  entries, 43 equal. The one difference is `semantics/.../channels.json`; see Differences.
- **The other two variants** give equal goldens for nexus-caller and activity.
- **Negative controls.** Two sabotages show that CEL results reach the goldens:
  - `SPIKE_SABOTAGE=1` negates every Boolean a CEL function returns, which covers Properties,
    monitor verdicts and `ends`. 2 of nexus-caller's 4 entries differ (`logs/goldens-sabotage.log`).
  - `SPIKE_SABOTAGE=steps` drops the last step record of every non-empty list a step function
    returns. 2 of nexus-caller's 4 entries differ, and 3 of activity's 4
    (`logs/goldens-sabotage-steps.log`).

  The expr-mode baseline in the copy passes (`logs/goldens-expr-baseline.log`).
- **What remains unverified:**
  - nexus-close's goldens and Queries. With cel-go its golden run did not finish within a 5-minute
    cap at `GOMAXPROCS=2`. The reader's own evaluator finishes the same run, under the same cap, in
    44 s with all 4 entries equal (`logs/goldens-expr-nexus-close.log`). The side-by-side run with
    `Check` was stopped for machine load. Only nexus-close's tables were compared: step functions,
    starts, `ends`, evidence and refinement maps. CEL never evaluated 96 of its functions: all 69
    Property functions, the 8 monitor `next`/`violated` functions, and 19 close-policy helpers
    reached only from those (`out/nexus-close/compare.json`, `NeverEvaluated`).
  - The lowerer goldens in `tools/umpire/lower/testdata/migration`: Case Programs, Contracts,
    fingerprints and identities. The lowerer imports the reader by module path and was not copied.
  - The copied reader's full test suite in `cel` mode.

### cel-java against cel-go (`logs/cel-java.log`)

Each distinct call above was exported with its checked tree (`out/<model>/java/`) and evaluated with
cel-java. The results were then compared line by line with cel-go's.

| Model | Calls | cel-java standard runtime | cel-java planner runtime |
| --- | --- | --- | --- |
| nexus-caller | 4,911 | 0 differences | 0 differences |
| activity | 7,546 | 0 | 0 |
| activity-system | 13,064 | 0 | 0 |
| activity-race | 693 | 0 | 0 |
| nexus-control | 4,811 | 0 | 0 |
| synthetic edge cases | 10 | 0 | not run |

This tests cel-java's runtime on cel-go's checked trees. It does not show that cel-java's own
checker accepts the trees, which a JVM lifter emitting CEL would need.

**One pitfall.** If cel-java builds its own descriptors from the `FileDescriptorSet`, 1,017 of
nexus-caller's 4,911 results differ (1,890 `diff` lines;
`actual-cel-java-standard-own-descriptors.txt`).

- **Cause.** protobuf-java message equality requires the same `Descriptor` object, so an input
  message never equals a message that CEL constructs unless the runtime is given the caller's
  `FileDescriptor` objects.
- **What the differing lines look like.** They fall into three kinds: 466 step lists that came back
  empty, 107 Booleans that came back `false`, and 444 results from the other branch of a
  conditional. Each is what a false message equality produces.
- **This is a wiring concern for any protobuf-java consumer,** not a semantic difference between the
  two libraries.

### Sizes (`out/sizes.json`)

Sizes are compact ProtoJSON bytes. `Expr` sizes are given with and without positions. The CEL tree
holds no positions; they live in its `SourceInfo`.

**Largest step function.** "With callees" means the step function plus each function it calls,
once each.

| Model | Function | Expr (no pos.) | Expr with callees (no pos.) | CEL inlined | + SourceInfo | CEL with host calls | CEL nodes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| nexus-caller | `Protocol$.handlerReplyStep` | 6,851 (3,059) | 9,596 (3,829) | 16,393 | 21,623 | 5,623 | 238 |
| activity | `protocolControlStep` | 8,586 (3,846) | 11,846 (4,973) | 16,707 | 21,773 | 7,464 | 242 |
| nexus-control (worst) | `Control$.forgedComplete` | 1,670 (648) | 10,682 (4,051) | 31,780 | 39,474 | 1,056 | 464 |

- **With positions on both sides** (CEL plus `SourceInfo` against `Expr` with positions), inlined
  CEL is **2.3×** (nexus-caller), **1.8×** (activity) and **3.7×** (nexus-control) what the IR holds
  today for the step function and its callees. This is the measure of IR growth.
- **Without positions** the ratios are 4.3×, 3.4× and 7.8×. This measures the expression structure
  alone. CEL's positions are cheaper than the IR's because they are a map of node ids to offsets, not
  a message on every node. A side table naming the file per node would add to them.
- With calls as host functions, the CEL tree without positions is 1.8–1.9× the function's own
  position-free body. The extra comes from constructing enum cases as messages and from an id on
  every node and entry.

**Whole Model, all functions.**

| Model | Expr (no pos.) | CEL inlined | + SourceInfo | CEL host calls |
| --- | --- | --- | --- | --- |
| nexus-caller | 63,441 (26,851) | 84,690 | 124,021 | 50,381 |
| activity | 87,008 (36,819) | 106,461 | 161,049 | 67,987 |

With positions, inlining makes the IR's expression payload 2.0× (nexus-caller) and 1.9× (activity)
its current size. Without positions it is 3.2× and 2.9×.

### Lines of Go (`out/funclines.tsv`, `wc -l`)

| | Lines |
| --- | --- |
| Reader evaluator adoption deletes (`eval.go`: `env`, `Call`, `apply`, `Eval`, `eval`, `evalAll`, `field`, `construct`, `copy`, `unary`, `binary`, `match`, `bindPattern`, `indexOf`; `channels.go`: `inbox`) | 339 of 874 |
| Validator expression half (`validate.go`: `expr`, `operator`, `inbox`, `call`, `construct`, `match`, `pattern`; these are scope, arity, field and operator checks that CEL's checker replaces) | 151 of 1,178 |
| Validator returns-steps check (`returnsSteps`, `binaryReturns`, `spellSteps`, `spellValue`), replaced only if step records became typed messages | 88 more |
| Prototype translator (`celir/translate.go`) | 1,005 |
| Descriptors and value conversion (`celir/schema.go`) | 408 |
| cel-go environment, host functions and evaluation (`celir/eval.go` up to `Quote`) | 354 |
| Reader dispatch (`eval_cel.go`, excluding the comparison recorder) | about 200 |

`Value`, `Key`, `conforms`, `fieldNames` and `literal` stay: tables, catalogs and the validator's
declaration checks use them.

- **In the prototype**, translation happens in Go from `Expr` trees.
- **In an adoption**, the 1,005 lines of translation would move to the lifter (Scala), which has the
  static types this translator has to reconstruct. The descriptors, conversion, environment and
  dispatch (about 960 lines) would stay in Go.

### Time to derive tables (`out/timing.json`, `logs/timing.log`)

Measured with 7 runs each, under `GOMAXPROCS=2` and `nice -n 15` for both evaluators, while another
worker's validation ran on the same machine. Absolute numbers are inflated; the ratios are
comparable.

| Model | Build median, expr / cel-go | Check median, expr / cel-go | Warm Build min, expr / cel-go |
| --- | --- | --- | --- |
| nexus-caller | 10.8 / 58.0 ms | 32.4 / 151.3 ms | 9.5 / 46.9 ms |
| activity | 17.2 / 91.6 ms | 70.8 / 249.5 ms | 16.7 / 77.7 ms |
| activity-race | 1.8 / 11.3 ms | 5.4 / 28.4 ms | 1.1 / 5.4 ms |
| nexus-control | 11.3 / 52.5 ms | 32.8 / 112.3 ms | 10.6 / 40.9 ms |

- "Warm" reuses an interpreter whose CEL programs are already compiled.
- The strict-logic and host-call variants are within ±10% of inlined cel-go.
- **Where cel-go mode spends its time.** A CPU profile on activity (`out/profile/cpu.out`) puts
  cel-go's own `Eval` at about 45% of cel-go-mode time. Converting arguments to dynamic protobuf
  messages takes about 30%, compiling about 10%, and converting results back about 5%.
- **The ratio is an upper bound for an adopted design.** About a third of cel-go-mode time goes on
  converting values: 30% converting arguments and 5% converting results. The prototype needs that
  because the reader keeps its own value type; an adoption with protobuf-native state might not.
  The cold Build ratios are 4.6–6.3×; with compiled programs reused they are 3.8–5.0×.
- **nexus-close.** The golden run with the reader's evaluator finishes in 44 s. With cel-go it did
  not finish within 300 s under the same limits, so the ratio there is above 6.8×.
- **cel-java** evaluated nexus-caller's 4,911 distinct calls in about 0.19 s on a warm second pass.
  That is not a table derivation, so it is not compared.

### Dependencies (`out/deps-*.txt`, `out/deps-gomod.diff`)

These were measured with `go get cel.dev/cel-go@v0.32.0` on a copy of the server's `go.mod` and
`go.sum`.

- **Module build list.** One module is added (`cel.dev/cel-go v0.32.0`) and `antlr4-go/antlr/v4`
  moves from v4.13.0 to v4.13.1. The build list already holds `github.com/google/cel-go v0.25.0`,
  which the main module does not need, and `cel.dev/expr`.
- **Linked packages.** The reader's build gains 40 non-standard-library packages:
  - 23 from `cel.dev/cel-go`
  - antlr, `go.yaml.in/yaml/v3`, `golang.org/x/exp` and `cel.dev/expr`
  - 2 from `genproto/googleapis/api`
  - 11 more from `google.golang.org/protobuf`

  `out/deps-new-pkgs.txt` lists them. The server binary (`cmd/server`) links none of them today, and
  the reader is tooling, so the server binary would not change.

### The subset option (`out/subset.json`, `logs/subset.log`)

These are the roots a "Properties and monitors only" subset would carry, each with its callees
inlined. "Needs descriptors" means the checked tree names a message type. "Rules" counts roots that
need each rule of the spec's table.

| Model | Property roots | Monitor roots (next / violated / after) | Progress roots | Roots that are plain CEL | Need descriptors | Need inlined calls | Need match |
| --- | --- | --- | --- | --- | --- | --- | --- |
| nexus-caller | 7 | 0 | 0 | 0 | 7 | 0 | 0 |
| activity | 11 | 0 | 0 | 0 | 11 | 1 | 1 |
| activity-system | 39 | 5 (2 / 2 / 1) | 0 | 0 | 44 | 16 | 2 |
| activity-race | 2 | 5 (2 / 2 / 1) | 0 | 0 | 7 | 2 | 2 |
| nexus-close | 110 | 8 (4 / 4 / 0) | 20 | 2 | 136 | 81 | 56 |
| nexus-control | 1 | 0 | 0 | 0 | 1 | 0 | 0 |

- **Every root but two needs the descriptors.** A Property compares the step's state or facts with
  enum cases, so it needs the enum-to-oneof encoding. The two exceptions are nexus-close monitor
  verdicts.
- **What a subset would still have to build.** The descriptor build and value conversion
  (`schema.go`, 408 lines) would be needed even for the subset, and so would the inlining and match
  rules for about half the nexus-close roots.
- **Realization operands.** The realizations hold 26 `Operand` nodes in nexus-caller, 99 in activity,
  67 in activity-race and 28 in nexus-control. `Operand` has 11 cases: literal, environment, run,
  learned value, projected, path, present, equal, all, greater and not. Each maps directly onto a
  CEL variable, selection, `has()`, `==`, `&&`, `>` or `!`, with no rule from the table. Its values
  are Temporal API protobuf messages, not the Model's types.

## Differences found

- **Send to an unordered channel** (`model/lifter/testdata/lifts/Channels.scala:39`,
  `flashStep` on channel `radio`) has no translation in the prototype. It inserts at the delivery's
  catalog position, which the translator cannot compute without the catalog. A host function, or the
  catalog order as a constant list in the tree, would supply it. Without a translation, 756 of 2,824 calls of the channels
  fixture fail, and its `semantics` golden differs. No checked-in Model uses channels.
- No other difference was found on any Model, Query or golden that ran.

## Edge cases (`out/edges.json`, `logs/edges.log`, `logs/fallible.log`)

- **64-bit integers.**
  - Every bounded counter in the Models stays in range. A result outside a counter's range is still
    rejected by the reader's domain check after evaluation ("lands in …, which is outside the state
    domain"). That message comes from `result`, not from the evaluator, so it is unchanged.
  - The behaviors differ only at int64 overflow: `x + MaxInt64` at `x = 1` wraps to
    `-9223372036854775808` in the reader and is `integer overflow` in cel-go and cel-java.
  - No Model is near int64. Each Model has at most 2 additions, all on bounded counters.
- **Error absorption in `&&` and `||`.** These outcomes are with CEL's own operators, where the left
  operand fails and the right decides:
  - `req(-1) == 0 && false` gives `false` in cel-go and cel-java.
  - `req(-1) == 0 || true` gives `true`.
  - `hole && false` gives `false`.
  - The reader reports the precondition error or the hole.
  - Translating `a && b` as `a ? b : false` (`StrictLogic`) restores the reader's result, at no
    measurable cost.

  **Can a Model tell the difference?** A static scan of all six Models found **no** `&&` or `||`
  whose left operand can fail (a hole, a non-exhaustive match, a call with a precondition, or
  arithmetic). Both variants also gave identical results on every call measured. So no current
  Model can tell, but a future one with a hole could.
- **Lazy bindings.**
  - `cel.bind` is documented as "not guaranteed to be evaluated before use". A `let` or an inlined
    argument whose value fails and is never used gives `5` in CEL, where the reader reports the
    error. Calls as host functions stay strict.
  - The scan found 3 argument sites whose value can fail (`Nexus.scala:237`, `:248`,
    `standaloneactivity/Model.scala:330`). Each feeds a parameter its callee always uses, and the
    side-by-side runs show no difference.
- **Source positions.**
  - CEL's `SourceInfo` holds one `location` and per-node offsets into one text. The prototype
    encodes each node's IR line as its offset, and cel-go reports the failing node's id
    (`types.Err.NodeID`). For example, an overflow in the outer addition is reported at
    `Edge.scala:10`, not at the inner addition's line 9.
  - Inlining crosses files, and `SourceInfo` cannot name a second file. An overflow inside an
    inlined callee from `Other.scala:41` reads as `Edge.scala:41` from `SourceInfo` alone. The
    prototype's side table (node id to IR position) gives the right `Other.scala:41`.
  - Model errors the reader raises itself (precondition, no case matches, hole) are reported at
    their own IR positions, because the positions are constant arguments of the host functions.
  - Adoption needs that side table, or one CEL tree per source file.
- **IR growth from inlining** is measured above. With positions, the largest step function grows
  1.8–2.3×, and 3.7× in nexus-control. Without positions the ratios are 3.4–4.3× and 7.8×.

## Adoption cost beyond the prototype

- **Lifter.** The lifter needs a CEL emitter in place of `Expressions.scala` (450 lines). It would
  carry every rule above: oneof encoding for enum literals, constructors and fields; match to
  conditionals with exhaustiveness; inlining with precondition guards and hygienic renaming (or
  host-function declarations); `copy` to full construction; step records; lambdas; and channels,
  with the unordered send as a host function. It also has to emit descriptors (or the IR's types
  as before, with descriptors derived in Go) and a position side table.
- **Quint and P exports.** Their expression writers (`quint.go` lines 350–754, about 405 lines;
  `p.go` lines 273–505, about 233 lines) would have to read CEL trees. After inlining those trees
  have no calls, matches or `copy` left, so the exported specifications would lose their structure:
  helper functions and pattern matches become nested conditionals and `has()` tests.
- **`SEMANTICS.md`.** The Expressions section (35 of about 695 lines) would point to CEL. New sections
  would define the message encoding of types, `&&`/`||` strictness, binding strictness, overflow
  and the host functions. Holes, preconditions, steps, catalogs, keys and admission stay.
- **IR schema.** The schema needs a CEL expression message in place of `Expr` in `Function`,
  `Machine.starts`/`ends`, `Monitor.initial`, `Composition.ends` and `Scenario.start`. It also needs
  descriptors or a rule for deriving them, and a position table. `Expr`, `Pattern`, `Value` (as an
  expression literal) and the validator's expression half would go. Two Go sites that construct
  `Expr` trees would construct CEL trees instead: `tools/umpire/internal/golden/job.go` builds a
  synthetic Model's expressions, and `tools/umpire/export/checked.go` builds a literal `true` body.
- **Regeneration.** Every checked-in IR file would change: the 6 under `model/ir` and the lifter's expected
  fixtures. So would the reader's migration goldens (inputs and anything keyed on IR bytes) and the
  Cases and identities derived from IR content.

## Not measured, and why

- **nexus-close.** Only its tables were compared (624,960 pairs, 0 mismatches).
  - CEL never evaluated 96 of its functions: all 69 Property functions, 8 monitor functions and 19
    close-policy helpers. For this Model, R2 is verified for step functions, starts, `ends`, evidence
    and refinement maps only.
  - The cel-go golden run did not finish within 5 minutes on two processors. The reader's own
    evaluator finishes the same run in 44 s.
  - The side-by-side run with `Check` was stopped because of machine load.
- **The lowerer goldens** (Case Programs, Contracts, fingerprints and identities) and the copied
  reader's full test suite under `UMPIRE_EVAL=cel` were not run.
- **Realization operands** (`Operand`) and the Testpilot Case's expression language were not
  translated. Both belong to the Case side; see "The subset option".
- **Holes, non-exhaustive matches and precondition failures** occur in no checked-in Model. They were
  exercised through the synthetic edge Model only, where cel-go and cel-java agree on all 10 calls.

## Where the prototype lives and how to rerun it

Everything is under `.flow/tmp/fn116-spike/`. `run.sh` runs Go niced on two processors, and
`run-java.sh` runs the JVM harness with two processors and a 2 GB heap.

```sh
cd .flow/tmp/fn116-spike
./run.sh test -tags test_dep -p 1 -count=1 ./tools/umpire/model -run '^TestSpikeTranslate$' -v
SPIKE_MODELS=nexus-caller,activity ./run.sh test -tags test_dep -p 1 -count=1 ./tools/umpire/model -run '^TestSpikeCompare$' -v
UMPIRE_EVAL=cel SPIKE_GOLDEN=ir/nexus-caller ./run.sh test -tags test_dep -p 1 -count=1 ./tools/umpire/model -run '^TestSpikeGoldens$' -v
./run-java.sh ../out/nexus-caller/java standard && diff out/nexus-caller/java/expected-cel-go.txt out/nexus-caller/java/actual-cel-java-standard.txt
./run.sh test -tags test_dep -p 1 -count=1 ./tools/umpire/model -run '^TestSpike(Sizes|Timing|Edges|Fallible)$' -v
```

Variants are selected with environment variables:

- `UMPIRE_CEL_LOGIC=strict` translates `&&`/`||` as conditionals.
- `UMPIRE_CEL_CALLS=host` makes calls host functions.
- `SPIKE_CHECK=0` compares tables only.
- `SPIKE_EXPORT=0` skips the cel-java export.
- `SPIKE_MODELS` names the Models to run.
- `SPIKE_SABOTAGE=1` runs the negative control.

Results land in `out/` and logs in `logs/`.
