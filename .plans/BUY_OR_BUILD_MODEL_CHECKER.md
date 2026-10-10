# Buy or build Umpire's model checker

Research findings, 2026-10-10. This compares extending Umpire's own checker, generating
Quint source or IR, and generating TLA+ directly. Recommendations are engineering judgments;
they do not amend the shared specification or approve a backend migration.

## Recommendation

**Keep the native Go checker and use generated Quint source as the first external checking
route. Compare it with a small direct TLA+ exporter before expanding either into full coverage.**
Use TLC for exhaustive checking and temporal properties over finite models; use Apalache for
symbolic safety checks with explicit execution bounds. Avoid building a general symbolic or
temporal model checker ourselves.

For Umpire, these are complementary responsibilities. Go already interprets the IR, answers
Queries, checks refinement and progress, generates deterministic witnesses, and supports runtime
conformance. External checkers contribute independent evaluation and search algorithms. None of
the three options eliminates Umpire's responsibility for Query semantics, realization, evidence,
Case production, and replay.

| Option | Principal benefit | Principal cost | Recommendation for Umpire |
| --- | --- | --- | --- |
| 1. Extend our checker over Umpire IR | Exact domain semantics and integration with existing Go tooling | Own correctness, memory use, search algorithms and every reduction | Continue the existing bounded checker in Go; constrain its scope |
| 2. Generate Quint source | Existing exporter, type/effect checks, evaluator, and access to Apalache and TLC | Another translation layer, tool dependencies and current wrapper limitations | Preferred first external route |
| 2a. Generate Quint JSON IR | Skip source parsing; direct Apalache import exists | Construct typed, resolved, flattened IR and track its schema | Consider only after measuring a source-pipeline bottleneck |
| 3. Generate typed TLA+ directly | Control encoding, checker configuration and trace handling | Own relational lowering, annotation generation and compatibility with two checkers | Credible alternative if Quint's extra layer provides too little value |

If choosing just one *new external* integration, choose option 2's source path today. If the
long-term requirement is primarily headless Apalache/TLC checking and Quint's evaluator becomes
unnecessary, option 3 may become the smaller maintained system. A greenfield checker rewrite in
Go, Scala or Rust has the weakest justification.

The [exporter estimates](#effort-estimates-for-the-exporter-routes) budget **5-10 person-weeks
for the existing Quint route and 7-12 for direct TLA+**, for the same scoped core. Direct export
means one common typed TLA+ emitter with two checker adapters. Quint also needs backend adapters;
it mainly saves compiler work. Both routes must replace the current growing execution-history
variable with finite observer state before exhaustive TLC checking.

## What the repository already tells us

The inspected checkout was `d4b8bb6bedb898a023c17d77129ec5434626184b`, with an existing local edit
to `Heartbeat.scala`. The findings concern the current files and separately identified historical
evidence; recorded gate results were not rerun for this study.

- [UMPIRE4_VISION.md](UMPIRE4_VISION.md#high-level-architecture) gives Go the interpretation,
  bounded Search, witness and judgment responsibilities. Other checkers supply a second opinion.
  [UMPIRE4_SPEC.md](UMPIRE4_SPEC.md#source-of-truth) keeps behavior in the Model, and
  [UMPIRE_MODULES.md](UMPIRE_MODULES.md#module-ownership) assigns interpretation, checking and
  export to separate Go modules. Replacing the native checker as the authority would require an
  explicit architecture decision.
- [SCALA.md](SCALA.md) recommends Quint first and separates Scala authoring from Go execution.
  Its older descriptions of checker capabilities need to be read against current source.
  [UMPIRE4_TLA_COMPAT.md](UMPIRE4_TLA_COMPAT.md#recommendation) already proposes comparing the
  Quint route with a minimal direct export and supplies the semantic translation contract.
- [QUINT_MODULE_LAYOUT.md](QUINT_MODULE_LAYOUT.md) adopted Quint's module reading order for
  Scala Models. That authoring decision does not select a verification backend.
- [model/SEMANTICS.md](../model/SEMANTICS.md) defines finite typed values, ordered expression
  evaluation, atomic alternatives, channels, monitors, holes, compositions, refinement, bounded
  Queries and progress. These are the semantics every route must preserve.
- The [current export README](../tools/umpire/export/README.md) documents independent Quint
  translation and comparison of transitions, Properties and monitor products. Its exhaustive
  agreement dump computes a reachable-state fixed point inside Quint, then uses one simulator
  step to output it. This particular dump is deterministic enumeration; ordinary `quint run`
  samples executions.
- The same README records unsupported Query, progress and refinement agreement, refusal of
  channels and holes, and an Apalache inlining failure on Nexus close. Four activity monitor
  checks use bounded Apalache verification; Nexus monitor agreement uses the evaluator instead.
  Existing Quint export is useful evidence, but it is not full IR-to-model-checker coverage.

There is concrete resource pressure on both routes. The deferred
[fn-154 Quint JSON work](../.flow/specs/fn-154-bound-memory-for-exhaustive-quint-json.md) records a
1,894,733,345-byte synthesized Activity dump and a 141,982,619-byte section producing
2,387,566,128 bytes of generic decoding allocations. The
[fn-157 native verification investigation](../.flow/specs/fn-157-bound-native-verification-memory-and.md)
records native generator and checker OOMs, including a generator killed at 8,581,584 KiB anonymous
RSS. These are observations from those runs, not current benchmarks or proof of a particular
allocation cause. Switching to Rust or removing Quint would not, by itself, resolve both problems.

## What each kind of check establishes

| Check | What a completed clean result establishes | Limit that must remain visible |
| --- | --- | --- |
| Native Umpire Query | Its Property holds, or no requested witness exists, over the exact Scenario within its Limits | Holes, unread claims, search ceilings and unsupported constructs |
| Ordinary Quint simulation | The sampled executions contained no detected invariant violation | Samples and seed; no exhaustive guarantee |
| Apalache bounded checking | No encoded violation exists within the specified execution length | Model domains, assumptions, encoding and depth |
| TLC exhaustive model checking | The selected Properties hold throughout the completely explored finite model | Finite abstraction, fairness and any exploration restrictions |
| Apalache inductiveness checking | A supplied inductive invariant can establish safety for all execution lengths of the encoded system | Separate initialization, preservation and safety-implication obligations |

Quint exposes both Apalache and TLC through `quint verify --backend apalache|tlc`.
Its Rust evaluator is a simulation/testing backend, not an exhaustive model checker.
[Quint model checkers](https://quint.sh/docs/model-checkers),
[simulator](https://quint.sh/docs/simulator).

Apalache's ordinary bounded success is not an all-length proof. Its inductiveness workflow checks
initialization and one-step preservation of a stronger invariant, plus that invariant's implication
of the desired safety Property. Umpire would need to declare and validate these obligations;
ordinary `quint verify` does not automatically synthesize them.
[Apalache running and inductiveness](https://apalache-mc.org/docs/apalache/running.html).

TLC's complete exploration can cover executions of arbitrary length of a finite instance, including
cycles. It does not establish correctness for arbitrary task counts, channel capacities or integer
domains. [Lamport's account of TLC](https://lamport.azurewebsites.net/tla/book-02-03-04.pdf).

## What mature checkers would be expensive to reproduce

**A finite successor evaluator and breadth-first search are feasible to own. A general temporal
or symbolic verification engine is a much larger commitment.** TLA+ is the specification language;
TLC and Apalache are different checking engines. Their valuable capabilities extend beyond parsing
the language or enumerating transitions.

| Capability | What an established engine supplies | What Umpire would have to own |
| --- | --- | --- |
| Temporal logic and fairness | TLC checks temporal obligations over a finite graph, including infinite behaviors represented by cycles | General formula handling, fairness-sensitive cycle acceptance, and replayable prefix-plus-loop counterexamples |
| Exploration beyond an in-memory prototype | TLC supplies disk-backed fingerprints and queues, multiple workers, checkpoints and recovery | Compact state identity, storage and queue lifetimes, concurrent exploration, recovery, and trustworthy completion accounting |
| Sound state-space reductions | TLC offers symmetry reduction with restrictions; SPIN offers mature partial-order reduction | Prove which identities or interleavings can be merged while preserving the chosen Properties, monitors and fairness |
| Symbolic search | Apalache translates supported typed TLA+ into SMT problems and reconstructs solver counterexamples | Encode values and transition relations, manage solver queries, distinguish unknown from unsatisfiable, and decode models |
| Long-term correctness work | Established engines have regression suites and fixes accumulated across many models | Maintain negative controls and regressions for every interaction among search, reductions, concurrency and temporal semantics |

For TLC's temporal and operational facilities, see the
[TLC tool documentation](https://github.com/tlaplus/tlaplus/blob/master/general/docs/current-tools.md)
and [Lamport's TLC chapter](https://lamport.azurewebsites.net/tla/book-02-03-04.pdf).
For partial-order reduction and generated verification code, see the
[SPIN manual](https://spinroot.com/spin/Man/Spin.html). For symbolic checking, see
[Apalache's architecture](https://apalache-mc.org/docs/apalache/index.html).

Reductions are especially easy to get wrong. Two tasks that appear interchangeable may differ in
Facts, monitor state or visible identity. Two Steps that leave the machine state unchanged may
still exercise different action classes. TLC also warns that symmetry reduction is unsuitable for
liveness checking because it can conceal violations. Umpire would need to qualify each reduction
against its own semantics, rather than treating smaller graphs as automatically equivalent.
[TLC symmetry restrictions](https://nightly.tlapl.us/doc/model/model-values.html).

Mature tools still have bugs. TLC's 1.7.4 release fixed a case where multiple-worker liveness
checking could miss a violation. This illustrates the difficulty of combining concurrency and
temporal checking, and the value of an established regression history; it is not a claim that an
external result needs no qualification.
[TLC 1.7.4 release](https://github.com/tlaplus/tlaplus/releases/tag/v1.7.4).

Umpire already handles deadlock, deadline misses and some fair non-progress cycles. Reusing an
external checker buys broader algorithms while keeping Umpire's Query interpretation, domain
diagnostics and witness qualification. The finite IR also gives a custom checker useful constraints:
we do not need arbitrary TLA+ evaluation to implement the native bounded contract.
[Native progress engine](../tools/umpire/internal/engine/progress.go),
[IR semantics](../model/SEMANTICS.md).

## Free tools and their fit for Umpire

There is no single best checker across distributed protocols, relational models, probabilistic
systems and implementation code. The following are free, open-source candidates; the ranking is
an assessment of Umpire's needs, not a performance benchmark.

| Tool | Strongest use | Fit and limitation for Umpire |
| --- | --- | --- |
| **TLC** | Exhaustive finite-state safety and temporal/fairness checking of TLA+ | First choice for finite distributed behavior and lasso evidence; explicit enumeration still faces state explosion. [Documentation](https://github.com/tlaplus/tlaplus/blob/master/general/docs/current-tools.md) |
| **Apalache** | SMT-based bounded safety checking and supplied inductive-invariant checks | Complement TLC when enumerating the reachable graph is costly; requires supported types/operators and explicit bounds or proof obligations. [Documentation](https://apalache-mc.org/docs/apalache/index.html) |
| **SPIN** | Asynchronous communicating processes, LTL and partial-order reduction | Strong alternative if interleaving explosion becomes the measured bottleneck; requires a Promela exporter and qualification of Umpire's atomic Steps. Exact and approximate modes must be distinguished. [Manual](https://spinroot.com/spin/Man/Spin.html) |
| **mCRL2** | Communicating processes, behavioral equivalence and modal mu-calculus Properties | Worth considering if independent behavioral equivalence/refinement analysis becomes a priority; its equivalence definitions need an explicit mapping to Umpire refinement. [Analysis documentation](https://mcrl2.org/web/user_manual/language_reference/pbes.html) |
| **NuSMV** | BDD/SAT checking of finite transition systems with CTL/LTL | Useful for a compact finite encoding; less natural for Umpire's rich values and diagnostics than the existing Quint/TLA+ route. [Overview](https://nusmv.fbk.eu/overview.html) |
| **Alloy 6** | Relational structures, topology, ownership and temporal constraints | Useful for structural design questions; finite object scopes remain explicit even when using complete temporal analysis. A separate relational encoding would be needed. [Temporal analysis](https://alloytools.org/alloy6.html) |
| **Storm / PRISM** | Quantitative probabilistic checking, such as failure probabilities and expected costs | Appropriate only after the Model defines meaningful probabilities or rates. Nondeterministic loss does not itself supply a probability distribution. [Storm](https://www.stormchecker.org/), [PRISM](https://www.prismmodelchecker.org/) |
| **CBMC** | Bounded verification of actual C/C++ code, assertions and memory safety | Useful for implementation verification in those languages; does not directly cover Umpire's Go interpreter or Scala authoring. [Project](https://www.cprover.org/cbmc/) |
| **P** | Event-driven state machines, monitors and systematic schedule exploration | Close domain fit, but adds another language/compiler. Ordinary schedule testing is not an exhaustive verdict; qualify its distinct exhaustive/proof backends separately. [Checking guide](https://p-org.github.io/P/getstarted/usingP/) |

Quint belongs in the integration comparison as a typed language, compiler and evaluator that
connects to TLC and Apalache. Selecting Quint does not add a third exhaustive checking engine.
[Quint backends](https://quint.sh/docs/model-checkers).

“Free” must include the intended use. nuXmv's free binary license excludes commercial use, and
UPPAAL's free license is restricted to academic use; neither belongs in an unrestricted commercial
shortlist. Open-source tools can also have redistribution obligations, so pin the selected artifacts
and their licenses when packaging them.
[nuXmv licensing FAQ](https://nuxmv.fbk.eu/faq-list.html),
[UPPAAL licensing](https://uppaal.org/downloads/).

**Selection for this plan:** qualify TLC plus Apalache through the Quint/direct-TLA+ experiment.
Keep SPIN as a targeted alternative if measured interleaving costs justify another exporter, and
mCRL2 as a targeted alternative for equivalence analysis. The remaining tools answer other kinds of
questions and do not justify expanding the initial backend scope.

## 1. Build on Umpire IR

### Benefits

**Much of the bounded checker already exists.**
[interp/machine.go](../tools/umpire/interp/machine.go) constructs finite catalogs and transition
tables; [engine/search.go](../tools/umpire/internal/engine/search.go) searches the product of
machine state, Scenario position, Property activation and monitor state.
[engine/progress.go](../tools/umpire/internal/engine/progress.go) handles deadlocks, fair
non-progress cycles and deadline misses. This option means extending these modules, rather than
starting from a parser and graph traversal.

The native path can preserve Umpire's distinctions directly. Disabled actions, declaration errors,
unknown behavior, unexercised Properties and exhausted Search remain separate outcomes. It can
choose witnesses deterministically, retain Definition IDs and Behavior Fingerprints, and feed
existing lowering and conformance without roundtripping through an external trace format.
[Checker receipts](../tools/umpire/check/checking.go),
[IR results](../model/SEMANTICS.md#results).

It also offers a useful place for Umpire-specific diagnostics and coverage. For example, the checker
can explain which action class exercised a promise and which realization prevents its witness from
becoming a Case. A generic model checker does not know these concepts.

### Costs and risks

The current interpreter enumerates finite state catalogs and state/action pairs before Search.
Those catalogs include unreachable combinations. Reusing tables, compacting values or eventually
evaluating reachable successors on demand may help, but identity generation, refinement and
whole-table diagnostics still need their documented behavior. A faster graph traversal alone does
not remove the earlier allocation work.
[Catalog and behavior construction](../tools/umpire/interp/machine.go),
[fn-157 investigation boundaries](../.flow/specs/fn-157-bound-native-verification-memory-and.md).

Building a more general checker adds responsibilities beyond IR evaluation. We would own temporal
formula handling, fairness algorithms, complete state identity, trace reconstruction, parallel
search, storage limits, and the correctness conditions of symmetry or partial-order reduction.
Building symbolic checking would additionally require IR-to-SMT encoding, solver integration and
model decoding. Calling an existing SMT solver still leaves that verification engine ours.

A shared evaluator creates a shared blind spot. Search, lowering and runtime conformance can agree
because they all execute the same incorrect interpretation. External translation helps challenge
that interpretation, but even two tools cannot detect a requirement omitted from the authored
Model. [MODEL_ASSURANCE.md](MODEL_ASSURANCE.md#independent-evidence-and-closing-the-gap).

**Scope the build decision.** Owning finite IR evaluation and bounded Umpire Queries is justified.
Owning a competitor to TLC and Apalache needs evidence of a requirement that those engines cannot
meet economically.

### Which implementation language?

This table ranks fit for this repository, not hypothetical maximum performance.

| Language | Advantages for Umpire | Costs and risks | Verdict |
| --- | --- | --- | --- |
| **Go** | Reuse the existing interpreter, protobuf schema, checking, replay, lowering and conformance; one runtime and one implementation of native semantics | Allocation-heavy value/table representations need measurement and repair; a new SMT or temporal engine would still be substantial work | Best choice for extending the native checker and for exporters |
| **Scala** | Familiar typed ASTs and pattern matching beside the lifter; natural fit if deliberately extending Apalache's Scala libraries | Moving checking into the JVM separates it from Go consumers or introduces another evaluator; JVM deployment and heap lifetimes remain | Choose for an upstream Apalache/compiler contribution, not merely because Models are Scala |
| **Rust** | Explicit ownership and compact representations can be attractive for a measured search/storage kernel | Rewrite or cross-language process/FFI boundary, protobuf/value conversion, duplicate semantic maintenance, and another toolchain | Consider only for a profiled kernel with a demonstrated benefit |

Scala authoring does not imply that Scala should execute the Model. Rust's memory model does not
remove a Cartesian state space or a multi-gigabyte JSON boundary. First measure retained data,
transient copies and lifetime overlap; then change the representation or the implementation that
owns the measured cost. No language performance comparison was run for this report.

## 2. Emit Quint source or IR

### Benefits

Quint source offers another implementation of types, expression evaluation and action validation.
Its effect checks distinguish pure expressions from state reads, updates and temporal formulas,
helping detect malformed generated actions. Umpire retains its own IR admission, since valid Quint
alone cannot establish that the translation preserved a Query's meaning.
[Quint effect-system design](https://quint.sh/docs/development-docs/architecture-decision-records/adr004-effect-system).

The existing `.qnt` exporter reduces the incremental work. Generated source can be inspected,
evaluated and checked externally without asking feature authors to maintain a second specification.
Quint also provides simulation, tests and readable development artifacts. Their value is greatest
when we use them for independent evaluation and debugging; Scala already supplies Umpire's primary
authoring experience.

The route can use symbolic safety checking through Apalache and finite temporal checking through
TLC. TLA+ export is available as `quint compile model.qnt --target tlaplus`.
Compilation invokes Apalache; it is not a standalone Quint pretty-printer.
[Quint CLI](https://quint.sh/docs/quint),
[pinned compiler](https://github.com/quint-co/quint/blob/v0.33.0/quint/src/compileToTlaplus.ts).

### Costs and risks

Quint reduces general compiler work, but Umpire still has to encode atomic Steps, Facts, activation,
monitors, channels, refinement and Query scope. A typed `.qnt` module may faithfully execute its own
definitions while answering a different question from Go. Current exporter support gaps are
concrete examples of the remaining work.
[Export support profile](../tools/umpire/export/README.md#what-the-exporters-refuse).

The pipeline adds failure and diagnostic stages. A source-location map must relate Scala/IR nodes
to generated Quint, flattened definitions, exported TLA+ and decoded checker values. The current
npm installation adds Node/npm; checking adds a JVM and external tools. More stages also create
more version combinations to qualify.

**Quint's TLC wrapper currently needs additional integration work for Umpire.** In v0.33.0 it gets
the TLC JAR from the Apalache distribution, disables TLC's built-in deadlock check, shows raw TLC
output only at verbosity 3 or higher, and removes temporary output. Its result path does not return
a structured TLC counterexample like the Apalache ITF path. Umpire needs replayable witnesses and
explicit deadlock meaning, so the simplest reliable arrangement may be Quint compilation followed
by our own TLC invocation and trace adapter.
[Verifier](https://github.com/quint-co/quint/blob/v0.33.0/quint/src/verify.ts),
[TLC adapter](https://github.com/quint-co/quint/blob/v0.33.0/quint/src/tlc.ts).

Quint v0.33.0 also warns that Apalache temporal checking may give incorrect results and recommends
TLC. Direct TLA+ emission would still use the same Apalache engine, so removing Quint does not
establish reliable Apalache liveness. Prefer TLC for temporal assurance; keep bounded deadline
checks expressible as safety obligations where their mapping is exact.
[Quint warning](https://github.com/quint-co/quint/blob/v0.33.0/quint/src/cliCommands.ts#L655).

The large exhaustive dump is a particular agreement design cost, not an unavoidable cost of every
Quint model check. Streaming agreement data and giving a checker a compact transition relation are
different jobs. They need separate resource evidence, and neither may silently replace exhaustive
agreement with sampling.
[fn-154](../.flow/specs/fn-154-bound-memory-for-exhaustive-quint-json.md).

### Source versus JSON IR

**Generate source first.** The normal Quint CLI parses source text; JSON is an output of its compiler
stages, not a drop-in input to `quint verify generated.json`. Source generation lets Quint build
resolution, typing and effect metadata, and gives reviewers an artifact they can read.
[CLI loader](https://github.com/quint-co/quint/blob/v0.33.0/quint/src/cliCommands.ts#L167).

There is nevertheless a real direct-IR route. Apalache 0.62.1's JSON-RPC `loadSpec` accepts Quint
JSON using `format: "qnt"`. Its importer requires `stage`, `modules`, `types` and `table`; it
translates the first module and expects flattened definitions. Typed, resolved, flattened compiler
output is the appropriate input. Raw parsed syntax is insufficient. Effects and source maps are
not required by that importer, although Umpire would still need its own admission and diagnostic
mapping.
[Quint IR importer](https://github.com/apalache-mc/apalache/blob/v0.62.1/tla-io/src/main/scala/at/forsyte/apalache/io/quint/QuintIR.scala#L104),
[format loader](https://github.com/apalache-mc/apalache/blob/v0.62.1/passes/src/main/scala/at/forsyte/apalache/tla/passes/imp/SanyParserPassImpl.scala#L55).

`loadSpec` loads and preprocesses input for symbolic exploration. A successful load is not an
all-path verification result; the adapter still needs an actual checking workflow.
[Apalache JSON-RPC protocol](https://github.com/apalache-mc/apalache/blob/v0.62.1/json-rpc/README.md#31-method-loadspec).

Hand-emitting that format can remove text parsing, but transfers ID assignment, name-reference
tables, type schemes and flattening to Umpire. It also bypasses the Quint frontend checks unless
we invoke them separately. IR integration is justified only if those savings outweigh maintaining
this compiler boundary. A Quint-language IR importer and Apalache's own TLA IR are distinct formats.

## 3. Emit TLA+ directly

### Benefits

Direct generation gives Umpire control over action shape, helper expansion, enabledness,
configuration and source mapping. It removes dependence on Quint's language lowering and lets us
run checker commands directly, retain artifacts, classify failures and decode traces ourselves.
For a small, deliberately restricted IR subset, that can be easier to maintain than adapting
several compiler representations.

One readable `.tla` artifact can target both TLC and Apalache if we deliberately choose their
shared executable subset. Add type annotations for Apalache and keep separate checker
configurations. TLA+ remains useful to reviewers and existing TLA+ tooling even if Quint is absent.

Direct export also permits experiments with solver-friendly encoding that a generic compiler may
not choose. For example, explicit per-class actions and bounded records may be preferable to
deeply nested list transformations. This is a hypothesis to measure on Umpire fixtures, not evidence
that direct export will outperform the existing route.

### Costs and risks

We must implement the relational action lowering Quint currently supplies. Every branch needs the
right old-state reads, nondeterministic choices and complete next-state assignments. Apalache also
needs usable types, particularly around empty collections, tuple/sequence distinctions and tagged
values. Its transition extraction requires an assignment for every state variable in each symbolic
transition.
[Apalache assignment rules](https://apalache-mc.org/docs/apalache/principles/assignments.html),
[sequence typing](https://apalache-mc.org/docs/lang/sequences.html).

Valid TLA+ does not imply support by Apalache. The supported-feature table excludes direct
`ENABLED`, `WF` and `SF`, unrestricted `CHOOSE`, recursion and several module constructs.
Apalache parses `ASSUME` without propagating it to the solver. Generated domain or behavior
restrictions must enter the actual checked initialization/actions or other supported encoding;
writing them only as `ASSUME` can lose them.
[Apalache supported features](https://apalache-mc.org/docs/apalache/features.html).

Consequently, a TLC fairness specification and an Apalache safety specification may need different
generated wrappers over the same declared relation. Neither wrapper may add assumptions. Direct
Apalache-only checking would give up TLC's mature finite temporal path and would not match the
capabilities available through option 2.

Removing Quint does not remove Apalache's inliner, SMT encoding, solver costs or bounded nature.
It may allow an encoding that avoids the recorded Nexus failure, but that needs an actual check.
Direct generation also sacrifices Quint's evaluator as a convenient independent semantic oracle,
unless we retain that path separately.

Apalache's own JSON IR and published Scala libraries are alternative interfaces, but they make
Umpire depend on compiler structures. The JSON format is versioned and may reject obsolete
encodings. Prefer `.tla` as the initial boundary; select a library or IR only for a demonstrated
integration need.
[Apalache JSON format](https://apalache-mc.org/docs/adr/005adr-json.html),
[library publication changes](https://github.com/apalache-mc/apalache/blob/v0.62.3/CHANGES.md).

## Semantic work shared by all external routes

The hard part is preserving the question. These obligations come from
[IR semantics](../model/SEMANTICS.md) and the
[existing compatibility plan](UMPIRE4_TLA_COMPAT.md#semantic-contract).

| Umpire concept | Required external encoding or check |
| --- | --- |
| Atomic alternatives | Preserve every result and read all inputs from the old state; an empty result list remains disabled |
| Facts and repeated Steps | Preserve observable emissions and ordered result identity; two equal consecutive Steps must remain two occurrences |
| Stuttering | Artificial stutters emit no Fact, update no monitor and consume no modeled step/deadline; real unchanged-state Steps retain their effects |
| Deadlock and ends | Check absence of real enabled actions against declared ends; global stuttering must not hide a stuck nonterminal state |
| Properties and monitors | Preserve trigger, before/after values, activation and evaluation point; monitor state belongs in the explored product |
| Query | Preserve its Scenario, selected start, schedule, ending and Limits; a `find` must exercise its Property |
| Refinement | Preserve mapped starts, carried outcomes/Facts, visibility and legal stutters; generic temporal implication is not a substitute |
| Composition | Preserve all start combinations, interleaving and synchronized result products; synchronization counts as one Step |
| Channels | Preserve FIFO or multiset order, multiplicity, capacity, loss and atomic redelivery; a set of messages loses duplicates |
| Holes and errors | Initially refuse unsupported cases; never turn unknown behavior or an out-of-domain successor into a disabled action |
| Arithmetic | Preserve current `int64` meaning or prove no intermediate overflow; mathematical integers silently change behavior otherwise |
| Fairness and bounded progress | Match action-class granularity and composed enabledness; keep deadline, deadlock and fair-cycle outcomes separate |
| Resource limits | Timeout, memory exhaustion, unsupported input and backend failure remain incomplete checks |

Weak fairness permits an arbitrarily long finite scheduling delay. It cannot establish delivery
within a fixed number of Steps. Export bounded progress using explicit finite obligation state or an
equivalent bounded encoding, and keep fair-cycle checking separate. A destination reached on the
last permitted Step is on time; a source that is already a destination discharges immediately.
[Compatibility boundary cases](UMPIRE4_TLA_COMPAT.md#bounded-progress-and-fairness).

Likewise, the IR's `search` Limit counts native work/product states. Apalache's `--length`, TLC's
queue and an operating-system timeout are different quantities. They need explicit receipt fields;
no flag substitution makes these budgets equivalent. A backend can supplement a native Query with
a separately scoped check without claiming exact Query agreement.

## Architecture and decision experiment

Keep a deep export module whose public responsibility is to admit a requested backend profile,
produce a specification plus Definition/source bindings, run the checker, and decode its result.
It should hide generated names, finite encodings, process lifecycle, checker flags and trace formats.
Use the existing `export`, `interp` and `check` boundaries; do not introduce another general-purpose
semantic IR simply to support several printers.

Build the transition encoding from admitted IR expressions, rather than exporting a Go-computed
transition table as the only oracle. Table export can test an external search algorithm, but cannot
independently test how the interpreter read those expressions. For larger symbolic models, export
admission must also avoid constructing every native table first. The current
[Slice.OpenWithin](../tools/umpire/export/slice.go) constructs the native Realizer, so merely
changing the output language does not bypass native table costs.

Keep Go's deterministic witness selection for generated Cases. An external counterexample is
candidate evidence; decode it and replay every Step through a fresh native interpretation before
lowering. Solver choice and parallel search can change the witness they return, even when their
verdict is unchanged. Replayed witnesses establish validity, not exhaustive absence of other bugs.

Run a bounded comparison before committing to broader backend development:

1. Pin identical IR inputs and support profiles. Include a passing and faulty activity model,
   Nexus close, nondeterministic alternatives, repeated Facts, a channel, synchronized composition,
   class-scoped fairness, bounded progress, a hole and an arithmetic boundary. Use small synthetic
   controls as well as actual feature Models.
2. Extend the Quint source path for the selected profile and build a minimal direct typed TLA+
   path for the same profile. Compare both against the Go evaluator on complete small relations;
   do not give either exporter Go's precomputed answers as its specification.
3. Check safety with the same Apalache version and execution bound. Check finite temporal behavior
   with the same TLC version and explicit fairness. Retain intermediate artifacts and replay every
   reported counterexample, including its loop when relevant.
4. Measure generation, flattening, checker time, peak RSS and scratch use separately. Include the
   current failing Nexus shape and a larger parameter instance; report unsupported cases and
   failed checks alongside successes.
5. Compare maintenance burden as well as time. Count Umpire-specific mappings, unsupported
   constructs, source-mapping work and trace-adapter work. Determine whether the extra Quint
   stages contribute independent checking or primarily transport the same representation.

| Evidence from the comparison | Decision it supports |
| --- | --- |
| Quint checks the required profile, gives useful independent diagnostics, and fits resource budgets | Continue option 2 |
| Direct TLA+ avoids a demonstrated compiler/encoding problem and needs less maintained adapter code | Prefer option 3 for external checking; retain independent agreement controls |
| Text parsing dominates cost while typed JSON produces equivalent artifacts and results | Investigate option 2a with explicit version compatibility |
| Native value/storage work dominates and external pipelines do not address it | Repair the Go representation/lifetimes; assess a Rust kernel only after profiling |
| A required claim fits neither external engine's supported semantics | Specify that claim and prototype the smallest native extension |

The existing resource investigations retain their own scheduling and preservation requirements.
This research does not activate their repairs, certify their gates or justify shrinking their
Models and assertion populations.

## Effort estimates for the exporter routes

Added 2026-10-10. **Budget approximately 5-10 person-weeks to extend the existing Quint source
path into a scoped dual-backend integration, versus 7-12 person-weeks for direct TLA+.** These are
planning estimates based on the inspected implementation and missing capabilities, not measured
delivery rates. They assume one experienced full-time engineer, five working days per week, with
review and differential testing included. Staff unfamiliar with the IR or TLA+ need additional
learning time. Two engineers would not halve elapsed time because the semantic encoding and its
qualification have sequential dependencies.

### One common TLA+ emitter, two checker adapters

TLC and Apalache both consume TLA+. The direct design should share expression/value lowering and
the atomic transition relation, then produce separate checker configurations and any required
wrappers. Apalache adds type annotations and its supported-subset constraints; TLC adds its own
finite-state, fairness and deadlock configuration. Each needs result and trace decoding. The
shared emitter need not generate identical wrapper text for both tools.
[Apalache assignment requirements](https://apalache-mc.org/docs/apalache/principles/assignments.html),
[supported subset](https://apalache-mc.org/docs/apalache/features.html).

Similarly, a single Quint emitter does not eliminate backend-specific integration. Quint supplies
much of the language compilation, but its current TLC wrapper discards structured trace information
and disables built-in deadlock checking. Umpire's replay and failure classification still need
qualified adapters, potentially running TLC directly on Quint-generated TLA+.
[Quint TLC wrapper](https://github.com/quint-co/quint/blob/v0.33.0/quint/src/tlc.ts).

### Remaining effort from the current checkout

| Deliverable | Extend existing Quint source export | Shared direct TLA+ export for TLC and Apalache |
| --- | --- | --- |
| Comparable demonstration on one small machine, including a faulty control, actual checking by both tools and replay | 3-7 working days | 8-15 working days |
| Usable integration for the explicitly scoped core below | 5-10 person-weeks total | 7-12 person-weeks total |
| Broad parity with every supported native IR declaration and Query | Estimate after the core experiment; several additional work packages | Same; a second printer does not supply semantic parity |

The demonstration is included in the integration estimate. These columns measure incremental work
from today's repository. A greenfield Quint exporter would lose the reuse advantage. Running the
existing narrow Apalache monitor checks is already implemented; this estimate concerns new coverage
and TLC integration.

The usable core includes finite records and tagged enums, bounded integers, finite-list values,
supported pure expressions, all starts and nondeterministic alternatives, same-step and transition
Properties, single-machine monitors with their declared evaluation points, and currently supported
unmonitored compositions. It includes a selected profile of pinned/free `verify` and same-step
`find` Queries, preserving schedule, step bounds, completion, activation and monitor verdicts.
It also includes located refusals, source/Definition bindings, backend receipts, process cancellation,
decoded counterexamples and fresh native replay.

Backend resource budgets remain separately reported. This scope does not promise that SMT work or
TLC exploration counts reproduce the native Query's `search` accounting. Exact work-budget parity
needs a separate contract. Channels, progress/fairness, external refinement, monitored compositions,
arbitrary closures, hole-aware checking and table-free export admission are excluded from the core
and must remain explicit unsupported cases.

### Bottom-up budget for the usable core

| Work package | Quint path, person-days | Direct TLA+ path, person-days |
| --- | ---: | ---: |
| Specify support profile, semantic mappings and acceptance controls | 3-5 | 3-5 |
| Extend or implement value, type and pure-expression emission | 4-8 | 10-18 |
| Finite transition/monitor/composition wrappers and selected Query encodings | 8-15 | 8-15 |
| Checker lifecycle, configurations, trace decoding and replay | 5-10 | 5-10 |
| Differential tests, negative controls, version qualification and documentation | 5-10 | 8-12 |
| **Total** | **25-48** | **34-60** |

These totals explain the rounded 5-10 and 7-12 person-week estimates. Differential fixture design
appears in the first row; implementing and running the checks appears in the final row. The central
estimate gives Quint about a two-week advantage. The ranges overlap, and observed compiler problems
could reverse that advantage. Treat the estimate as a scope budget, not a precise completion date.

Quint's main savings come from current source translation and its static checks. At this checkout,
[quint.go](../tools/umpire/export/quint.go) has 1,069 lines and
[composed.go](../tools/umpire/export/composed.go) has 396. They already translate types, functions,
ordered matches, record copies, alternatives and composition declarations. These line counts show
existing responsibilities; they are not a conversion from lines of code to implementation time.
The receipt comparison, ITF reader and replay infrastructure are also reusable, though their
representation-specific portions need adaptation.

### A concrete TLC prerequisite changes the estimate

The current [stateful Quint wrapper](../tools/umpire/export/quint.go) declares `hist` as a list and
appends the selected class and Step after every transition. The
[current verifier](../tools/umpire/export/verify.go) reads that history to reconstruct its bounded
Apalache counterexample. Even a finite machine with a self-loop therefore creates infinitely many
distinct checker states as its history grows. Changing only the backend flag would not provide a
finite exhaustive TLC model.

Both routes need a checking representation with finite observer state. Retain the last selected
class/result and any finite occurrence marker needed for real unchanged-state Steps, then reconstruct
the execution from the checker's trace. Avoid making the accumulated history a searched state
variable. Qualify this representation against activation, monitor updates, stuttering and replay.
The transition-wrapper and trace rows above include this work.

Apalache 0.62.1 already pins `tla2tools` 1.8.0, whose TLC supports `-dumpTrace json`. Its JSON
envelope contains a counterexample state/action graph and variable names, rather than Apalache's
ITF format. A dedicated decoder can normalize that graph into Umpire's replay representation,
including its loop-closing edge. Pin and test the actual safety/lasso output; there is no need to
assume console parsing or an upstream trace feature must be built first.
[Apalache dependency](https://github.com/apalache-mc/apalache/blob/v0.62.1/project/Dependencies.scala#L45),
[TLC 1.8.0 JSON export](https://github.com/tlaplus/tlaplus/blob/v1.8.0/tlatools/org.lamport.tlatools/src/tlc2/TLC.java#L766),
[trace envelope](https://github.com/tlaplus/tlaplus/blob/v1.8.0/tlatools/org.lamport.tlatools/src/tla2sany/StandardModules/_JsonTrace.tla).

The existing composition export is also primarily an agreement dump. `QuintExport.Check` accepts a
machine with monitors; it does not turn every exported composition and Property into a checker
entrypoint. The core budget includes that missing integration rather than counting dump generation
as completed composed verification.
[Existing check entrypoint](../tools/umpire/export/quint.go),
[export coverage](../tools/umpire/export/README.md#what-is-compared).

### Additional scope and uncertainty

These are rough increments after the usable core, largely shared by both routes. They are not
independent commitments and should be re-estimated after the first implementation.

| Extension | Indicative additional effort | Work that determines the cost |
| --- | --- | --- |
| Bounded channels and derived delivery/loss | 1-3 person-weeks | Ordering, multiplicity, redelivery and receiver-disabled behavior |
| Bounded progress plus TLC fairness and lasso replay | 2-4 person-weeks | Deadline obligations, action-class enabledness, deadlock/cycle distinctions |
| Independent external checking of Umpire refinement | 2-4 person-weeks | Carrying outcomes/Facts, visibility, mapped starts and legal stutters |
| Export without first constructing native tables | 2-5 person-weeks | Separate admission/export lifetimes and independently validated semantics |

The final row covers a bounded redesign and qualification, not a guarantee that every large Model
will fit. Correlated or overlapping obligations, monitored compositions, arbitrary closures and
hole-aware partial results require separate estimates. Keep refinement in the existing native
checker if independent external refinement is not an immediate requirement.

The recorded Nexus inliner failure could add an encoding investigation to either route. Native OOM
repair and streaming the exhaustive Quint agreement dump remain the separately tracked fn-157 and
fn-154 work. Neither is included as an implicitly solved problem in these estimates.

**Decision implication.** Quint appears cheaper incrementally by weeks, rather than by an order of
magnitude. Most effort belongs to Umpire semantics, finite-state wrappers, evidence and qualification,
which both routes need. Spend a 5-10 person-day total investigation budget on matched small prototypes,
prioritizing the finite history representation, shared TLA+ value encoding and trace replay. This is a
time-boxed decision experiment, not a promise to finish both demonstrations within that budget.
Continue Quint if it preserves the required profile with less adapter work; choose direct TLA+ if
compiler or wrapper constraints consume its estimated savings.

## Toolchain facts to pin

The repository pins Quint **0.33.0** and Apalache **0.62.1**. As checked on 2026-10-10,
Quint's latest release is still 0.33.0, while Apalache's latest release is **0.62.3**.
The latter includes printer and large-expression fixes; it has not been demonstrated here to fix
Umpire's Nexus inlining failure.
[Repository tool pins](../tools/umpire/export/README.md#tools),
[Quint release](https://github.com/quint-co/quint/releases/tag/v0.33.0),
[Apalache release](https://github.com/apalache-mc/apalache/releases/tag/v0.62.3).

Apalache 0.62.x requires Java **21 or newer**, despite older Quint documentation mentioning
Java 17. Its 0.62.1 release added direct Quint JSON loading, and its 0.59.0 release published the
`tla-ir` and `tla-io` Scala libraries. Qualify a complete toolchain, rather than assuming versions
from different documentation pages fit together.
[Apalache version history](https://github.com/apalache-mc/apalache/blob/v0.62.3/CHANGES.md).

This report used repository plans and source plus first-party documentation, tagged source and
release metadata. It did not run a backend benchmark or the heavyweight verification gates.
The comparison above is proposed validation work, not a measured performance ranking.
