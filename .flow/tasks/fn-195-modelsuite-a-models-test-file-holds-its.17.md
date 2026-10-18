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
Generic Realization and ordinary native fixture kits now expose explicit waited-class projections to the framework's private checked-coverage consumer. Coverage retains exact machine/action identity, only the finite catalogs needed by performed and awaited classes, immutable class snapshots and the original acquisition failure; both repeated acquisition and direct retained-carrier admission refuse a changed machine association.

Source checkpoint frozen at c9724c3c83f9de9f9807c7f3b886ca3f56ac3d70. Both refreshed quality axes returned zero findings, and the conductor explicitly authorized Flow closure. No command handle remains active.

### Acceptance evidence

- Generic actual serverSteps and outside-package custom kits admit exact performed/waited values. Supported empty waits admit; opaque steps, unsupported kits, foreign same-class machine instances, foreign/unbound ordinary performances and malformed inputs refuse. Existing raw Performance products/copies and Realization copies retain their behavior.
- The native controls validate every supplied alternative and repeated/concrete class path, correlated whole-record domains, empty selected catalogs and malformed retained metadata. A genuine production CompositionRuntime Boolean sync supplies the four literal FF/FT/TF/TT rows; checked coverage admits only the selected class and rejects another valid but uncovered row. The unchanged expectedMachine signature remains an individual machine boundary.
- Native validation and ordinary performed-class validation precede wait projection. Existing selected-choice controls pass through the new boundary. Constructors invoke no new projection. Retained snapshots, changing exact association, original exceptions and swallowed reentry controls pass. The admitted finite fixtures keep guard/effect/command/dependency/conversion/end poison counts at zero and retain cause, wait bound and cleanup. Script-when, evidence, controls and a kit-declared dependency confer no coverage.
- Fresh source and separately packaged clients pass under unchanged strict flags and pinned tools. Checker-absent declarations compile/load, retain the existing missing-runtime refusal and refuse external access to the private coverage API. SEMANTICS documents this exact bounded protocol.

baseline: green. Pre-edit checkercontrol.CheckerTest ran 4 tests and NativePerformanceSuite ran 6, both exit 0. The initial receipt check required execution. See .flow/tmp/fn195-native-wait/baseline.md and its two logs.

### Final verification

| Recipe | Exit | Executed tests |
| --- | --- | --- |
| source-v5 | 0 | 30, NativePerformanceSuite 6 and NativeWaitSuite 24 |
| library-v3 package | 0 | package only |
| client-v3 against that immutable library | 0 | 30, the same two suites |
| declarations-source-v3 without checker | 0 | 3 |
| declarations-library-v3 package | 0 | package only |
| declarations-client-v3 without checker | 0 | 3 |
| final-checker-v3.log, exact parent Quick at frozen source HEAD | 0 | 4 |
| verify-v3.sh and final-bindings-v3 checksum check | 0 | source/tool/artifact/refusal bindings |

Four touched Scala files were formatted and a repeat left identical bytes before the final snapshots. Scala CLI 1.17.1, Scala 3.9.0, JDK 27, munit 1.2.0 and ScalaPB 0.11.20 are recorded per recipe. The frozen IR dependency remains b9e0538afffad2b33b9fc8fc9eb2a3f4e63b10928fa38506b585747535e85c92.

Native library SHA-256 is 0eeab6acc7bb808a8f7efd71413b572abe01f46b360172d9348c3a7a4ef38b15. Checker-absent declarations library SHA-256 is 82d0b53b6e979ed68fefbe84572d325888badc39e9b3587c1c0347bcdd9f6a4e. The 516-row final manifest is .flow/tmp/fn195-native-wait/final-bindings-v3.sha256, SHA-256 d1b993b96c6e2bab8c3e36ec8435d913e42ca9533b7c96c3c7094f14800fda2a. Complete commands, source snapshots, tool receipts, terminal exits and logs live in the named recipe directories. The exact Quick gate receipt is .flow/tmp/green-receipts/c9724c3c-unittest.json.

Gate classify returned FULL, exit 1, with `unmatched: model/framework/SEMANTICS.md`. The parent Quick and native focused gates ran; no docs-only skip or full native batch pass is claimed. Actual Make/Go/export and the broad batch gates remain held.

### Retained failures and limits

The API-absence RED is committed at 78069e14fb. Both changing-owner REDs have preserved pre-fix sources; the direct retained-carrier RED is bound by red-retained-owner-inputs.sha256. attempts.md preserves all failed fixture attempts and intermediate checkpoints. The v5/v6 eager-catalog/malformed-metadata failures retain their original logs but lack a contemporaneous pre-fix source snapshot and receive no immutable-source or final aggregate credit. Earlier ca859 proof rows remain historical and are not refreshed or summed into final counts.

The conductor's e573 quality reports recorded standards 0 findings and correctness 1 Should Fix. After reading receiving-code-review, the numeric hypothesis was reproduced: red-numeric.log exited 1 with exactly six intended failures and 18 passing existing wait controls. Ordinary and genuine composed Int catalogs wrongly accepted Long values at performed, awaited and retained admission boundaries; ordinary cases also established the existing native input-domain refusal. The one-line fix uses per-value `.equals` rather than whole-list equality. All six regressions pass in both final 30-test recipes. Pre-fix inputs and the entire earlier e573 checkpoint remain frozen; attempts-numeric.md records this correction.

Fresh conductor-owned v2 audits at c9724c3c83 returned correctness 0 Critical/0 Should Fix/0 Consider and standards 0 Should Fix/0 Consider. The full original and refreshed reports and explicit closure acceptance are retained in .flow/tmp/fn19517/quality-reports.md. Root independently verified both historical and final bindings, nested manifests, test snapshots and formatted bytes. These are bounded read-only quality audits, not formal phase-batch review, broad-gate, R4 or C credit; those obligations remain deferred.

The supplied-vector protocol consumes concrete class paths; it executes no Checker.sample integration. No universal custom-catalog purity claim is made. Actual system-kit adoption, complete realization discovery, capability generation, P/X consumer integration, exact conformance binding and C/ModelSuite eligibility remain open in .11 and its prerequisites. Closing the independent .15/.16 feasibility reports cannot replace their required production mechanisms. Final README/global guidance synchronization remains .5. All .8/.14, fn-196.2 and Temporal/IR/Go/export/Case/server/publishing holds remain unchanged; export compile-only permission is still ungranted. Old .10/.18 proof files and manifests were not edited.

Similar code search reused NativeRuntime.classesOf and the existing raw acquisition/choice validation, extended ConstructedRealization and generic serverSteps, and followed the existing braced-package public-client fixture pattern. One helper supplied read-only validation analysis and task-owned scratch recipe preparation; the worker owned all production edits, compiler/JVM commands and commits.

stage: impl-review - skipped(config: REVIEW_MODE=none; explicit owner phase-batch deferral)
Tier: session (jev-unavailable(no_key))
Tracker sync: n/a (bridge inactive)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 78069e14fb766e52d523babb8f53080e99a875b8, ca859ac1ae1c403ba04e8ea9bef934e02f7be6e0, e573fe2e7a7edf43eff3982447d9de305ebcfacc, c9724c3c83f9de9f9807c7f3b886ca3f56ac3d70
- Tests: mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only checkercontrol.CheckerTest, mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only framework.realize.NativePerformanceSuite, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-wait/run.sh source-v5 source, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-wait/run.sh library-v3 library, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-wait/run.sh client-v3 client /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-wait/library-v3/native-wait-library.jar, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-wait/run.sh declarations-source-v3 declarations-source, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-wait/run.sh declarations-library-v3 declarations-library, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-wait/run.sh declarations-client-v3 declarations-client /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-wait/declarations-library-v3/native-wait-declarations.jar, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-wait/verify-v3.sh, sha256sum -c .flow/tmp/fn195-native-wait/final-bindings-v3.sha256, sha256sum .flow/tmp/fn19517/quality-reports.md, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf model/framework/realize/Constructed.scala model/framework/realize/Realize.scala model/framework/realize/NativePerformance.test.scala model/framework/realize/NativeWait.test.scala
- PRs: