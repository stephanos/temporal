# Proto tags: Domains, Tags and validation in the Umpire IR

Research note, 2026-10-06. Revised the same day with the owner's decisions:

- the words Domain and Tag;
- no links to server code;
- input classes derived from Domains;
- generic Tags with data, interpreted by consumers;
- cross-field Conditions;
- exhaustive tagging;
- message templates;
- run-time bounds through fn-125.

The owner's answers to the open questions are in section 11, "Decisions (2026-10-06)". The design
below is written to them.

The question: how do we tag any protobuf message and its fields with metadata (a meaning such as
`namespace`, plus the validation it brings), carry that metadata in the Umpire IR rather than in a
separate IR, and keep it tightly connected to the Model? Planned use: generate valid, boundary and
violating inputs for Cases and Exploration. Visionary use, kept open but not scoped: generate Go
validation code.

The words are fixed in section 7:

- A **Tag** is a name plus data. It attaches to a message, a field path or a tagged Domain.
- A **Tag Definition** declares a Tag's name, data schema, targets, whether it rejects, and its
  message template.
- A **Domain** is a set of values: the framework's existing concept, an input's domain, now in
  two forms. A *finite* Domain lists Model values (`Finite[T]`, today's input domains). A *tagged*
  Domain describes protobuf field values by Tags, and its derived classes are what make it finite
  for the Model.
- `unconstrained` is the no-op Tag that exhaustive tagging uses for fields with nothing to say.

Grounded in `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_VISION.md`, `.plans/UMPIRE_MODULES.md`,
`model/README.md`, `model/SEMANTICS.md`, the working-tree `ir.proto`, `model/umpire`,
`model/temporal`, `tools/umpire`, `.flow/specs/fn-125-*`, `.plans/DYNAMIC_CONFIG.md`, and the
server's validators. Line numbers are from the working tree on 2026-10-06. External claims link to
primary sources. **[unverified]** marks a claim that could not be confirmed.

Server citations appear only in the research sections (1, 4). The design links to no server code.

## Answer

**The IR gains a generic, Temporal- and validation-agnostic tag mechanism.**

- A Tag carries data as named `Operand`s. `Operand` is the IR's existing protobuf-literal and
  condition language.
- Tag Definitions declare each Tag's schema, targets, whether it rejects, and its default
  message template, so the lifter and Go admission check every use's shape.
- A tagged Domain is a set of protobuf values described by Tags. It is the same concept as an
  input's finite domain, not a new one.
- Every request message of an RPC a realization uses has one exhaustive Tag Set, checked at lift
  against its descriptor and recursing into every message it reaches. A Tag Set is the tagged
  Domain of its message.
- Source location follows ownership.
  - An RPC's request Tag Set lives in the package of the feature that owns the RPC.
  - Shared Domains, shared message Tag Sets, the kit's Tag Definitions and its settings live in
    `model/temporal/foundations/`, the renamed `shared/`, beside the task queue. fn-142 performs
    the rename and moves the worker and the client to `model/temporal/actors/` and `Bounds.scala`
    to `model/temporal/`.
  - IR files follow source: each feature's IR file carries its own Tags, and a foundations IR file
    carries the shared ones and is referenced by the others.
  - Every RPC a realization uses has exactly one owning package and an exhaustive Tag Set. Every
    message type it reaches has exactly one owning Tag Set. Unused RPCs need nothing, and a gate
    check enforces the rest.

**Meaning lives in consumers, not in the IR.**

- The framework ships the first Tag library: validation (`required`, `length`, `pattern`,
  `range`, `requires`, …, plus `unconstrained`). Its names stay close to protovalidate, and its
  meanings are normative in `model/SEMANTICS.md`.
- The kit (in `foundations/`) adds Temporal markers (`namespace`, `workflowId`, …), shared Domains, and the
  dynamic-config keys bounds refer to.
- A new Temporal-agnostic Go module, `tools/umpire/values`, interprets the validation library: it
  checks values and generates valid, boundary and violating members. Every Tag a consumer does not
  interpret is reported. A rejecting one becomes a Known Gap.
- Run-time limits resolve through fn-125.5 and a slice of fn-125.6, pulled forward. Lowering
  records each key a generated value rests on in the Case's required settings, as an
  `atMost`/`atLeast` relation, and remote Profiles fail closed.

Why this design:

- **Proto options cannot be the source today.**
  - The public request messages come from the external `go.temporal.io/api` module as compiled
    `.pb.go` files (`go.mod:75`). The module cache holds no `.proto` sources.
  - `proto/api.binpb` is rebuilt from the linked Go registry (`Makefile:335-338`).
  - Protobuf has no way to attach options to another file's messages. protovalidate has no overlay
    either ([protovalidate](https://github.com/bufbuild/protovalidate); absence searched,
    **[unverified]**).
  - ScalaPB's auxiliary options do apply to other files, but they carry only ScalaPB's own options
    ([ScalaPB customizations](https://scalapb.github.io/docs/customizations/)).
  - The server uses neither protovalidate nor PGV.
  - The public API does not carry `google.api.field_behavior`. Its comments say the annotation is
    "not available in our gogo fork" (module cache, `workflowservice/v1/request_response.pb.go:6618-6620`).
- **This is Smithy's model.** In Smithy, validation traits are ordinary traits whose shape is
  declared ([defining traits](https://smithy.io/2.0/spec/model.html#defining-traits)).
  - The IR knows only "a Tag of a declared shape". So it stays as the vision wants: Temporal
    appears "only as declared names and payload types" (`.plans/UMPIRE4_VISION.md:55-60`).
  - Validation is one Tag library among future ones (documentation, Driver hints, response
    checks), and none of them costs an IR change.
- **Tight connection to the Model.**
  - An action takes one input over its request's Tag Set. That input's domain is the
    **single-fault** classes: `valid`, plus one class per (field path, rejecting Tag), with every
    other field valid. That is 1 + Σ rejecting Tags, linear, not a cross product.
  - The input is an ordinary finite input: the checker, the IR and the Quint export see an enum, as
    they do today.
  - Each class's Abstraction Claim gets a defined member set, which is what the spec says
    Exploration samples (`.plans/UMPIRE4_SPEC.md:247-250`, `:453-459`).
  - A violating class's row is `rejects(invalidArgument)`, using fn-139's shared `Rejection`
    (`.flow/specs/fn-139-actor-grouped-rules-per-rpc-actions.md:55`).
  - A kit-level `Validated` capability law holds every violating class to that status code and to
    no state change. fn-139.8's code table checks the observed code (`:61`), and Conformance
    compares status codes only.
- **Tags are tested claims, not documentation.** The server has no single declarative source: its
  rules are spread over an interceptor, the frontend, CHASM validators and history (section 4).
  Drift shows up as a violating-boundary Case whose Verdict turns violated.
- **Exhaustiveness is a forcing function.** When `go.temporal.io/api` gains a field, `make
  umpire-check-model` fails, naming it, until someone tags it.

## 1. Current state in the repo

### How requests and inputs are represented

- **Model inputs are finite abstract classes, never concrete request values.**
  - An action has `inputs` (finite `TypeRef`s), `schemas` and `examples` (`ir.proto:322-343`).
  - An `Example` is an Abstraction Claim: a class value plus a free-text `example`
    (`ir.proto:345-349`; `model/umpire/Action.scala:29-31`, `:120-124`). Admission checks only
    that its class exists (`model/SEMANTICS.md:705-706`).
  - Lowering copies the claims on a Case's path into provenance (`tools/umpire/check/claims.go:236-250`,
    `tools/umpire/lower/internal/producer/producer.go:442-447`, `case.proto:35-38`). Nothing reads
    them for values.
- **Concrete request values are realization constants**, for example `field(_.namespace) :=
  workerNamespace` (`model/README.md:1324-1345`).
  - An `Assignment` is a path plus an `Operand` (`ir.proto:1152-1156`). An `Operand` is a literal
    `ProtoValue`, an environment binding, the run id, a learned value, or the conditions `Present`,
    `Equal`, `All`, `Greater` and `Not` (`ir.proto:1236-1290`, `:1302-1317`).
  - Namespace and task queue are Profile-owned bindings (ART-13, `.plans/UMPIRE4_SPEC.md:347-354`;
    `model/temporal/realize/Kit.scala:59-62`). IDs are the run id (`:65`), and type names come from
    `perCase` (`:68`).
- **The lifter already reads ScalaPB descriptors.**
  - `messageDescriptor` loads a message's companion and takes `companion.scalaDescriptor`
    (`model/irgen/Realizations.scala:163-172`).
  - `selectorPath` walks a typed `Field[Root, V]` selector's lambda through that descriptor,
    including oneof members, to an IR path (`:235-315`).
  - `Action.schema` records `companion.scalaDescriptor.fullName` (`model/umpire/Action.scala:70-72`).
- **Typed realization conditions already exist, and they are a deep embedding.**
  - `Condition[Root]` builds data: a selector, an optional value, children
    (`model/umpire/realize/Typed.scala:73-105`).
  - The lifter turns it into `Operand` by reading the call tree by method name (`present`,
    `equal`/`greater`, `not`, `all`: `model/irgen/Realizations.scala:501-545`).
  - `equal` and `greater` build identical values (`Typed.scala:84-96`). fn-141.7 already tracks
    this: "`Condition.equal`, `greater`, `not` and `all` keep their operator; today two pairs build
    identical values" (`.flow/tasks/fn-141-shrink-the-ir-generator-one-description.7.md:16`).
- **Go re-checks paths and kinds against `protoregistry.GlobalFiles`**
  (`tools/umpire/lower/descriptor.go:37-48`; `model/SEMANTICS.md:781-786`).
- **Testpilot's expression language already has `AnyExpression`** (`proto/internal/temporal/server/api/testpilot/v1/expression.proto:22`, `:57`),
  but the IR `Operand` has no `Any`.
- **Exploration enumerates authored Scenario alternatives only, at most 4096**
  (`model/SEMANTICS.md:899-907`). The spec's class-member target (`.plans/UMPIRE4_SPEC.md:453-459`)
  is not implemented. No Model has an invalid-input class.

### Where validation knowledge already hides

- **The activity realization restates the deadline rule in a comment.** "The server refuses a
  start that sets neither a start-to-close nor a schedule-to-close deadline"
  (`model/temporal/features/activity/standalone/system/Realization.scala:72-84`). The rule is in
  `chasm/lib/activity/validator.go:171-194`, and nothing checks it.
- **Realizations write 13 RPC requests** (`grep METHOD_ model/temporal`):
  - StartActivityExecution, Pause, Unpause, RequestCancel, Terminate and Describe for activities;
  - Start, RequestCancel, Terminate and Describe for Nexus operations;
  - StartWorkflowExecution, DescribeWorkflowExecution and GetWorkflowExecutionHistory.

  They also write ten `Proto[...]` messages, such as `Failure`, `ApiCommand` and
  `StartOperationResponse`. `StartActivityExecutionRequest` alone has 22 fields.
- **fn-125 deliberately leaves limits unmodeled.** It defers ID lengths and blob sizes "until a
  Query needs one" (`.flow/specs/fn-125-represent-dynamic-configuration-in-the.md:134`). The spec
  is deferred after task 1 (`:193-195`; `MILESTONES.md` fn-125).

### Constraints the design must respect

| Source | Constraint |
| --- | --- |
| `.plans/UMPIRE4_VISION.md:93-105` | Knowledge only in Models; others mechanical; Temporal at the edges; one meaning, one source; fail closed |
| `.plans/UMPIRE_MODULES.md:31`, `:92-100` | The DSL names no Temporal concept (`TestFrameworkNamesNoTemporal`). So the validation Tag library may live in the framework, and `namespace` may not |
| `.plans/UMPIRE_MODULES.md:30`, `:52` | IR depends only on protobuf support; Testpilot imports nothing from `tools/umpire` or `model` |
| AUT-04/05 (`.plans/UMPIRE4_SPEC.md:263-268`) | Stable IDs; cross-language data, no callbacks. A condition is data, never a Scala lambda |
| PLN-02, ART-11 (`:304`, `:340-342`) | Same inputs and seed give the same Cases, byte for byte |
| SEM-19 (`:149-153`) | One word per concept (section 7) |
| `.plans/DSL_SIMPLIFICATION.md:65`, `:192` | No Scala annotations; the lifter reads `val`/`def` and constructed values |
| AGENTS.md | No new third-party libraries; only Go's stdlib `regexp/syntax` |
| `.plans/UMPIRE_CEL_SPIKE.md:3-39` | The IR keeps its own expressions; `Operand` "already fits CEL" |
| fn-125 "Declared, not defaulted" (`:91`), decision 3 (`:157`) | No component assumes a server default; a remote Profile that cannot state a required key fails closed |
| Owner, 2026-10-06 | The Model links to no server code: no `because`, citation or source path in the DSL, the IR or a Tag |

## 2. Prior art

| System | Named reusable constraint | Composition | Runtime parameters | Overlay | Input generation | Go code |
| --- | --- | --- | --- | --- | --- | --- |
| protovalidate | **Predefined rules**: an extension of `buf.validate.StringRules` with `(buf.validate.predefined).cel`, its value bound as `rule` ([docs](https://protovalidate.com/schemas/predefined-rules/)) | AND ([standard rules](https://protovalidate.com/schemas/standard-rules/)) | Fixed in the schema **[unverified absence]** | None | `example` field only ([validate.proto](https://raw.githubusercontent.com/bufbuild/protovalidate/main/proto/protovalidate/buf/validate/validate.proto)); FauxRPC, third party ([docs](https://fauxrpc.com/docs/protovalidate/)) | Runtime, with native Go plus a CEL fallback checked by a double conformance run ([blog](https://buf.build/blog/faster-protovalidate)) |
| protoc-gen-validate | None **[unverified]** | AND | None | None | None | Generated `Validate()`; in maintenance ([repo](https://github.com/bufbuild/protoc-gen-validate)) |
| XML Schema 1.1 | Simple types restricted by constraining facets: `length`, `minLength`, `maxLength`, `pattern`, `enumeration`, the inclusive and exclusive bounds, … ([Part 2 §4.3](https://www.w3.org/TR/xmlschema11-2/#rf-facets)) | Restriction only narrows the value space (same page) | None | n/a | n/a | n/a |
| Google AIPs | `field_info.format` ([proto](https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/field_info.proto)), `resource_reference` ([AIP-123](https://google.aip.dev/123)) | n/a | n/a | n/a | n/a | Descriptive only ([AIP-203](https://google.aip.dev/203)) |
| Smithy | Constrained named shapes; **traits are shapes**: `@trait` with a selector, the trait's value checked against its declared shape ([model](https://smithy.io/2.0/spec/model.html)) | Member traits supersede the target's | None | **`apply`** (same page) | Hand-written compliance tests ([spec](https://smithy.io/2.0/additional-specs/http-protocol-compliance-tests.html)) | smithy-rs newtypes plus violation enums ([RFC-0025](https://smithy-lang.github.io/smithy-rs/design/rfcs/rfc0025_constraint_traits.html)) |
| JSON Schema / OpenAPI 3.1 | `$ref`; `format` is an annotation unless asserted ([validation](https://json-schema.org/draft/2020-12/json-schema-validation)) | `$ref` siblings AND ([core](https://json-schema.org/draft/2020-12/json-schema-core)) | None | Overlay 1.0 ([spec](https://spec.openapis.org/overlay/v1.0.0.html)) | Schemathesis positive and negative modes; boundaries 1, 2, 3, 9, 10, 11 for min 2 / max 10 ([docs](https://schemathesis.readthedocs.io/en/stable/explanations/data-generation/)) | n/a |
| Kubernetes | Type-scoped formats, declarable in YAML ([validation-gen](https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/code-generator/cmd/validation-gen/validators)) | AND; CEL for exceptions | VAP `params` ([docs](https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/)) | Tags in source | n/a | validation-gen emits native Go behind a shadow mode ([KEP-5073](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/5073-declarative-validation-with-validation-gen)) |
| Hypothesis / rapid | `register_type_strategy`, `from_regex` ([docs](https://hypothesis.readthedocs.io/en/latest/reference/strategies.html)); rapid `StringMatching` ([pkg](https://pkg.go.dev/pgregory.net/rapid)) | n/a | n/a | n/a | Construction, plus filter, plus hand-written | n/a |
| CEL | n/a | n/a | n/a | n/a | Partial evaluation ([cel-go](https://github.com/google/cel-go)); a Z3 verifier yields counterexamples, untested as a generator ([verifier](https://github.com/cel-expr/cel-java/tree/main/verifier)) **[unverified]** | `EstimateCost` ([pkg](https://pkg.go.dev/github.com/google/cel-go/cel)) |

What transfers:

1. **Tags whose shape is declared, as in Smithy.** Meaning is attached by libraries and consumers.
   The IR checks the shape.
2. **A tagged Domain is a named bundle, as Smithy shapes, protovalidate predefined rules and XSD restricted
   types are.** Composition is tighten-only AND; loosening must be explicit.
3. **A closed, interpretable core library.** Length, range, pattern, enumeration, presence and
   size are generated by construction and compile to native Go. Anything else is
   generate-and-filter or hand-written.
4. **Symbolic parameters resolved per run** (VAP). protovalidate's `rule` cannot express
   `limit.maxIDLength`.
5. **Violate exactly one Tag at its boundary** (Schemathesis). Name the Tag violated (smithy-rs).
6. **Pin the regex dialect, the anchoring and the length unit.** The server's lengths are Go `len`
   bytes.
7. **Prove a generated validator on the generated corpus** (protovalidate's double run, K8s shadow
   mode).

## 3. Existing Scala libraries

None of these can be the core.

| Library | What it is | Why not the core |
| --- | --- | --- |
| **iron** | `String :| (Alphanumeric & MinLength[5])`. "Constraint parameters are held by the dummy type as type parameters, not constructor parameters"; `test` is `inline`; `RuntimeConstraint` avoids the inline summoning ([reference](https://iltotore.github.io/iron/docs/reference/constraint.html)) | The lifter reads values and `val`/`def` trees (`.plans/DSL_SIMPLIFICATION.md:178`). A constraint that exists only as a type gives it nothing to read. A dynamic-config bound cannot be a literal type |
| **iron-scalacheck** | `Arbitrary` per supported constraint, filtering otherwise ([module](https://iltotore.github.io/iron/docs/modules/scalacheck.html)) | Inherits the type-level form, and generates in Scala while Cases are produced in Go (`.plans/UMPIRE_MODULES.md:46`) |
| **refined** | Type-level predicates (`Int Refined Positive`), checked at compile time; `refineV` at runtime; `refined-scalacheck` ([repo](https://github.com/fthomas/refined)) | Same as iron |
| **scalapb-validate** | A generator that "uses the same validation rules provided by protoc-gen-validate", `[(validate.rules).string.email = true]` ([docs](https://scalapb.github.io/docs/validation/)); its docs do not mention protovalidate | Needs PGV annotations in `.proto` sources, which this repo does not have |
| **smithy4s** | Trait values reflected at runtime as schema hints, e.g. `.addHints(smithy.api.Length(Some(1), None))` ([schemas](https://disneystreaming.github.io/smithy4s/docs/design/schemas/)); constraint traits reified into refinements ([v0.19.13](https://github.com/disneystreaming/smithy4s/releases/tag/v0.19.13)) | The right model, value-level, but it brings a second schema system next to the descriptors and the IR (`.plans/UMPIRE4_VISION.md:101-102`) |
| **ScalaCheck** | `Gen`, `Arbitrary` | Relevant only if generation moved from Go to Scala. Go is the single evaluator (`model/README.md:93-94`) |

**Conclusion:** write a small tag mechanism and a validation Tag library ourselves, as values the
lifter reads. Borrow the library's names and meanings from protovalidate's standard rules
(`min_len`, `max_len`, `pattern`, `in`, `not_in`, `required` and its presence semantics) and from
XSD's constraining facets (`length`, `minLength`, `maxLength`, `pattern`, `enumeration`, the bounds,
and the rule that restriction only narrows).

## 4. Temporal validation inventory

`DC` = `common/dynamicconfig/constants.go`. `IA` = `InvalidArgument`. `MaxIDLength` =
`limit.maxIDLength`: global, default 1000, shared by namespace, task queue, IDs, types, signal name,
identity and request ID (`DC:527-532`).

| Concept | Rule | Static / dynamic | Enforced at | Error |
| --- | --- | --- | --- | --- |
| namespace | required | static | `common/rpc/interceptor/namespace_validator.go:376-380` | IA "Namespace not set on request." (`:40`) |
| namespace | `len ≤ MaxIDLength`; **no charset, regex or reserved names** | `limit.maxIDLength` | `namespace_validator.go:183`, `:191-195` (interceptor, before lookup); again `service/frontend/workflow_handler.go:6970-6976`, `service/history/api/create_workflow_util.go:302-304` | IA "Namespace length exceeds limit."; history: "Namespace exceeds length limit." |
| namespace (Register) | retention ≥ min; bad binaries ≤ N; duplicate | `system.namespaceMinRetention*` (`DC:241`, `:246`), `frontend.maxBadBinaries` (`DC:915`) | `service/frontend/namespace_handler.go:1291-1306`, `:512-519`, `:128-132` | IA; AlreadyExists |
| workflow id | required; `len ≤ MaxIDLength` | dynamic | `chasm/lib/workflow/validator.go:51-61`; `service/frontend/validators.go:15-28` | IA |
| run id | optional; `uuid.Validate` if set | static | `service/frontend/validators.go:22-26` | IA "Invalid RunId." |
| workflow / activity type | required; `len ≤ MaxIDLength` | dynamic | `workflow_handler.go:654-660`; `chasm/lib/activity/validator.go:102-117` | IA |
| activity id | required; `len ≤ MaxIDLength` | dynamic | `chasm/lib/activity/validator.go:102`, `:113` | IA "activityId exceeds length limit. Length=%d Limit=%d" |
| task queue | set; `len ≤ MaxIDLength`; no `/_sys/` on a root partition; no user `temporal-sys-per-ns-*`; promised UTF-8 and whitespace checks **not implemented** | dynamic | `common/tqid/task_queue_validator.go:93-146`; `common/primitives/task_queues.go:46-72` | IA |
| signal / update / query name | required; `len ≤ MaxIDLength` | dynamic | `workflow_handler.go:2315-2321`, `:5547`, `:3315` | IA |
| request id | empty is **auto-filled**; `len ≤ MaxIDLength` | dynamic | `workflow_handler.go:6948-6963`; `chasm/lib/activity/validator.go:374-376` | IA |
| identity | `len ≤ MaxIDLength` | dynamic | `workflow_handler.go:1094`; `chasm/lib/activity/validator.go:381` | IA |
| duration | same sign; ≥ 0; capped at 100 y | static | `common/primitives/timestamp/duration.go:12`, `:67-88` | IA |
| activity timeouts (cross-field) | start-to-close or schedule-to-close > 0; others filled in | static | `chasm/lib/activity/validator.go:151-214` | IA (`:194`) |
| retry policy | coefficient ≥ 1; max ≥ initial; attempts ≥ 0 | static | `common/retrypolicy/retry_policy.go:103-140` | IA |
| cron | parses; not combined with start delay | static | `common/backoff/cron.go:15-29` | IA |
| ID policies (cross-field) | forbidden combinations | static | `chasm/lib/workflow/validator.go:111-124`, `:204-206` | IA |
| priority | fairness key ≤ 64 bytes (**hard-coded**) | static | `common/priorities/priority_util.go:11`, `:51-62` | IA |
| search attributes | count, defined, type, sizes | `frontend.searchAttributes*` (`DC:940-950`), **plus server state** | `common/searchattribute/validator.go:79-232` | IA; Unavailable |
| memo | size | `limit.memoSize.*`, namespace (`DC:430-435`) | `create_workflow_util.go:256-268` | IA |
| payloads | rejected when size > warn **and** > error | `limit.blobSize.{warn,error}`, namespace (`DC:420-425`) | `common/util.go:604-633` | IA |
| payloads (worker answers) | oversize **converts to a failure** | same | `workflow_handler.go:1478-1500`, `:1678-1700` | none on the RPC |
| header | not enforced | none | `create_workflow_util.go:239` | none |
| links, callbacks | count, size, fields | `frontend.maxlinksPerRequest`, `frontend.linkMaxSize`, `system.maxCallbacksPerWorkflow`, `frontend.callback*` | `common/links/validator.go:29-131`; `common/callbacks/validator.go:94-204` | IA / Unimplemented |
| user metadata | sizes (CHASM only) | `limit.userMetadata*Size`, namespace (`DC:3752-3757`) | `chasm/lib/activity/validator.go:416-432` | IA |
| long-poll deadline | set; ≥ 2 s | static | `common/util.go:124-126`, `:638-676` | IA / FailedPrecondition |

Implications:

- **Nearly every limit is dynamic, and most hang off one global key.** A few keys are
  namespace-scoped (blob, memo, metadata, search attributes) (section 8).
- **Validation runs in layers, with different messages per layer.** Conformance compares status
  codes only (fn-139, `:172`; decision 11).
- **Some rules repair the request or convert the problem instead of rejecting** (auto-fill,
  capping, conversion to a failure). Only a rejecting Tag derives a class.
- **Some rules depend on server state.** For now each is a rejecting Tag that no consumer
  interprets, recorded as a Known Gap (decision 13).

## 5. Generating inputs from the validation library

| Library Tag | Valid by construction | Boundary members | Violating (breaks only this Tag) | Tractable? |
| --- | --- | --- | --- | --- |
| `required`, `notDefault` | any member of the other Tags | n/a | unset; zero value | yes |
| `length(min, max, unit)` | ASCII filler | min, min+1, max−1, max; multibyte runes | min−1, max+1 | yes, once the bound resolves (section 8) |
| `pattern(re2)` | walk `regexp/syntax` | shortest and longest matches | mutate, then check the compiled regexp rejects it | yes |
| `notPrefix`, `in`, `notIn` | avoid or pick | the prefix minus one character | the prefix plus a valid suffix; a listed literal | yes |
| `range`, duration Tags | interval members | edges | edges ± 1; mixed signs | yes |
| `maxBytes` (serialized size) | filler measured with `proto.Size` | N − 1, N | N + 1 | yes; large N is answered by section 8 |
| `format(name)` | a generator registered by name (uuid built in) | per generator | per generator | uuid yes; cron by hand |
| `requires(condition)` | enumerate satisfying assignments of its atoms | n/a | an assignment that falsifies only this condition, other Tags kept valid | yes for presence, equality and comparison with literals; otherwise a Known Gap |
| `unconstrained` | any value of the field's protobuf type, or unset | n/a | none (not rejecting) | yes |
| any Tag `values` does not interpret | — | — | — | reported; a Known Gap if the Tag rejects |

Rules:

- **Seeded and deterministic** (PLN-02, ART-11). Generated values are literals in the Case, so
  Testpilot does not change.
- **Each member names the class it realizes** (`valid`, or the Tag it violates) in Case provenance,
  as an extension of `AbstractionClaim` (`case.proto:35-38`).
- **Profile-owned fields** (namespace, task queue; ART-13) keep the Profile's value in the `valid`
  class and in every class that violates another field. Only their own violating classes vary
  them (decision 3).
  - A valid but unregistered namespace is `NotFound`.
  - A too-long one is rejected before lookup (`namespace_validator.go:183`).

## 6. Where tags live, and how they reach the IR

| Option | Source | Pros | Cons |
| --- | --- | --- | --- |
| A. Upstream proto options in `temporalio/api` | API protos | Shared with SDKs; standard tooling | Another repo's release cycle; static values only; no link to classes or Rejections |
| B. Sidecar overlay (textproto or YAML keyed by FQN) | Overlay file | No upstream dependency | Untyped names (the DSL forbids proto-name strings, `.plans/UMPIRE_MODULES.md:86-90`); disconnected from the Model |
| **C. Scala: framework mechanism and library, kit markers and Domains** | `model/` | Typed selectors; IDs; fn-125 keys; derived classes; exhaustiveness against descriptors | Restates rules that live in Go |
| D. C, plus a later import of or export to upstream options | `model/` | Interoperability without ownership | More machinery, built only when needed |

**Recommendation: C.** Upstream options are out for now (decision 2). The validation library keeps
protovalidate's names and meanings, so a later import or export stays mechanical.

## 7. Recommended design (sketches)

### Words (SEM-19)

| Word | Meaning |
| --- | --- |
| **Tag** | A name plus data, attached to a message, a field path (or its elements, map keys or map values) or a tagged Domain |
| **Tag Definition** | A Tag's declared name, data schema, targets, whether it rejects, and its default message template |
| **Domain** | A set of values. Finite Domains list Model values (`Finite[T]`). Tagged Domains describe protobuf values by Tags and are finite through their classes. Restricting a base tagged Domain ANDs its Tags |
| **Tag Set** | The tagged Domain of one message type: Tags on every field, recursing into every message it reaches, plus message-level Tags |
| **`unconstrained`** | The framework's no-op Tag |

Why `unconstrained` over the other candidates:

- `untagged` contradicts being a Tag.
- `free` already means "any action at every step" for a Scenario (`model/SEMANTICS.md:394-395`).
- `unconstrained` says what consumers do with the field: any value of its protobuf type, or unset.

Each Tag Definition name is itself a word under SEM-19. The lifter refuses two definitions of one
name. Synonyms (`length` beside `maxLength`) are a review matter.

### One Domain: finite and tagged

The owner resolved SEM-19 by unification, not renaming. The framework already has domains:

- an action's inputs "are finite domains; each assignment of them is one class"
  (`model/umpire/Action.scala:34`);
- `ActionDecl.domains` holds one `Finite[?]` per input (`:45`, `:82`);
- a token carries its domain (`:94`, `Input.domain` at `:149`);
- `model/umpire/Domain.scala` is the file of `Finite`.

A tagged Domain is the same concept with different members:

| | Finite Domain | Tagged Domain |
| --- | --- | --- |
| Values | Model values (enum cases, records, `UpTo[n]`) | Protobuf values: one field's (a field Domain such as `id`) or one message's (a Tag Set) |
| Given by | Listing (`Finite[T]`, derived through `Mirror`) | Tags (`required`, `length(max = maxIdLength)`, …) |
| Finite for the Model as | Its values | A Tag Set's single-fault classes: `valid` plus one per (field path, rejecting Tag) |
| One class stands for | One value | Every message whose only fault is the class's, or every valid message (an Abstraction Claim) |

**Inputs come from Tag Sets, not field Domains.** A field Domain (`id`, `namespaceName`) is a named
bundle a Tag Set reuses. It derives no input of its own. `input(startActivity)` is an ordinary
`Input[startActivity.Class]` whose `domain` is `startActivity.classes: Finite[startActivity.Class]`:

- `ActionDecl.domains` gets that `Finite`, as for any input.
- The checker builds action classes from it by the existing rule (`model/SEMANTICS.md:136-138`),
  for example `start-valid` and `start-activityId_length`.
- Step functions compare against its cases.
- The Quint export sees one more enum.
- What is new is a reference from the derived enum to the Tag Set, which only lowering,
  exploration and `values` read (IR below).

**Changes in `model/umpire`:**

- `Domain.scala` gains `sealed trait Domain[T]`. `Finite[T]` extends it, unchanged in name and
  use, so no Model changes.
- New `abstract class Tagged[V](tags: Tag*) extends Domain[V]`, for field Domains.
- `TagSet[M]` is the message form, with `type Class` and `classes: Finite[Class]`, derived when
  it is constructed.
- `ActionDecl` gains `tagged: List[Option[TagSet[?]]]` beside `tokens`.
- `isValid` and `violates` are sugar over the derived cases.
- `domains`, `Input.domain` and `Finite` keep their names, which now mean what the unified word
  says.

**The five spec sites, read with the one meaning.** Decision 16 adopts these rewordings; this note
does not edit those files.

| Site | Text | Fits? |
| --- | --- | --- |
| `.plans/UMPIRE4_SPEC.md:59` | "a connector joins two domains explicitly" (Capability) | **No.** Here "domain" means a subject area. Reword: "joins two components explicitly" |
| `:61` | "A hole in a mapping's source domain is an unmapped source" (Known Gap) | Yes: the set of values a mapping is defined over |
| `:189` | "their finite domains -- states, Actions, Model Outcomes, and Facts" (Model) | Yes: finite Domains |
| `:217` | "Each member of an input domain is a class" (Action) | Yes. Extend: "…; a Tag Set's members are its single-fault classes, each standing for the protobuf messages whose only fault, if any, is the class's" |
| `:224` | "The Fact domain is a Model's own type" (Fact) | Yes: a finite Domain |

`model/README.md:753` ("the domain roles `caller` and `handler`") uses the subject-area sense and
is reworded like `:59`. `model/SEMANTICS.md:286` and `:904-907` fit.

**fn-141 (decision 15: land after fn-141.9).** fn-141 exports declarations from constructed values
and lifts only function bodies (`.flow/specs/fn-141-shrink-the-ir-generator-one-description.md:49-57`,
R6 `:92`).

- Tag Sets, field Domains and their classes are declaration-level values, derived by running
  Scala. R3: declaration-level sugar "resolves by running" (`:89`). The exporter emits them with
  no lifter matcher.
- A class reference such as `startActivity.violation(_.activityId, lengthTag)` is a
  declaration-level value. A guard that reads it closes over it, and R8 binds a captured value of a
  finite Model type as the IR value it is (`:94`).
- `isValid` and `violates` inside a step function are function-level sugar, expanded generically
  (R4, `:90`).
- fn-141.9 exports "actions, inputs" (`MILESTONES.md` fn-141 table).
- `Condition`'s operator loss is fn-141.7's (section 1).

### Single-fault classes

**What a Tag Set derives.** A Tag Set's classes are:

- `valid`: every Tag on every reachable field holds;
- one class per (field path, rejecting Tag): that Tag fails at that path, and every other field is
  valid.

So a request has **1 + Σ rejecting Tags** classes: linear, never a cross product.

Paths and names:

- Paths reach through nested Tag Sets (`task_queue.name`), repeated elements (`links[*]…`, the
  fault in one element) and map keys or values.
- A recursive type (`Failure.cause`, reachable from `StartWorkflowExecutionRequest`) is entered at
  most once per path. Deeper positions are the same class, and the generator places the fault at
  the shallowest one.
- A message-level Tag (`requires`) has no path.
- Class names are the Scala selector path and the Tag's name joined by `_` (`activityId_length`,
  `taskQueue_name_notPrefix`, `requires`). These are identifiers in Scala, Quint and TLA+. The IR
  carries the structured path beside the name.

**Count for the sketch below.** `StartActivityExecutionRequest` derives 26 classes: `valid`, plus 25
rejecting Tags.

| Fields | Rejecting Tags | Count |
| --- | --- | --- |
| `namespace` | `required`, `length` | 2 |
| `identity` | `length` | 1 |
| `request_id` | `length` (its `required` only normalizes) | 1 |
| `activity_id` | `required`, `length` | 2 |
| `activity_type`, `activity_type.name` | `required`; `required`, `length` | 3 |
| `task_queue`, `task_queue.name` | `required`; `required`, `length`, `notPrefix` | 4 |
| five durations | `sameSign`, `nonNegative` each | 10 |
| `input` | `maxBytes` | 1 |
| message | `requires` | 1 |

**`grouped` is dropped.**

- Its only purpose was size, which single-fault removes.
- A Model that does not tell faults apart writes `!req.isValid` in one rule. Every violating class
  keeps its own Case, which is the coverage the classes exist for.
- An action that should only ever send a valid request declares no Tag Set input. Lowering still
  checks its literals against the Tag Set.

**Interaction with the action's other inputs.**

- A request input multiplies with the action's own finite inputs. For example, the activity's three
  `Timeout` inputs give 26 × 8 action classes.
- That is the existing catalog rule, and Query totals show it.
- A Model whose violating rows ignore the other inputs keeps them in one rule, `when(!req.isValid)`.

### Why generic Tags, and what that costs

**Gained:**

- The IR stays validation- and Temporal-agnostic.
- Semantic markers (`namespace`), documentation Tags, Driver hints or response checks arrive as
  Tag libraries without a schema change.
- The IR knows Temporal "only as declared names and payload types" (`.plans/UMPIRE4_VISION.md:55-60`).

**Lost:** the IR schema no longer fixes what `length` means, so two consumers could disagree.

**Settled (decision 10):**

1. **The core validation Tags' meanings are normative in `model/SEMANTICS.md`.** A "Tag library"
   section sits beside the IR's evaluation rules, so "one meaning, one source" holds
   (`.plans/UMPIRE4_VISION.md:101-102`).
2. **One Go interpretation.** `tools/umpire/values` owns the interpreters, and lowering,
   exploration and conformance use them.
3. **Fail closed.**
   - Every consumer lists the Tags it does not interpret.
   - An uninterpreted *rejecting* Tag on a field a Case writes is a Known Gap in that Case.
     Server-state rules are exactly this, for now (decision 13).
   - Any other uninterpreted Tag shows in `umpire-lint` and the inventory.

**Module map:**

- The IR row is unchanged.
- The DSL gains the mechanism and the validation library, with no Temporal words.
- `model/temporal/foundations` gains markers, shared field Domains, shared Tag Sets and keys; each feature gains its request Tag Sets and `owns`; `model/temporal/capabilities` gains `Validated`.
- The values module interprets only framework Tags.

**Class derivation reads `rejects`.**

- Only a Tag whose definition rejects derives a class. A marker, a normalizing Tag (decision 6:
  marked only) or `unconstrained` derives none.
- A rejecting Tag no consumer generates for still derives its class. Lowering reports that class's
  Cases `unsupported`, with a located reason.

### Framework (Temporal-agnostic, `model/umpire/Tags.scala`, `model/umpire/tags/Validation.scala`, sketch)

```scala
// The kind of one data field of a Tag Definition.
enum DataKind:
  case text, number, flag, enumName // a ProtoValue literal of that kind
  case condition // a Condition over the tagged message's fields
  case quantity // a number literal, or a Setting resolved per Case (section 8)
// Where a Tag may stand.
enum Target:
  case message, text, number, duration, bytes, messageField, each, key, value, domain

// A Tag Definition, named after its `val`. `rejects` makes a violated value one the system rejects
// with that Rejection (fn-139); only such Tags derive classes. `message` is the default template.
final class TagDefinition private[umpire] (/* fields, targets, rejects, message */):
  def apply(data: (String, Any)*): Tag
def definition(on: Target*)(fields: (String, DataKind)*): TagDefinition
extension (d: TagDefinition)
  def rejects(r: Rejection): TagDefinition
  def message(template: String): TagDefinition // placeholders checked at lift

// One use; `.message` overrides the template; `.normalizes` marks a repaired, accepted value.
final class Tag private[umpire] (/* definition, data, template, normalizes */):
  def message(template: String): Tag
  def normalizes: Tag

// model/umpire/Domain.scala: one concept, two forms. Finite[T] is today's trait, now a Domain.
sealed trait Domain[T]
trait Finite[T] extends Domain[T]:
  def values: IndexedSeq[T] // unchanged

// A field Domain: protobuf values of kind V, described by its Tags in order; given a base, the
// base's Tags first, ANDed.
abstract class Tagged[V](tags: Tag*)(using base: Option[Tagged[V]] = None) extends Domain[V]:
  def message(of: TagDefinition, template: String): Tagged[V] // overrides one rejecting Tag's template

// A Tag Set: the tagged Domain of message M, exhaustive and recursive, with single-fault classes.
final class TagSet[M <: GeneratedMessage] private[umpire] (/* field tags, message tags */)
    extends Domain[M]:
  type Class
  def classes: Finite[Class] // valid, then one per (path, rejecting Tag); derived on construction
  def valid: Class
  def violation[V](at: M => V, t: TagDefinition): Class // refused on construction if absent
  def violation(t: TagDefinition): Class // a message-level Tag
def tag[M <: GeneratedMessage](lines: TagLine[M]*): TagSet[M] // `field`, `each`, `keys`, `values`

// An input over a Tag Set is an ordinary input over its classes.
def input[M <: GeneratedMessage](s: TagSet[M]): Input[s.Class]
extension [M <: GeneratedMessage](s: TagSet[M])
  def isValid(x: s.Class): Boolean // sugar: x == s.valid
  def violates[V](x: s.Class, at: M => V, t: TagDefinition): Boolean // sugar: x == s.violation(at, t)

// The validation library: names and meanings from protovalidate and XSD, normative in SEMANTICS.md.
val unconstrained = definition(Target.text, Target.number, Target.duration, Target.bytes,
  Target.messageField, Target.each, Target.key, Target.value)()
val requiredTag = definition(Target.text, Target.messageField)()
  .rejects(Rejection.invalidArgument).message("{field} is not set.")
val lengthTag = definition(Target.text)("min" -> DataKind.quantity, "max" -> DataKind.quantity,
  "unit" -> DataKind.enumName)
  .rejects(Rejection.invalidArgument).message("{field} length exceeds limit of {max}.")
val requiresTag = definition(Target.message)("condition" -> DataKind.condition)
  .rejects(Rejection.invalidArgument).message("{message} violates a required combination.")
def required: Tag = requiredTag()
def length(min: Quantity = 0L, max: Quantity, unit: Unit = Unit.bytes): Tag =
  lengthTag("min" -> min, "max" -> max, "unit" -> unit)
def requires(c: Condition[?]): Tag = requiresTag("condition" -> c)
// … pattern, notPrefix, in, notIn, range, count, maxBytes, format, sameSign, nonNegative

// Conditions: realize's typed Condition, plus `any` and `exactlyOne`.
def any[R](first: Condition[R], rest: Condition[R]*): Condition[R] // lifts to Operand.any
def exactlyOne[R](cs: Condition[R]*): Condition[R] = // sugar
  any(cs.indices.map(i => all(cs(i), cs.patch(i, Nil, 1).map(not)*))*)
```

**Conditions are a deep embedding.**

- A `Condition` is data the lifter reads, not a closure.
- `r => r.startToCloseTimeout.nonEmpty || r.scheduleToCloseTimeout.nonEmpty` has type `Req =>
  Boolean`, not `Condition[Req]`, so it does not compile where a Tag expects a condition.
- Field selectors are restricted lambdas that the lifter reads through descriptors
  (`model/irgen/Realizations.scala:235-315`).

**Why `Operand` and not the Model's `Expr`:**

- Tags speak about protobuf field paths. `Operand` already types its paths and literals against
  descriptors at lowering (`model/SEMANTICS.md:781-786`).
- `Expr` speaks about the Model's finite types, and its values enter state keys and fingerprints.
- Evaluating `Expr` over protobuf messages would need the value conversion the CEL spike measured
  at about a third of evaluation time (`.plans/UMPIRE_CEL_SPIKE.md:20-22`).

**`Any` is a first-class `Operand` node, not sugar for `not(all(not …))`.**

- Testpilot already has `AnyExpression`, so lowering maps one to one.
- Diagnostics and templates read a disjunction.
- Every `Operand` consumer gains one case: `tools/umpire/realization/operand.go`, `payload.go`,
  `validate_realization.go`, `tools/umpire/lower/realization.go`, `tools/umpire/conformance/guard.go`,
  `tools/umpire/lint/api.go`.
- `exactlyOne` stays sugar.

### Foundations and a feature (Temporal, sketch)

```scala
// Layout after fn-142: foundations/ (taskqueue, plus tags and common below), actors/ (worker,
// client), model/temporal/Bounds.scala.
// ---- model/temporal/foundations/tags/Tags.scala (kit Tag Definitions, settings, shared Domains)
// fn-125.5's typed keys, pulled forward (section 8).
val maxIdLength = DynamicSetting.int("limit.maxIDLength", scope = Scope.global)
val blobSizeError = DynamicSetting.bytes("limit.blobSize.error", scope = Scope.namespace)

// Markers: no data, not rejecting. Consumers that care read them; `values` lists them as ignored.
val namespace = definition(Target.text)()
val identifier = definition(Target.text)()

// Field Domains: named bundles that Tag Sets reuse.
object id extends Tagged[String](identifier(), required, length(max = maxIdLength))
object namespaceName extends Tagged[String](namespace(), required, length(max = maxIdLength))
object taskQueueName extends Tagged[String](notPrefix("/_sys/"))(using Some(id))
object requestId extends Tagged[String](length(max = maxIdLength), required.normalizes)
object timeout extends Tagged[Duration](sameSign, nonNegative)

// ---- model/temporal/foundations/taskqueue/TaskQueue.scala (beside the task-queue entity)
val taskQueue = tag[TaskQueue](
  field(_.name) is taskQueueName,
  field(_.kind) is unconstrained,
  field(_.normalName) is unconstrained
)
// ---- model/temporal/foundations/common/Common.scala: messages more than one feature reaches.
// Tag Sets recurse: every message a request reaches has its own, field by field.
val activityType = tag[ActivityType](field(_.name) is id)
val payloads = tag[Payloads](each(_.payloads) is unconstrained)
val payload = tag[Payload](
  keys(_.metadata) is unconstrained,
  values(_.metadata) is unconstrained,
  field(_.data) is unconstrained,
  each(_.externalPayloads) is unconstrained
)
val externalPayload = tag[Payload.ExternalPayloadDetails](field(_.sizeBytes) is unconstrained)
val header = tag[Header](keys(_.fields) is unconstrained, values(_.fields) is unconstrained)
// … RetryPolicy, SearchAttributes, Memo, UserMetadata, Priority, Callback and its variants, Link
// and its variants, OnConflictOptions: each tagged field by field.

// ---- model/temporal/features/activity/standalone/Standalone.scala (the feature owns the RPC)
object exports:
  val ir = irFile("activity-standalone")
  val rpcs = owns(
    METHOD_START_ACTIVITY_EXECUTION, METHOD_DESCRIBE_ACTIVITY_EXECUTION,
    METHOD_PAUSE_ACTIVITY_EXECUTION, METHOD_UNPAUSE_ACTIVITY_EXECUTION,
    METHOD_REQUEST_CANCEL_ACTIVITY_EXECUTION, METHOD_TERMINATE_ACTIVITY_EXECUTION /* , … */
  )

// Every field of StartActivityExecutionRequest (22), or the lift fails naming the missing ones.
val startActivity = tag[StartActivityExecutionRequest](
  field(_.namespace) is namespaceName,
  field(_.identity) is length(max = maxIdLength),
  field(_.requestId) is requestId,
  field(_.activityId) is id.message(lengthTag, "ActivityId exceeds length limit of {max}."),
  field(_.activityType) is required, // inside: ActivityType's Tag Set
  field(_.taskQueue) is required, // inside: TaskQueue's Tag Set
  field(_.scheduleToCloseTimeout) is timeout,
  field(_.scheduleToStartTimeout) is timeout,
  field(_.startToCloseTimeout) is timeout,
  field(_.heartbeatTimeout) is timeout,
  field(_.startDelay) is timeout,
  field(_.retryPolicy) is unconstrained, // inside: RetryPolicy's Tag Set
  field(_.input) is maxBytes(blobSizeError), // inside: Payloads' Tag Set
  field(_.idReusePolicy) is unconstrained,
  field(_.idConflictPolicy) is unconstrained,
  field(_.searchAttributes) is unconstrained,
  field(_.header) is unconstrained,
  field(_.userMetadata) is unconstrained,
  field(_.priority) is unconstrained,
  each(_.completionCallbacks) is unconstrained,
  each(_.links) is unconstrained,
  field(_.onConflictOptions) is unconstrained,
  requires(any(present(_.startToCloseTimeout), present(_.scheduleToCloseTimeout)))
)

// ---- model/temporal/capabilities/Validate.scala, beside the other laws
// The kit's law (decision 7): a violating class is rejected with its Tag's Rejection and changes
// no state. A machine declaring Validated on an action with a Tag Set input gets it, as Closable
// brings closedIsRejectedUniformly.
val Validated = CapabilityKind(/* the action and its Tag Set input */)
```

### Exhaustive tagging (decision 9)

**Checked at lift, not by a macro.**

- For each `tag[M]`, the lifter loads `companion.scalaDescriptor` the way `messageDescriptor`
  does (`model/irgen/Realizations.scala:163-172`), and compares its fields with the Tag Set's
  selectors (`selectorPath`, `:235-315`).
- It fails with `StartActivityExecutionRequest is not tagged exhaustively: retry_policy, links[*]
  (tag each, or mark it unconstrained)`.
- It then recurses. Every message type reachable through any field needs its own Tag Set, or it
  fails with `TaskQueue, reached from StartActivityExecutionRequest.task_queue, has no Tag Set`.
- The check runs in `make umpire-check-model`. A second `tag[M]` for one `M` is refused.

**Recursion everywhere; no whole-value Tag Sets.**

- `Payloads`, `Payload`, `Header`, `Memo` and `SearchAttributes` are tagged field by field like any
  message, with `unconstrained` where there is nothing to say.
- A Tag on a message-typed field speaks about the field: presence, or serialized size with
  `maxBytes`. Its type's Tag Set speaks about the inside. Both always apply, ANDed.
- `unconstrained` on a message-typed field means the field itself is arbitrary or unset. It never
  excuses the type from having a Tag Set.

**Size of the obligation.** The 13 requests realizations write reach **62 message types with 281
fields**, excluding `google.protobuf` types. That is a descriptor walk over the linked
`go.temporal.io/api` registry on 2026-10-06. Each type is tagged once and reused.

**Edge cases:**

| Case | Rule |
| --- | --- |
| oneof | Each member is a field and is tagged. Protobuf enforces at most one; `requires(exactlyOne(...))` states "exactly one" |
| repeated | `field(_.xs)` tags the collection (`count`). `each(_.xs)` tags the element, and a message element's type has its own Tag Set |
| map | `keys(_.m)` and `values(_.m)`; a message value's type has its own Tag Set |
| deprecated | Tagged like any field, since it is still on the wire. `values` leaves it unset by default, reading the descriptor's `deprecated` option |
| well-known types | `Duration`, `Timestamp`, the wrappers and `Empty` are leaves, tagged as values (`sameSign`, `nonNegative`) |
| recursive types | Tagged once. Classes enter a type at most once per path (single-fault classes, above) |

**Scope:** every request message of an RPC a realization uses, recursively (section 7, Ownership). The ten `Proto[...]`
messages and responses are later.

**Upstream:** when `go.temporal.io/api` adds a field to any reachable message, the gate fails until
someone tags it. That forcing function is intended.

### Ownership: source, IR files and used RPCs (decisions 18-20)

**Source location follows ownership.**

| What | Where |
| --- | --- |
| A used RPC's request Tag Set (later its response's) | The package that owns the RPC, e.g. `model/temporal/features/activity/standalone/` for `StartActivityExecution` |
| A message type only one package's Tag Sets reach | That package |
| Shared field Domains (`namespaceName`, `id`, `taskQueueName`), shared message Tag Sets (`TaskQueue`, `Payloads`, `Header`, …), the kit's Tag Definitions and markers, the settings | `model/temporal/foundations/`, in subfolders by subject (`foundations/taskqueue`, `foundations/common`, `foundations/tags`) |
| The framework's Tag Definitions (the validation library, `unconstrained`) | `model/umpire` |
| The `Validated` capability law | `model/temporal/capabilities`, beside the other laws (`.plans/UMPIRE_MODULES.md:36`) |

**`model/temporal/shared/` is renamed `model/temporal/foundations/`** (decision 21). fn-142
performs it, with its own layout:

- `foundations/` holds the task queue (`foundations/taskqueue`), plus the shared Domains, Tag Sets,
  markers and settings keys that this design adds.
- `actors/` holds the worker (`actors/worker`) and the client (`actors/client/Client.scala`).
- `Bounds.scala` sits in `model/temporal/`.

Notes on the move:

- Today `shared/` holds what features share: the task-queue and worker entities and `Bounds.scala`
  (`.plans/UMPIRE_MODULES.md:32`, `:37`).
- Two homes for shared Tag content would make "where does a shared thing go" a judgment call per
  change. With one home, the `TaskQueue` Tag Set sits beside the task-queue entity in
  `foundations/taskqueue`.
- **Cost:** Definition IDs are fully qualified Scala names, and `Machine.family` is the declaring
  package (`model/irgen/Context.scala:318-326`, `ir.proto:355-357`). The move therefore changes IDs
  exactly by the package mapping: `temporal.shared.taskqueue` → `temporal.foundations.taskqueue`,
  `temporal.shared.worker` → `temporal.actors.worker`. IR files, Cases, fixtures and the canary Case
  change with them, in one regeneration batch, as fn-126's renames did. That batch is accepted and
  is its own phasing step.
- The module map's Models row (`.plans/UMPIRE_MODULES.md:32`), the task-queue row (`:37`) and the
  IR generator's structure lint follow the new paths.

**IR files follow source.**

- `model/ir/foundations.json` carries the shared Tags.
- Each feature's IR file carries its own Tags and lists `foundations` in `uses`.
- A feature whose realization uses an RPC another package owns adds that owner's IR file to `uses`.
  Example: the Nexus caller uses `StartWorkflowExecution`, which the workflow package owns. That is
  a data reference, not a Scala import, so the module map's rule against features importing features
  holds.

**Obligations apply to used RPCs only.**

- An RPC is **used** when a realization writes it as a request (`rpc`, `readUntil`, `await`) or
  reads its response (`Recorded.read`, `Recorded.single`).
- Each used RPC needs:
  - exactly one owning package, declared by `owns(METHOD_…, …)` in that package's `object exports`
    beside its `irFile`, with the generated gRPC method constants Models already import
    (`.plans/UMPIRE_MODULES.md:85-90`);
  - an exhaustive, recursive Tag Set for its request, and later for its response.
- An unused RPC needs nothing.
- For context only: WorkflowService has 123 RPCs, OperatorService 12 (from the linked
  `go.temporal.io/api` service descriptors, 2026-10-06), and AdminService 46
  (`proto/internal/temporal/server/api/adminservice/v1/service.proto`, internal). Coverage is
  whatever realizations use.

**Gate checks**, in the gate's lint step (fn-141.1 moves the lints there):

- **Every used RPC is owned exactly once and tagged.** The check walks each realization's commands
  and evidence for the methods they name, and resolves each method through the service descriptors
  as the lifter already does (`generated.scalaDescriptor.services`, `model/irgen/Realizations.scala:206`).
  It fails naming the realization, the RPC and what is missing, for example:
  `ActivityRealization uses WorkflowService.StartActivityExecution, whose request is not tagged
  exhaustively: retry_policy, links[*]`, or `NexusCallerRealization uses
  WorkflowService.StartWorkflowExecution, which no package owns`.
- **Every message type reachable from a used RPC has exactly one owning Tag Set.**
  - A second `tag[M]` anywhere is refused.
  - A package's Tag Sets may reach only its own types and foundations types.
  - A type two packages reach must move to foundations.
- **New fields are forcing functions.** When `go.temporal.io/api` adds a field to a message a used
  RPC reaches, the gate fails until someone tags it. Starting to use a new RPC likewise requires its
  owner and Tag Set first.

**Packages that own today's used RPCs.** The 13 used RPCs leave three owning packages, one of them
new:

| Package | Used RPCs | Used by |
| --- | --- | --- |
| `features/activity/standalone` | `StartActivityExecution`, `DescribeActivityExecution`, `PauseActivityExecution`, `UnpauseActivityExecution`, `RequestCancelActivityExecution`, `TerminateActivityExecution` | its own realization |
| `features/nexus/standalone` | `StartNexusOperationExecution`, `DescribeNexusOperationExecution`, `RequestCancelNexusOperationExecution`, `TerminateNexusOperationExecution` | its own realization |
| `features/workflow` (created now, ownership-only: a feature file holding only `exports` and the request Tag Sets; decision 22) | `StartWorkflowExecution`, `DescribeWorkflowExecution`, `GetWorkflowExecutionHistory` | the Nexus caller (`features/nexus/workflow/Realization.scala`) |

Worker-side answers (`Failure`, workflow commands, Nexus replies) travel through the SDK worker,
not as realization RPCs. Their messages are the "ten written `Proto[...]` messages", which come
later.

A future mapping to CODEOWNERS is possible; out of scope.

### Message templates

**Where they come from.**

- A Tag Definition gives the default template (`lengthTag.message("{field} length exceeds limit of
  {max}.")`).
- A use overrides it, and so does a field Domain for one of its rejecting Tags
  (`id.message(lengthTag, "...")`).

**Placeholders are closed and checked at lift.** They are the definition's data fields (`{max}`,
`{pattern}`), `{field}`, `{message}` and `{value}`. An unknown placeholder fails the lift. A
quantity placeholder interpolates the value the Profile states for its Setting (section 8).

**The IR holds a parsed template**, a list of literal and placeholder parts. Go never parses text,
and the placeholder check happens once, at lift.

**Consumers (decision 11):** the values checker's diagnostics and a future codegen. Conformance
compares status codes only, never messages.

**fn-139 (decision 12).** fn-139's `rejects(r).because(text)` is renamed `.message(template)`, with
this template form (`.flow/specs/fn-139-actor-grouped-rules-per-rpc-actions.md:55`). The Tag
Definition's `rejects(r)` picks the `Rejection`, which fixes the status code through fn-139.8's
table. The template is the message within it.

**Templates restate expected behavior and link to no code**, consistent with the no-`because` rule.

### Connecting Tag Sets to the Model (sketch)

```scala
val req = input(startActivity) // Input[startActivity.Class]: valid and 25 violating classes
val start = action(client).input(req)
on(client.start) {
  when(startActivity.isValid(req)) ~> effects.schedule
  when(!startActivity.isValid(req)) ~> rejects(Rejection.invalidArgument) // fn-139
}

// A Model that tells one fault apart names its class, a declaration-level value:
val tooLongId = startActivity.violation(_.activityId, lengthTag) // class `activityId_length`
// … when(req == tooLongId) ~> …, or the sugar startActivity.violates(req, _.activityId, lengthTag)
```

**Derivation and naming.**

- The classes are `valid`, then one per (path, rejecting Tag) in field order, depth first, with
  message-level Tags last. They are computed when the Tag Set is constructed.
- Action class keys follow the existing rule (`start-valid`, `start-activityId_length`).
- The Model writes no class names and no mapping.
- The realization writes the request once, `member(req)`, and lowering fills every tagged field
  from the class.

**Trade-off.** Class names come from the Tag Set, so tagging a new field or renaming a selector
renames or adds classes, Query totals and IDs in every Model with that input. The count stays
linear in rejecting Tags.

**Abstraction Claims.**

- Each class is a claim whose members its Tags define, with the Model writing no example.
- Lowering takes the canonical member from `values`: for `valid`, every field at a valid value;
  for `activityId_length`, an ID of length `max + 1` with every other field valid. It records that
  member as the example.
- Exploration's class-member targets draw boundaries, then seeded samples, within the class.
- A divergent member splits the class (`.plans/UMPIRE4_SPEC.md:247-250`, `:453-459`).
- Profile-owned fields stay the Profile's except in their own violating classes (decision 3).

**`Validated` (decision 7).** The kit's capability law says every violating class's row rejects
with its Tag's `Rejection` and keeps the state, for every machine that declares the capability
(`.plans/SEMANTIC_PROTOCOLS.md:95-150`).

- The checker proves it over the table.
- A Run is checked on the status code through fn-139.8, and on no state change through the
  evidence the Model's Facts already name.

**First increment, with no Model change.** Lowering checks every literal operand against the Tag
Sets and refuses a violating one, naming the Tag. For example, a start with neither deadline is
refused with `startActivity.requires`.

### IR additions (sketch, `ir.proto`)

**Placement (decision 1, refined by decision 18).** IR files follow source.

- A foundations IR file, `model/ir/foundations.json`, holds the kit's Tag Definitions, the shared
  field Domains, the shared message Tag Sets and the settings. It is the kit-level file.
- Each feature's IR file holds its own Tag Sets, the requests of the RPCs it owns, and names
  `foundations` in `uses`.
- A feature file keeps the derived class enum in its own `types`, so the interpreter, the checker
  and the Quint export still read a self-contained file.
- Lowering, exploration and `values` load the files a feature `uses`. They check that each derived
  enum matches its Tag Set's derivation and its recorded fingerprint.

```proto
message Model {
  // ... fields 1-16 unchanged
  repeated TagDefinition tag_definitions = 17; // foundations: the kit's; the framework's are lifted there too
  repeated Domain domains = 18; // field Domains: shared ones in foundations, a feature's own here
  repeated TagSet tag_sets = 19; // shared message types in foundations; a feature's requests here
  repeated SettingDeclaration settings = 20; // fn-125.5's keys; foundations only
  repeated string uses = 21; // the IR files this one references, by IR file name (`foundations`)
}

// A Tag's declared shape. The IR gives it no meaning; model/SEMANTICS.md's Tag library section
// gives the framework Tags theirs.
message TagDefinition {
  string name = 1; // e.g. `umpire.tags.length`, `temporal.tags.namespace`
  Position position = 2;
  repeated DataField fields = 3;
  repeated TagTarget targets = 4;
  Value rejects = 5; // set: rejected with this Rejection (fn-139), and it derives a class
  Template message = 6; // the default
}
message DataField {
  string name = 1;
  enum Kind {
    KIND_UNSPECIFIED = 0;
    KIND_TEXT = 1;
    KIND_NUMBER = 2;
    KIND_FLAG = 3;
    KIND_ENUM_NAME = 4;
    KIND_CONDITION = 5; // a condition Operand over the tagged message's paths
    KIND_QUANTITY = 6; // a number literal, or a `setting` Operand
  }
  Kind kind = 2;
  bool repeated = 3;
  bool optional = 4;
}
enum TagTarget {
  TAG_TARGET_UNSPECIFIED = 0;
  TAG_TARGET_MESSAGE = 1;
  TAG_TARGET_FIELD = 2;
  TAG_TARGET_ELEMENT = 3;
  TAG_TARGET_MAP_KEY = 4;
  TAG_TARGET_MAP_VALUE = 5;
  TAG_TARGET_DOMAIN = 6;
}

// One use: a definition and its data, each field one Operand the definition's schema types.
message Tag {
  string definition = 1;
  string name = 2; // stable ID
  Position position = 3;
  repeated DataValue data = 4;
  Template message = 5; // an override, or unset
  bool normalizes = 6; // marked only (decision 6): accepted and repaired, derives no class
}
message DataValue {
  string field = 1;
  Operand value = 2;
}

// A field Domain: a named bundle of Tags over one protobuf value kind.
message Domain {
  string name = 1;
  Position position = 2;
  repeated string bases = 3;
  repeated Tag tags = 4;
}

// A Tag Set: the tagged Domain of one message type, exhaustive; inner message types have their own.
message TagSet {
  string name = 1; // e.g. `temporal.tags.startActivity`
  string message = 2;
  Position position = 3;
  repeated Tag message_tags = 4;
  repeated FieldTags fields = 5;
}
message FieldTags {
  string path = 1;
  TagTarget target = 2; // FIELD, ELEMENT, MAP_KEY or MAP_VALUE
  repeated string domains = 3;
  repeated Tag tags = 4;
}

// A parsed message template.
message Template {
  repeated TemplatePart parts = 1;
}
message TemplatePart {
  oneof part {
    string text = 1;
    string data = 2; // a data field of the Tag's definition
    Builtin builtin = 3;
  }
  enum Builtin {
    BUILTIN_UNSPECIFIED = 0;
    BUILTIN_FIELD = 1;
    BUILTIN_MESSAGE = 2;
    BUILTIN_VALUE = 3;
  }
}

// Operand gains two members (fields 13 and 14 of its oneof):
//   Any any = 13;          // holds when some operand does, read left to right
//   string setting = 14;   // a SettingDeclaration's key: the value the Case's settings give it
message Any {
  repeated Operand operands = 1;
}

// The derived classes are an ordinary enum in the feature file (`ir.proto:63-70`), linked to the
// feature's request Tag Set (decision 17):
message Enum {
  repeated Case cases = 1;
  string domain = 2; // new: the Tag Set these cases are classes of; empty for a finite Domain
  string domain_fingerprint = 3; // new: that Tag Set's fingerprint, covering the foundations Tag Sets it reaches
}
message Case {
  string name = 1; // `valid`, `activityId_length`, `requires`
  repeated Field fields = 2;
  Violation violates = 3; // new: unset for `valid`
}
message Violation {
  string path = 1; // the IR field path, e.g. `task_queue.name`; empty for a message-level Tag
  string tag = 2; // the Tag's stable ID
}
// Action.inputs stay `repeated Param` (ir.proto:331): a Tag Set input is a Param whose TypeRef names
// the derived enum. There is no IR `Input` message, and none is added.
```

**Why `Operand` for the data**, rather than bare `ProtoValue` or the Model's `Value`:

- `Operand` already unifies a `ProtoValue` literal with the condition nodes, and gains `setting`.
  The schema says which kind is allowed where.
- A bare `ProtoValue` would need a parallel oneof.
- `Value` is the Model's finite form: `TypeRef` has no text type, and its values enter keys and
  fingerprints.
- A repeated data field is several `DataValue`s of one name.

**Admission adds these checks (sketch):**

- every Tag names a definition, and its data matches the schema;
- targets, paths, kinds and literals fit the descriptors;
- templates name only declared placeholders;
- Tag Sets are exhaustive, recursive, and unique per message;
- a feature file's `uses` resolve;
- an enum with a `domain` has exactly the cases the Tag Set derives, in order, with no fields, its
  `violates` and its fingerprint matching;
- an input of such a type carries no hand-written example;
- `setting` Operands name declared settings, and only in quantity fields;
- conditions stay within presence, equality and comparison.

**Fingerprints.**

- A Tag Set's fingerprint covers its Tags, its definitions and the Tag Sets it reaches.
- Through `domain_fingerprint`, it enters the derived enum's fingerprint and so every input, step
  and Query over it.
- The interpreter, checker and Quint export ignore `domain`, `domain_fingerprint` and `violates`,
  and read an ordinary enum.
- Tag Sets used only for the literal check fingerprint nothing, as with `ApiBehavior`
  (`ir.proto:738-744`).
- Templates are not behavior (decision 11).

### Go module (`.plans/UMPIRE_MODULES.md` row, sketch)

| Module | One job | Public interface | Permitted dependencies |
| --- | --- | --- | --- |
| Values, `tools/umpire/values` | Interpret the framework's Tag library: check messages against Tag Sets and generate class members. | `Admit(kit, model)`; `Check(tagSet, message, settings) []Violation` (by path and Tag, with the interpolated message); `Members(tagSet, class, settings, seed)` (a whole request); `Ignored(kit) []TagUse`; `Register(format, generator)` | Umpire IR, `protoreflect`, stdlib `regexp/syntax`, interp for `Operand`; no Testpilot, no Temporal names |

Consumers:

- `lower`: members, the literal check, Known Gaps for ignored rejecting Tags, required settings;
- `explore`: class-member targets;
- `lint`: ignored Tags.

Testpilot is untouched. It is a deep module behind a small interface, testable on fixtures (MOD-08).

## 8. Run-time bounds and dynamic configuration (fn-125)

### Bounds reference fn-125's keys (decision 4)

A quantity in a Tag's data is a literal or a `setting` Operand naming a kit `DynamicSetting`. That
is fn-125 Part B's typed declaration: key, codec, scope, and a Go test pinning it to
`common/dynamicconfig/registry.go` (`.flow/specs/fn-125-represent-dynamic-configuration-in-the.md:31`).
It lives in foundations and its IR file. One declaration, no second registry.

**Pulled forward:**

- **fn-125.5, whole:** typed keys with scope, and the registry pin test.
- **From fn-125.6:** required settings with an origin, and Part C's union and conflict rule
  (`:33`).
- **The `atMost`/`atLeast` relations** that R10 defines (`:124`), carried in required settings.

Everything else in fn-125 stays deferred: Model settings, `under`, the Nexus encoding, and the
preparation checks of fn-125.9 beyond what relations need.

### Resolution: lowering records relations in the Case (decision 5)

- **Lowering resolves each bound to a relation, not an exact value.** A valid member of length
  *n* requires `maxIDLength atLeast n`; a violating member of length *n* requires `maxIDLength
  atMost n − 1`. These enter `Program.required_settings` with origin "a Tag bound a generated value
  rests on", a fourth origin beside Part C's three.
- **Remote Profiles must state the value** or the Case is `PreparationUnavailable` (decision 3 of
  fn-125, `:157`), never vacuous. That fits "Declared, not defaulted" (`:91`), SEM-16 and EVD-01:
  the Case stays literal, and Testpilot does not change.
- **QLF-01** (`.plans/UMPIRE4_SPEC.md:523-524`): a limit that decides whether `max + 1` is rejected
  changes behavior, so the Case records it rather than leaving it to the Profile.
- **The alternative was rejected.** A symbolic value resolved by Testpilot at preparation (`setting
  ± offset` plus a filler) would grow Testpilot and the Case format.

**Member size (a design choice, not a decision).**

- `values` picks boundaries from the key's registry default, so most Cases run on unconfigured
  servers.
- For `maxBytes` it picks a small *n*, which requires `limit.blobSize.error atMost n − 1`. Blob
  Cases then run where the Profile states such a limit, locally through R9's harness (`:123`), and
  are `PreparationUnavailable` elsewhere.
- That keeps 2 MiB values out of checked-in Cases.

### Scope

| Scope | Example keys | How a Case's requirement resolves |
| --- | --- | --- |
| global | `limit.maxIDLength` | One value for the cluster |
| namespace | `limit.blobSize.error`, `limit.memoSize.error`, `limit.userMetadata*Size` | For the namespace the Case's namespace role binds to. The requirement names the symbolic binding (`temporal.worker.namespace`, `model/temporal/realize/Kit.scala:27`), and preparation resolves it to the physical namespace from the Profile (ART-13) |
| task queue | (none in the inventory) | Likewise, through the task-queue role |

Testpilot's `RequiredSetting` is only `key` and `value` today
(`proto/internal/temporal/server/api/testpilot/v1/program.proto:36-39`). Relations and scoped keys
add a relation and a binding reference to it.

### Everything that reads the resolved value

- **Message templates:** `{max}` interpolates the value the Profile states, in diagnostics.
- **The Go checker:** `Check` takes the settings as an argument, never a default.
- **Exploration members:** each member records the relation it rests on.

### Conflicts

Lowering takes the union of every origin, as Part C requires (`:33`): Tags, API preconditions,
bound assumptions and a Query's valuation. Requirements that cannot all hold refuse the Case,
naming both origins. For example, under single-fault a violating ID of length 1001 (`atMost 1000`)
beside another field's valid member of length 1001 (`atLeast 1001`) on the same key. Nothing is
resolved by "last wins".

## 9. Go validation codegen

Not planned (decision 14). The IR keeps it possible:

- Tag Definitions, normative library meanings, `rejects`, parsed templates and `setting`
  quantities;
- the values module's interpreters to reuse;
- K8s-style shadow mode to prove it.

Any spec rule letting a Generated View become server code is deferred with it.

## 10. Phasing

Starts after fn-141.9 (decision 15).

1. Tag mechanism in the framework and IR: definitions, Tags, field Domains, Tag Sets with
   single-fault classes, templates, `Any`, `setting`, `uses`. Exporter and Go admission. Recursive
   exhaustiveness on. The Tag library section in `model/SEMANTICS.md`.
2. Rename `model/temporal/shared/` to `model/temporal/foundations/` (decision 21), performed by
   fn-142: the task queue moves to `foundations/taskqueue`, the worker and the client to `actors/`,
   and `Bounds.scala` to `model/temporal/`. One regeneration batch: Definition IDs change exactly by
   the package mapping, and IR files, Cases, fixtures and the canary Case change with them, plus the
   module-map and structure-lint updates. No other change rides along.
3. Ownership:
   - `features/workflow` created as an ownership-only package (decision 22);
   - `owns` for the 13 used RPCs, in three packages;
   - the used-RPC and message-ownership gate checks.
4. fn-125.5 and the required-settings slice of fn-125.6, pulled forward. Foundations markers,
   Domains and shared Tag Sets; feature Tag Sets for the 13 requests realizations write and the 62
   types they reach.
5. `tools/umpire/values` check, and lowering's literal check. Case bytes unchanged.
6. Tag Set inputs, the `Validated` law, violating classes with `rejects`, generated members and
   relations in required settings. Depends on fn-139, including its `.message(template)` rename.
7. Later: the ten written `Proto[...]` messages, then responses, each in its owner's package.

## 11. Decisions (2026-10-06)

| # | Question | Decision | Where it lands |
| --- | --- | --- | --- |
| 1 | Placement of Tag Definitions, Domains and settings | One kit-level IR file, referenced by the feature IR files. Refined by 18: foundations is that file, and features add their own | §7 IR (`uses`, derived enums stay in feature files) |
| 2 | Upstream protovalidate / `field_behavior` | Not now; keep the vocabulary close to protovalidate | §6, §3 |
| 3 | Provisioning namespaces and task queues | Deferred. The valid class uses the Profile's values; only their violating classes vary | §5, §7 Abstraction Claims |
| 4 | fn-125 | Pull forward only fn-125.5 and the required-settings slice of fn-125.6 | §8 |
| 5 | Limits | Relations (`atMost`/`atLeast`), not exact values | §8 |
| 6 | Normalizing Tags | Marked only (`Tag.normalizes`); no class, no modeled value | §7 |
| 7 | Violating classes | Status code plus no state change, through a kit-level `Validated` capability law | §7 |
| 8 | Class shape | Single-fault: `valid` plus one class per (field path, rejecting Tag), 1 + Σ, through one action-level input over the request's Tag Set. `grouped` dropped; the class-count question dropped | §7 Single-fault classes |
| 9 | Exhaustiveness | Requests now, recursing into every reachable message. No whole-value Tag Sets | §7 Exhaustive tagging |
| 10 | Tag meanings | Core validation Tags normative in `model/SEMANTICS.md` | §7 |
| 11 | Messages | Conformance compares status codes only. Templates serve diagnostics and future codegen. No pattern match, no redaction policy | §7 Message templates |
| 12 | fn-139 | Rename `rejects(r).because(text)` to `.message(template)`, with the same template form | §7 Message templates; fn-139 to update |
| 13 | Server-state-dependent rules | A rejecting Tag left uninterpreted, recorded as a Known Gap, for now | §4, §7 |
| 14 | Codegen spec rule | Deferred; codegen not planned | §9 |
| 15 | Sequencing | After fn-141.9 | §7, §10 |
| 16 | Spec wording | Reword `.plans/UMPIRE4_SPEC.md:59` and `model/README.md:753`, extend `:217`, as listed in §7 (not edited here) | §7 One Domain |
| 17 | IR link | On the enum type: `Enum.domain` (plus `domain_fingerprint`) and `Case.violates`, which names the field path and the Tag | §7 IR |
| 18 | Source location (user, 2026-10-06) | Follows ownership. A used RPC's request Tag Set lives in the owning package. Shared Domains, shared message Tag Sets, kit Tag Definitions and settings live in `model/temporal/foundations/`. Framework Tag Definitions stay in `model/umpire`. IR files follow source | §7 Ownership |
| 19 | RPC ownership (user; scope corrected) | Only RPCs a realization uses carry obligations: exactly one owning package and an exhaustive Tag Set (request now, response later). An unused RPC needs nothing. Every message type reachable from a used RPC has exactly one owning Tag Set; a type two packages reach belongs to foundations. The gate fails on a used RPC with no owner or no exhaustive Tag Set, naming the realization, the RPC and the missing fields | §7 Ownership |
| 20 | CODEOWNERS (user; scope corrected) | Skipped. Ownership is package ownership, enforced by the gate, with no mapping to teams. A future mapping to CODEOWNERS is possible; out of scope | §7 Ownership |
| 21 | `shared/` and `foundations/` | Rename `model/temporal/shared/` to `model/temporal/foundations/`, accepting one regeneration batch of Definition IDs and Cases, as its own phasing step. fn-142 performs it, with its layout: `foundations/` holds the task queue and this design's shared Tag content, `actors/` the worker and the client, and `Bounds.scala` sits in `model/temporal/` | §7 Ownership; §10 step 2 |
| 22 | `features/workflow` | Create it now, as an ownership-only package owning `StartWorkflowExecution`, `DescribeWorkflowExecution` and `GetWorkflowExecutionHistory` | §7 Ownership; §10 step 3 |

Decision 21 is performed by fn-142 ("Split model/temporal/shared into foundations and actors"), which this design adopts: `foundations/` (the task queue plus this design's shared Domains, Tag Sets, markers and settings keys), `actors/` (the worker and the client) and `model/temporal/Bounds.scala`.

No open questions remain.
