# Proto annotations: semantic types and validation in the Umpire IR

Research note, 2026-10-06. The question: how do we annotate any protobuf message and its fields
with metadata (semantic annotations such as `namespace`, plus the validation rules they bring),
carry that metadata in the Umpire IR rather than in a separate IR, and keep it tightly connected
to the Model? Planned use: generate valid, boundary and invalid inputs for Cases and Exploration.
Visionary use, kept open but not scoped: generate Go validation code.

Grounded in `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_VISION.md`, `.plans/UMPIRE_MODULES.md`,
`model/README.md`, `model/SEMANTICS.md`, the working-tree `ir.proto`, `model/umpire`,
`model/temporal`, `tools/umpire`, and the server's validators. Line numbers are from the working
tree on 2026-10-06. External claims link to primary sources. **[unverified]** marks a claim that
could not be confirmed.

## Answer

**Declare semantic types in the Scala Temporal kit, built from a closed constraint vocabulary that
the Temporal-agnostic framework provides. Attach them to protobuf fields with the typed field
selectors realizations already use. Lift them into a new section of the existing IR. A new
Temporal-agnostic Go module checks and generates values from them.** Do not use proto custom
options as the source of truth, and do not use a sidecar overlay file.

Why this design:

- **Proto options cannot be the source today.**
  - The public request messages come from the external `go.temporal.io/api` module as compiled
    `.pb.go` files (`go.mod:75`). The module cache holds no `.proto` sources.
  - `proto/api.binpb` is rebuilt from the linked Go registry (`Makefile:335-338`). This repo has
    nothing it could annotate.
  - Protobuf has no way to attach options to another file's messages. protovalidate has no overlay
    either ([protovalidate](https://github.com/bufbuild/protovalidate); absence searched,
    **[unverified]**).
  - ScalaPB's auxiliary options do apply to other files, but they carry only ScalaPB's own options
    ([ScalaPB customizations](https://scalapb.github.io/docs/customizations/)).
  - The server uses neither protovalidate nor PGV.
  - The public API does not even carry `google.api.field_behavior`. Its comments say the
    annotation is "not available in our gogo fork" (module cache,
    `workflowservice/v1/request_response.pb.go:6618-6620`).
- **The kit is where the vision says Temporal knowledge lives.**
  - "The Models are the only smart component", and "Temporal stays at the edges"
    (`.plans/UMPIRE4_VISION.md:93-100`).
  - A field's validation decides whether a request is rejected. That is behavior, so it belongs
    to a Model Definition with an ID and a fingerprint (`.plans/UMPIRE4_SPEC.md:47-53`, AUT-04
    `:263`). It is not Generated Data, which "describes what information exists, not how it
    affects behavior" (`:49-50`).
  - The framework supplies the generic vocabulary (length, pattern, range, presence, size). The
    kit supplies the Temporal names (`namespace`, `taskQueueName`, `limit.maxIDLength`). The IR
    and Go carry only declared names, as the vision requires (`:55-60`).
- **Tight connection to the Model is the point.**
  - An action input class can be declared as a partition of a semantic type (valid, too long,
    missing).
  - The class's Abstraction Claim then gets a defined member set, which is exactly what the spec
    says Exploration samples (`.plans/UMPIRE4_SPEC.md:247-250`, `:453-459`).
  - An invalid class's row is `rejects(invalidArgument)`, using fn-139's shared `Rejection`
    (`.flow/specs/fn-139-actor-grouped-rules-per-rpc-actions.md:55`). fn-139.8's code table
    checks the observed code (`:61`).
- **The annotations become tested claims, not documentation.**
  - The server has no single declarative source to stay in sync with. Its rules are spread over
    an interceptor, the frontend, CHASM validators and history (inventory below).
  - So drift is caught the Umpire way: an invalid-boundary Case whose Verdict turns violated.

The constraint vocabulary is closed and typed, with constraint IDs and parameterized bounds. That
keeps Go codegen open: it is the same shape Kubernetes' `validation-gen` compiles to native Go
([KEP-5073](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/5073-declarative-validation-with-validation-gen)).
An export to protovalidate predefined rules is a later Generated View, not a source.

## 1. Current state in the repo

### How requests and inputs are represented

- **Model inputs are finite abstract classes, never concrete request values.**
  - An action has `inputs` (finite `TypeRef`s), `schemas` (message full names) and `examples`
    (`ir.proto:322-343`). An `Example` is an Abstraction Claim: a class value plus a free-text
    `example` (`ir.proto:345-349`; `model/umpire/Action.scala:29-31`, `:120-124`).
  - Admission checks only that an example belongs to a class of a one-input action
    (`model/SEMANTICS.md:705-706`). The string is opaque.
  - Lowering copies the claims on a Case's path into provenance (`tools/umpire/check/claims.go:236-250`,
    `tools/umpire/lower/internal/producer/producer.go:442-447`, `case.proto:35-38`). Nothing reads
    them for values.
- **Concrete request values are realization constants.** For example `field(_.namespace) :=
  workerNamespace` (`model/README.md:1324-1345`).
  - An `Assignment` is a field path plus an `Operand` (`ir.proto:1152-1156`). An operand is a
    literal `ProtoValue`, an environment binding, the run id, a learned value, or a small boolean
    language over paths (`ir.proto:1236-1255`, `:1302-1317`).
  - The kit binds namespace and task queue to Profile-owned environment values (ART-13,
    `.plans/UMPIRE4_SPEC.md:347-354`; `model/temporal/realize/Kit.scala:59-62`). IDs come from the
    run id (`:65`), and type names come from `perCase` (`:68`).
- **Descriptors are already the authority on field shape, on both sides.**
  - Scala: the Models compile against ScalaPB classes for the linked API (`.plans/UMPIRE_MODULES.md:85-90`).
    The lifter resolves typed `Field[Root, V]` selectors (`model/umpire/realize/Typed.scala:4`)
    through ScalaPB descriptors (`model/irgen/Realizations.scala:163-168`).
  - Go: lowering re-checks every message, path and value kind against
    `protoregistry.GlobalFiles` (`tools/umpire/lower/descriptor.go:37-48`;
    `model/SEMANTICS.md:781-786`).
- **Exploration enumerates authored Scenario alternatives only.** Admission caps it at 4096
  combinations (`model/SEMANTICS.md:899-907`; `model/README.md:1394-1398`).
  - The spec's "class-member target" is defined (`.plans/UMPIRE4_SPEC.md:453-459`), but
    `tools/umpire/explore` has no implementation of it.
  - No Model has an invalid-input class today.
- **A Case's Contract does not check ID spellings or payload equality** (`model/README.md:1387-1390`).

### Where validation knowledge already hides, unchecked

- **The activity realization restates a server rule in a comment.** "The server refuses a start
  that sets neither a start-to-close nor a schedule-to-close deadline", so it writes a 300 s
  deadline (`model/temporal/features/activity/standalone/system/Realization.scala:72-84`;
  `model/temporal/realize/Kit.scala:75-86`).
  - The rule is `chasm/lib/activity/validator.go:171-194`.
  - Nothing checks that the realization obeys it.
- **ID lengths, blob sizes and similar limits are deliberately unmodeled.** fn-125 says they
  "stay unmodeled until a Query needs one" (`.flow/specs/fn-125-represent-dynamic-configuration-in-the.md:134`).
  - fn-125 Part B already plans one typed kit declaration per dynamic-config key, with a Go test
    that pins each declaration to the server registry (`:31`).
  - fn-125 also says a value a request field can state is set in the request and visible in Case
    bytes (`:27`).
- **The internal protos carry custom options, but not on the public messages.** Their options
  name request field paths by string, for example `option (routing).workflow_id =
  "start_request.workflow_id"` (`proto/internal/temporal/server/api/historyservice/v1/request_response.proto:39-67`).
  This is a precedent for field-path metadata.

### Constraints the design must respect

| Source | Constraint |
| --- | --- |
| `.plans/UMPIRE4_VISION.md:93-105` | Knowledge only in Models; others mechanical; Temporal at the edges; one meaning, one source; fail closed; artifacts are the interfaces |
| `.plans/UMPIRE_MODULES.md:31` | The DSL (`model/umpire`) names no Temporal concept; `TestFrameworkNamesNoTemporal` holds it to that, and "namespace" and "task queue" are on its word list (`:92-100`) |
| `.plans/UMPIRE_MODULES.md:30`, `:52` | IR depends only on protobuf support; Testpilot imports nothing from `tools/umpire` or `model` |
| AUT-04/05 (`.plans/UMPIRE4_SPEC.md:263-268`) | Stable Definition IDs; cross-language data, no callbacks, so a constraint cannot be a Scala lambda |
| PLN-02, ART-11 (`:304`, `:340-342`) | Same inputs and seed give the same Cases, byte for byte; generated values must be seeded |
| SEM-19 (`:149-153`) | One word per concept. "Rule" is taken by Contract Rules and the machines' `rules` section; "Type" is the IR's finite types (`ir.proto:51-61`) |
| `.plans/DSL_SIMPLIFICATION.md:65`, `:178`, `:192` | Lifter reads `val`/`def`; Scala annotations (`@binds`) rejected because the runtime cannot see them. So no `@namespace` Scala annotations; use vals |
| AGENTS.md | No new third-party libraries unless asked. `rapid` and `protovalidate-go` are not in `go.mod`; only Go's stdlib `regexp/syntax` is free |
| `.plans/UMPIRE_CEL_SPIKE.md:3-39` | The IR keeps its own expressions; CEL is an export target. `Operand` "already fits CEL", so it is the natural cross-field language |

## 2. Prior art

| System | Named reusable constraint | Composition | Runtime parameters | Overlay for protos you don't own | Input generation | Go code |
| --- | --- | --- | --- | --- | --- | --- |
| protovalidate | **Predefined rules**: an extension of `buf.validate.StringRules` with `(buf.validate.predefined).cel`, whose value is bound as `rule` ([docs](https://protovalidate.com/schemas/predefined-rules/)) | AND of all rules on a field ([standard rules](https://protovalidate.com/schemas/standard-rules/)) | Value fixed in the schema; none at runtime **[unverified absence]** | None | None from Buf; `example` field only ([validate.proto](https://raw.githubusercontent.com/bufbuild/protovalidate/main/proto/protovalidate/buf/validate/validate.proto)); third-party FauxRPC uses lengths, patterns and formats, not CEL ([FauxRPC](https://fauxrpc.com/docs/protovalidate/)) | Runtime evaluation; standard rules are native Go since v1.3.0, CEL is the fallback, and conformance runs twice to prove they agree ([blog](https://buf.build/blog/faster-protovalidate), [pkg](https://pkg.go.dev/buf.build/go/protovalidate)) |
| protoc-gen-validate | None **[unverified]** | AND | None | None | None | Generated `Validate()` / `ValidateAll()`; in maintenance mode ([repo](https://github.com/bufbuild/protoc-gen-validate)) |
| Google AIPs | `field_info.format` (UUID4, IPV4, …) ([field_info.proto](https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/field_info.proto)); `resource_reference` ([AIP-123](https://google.aip.dev/123)) | n/a | n/a | n/a | n/a | Descriptive only: `field_behavior` "does not itself add any validation" ([AIP-203](https://google.aip.dev/203)) |
| Smithy | Constrained named shape: `@length(min:1,max:255) string NamespaceName` ([constraint traits](https://smithy.io/2.0/spec/constraint-traits.html)) | **Member traits supersede** the target's ([model](https://smithy.io/2.0/spec/model.html)) | None | **`apply`**: lists concatenate, equal values dedupe, other conflicts are errors (same page) | Hand-written malformed-request tests ([compliance tests](https://smithy.io/2.0/additional-specs/http-protocol-compliance-tests.html)) | smithy-rs: constrained newtypes plus a violation enum per trait ([RFC-0025](https://smithy-lang.github.io/smithy-rs/design/rfcs/rfc0025_constraint_traits.html)) |
| JSON Schema / OpenAPI 3.1 | `$ref` reusable schemas; `format` is only an annotation unless the assertion vocabulary is on ([validation](https://json-schema.org/draft/2020-12/json-schema-validation)) | `$ref` with sibling keywords ANDs, so it only tightens ([core](https://json-schema.org/draft/2020-12/json-schema-core)) | None | **Overlay 1.0**: JSONPath `target` with `update`/`remove` ([spec](https://spec.openapis.org/overlay/v1.0.0.html)) | Schemathesis: positive and negative modes; boundary lengths 1, 2, 3, 9, 10, 11 for min 2 / max 10 ([docs](https://schemathesis.readthedocs.io/en/stable/explanations/data-generation/)); hypothesis-jsonschema builds by construction, not filter ([repo](https://github.com/python-jsonschema/hypothesis-jsonschema)) | n/a |
| Scala iron / refined | `type Username = String :| (Alphanumeric & MinLength[5])` ([iron](https://iltotore.github.io/iron/docs/reference/constraint.html)) | `&`, `|`, `DescribedAs`; refined `And`/`Not` ([refined](https://github.com/fthomas/refined)) | Type-level literals only | n/a | iron-scalacheck: a generator per known constraint, filtering otherwise ([module](https://iltotore.github.io/iron/docs/modules/scalacheck.html)) | n/a |
| Kubernetes | `+k8s:format=dns-label`, with formats scoped to a type; formats declarable in YAML ([validation-gen](https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/code-generator/cmd/validation-gen/validators)) | Tags AND; CEL for exceptions | ValidatingAdmissionPolicy `paramRef`, read as `params.x` ([VAP](https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/)); `+k8s:ifEnabled` | Tags in Go source | n/a | **validation-gen** emits native Go; shadow mode counts mismatches against hand-written validation ([KEP-5073](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/5073-declarative-validation-with-validation-gen)) |
| Hypothesis / rapid | `register_type_strategy`, `from_regex(fullmatch=True)` ([strategies](https://hypothesis.readthedocs.io/en/latest/reference/strategies.html)); rapid `StringMatching` (Go `syntax.Perl`) ([pkg](https://pkg.go.dev/pgregory.net/rapid)) | n/a | n/a | n/a | By construction, plus filter, plus hand-written generators | n/a |
| CEL | n/a | n/a | n/a | n/a | Partial evaluation leaves a residual predicate ([cel-go](https://github.com/google/cel-go)); cel-java's Z3 verifier yields counterexamples ([verifier](https://github.com/cel-expr/cel-java/tree/main/verifier)), but its use as an input generator is **[unverified]** | Cost estimation: `EstimateCost`, `CostLimit` ([pkg](https://pkg.go.dev/github.com/google/cel-go/cel)) |

What transfers:

1. **A named semantic type is a bundle of constraints, referenced by name.** The models are
   Smithy's constrained shape, protovalidate's predefined rule and validation-gen's type-scoped
   format. Generators and later codegen key off the name.
2. **Use tighten-only AND by default (JSON Schema `$ref`, protovalidate).** Smithy-style override
   is allowed only explicitly, with a reason, like capabilities' `except`/`overriding`
   (`.plans/SEMANTIC_PROTOCOLS.md:147-150`). Then a field can never accept what its semantic type
   rejects unless a reviewer approved it.
3. **Keep the vocabulary closed, with an escape hatch.** Length, range, pattern, enum membership,
   presence and size can each be generated by construction and compiled to native Go. Any other
   predicate is generate-and-filter or a hand-written generator (validation-gen, hypothesis-jsonschema,
   iron-scalacheck).
4. **Parameters are symbolic and resolved per run** (VAP `params`). protovalidate's `rule` is
   parameterized but fixed in the schema, which cannot express `limit.maxIDLength`.
5. **Invalid inputs violate exactly one constraint, at its boundary** (Schemathesis coverage and
   negative modes). Each generated value is tagged with the constraint it targets (smithy-rs's
   per-trait violation enum).
6. **Pin the regex dialect, the anchoring and the length unit.** Smithy and JSON Schema use
   unanchored ECMA-262. protovalidate uses RE2. The server measures lengths with Go `len`, which
   counts bytes.
7. **Overlays work when they are keyed by fully qualified field and merged strictly** (Smithy
   `apply`, OpenAPI Overlay). Our typed `Field` selector is a type-checked version of the same
   idea.
8. **Prove a codegen'd validator against the reference evaluator on the same generated corpus**
   (protovalidate's double conformance run, K8s shadow mode). The generated inputs double as the
   conformance suite.

## 3. Temporal validation inventory

### Rules

`DC` = `common/dynamicconfig/constants.go`. `IA` = `InvalidArgument`. `MaxIDLength` =
`limit.maxIDLength`: global, default 1000, shared by namespace, task queue, workflow/activity/timer
IDs and types, signal name, identity and request ID (`DC:527-532`).

| Concept | Rule | Static / dynamic | Enforced at | Error |
| --- | --- | --- | --- | --- |
| namespace | required | static | `common/rpc/interceptor/namespace_validator.go:376-380` | IA "Namespace not set on request." (`:40`) |
| namespace | `len ≤ MaxIDLength`; **no charset rule, regex or reserved names** ("currently only a max length check") | `limit.maxIDLength` | `namespace_validator.go:183`, `:191-195` (interceptor, before lookup); again `service/frontend/workflow_handler.go:6970-6976`, `service/history/api/create_workflow_util.go:302-304` | IA "Namespace length exceeds limit." |
| namespace (Register) | retention a valid duration ≥ min; bad binaries ≤ N; duplicate | `system.namespaceMinRetention{Global,Local}` (`DC:241`, `:246`), `frontend.maxBadBinaries` (`DC:915`) | `service/frontend/namespace_handler.go:1291-1306`, `:512-519`, `:128-132` | IA; AlreadyExists |
| workflow id | required; `len ≤ MaxIDLength` | dynamic | `chasm/lib/workflow/validator.go:51-61`; `service/frontend/validators.go:15-28` | IA |
| run id | optional; if set, `uuid.Validate` (accepts hyphenless, braced and urn forms) | static | `service/frontend/validators.go:22-26` | IA "Invalid RunId." |
| workflow / activity type | required; `len ≤ MaxIDLength` | dynamic | `workflow_handler.go:654-660`; `chasm/lib/activity/validator.go:102-117` | IA |
| activity id | required; `len ≤ MaxIDLength` | dynamic | `chasm/lib/activity/validator.go:102`, `:113` | IA |
| task queue | set; empty name → default or IA; `len ≤ MaxIDLength`; no `/_sys/` prefix on a root partition; no `temporal-sys-per-ns-*` from users; UTF-8 and whitespace checks promised in doc comments, **not implemented** | dynamic | `common/tqid/task_queue_validator.go:93-146`; `common/primitives/task_queues.go:46-72` | IA |
| signal / update / query name | required; `len ≤ MaxIDLength` (query: required only) | dynamic | `workflow_handler.go:2315-2321`, `:5547`, `:3315` | IA |
| request id | empty is **auto-filled with a UUID**; `len ≤ MaxIDLength` | dynamic | `workflow_handler.go:6948-6963`; `chasm/lib/activity/validator.go:374-376` | IA |
| identity | `len ≤ MaxIDLength` (worker APIs, standalone activity start) | dynamic | `workflow_handler.go:1094`; `chasm/lib/activity/validator.go:381` | IA |
| duration | nil OK; seconds and nanos same sign; ≥ 0; above 100 y **silently capped** | static | `common/primitives/timestamp/duration.go:12`, `:67-88` | IA |
| activity timeouts (cross-field) | start-to-close or schedule-to-close > 0; missing ones filled in; heartbeat ≤ start-to-close | static | `chasm/lib/activity/validator.go:151-214` | IA "a valid StartToCloseTimeout or ScheduleToCloseTimeout must be set…" (`:194`) |
| retry policy | `maximum_attempts == 1` skips the rest; coefficient ≥ 1; max interval ≥ initial; attempts ≥ 0; timeout-type names valid | static, with DC defaults | `common/retrypolicy/retry_policy.go:103-140` | IA |
| cron | parses with `robfig/cron.ParseStandard` and has a reachable next time; not combined with start delay | static | `common/backoff/cron.go:15-29`; `chasm/lib/workflow/validator.go:97-110` | IA |
| ID reuse / conflict policies (cross-field) | TERMINATE_IF_RUNNING with a conflict policy, REJECT_DUPLICATE with TERMINATE_EXISTING, SignalWithStart with FAIL all rejected | static | `chasm/lib/workflow/validator.go:111-124`, `:204-206` | IA |
| on-conflict options (cross-field) | attach callbacks ⇒ attach request id | static | `workflow_handler.go:6644-6651` | IA |
| priority | key ≥ 0; fairness key ≤ 64 bytes (**hard-coded**); weight ≥ 0 | static | `common/priorities/priority_util.go:11`, `:51-62` | IA |
| search attributes | key count; defined; not system; type decodes; value size; total size | `frontend.searchAttributes{NumberOfKeys,SizeOfValue,TotalSize}Limit` (`DC:940-950`), **plus server state** (definitions) | `common/searchattribute/validator.go:79-232` | IA; Unavailable when metadata cannot load (`:98`) |
| memo | size ≤ N | `limit.memoSize.*` (`DC:430-435`) | `create_workflow_util.go:256-268` | IA |
| payloads (input, signal, query args) | rejected only when size > warn **and** > error | `limit.blobSize.{warn,error}` 512 KiB / 2 MiB, namespace (`DC:420-425`) | `common/util.go:604-633` | IA "Blob data size exceeds limit." |
| payloads (heartbeat, activity result, WFT failure) | oversize **converts to a failure** or truncates; RPC accepted | same | `workflow_handler.go:1478-1500`, `:1678-1700`, `:1303-1317` | none on the RPC |
| header | not enforced; only a metric is recorded | none | `create_workflow_util.go:239`; `workflow_handler.go:3327` | none |
| links | count; size; required fields per variant; unknown variant rejected | `frontend.maxlinksPerRequest`, `frontend.linkMaxSize` (`DC:1149-1154`) | `common/links/validator.go:29-131` | IA |
| callbacks | count; kind enabled; URL length and rules; header size | `system.maxCallbacksPerWorkflow`, `frontend.callback*` (`DC:1129-1139`) | `common/callbacks/validator.go:94-204` | IA / Unimplemented (`:130`) |
| user metadata | summary ≤ 400, details ≤ 20000 (standalone activity and Nexus only) | `limit.userMetadata*Size` (`DC:3752-3757`) | `chasm/lib/activity/validator.go:416-432` | IA |
| versioning / deployment | structure; name has no `.` or `:`; no leading `__`; build-id length | `limit.maxIDLength`, `limit.workerBuildIdSize` (`DC:533`) | `common/worker_versioning/worker_versioning.go:610-638`, `:750-826` | IA |
| long-poll deadline | set; ≥ 2 s | static | `common/util.go:124-126`, `:638-676` | IA / **FailedPrecondition** |

### What the inventory means for the design

- **Most limits are dynamic, and nearly all of them hang off one key.** Bounds must be parameters
  that name a kit-declared setting with its default (fn-125 Part B). A literal would be wrong
  under any server that overrides `limit.maxIDLength`.
- **Validation happens in layers.** The interceptor runs before the handler, and history
  re-validates with other messages (`create_workflow_util.go:273-328`).
  - Conformance must compare status codes, never message text. fn-139 already decided this
    (`.flow/specs/fn-139-actor-grouped-rules-per-rpc-actions.md:172`).
  - An invalid input must violate exactly one constraint, so the order of checks does not matter.
- **Not every rule rejects.** Some rules repair the request (auto-filled request ID, capped
  durations, filled-in timeouts). Others convert the problem into a failure (oversized activity
  result). A constraint needs an **effect**: reject, normalize or convert. Only "reject" yields
  invalid-input Cases with an expected code.
- **Some rules depend on server state, not just the request.** Search-attribute definitions,
  enabled callback kinds and an existing namespace are examples. They are not request constraints.
  They belong to Model state or the environment (QLF-01).
- **Some rules exist only in doc comments or as hard-coded constants.** The task queue's promised
  UTF-8 and whitespace checks and the 64-byte fairness key are examples. An annotation records
  what the server actually enforces, and a Case is what proves it.

## 4. Generating inputs from constraints

Per constraint: how to construct a valid value, which boundaries to try, and how to violate it
exactly once.

| Constraint | Valid by construction | Boundary members | Invalid (violates only this) | Tractable? |
| --- | --- | --- | --- | --- |
| presence (`required`, `notDefault`) | any member of the other constraints | n/a | unset; zero value (protovalidate's distinction between the two, [standard rules](https://protovalidate.com/schemas/standard-rules/)) | yes |
| length `[min, max]` with a unit | ASCII filler of length n | min, min+1, max−1, max; multibyte runes so byte length ≠ rune length | min−1, max+1 | yes, once the parameter is resolved for the Case |
| pattern (RE2, anchored) | walk `regexp/syntax` (Go stdlib) | shortest and longest matches under the length bounds | mutate one character outside the class, then check that the compiled regexp rejects it; the complement of a regex is not constructible in general | yes, with check-after-mutate |
| forbidden prefix / `notIn` | avoid it | the prefix minus one character | the prefix plus a valid suffix; a listed literal | yes |
| integer / duration range | interval members | edges; for durations, seconds and nanos of the same sign | edges ± 1; mixed signs | yes |
| serialized size of a message or payload | filler sized with `proto.Size` | N − 1, N | N + 1 | yes, but 2 MiB values do not belong in checked-in Case bytes (open question 5) |
| format (uuid; cron) | uuid by construction; cron needs a hand-written generator registered under the format's name | hyphenless uuid, braced uuid | malformed text | uuid yes; cron by hand |
| cross-field `requires` (an `Operand` over presence and equality) | enumerate truth assignments of its few atoms; pick one that satisfies it | n/a | an assignment that falsifies only this constraint | yes for presence and equality atoms (a few atoms, brute force) |
| opaque (escape hatch, cited) | none | none | none | no: a Known Gap on every Case that touches the field |

Rules for all generation:

- **Seeded and deterministic.** The seed comes from the Case or exploration candidate identity, as
  PLN-02 and ART-11 require. A generated value is a literal in the Case, so Testpilot does not
  change (EVD-01).
- **Each value is tagged with its target constraint ID.** Lowering records which constraint a
  member satisfies or violates in Case provenance, as an extension of `AbstractionClaim`
  (`case.proto:35-38`).
- **Fields the Profile owns are environment-bound.** Namespace and task queue are ART-13 bindings.
  - A *valid* member cannot be invented without provisioning a resource, because a valid but
    unregistered namespace is `NotFound`, not accepted.
  - An *invalid* member works, because the interceptor checks the name's length before lookup
    (`namespace_validator.go:183`).
  - So generation varies only invalid members of bound fields until the Driver can provision
    resources (open question 4).
- **What a hand-written generator is needed for:** cron, arbitrary CEL, server-state-dependent
  rules (search attributes), and any constraint over more than presence and equality.
  - A semantic type may name a generator. The Go module registers it by that name, like
    Hypothesis's `register_type_strategy`.
  - An unregistered generator is refused at admission, never silently skipped.

## 5. Where annotations live, and how they reach the IR

| Option | Source of truth | Path to the IR | Pros | Cons |
| --- | --- | --- | --- | --- |
| **A. Upstream proto options** (protovalidate standard plus predefined rules, e.g. `temporal.api.validate.namespace`, in `temporalio/api`) | API protos | Go and ScalaPB descriptors carry the options; the lifter, or Go, reads them into the IR | One source shared with SDKs; standard tooling; a runtime evaluator already exists | Owned by another repo and released with the `go.temporal.io/api` pin; that repo cannot even use `field_behavior` today; `rule` is static, so no `limit.maxIDLength`; no link to Model classes, rejections or IDs; makes descriptor data the smart component, against vision `:93` |
| **B. Sidecar overlay** (textproto or YAML keyed by FQN, Smithy-`apply` style) | Overlay file | The lifter or gate reads it | No upstream dependency; easy to edit | A third authoring language; untyped field names (the DSL forbids proto-name strings, `.plans/UMPIRE_MODULES.md:86-90`); disconnected from the Model; duplicates the DSL |
| **C. Kit declarations in Scala** | `model/temporal` | The lifter lifts vals into a new IR section; Go admits them against descriptors | Typed selectors; Definition IDs and fingerprints; parameters bound to fn-125 settings; input classes and `rejects` connect directly; fits the module map | Restates rules that live in Go (true of every option except server codegen); Scala-only authoring |
| **D. C, plus a later import of upstream options and an export to them** | `model/temporal` | As C; if upstream ever adopts protovalidate, the lifter reads those options too and refuses disagreement with the kit | Keeps A's interoperability without its ownership problems | More machinery, built only once needed |

**Recommendation: C now, shaped so that D stays possible.** The vocabulary maps one to one onto
protovalidate's standard rules plus predefined rules, so export and import are mechanical.

Options A and B also fail "one meaning, one source" (`.plans/UMPIRE4_VISION.md:101-102`) unless
they become the *only* source. Neither can carry the connection to classes and `Rejection`s that
the Model needs.

## 6. Recommended design (sketches)

### Words (SEM-19, needs approval)

| Word | Meaning | Why not the obvious word |
| --- | --- | --- |
| **Semantic Type** | A named bundle of constraints over one protobuf value kind | Bare "Type" is the IR's finite type |
| **Constraint** | One checkable condition with an ID, an effect and a citation | "Rule" is a Contract Rule and the `rules` section |
| **Annotation** | Attaching Semantic Types and Constraints to one field path, or to one message for cross-field constraints | — |

### Framework (Temporal-agnostic, `model/umpire/Constraints.scala`, sketch)

```scala
// A value a constraint bounds by: a literal, or a parameter the environment supplies. A system's kit
// extends Parameter, as it extends realize's Setting.
trait Parameter:
  def default: Long
enum Bound:
  case literal(n: Long)
  case parameter(p: Parameter)
given Conversion[Long, Bound] = Bound.literal(_)
given Conversion[Parameter, Bound] = Bound.parameter(_)

enum Unit:
  case bytes, runes
enum Effect:
  case rejects, normalizes, converts

// One constraint on a value of type V; `because` cites the implementation it restates.
final class Constraint[V] private[umpire] (/* kind, effect, because */)

object text:
  def required: Constraint[String]
  def length(min: Bound = 0L, max: Bound, unit: Unit = Unit.bytes): Constraint[String]
  def pattern(re2: String): Constraint[String] // fully anchored
  def notPrefix(p: String): Constraint[String]
  def format(name: String): Constraint[String] // generated by a registered generator
object span: // google.protobuf.Duration-shaped values
  def nonNegative: Constraint[Duration]
  def sameSign: Constraint[Duration]
  def positive: Constraint[Duration]
object sized:
  def bytes(max: Bound): Constraint[Any] // serialized size

// A named bundle, named after its `val`; `extend` ANDs another type's constraints.
final case class SemanticType[V](constraints: Vector[Constraint[V]], extend: Vector[SemanticType[V]] = Vector.empty)
def semantic[V](cs: Constraint[V]*): SemanticType[V]

// Annotations of one message's fields. `field` is realize's typed selector, so a path or a value
// kind the descriptors lack does not compile, or is refused at lift.
def annotate[M <: GeneratedMessage](lines: AnnotationLine[M]*): Annotations[M]
```

### Kit (Temporal, e.g. `model/temporal/semantics/Semantics.scala`, sketch)

```scala
// fn-125 Part B's typed key, pinned to common/dynamicconfig by a Go test.
val maxIdLength = DynamicSetting.int("limit.maxIDLength", default = 1000, scope = global)

val identifier = semantic[String](
  text.required.because("chasm/lib/workflow/validator.go:51-61"),
  text.length(max = maxIdLength).because("common/dynamicconfig/constants.go:527-532")
)
val namespaceName = semantic[String](
  text.required.because("common/rpc/interceptor/namespace_validator.go:376-380"),
  text.length(max = maxIdLength).because("namespace_validator.go:191-195")
)
val taskQueueName = semantic[String](
  text.notPrefix("/_sys/").because("common/tqid/task_queue_validator.go:144-146")
).extend(identifier)
val requestId = semantic[String](
  text.length(max = maxIdLength),
  text.required.normalizes.because("workflow_handler.go:6948-6963: empty is filled with a UUID")
)
val timeout = semantic[Duration](span.sameSign, span.nonNegative)
  .because("common/primitives/timestamp/duration.go:67-88")

val startActivity = annotate[StartActivityExecutionRequest](
  field(_.namespace) is namespaceName,
  field(_.activityId) is identifier,
  field(_.getActivityType.name) is identifier,
  field(_.getTaskQueue.name) is taskQueueName,
  field(_.requestId) is requestId,
  field(_.identity) is text.length(max = maxIdLength), // a field-local constraint
  field(_.getStartToCloseTimeout) is timeout,
  requires(present(_.getStartToCloseTimeout) or present(_.getScheduleToCloseTimeout))
    .because("chasm/lib/activity/validator.go:171-194")
)
```

### Connecting annotations to the Model (sketch)

```scala
// An input whose classes partition a semantic type: `valid` holds every constraint; each other class
// violates exactly the constraint it names. The lifter checks each class names a constraint of the
// type that rejects.
enum IdClass derives Finite:
  case valid, missing, tooLong
val activityId = input[IdClass].partitions(identifier, missing -> text.required, tooLong -> text.length)

on(client.start) {
  when(activityId == IdClass.valid) ~> effects.schedule
  when(activityId != IdClass.valid) ~> rejects(Rejection.invalidArgument) // fn-139
}
```

The realization then writes `field(_.activityId) := member(activityId)`. Lowering asks the Go value
module for the class's canonical member, for example the `max` boundary for `valid` and `max + 1`
for `tooLong`. That value is the Abstraction Claim's example. Exploration's class-member targets
draw the class's other members, so a divergent member is the counterexample that splits the class
(`.plans/UMPIRE4_SPEC.md:247-250`).

The same law can be stated once for every entity as a capability, `Validated`: every
rejecting-constraint class of an annotated action is rejected with its `Rejection` and changes no
state. The kit then states it once rather than in every Model (`.plans/SEMANTIC_PROTOCOLS.md:95-150`).

The first increment needs no Model change. Lowering checks every **literal** operand a realization
writes against the field's annotations, and refuses a Case that writes a value its annotation
rejects. For example, it would refuse a start with neither deadline, with the constraint's
citation in the error.

### IR additions (sketch, `ir.proto`)

```proto
message Model {
  // ... fields 1-16 unchanged
  // The Semantic Types this file's annotations and inputs reference, closed under `extends`.
  repeated SemanticType semantic_types = 17;
  repeated Annotation annotations = 18;
}

// A named bundle of constraints over one protobuf value kind. A Model Definition:
// `<family>.semantic.<name>`, with a fingerprint over its constraints and none over its citations.
message SemanticType {
  string name = 1; // fully qualified, e.g. `temporal.semantics.namespaceName`
  Position position = 2;
  ValueKind kind = 3; // text, integer, duration, message, bytes
  repeated string extends = 4; // ANDed
  repeated Constraint constraints = 5;
  string generator = 6; // a registered generator's name, or empty
}

message Constraint {
  string id = 1; // `<semantic type or annotation>.<n>`, stable
  Position position = 2;
  string because = 3; // the implementation it restates; not fingerprinted
  enum Effect {
    EFFECT_UNSPECIFIED = 0;
    EFFECT_REJECTS = 1;
    EFFECT_NORMALIZES = 2;
    EFFECT_CONVERTS = 3;
  }
  Effect effect = 4;
  Value rejection = 5; // the Model's Rejection value, when it rejects
  oneof kind {
    Empty required = 6;
    Empty not_default = 7;
    Length length = 8;
    string pattern = 9; // RE2, fully anchored
    string not_prefix = 10;
    Literals in = 11;
    Literals not_in = 12;
    Range range = 13;
    Bound max_serialized_bytes = 14;
    Range count = 15; // items of a repeated or map field
    string format = 16; // a registered format
    Operand requires = 17; // a cross-field condition over the message's paths
    string opaque = 18; // a rule Umpire neither generates nor checks; always a Known Gap
  }
}

message Bound {
  oneof kind {
    int64 literal = 1;
    // A value the environment supplies: its key, as the kit declares it, and the default a Case
    // requires when no Profile states one.
    Parameter parameter = 2;
  }
}
message Parameter { string key = 1; int64 default = 2; }
message Length { Bound min = 1; Bound max = 2; Unit unit = 3; }
enum Unit { UNIT_UNSPECIFIED = 0; UNIT_BYTES = 1; UNIT_RUNES = 2; }
message Range { Bound low = 1; Bound high = 2; }
message Literals { repeated ProtoValue values = 1; }

// Semantic Types and Constraints attached to one field of one message, or, with no path, to the
// message itself.
message Annotation {
  string message = 1; // full name
  string path = 2; // the IR's field-path selectors
  Position position = 3;
  repeated string types = 4;
  repeated Constraint constraints = 5; // tighten only
  repeated Override overrides = 6; // loosening, each with a reason
}
message Override { string constraint = 1; string because = 2; Constraint replacement = 3; }

// ...and on Action, for an input whose classes partition a Semantic Type:
message Partition {
  string input = 1;
  string semantic_type = 2;
  repeated ClassConstraint classes = 3;
}
message ClassConstraint { Value class = 1; string violates = 2; } // empty `violates`: the valid class
```

Admission rules the Go reader adds (`tools/umpire/ir`, sketch):

- An annotation names a message, path and value kind that the linked descriptors have.
- A constraint fits its field's kind. For example, `length` on a string and `max_serialized_bytes`
  on a message.
- The regex compiles as RE2.
- The annotation's constraints, ANDed with its types', admit at least one value. An empty valid
  set is refused, like an Unsatisfiable Scenario (PLN-05).
- A partition's classes name rejecting constraints of their type.
- A parameter key is one the kit declares.
- Every `format` and `generator` is registered.

Fingerprints:

- A Semantic Type's fingerprint enters the fingerprint of the input that partitions it. So
  changing `namespaceName` changes the Queries whose classes it defines, and nothing else.
- Annotations used only for the literal-operand check fingerprint nothing, like `ApiBehavior`
  (`ir.proto:738-744`).

### Go module (`.plans/UMPIRE_MODULES.md` row, sketch)

| Module | One job | Public interface | Permitted domain dependencies |
| --- | --- | --- | --- |
| Values, `tools/umpire/values` | Check and generate protobuf field values against declared Constraints. | `Admit(model)`, `Check(annotation, value) []Violation`, `Members(type, class, bounds, seed)` (canonical member first, then boundaries, then sampled members), `Register(format, generator)` | Umpire IR, `protoreflect`, stdlib `regexp/syntax`; interp for `Operand` evaluation if needed; no Testpilot, no Temporal names |

Consumers:

- `lower` resolves members, checks literal operands, and binds parameters through the Case's
  required settings. This follows fn-125's "bound assumption" relation, or the Profile states the
  value.
- `explore` uses class-member targets.
- Testpilot is untouched: a Case still carries literals.

This is a deep module behind a small interface, testable on fixtures without a cluster (MOD-08).

## 7. Keeping Go validation codegen open

The IR has everything a native Go validator generator needs:

- a closed vocabulary;
- constraint IDs to name violations by;
- effects and rejections, which give the status code through fn-139.8's table;
- parameters, compiled to dynamic-config lookups by key.

This mirrors validation-gen (tags → IR → native Go). Before enforcing anything, it would run in
K8s's shadow mode, comparing against the hand-written validators and counting mismatches. The
generated inputs (§4) are its conformance corpus.

A generated validator would be a Generated View bound to the IR checksum (ART-07). An export to
protovalidate predefined rules is the same: a view, not a source.

Two things block it, and both are decisions, not mechanics:

- **Precedence inverts.** The Model would drive server validation. SEM-01 makes the Model
  authoritative for the behavior it covers (`.plans/UMPIRE4_SPEC.md:133-135`), but no rule yet
  lets Umpire output be server code.
- **Normalizing effects** (auto-fill, capping) need value-producing semantics that the closed
  vocabulary only marks today.

Avoid now what would close the door:

- Scala lambdas as constraints (AUT-05).
- Free-form CEL as the main form.
- Literal limits where the server reads dynamic config.
- Message text as the identity of a violation.

## 8. Phasing (if adopted)

1. **Framework vocabulary, lifter, IR section and Go admission** against descriptors. No Case
   changes.
2. **Kit Semantic Types** for every field today's realizations write: namespace, activity ID and
   type, task queue, request ID, identity, the three deadlines plus the cross-field deadline
   rule. Depends on fn-125 Part B's typed keys, or a minimal `Parameter` until then.
3. **Lowering's literal-operand check.** It catches realization drift and leaves Case bytes
   unchanged.
4. **Partitions, the `Validated` capability, invalid classes with `rejects(invalidArgument)`, and
   generated members in `lower` and `explore`.** Depends on fn-139 (`Rejection`, code table).
5. **Visionary:** protovalidate export, then Go codegen behind shadow mode.

## Open questions for the human

1. **Words (SEM-19):** "Semantic Type", "Constraint", "Annotation", given that "Type" and "Rule"
   are taken?
2. **Where Semantic Types live in the IR:** as a per-file closure (each IR file self-contained,
   the same type repeated with the same fingerprint), or in one kit-level IR file that the others
   reference? Both are "the Umpire IR". The closure keeps each file independently admissible.
3. **Upstream:** should we propose protovalidate (or at least `field_behavior`) to `temporalio/api`
   so that option D's import has a source? That depends on the gogo fork constraint those
   comments cite.
4. **Environment-bound fields:** may the Driver provision per-Case namespaces and task queues so
   that *valid* members of those types can vary, or do they stay Profile-fixed with only invalid
   members generated?
5. **Large values:** boundary members of `limit.blobSize.error` (2 MiB) and memo limits would bloat
   checked-in Cases. Should Testpilot gain a deterministic filler operand (`bytes(n, seed)`), which
   is a Case-format change, or do size boundaries stay out of checked-in Cases?
6. **Normalizing rules** (auto-filled request ID, capped durations, filled-in timeouts): record
   them only as non-rejecting (`normalizes`), or model the resulting value so Conformance can check
   it?
7. **Expectation for invalid classes:** is it only the status code (fn-139's decision), or also
   "no state change", stated by the `Validated` law and checked by a follow-up read?
8. **Response annotations:** should annotations also type *response* fields (for example, `run_id`
   is a UUID), so Conformance can check observed values? That is out of the stated scope.
9. **Server-state-dependent rules** (search attributes defined, callback kinds enabled): Model
   state, environment, or an `opaque` constraint with a Known Gap?
10. **Codegen precedence:** if Go validation is ever generated from the IR, does that need a new
    spec rule (GOV-02) letting a Generated View become server code?
