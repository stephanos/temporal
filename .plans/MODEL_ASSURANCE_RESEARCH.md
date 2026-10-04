# Model assurance: academic mechanisms and practical experiments

Research sweep, 2026-10-03. Companion to [the assurance plan](MODEL_ASSURANCE.md), grounded in
[the vision](UMPIRE4_VISION.md), [current semantics](../model/SEMANTICS.md), and the
[modality analysis](MODALITIES.md). These are recommendations, not implemented capabilities or
changes to the shared specification. Industry system-testing experience is collected in
[UMPIRE4_INSPIRE.md](UMPIRE4_INSPIRE.md).

The strongest common lesson is that passing checks need explanations too. A useful assurance
report establishes what made a promise applicable, which behavior challenged it, and which
observation could have distinguished success from failure. Structural coverage, a generated
witness, a proof under assumptions, and an observed implementation outcome are different evidence.

The nine sources below were checked at their primary publication, author-uploaded paper, or
official tool documentation. Each section distinguishes the source's result from a proposed
Umpire application. The experiments deliberately fit the existing reader, lint, exploration,
realization, and replay surfaces. They do not propose adopting the source tools as dependencies.

## 1. Vacuity: a true Property can ignore its apparent obligation

**Published result.** Beer, Ben-David, Eisner and Rodeh formalize vacuity beyond the familiar
implication whose antecedent never holds. They also introduce interesting witnesses that explain
nontrivial satisfaction; their practical algorithm targets a useful ACTL subset. Kupferman and
Vardi extend vacuity analysis and witness generation to broader temporal specifications. Their
question concerns whether changing a subformula can affect satisfaction, rather than whether the
checker happened to evaluate its syntax. [Beer et al., *Efficient Detection of Vacuity in Temporal
Model Checking*, FMSD 2001](https://research.ibm.com/publications/efficient-detection-of-vacuity-in-temporal-model-checking);
[Kupferman and Vardi, *Vacuity Detection in Temporal Model Checking*, STTT 2003](https://link.springer.com/article/10.1007/s100090100062).

**Umpire application.** Add two levels to the planned fn-120 findings. First, named patterns
expose their trigger and return a witness where it holds. Second, selected Boolean occurrences
can be challenged under the existing finite evaluator. Report occurrence identity and polarity;
do not silently replace every occurrence of a shared predicate. Start with supported implication
and conjunction patterns, not an assertion of general temporal vacuity detection.

`Receipt.Exercised` is a useful first condition, but reading a Property whose body is
`!paused || notAdmitted` does not show that a paused delivery was ever considered. Keep the
Property's application event, its internal antecedent, and its consequential assertion separate.
Distinguish semantic absence over a completed finite domain from absence up to a path bound and
from an interrupted search. Missing trigger coverage is a quality finding, not a product failure.

**Pilot and acceptance.** Prepare three otherwise passing fixtures: no paused state is reachable;
paused delivery is reachable and rejected; and paused delivery is reachable but the consequent is
replaced by `true`. The first yields an absent-trigger finding, the second a replayable trigger
witness, and the third loses its curated negative-control protection. A zero-step fixture must
not acquire trigger credit merely because the declaration was inventoried.

**Limit.** Nonvacuity does not establish that the author stated the right requirement. Logically
redundant clauses can be intentional. Report a reason to inspect the clause rather than deleting
it automatically. A reachability witness alone is weaker than a full interesting-witness result.

## 2. Mutation coverage: removing behavior needs a different detector

**Published result.** Chockler, Kupferman and Vardi distinguish falsity coverage from vacuity
coverage. A mutation can either make the specification false or leave it true only vacuously.
Removing behaviors cannot falsify an already satisfied universal specification when the mutant's
behavior set is a subset of the original. They also distinguish changing a state throughout the
transition structure from changing one occurrence in an execution tree. These measure different
fault classes. [*Coverage Metrics for Formal Verification*, CHARME 2003, author-uploaded
paper](https://www.researchgate.net/publication/2908607_Coverage_Metrics_for_Formal_Verification).

**Umpire application.** Give every mutation an explicit semantic class: behavior addition,
behavior removal, result replacement, or evidence transformation. The existing six operator
families need this second dimension. Guard strengthening can remove the only successful path;
guard weakening can admit a forbidden one. Counting both against only safety violations would
misclassify useful omission controls as inexplicable survivors.

Bind removal operators to required witnesses and progress obligations from the modality plan.
For a permitted but optional alternative, record loss of coverage without inventing a new must
requirement. If removing a transition creates a deadlock or changes finite-path completion rules,
the subset premise needs checking: an evaluator may give that termination distinct meaning.

**Pilot and acceptance.** Delete every successful admission result. Safety may still pass, but
the existing required-success witness must fail. Delete an explicitly optional alternative in a
separate fixture; report the semantic delta without declaring the product contract violated.
Retain the exact obligation responsible for each detection in the report.

**Limit and later experiment.** Start with persistent IR mutations. A future single-occurrence
mutation could expose a transient missed check hidden by later retries, but requires an explicit
activation state and a proof that the frozen oracle is unchanged. Do not call that equivalent to
an ordinary source edit or to a real fault injection. No coverage family establishes completeness.

## 3. Requirements-first mutation can select useful tests

**Published result.** Fraser and Wotawa combine mutations of behavioral models and temporal
requirements with model checking. Counterexamples to specially constructed coverage properties
become test candidates; related checks measure whether specification fragments matter to the
tests. Their analysis can reveal specification vacuity as well as uncovered model behavior.
This makes test selection and specification assessment related activities, while preserving their
different questions. [*Using Model-Checkers for Mutation-Based Test-Case Generation, Coverage
Analysis and Specification Analysis*, ICSEA 2006, author-uploaded
paper](https://www.researchgate.net/publication/221159017_Using_Model-Checkers_for_Mutation-Based_Test-Case_Generation_Coverage_Analysis_and_Specification_Analysis).

**Umpire application.** Preserve a relation from mutant to distinguishing witness to named
Property, then to realizable Case and observed Run. A survivor with an unreached mutation site
suggests a new search target. A changed state that reaches no distinguishing assertion suggests
a missing or weak Property. A model distinction erased by the realization suggests an evidence
problem. Present those diagnoses separately instead of recommending more random tests for all.

Generate only candidate Queries from those witnesses. Required behavior still comes from the
Model. A generated bad-model counterexample is not automatically a legal executable Case under
the good Model; replay and realization admission determine what can be run. Preserve an offline
regression when the defect cannot be driven through the current live surface.

**Pilot and acceptance.** For stale admission, demonstrate the same frozen Property rejecting
the mutant Model, identify the shortest known enabling prefix, and determine whether existing
controls can realize that prefix. Report `model-detected`, `case-realizable`, and `observed` as
separate facts. Add a second mutant whose state change is reached but invisible to existing
Properties; the report must identify an oracle gap rather than a generator gap.

**Limit.** The papers' equality and test-generation results depend on their model formalism.
Umpire should use exact scoped transition comparisons only where supported. A matching verdict
vector does not prove equivalence, and a model-generated expected output is not independent
evidence that the expected output is correct.

For nondeterministic Models, separate a possible distinguishing execution from a test guaranteed
to distinguish every allowed outcome. A mutant may admit a forbidden result while also producing
all the good results in the sampled Runs. Conversely, a change between two already permitted
results is not a behavioral failure. The pilot should include both cases and report overlap of
permitted outcomes, actual observation, and any control needed to force the distinguishing path.

## 4. Overconstraint: explain why required behavior cannot exist

**Published result.** Torlak, Chang and Jackson extract minimal unsatisfiable cores at the
specification level, mapping resolution proofs back through the translation. Their examples
diagnose overconstrained models, weak assertions, and inadequate finite scopes. Minimal means
that removing any constraint from that core restores satisfiability, not that it is the smallest
possible core. The correctness results assume a suitable translation and resolution engine.
[*Finding Minimal Unsatisfiable Cores of Declarative Specifications*, FM
2008](https://groups.csail.mit.edu/sdg/pubs/2008/mincore-fm08.pdf).

**Umpire application.** Begin with a less ambitious diagnostic over existing finite searches:
when a required witness disappears, show the first blocked step and its named guards, scenario
restrictions, starts, input classes, and assumptions. Distinguish an empty start set, disabled
action, unreachable target, hole, and exhausted resource budget. These are materially different
causes of “no example found.”

An optional follow-up experiment can temporarily relax one named assumption at a time in an
isolated clone, retaining Properties and explicit bounds, and report a recovered witness. Call
this assumption sensitivity. It is not an unsatisfiable core, automatic repair, or proof that the
assumption is wrong. Safety results from the modified experiment must not replace the baseline.

**Pilot and acceptance.** Combine two independently plausible restrictions that jointly exclude
retry recovery. A completed pilot search reports the absent required witness and both restrictions.
Relaxing the responsible restriction produces a replayable path; removing an unrelated one does
not. Repeat with an insufficient resource budget and demand an unresolved result, with no core or
unreachability claim. Keep the owner decision to alter an assumption explicit.

**Limit.** Do not introduce SAT/SMT or a general theorem prover merely to obtain cores. True
core extraction becomes worthwhile only if a selected backend already supplies proof support
and source mappings. Core membership reflects one explanation; absence from one core does not
prove a declaration unnecessary for every requirement or environment.

## 5. Requirements coverage needs a valid observation after the action

**Published result.** Arts and Hughes find that requirement labels alone produce weak tests.
Their refinement of coverage records relevant preconditions, delays credit until the effect is
observed, associates observations with the right entity, and cancels pending credit after an
intervening operation invalidates the observation. They distinguish applicability conditions from
conditions needed to test a requirement meaningfully. Even the improved suite missed a seeded
cross-feature state-corruption fault that random testing found. [*How Well Are Your Requirements
Tested?*, ICST 2016](https://publications.lib.chalmers.se/records/fulltext/232552/local_232552.pdf).

**Umpire application.** Track pending coverage evidence for a law instance, keyed by operation,
attempt, and relevant version. Record when the trigger occurred, which observation can discharge
the coverage obligation, and which intervening transition invalidates that attribution. Reuse
monitor/evidence machinery; this is assurance metadata, not a second runtime verdict system.

For pause/unpause, observing “not paused” after unpause cannot establish that pause took effect.
For a write-like operation, supplying the already stored value can satisfy the ordinary Property
while failing to challenge a dropped-write bug. Add a coverage condition requiring distinguishable
before/after representatives where appropriate, without changing when the product promise applies.

**Pilot and acceptance.** Compare three histories: pause then describe paused; pause then unpause
then describe active; and an idempotent pause on an already paused entity. All can be legal; only
the first demonstrates a changed state caused by the new pause. Report the second as superseded
coverage evidence and the third as idempotency coverage. Neither loses ordinary conformance credit.

**Limit.** Cancellation of coverage credit does not cancel a safety Property or erase a violation.
Some historical evidence remains valid after later transitions and must not be cancelled. Keep
independent mixed workloads even after every named law has a witness: cross-entity interference
and omitted requirements can remain outside a minimal requirements-covering suite.

## 6. Observational conformance must preserve uncertainty and silence

**Published result.** Timmer, Brinksma and Stoelinga formalize an extension/reformulation of
Tretmans' ioco testing theory with explicit quiescence. Conformance restricts observable outputs
after specified behavior; nondeterministic specifications can admit more than one valid response.
The development separates a sound test suite from a complete one and gives test derivation
algorithms. Silent internal actions and possible states after observations are part of the formal
account. [*Model-based Testing*, 2011, university-hosted
paper](https://ris.utwente.nl/ws/portalfiles/portal/5351023/TBS11.pdf).

**Umpire application.** Use this as a checklist for the existing evidence boundary, not a claim
that Umpire implements ioco. An observation can leave several candidate Model executions.
Preserve them, and explain which observable fact would distinguish a candidate violating a
Property from a satisfying candidate. The selected generation witness must not become the only
legal explanation of a Run. A reachable alternative result can be conformant even when it was
not the generator's target.

Treat “no event was collected,” “the source closed with no such event,” and “progress was required
by an expired deadline under satisfied assumptions” separately. A polling timeout is not evidence
of theoretical quiescence. Reuse source-closure and deadline semantics rather than adding a generic
silence-success rule. An accepted request and its durable effect need separate support.

**Pilot and acceptance.** Give the checker two valid explanations of an RPC acknowledgment, one
with a commit and one without. A commit-dependent claim remains inconclusive until a correlated
commit observation is added. Removing that observation restores uncertainty. A separate fixture
uses sufficient closed-source evidence to reject an unexpected absence. Reordering unrelated
timestamps does not change these judgments.

**Limit.** Formal test completeness assumptions do not hold automatically for finite sampled
Runs, distributed observers, or Umpire's current execution limits. The analysis explains the
observations available; it cannot conclude that an invisible forbidden transition never occurred.

## 7. State-machine shrinking must retain valid commands and the same defect

**Official mechanism.** QuviQ's state-machine testing documentation describes symbolic command
sequences, preconditions reused during shrinking, and optional repair followed by renewed
precondition checks. It also discusses repeated executions during shrinking of intermittent
concurrency failures. Symbolic references allow command results to be reused without baking
runtime identities into the generated program. [QuviQ `eqc_statem`, official
documentation](https://www.quviq.com/documentation/eqc/eqc_statem.html).

**Umpire application.** Keep reduction behind the existing legal-edit and replay interfaces.
A candidate must preserve references to created operations, the trigger, the relevant fault
activation, the evidence needed to decide the claim, and the same failure key. Command deletion
that merely creates a malformed Case or loses an observation is an unsuccessful reduction.
State and input abstraction reductions need their own checks; shortening a schedule can change
whether the original stale-attempt relationship exists.

**Pilot and acceptance.** Reduce a held-delivery/pause/release failure containing unrelated
describes. Permit those describes to disappear. Reject reductions that remove creation, bind the
release to another attempt, remove the pause trigger, or turn a violated judgment into inconclusive.
Keep the original failing artifact until the smaller one replays offline and meets the existing
two-fresh-Run reproduction condition for execution regressions.

**Limit.** Two successful reproductions meet the current workflow criterion; they do not establish
a probabilistic reliability guarantee. Record attempted and successful replays, controlled schedule
information, and reduction budget. A seed reproduces generation, not every deployment interleaving.
Do not import QuviQ's retry count or minimality terminology as an unqualified Umpire guarantee.

## 8. Component promises require compatible communication assumptions

**Published result.** Van Cuyck, van Arragon and Tretmans show that uioco conformance alone is
not compositional and introduce mutual acceptance between component specifications. Their theorem
preserves conformance under parallel composition for mutually accepting models and conforming
implementations, with its stated model assumptions. They also explain that global quiescence can
hide a component that stops producing required output while another keeps producing output.
[*Compositionality in Model-Based Testing*, ICTSS 2023, extended author
version](https://arxiv.org/pdf/2307.03701).

**Umpire application.** In addition to fn-122 capability laws and modality refinement, review
every reachable provider output against the receiving component's modeled input handling. Typed
messages can still arrive in states where the consumer silently disables them. A buffer, rejection,
ignore outcome, or unsupported boundary is a behavioral decision, not a type-level guarantee.

**Pilot and acceptance.** Compose a queue that can redeliver retained work after restart with an
activity admission component that fails to describe redelivery while paused. The diagnostic names
the sending row, receiving state, and missing decision. Fixing the boundary must preserve rejection
of stale admission. Separately stop progress for operation A while operation B continues; B's
observations cannot discharge A's recovery claim. Use synthetic records until live multi-operation
support exists, and label that evidence accurately.

**Limit.** Umpire's provider/refinement semantics are not uioco, and the theorem is not transferable
without a mapping and proof. Use the paper's compatibility question as a finite diagnostic now.
Existing component progress assumptions remain necessary; a successful interface check does not
establish liveness, fault tolerance, or correctness of the implementation's composition mechanism.

## Adoption order and report changes

| Stage | Smallest useful addition | Evidence needed to retain it |
| --- | --- | --- |
| Adopt now in the plan | Separate trigger reachability, meaningful challenge, and observation credit | Three vacuity controls and three pause-observation histories |
| Adopt now in the plan | Classify mutation direction and bind omission detectors explicitly | Successful-admission deletion fails its named positive obligation |
| Adopt now in the plan | Explain absent required witnesses with first blocked steps | Overconstraint and budget-exhaustion fixtures remain distinct |
| Adopt now in the plan | Retain mixed workloads and operation-specific progress evidence | Cross-operation evidence cannot satisfy the wrong claim |
| Bounded experiment | Predicate-occurrence sensitivity and assumption-relaxation diagnostics | Located findings with reproducible witnesses and honest unresolved outcomes |
| Bounded experiment | Mutation-guided Query candidates and legal reduction | Candidate survives admission; reduced example retains its defect and evidence |
| Later, only if justified | Proof cores, single-occurrence mutations, formal composition mapping | A concrete pilot gap the simpler checks cannot diagnose |

Extend existing assurance reports with the trigger witness, adequacy condition, pending observation,
reason coverage credit was lost, mutation semantic class, distinguishing Property, and exact search
scope. Fields can be absent when inapplicable; absence must not silently mean satisfied. Do not add
a single aggregate quality percentage. Keep all recommendations within fn-120's diagnostics,
fn-122's laws, the modality report, and the current replay and conformance boundaries.
