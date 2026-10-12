# Model migrations

DSL changes ship a task-named Scalafix rule under `model/migrate/rules/`, prove it on copied `activity_mini` sources before the real-feature sweep, and follow `model/migrate/README.md`; production loading remains held in fn-166.10.

## Declared gaps: Fn167Task4Gaps

The pure transformation turns a feature's accepted lint findings into `@gap` stacks on the
declarations that own them. Its caller resolves each finding key `(kind, owner, subject)` to a
declaration identity (`<package>.<objects>.<name>`) and the typed subject constructor that names
it there; it also supplies fn-167.3's proven base keys and the `except`/`overriding` waivers.
The rule writes one `@gap(subjects*)(because = ...)` per declaration and reason, in the order the
acceptances first name them, inserted above the declaration with no other byte changed.
A `capability-waiver` acceptance whose reason equals its declaration's is dropped, because the
waiver lowers on its own. A finding whose base's acceptance covers it is dropped as inherited.
Already declared gaps of the same reason and subjects converge.

Duplicate acceptances, an unresolved key, a waiver no declaration states or whose reason differs,
an inherited finding whose base is not accepted, a declaration found zero or several times or not
starting its line, and a subject already declared under another reason refuse the whole run: no
source is returned. `Fn167Task4GapInputs` reads the existing `<file>.lint.json` acceptances and the
lifted waiver file. The runnable IR-backed placement resolver and the real-feature retirement of
acceptance files are fn-167.7's. For other shapes, manually place each finding's `@gap` on the
declaration that owns it, with the subject constructor of its lint kind, and keep its exact reason.

```sh
mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.GapsSuite
timeout 1800 /bin/bash model/migrate/prove-mini.sh umpire.migrate.RewriteGapsMini compare
```

The copied-mini runner reads `proof/gaps-mini.lint.json` (one unproduced status fact and the
mini's two capability waivers, with the reasons from the copy's own lift), places the hole on
`ActivityProduct` by hand, and requires one insertion, two dropped waivers, convergence and
refusal of a reworded waiver. `prove-mini.sh` now lifts every copy from the copy's own directory,
so positions and source checks read the copied sources rather than the authored ones (an unchanged
copy lifts byte for byte as before). The `compare` mode requires the complete mini IR to equal the
original but for the positions below the insertion in `Product.scala`, which move down by its line
count; waivers compare byte for byte. Gap metadata is not
yet carried in the IR (fn-167.5), so the lift has no other delta. Production loading, IR transport,
gap-aware lint, real-feature migration and the batch-end broad validation and review remain held.

## Facet wrapping: Fn161Task4FacetWrapping

The selected declared machines already use facets: worker Polling/Serving, taskqueue
Product and Protocol/Queueing, mini Product and Protocol/Dispatching and Completion,
and mini AnswerDispatch/Answering. Both facet admission switches default on for Temporal
declarations. The copied-source proof independently discovers the six machines from compiled
inheritance, excluding Derived/Built compositions, and checks their actual mixins and the
worker, taskqueue and mini FacetName vocabularies. Default lifting checks their declarations
and paired action ownership; no switch or authored source is changed to manufacture migration.

The compiled rule supports an inline answering namespace in mini protocol AnswerDispatch,
or completion in mini Product/Protocol with an existing Dispatching mixin. It requires the
inventoried Machine/Phased types, framework imports, facet aggregation order and namespace
sections. It preserves the namespace body bytes, effects, claims and machine-wide declarations,
adding the typed facet and Completion's Dispatching self-type. Existing facets converge unchanged.
Unsupported inventoried types, competing imports, collisions, nested machines, unowned machine sections,
escaping core helpers and unsupported namespace declarations refuse the whole document.
Unrelated owners are left untouched.

Scalafix cannot create files. Its intermediate machine-plus-trait document is not an admitted
layout. The proof's narrow companion operation splits only the three inventoried matching
file pairs inside a private copied-source tree. It checks independently captured source pins,
refuses existing destinations and symlinks, and restores only captured parent-clause layout
and temporary seed imports; unaccounted source or semantic changes cannot be discarded.
For other shapes, manually inventory action ownership, imports, vocabulary and dependencies;
keep machine-wide state/init/end/projection/refinement/monitors/Queries in their original owner,
move each owned namespace intact into its matching facet file, add its typed dependency,
and compile/lift before accepting the extraction.

```sh
mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.FacetWrappingTest
mise exec -- scala-cli test --server=false model/irgen --test-only umpire.irgen.FacetAdmissionTest
timeout 600 /bin/bash model/migrate/prove-mini.sh umpire.migrate.RewriteFacetWrappingMini
```

The proof actually compiles three copied legacy namespaces against the current API; current
source admission refuses those inline namespaces without emitting partial IR. It does not
claim a legacy accepted lift or normalize its private physical paths. The compiled rule and
companion operation restore independently captured accepted sources byte for byte, then the
runner compiles/lifts them and compares complete raw IR and waivers, positions included.
The permitted accepted-to-restored delta is empty; `check-facet-wrapping-delta.mjs` and its
independent corruption tests enforce that boundary. Real-source wrapping, production loading,
IR/Go propagation, Cases, live, canonical gates and lint/manual acceptance remain held;
broad validation and independent review belong to the source batch end.

## Input defaults: Fn159Task10InputDefaults

Omitted inputs use only `input[A](default = value)`. An `input[A]` token and a named string
slot without a token are required. Full positional calls, `:=`, input order, token identity
and action bindings retain their meaning. The native construction and the lifter independently
refuse omission, naming the token; the lifter also locates the call. Catalog enumeration never
selects an input default. Finite records, field-bearing enum cases, Option and Boolean/zero
constants remain expressible without an expression evaluator or transport fields.

The compiled rule uses this explicit selected-source omission inventory, captured before the change:

| Declaration | Old omitted value | Owners and omission sites |
| --- | --- | --- |
| mini `scheduleToStart: Timeout` | `Timeout.unset` | `client.start`: Product.scala:46, Protocol.scala:73/78, Queue.scala:27, MiniActivityRun.scala:18/25; shared token on `taskQueue.schedule` at Queue.scala:27 |
| mini `startToClose: Timeout` | `Timeout.unset` | those same `client.start` empty/partial calls and two deadline presets |
| mini `maxAttempts: MaxAttempts` | `MaxAttempts.one` | Product.scala:46, Protocol.scala:78, Queue.scala:27 omit it; Protocol.scala:73 and MiniActivityRun.scala:18/25 explicitly select one/two |
| activity `failure: Failure` | required | every selected `respondFailed` class supplies fatal/retryable, including bindings, capabilities, witnesses and realization evidence |

The three mini tokens are declared at MiniActivity.scala:41–43; shared failure is Activity.scala:27.
Foundations and actors declare no further input tokens. Full pre-edit raw mini and matching jar
classpath are retained in the task handover. Both deadline presets retain all eight distinct
classes and all seven evidence bindings. Required non-Timeout preset inputs are never eagerly
defaulted; the Timeout contract still requires and checks their own defaults.
The deadline helper requires `unset` and selects `expires` by name, independent of enum order;
ordinary input defaults remain arbitrary supported finite values.

The rule rewrites only inventoried top-level bare declarations with proven framework imports in
`temporal.features.activity.mini`; it preserves already declared defaults and the inventoried
shared required failure. Unknown names/types/packages, competing imports, shadows, nested inputs,
unsupported constructor spelling and ambiguous inventory refuse the document atomically.
It does not infer an old catalog from arbitrary code. For other shapes, manually inventory each
actual token owner, every omission and the exact old concrete value, including deadline needs;
declare only those proven defaults and keep all explicitly supplied required tokens required.
Resolve ambiguity before changing any declaration and compile/lift the selected sources afterwards.

```sh
mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.InputDefaultsTest
timeout 600 /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/migrate/prove-mini.sh umpire.migrate.RewriteInputDefaultsMini
```

The copied-mini proof removes the three explicit defaults from actual copied MiniActivity
declarations, parses that legacy text, runs the compiled rule and requires exact accepted-source
restoration, convergence and atomic unknown-input refusal. Accepted/restored source independently
compiles and lifts; complete raw mini IR and waivers compare byte for byte, positions included.
That legacy text is parsed, not compiled as the old API; the task's independent frozen pre-edit
jar/lift records the old API baseline. Original expected fixture JSON, immutable native tables,
payloads, pins and historical maps remain preserved. Production loading, real-feature editing or
migration, consumers, Cases, live, canonical Make/Go, lint/manual acceptance and broad validation
with independent review remain held for their owner checkpoints.

## Bounded integers: Fn159Task9BoundedIntegers

Fields and channel messages declare their own `UpTo[N]` bound. The compiled rule accepts a
top-level case class with one local literal nonnegative `Finite[Int] = Finite.upTo(n)` in its
`Finite[State]` derivation, and a typed `Channel[Int] = channel[Int](...)(using Finite.upTo(n))`.
It changes the field/message types, removes the retired scalar given or channel argument,
and rewrites positional or declaration-ordered named constructors, explicit typed-parameter
copy writers and associated Inbox sends together. Supported numeric writes are in-range literals,
bounded field reads and nonnegative increments capped with `.min(n)`. Scalar helper Ints and
named opaque integer types keep their meaning and source. Already current source converges.

Unknown/negative bounds, duplicate catalogs, ambiguous Inbox associations, nested declarations,
competing imports, shadows, escaped constructors, default field values and unsafe or untyped
writers refuse the document atomically. This source rule proves only these explicit shapes.
For other forms, manually inventory the actual field/channel owner and every writer, resolve a
constant nonnegative bound, change the type to `UpTo[N]`, wrap every bounded write with `UpTo[N]`,
update the associated Inbox type and remove only that retired local given. Keep unrelated
opaque catalogs and scalar helpers. Check ranges before wrapping expressions that may overflow.

```sh
mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.BoundedIntegersTest
timeout 600 /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/migrate/prove-mini.sh umpire.migrate.RewriteBoundedIntegersMini
```

The copied mini already uses UpTo; its source remains unchanged. A separately labelled scratch
legacy probe exercises actual field/channel/Inbox/write migration, idempotence and refusal.
Accepted and restored probes compile/lift independently with complete raw positions compared.
The authored lifter fix also gives `Option[UpTo[N]]` its bound-bearing synthetic identity: the
mini's old single `Option[UpTo]` wrapper had the correct 0..2 range, while two different bounds
previously collided. The complete authored pre/post artifacts retain that explicit identity delta;
ordinary declaration IDs, positions, values, Query budgets/totals and every native row stay fixed.
Existing fixture JSON stays unchanged; actual source-coordinate deltas are checked separately.
Production loading, real-feature migration, consumers, Cases, live, lint/manual acceptance,
canonical Make/Go gates and the single batch-end broad validation/review remain held.

## Capability kinds: Fn159Task8CapabilityNames

The confirmed server capability kinds are `Retryable` and `Expirable`; the selected
sources already use them. The compiled rule migrates `Retries` and `Deadline` through
explicit top-level imports from `temporal.capabilities.system`, fully qualified names
and member imports, including constructor/type, override and bound references. A paired
class/companion declaration in that exact package must explicitly extend `CapabilityOf`
and `CapabilityKind`. Unrelated symbols, literals, comments, `scheduleDeadline`, timer
objects and `Timeout` vocabulary retain their bytes. Accepted source converges.

Competing imports, shadowing, target collisions, references outside an import's visible
lexical scope, nested imports, aliases, wildcards and
unsupported declarations refuse the whole document atomically with a manual remedy.
This syntax proof establishes only these explicit source shapes, not arbitrary program
ownership. Other qualifier spellings, including `_root_`, are outside recognition and require
the manual route. For other shapes, manually resolve each legacy symbol's actual owner, inventory
all callers and imports, qualify owned references to the confirmed names, and retain every
unrelated reference. Rename owned `Retries.scala`/`Deadline.scala` and their co-located tests
to `Retryable.scala`/`Expirable.scala` after checking destination collisions; file moves are
outside Scalafix patches. Review every origin change separately from other deadline names.
The authored declaration files already have their current names and are not moved again.

```sh
mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.CapabilityNamesTest
timeout 600 /bin/bash model/migrate/prove-mini.sh umpire.migrate.RewriteCapabilityNamesMini
```

The copied-mini runner introduces actual legacy imports, constructors and the retry override
in `Completion.scala`, applies the compiled rule and requires exact accepted-source restoration,
idempotence and atomic ambiguity refusal. Accepted/restored sources compile and lift independently;
complete raw mini IR and waivers compare byte for byte, positions included. Legacy text is parsed
and transformed, not compiled against an old API. Independent native class/companion and isolated
lifter tests check emitted kind/origin IDs and actual mini applicability, bindings, override reason,
Property bodies and Query budgets/totals. The complete sixteen-table native baseline remains fixed.
Production loading, real-feature application, consumers, Cases, live, lint/manual acceptance and
the single batch-end broad validation/review remain held; this proof grants none of their credit.

Name a rule `Fn<spec>Task<task><Change>.scala`, with the same class and Scalafix rule
name in `umpire.migrate`. Keep its focused source fixtures in `test/`. Use the existing
Scalafix 0.14.9 dependency and JDK 25; `project.scala` isolates this rule build from
the DSL and Models. The public rule API follows the [Scalafix tutorial](https://scalacenter.github.io/scalafix/docs/developers/tutorial.html).
The fixture applies patches through the version-pinned [Rules.syntacticPatch](https://github.com/scalacenter/scalafix/blob/v0.14.9/scalafix-core/src/main/scala/scalafix/internal/v1/Rules.scala),
using [SyntacticDocument.fromTree](https://github.com/scalacenter/scalafix/blob/v0.14.9/scalafix-core/src/main/scala/scalafix/v1/SyntacticDocument.scala).
That internal API belongs only to the isolated source proof, not a production loader.

## Worked rename: Fn166Task9QueueAlias

`Queue.scala` and `ScheduledQueue.scala` import `foundations.taskqueue.queue` as
`foundationQueue`. The rule renames that local alias to `taskQueueSource` and its
qualified uses, inside `temporal.features.activity.mini.protocol`. It changes no
exported definition. Equal-length identifiers retain source positions as well as
the referenced action's identity. String literals and comments stay untouched.

An already renamed import or unrelated package/import is unchanged. A matching
old import with a target-name collision, another old binding/import, or a bare use
is refused with `ambiguous-queue-alias` before any rewrite. The patch is atomic so
a suppression cannot leave half a rename. This deliberately narrow source rule
does not infer arbitrary lexical bindings or implement a general rewrite engine.

Run from the repository root:

```sh
mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.QueueAliasTest
timeout 600 /bin/bash model/migrate/prove-mini.sh
```

The proof creates an ignored unique directory under `.flow/tmp/`, copies the mini,
the shared framework/kit/foundations/actors and linked API jar, then packages and
lifts the selected mini. The test runner applies the compiled Scalafix rule to
both copied queue files and checks that a second application is unchanged. It
packages and lifts the same source paths again, comparing full `activity-mini.json`
and waiver bytes with `cmp`, including all declaration identities and positions.
Its classpaths replace mutable compilation output with the corresponding frozen
jar. Logs and both jars/lifts remain in the printed proof directory for inspection.
It neither recaptures the native batch baseline nor writes generated feature data.

This is source compilation, transformation and exact lift proof. The existing
`Makefile` ScalaCLI Scalafix route currently loads built-ins only; custom-rule
loading/configuration and the executable Make route remain in fn-166.10. Real
features migrate only in the owner's sweep. Case, live, Go, lint and full-set gates
are separate and receive no credit from this source proof. A migration that
cannot preserve bytes must state its expected semantic/identity change; a change
the rule cannot express must record its concrete manual recipe.

## Qualified rules: Fn159Task1QualifiedActions

The rule qualifies bare `on` and `disabled` targets from one visible selective import,
including a renamed import, or from a matching wildcard import inside `from(declarer)`.
It qualifies the action a class calls, leaving its arguments alone. Imports no longer
used by the rewritten targets are removed; `from` retains its grouping header.
Ambiguous imports, shadowed names and imports a `from` cannot safely remove refuse the
document atomically with a manual qualified remedy. An ordinary wildcard import outside
`from` has no provable declarer: qualify those targets manually. Qualified source is unchanged.

```sh
mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.QualifiedActionsTest
timeout 600 /bin/bash model/migrate/prove-mini.sh umpire.migrate.RewriteQualifiedMini
```

The copied-mini runner introduces an imported bare poll into its isolated `Dispatching.scala`,
then applies this actual compiled Scalafix rule, checks idempotence and atomic ambiguity refusal,
and requires exactly the accepted source bytes. The shared proof packages and lifts the copied
mini before and after, comparing full IR and waiver bytes, positions included. This proves
source transformation and preserved mini identities/meaning; production loading, real-feature
application, consumer gates and the canonical Make route remain held.

## Qualified phases: Fn159Task3QualifiedPhases

The source rule reads the phase type from an explicit `Phased[S, P]` in the document.
It qualifies visible selective and renamed imports of that type's members wherever used:
init, states, effects, rules and properties. A wildcard is rewritten only when the enum's
cases are declared in that same document. Unrelated enum imports stay intact.
Ambiguous imports, shadowing, mixed importer statements, unsupported selectors and an external
wildcard refuse the whole document before patching. Framework wildcard imports are the known
DSL surface; arbitrary competing wildcards are refused. Inherited phase types with no local
`Phased` declaration and external wildcard imports require manual qualification through their
actual phase type, followed by removal of the member import.

```sh
mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.QualifiedPhasesTest
timeout 600 /bin/bash model/migrate/prove-mini.sh umpire.migrate.RewriteQualifiedPhasesMini
```

The copied-mini runner introduces a renamed phase-member import in its isolated
`Dispatching.scala`, rewrites rule and state references, checks idempotence and atomic
ambiguity refusal, and requires exact accepted-source restoration. The shared proof compares
the compiled complete mini's raw IR and waiver bytes, including every identity and position.
Production loading, real-feature application and consumer gates remain held.

## Phase entries: Fn159Task7Enters

The compiled rule removes one-state-argument `enter(s.copy(field = Type.case))`
effects and their rule references or `choose` calls, including simple local aliases.
It proves the field's explicit state type, `Phased[S,P]` projection and exact copy writer,
and a locally declared plain enum or the Recorded case's matching status fact. Extra facts,
counter writes, rejection and conditional or choice-bearing definitions stay intact.
An external phase is never assumed plain because its enum is absent.

An inherited facet requires explicit source context: the copied-mini runner supplies its
actual `Faults.scala` and `AnswerDispatch.scala`, proving AnswerPhase's plain enum and
the concrete writer. Without that context, only private definitions/owners are rewritten;
public effects require a caller inventory. Competing imports and lookalike DSL declarations,
unknown types/status, absent or incorrect writers and escaping uses
refuse the whole document atomically. Public definitions with unknown external consumers
require a manual inventory of every caller, the concrete projection/writer and phase status;
replace matching phase-only steps by enters, retain genuine logic, then remove unused defs.
The syntactic source harness does not establish arbitrary whole-program/export visibility.

```sh
mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.EntersTest
timeout 600 /bin/bash model/migrate/prove-mini.sh umpire.migrate.RewriteEntersMini
```

The runner restores legacy backOff/resume definitions and calls in copied actual Answering,
transforms them through Scalafix, checks convergence and escape refusal, and restores exact
accepted source bytes. Accepted/restored complete IR and waivers compare byte for byte,
including source coordinates. The retired source is parsed, not compiled as an old API.
This is distinct from the authored pre/post migration, whose two deleted function identities,
inlined rule bodies and explicitly inventoried source positions change while every native row,
ordinary declaration, Query budget/total and negative control remains fixed. Production loading,
real-feature sweep, consumers, Cases and live execution remain held.

`proof/check-enters-delta.mjs` checks the complete authored raw lifts, with two explicitly
named call expansions and a closed source-coordinate ledger for Answering/AnswerDispatch.
Every remaining value and declaration order must match. It validates all source positions
individually and keeps the full raw changes and deleted functions in an evidence ledger;
it does not globally remove positions or rename strings. `proof/test-enters-delta.mjs`
requires six independently mutated fact, state, outcome, choice, total and source controls
to fail. Function-body digests in that ledger are byte identities, not Go Behavior Fingerprints.

## Central bounds: Fn159Task5CentralBounds

The source rule maps literal named `Limits(steps = ..., actions = ..., search = ...)`
budgets to the eleven declarations in `temporal.Bounds`, including `fourWide`, `fiveWide`,
`twelveWide` and the old control budget's `eight`. A literal `1 << n` search is accepted.
It supports one local declaration in a lexical block, rewrites its unambiguous bare uses,
and removes that declaration. A visible selective central import keeps the accepted short
name; otherwise uses become `temporal.Bounds.<name>`. A block whose only remaining statement
is the Query expression is unwrapped. Applying the rule twice converges.

Unknown budgets, shadowed names, competing imports, qualified uses, multiple or nested
local declarations, and exported/member declarations refuse the document atomically.
This syntactic rule cannot establish external users of a public binding. For those bindings,
manually inventory every caller across the source set, match the exact budget to Bounds,
qualify or import the chosen central name at each caller, then remove the old declaration.
An unknown budget requires an explicit Bounds decision before removing it. Keep
`dispatchSteps` as an integer progress budget; it is not a Query Limits declaration.

```sh
mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.CentralBoundsTest
mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.BoundsInventoryTest
timeout 600 /bin/bash model/migrate/prove-mini.sh umpire.migrate.RewriteCentralBoundsMini
```

The copied-mini runner adds a known local bound only to the isolated Queue Query, applies
the actual compiled rule, and independently requires exact accepted-source restoration.
The shared runner packages and lifts before/after selected sources at the same paths and
compares complete raw mini IR and waiver bytes, including every source position.
The complete Bounds inventory and actual mini Query budgets are checked independently.
Production loading, real-feature migration, Cases and consumer gates remain held.

## Observation labels: Fn159Task6ObservationRead

The rule removes a literal retired `read` argument from `Observation(name, on, read)`
or named constructors that supply `on`. It resolves bare constructors through one
visible `framework.*` or `framework.Observation` import, or accepts the qualified
`framework.Observation` spelling when its package is unshadowed. `_root_.framework.Observation`
resolves the package explicitly even with competing imports. Names, entities, typed response selectors, comments,
strings and unrelated APIs retain their bytes. Already accepted source converges.

Shadowed names, competing imports, unsupported arguments and nonliteral labels refuse
the entire document. The rule does not resolve arbitrary constructor aliases or
`new` or explicit companion `.apply` expressions. These forms refuse atomically.
Manually resolve those constructors to the framework API, retain
name/on and typed selectors, and evaluate any effectful label explicitly before
removing its argument. Add a comment describing the actual selected response field.
An `Observed[Message]` identifies its protobuf message; the response selector supplies
the field path. A full `ActivityExecutionInfo` read selects `info`, not `info.attempt`.

```sh
mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.ObservationReadTest
timeout 600 /bin/bash model/migrate/prove-mini.sh umpire.migrate.RewriteObservationMini
```

The mini has no authored Observation. Its copied-source proof adds a separate scratch
probe from the isolated observation fixture, migrates both named and positional literal
labels through the compiled rule, and requires exact accepted-source restoration.
Accepted and restored probes compile/lift independently with complete metadata and
positions compared byte for byte. Probe evidence is separate from the copied mini's
unchanged complete raw IR and waiver comparison. Native API and isolated lifter tests
check the names, aliases, machine evidence, typed message/path/targets and invalid path.
Production loading, real-feature application and consumer gates remain held.
