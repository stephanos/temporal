# Model migrations

DSL changes ship a task-named Scalafix rule under `model/migrate/rules/`, prove it on copied `activity_mini` sources before the real-feature sweep, and follow `model/migrate/README.md`; production loading remains held in fn-166.10.

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
