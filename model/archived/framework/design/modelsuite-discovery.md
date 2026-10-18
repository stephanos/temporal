# Compiled ModelSuite discovery proof

The native author can keep ordinary stable-term Scala inheritance. Fresh compiler artifacts find
the two runnable suites that current source discovery omits. This establishes a bounded generic
metadata mechanism. Production replacement remains **NO-GO under the current admission** because
the complete held export inventory and actual staged gate callers remain unproved. fn-195.8 stays
incomplete. This report changes no production discovery or fn-196.2 policy contract.

## Decision and scope

Base commit is `125ca601644fc728b268c4d04dc152185a592a6f`. Product sources remain at the preceding
`ff0dcaf55d` checkpoint. The original RED is retained verbatim under
`.flow/tmp/fn1958/metadata-probe/`. Its JUnit run executes four tests, while production Registry
returns zero suites for the same two public no-argument leaves. No source parser or author
restriction was added to repair it.

The proposed producer reads explicitly named compiler outputs and hierarchy artifacts. It combines
JDK 27 classfile facts with Scala 3.9 TASTy source ownership. The checker is absent from compilation
and inspection classpaths. Actual execution is a separate stage with the frozen checker jar.

All new experiments are under `.flow/tmp/fn19512/`. Their scope excludes `model/temporal` reads,
searches and execution, held export compilation, actual repository Make, Go, lifting, IR/schema
generation, backend work, Testpilot/server execution and Case production. The existing IR jar is
an unchanged library dependency. No public API, dependency version or warning flag changed.

## Executed inventory and independent execution

`initial.sh`, `inspect-original.sh`, `matrix.sh`, `controls.sh`, `join.sh`,
`final-controls.sh` and `final-negatives.sh` retain their exact commands and logs. The `*.commands`
and `*.exits` files preserve terminal command observations. Commands use `--server=false`, Scala
3.9.0, JDK 27, MUnit 1.2.0 and the established complete strict flag list, including `-Werror`,
`-Wsafe-init`, `-Yretain-trees` and strict equality. The pre-edit parent Quick command passed its
four `checkercontrol.CheckerTest` tests. `git diff --check` also passed before edits.

The independently written expected inventories are `expected-original.txt`, `expected-matrix.txt`
and `expected-external.txt`. `final-controls.sh` compares each sorted joined result to its expected
list. Each comparison exits zero. The names below are actual JVM selectors from classfile bytes.

| Batch / explicit module | Eligible selectors | Independently executed result |
| --- | --- | --- |
| Original retained author | `pathprobe.ExplicitLeaf`, `pathprobe.InferredLeaf` | Fresh source `OK (4 tests)`; separately packaged author jar `OK (4 tests)` |
| Matrix | `duplicate.a.First`, `duplicate.b.Second`, `inventory.Direct`, `inventory.Empty`, `inventory.Indirect`, `inventory.ModelChild`, `inventory.ModelDirect`, `inventory.Nested$a$plusb`, `inventory.Private` | Eight selected selectors in `matrix.sh` execute `OK (9 tests)` in each fresh source and packaged-library route. Separate `private-junit.log` executes `inventory.Private`, `OK (1 test)` |
| External library client | `libraryclient.External` | Freshly packaged client consumes the freshly packaged original author library and executes `OK (2 tests)` |

The matrix has nine eligible classes. Nine executed tests in the eight-selector matrix run are
not ten tests. The one-test private-class receipt is separate. `inventory.Empty` remains in the
inventory despite having no authored tests. Aggregate JUnit counts are nonzero, but no claim is
made that an empty suite executes a test of its own.

`Catalog.java` also reports abstract helpers, a trait's implementing superclass, argument-only
constructors, a private constructor, an instance-inner class requiring an outer parameter, and a
singleton object. Its public/concrete/public-`()V` predicate excludes the ineligible classes.
Scala's top-level `private class Private` emits a public JVM class and constructor here; its actual
one-test JUnit execution succeeds. Source spelling alone cannot decide runtime eligibility.

The direct/indirect FunSuite, typed alias through a helper trait, direct/inherited ModelSuite,
explicit/inferred stable-term path, no-test, nested encoded-name and duplicate-basename controls
all have explicit expected names. The external client sees the original author jar only as a
hierarchy dependency. Its candidate inventory contains only `libraryclient.External`; the library's
two runnable leaves do not leak into the client module inventory.

## Actual JVM/source/ancestor join

`Typed.scala` traverses compiled `ClassDef` symbols and reads canonical source paths and declaration
spans from TASTy. It derives a candidate JVM key from the compiler's package/class owner chain and
`NameTransformer.encode`, then `Ownership.scala` joins that key against the authoritative
classfile selector. Missing keys or conflicting source owners refuse. This is a bounded exercised
join, not a claim that a source-symbol full name is always a JVM selector.

`join-original.log`, `join-matrix.log` and `join-external.log` are the final joined generation.
Every admitted row has a source owner in that batch's explicitly supplied canonical source list.
Each author ancestor up to ModelSuite/FunSuite has its own source symbol, canonical file and exact
span, emitted as `JOINED-ANCESTOR`. The caller assigns the explicit batch/module identity; it does
not infer module ownership from package names. Multiple modules claiming the same binary name
must refuse.

Concrete joined rows include:

- `inventory.Nested$a$plusb` joins the compiled `inventory.Nested$.a+b` symbol to
  `Matrix.test.scala`, then to its `Parent` ancestor. The join uses compiler symbols and bytecode
  keys, without parsing the authored nested name.
- `duplicate.a.First` and `duplicate.b.Second` join to different canonical `a/Repeated.test.scala`
  and `b/Repeated.test.scala` paths. Equal `SourceFile` basenames supply no ownership authority.
- `libraryclient.External` joins its own declaration at span `22..77` and
  `pathprobe.Helpers.Base` in the original retained author at span `317..437`. That superclass
  metadata comes from the separate original author library, without constructing `Bindings`.

`join-skipped.log` applies a declared skipped source after metadata and ancestor admission. It
retains eight admitted matrix names and one explicit skipped name, `duplicate.a.First`.
`expected-skipped-diff.log` is empty and exits zero. The final producer must also preserve the
current Registry rule that skipped source may supply ancestry. This experiment proves post-join
candidate exclusion; it does not independently exercise a child inheriting from that skipped file.

The join has not proved every legal Scala local/anonymous/opaque library encoding. The production
leaf must reject unavailable or ambiguous joins and add focused controls for any newly admitted
encoding. It must never fall back to a source parser or guess a selector.

The final all-ancestor-records-missing negative does not establish completeness of a nonempty chain.
The later `partial-ancestor.sh` removes only the `pathprobe.Helpers.Base` ancestor row from the
external client's metadata. The leaf's own ancestor row remains. Current `Ownership.scala`
incorrectly accepts that input with raw exit zero in `partial-ancestor-accepted.log`. The expected
refusal control exits one and records **RED** in `partial-ancestor.exit`. This is a prototype
completeness hole, preserved for the production worker's first regression. It is not a passing
metadata refusal or a production fix. The independent classfile still names `Helpers$Base` as the
superclass, so it provides the missing completeness authority.
The first two packaged-library experiments keep original source files on disk, and `Typed.scala`
calls `toRealPath`. The later source-absent probe below proves a separate artifact-local route.

### Helper library without its original source

`source-absent.sh` compiles a fresh task-owned helper library and a stable-term child in a fresh
client workspace. It then moves only its new `source-absent/Library.scala` to a recoverable saved
name. The original source path is absent throughout inspection. It never moves the retained `.8`
author or a frozen product source. The script restores its fixture with an exit trap and verifies
its original source/jar hashes afterward.

`AbsentInspect.scala` reads `absentlib.Helpers.Base` from the helper jar's TASTy without filesystem
`toRealPath`. `source-absent/absent-inspection.log` records the missing original path, span
`221..374`, embedded zero-based lines `5..9`, and the retained ClassDef tree. That tree includes
the nested `unit.retainedInline` declaration and its body `17`. The exact independent classfile
selector is `absentclient.Leaf`, public/concrete with public `()V`, and its binary superclass is
`absentlib/Helpers$Base`. `absent-classfile.log` and the inspector both exit zero. The helper
constructor sentinel stays absent until separate JUnit execution, which runs `OK (1 test)`.

This proves artifact-local typed ancestor acquisition with the library's source absent. The
location tuple is `(library artifact hash, normalized recorded source path, compiler symbol,
span/embedded line)`. The fixture's explicit classpath and per-artifact hash manifest bind that
location to its helper jar. Production ownership must come from the independently indexed artifact,
not the namespace-prefix shortcut used to label the two known fixture origins in this probe.
This adds no public requirement to distribute original source files. Reusing Index's legacy
filesystem-backed `sourceLine` would reintroduce that restriction; suite admission must use the
retained positions demonstrated here. Permission-denied source access was not separately tested.

## Inertness and metadata refusals

Matrix constructors, the Model constructor, nested object initializer and test bodies append to
`INVENTORY_SENTINEL`. Compilation and both metadata inspections finish with the marker absent,
recorded as `compile_inspect_sentinel=absent` in `matrix.exits`. Separate execution then appends
`direct`, `parent`, `model` and `test`. `ObjectInit.java` explicitly initializes the nested object
only in a later execution stage; `object-sentinel` contains `object` and `object-init.log` exits
zero. Thus the object marker is capable of firing. Original retained authors were not edited to
add instrumentation; their absence of instrumentation supplies no sentinel evidence.

`Catalog.java` parses bytes and follows superclass names within named artifacts. It never loads an
author `Class`, calls a constructor or accesses an object getter. The inspector reads TASTy. Its
classpath excludes `checker.jar`, so declaration inspection does not require checker loading.
Candidate classes come from the named output directories only. Hierarchy resolution uses explicit
dependency artifacts, rather than discovering candidates on the application classpath.

| Control | Terminal observation | Bound |
| --- | --- | --- |
| Duplicate candidate binary ownership | `duplicate-binary.log`, exit 1, `duplicate binary identity` | Same candidate output claimed twice refuses |
| Missing hierarchy artifact | `missing-hierarchy.log`, exit 1, `missing hierarchy metadata` | Cannot resolve ModelSuite from original author alone |
| Missing candidate output | `missing-candidates.log`, exit 1 | Missing output refuses |
| Partial TASTy/hierarchy | `partial-tasty.log`, exit 1 | Leaf TASTy without its required artifacts refuses |
| Incompatible classfile | `incompatible-classfile.log`, exit 1 | Malformed bytes refuse |
| Wrong declared source universe | `final-ownership-outside.log`, exit 1, `source outside explicit input universe` | Final join generation |
| Partial source ownership | `final-ownership-partial.log`, exit 1, `missing source ownership` | Final join generation |
| Conflicting source ownership | `final-ownership-conflict.log`, exit 1, `conflicting source ownership` | Final join generation |
| Missing ancestor declarations | `final-ancestor-missing.log`, exit 1, `missing exact ancestor declarations` | Final join generation |
| One omitted ancestor, leaf row retained | `partial-ancestor-accepted.log`, raw exit 0; `partial-ancestor.sh`, exit 1 expected-RED | Prototype incorrectly accepts incomplete chain; production regression required |
| Fresh/stale/missing/partial bytes | `freshness.log`, exit 0 for four assertions | Exact bytes pass; changed source, absent jar and truncated jar refuse |

Earlier `ownership-*.log` controls belong to the pre-ancestor generation. They do not certify the
final source. The `final-ownership-*` controls repeat the focused refusals against final frozen
join bytes. Failed first probes remain raw. `original-classfile.log` rejected Scala CLI's absent
empty `classes/main` output. `original-classfile-v2.log` handles that exact compiler-output case.
`matrix-compile.log` retains the initial strict failure caused by a top-level alias in a `.test`
filename and unsafe fixture initialization. `matrix-compile-v2.log` moves the test alias into an
object and makes the argument-only control independent of the registering parent. Strict flags
remain unchanged. Neither failure is presented as an author-syntax restriction.

The minimal freshness contract binds canonical source bytes, explicit source/module inputs and
skipped roots, candidate artifacts, ordered hierarchy artifact identities, compiler flags and the
toolchain. The caller creates fresh private outputs, publishes only after successful compilation,
complete inspection and a second unchanged-input check, and rejects partial/crashed generations.
Repeating a run produces a new generation. Missing outputs cannot count as an empty inventory.
This proof tests hash comparisons and local refusal primitives. It does not implement or prove a
transactional production publisher, dependency-wide duplicate ownership reconciliation or concurrent
writer protocol. Those guards remain acceptance obligations of the producer/caller implementation.

The independent production closure contract is concrete. First index every explicitly supplied
candidate and hierarchy artifact by binary class identity and physical artifact owner, refusing
distinct owners before resolving any reference. Repeated references to the same canonical artifact
may deduplicate; first-classpath-match selection cannot resolve conflicting ownership. Derive the
expected author ancestor closure from classfile superclass and interface edges, bounded by the
explicit trusted framework/MUnit/runtime artifacts. Join every required author binary identity to
exactly one typed declaration and require set/edge agreement between that independent graph and
the typed ancestor graph. A single missing helper, interface or owner is a refusal even when other
rows remain. The scratch Catalog follows superclasses only and the scratch Ownership checks
nonemptiness. Their successful fixture rows do not implement this complete closure contract.

## Proposed staged policy sequence

The admitted direction is to parse and normalize `BudgetPolicy` before any child command, compile
only admitted inert metadata, validate exact overrides against the complete joined inventory, and
only then perform source/scope admission and suite execution. Parsing already rejects explicit
empty policy, malformed key/value shape, duplicate keys and nonpositive/non-Int budgets.

`PolicyDirect.scala` executes this proposed direct sequence with real counted subprocess starts.
`policy-direct-v2.log` exits zero. Nine malformed/empty/duplicate/range inputs have actual child
delta zero. An unknown override reaches one classfile-inspection child over the matrix output,
whose keys must equal the joined source-owned inventory. Validation refuses before the next child.
The constructor/Model/object/test marker remains absent at the independently named
`policy-direct-author-sentinel`. A known override plus an unselected known override retain the exact
sorted normalized transport through a real environment subprocess.

This is a direct sequence proof. It runs neither production FrameworkLoop nor ChangeCheck. The
older `Policy.scala` repeats a sequence under caller labels and prints constant execution counts;
`policy.log` provides no evidence for actual caller ordering or measured setup/claim/test counters.
Its deliberate missing `GuideSourcesTest` key shows only that an incomplete supplied universe
rejects a valid name. It proves no impossibility of obtaining held metadata. The stronger current
fn-196.2 before-compiler contract remains unchanged.

## Held universe and the unresolved admission

Current FrameworkLoop validates budgets against all declared suites before its held filter
(`FrameworkLoop.scala:178-184`). Its exact held check identity
`framework.check.lint.GuideSourcesTest` can be compiled inertly in the already admitted check build;
its held status prohibits executing its source-reading body. The small fixture inventory does not
represent that complete real check-module inventory.

The remaining unproved universe is the complete export module, including its held outer suites.
`FrameworkSuites.genericExport` and `integration` classify discovered suites; their simple-name
lists do not establish complete JVM identities or current source ownership. The proposed command
`scala-cli compile --test --server=false model/framework/export --print-class-path
--compilation-output <fresh-output>` would compile export implementation and held outer suites.
Its `project.scala` includes framework lint/inspect/reify-body/DSL/checker/realize, check helpers and
the existing IR jar. This command was **not run** because the conductor retained the explicit
export-compilation hold.

`DeadlinePresets.test.scala:19-50` embeds source importing Temporal and workflow API types. Its
fixture builder compiles that source, and suite construction accesses `ModelBuild.jar`. These facts
explain why initialization and fixture execution are held. Those imports are inside a string;
they do not prove that compiling the outer suite declaration requires Temporal sources. A complete
inert outer-class compilation might avoid that runtime closure. This task has no admitted fresh
producer or independently bound complete artifact proving it. No blanket compiler-impossibility
claim follows from the hold.

Therefore exact known-held export overrides and complete-universe staged validation remain
**unproved**. A default-only or generic-only policy run cannot settle them. The missing prerequisite
is an admitted, explicit outer-declaration metadata closure, or an equally current independently
bound compiler artifact for every held identity. Its proof must avoid Temporal reads, held fixture
compilation and initialization, preserve all unselected identities, and refuse newly unclassified
suites. Current source discovery and the hand-maintained hold map are not a complete typed fallback.

## Internal interface and production boundaries

Keep the new leaf in `framework.inspect`, with no dependency on Registry or Tools. Its internal
input names the module identity, exact canonical source/hash manifest, candidate outputs, skipped
roots, hierarchy artifacts/hashes, compiler flags and toolchain. Its result retains actual binary
name, source symbol, canonical source/span, constructor/visibility/abstract facts, exact author
ancestor declarations and the input fingerprint. Live module sources use canonical filesystem
identity; library declarations use artifact-local recorded source identity and retained positions.
Libraries resolve ancestry without becoming
candidates. The leaf launches no compiler and initializes no author class.

fn-195.3 consumes the same complete typed source/ancestor records before test-name filtering,
including no-test suites, so erased-inline and section admission can locate an inherited offending
member. Discovery alone implements none of those final lint rules. fn-195.10 independently proves
exact-instance realization candidates, waits and capability completeness; this class inventory
supplies none of that runtime authority.

The smallest producer-only implementation candidate is M with three prospective files:
`model/framework/inspect/SuiteInventory.scala`, `SuiteInventory.test.scala` and the existing
`inspect/project.scala` for MUnit's already used test dependency. Its five acceptance groups are
explicit input/freshness refusals, exact classfile-to-typed source/ancestor joins and eligibility,
candidate/hierarchy/skipped-root separation, inert checker-absent behavior, and the no-test typed
handoff. Production tests must cover the unproved publication and dependency-ownership guards;
the scratch programs are not production-ready implementations.

The producer's regression acceptance explicitly covers a single omitted ancestor and an otherwise
valid separately compiled helper jar whose original source files are unavailable. The bounded
source-absent mechanism is now proved; permission-denied and conflicting library provenance still
need production controls. A failed library join must produce an explicit refusal, never a new
author-source-availability restriction.

Keep full gate integration a conditional blocked shell. The concrete surface includes
`Registry.scala`, `Tools.scala`, `FrameworkLoop.scala`, `ChangeCheck.scala`, their discovery,
BudgetGate, FrameworkLoop and ChangeCheck tests, and affected native-check documentation. Its
acceptance retains complete held/unselected identities and both consumers. FrameworkLoop needs
explicit parse/metadata/validate stages before selection. ChangeCheck needs fresh inventory and
policy validation **before `Scope.of`** (`ChangeCheck.scala:531-535`) and before the empty-scope
success path, rather than merely in `run`. Registry must consume the result instead of hiding a
compiler behind `declared`. Source admission remains coordinated with fn-195.3.

This caller/producer/held-admission surface is not one M task. The conductor must record the actual
producer and blocked complete-universe/caller prerequisite tasks as dependencies of fn-195.8 before
closing this proof. A later scoped contract amendment may replace fn-196.2's stronger ordering only
after the complete proof succeeds. No such amendment is justified by this bounded generic result.

## Artifact identities and reproduction

`input-artifacts.sha256` binds the fresh author/client jars and all author sources. Rechecking it
and `.flow/tmp/fn1958/source-freeze-v3.sha256` exits zero in the final controls. The latter covers
all 14 frozen product paths. `java-version.log` and `scala-cli-version.log` record tool identities.
The final proof-artifact manifest binds scripts, sources, commands and logs separately from task
receipt files.

| Artifact | SHA-256 |
| --- | --- |
| Fresh original author jar | `7a8786b227c3a1fd73816f93c1b8e119ba18ecc9c93c5b71c947cedd08bf7bf4` |
| Fresh matrix author jar | `8c19eefd66620879150039e06c69b646419b18b52fe251e217b930c730ba0758` |
| Fresh external client jar | `aad346e20d681fb1c669e9561429da8879e874774c9f6dfea52b34e193409756` |
| Source-absent helper jar | `de58e6a1a530005af8c963dc7cf5630575683a161a67b4314a56cc5ed340c739` |
| Source-absent client jar | `403f975f472c79ebe9281260ccffdbe3e3361dc28cc619e3aa3760810808615b` |
| Frozen declarations jar | `4a2f9c1c276987558a8141b2f565bc76a059150224059800268d05c90c95f224` |
| Frozen checker jar | `f21e97611ca84a24964bf3b66650fe5fec424cceb53209de11fb4bb2f50abd87` |
| Existing IR jar | `b9e0538afffad2b33b9fc8fc9eb2a3f4e63b10928fa38506b585747535e85c92` |

`reproduce.sh` allocates fresh task-local outputs and workspaces, copies the disposable fixture
sources and runs the bounded recipes serially. `reproduction.log` records terminal exit zero for
the assembled fresh generation at `reproduction.XS6sIs`, including exact inventories and the final
direct policy/ownership controls. `reproduce.md` describes stage provenance and the unchanged
holds. The later `source-absent.sh` is a separate serial recipe with its own commands, terminal
exits and restoration checks. A wrapper success cannot promote the unproved held universe or
actual production caller ordering.
