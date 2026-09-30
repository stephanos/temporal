## Umpire, the Nim surface.
##
## The Model files author against this module: `Step`, `Machine`, finite enumeration, the
## refinement check, `Property`, `Scenario`, `Limits`, `Query`, `Set`, composition, and the macros
## that give a Model file its Lean-shaped surface. Bodies that belong to the search engine are
## sketched with a comment; what this file shows is how a Model binds to them and where each check
## runs.
##
## Two rules make the design work:
##
## 1. Everything that computes is a plain proc over plain values. A plain proc runs in the
##    compile-time VM inside a `const` or a `static:` block exactly as it runs in a test, so the
##    state table, the refinement walk and the Query search all happen while the Model compiles.
## 2. The macros only rearrange syntax. Nim's own parser turns `machine nexusProduct:` followed by
##    an indented block of `key: value` lines into `Command(machine, nexusProduct, StmtList(...))`,
##    where each line is `Call(key, StmtList(value))`. A macro reads that tree, checks the keys,
##    and emits a `const`; a value the VM computed is then handed to `admit`, whose `static`
##    parameter lets a rejection be pinned to the author's own node with `error(msg, node)`.
##
## Five of the spec's field names are Nim keywords: `for`, `in`, `from`, `when`, `bind`. The DSL
## accepts them stropped (`` `for`: operation ``), which is legal Nim and the price of keeping the
## Lean spelling; `fieldsOf` unwraps the backticks.

import std/[macros, macrocache, options, sequtils, strutils, algorithm, tables]

# ── Vocabulary ────────────────────────────────────────────────────────────────────────────────────

type
  Party* = enum
    caller, handler, worker, network, operator
    system  ## reserved: the server, which owns the timers

  ActionKey* = string
    ## One action class: the action's name followed by the keys of its inputs, `-`-joined, so
    ## `handlerReply(handlerError(true))` is `"handlerReply-handlerError-true"` and a nullary timer
    ## is its own name. The same spelling the Lean tables use, so fixtures compare side by side.

  Step*[S, O, F] = object
    ## One row's right-hand side. A step function `S -> inputs -> seq[Step]` returns `@[]` when the
    ## action is not enabled in `S`.
    outcome*: O
    state*: S
    facts*: seq[F]

  Row*[S, O, F] = object
    source*: S
    action*: ActionKey
    step*: Step[S, O, F]

  StateTable*[S, O, F] = object
    states*: seq[S]
    rows*: seq[Row[S, O, F]]

  Refinement* = object
    ## Every protocol row keyed `<state key>-<action key>`, with the product step it maps to, or
    ## `none` when the mapped source and target agree: a product stutter.
    rows*: seq[(string, Option[ActionKey])]
    rejected*: Option[string]

  Machine*[S, O, F] = object
    name*, entity*: string
    starts*, ends*: seq[S]
    timers*, unobservable*: seq[string]
    evidence*: seq[(string, string)]  ## fact name -> recorded event or observation name
    catalog*: seq[ActionKey]          ## every class the machine steps on, in canonical order
    table*: StateTable[S, O, F]
    refinement*: Option[Refinement]

# ── Finite enumeration ────────────────────────────────────────────────────────────────────────────

type Finite* = concept type T
  ## A state or input type the table builder can enumerate. Enums and ranges are finite by
  ## definition; an object is finite when `finite` derived its `enumerate` from its fields.
  enumerate(T) is seq[T]

proc enumerate*[T: Ordinal](_: typedesc[T]): seq[T] =
  ## `low..high` is the whole point of using enums and `range[0..n]` for state fields: the
  ## compiler already knows the domain, so nothing is listed twice.
  toSeq(low(T) .. high(T))

proc key*[T: Ordinal](x: T): string = $x

proc key*[T: object](x: T): string =
  ## A state's key lists its fields in declaration order (`"scheduled-0-unset-unset-unset"`); a
  ## variant's key is its kind followed by the active branch (`"handlerError-true"`). `fieldPairs`
  ## only visits the active branch of a `case kind` object, which is what makes this one proc.
  var parts: seq[string]
  for _, value in fieldPairs(x):
    parts.add key(value)
  parts.join("-")

macro finite*(T: typedesc): untyped =
  ## Derives `enumerate(typedesc[T])` for an object type as the cartesian product of its fields'
  ## enumerations, in declaration order, so the first value has every field at its first value. A
  ## `case kind` object contributes, per kind, the product of that branch's fields:
  ## `handlerError(retryable: bool)` is one constructor and two members, which is the granularity
  ## an example is written at.
  let sym = T.getType[1]
  let objTy = sym.getImpl[2]
  objTy.expectKind nnkObjectTy
  let recList = objTy[2]
  let value = ident"value"

  proc loops(fields: NimNode, bound: seq[(NimNode, NimNode)]): NimNode =
    ## Nested `for` loops over `fields`; the innermost adds the object built from `bound`.
    if fields.len == 0:
      var ctor = newTree(nnkObjConstr, sym)
      for (name, v) in bound: ctor.add newColonExpr(name, v)
      return quote do: `value`.add `ctor`
    let head = fields[0]
    var rest = newTree(nnkRecList)
    for i in 1 ..< fields.len: rest.add fields[i]
    case head.kind
    of nnkIdentDefs:
      let name = head[0].basename
      let inner = loops(rest, bound & @[(name, name)])
      let fieldTy = head[1]
      result = quote do:
        for `name` in enumerate(`fieldTy`): `inner`
    of nnkRecCase:
      # `case kind: Kind` then one branch per constructor; a branch's fields nest inside it.
      let kind = head[0][0].basename
      let kindTy = head[0][1]
      var caseStmt = newTree(nnkCaseStmt, kind)
      for i in 1 ..< head.len:
        let branch = head[i]
        var fieldsOfBranch = newTree(nnkRecList)
        for def in branch[^1]:
          if def.kind == nnkIdentDefs: fieldsOfBranch.add def
        let body = loops(fieldsOfBranch, bound & @[(kind, kind)])
        caseStmt.add (if branch.kind == nnkOfBranch: newTree(nnkOfBranch, branch[0], body)
                      else: newTree(nnkElse, body))
      result = quote do:
        for `kind` in enumerate(`kindTy`): `caseStmt`
    else:
      error("finite: unsupported field", head)

  let body = loops(recList, @[])
  result = quote do:
    proc enumerate*(_: typedesc[`sym`]): seq[`sym`] =
      var `value`: seq[`sym`]
      `body`
      `value`

proc saturatingSucc*[T: Ordinal](x: T): T =
  ## The attempt count is bounded by `attemptBound`; a retry past the bound stays at it.
  if x == high(T): x else: succ(x)

# ── Tables, reachability, refinement ──────────────────────────────────────────────────────────────

proc transitions*[S, O, F](m: Machine[S, O, F]): seq[Row[S, O, F]] = m.table.rows

proc reachable*[S, O, F](m: Machine[S, O, F]): seq[S] =
  ## Breadth-first from `starts` over the table. Not every state in the type is reached: the
  ## protocol's deadline fields are set only by the schedule command.
  result = m.starts
  var i = 0
  while i < result.len:
    for row in m.table.rows:
      if row.source == result[i] and row.step.state notin result:
        result.add row.step.state
    inc i

proc stuck*[S, O, F](m: Machine[S, O, F]): Option[S] =
  ## A non-end state with no row out of it, if any.
  for state in m.reachable:
    if state notin m.ends and not m.table.rows.anyIt(it.source == state):
      return some(state)

proc refinementOf*[S, O, F, PS, PO, PF](
    protocol: Machine[S, O, F], product: Machine[PS, PO, PF],
    map: proc(state: S): PS): Refinement =
  ## Walks every protocol row through `map`. A row whose mapped source and target agree is a
  ## stutter. Otherwise the product must have some row between the mapped states, under any action
  ## class: the row is recorded as the same class when the product steps on it (`handlerReply-async`),
  ## and as the first product row between the states when it does not (a protocol timer is the
  ## product's `timeout`; an activity's retry is the product's `attemptResult-failed-true`). A
  ## protocol row the product cannot account for rejects the refinement, and `admit` pins that to
  ## the `refines:` line.
  for row in protocol.table.rows:
    let source = map(row.source)
    let target = map(row.step.state)
    let rowKey = key(row.source) & "-" & row.action
    if source == target:
      result.rows.add (rowKey, none(ActionKey))
      continue
    let candidates = product.table.rows.filterIt(it.source == source and it.step.state == target)
    let same = candidates.filterIt(it.action == row.action)
    if same.len > 0:
      result.rows.add (rowKey, some(same[0].action))
    elif candidates.len > 0:
      result.rows.add (rowKey, some(candidates[0].action))
    else:
      result.rejected = some("protocol row " & rowKey & " maps " & key(source) & " -> " &
        key(target) & ", which is no product step and no stutter")
      return

# ── Claims, paths, search ─────────────────────────────────────────────────────────────────────────

type
  PropertyKind* = enum sameStep, transition

  Property*[S, O, F] = object
    name*, machine*: string
    case kind*: PropertyKind
    of sameStep:
      trigger*: ActionKey  ## `when:` -- a class, or an action name matching every class of it
      holds*: proc(step: Step[S, O, F]): bool {.nimcall.}
    of transition:
      holdsAcross*: proc(before, after: Step[S, O, F]): bool {.nimcall.}

  Scenario*[S] = object
    name*, model*: string
    starts*: S
    actions*: seq[ActionKey]

  Limits* = object
    steps*, actions*, search*: int

  QueryMode* = enum find, verify

  Query* = object
    name*, property*, scenario*, limits*: string
    mode*: QueryMode
    witness*: seq[ActionKey]  ## `find`: the path on which the claim held; realized by a set
    rejected*: Option[string]

  Purpose* = enum functional, canary, exploratory
  Binding* = enum driven, observed
  Repeat* = enum implementation
  Cover* = enum rows, results, classMembers

  Set* = object
    name*: string
    purpose*: Purpose
    bindings*: seq[(Party, Binding)]
    repeat*: Option[Repeat]
    queries*: seq[string]
    machine*: string
    cover*: set[Cover]
    budget*: string

proc triggers*(prop: ActionKey, action: ActionKey): bool =
  action == prop or action.startsWith(prop & "-")

proc search*[S, O, F](m: Machine[S, O, F], property: Property[S, O, F], scenario: Scenario[S],
    limits: Limits, mode: QueryMode, name: string): Query =
  ## Follows the Scenario's classed actions from its start through the table, branching where an
  ## action has more than one row, within `limits.steps`, and cut at `limits.search` candidates. A
  ## `find` succeeds on the first path whose triggering step satisfies the claim; a `verify` fails on
  ## the first pair of consecutive steps that falsifies it, and otherwise verifies. The witness is
  ## the path's action keys, which is what a set realizes.
  result = Query(name: name, property: property.name, scenario: scenario.name, mode: mode)
  var frontier: seq[(S, seq[Step[S, O, F]])] = @[(scenario.starts, @[])]
  var visited = 0
  for action in scenario.actions:
    var next: typeof(frontier)
    for (state, path) in frontier:
      for row in m.table.rows:
        if row.source == state and row.action == action:
          next.add (row.step.state, path & row.step)
          inc visited
          if visited > limits.search:
            result.rejected = some("search budget " & $limits.search & " exhausted")
            return
    frontier = next
    if frontier.len == 0 or frontier[0][1].len > limits.steps:
      result.rejected = some("no row for " & action & " on the path")
      return
  for (_, path) in frontier:
    case property.kind
    of sameStep:
      for i, step in path:
        if property.trigger.triggers(scenario.actions[i]) and property.holds(step):
          result.witness = scenario.actions
          return
    of transition:
      for i in 1 ..< path.len:
        if not property.holdsAcross(path[i - 1], path[i]):
          result.rejected = some(property.name & " falsified after " & scenario.actions[i])
          return
  if mode == find:
    result.rejected = some(property.name & " never holds on " & scenario.name)
  else:
    result.witness = scenario.actions

# ── Compile-time registries ───────────────────────────────────────────────────────────────────────
#
# The vocabulary commands register what they declare, the way Lean's environment extension does,
# and the commands that reference it read the tables at expansion time. `macrocache` is the one
# compile-time store that survives across modules, which is what lets `nexus_caller.nim` step on
# the `workerStop` that `worker.nim` declared.

const
  entities = CacheTable"umpire.entities"      # name -> (key, refer)
  actions = CacheTable"umpire.actions"        # name -> Bracket of `input: Type` pairs
  observations = CacheTable"umpire.observations"
  machines = CacheTable"umpire.machines"      # name -> state type node
  properties = CacheTable"umpire.properties"
  scenarios = CacheTable"umpire.scenarios"
  limitsDecl = CacheTable"umpire.limits"
  queries = CacheTable"umpire.queries"

proc nameOf(n: NimNode): string =
  ## An identifier, or a stropped keyword such as `` `for` ``.
  if n.kind == nnkAccQuoted: n[0].strVal else: n.strVal

proc fieldsOf(body: NimNode, allowed: openArray[string]): OrderedTable[string, NimNode] =
  ## The `key: value` lines of a colon-block body. A key not in `allowed` is an error pinned to
  ## that line, so a misspelled `evidence:` is caught where it is written.
  if body.kind == nnkNilLit: return
  for line in body:
    if line.kind != nnkCall or line.len != 2 or line[1].kind != nnkStmtList:
      error("expected `key: value`", line)
    let k = nameOf(line[0])
    if k notin allowed:
      error("unknown field `" & k & "`; expected one of " & allowed.join(", "), line[0])
    result[k] = if line[1].len == 1: line[1][0] else: line[1]

proc require(fields: OrderedTable[string, NimNode], key: string, at: NimNode): NimNode =
  if key notin fields: error("missing `" & key & ":`", at)
  fields[key]

proc lookup(table: CacheTable, name: NimNode, what: string): NimNode =
  ## A reference to something a command must have declared earlier in this or an imported module.
  for k, v in table:
    if k == nameOf(name): return v
  error("no " & what & " named `" & nameOf(name) & "` is declared", name)

proc lookupOrTimer(name: NimNode): NimNode =
  ## Timers are actions no `action` command declares; a machine owns them under `timers:`.
  for k, v in actions:
    if k == name.strVal: return v
  newTree(nnkBracket)

proc declared(table: CacheTable, name: string): bool =
  for k, _ in table:
    if k == name: return true

proc actionClass(call: NimNode): NimNode =
  ## `handlerReply(handlerError(true))` -> the expression `"handlerReply-" & key(handlerError(true))`,
  ## typed by Nim like any other call, so a misspelled class is a type error at the author's node.
  if call.kind == nnkIdent:
    discard lookupOrTimer(call)
    return newLit(call.strVal)
  call.expectKind nnkCall
  result = newLit(call[0].strVal)
  for arg in call[1 ..^ 1]:
    result = infix(result, "&", infix(newLit"-", "&", newCall("key", arg)))

proc lambda(arrow: NimNode, paramTy: NimNode): NimNode =
  ## `step => body` or `(before, after) => body`, the `std/sugar` arrow, into a `{.nimcall.}` proc
  ## literal typed by the machine's step type. A nimcall literal can live in a `const`.
  if arrow.kind != nnkInfix or arrow[0].strVal != "=>":
    error("expected `params => predicate`", arrow)
  var params = @[ident"bool"]
  let ps = if arrow[1].kind in {nnkPar, nnkTupleConstr}: toSeq(arrow[1]) else: @[arrow[1]]
  for p in ps: params.add newIdentDefs(p, paramTy)
  result = newProc(params = params, body = arrow[2], procType = nnkLambda,
    pragmas = newTree(nnkPragma, ident"nimcall"))

# ── Node-pinned admission of VM-computed results ──────────────────────────────────────────────────

macro admit*(refinement: static Refinement, at: untyped): untyped =
  ## The VM has already walked the refinement by the time this macro runs; the error, if any, is
  ## reported at the `refines:` line the author wrote.
  if refinement.rejected.isSome:
    error("refinement rejected: " & refinement.rejected.get, at)
  newEmptyNode()

macro admit*(query: static Query, at: untyped): untyped =
  if query.rejected.isSome:
    error("query `" & query.name & "`: " & query.rejected.get, at)
  newEmptyNode()

# ── The commands ──────────────────────────────────────────────────────────────────────────────────

macro entity*(name: untyped, body: untyped = nil): untyped =
  let f = fieldsOf(body, ["key", "refer"])
  entities[name.strVal] = newTree(nnkPar, f.getOrDefault("key", newEmptyNode()),
    f.getOrDefault("refer", newEmptyNode()))
  newEmptyNode()

macro action*(name: untyped, body: untyped = nil): untyped =
  ## Registers the action's party, entity, schema, inputs, results and examples. Only the inputs
  ## are read back: `machine` enumerates a class per assignment of them. The schema is any
  ## expression (`a.b.C | a.b.D` parses as an infix over dotted names) kept as its `repr`.
  let f = fieldsOf(body, ["party", "creates", "on", "schema", "input", "results", "examples"])
  var inputs = newTree(nnkBracket)
  if "input" in f:
    for line in (if f["input"].kind == nnkStmtList: f["input"] else: newStmtList(f["input"])):
      inputs.add newColonExpr(line[0], line[1][0])
  if "creates" in f: discard lookup(entities, f["creates"], "entity")
  if "on" in f: discard lookup(entities, f["on"], "entity")
  actions[name.strVal] = inputs
  newEmptyNode()

macro observation*(name: untyped, body: untyped): untyped =
  let f = fieldsOf(body, ["on", "read"])
  discard lookup(entities, f.require("on", name), "entity")
  observations[name.strVal] = f.require("read", name)
  newEmptyNode()

proc phasesOf(bracket: NimNode, stateTy: NimNode): NimNode =
  ## `[succeeded, failed]` -> `{PhaseOf(S).succeeded, ...}`; every state with one of the phases.
  bracket.expectKind nnkBracket
  result = newTree(nnkCurly)
  for p in bracket: result.add p
  result = newCall(bindSym"phaseStates", stateTy, result)

proc phaseStates*[S](_: typedesc[S], phases: set[typeof(default(S).phase)]): seq[S] =
  enumerate(S).filterIt(it.phase in phases)

proc firstPerPhase*[S](states: seq[S]): seq[S] =
  ## `starts: [unscheduled]` names one state: the phase with every other field at its first value,
  ## which enumeration order puts first.
  for s in states:
    if not result.anyIt(it.phase == s.phase): result.add s

macro machine*(name: untyped, body: untyped): untyped =
  ## `machine <name>:` with `` `for` ``, `state`, `starts`, `ends`, `timers`, `unobservable`,
  ## `evidence`, `steps`, and optionally `refines` + `map`, or `` `from` `` + `restrict` for a
  ## restriction of an existing machine. Emits a `const` whose table the VM builds, and an `admit`
  ## of the refinement pinned to the `refines:` line.
  let f = fieldsOf(body, ["for", "state", "starts", "ends", "timers", "unobservable", "evidence",
    "steps", "refines", "map", "from", "restrict"])
  let m = genSym(nskVar, "m")
  let nameLit = newLit(name.strVal)

  if "from" in f:
    # A restriction keeps the base machine's states and the rows of the named actions only.
    let base = f["from"]
    discard lookup(machines, base, "machine")
    let keep = f.require("restrict", name)
    machines[name.strVal] = machines[base.strVal]
    return quote do:
      const `name`* = restrict(`base`, `keep`)

  let stateTy = f.require("state", name)
  let entityName = f.require("for", name)
  discard lookup(entities, entityName, "entity")
  machines[name.strVal] = stateTy

  let steps = f.require("steps", name)
  var timers: seq[string]
  if "timers" in f:
    for t in f["timers"]: timers.add t.strVal

  # One loop nest per `action: stepFn` line, over every state and every assignment of the action's
  # declared inputs. A step name that is neither a declared action nor one of this machine's timers
  # is the author's mistake, reported at that line.
  var build = newStmtList()
  var catalog = newStmtList()
  var firstStep: NimNode
  for line in steps:
    let actionName = line[0]
    let stepFn = line[1][0]
    var inputs = lookupOrTimer(actionName)
    if inputs.len == 0 and actionName.strVal notin timers and
        not actions.declared(actionName.strVal):
      error("step names an undeclared action `" & actionName.strVal & "`; declare it with " &
        "`action`, or list it under `timers:`", actionName)
    let source = ident"source"
    var call = newCall(stepFn, source)
    var classKey = newLit(actionName.strVal)
    var sampleCall = newCall(stepFn, newCall("default", stateTy))
    for inp in inputs:
      call.add inp[0]
      sampleCall.add newCall("default", inp[1])
      classKey = infix(classKey, "&", infix(newLit"-", "&", newCall("key", inp[0])))
    if firstStep.isNil: firstStep = sampleCall
    var inner = quote do:
      for step in `call`:
        `m`.table.rows.add Row[typeof(`m`.table.states[0]), typeof(step.outcome),
          typeof(step.facts[0])](source: `source`, action: `classKey`, step: step)
    var classLine = quote do:
      `m`.catalog.add `classKey`
    for inp in inputs.reversed:
      let (n, t) = (inp[0], inp[1])
      inner = quote do:
        for `n` in enumerate(`t`): `inner`
      classLine = quote do:
        for `n` in enumerate(`t`): `classLine`
    build.add quote do:
      for `source` in `m`.table.states: `inner`
    catalog.add classLine

  var evidence = newTree(nnkBracket)
  if "evidence" in f:
    for line in f["evidence"]:
      evidence.add newTree(nnkTupleConstr, newLit(line[0].strVal), newLit(line[1][0].strVal))
  let starts = phasesOf(f.require("starts", name), stateTy)
  let ends = phasesOf(f.require("ends", name), stateTy)
  let timersLit = newLit(timers)
  let unobservable = if "unobservable" in f: newLit(f["unobservable"].mapIt(it.strVal))
                     else: newLit(newSeq[string]())

  var refine = newEmptyNode()
  var check = newEmptyNode()
  if "refines" in f:
    let product = f["refines"]
    discard lookup(machines, product, "machine")
    let map = f.require("map", name)
    refine = quote do:
      `m`.refinement = some(refinementOf(`m`, `product`, `map`))
    check = quote do:
      admit(`name`.refinement.get, `product`)
    check.copyLineInfo(product)

  result = quote do:
    const `name`* = block:
      var `m` = machineFor(`nameLit`, `entityName`.astToStr, typeof(`firstStep`))
      `m`.table.states = enumerate(`stateTy`)
      `m`.starts = firstPerPhase(`starts`)
      `m`.ends = `ends`
      `m`.timers = `timersLit`
      `m`.unobservable = `unobservable`
      `m`.evidence = @`evidence`
      `build`
      `catalog`
      sort(`m`.catalog)
      `refine`
      `m`
    `check`

proc machineFor*[S, O, F](name, entity: string, _: typedesc[seq[Step[S, O, F]]]):
    Machine[S, O, F] =
  ## The outcome and fact types are read off the first step function's return type, the way Lean
  ## infers them, so a machine block names only its state type.
  Machine[S, O, F](name: name, entity: entity)

proc restrict*[S, O, F](base: Machine[S, O, F], keep: openArray[string]): Machine[S, O, F] =
  ## The base machine's states, and only the rows and classes of the actions named.
  result = base
  result.table.rows = @[]
  result.catalog = @[]
  for row in base.table.rows:
    for action in keep:
      if action.triggers(row.action): result.table.rows.add row
  for class in base.catalog:
    for action in keep:
      if action.triggers(class): result.catalog.add class

macro property*(name: untyped, body: untyped): untyped =
  ## `machine:`, optional `` `when` ``, and `holds:` as an arrow lambda. With `when:` it is a
  ## same-step claim over `Step`; without, a transition claim over two steps.
  let f = fieldsOf(body, ["machine", "when", "holds"])
  let m = f.require("machine", name)
  discard lookup(machines, m, "machine")
  let stepTy = quote do: typeof(`m`.table.rows[0].step)
  let holds = lambda(f.require("holds", name), stepTy)
  let nameLit = newLit(name.strVal)
  let machineLit = newLit(m.strVal)
  properties[name.strVal] = m
  if "when" in f:
    let trigger = actionClass(f["when"])
    quote do:
      const `name`* = Property[typeof(`m`.table.states[0]), typeof(`m`.table.rows[0].step.outcome),
          typeof(`m`.table.rows[0].step.facts[0])](
        name: `nameLit`, machine: `machineLit`, kind: sameStep, trigger: `trigger`, holds: `holds`)
  else:
    quote do:
      const `name`* = Property[typeof(`m`.table.states[0]), typeof(`m`.table.rows[0].step.outcome),
          typeof(`m`.table.rows[0].step.facts[0])](
        name: `nameLit`, machine: `machineLit`, kind: transition, holdsAcross: `holds`)

macro scenario*(name: untyped, body: untyped): untyped =
  ## `model:`, `starts:` (a phase; or `member.phase` on a composition), `actions:` as a bracket of
  ## classed actions, each typed by Nim through `key`.
  let f = fieldsOf(body, ["model", "starts", "actions"])
  let m = f.require("model", name)
  let stateTy = lookup(machines, m, "machine")
  let start = f.require("starts", name)
  var acts = newTree(nnkBracket)
  for a in f.require("actions", name): acts.add actionClass(a)
  let nameLit = newLit(name.strVal)
  let modelLit = newLit(m.strVal)
  scenarios[name.strVal] = m
  quote do:
    const `name`* = Scenario[`stateTy`](name: `nameLit`, model: `modelLit`,
      starts: startState(`stateTy`, `start`), actions: @`acts`)

macro limits*(name: untyped, body: untyped): untyped =
  let f = fieldsOf(body, ["steps", "actions", "search"])
  let (s, a, r) = (f.require("steps", name), f.require("actions", name), f.require("search", name))
  limitsDecl[name.strVal] = name
  quote do:
    const `name`* = Limits(steps: `s`, actions: `a`, search: `r`)

macro query*(name: untyped, body: untyped): untyped =
  ## `find:` or `verify:` a property `` `in` `` a scenario under `limits:`. The search runs in the
  ## VM while the module compiles; a claim the path never reaches is an error at the `find:` line.
  let f = fieldsOf(body, ["find", "verify", "in", "limits"])
  let mode = if "find" in f: "find" elif "verify" in f: "verify"
             else: (error("a query is `find:` or `verify:`", name); "")
  let prop = f[mode]
  let machineName = lookup(properties, prop, "property")
  let scen = f.require("in", name)
  discard lookup(scenarios, scen, "scenario")
  let lim = f.require("limits", name)
  discard lookup(limitsDecl, lim, "limits")
  let modeSym = ident(mode)
  let nameLit = newLit(name.strVal)
  queries[name.strVal] = name
  result = quote do:
    const `name`* = search(`machineName`, `prop`, `scen`, `lim`, `modeSym`, `nameLit`)
    admit(`name`, `prop`)
  result[1].copyLineInfo(prop)

macro set*(name: untyped, body: untyped): untyped =
  ## `purpose:`, `` `bind` `` lines of `party: driven | observed`, optional `repeat:`, and either
  ## `queries:` or, for an exploration, `machine:`, `cover:` and `budget:`. Every query named must
  ## be declared and must be a `find`; that is checked here rather than at run time.
  let f = fieldsOf(body, ["purpose", "bind", "repeat", "queries", "machine", "cover", "budget"])
  var bindings = newTree(nnkBracket)
  for line in f.require("bind", name):
    bindings.add newTree(nnkTupleConstr, line[0], line[1][0])
  var qs = newTree(nnkBracket)
  if "queries" in f:
    for q in f["queries"]:
      discard lookup(queries, q, "query")
      qs.add newLit(q.strVal)
  var cover = newTree(nnkCurly)
  if "cover" in f:
    var n = f["cover"]
    while n.kind == nnkInfix: (cover.add n[2]; n = n[1])
    cover.add n
  let purpose = f.require("purpose", name)
  let repeatOpt = if "repeat" in f: newCall("some", f["repeat"]) else: quote do: none(Repeat)
  let machineLit = if "machine" in f: newLit(f["machine"].strVal) else: newLit""
  let budget = if "budget" in f: newLit(f["budget"].strVal) else: newLit""
  let nameLit = newLit(name.strVal)
  quote do:
    const `name`* = Set(name: `nameLit`, purpose: `purpose`, bindings: @`bindings`,
      repeat: `repeatOpt`, queries: @`qs`, machine: `machineLit`, cover: `cover`, budget: `budget`)

macro compose*(name: untyped, body: untyped): untyped =
  ## A product of machines of different entities. `members:` names each member's field of the
  ## composed state type; `sync:` pairs two member actions (`a.x || b.y`) that fire as one row;
  ## every other member action steps its own field alone. `starts:`/`ends:` name member phases.
  let f = fieldsOf(body, ["for", "state", "members", "sync", "starts", "ends"])
  let stateTy = f.require("state", name)
  var members = newTree(nnkBracket)
  for line in f.require("members", name):
    discard lookup(machines, line[1][0], "machine")
    members.add newTree(nnkTupleConstr, newLit(line[0].strVal), line[1][0])
  var syncs = newTree(nnkBracket)
  if "sync" in f:
    for line in f["sync"]:
      let pair = line[1][0]  # Infix(||, a.x, b.y)
      syncs.add newTree(nnkTupleConstr, newLit(line[0].strVal),
        newLit(pair[1].repr), newLit(pair[2].repr))
  let (starts, ends) = (f.require("starts", name), f.require("ends", name))
  machines[name.strVal] = stateTy
  let nameLit = newLit(name.strVal)
  quote do:
    const `name`* = product(`nameLit`, `stateTy`, `members`, @`syncs`,
      memberPhases(`stateTy`, `starts`), memberPhases(`stateTy`, `ends`))

proc product*[S](name: string, _: typedesc[S], members: auto, syncs: seq[(string, string, string)],
    starts, ends: seq[S]): auto =
  ## Builds the composed table: for each composed state, each member's rows step that member's
  ## field; a synced pair adds one row under the sync's name where both members have a row, and
  ## the members' own rows for the pair are dropped. The composed outcome and facts are the first
  ## member's, which is the entity the composition is `for`. Body elided: it is `machineFor`,
  ## `enumerate(S)` and one loop per member, like `machine` above.
  discard

template startState*(S: typedesc, phase: untyped): untyped =
  ## `unscheduled` or `operation.unscheduled`: the first enumerated state at that phase.
  firstPerPhase(phaseStates(S, {phase}))[0]

template memberPhases*(S: typedesc, phases: untyped): untyped =
  ## `[operation.succeeded, worker.polling]`: composed states whose named member is at the phase.
  ## Elided: filters `enumerate(S)` by each `member.phase` pair.
  newSeq[S]()
