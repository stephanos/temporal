# fn-89-one-contract-rule-per-entity One Contract Rule per entity

## Umpire4 architecture reconciliation

This spec lets a Contract Rule be declared once for an entity and instantiated once per instance of
it, instead of the Producer emitting one copy of the Rule per instance. The glossary word for the
thing declared is **Rule**, not monitor: SEM-19 gives "Monitor" to the Run-local state machine an
Evaluator creates for a whole Contract, so this spec calls the declared machine a Rule and each
per-instance evaluation a **Rule instance**. `FormatVersion` stays `1.0` (fn-87's decision: the wire
has no compatibility promise and no Case outside the repository exists), and the change is additive,
so ART-04 needs no migration: a Rule that declares no instances is evaluated exactly as today.

Two rule texts need a restatement drafted under GOV-02 (R9):

- **Glossary, Rule.** Add: a Rule MAY declare typed instance values and a list of Rule instances,
  each assigning every instance value and naming its own rule ID; each Rule instance is evaluated as
  its own state machine and concludes in its own `RuleVerdict`. Where a rule text says "rule" of a
  conclusion, a Deadline counter or support (EVD-12, EVD-13, EVD-21, Verdict), it means one Rule
  instance, and a Rule with no instances is its own single instance.
- **SEM-17.** "bounded captures are rule-local and Run-local" becomes "local to one Rule instance
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
  once with every instance value typed, then checks each instance's assignments. Capture analysis
  runs once. Contract ceilings are charged per Rule instance, exactly as they would be for the
  expanded copies, so every Case admitted today is admitted after, and no ceiling loosens.
- **Evaluator (Go, verification).** One `ruleState` per Rule instance (state, captures, Deadline
  counter, support), each reading its own instance values. Rule instances are evaluated in Rule
  declaration order, then instance order, which is the order the expanded copies had. Online
  `Observe` and offline `Evaluate` share the path (SEM-17 unchanged).
- **Producer (Lean, `Umpire.Case.Producer` and `Umpire.Case.Projection.lower`).** For a Scenario
  over N > 1 instances, a relation lowers once with every realization-assigned literal (request
  literals and the capture selector's value) replaced by an instance value; each `Placement`
  contributes one Rule instance whose rule ID is the Rule's ID plus `Placement.suffix` and whose
  values are that placement's literals. The `DerivedRule` certificate's `literals_assigned`
  obligation becomes: every instance value any Rule instance assigns is one that instance's
  realization assigns. Over one instance the Producer emits today's plain Rule unchanged.
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
  order, statuses, terminal states and supporting sequences all match; a violated pairing
  (crossed operations, as `Temporal.Feature.Nexus.Pair.Tests` states it) stays violated.
- **Byte stability.** Every committed Case fixture other than `nexusPairTests-bothComplete-case.json`
  (the 13 other functional fixtures, the canary's `nexusCallerCanary-syncCompletion-case.json`, and
  the conformance corpus) regenerates byte-identical. The pair fixture changes only in
  `contract.rules` and the provenance `localNames` rows for the Rule; its Program, definitions,
  sources, gaps and claims are byte-identical. Model goldens (`umpire-check-goldens`) carry no
  Contract and do not change.
- **Local names.** The Rule's ID maps to `<property>.relation`; each Rule instance's rule ID keeps
  its row (`relation-1` → `<property>.relation-1`) so a Verdict's rule IDs stay resolvable through
  provenance. The capture and transition IDs lose their suffix.
- **Admission cost.** Ceilings are charged per Rule instance (rules, states, transitions, captures,
  per-event and total work), matching the expansion. Binding work is charged once.
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
  `Reference.instance_value_id` as shown; `make proto` and `make umpire-check-testpilot-protocol`
  pass and every new message carries a leading comment. Errors: no error surface beyond the protocol
  tests (`TestProtocolMessagesCarryLeadingComments`, the Case import closure test).
- **R2:** Preparation binds a Rule with instances once and admits it when every instance assigns
  every declared value exactly once with a value of its declared type. Errors, each rejected before
  Driver I/O with a location under `contract.rules[i]`: instance values declared with no instances;
  instances with no instance values; an instance value typed other than scalar or enum; a duplicate
  instance value ID; an instance omitting, repeating or naming an undeclared value; a value of the
  wrong type; an invalid or duplicate Rule-instance rule ID, including one equal to another Rule's
  ID, another instance's, or a correlated rule's; `instance_value_id` naming an undeclared value, or
  used outside a Contract transition predicate.
- **R3:** The Evaluator keeps one Run-local state per Rule instance and produces one `RuleVerdict`
  per instance, in Rule declaration then instance order; online `Observe` and offline `Evaluate`
  answer identically. Errors: ceilings are charged per instance and a Case over them rejects at
  preparation naming the ceiling; runtime work and capture ceilings fail as today.
- **R4:** A differential Go test proves Verdict identity: for Contracts with instances and their
  test-built expansion, over recorded Runs covering satisfied, violated (crossed pairing),
  inconclusive, incomplete and deadline-expired outcomes, the two Verdicts are byte-identical.
  Errors: any differing Verdict fails naming the Run and the first differing rule ID.
- **R5:** The Producer emits one Rule per relation per Case for a Scenario over N > 1 instances,
  with N Rule instances whose rule IDs are the Rule ID plus `Placement.suffix` and whose values are
  each placement's realization-assigned literals, and emits today's plain Rule over one instance.
  The `DerivedRule` certificate proves every assigned instance value is one its instance's
  realization assigns. Errors: per-instance lowerings that do not share one shape reject as
  `relation.instance-shape`; a missing per-instance literal rejects under the existing
  `relation.capture-literal-unassigned` / `relation.literal-unassigned` codes.
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
- **R9:** The Rule glossary and SEM-17 restatements are drafted in `.plans/UMPIRE4_SPEC.md`, marked
  awaiting GOV-02 approval; `.plans/UMPIRE4_ORDER.md` moves the carried-forward item into the queue
  and the Testpilot README's extension checklist and verification README describe Rule instances.
  Errors: no error surface.
- **R10:** A `static-preparation-rejection/instance-value` conformance sub-entry pins the R2
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

## Decision Context
<!-- scope: both — conditionally substructured -->

**Instances declared statically in the Case, bound once in preparation** over expanding copies in
Prepare (keeps the duplicated prepared state and synthetic error locations) and over runtime
discovery by observed key like the correlated contract (changes Verdict shape and rule-ID sets, and
the Producer already knows every instance).

**Each Rule instance names its own rule ID** over a runtime naming convention (`rule_id + "-" + n`):
the Verdict stays byte-identical and the runtime invents no names (SEM-16).

**Plain Rule over one instance** over always emitting instances: mirrors `Placement.suffix` (no
suffix over one instance), keeps every single-instance fixture byte-identical, and a one-instance
Rule carries nothing to share.

**Every realization-assigned literal becomes an instance value** over only the literals that
differ across instances: the realization is what varies per instance, and the `DerivedRule`
certificate already names exactly those literals, so the proof obligation generalizes directly.

**Ceilings charged per instance** over per declaration: evaluation cost is per instance, and it
keeps admission identical to the expansion, so no Case's admission changes.

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
