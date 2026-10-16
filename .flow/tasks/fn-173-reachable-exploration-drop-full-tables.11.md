---
satisfies: [R6, R7]
---
# fn-173-reachable-exploration-drop-full-tables.11 Finite record input domains and input-reading native guards

## Description
Complete native finite domain and guard authoring on fn165's settled record inputs; use small framework-only examples. Full six-sample reachable-graph equivalence remains .6's later proof.

**Size:** M
**Files:** framework DSL Action, InputSource, Syntax, Rules tests, small samples and framework-local guidance.
**Touches:** [model/framework/dsl/**, model/framework/samples/**, model/framework/README.md, model/framework/migrate/README.md]

### Approach
- Add action-local/per-field finite oneOf domains, including explicit Range expansion, using the record frontend. Do not copy stale token-based design spellings or install global Finite[Int]. Distinguish finite oneOf values from expect's Unit branch alternatives.
- Supply explicit record-field domains before whole-record/per-field Finite lookup: a modifier on an already constructed RecordAction cannot solve missing Finite[Int]. Reuse concrete typed InputField descriptors, including independent domains for two Int fields, without standalone tokens or author-supplied field-name strings. Range expansion preserves its iteration order; collection-valued arguments remain individual values, not implicitly flattened iterables.
- Preserve a selected whole-record witness's correlations, order and actual object identities. Do not silently replace it with a Cartesian product of projections. Make whole-record versus per-field selection modes explicit; reject a call explicitly supplying both until an intersection/order contract is defined. No-override declarations keep their existing selected-witness path.
- Bind typed where((s,in)=>...) through existing rule dispatch and show actual input-dependent enabling/results. Preserve no-input guard behavior and class/default/domain order.
- Pass the same retained actual input to guard and effect, filtering exact class targets before guard evaluation. Preserve ordered short-circuit across state/input conditions and skip effects for false guards. Preserve state-only no-input, pinned-class and heterogeneous-target behavior; refuse unsupported input-reading contexts explicitly rather than through callback casts.
- New explicit oneOf domains reject emptiness and duplicate values when constructed, without sorting or silent deduplication. Ordinary selected Finite metadata stays lazy, including intentional empty/throwing-catalog controls. Installing field domains must not invoke record constructors/default getters; materialized classes, including omitted defaults, must satisfy selected field and whole-record membership before use. An already supplied scalar default may be validated immediately without evaluating it again. Preserve receiver/argument/default evaluation order and exactly-once behavior.
- Reject malformed domain bindings, invalid materialized/default values, wrong guard arity/types and unsupported native DSL forms. Full collection-expression subset/capture refusals remain original .5's obligation: Data evidence or a successful host guard callback does not establish operation liftability, and this task must not add a Scala semantic checker. Native samples demonstrate the settled author surface without reached-graph/Go or export proofs.

### Investigation targets
**Required:** model/framework/dsl/Action.scala; model/framework/dsl/InputSource.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Inputs.test.scala; model/framework/dsl/Rules.test.scala; model/design/bounded-collections.md.

### Verification
Run touched native input/default/rule/expect tests and check/lint selections with nonzero counts. Keep original .5 lifting/IR and .6 graph-isomorphism proofs held.

## Acceptance
- [ ] Explicit finite domains retain order/defaults without global numeric enumeration; malformed domains and out-of-domain defaults reject.
- [ ] Per-field domains work without global Finite[Int]; correlated whole-record witnesses retain their joint membership/order/identity, and unexplained explicit mixed selection modes refuse.
- [ ] New oneOf emptiness/duplicates reject immediately; ordinary metadata remains lazy, domain installation invokes no record/default getter, and materialized omitted defaults are validated before use with valid paired controls.
- [ ] Actual native dispatch depends on both state and supplied record input; wrong guard shapes fail compilation.
- [ ] Guard/effect input identity, exact-class filtering, state/input short-circuit and false-guard effect suppression hold through native facets/derivations without weakening no-input/class-specific/heterogeneous state-only behavior.
- [ ] Domain oneOf and expectation oneOf coexist without ambiguous authoring; native examples/docs and focused tests pass.
- [ ] Full liftability, numeric interpreter parity and six-sample reached-graph equivalence remain unclaimed.

## Done summary
Native action authors can declare independent finite record-field and scalar domains before finite lookup, without a global numeric catalog. Typed native guards read the same retained input as effects, preserving exact-class filtering, ordered short-circuiting, defaults, correlated whole-record witnesses, facets and derivations.

stage: impl-review - skipped(config: REVIEW_MODE=none)
Tier: session (jev-unavailable(no_key))

### Native acceptance

- Final surface: `.inputs[In](inputField[In](_.field).in(domains.oneOf(...)), ...)`, explicit `.inputs[In](wholeWitness)`, and scalar `.input[A](name = ..., domain = ...)`. Two Int fields have independent domains; remaining fields use lazy typed Finite fallback. Enumeration follows record order, last field fastest. Range iteration order is retained; collection arguments remain whole values. Empty and duplicate explicit domains refuse immediately.
- Typed direct-field descriptors carry owner/type evidence across jars. Foreign, computed and wrong-type selectors refuse compilation; duplicate bindings, missing field catalogs and explicit mixed whole/field selection refuse construction. Installing domains invokes no record constructor/default getter. Materialized supplied/default values validate field and joint membership. Field mode validates directly without constructing Cartesian records; ordinary selected empty/throwing witnesses stay lazy.
- Whole-record witnesses remain authoritative, preserving joint membership, authored order and actual object identity. Missing per-field Finite metadata uses lazy typed projections, never an unchecked field lookup or a replacement Cartesian catalog. Scalar products, Tuple1, Tuple2 and EmptyTuple stay genuine single inputs.
- Binary `where((s, in) => ...)` infers the exact retained input before typing its body. A single unary member plus binary extension preserves the existing `where(identity)` form. Guards and effects receive identical objects, with class filtering first and state/input/phase conditions short-circuiting in written order. False guards skip effects. Rebind/restrict/extend, rule-group and facet paths retain this adapter. Zero-input, pinned and heterogeneous contexts retain state-only behavior; unsupported input-reading shapes fail compilation.
- Retained guard metadata validates binary arity and selected input schema at registration and collection without catalog/guard/effect execution. Positive collection plus exact negative diagnostics prevent unrelated metadata failures from earning credit. No new capture proof, serialized evaluator or Scala operation checker was introduced.
- New public collection-state author checks exposed .10's private sealed proof accessibility warning. CollectionMember and MemberFields companions are now publicly accessible while their evidence remains sealed. Full native warning flags pass, external forgery refuses, and collection-contained Inbox still refuses. Native comparison and ordering behavior from .10 remains intact.

### Surface and timing evidence

Unqualified finite oneOf forwarding overloads were actually tried, including after expectation inference was corrected. The isolated paired eighth/ninth runs removed only those forwarders: ordinary unchanged sample branches lost Alternatives with the overloads and compiled without them. `domains.oneOf` is the evidenced explicit finite-constructor fallback; expectation oneOf/branch remains a distinct Unit form.

An independent probe showed direct fresh-record expectation field reads worked while nested branches failed. Moving expectation context clauses before bodies repaired those same field reads; ExpectationBlock validation was unchanged. The native regression and separate author/client evaluate direct/nested collection-field predicates with independent true/false pins.

FacetBindings retains its zero-catalog/guard/effect acquisition assertion. Its exact post-step catalog pin changes from zero to one because the explicitly supplied scalar now undergoes required input-domain validation. Later metadata collection preserves the complete observed vector. Captured failed observation: `fn17311-facet-validation-red-excerpt.md` (the original log path was reused for the final pass). No whole-state enumeration or registration timing was relaxed.

### Verification

Baseline: green, actual pre-edit Input* command exited 0 with 36 tests at pre-edit HEAD `9a96f1c2ee8b5ca67a19d754b5cc07de67207009`; log `fn17311-baseline-input.log`. The initial baseline lint invocation's terminal exit was not captured and earns no passing credit; final lint was observed properly.

All final native commands used DSL/realize/sample roots, required nonempty selection and exited 0. The canonical test prefix is `mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only`.

- `framework.dsl.Input*`: 51 tests; `*Suite`: 90; `RulesTest`: 26; `Expect*`: 19; `NativeDataTest`: 9; `Equality`: 4; `FiniteSchemaTest`: 1; `FacetsTest`: 12; `RuleSourceTest`: 14; `SampleExpectTest`: 6. These total 232 focused native passes, not a complete native/table/law gate. Logs `fn17311-verified-<selection>.log`, with wildcards spelled `all`.
- Native lint command `mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite'`: exit 0, 29 tests, 107-file native scan, zero findings; `fn17311-lint-final.log`.
- `mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn17311-external-recipe.sh`: exit 0; fresh framework library, separate public author library and separate client; 59 runtime checks and ten line/diagnostic-checked compile refusals. Every stage uses the full native strict/warning flags. Log `fn17311-external-final-confirmed.log`; recipe and scaffold retain exact stages and limitations. Public proof uses no reflection or framework-package privacy bypass; private dispatch has its independent framework tests.
- Scoped Scalafmt covered 14 changed Scala files; a second pass preserved every SHA-256, fmt --check exited 0 and git diff --check passed. Manifest `fn17311-format-before.sha256`; logs `fn17311-format-{second,idempotence,check}.log`.
- Classification returned FULL, exit 1, for unmatched `model/framework/README.md`; `fn17311-gate-classify.log`. Scoped green receipts under `.flow/tmp/green-receipts/86c9a904-*` certify only their named commands at source commit `86c9a9049f3e50e1406adc42b9d882df82d25beb`. No docs-only skip, full-native-304 pass, broad gate or SHIP verdict is claimed. Spec validation returned valid with no errors/warnings.

Red evidence includes missing domain/binary guard APIs (`fn17311-domains-red.log`), public proof accessibility (`fn17311-input-first.log`), nested expectation inference (`fn17311-expect-isolation.log`), unqualified overload collision (`fn17311-input-eighth.log`), and unary identity inference (`fn17311-input-ninth.log`). Refuted intermediate approaches were removed. Early external runs were explicitly inconclusive for diagnostic matching; final matching requires the actual typed owner pair and the sealed-trait refusal across ANSI coloring.

### Handover and held work

Source commit: `86c9a9049f3e50e1406adc42b9d882df82d25beb`. The full pre-edit range also contains conductor-owned metadata-only commit `f4d2cd2f2bfa4ec558b1b3e58cd6f6c7c7a9d73a` for parent/.12 policy; it is not .11 implementation. Three disjoint assistants implemented domains, guards and external scaffolding; this worker reconciled them and owned every compiler, source commit and task lifecycle operation. All assistants and started commands are terminal.

Only native .11 closes. Original .2-.9, complete R6/R7 lifting/collection capture, interpreted numeric parity, all six reached-graph proofs, whole-carrier retirement, native explorer/checker and backend/system integration remain held. No Temporal restoration/read, exporter/reify/schema/backend/toolchain change, regeneration, MILESTONES edit, .12 implementation or parent closure was added by this worker. Formal review and broad gates remain the owner's batch-end work.

Tracker sync: n/a (bridge inactive).
stage: plan-sync - skipped(config: planSync.enabled=false)
## Evidence
- Commits: f4d2cd2f2bfa4ec558b1b3e58cd6f6c7c7a9d73a, 86c9a9049f3e50e1406adc42b9d882df82d25beb
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.Input*', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.*Suite', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.RulesTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.Expect*', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.NativeDataTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.Equality', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.FiniteSchemaTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.FacetsTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.RuleSourceTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.SampleExpectTest', mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite', mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn17311-external-recipe.sh, sha256sum -c .flow/tmp/fn17311-format-before.sha256, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/dsl/Action.scala model/framework/dsl/ActionInputs.scala model/framework/dsl/Domain.scala model/framework/dsl/DomainChoices.scala model/framework/dsl/FacetBindings.test.scala model/framework/dsl/InputField.scala model/framework/dsl/InputSource.scala model/framework/dsl/Machine.scala model/framework/dsl/RuleBindings.scala model/framework/dsl/RuleInput.scala model/framework/dsl/Syntax.scala model/framework/dsl/InputDomains.test.scala model/framework/dsl/InputGuards.test.scala model/framework/samples/NativeInputs.scala, git diff --check, python3 /home/agent/.codex/scripts/flowctl.py validate --spec fn-173-reachable-exploration-drop-full-tables --json
- PRs: