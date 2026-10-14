# Java tools for the Umpire semantic model

Java's ecosystem can supply parts of Umpire's model analysis directly. The strongest additions
to [SCALA.md](SCALA.md) are **JavaSMT for symbolic definition checks**, **AutomataLib for finite
automata algorithms and test generation**, and **Theta as a ready-made symbolic checker**.
**Alloy 6** merits a separate experiment for Nexus ownership and routing. **MTSA** is worth studying
for gradual contracts and refinement.

Research checked on 2026-09-30 against primary documentation and source, using
[PROTOTYPE.md](../../PROTOTYPE.md), [the vision](UMPIRE4_VISION.md), and
[the implemented IR semantics](../model/SEMANTICS.md) to judge fit. These are candidates
for the Scala/IR proposal; the existing Lean-based specification remains unchanged. No tool was
installed or benchmarked, and no dependency was adopted. Recommendations below are inferences
from documented capabilities and Umpire's requirements.

## Candidates by modelling goal

| Tool | Direct modelling contribution | Priority |
|---|---|---|
| JavaSMT + SMTInterpol | Clause coverage, shadowing, invariant preservation, local refinement obligations, bounded counterexamples | First symbolic diagnostic candidate |
| AutomataLib | Finite transition systems, composition, modal refinement, distinguishing test sequences | First finite-algorithm candidate |
| Theta | Symbolic reachability with abstraction refinement; existing model-checking algorithms | Compare with planned Apalache backend |
| TLC | Reachable-state safety, deadlocks, and fair non-progress cycles in a finite model | Keep the planned temporal backend |
| Alloy 6 / Pardinus | Relational ownership, identity, reconstruction and routing; temporal counterexamples | Useful second backend for the Nexus specimen |
| MTSA / LTSA | Partial behavioural contracts, composition and model elaboration | Study semantics before integration |
| LearnLib | Active/passive model learning and conformance-test strategies | Later, for controlled black-box experiments |
| RV-Monitor | Parameterized event monitors, including offline log checking | Reference or differential oracle |
| Choco | Constraint-based generation of bounded scenarios and schedules | Optional generation tool |

Scala can call these Java APIs from a build or verification command. A Go front end can also use
their standalone tools through generated artifacts. Their availability therefore does not settle
the authoring-language choice. Keep the JVM outside Go's deterministic exploration process.
Generate immutable model artifacts, run verification separately, and replay witnesses in Go.

## JavaSMT and SMTInterpol

JavaSMT is a typed Java API over multiple SMT solvers, not a transition-system checker. It exposes
incremental solving, models, unsatisfiable cores, formula traversal, interpolation and other
solver-dependent capabilities. Its current GitHub release is
[6.0.0](https://github.com/sosy-lab/java-smt/releases/tag/6.0.0), published 2026-03-02; the API requires
Java 11+. Solver and platform support vary, while SMTInterpol and Princess avoid native solver
bindings. [JavaSMT README](https://github.com/sosy-lab/java-smt)

SMTInterpol is implemented in Java and accepts SMT-LIB 2.6. Its current documented theories include
uninterpreted functions, linear integer/real arithmetic, arrays, bitvectors and datatypes.
Quantifiers exclude datatypes; interpolation supports quantifier-free fragments except datatypes.
These details matter: older descriptions presenting it as arithmetic-only are incomplete.
[SMTInterpol documentation](https://ultimate.informatik.uni-freiburg.de/smtinterpol/)

**Inferred Umpire application:** lower the IR directly into formulas and ask:

- Can an admissible state/input match no clause? This identifies a specification hole.
- Can a later clause ever win? This identifies fully shadowed clauses.
- Do imported validity constraints survive each atomic update?
- Does an authored invariant hold initially and remain true after every permitted transition?
- Can the stale queued-task admission violate `PausedIsNotDispatched` within a bounded trace?

For ordered clauses, encode effective guard `E_i = D AND G_i AND NOT(G_1 OR ... OR G_(i-1))`,
including patterns in `G_i` and admissible state/input constraints in `D`. The hole predicate is
`D AND NOT(OR G_i)`. A matching `disabled` clause still consumes the match; it cannot fall through
to a later clause. Raw guard overlap is not automatically an error when order deliberately
selects a winner. Check local holes separately from reachable holes. Reachable holes leave affected
behaviour checks incomplete; treating them as disabled transitions could produce false success.

For refinement, encode the state projection **and** event/result mapping. A detailed step with a
public result cannot count as invisible stutter merely because projected states are equal.
Abstract nondeterminism can require quantified or enumerated inclusion obligations; JavaSMT does
not supply a refinement algorithm. Likewise, it does not supply reachability, PDR, induction
strengthening, fairness or liveness checking. Umpire would own those encodings and algorithms.

An unsatisfiable initiation-plus-inductiveness obligation can establish safety for all finite
trace lengths under its assumptions, even with symbolic integers. Failure of inductiveness can
exhibit an unreachable state; it is not automatically a reachable execution bug. A successful
depth-`k` bounded check establishes only its declared bound. Unknown and timeouts stay unresolved.

Embed JavaSMT in a separate JVM verification command, or avoid the wrapper and emit SMT-LIB to
SMTInterpol's standalone CLI. Its documented CLI reads a script or standard input.
[SMTInterpol usage](https://github.com/ultimate-pa/smtinterpol)
JavaSMT is Apache-2.0; SMTInterpol is LGPL-3.0. Other selected solvers have their own licenses.
[JavaSMT source](https://github.com/sosy-lab/java-smt),
[SMTInterpol license](https://github.com/ultimate-pa/smtinterpol/blob/master/LICENSE)

## AutomataLib

AutomataLib is a Java library independent of LearnLib. It supplies finite automata and transition
systems, graph traversal, shortest paths, equivalence, minimization, and model-based test-generation
algorithms. Its MTS utilities implement conjunction, parallel composition and modal refinement.
These are reusable pieces for composed product/system models and generating distinguishing
regressions. LTL checking uses an external LTSmin integration; adopting the data structures alone
does not introduce another complete temporal checker.
[AutomataLib overview](https://github.com/LearnLib/automatalib),
[MTS utilities](https://github.com/LearnLib/automatalib/blob/develop/util/src/main/java/net/automatalib/util/ts/modal/MTSs.java)

The checked `ModalRefinement` implementation matches transitions with the same label and computes
a relation over pairs of finite states. It has no special silent-action closure. Umpire's state
projection, event/result mapping and invisible stutter therefore need an explicit adapter or a
different relation. In particular, equal projected states cannot hide a changed public outcome.
The library's modal `MUST` requires a transition in the implementation; it does not independently
establish eventual execution under fairness.
[Refinement implementation](https://github.com/LearnLib/automatalib/blob/develop/util/src/main/java/net/automatalib/util/ts/modal/ModalRefinement.java)

`WMethodTestsIterator` generates transition-cover prefixes, bounded middle sequences and
characterizing suffixes from a deterministic automaton. It returns input words, so Umpire could
generate regression artifacts on the JVM and execute them later with Go. Distinguishing suffixes
can expose states that accept the same first action but react differently afterward, such as
paused versus dispatchable activities. Applying the W-method to Temporal needs a deterministic
abstraction and its finite-state/reset assumptions. General asynchronous nondeterminism requires
a different conformance relation. Keep the full IR graph for verification; minimizing it for test
generation must preserve the outputs relevant to the test.
[W-method source](https://github.com/LearnLib/automatalib/blob/develop/util/src/main/java/net/automatalib/util/automaton/conformance/WMethodTestsIterator.java)

The current release checked was **13.0.0**, published 2026-08-12, under Apache-2.0.
[Release](https://github.com/LearnLib/automatalib/releases/tag/automatalib-13.0.0)

## Theta

Theta is a Java/Kotlin verification framework with standalone model checkers and library modules.
Its Symbolic Transition System (STS) represents variables, initial states, a relation over current
and next states, and a safety property. That is a direct target for the Umpire IR. Its Extended STS
(XSTS) additionally expresses atomic statement sequences, assumptions, assignments, havoc and
nondeterministic choices. A translation can consume expression data instead of executing Scala
closures or modelling JVM bytecode.
[Theta overview](https://github.com/ftsrg/theta),
[STS formalism](https://github.com/ftsrg/theta/blob/master/subprojects/sts/sts/README.md),
[XSTS formalism](https://github.com/ftsrg/theta/blob/master/subprojects/xsts/xsts/README.md)

The XSTS CLI documents CEGAR, LTLCEGAR, bounded checking, k-induction, decision-diagram and Horn
backends, with concrete counterexample export. This makes Theta an alternative to building a
reachability checker around JavaSMT. CEGAR refines a solver's abstraction while answering a query;
it does not itself establish Umpire's authored system-to-product refinement.
[XSTS CLI](https://github.com/ftsrg/theta/blob/master/subprojects/xsts/xsts-cli/README.md)

An exporter must preserve clause order, domain bounds, frame conditions and the IR's update
evaluation rules. XSTS normally alternates environment and internal transitions; putting faults
in its environment block would silently impose an ordering absent from Umpire. STS avoids that
extra scheduling convention. Alternatively, an XSTS exporter can put all Umpire actions into one
transition choice and account for the skip environment steps, including their effect on temporal
properties. Neither representation supplies Umpire's hole or evidence semantics automatically.

Start with safety checking. Theta's documented LTL option needs a separate test of Umpire's
fairness encoding before treating it as a TLC replacement. The CLI documents Java 21 and native
Z3 libraries. Current release **8.0.4** was published 2026-09-28; the project is Apache-2.0.
Compare the same faulty/corrected pause-admission model through Theta and Apalache before choosing
another backend. No relative performance claim is established here.
[STS CLI requirements](https://github.com/ftsrg/theta/blob/master/subprojects/sts/sts-cli/README.md),
[Release](https://github.com/ftsrg/theta/releases/tag/v8.0.4)

## MTSA and LTSA

MTSA models and analyzes labelled and modal transition systems, extends LTSA, and studies
incremental elaboration of partial behaviour models. LTSA provides finite communicating-machine
composition, action relabelling/hiding, and safety/liveness analysis using FSP. This is relevant
to matching, worker, channel and persistence contracts that gain detail over time.
[MTSA project](https://mtsa.dc.uba.ar/),
[LTSA and FSP](https://www.doc.ic.ac.uk/~jnm/LTSdocumention/FSP-notation.html),
[LTSA's Java implementation](https://www.doc.ic.ac.uk/~jnm/LTSdocumention/LTSA.html)

Modal models distinguish behaviour an implementation must provide from behaviour it may provide.
The developers describe eliciting more detail through scenarios, properties and model analysis.
This gives Umpire prior art for gradual contracts and consistency checking.
[Developer paper on partial model elaboration](https://www.cs.toronto.edu/~chechik/pubs/tse09.pdf)

There is still a semantic gap. A Umpire omission represents missing knowledge; a modal optional
transition describes permitted implementation choices. Filling an omission and refining a
contract must remain separate operations. Likewise, hiding internal actions needs a relation
that preserves Umpire's public results and progress obligations. Study these definitions before
mapping the IR to an MTS. The current site links a repository and jar, but this investigation
could not inspect that repository or establish a stable headless API. MTSA is therefore a
research reference before it is an adoption recommendation.

## LearnLib

LearnLib supplies active and passive automata learning, counterexample processing, symbol
abstraction, caching and test-based equivalence strategies. Its current release is **19.0.0**
(2026-08-12), under Apache-2.0. It could help investigate unmodelled behaviour in a controlled
public-API slice, then suggest candidate regressions or missing contracts. A learned implementation
model remains experimental evidence; it cannot become the expected product contract automatically.
[LearnLib features and releases](https://learnlib.de/)

The `SUL` adapter exposes setup, teardown and input/output steps. A separate Java experiment
could call a Go driver, but this would be outside checkpointed Go exploration. Successful Mealy
learning requires a repeatable abstraction of each queried input sequence. Asynchronous outcomes,
hidden scheduler choices and fresh IDs require explicit abstraction/control before using that
interface. Production canaries lack the reset/control assumptions of many learning experiments.
[SUL source](https://github.com/LearnLib/learnlib/blob/develop/api/src/main/java/de/learnlib/sul/SUL.java)

LearnLib explicitly documents that failure to find a counterexample does not establish
equivalence. Record the test budget and assumptions. Since Umpire already intends to author the
model, AutomataLib's generation algorithms are the earlier reuse opportunity.
[Equivalence-oracle contract](https://github.com/LearnLib/learnlib/blob/develop/api/src/main/java/de/learnlib/oracle/EquivalenceOracle.java)

## RV-Monitor

RV-Monitor generates parameterized Java monitors from event specifications, including temporal
logic and finite-state descriptions. Applications can call generated event methods explicitly.
Its repository includes a log adapter that reads a file and feeds correlated events into monitors,
so Java bytecode instrumentation is optional. This makes offline differential checking of an
exported Nexus event-monitor subset plausible.
[RV-Monitor documentation](https://github.com/runtimeverification/rv-monitor/blob/master/docs/src/docs/quickstart.rst),
[Offline log adapter](https://github.com/runtimeverification/rv-monitor/blob/master/examples/java/FSM/PostfixLog/PostfixLogAdapter.java)

Keep the Go monitor authoritative for execution and canaries. RV-Monitor's trace language would
need an exporter from the same IR, with tests of finite-trace endings, instance correlation,
deadlines, missing evidence and verdicts. A totally ordered log monitor alone does not retain all
executions consistent with partially observed causal order. JavaMOP's AspectJ integration helps
instrument Java applications; it cannot instrument the Temporal Go server.
[JavaMOP requirements](https://github.com/runtimeverification/javamop/blob/master/INSTALL.md)

Treat RV-Monitor as a reference or optional oracle. The checked master commit is dated 2020-12-18,
so this investigation does not claim active development or current JDK compatibility.
[Checked commit](https://github.com/runtimeverification/rv-monitor/commit/d01f5360c8c81a5c1a2b367b04beeba6080d384e)

## Alloy 6, Kodkod and Pardinus

Alloy 6 adds mutable relations, next-state expressions and future/past temporal operators.
Counterexamples are lasso traces. Object signatures retain finite scopes; time can be bounded or
checked completely using `1.. steps`. Complete checking remains finite-domain checking, not a
proof for arbitrarily many operations/runs. It requires an appropriate external NuSMV/nuXmv backend.
[Alloy 6 semantics](https://alloytools.org/alloy6.html)

**Inferred fit:** the Nexus caller-close/reset specimen is unusually suitable for relational
modelling: logical operations, original/successor runs, retained outcomes, cancellation principals,
ownership and routing form linked relations. Assertions can check whether acknowledgment implies
commitment at the current owner or a retained delivery obligation. Temporal assertions can search
for lost-outcome/non-progress cycles under authored fairness assumptions. Alloy does not infer
those assumptions or observation sufficiency.

Alloy's distribution can be embedded as a Java API and currently requires Java 17+. Its released
CLI source supports `exec`, JSON/XML solutions and a `receipt.json` containing settings and
solutions. This gives a practical external adapter and witness-decoding path.
[Alloy repository](https://github.com/AlloyTools/org.alloytools.alloy),
[CLI source](https://github.com/AlloyTools/org.alloytools.alloy/blob/master/org.alloytools.alloy.cli/src/main/java/org/alloytools/alloy/cli/CLI.java)
The latest GitHub release checked was
[6.2.0](https://github.com/AlloyTools/org.alloytools.alloy/releases/tag/v6.2.0), published 2025-01-09.

Kodkod is the lower-level relational engine; Pardinus extends it with temporal, decomposed and
bounded/unbounded-time solving. Direct Pardinus embedding can bypass Alloy text, but requires
constructing relational formulas and explicit bounds. An Alloy export initially offers more
readable review artifacts.
[Pardinus API source](https://github.com/AlloyTools/org.alloytools.alloy/blob/master/org.alloytools.pardinus.core/src/main/java/kodkod/engine/PardinusSolver.java)

The distributed Alloy/Kodkod license notices are MIT-style; Pardinus source carries MIT terms.
The root license file explicitly says its Apache text is not yet valid and current code is MIT.
Use component/release notices rather than GitHub's ambiguous top-level classification; packaged
solvers carry separate notices.
[Alloy notice](https://github.com/AlloyTools/org.alloytools.alloy/blob/master/org.alloytools.alloy.dist/LICENSES/Alloy.txt),
[Kodkod notice](https://github.com/AlloyTools/org.alloytools.alloy/blob/master/org.alloytools.alloy.dist/LICENSES/Kodkod.txt),
[root license](https://github.com/AlloyTools/org.alloytools.alloy/blob/master/LICENSE)

## TLC

TLC is an explicit-state checker for an executable subset of TLA+, particularly finite systems.
Its CLI accepts generated `.tla`/`.cfg` files, checks deadlocks by default, and can export error
traces as JSON. This already matches the proposed separate verification process.
[TLC usage](https://docs.tlapl.us/using:tlc:start)
Its liveness algorithm constructs a behaviour graph and finds violating strongly connected
components. A fair non-progress cycle is therefore stronger evidence than an unfinished finite
search prefix.
[TLC liveness implementation](https://docs.tlapl.us/codebase:liveness)

**Inferred fit:** retain TLC for finite-scope Nexus progress and bounded counters/queues. An
exhaustive completed finite-state run covers arbitrary trace lengths inside that scope; it does
not establish an unbounded population theorem. Preserve first-match order, holes, frame conditions
and mapped results in generated actions. Terminal stuttering and fairness require explicit
semantic choices. Symmetry should not be enabled for liveness checking.
[TLC symmetry guidance](https://learning.tlapl.us/blocking-queue/symmetry/)

The tools are Java, require Java 11+, and are MIT-licensed. GitHub's latest stable release endpoint
reports [1.7.4](https://github.com/tlaplus/tlaplus/releases/tag/v1.7.4); current master builds are
published through the 1.8.0 prerelease. Pin a concrete version rather than assuming current source
features exist in the stable jar. [TLA+ tools README](https://github.com/tlaplus/tlaplus)

## Lower-priority tools

**OpenJML and KeY** verify Java implementations against JML contracts. They could prove properties
of a Java IR evaluator or transformation if one existed. Applying them to the Scala model would
require Java source generation and a semantic correspondence argument. As with Stainless proving
an expression-tree builder, a proof about the generated program establishes model behaviour only
when it reasons about the IR's evaluation semantics. Direct SMT obligations avoid that additional
Java-program representation for the current proposal.
[OpenJML scope](https://www.openjml.org/about/),
[KeY](https://www.key-project.org/)

**Java PathFinder** explores Java bytecode programs. It is useful if Umpire needs to verify a JVM
implementation, but provides no direct control over the Go server's goroutines, clocks or fault
hooks. Feeding it a generated Java interpreter introduces another executable model to maintain.
[JPF scope](https://github.com/javapathfinder/jpf-core)

**jqwik** offers stateful property tests with invariants and shrinking of failing action sequences.
That supports Scala-to-IR-to-Go differential tests and shrinking regression inputs. It overlaps
the ScalaCheck Commands role already proposed in SCALA.md; a Java test harness would be a concrete
reason to choose it. Persist explicit action sequences and bound values, since a test seed alone
does not define Umpire's portable regression artifact.
[jqwik stateful testing](https://jqwik.net/docs/current/user-guide.html#stateful-testing)

**Xtext/EMF and VIATRA** support a separate DSL and structural model tooling. Xtext maps a grammar
to an EMF model with linking/validation; VIATRA queries model graphs. They could diagnose missing
bindings or build a dedicated modelling editor, but behavioural reachability and refinement need
other algorithms. The current proto IR and Scala compiler already supply much of that structural
surface. Adopting an EMF representation would add another schema and adapter, so defer these until
a dedicated editor or large incremental graph query has a concrete use case.
[Xtext grammar and EMF](https://eclipse.dev/Xtext/documentation/301_grammarlanguage.html),
[VIATRA queries](https://eclipse.dev/viatra/documentation/query-language.html)

Choco is a Java constraint-programming library with finite integer/Boolean/set domains, scheduling
constraints and automaton constraints. **Inferred use:** generate admissible scenario parameters,
causal event orders and optimized schedules before Go execution. It does not supply Umpire's
transition/refinement/liveness semantics. Prefer a solver already selected for verification until
constraint generation demonstrates a specific advantage. It is BSD-licensed.
[Choco](https://choco-solver.org/),
[constraints](https://choco-solver.org/docs/modeling/intconstraints/)

CPAchecker is written in Java but primarily verifies C programs through configurable analyses,
including abstraction, interpolation, k-induction and PDR. Exporting IR to C or implementing a new
analysis would add a semantic translation to validate. JavaSMT exposes its useful solver layer
more directly. CPAchecker is Apache-2.0, with independently licensed dependencies; bundled MathSAT
has research/evaluation restrictions.
[CPAchecker tutorial](https://arxiv.org/abs/2409.02094),
[CPAchecker README](https://github.com/sosy-lab/cpachecker)

## What to investigate first

Begin with **JavaSMT diagnostics and an AutomataLib generation experiment**, each behind a small
interface consuming the IR. Choose JavaSMT if the verifier benefits from a Scala/JVM library API;
direct SMT-LIB export to SMTInterpol or another solver remains available to every front end.
AutomataLib can return finite test words without owning the execution driver.

Use the prototype's faulty and corrected pause/admission designs as controls. The symbolic check
should find the declared stale-task admission trace, reject it in the corrected design within the
same scope, and report a reachable omitted clause as incomplete. Add a shadowed clause and an
unreachable inductiveness counterexample to distinguish definition diagnostics from execution bugs.
For finite test generation, determine whether distinguishing suffixes exercise a meaningful
pause/unpause or retry case that transition coverage alone misses.

Next, **compare Theta with the planned Quint/Apalache path** on exactly the same IR and bounds.
Check agreement on every transition in a small finite domain before comparing witness length,
runtime or diagnostics. Theta earns a place if its existing algorithms or trace reports improve
that concrete workflow. Keep Alloy as a focused **Nexus relation/ownership experiment**, rather
than implementing every exporter at once.

The reusable module worth extracting is an IR verification adapter with an input model, claim and
scope, and an output verdict, supporting assumptions, coverage, diagnostics and optional witness.
Solver formulas, solver contexts and library automata stay behind it. A backend must report
unsupported expressions or constraints, missing evidence and exhausted bounds; it cannot silently
translate them away. Map witnesses back to Definition IDs and source positions, pin the model
fingerprint and backend version, and replay executable finite witnesses through the Go interpreter.
An infinite liveness counterexample needs a loop and its fairness evidence as well as a finite
prefix; prefix replay alone does not validate the liveness claim.

No reviewed tool supplies the whole observation layer described in PROTOTYPE.md. Umpire still
needs to retain executions compatible with causal observations, distinguish trace conformance
from property satisfaction, and report ambiguity as inconclusive. Likewise, proving a model
within its scope does not establish that the real Temporal implementation conforms. The Go
drivers and observation mappings remain essential for that claim.
