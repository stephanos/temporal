# Umpire 4 research: formal models for verifying Temporal

Status: decision-support research, substantially expanded 2026-08-30. This is not a replacement for the
normative [Umpire 4 specification](UMPIRE4_SPEC.md), the
[Umpire vision](UMPIRE4_VISION.md), or the [current model documentation](../model/README.md).

## Purpose and source discipline

Umpire's goal is unusually ambitious: one Lean-owned behavioral model should define Temporal
product meaning, generate bounded experiments, support model checking and exploration, and judge
causally qualified evidence from local, CI, staging, black-box, and eventually canary runs. Temporal
is a particularly demanding subject because a cluster coordinates durable Workflow state across
History, Matching, Persistence, SDK Workers, and asynchronous task processing; the project's own
architecture describes these as independently scaled services joined by requests, persisted Event
History, Commands, and Tasks ([Temporal architecture](https://github.com/temporalio/temporal/blob/main/docs/architecture/README.md),
[History Service](https://github.com/temporalio/temporal/blob/main/docs/architecture/history-service.md)).

This guide asks what prior formal-methods, model-based-testing, deterministic-simulation, and
runtime-validation efforts actually establish, where they stopped, and what Umpire should copy.
Sources are ordered by trust:

1. papers, official documentation, source repositories, and first-party retrospectives;
2. engineering blogs written by the practitioners involved; and
3. substantive discussions, used to identify questions rather than establish outcomes.

Status terms are deliberately narrow. They describe the strongest publicly documented outcome, not
an intrinsic ranking of technical merit:

- **Sustained production adoption** means a first party documents repeated current use in shipping
  or operating a production system. It does not mean the method proves the complete product.
- **Shipped case study** means the subject or verified component shipped, or was exercised against a
  shipped implementation, and the source reports a concrete outcome in the studied scope.
- **Research demonstrator** means a paper and usually an artifact establish feasibility on selected
  systems, but public evidence does not establish continuing product-team ownership.
- **Constrained / maintenance burden** means the method produced value but its language, harness,
  trusted base, environment substitution, authoring cost, or ongoing bridge cost materially limits
  the transferable claim.
- **Cancelled / retired** is used only when a first party explicitly says the effort or tool was
  stopped, replaced, archived, or withdrawn. It says why one effort stopped; it is not shorthand for
  “formal methods failed.”
- **Inconclusive** means public evidence is insufficient to claim sustained adoption, retirement, or
  failure. Repository activity and citation count alone do not establish an outcome.

Paragraphs labeled **Documented** summarize what a cited source says. Paragraphs labeled
**Inference for Umpire** are recommendations derived from those sources, not claims made by them.
No absence of public evidence is treated as evidence that a project failed.

### Research method and limitations

**Documented.** This revision reviewed papers, official documentation, source repositories, and
first-party engineering retrospectives available online on 2026-08-30. Peer-reviewed papers and
artifact repositories establish the research results they report; first-party product accounts
establish how their authors say the tools were used. Practitioner discussions were used only to find
questions and primary sources, never to establish success, failure, or adoption. Marketing claims
without a reproducible artifact or technical report are labeled as such or omitted.

**Limitations.** Public reports systematically under-document engineer-weeks, abandoned model
branches, false starts, and long-term maintenance. Bug counts are not comparable across tools because
subjects, versions, workloads, oracles, and search budgets differ. Papers commonly evaluate systems
selected for suitability and rarely report negative replications. “No public evidence” therefore
means unknown, not unused. URLs can also move after this research date; DOI, publisher, and repository
links are preferred where available.

## Executive decision brief

The research supports the core direction in the Umpire 4 specification, with important limits:

1. **Keep the semantic model above the implementation, but continuously test the gap.** AWS reports
   that TLA+ found design errors that testing had not, while also warning that formal methods reason
   about models rather than systems ([AWS experience report](https://www.amazon.science/publications/how-amazon-web-services-uses-formal-methods)).
   Cedar supplies the closest Lean precedent: prove properties of an executable Lean model, then use
   differential randomized testing against the production implementation
   ([Cedar VGD paper](https://arxiv.org/abs/2407.01688),
   [cedar-spec](https://github.com/cedar-policy/cedar-spec)).

2. **Make model-to-implementation correspondence a product, not glue.** MongoDB cancelled one
   server trace-checking effort when mapping a large concurrent implementation to an overly
   abstract pre-existing model cost more than it was worth, yet succeeded with model-generated
   tests for the smaller Realm Sync algorithm and reached full branch coverage of that algorithm
   ([first-party retrospective](https://emptysqua.re/blog/extreme-modelling-in-practice/),
   [VLDB paper](https://www.vldb.org/pvldb/vol13/p1346-davis.pdf)). TraceLink later automated much of
   the trace mapping in a compiler-controlled setting and exposed nine previously undetected model,
   compiler, instrumentation, and environment-assumption defects
   ([TraceLink paper](https://doi.org/10.1145/3763128)).

3. **Use multiple assurance methods without flattening them into one verdict.** TLA+/TLC,
   Apalache, Veil, Lean proofs, randomized exploration, real executions, and runtime evidence have
   different scopes and trust bases. Apalache explicitly calls bounded model checking incomplete
   beyond the chosen trace length ([Apalache documentation](https://apalache-mc.org/docs/apalache/running.html));
   Coyote explicitly says systematic testing is not theorem-proving verification
   ([Coyote overview](https://microsoft.github.io/coyote/)). This supports Umpire's distinct
   Assurance Methods and Stage Statuses rather than a single `verified` bit.

4. **Treat deterministic replay as a debugging primitive, not a coverage claim.** FoundationDB
   runs a whole cluster deterministically in one process under generated faults
   ([FoundationDB testing documentation](https://apple.github.io/foundationdb/testing.html));
   TigerBeetle emphasizes that a seed makes a failed run reproducible while separately describing
   randomized simulation as the exploration mechanism
   ([TigerBeetle architecture](https://github.com/tigerbeetle/tigerbeetle/blob/main/docs/ARCHITECTURE.md)).

5. **Make faults first-class requests with separate realization evidence.** Filibuster synthesizes
   partial-failure tests at service-call sites, lineage-driven fault injection searches faults that
   can invalidate a successful outcome, and Mallory uses causal behavior summaries to guide faults
   ([Filibuster paper](https://christophermeiklejohn.com/publications/filibuster-socc-2021.pdf),
   [Molly paper](https://people.ucsc.edu/~palvaro/molly.pdf),
   [Mallory paper](https://arxiv.org/abs/2305.02601)). None of these makes the request to inject a
   fault proof that the intended fault occurred; Umpire's Execution Receipts are therefore a
   necessary strengthening.

6. **Make evidence completeness and causal order part of the oracle.** Lamport's happens-before
   relation is a partial order, not a synchronized-wall-clock order
   ([original paper](https://lamport.azurewebsites.net/pubs/time-clocks.pdf)); OpenTelemetry may
   deliberately drop spans through sampling ([OpenTelemetry tracing specification](https://opentelemetry.io/docs/specs/otel/trace/sdk/)).
   Missing telemetry cannot safely prove absence, and timestamps cannot safely create causality.

7. **Let model meaning guide exploration.** A 2025 OOPSLA study used abstract model-state coverage
   to guide distributed-system fuzzing, outperformed random, line-coverage, and trace-coverage
   guidance in its experiments, and reported 12 previously unknown implementation bugs, four found
   only by model-guided fuzzing ([ModelFuzz paper and artifact record](https://repository.tudelft.nl/record/uuid%3A66d18d3c-fead-4df0-8310-5df11370db13)).
   This directly supports model-owned semantic coverage rather than treating Go line coverage as
   Umpire's exploration objective.

8. **Optimize for ordinary engineers maintaining models every week.** AWS's ShardStore team used
   small executable reference models, property-specific checkers, and engineering-team ownership;
   its paper reports that the checks prevented 16 issues from reaching production
   ([ShardStore paper](https://www.amazon.science/publications/using-lightweight-formal-methods-to-validate-a-key-value-storage-node-in-amazon-s3)).
   Long-running industrial surveys likewise identify tooling, examples, education, and workflow
   integration—not only logic—as adoption barriers
   ([NASA roundtable](https://shemesh.larc.nasa.gov/fm/fm-paper-ieee-roundtable.html),
   [industrial-practice review](https://epubs.stfc.ac.uk/manifestation/48912103/STFC-AAM-2021-007.pdf)).

## Claims discipline: non-equivalences Umpire must preserve

The original five distinctions remain foundational. The additional distinctions below prevent
several adjacent forms of claim inflation exposed by specification testing, runtime verification,
history checking, provenance, and assurance-case practice.

| Non-equivalence | External evidence | Umpire consequence |
| --- | --- | --- |
| **Model correctness is not implementation conformance.** | AWS cautions that the model must capture the significant aspects of the real system ([AWS](https://www.amazon.science/publications/how-amazon-web-services-uses-formal-methods)); MongoDB's two conformance efforts show that maintaining the bridge is its own engineering problem ([MongoDB](https://emptysqua.re/blog/extreme-modelling-in-practice/)); TraceLink found incorrect model assumptions even after the models passed checking ([TraceLink](https://doi.org/10.1145/3763128)). | Preserve Feature meaning, System meaning, checked Implementation Links, Observation mappings, and Evidence Links as separately reviewable obligations (`SEM-08`, `VER-03`, `EVD-09`). |
| **Bounded search is not proof outside the bound.** | Apalache says a counterexample longer than `k` is missed by bounded model checking ([docs](https://apalache-mc.org/docs/apalache/running.html)); TLC checks finite instances of a generally expressive TLA+ specification ([TLA+ overview paper](https://lamport.azurewebsites.net/pubs/spec-and-verifying.pdf)). | `Limit Reached` must not mean no counterexample; exhaustive claims require the exact finite domain and successful completion (`PLN-01`, `PLN-03`, `PLN-04`, `VER-06`). |
| **Deterministic replay is not exhaustive exploration.** | TigerBeetle says determinism lets a failing seed reproduce the same physical path, while its simulator still relies on workloads and randomized testing to discover paths ([architecture](https://github.com/tigerbeetle/tigerbeetle/blob/main/docs/ARCHITECTURE.md)); Coyote runs many controlled iterations and emits a reproducible trace when one fails ([using Coyote](https://microsoft.github.io/coyote/get-started/using-coyote/)). | Report replay and exploration separately. An Exact Replay receipt establishes reproducibility of one trace, not coverage of the space (`EXP-01`, `EXP-03`, `EXP-05`). |
| **Successful execution is not Property satisfaction.** | Model-based testing literature separates the generated test sequence from the model-derived oracle and verdict ([Spec Explorer introduction](https://learn.microsoft.com/en-us/archive/msdn-magazine/2013/december/model-based-testing-an-introduction-to-model-based-testing-and-spec-explorer)); Temporal's current feature runner can execute, replay, and compare histories, but those checks are purpose-specific rather than a universal product-semantic proof ([Temporal features](https://github.com/temporalio/features)). | Execution reports attempts and outcomes; Observation and Property evaluation decide what the run proves (`EVD-02`, `EVD-05`). |
| **A requested Action or fault is not evidence it occurred.** | Fault-injection systems generate or schedule fault cases at interception points ([Filibuster](https://christophermeiklejohn.com/publications/filibuster-socc-2021.pdf)); controlled execution systems distinguish scheduling decisions from the actual observed run ([Coyote](https://microsoft.github.io/coyote/get-started/using-coyote/)). | Require an identity- and trace-point-bound Execution Receipt before the Action or fault can establish a Model Fact (`EVD-04`, `EVD-06`). |
| **A proof that code refines a specification is not proof that the specification is adequate.** | IronSpec found ten specification bugs across all six evaluated real-world verified systems ([USENIX](https://www.usenix.org/conference/osdi24/presentation/goldweber)); TraceLink found environment and model assumptions that internal model checking could not expose ([TraceLink](https://doi.org/10.1145/3763128)). | Test the authority itself with sanity properties, witnesses, mutations, independent Feature/System authoring, and live conformance (`SEM-01`, `SEM-08`, `AUT-03`, `EXP-05`). Record specification adequacy work separately from `VER-06` proof receipts. |
| **Property satisfaction is not non-vacuous satisfaction.** | IBM's vacuity work reports that valid temporal formulas can hide an unsatisfiable antecedent and describes “interesting witnesses” for non-trivial validity ([IBM](https://research.ibm.com/publications/efficient-detection-of-vacuity-in-temporal-model-checking)). | Require positive trigger witnesses for implication and progress Properties; retain `unsatisfiable` and antecedent-unreachable as distinct authoring failures (`PLN-05`, `SEM-09`, `AUT-03`). |
| **The same Test Plan is not the same evidence strength.** | Elle states which anomalies an observed history can detect; OpenTelemetry permits sampling; assurance-case guidance distinguishes evidence existence from its integrity ([Elle](https://github.com/jepsen-io/elle), [OpenTelemetry](https://opentelemetry.io/docs/specs/otel/trace/sdk/), [GSN](https://www.faa.gov/about/office_org/headquarters_offices/ang/redac/redac-sas-201503-gsn-community-standard-v1.pdf)). | `ART-05` preserves experimental intent, while `EVD-03`, `EVD-04`, and `QLF-03` must still qualify each environment's Evidence policy, gaps, and source trust independently. |
| **Code coverage is not semantic coverage, and neither is evidence coverage.** | Realm Sync's branch-coverage result measured exercised implementation branches; ModelFuzz separately measured abstract model states and found bugs missed by line- and trace-guided baselines ([Realm Sync](https://www.vldb.org/pvldb/vol13/p1346-davis.pdf), [ModelFuzz](https://repository.tudelft.nl/record/uuid%3A66d18d3c-fead-4df0-8310-5df11370db13)). | Report source branches, Model Coordinates, Property clauses, fault intents, and Evidence closure as different metrics (`EXP-02`, `EXP-03`, `EVD-03`). None alone is a correctness claim. |
| **A complete observed history is not a complete system history.** | Porcupine decides linearizability of supplied call/return histories against a sequential model; Elle detects particular transactional cycles from client-visible operations ([Porcupine](https://github.com/anishathalye/porcupine), [Elle](https://raw.githubusercontent.com/jepsen-io/elle/master/paper/elle.pdf)). | A history-check result applies to the recorded operations and model. It does not establish unobserved worker, persistence, cleanup, or authorization behavior (`EVD-03`, `EVD-08`, `QLF-03`). |
| **A finite run can refute safety but normally cannot prove unbounded liveness.** | Safety violations have finite bad prefixes; general liveness concerns infinite behavior ([Alpern and Schneider](https://research.ibm.com/publications/recognizing-safety-and-liveness)). Runtime monitorability excludes many properties without finite decisive prefixes ([monitorability study](https://doi.org/10.1016/j.tcs.2014.02.052)). | Runtime Properties must be bounded-response claims with explicit units when a finite Execution is the evidence; unbounded liveness needs a distinct proof and fairness assumptions (`SEM-09`, `PLN-01`, `VER-06`). |
| **Artifact checksum is not authenticity; provenance is not semantic validity.** | W3C PROV records origins and derivations for trust assessment but does not make the derived content true ([PROV](https://www.w3.org/TR/prov-overview/)); in-toto and SLSA address supply-chain provenance and integrity rather than domain semantics ([in-toto](https://github.com/in-toto/docs/blob/master/in-toto-spec.md), [SLSA](https://slsa.dev/spec/v1.2/)). | `ART-02` checks identity and freshness. Signatures/attestations may authenticate producers. Only Run Evaluation can establish Model Facts and Properties (`EVD-02`, `EVD-09`, `QLF-03`). |
| **Evidence is not an assessment, and an assessment is not a deployment decision.** | The RATS architecture separates Evidence, appraisal policy, Attestation Result, and the relying party's decision ([RFC 9334](https://www.rfc-editor.org/rfc/rfc9334.html)); assurance cases join claims, argument, assumptions, and evidence ([NIST](https://csrc.nist.gov/glossary/term/assurance_case)). | Keep raw Evidence, Observation/Property evaluation, Claim Assessment, and environment rollout policy distinct (`EVD-02`, `EVD-05`, `QLF-02`, `QLF-03`). |
| **Workflow-net soundness is not Temporal product correctness.** | Workflow-net soundness addresses proper completion, dead transitions, and residual tokens, while expressive extensions can make analysis much harder or undecidable ([classification](https://www.vdaalst.com/publications/p628.pdf)). | Use workflow theory as a reusable structural Property library, not as Feature semantics. Cancellation, retries, timeouts, Nexus behavior, and Evidence meaning remain explicit model obligations (`SCP-01`, `SEM-01`, `MOD-06`). |

**Inference for Umpire.** Every CLI, artifact, Result, and CI summary should name exactly one stage
and one Assurance Method. Avoid umbrella phrases such as “the scenario is verified.” Prefer precise
statements such as “the finite Lean target was exhaustively searched under these Limits,” “this run
produced a receipt for the requested fault,” or “qualified evidence established this Feature fact.”

## Comparative matrices by Umpire seam

The matrices compare roles, not overall tool quality. `Primary` means the method directly addresses
the seam; `Supporting` means it contributes useful evidence; `No` means another method must own the
seam. A method may be excellent while intentionally covering only one column.

### Method-to-seam fit

| Method family | Semantic authority | Exploration | Planning / Artifacts | Execution control | Evidence interpretation | Replay / promotion | Optional proof | Developer adoption |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| TLA+/TLC, Alloy, SPIN | Primary design model | Primary, finite/bounded | Supporting counterexample traces; portability varies | No | No | Supporting | No kernel proof | Strongest where models stay small and design-facing |
| Apalache, Veil, Ivy | Consumes a formal authority, but translation must be checked | Primary symbolic/invariant checking | Supporting backend receipts | No | No | Counterexample must replay canonically | Primary, with different solver/trust classes | Specialist; backend assumptions must remain visible |
| Quint + Quint Connect | Quint is authority in that ecosystem | Primary simulation/checking | Primary trace transfer for supported drivers | Supporting driver control | Supporting state comparison | Seeded replay | No kernel proof | Familiar syntax helps; bridge maintenance remains material |
| P / PObserve | P model/monitors are authority in that ecosystem | Primary systematic scheduling | Supporting error traces | Primary for P-controlled events | Promising runtime log monitoring; public evaluation limited | Primary within controlled P traces | No kernel proof | Documented AWS use; a second Umpire DSL would violate `SCP-03` |
| Cedar VGD | Lean model is semantic authority | Supporting randomized generation | Primary generated cases | Primary for pure Rust comparison | Primary function-level DRT | Retains failing inputs | Primary Lean proofs | Strong precedent, but less distributed Evidence complexity |
| TraceLink / TLA+ trace validation | Source model is authority | No new exploration | Supporting mapping metadata | No | Primary trace admission | Primary concrete trace replay | No implementation proof | Compiler/instrumentation control reduces mapping burden |
| FoundationDB, TigerBeetle, Antithesis | Implementation assertions/oracles, not product semantics by default | Primary randomized/guided execution | Primary seeds and execution inputs | Primary simulated environment | Supporting; oracle-specific | Primary deterministic replay | No | High payoff where determinism is architectural or virtualized |
| CHESS, Coyote, Shuttle | Implementation assertions, not Feature semantics | Primary schedule exploration | Primary schedules/seeds | Primary intercepted concurrency | Supporting | Primary schedule replay | No | Low-friction only for supported runtimes/nondeterminism |
| MaceMC, MODIST, SAMC/FlyMC | Implementation plus supplied checker/oracle | Primary event/fault interleavings | Primary controlled-event traces | Primary at interposition points | Supporting; usually property-specific | Primary | No | Retrofitting exposes coverage but incurs hooks/state explosion |
| Jepsen, Elle, Porcupine, Troubadour | Supplied abstract operation/transaction model | No or workload-specific | Primary client histories | Supporting workload/fault control | Primary history checking | Primary history replay | Solver/decision procedure, not code proof | Valuable black-box seam with detectability limits |
| Molly, Filibuster, Mallory, FATE/DESTINI | Existing oracle/test semantics | Primary fault selection | Primary fault scenarios | Primary at supported sites | Supporting lineage/causal feedback | Supporting | Bounded/configuration-specific at best | Useful only if faults and outcomes are actually receipted |
| IronFleet, Verdi, Grove, Anvil, Verus | Formal spec/refinement chain | No campaign exploration | Proof artifacts | Verified implementation or extracted code, not test control | No live-Evidence oracle by default | No runtime promotion loop | Primary | Highest assurance, highest specialization/proof maintenance |
| W3C PROV, in-toto, SLSA, RATS, SACM/GSN | No behavioral semantics | No | Primary provenance/claim packaging | No | Supporting source/integrity qualification | Supporting artifact lineage | No domain proof | Mature vocabulary; dangerous if mistaken for semantic validity |

### Required decision at each Umpire seam

| Umpire seam | What must remain authoritative | Concrete mechanism to copy | Dominant failure mode | Required receipt / metric | Governing rules |
| --- | --- | --- | --- | --- | --- |
| Semantic authority | Handwritten Feature/System Model Definitions | Cedar-style model governance; independent Feature/System ownership; mutation and witness checks | Model is internally consistent but wrong, weak, or implementation-shaped | Definition IDs, fingerprints, sanity witnesses, model mutation score | `SEM-01`–`SEM-08`, `AUT-03`, `AUT-04` |
| Exploration | Lean-declared Behavior, Query, scoring, and Limits | TLC/Alloy counterexamples; ModelFuzz semantic feedback; Molly lineage pruning | State explosion, biased search, silent truncation | exact strategy/seed/Limit, semantic-coordinate coverage, completion status | `PLN-01`–`PLN-05`, `EXP-01`–`EXP-04` |
| Planning / Artifacts | Canonical Lean interpretation of serializable data | Quint/ITF-style portable traces plus in-toto-like provenance | Generated driver becomes a second semantics; stale readers accept changed meaning | artifact checksum, schema version, fingerprints, Known Gaps, migration ID | `AUT-05`, `ART-01`–`ART-07`, `PLN-06` |
| Execution control | Thin environment adapter consuming the Test Plan | Coyote/DST interception contracts and exact seeds | Uncontrolled entropy or a scheduled Action/fault never takes effect | controlled/recorded/unsupported nondeterminism inventory; per-request receipt rate | `EVD-01`, `EVD-06`, `QLF-01`, `QLF-02` |
| Evidence interpretation | Lean Observation and Implementation Links | TraceLink causal mappings; Elle detectability declarations; partial-order RV | missing/reordered/ambiguous telemetry silently proves absence | source/identity/causal closure; ambiguity and loss counts; Evidence Link | `SEM-08`, `EVD-02`–`EVD-09` |
| Replay / promotion | Exact referenced model and human-reviewed Regression | DST schedule replay, model-aware shrinking, Cedar retained counterexamples | replay uses changed semantics; shrink removes the causal/evidence core | exact-replay rate, reproduction rate, reduction ratio, reviewed cause class | `EXP-01`, `EXP-04`, `EXP-05`, `VER-05` |
| Optional proof | Checked Target/Property with an explicit translation | Veil/Apalache-style backend receipt; IronFleet/Grove explicit TCB | solver result flattened into “verified”; assumptions or unsupported constructs hidden | Assurance Method, solver/proof versions, assumptions, Limits, TCB | `MOD-05`, `VER-01`–`VER-06` |
| Developer adoption | Temporal engineers owning ordinary model changes | AWS “debugging designs,” Cedar RFC gate, ShardStore co-review | specialist-owned model drifts; diagnostics are unusable; CI becomes too slow | median edit-to-feedback time, non-specialist ownership, semantic-review latency, flaky-run rate | `AUT-01`, `MOD-06`–`MOD-08`, `SCP-01` |

**Inference for Umpire.** No existing project is primary in every column. Umpire's opportunity is the
checked composition of seams; its chief risk is allowing the boundary between two good tools to
become an untested trust gap. Each seam therefore needs its own fixture, negative control, status,
receipt, and owner before end-to-end “green” is meaningful.

## Landscape at a glance

| Project or practice | Publicly documented outcome/status | What it establishes | Main lesson for Umpire |
| --- | --- | --- | --- |
| TLA+ at AWS | **Documented industrial adoption at the 2014 report date.** AWS says seven teams had used TLA+ across critical distributed-system designs since 2011 ([paper](https://www.amazon.science/publications/how-amazon-web-services-uses-formal-methods)). This source alone does not establish 2026 usage. | High-level design properties for finite model instances. | Copy early design debugging, simple state-machine models, and counterexample-first workflow; do not infer implementation conformance. |
| P at AWS / P# and Coyote at Azure | **Sustained production adoption, with ecosystem constraints.** P's maintainers list use across S3, EBS, DynamoDB, MemoryDB, Aurora, EC2, and IoT ([P](https://p-org.github.io/P/)); the P# report describes several Azure services ([paper](https://arxiv.org/abs/2002.04903)). | Systematic exploration of communicating state machines or controlled async code, with reproducible schedules. | Familiar state-machine authoring and immediate executable feedback matter. Preserve Umpire's one-language rule instead of importing another DSL. |
| Cedar | **Maintained product-development practice.** The project currently maintains a Lean semantics/proof repository and DRT against the Rust implementation ([cedar-spec](https://github.com/cedar-policy/cedar-spec), [security design](https://github.com/cedar-policy/cedar-docs/blob/main/docs/collections/_other/security.md)). | Theorems about the Lean model plus sampled implementation-model equivalence checks. | Closest precedent for Lean authority, differential execution, explicit unmodeled areas, and governance that updates model and DRT with features. |
| Alloy Analyzer | **Maintained mature tool; cases vary.** Alloy performs automatic relational instance/counterexample search inside a user-set finite scope ([language reference](https://alloytools.org/download/alloy-language-reference.pdf)); the small-scope hypothesis is empirical, not a proof beyond that scope ([study](https://groups.csail.mit.edu/sdg/pubs/2002/SSH.pdf)). | Structural and relational design errors in compact finite universes. | Copy small counterexamples and explicit scope; use only as conceptual guidance because `SCP-03` reserves Lean as Umpire's model language. |
| SPIN / Promela | **Mature tool with shipped industrial case studies.** SPIN checks asynchronous process models and supplies explicit state-space reduction modes ([official overview](https://spinroot.com/gerard/pdf/Advances2005.pdf)). | Safety/liveness properties over the Promela model under the selected search configuration. | Long-lived tooling and partial-order reduction are instructive; search modes that trade completeness for memory must be explicit `VER-06` methods. |
| Quint / Quint Connect | **Maintained tool; early shipped MBT cases and explicit maintenance caution.** Quint supplies typed TLA-style simulation/checking; Connect replays model traces against Rust, while its first-party launch account says even its authors struggled with MBT bridge cost ([Connect](https://github.com/quint-co/quint-connect), [retrospective](https://quint.sh/posts/quint_connect)). | Design checking plus trace-derived conformance tests for supported Rust drivers. | Copy approachable diagnostics and portable traces; Umpire should automate its bridge while retaining Lean authority and not overstate early case-study metrics. |
| MongoDB server trace checking | **Cancelled by its authors.** The effort became more expensive than its expected value ([retrospective](https://emptysqua.re/blog/extreme-modelling-in-practice/)). | Intended conformance of observed C++ traces to an existing TLA+ model. | Start from a narrow traceable seam; avoid retrofitting a very abstract model over arbitrary internal concurrency. |
| MongoDB Realm Sync model-generated tests | **Shipped case study.** The authors report full branch coverage of the specified algorithm ([retrospective](https://emptysqua.re/blog/extreme-modelling-in-practice/), [paper](https://www.vldb.org/pvldb/vol13/p1346-davis.pdf)). | Implementation behavior over all enumerated model behaviors in the study's scope. | A closed finite target and test-generation seam can pay off sooner than whole-server trace conformance. |
| TraceLink / PGo | **Research demonstrator; constrained by compiler-controlled tracing and a stated TCB.** It found nine defects and validated traces up to 100,000 events in its evaluation ([paper](https://doi.org/10.1145/3763128), [artifact](https://zenodo.org/records/15151497)). | Whether recorded implementation traces are admitted by the source model. | Make mappings, causal metadata, and trusted components explicit; validate instrumentation itself; separate finite checking abstractions from runtime validation domains. |
| TLA+ trace validation | **Research demonstrator plus official-tool workflow.** Instrumented programs record specification-variable updates and TLC validates constrained traces; evaluated discrepancies included model and implementation issues ([paper](https://arxiv.org/abs/2404.16075), [TLC guide](https://docs.tlapl.us/using%3Atlc%3Atrace_validation)). | Whether a partial concrete trace can be extended to a model behavior. | Explicitly record which variables/events are missing; partial observability can be sound only under a precise completion relation, not ad hoc event matching. |
| ShardStore | **Shipped case study with team ownership documented in the paper, deliberately lightweight.** The paper reports 16 prevented issues, small checker/model overhead, and contributions by non-formal-methods engineers ([paper](https://www.amazon.science/publications/using-lightweight-formal-methods-to-validate-a-key-value-storage-node-in-amazon-s3)). | Property-specific validation of a production storage node, not full verification. | Decompose correctness and choose the cheapest sufficient method per property. Copy co-review and ownership discipline, not the same-language placement that would violate Lean authority. |
| FoundationDB simulation | **Sustained production adoption.** Deterministic whole-cluster simulation is part of the product's test practice ([docs](https://apple.github.io/foundationdb/testing.html), [SIGMOD paper](https://www.foundationdb.org/files/fdb-paper.pdf)). | Real implementation behavior under simulated workloads, time, network, process, and storage faults. | Determinism and fault virtualization are powerful, but they require architectural investment and still need trustworthy workload oracles. |
| TigerBeetle VOPR | **Sustained production adoption.** The project runs the real cluster code in a single-threaded deterministic simulator ([architecture](https://github.com/tigerbeetle/tigerbeetle/blob/main/docs/ARCHITECTURE.md)). | Real-code safety checks over randomized simulated runs. | Keep real-code execution complementary to model checks; retain seeds and exact inputs; distinguish “many behaviors” from all behaviors. |
| Antithesis | **Commercial sustained-use claims; technical platform documented, independent outcome evidence varies by case.** It executes ordinary software in a deterministic hypervisor, branches timelines, injects faults, and searches for property violations ([architecture](https://antithesis.com/docs/introduction/how_antithesis_works/)). | Reproducible whole-system executions under a virtualized environment and supplied properties. | Potential future backend for the same `ExperimentSpec`; first prove receipts, identity, and Evidence export rather than equating platform determinism with Umpire semantics. |
| Shuttle / Turmoil / MadSim | **Maintained tools with first-party project use; outcome evidence is case-specific.** Shuttle controls supported Rust concurrency; Turmoil and MadSim provide seeded async/network simulation ([Shuttle](https://docs.rs/shuttle/latest/shuttle/scheduler/index.html), [Turmoil](https://github.com/tokio-rs/turmoil), [MadSim](https://github.com/madsim-rs/madsim)). | Reproducible schedules or simulated network behavior for code using supported interception surfaces. | The decisive question is uncontrolled entropy and substituted dependencies, not language popularity; require a backend capability receipt. |
| CHESS / Coyote | **CHESS historically adopted inside Microsoft; Coyote is the maintained successor line.** CHESS found bugs in code already stress-tested for months ([CHESS paper](https://www.microsoft.com/en-us/research/publication/chess-a-systematic-testing-tool-for-concurrent-software/)); Coyote documents Azure use ([repository](https://github.com/microsoft/coyote)). | Controlled thread/task schedules for bounded test scenarios. | Systematic scheduling beats sleep-based stress tests, but only for intercepted sources of nondeterminism and bounded iterations. |
| Jepsen / Elle | **Sustained black-box testing toolkit with shipped studies.** Elle checks observed histories using cycle inference and documents which anomalies are and are not detectable ([repository](https://github.com/jepsen-io/elle), [paper](https://raw.githubusercontent.com/jepsen-io/elle/master/paper/elle.pdf)). | Evidence of consistency violations in concrete client histories; not system-wide correctness. | Black-box checks are valuable staging/canary complements. Make observable operations, completeness, and inference limits explicit. |
| Porcupine | **Sustained tool with documented production-system users.** It checks supplied concurrent histories for linearizability against executable sequential Go models and reports state-space-explosion limits ([repository](https://github.com/anishathalye/porcupine)). | Linearizability of one recorded history/model, not history generation or whole-system behavior. | A useful specialized Evidence adapter; keep its Go model from becoming Umpire behavioral authority. |
| Troubadour | **Research demonstrator.** Its SMT encoding checks whether transaction observations have a correct execution under specified isolation; the evaluation reports two new bugs in PostgreSQL/an industrial system ([paper](https://doi.org/10.1145/3720504)). | Observational correctness for the supplied transactions, observations, semantics, and isolation model. | A stronger analogue than linearizability for semantically rich observations; still bounded by what transactions and values the Evidence exposes. |
| Filibuster | **Research demonstrator and subsequent JVM implementation.** The original study evaluated four industrial-derived microservice bug scenarios ([paper](https://christophermeiklejohn.com/publications/filibuster-socc-2021.pdf), [JVM repository](https://github.com/filibuster-testing/filibuster-java-instrumentation)). | Resilience of functional-test paths under synthesized RPC/database failure cases. | Generate failure variants from existing tests, but do not let functional tests become the semantic source of truth. |
| Molly / lineage-driven fault injection | **Research demonstrator.** Molly uses data lineage and satisfiability to target fault combinations ([paper](https://people.ucsc.edu/~palvaro/molly.pdf)). | Whether a successful outcome remains derivable after selected omissions/failures. | Evidence lineage can prune fault space; Umpire's Evidence Links could provide a semantically stronger lineage substrate. |
| FATE / DESTINI | **Research demonstrator with broad case evaluation.** The authors report 40,000 scenarios, 74 recovery specifications, 16 new bugs, and 51 reproduced old bugs across HDFS, ZooKeeper, and Cassandra ([paper](https://www.usenix.org/conference/nsdi11/fate-and-destini-framework-cloud-recovery-testing)). | Recovery behavior under injected failures and declared recovery invariants. | Separate fault enumeration from recovery semantics, and require realized-fault plus cleanup receipts; raw scenario count is not semantic coverage. |
| Mallory | **Research demonstrator.** The authors report causal-summary-guided fault selection and confirmed bugs/CVEs across evaluated systems ([paper](https://arxiv.org/abs/2305.02601), [artifact repository](https://github.com/dsfuzz/mallory)). | Behavior-novelty-guided fault exploration of live distributed implementations. | Use causal semantic novelty to prioritize experiments, while keeping the policy and its Limits inspectable. |
| ModelFuzz | **2025 research demonstrator.** It uses TLA+ model-state coverage to guide real implementations and reports 12 new bugs ([published record](https://repository.tudelft.nl/record/uuid%3A66d18d3c-fead-4df0-8310-5df11370db13)). | Model-guided schedule/fault exploration, not proof. | Direct evidence for `EXP-02`: make Model Coordinates and semantic transitions the feedback signal. |
| MET for CRDTs | **Research demonstrator used in the authors' CRDT development.** MET model-checks TLA+ designs, generates implementation tests, and controls message reordering ([paper](https://arxiv.org/abs/2204.14129)). | Design and implementation defects reachable through generated model traces and controlled delivery permutations. | Strong precedent for one trace feeding both design checking and implementation exploration; validate the driver and keep finite trace coverage distinct from proof. |
| MaceMC / MODIST / SAMC / FlyMC | **Research demonstrators with mature-system evaluations.** MODIST reports 35 bugs across three systems; SAMC uses semantic-aware prioritization; FlyMC reports systematic reductions and confirmed bugs ([MODIST](https://www.usenix.org/legacy/events/nsdi09/tech/full_papers/yang/yang_html/), [SAMC](https://www.usenix.org/conference/osdi14/technical-sessions/presentation/leesatapornwongsa), [FlyMC](https://ucare.cs.uchicago.edu/pdf/eurosys19-flyMC.pdf)). | Implementation interleavings and faults exposed through language/runtime hooks or interposition. | Retrofitting can find real bugs but remains state-space- and hook-limited. Reuse semantic prioritization without making execution control the source of product meaning. |
| IronFleet | **Constrained verified-implementation research case study.** It verified safety and liveness of two new Dafny distributed systems with competitive reference performance ([paper](https://www.microsoft.com/en-us/research/wp-content/uploads/2015/10/ironfleet.pdf)). | Machine-checked refinement from centralized spec to implementation under explicit assumptions. | Deep end-to-end proof is possible, but the authors emphasize new verification-friendly code and considerable developer assistance; it is not a near-term method for all existing Temporal Go. |
| Verdi | **Constrained verified-implementation research case study.** It verified Raft linearizability and extracted OCaml handlers joined to runtime shims ([paper](https://homes.cs.washington.edu/~mernst/pubs/verify-distsystem-pldi2015-abstract.html), [repository](https://github.com/uwplse/verdi)). | Proofs under explicit network/fault semantics, transferred through verified system transformers. | Capability contracts and fault-model transforms are worth copying; runtime shims remain part of the trusted boundary. |
| Grove / Perennial | **Active research framework with verified Go-subset case studies.** Grove verifies distributed components using Goose's model of a subset of Go ([Perennial](https://github.com/mit-pdos/perennial), [Grove paper](https://iris-project.org/pdfs/2023-sosp-grove.pdf)). | Program-level proofs for selected Go components under a formal network and library model. | A future narrow module-level proof path is plausible; whole-repository translation is neither necessary nor supported by this evidence. |
| Anvil | **Research demonstrator with deployable verified controllers.** Anvil verifies safety/liveness of new Rust Kubernetes controllers against a Verus/TLA cluster model and reports four deep bugs found during verification ([paper](https://www.usenix.org/system/files/osdi24-sun-xudong.pdf), [repository](https://github.com/anvil-verifier/anvil)). | Refinement/liveness of selected controller code under an explicit Kubernetes/environment model. | Closest workflow-controller proof analogue: useful for a future narrow new component, not evidence that arbitrary existing Temporal Go is verified. |
| Verus / verified IronKV | **Research demonstrator, explicitly partial relative to IronFleet.** The port verifies IronKV's host program but says it omits IronFleet's distributed refinement layer ([repository](https://github.com/verus-lang/verified-ironkv), [Verus paper](https://www.microsoft.com/en-us/research/uploads/prod/2024/09/Verus.pdf)). | Implementation-level Rust obligations for the port, not the complete distributed end-to-end theorem. | Never infer a stronger transitive claim than the artifact states; each missing refinement layer is a Known Gap and separate Assurance Method. |
| Aneris / Iris | **Active research framework.** Aneris verifies safety properties of distributed programs over unreliable UDP-like communication and includes CRDT/causal-memory case studies ([project](https://iris-project.org/aneris/)). | Modular program proofs under a formal language/network semantics. | Capability-contract decomposition and explicit communication assumptions are useful future proof patterns; not a practical whole-Temporal route today. |
| Veil | **Promising, early, and optional.** Veil is embedded in Lean 4, targets automated and interactive safety proofs, and says liveness is future work; a 2.0 preview is under development ([repository](https://github.com/verse-lab/veil), [CAV paper](https://verse-lab.github.io/papers/veil-cav25.pdf)). | Safety of transition-system specifications in supported logical fragments, with interactive Lean fallback. | Keep the opt-in boundary and distinct solver trust class already required by Umpire; do not make Veil vocabulary leak into ordinary models. |
| Apalache | **Maintained but documented as experimental.** It provides symbolic TLA+ checking and inductiveness workflows ([docs](https://apalache-mc.org/docs/apalache/index.html)). | Bounded or inductive checks through SMT under typed, finite-data assumptions. | Optional backends can expand scale, but receipts must expose solver, bound, assumptions, and whether the result was bounded or inductive. |
| Ivy | **Research tool with real protocol proofs.** Ivy combines decidable fragments, interactive invariant discovery, and refinement ([official site](https://microsoft.github.io/ivy/)); Stellar publishes Ivy safety and liveness proof artifacts ([Stellar proofs](https://github.com/stellar/scp-proofs)). | Parameterized protocol proof under the authored abstraction. | Automated invariant discovery is useful, but a second public modeling language would conflict with Umpire's one-language rule. |
| Workflow nets / BPMN formalization | **Sustained research lineage; product applicability depends on mapping.** Workflow-net soundness has multiple decidable notions, but extensions can become undecidable; BPMN analyses depend on a semantics-preserving translation ([soundness](https://www.vdaalst.com/publications/p628.pdf), [BPMN mapping](https://doi.org/10.1016/j.scico.2018.05.008)). | Structural completion/dead-transition properties of the formal workflow representation. | Mine reusable structural Properties and counterexamples, but do not translate Temporal workflows wholesale or mistake translation correctness for Feature semantics. |

### Case dossiers: scope, trust, cost, failure mode, and rule-level decision

Published numbers are included only to characterize the cited project. They are not normalized
benchmarks and must not be used to predict Umpire's bug yield or staffing.

| Case | Concrete scope and documented result | Trust boundary and assumptions | Cost / maintenance evidence | Transferable failure mode | Exact Umpire decision |
| --- | --- | --- | --- | --- | --- |
| AWS TLA+ | Seven teams in the 2014 report; design models for S3, DynamoDB, EBS, and a lock service found subtle protocol/design errors. | The authored abstraction, finite constants, TLC/toolchain, and the judgment that modeled aspects are significant. No code conformance. | The source emphasizes small models and engineer usability but supplies no comparable total engineer-week figure. | A correct finite model omits a production mechanism or implementation diverges later. | Keep Feature semantics implementation-independent (`SEM-01`, `MOD-03`); expose exact finite Limits (`PLN-01`, `VER-06`); demand live correspondence evidence (`SEM-08`, `EVD-09`). |
| Cedar VGD | Executable Lean semantics and theorems are differentially tested against idiomatic Rust; unmodeled code receives property tests. | Lean model/spec, proved theorem statements, generators, serialization, Rust adapter, and sampled DRT inputs. | Feature governance requires applicable formal-model and DRT work; public sources do not quantify lifetime maintenance cost. | Both sides share a bad generator/encoder, or an unmodeled Rust path escapes DRT. | Copy model-change governance (`SEM-01`, `ART-07`), retain explicit Known Gaps (`QLF-03`), and make encoding/Observation negative controls mandatory (`EVD-03`, `AUT-03`). |
| MongoDB server trace checking | Large C++ server traces were mapped to an existing TLA+ model; authors cancelled the effort when cost exceeded expected value. | Existing abstract model, trace instrumentation, reconstruction of concurrent low-level events, and fuzzed executions. | First-party evidence identifies bridge complexity—not the logic—as the dominant cost; no exact engineer-hours published. | Retrofitted abstraction has no stable observable correspondence and needs continual special cases. | Do not expand beyond a model family until one Evidence profile and Implementation Link are feasible (`SEM-08`, `EVD-09`, `SCP-01`). |
| Realm Sync | Enumerated model behaviors generated C++ tests and reached full branch coverage of the specified algorithm. | Model completeness in the selected finite domain, test generator, C++ driver, branch instrumentation, and asserted state mapping. | Smaller, algorithm-shaped seam was tractable; exact lifetime cost is not reported. | Full code coverage hides a wrong model or misses behavior outside the finite enumeration. | Use `FiniteMachine` only when enumerators are authoritative (`AUT-08`); report code and semantic coverage separately (`EXP-02`, `EXP-03`). |
| TraceLink / PGo | Nine defects across model, compiler, instrumentation, and environment assumptions; validation evaluated traces up to 100,000 events. | Source model, compiler, trace instrumentation, causal metadata, runtime-to-model map, and an explicit TCB; some bug classes remain undetectable. | Compiler control automates much of the mapping; it does not demonstrate the cost for arbitrary hand-written Go. | Finite model-checking symbols reject real values; missing/misordered events or instrumentation corruption cause false admission/rejection. | Separate finite authority from general Evidence recognizer (`AUT-08`), fail closed on mapping inconsistency (`EVD-04`), and version every Evidence Link (`EVD-09`). |
| TLA+ trace validation | Instrumented executions constrain TLC; evaluated discrepancies occurred in every studied program, while official examples report CCF data-loss issues. | Logged specification-variable updates, omitted-variable semantics, TLC constraints, event order, and instrumentation correctness. | Instrumentation is manual unless compiler/tool support exists; exact maintenance figures vary by subject. | A partial trace appears valid because a missing variable lets the checker invent a completion. | Every omitted coordinate must be declared and the completion relation checked (`EVD-03`, `EVD-04`); “admitted completion exists” must be a narrower status than exact conformance (`EVD-05`). |
| FoundationDB simulation | Real cluster code executes in one deterministic process with simulated time/network/storage/process faults; product practice is documented. | Simulator implementations of the environment, deterministic interfaces, workload generator, assertions, and absence of uncontrolled nondeterminism. | Requires simulation-oriented architecture and substitutes system interfaces; no public single total maintenance number. | Simulator differs from the OS/network/storage or workload oracle never asks the bad question. | Borrow seed/input retention (`PLN-02`, `ART-06`), but keep simulator fidelity as a Known Gap and real Evidence policy (`EVD-03`, `QLF-03`). |
| Coyote / Shuttle | Supported async/concurrency operations are intercepted and repeatedly scheduled; failures produce replayable schedules. | Binary/runtime instrumentation, recognized task/sync primitives, declared nondeterministic choices, and test assertions. External/custom concurrency can escape control. | Low entry cost for supported runtimes; custom thread pools and dependencies require modeling/instrumentation. | A supposedly exhaustive scheduler missed an uncontrolled source of nondeterminism. | A backend receipt must enumerate controlled, recorded, and unsupported choices (`EVD-06`, `QLF-03`); replay never implies exploration completeness (`EXP-03`). |
| MODIST | Three mature systems totaling 237.6 KLOC; paper reports 35 previously unknown bugs, 31 developer-confirmed at publication, including ten protocol-level bugs. | Interposition layer, Windows APIs/virtual clock, deterministic failure simulator, supplied invariants, and explored event classes. | Avoids source rewrites but still depends on interposition coverage and search heuristics. | Transparent control appears complete while an API, timer, storage action, or state equivalence is outside the hook set. | Treat implementation-level checking as optional Execution/Exploration, never semantic authority (`EVD-01`, `EXP-02`); publish hook coverage and Limit status (`PLN-04`, `QLF-03`). |
| SAMC / FlyMC | SAMC guides event exploration with system semantics; FlyMC reports an average 16× and up to 78× speedup, reproduced 12 known bugs, and found ten developer-confirmed bugs across eight systems. | Instrumented communication/fault events, semantic annotations/state abstraction, independence/symmetry assumptions, checkers, and workloads. | Reductions improve search, but application integration and semantic hints are system-specific. | Unsound independence/symmetry or heuristic priority suppresses the responsible ordering. | Model-owned reduction predicates may optimize search only if their completeness effect is explicit (`EXP-02`, `PLN-03`, `VER-06`). Heuristic runs cannot claim exhaustive absence (`PLN-04`). |
| Jepsen / Elle / Porcupine | Concrete client histories are checked for selected consistency/linearizability properties; Elle scales to large transactional histories and states detectability limits. | Client operation capture, nemesis/fault harness where used, sequential/transaction model, invocation/response identity/order, and checker algorithm. | Per-system workload/model engineering dominates; checker reuse is high. | The test never generates the responsible workload, an operation is lost/misclassified, or the model is weaker than product behavior. | Make black-box profiles explicit (`QLF-03`), retain raw histories and mapping (`EVD-08`, `EVD-09`), and never infer unobserved behavior (`EVD-04`). |
| FATE / DESTINI | 40,000 scenarios and 74 specifications across HDFS, ZooKeeper, Cassandra; 16 new and 51 old bugs reported. | Injection hooks, enumerated failures, recovery specification, workload, and ability to observe recovery/cleanup. | Broad systematic campaigns; specifications and integration remain system-specific. | Scenario scheduled but fault not realized; recovery verdict ignores residual state or cleanup. | Require fault-point receipts (`EVD-06`), full lifecycle/cleanup (`EVD-08`), and separate scenario count from fault-intent/Property coverage (`EXP-02`). |
| IronFleet | Two new Dafny systems proved for safety and liveness; about 3.7 person-years, roughly 39K proof lines for 5K executable lines, and about six hours for full serial verification. | High-level spec, refinement layers, small main loop, Dafny/SMT/.NET/Windows/hardware, UDP/source integrity; liveness also assumes fairness, resources, and eventual synchrony/delivery. | Incremental checks were much faster, but the overall proof investment was substantial and code was verification-oriented. | An informal final bridge, environmental premise, or fairness assumption is omitted from the claim. | Reserve end-to-end proofs for narrow kernels (`SCP-01`, `MOD-06`); enumerate every TCB layer and separate safety/liveness Assurance Methods (`VER-04`, `VER-06`, `SEM-09`). |
| Verdi | PLDI result was initially conditional on a safety proof; later work added about 45K proof lines and 90 invariants for a runnable linearizable Raft RSM. No liveness, Byzantine faults, membership change, or log compaction. | Coq specs/proofs/extraction, OCaml compiler/runtime and network shim, modeled faults, and correspondence to physical persistence/networking. | Follow-up identifies proof maintenance as the main challenge and uses structural tactics/transfer lemmas. | A conditional theorem is reported as closed end-to-end verification; feature additions invalidate cross-cutting invariants. | Receipts must expose open premises and unsupported features (`VER-04`, `QLF-03`); favor deep capability contracts and reusable lemmas (`MOD-06`, `MOD-07`). |
| Grove | vKV safety/linearizability, exactly-once RPC, recovery, reconfiguration, and leases over a Go subset; about 12 proof lines per code line. It does not prove progress. | Goose translation, Coq/Iris/Grove, 120 LOC trusted network library, 50 LOC trusted filesystem library, and bounded-clock/TrueTime-like premise. | Adding leases changed five components while earlier protocol proofs stayed largely reusable, evidence that interface depth matters. | Safety is described as availability, or trusted clock/filesystem behavior fails. | Keep proof scope in the Claim Assessment (`QLF-03`, `VER-06`); use deep module seams (`MOD-06`, `MOD-08`) and test trusted adapters adversarially (`EVD-03`). |
| Anvil | Three runnable Rust Kubernetes controllers plus controller-specific safety/liveness; four deep bugs found during verification. | Kubernetes/environment/API specs, wrappers, Verus/Z3, compiler/OS; liveness assumes fault classes eventually stop, weak fairness, and cooperating controllers. | 28 reported Fluent Bit features averaged under a day and 47 changed lines, including 19 proof lines; this is one framework/case, not universal cost. | Trusted external API omits partial failure; an end-to-end test found exactly such a liveness bug. | Model eventual stabilization/fairness explicitly (`SEM-09`); pair proofs with live negative controls at API/failure seams (`EVD-04`, `VER-04`). |
| Verus IronKV | Verifies the Rust host program against a protocol-level host spec; explicitly omits IronFleet's protocol-to-high-level proof and does not implement crash recovery. | Top-level host spec, Verus/solvers, Rust compiler, runtime; missing distributed-refinement and recovery layers. | Port reports verification reduction from 201 to 18 seconds on eight cores and proof/code ratio from 4.2 to 2.9. | “Verified host” is summarized as “verified distributed service.” | Use layer-specific Assurance Methods and Known Gaps (`VER-06`, `QLF-03`); a missing Implementation Link prevents a Feature-level claim (`SEM-08`). |
| IronSpec | Fourteen specs, six from real verified codebases; ten author-confirmed specification bugs across all six. Sixty-one mutations survived, of which 13 represented intended behavior. | Human intent oracle, sanity checks, Spec-Testing Proofs, mutation operators, and the reviewed mapping from survivors to intended behavior. | Adds review and proof/test work; mutation survivors require judgment and are not automatically faults. | Weak/strong fields, adversary mismatch, vacuity, or mutually cancelling spec defects survive a complete code proof. | Add authoring adequacy gates (`AUT-03`, `PLN-05`), positive/negative witnesses, and separate model/Property/Observation/Link mutation scores; require human review under `GOV-02`. |
| Workflow-net soundness | Eight soundness notions are decidable for classical workflow nets, but expressive extensions often become undecidable; bounded WF-net soundness is PSPACE-complete. | Correct translation to the selected Petri-net class and the structural meaning of places/transitions/tokens. Domain outcomes and implementation are outside the theorem. | State-space and translation complexity grow with expressiveness; published workflow theory does not quantify Temporal model maintenance. | A BPMN/workflow translation changes cancellation or multi-instance semantics, or structural completion passes while product obligations fail. | Build a small Lean structural library only for recurring Temporal cases (`SCP-01`, `MOD-06`); retain Feature-specific Properties and bounded progress (`SEM-05`, `SEM-09`). |

## Detailed case studies and decisions

### 1. AWS TLA+: design debugging succeeded because the abstraction was intentional

**Documented.** AWS used TLA+ to specify distributed algorithms for services including S3,
DynamoDB, EBS, and a distributed lock manager. The authors report finding subtle design errors,
including long counterexamples, and describe formal specification as accelerating optimization and
design review. They explicitly caution that a checked model does not guarantee the implementation
or that the model captured every important aspect
([AWS experience report](https://www.amazon.science/publications/how-amazon-web-services-uses-formal-methods)).
Lamport similarly describes TLA+ as modeling above the code level and states that such modeling does
not prevent coding errors ([high-level TLA+ guide](https://lamport.azurewebsites.net/tla/high-level-view.html)).

**What to copy.** Keep models small enough that counterexamples explain a design, model protocols
before or alongside implementation changes, and present checking to engineers as executable design
debugging. Stable named Actions, Properties, and traces are more useful than exposing theorem-prover
plumbing.

**What not to copy blindly.** TLA+ frequently uses a finite model-checking configuration whose
constants are smaller than production. Intel's first-party account calls formal verification an
imperfect validation tool and warns that a model is an abstraction checked only for restricted
parameters ([Intel experience](https://lamport.azurewebsites.net/tla/intel-excerpt.html)). Umpire
should never let a small-instance success become an unqualified product claim.

**Inference for Umpire.** The split between `Temporal.Feature` and `Temporal.System` is stronger
than a single implementation-shaped spec and should be preserved. Feature models should stay
stable across refactors; System models and Implementation Links should carry the burden of matching
today's mechanics.

### 2. Cedar: the closest precedent for a Lean source of truth plus production conformance

**Documented.** Cedar's verification-guided development process has three parts: an executable Lean
model with machine-checked properties, idiomatic Rust production code compared to the model with
differential randomized testing, and property-based tests for unmodeled production areas
([VGD paper](https://arxiv.org/abs/2407.01688)). The open repository contains the Lean semantics,
proofs, generators, fuzz targets, and DRT infrastructure
([cedar-spec](https://github.com/cedar-policy/cedar-spec)); Cedar's RFC process requires applicable
formal-model and DRT changes before feature stabilization
([RFC governance](https://github.com/cedar-policy/rfcs/blob/main/README.md)).

**What to copy.** Treat the formal model as executable ground truth; make the implementation-model
comparison routine and randomized; generate structured inputs on both sides; retain failing cases;
and make “update the model and conformance harness” part of feature completion rather than a later
verification project.

**Important difference.** Cedar's core semantic comparison is mostly a pure request/entities/
policies-to-decision function. Temporal behavior spans multiple processes, durable histories,
late-bound identities, partial failures, cleanup, and incomplete telemetry. Umpire needs explicit
Execution, Evidence, causal order, and Observation layers that Cedar's simpler function-level DRT
does not establish.

**Inference for Umpire.** Build the first live Nexus comparison as a small distributed analogue of
Cedar DRT: one canonical Lean-selected experiment, one implementation run, one normalized
Evidence-backed trace, and one exact semantic comparison. Do not start with whole-cluster universal
conformance.

### 3. MongoDB: one cancelled bridge and one successful bridge

**Documented.** MongoDB's server case captured traces while fuzzing C++ and attempted to check them
against a TLA+ model. The team cancelled it because the effort exceeded the expected value. The
paper attributes the cost to the large server, internal concurrency, and a pre-existing model whose
abstraction was poorly suited to trace checking. Realm Sync instead enumerated model behavior into
C++ unit tests and reached 100% branch coverage of the specified algorithm
([retrospective](https://emptysqua.re/blog/extreme-modelling-in-practice/),
[VLDB paper](https://www.vldb.org/pvldb/vol13/p1346-davis.pdf)).

**What to copy.** Choose the conformance technique when authoring the model, keep observable state
and Actions traceable, and begin with a small finite subsystem whose boundary can be driven and
observed without reconstructing every internal thread interleaving.

**What to avoid.** Do not make run evaluation depend on guessing which arbitrary low-level events
correspond to a high-level Action after the fact. Do not require full internal traces for a
black-box claim. Do not let a model omit the information needed for its future conformance seam.

**Inference for Umpire.** The Nexus caller-closure vertical slice is the right scale. The checked
Observation and Implementation Link declarations should be developed with the runtime adapter, not
years before it. Each new model family should demonstrate one feasible Evidence profile before its
public semantics become broad.

### 4. TraceLink: Implementation Links and Evidence Links can close a real class of gaps

**Documented.** TraceLink maps PGo implementation traces back to MPCal/TLA+ semantics. Its
evaluation found nine issues across network assumptions, compiler behavior, instrumentation,
timeouts, failure detectors, and symbolic abstractions. Some were model inaccuracies that ordinary
model checking could not reveal because the model was internally consistent. TraceLink explicitly
describes its trusted computing base and classes of undetectable bugs
([paper](https://doi.org/10.1145/3763128)).

Three findings are especially relevant:

- Instrumentation errors can and should fail closed when the recorded causal facts are internally
  inconsistent.
- A finite set used only to make model checking enumerable may be the wrong domain for validating
  real runtime inputs; TraceLink split the general runtime domain from the finite checking
  approximation.
- Real traces expose forgotten environment assumptions such as perfect failure detectors or
  always/never timeouts.

**Inference for Umpire.** Preserve two representations when necessary: a semantic domain valid for
Evidence interpretation and a finite authority used for a particular exhaustive plan. Bind them by
checked abstraction/refinement facts; never let the finite enumeration silently redefine product
meaning. This is a concrete use for Known Gaps, typed Limits, and separate Behavior Fingerprints.

### 5. FoundationDB and TigerBeetle: deterministic simulation verifies real code, at a price

**Documented.** FoundationDB's simulator runs an entire cluster deterministically in a
single-threaded process and injects machine, network, disk, and timing behavior. The FoundationDB
paper calls out “deep bugs” requiring particular crash/restart sequences and describes simulation
as integrated with development ([SIGMOD paper](https://www.foundationdb.org/files/fdb-paper.pdf),
[testing docs](https://apple.github.io/foundationdb/testing.html)). TigerBeetle's VOPR follows the
same lineage, running real cluster code with generated workloads and faults and retaining a seed for
reproduction ([architecture](https://github.com/tigerbeetle/tigerbeetle/blob/main/docs/ARCHITECTURE.md)).

**What to copy.** Centralize nondeterministic choices, virtualize faults behind narrow interfaces,
make all randomness seeded, speed logical time, preserve the exact replay input, and run the real
implementation wherever feasible.

**Constraint.** These systems were engineered for simulation. FoundationDB's simulator substitutes
its own network and process environment; TigerBeetle deliberately prizes system-wide determinism.
Retrofitting the complete existing Temporal server into a single deterministic runtime would be a
different and much larger program than Umpire's current artifact-driven external execution.

**Inference for Umpire.** Borrow the reproducibility contract, not the implementation architecture.
Umpire's thin runtime should control faults and SDK participants at explicit seams while allowing
the existing Temporal cluster to remain distributed. Gomad or purpose-built in-process harnesses can
be optional execution backends only when they consume the same Lean-produced experiment and emit
the same Evidence contract.

### 6. CHESS, Coyote, and P: control the nondeterminism you claim to explore

**Documented.** CHESS systematically enumerated thread schedules for concise concurrent scenarios,
found previously unknown bugs in code that had undergone long stress testing, and reproduced each
failing schedule ([CHESS report](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/osdi2008-CHESS.pdf)).
Coyote serializes async executions and takes control of supported scheduling and declared
nondeterministic choices; its documentation warns that unsupported external concurrency may make a
trace unreproducible ([Coyote usage](https://microsoft.github.io/coyote/get-started/using-coyote/)).
P makes asynchronously communicating state machines the programming/modeling abstraction and
provides systematic checking and reproducible error traces
([P paper](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/tr-8.pdf),
[P documentation](https://p-org.github.io/P/)).

**What to copy.** Every source of modeled nondeterminism should be either controlled, recorded, or
named as a Known Gap. Exploration receipts should state which scheduler, network, clock, fault, and
input choices were actually under control.

**What not to copy.** Umpire should not require Temporal engineers to rewrite production Go as P
machines or Coyote actors. The model should describe semantic choices; runtime adapters should
control only the seams needed by a concrete Temporal experiment.

### 7. ShardStore: decompose correctness and make maintenance a normal code-review task

**Documented.** ShardStore deliberately rejected full formal verification as impractical for the
storage node's scale. It used executable Rust reference models for sequential behavior, stateless
model checking for concurrency, and property-based techniques for crash consistency. The combined
property definitions and harnesses were about 12% of the ShardStore code, and non-formal-methods
engineers extended them as features evolved
([paper](https://www.amazon.science/publications/using-lightweight-formal-methods-to-validate-a-key-value-storage-node-in-amazon-s3)).

**What to copy.** Use the most suitable method per claim; make checks fast enough for local and CI
use; make artifacts and diagnostics readable; train domain engineers to own model changes; and
review semantic changes next to implementation changes.

**What not to copy.** ShardStore colocated models in the implementation language to reduce drift.
Umpire has deliberately chosen Lean as the sole behavioral authority. Copy the lifecycle discipline
and co-review requirement, but keep Go runtimes and Generated Views unable to define meaning.

### 8. IronFleet, Verdi, and Grove: full implementation proofs are a selective tool

**Documented.** IronFleet combined TLA-style refinement and Hoare logic to verify safety and
liveness down to new Dafny implementations, but the paper says correctness depends on explicit
assumptions, verification required considerable developer help, and the work targeted newly written
verification-friendly code ([IronFleet](https://www.microsoft.com/en-us/research/wp-content/uploads/2015/10/ironfleet.pdf)).
Verdi proved distributed systems in Coq and extracted event handlers to OCaml linked with network
runtime shims ([Verdi](https://github.com/uwplse/verdi)). Grove extends Perennial/Iris and Goose to
reason about programs written in a defined subset of Go; its case studies verify selected services
and libraries, not arbitrary Go repositories ([Perennial](https://github.com/mit-pdos/perennial),
[Goose](https://github.com/goose-lang/goose)).

**Inference for Umpire.** Do not set “prove the Temporal server implementation correct” as the
program's success criterion. Use kernel proofs for reusable semantic machinery and carefully chosen
high-value modules. Use model-based execution and Evidence-backed conformance for the large existing
Go system. A future Grove/Goose-style experiment could target one deep pure Go module behind a
stable capability contract without changing Umpire's overall architecture.

### 9. Veil, Ivy, and Apalache: optional backends should add assurance, not semantics

**Documented.** Veil embeds an automated/interactive transition-system verifier in Lean and
currently focuses on safety; its repository labels liveness as future work and Veil 2.0 as a preview
([Veil](https://github.com/verse-lab/veil)). Ivy structures protocol proofs around decidable
first-order fragments plus interactive generalization
([Ivy](https://microsoft.github.io/ivy/)). Apalache translates supported TLA+ checks to SMT and
distinguishes bounded checking from inductiveness checking
([Apalache](https://apalache-mc.org/)).

**Inference for Umpire.** The existing `Umpire.Verify.Veil` isolation is correct. A checker view
should consume an already checked Target and Property, publish the translation assumptions, and
return a backend-specific receipt. It must not introduce a second Property or Behavior language.
Counterexamples must replay through canonical Lean semantics before promotion.

### 10. Jepsen, Elle, Filibuster, and lineage-driven fault injection: black-box evidence is useful but partial

**Documented.** Elle infers transactional anomalies from client-visible histories and explicitly
documents that not every anomaly is detectable ([Elle](https://github.com/jepsen-io/elle)).
Filibuster starts from a passing functional test and explores failures at discovered service calls
([paper](https://christophermeiklejohn.com/publications/filibuster-socc-2021.pdf)). Molly uses
lineage from successful results to search fault combinations that could invalidate those results
([paper](https://people.ucsc.edu/~palvaro/molly.pdf)).

**What to copy.** Design black-box properties around what a public API and participant can
authoritatively observe. Use semantic lineage to avoid combinatorial fault enumeration. Retain
unknown and incomplete results when the history cannot distinguish allowed from forbidden behavior.

**Inference for Umpire.** A staging run using only public gRPC and participant receipts can support
a narrower Claim Assessment than a white-box run with history and internal traces. Both can consume
the same Test Plan and Property, but their Evidence policies and Known Gaps must differ. “Same
experiment” need not mean “same assurance.”

### 11. Alloy, SPIN, Apalache, and Quint: a bound or reduction is part of the claim

**Documented.** Alloy searches relational structures inside a user-supplied finite scope. Its
“small scope” work provides empirical evidence that many studied defects have small counterexamples,
not a theorem that every bug does
([language reference](https://alloytools.org/download/alloy-language-reference.pdf),
[small-scope evaluation](https://groups.csail.mit.edu/sdg/pubs/2002/SSH.pdf)). SPIN's mature
Promela workflow combines explicit-state exploration, partial-order reduction, compression, and
memory-saving modes; the selected mode determines whether search is exhaustive or approximate
([SPIN overview](https://spinroot.com/gerard/pdf/Advances2005.pdf)). Apalache uses SMT for bounded
symbolic checking and separately supports inductiveness checking; its documentation explicitly calls
the default bounded technique incomplete
([principles](https://apalache-mc.org/docs/apalache/principles/index.html),
[running](https://apalache-mc.org/docs/apalache/running.html)). Quint adds typed, familiar syntax,
simulation, and Apalache-based checking, while Quint Connect replays generated traces through Rust
drivers ([Quint](https://github.com/quint-co/quint/blob/main/quint/README.md),
[Connect](https://github.com/quint-co/quint-connect)).

**What to copy.** Treat scope, symmetry, reduction, search depth, fairness, solver, and state
canonicalization as material receipt fields. Preserve human-readable counterexamples and support
fast simulation before expensive checking. Make any optimization that changes completeness visible
at the status level, not buried in logs.

**Inference for Umpire.** `PLN-03`, `PLN-04`, and `VER-06` should be executable schema constraints:
an “exhaustive” receipt must identify a finite authoritative domain and a completion proof; a bounded,
random, bit-state, or heuristic receipt must be structurally unable to serialize that label. Umpire
should copy Quint's approachability and artifacts without adding Quint, Promela, or Alloy as a second
behavioral language (`SCP-03`, `AUT-07`).

### 12. P, PObserve, MET, and model-based testing: the bridge is a maintained module

**Documented.** P uses communicating state machines and synchronously composed monitor machines;
its public site documents AWS use across several services
([P](https://p-org.github.io/P/), [monitor semantics](https://p-org.github.io/P/manual/monitors/)).
PObserve is now an official runtime-monitoring component, but public technical material currently
establishes its intended log-to-monitor architecture more strongly than a peer-reviewed production
evaluation; it should be classified as promising, not proven at Umpire's scale
([repository](https://github.com/p-org/PObserve)). MET combines TLA+ design checking, trace-derived
CRDT implementation tests, and controlled message permutations
([MET](https://arxiv.org/abs/2204.14129)). Quint Connect's authors candidly identify bridge code and
model duplication as historic MBT maintenance problems, including within their own work
([first-party account](https://quint.sh/posts/quint_connect)). Classic conformance theory also makes
the oracle relation explicit: an input/output implementation conforms only with respect to a chosen
transition model and observation relation
([ioco overview](https://research.cs.queensu.ca/TechReports/Reports/2008-548.pdf)).

**What to copy.** Give the bridge a named interface, tests, owners, version, and generated fixtures.
Keep observer state side-effect-free. Make every driver explain how an abstract Action is enabled,
how runtime identity is bound, which concrete outcome is observed, and how quiescence or timeout is
represented.

**Inference for Umpire.** `Temporal.System` Observation and Execution adapters are deep modules, not
disposable glue (`MOD-02`, `MOD-06`). Each model family should budget and measure bridge maintenance:
lines touched per semantic change, review latency, adapter-only defects, and stale-artifact rejects.
If those costs grow proportionally with unrelated implementation internals, stop broadening the model
and redesign the observation seam.

### 13. MaceMC, MODIST, SAMC, and FlyMC: controlling real implementations finds a different bug class

**Documented.** MaceMC checked systems written in the Mace DSL and emphasized liveness-error search
([NSDI paper](https://www.usenix.org/legacy/event/nsdi07/tech/killian/killian_html/index.html)).
MODIST interposed on unmodified Windows applications and reported 35 previously unknown defects
across Berkeley DB, MPS, and PACIFICA; its transparency still depended on capturing every relevant
action and on supplied oracles
([paper](https://www.usenix.org/legacy/events/nsdi09/tech/full_papers/yang/yang_html/)). SAMC used
semantic knowledge to prioritize event schedules, while FlyMC used symmetry, independence, and
parallel flips to reduce systematic exploration
([SAMC](https://www.usenix.org/conference/osdi14/technical-sessions/presentation/leesatapornwongsa),
[FlyMC](https://ucare.cs.uchicago.edu/pdf/eurosys19-flyMC.pdf)). CMC illustrates the other end of the
trade-off: direct checking of implementation code with more invasive integration
([CMC](https://www.usenix.org/legacy/event/nsdi04/tech/full_papers/musuvathi/musuvathi_html/)).

**What to copy.** Interpose only at explicit high-value nondeterministic boundaries; use model state
and independence to prioritize rather than merely randomize; reproduce schedules without claiming
the intercepted implementation is fully controlled. Publish hook coverage and an inventory of
timeouts, clocks, storage, process, message, SDK, and scheduler behavior outside the controller.

**Inference for Umpire.** These are candidate Execution/Exploration backends, not semantic backends.
`EXP-02` can use Lean Model Coordinates to guide schedule flips, while `EVD-01` prevents an
implementation-level checker from inventing Feature behavior. If a reduction's soundness is not
established for the selected target, its “no bug” result remains bounded/heuristic under `PLN-04`.

### 14. Porcupine and Troubadour: specialize history checking instead of universalizing it

**Documented.** Porcupine takes a sequential executable model and concurrent call/return history and
decides linearizability; it warns that state-space explosion can still make histories slow and that
timestamp collection itself needs care
([repository](https://github.com/anishathalye/porcupine)). Troubadour goes beyond a sequential object:
it symbolically asks whether observed SQL transaction results could arise from semantically correct
transactions under a chosen isolation level and reports two new bugs in its evaluation
([paper](https://doi.org/10.1145/3720504)). Elle infers dependency cycles for specific transaction
anomalies and documents incomplete detectability
([paper](https://raw.githubusercontent.com/jepsen-io/elle/master/paper/elle.pdf)).

**What to copy.** Partition histories when the Property is compositional, use specialized decision
procedures where they give better scale, and keep the exact operation/transaction semantics beside
the checker receipt. Surface `unknown` or timeout rather than interpreting checker exhaustion as
acceptance.

**Inference for Umpire.** A specialized Go checker may consume a Generated View, but its model cannot
be authoritative (`SEM-01`, `ART-07`). The receipt should identify the source Lean Property and
fingerprint, checker algorithm/version, exact history checksum, partitioning theorem/assumption,
result, and timeout. Exact Replay must still interpret the counterexample in Lean before promotion
(`VER-05`, `EXP-05`).

### 15. Fault injection: fault selection, realization, and recovery judgment are three systems

**Documented.** Chaos Monkey randomly terminates production instances to exercise resilience
([official documentation](https://netflix.github.io/chaosmonkey/)). FATE/DESTINI separated failure
enumeration from declarative recovery specifications and reported 40,000 scenarios, 74 specs, 16 new
bugs, and 51 reproduced old bugs across three cloud systems
([paper](https://www.usenix.org/conference/nsdi11/fate-and-destini-framework-cloud-recovery-testing)). Filibuster derives
service-call failure variants from passing functional tests; Molly uses lineage to search combinations
that cut all derivations of a good outcome; Mallory uses causal summaries as greybox feedback
([Filibuster](https://christophermeiklejohn.com/publications/filibuster-socc-2021.pdf),
[Molly](https://people.ucsc.edu/~palvaro/molly.pdf), [Mallory](https://arxiv.org/abs/2305.02601)).

**What to copy.** Separate (1) the policy that selects a fault intent, (2) the mechanism that attempts
it, and (3) the oracle that judges recovery. Prefer lineage- or model-guided minimal fault sets over
raw combinations. Preserve a steady-state/cleanup check after the nominal Property verdict.

**Inference for Umpire.** `Fault Request`, `Execution Receipt`, and Run Evaluation already encode
this three-way split. Add realization metrics: requested, intercepted, applied, duration/extent
confirmed, target identity matched, recovery observed, and cleanup completed. A campaign with a high
request count but low realized-fault rate is an execution-adapter defect, not semantic exploration
(`EVD-05`, `EVD-06`, `EVD-08`).

### 16. IronFleet, Verdi, Grove, Anvil, Verus, and Aneris: proof scope is a layered graph

**Documented.** IronFleet linked centralized specifications to executable Dafny implementations,
including safety and assumption-dependent liveness, at a reported cost of about 3.7 person-years and
a 3.6:1 proof-annotation-to-code ratio
([paper](https://pdos.csail.mit.edu/6.5840/papers/ironfleet.pdf)). Verdi's initial Raft result had an
open safety premise; later work closed the linearizability proof with roughly 45K proof lines and 90
invariants, still without liveness, Byzantine faults, membership change, or log compaction
([initial paper](https://homes.cs.washington.edu/~mernst/pubs/verify-distsystem-pldi2015.pdf),
[completion/maintenance](https://homes.cs.washington.edu/~mernst/pubs/raft-proof-cpp2016.pdf)). Grove
reports roughly 12 proof lines per Go line for its verified components and trusts small network and
filesystem libraries plus a bounded-clock premise
([paper](https://pdos.csail.mit.edu/6.824/papers/grove.pdf)). Anvil verifies new Rust Kubernetes
controllers with stabilization/fairness assumptions; runtime testing found a liveness bug caused by
an omitted failure in a trusted external API spec
([paper](https://www.usenix.org/system/files/osdi24-sun-xudong.pdf)). The Verus IronKV repository
explicitly says it proves only the host-program layer and omits IronFleet's distributed refinement
([repository](https://github.com/verus-lang/verified-ironkv)). Aneris supplies modular safety reasoning
for programs over unreliable UDP-like networking
([project](https://iris-project.org/aneris/)).

**What to copy.** Draw an assurance graph from Feature statement through model/refinement/host/runtime
layers. Make every edge name its theorem or test, assumptions, trusted code, and unsupported failure
modes. Invest proof effort behind deep, stable interfaces; Grove's lease evolution and Anvil's feature
maintenance show why proof architecture affects change cost.

**Inference for Umpire.** `VER-04` receipts should be graph nodes, not flat badges. A Claim Assessment
must stop at the first absent edge. Near term, prove reusable Lean semantics and canonicalization;
consider program proof only for a narrow new component whose capability contract is stable. Existing
Temporal Go remains primarily a model-based execution/evidence problem (`SCP-04`, `MOD-06`).

### 17. Specification defects: proof success needs adversarial adequacy tests

**Documented.** IronSpec found ten author-confirmed specification bugs across all six evaluated
real-world verified systems; surviving mutations also included intended behavior, so mutation is a
review signal rather than an automatic verdict
([paper](https://www.usenix.org/conference/osdi24/presentation/goldweber),
[artifact](https://github.com/GLaDOS-Michigan/IronSpec)). An empirical review of IronFleet, Verdi,
and Chapar found defects clustered in specifications, shims, and tool interfaces; one mutation
disabled deduplication yet still verified because the high-level spec did not require exactly-once
behavior
([study](https://homes.cs.washington.edu/~arvind/papers/dsbugs.pdf)). IBM's vacuity work reports that
trivially true formulas consistently indicated a model, property, or environment problem in its
hardware practice
([paper](https://research.ibm.com/publications/efficient-detection-of-vacuity-in-temporal-model-checking)).
Mutation-model-checking experiments likewise found many model mutations not rejected by the studied
properties
([FSE](https://orbilu.uni.lu/bitstream/10993/59630/1/NIER_FSE23-1.pdf)).

**What to copy.** For each important Property require: a satisfying trace, a violating trace, a
reachable trigger, a non-vacuous witness, constrained outputs/effects, and targeted model mutants.
Mutate Feature transitions, System mechanisms, Implementation Links, Observation rules, ordering,
completeness, and fingerprints separately so a surviving mutant has one likely owner.

**Inference for Umpire.** Publish four non-interchangeable qualification metrics: model-transition
mutation score, Property vacuity/trigger coverage, Observation/Link mutation score, and implementation
mutation score. `GOV-02` must review survivors; a score is diagnostic evidence, never permission to
alter product semantics.

### 18. Runtime evidence, provenance, and assurance: preserve uncertainty and decision layers

**Documented.** Partial-order runtime verification analyzes executions without inventing a total
order; failure-aware work uses multi-valued results because asynchronous loss and reordered messages
can leave a monitor unable to decide
([partial-order RV](https://users.ece.utexas.edu/~garg/dist/PartialOrderVerification.pdf),
[failure-aware RV](https://drops.dagstuhl.de/entities/document/10.4230/LIPIcs.FSTTCS.2015.590)).
ShiViz adds vector-clock instrumentation because ordinary logs do not reconstruct happens-before
([paper](https://homes.cs.washington.edu/~mernst/pubs/shivector-shiviz-icse2014.pdf)). W3C PROV
models entities, activities, agents, derivation, and provenance constraints
([overview](https://www.w3.org/TR/prov-overview/)). RFC 9334's attestation architecture separates
Evidence, appraisal policy, Attestation Result, and the relying party's final decision
([RATS](https://www.rfc-editor.org/rfc/rfc9334.html)). GSN/SACM similarly separates claims,
argument/context, and evidence
([GSN](https://www.faa.gov/about/office_org/headquarters_offices/ang/redac/redac-sas-201503-gsn-community-standard-v1.pdf),
[SACM](https://www.omg.org/spec/SACM/2.2/PDF)).

**What to copy.** Represent ordering edges as direct causal edge, source-local sequence, derived
bound, arbitrary presentation order, or unknown. Treat evidence provenance and integrity as input to
assessment, not proof of semantic content. Let environment rollout policy consume a Claim Assessment
without changing it.

**Inference for Umpire.** Evidence Links can borrow PROV's derivation shape without adopting its
ontology wholesale: Model Fact, source records, normalization/Observation activity, mapper identity,
causal/completeness premises, and producing agent/tool. `EVD-04` should yield `unknown` when an
order-sensitive Property lacks the needed relation. `QLF-02` then decides whether that assessment is
acceptable for local, staging, or canary use; it must never rewrite unknown to pass.

### 19. Workflow nets and BPMN: structural soundness is useful, but the translation is the hard part

**Documented.** Classical workflow-net soundness captures proper completion, absence of residual
tokens, and absence of dead transitions. The literature distinguishes eight soundness notions and
shows that expressive extensions often make analysis undecidable
([classification](https://www.vdaalst.com/publications/p628.pdf)); bounded workflow-net soundness is
PSPACE-complete
([complexity](https://doi.org/10.3233/FI-2014-1005)). BPMN model checking first requires a precise
formal mapping, and mapping choices determine how exceptions, cancellation, event subprocesses, and
other constructs are interpreted
([BPMN mapping](https://doi.org/10.1016/j.scico.2018.05.008)). Workflow-pattern analysis decomposes
control-flow expressiveness into recurring forms rather than one universal correctness property
([patterns](https://www.vdaalst.com/publications/p562.pdf)).

**What to copy.** Reuse the questions: can every admitted case reach completion, can completion leave
residual active work, is a branch dead, and can cancellation strand a token/obligation? Keep these as
parameterized structural Properties over Umpire traces. Use patterns to organize examples and negative
controls.

**What not to copy blindly.** Temporal Workflow behavior includes durable event history, replay,
Activities, Signals, Updates, retries, timers, Child Workflows, Nexus, worker/version behavior, and
implementation-specific delivery. A Petri-net or BPMN translation can erase exactly the identity,
causal, failure, and evidence distinctions Umpire needs.

**Inference for Umpire.** Implement only structural lemmas justified by concrete Temporal families
(`SCP-01`). First candidates are “all admitted terminal Feature states discharge declared obligations,”
“caller closure leaves no live Nexus operation beyond the bound,” and “every modeled cancellation
branch is reachable.” Keep `SEM-09` bounded and do not call structural soundness a liveness proof.

### 20. Industrial sustainability: publication is an event; ownership is the outcome

**Documented.** Long-running industrial experience ranges from sustained railway and nuclear use to
tools that remained specialist or product-specific. A 13-year shutdown-system case shows formal
methods embedded in an extended safety-critical lifecycle
([case](https://doi.org/10.1007/978-3-540-45236-2_9)); a 25-year B/Event-B account emphasizes method,
tool, training, and organizational trajectory
([experience](https://arxiv.org/abs/2005.07190)). The CICS Z work has both a first-party account and
later independent historical analysis, useful because retrospective interpretations differ
([IBM](https://research.ibm.com/publications/use-of-software-engineering-including-the-z-notation-in-the-development-of-cics),
[independent quantitative critique](https://doi.org/10.1016/S0164-1212%2896%2900122-7),
[history](https://doi.org/10.1145/3522577)). The independent analysis judged the case valuable but
found the headline quantitative claims insufficiently substantiated; this is a rare warning against
transferring productivity or defect-rate figures without comparable baselines. The Farsite
retrospective documents a research system
whose technology influenced later work without equating research success with sustained deployment
([retrospective](https://www.microsoft.com/en-us/research/wp-content/uploads/2007/04/OSR2007-4aa.pdf)).
Industrial obstacles repeatedly include notation, tool integration, training, review workflow, and
management incentives
([issues](https://brucker.ch/publications/altenhofen.ea-issues-2010/),
[survey](https://vsr.sourceforge.net/fmsurvey.htm)).

**Inference for Umpire.** Define adoption as ordinary Temporal engineers changing and reviewing the
model with bounded feedback—not a one-time proof, publication, or specialist demo. Track quarterly:
active non-core authors, model changes paired with product changes, median counterexample-understanding
time, bridge-fix burden, CI cost, stale artifact catches, and defects/decisions influenced. If only
Umpire specialists can maintain a family after two product iterations, classify it as constrained and
redesign or retire the family rather than preserving a ceremonial model.

## What Umpire should copy or mimic

### A. The Cedar loop: model, prove, compare, retain

For each model family:

1. Author executable semantics and pure Properties in Lean.
2. Prove reusable semantic laws in the Lean kernel where the cost is justified.
3. Generate bounded cases from that same model.
4. Compare qualified implementation observations to the model.
5. Retain minimized counterexamples as exact, reviewed regressions.

This combines Cedar's executable Lean model and DRT
([Cedar](https://arxiv.org/abs/2407.01688)) with QuickCheck-style counterexample retention and
shrinking; model-based shrinking research notes that preserving state-machine validity during
reduction requires model-aware reducers rather than generic deletion
([model-based shrinking paper](https://jeapostrophe.github.io/conferences/2013-tfp/proceedings/tfp2013_submission_15.pdf)).

### B. The AWS/TLA+ workflow: debug designs before debugging deployments

Make list/explain, trace rendering, and counterexample inspection fast enough that a Temporal
engineer can use them while designing a feature. AWS's adoption account emphasizes the value of
finding design errors early and communicating the practice as “debugging designs”
([AWS paper](https://www.amazon.science/publications/how-amazon-web-services-uses-formal-methods)).

### C. The ShardStore decomposition: one claim, one cheapest sufficient method

Examples:

- use a Lean proof for canonicalization or a reusable transition theorem;
- use exact finite search for a small complete target;
- use Veil or another solver only for an opted-in supported view;
- use deterministic model exploration to select a Test;
- use a real Temporal run to assess implementation behavior;
- use source-specific checks for persistence, races, authorization, and performance.

ShardStore's paper explicitly decomposes correctness and applies different automated techniques to
different properties ([ShardStore](https://www.amazon.science/publications/using-lightweight-formal-methods-to-validate-a-key-value-storage-node-in-amazon-s3)).
This reinforces Umpire's focused-complement rule (`SCP-04`).

### D. Deterministic simulation's reproducibility contract

Every selected point should retain:

- the exact model identity and Behavior Fingerprint;
- the policy, seed, ordered choices, and all stage Limits;
- the canonical Test Plan checksum;
- runtime bindings learned during execution;
- Action/fault attempts and realization receipts;
- causal Evidence and completeness facts; and
- cleanup outcome.

FoundationDB, TigerBeetle, CHESS, and Coyote all show the debugging value of reproducible runs
([FoundationDB](https://apple.github.io/foundationdb/testing.html),
[TigerBeetle](https://github.com/tigerbeetle/tigerbeetle/blob/main/docs/ARCHITECTURE.md),
[CHESS](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/osdi2008-CHESS.pdf),
[Coyote](https://microsoft.github.io/coyote/get-started/using-coyote/)).

### E. TraceLink's auditable correspondence

An accepted Model Fact should say which mapper version, records, bindings, causal relations, and
completeness checks established it. A Feature fact derived through an Implementation Link should
also retain its System coordinate and forward-correspondence step. TraceLink demonstrates that
instrumentation, environment models, and abstraction mappings themselves contain bugs
([TraceLink](https://doi.org/10.1145/3763128)).

### F. Temporal's existing exact-history and feature baselines

Temporal already has valuable behavior checks that Umpire should reuse or complement. The SDK
feature suite runs the same feature across SDK versions, replays histories, scrubs
execution-dependent values, and compares exact Event sequences
([temporalio/features](https://github.com/temporalio/features)). The server testing guide provides
isolated test clusters, exact history assertions, deterministic test identifiers, test hooks for
specific race windows, and optional OpenTelemetry capture
([server testing guide](https://github.com/temporalio/temporal/blob/main/docs/development/testing.md)).

**Inference for Umpire.** Do not rebuild these as generic Umpire machinery. Use them as execution
adapters or independent qualifications where they fit. Umpire adds value where model-owned
semantics, cross-environment artifacts, first-class faults, bounded exploration, Evidence
qualification, and honest Claim Assessments are missing.

## What Umpire should avoid

1. **A second source of behavioral truth.** Generated Go tests, runtime code, Evidence adapters,
   checker translations, and documentation must not add semantics. Cedar's DRT works because the
   Lean model remains the comparison authority ([cedar-spec](https://github.com/cedar-policy/cedar-spec)).

2. **Retrofitting whole-server trace equivalence before proving one narrow seam.** MongoDB's
   cancelled case is the direct warning
   ([retrospective](https://emptysqua.re/blog/extreme-modelling-in-practice/)).

3. **Overly abstract runtime validation domains.** A finite symbolic set useful for model checking
   may reject legitimate real input. TraceLink had to separate its finite checking approximation
   from a general runtime representation ([TraceLink](https://doi.org/10.1145/3763128)).

4. **A universal `pass` status.** Execution, Observation, Implementation Link application,
   Property evaluation, verification, cleanup, and Claim Assessment fail for different reasons and
   support different claims. Coyote's explicit “not a verification system” warning is a good model
   of status honesty ([Coyote](https://microsoft.github.io/coyote/)).

5. **Wall-clock ordering of distributed facts.** Use modeled order, source-local sequence, and
   cause/effect. Lamport's relation is the foundation
   ([paper](https://lamport.azurewebsites.net/pubs/time-clocks.pdf)).

6. **Treating telemetry presence as telemetry completeness.** OpenTelemetry sampling can discard
   whole spans and their events ([specification](https://opentelemetry.io/docs/specs/otel/trace/sdk/)).
   Evidence profiles must declare sampling, retention, closure, loss, duplication, and identity
   policy.

7. **Silent state-space truncation.** Bounded checking that reaches a Limit must stay
   inconclusive. Apalache explicitly documents the missed-longer-trace risk
   ([docs](https://apalache-mc.org/docs/apalache/running.html)).

8. **Coverage as correctness.** Full branch coverage in Realm Sync was useful evidence about test
   reach, not proof that its model or oracle was complete
   ([MongoDB paper](https://www.vldb.org/pvldb/vol13/p1346-davis.pdf)). Model coverage, code
   coverage, Evidence coverage, and Property-clause coverage should remain separate metrics.

9. **Automatically promoting every discovered failure.** A failure may be an implementation bug,
   a model bug, an Observation bug, an environment assumption, instrumentation corruption, or an
   intentionally injected control. TraceLink found examples across those categories
   ([paper](https://doi.org/10.1145/3763128)). Require exact replay, model-aware minimization, cause
   classification, and human review.

10. **A monolithic “formal verification” adoption gate.** IronFleet, Verdi, ShardStore, Coyote,
    and AWS TLA+ succeed at different assurance levels. Select methods per risk and claim rather
    than making full code proof a prerequisite for useful model-based testing
    ([IronFleet](https://www.microsoft.com/en-us/research/wp-content/uploads/2015/10/ironfleet.pdf),
    [ShardStore](https://www.amazon.science/publications/using-lightweight-formal-methods-to-validate-a-key-value-storage-node-in-amazon-s3)).

11. **Tool-first scope.** NASA's industrial roundtable identifies inadequate domain examples and
    “build it and they will come” thinking as adoption impediments
    ([roundtable](https://shemesh.larc.nasa.gov/fm/fm-paper-ieee-roundtable.html)). Every Umpire
    capability should remain justified by a concrete Temporal regression, exploration, evidence,
    or verification use case.

## Underexplored opportunities for Umpire

These are recommendations, not documented Umpire commitments.

### 1. Lean-native semantic coverage for implementation fuzzing

**Opportunity.** Export stable Model Coordinates—state classes, transitions, Property clauses,
fault intents, Observation alternatives, and Implementation Link branches—as exploration feedback.
ModelFuzz shows that abstract model-state coverage can find implementation bugs missed by random,
line-, and trace-guided approaches
([published study](https://repository.tudelft.nl/record/uuid%3A66d18d3c-fead-4df0-8310-5df11370db13)).

**Experiment.** For the Nexus variation space, compare equal execution budgets under random seed
selection, Go coverage guidance, and Lean semantic-coordinate guidance. Report all three coverage
families without claiming exhaustiveness.

### 2. Evidence-link-guided fault selection

**Opportunity.** Use Evidence Links as lineage: select faults that cut distinct derivations of an
accepted fact, then prioritize minimal sets predicted to make the fact unknown, conflicting, or
violated. Molly demonstrates SAT-guided failure selection from lineage
([Molly](https://people.ucsc.edu/~palvaro/molly.pdf)); Mallory demonstrates causal-summary feedback
for fault scheduling ([Mallory](https://arxiv.org/abs/2305.02601)).

**Potential advantage.** Unlike generic call-graph lineage, Umpire's links can include semantic
bindings, modeled order, completeness, and the exact Property clause. That could make fault
selection both more explainable and more targeted.

### 3. Mutation qualification for models and evidence mappings

**Opportunity.** Deliberately mutate transition outcomes, Property clauses, Observation mappings,
Implementation Link coordinates, causal parents, and completeness flags. Require the appropriate
model tests or negative controls to reject each non-equivalent mutant. Mutation-model-checking
research treats surviving mutants as evidence that a specification may be too weak
([FSE paper](https://orbilu.uni.lu/bitstream/10993/59630/1/NIER_FSE23-1.pdf)); vacuity research
shows that “request implies eventual response” can pass merely because requests never occur
([robust vacuity paper](https://arxiv.org/abs/1002.4616)). IronSpec evaluated specifications from
six verified codebases and found specification bugs in all six, demonstrating that a proved
implementation can still satisfy the wrong or incomplete contract
([OSDI 2024 paper](https://www.usenix.org/system/files/osdi24-goldweber.pdf)).

**Umpire fit.** The current separate Behavior satisfiability, Property, Observation, and
Implementation Link layers make responsibility-local mutations possible. This is stronger than a
single end-to-end golden that may pass for the wrong reason.

### 4. Dual-domain checking: finite exploration plus general trace validation

**Opportunity.** Let a Target provide a finite proof-carrying enumerator for exhaustive planning
and a more general semantic recognizer for runtime Evidence. Check an explicit abstraction relation
between them. TraceLink's symbolic-request defect shows why a model-checking-only finite
approximation should not define the valid runtime domain
([TraceLink](https://doi.org/10.1145/3763128)).

**Guardrail.** An exhaustive claim applies only to the finite enumerated domain. A runtime trace
admitted by the general domain does not retroactively enlarge that exhaustive claim.

### 5. Cross-SDK differential semantics using Temporal's feature corpus

**Opportunity.** Bind a selected `ExperimentSpec` to more than one SDK participant while evaluating
all runs against the same Lean Feature model. Temporal's feature repository already provides
cross-language snippets and exact scrubbed-history comparison
([features](https://github.com/temporalio/features)).

**Value.** This would distinguish “all SDKs agree” from “each SDK conforms to the model.” Agreement
can be wrong; model conformance can identify the divergent side and explain the violated clause.

### 6. Evidence-policy gradients across white-box and black-box environments

**Opportunity.** Compile one model-owned observation requirement into named Evidence profiles:
white-box local, public-gRPC staging, participant-only, and canary-safe. Each profile should state
which Model Facts are supportable and which become Known Gaps. Elle's explicit detectability limits
are a useful precedent ([Elle](https://github.com/jepsen-io/elle)); OpenTelemetry sampling shows why
profile completeness cannot be assumed ([spec](https://opentelemetry.io/docs/specs/otel/trace/sdk/)).

**Payoff.** The byte-identical Test Plan can travel across environments without pretending that the
resulting claims are identical.

### 7. Model-assumption monitoring

**Opportunity.** Promote important System assumptions—failure-detector accuracy, timeout ranges,
queue delivery capabilities, clock policy, and Evidence retention—into named monitorable contracts.
TraceLink found forgotten or inaccurate timeout and failure-detector assumptions by checking real
traces ([TraceLink](https://doi.org/10.1145/3763128)).

**Result shape.** An assumption failure should be its own status and Known Gap, not a Feature
Property violation unless the Feature contract truly forbids the behavior.

### 8. Causal trace quality as a first-class metric

**Opportunity.** Score a run's evidence not only by record count but by identity closure, causal
parent closure, source coverage, ambiguity, and unsupported branches. ShiViz's experience report
notes that ordinary distributed logs do not contain enough information to recover ordering and
adds vector-clock data for that purpose
([ShiViz paper](https://www.cs.ubc.ca/~bestchai/papers/cacm2016-shiviz.pdf)).

**Guardrail.** A lower quality score is diagnostic; it must never be converted into proof of absence.

### 9. Semantic, evidence-preserving minimization

**Opportunity.** Reduce a failure along separately authored dimensions: Space choices, optional
Actions, fault intents, participants, model inputs, and non-responsible Evidence facts. Delta
debugging formalizes repeated reduction to a smaller failure-inducing input
([Zeller](https://www.st.cs.uni-saarland.de/papers/tse2002/)); state-machine research shows that
generic shrinking can break preconditions and produce invalid traces
([model-based shrinking](https://jeapostrophe.github.io/conferences/2013-tfp/proceedings/tfp2013_submission_15.pdf)).

**Umpire advantage.** Because the Behavior and Target own validity, every candidate reduction can be
rechecked before a concrete rerun. The minimized artifact should retain the same responsible
Property clause and Evidence core.

### 10. Verification-aware change impact

**Opportunity.** Use Definition IDs, Behavior Fingerprints, consumed capability contracts, and
Implementation/Evidence Link dependencies to compute which regressions, model checks, generated
views, and environment profiles are stale after a change. Cedar's RFC gate requires applicable
model and DRT updates before stabilization
([Cedar RFCs](https://github.com/cedar-policy/rfcs/blob/main/README.md)).

**Guardrail.** Change impact can select required work; it cannot automatically approve a semantic
change or migration.

### 11. Counterexample diversity, not only shortest trace

**Opportunity.** Retain a small portfolio of counterexamples that differ by responsible transition,
fault, causal structure, or Known Gap, not merely the globally shortest trace. CHESS and Coyote show
the utility of reproducible schedules, while Mallory shows that causal behavior summaries can
identify genuinely new behavior
([CHESS](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/osdi2008-CHESS.pdf),
[Mallory](https://arxiv.org/abs/2305.02601)).

**Reason.** One short regression prevents recurrence; diverse witnesses inform model completeness
and exploration policy.

### 12. “Negative controls” for the assurance pipeline

**Opportunity.** Maintain deliberately faulty controls for each stage: unsatisfiable Behavior,
planner truncation, unrealized fault, missing record, wrong causal parent, stale fingerprint,
Implementation Link mismatch, Property violation, and incomplete cleanup. The ShardStore and Cedar
processes both use multiple independent validation mechanisms rather than trusting one test path
([ShardStore](https://www.amazon.science/publications/using-lightweight-formal-methods-to-validate-a-key-value-storage-node-in-amazon-s3),
[Cedar](https://github.com/cedar-policy/cedar-spec)).

**Goal.** Prove the pipeline can reject bad evidence and bad mappings, not only accept a happy-path
fixture.

## Falsifiable Nexus pilot and expansion gates

These are proposed decision experiments, not forecasts. Before running them, record the baseline,
hardware/environment, campaign Limit, responsible owner, and exact measurement query. Do not change
a target after seeing the result without preserving both versions.

| Hypothesis | Pilot design | Primary metrics | Pass signal for expansion | Kill / redesign signal |
| --- | --- | --- | --- | --- |
| **H1: ordinary engineers can maintain the model.** | Have two Temporal engineers outside the Umpire core independently add one Nexus variation and explain a seeded counterexample. | active edit time, elapsed review time, Umpire-core interventions, incorrect semantic edits, explanation accuracy | both changes land through the documented interface with no internal-Umpire edits and reviewers agree on trace meaning | either engineer needs to understand planner/checker internals; repeated silent semantic mistakes; model API changes for an ordinary variation (`AUT-01`, `MOD-06`) |
| **H2: the Feature/System split reduces coupling.** | Apply one implementation-only Nexus refactor and one product-semantic change separately. | changed definitions, fingerprints invalidated, artifacts/tests rerun, irrelevant churn | implementation refactor leaves Feature fingerprints stable; semantic change invalidates exactly the dependent links/artifacts | either change forces broad unrelated model rewrites or fails to invalidate a dependent artifact (`SEM-08`, `ART-02`) |
| **H3: one artifact really travels.** | Execute one checksum-identical `ExperimentSpec` locally, in CI, and against a black-box environment. | artifact checksums, environment bindings, behavior-affecting diffs, Evidence profile/claim differences | byte-identical plan; only allowed environment bindings vary; claims differ honestly with Evidence strength | adapter needs to edit Actions/outcomes/Properties/Limits, or the UI labels all three claims alike (`ART-03`, `ART-05`, `QLF-01`) |
| **H4: receipts prevent false fault claims.** | Include successful fault, blocked fault, wrong-target fault, delayed fault, and no-op control. | requested/intercepted/applied/confirmed rates, target-coordinate match, false realized count | all five classify at Execution without Property logic; zero scheduled-only faults count as realized | any missing/wrong/no-op fault establishes the intended Model Fact (`EVD-05`, `EVD-06`) |
| **H5: evidence fails closed without excessive inconclusiveness.** | Delete, duplicate, reorder, misbind, and conflict records in golden Evidence fixtures. | mutant detection by class, false accept rate, false inconclusive rate on clean runs, diagnostic localization | zero false accepts for material mutants; clean fixture passes; each mutant identifies the Evidence rule/coordinate | a material mutant passes, or most valid runs become unknown because the Evidence contract is impractical (`EVD-03`, `EVD-04`, `EVD-09`) |
| **H6: semantic guidance improves discovery under equal budget.** | Run random, Go-coverage-guided, and Lean-coordinate-guided Nexus campaigns with equal execution count/time and a seeded hidden-bug corpus. | unique model states/transitions/Property clauses, realized fault intents, distinct confirmed failures, time to first failure | semantic guidance covers more relevant coordinates or finds seeded failures materially faster without lower realization quality | no repeatable advantage after preregistered campaigns, or guidance cost consumes the budget (`EXP-02`) |
| **H7: replay is exact and promotion is disciplined.** | Discover, replay, reduce, classify, and promote one controlled failure; change the model afterward and attempt stale replay. | replay rate, semantic/evidence core retained, reduction ratio, stale rejection, human classification agreement | exact replay succeeds from artifact alone; model-aware reduction retains cause; stale replay rejects; promotion retains provenance | reproduction requires undocumented state; reducer changes the responsible clause; stale artifact runs silently (`EXP-05`, `VER-05`, `ART-02`) |
| **H8: bounded checking reports honest status.** | Exercise complete finite search, no-counterexample-before-Limit, unsatisfiable Behavior, and optional backend timeout. | serialized Stage Status/Assurance Method, UI/CI rendering, prohibited claim strings | each case is distinct end-to-end and only the complete finite search may say exhaustive | timeout/Limit/unsatisfiable becomes pass or “verified” anywhere (`PLN-03`–`PLN-05`, `VER-06`) |
| **H9: maintenance cost stays sublinear in scenario count.** | Add three variations sharing Actions/Observations, then change one shared capability. | adapter/model LOC changed, duplicated mappings, test runtime, review time | shared deep modules absorb the common change; cost does not repeat per scenario | every variation requires bespoke runtime/oracle code or Evidence mapping (`MOD-06`–`MOD-08`) |
| **H10: a 10× campaign remains bounded and safe.** | Increase candidate and execution count 10× under fixed environment controls; inject executor crash and partial cleanup failures. | queue growth, memory, time/trace, cancellation latency, orphaned workflows/operations, cleanup completion, claim loss | Limits apply independently, backpressure is explicit, crash recovery preserves attempts/Evidence, and incomplete cleanup blocks qualification | unbounded queue/resource growth, lost Run records, unsafe retry duplication, orphaned resources, or cleanup failure hidden by pass (`PLN-01`, `EVD-08`, `QLF-02`) |

### Immediate kill, defer, and continue signals

- **Kill or redesign the current architecture** if Go/generated views can change behavior without a
  Lean fingerprint change; a missing/unrealized fault can pass; an Evidence mutant can establish a
  fact; or a bounded/timeout result can serialize as exhaustive. These violate Umpire's defining
  claim discipline rather than merely missing a performance target.
- **Defer environment expansion** if local/CI Evidence identity and causal closure are not reliable,
  cleanup is not independently reported, or the same plan requires behavior-affecting edits. Canary
  execution adds risk without strengthening a broken seam.
- **Defer optional solver/proof integration** if canonical Lean replay, assumption/TCB receipts, and
  distinct Assurance Methods are incomplete. Another backend would multiply ambiguity.
- **Constrain or retire a model family** if two product iterations require specialist-only bridge
  surgery, the System model tracks internal threads rather than stable mechanisms, or mutation tests
  show the Property does not distinguish intended wrong behavior.
- **Continue and broaden** only when a narrow family produces useful design decisions or real
  regressions, ordinary engineers maintain it, exact artifacts/replay work, negative controls fail at
  the responsible stage, and incremental cost is measurable and acceptable.

### Questions that must be answered before expanding beyond Nexus

1. Which next domain has a stable user-visible Feature contract and an observable, controllable
   System seam—Activities, Updates, Child Workflows, versioning, visibility, persistence, or task
   dispatch—and why is it a better second family than the alternatives?
2. Which behavior is shared capability versus domain-specific semantics? Can the shared part become a
   deep `Umpire` or Temporal capability module without importing names or assumptions across forbidden
   boundaries (`SCP-02`, `MOD-01`, `MOD-03`, `MOD-10`)?
3. Is the finite planning domain authoritative, or merely an exploration approximation? What general
   recognizer admits real runtime identifiers, payloads, and histories without redefining the finite
   claim (`AUT-08`)?
4. What exact public, white-box, and participant sources can establish each Model Fact? What missing
   source or causal edge turns the result unknown?
5. Which operations and faults can be intercepted without modifying Temporal product behavior? What
   receipt proves the intended target and duration, and which nondeterminism remains uncontrolled?
6. What bounded progress unit matches the product promise: event count, Workflow Task, retry attempt,
   logical timer, or causal step? Is any unbounded liveness claim being smuggled into finite Execution?
7. What failure reduction dimensions preserve semantic validity, late-bound identity, and Evidence
   closure? What must never be deleted from a witness?
8. Which existing Temporal unit, functional, history-replay, race, persistence, or SDK feature test
   remains the cheaper and more authoritative method (`SCP-04`)?
9. Who outside the Umpire core will own the model, Observation adapter, and regressions through two
   product changes? What is the acceptable edit-to-feedback and CI budget?
10. What is the threat/data policy for captured payloads, credentials, tenant identifiers, and canary
    faults? Which fields are redacted before hashing, and can redaction destroy identity or causality?
11. What happens after an executor, worker, observer, or artifact store crash? Can Umpire reconstruct
    attempts, partial Evidence, cleanup obligations, and the distinction between not-run and unknown?
12. At 10× load, where are backpressure and per-stage Limits enforced? Can an expensive checker,
    reduction loop, or Evidence join starve cleanup or pinned Regressions?
13. Which assumptions are monitorable in production, which are only testable in a simulator, and which
    remain trusted Known Gaps? Does the Claim Assessment show that partition?
14. What result would cause the team to stop modeling this domain? Precommit the technical and
    maintenance kill signals before building the adapter.

## Recommended Umpire adoption sequence

This research favors a staged vertical slice over broad platform construction:

1. **One finite Feature target and one independently authored System target.** Keep the state space
   small enough for exact search and explanation.
2. **One checked Implementation Link.** Demonstrate correspondence on synthetic accepted System
   traces before live execution.
3. **One canonical experiment artifact.** Bind all identities, fingerprints, Limits, seed, and
   requested Actions without claiming they occurred.
4. **One local real execution.** Capture Action/fault receipts, causal records, divergence, and
   cleanup independently.
5. **One strict Run Evaluation.** Normalize Evidence, fail closed, apply the Implementation Link,
   then evaluate the unchanged Feature Property.
6. **One negative control.** Inject a labeled duplicate or omitted delivery and demonstrate the
   exact responsible stage and clause.
7. **Exact replay and model-aware reduction.** Reproduce the same semantic failure and minimize it
   without editing admitted Evidence.
8. **Byte-identical CI and black-box execution.** Reuse the artifact; vary only environment binding
   and Evidence policy.
9. **Model-guided exploration.** Compare semantic guidance to random selection under equal Limits.
10. **Only then evaluate optional Veil integration, broader catalogs, campaign coordination, or
    production canary execution.**

**Inference for Umpire.** This order matches the successful small-scope conformance pattern at
Realm Sync, the staged model/prove/compare loop in Cedar, and the maintenance emphasis at
ShardStore, while avoiding the whole-server trace-mapping trap documented by MongoDB
([MongoDB](https://emptysqua.re/blog/extreme-modelling-in-practice/),
[Cedar](https://arxiv.org/abs/2407.01688),
[ShardStore](https://www.amazon.science/publications/using-lightweight-formal-methods-to-validate-a-key-value-storage-node-in-amazon-s3)).

## Decision checklist for each new model family

### Meaning

- What user-visible behavior belongs in `Temporal.Feature`?
- What current mechanism belongs in `Temporal.System`?
- What is generated structure rather than behavior?
- Which assumptions and Known Gaps are explicit?
- Which Property clauses can pass vacuously, and what Behavior witness prevents that?

### Search and assurance

- Is the target finite because the domain is truly authoritative, or only because checking needs an
  approximation?
- What exact Limit and unit bound each decision, search, execution, observation, and reduction?
- Which result is a kernel proof, solver-backed check, exhaustive finite search, bounded search,
  randomized Test, or concrete replay?
- What would make “no counterexample” inconclusive?

### Execution

- Which Actions are controllable and which outcomes remain model-owned?
- Which fault point can the runtime actually intercept?
- What receipt proves the requested Action or fault occurred at the intended Model Coordinate?
- Which nondeterministic choices are controlled, recorded, or unsupported?
- How are late-bound Temporal identities resolved without changing the Test Plan?

### Evidence

- Which source is authoritative for each record?
- What proves record identity, causal order, completeness, and closure?
- Can sampling, buffering, retries, duplication, or loss create a false absence?
- Which fields are retained, hashed, redacted, or rejected?
- Can the mapping explain every accepted Model Fact with coordinate-complete Evidence Links?

### Maintenance

- Can a Temporal engineer explain the model and a counterexample without reading Umpire internals?
- Does a semantic change invalidate fingerprints and exactly the dependent artifacts?
- Is the model/conformance change reviewed beside the product change?
- Do mutation and negative-control tests prove that the oracle rejects meaningful wrong behavior?
- Is there one exact replay path and one human-reviewed promotion path?

## Annotated reference list

### Formal specification and industrial adoption

- Chris Newcombe et al., [“How Amazon Web Services Uses Formal
  Methods”](https://www.amazon.science/publications/how-amazon-web-services-uses-formal-methods).
  The essential industry account: design bugs found, adoption tactics, abstraction value, and the
  model/implementation caveat.
- Leslie Lamport et al., [“Specifying and Verifying Systems with
  TLA+”](https://lamport.azurewebsites.net/pubs/spec-and-verifying.pdf). Formal basis and finite
  model-checking scope.
- Leslie Lamport, [“A High-Level View of
  TLA+”](https://lamport.azurewebsites.net/tla/high-level-view.html). Practitioner-oriented account
  of modeling above code and the distinction from coding-error detection.
- National Research Council/NASA participants, [“Impediments to Industrial Use of Formal
  Methods”](https://shemesh.larc.nasa.gov/fm/fm-paper-ieee-roundtable.html). Tooling, domain examples,
  and adoption-process warnings that remain relevant to Umpire authoring.
- Wolfgang Grieskamp, [“A Perspective on the Model-Based Testing Field” and Spec Explorer adoption
  obstacles](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/tr-2006-111.pdf).
  Reports limited product-team penetration and concrete authoring, state-explosion, documentation,
  and integration barriers.
- Leslie Lamport, [*Specifying Systems*, TLC
  chapter](https://lamport.azurewebsites.net/tla/book-01-08-21.pdf). Primary description of TLC's
  finite explicit-state semantics, safety/liveness checks, fairness, and configuration. Use it to
  define what an “exhaustive TLC-like” Umpire receipt would have to expose.
- Daniel Jackson et al., [Alloy language
  reference](https://alloytools.org/download/alloy-language-reference.pdf). Establishes that Analyzer
  search is finite and scope-bound; useful for `PLN-04` language and compact counterexample design.
- Derek Rayside et al., [evaluation of Alloy's small-scope
  hypothesis](https://groups.csail.mit.edu/sdg/pubs/2002/SSH.pdf). Empirical support for finding many
  defects at small sizes, not permission to generalize beyond the selected scope.
- Gerard Holzmann, [“Software Model Checking with
  SPIN”](https://spinroot.com/gerard/pdf/Advances2005.pdf). Mature account of explicit-state search,
  partial-order reduction, compression, and the completeness/performance trade-offs each mode makes.
- Quint project, [language/tool overview](https://github.com/quint-co/quint/blob/main/quint/README.md)
  and [model-based-testing guide](https://quint.sh/docs/model-based-testing). Typed TLA-style authoring,
  simulation, trace generation, checking, and implementation replay; useful developer-experience
  comparison, not a candidate second Umpire language.
- Apalache, [symbolic-checking principles](https://apalache-mc.org/docs/apalache/principles/index.html).
  Explains the typed transition restrictions and symbolic representation that belong in backend
  assumption receipts.
- AWS, [ShardStore lightweight-formal-methods case
  study](https://www.amazon.science/publications/using-lightweight-formal-methods-to-validate-a-key-value-storage-node-in-amazon-s3).
  Property-by-property decomposition, production-node validation, 16 reported prevented issues, and
  evidence that non-specialists maintained the checks.
- Jonathan Bowen and Michael Hinchey, [“Ten Commandments Revisited”](https://ntrs.nasa.gov/citations/20050210103).
  Industrial-practice guidance emphasizing fit, education, integration, and management rather than
  treating logic selection as the whole adoption problem.

### Lean and model-to-implementation conformance

- Daan Leijen et al., [“How We Built Cedar: A Verification-Guided
  Approach”](https://arxiv.org/abs/2407.01688). Closest Umpire analogue: Lean executable semantics,
  proofs, differential randomized testing, and PBT for unmodeled code.
- Cedar project, [formal semantics and DRT repository](https://github.com/cedar-policy/cedar-spec)
  and [RFC process](https://github.com/cedar-policy/rfcs/blob/main/README.md). Concrete structure and
  lifecycle governance.
- A. Jesse Jiryu Davis et al., [“eXtreme Modelling in
  Practice”](https://www.vldb.org/pvldb/vol13/p1346-davis.pdf) and
  [first-party retrospective](https://emptysqua.re/blog/extreme-modelling-in-practice/). The most
  useful paired success/cancellation report for selecting a conformance technique.
- Finn Hackett and Ivan Beschastnikh, [“TraceLinking Implementations with Their Verified
  Designs”](https://doi.org/10.1145/3763128). Automated trace validation, causal instrumentation,
  model-assumption bugs, finite/runtime domain separation, and explicit TCB.
- Charlie Goldweber et al., [“IronSpec: Increasing the Reliability of Formal
  Specifications”](https://www.usenix.org/system/files/osdi24-goldweber.pdf). Specification sanity
  checks and mutation tests for detecting properties that are vacuous, weak, contradictory, or
  simply not the intended contract.
- P project, [official toolchain and industrial-use overview](https://p-org.github.io/P/) and
  [monitor semantics](https://p-org.github.io/P/manual/monitors/). Communicating machines,
  systematic exploration, side-effect-free monitors, and AWS adoption claims; good comparison for
  authoring/monitor separation.
- PObserve, [runtime-monitoring repository](https://github.com/p-org/PObserve). Evidence that the P
  ecosystem is building a log-to-model monitoring bridge. Public evaluation is currently too thin
  to infer sustained production value; use as an architecture lead.
- Gabriela Moreira and Erick Pintor, [Quint Connect launch and MBT
  retrospective](https://quint.sh/posts/quint_connect). Rare first-party candor about bridge
  maintenance and duplicated-model failure modes, plus a concrete Rust trace-replay design.
- Quint Connect, [framework repository](https://github.com/quint-co/quint-connect). Defines the
  driver/state-comparison interface and seeded reproduction contract against which Umpire's
  generated artifact/runtime seam can be compared.
- Yuqi Zhang et al., [MET for CRDTs](https://arxiv.org/abs/2204.14129). Combines TLA+ checking,
  trace-derived implementation tests, deterministic driving, and message-reordering exploration;
  particularly relevant to `EXP-01` and `ART-05`.
- Andreas Cirstea et al., [TLA+ trace-validation
  study](https://arxiv.org/abs/2404.16075) and the [official TLC trace-validation
  guide](https://docs.tlapl.us/using%3Atlc%3Atrace_validation). Constrained trace completion, partial
  variable observation, and live discrepancies; strong warning that granularity and missing events
  are semantic obligations.
- Microsoft, [Spec Explorer introduction](https://learn.microsoft.com/en-us/archive/msdn-magazine/2013/december/model-based-testing-an-introduction-to-model-based-testing-and-spec-explorer).
  Clear decomposition of model, generated test, adapter, oracle, and verdict; paired with the
  Grieskamp report, it shows both technical value and adoption friction.
- Jan Tretmans et al., [model-based testing of distributed systems
  overview](https://research.cs.queensu.ca/TechReports/Reports/2008-548.pdf). Useful formal vocabulary
  for input/output conformance, quiescence, controllability, and observability; Umpire needs explicit
  distributed analogues rather than implicit driver assumptions.
- IronSpec, [artifact repository](https://github.com/GLaDOS-Michigan/IronSpec). Reproducible source
  for sanity checks, Spec-Testing Proofs, and mutation; important because surviving mutants need
  review rather than automatic rejection.
- Ivan Beschastnikh et al., [empirical defects in IronFleet, Verdi, and
  Chapar](https://homes.cs.washington.edu/~arvind/papers/dsbugs.pdf). Finds trust-boundary and
  specification/interface bugs while reporting no protocol bugs in the reviewed period; supports
  testing the proof perimeter, not dismissing the proof.
- Ilan Beer et al., [efficient vacuity detection](https://research.ibm.com/publications/efficient-detection-of-vacuity-in-temporal-model-checking).
  Formalizes trivial validity and interesting witnesses; direct precedent for Umpire trigger and
  non-vacuity qualification.
- Maxime Cordy et al., [mutation model
  checking](https://orbilu.uni.lu/bitstream/10993/59630/1/NIER_FSE23-1.pdf). Demonstrates how many
  mutated behaviors can survive apparently reasonable properties and motivates separate model and
  Property mutation metrics.

### Real-code exploration and fault injection

- FoundationDB, [simulation documentation](https://apple.github.io/foundationdb/testing.html) and
  [SIGMOD system paper](https://www.foundationdb.org/files/fdb-paper.pdf). Whole-system deterministic
  simulation and deep fault sequences.
- TigerBeetle, [architecture and VOPR
  design](https://github.com/tigerbeetle/tigerbeetle/blob/main/docs/ARCHITECTURE.md). A current,
  unusually clear account of determinism, simulation, and the formal-model/implementation gap.
- Madan Musuvathi et al., [“CHESS: A Systematic Testing Tool for Concurrent
  Software”](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/osdi2008-CHESS.pdf),
  and [Coyote](https://microsoft.github.io/coyote/). Controlled scheduling, reproducible traces, and
  honest bounded-testing language.
- Junfeng Yang et al., [“MODIST: Transparent Model Checking of Unmodified Distributed
  Systems”](https://www.usenix.org/legacy/events/nsdi09/tech/full_papers/yang/yang_html/index.html).
  Historical implementation-level exploration through interposition; useful when judging the cost
  of transparent control over an existing system.
- Christopher Meiklejohn et al., [“Service-Level Fault Injection
  Testing”](https://christophermeiklejohn.com/publications/filibuster-socc-2021.pdf). Turning passing
  functional paths into systematic partial-failure cases.
- Peter Alvaro et al., [“Lineage-driven Fault
  Injection”](https://people.ucsc.edu/~palvaro/molly.pdf). SAT-guided fault selection from outcome
  provenance.
- Ruijie Meng et al., [“Greybox Fuzzing of Distributed
  Systems”](https://arxiv.org/abs/2305.02601). Causal behavior summaries as feedback for adaptive
  fault selection.
- Ege Berkay Gulcan et al., [“Model-Guided Fuzzing of Distributed
  Systems”](https://repository.tudelft.nl/record/uuid%3A66d18d3c-fead-4df0-8310-5df11370db13).
  Direct precedent for using formal-model coverage to guide implementation testing.
- Antithesis, [deterministic-simulation explanation](https://antithesis.com/docs/resources/deterministic_simulation_testing/)
  and [platform architecture](https://antithesis.com/docs/introduction/how_antithesis_works/). A
  deterministic hypervisor avoids source-language substitution but still depends on supplied
  properties, external-dependency modeling, and exploration; useful future execution-backend option.
- AWS Labs, [Shuttle scheduler documentation](https://docs.rs/shuttle/latest/shuttle/scheduler/index.html).
  The uncontrolled-nondeterminism caveat is unusually precise: even an exhaustive scheduler wrapper
  cannot prove the program contains no untracked entropy.
- Tokio project, [Turmoil](https://github.com/tokio-rs/turmoil). Seeded single-threaded network
  simulation for Rust async services. Establishes a useful interface pattern; repository evidence
  alone does not establish general defect-finding outcomes.
- MadSim project, [deterministic distributed-system simulator](https://github.com/madsim-rs/madsim).
  Tokio/tonic/dependency substitution and injected process/network faults show both the convenience
  and fidelity cost of simulator-specific interfaces.
- Charles Killian et al., [MaceMC liveness
  checking](https://www.usenix.org/legacy/event/nsdi07/tech/killian/killian_html/index.html). Real
  implementation exploration within a distributed DSL; useful contrast with transparent
  interposition and a reminder that finite-run liveness detection relies on approximations.
- Haryadi Gunawi et al., [SAMC](https://www.usenix.org/conference/osdi14/technical-sessions/presentation/leesatapornwongsa).
  Semantic-aware implementation model checking; direct precedent for letting model coordinates
  prioritize schedules without making the execution controller authoritative.
- Lukman Rahman et al., [FlyMC](https://ucare.cs.uchicago.edu/pdf/eurosys19-flyMC.pdf). Systematic
  state-space reduction with reported confirmed defects and speedups; reductions depend on symmetry
  and independence obligations that Umpire receipts should surface.
- Madan Musuvathi et al., [CMC](https://www.usenix.org/legacy/event/nsdi04/tech/full_papers/musuvathi/musuvathi_html/).
  Early direct implementation-level checking, valuable for comparing invasive integration with
  MODIST's interposition and MaceMC's language restriction.
- Huayang Guo et al., [DEMETER dynamic interface
  reduction](https://www.microsoft.com/en-us/research/publication/practical-software-model-checking-via-dynamic-interface-reduction/).
  Shows orders-of-magnitude reduction can make meaningful spaces complete, but only relative to the
  reduced interface and supplied state abstraction.
- Haryadi Gunawi et al., [FATE and DESTINI](https://www.usenix.org/conference/nsdi11/fate-and-destini-framework-cloud-recovery-testing).
  Separates systematic failure scenarios from declarative recovery specifications and supplies one
  of the broadest quantified distributed recovery-testing evaluations.
- Netflix, [Chaos Monkey official documentation](https://netflix.github.io/chaosmonkey/). A durable
  operational example of production fault injection. It validates resilience only for injected
  instance termination and the watched outcome, not semantic completeness.

### Verified implementations and optional checkers

- Chris Hawblitzel et al., [“IronFleet: Proving Practical Distributed Systems
  Correct”](https://www.microsoft.com/en-us/research/wp-content/uploads/2015/10/ironfleet.pdf).
  End-to-end safety/liveness proof, explicit assumptions, proof burden, and new-code constraint.
- James Wilcox et al., [“Verdi: A Framework for Implementing and Formally Verifying Distributed
  Systems”](https://homes.cs.washington.edu/~mernst/pubs/verify-distsystem-pldi2015-abstract.html)
  and [repository](https://github.com/uwplse/verdi). Fault-model transforms, Raft proof, extraction,
  and runtime boundary.
- Upamanyu Sharma et al., [“Grove: a Separation-Logic Library for Verifying Distributed
  Systems”](https://iris-project.org/pdfs/2023-sosp-grove.pdf) and
  [Perennial repository](https://github.com/mit-pdos/perennial). Modern selective verification of
  distributed Go-subset components.
- Veil team, [repository](https://github.com/verse-lab/veil) and
  [CAV 2025 paper](https://verse-lab.github.io/papers/veil-cav25.pdf). Lean-embedded automated and
  interactive safety verification; relevant to Umpire's deliberately optional checker seam.
- Apalache team, [symbolic model checking
  documentation](https://apalache-mc.org/docs/tutorials/symbmc.html) and
  [bounded-checking limitations](https://apalache-mc.org/docs/apalache/running.html). Model for
  precise backend receipt language.
- Xudong Sun et al., [Anvil OSDI 2024](https://www.usenix.org/system/files/osdi24-sun-xudong.pdf)
  and [repository](https://github.com/anvil-verifier/anvil). Runnable verified Kubernetes controllers,
  explicit eventual-stability assumptions, maintenance data, and a trusted-API failure exposed by
  runtime testing.
- Andrea Lattuada et al., [“Verus: A Practical Foundation for Systems
  Verification”](https://www.microsoft.com/en-us/research/uploads/prod/2024/09/Verus.pdf). Quantifies
  proof/checking improvements for systems-scale Rust verification and clearly separates source-level
  verification from its compiler/runtime trust base.
- Verus project, [verified IronKV port](https://github.com/verus-lang/verified-ironkv). Exceptionally
  useful explicit limitation: host implementation proof only, without IronFleet's distributed
  refinement layer or crash recovery.
- Morten Krogh-Jespersen et al., [Aneris project and papers](https://iris-project.org/aneris/).
  Higher-order distributed separation logic for unreliable UDP-like networks and modular CRDT/causal
  case studies; a future selective proof pattern, not a whole-server proposal.
- James Wilcox et al., [Verdi Raft completion and maintenance
  paper](https://homes.cs.washington.edu/~mernst/pubs/raft-proof-cpp2016.pdf). Documents the move from
  a conditional result to roughly 45K proof lines and 90 invariants, plus proof-engineering techniques
  for controlling change cost.
- Grove team, [artifact](https://github.com/mit-pdos/grove-artifact). Reproduction source for the
  verified components and TCB described by the paper; useful for distinguishing verified Go-subset
  code from trusted network/filesystem libraries.
- seL4 project, [verification assumptions](https://www.sel4.org/Verification/assumptions.html).
  Outside distributed systems but exemplary TCB disclosure: hardware, boot, assembly, DMA, and other
  premises remain visible instead of being absorbed by the word “verified.”

### Runtime evidence, ordering, and black-box checking

- Leslie Lamport, [“Time, Clocks, and the Ordering of Events in a Distributed
  System”](https://lamport.azurewebsites.net/pubs/time-clocks.pdf). Why causality and wall-clock time
  are not interchangeable.
- OpenTelemetry, [Tracing SDK specification](https://opentelemetry.io/docs/specs/otel/trace/sdk/).
  Sampling and recording rules that make completeness an explicit Evidence concern.
- Peter Bailis et al., [Elle repository](https://github.com/jepsen-io/elle) and
  [paper](https://raw.githubusercontent.com/jepsen-io/elle/master/paper/elle.pdf). Scalable inference
  from observed histories with clearly stated detectability boundaries.
- Ivan Beschastnikh et al., [“Debugging Distributed Systems” / ShiViz experience
  report](https://www.cs.ubc.ca/~bestchai/papers/cacm2016-shiviz.pdf). Why ordinary logs need causal
  instrumentation before they can reconstruct distributed order.
- Anish Athalye, [Porcupine](https://github.com/anishathalye/porcupine). Fast linearizability
  decision procedure, executable sequential Go model, visualizer, and explicit history/state-space
  limitations; a strong specialized checker but not a semantic source for Umpire.
- Lauren Pick et al., [Troubadour observational-correctness
  checker](https://doi.org/10.1145/3720504). Combines transaction semantics and isolation constraints
  over supplied observations, finding two new bugs in the evaluation; useful beyond simple object
  linearizability.
- Kevin Havelund et al., [partial-order runtime
  verification](https://users.ece.utexas.edu/~garg/dist/PartialOrderVerification.pdf). Checks
  distributed traces without an arbitrary totalization; supports Umpire's causal-order-first Evidence
  policy.
- Borzoo Bonakdarpour et al., [failure-aware runtime
  verification](https://drops.dagstuhl.de/entities/document/10.4230/LIPIcs.FSTTCS.2015.590).
  Multi-valued semantics under network failure and reordered communication; precedent for `unknown`
  rather than false pass/fail certainty.
- Jonathan Mace et al., [Pivot Tracing](https://www.microsoft.com/en-us/research/publication/pivot-tracing-dynamic-causal-monitoring-for-distributed-systems-2/).
  Happened-before joins across component boundaries show how causal relations can support useful
  queries without trusting wall-clock order.
- Ivan Beschastnikh et al., [ShiViz/XVector instrumentation and
  evaluation](https://homes.cs.washington.edu/~mernst/pubs/visualize-distributed-tosem2020-abstract.html).
  Concrete evaluation of causality-carrying logs; useful for designing Umpire Evidence profiles and
  debugging views.
- Leslie Lamport and Bowen Alpern, [recognizing safety and
  liveness](https://research.ibm.com/publications/recognizing-safety-and-liveness). The foundation for
  the claim that finite bad prefixes can refute safety while unbounded liveness needs infinite-trace
  reasoning or extra assumptions.
- Andreas Bauer, [runtime monitorability study](https://doi.org/10.1016/j.tcs.2014.02.052). Formal
  limit on which properties can yield decisive finite runtime verdicts; supports bounded-response
  Properties rather than unbounded “eventually” from test runs.

### Workflow semantics and structural correctness

- Wil van der Aalst, [original workflow-net verification
  paper](https://vdaalst.com/publications/p44.pdf). Defines the basic soundness question for workflow
  procedures and shows how Petri-net analysis detects improper completion and dead transitions.
- Wil van der Aalst et al., [soundness classification and decidability](https://www.vdaalst.com/publications/p628.pdf).
  Separates eight notions and shows that seemingly useful expressive extensions can cross into
  undecidability; primary caution against one vague `workflow sound` bit.
- Guanjun Liu et al., [workflow-net soundness complexity](https://doi.org/10.3233/FI-2014-1005).
  PSPACE-completeness for bounded WF-nets; a scalability warning even before Temporal-specific data,
  retries, and faults are added.
- Remco Dijkman et al., [formal BPMN mapping and analysis](https://doi.org/10.1016/j.scico.2018.05.008).
  Demonstrates that model checking a business notation depends on a defensible translation; relevant
  to Umpire's prohibition on generated artifacts adding semantics.
- Nick Russell et al., [workflow-pattern expressiveness analysis](https://www.vdaalst.com/publications/p562.pdf).
  Pattern-oriented decomposition of control flow; useful source of small structural Properties and
  negative controls rather than a complete Temporal Feature model.

### Provenance, attestation, and assurance cases

- W3C, [PROV family overview](https://www.w3.org/TR/prov-overview/) and
  [primer](https://www.w3.org/TR/prov-primer/). Standard entity/activity/agent derivation vocabulary;
  helpful for Evidence Links while explicitly not establishing truth of derived content.
- IETF, [RFC 9334: RATS architecture](https://www.rfc-editor.org/rfc/rfc9334.html). Clean separation
  of raw Evidence, appraisal policy, Attestation Result, and relying-party decision maps closely to
  Execution, Run Evaluation, Claim Assessment, and rollout control.
- in-toto, [supply-chain layout/link specification](https://github.com/in-toto/docs/blob/master/in-toto-spec.md).
  Signed step/material/product provenance and threshold roles; useful artifact-authenticity model,
  but no substitute for behavioral semantics.
- SLSA, [provenance specification v1.2](https://slsa.dev/spec/v1.2/). Current build provenance and
  producer/subject identity vocabulary; reinforces checksum-versus-authenticity separation.
- GSN community, [Goal Structuring Notation standard](https://www.faa.gov/about/office_org/headquarters_offices/ang/redac/redac-sas-201503-gsn-community-standard-v1.pdf).
  Claims, strategies, context, assumptions, and evidence integrity; a good review model for Umpire
  Claim Assessments without requiring graphical GSN artifacts.
- OMG, [Structured Assurance Case Metamodel 2.2](https://www.omg.org/spec/SACM/2.2/PDF). Machine-readable
  assurance-case vocabulary and artifact references; useful comparison for portable Claim Assessment
  schemas, though adopting the full metamodel would add unjustified complexity.
- NIST, [assurance-case definition](https://csrc.nist.gov/glossary/term/assurance_case). Concise
  authoritative statement that assumptions and underlying evidence belong in an auditable claim,
  supporting `QLF-03`.

### Industrial sustainability and limits

- CICS team, [first-party Z-method account](https://research.ibm.com/publications/use-of-software-engineering-including-the-z-notation-in-the-development-of-cics)
  with an [independent quantitative critique](https://doi.org/10.1016/S0164-1212%2896%2900122-7) and
  [later history](https://doi.org/10.1145/3522577). The critique accepts the case's value while
  rejecting unsupported headline metrics; the three-way comparison reduces narrative bias.
- Microsoft Research, [Farsite retrospective](https://www.microsoft.com/en-us/research/wp-content/uploads/2007/04/OSR2007-4aa.pdf).
  A research system can produce influential techniques without becoming sustained production
  infrastructure; supports nuanced outcome categories.
- Jean-Raymond Abrial et al., [25-year B/Event-B trajectory](https://arxiv.org/abs/2005.07190).
  Longitudinal view of tool, training, and organizational evolution across rail, smartcard, and
  automotive applications.
- Ralf Huuck et al., [practical issues applying formal methods in
  industry](https://brucker.ch/publications/altenhofen.ea-issues-2010/). Concrete integration,
  notation, scalability, and skill obstacles that belong in Umpire's adoption metrics.
- Jim Woodcock et al., [formal-methods industrial survey](https://vsr.sourceforge.net/fmsurvey.htm).
  Broad case inventory; useful for pattern discovery but heterogeneous enough that individual
  outcomes should be checked against their primary case reports.
- NASA/AECL authors, [13-year nuclear shutdown-system
  case](https://doi.org/10.1007/978-3-540-45236-2_9). A sustained safety-critical lifecycle example
  showing that longevity depends on integration into routine engineering and assurance work.

### Temporal-specific baseline

- Temporal, [server architecture](https://github.com/temporalio/temporal/blob/main/docs/architecture/README.md),
  [History Service architecture](https://github.com/temporalio/temporal/blob/main/docs/architecture/history-service.md),
  and [Nexus architecture](https://github.com/temporalio/temporal/blob/main/docs/architecture/nexus.md).
  The concrete distributed mechanisms Umpire must model selectively.
- Temporal, [SDK feature and history compatibility
  suite](https://github.com/temporalio/features). Existing cross-language execution, history replay,
  exact normalized comparison, and CI practice.
- Temporal, [server testing guide](https://github.com/temporalio/temporal/blob/main/docs/development/testing.md).
  Existing unit/functional-test hooks, exact history checks, test clusters, and trace capture that
  Umpire should complement rather than duplicate.

## Research gaps and cautions

Link validation on 2026-08-30 found 138 unique HTTP(S) references. A parallel `curl` range-request
check followed redirects and received a successful 2xx/3xx response for 134. Four DOI resolver URLs
returned HTTP 403 to the automated client even though their publisher/search metadata resolved in
the research pass: TraceLink, Troubadour, workflow-net complexity, and the later CICS history. This
check cannot establish source quality, the truth of a claim, paywalled full-text access, JavaScript-
rendered content, redirect stability, or whether a repository will remain maintained. Local relative
links were checked separately against this repository.

- Most published distributed-system verification papers evaluate a handful of protocols or newly
  written systems. They do not establish that the same proof engineering scales to the entire
  existing Temporal Go server. IronFleet explicitly limits its claim to newly written
  verification-friendly code ([paper](https://www.microsoft.com/en-us/research/wp-content/uploads/2015/10/ironfleet.pdf)).
- Industrial reports are often first-party experience reports rather than controlled independent
  replications. Treat their bug counts and productivity observations as evidence about those
  projects, not universal effect sizes.
- Model-based testing can fail for organizational and tooling reasons even when its semantics are
  sound. Microsoft's internal report estimated only 5–10% of product teams had used or tried its MBT
  tools and listed authoring, state explosion, documentation, and workflow integration as obstacles
  ([Microsoft report](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/tr-2006-111.pdf)).
- Runtime validation is limited by observation. Trace sampling, missing causal metadata, ambiguous
  bindings, and unmodeled environment behavior all weaken claims
  ([OpenTelemetry](https://opentelemetry.io/docs/specs/otel/trace/sdk/),
  [TraceLink](https://doi.org/10.1145/3763128)).
- No reviewed source found here demonstrates the complete Umpire combination—one Lean-owned product
  and system model reused unchanged for proof, bounded planning, real Temporal execution, causal
  evidence interpretation, black-box qualification, model-guided fuzzing, exact replay, and reviewed
  promotion. That combination is the project's principal opportunity and also its principal
  integration risk.

The practical success criterion is therefore not “formally verify Temporal” in one step. It is to
make each bounded claim precise, reproducible, auditable, and useful enough that Temporal engineers
choose the model again for the next regression and the next unknown bug.
