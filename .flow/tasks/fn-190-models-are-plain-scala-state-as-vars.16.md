---
satisfies: [R2, R11, R12, R13, R14, R16]
---
# fn-190-models-are-plain-scala-state-as-vars.16 DSL completion: delete the record-state vocabulary from the DSL

## Description
**DSL completion phase (owner, 2026-10-17).** After fn-190.15 and fn-193.2 (cross-spec: plain compositions), delete what fn-190.4 made `private[framework]`: `RecordMachine` and its `Owner`/`Declaring` plumbing, `Rules`/`~>`/`where`/`always`/`rejects`/`enter`, `property`/`expect`/`across`, `query`/`Limits` construction, `assume`/`Assumption`, the named `leadsTo` overloads, `restrict`/`assuming`, and the internal tests that only exercised them. Rewrite DSL tests that still matter in the plain form. Export/testdata fixtures are out of scope (re-anchored in the IR/plugin phase).

**Native preservation seams (conductor, 2026-10-18).** Rewriting the remaining tests must preserve native fault-budget and durability behavior, including direct var budget selectors, budgeted native transition/crash binding, exact field ownership and nested immutable-record updates rebuilt atomically with the generated fault's own budget. Preserve selected finite/opaque validity evidence and reject malformed dormant metadata before callbacks. Preserve per-bound-effect choice-token validation without evaluating effects or guards; an unavailable declaration inventory must be explicitly unknown/refused, never invented empty or broadened to every choice of a Machine. Record any remaining separate-library metadata boundary honestly before completion.

**Construction/container split.** Delete `query`/`QueryDecl`/`QueryOn`/`QueryIn` and authorable `Limits` construction here; the broader `Query`/`ClaimQuery`/`Witness` container retirement belongs to fn-195.3. Any necessary interim bounds metadata is private and non-authorable. Rewrite substantive provenance, acquisition-laziness, crash/budget and environment tests in the plain form or as direct pure metadata tests; delete only the obsolete parser/authoring contracts and inventory the count/coverage delta.


**Native illustration/container preservation (conductor, 2026-10-18).** Preserve recursive structural validity for native State vars (records, concrete sum cases, bounded leaves and native containers) without enumerating whole-state catalogs. Retain selected opaque leaf evidence at the var's declaration scope and selected Channel message evidence at illustration admission. Native Inbox associations must name an exact owning var and rooted immutable-field/optional/exact-sum-case coordinate without evaluating selectors; foreign owners, missing/repeated/conflicting associations and duplicate channel storage remain refusals. Preserve the fourteen illustration-state contracts in native owned snapshots. Witness/Query/Property containers pending fn-195.3 retain their meaningful identity, immutable-copy order, selected-domain, typed-attachment, provenance and short-circuit controls through private metadata construction, never restored public builders.

**Size:** L

### Verification
- grep shows none of the retired names defined in model/framework/dsl.
- Native dsl/realize/samples suites and lint RuleSuite pass.
## Acceptance
- [ ] Record-state vocabulary deleted from framework.dsl
- [ ] Native suites pass

## Done summary
Plain Scala Machines now own the framework DSL surface. Record-state authoring, its effect/expectation engines and retired query/assumption builders are deleted; native provenance, fault budgets, durability, choice selection and retained-container contracts remain covered.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none)

### Implementation and boundaries

Source checkpoint `d7b02a26b7a6fa380ec52096a2b552fbbf3fa74c` replaces RecordMachine with a sealed private native declaration base for Machine/Derived/Built; Composition uses the native model interface. It deletes Owner/Declaring, Rules/Firing/Case and rule operators, record is/effect/expect contexts, property/across authoring, query/QueryDecl/QueryOn/QueryIn, authorable Limits and bounds mutators, assume/Assumption, named leadsTo overloads, restrict/assuming and record-callback choice overloads. Native bounded progress remains.

The existing product checker remains the executor. NativeCheck.Search retains on-demand successors and the shared Result retains concrete stored predecessor edges. Exact Programs.Fires/effect identity now avoids invoking one native effect twice during one rule evaluation. Class filtering, first enabled rule and branch replay retain their controls. Replay/find/sample/error and caller-budget configuration remain in fn-196.2.

Native budget selectors capture an exact owning var or rooted immutable record field. Budgeted native input/no-input transitions and crashes preserve declaration identity. Generated crashes rebuild sibling resets and their own budget write atomically per root; authored fault effects retain control over consumption. Generated storage-loss assumptions retain identity through copies and derivations. Recursive per-var structural evidence preserves selected finite/opaque validity and exact optional/sum Inbox paths, with one whole-State ownership inspection and no selector execution or whole-state catalog walk.

Native Unit choice effects retain immutable token inventories per bound action/class and replacement. Acquisition invokes no guard, effect or opaque helper. Fresh separate builds prove stored Model metadata and transitive parameterized effect-library/rebind metadata. Opaque outside-namespace helpers produce an explicit unknown-inventory refusal. The pinned recipe includes `-Yretain-trees`. Extra no-flag probes showed available static inventories can survive without it; they do not establish universal flag necessity. Their failed expectations are recorded as refuted probe assumptions. Documentation-only follow-up `270d8cd03b` qualifies this wording after the executable gates, with no executable changes.

Query/ClaimQuery/Witness/Property containers remain until fn-195.3. Their QueryBounds metadata is private and non-authorable; direct metadata tests preserve negative-bound validation, zero/empty names, copies, identity, duplicate collection and conflicts. Native claim-to-Property bridges and local metadata helpers remain, so the grep assertion concerns retired author definitions, not zero text matches of every old word.

Exclusive dead Rules/Phased realization phases, payloads and eager lint lookups are removed. Shared body interfaces still used by held export remain. Realization-mode compilation does not claim native full-body IR lifting, export equivalence or Go parity. REFERENCE contains no Scala filenames or source-definition pointers.

### Verification

Fresh own baseline was green, native 657 and lint 31, before edits. No baseline handoff or receipt shortcut was used. All commands were bounded foreground calls, with terminal statuses observed from their owning handles. Exact commands are in `.flow/tmp/fn19016/final-commands.md`; intermediate reds and scope notes are in `verification.md`.

| Final log | Observed result |
| --- | --- |
| final-native6.log | session75326 rc0, 600 tests across77 suites; strict compiler flags |
| check-final3.log | session27124 rc0, JUnit 46; real block scan 152 files/0 findings |
| recipe-lint-final.log | session10712 rc0, 31 actual RuleSuite tests |
| real-lint-final.log | session26052 rc0, 160 suites classified and real boundary/syntax/block/comment scans |
| recipe-compile.log | session32433 rc0, README native checker+samples compile |
| declarations-final.log | session25027 rc0, DSL+realize without checker roots/classpath |
| core-final.log | session1638 rc0, fresh framework-final.jar and dependency classpath |
| plugin1.log | session34611 rc0, fresh realization-plugin assembly |
| samples-final.log | session81837 rc0, six primary samples plus Inputs, combined-source realization mode |
| sample-client-final.log | session57329 rc0, same seven production samples as separate-library clients, strict flags and fresh plugin |
| choice-final-library.log / choice-final-client.log | sessions10984/27430 rc0, fresh strict library and 2 actual stored/cross-library choice tests |
| format-stability.log / format-check-final.log | rc0, repeat formatter unchanged and non-mutating check of all changed/new Scala files |

Retired author-definition grep and source-free REFERENCE checks passed. `gate classify` returned FULL1 against base ef952; no tier-B skip was claimed. Native and framework-check green receipts were written at source checkpoint d7b02a26. The later SEMANTICS wording commit is doc-only. Spec validation before completion returned valid with no errors/warnings and 18 tasks.

The final 46 checks use an absolute UMPIRE_FRAMEWORK_BUILD. Existing tests prove both real Go-backed SamplesSuite bodies are held before execution; their bodies/assertions are unchanged. `make model-check-changed` routes to held temporal.check.changed, so no Make green is claimed. Broad GuideSourcesTest, historical exporter/testdata fixtures, Temporal sources, schema/IR generation, Go and the full framework release gate remain held.

### Coverage accounting

Actual runtime registration delta is 657 to 600, net -57. `final-registration-delta.json` records the exact ordered-name comparison and suite deltas. 565 qualified test names are unchanged;92 old names disappear and 35 new names appear. These include 30 renames/rewrites, not 92 deleted contracts.

Three obsolete-engine suites remove 43 registrations, Effects 5, Expect 18 and Rules 20. `retired-test-classification.md` audits each old body and distinguishes retired parser/evaluator contracts from current native behavior. Automatic Draft status recording, standalone expectation concurrency/context machinery, old transitive expectation-helper restrictions and historical rule grammar are deliberately retired. Their exact proofs are not claimed to transfer to native Scala.

The remaining 19 removed slots are QueryConfiguration 7 owner-reach/bounds authoring controls, NativeAssumptions 3 authored assumptions, ResidualSignature 3 authored assumption/fairness controls, ComposedReference 2 authored fairness chains, Choices 1 record-list choice contract, NativeFault 1 record choice-program contract, RuleBindings 1 Firing parser lifecycle, and Sugar 1 Never pattern authoring. Private QueryBounds validation was restored as a renamed retained test rather than discarded with public Limits.

Five re-homed/compact retained-semantic registrations add back grouped monitor discovery in IrFile and four controls for ordinary sticky monitors, reordered typed member-fact witnesses, non-Model/direct duplicate composition admission, and native duplicate/disabled/foreign/nested headings. These protect real surviving behavior without restoring old authoring.

Substantive rewritten suites retain CoreBindings 16, RuleBindings 10, FacetBindings 10, DerivationBindings 3, Headers 9, Evidence 5, NativeCrash 13, DurabilityFields 8, NativeBudget 3, IllustrationState 14, native choice execution 3, PropertyIllustrations 9, Witness 7, IrFile 9 and ResidualSignature 7. They use real native State schemas or direct pure metadata. Nested constructor invariants, exact callback/capture/callsite identity, typed closure and curried proof validation, selected leaf evidence, malformed dormant metadata, per-binding budgets and no-callback acquisition assertions remain. Metadata-only packet tests do not claim held full native-body plugin capture.

### Handoff

Base commit is ef952d9adca06543b3b586b4e481393c999b470d. Four intervening conductor commits 3ec398f1c3, 52d67cbd87, a7f776bd77 and 9c3f85f066 own milestone/task metadata and are not worker framework implementation. Worker source and wording commits are d7b02a26b7 and 270d8cd03b. No MILESTONES edits, Temporal/compiler inputs, Go/schema/generated IR, exporter/testdata migration, ignored comparison corpus or frozen native baseline changes were made by this worker. The old unmerged runtime worktree remains untouched.

Independent review is deferred to the owner's phase batch; no SHIP verdict is claimed. No feature map or plan-sync mutation ran. The source/build lane is released only after the committed done receipt and final clean-tree check reported to the conductor.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3ec398f1c3cdf67c1d4ee3dd14f84f12e0d97e41, 52d67cbd87955d1a79c996aeba6ba5867beb5c05, a7f776bd777a7b1fbae25a540682474efb0d2dbb, 9c3f85f066e7413dbbeef589fca0b07ff89c2614, d7b02a26b7a6fa380ec52096a2b552fbbf3fa74c, 270d8cd03b32b98db7d0c7409518b84b423e6894
- Tests: timeout 600 mise exec -- scala-cli test --server=false --workspace .flow/tmp/fn19016/final-native-workspace model/framework/checker model/framework/samples --require-tests, timeout 600 env UMPIRE_FRAMEWORK_BUILD=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn19016/build mise exec -- scala-cli run --test --server=false --workspace .flow/tmp/fn19016/check-final-workspace model/framework/check --main-class org.junit.runner.JUnitCore -- framework.check.FrameworkLoopSuite framework.check.lint.BoundarySuite framework.check.lint.BlockFormRuleSuite framework.check.lint.CommentRuleSuite framework.check.lint.SyntaxRuleSuite framework.check.lint.ReferenceSurfaceTest, timeout 600 env UMPIRE_FRAMEWORK_BUILD=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn19016/build mise exec -- scala-cli test --server=false --workspace .flow/tmp/fn19016/recipe-lint-workspace model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite', timeout 600 env UMPIRE_FRAMEWORK_BUILD=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn19016/build mise exec -- scala-cli run --server=false --workspace .flow/tmp/fn19016/real-lint-workspace model/framework/check .flow/tmp/fn19016/Lint.scala --main-class checkerLint, timeout 600 mise exec -- scala-cli compile --server=false --workspace .flow/tmp/fn19016/recipe-compile-workspace model/framework/checker model/framework/samples, timeout 600 mise exec -- scala-cli compile --server=false --workspace .flow/tmp/fn19016/declarations-final-workspace model/framework/dsl model/framework/realize, timeout 600 mise exec -- scala-cli compile --server=false --workspace .flow/tmp/fn19016/core-final-workspace --print-class-path --compilation-output .flow/tmp/fn19016/build/framework-final.jar model/framework/checker, timeout 600 mise exec -- scala-cli --power package --server=false --workspace .flow/tmp/fn19016/plugin1-workspace --assembly --preamble=false --main-class-in-manifest=false --provided org.scala-lang:scala3-compiler_3 model/framework/reify/plugin -f -o .flow/tmp/fn19016/reify1.jar, timeout 600 mise exec -- scala-cli compile --server=false --workspace .flow/tmp/fn19016/samples-final-workspace model/framework/checker model/framework/samples --scalac-option -Xplugin:/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn19016/reify1.jar --scalac-option -P:umpire-reify:mode=realization, git diff --name-only --diff-filter=ACM -z ef952d9adca06543b3b586b4e481393c999b470d..HEAD -- 'model/framework/**/*.scala' | xargs -0 mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf, task_core_cp=$(tail -1 .flow/tmp/fn19016/core-final.log) && timeout 600 mise exec -- scala-cli compile --server=false --workspace .flow/tmp/fn19016/sample-client-final-workspace --scala 3.9.0 --jvm 27 --classpath "/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn19016/build/framework-final.jar:${task_core_cp#*:}" model/framework/samples --scalac-option -Werror --scalac-option -deprecation --scalac-option -feature --scalac-option -unchecked --scalac-option -Wunused:all --scalac-option -Wvalue-discard --scalac-option -Wnonunit-statement --scalac-option -Wsafe-init --scalac-option -Wimplausible-patterns --scalac-option -Yretain-trees --scalac-option -language:strictEquality --scalac-option -Xplugin:/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn19016/reify1.jar --scalac-option -P:umpire-reify:mode=realization, task_core_cp=$(tail -1 .flow/tmp/fn19016/core-final.log) && timeout 600 mise exec -- scala-cli compile --server=false --workspace .flow/tmp/fn19016/choice-final-library-workspace --scala 3.9.0 --jvm 27 --classpath "/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn19016/build/framework-final.jar:${task_core_cp#*:}" --compilation-output .flow/tmp/fn19016/build/choice-final-library.jar .flow/tmp/fn19016/LibraryChoices.scala --scalac-option -Werror --scalac-option -deprecation --scalac-option -feature --scalac-option -unchecked --scalac-option -Wunused:all --scalac-option -Wvalue-discard --scalac-option -Wnonunit-statement --scalac-option -Wsafe-init --scalac-option -Wimplausible-patterns --scalac-option -Yretain-trees --scalac-option -language:strictEquality, task_core_cp=$(tail -1 .flow/tmp/fn19016/core-final.log) && timeout 600 mise exec -- scala-cli test --server=false --workspace .flow/tmp/fn19016/choice-final-client-workspace --scala 3.9.0 --jvm 27 --classpath "/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn19016/build/framework-final.jar:/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn19016/build/choice-final-library.jar:${task_core_cp#*:}" --dependency org.scalameta::munit:1.2.0 .flow/tmp/fn19016/ClientChoices.scala --require-tests --scalac-option -Werror --scalac-option -deprecation --scalac-option -feature --scalac-option -unchecked --scalac-option -Wunused:all --scalac-option -Wvalue-discard --scalac-option -Wnonunit-statement --scalac-option -Wsafe-init --scalac-option -Wimplausible-patterns --scalac-option -Yretain-trees --scalac-option -language:strictEquality
- PRs: