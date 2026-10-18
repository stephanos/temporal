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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
