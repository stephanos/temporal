# fn-89-one-contract-rule-per-entity One Contract Rule per entity

> HTML render lens (local): open `.flow/artifacts/fn-89-one-contract-rule-per-entity/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Umpire4 architecture reconciliation

This spec lets a Contract Rule be declared once for an entity and instantiated once per instance of
it, instead of the Producer emitting one copy of the Rule per instance. The glossary word for the
thing declared is **Rule**, not monitor: SEM-19 gives "Monitor" to the Run-local state machine an
Evaluator creates for a whole Contract, so this spec calls the declared machine a Rule and each
per-instance evaluation a **Rule instance**. `FormatVersion` stays `1.0` (fn-87's decision: the wire
has no compatibility promise and no Case outside the repository exists), and the change is additive,
so ART-04 needs no migration: a Rule that declares no instances is evaluated exactly as today.

Two rule texts need a draft under GOV-02 (R9), in the document's existing marker style (an indented
`*Amendment|Restatement (drafted by fn-89; awaiting GOV-02 approval.)*` line under the unchanged
original):

- **Glossary, Rule (Amendment, since it only adds).** Add: a Rule MAY declare typed instance values and a list of Rule instances,
  each assigning every instance value and naming its own rule ID; each Rule instance is evaluated as
  its own state machine and concludes in its own `RuleVerdict`. Where a rule text says "rule" of a
  conclusion, a Deadline counter or support (EVD-12, EVD-13, EVD-21, Verdict), it means one Rule
  instance, and a Rule with no instances is its own single instance.
- **SEM-17 (Restatement).** "bounded captures are rule-local and Run-local" becomes "local to one Rule instance
  and Run-local".

ART-09, ART-13, SEM-16 and EVD-18 are unaffected: the Contract stays closed and producer-neutral,
its bounds unchanged, and the six conformance classes unchanged (a new rejection sub-entry is
allowed by the extension checklist). Model syntax, `instances:`, Search, fn-88's Property monitor
lowering and the correlated contract are untouched.

## Goal & Context
<!-- scope: business -->

A Scenario over `instances: N` (fn-85) produces a Case whose Program runs N copies of the entity's
actions. For each field relation the path performs, the Producer lowers the relation once per
instance (`Placement`), so the Contract carries N Rules that differ only in a `-<n>` suffix on their
rule, capture and transition IDs and in the literal that selects the instance's own event.

Measured in the committed fixtures: of the 14 generated Case fixtures, one runs over more than one
instance, `nexusPairTests-bothComplete-case.json`. Its two Rules `relation-1` and `relation-2` are
134 pretty-printed lines (3,577 bytes) each; a `diff` of the two shows eight differing lines, seven
of them ID suffixes and one the literal `complete-1` / `complete-2`. The rest of the file is
1,369 lines (42,462 bytes), so the copy is ~8% of the fixture at N = 2 and grows linearly: fn-85's Syntax admits
up to nine instances, and the planned five-operation models would carry five copies per relation.

The duplication is not only size. Every copy is a separate proof obligation in the lowering, a
separate preparation (binding, capture analysis) in the runtime, and a separate place for a
Producer bug to make instance 2 diverge from instance 1 silently. The Model declares one relation
over one entity; the Case should say that once and bind what differs per instance.

Who benefits: model authors adding multi-instance Scenarios (fixtures stay readable and diffs show
one Rule), non-Lean Producers (SEM-18) who get a declared way to say "this Rule, for each of these
keys", and reviewers, who read one Rule and a table of instance values.

## Architecture & Data Models
<!-- scope: technical -->

```text
Model relation (one, over the entity) ──lower once──▶ Rule with instance values (typed, symbolic)
Scenario instances: N ──Placement per instance──▶ N Rule instances { rule_id, values }       Producer
                                   │
                      Case (FormatVersion 1.0, Contract.rules[i].instances)
                                   │
                 Prepare: bind the Rule ONCE; check every instance's values against the types
                                   │
                 Evaluator: one Run-local state per Rule instance, instance values in scope
                                   │
                 Verdict: one RuleVerdict per Rule instance, IDs and order as today
```

- **Rule with instances (protocol).** A `ContractRule` gains declared instance values (ID and
  scalar or enum type) and a list of Rule instances. A predicate reads an instance value through a
  new `Reference` arm, admitted only in the Contract transition-predicate context. A Rule with no
  instance values and no instances is a plain Rule, evaluated once under its own `rule_id`.
- **Preparation (Go, verification).** Binds the Rule's states, captures, transitions and predicates
  once with every instance value typed by its declaration and always available (like a literal, an
  instance value is never absent), then checks each instance's assignments. Every use of an
  instance value must sit where its declared type is the expected type, so the expansion (each
  value inlined as a literal of its declared type) binds the same way. Instance values are text,
  integer or enum typed; a boolean is rejected, because capture analysis prunes paths on boolean
  literals and a Rule analyzed once cannot prune per instance. Capture analysis runs once.
  **Admission equals the expansion's:** every Contract ceiling (rule count, states, transitions,
  captures, binding work, per-event and total work) is charged per Rule instance as the expanded
  copy would charge it, with each instance value reference charged what the inlined literal costs
  (its literal check and value surface in binding, its size in expression work). The work is done
  once; only the accounting is multiplied. So a Contract that passes the instance checks is
  admitted exactly when its expansion is, rejecting on the same ceiling, and no instanced Contract
  is admitted whose expansion would be rejected.
- **Evaluator (Go, verification).** One `ruleState` per Rule instance (state, captures, Deadline
  counter, support), each reading its own instance values; runtime work charges an instance value
  reference what the inlined literal would cost, so remaining per-event budgets match the
  expansion's. Rule instances are evaluated in Rule
  declaration order, then instance declaration order (never ID order), which is the order the
  expanded copies had. Everything the Evaluator names by rule ID (the `RuleVerdict`, transition
  traces, the Executor-stop `Violation`, the all-satisfied count, the offset where correlated
  verdicts start) names and counts Rule instances. Online `Observe` and offline `Evaluate` share the
  path (SEM-17 unchanged).
- **Producer (Lean, `Umpire.Case.Producer` and `Umpire.Case.Projection`).** Lowering stays per
  placement: each `Placement` is lowered as today to a `DerivedRule` with its own certificate. Over
  N > 1 instances the Producer then folds a relation's N derivations into one instanced Rule:
  the N shapes must agree once their literal is erased (same shape kind, negation, reads, capture
  state names, and literal type), else `relation.instance-shape`. The one literal the shape's
  predicate compares against (a request literal in the safety shape, the capture selector's value
  in the capture shape) becomes the Rule's single instance value, named by the last segment of that
  literal's field path (`operation` in the pair Case; the safety shape keeps its literal's path for
  this) and typed by the field's schema type; a boolean literal is not folded, and its relation
  keeps today's per-placement plain Rules; each placement contributes one Rule
  instance whose rule ID is the Rule's ID plus `Placement.suffix` and whose value is that
  placement's literal. The Rule renders once with the unsuffixed rule suffix, so its capture and
  transition IDs carry no instance suffix. The certificate of the folded Rule is the list of the N
  per-placement `DerivedRule`s plus the shape agreement, so `literals_assigned` holds per instance
  by construction: every value an instance assigns is one that instance's realization assigns.
  Request coverage (`coverage.inputs`) stays one mapping per placement, unchanged. Over one instance
  the Producer emits today's plain Rule unchanged.
- **Tools.** `umpire-assess`'s rule-set check derives the expected Verdict rule IDs from Rule
  instances; nothing else in `tools/` reads Contract rules.

## API Contracts
<!-- scope: technical -->

Additions to `contract.proto` and `expression.proto`. Field names are the contract; numbers are
appended densely per the extension checklist.

```proto
message ContractRule {
  // ... fields 1-7 unchanged ...
  // Values each Rule instance assigns; a predicate reads one through Reference.instance_value_id.
  repeated ContractInstanceValue instance_values = 8;
  // Empty: the Rule is evaluated once under rule_id. Otherwise once per instance, never under rule_id.
  repeated ContractRuleInstance instances = 9;
}

// ContractInstanceValue declares one value every Rule instance of its Rule assigns.
message ContractInstanceValue {
  string instance_value_id = 1;
  // A scalar or enum type.
  SingularType type = 2;
}

// ContractRuleInstance is one evaluation of its Rule, concluding under its own rule ID.
message ContractRuleInstance {
  // Unique among every rule ID the Contract's Verdict can name.
  string rule_id = 1;
  // Exactly one assignment per declared instance value, in declaration order.
  repeated ContractInstanceAssignment assignments = 2;
}

// ContractInstanceAssignment gives one instance value its value for one Rule instance.
message ContractInstanceAssignment { string instance_value_id = 1; Value value = 2; }

message Reference {
  oneof reference {
    // ... arms 1-12 unchanged ...
    // A value the evaluated Rule instance assigns; admitted only in a Contract transition predicate.
    string instance_value_id = 13;
  }
}
```

The typed Nexus pair Case's Contract carries one Rule `relation` with instance value `operation`
(text) and instances `relation-1` = `complete-1`, `relation-2` = `complete-2`; its capture is
`nexusOperationScheduled-relation`, its transitions `capture-nexusOperationScheduled-relation` and
`match-nexusOperationCompleted-relation`. `Verdict` and `RuleVerdict` are unchanged. `Testpilot.Authoring`
gains the constructors for the three messages and the reference arm.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Verdict identity.** For every Case and Run, the Verdict of the instanced Contract is
  byte-identical to the Verdict of its expansion (each Rule instance written as a plain Rule with the
  instance values inlined as literals and the capture and transition IDs suffixed). Rule-ID set,
  order, statuses, terminal states and supporting sequences all match. The pair Case's capture
  shape has no violated state (a crossed completion leaves it pending, the Known Gap
  `crossed-completion-is-inconclusive`); a violated outcome is exercised with a safety-shaped Rule
  whose reject transition compares against the instance value.
- **Byte stability.** Every committed Case fixture other than `nexusPairTests-bothComplete-case.json`
  (the 13 other functional fixtures, the canary's `nexusCallerCanary-syncCompletion-case.json`, and
  the conformance corpus) regenerates byte-identical. The pair fixture changes only in
  `contract.rules` and the provenance `localNames` rows for the Rule; its Program, definitions,
  sources, gaps and claims are byte-identical. Model goldens (`umpire-check-goldens`) carry no
  Contract and do not change.
- **Local names.** The Rule's ID maps to `<property>.relation`; each Rule instance's rule ID keeps
  its row (`relation-1` → `<property>.relation-1`) so a Verdict's rule IDs stay resolvable through
  provenance. The capture and transition IDs lose their suffix.
- **Admission cost.** Every ceiling, binding work included, is charged per Rule instance at the
  inlined literal's cost, matching the expansion; the rule-count ceiling is checked against the total instance count before any
  per-instance state is allocated, so a Case declaring many instances rejects before it costs
  memory.
- **Names.** Every Rule ID, Rule-instance rule ID and correlated rule ID is distinct: an instance
  may not reuse its own Rule's ID, another Rule's, another instance's, or a correlated rule's. The
  Rule's own ID names no Verdict entry but stays reserved, since provenance maps it.
- **Locations.** Preparation diagnostics follow today's grammar, which names rules by ID:
  `contract.rules[<rule_id>].instance_values[<instance_value_id>]` and
  `contract.rules[<rule_id>].instances[<instance rule_id>].assignments[<instance_value_id>]`; a
  failure specific to one instance names that instance. An instance value reference that is empty
  or names an undeclared value is located at the transition predicate that holds it, validated
  before binding, since the shared expression binder reports unlocated scope errors.
- **Degenerate but admitted.** A Rule with exactly one instance, and a declared instance value no
  predicate reads, are admitted: each is equivalent to its expansion. The Lean Producer emits
  neither.
- **Instance count.** fn-85's Syntax caps `instances:` at nine; the runtime imposes no instance
  count beyond the rule-count ceiling and orders by declaration, so `relation-10` after
  `relation-9` is no special case.
- **Coverage.** Request coverage is per placement and unchanged; only the Contract shares.
- **Deadlines.** Each Rule instance has its own `rule_events` counter, reset, stopped and frozen per
  EVD-21; one instance's transition does not reset another's.
- **Executor stop.** A safety violation of any Rule instance stops the Run (EVD-14) exactly as the
  corresponding copy did.
- **Out-of-repo Cases.** None exist (fn-87); a pre-change binary rejects the new fields as unknown,
  which is ART-04's required behavior.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The protocol adds `ContractRule.instance_values`, `ContractRule.instances`,
  `ContractInstanceValue`, `ContractRuleInstance`, `ContractInstanceAssignment` and
  `Reference.instance_value_id` as shown, and `Testpilot.Authoring` gains their constructors;
  `make proto`, `make umpire-check-testpilot-protocol` and `make umpire-check-testpilot-authoring`
  pass and every new message carries a leading comment. Errors: no error surface beyond the protocol
  tests (`TestProtocolMessagesCarryLeadingComments`, the Case import closure test) and the Authoring
  tests.
- **R2:** Preparation binds a Rule with instances once and admits it when every instance assigns
  every declared value exactly once, in declaration order, with a value of its declared type.
  Errors, each rejected before Driver I/O with a location in today's by-ID grammar
  (`contract.rules[<rule_id>]…`, naming the instance when the failure is one instance's): instance
  values declared with no instances; instances with no instance values; an instance value ID empty,
  invalid or duplicated; an instance value type unset, boolean, or other than scalar or enum; an
  instance value used where its declared type is not the expected type; an instance
  omitting, repeating, reordering or naming an undeclared value; an assignment value unset or of the
  wrong type, including an enum value its enum does not define; an empty, invalid or duplicate
  Rule-instance rule ID, including one equal to its own Rule's ID, another Rule's ID, another
  instance's, or a correlated rule's; `instance_value_id` empty, naming an undeclared value, or used
  outside a Contract transition predicate. A Rule with one instance, or with a declared value no
  predicate reads, is admitted.
- **R3:** The Evaluator keeps one Run-local state per Rule instance and produces one `RuleVerdict`
  per instance, in Rule declaration then instance declaration order; the Executor-stop `Violation`,
  transition traces, the all-satisfied count and the correlated verdicts' position count Rule
  instances; online `Observe` and offline `Evaluate` answer identically. A Contract that passes R2 is
  admitted exactly when its expansion is, except that the authored Contract (and its Case) is also
  held, as written, to the surface and byte ceilings that bound untrusted input before it is walked,
  so declarations or assignments no predicate reads can reject a Contract whose expansion, which
  drops them, is admitted. Errors: every ceiling (rules, states, transitions,
  captures, binding, per-event and total work) is charged per instance with each instance value
  reference charged the inlined literal's cost, and a Case over one rejects at preparation naming
  the same ceiling its expansion would; the rule-count ceiling is checked before per-instance
  allocation; runtime work and capture ceilings fail as today.
- **R4:** A differential Go test proves Verdict identity: for Contracts with instances and their
  test-built expansion (each Rule instance a plain Rule with its values inlined as literals of their
  declared type), over Runs covering satisfied, violated (a safety-shaped Rule keyed on the instance
  value), inconclusive (the pair capture shape under a crossed completion), incomplete and
  deadline-expired outcomes, an event carrying no instance's value and an event matching no instance, the two
  Verdicts are byte-identical online and offline, and the two admissions agree. The cases include a
  plain Rule beside an instanced one, a one-instance Rule, a correlated rule after the instances,
  and ceiling cases where both admissions reject on the same ceiling.
  Errors: any differing Verdict fails naming the Run and the first differing rule ID (or the
  differing rule-ID sets); a mutation check (two instances' values swapped) must make the test
  fail, so the harness is not vacuous.
- **R5:** The Producer emits one Rule per relation per Case for a Scenario over N > 1 instances,
  with one instance value (the literal the rule's predicate compares against, named by the last
  segment of its field path and typed by its schema type) and N Rule instances whose rule IDs are the Rule ID plus
  `Placement.suffix` and whose values are each placement's literal; capture and transition IDs carry
  no instance suffix, request coverage stays per placement, and over one instance the Producer emits
  today's plain Rule, as it does for a relation whose compared literal is boolean. The folded Rule's certificate carries each placement's `DerivedRule`, so every
  assigned instance value is one its instance's realization assigns. Errors: per-placement
  derivations whose shapes disagree once the literal is erased (kind, negation, reads, capture state
  names, literal type) reject as `relation.instance-shape`; a missing per-instance literal rejects
  under the existing `relation.capture-literal-unassigned` / `relation.literal-unassigned` codes.
- **R6:** `nexusPairTests-bothComplete-case.json` regenerates to one Rule `relation` with instances
  `relation-1` and `relation-2`; every other Case fixture and conformance entry regenerates
  byte-identical; `make umpire-check-case-runtime-conformance` and `make canary-check-case` pass.
  Errors: a byte change in any other fixture, or outside `contract.rules` and the Rule's local-name
  rows of the pair fixture, is a failure to explain before the change lands.
- **R7:** `TestTestpilotNexusPairCase` passes unchanged in what it asserts of the Verdict (rule IDs
  `relation-1`, `relation-2`, statuses, per-instance correlated evidence), and the Pair model tests
  and fixture test assert one Rule with two instances. Errors: no error surface beyond the tests.
- **R8:** `umpire-assess` derives the Contract's expected rule IDs from Rule instances (plain Rules
  and correlated rules as today). Errors: a Verdict naming the Rule's own ID instead of its
  instances, or missing an instance, is reported by the existing rule-set check.
- **R9:** The Rule glossary Amendment and the SEM-17 Restatement are drafted in `.plans/UMPIRE4_SPEC.md`,
  marked awaiting GOV-02 approval; `.plans/UMPIRE4_ORDER.md` records fn-89 as delivered at close
  (the carried-forward item already moved into the queue); the Testpilot README's extension
  checklist, the verification README, the Case Runtime design's Contract IR section and the
  Producer's typed-field-lowering architecture notes describe Rule instances. Errors: no error
  surface.
- **R10:** A `static-preparation-rejection/instance-value` conformance sub-entry pins one R2
  rejection a non-Lean Producer sees; `make umpire-check-regression`, `make lint-model`
  (`LEAN_NUM_THREADS=1`, no new findings over the baseline) and `make lint-code-fast` pass.
  Errors: no error surface beyond the gates.

## Boundaries
<!-- scope: business -->

- No dynamic instantiation keyed by an observed value (the correlated contract's operation model);
  instances are declared statically by the Producer.
- No change to the correlated contract, the per-role outage-order Rule, or the Program's
  per-instance entrypoints.
- No field path through a repeated field (the separate carried-forward item from fn-86 R2).
- No `FormatVersion` bump, no compatibility shim, no change to `Verdict` or `RuleVerdict`.
- No change to model syntax, `instances:`, Search, or fn-88's Property monitor lowering.
- No per-instance parameterization of anything but predicate operands (no per-instance states,
  transitions, Deadlines or kinds).
- No change to the Realization modules or request coverage; no Rule instances across entities
  (fn-92's later Producer spec owns that, and builds on this surface).
- No change to the Lean toolchain; see Dependencies.

## Dependencies & sequencing

- **fn-90** (open, finishing): its pair-test work closed as not reproduced with no Model, Producer,
  fixture or Contract change, and confirmed the pair test correlates evidence by rule ID and
  scheduled event ID, not position. This spec keeps both, so R7 holds unchanged. The remaining
  dependency is ordering the `UMPIRE4_ORDER.md` edits.
- **fn-88.12** may move the model toolchain from Lean 4.33.1 to 4.32.0. The Lean work here uses no
  4.33-only API and keeps its proofs structural (membership through the existing `decideMem`,
  `rw`/`subst`/`cases`, no `grind`, no `native_decide`), so it builds on either. A Lean task lands
  wholly on one toolchain; if the move lands mid-task, rebase and rebuild before the gates.
- **fn-93 A6** edits the Producer's raw-Property round-trip in the same file as R5's fold; the fold
  stays in its own declarations so the two merge textually. fn-93 A7 already waits for this spec,
  which leaves the Realization modules alone.
- **fn-92** depends on this spec only for the pair fixture's ordering. Its later cross-entity
  Producer spec composes with this surface: a Rule instance names its own rule ID and assigns
  declared values, with no assumption that instances come from one entity's `Placement`.

## Decision Context
<!-- scope: both — conditionally substructured -->

**Instances declared statically in the Case, bound once in preparation** over expanding copies in
Prepare (keeps the duplicated prepared state and synthetic error locations) and over runtime
discovery by observed key like the correlated contract (changes Verdict shape and rule-ID sets, and
the Producer already knows every instance).

**Each Rule instance names its own rule ID** over a runtime naming convention (`rule_id + "-" + n`):
the Verdict stays byte-identical and the runtime invents no names (SEM-16). It also keeps the
surface open to instances that are not numbered placements of one entity (fn-92).

**Plain Rule over one instance** over always emitting instances: mirrors `Placement.suffix` (no
suffix over one instance), keeps every single-instance fixture byte-identical, and a one-instance
Rule carries nothing to share.

**The compared literal becomes the instance value** over every request literal of the realization:
a rendered shape compares against exactly one literal (`Shape.literals`), which is the only thing
that differs between the per-placement rules; request literals the predicate never reads stay in
per-placement coverage, and declaring them would add values no predicate reads.

**Fold N certified derivations** over a new N-realization lowering: each placement keeps today's
`DerivedRule` certificate unchanged, and the folded Rule's obligation is the conjunction of theirs
plus a decidable shape agreement, so no existing proof is restated and the proofs stay
toolchain-neutral.

**Admission charged per instance, binding work included,** over charging binding once: admission
then equals the expansion's for every Case, so "no ceiling loosens" is true by construction rather
than by argument.

**No boolean instance values** over per-instance capture analysis: capture analysis prunes on
boolean literals, so a once-analyzed Rule could reject where its expansion is admitted; a boolean
compared literal keeps today's per-placement Rules, which no current Case uses over N > 1.

**Word choice.** The carried-forward title says "monitor"; SEM-19 reserves Monitor for the Run-local
Contract state machine, so the protocol and prose say Rule and Rule instance.

## Glossary Conflicts

- "Contract monitor" in the carried-forward item and this spec's title names a Rule, not a Monitor
  (UMPIRE4_SPEC Runtime concepts). Proposed title: "One Contract Rule per entity".

## Quick commands

```bash
go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/evaluation/...
make proto umpire-check-testpilot-protocol umpire-check-testpilot-authoring
make umpire-check-case-runtime-conformance canary-check-case
LEAN_NUM_THREADS=1 make lint-model
TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression
```

## Early proof point

Task fn-89-one-contract-rule-per-entity.3 validates the core approach: the differential test shows
an instanced Contract prepared once and evaluated per instance yields the expansion's Verdict and
admission byte for byte. If it fails, re-evaluate binding once (fall back to expanding in Prepare
behind the same wire format) before the Producer fold (.4), which depends on it, lands.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1  | Protocol messages, reference arm and Authoring constructors | fn-89-one-contract-rule-per-entity.1 | — |
| R2  | Preparation binds once, rejects malformed instances | fn-89-one-contract-rule-per-entity.2 | — |
| R3  | Per-instance evaluation, admission equals expansion | fn-89-one-contract-rule-per-entity.2, fn-89-one-contract-rule-per-entity.3 | — |
| R4  | Differential Verdict-identity test | fn-89-one-contract-rule-per-entity.3 | — |
| R5  | Producer folds per-placement rules into one instanced Rule | fn-89-one-contract-rule-per-entity.4 | — |
| R6  | Pair fixture regenerates; everything else byte-identical | fn-89-one-contract-rule-per-entity.4, fn-89-one-contract-rule-per-entity.5 | — |
| R7  | Pair tests assert one Rule, two instances; Verdict unchanged | fn-89-one-contract-rule-per-entity.4 | — |
| R8  | `umpire-assess` rule set from instances | fn-89-one-contract-rule-per-entity.5 | — |
| R9  | GOV-02 drafts and docs | fn-89-one-contract-rule-per-entity.6 | — |
| R10 | Conformance sub-entry and gates | fn-89-one-contract-rule-per-entity.5, fn-89-one-contract-rule-per-entity.6 | — |


