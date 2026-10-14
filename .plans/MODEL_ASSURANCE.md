---
status: draft
---

# Finding and closing gaps in Umpire Models

Design and implementation plan, 2026-10-03.

Extended the same day with a second industry and academic sweep. The
[research companion](MODEL_ASSURANCE_RESEARCH.md) records the academic mechanisms, evidence and
limits; the industry findings below turn first-party experience into pilot requirements. The
broader [industry survey](UMPIRE4_INSPIRE.md) remains the architecture background. Recommendations
here are Umpire design proposals, not results of experiments already run on this repository.

The [model refinement design](MODEL_REFINEMENT.md) records the product/interface starting point,
named request shapes and selectively deeper refinements. This assurance plan owns their proposed
exhaustiveness, mapping-mutation and promise-preservation checks; model depth remains independent
of environment access.

## Outcome

A feature developer should learn quickly when a Model omits a case, makes a promise vacuously,
admits forbidden behavior, excludes required behavior, or cannot observe the distinction its
Property needs. The finding should point to the declaration, show a small example, identify what
remains unknown, and lead to a regression that fails if the gap returns.

Start with constraints that prevent mistakes, then cheap semantic lint, then targeted mutation and
independent execution evidence. Mutation testing is useful for challenging existing promises; it
cannot discover every promise an author forgot to state. No single score establishes model quality.

The [vision](UMPIRE4_VISION.md) is the starting point, especially #DESIGN, #CONFORMANCE, #GUIDANCE,
#PROTOCOLS, #REPLAY and #AUTHORING. This plan preserves the [shared specification](UMPIRE4_SPEC.md),
the [module boundaries](UMPIRE_MODULES.md), and the distinct results in
[model semantics](../model/SEMANTICS.md). It proposes work; it changes neither today's semantics nor
the existing Flow specs. It introduces no exception to a shared architectural rule.

## The failures we need to distinguish

| Failure | Example in Umpire's domain | Best first defense |
| --- | --- | --- |
| Missing vocabulary or promise | Reset or heartbeat timeout is outside the activity Model; a newly added API field has no behavioral classification | An explicit feature boundary and a change-sensitive inventory |
| Missing transition case | A new control enum member falls through to `Nil`, silently making the request impossible | Exhaustive typed matches and reviewable disabled/rejected/unknown decisions |
| Model permits a bug | Admission trusts stale eligibility and starts a paused activity | Independent safety Properties, protocol laws and behavior mutations |
| Model excludes valid behavior | A guard removes every retry or every successful admission | Positive witnesses, progress checks and independent valid traces |
| Property says too little | An implication's antecedent never holds, or a completion claim checks only a recorded fact | Trigger coverage and clause sensitivity checks |
| Evidence says too little | An RPC response is treated as a durable commit, or two attempts share an identity | Evidence admission, ambiguous-trace tests and observation sensitivity |
| Tool interprets the Model incorrectly | A lift drops an alternative, or a checker ignores a monitor state | Hand-derived semantic fixtures, backend comparisons and tool tests |

An omission in this table is not automatically a `Known Gap`. Use that term only with the meaning
the shared specification assigns it. A quality finding, an unknown Model hole, an unmapped source,
and a runtime Known Gap keep their own identities and consequences.

## Existing work and ownership

The reader already checks finite domains, references, row validity, refinements, holes, progress
and bounded search. `Receipt.Exercised` already records whether a claim was read. The conformance
tests already cover missing evidence, source closure, crossed identities, clock skew and replay.
The activity admission and queue Models already include intentionally faulty designs. Reuse these
as the initial assurance corpus.

Several useful foundations are still planned, rather than implemented contracts to assume:

| Existing work | Responsibility this plan leaves there | What this plan adds |
| --- | --- | --- |
| [fn-112](../.flow/specs/fn-112-make-the-standalone-activity-scala.md) | Readable claim patterns, shared declarations and Query totals | Checks that constraints preserve discoverable failure states; initial-state invariant checks |
| [fn-114](../.flow/specs/fn-114-state-every-scala-model-declaration-once.md) | Declaration discovery and complete roots | Quality inventory over those roots, including deletions |
| [fn-118](../.flow/specs/fn-118-declare-how-temporal-apis-behave-once.md) | Shared API behavior and wait hints | Accounting for new API inputs and boundaries without inferring semantics from descriptors |
| [fn-120](../.flow/specs/fn-120-adopt-what-quint-does-well-named.md) | Named choices, cheap model lint, accepted findings, explorer, ITF and R15 counts with denominators | Antecedent coverage, sensitivity, semantic mutation and assurance reports through those surfaces |
| [fn-122](../.flow/specs/fn-122-capabilities-and-their-laws.md) | Capability laws, automatic interaction laws, explicit exceptions | Nonvacuous law bindings and mutation controls for the law families |
| [fn-123](../.flow/specs/fn-123-declare-faults-as-the-environments.md) | Typed faults, budgets, durability and realizable choices | Tests of the fault envelope, crash boundaries and recovery premises |
| [Modalities study](MODALITIES.md) | Proposed per-state/per-operation views and findings about permitted, required and prohibited behavior | Reuse those views for missing rejection cases, witness-only claims and loss of required behavior |

Do not build a second linter, explorer, capability catalog or suppression format. fn-120 explicitly
leaves mutation-based lint for later; this is that follow-up. Query `total` is a static combination
count, not evidence that a branch, promise or implementation schedule was exercised.
fn-123 and the modalities study are additional current planning inputs, not shipped capabilities.

## Design constraints that prevent gaps

### Keep the modeled boundary explicit

For the activity pilot, record which operations, input classes, deadlines, delivery behaviors and
observation sources are in scope. Bind entries to existing Definition IDs or to a located,
reasoned exclusion. Start by accounting for the reset and heartbeat-timeout exclusions already
written in `model/temporal/standaloneactivity/Model.scala`.

Derive candidates from the selected API descriptors, feature declarations and retained functional
assertions. Descriptors identify fields and methods, not their meaning. The feature owner decides
whether an item is modeled, represented by an existing abstraction, outside this Model's boundary,
or unsupported. Preserve that decision beside the feature and reject dangling or stale references.
A newly discovered field must become an unclassified item, never inherit an implicit exclusion.
Keep the inventory scoped to selected feature APIs so unrelated server changes do not flood it.

For each input abstraction, state the distinctions it retains and the ones it merges. Exercise
boundary representatives and at least one additional member of a non-singleton class when feasible.
A divergence splits the class and produces a regression. A representative example alone does not
establish the Abstraction Claim for the whole class.

### Make transition decisions exhaustive and meaningful

Use finite enums, typed references and compiler-checked pattern matches. Keep the existing
`-Werror` build. Add a located quality rule for catch-all patterns over closed behavioral enums,
including enum matches in Properties, evidence maps and realization helpers. An intentional
catch-all needs a scoped reason and an inventory of the values it currently covers; a new member
invalidates that acceptance. Do not ban catch-alls for genuinely open data.

Refuse Scala `ensuring` in lifted behavioral declarations with an error directing the author to a
Property or invariant. The lifter currently strips it, so a familiar-looking postcondition can
provide no check at all. Review `require` separately. It is a function precondition whose violation
is an evaluation error, not a product invariant, a modeled rejection or a way to hide invalid
successor states. Pin those distinctions with source and hand-authored IR fixtures.

Every declared state/action pair has one of the existing semantic dispositions: enabled results,
disabled, or an explicit/undeclared hole. A rejected external request is an enabled step with a
rejecting outcome. Disabled means the action cannot occur in the modeled world. Unknown behavior
must remain a hole or an unsupported boundary, never become disabled to make a check pass.

Generate the decision inventory from the IR and the explorer's branch explanations. Do not make
authors maintain a duplicate state/action matrix. Review changes that disable formerly reachable
actions, remove nondeterministic alternatives, narrow starts or input domains, or turn a rejected
request into an impossible action. Such changes can remove the only counterexample.

### Check invariants without assuming away violations

Distinguish representation constraints from product promises. A sum type may prevent meaningless
combinations. It must still represent a product failure the checker is meant to find. For example,
keep the admission Model's `Active.two` value even though `atMostOneActive` rejects it. Typing that
value out of the domain would make double admission impossible to discover.

An authored state invariant must hold at every declared start and every reachable successor.
Existing step Properties do not, by themselves, check an initial state with no outgoing steps.
Add the smallest explicit state-invariant declaration and IR representation needed to retain that
distinction. Go checks the initial predicate and successor predicate using the same interpreter;
do not manufacture a fake action or reuse `starts` as an assertion. Keep transition Properties for
relations such as terminal phase preservation. Test zero-step and multiple-start Models explicitly.

Neither invariants nor monitors filter transitions. Keep assumptions about dependencies and fairness
visible, and keep safety checks active when progress assumptions are unavailable. Pair important
safety claims with required successful witnesses and bounded progress obligations where applicable.
"Never admits anything" must not satisfy the full admission contract.

### Reuse laws without making the oracle circular

Adopt fn-122's laws and automatically selected capability interactions. Add local Properties for
the feature's differences and account for exceptions with the existing acceptance mechanism.
Verify that each binding can reach its law's precondition within its declared bounds. Check that
changing a capability binding cannot silently remove the law from the required inventory.

A Property may share vocabulary and state projections with a transition. It should not compute its
expected result by calling the very transition helper it judges. Report that dependency pattern for
review. During behavioral mutation, freeze the Property, monitor, law bindings, assumptions and
expected outcomes, including all their transitive helper functions. Shared callees are either
cloned onto the transition-only call path or excluded with a reason; mutating both sides is invalid.

## Lint and coverage that explain what was checked

Extend fn-120's findings and accepted-findings handling. Admission errors remain admission errors;
lint never converts malformed input into a partial quality report that looks successful.

| Check | Evidence and action |
| --- | --- |
| Unaccounted declaration or API change | Report the source item and feature boundary entry it lacks |
| Missing or catch-all enum case | Locate the match and list affected values; include a new-member refusal fixture |
| Unasked or unexercised Property | Use fn-120's inventory and `Receipt.Exercised`; name the Query and its bounds |
| Vacuous antecedent | For named implication/pattern laws, record whether the antecedent actually held, with a witness; evaluating `!p || q` alone is insufficient |
| Unreachable guard or named alternative | Reuse reader reachability and fn-120 branch data; bounded absence is reported with its bound |
| Unchecked initial state | Name the missing or failing invariant check and the start value |
| Missing required behavior | Replay named positive witnesses and report a no-longer-executable path at its first disabled step |
| Lost law, observation or realization | Compare complete inventories in both directions, including removed roots, bindings and accepted findings |
| Shared transition/oracle computation | Report the common callee and the Property it can make self-confirming |
| Changed abstraction or assumption | Explain which Queries and claims rely on it and need renewed evidence |

For arbitrary Boolean Properties, start with located branch observations and targeted predicate
mutations. Do not claim general logical vacuity analysis. A trigger unreachable within a depth
bound is different from an unreachable trigger established over a completed finite exploration.
An exhausted resource budget establishes neither.

Report coverage as a chain of separate counts with named missing items: declared obligations,
reachable triggers, explored transitions/alternatives, executable Cases, targets observed in Runs,
and Properties conclusively assessed. A passing Contract cannot fill the last column. Include
unrealizable Cases and ambiguous assessments in the denominator of their own stage.

## Targeted mutation testing

### Start with the IR and a small defect catalog

Mutate one located expression or declaration in a cloned, admitted IR Model, then use the existing
reader and checker. This avoids recompiling Scala for every behavioral mutant and preserves the
source mapping. Source/lifter mistakes remain covered by compile/lift fixtures. A first prototype
needs no new third-party mutation library.

The pilot catalogs concrete defects in activity admission and queue custody:

| Mutation family | Deliberate fault | Expected detector |
| --- | --- | --- |
| Guard weakening | Admit a delivery after pause or terminal closure | `notAdmittedWhilePaused`, terminal finality and their existing witnesses |
| State update | Fail to increment/decrement active attempts, reopen terminal phase | Active-attempt and terminal invariants/monitors |
| Outcome/fact | Report accepted after rejection, omit required status or admission fact | Independent postcondition and evidence checks |
| Custody/acknowledgment | Lose retained work on crash or acknowledge before retaining it | Queue Properties and provider refinement checks |
| Boundary | Change a deadline/retry comparison at the finite boundary | Boundary-class witness and bounded progress check |
| Behavior deletion | Remove successful admission, retry, or a required start/alternative | Positive witness or progress obligation, even if every safety Property still holds |

Keep new mutants separate from today's intentional faulty Models. First demonstrate that the
existing `staleAdmission`, `forgetfulQueue` and `volatileQueue` controls fail for their documented
reasons. Baseline failures must be accounted for before any new mutation is scored.

### Keep four experiments separate

1. **Behavior mutations test the promises.** Change transitions with Properties, assumptions,
   Scenarios, limits and expectations frozen. A replayed counterexample or failed required positive
   obligation detects the mutant. A changed fingerprint or golden alone does not.
2. **Property mutations test the strength of the claims and the assurance suite.** Delete a clause,
   change a guard or replace a predicate with `true`. Check which curated bad behaviors cease to be
   rejected. A weaker Property passing the good Model is expected and is not a detection. If every
   result stays the same, report a candidate redundant/untested clause, not proof of equivalence.
3. **Evidence mutations test the observation boundary.** Remove a commit record, cross operation or
   attempt identities, break source closure, and alter mappings. Run these against frozen good and
   bad records. Missing evidence may correctly make a judgment inconclusive; it must never establish
   the original success. A malformed record rejected on admission tests that boundary only.
4. **Tool defects test the engine.** Keep small hand-derived IR/trace fixtures with known answers and
   existing backend comparisons. Later source mutation of the checker/lifter is optional. Mutating
   a Model and asking several engines to agree still cannot validate an omitted product requirement.

Add metamorphic checks only with explicit semantic preconditions. Changing unrelated source clock
values must preserve a causally ordered assessment. Bijectively renaming identities in an admitted
generic fixture must preserve its verdict. Removing relevant evidence must not manufacture a
positive conclusion. Reordering causally independent events is checked only where independence is
declared. Compare judgments and supporting relationships, not identities that legitimately change.

### Make the result useful to an author

For each mutant retain the base and mutant digests, operator/version, affected Definition ID,
expression path, source position, frozen oracle identities, exact scope and limits, and a witness
or explanation. Track whether its site was reached, whether behavior changed there, and whether a
claim or observation distinguished that change. These distinguish a missing Scenario from a weak
Property or missing observation.

Use separate outcomes for behavioral detection, structural rejection, survived, unreached,
equivalent within a completed declared scope, inconclusive/limit, tool error, and not run. A
timeout is inconclusive. A crash, validation error or unsupported mutant is never a behavioral
kill. Exact table equality can justify scoped equivalence for transition-only changes with identical
domains and frozen claims; equal sampled verdicts cannot. Keep broader equivalence a reviewed claim.

Start with deterministic operator/site ordering and fixed budgets. Cache only exact base/oracle,
operator, tool-version and scope identities. Dependency-based selection includes shared helpers,
laws, starts, domains and assumptions; uncertain dependencies fall back to the full pilot slice.
Selected-out work is visible. Never silently increase an author's search bounds.

Require every curated must-detect defect to have its named detector and a replayable witness. Report
raw counts by family before adding any score. Exploratory survivors become review findings; do not
immediately block every Model change on arbitrary generated mutants. Redundant or equivalent mutants
can cost time without adding confidence, as [PIT's operator discussion](https://blog.pitest.org/less-is-more/)
illustrates. [Stryker's status taxonomy](https://stryker-mutator.io/docs/mutation-testing-elements/mutant-states-and-metrics/)
is useful prior art, but its counting of timeouts as detected is inappropriate for Umpire's bounded
proof claims.

## Independent evidence and closing the gap

Use an independently authored producer-neutral workload at the held admission boundary, supported
by the existing Driver, to challenge the Model's preferred paths. Vary pause/delivery order, lost
acknowledgments and redelivery without deriving the schedule from its selected witness. Keep the
product assertions in the Model; the workload adds requests and records, not a second feature oracle.
Exercise a deliberately faulty implementation or component test double whose decision logic is
independent of the mutant Model. A scripted Driver returning a Model's expected result does not
meet this acceptance criterion.

Maintain two independent trace classes: reviewed legal behavior that the Model must accept and
reviewed faulty behavior it must reject or classify as evidentially inconclusive. Record the
product rationale for those labels. An implementation trace alone does not establish intended
behavior. Compare the unmodified and faulty implementations under the same Property and sufficient
evidence. Cross-operation execution remains gated by the existing one-operation/multiple-activity
limits; crossed-identity synthetic records can test the checker earlier.

Each actionable report should support this loop:

1. Reproduce offline from exact Model/Case/Run identities and show the first differing step.
2. Classify the defect as a Model transition, missing Property, abstraction, realization/evidence,
   tooling, or implementation problem. Do not automatically weaken the Model to fit the trace.
3. Reduce while preserving the failure key, triggering condition, mutation site and valid scope.
   Use checker witness replay for design failures and the existing two-fresh-Run reduction process
   for execution failures. A failure that cannot yet be realized stays a model regression.
4. Fix the owning declaration and rerun the unmodified good behavior and the deliberate bad control.
   For a transition fix, preserve the Property. For a missing/incorrect Property, review the intended
   promise explicitly rather than freezing a known-wrong assertion.
5. Retain a named regression and the relevant negative control. Regenerating goldens or accepting a
   finding does not close a behavioral defect. Runtime promotion retains its existing review gate.

Findings use source ownership and stable definition references. An acceptance is exact and reasoned,
is invalidated when its behavioral subject changes, and fails when stale. Reuse fn-120's acceptance
file. A quality acceptance does not suppress a violation, alter a Verdict, or implement the separate
Known Bug lifecycle envisioned by #KNOWNBUGS.

## Industry lessons with concrete consequences

### ShardStore found errors in its reference models too

Amazon's ShardStore work separates functional correctness, crash consistency and concurrency,
using small executable reference models. Its issue table includes an incorrectly updated reference
model after a crash and a model that reused locators another component expected to be unique.
It also describes a missed cache-path bug caused by tests always configuring a large cache.
These are concrete examples of model defects and generator omissions, even with substantial
validation. [Bornholt et al., SOSP 2021, sections 3, 8.3 and figure 5](https://cdn.amazon.science/77/5e/4a7c238f4ce890efdc325df83263/using-lightweight-formal-methods-to-validate-a-key-value-storage-node-in-amazon-s3-2.pdf).

For Umpire, make the pilot's defect ledger cover both sides of correspondence. Include a wrong
Model transition, a wrong implementation transition, a wrong identity assumption, and a missing
input condition. Test ownership/custody invariants across component interfaces, not just inside
one machine. Add tiny-but-valid queue capacity and retry-bound configurations so capacity failures
are reachable. These are declared configurations, never unrecorded Profile behavior.

Acceptance requires the same diagnostics to locate a Model defect and an implementation defect
without presuming either is authoritative merely because it is executable.

### FoundationDB measures the condition that mattered

FoundationDB's simulation work uses conditional coverage to count runs reaching specific
conditions. Its buggification makes unusual but contract-permitted behavior common, varies subsets
of fault sites, and deliberately avoids fault rates that trap runs in a small space. The paper
also reports defects caused by assuming a stronger operating-system contract than reality supplied.
[Zhou et al., SIGMOD 2021, section 4](https://www.foundationdb.org/files/fdb-paper.pdf).

For Umpire, report the sequence `dispatch pending -> pause committed -> old delivery admitted or
rejected`, including the evidence supporting each stage. Counting a pause and a delivery somewhere
in a Run is insufficient. For lost admission responses, separately count fault requested, fault
realized, durable commit observed, retry observed, and obligation assessed.

Use fn-123's explicit fault budgets. Exercise no-fault, one-fault and recovery phases, then selected
combinations. Preserve safety while recovery premises are false. Test provider assumptions against
real boundaries where possible; label untested premises. Acceptance includes a campaign that tries
many faults but never reaches its intended trigger and correctly reports zero realized coverage.

### TigerBeetle shows why more fuzzers can share one blind spot

TigerBeetle's first-party report describes a query defect missed by multiple fuzzers because related
fields were generated together. Matching objects occupied convenient aligned index regions, so
the failing intersection behavior never arose. An independently generated workload exposed it;
less constrained inputs and a more precise reference oracle made reproduction straightforward.
[TigerBeetle, Fuzzer Blind Spots, 2025](https://tigerbeetle.com/blog/2025-06-06-fuzzer-blind-spots-meet-jepsen/).

For Umpire, audit relationships between generated values, not only each value's marginal coverage.
The same activity ID across distinct runs, the same attempt number across distinct activities,
duplicate deliveries within one attempt, and reordered observations are different relationships.
Generate them independently where the API permits them. Do not enforce convenient uniqueness that
the real boundary does not guarantee.

Start with a deliberately crossed-identity evidence control, then run independent workloads when
multi-operation realization supports them. Acceptance requires a defect missed by the original
correlated generator and reached by the challenger under the same budget. Report each generator's
observed relationship classes; do not claim statistical independence just because two generators
have different code.

### Sieve turns observation timing into a test dimension

Sieve perturbs a controller's view into intermediate, stale and unobserved states. It uses precise
fault timing and compares both final states and summaries of updates. Its authors also describe
masking fields learned to vary between reference runs. Its tested space is derived from workloads
and perturbation patterns, not every possible implementation behavior.
[Sun et al., Sieve authors' account, sections 2.1 and 2.2](https://www.usenix.org/publications/loginonline/sieve-chaos-testing-kubernetes-controllers).

For Umpire, challenge three cuts separately: commit before response, old eligibility after pause,
and a transient status that a poll never sees. A correct final status cannot erase an illegal
intermediate admission. A missed transient read does not prove the transition never happened.
Add corresponding record pairs to evidence assurance before requiring live actuators for all three.

Adopt the perturbation categories, but require declared semantic projections instead of learning
ignore lists from passing runs. ART-11 forbids generic normalization. Acceptance includes two Runs
with the same final status and different intermediate safety outcomes, and a varying operation ID
that must still be checked rather than masked.

### CHESS makes atomicity assumptions visible

CHESS controls scheduling and replays recorded schedules. It prioritizes executions with few
preemptions; its paper explicitly notes that suppressing preemptions in selected modules gains
scale while potentially missing interactions across those boundaries.
[Musuvathi et al., OSDI 2008, sections 4.3 and 4.4](https://static.usenix.org/events/osdi08/tech/full_papers/musuvathi/musuvathi_html/index.html).

For Umpire, record an atomicity assumption for each coarse action relevant to a promise. Admission
may need distinct eligibility-read, durable-commit and response boundaries; a single atomic
`attemptStart` cannot reveal every ordering between them. Refine just that seam and compare it
with the coarse view. Implement controlled scheduling only where the real component exposes the
required boundaries. A Model schedule is not evidence of a controlled implementation schedule.

Acceptance includes one failure hidden by a deliberately coarse action and exposed after adding
the missing cut, with the same product Property. Preserve its causal schedule, source revision and
dependency assumptions. A new scheduler or a whole-Temporal simulator is outside this plan.

### Semantic diffs should explain model edits

ADDiff compares activity diagrams through execution traces admitted by one diagram but not the
other. This is a useful precedent for presenting a behavioral difference as a witness rather than
only a syntax diff. Its algorithm and modeling language are not Umpire's.
[Maoz, Ringert and Rumpe, ADDiff](https://arxiv.org/abs/1409.2352).

For Umpire, compare before/after Models in both directions over an explicitly matched vocabulary
and finite scope. Show one newly permitted trace and one removed trace where either exists, plus
changed outcomes, facts and obligations. Compare complete result sets for nondeterministic steps,
not just the first result or the witness one search happened to select. If identities or domains
cannot be matched, report that boundary instead of inventing a normalization.

Start by replaying required witnesses and showing changed reachable rows; bounded trace-language
comparison is a later extension of the same report. A row delta is not automatically a reachable
behavior delta. Acceptance removes one retry path while preserving all existing safety answers;
the report must still explain the removed behavior. A doc-only edit produces no behavioral delta.

## Research-driven assurance experiments

The [academic companion](MODEL_ASSURANCE_RESEARCH.md) provides the sources and precise boundaries
for the following experiments. These add evidence about the adequacy of the specification, not a
claim that its requirements are complete.

| Experiment | Pilot question | Required diagnostic |
| --- | --- | --- |
| Satisfiability and positive examples | Can admission actually succeed, and can each important obligation activate? | A valid witness or a scoped explanation; resource exhaustion is unresolved |
| Vacuity and clause sensitivity | Did the paused antecedent occur, and could the forbidden admission change the result? | Separate predicate evaluation, antecedent activation and discrimination evidence |
| Overconstraint diagnosis | Did a guard, assumption, input partition or Scenario remove all interesting behavior? | The first blocking condition and a proposed relaxation to inspect, never an automatic semantic edit |
| Observational distinction | Can the current observation set distinguish a legal admission from a forbidden one? | Two candidate executions with opposite Property results and the same evidence, or a completed scoped check |
| Deferred observation | Does a triggered promise get an observation belonging to the right operation and version? | Pending, observed, superseded or inconclusive coverage credit; issuing a read is not coverage |
| Mutation discrimination | Is a surviving mutant a different allowed choice or a defect our tests should expose? | Difference in allowed observable behavior, with nondeterminism and oracle identity preserved |
| Component compatibility | Can each receiver account for reachable provider outputs, and is progress checked for the right operation? | Located unmatched communication or missing operation-specific progress evidence |
| Structure-preserving reduction | Does a shorter example retain identities, prerequisites and the reason for failure? | Valid minimized example, rejected edits and the completed reduction budget |

### Test both the presence and the force of a claim

For a named conditional law `P implies Q`, retain evidence that the declaration exists, some check
selected it, `P` held, and the check rejected a deliberate `P and not Q` example. Also keep legal
examples with `P and Q` and, where possible, with `not P`. This catches both a law weakened to
`true` and a law strengthened accidentally into unconditional `Q`. These are distinct assurance
checks; ordinary code branch coverage supplies none of the product rationale.

For conjunctions, report discrimination per clause. A different failing conjunct must not hide the
mutated clause. If no valid example separates them, record possible redundancy or bounded
indistinguishability. Do not demand impossible truth-table combinations or claim full logical
vacuity analysis for arbitrary expressions. Keep counterfactual faulty behavior separate from legal
Model paths; a sound Model should prevent some of the deliberately bad examples.

### Define detection for nondeterministic Models

A mutant is not faulty merely because its chosen example differs from the original search's chosen
example. Compare allowed observable behavior under matching inputs and assumptions. If both
outcomes are permitted by the unchanged contract, either may occur. New forbidden behavior needs
a violating execution; missing behavior needs an independently declared requirement to preserve
that behavior. Removing an optional alternative can be a valid refinement.

Keep three result columns: structural validity, behavioral difference and violated obligation.
For example, an admission mutant can be structurally valid, change an internal bookkeeping field,
and still satisfy every declared observable promise. That is a survivor to investigate, not proof
of a missing assertion. Conversely, reducing all behavior to a safe deadlock may preserve safety
while violating an enabledness or progress obligation. Include both cases in the mutation runner's
own tests. The academic companion's mutation and conformance sections explain the underlying
distinction between a changed model and an observable contract violation.

### Challenge overconstraint through reviewed diagnostics

Start with a feasible unmodified pilot and retain witnesses for its required behaviors. When a
witness disappears, report where its path first becomes disabled and which changed declarations
contributed. In a bounded diagnostic copy, relax one candidate guard or Scenario restriction and
see whether the witness returns. The diagnostic never changes a normative Model, its assumptions,
Case generation or its verdict.

A solver-backed unsatisfiable core is optional later work. The existing explicit-state checker
does not produce one. A single blocking path is not a proof that no path exists; a bounded
no-instance result does not prove global inconsistency. If a reduction of constraints is attempted,
call it minimal only relative to the completed reduction procedure and its scope.

### Find observation gaps before spending a live Run

For a selected Property and bounded Model slice, search for two executions whose declared
evidence projections agree while their Property outcomes differ. Include operation/attempt identity,
commit status, source closure and causal ordering in the projection. Their pair explains why more
Runs with the same observations cannot settle that claim. Show the first hidden distinction and
which additional evidence would separate this particular pair, without claiming it resolves every
ambiguity.

This is a proposed bounded analysis in the conformance layer, which owns observation meaning.
It must reuse its candidate-execution semantics. Equal serialized records or a hand-built
projection in the assurance command would create a competing interpretation. An unsupported
projection or incomplete paired search stays unresolved. The first pilot uses synthetic records;
live claims wait for actual collection of the separating observation.

### Track observation credit, not just executed actions

Separate a requirement's applicability from the conditions under which an execution tests it.
A successful pause request can activate a requirement without a later observation establishing its
effect. Record pending coverage credit against the operation, attempt and relevant state version;
discharge it only with an observation adequate for that requirement. A subsequent unpause can
supersede the opportunity to observe the paused state. Mark that credit superseded, not covered.
This cancels a coverage opportunity, never the underlying Property or a safety violation already
established. The research companion's Arts–Hughes example shows why requirement labels alone miss
this distinction, and why even full coverage by that measure still needs mixed-feature workloads.

Keep three cases distinct: no record was collected, a closed source supports a declared absence
claim, and a timeout supports a conclusion under explicit timing and progress premises. Silence
alone is not an observed rejection or quiescence. Reuse conformance's source-closure and alternative
execution rules; this report must not infer an outcome from the collection deadline.

### Check compatibility beyond typed interfaces

Types can establish that a delivered value fits a receiver without establishing that the receiver
models it in the current state. For each reachable provider output in the pilot composition, check
that the receiving state has an explicit handling decision: acceptance, buffering, rejection,
intentional ignore or a located unsupported boundary. Use fn-122 laws and the modality views;
respect the composition's actual synchronous or queued communication semantics.

Start with retained work redelivered after restart while admission is paused. The finding should
name the sending transition, receiving state and absent decision. Separately check that progress
for operation B cannot satisfy operation A's pending recovery claim. These are finite compatibility
diagnostics, not an imported ioco/uioco theorem or proof of compositional liveness. The companion
records the stronger assumptions those academic results require.

### Keep defect discovery separate from requirement inference

A surviving behavioral mutant is a question for the owner. The mutant might violate an unstated
promise, expose missing observations, preserve intended behavior, or fall outside the modeled
contract. Show a distinguishing trace if one exists and ask which behavior is intended. Generate
regression proposals only after that intent is represented by a reviewed Property or scope
decision. Do not turn every survivor into an automatically synthesized prohibition.

## Implementation sequence

Paths below identify current files. Where fn-112/114/122 move declarations, follow the owning
Definition ID to its new file instead of introducing a parallel implementation.

1. **Pin the pilot obligations and evidence.** In `model/temporal/standaloneactivity` and the queue
   feature location established by fn-112, account for modeled and excluded behavior and bind the
   existing positive/negative admission and custody controls. Add focused assurance fixtures beside
   `tools/umpire/model/activity_system_test.go`. Record actual baseline results, latency and current
   unsupported cases. Retain the legal examples, bad controls, atomicity assumptions, input
   relationship classes and observation obligations identified by the research sweep. Deliver the
   first missing-behavior example before building a general mutator.
2. **Enforce exhaustive authoring and state invariants.** Extend `model/umpire/Claims.scala`,
   `model/lifter/Claims.scala`, `model/lifter/Expressions.scala`, IR schema/admission and
   `tools/umpire/model/checking.go` for initial/successor invariant checks, located match findings
   and rejection of ignored `ensuring` contracts.
   Reuse `Validate`, `Build` and `Check`. Add compiler/lifter refusal fixtures, hand-authored IR checks
   and multiple-start/zero-step examples. Exporters must explicitly support or refuse the new
   invariant form. Keep existing step-Property meaning and historical IR compatibility unchanged.
3. **Extend the existing quality inventory.** After fn-120's lint and fn-114's roots settle, add
   trigger/antecedent, decision and change-impact data through their reader/lint surfaces. Reuse
   `Receipt.Exercised`, `Holes`, `Limits`, `Table` and source positions; extend observations only where
   they cannot represent the needed distinction. Integrate fn-122 law findings as it lands. Report
   missing required behavior and removed obligations as well as new declarations. Add positive
   witness replay, first-blocking-condition explanations and before/after reachable-row summaries
   to the fn-120 report, distinguishing newly allowed from removed behavior.
4. **Build the bounded semantic mutation pilot.** Put the generic IR orchestration in a proposed
   `tools/umpire/assurance` module and a thin `tools/umpire/cmd/umpire-assure` command. Its public
   operation takes admitted IR, selected declarations and explicit budgets and returns a report.
   It consumes the reader's public facade and fn-120 diagnostics, never its private checker. Start
   with the six defect families above, deterministic cloning, frozen-oracle checks, witness replay,
   nondeterministic behavior comparison and honest result accounting. Add the module and permitted
   imports to `UMPIRE_MODULES.md` when implementing; feature policy remains under `model/temporal`,
   not in the tool.
5. **Challenge Properties and evidence independently.** Add curated clause controls and good/bad
   record pairs beside `tools/umpire/conformance/{closing,identity,assessment}_test.go`. Reuse
   `conformance.Prepare`, existing offline assessment, `recordedrun` and replay APIs. Add the held
   admission independent workload in the functional test harness. Extend conformance diagnostics
   with a bounded prototype for pairs of indistinguishable executions and pending observation credit,
   keeping its algorithms and caps in `tools/umpire/conformance`. Cover intermediate, stale and
   skipped observations before adding their live actuators. Keep runtime modules free of
   Model imports and keep synthetic record results distinct from real implementation executions.
6. **Integrate reports and fast gates.** Have `model/gate/Gate.scala` consume the cheap lint report
   and run the bounded must-detect set. Extend `Makefile` and `.github/workflows/umpire.yml` with a
   separate, explicitly bounded wider campaign. Render witnesses with the existing explorer/trace
   tooling; reuse `tools/umpire/explore` reduction where its contract applies. Document the workflow
   in `model/README.md` and record only implemented semantics in `model/SEMANTICS.md`.
7. **Prove reuse on a second feature.** Apply the same report and mutation runner to Nexus caller
   completion/closure, then to fn-122's second entity when available. Add only feature declarations
   and focused controls. If it needs feature-specific branches in the generic runner, revisit the
   interface before expanding the catalog.

Steps 1 and the IR mutation prototype can use today's hand-authored claims. They do not wait for
the entire DSL roadmap. Step 2 coordinates any schema edit after the Query-total/named-choice schema
work already sequenced by fn-112 and fn-120, and avoids overlapping fn-123's schema edits. Step 3
extends their completed foundations. Step 5's synthetic records can proceed before its live
workload. No implementation is part of this planning
change, and no existing spec's behavior-preserving freeze is relaxed by this document.

The research expands the acceptance evidence, not the first release's mandatory toolset:

| Delivery tier | Work | Gate |
| --- | --- | --- |
| First useful release | Existing lint plus explicit boundaries, initial-state checks, required witnesses, curated defects and source diagnostics | Deterministic malformed-input and must-detect failures; no percentage threshold |
| Bounded pilot after that | Clause sensitivity, input-relation challenger, observation-pair analysis, changed-row explanations and one independent execution seam | Each feature has a demonstrable seeded failure, honest inconclusive result and bounded cost |
| Wider campaigns | Selected combinations of faults, detailed cut points, more class representatives and periodic full dependency checks | Report unfinished scope; retain pinned regressions independently of campaign budget |
| Separate future work | SMT unsat cores, full temporal vacuity analysis, general trace-language differencing, new scheduler, arbitrary stateful shrinking and formal refinement proofs | Only proceed when the bounded pilot exposes a concrete need; no dependency of the first release |

## Verification and acceptance

| Acceptance example | Required result |
| --- | --- |
| Add a finite control value without handling it | Compiler/lifter or scoped lint fails at the match; a broad default cannot silently accept it |
| Write a postcondition with Scala `ensuring` | Lifter refuses it at its source and names the supported Property/invariant declaration |
| Introduce a hole behind a reachable guard | Result remains incomplete and names the hole; an independently found violation still stands |
| Start in `Active.two`, including a start with no transitions | Initial-state invariant fails; successor-only coverage cannot pass it |
| Never reach the paused/terminal antecedent | Report the unexercised condition, its scope and limits, even when the predicate was evaluated |
| Disable every successful admission or remove retry | Required positive witness/progress check fails while safety-only success remains visible |
| Admit a held stale delivery after pause | Unchanged Property kills the behavioral mutant with a replayable counterexample |
| Replace that Property with `true` | Its curated bad control is no longer rejected; the assurance suite detects the lost protection |
| Mutate a helper used by both step and Property | Runner refuses the experiment unless the oracle's transitive semantics stay frozen |
| Lose a commit record or cross an attempt ID | No unsupported success; report nonconformance, error or inconclusive according to the evidence |
| Only change unrelated clock readings | Same assessment and causal support, extending the existing clock-skew regression |
| Exhaust a budget or break the mutant's types | Separate unresolved or structural result; no behavioral kill credit |
| Delete a Property, root or capability binding | Required inventory comparison identifies the missing obligation |
| Independent workload exposes a missing legal path | Report the first mismatch; reviewed correction admits it and retains the bad-path rejection |
| Fix a discovered gap | Named regression fails on the old defect, passes the corrected behavior, and survives offline replay |
| Change `P implies Q` into unconditional `Q` | A reviewed legal `not P` example detects the strengthening where such an example exists |
| One failing conjunct hides a second weak clause | Report clause discrimination separately, with a separating example or an explicit inability to find one |
| Remove an optional nondeterministic alternative | Report the change without inventing a violated requirement; a required alternative still fails its obligation |
| Good and bad executions have identical declared observations | Retain both executions and report the unresolved Property plus their hidden distinction |
| Observe an old run's status after a new obligation activates | Observation cannot discharge the new obligation; report the matching identity/version that is missing |
| Unpause before collecting an adequate observation of pause | Earlier pause coverage credit is superseded, not earned; Properties remain in force |
| Collection times out without a closed source or justified timing premise | Report missing evidence, not an inferred rejection or proven absence |
| A provider redelivers while the receiving state has no modeled decision | Compatibility diagnostic names the sending row, receiving state and unsupported handling |
| Operation B completes while operation A remains stuck | B's evidence cannot discharge A's progress obligation |
| Correct final state follows an illegal intermediate admission | Safety violation remains authoritative despite later recovery |
| Fault command runs but misses the intended commit window | Fault realization and target coverage remain distinct; this is not a detected product bug |
| Small valid capacity exposes a branch always missed at large capacity | Challenger reports the newly reached condition with its declared configuration |
| Shrinking removes the pause, commit or identity relation required by the failure | Reject the edit even if some unrelated failure still occurs |
| A coarse atomic step hides the seeded race | A supported detailed seam exposes it under the unchanged product Property; unsupported control remains explicit |
| A changed Model retains all safety verdicts but loses required retry behavior | Before/after report supplies the lost witness and changed declaration |

Use the repository's existing validation entrypoints for implementation: `make lint-model`,
`make umpire-check-model`, and focused `go test -tags test_dep` runs over reader, lowerer,
conformance and assurance packages. Run `make lint-code-fast` for Go changes. Use
`make umpire-check-live-tests` for the independent execution milestone. Backend evidence comes only
from an actual `make umpire-check-backends` run on a worker with its pinned tools; ordinary exporter
unit tests and skipped tool runs do not establish independent agreement.

The proposed `umpire-assure` command must produce deterministic reports from the same exact inputs,
reject stale or incompatible saved evidence, and name every budget-limited or unrun item. Test the
reporter's own status accounting with one fixture per outcome and a positive nonzero execution
floor. Protect the unmodified source, IR and Case trees from all mutation runs.

Measure edit-to-first-actionable-finding, bounded campaign duration, confirmed defects per operator
family, unresolved findings and time to a retained regression. Establish a pilot baseline before
setting CI latency thresholds. Compare the same defect catalog and budgets before and after the
new checks. Faster output is useful only if it retains the required detections. Wider campaigns
remain separate from the cheap authoring gate so 10x more sites cannot silently make every edit
run 10x longer; unfinished work remains visible.

## Limits and tradeoffs

Exhaustiveness applies to declared domains and bounds. It cannot establish completeness of product
requirements, arbitrary payload abstractions or real distributed schedules. Runtime evidence is
sampled. Agreement between tools sharing the same IR does not establish that the IR describes the
right product. Preserve independently justified examples at every boundary.

Avoid a repository-wide mutation percentage, automatic Property weakening, automatic suppression
of survivors, or a new general temporal-logic engine. Start with one semantic mutation at a time;
higher-order mutants and automatic assumption minimization wait for concrete pilot evidence.
Use scoped finite comparisons and bounded storage rather than materializing every path. The
assurance command operates offline by default; live experiments use the existing authorized,
isolated Testpilot execution path. An interrupted offline campaign leaves an incomplete report,
does not publish regressions, and does not modify checked-in artifacts.

## Pattern Survey

### Analogous Features

- `model/umpire/Machine.scala:19` — Typed `~>` bindings reject mismatched action inputs. `evidence` at line 79 accepts a total fact-to-observation function; compiler exhaustiveness warnings become errors through `model/project.scala:7`.
- `tools/umpire/model/validate.go:21` — `Validate` checks the entire IR, aggregates located errors, and rejects unknown declarations, invalid finite catalogs, incompatible claims, recursion and malformed realizations before checking.
- `tools/umpire/model/machine.go:158` — `HoleRow` distinguishes unknown behavior from disabled state/action pairs. `tools/umpire/model/eval.go:601` treats an unmatched expression as an undeclared hole, preserving the expression’s source position.
- `tools/umpire/model/internal/checker/search.go:377` — `conclude` can return `VerifiedWithinLimits` with `Exercised == false`; the reader preserves that flag (`checking.go:636`). Selected feature tests explicitly assert it, but the result kind alone does not establish nonvacuity.
- `model/lifter/Expressions.scala:165` — `stripContracts` lifts `require` but discards Scala `ensuring`. Postconditions written in that form therefore are not Model checks.
- `model/temporal/standaloneactivity/Model.scala:386` — Wildcard branches return `Nil`, which means disabled behavior. Such branches compile exhaustively while admitting future enum cases into that default; compilation cannot determine whether the omission was intentional.
- `tools/umpire/model/checking_test.go:3` — Checker controls use hand-derived expectations and one mutation of an otherwise checked Model. `model/temporal/standaloneactivity/System.scala:127` also declares an intentionally faulty admission design.
- `tools/umpire/conformance/assessment_test.go:349` — Scripted evidence fixtures compare live and replayed assessments; `TestClockSkewChangesNoAssessment` at line 369 challenges timestamp dependence. `closing_test.go:71` and `closing_test.go:186` test unjustified absence inference and crossed attempt/delivery identities.
- `tools/umpire/export/agreement.go:25` — `QuintAgreement` compares reachable transitions, monitor products and Property readings, replays external counterexamples, and lists unsupported portions. `tools_test.go:17` makes external execution opt-in; ordinary tests do not establish backend agreement.
- `.flow/specs/fn-120-adopt-what-quint-does-well-named.md:102` — Planned lint covers unused/unreachable declarations, unrealized actions, unchecked refinements and untriggered verify Queries. Line 121 explicitly excludes mutation-sensitivity lint. This is planned work, not implemented infrastructure.
- `.flow/specs/fn-122-capabilities-and-their-laws.md:6` — Planned capabilities centralize repeated behavioral promises and interaction laws. This remains planned work, not an existing reusable law library.

### Reusable Utilities

- `tools/umpire/model/checking.go:245` — `Check` — Gives the admitted declarations structured receipts, including limits, assumptions, witnesses, holes and source context; useful existing result ownership.
- `tools/umpire/model/checking.go:129` — `Receipt` — Carries `Explored`, `Expanded` and `Exercised` separately, avoiding inference from a success label.
- `tools/umpire/model/machine.go:18` — `Interpreter.Members` — Enumerates finite catalogs with explicit ceilings; `Machine` at line 180 exposes classes, transitions, holes and work counts.
- `tools/umpire/model/checking_test.go:25` — `mutated` — Clones admitted fixture IR before applying explicit mutations; existing test-only mutation seam.
- `tools/umpire/conformance/conformance.go:107` — `Prepare` — Binds Model, Query and Case for assessment; `Factory.New` creates assessors for independently supplied evidence.
- `tools/umpire/model/eval.go:131` — `Error` and `Hole` — Preserve the distinction between malformed declarations and unknown behavior with located diagnostics.
- `model/gate/ProtoLiterals.scala:140` — `check` — Existing source-level lint integrated into the gate, rejecting free-text protobuf names.

### Convention Anchors

- Layer ownership: Scala declares, the lifter translates, and the Go reader admits/interprets/checks. Tests enforce module boundaries in `tools/umpire/model/ownership_test.go`; tools consuming model semantics use the public reader.
- Refusal fixtures: `model/lifter/test/Fixtures.test.scala:227` asserts compiler/lifter refusals at exact source lines. Ordinary tests do not regenerate approved fixtures.
- Generated artifacts: `model/gate/Gate.scala:306` checks source rules, compilation, lifts, Cases and Go tooling; `--update` explicitly regenerates artifacts.
- Evidence independence: `tools/umpire/conformance/fixtures_test.go:314` supplies a scripted Driver, while export compares another interpreter’s reading. These exercise different failure surfaces.

### Proposed Alignment

Follow the existing typed authoring, reader admission, located receipts and independent evidence-test boundaries; coordinate with planned fn-120 lint and fn-122 laws. Existing checks leave semantic omissions, wildcard-disabled behavior, ignored Scala postconditions and unexercised Properties to address explicitly. Mutation controls already exist as individual tests, but the surveyed code does not provide a general mutation campaign or a completeness guarantee.

## Context files

- `UMPIRE4_VISION.md`, `UMPIRE4_SPEC.md` and `UMPIRE_MODULES.md` in this directory set the goals,
  shared rules and module ownership.
- `model/README.md` and `model/SEMANTICS.md` define today's pipeline, evidence and result meanings.
- `model/temporal/standaloneactivity/{Model,Claims,System,Realization}.scala` hold the pilot behavior,
  independent claims, faulty designs and observations.
- `model/umpire/Claims.scala`, `model/lifter/{Claims,Expressions}.scala` and
  `tools/umpire/model/{machine,checking,validate}.go` anchor authoring and interpretation.
- `tools/umpire/conformance/{closing,identity,assessment}_test.go` pin evidence and replay boundaries.
- `tools/umpire/export/README.md` distinguishes actual backend checks from their unsupported scope.
- The linked fn-112/114/118/120/122 specs establish the handoffs this plan extends.
- `MODEL_ASSURANCE_RESEARCH.md`, `UMPIRE4_INSPIRE.md`, `MODALITIES.md` and fn-123 ground the
  research-driven experiments, coverage views, fault semantics and deferred extensions.
