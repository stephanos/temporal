---
satisfies: [R1, R2, R3, R4, R6, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.22 Construct complete isolated rule-aware derivation contributions

## Description
After committed .21, construct complete isolated rule-aware binding contributions for ordinary Rules, Facets and mixed core/rule derivation chains (partial R1/R2/R3/R4/R6/R9). Retain complete ordered operation evidence before native reduction loses operands or Rule metadata. Preserve every original full .4 obligation and its full .1/.3 prerequisites. This task neither builds complete Machines nor switches production emission.

**Size:** M, cohesive wide pipeline. Capture, operation retention, callback provenance and lowering share one inseparable contract; splitting them would leave admitted ruled derivations unusable. Reuse existing capture packets, operation history, RuleLowering and dependency validation rather than introducing a subsystem.
**Files:** `model/framework/dsl/{CoreBindings,Machine,RuleBindings,FacetBindings,Syntax,Reify}.scala` and affected focused tests; `model/framework/reify/plugin/{ReifyPlugin,PluginContext}.scala`, narrowly necessary shared `reify/body` hooks; new `model/framework/export/ConstructedDerivations.test.scala` and `export/testdata/constructedDerivations/`; `model/README.md`.
**Touches:** [model/framework/dsl/**, model/framework/reify/**, model/framework/export/ConstructedDerivations.test.scala, model/framework/export/testdata/constructedDerivations/**, model/README.md]

### Approach
- Freeze independent plugin-free old outputs and located refusals before production edits. Start with a genuine semantic RED from an old-admitted ruled derivation through the existing constructed collector. Freeze source and derived roots together, then compare the complete reached contribution projection; complete old Models can retain discarded source/intermediate Functions.
- Extend private ordered operation evidence at `CoreBindings.scala:7` and `Machine.scala:650` before maps, grouping, Rule.copy or restriction discard information. Retain complete raw source/operand snapshots, source rules mode, disabled distinctions, operation order and actual live identities. One RuleGroup with several cases is one operand; two groups targeting that action are duplicate operands. Preserve .16's initial duplicate raw Bindings admission and anonymous last-definition qualification.
- Capture exact derivation callsites/RuleGroups through the existing before-pickler capture and after-pickler runtime retention at `ReifyPlugin.scala:1720`. `Syntax.scala:1112` constructs the group using the source Owner before the derived object exists. Distinguish that runtime source from the compiler-known deriving declaration. Reuse typed body lowering, stored section/helper/facet carriers and reached-only refusal handling; never recover a foreign author body.
- Retain the measured authored effect discriminator from `Declarations.scala:979` through private compiler-owned evidence. Match forwarded-def identity rather than callback identity or IR equality. Bind it to the original row/template and capture; framework-owned replacements explicitly retain their new adapted callback provenance while ordinary author Rule copies still lose metadata. Keep public Rule's seven-field product and StepBinding/RuleGroup author behavior unchanged.
- Replace the blanket operation refusal at `RuleBindings.scala:281` with validation of each initial/intermediate operation before final selection. Reuse `RuleLowering.scala:50` and `CoreBindings.dependencies` for ordered roots, complete helper/Type closure and exact external references. Bare rebind keeps guards/classes and installs the replacement effect; RuleGroup rebind replaces rows in place; admitted extensions append in operand order. Preserve untouched source roots, changed derived roots, exact names and positions, including disabled-to-complete-function replacement.
- Keep original facet assembly/namespace/Rules/action/class provenance and assembly-specific projection/writer context distinct from replacement provenance. Repeated and alternating source/derived/A/B collection must not mutate original captures or shared templates. Validate historical ruled mode even after empty restriction and every invalid operation hidden by later overwrite/restriction.
- Update only README's staged capability and obsolete blanket unsupported-history wording. Keep full Machine/evidence/monitor/refinement assembly, model-wide shared helper/source-index reconciliation, cross-level facet ownership, nested witnesses, channel/G01/name/consumer validation, production activation, reader retirement and R5 open in .4. Leave original SEMANTICS, designs, goldens and six sample sources unchanged.

### Investigation targets
**Required** (read before coding):
- `model/framework/export/Declarations.scala:944-1135` and `reify/body/Expressions.scala:116-139` - authoritative derivation/effect-identity behavior.
- `model/framework/dsl/Machine.scala:85-150,622-795` and `CoreBindings.scala:7-47,132-225` - products, native reduction, ordered evidence and dependency validation.
- `model/framework/dsl/RuleBindings.scala:274-445` and `FacetBindings.scala:179` - exact live provenance, instantiated contextual templates and contribution closure.
- `model/framework/dsl/Syntax.scala:625-670,1099-1141` and `reify/plugin/ReifyPlugin.scala:1720-1989` - Firing ownership and typed/runtime capture phases.
- `model/framework/export/ConstructedRules.test.scala` and `ConstructedFacets.test.scala` - independent old oracle, complete projection, saved-tree replay, jar/warm tests and native timing controls.
**Optional**:
- `model/framework/export/ConstructedBindings.test.scala`, `dsl/CoreBindings.test.scala`, `dsl/RuleBindings.test.scala` - preserve existing derived/raw admission and malformed metadata checks.
- `.flow/tmp/fn174-derivation-plan-probe.ZYkmGi0Q/attempt2/` - frozen planning probe source, commands, jars, complete Models and actual diagnostics.

### Key context
Planning probe runner handles 29560 and 21419 joined with actual exit 0; both plugin-free fixture compilations exited 0. The first attempt's external-effect placement/raw Step-lambda refusals remain recorded, not rewritten into positive evidence. Corrected attempt2 independently establishes SameNamed/SameLambda/AlphaLambda/DisabledRebound/Chain exit 0 and DistinctNamed/DisabledExtended/RestrictedExtended/HiddenInvalid exit 1 at their actual source lines. Distinct named defs with equal bodies remain distinct effects; forwarding lambdas to the same def match even after parameter alpha-renaming. A disabled bare rebind yields the complete named Function root, not a ruled root. Empty restriction retains rules mode. Repeated rebind names the changed root `chain.rules.go`; invalid effect merging still refuses after empty restriction. Preserve these measured distinctions and investigate additional chain bookkeeping against the old oracle before normalizing it.

The .21 single-root A/B facet oracle and frozen combined A+B/B+A left-writer helper qualification remain independent. Isolated closure equality does not establish union equality. Exact SourceFile-collision position compatibility remains a separate full .4 obligation. Do not rename helpers, clear all foreign positions or newly reject old-admitted shared rules-only traits.

No new dependency, public author API, annotation, IR schema, exporter fallback or second production producer. No Temporal/tools/Go/root gate/generated/registry/module-map/migration/regeneration work. Review remains deferred at the owner's source-batch boundary; no SHIP claim. Preserve all prior sources, artifacts, logs, jars and caches.

### Quick commands
Run focused suites serially where Scala build outputs overlap, retaining actual process handles and logs.

```bash
mise exec -- scala-cli test model/framework/dsl model/framework/realize --server=false
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedDerivationsSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedBindingsSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedRulesSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedFacetsSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedEndpointsSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedHeadersSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.FiniteSchemaSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ReifyBoundarySuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ReifySamplesSuite
```

Run pinned formatting on exact owned Scala paths using `mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf <exact paths>`. Root pre-plan DSL/realize baseline passes 238 cases at e416dec5c693; handle 56352 joined exit 0, log `/tmp/fn174-derivations-plan-dsl-baseline-20261015-a.log`. Existing broad gates remain held and uncredited.

## Acceptance
- [ ] Freeze independent complete plugin-free old Models, source/jars/commands and located refusal controls before production edits, including a genuine old-admitted constructed semantic RED. The new standalone derivation suite proves complete ordered binding and reached Function/Type/Channel/Hole closure equality with only Position removed and positions checked separately. Cover ordinary Rules, Facets and core sources receiving RuleGroups, zero-to-six inputs, named/forwarding/anonymous/partial/curried effects, whole/class/mixed rules, disabled/timer/fault actions, untouched neighbors and source/derived roots. Preserve measured historical admission rather than broadening raw inline-effect or opaque-helper forms (R1/R3/R4/R9).
- [ ] Retain complete immutable ordered raw source/operand/operation evidence and validate every intermediate before reduction, including after later restriction/overwrite. Preserve initial duplicate Bindings admission and its .16 qualification; reject actual duplicate operands, unknown replacement, already-bound extension, phase-bearing derived cases, ambiguous effect merges and bare extension of ruled sources at their measured old stages. One multi-case RuleGroup counts once. Include empty operations, disabled rebind, empty restriction retaining rules mode, repeated chains, both operation orders and Derived-after-Derived sources (R3/R4/R9).
- [ ] Compiler-owned effect/callsite metadata preserves measured forwarded-def identity and all exact source/assembly/action/class/row/callback/adapted-effect provenance. RuleGroup ownership does not counterfeit the source as its declaring derived object. Missing, copied, stale, conflicting, malformed, wrong-arity/type/lexical/callee/curried-call or foreign evidence refuses contextually. Reuse shared RuleLowering and dependency validation; preserve product/copy contracts, reached-only refusals and stored-carrier admission without foreign body reads or new author syntax (R1/R2/R4/R9).
- [ ] Bare replacement retains original guards/classes/order, RuleGroup replacement replaces in place and admitted extension appends in authored order, with exact source/changed root naming and positions. Original and derived captures/templates remain immutable under repeated source/derived/source and A/B/A collection. Measure existing native construction/table/group/operand acquisition separately; collection afterward adds zero operand, callback, guard/effect, catalog, getter, endpoint/refinement or projection/writer evaluations. Validate history without publishing unreachable intermediate helpers (R3/R4/R6/R9).
- [ ] On frozen final bytes, pass standalone derivation equivalence/refusal controls, separate native JVMs, same-unit/cross-file/precompiled facet/helper jars, relevant warm body-edit/restoration invalidation and complete current-reader saved-tree replay. Pass all task Quick commands, exact pinned formatting, unchanged sample/oracle/protected manifests and tested-to-committed blob checks; join every process. Record production Scala delta and all actual limitations. README retains full .1/.2/.3/.4, model-wide helper/source-index reconciliation, complete Machine/evidence/monitor/refinement/ownership/G01/consumer integration, R5, source-batch review, sole activation and reader/public Bindings retirement as open (R1/R2/R3/R4/R6/R9).


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
