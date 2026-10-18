# Native conformance admission proof

Conformance C remains blocked. The executed experiments expose incomplete realization discovery and missing native capability generation. A private kit-owned awaited-class projection passes bounded native controls. This report completes the proof investigation for fn-195.10; it does not implement R4, admit C, or certify a complete realization inventory.

## Source and evidence boundary

The source checkpoint is `59bbc51395c3d7f34e129c0dc518a28110102990` on `umpire`. All experiment sources, copied framework sources, compiler workspaces, jars, command records, diagnostics and terminal statuses live under [fn1952-conformance-proof](../../tmp/fn1952-conformance-proof/). The durable change is this report. Product DSL, checker, realization and sample sources are unchanged.

The experiment copies 94 ordinary Scala files from the established native DSL, realization and checker roots, including `model/OutputPaths.scala`. It excludes their test fixtures and all held roots. Three scratch adaptations implement the one investigated acquisition alternative. `MachineDeclaration` owns a lazy epoch, final `nativePerformances` associates a successfully validated candidate with that epoch, and generic `Realization.runs` initializes before that association. The last change responds to an executed safe-init refusal. These adaptations are absent from product sources.

Scala CLI is 1.17.1, Scala is 3.9.0, JVM selection is 27, munit is 1.2.0 and ScalaPB runtime-grpc is 0.11.20. Every compilation preserves `-Werror -deprecation -feature -unchecked -Wunused:all -Wvalue-discard -Wnonunit-statement -Wsafe-init -Wimplausible-patterns -Yretain-trees -language:strictEquality`, with `--server=false` and a fresh workspace per recorded version. No suppression or dependency upgrade resolves a refusal. CLI directive notices and the pinned runtime's `sun.misc.Unsafe` notice remain visible in logs.

The existing IR jar is only an unchanged native dependency. Both original and scratch copy have SHA-256 `b9e0538afffad2b33b9fc8fc9eb2a3f4e63b10928fa38506b585747535e85c92`. No conversion, exporter, reify, IR generation, Go, actual repository Make, Testpilot, server, Case producer or `model/temporal` content was used. The historical Bloop PID 93084 and other owners' worktrees remain untouched.

## Executed observations

[run.sh](../../tmp/fn1952-conformance-proof/run.sh) records expanded commands, captured output and terminal status before returning. Each version retains caller-source copies and hashes. The later versions also retain the three adapted native sources and a full native-source manifest. The source arrangement compiles normal main definitions and test callers together through Scala CLI's main/test stages. The library arrangement first packages the native implementation and declaration fixtures, then compiles only the three caller files against that jar and the frozen schema dependency. Its `ide-inputs.json` and `ide-options-v2.json` retain that boundary and every strict option.

| Observation | Expected and observed terminal result | Retained evidence |
| --- | --- | --- |
| Source behavioral controls, version 10 | Exit 0; 28 tests | [source.log](../../tmp/fn1952-conformance-proof/combined-v10/source.log) |
| Source completeness oracles, version 10 | Exit 1; exactly two intended assertion failures | [oracle-source.log](../../tmp/fn1952-conformance-proof/oracle-source-v10/oracle-source.log) |
| Separate native/declaration library | Exit 0; library jar written | [library.log](../../tmp/fn1952-conformance-proof/library-v10/library.log) |
| Packaged-library behavioral controls, version 10 | Exit 0; 28 tests | [client.log](../../tmp/fn1952-conformance-proof/client-v10/client.log) |
| Packaged-library completeness oracles, version 10 | Exit 1; exactly two intended assertion failures | [oracle-client.log](../../tmp/fn1952-conformance-proof/oracle-client-v10/oracle-client.log) |

The 28 behavioral controls are inventory 10, wait admission 10, capability/expectation 7 and replay eligibility 1. A passing counterexample control confirms the observed gap, rather than satisfying the missing production requirement. The independent oracles demand a complete inventory of two lazy declarations and an executable generated capability predicate. They fail with obtained inventory size 1 versus expected 2, and `native Capabilities.claim discarded the executable predicate`. Neither failure comes from compilation, missing fixtures, an empty test selection or a timeout.

Version 11 strengthens the caller boundary with a client-defined subclass of a library base, external realization owners on both sides of the jar, and two concrete MUnit selector suite objects. The immutable version-10 library remains valid because its packaging inputs do not change. Final version-11 receipts are recorded in the verification addendum below.

Earlier failures remain under `inventory-v1/` and `combined-v1/` through `combined-v9/`. Compiler-only failures include fixture syntax, strict equality, private access, captured non-finite constructor data, import ambiguity and nested actor initialization. Version 8 reached runtime but its aggregate failed on fixture mistakes, including the required fault actor, bare action ownership and an opaque choice program. Those rows receive no final aggregate credit. Version 3 is an additional meaningful negative result. Passing `this` to the new association hook at the original generic acquisition point fails safe-init because `runs` is still uninitialized. Its source snapshots and exact diagnostic are retained in [combined-v3](../../tmp/fn1952-conformance-proof/combined-v3/). Initial `inventory-v1` has its command, log and status but no pre-edit source snapshot, so it supplies diagnostic history only.

## Realization universe and counterexample

The required candidate universe is the set of distinct successfully admitted `ConstructedRealization` values for the exact Model instance, obtained from every eligible stable realization declaration in the admitted finite source/library artifact set. It must include generic and custom kits, eager and lazy owning values, inherited declarations and external owners. Aliases to one value count once by reference; equal-looking distinct values remain distinct. Machine class names, structural equality, empty performance vectors and package proximity cannot replace exact `candidate.machine eq model` ownership.

The experiment proves no complete implementation of that universe. Its chosen alternative is an owner-local acquired-value epoch, investigated through the smallest counterexamples. It safely observes successful native acquisition, deduplicates references, closes before selection, refuses zero or many implicit candidates, validates explicit selection by exact owner and reference membership, and invalidates on a distinct late association. Reacquiring the same candidate after close remains valid. Invalid dormant native machines refuse before association and dependency exposure. Those mechanics do not supply missing declarations.

The minimal counterexample uses ordinary declarations.

```scala
val first = generic(model)
lazy val second = generic(model)
```

Before reading `second`, automatic selection accepts `first` from an observed set of one. Reading `second` afterward refuses as a distinct late candidate and invalidates the epoch. Reading `second` before selection produces two candidates and refuses omitted `through`. Thus declaration-access order changes admission. Late invalidation cannot make the earlier interval of implicit authority into a completeness proof. Returning an immutable declaration earlier, or calling the observed set a complete inventory, would preserve that error.

| Boundary | Executed behavior in source and library controls | Consequence |
| --- | --- | --- |
| Zero and one acquired candidate | Zero refuses; one eager generic selects the same reference | No invented realization; completeness still absent |
| Many acquired custom candidates, reverse order | Omitted selection refuses both orders; either explicit same-owner reference admits | Order-independent multiplicity works only for the acquired set |
| Second lazy generic | Selection-before-access accepts one; access-before-selection observes two | Minimal completeness failure |
| Inherited and external lazy getters | Suite construction leaves them unacquired; later access refuses after earlier selection | Construction of the suite is not construction of all declaration owners |
| Eager inherited and external values | Once constructed, all three distinct candidates appear | Observation works; discovery is not established |
| Eager and delayed custom kits | An initialized eager kit associates; the ordinary custom kit remains unseen until requested | The generic constructor hook cannot certify custom construction completion |
| Repetition, dependency acquisition and generic copies | Repeated same-reference acquisition deduplicates; a distinct generic copy makes two | Identity, rather than case-class equality, governs candidates |
| Invalid dormant native machine | Repeated acquisition and dependency requests refuse; dependency counter stays unchanged | Native validation precedes association |
| Same primary class, foreign exact instance | Explicit foreign selection refuses; independent owners select their own references | Exact instance partition is available |
| Released owners | Weak references to the machine and realization clear within the bounded GC control | This owner-local cycle can be collected; no global strong registry is introduced |

The supported experiment universe is deliberately narrower than the required declaration universe. It contains already acquired values and the explicitly constructed fixture owners. No complete contract is admitted for untouched lazy/external/custom declarations, arbitrary factory-created owners, getter recursion or getter failure. Retained bounded-getter metadata and a framework construction-completion epoch remain unproved alternatives. This task stops after the first alternative's counterexamples instead of building three speculative mechanisms.

The prerequisite must supply an authoritative finite declaration-owner inventory, exact runtime receiver bindings, safe acquisition of every supported getter/kit after initialization, and a close boundary before any default selection becomes authoritative. It must refuse any unsupported boundary before that selection. Metadata that only enumerates getter symbols still owes the exact external receiver and safe construction proof. A late-refusal rule alone cannot satisfy this contract. No author candidate lists, registration calls, relocation rules, package scans or global registry are proposed as substitutes.

## Performed and awaited classes

[WaitProbe.scala](../../tmp/fn1952-conformance-proof/WaitProbe.scala) supplies a private scratch kit protocol over real native `SystemStep` values. Its `KitWait` retains the actual class, cause and wait bound. A final scratch adapter first checks exact realization/Model identity and acquires `nativePerformances`; it then asks the kit for awaited classes. The adapter validates each class against the exact machine's bound ActionDecl and finite input catalog, then checks every selected step of every alternative against performed or awaited coverage.

Both a generic `Realization` and an ordinary custom `ConstructedRealization` admit one performed class and one waited class. A different input of the same action, an unsupported class, a foreign local action, a foreign same-primary-class Model, malformed input values and any unsupported alternative refuse. Existing selected-choice validation admits a retained token and refuses an undeclared token. Invalid machine acquisition refuses before projection. Opaque `SystemStep` and custom kits with no projection refuse explicitly, rather than certifying an empty wait set.

The native generic realization also carries actual script-`when`, evidence-confirmation and `HoldDispatched` dependencies. Calling only the pure `sourceActions` projection returns those declarations and the performed class while omitting the waited class. None of those unrelated dependencies earns wait coverage. All guard, effect, command, dependency, conversion and `end` poison counters remain zero during admission. Cause, wait bound, cleanup and the existing declarations remain unchanged. This control enumerates only the selected finite input catalog; it does not establish arbitrary custom input-catalog purity or run a server.

Ordinary `nativePerformances` itself also accepts an unbound ordinary performance in the counterexample. C must therefore check class membership as well as `realization.machine eq model`; selected-choice checks alone do not cover ordinary performances.

Prospective production owners are `realize/Constructed.scala` for final acquisition, the open native kit/SystemStep interface in `realize/Realize.scala` for an explicit awaited projection, and the declaring system kit for actual class/cause/bound semantics. The conformance adapter must validate every branch and sampled class without interpreting command or evidence dependencies as waits. The private protocol proves feasibility for the declared fixture kit; no product waited API exists after this task.

## Capability path counterexample

[CapabilityProbe.scala](../../tmp/fn1952-conformance-proof/CapabilityProbe.scala) declares a genuine generic `Reachable[S]` capability through an ordinary Model `Capabilities` section. Its `path` field contains the Model's actual action class, and its `accepted` field supplies a nontrivial executable companion predicate. The companion selects the final class and compares the native step outcome with that field. The replacement negates the predicate. An original binding, a named overridden binding and a waived neighboring law use the existing declaration syntax. Native Model validation succeeds for the fixture.

Direct companion invocation retains the exact owner, selected class and callable predicate; true and false outcome controls distinguish the original and replacement. This is ingredient evidence only. Current `Capabilities.claim` returns an unnamed empty `PropertyDecl` for the ordinary law, waived law, replacement and an unrelated object alike. It retains the Model reference but no selected class, predicate, native claim, progress claim or illustration. `except` and `overriding` retain no effective native directive records. The native `bound` call does not compile. A private exact-owner admission check rejects the same-primary-class foreign instance, while the product `claim` API itself does not perform that proposed selection check.

No generated native path with effective law selection and bound reaches replay. The experiment intentionally retains that missing-mechanism result. Manually invoking a companion, assembling a ReplayScript from its fields or retaining the empty Property would leave native generation and waiver/override/bound selection unproved. The separate replay eligibility control below supplies no capability-generation credit.

Production prerequisites are DSL-owned retention of actual binding and law identities, companion-field/path/claim expansion across source/library boundaries, effective waiver and named-instance override selection, and selected native bounds on the exact owner. `dsl/Capabilities.scala` owns declaration capture; generic capability companions or their kits own executable laws and path ingredients; one checker adapter owns replay. A realization/conformance carrier can consume an admitted generated path only after those mechanisms exist. Keep Model capability declarations intact and bind `RunExpectation` in the test.

## Proposed expectation contract

These are complete convenience-constructor proposals for later C, represented and checked with existing `RunExpectation`. They are authored expectations and imply no observed Run or server verdict. Field order is the existing nine-field order. `[]` means an empty monitor vector.

| Field | Path `satisfied` | Sample `conformant` | Property-only `inconclusive(reason, because)` |
| --- | --- | --- | --- |
| conformance | conformant | conformant | conformant |
| property | satisfied | inconclusive | inconclusive |
| contract | satisfied | satisfied | satisfied |
| disposition | completed | completed | completed |
| cleanup | succeeded | succeeded | succeeded |
| reason | None | Some(neverEvaluated) | Some(reason) |
| conformanceReason | None | None | None |
| monitors | [] | [] | [] |
| because | empty string | `A sampled path selects no individual Property` | supplied nonblank text |

The sample default does not invent a satisfied singular Property. Automatic Model claims remain separate checks, and any capability sample that selects a particular law must preserve that law's own claim metadata. The authored Contract still expects its selected concrete path obligations to pass. The explicit shorthand covers `explanationsDisagree`, `neverEvaluated` and `unreadable`, where the Property assessment alone is inconclusive while the Run can remain conformant and its Contract satisfied. The scratch helper refuses `noEvidence`, `incomplete`, `hole`, `unexplained` and `everyExplanationViolates`; those require an explicit full `RunExpectation` with independently chosen fields. This restriction is a proposed shorthand domain, not shipped behavior or a mapping from every Reason to the same outcome.

Current private constructors and restricted `copy` prevent satisfied explanations and outcome-changing copy bypasses. They still accept omitted, empty, whitespace-only and copied-to-empty explanations for non-satisfied outcomes. The experiment constructs each such value and requires a private conformance-boundary validator to reject it. Later C must validate both construction and retained/copied values at admission. Non-satisfied Property/Contract expectations require `because.trim.nonEmpty`; fully satisfied Property/Contract values retain the existing empty-string rule. Reason presence must agree with those authored assessments, and conformanceReason must agree independently with conformance. A full expectation preserves stopped/incomplete disposition and failed/timed-out cleanup as independent fields.

Monitors keep their own name/reference, outcome, reason and because. Their non-satisfied text must also be nonblank, their satisfied text empty and their reason consistent with their own outcome. The mixed-field control retains nonconformant conformance, inconclusive Property, violated Contract, stopped-by-monitor disposition, failed cleanup, different reasons and both satisfied/violated monitors. No field is derived from another. The default empty monitor vector asserts no monitor-specific expectation; automatic monitor obligations retain their existing owner.

## Minimal immutable declaration for later C

The proposed declaration retains the exact Model reference, exact admitted realization reference and the complete-inventory admission evidence; path or sample kind; the DSL-owned positioned authored body or generated capability descriptor; selected concrete native replay paths and starts; full authored expectation; and the original test/source location. A generated capability descriptor must retain its binding/law identity, exact owner, effective waiver/override and bound provenance. These are contract ingredients, not a product carrier implemented here. Public DSL/realization signatures must not expose checker implementation result classes.

Identity ingredients are the qualified stable Model-holder key, concrete suite/test declaration identity, canonical nonempty slug, source `RunPosition(file, line, text)` and, for sampling, explicit signed Long seed plus zero-based index in retained draw order. Reuse the current `Locale.ROOT` lowercase and `[^a-z0-9]+`-to-hyphen slug normalization, trim edge hyphens, and reject duplicate/empty slugs at their source. Planned path identities are `<instance>.<slug>`; planned sampled identities are `<instance>.<slug>.<seed>.<index>`. Absolute checkout paths, memory addresses, object hash codes, classloader identity, initialization order and timestamps supply no canonical identity. Exact references remain separate runtime authority. Source locations diagnose declarations without changing the planned identity when only line placement changes.

Admission first validates exact ownership, complete realization selection, all selected performed/awaited classes and expectation metadata. Native Model replay then establishes eligibility for every concrete selected path. Successful completed replay permits the producer-facing eligible view. Failure or unknown remains visible with the original declaration, expectation, diagnostic, source and any retained native trace, while supplying no eligible Case. An expected inconclusive Run never excuses an unknown or failed Model replay. Sampling preserves requested seed/count and every attempted index without renumbering failures away. Examples, reachability-find witnesses and unit bodies never enter the producer-facing view.

The executed replay control uses real `Checker.replay` and distinguishes a completed verified path, disabled step, thrown effect, located false assertion and budget-cut unknown. Only the completed verified result passes the private eligibility predicate. This validates an existing native seam, not the future immutable carrier or producer.

## Required production integration and sizing

C does not fit one admitted integration task yet. Complete construction/owner metadata and capability retention/expansion are separate missing mechanisms. The conductor must record actual prerequisites and keep fn-195.11 blocked until their source/library proofs and integrated implementations exist. The bounded wait-projection adaptation is a smaller concrete interface change, but still needs its kit owners and native callers. This report supplies no unconditional M estimate for that combined surface.

After those prerequisites and P/S/X integrate, repeat the complete inventory matrix at the real ModelSuite post-construction, pre-MUnit-snapshot boundary. Include inherited/external library owners, untouched lazy declarations, delayed custom kits, same-reference aliases, distinct copies, invalid candidates, getter failures/reentry, zero/one/many default selection, explicit foreign `through`, repeated enumeration and two fresh suites. Verify that invalidation cannot leave an earlier immutable declaration eligible and that released owners remain collectible.

Repeat performed/awaited coverage on P's actual captured alternatives, repeated steps and X's seeded sample paths, preserving exact input values and existing choice checks. Repeat capability generation from real retained declarations through effective waiver/override/bounds into native replay. Repeat all nine-field defaults and because/copy/monitor controls through the actual conformance entry point. Verify real suite identity, canonical slugs, starts, immutable budgets, nonzero executed tests, sticky failures and source/library discovery under S. Keep failed/unknown replay visible and the producer selection limited to conformance. Final source admission and later vocabulary retirement must preserve these contracts before deleting their old declarations.

Formal implementation review remains deferred by the owner's explicit phase-batch policy. The conductor owns Flow closure and prerequisite decisions. No source-level conformance support, server assessment, emitted Case bytes, Make-green or R4 implementation completion is claimed.

## Verification addendum

The final caller version is 11. Its source and packaged routes each exit 0 with 28 tests. Its source and packaged completeness oracles each exit 1 with exactly the same two intended semantic failures. The immutable library remains version 10. No packaging input changed between those caller versions, as verified by byte comparison and the native-source manifests.

| Final observation | Terminal receipt | Host handle, joined |
| --- | --- | --- |
| Source 28 controls | [combined-v11/source.log](../../tmp/fn1952-conformance-proof/combined-v11/source.log), status 0 | 39988 |
| Library 28 controls | [client-v11/client.log](../../tmp/fn1952-conformance-proof/client-v11/client.log), status 0 | 21091 |
| Source two completeness failures | [oracle-source-v11/oracle-source.log](../../tmp/fn1952-conformance-proof/oracle-source-v11/oracle-source.log), status 1 | 77715 |
| Library two completeness failures | [oracle-client-v11/oracle-client.log](../../tmp/fn1952-conformance-proof/oracle-client-v11/oracle-client.log), status 1 | 97154 |
| Package immutable library | [library-v10/library.log](../../tmp/fn1952-conformance-proof/library-v10/library.log), status 0 | 7559 |
| Verify statuses, counts and source/artifact bindings | [verification-v11.log](../../tmp/fn1952-conformance-proof/verification-v11.log), status 0 | Foreground, completed |

Exact final invocations from the repository root are below. Each runner invocation expands to the pinned Scala CLI command in its version's `.command` file. Expected oracle exit 1 is checked as such, rather than rounded into a passing suite.

```sh
env -u BASH_ENV bash .flow/tmp/fn1952-conformance-proof/run.sh combined-v11 source
env -u BASH_ENV bash .flow/tmp/fn1952-conformance-proof/run.sh library-v10 library
env -u BASH_ENV bash .flow/tmp/fn1952-conformance-proof/run.sh client-v11 client .flow/tmp/fn1952-conformance-proof/library-v10/proof-library.jar
env -u BASH_ENV bash .flow/tmp/fn1952-conformance-proof/run.sh oracle-source-v11 oracle-source
env -u BASH_ENV bash .flow/tmp/fn1952-conformance-proof/run.sh oracle-client-v11 oracle-client .flow/tmp/fn1952-conformance-proof/library-v10/proof-library.jar
env -u BASH_ENV bash .flow/tmp/fn1952-conformance-proof/verify.sh
```

The package command already ran and its output path exists; replaying this receipt list requires fresh version/workspace/output paths. The list records invocation identity, not a recommendation to overwrite frozen evidence. [processes.md](../../tmp/fn1952-conformance-proof/processes.md) records every joined compiler/test handle, superseded exit and the failed metadata-command attempt. The final process query observes zero proof-owned JVMs. The verifier reads statuses/counts/hashes and never reruns a suite merely to observe greenness.

| Frozen item | SHA-256 |
| --- | --- |
| Complete final command/log/manifest bindings | `e7aacf068ae979ed352c2ba9ab4b447d9d4bcc3e885b75d6f983074e161e7e40` |
| Packaged native/declaration library | `10b735be051ad15fcc9f5ef11fb502aad509f607a933e717999a8afd6953dae1` |
| Inventory declaration fixtures | `328ae1cbb404262b4640c50976b434186980ace85d8cddb9d1abd1aa2c724cb2` |
| Wait declaration/projection fixtures | `21dfbb729d38bd65a5324abf8aa65b8e3a4d2543aa45326a6c53ba2159eb9778` |
| Capability/expectation fixtures | `0b81ba3da136afd08debb8d5b09409881d2f76d5adb6c14a1640e9143106b1de` |
| Version-11 outside-library caller | `6acf920f88bcbc88740794ba9a7e5becadf11a885755694da344612ed5dfe172` |

The complete [final-bindings-v11.sha256](../../tmp/fn1952-conformance-proof/final-bindings-v11.sha256) joins exact command files, logs, status files, library bytes, tool observations, verification script and each version's source manifests. The [final source manifest](../../tmp/fn1952-conformance-proof/combined-v11/sources.sha256) includes both test callers; [native.sha256](../../tmp/fn1952-conformance-proof/combined-v11/native.sha256) covers all 94 native source files. Adapted native source snapshots live beside the caller snapshots. The managed JVM installation version check prints Zulu27.28+101-CA, build 27+35; semantic commands explicitly select JVM 27.

The pre-edit Quick baseline reused the exact green `unittest` receipt at `59bbc513`, with no baseline handoff and no invented scratch-proof receipt. Its command is `mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only checkercontrol.CheckerTest`.

```text
GATE_SKIPPED:unittest:green-receipt 59bbc513 - baseline reused from prior post-gate pass
GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched)
```

Final gate classification reports `TIER_B: docs-only (1 files)` in [gate-classify-v11.log](../../tmp/fn1952-conformance-proof/gate-classify-v11.log). The only staged path is this Markdown report. `git diff --check` passes; no repository Make/Go or held full gate is substituted for that classification.

The conductor retains normal Flow closure and records actual prerequisite tasks before admission of C. The proof's green aggregate means all expected positive and negative observations matched. Realization completeness and native capability generation remain failed requirements for C.
