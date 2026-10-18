# Native capability retention feasibility

An ordinary single-binding `claim(Reachable.reached[State])` can retain its real companion predicate and generate its declared native path across a packaged-library boundary. A disposable inline-capture route passes source and outside-package replay controls. Multiple bindings, directive collisions, public capability-law authoring and selected authored bounds remain unresolved production contracts. This report supplies no R4 implementation or fn-195.11 release.

## Boundary and mechanism

The source base is `3b795ae1d1c0bbe18e11f390c86a0f89ce0888d9`. The earlier [conformance admission report](conformance-admission.md) remains the discarded-predicate authority at its own checkpoint. The [realization-owner report](realization-owner-feasibility.md) remains a bounded negative discovery result. Its closure supplies no complete realization inventory.

All executable changes live under [fn195-capability-native-proof](../../tmp/fn195-capability-native-proof/). The probe copies the native DSL, realization and checker sources, plus the unchanged frozen schema jar. Four scratch paths differ from product sources. `Capabilities.scala` captures the existing `claim`, `except` and both `overriding` calls; `CapabilityCapture.scala` retains a compiler-selected method identity, companion reference, named fields and original callable; `NativeCapability.scala` expands initialized direct bindings; `NativeCapabilityReplay.scala` adapts an admitted expansion to existing native `Replay.run`. The three new files total 155 lines, excluding the modified capability entry points, fixtures and evidence machinery. This is a prototype size observation, not a production estimate.

The first fixture preserves `object capabilities extends Capabilities`, the explicit `val original: Capability = Reachable[State](...)` annotation and ordinary claim spelling and `Property` return shape. Capture occurs before conversion to an unidentified `AnyRef` function. It retains the selected method's qualified name and generic signature, the actual companion and original callable. Runtime acquisition does not search method names to recover a law. The macro admits direct unchanged argument forwarding and rejects the tested altered-argument lambda.

Acquisition checks the section's exact Model and separately resolves the Model's capabilities getter to the exact section receiver. It obtains named initialized values through the existing `NativeRuntime.section` seam. The actual case-class `Product` fields survive the `Capability` alias. The probe maps the two field names into the captured three-argument callable, passes the exact Model, retains the actual generated `Property`, checks its owner and nonempty predicate, and validates every path class against that Model's bound classes and inputs. It also requires the predicate selector to select a step on the generated path.

The descriptor retains exact Model, binding and Property references; binding getter; original and effective law; actual concrete path; and the effective directive with its reason and named binding references. The adapter receives only the section, returned Property, exact Model and caller budget. Fixtures supply it no path, callback, ReplayStep or ReplayScript. The adapter produces replay operations internally and evaluates the retained predicate using native last-step context. Capture and acquisition do not evaluate the predicate.

The class match resolves the case-class binary name derived from the captured companion using that companion's classloader, then compares exact Class identity. This is a bounded ordinary Scala case-class naming assumption. The probe does not establish arbitrary companion layouts, hostile duplicate artifacts or general classloader admission.

## Executed observations

Each directory below retains the expanded command, output, terminal status, caller snapshots and hashes. Final framework compilation reads its immutable version-local native snapshot. The separate library contains the declaration and internal generic law. Its outside-package client compiles only `Client.test.scala` against the staged framework and declaration jars. Its source directory contains no `Library.scala`; the logged compiler inputs and runtime class origins identify those artifacts.

| Observation | Exit and selected count | Evidence directory |
| --- | --- | --- |
| Actual parent Quick baseline before edits | 0; four CheckerTest cases | `baseline-quick.log` |
| Original empty-claim oracle | 1; one intended semantic failure | `red-v3` |
| First retained source predicate | 0; one case | `source-capture-v1` |
| First generated source replay and directives | 0; five cases | `source-replay-v3` |
| First declaration-library package | 0; jar written | `library-checkpoint-v1` |
| First outside-package ordinary replay | 0; one case containing four field/path variants | `client-checkpoint-v2` |
| Descriptor lifetime and vacuous-selector oracles | 1; eight passing cases and two intended semantic failures | `matrix-red-v1` |
| Final framework package | 0; jar written | `framework-final-v1` |
| Final ordinary source controls | 0; eleven cases | `source-final-v2` |
| Separately selected direct-companion ingredient | 0; one case, no expansion credit | `ingredient-final-v1` |
| Final declaration-library package | 0; jar written | `library-final-v2` |
| Final outside-package controls | 0; five cases | `client-final-v1` |
| Altered forwarding lambda before guard | 0; compiler admitted the altered callable | `altered-before-v1` |
| Same altered forwarding lambda after guard | 1; intended located compilation refusal | `altered-after-v1` |
| DSL/realization-only package | 0; jar without checker classes | `declarations-final-v1` |
| Declaration-only load and missing-checker acquisition | 0; one outside-package case | `nochecker-final-v1` |

The final source and client counts are separate receipts. The earlier combined `source-final-v1` has twelve cases, including the ingredient. No count sums versions or treats an ingredient as ordinary expansion evidence. Initial fixture/macro compile failures remain in their original directories and earn no semantic credit. [processes.md](../../tmp/fn195-capability-native-proof/processes.md) identifies joined command handles and the superseded mutable-root packaging observation.

The four ordinary field/path variants have independently expected verdicts. `(accepted=true, advance)` verifies, changing only accepted to false refutes, changing only the declared path to retreat changes the actual trace and false outcome and refutes, and `(accepted=false, retreat)` verifies. Every replay retains one concrete step. Original predicate tallies remain zero through acquisition and rise to one in replay. The global replacement refutes the positive original with original tally zero and replacement tally one. The outside-package client repeats these controls and records both Model and law origins from `library-final-v2/library.jar`, with Zulu Java 27 at the actual runtime Java home.

The first semantic oracle reproduces the product's empty `PropertyDecl`. The later two semantic REDs show a returned Property losing its descriptor after a second claim, and a generated law whose selector misses its path being admitted. The final probe appends independent immutable descriptors and refuses the missing selector before replay. A successful claim followed by a foreign-generated-property refusal preserves the earlier replayable Property. Multiple-binding expansions remain observable with `admitted=false`; attempting to replay one refuses. A failed singular claim therefore supplies no replay admission.

Missing fields, an unrelated companion, a waived requested law, a same-primary-class foreign exact Model, a foreign generated Property, foreign action identity and malformed class inputs refuse. An incompatible `accepted: String` law reaches the captured callable's erased cast and throws ClassCastException before its predicate. This is an observed refusal, not a finished located field-type diagnostic. A separate direct companion test constructs a real predicate but leaves the section's generated-expansion inventory empty.

## Directives and unresolved singularity

The source and packaged fixtures retain two ordinary named bindings with different accepted fields. Their internal expansion view preserves both exact bindings and their own properties and effective laws. The existing `claim(law): Property` has no binding selector. The probe refuses that singular result rather than choosing a first binding or conjoining obligations. Internal expansion retention proves a candidate seam; it does not resolve the public contract or earn multi-binding replay admission.

| Declared directives for requested law | Observed probe result |
| --- | --- |
| None | Original law on both internal expansions; singular claim refuses two bindings |
| One global replacement | Replacement on both internal expansions; single-binding fixture proves replay behavior |
| One named replacement | Only `overridden` changes; unselected `original` retains its own law |
| Waived neighboring law | Requested reached law remains available |
| One waiver of requested law | Refuses before replay |
| Duplicate waivers | Ambiguous-directive refusal in both orders |
| Duplicate global replacements | Ambiguous-directive refusal in both orders |
| Two replacements for the same named binding | Ambiguous-directive refusal in both orders |
| Global plus named replacement | Ambiguous-directive refusal in both orders |
| Waiver plus replacement | Ambiguous-directive refusal in both orders |
| Named replacement selecting a foreign binding | Refuses unselected or foreign binding |

The collision refusals are chosen probe boundaries. No admitted contract selected precedence for these combinations. The observations cover reversed directive execution order and fixed direct-val declaration order, not every inherited or helper declaration arrangement. The existing section seam pairs declared getters with fields and does not establish complete inherited/helper capability discovery. No registration list, relocation or replacement DSL fills that gap.

## Bounds and later production ownership

The concrete generated path has length one. An internal finite-path control refuses a bound of zero and replays at bound one. With that same path bound and caller budget one, native replay returns unknown and incomplete; budget eight verifies the positive. The derived finite length and immutable caller execution budget are separate facts. There is no selected authored native bound to retain. No Limits builder or new bound syntax was introduced, and this internal control does not supply the missing authored-bound contract.

The transported law uses framework-private `PropertyBuilder.sameStep`, matching the frozen generic-kit ingredient. Current plain-form authors cannot create that law through the retired public property builder. An outside-package consumer proves library-law transport, not a new public law DSL. The prototype additionally supports only the tested owner-plus-two-fields method template and a `List[Class]` path field. It retains the generic signature and callable but does not furnish a complete instantiated parameter schema, arbitrary arities, multi-kind laws, arbitrary path carriers or located type-error diagnostics.

The evidence supports a later DSL owner for typed law/directive retention and actual binding-field expansion, followed by a checker adapter consuming an admitted descriptor. Production sizing must first settle singular selection, collision policy, public law authoring and selected bounds. Complete capability-section discovery, initialization/close, post-acquisition mutation and concurrency also need their own admission evidence; repeated immutable probe claims do not establish those contracts. The complete realization mechanism remains independently blocked by .15. Existing .17 coverage and .18 expectation leaves supply no actual C/ModelSuite integration.

Keep fn-195.11 blocked until complete mechanisms are implemented, committed and integrated with its other leaves and .9. The .8/.14 and fn-196.2 holds remain. This investigation does not change production DSL, capability machinery, registry, checker, feature maps, plans, dependency ledgers or MILESTONES.

## Reproduction and immutable bindings

The pinned runner uses Scala CLI 1.17.1, Scala 3.9.0, JVM selection 27, munit 1.2.0 and ScalaPB runtime-grpc 0.11.20. All compiles preserve `--server=false` and the strict options below. Version-local snapshots contain 187 Scala paths, including test sources; 98 paths are non-test Scala files. Library packaging excludes test-stage sources.

```text
-Werror -deprecation -feature -unchecked -Wunused:all -Wvalue-discard
-Wnonunit-statement -Wsafe-init -Wimplausible-patterns -Yretain-trees
-language:strictEquality
```

| Frozen artifact | SHA-256 |
| --- | --- |
| Framework/macro/checker jar | `ba0e56c167946ba7e6d5c291126cb87bfe1dc5bd77ece5da047b4b65abc042b8` |
| Declaration and generic-law library | `a90fdeec41eb21f2316f342df502da49ceed9b0a3a0b62304112f4878de687ca` |
| Final source/library declaration fixture | `b41805de103a2a921b5d2730d345729fa326d477a150bc868585855e797064e5` |
| Final outside-package client | `73e8f742b3e962d12286da8e26a6198710917a0d3d70c6a341f7b8d12d1aef83` |
| Frozen IR compilation dependency | `b9e0538afffad2b33b9fc8fc9eb2a3f4e63b10928fa38506b585747535e85c92` |

The complete [final-bindings.sha256](../../tmp/fn195-capability-native-proof/final-bindings.sha256) has SHA-256 `e908456eab1c66933892475e057cc97f81cf36bb76568047655aee737d22acbc`. It binds commands, logs, statuses, source snapshots, source and compiler-artifact inventories, staged jars, tools and verifier code. The final framework, source and client artifact inventories contain 1361, 26 and 6 class/TASTy/compiler-input entries respectively. The [evidence verifier](../../tmp/fn195-capability-native-proof/verify.sh) exits 0 in [verification.log](../../tmp/fn195-capability-native-proof/verification.log). The final process snapshot contains zero proof-owned JVMs.

The exact parent Quick command ran before edits with four passing cases, without a handoff or receipt skip. Final Flow scope classification reports `TIER_B: docs-only (1 files)` and `git diff --check` passes. The changed-scope repository Make gate remains held and receives no verification credit.

```text
GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched)
```

The exact expanded command lives in each receipt directory's `command.txt`. The runner refuses an existing version path; reproducing a compile/test requires a fresh path. The evidence verifier reads completed logs, status files and hashes without rerunning suites. Commands use `env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-capability-native-proof/run.sh <fresh-version> <mode> <framework-jar> [declaration-jar]` from the repository root. No suite reran merely to observe its exit code.

Actual repository Make, Go, exporter, compiler-plugin/reify, lifting, IR/schema/backend/generated/regeneration, Testpilot, server, recorded Run and Case operations remain unexecuted. No `model/temporal` content was read or searched. The historical Bloop PID93084 and other-owner worktrees remain untouched. Export compile-only authority remains pending and ungranted. Formal implementation review remains phase-batch deferred; this report claims no SHIP verdict.
