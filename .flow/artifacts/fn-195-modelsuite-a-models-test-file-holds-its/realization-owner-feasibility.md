# Realization owner feasibility

Complete realization inventory remains blocked. A disposable typed stable-path resolver obtains exact existing external receivers in source and across a library boundary, including a strict Vector holding owners. Its broader getter traversal executes an unrelated lazy helper while searching for owners. A narrow metadata-directed refusal prevents that execution and stops both implicit and explicit selection. Neither version establishes complete safe discovery for ordinary declarations.

This investigation completes fn-195.15's bounded negative route. It supplies no R4 implementation, fn-195.11 release, suite-inventory admission, conformance integration or general impossibility result. The frozen [.10 report](conformance-admission.md) remains the earlier acquisition-epoch observation authority.

## Independently bounded inputs

The source base is `22d96ea2e46760d4b25629e17ed43dddb72df0ed`. All disposable sources, snapshots, commands, statuses, artifacts and diagnostics are under [fn195-realization-owner-proof](../../tmp/fn195-realization-owner-proof/). Only this report changes tracked product history.

The declared author universe consists of the retained classes and public immutable members in the explicitly prepared `Library.scala` and `Client.scala` files. The prototype selects those two source basenames from the supplied artifacts' typed trees. It excludes mutable fields, private fields and module declarations from ordinary getter traversal; admitted static modules are separate roots. These are prototype bounds, not author restrictions or a completeness claim. Basename matching does not validate arbitrary source provenance, artifact trust or classloader identity. A production mechanism would need the stronger artifact/source ownership checks that existing `SuiteInventory` demonstrates.

The source arrangement compiles those files with an unchanged copied native framework and compiles the oracle separately as test sources. The library arrangement packages the copied framework and `Library.scala`, then compiles only `Client.scala`, `Resolver.scala` and `Owner.test.scala` against that jar. Library-owned `External.second` and client-owned `ClientExternal.second` both participate. Each command uses a fresh compiler workspace. The compiler input and option JSON files are retained alongside the generated class/TASTy hashes.

The resolver receives only the actual suite and artifact paths, obtained independently from the library Model and client suite class code locations. It derives module binary names, stable getter names and realization result types from metadata. Expected receiver and candidate references stay in the oracle. No owner list, candidate list, registration, relocation, acquired-so-far registry or arbitrary classpath scan supplies discovery authority.

For the first checkpoint, setup initializes `ModelRoot.model` and `ExternalRoot.owner` without forcing the external owner's lazy `second`. `ClientSuite` binds the same exact Model and eagerly constructs `first`. Module metadata supplies the path through the singleton's `MODULE$` and the existing `owner` getter. The resolver invokes `second` on that receiver. An identity map bounds repeated visits without invoking user equality. Traversing a strict immutable Vector projects existing stored references, including `Vector[AnyRef]`; the prototype does not interpret arbitrary collection callbacks.

## Executed observations

Each row names a retained directory under the proof root. Its `command.txt`, `output.log`, `status`, source snapshot and source manifest identify the observation. Exit codes determine outcomes; the verifier reads existing logs rather than rerunning suites.

| Observation | Terminal result | Receipt directory |
| --- | --- | --- |
| Actual parent Quick baseline | Exit 0, 4 `CheckerTest` cases | `baseline-quick.log`, `baseline.md` |
| Suite-root-only negative control | Exit 1, exactly one intended size 1 versus 2 failure | `red-direct-v3` |
| Exact direct external receiver, source | Exit 0, 1 case | `direct-v6` |
| Initial separate library package | Exit 0, jar written | `library-direct-v6` |
| Exact direct external receiver, packaged client | Exit 0, 1 case | `client-direct-v6` |
| Stable-getter-only Vector negative control | Exit 1, 2 cases pass and one intended size 1 versus 2 failure | `red-vector-v7` |
| Expanded source controls and safety oracle | Exit 1, 11 cases pass and exactly one intended poison mismatch | `matrix-v11` |
| Expanded separate library package | Exit 0, jar written | `library-matrix-v11` |
| Expanded packaged controls and safety oracle | Exit 1, 11 cases pass and exactly one intended poison mismatch | `client-matrix-v11` |
| Narrow refusal, source | Exit 0, 1 case containing implicit and explicit branches | `safe-source-v12` |
| Narrow refusal, packaged client | Exit 0, 1 case containing implicit and explicit branches | `safe-client-v12` |

The expanded source and packaged routes execute the same eleven positive cases. They establish direct library and client external receiver acquisition; strict Vector projection; constructor stable fields and inherited getters; override dispatch deduplicated to one runtime reference; same-reference aliases versus distinct generic copies; delayed custom-kit acquisition without dependency or conversion callbacks; foreign exact Model and unlisted-reference refusals, including zero implicit candidates; repeated acquisition and reversed container order; getter failure before selection; and a bounded released-owner GC witness. This count is eleven cases, not a separate test for every subassertion.

Two discovered distinct candidates refuse omitted `through` before the prototype returns a selection. Explicit selection requires exact `candidate.machine eq suite.model` ownership and reference membership in the obtained inventory. These are bounded fixture selection checks. They do not certify the inventory's completeness or authorize production conformance declarations.

The inheritance case invokes the actual inherited getter. The override case discovers both declarations, invokes through JVM dispatch and deduplicates the resulting identical reference. That observation does not settle whether the parent contract should retain both declaration identities. The GC case observes released references clearing within at most 30 requested collections. It proves no universal collection guarantee or concurrent lifetime contract.

Earlier `red-direct-v1/v2`, `direct-v4/v5` and `matrix-v8/v9` retain fixture compiler or metadata-mapping failures. They receive no final semantic credit. `matrix-v10` reached the same safety failure with ten passing cases, before the final zero/collection checks. Every version keeps its own snapshots; no earlier row was refreshed.

## First core safety failure and safe refusal

The initialized owner is ordinary source code, with one realization declaration and an unrelated deferred helper.

```scala
class DeferredStorage(val model: Model):
  lazy val first = generic(model)
  lazy val unrelated: AnyRef =
    Poison.touches += 1
    throw IllegalStateException("unrelated deferred storage executed")
```

The original resolver treats every admitted public immutable member as a possible receiver edge. After obtaining `first`, it invokes `unrelated` to look for further owners. Source and packaged oracles both require the independently retained poison count to stay zero. Each observes one instead. The acquisition throws before returning an inventory, but the unrelated callback has already executed. Thus this failure concerns discovery safety, rather than a false successful uniqueness result.

The missing authority is a safe initialized-storage edge. A stable typed getter symbol does not prove that reading an owner/helper field only returns an existing reference. It may initialize arbitrary deferred source code. Excluding all such declarations would impose an unsupported author restriction; invoking them indiscriminately violates safe discovery. This experiment stops at that core failure rather than claiming complete construction machinery.

The follow-up control retains an additional metadata bit for a lazy non-realization member whose dealiased type is `AnyRef`. Before invoking that member, `acquireSafe` throws `UnsupportedOperationException` naming unsupported deferred receiver storage. Source and packaged controls prove that the poison stays zero and the selection boundary is never reached for both omitted and explicit `through`. The explicit reference remains privately held by the oracle; it does not excuse incomplete discovery under the selected contract.

This is a narrow refusal for the reproduced shape. It does not prove safe acquisition for every lazy result type, establish whether a field is already initialized, solve module initialization or exclude any ordinary declaration from the parent's required universe. No safe-refusal result supplies an eligible immutable declaration. No already issued declaration can be invalidated in this control because selection never returns one.

## Remaining boundaries and later ownership

The minimal failure leaves uninitialized modules and owners, invalid dormant candidates, getter reentry, distinct late candidates, atomic close, concurrent acquisition and sticky eligibility invalidation unverified. No concurrency contract is claimed. The prototype can initialize metadata-derived modules while obtaining `MODULE$`; only the initial explicitly initialized fixture setup has the stated initialization evidence. No general safe module-loading result follows.

Eager generic and delayed custom kits were exercised. An eager custom-kit variant, arbitrary private stable storage, additional standard containers, recursive lazy owner creation and factory/caller-local receiver authority were not established. A factory result stored in an ordinary stable field and an unrooted caller-local owner must not be conflated. Stable Model-holder identity alone settles neither eligibility nor exact receiver acquisition. Unsupported or unresolved boundaries cannot silently count as absent candidates. The route offers no complete close independent of those boundaries, and no complete late-acquisition or declaration-order proof.

The prospective owners remain `framework/inspect` for bounded declaration/artifact identity and receiver-path metadata, `dsl/ModelSuite.scala` for the actual post-construction pre-registration boundary, and `realize/Constructed.scala` for final validated acquisition. The missing safe initialized-storage and module-acquisition contract must be selected and proved before assigning production implementation tasks. This report supplies no evidence-backed complete mechanism decomposition. Existing `SuiteInventory` inventories suite ancestry and supplies no realization-owner acquisition by itself.

fn-195.11 remains blocked on actual integrated complete mechanisms and its other prerequisites, including .9. The .8/.14 and fn-196.2 holds remain. No production DSL, realization, checker, inventory or registry source changed.

## Reproduction and immutable bindings

The runner uses Scala CLI 1.17.1, Scala 3.9.0, JVM selection 27, munit 1.2.0, ScalaPB runtime-grpc 0.11.20 and tasty-inspector 3.9.0. Safe-control runtime output identifies Zulu 27+35 and its actual Java home. `tools.txt` separately records the host mise Java selection, Temurin 27+35. Every compilation retains `--server=false` and all flags below.

```text
-Werror -deprecation -feature -unchecked -Wunused:all -Wvalue-discard
-Wnonunit-statement -Wsafe-init -Wimplausible-patterns -Yretain-trees
-language:strictEquality
```

The copied native manifest covers 95 Scala files. The unchanged frozen IR jar is a compilation dependency only, with original and copied SHA-256 `b9e0538afffad2b33b9fc8fc9eb2a3f4e63b10928fa38506b585747535e85c92`. The expanded library SHA-256 is `ce9eb4ae1b3fd0efee99aeb7245ba78613f8fa4e32279630e0c95ca0b5bc64a3`. Source/library oracle snapshots are byte-compared by the verifier. Per-stage artifact manifests bind compiled class and TASTy bytes; `final-bindings.sha256` binds commands, logs, exits, source/artifact manifests, libraries, tool observations and verifier code.

The complete bindings manifest has SHA-256 `72f13ff35f6cd52d2bae25f372d08b6ab4b1f8c5f4a7db5c91d6e51fee512374`. The evidence verifier exits 0 in `verification.log`. [processes.md](../../tmp/fn195-realization-owner-proof/processes.md) records every joined command handle and the earlier incomplete binder attempt. No proof-owned JVM remains.

The exact parent Quick command ran before edits, without a handoff or receipt skip.

```sh
mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only checkercontrol.CheckerTest
```

The recorded experiment invocations use the task runner below, with fresh version/output paths and a 600-second limit per compiler/test command. The original packaged oracle uses the frozen `matrix-v11` source snapshots. Replaying these names would overwrite retained evidence and the runner refuses existing output directories. `verify.sh` only checks existing evidence and is safe to repeat.

```sh
env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-realization-owner-proof/run.sh matrix-v11 source
env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-realization-owner-proof/run.sh library-matrix-v11 library
env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-realization-owner-proof/run.sh client-matrix-v11 original-client .flow/tmp/fn195-realization-owner-proof/library-matrix-v11/library.jar
env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-realization-owner-proof/run.sh safe-source-v12 safe-source
env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-realization-owner-proof/run.sh safe-client-v12 safe-client .flow/tmp/fn195-realization-owner-proof/library-matrix-v11/library.jar
env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-realization-owner-proof/verify.sh
```

Expected safety-oracle exit 1 remains a failed route requirement. A green evidence verifier means that the specified positive and negative observations matched their recorded expectations. It does not turn the safety failure into a passing implementation.

Flow's final scope classification reports `TIER_B: docs-only (1 files)`. `git diff --check` passes. The final Quick gate is skipped under that classification, after the actual four-case baseline ran.

```text
GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched)
```

The repository's changed-scope Make gate remains held and receives no verification credit.

Actual repository Make, Go, export/reify/lifting/IR/schema/backend/generated/regeneration, Testpilot, server, recorded Run and Case operations remain unexecuted. No `model/temporal` content was read or searched. Historical Bloop PID 93084 and other-owner worktrees remain untouched. Export compile-only permission remains pending and ungranted. Formal implementation review remains phase-batch deferred; this report claims no SHIP verdict.
