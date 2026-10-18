---
satisfies: [R1, R7]
---
# fn-195-modelsuite-a-models-test-file-holds-its.13 Implement bounded inert compiled suite inventory

## Description
Implement the in-process metadata leaf proved by `.12`, advancing R1 and local R7. Consume `model/design/modelsuite-discovery.md` at source checkpoint `41454f742a`. Keep compiler orchestration, Registry migration, policy changes and complete gate admission in the separate blocked integration prerequisite. This task supplies reusable typed records to `.3`; it implements no final section rules.

**Size:** M, three files and five acceptance groups. If the source-less provenance or independent hierarchy closure requires another subsystem, preserve the concrete failing experiment and report the dependency before widening this task. Do not replace a supported library case with a new author restriction.

**Files:** `model/framework/inspect/SuiteInventory.scala` (new), `model/framework/inspect/SuiteInventory.test.scala` (new), `model/framework/inspect/project.scala` (existing pinned MUnit 1.2.0 test dependency only).

**Touches:** [model/framework/inspect/SuiteInventory.scala, model/framework/inspect/SuiteInventory.test.scala, model/framework/inspect/project.scala]

### Approach

- Expose one framework-internal in-process inspection entry point over immutable request/result carriers. Keep it independent of Registry, Tools, policy and checker implementation. It launches no compiler, process, suite or author initializer. Existing JDK 27 classfile and Scala 3.9 TASTy APIs are sufficient; no new dependency or version is admitted.
- The request independently names module identity, candidate compiled outputs, explicit hierarchy artifacts, source/hash manifest, skipped roots, compiler flags and toolchain identity. Validate actual bytes against that request. Derive no completeness universe from the returned inventory itself. The caller owns fresh compilation, publication and input rechecks; the leaf validates the supplied immutable generation and returns a bound result.
- Read exact JVM class names, access/abstract/interface flags, public no-argument constructors and superclass/interface edges from classfile bytes. Enumerate candidates only from candidate outputs. Hierarchy artifacts resolve ancestry without contributing candidate suites. Preserve plain FunSuite and ModelSuite inheritance and the exact binary selectors accepted by the runner.
- Index physical artifact ownership before resolving references. Repeated references to the same canonical artifact may deduplicate. Distinct candidate or hierarchy owners for the same required binary identity refuse, including candidate/dependency and dependency/dependency conflicts. Do not let classpath order choose an ancestor silently. Bound runtime/framework/MUnit stopping points by their explicit trusted artifacts, not a namespace-prefix shortcut.
- Traverse the existing typed-tree machinery and join compiler symbol owner chains to actual classfile identities. Reuse `Index.trees` and source provenance where appropriate; `Index.defs` indexes only values/methods and is not a class index. Keep the leaf self-contained unless a concrete need justifies a separate scope approval.
- Derive the required author ancestor graph independently from classfile superclass and interface edges, then require typed declaration membership and edge agreement. Retain the minimal `.12` RED that drops only `Helpers.Base` while the leaf row remains. A nonempty ancestor list or a superclass-only walk does not establish closure.
- Give current candidate sources canonical filesystem identity plus verified bytes. Give a library declaration artifact-local identity consisting of its verified artifact, recorded source path, compiler symbol and retained span/line. The `.12` source-absent probe proves retained ancestor trees and positions without source-file access. Do not call filesystem `toRealPath` or sourceLine on an unavailable library source, infer artifact origin from its package, or trust caller-generated text ownership rows without checking the compiled artifact.
- Preserve no-test suites and exact source/ancestor definitions for `.3`'s later typed admission. Apply candidate skips after ancestry is complete; a skipped source may still supply a valid child's ancestor. Return explicit exclusions and stable diagnostics for unavailable or ambiguous required metadata rather than dropping a runnable suite or falling back to the source parser.

### Investigation targets

Required:
- `model/design/modelsuite-discovery.md` for the proved mechanism, independent closure contract and exact evidence limits.
- `model/framework/inspect/Index.scala:78` and `model/framework/inspect/project.scala` for typed trees, provenance and established dependencies.
- `model/framework/dsl/Sections.scala:49` for the established JDK classfile inspection pattern and reflective bridge.
- `.flow/tmp/fn19512/partial-ancestor.sh` and `partial-ancestor-accepted.log` for the retained incomplete-chain RED.
- `.flow/tmp/fn19512/source-absent.sh` and `AbsentInspect.scala` for the source-less library probe and its artifact-origin shortcut that production must replace.
- `.flow/tmp/fn19512/independent-audit.md` for constructor, count, provenance and policy evidence boundaries.

Optional:
- `.flow/tasks/fn-195-modelsuite-a-models-test-file-holds-its.3.md` for the final source-admission consumer. Do not implement its lint rules here.

### Verification

Start with a failing production regression for the single omitted ancestor, then exercise superclass and interface omissions independently. Use fresh source and packaged-library fixtures with exact expected identities and artifact owners. Repeat `.12`'s stable-term explicit/inferred bases, direct/transitive FunSuite and ModelSuite, encoded nested names, duplicate basenames, public-JVM Scala-private classes, private/argument/outer constructors, abstract/trait/object exclusions and no-test suites. Include a child inheriting from a skipped source and a helper jar whose original source is absent; simulate unreadable library source without requiring access to it.

Verify corrupted/missing/incompatible/partial/stale artifacts, swapped records/generations, duplicate candidate and hierarchy definitions, conflicting source ownership and partial ancestor graphs refuse at the responsible artifact/class. Constructor, object, Model and test sentinels remain untouched during inspection and fire in a separately executed control. Declare/load without the checker and retain unchanged strict flags. Tests may compile disposable fixtures in one coordinated task-owned lane; the production leaf never launches subprocesses.

Run the parent canonical CheckerTest Quick, the new focused inspect suite and applicable admitted native checks. Report actual commands, terminal exits, exact selected names, counts, artifact bindings and intended failures. Full gate/caller or held-universe completion receives no credit from leaf tests. Keep the internal records private to the framework; do not add a user-facing author API or Scala-file navigation to the reference.

### Quick commands

```bash
mise exec -- scala-cli test --server=false model/framework/inspect --require-tests --test-only framework.inspect.SuiteInventoryTest
mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only checkercontrol.CheckerTest
git diff --check
```

### Boundaries

No Registry/Tools/gate integration, policy format/order change, general cache/publisher service, final `.3` lint, realization inventory or public conformance API. No actual repository Make, Go, IR/export/lifting/schema/backend/generated/regeneration or Temporal work. The entire `model/temporal` tree stays held from reads/search/build. Do not compile the held export module to obtain fixtures or metadata. Complete held/unselected inventory and actual caller admission remain blocked in the integration prerequisite. `.8` stays incomplete after this leaf lands.

## Acceptance
- [ ] One internal in-process leaf validates independently supplied module/source/output/hierarchy/toolchain identities and returns an immutable input-bound inventory. Missing, changed, incompatible, malformed, partial and mixed-generation artifacts refuse with artifact/class context. The leaf owns no subprocess, compiler, policy, cache or transactional publisher.
- [ ] Exact classfile JVM names and runnable flags join to verified typed source/ancestor declarations. The retained single-omitted-ancestor RED, missing interface edge and mismatched typed/classfile closure refuse even with nonempty records. Encoded nested and stable-term inheritance, duplicate basenames, no-test suites and runner-observed constructor/private distinctions have fresh source/library controls.
- [ ] Candidate outputs and hierarchy dependencies remain distinct. Distinct required binary owners across candidate/candidate, candidate/dependency and dependency/dependency artifacts refuse independently of input ordering; repeated references to the same artifact retain deterministic semantics. Skipped leaves stay excluded while skipped declarations can supply ancestry. Source-less/unreadable helper libraries use verified artifact-local provenance and retained typed trees/positions without requiring original sources or trusting package prefixes/text records.
- [ ] Compile/load/inspection works without the checker and without initializing author suites, Models, helper objects or test bodies. Instrumented sentinels stay absent until an independently executed positive control. The exact expected identities and nonzero independent runner counts match fresh artifact generations under unchanged strict flags.
- [ ] The leaf preserves complete no-test/source/ancestor records for `.3` without implementing final section rules. Focused inspect and parent Quick checks, scope/boundary checks and artifact hashes support the checkpoint. The receipt explicitly leaves actual Registry discovery, both staged gate callers, complete held-suite identities and fn-196.2's contract amendment to the blocked integration prerequisite; it does not close `.8` or all R7.


## Done summary
The framework can now inspect compiled runnable suite identities and their complete author declarations without initializing suites, Models, helper objects or tests. The private `SuiteInventory` leaf binds its result to canonical artifact/source hashes, the explicit candidate and hierarchy roles, module identity, skips, compiler flags and toolchain; it preserves retained typed trees for fn-195.3.

Baseline: green. The existing inspect compile and `git diff --check` exited 0 before edits. The unchanged base reused the canonical parent CheckerTest receipt. The new SuiteInventoryTest selector did not exist at baseline and was not reported as a behavioral failure.

GATE_SKIPPED:unittest:green-receipt 3f34ed08 - baseline reused from prior post-gate pass

Tier: session (jev-unavailable(no_key))

stage: impl-review - skipped(config: REVIEW_MODE=none; policy: owner phase-batch formal-review deferral)

stage: plan-sync - skipped(config: planSync.enabled != true)

Tracker sync: n/a (bridge inactive)

The final source checkpoint is `d472614eb16ed98b52177daffe621fdd4bfc0ca7`, based on `3f34ed085b3d36ad4a9b9679170898724e249c8b`. Implementation touches only SuiteInventory.scala, SuiteInventory.test.scala and the existing inspect/project.scala's pinned MUnit 1.2.0 test dependency. Production launches no process or compiler, imports no checker, Registry or Tools, and owns no policy, cache or publication protocol.

Independent classfile ownership rejects required candidate/candidate, candidate/dependency and dependency/dependency collisions. Canonical repeats deduplicate. Actual superclass and interface edges establish the author closure before typed declaration membership, direct edge agreement and source skips. Scala trait superclass constraints are checked against the implementing classfile closure, and the compiler's Serializable edge on objects is accounted for. Library provenance uses its verified physical artifact, recorded source path, compiler symbol and retained position. It performs no library-source toRealPath or sourceLine lookup.

The retained omitted-Base regression first failed in `first-red-v2.log`, exit 1, and was committed in `1929dadd6c` before implementation. Missing-interface metadata, absent interface classfiles and an omitted classfile interface edge have separate controls. The later `toolchain-red.log`, exit 1, proves matching classfile/TASTy UUIDs previously accepted a changed tooling-version string. `8d55a60188` pins that failure; `c70d2c2372` requires the TASTy compiler identity to agree with the request. The controls refuse corrupt/missing/stale/incompatible inputs, swapped root TASTy generations and conflicting/outside source ownership. The final corrupt-edge negative intentionally retains the compiler's diagnostic before the leaf returns its contextual refusal.

The root's task-scoped correctness and standards axes at c70 reported no Critical or Should Fix findings and no standards findings. Their one Consider item was the corrupt-ZIP diagnostic's missing artifact path. The strengthened existing negative table retains its genuine RED in `artifact-context-red.log`, exit 1, and commit `469ed2af26`. The four-line production correction in `d472614eb1` adds path context around both initial artifact decoding and the final artifact recheck. `artifact-context-green.log` records the same 12 tests at exit 0. The corrected fixture's refusals.log:17 now includes its exact corrupt.jar path and the underlying ZIP reason. The reports remain at `quality-reports.md`; these task-scoped axes supply no formal review verdict.

Root independently re-ran the canonical focused inspect command at the corrected clean checkpoint. Its joined exit 0 and 12 passing tests are retained in `root-inspect-v2.log`, outside the worker's frozen manifest. Root also honored the corrected parent receipt, checked all 74 new proof rows and the original manifest digest, inspected the corrective delta and released closure. `root-final-delta.md` records its acceptance check and the resolved advisory. Formal batch review remains deferred.

Corrected-generation verification ran the exact focused inspect Quick command with 12 passing tests, exit 0, in `artifact-context-green.log`, and the canonical parent CheckerTest Quick with four passing tests, exit 0, in `verify-checker-v2.log`. The full-gate classifier returned FULL. Both actual test commands received green receipts at the checkpoint. `git diff --check` passed, and repeating the configured formatter left all three source hashes unchanged in `formatted-v2.sha256`. A premature parent receipt was deleted before any reuse and regenerated only after joining its terminal exit 0; `receipt-correction-v2.md` records that operational correction.

Corrected-generation fixture evidence is under `.flow/tmp/fn19513/fixtures-7020610902603597282/`. Each subprocess has an exact argument-per-line `.command`, terminal `.exit` and raw `.log`. The hand-listed matrix has 11 eligible selectors, including three empty suites. Its fresh source and packaged-library JUnit runs each execute `OK (10 tests)`; those are separate runs. The checker-absent plain control independently executes `OK (4 tests)`. The later control initializes the Model and helper object; the constructor, Model, object, helper and test sentinel kinds are all observed after the pre-execution absence assertion. Fresh declarations and author libraries compile with the complete unchanged strict flags and without the checker; a fresh checker build is introduced only for the later full runtime controls.

The source-less helper control moves only its disposable fixture source, inspects the retained ancestor tree and positions, and restores the original bytes. Its separate permission-denied control sets permissions only on that fixture, verifies it is unreadable, succeeds without source availability, and restores permissions and bytes. Candidate outputs stay distinct from hierarchy libraries. A skipped helper leaf stays excluded while its source supplies the admitted child's ancestry. Directory outputs have independently computed hashes and reject symbolic entries outside their physical ownership.

The final immutable 74-row `proof-artifacts-v2.sha256` manifest checks successfully and has SHA-256 `a9cdfc74e6be357fb54069e86b636679ec2c542597e1d6e0e6b5804067da698d`. It includes the corrected source checkpoint, existing IR dependency, corrected fixture inputs/artifacts/commands/logs and versioned verification logs. It excludes handovers, conductor planning/audit files and compiler caches. The original 72-row `proof-artifacts.sha256` remains byte-for-byte unchanged at SHA-256 `a74b53ba99aaf953da1dbbf9ec6c4176461090c008b62d3858d09d18717ecfa5`. Its two intentionally changed source rows bind c70 and were checked against those git blobs; its other 70 rows still check. The original fixture generation `fixtures-12473930701831816053` and all original logs remain intact. The previous 14 frozen product source hashes and all 129 original fn-195.12 proof rows also still check successfully.

Canonical source/artifact identity and byte checks bind the supplied immutable generation. The caller must supply outputs from one fresh compilation and owns publication and input rechecks. Root class/TASTy UUIDs reject the exercised root metadata swap, but Helpers$Base.class has no own TASTY UUID. The leaf cannot attest an arbitrary same-hierarchy nested classfile replacement when its new hash is independently authorized by the caller. Compiler flags are bound request data rather than a claim that classfiles encode every compiler option.

This completes the bounded metadata producer's R1/local R7 contribution. It preserves no-test and inherited source records for fn-195.3 and implements none of its final section rules. Actual Registry discovery, both staged gate callers, complete held-suite identities and fn-196.2's ordering-contract amendment remain in the blocked integration prerequisite. fn-195.8 and fn-195.14 remain incomplete. No actual repository Make, Go, held export compilation, lifting, schema/IR regeneration, Temporal read/search/build, backend, Testpilot/server or Case production was performed.

Defect route:
- prior fixes: consumed fn-195.12's retained incomplete-ancestor RED and the existing Index/classfile patterns; no production SuiteInventory existed at base. External PR/issue lookup was not done because this admitted task creates the new internal leaf; tracker bridge inactive.
- diagnosis: a surviving leaf row and nonempty typed ancestry cannot establish closure. Required membership and edges now come independently from classfile bytes and uniquely indexed owners. The tooling-version RED exposed a separate request/artifact identity gap and now refuses.
- introduced by: skipped because the new production leaf has no known-good prior revision; the fn-195.12 prototype's gap was retained without modifying its frozen evidence.
- base: the omitted-Base production regression fails in first-red-v2.log; compiler provenance fails in toolchain-red.log; missing ZIP path context fails in artifact-context-red.log. Head passes all three within the corrected 12-test run and retains independent complete-artifact positives.
- live: no interactive surface. Fresh JUnit source/library executions provide the bounded runtime controls.
## Evidence
- Commits: 1929dadd6cd327d5c76d8cc468e85287c8e34071, 722b77c1c726ebe7a2b37fa605a0f0d61fc9e7b1, 8d55a60188df93b77427e6eb08f6991b944d0dcd, c70d2c237233ff24f524f17a82d39d8080a4ce16, 469ed2af26b796340b3c073dbc5061a22cdf2d94, d472614eb16ed98b52177daffe621fdd4bfc0ca7
- Tests: mise exec -- scala-cli test --server=false model/framework/inspect --require-tests --test-only framework.inspect.SuiteInventoryTest, mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only checkercontrol.CheckerTest, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf model/framework/inspect/SuiteInventory.scala model/framework/inspect/SuiteInventory.test.scala model/framework/inspect/project.scala, git diff --check, sha256sum -c .flow/tmp/fn19513/proof-artifacts-v2.sha256, sha256sum -c .flow/tmp/fn1958/source-freeze-v3.sha256, sha256sum -c .flow/tmp/fn19512/proof-artifacts.sha256, GATE_SKIPPED:unittest:green-receipt 3f34ed08 - baseline reused from prior post-gate pass
- PRs: