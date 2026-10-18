---
satisfies: [R4, R7]
---
# fn-195-modelsuite-a-models-test-file-holds-its.17 Add framework native waited-class projection and checked coverage

Touches: [model/framework/realize/Constructed.scala, model/framework/realize/Realize.scala, model/framework/realize/NativePerformance.test.scala, model/framework/realize/NativeWait.test.scala, model/framework/SEMANTICS.md, .flow/tmp/fn195-native-wait/**]

## Description
Implement the bounded framework protocol proved feasible by .10 for an explicitly supplied realization. Add exact performed/waited coverage without depending on automatic realization discovery or capability generation. Generic Realization and ordinary native fixture kits receive production controls; actual system-kit adoption remains held.

**Size:** M, five expected files, four acceptance groups and one adapted native acquisition protocol.
**Files:** `model/framework/realize/Constructed.scala`, `Realize.scala`, `NativePerformance.test.scala`, new `NativeWait.test.scala`, and the native realization section of `model/framework/SEMANTICS.md`. Final README/global guide synchronization belongs to existing .5.

### Approach
- Extend `Constructed.scala:15` with protected `declaredWaitedClasses: Vector[ClassRef]`, defaulting to an explicit unsupported refusal, and private[framework] final `nativeCoverage(expectedMachine: MachineDeclaration[?, ?, ?]): ConstructedRealization.NativeCoverage`. Keep the immutable carrier private[framework] with construction private[realize]; it admits supplied `Vector[Vector[ClassRef]]` through exact owner/catalog/class checks. Preserve raw nativePerformances behavior and current constructor/product/copy shapes. Adapt names to local conventions only when necessary; keep this mechanism bound.
- Add public `nativeWaitedClasses: Vector[ClassRef]` to SystemStep at `Realize.scala:33`, defaulting to an explicit unsupported refusal. Generic Realization overrides declaredWaitedClasses by flattening that projection from its actual serverSteps; an empty serverSteps vector is authoritative empty. Custom kits explicitly override to declare empty or supported waits. Preserve kit-owned cause/bound values. Invoke no new virtual projection or association callback from a superclass constructor.
- Reuse `NativeRuntime.classesOf` for whole and composed finite classes. Validate exact Model/action ownership, declared input catalogs and every supplied alternative/repeated/concrete sampled class against performed or waited coverage. Add ordinary unbound/malformed performance checks at this new consumer boundary while preserving selected-choice controls.
- Return retained immutable checked coverage for the request. Preserve deterministic repeated acquisition or explicit failure before partial coverage escapes; retain the original cause on projection failure/reentry. Keep script-when, evidence, control and dependency-only actions as negative coverage controls.
- Move the scratch protocol controls into NativeWait.test.scala, with an outside-package source/library client using the existing inline-fixture style. Update the native protocol/refusal semantics in SEMANTICS. Existing .5 consumes this interface for final framework guidance; .11 later applies coverage to actual P/X declarations under S policy.

### Investigation targets
**Required:**
- `model/framework/realize/Constructed.scala:15` - final acquisition and validation ordering.
- `model/framework/realize/Realize.scala:33` - open SystemStep and generic serverSteps.
- `model/framework/realize/NativePerformance.test.scala:5` - native generic/custom and choice controls.
- `model/framework/dsl/NativeRuntime.scala:90` - centralized finite class projection and unavailable runtime refusal.
- `.flow/tmp/fn1952-conformance-proof/WaitProbe.scala:88` - private protocol feasibility and poison counters.
- `model/framework/SEMANTICS.md:1292` - task-owned native realization documentation.
**Optional:**
- `model/framework/realize/Conversion.scala:570` - pure sourceActions dependency projection for negative coverage only; execute no conversion.

### Verification and handoff
At implementation admission, baseline the exact current native performance controls, then run new native wait/coverage controls and fresh source/separately packaged library clients under the existing strict flags and pinned tools. Use a fresh workspace for each complete recipe. Retain exact selectors, nonzero counts, source/artifact hashes, commands and terminal exits in task-owned `.flow/tmp/fn195-native-wait/`. DSL/realize declaration compile/load stays checker-independent; a projection that needs the existing runtime preserves its unavailable-checker refusal. Keep guard/effect/command/dependency/conversion/end counters zero for the admitted finite fixture catalog.

This leaf supplies checked coverage to later .11; complete candidate selection, ModelSuite registration, P/X integration, eligibility and system-kit adoption remain their existing obligations. Preserve .8/.14 and fn-196.2 holds, .9 -> .11, and phase-batch review deferral. Temporal reads, actual repository Make, Go, export/lifting/IR/schema/backend/generated/regeneration, server/recorded-Run and Case operations remain held; export compile-only permission remains pending and ungranted.

## Acceptance
- [ ] Generic Realization and an ordinary custom native fixture kit project actual waited classes with exact input values through the new explicit request. Empty supported waits admit; opaque steps and unsupported kits refuse. Same-primary-class foreign Models, actions and malformed input catalogs refuse at the checked exact-owner boundary. Existing raw acquisition and constructor/copy behavior remain compatible.
- [ ] Checked coverage validates every supplied finite alternative, repeated class and concrete sampled path, including whole/composed actions through the existing class seam. Unsupported or wrong-input classes and unbound/foreign ordinary performances refuse. Script-when, evidence, control and dependency-only classes earn no waited coverage.
- [ ] Native validation precedes projection; current selected-choice controls pass; no new superclass-constructor callback runs. Retained checked coverage stays immutable across later projections, and repeated acquisition/failure/reentry cannot publish partial coverage. Cause, wait bound, cleanup and dependency meaning remain intact, with zero poison callbacks for admitted fixture catalogs.
- [ ] Focused native and fresh outside-package source/library controls execute nonzero counts under unchanged strict flags; declaration-only compile/load and existing missing-runtime refusals hold. SEMANTICS describes the exact framework protocol and refusals. The receipt leaves actual system-kit adoption and .11's real P/X/suite integration open, and .5 retains final guidance synchronization.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
