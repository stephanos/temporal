# Tag protobuf messages with Domains in the Model and IR

Source: the research note `.plans/PROTO_ANNOTATIONS.md` (2026-10-06), cited below as "the report" with its section numbers. Its section 11, "Decisions (2026-10-06)", records the owner's answers to all 22 questions. They are final, and this spec states them without reopening them. The report holds the research (the inventory of the server's validators, prior art, Scala libraries, generation tactics). This spec cites it and does not repeat it.

Source tags: **[paraphrase Dn]** restates decision n of the report's section 11. **[paraphrase §n]** restates the design in section n of the report, which the owner's decisions shape. **[inferred]** is this spec's own derivation, made where the report is silent or reads two ways. Decision Context lists each [inferred] item that interprets the report.

## Goal & Context
<!-- scope: business -->

Today the Models know nothing about what a valid request is. An action's inputs are finite abstract classes. Concrete request values are realization literals that nothing checks. The server's validation rules (ID lengths, required fields, the activity's "start-to-close or schedule-to-close" rule) are spread over an interceptor, the frontend, CHASM validators and history (report §4). A Model restates them at most in a comment (report §1). No Case exercises an invalid request, and Exploration has no class members to sample (`.plans/UMPIRE4_SPEC.md`, Abstraction Claim and Exploration).

This spec lets the Models tag any protobuf message and its fields with metadata, a name plus data, carried in the Umpire IR, not in a separate IR [paraphrase §Answer]. The first Tag library is validation. A Tag Set describes a request message exhaustively. Its single-fault classes become an ordinary finite input of the action, so the checker, the Quint export and every Query see an enum, as they do today [paraphrase D8]. A new Temporal-agnostic Go module, `tools/umpire/values`, interprets the library. It checks realization literals and generates valid, boundary and violating members for Cases [paraphrase §7 Go module]. Planned use: generated inputs for Cases and Exploration. Go validation codegen stays possible but is not planned [paraphrase D14].

It serves the Model author, who states a rule once and gets Cases for every way of breaking it. It serves the reviewer, for whom an upstream API field nobody tagged fails the gate by name. And it keeps the vision's rules: knowledge only in the Models, Temporal at the edges, one meaning per source, fail closed (`.plans/UMPIRE4_VISION.md`, Rules).

**Deferred** by the owner on 2026-10-06. Its tasks are planned when it is revived, after fn-141.9 [paraphrase D15]. fn-141 is deferred too, so this spec waits on fn-141's revival. It adopts fn-142's layout of `model/temporal`.

## Architecture & Data Models
<!-- scope: technical -->

**Words (SEM-19)** [paraphrase §7 Words]. A **Tag** is a name plus data. It attaches to a message, to a field path (or the path's elements, map keys or map values), or to a tagged Domain. A **Tag Definition** declares a Tag's name, data schema, targets, whether it rejects (with which `Rejection`), and its default message template. A **Domain** is a set of values. It is one concept in two forms. A *finite* Domain lists Model values (`Finite[T]`, today's input domains). A *tagged* Domain describes protobuf values by Tags. A **Tag Set** is the tagged Domain of one message type. **`unconstrained`** is the framework's no-op Tag. Each Tag Definition's name is itself a word under SEM-19.

**Layers and owners.**

| Layer | Holds | Knows Temporal |
| --- | --- | --- |
| Framework (DSL module) | The mechanism: `Domain`, `Finite`, `Tagged`, `TagSet`, Tag Definitions, `input(tagSet)`, the `isValid`/`violates`/`violation` sugar, `any`/`exactlyOne`. The validation library and `unconstrained` | No (`TestFrameworkNamesNoTemporal`) |
| `model/temporal/foundations/` (fn-142's layout: `foundations/taskqueue` plus this spec's additions) | Kit Tag Definitions and markers (`namespace`, `identifier`, …), shared field Domains, shared message Tag Sets in subfolders by subject (the `TaskQueue` Tag Set beside the task-queue entity in `foundations/taskqueue`), and the dynamic-config keys | Yes |
| `model/temporal/actors/` (fn-142: the worker and the client), `model/temporal/Bounds.scala` | Nothing of this spec's. A message type that only an actor's package reaches would follow the ownership rule like any package | Yes |
| Each feature package | The request Tag Sets of the RPCs it owns, the message types only it reaches, and `owns(...)` in its `object exports` | Yes |
| `model/temporal/capabilities` | The `Validated` capability law | Yes |
| Umpire IR | Tag Definitions, Tags, Domains, Tag Sets, templates, settings, `uses`, the `Any` and `setting` operands, the class link on enums | Only as declared names |
| `tools/umpire/values` | Interprets the framework's Tag library: admission, check, members, ignored Tags | No |

[paraphrase D18, D10, §7 Ownership]

**One Domain** [paraphrase §7 One Domain]. `sealed trait Domain[T]`. `Finite[T]` extends it, and its name and use do not change, so no existing Model changes. `Tagged[V]` is a field Domain: a named bundle of Tags over one protobuf value kind, optionally on a base whose Tags come first. Composition is tighten-only AND. A field Domain derives no input. `TagSet[M]` is the message form. On construction it derives `type Class` and `classes: Finite[Class]`.

**Single-fault classes** [paraphrase D8]. A Tag Set's classes are `valid` (every Tag on every reachable field holds), then one class per (field path, rejecting Tag), in which that Tag fails at that path and every other field is valid. That is 1 + Σ rejecting Tags classes, linear. The order is field order, depth first, with message-level Tags last. A class name is the Scala selector path and the Tag's name joined by `_` (`activityId_length`, `taskQueue_name_notPrefix`, `requires`). Paths reach through nested Tag Sets, repeated elements (`[*]`, the fault in one element) and map keys or values. A recursive type is entered at most once per path. Only a Tag whose definition rejects derives a class. A marker, a normalizing use (`Tag.normalizes`, [paraphrase D6]) and `unconstrained` derive none. `grouped` does not exist.

**Inputs** [paraphrase §7 One Domain, Connecting]. `input(tagSet)` is an ordinary `Input[tagSet.Class]` whose domain is `tagSet.classes`. `ActionDecl` gains `tagged` beside `tokens`. The checker builds action classes by the existing rule (`start-valid`, `start-activityId_length`). A request input multiplies with the action's other finite inputs, which is the existing catalog rule. The Model writes no class names, no mapping and no example. The realization writes the request once (`member(req)`), and lowering fills every tagged field from the class.

**Exhaustive and recursive** [paraphrase D9]. For each `tag[M]`, the field list is compared with M's ScalaPB descriptor when the Model is lifted. Then it recurses: every message type reachable through any field needs its own Tag Set. A Tag on a message-typed field speaks about the field (presence, serialized size). The type's Tag Set speaks about the inside. Both apply, ANDed. `unconstrained` on a message-typed field never excuses the type from having a Tag Set. `Payloads`, `Payload`, `Header`, `Memo` and `SearchAttributes` are tagged field by field. There are no whole-value Tag Sets. Edge cases (oneof, repeated, map, deprecated, well-known types, recursion) follow the report's §7 table "Exhaustive tagging".

**Ownership** [paraphrase D18, D19, D20]. An RPC is **used** when a realization writes it as a request (`rpc`, `readUntil`, `await`) or reads its response (`Recorded.read`, `Recorded.single`). Each used RPC has exactly one owning package, declared by `owns(METHOD_…)` in that package's `object exports` beside its `irFile`, and an exhaustive Tag Set for its request. An unused RPC needs nothing. Every message type reachable from a used RPC has exactly one owning Tag Set. A package's Tag Sets reach only its own types and foundations types. A type that two packages reach lives in foundations. Today's 13 used RPCs fall to three packages: `features/activity/standalone` (6), `features/nexus/standalone` (4) and a new, ownership-only `features/workflow` (`StartWorkflowExecution`, `DescribeWorkflowExecution`, `GetWorkflowExecutionHistory`) [paraphrase D22].

**IR files follow source** [paraphrase D1, D18]. The foundations IR file, `foundations`, carries the kit's Tag Definitions (the framework's are lifted there too), the shared field Domains, the shared message Tag Sets and the settings. Each feature IR file carries its own Tag Sets and keeps each derived class enum in its own `types`, so the interpreter, the checker and the Quint export still read a self-contained file. It lists `foundations` in `uses`, plus the IR file of any package that owns an RPC its realization uses (the Nexus caller uses `workflow`). That is a data reference, not a Scala import, so features still import no features.

**Meaning lives in consumers** [paraphrase D10, D13]. The IR checks only the shape of a Tag. The core validation Tags' meanings are normative in a "Tag library" section of `model/SEMANTICS.md`. `tools/umpire/values` is the one Go interpretation, and lowering, exploration and lint use it. A consumer that does not interpret a Tag says so. An uninterpreted *rejecting* Tag on a field a Case writes is a Known Gap in that Case. Server-state rules are exactly that, for now. Any other uninterpreted Tag shows in `umpire-lint` and the inventory.

**Run-time bounds** [paraphrase D4, D5, §8]. A quantity in a Tag's data is a number literal or a `setting` operand that names a typed dynamic-config key declared once in foundations. Lowering resolves each bound a generated value rests on to a relation (`atMost`/`atLeast`), never to an exact value. The relation goes into the Case's required settings with its origin. A remote Profile that cannot state the value makes the Case `PreparationUnavailable`.

**Data flow.** Scala Models → (fn-141 exporter, declaration-level values; lifter for conditions) → IR files with `uses` → Go: `values.Admit` → lowering (literal check, then members and relations) → Case. The checker and the Quint export read the derived enum as an ordinary enum.

## API Contracts
<!-- scope: technical -->

**Scala, framework and kit.** The report's §7 sketches (framework, foundations and a feature, connecting Tag Sets to the Model) are the reference surface. Spellings settle within `.plans/DSL_OPERATORS.md` at the framework task. These forms are the contract:

```scala
// framework
val lengthTag = definition(Target.text)("min" -> DataKind.quantity, "max" -> DataKind.quantity, "unit" -> DataKind.enumName)
  .rejects(Rejection.invalidArgument).message("{field} length exceeds limit of {max}.")
def length(min: Quantity = 0L, max: Quantity, unit: Unit = Unit.bytes): Tag
def requires(c: Condition[?]): Tag
def any[R](first: Condition[R], rest: Condition[R]*): Condition[R]   // lifts to Operand.any
def exactlyOne[R](cs: Condition[R]*): Condition[R]                     // sugar over any/all/not
def tag[M <: GeneratedMessage](lines: TagLine[M]*): TagSet[M]          // field, each, keys, values, message-level Tags
def input[M <: GeneratedMessage](s: TagSet[M]): Input[s.Class]
// s.valid, s.violation(_.path, tagDefinition), s.violation(tagDefinition), s.isValid(x), s.violates(x, _.path, tagDefinition)

// foundations
object id extends Tagged[String](identifier(), required, length(max = maxIdLength))
object requestId extends Tagged[String](length(max = maxIdLength), required.normalizes)

// a feature
object exports:
  val ir = irFile("activity-standalone")
  val rpcs = owns(METHOD_START_ACTIVITY_EXECUTION, /* … */)
val startActivity = tag[StartActivityExecutionRequest](
  field(_.activityId) is id.message(lengthTag, "ActivityId exceeds length limit of {max}."),
  /* every other field */
  requires(any(present(_.startToCloseTimeout), present(_.scheduleToCloseTimeout))))

// fn-139's rejecting row, renamed (D12)
when(phases) ~> rejects(Rejection.failedPrecondition).message("…")
```

**Umpire IR (`ir.proto`).** The report's §7 "IR additions" sketch is the contract, and the shapes it shows are exhaustive: `Model.tag_definitions`, `domains`, `tag_sets`, `settings`, `uses`; `TagDefinition`, `DataField` (kinds text, number, flag, enum name, condition, quantity; `repeated`, `optional`), `TagTarget`, `Tag` (definition, stable name, position, data, template override, `normalizes`), `DataValue` (field, `Operand`), `Domain` (name, position, bases, tags), `TagSet` (name, message, position, message Tags, `FieldTags`), `FieldTags` (path, target, domains, tags), `Template`/`TemplatePart` (text, data field, built-in `field`/`message`/`value`), `Any { repeated Operand operands }`, `Operand.any` and `Operand.setting`, `Enum.domain`, `Enum.domain_fingerprint`, `Case.violates`, `Violation { path, tag }`. Field numbers are the next free ones when the task lands. The report's numbers (Model 17-21, Operand 13-14) are free on 2026-10-06. Action inputs stay `repeated Param`. No IR `Input` message is added. `SettingDeclaration` carries key, value kind (codec) and scope, with no citation.

**Testpilot schema.** `RequiredSetting` gains a relation (`EQUAL` by default, `AT_MOST`, `AT_LEAST`), an origin, and a reference to the symbolic role binding a scoped key resolves through (report §8 Scope). `AbstractionClaim` gains the class realized: `valid`, or the violated path and Tag (report §5). Default-empty fields leave existing Case bytes unchanged.

**Go, `tools/umpire/values`** [paraphrase §7 Go module]:

| Function | Contract |
| --- | --- |
| `Admit(kit, model)` | Admission of every Tag-related part of a feature file and the files it `uses` (R12's error list) |
| `Check(tagSet, message, settings) []Violation` | Violations by path and Tag, each with its interpolated message. `settings` is an argument, never a default |
| `Members(tagSet, class, settings, seed)` | A whole request in the class, with the relations it rests on. Seeded and deterministic |
| `Ignored(kit) []TagUse` | Every Tag use it does not interpret, marked rejecting or not |
| `Register(format, generator)` | Named format generators. `uuid` is built in |

## Edge Cases & Constraints
<!-- scope: technical -->

- **Profile-owned fields** (namespace, task queue; ART-13) keep the Profile's value in `valid` and in every class that violates another field. Only their own violating classes vary them. No per-Case provisioning [paraphrase D3].
- **Repairing rules** (auto-filled request ID, capped durations, oversize payloads turned into failures) are marked `normalizes` or left unmodeled. They derive no class and no modeled value [paraphrase D6].
- **Server-state rules** (search-attribute definitions and the like) are rejecting Tags that no consumer interprets. Their classes exist, and lowering reports their Cases `unsupported` with a located reason [paraphrase D13, §7].
- **Conflicting relations.** Lowering takes the union of all origins. Requirements that cannot all hold refuse the Case, naming both origins. Nothing is resolved by "last wins". [inferred] `values` picks a valid member's bounded lengths so that they cannot conflict with the class's own violating relation on a shared key, such as `limit.maxIDLength` across many fields. Boundary members at `max` belong to Exploration, not to the canonical member.
- **Member size.** `values` takes boundaries from the key's registry default, so most Cases run on unconfigured servers. For `maxBytes` it picks a small *n*, which requires `limit.blobSize.error atMost n − 1`. That keeps megabyte values out of checked-in Cases (report §8, a design choice).
- **Determinism.** The same IR, settings and seed give the same members and Case bytes (PLN-02, ART-11). Generated values are literals in the Case.
- **Fingerprints.** A Tag Set's fingerprint covers its Tags, their definitions and the Tag Sets it reaches. Through `domain_fingerprint` it enters the derived enum, and with it every input, step and Query over that enum. Templates are excluded (not behavior, [paraphrase D11]). A Tag Set used only for the literal check fingerprints nothing, as with `ApiBehavior`.
- **Renames ripple** (accepted trade-off, report §7). Tagging a new field or renaming a selector adds or renames classes, Query totals and IDs in every Model with that input. A foundations Tag Set change reaches every feature that reaches it.
- **Upstream is a forcing function.** When `go.temporal.io/api` adds a field to a reachable message, or a realization starts to use a new RPC, `make umpire-check-model` fails until the field is tagged or the RPC has an owner and a Tag Set.
- **No links to server code.** No Tag, Tag Definition, Domain, setting, template or IR field carries a citation, a `because` or a server source path (owner, 2026-10-06, report §1 constraints). Templates restate expected behavior.
- **Module map.** The framework names no Temporal word. The IR's dependencies are unchanged. Testpilot imports nothing from `tools/umpire` or `model`. Features import no feature, because `uses` is data.
- **fn-141 interplay.** Tag Sets, field Domains and their classes are declaration-level values that the exporter emits by running Scala, with no lifter matcher (fn-141 R3, R6). A class reference closed over by a guard binds as the finite value it is (fn-141 R8). `isValid`/`violates` in a step body is function-level sugar, expanded generically (fn-141 R4). `Condition` keeps its operator (fn-141.7, R5).
- **Not concurrent with fn-143** [inferred]. fn-143 renames the framework's folder and package. Both specs touch framework files, and one at a time keeps each diff check clean.
- **Model changes and conformance.** When a violating-class Case fails against the server, the Model is not fitted to the code. The owner decides (AGENTS.md).

## Acceptance Criteria
<!-- scope: both -->

- **R1:** [paraphrase §7 One Domain] The framework has one `Domain[T]` concept. `Finite[T]` extends it unchanged in name and use. `Tagged[V]` declares a field Domain of Tags over one protobuf value kind, optionally on a base whose Tags come first, ANDed. `TagSet[M]` is the message form, with `classes: Finite[Class]` derived on construction. Existing Models compile unchanged, and `make umpire-gen-model` leaves `model/ir` and `model/cases` byte-identical until a Model uses a Tag. Errors: a field Domain used on a field of another protobuf kind is refused at its line, naming the field, the Domain and both kinds; a base cycle is refused naming the cycle; a base of another value kind does not compile.
- **R2:** [paraphrase §7 Framework] A Tag Definition, named after its `val`, declares data fields (each of a `DataKind`, optionally repeated or optional), targets, an optional `rejects(Rejection)` and a default template. A Tag use supplies data, may override the template, and may be marked `normalizes`. Errors: two definitions of one name are refused naming both positions; a use with a missing, unknown, mistyped or wrongly repeated data field is refused at the use, naming the field and the schema; a use at a target its definition does not list is refused naming the target; `normalizes` on a Tag whose definition does not reject is refused [inferred]; a quantity field given anything but a number literal or a declared setting is refused.
- **R3:** [paraphrase D10, §5, §7] The framework ships the core validation library with `unconstrained`, with names and meanings close to protovalidate's standard rules and XSD's facets: `required`, `notDefault`, `length(min, max, unit)`, `pattern`, `notPrefix`, `in`, `notIn`, `range`, `count`, `maxBytes`, `format(name)`, `sameSign`, `nonNegative` and `requires(condition)`. `model/SEMANTICS.md` gains a normative "Tag library" section that gives each Tag's targets, data, whether it rejects, its default template and its meaning, and pins the regex dialect (RE2, Go's `regexp/syntax`), the anchoring and the length unit (default bytes, Go `len`). `TestFrameworkNamesNoTemporal` passes. Errors: SEMANTICS states each Tag's ill-formed data, and the lift refuses it at the use: `min > max`, a pattern that is not valid RE2, an unknown unit or format, an empty `in`.
- **R4:** [paraphrase §7 Conditions, `Any`] `requires` takes a framework `Condition`, a deep embedding lifted to `Operand`. `Operand` gains a first-class `Any` node, and the framework gains `any` and the sugar `exactlyOne`. Every `Operand` consumer handles `Any`: realization admission (operand, payload and validation), lowering, conformance guards and lint. Lowering maps `Any` one to one to Testpilot's `AnyExpression`. A Tag's condition stays within presence, equality and comparison with literals, under `all`, `any` and `not`. Errors: a condition outside that subset is refused at lift and again at admission, naming the node; a path outside the tagged message is refused naming the path; an `Any` with no operand is refused at admission; a Scala `Req => Boolean` where a condition is expected does not compile.
- **R5:** [paraphrase D11, §7 Message templates] Templates are parsed at lift into a `Template` of literal and placeholder parts, so Go never parses text. The placeholders are closed: the definition's data fields, `{field}`, `{message}` and `{value}`. A use overrides a definition's template, and so does a field Domain for one of its rejecting Tags. A quantity placeholder interpolates the value the Profile states for its setting. Templates serve the `values` diagnostics and a future codegen. Conformance compares status codes only. Errors: an unknown placeholder or an unbalanced brace is refused at lift, naming it and the allowed set; a field-Domain override that names a Tag Definition the Domain does not carry, or one that does not reject, is refused naming both.
- **R6:** [paraphrase D12] fn-139's `rejects(r).because(text)` is `rejects(r).message(template)`, with R5's template form. The Tag Definition's `rejects(r)` picks the `Rejection`, whose status code fn-139.8's table fixes, and the template is the message within it. If fn-139.1 has not landed when this work starts, fn-139 adopts the spelling. Otherwise this spec renames it in the framework, the lifter, every Model and every fixture. No `.because` on a `rejects` value remains. Errors: [inferred, SEM-20] `.because` on a `rejects` value is refused at its line with an error naming `.message`; a placeholder that a rule row cannot bind (any data placeholder) is refused at lift.
- **R7:** [paraphrase D9, §7 Exhaustive tagging] `tag[M](…)` takes `field`, `each`, `keys` and `values` lines and message-level Tags. At lift, its selectors are compared with M's descriptor, oneof members and deprecated fields included, and the check recurses: every message type reachable through any field needs its own Tag Set. Well-known types are leaves. Recursive types are tagged once. The check runs in `make umpire-check-model`, and Go admission repeats it against `protoregistry.GlobalFiles`. Errors: a missing field fails with `<Message> is not tagged exhaustively: <paths> (tag each, or mark it unconstrained)`; an untagged reachable type fails with `<Type>, reached from <Message>.<path>, has no Tag Set`; a second `tag[M]` for one M is refused naming both positions; `each` on a non-repeated field, `keys`/`values` on a non-map field, a field tagged twice, or a Tag whose target does not fit the field's kind is refused at the line.
- **R8:** [paraphrase D19, D20, §7 Ownership] A package declares the RPCs it owns with `owns(METHOD_…)` in its `object exports`, beside its `irFile`, using the generated gRPC method constants. A gate check, in the gate's lint step, walks each realization's commands and evidence for the methods they name (`rpc`, `readUntil`, `await`, `Recorded.read`, `Recorded.single`) and resolves each through the service descriptors. Every used RPC is owned exactly once and has an exhaustive request Tag Set. Every message type reachable from a used RPC has exactly one owning Tag Set. A package's Tag Sets reach only its own types and foundations types. An unused RPC carries no obligation. Errors: `<Realization> uses <Service>.<Rpc>, whose request is not tagged exhaustively: <paths>`; `<Realization> uses <Service>.<Rpc>, which no package owns`; an RPC owned by two packages is refused naming both; a Tag Set that reaches another feature's type is refused, naming the type and stating that a type two packages reach belongs in foundations.
- **R9:** [paraphrase D22] `model/temporal/features/workflow` exists as an ownership-only feature: one feature file that holds only `object exports` (its `irFile` and `owns` for `StartWorkflowExecution`, `DescribeWorkflowExecution` and `GetWorkflowExecutionHistory`) and those requests' Tag Sets, with no machine. The structure lint and the module map's Models row admit it. The Nexus caller's IR file lists `workflow` in `uses`. `features/activity/standalone` owns its six used RPCs and `features/nexus/standalone` its four. Errors: no error surface beyond R8 and the structure lint's existing refusals.
- **R10:** [paraphrase D18, §7 Foundations] Foundations declares the kit's Tag Definitions and markers (non-rejecting, no data), the shared field Domains (`id`, `namespaceName`, `taskQueueName`, `requestId` with `required.normalizes`, `timeout`), and the shared message Tag Sets, each beside its subject (the `TaskQueue` Tag Set beside the task-queue entity in `foundations/taskqueue`), all in `model/temporal/foundations/` as fn-142 lays it out. The worker and the client stay in `model/temporal/actors/`, and `Bounds.scala` stays in `model/temporal/`. The owning packages declare exhaustive Tag Sets for the 13 used requests and every type they reach (62 types with 281 fields on 2026-10-06, recounted at the task, `google.protobuf` excluded). `StartActivityExecutionRequest`'s Tag Set states the deadline rule as `requires(any(present(start_to_close_timeout), present(schedule_to_close_timeout)))`, and the realization's comment that restates the rule goes [inferred]. Errors: no Tag, Domain, Tag Definition or setting carries a citation or server path, which review checks; everything else is covered by R7 and R8.
- **R11:** [paraphrase D8, D6, §7 Single-fault classes] A Tag Set derives `valid` plus one class per (field path, rejecting Tag), in field order, depth first, with message-level Tags last, named by selector path and Tag name joined by `_`. Paths reach through nested Tag Sets, `[*]` elements and map keys or values. A recursive type is entered once per path. Markers, `normalizes` uses and `unconstrained` derive no class. A rejecting Tag that no consumer interprets still derives its class. `input(tagSet)` is an ordinary finite input. The checker builds action classes from it by the existing rule, and `isValid`, `violates` and `violation` are sugar over the derived cases. A test pins `startActivity`'s class list. Errors: `violation` that names a path or Tag the Tag Set does not carry is refused at construction, naming both; two paths that derive one class name are refused naming both; a hand-written example on a Tag Set input is refused at its line.
- **R12:** [paraphrase D1, D17, D18, §7 IR additions] The IR gains the messages and fields under API Contracts. The foundations IR file holds the kit's (and the framework's) Tag Definitions, the shared Domains, the shared Tag Sets and the settings. Each feature file holds its own Tag Sets and derived enums and lists what it references in `uses`. `Enum.domain`, `Enum.domain_fingerprint` and `Case.violates` link a derived enum to its Tag Set. `model/SEMANTICS.md` states that the interpreter, the checker and the Quint export read such an enum as an ordinary enum and ignore the Tag parts by rule, so the "reject what you do not implement" rule of Versions holds. Errors: admission (`values.Admit`) refuses, at the IR position: a Tag naming no definition or data off its schema; a target, path, kind or literal that does not fit the descriptors; a template naming an undeclared placeholder; a Tag Set that is not exhaustive, not recursive or not unique per message; an unresolved `uses`; an enum with a `domain` whose cases differ from the derivation (in order, fields, `violates` or fingerprint); an example on such an input; a `setting` operand outside a quantity field or naming no declared setting; a condition outside presence, equality and comparison.
- **R13:** [paraphrase D10, D13, §7 Go module] `tools/umpire/values` implements `Admit`, `Check`, `Members`, `Ignored` and `Register` as under API Contracts. It interprets only framework Tags, and it imports the Umpire IR, `protoreflect`, stdlib `regexp/syntax` and the reader package that types `Operand`, with no Testpilot, no Temporal name and no new third-party library. `.plans/UMPIRE_MODULES.md` gains its row, and the Lowering, Exploration and Lint rows may import it. Members are valid by construction, boundary (min, min+1, max−1, max, multibyte runes), or violating only their own Tag, and Profile-owned fields follow D3. Fixture tests (MOD-08) show every `Members(valid)` passes `Check`, and every `Members(violation)` fails `Check` on exactly that path and Tag. Errors: `Check` or `Members` without a value for a quantity's setting returns an error naming the key, never a default; a `requires` beyond the interpretable atoms, or any Tag it does not interpret, is listed by `Ignored`; Tags that admit no member (a pattern against a length) are an error naming the Tag Set, the path and both Tags.
- **R14:** [paraphrase §7 first increment] Lowering checks the request each realization writes against the request's Tag Set through `values.Check`, and refuses a violating literal, naming the realization, the field and the Tag. For example, a start with neither deadline is refused with `startActivity.requires`. [inferred] An operand that is not a literal (environment binding, run id, learned value) counts as present, and its own Tags are not evaluated. Case bytes are unchanged. Errors: a violation is a lowering refusal at the realization's position; a used RPC with no Tag Set is already a gate failure under R8.
- **R15:** [paraphrase D7, D8, §7 Connecting, Abstraction Claims] An action may take a Tag Set input. Its realization writes `member(req)` once, and lowering fills every tagged field from `values.Members` with a seed derived from the Case identity. The member is recorded as the class's Abstraction Claim example, and `AbstractionClaim` names the class realized. An uninterpreted rejecting Tag on a field the Case writes is a Known Gap in that Case. A class whose own Tag is uninterpreted lowers `unsupported` with a located reason. The standalone activity's start is the first adopter [inferred]. Errors: two lowerings of one Case give identical bytes (ART-11), or the test fails naming the first differing field; `member(req)` for an action with no Tag Set input is refused at its line.
- **R16:** [paraphrase D7] `model/temporal/capabilities` declares the kit-level `Validated` capability. A machine that declares it on an action with a Tag Set input gets the law: every violating class's row rejects with its Tag's `Rejection` and keeps the state. The checker proves it over the table, as for the other laws. A Run is checked on the status code through fn-139.8's table and on no state change through the evidence the Model's Facts already name. Errors: a row that accepts a violating class, rejects with another `Rejection` or changes state is a law violation naming the machine, the class and the row; `Validated` on an action with no Tag Set input is refused at its line.
- **R17:** [paraphrase D4, D5, §8] Pulled forward from fn-125. All of fn-125.5: each dynamic-config key a Tag bound uses is declared once in `model/temporal/foundations/`, with no citation of its server definition (the pin test is the link), typed (codec) and scoped (global, namespace, task queue), and carried in the foundations IR file's `settings`, and a Temporal-side Go test pins each declaration to the server registry (exists, codec, scope). From fn-125.6: `Program.required_settings` entries carry a relation and an origin, lowering takes the union of all origins, and a Tag bound a generated value rests on is a new origin. Relations only: a valid member of length *n* requires `atLeast n`, and a violating one `atMost n − 1`. A scoped key names the symbolic role binding, which preparation resolves from the Profile (ART-13). Preparation checks each relation against the value the Profile states. [inferred] The functional harness sets each related key to a value that satisfies its relation (the registry default when that does). Errors: an unknown key, a codec or scope mismatch fails the pin test, naming the key and its line; two origins that cannot both hold refuse the Case, naming both; a remote Profile that does not state a required key's value makes the Case `PreparationUnavailable`, naming the key, the relation and the origin; an unmet relation does the same.
- **R18:** [paraphrase D21] `model/temporal/shared/` becomes `model/temporal/foundations/` as its own step, with one regeneration batch and nothing else riding along. fn-142 is that step, and this spec depends on it. This spec adopts fn-142's layout: `foundations/` holds the task queue plus this spec's shared Domains, Tag Sets, markers and settings keys; `actors/` holds the worker and the client; `Bounds.scala` sits in `model/temporal/`. fn-142's batch changes Definition IDs exactly by the package mapping (`temporal.shared.taskqueue` → `temporal.foundations.taskqueue`, `temporal.shared.worker` → `temporal.actors.worker`). No foundations Tag content lands before it. Errors: no error surface beyond fn-142's own criteria.
- **R19:** [paraphrase D16; inferred for the glossary] The spec and doc wording follows the one meaning of Domain. `.plans/UMPIRE4_SPEC.md`'s Capability entry reads "a connector joins two components explicitly". Its Action entry is extended: "…; a Tag Set's members are its single-fault classes, each standing for the protobuf messages whose only fault, if any, is the class's". `model/README.md`'s "domain roles `caller` and `handler`" is reworded in the subject-area sense. The words Tag, Tag Definition, Tag Set and `unconstrained` are defined once in the spec's Model authoring concepts, which is a spec-text change under GOV-02 [inferred], and are used with one spelling in SEMANTICS, the README, the IR and Go (SEM-19). Errors: a second word for any of these concepts in the changed docs, IR or Go is a review finding.
- **R20:** [inferred] The rules of record describe the result. `.plans/UMPIRE_MODULES.md` updates the DSL row (the mechanism and the validation library), the Models row (foundations content, `owns`, ownership-only features), the Capabilities row (`Validated`), the new Values row, and the import rows of Lowering, Exploration and Lint. `model/README.md` explains how to tag a message, declare a field Domain, own an RPC, take a Tag Set input and declare `Validated`. The closing task passes the model gate, `make lint-model`, the Go tooling suites, `make umpire-check-cases` with the recorded deltas, and `make lint-code-fast`. Errors: no error surface beyond the gates.

**Error cases (negative-cases discipline):** each criterion above states its error cases inline or records that it has none.

## Order

The report's §10 phasing, with the requirements of each step. Each step leaves `model/ir` and `model/cases` with only the deltas it names.

| Step | Requirements | Case bytes |
| --- | --- | --- |
| 1. Tag mechanism in the framework and IR, exporter and Go admission, recursive exhaustiveness, SEMANTICS Tag library | R1-R5, R7, R11 (derivation), R12, R13 (`Admit`) | unchanged (no Model uses a Tag) |
| 2. `shared/` → `foundations/` | R18 (fn-142) | fn-142's |
| 3. Ownership: `features/workflow`, `owns` for 13 RPCs in three packages, gate checks | R8, R9 | unchanged |
| 4. fn-125.5 and the required-settings slice of fn-125.6; foundations and feature Tag Sets for the 13 requests and 62 types | R10, R17 (declarations, pin test) | unchanged |
| 5. `values` check; lowering's literal check | R13 (`Check`), R14 | unchanged |
| 6. Tag Set inputs, `Validated`, violating classes with `rejects`, generated members, relations in required settings (needs fn-139 with the D12 rename) | R6, R11 (inputs), R13 (`Members`), R15, R16, R17 (relations) | new violating-class Cases, required settings, claims |
| close | R19, R20 | — |

## Boundaries
<!-- scope: business -->

- No Go validation codegen, and no spec rule that lets a Generated View become server code [paraphrase D14].
- No generation of Go dynamic-config declarations from the Model. Foundations declares keys, and the pin test checks them against the registry.
- No mapping of package ownership to CODEOWNERS or teams [paraphrase D20].
- No obligations for unused RPCs [paraphrase D19].
- No response Tag Sets and no Tag Sets for the ten worker-side `Proto[...]` messages yet. Both come later, in their owner's package (report §10 step 7).
- No upstream protovalidate or `google.api.field_behavior` proposals [paraphrase D2].
- No conformance matching of error messages. Conformance compares status codes only, with no pattern match and no redaction policy [paraphrase D11].
- No per-Case provisioning of namespaces or task queues [paraphrase D3].
- Exploration's class-member targets are not wired here. `values.Members` is their interface, and report §10 schedules no step for them [inferred].
- The rest of fn-125 stays deferred: Model settings, `under`, the Nexus encodings, API preconditions, wait-bound assumptions, and fn-125.9's preparation checks beyond relations.
- No modeled value for repairing rules [paraphrase D6]. No interpretation of server-state rules [paraphrase D13].

## Decision Context
<!-- scope: both -->

### Motivation
<!-- scope: business -->

[paraphrase §Answer] Proto options cannot be the source today. The public request messages arrive as compiled Go from `go.temporal.io/api`, protobuf cannot attach options to another file's messages, and the server uses neither protovalidate nor PGV. Smithy's model fits: validation traits are ordinary traits with a declared shape, and meaning lives in libraries and consumers. That keeps the IR Temporal-agnostic, so a later library (documentation, Driver hints, response checks) costs no IR change. Tags are tested claims, not documentation: drift shows as a violating-boundary Case whose Verdict turns violated. Exhaustiveness forces every new upstream field to be looked at.

### Implementation Tradeoffs
<!-- scope: technical -->

**Owner decisions (2026-10-06), all final.**

| # | Decision | Lands in |
| --- | --- | --- |
| D1 | One kit-level IR file referenced by feature files, refined by D18 | R12 |
| D2 | No upstream protovalidate or `field_behavior` now; vocabulary stays close to protovalidate | R3, Boundaries |
| D3 | No provisioning; `valid` uses the Profile's namespace and task queue; only their own violating classes vary them | R13, Edge Cases |
| D4 | Pull forward only fn-125.5 and the required-settings slice of fn-125.6 | R17 |
| D5 | Limits are relations (`atMost`/`atLeast`), not exact values | R17 |
| D6 | Normalizing Tags are marked only (`Tag.normalizes`), with no class and no modeled value | R2, R11 |
| D7 | Violating classes: status code plus no state change, through the kit-level `Validated` law | R16 |
| D8 | Single-fault classes, 1 + Σ, through one action-level input over the request's Tag Set; `grouped` dropped | R11, R15 |
| D9 | Exhaustive request Tag Sets, recursing into every reachable message; no whole-value Tag Sets | R7 |
| D10 | Core validation Tags normative in `model/SEMANTICS.md` | R3 |
| D11 | Conformance compares status codes only; templates serve diagnostics and future codegen | R5 |
| D12 | fn-139's `.because(text)` becomes `.message(template)` | R6 |
| D13 | Server-state rules are uninterpreted rejecting Tags, recorded as Known Gaps | R13, R15 |
| D14 | Codegen not planned; its spec rule deferred | Boundaries |
| D15 | After fn-141.9 | dependency |
| D16 | Reword the spec's Capability entry and README's domain roles; extend the Action entry | R19 |
| D17 | IR link on the enum: `Enum.domain`, `domain_fingerprint`, `Case.violates` | R12 |
| D18 | Source location follows ownership; foundations for shared things; framework Tag Definitions in the framework; IR files follow source | R9, R10, R12 |
| D19 | Only used RPCs carry obligations; one owner, an exhaustive Tag Set, one owning Tag Set per reachable type | R8 |
| D20 | No CODEOWNERS; ownership is package ownership enforced by the gate | R8, Boundaries |
| D21 | Rename `shared/` to `foundations/` as its own step; fn-142 performs it, with its `actors/` and `Bounds.scala` layout | R18 |
| D22 | Create ownership-only `features/workflow` now | R9 |

**Rejected, one line each.** Upstream proto options: another repo's release cycle, static values only (report §6 A). A sidecar overlay: untyped proto names, which the DSL forbids, and disconnected from the Model (§6 B). iron, refined and scalapb-validate: type-level or PGV-based, nothing the lifter reads (§3). smithy4s: a second schema system. The Model's `Expr` for conditions: it speaks finite Model types and would need value conversion (§7). `Any` as `not(all(not …))`: Testpilot already has `AnyExpression`, and diagnostics read a disjunction. Symbolic bounds resolved by Testpilot: they would grow Testpilot and the Case format (§8). `grouped` classes and whole-value Tag Sets: dropped by D8 and D9.

**Interpretations this spec makes** (the report is silent or reads two ways):

- **The rename step is fn-142, and its layout is adopted** (settled by the owner, 2026-10-06). fn-142 renames `shared/` to `foundations/`, moves the worker and the client to `model/temporal/actors/`, and moves `Bounds.scala` to `model/temporal/`. fn-142's R3 now states that Definition IDs change exactly by the package mapping, because IDs are fully qualified names and `family` is the package.
- **Testpilot changes a little.** The report says Testpilot "does not change" (§5, §8) and is "untouched" (§7). Yet §8 adds a relation and a binding reference to Testpilot's `RequiredSetting`, and preparation must check relations. R17 and API Contracts include both, with no symbolic values.
- **The fn-125 slice includes relations and their preparation check** (settled by the owner, 2026-10-06). D4 says "only fn-125.5 and the required-settings slice of fn-125.6". The relations are fn-125 R10's (task .9), and D5 needs them, so R17 includes the relation and the preparation check for relations that Tag bounds use, and nothing else of fn-125.9. fn-125 records the absorption.
- **No citation on keys, and keys live in foundations** (settled by the owner, 2026-10-06). fn-125 R6 and task .5 now put the keys in `model/temporal/foundations/` with no citation of their server definition, and the registry pin test is the link.
- **Operand typing lives in `realization`, not `interp`.** The report's Values row names "interp for `Operand`". `Operand` is typed in `tools/umpire/realization`, so R13 names "the reader package that types `Operand`".
- **The literal check counts non-literal operands as present** (R14). The report says only "every literal operand".
- **fn-139's template has no data placeholders on a rule row** (R6). Where the parsed template sits in the IR row, beside or in place of today's `because` string, is settled at the task.
- **The local harness picks values that satisfy relations** (R17). fn-125 R10 runs a declared value. A relation declares none.
- **The first adopter of a Tag Set input is the standalone activity's start** (R15). The report's sketch uses it, and §10 names none.

## Parked unknowns

- Whether the spec's Abstraction Claim entry ("an author's claim … with the example the functional Case runs") and the Set entry ("using each class's example") need rewording for examples that lowering generates. D16 lists no such edit. It is resolved by the owner at R19's task.
- Whether the realization command `readUntil`, which fn-141 R2 lists as a removal candidate, still exists when R8's used-RPC walk is written.

## Quick commands

```bash
scala-cli test model/irgen
make lint-model
make umpire-check-model
make umpire-gen-model   # review the diff of model/ir and model/cases against the step's named deltas
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...
make umpire-check-cases
make lint-code-fast
```

## Dependencies

- **fn-141** (spec dependency). The real gate is fn-141.9, which exports actions and inputs from constructed values [paraphrase D15]. flowctl stores spec-level dependencies only, so the conductor may start once fn-141.9 is done. fn-139 (the D12 rename, the `Rejection` code table for R16) comes in transitively through fn-141.
- **fn-142** (spec dependency): the `shared/` → `foundations/` step (R18).
- **fn-125**: this spec absorbs fn-125.5 whole, the required-settings part of fn-125.6 (relations and origin, union, conflict refusal), and the relation and preparation-check part of fn-125.9 for relations that Tag bounds use (R17). fn-125 and its tasks .5, .6 and .9 record this. The rest of fn-125 stays there.
- **fn-143**: not concurrent (Edge Cases).
