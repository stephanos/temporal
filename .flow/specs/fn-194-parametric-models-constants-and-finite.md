## Conversation Evidence

> user (2026-10-17): "woud;nt' those disappear as we don't need them in the model anymore?" (on `Addr.king, Denom.banana` in a test)
> user (2026-10-17): "a parametric Model seems ineventiable and useful?"
> user (2026-10-17): "let's write a new flow spec for the parametric model first; and then a nother one for our ModelSuite"
> assistant proposal, confirmed by the owner: a Model is a class whose constructor parameters are its constants and finite domains; the test suite instantiates it; a parameterless Model stays an object. Quint's `const` plus `instance`.

## Goal & Context

<!-- Goal & Context: 40% [user], 60% [paraphrase] -->

Three of the six framework samples drifted from their Quint originals because a Model cannot take a
parameter. TwoPhaseCommit fixes three resource managers where Quint has `const resourceManagers`,
Voting fixes its acceptors and quorums where Quint has `const Acceptor` and `const Quorum`, and Bank
hard-codes four addresses that Quint's test module declares. The owner's split of 2026-10-17 is that
the Model states its promises and the test file asks the questions; the concrete world a question is
asked in is part of the question. So the constants and the finite domains a Model is generic over are
constructor parameters, and the suite that tests the Model supplies them.

This spec is DSL only (Phase 2, DSL completion, owner 2026-10-17): the class form, the `Finite`
constraint, the lint, and the three samples rewritten and passing on the Scala checker. Lifting an
instance for export and Case lowering is the IR phase and is bounded below.

## Architecture & Data Models

<!-- Architecture: 30% [user], 50% [paraphrase], 20% [inferred] -->

- **A Model is a class.** `class Model[Addr: Finite](holder: Addr, supply: Int) extends Machine[Outcome, Nothing]`.
  A Model with no parameters stays `object Model extends Machine[...]`, unchanged.
- **Two kinds of parameter.** A *domain* the Model is generic over is a type parameter bounded by
  `Finite`, the existing constraint the explorer enumerates; the type is declared where the Model is
  instantiated, typically an enum in the test file. A *constant* is a value parameter of a liftable
  type: a `Finite` type, `Int`, `Boolean`, or a `Set` or `Map` of those. A domain the Model owns,
  such as Bank's `Denom`, stays an enum inside the Model.
- **Instantiation.** `Bank.Model(holder = Addr.king, supply = 2)` in a suite's header
  (`class BankTest extends ModelSuite(Bank.Model(...))`, fn-195) or anywhere a Machine is
  taken. Each instance has its own vars, starts, reachable graph and answers; two instances of one
  Model are two Machines.
- **Naming.** An instance is named by the suite or `val` that holds it; results and Case names use
  that name. An instance held by neither is refused when a check asks its name.
- **Vars and starts.** Var initialisers and `oneOf` start domains may read the parameters
  (`var balances = Balances(Map(holder -> Map(Denom.banana -> supply, ...)))`).
- **Lint.** The structure lint that admits `object M extends Machine[...]` also admits
  `class M(...) extends Machine[...]` with the same section rules. A Model class with a type
  parameter not bounded by `Finite`, or a value parameter outside the liftable kinds, is refused at
  the parameter.
- **Realizations** bind to an instance, as they bind to an object today.
- **Later phase (bounded).** The lifter emits an instance as a Model plus its parameter values; Case
  identity includes them.

## API Contracts

```scala
// Model: parameters are the constants and the domains it is generic over
object Bank:
  enum Denom:
    case banana, burger
  class Model[Addr: Finite](holder: Addr, supply: Int) extends Machine[Outcome, Nothing]:
    var balances = Balances(of = Map(holder -> Map(Denom.banana -> supply, Denom.burger -> supply)))
    ...

// Instantiation: the concrete world, declared where the questions are asked
enum Addr:
  case king, donkeykong, peaches, mario
val smallBank = Bank.Model(holder = Addr.king, supply = 2)

// Voting: the quorum structure is a parameter the Model cannot know
Voting.Model(values = Set(1, 2), quorums = Set(Set(a1, a2), Set(a2, a3), Set(a1, a3)))
```

Errors, each a compile error or a located lint refusal naming the parameter: a type parameter with no
`Finite` bound; a value parameter whose type is not a liftable kind; a `Set` or `Map` parameter whose
element type is not `Finite`; an empty domain supplied to a parameter the Model enumerates
(`require` at construction, the same text as `oneOf`'s empty-domain refusal); a Model class with
sections outside its body (the existing structure refusal, extended to the class form).

## Edge Cases & Constraints

- A parameter is a constant: nothing in a Model assigns it, and the lifter later treats it as a
  literal. Reassigning one is the ordinary Scala error.
- Two instances of one Model in one suite need distinct names; the second unnamed one is refused.
- A parameterless Model keeps the object form so the six samples that need no parameter do not
  change shape; only Bank, Voting and TwoPhaseCommit become classes in this spec.
- The `VarSchema` given and the var runtime are per instance; nothing is shared through the class.

## Acceptance Criteria

- **R1:** A Model may be `class M[D: Finite, ...](constants...) extends Machine[O, F]`, with sections
  as in an object, var initialisers and `oneOf` starts reading its parameters; `object M extends
  Machine[O, F]` is unchanged. Errors: a type parameter without a `Finite` bound and a value
  parameter outside the liftable kinds are compile errors naming the parameter. [paraphrase]
- **R2:** Each instance has its own vars, starts and reachable graph; two instances of one Model
  checked in one process do not share state. Errors: no error surface beyond R1. [inferred]
- **R3:** An instance is named by the `val` or suite that holds it, and that name appears in check
  results. Errors: a check asked of an instance with no holder is refused with a message naming the
  Model. [inferred]
- **R4:** The structure lint admits the class form with the same section rules it applies to the
  object form. Errors: a Model class with declarations outside its sections is refused with the
  existing text, which names the class form beside the object form. [paraphrase]
- **R5:** Bank takes its address type and holder and supply as parameters, Voting its values and
  quorums, TwoPhaseCommit its resource-manager set; each sample's test file declares the concrete
  types and instantiates the Model; the native suites pass with their intended answers unchanged
  for the instances that reproduce today's constants. Errors: a changed answer is a finding to fix,
  not a regeneration. [paraphrase]
- **R6:** Realizations bind to an instance the way they bind to an object today; the realization
  fixtures compile against an instantiated Model. Errors: a realization naming a Model class rather
  than an instance is a compile error. [inferred]
- **R7:** SEMANTICS and the framework README describe parameters, the two kinds, instantiation and
  naming, with Voting as the worked example. Errors: none beyond review. [inferred]

## Boundaries

- Lifting an instance (parameter values in the IR, Case identity including them) and the Go or
  Scala lowering of it are the IR phase, not this spec. [paraphrase]
- No implicit instantiation through `given`s, no default parameter values standing for "the
  sample's world": the suite always writes the instance. [inferred]
- The Temporal Models stay out of the tree; their parameters (worker count, attempt limits) are
  decided when they return. [inferred]
- Realization environment bindings (namespace, task queue) are not Model parameters; they stay in
  the realization. [paraphrase]

## Decision Context

The owner asked whether the concrete addresses would disappear from the Model once the scenarios
moved to the test file, and then whether a parametric Model is inevitable. The conductor first
argued for keeping the enum as the Model's domain; the owner's push and the drift in three samples
decided it. [user] A domain the Model is generic over is a type parameter with `Finite`, because the
explorer needs a finite enumeration and `Finite` is the constraint it already has; a constant is a
value parameter because Scala already checks it. [paraphrase] A class rather than an abstract
`Params` trait or a `given` world: a constructor is the plainest Scala, and the owner has preferred
plain Scala over DSL vocabulary at every step of fn-190. [paraphrase] The object form stays for
parameterless Models so nothing that works today changes. [inferred] Rejected: keeping the enums in
the Model and parametrising only Voting (leaves Bank and TwoPhaseCommit drifted); a `Params` case
class per Model (one more declaration for no check). [inferred]

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | A Model may be `class M[D: Finite, ...](constants...) extends Machine[O, F]`, with sections as in an object, var initialisers and `oneOf` starts reading its parameters; `object M extends Machine[O, F]` is unchanged. Errors: a type parameter without a `Finite` bound and a value parameter outside the liftable kinds are compile errors naming the parameter. | fn-194.1, fn-194.2 | — |
| R2 | Each instance has its own vars, starts and reachable graph; two instances of one Model checked in one process do not share state. Errors: no error surface beyond R1. | fn-194.1, fn-194.2 | — |
| R3 | An instance is named by the `val` or suite that holds it, and that name appears in check results. Errors: a check asked of an instance with no holder is refused with a message naming the Model. | fn-194.1, fn-194.2 | — |
| R4 | The structure lint admits the class form with the same section rules it applies to the object form. Errors: a Model class with declarations outside its sections is refused with the existing text, which names the class form beside the object form. | fn-194.2 | — |
| R5 | Bank takes its address type and holder and supply as parameters, Voting its values and quorums, TwoPhaseCommit its resource-manager set; each sample's test file declares the concrete types and instantiates the Model; the native suites pass with their intended answers unchanged for the instances that reproduce today's constants. Errors: a changed answer is a finding to fix, not a regeneration. | fn-194.3 | — |
| R6 | Realizations bind to an instance the way they bind to an object today; the realization fixtures compile against an instantiated Model. Errors: a realization naming a Model class rather than an instance is a compile error. | fn-194.3 | — |
| R7 | SEMANTICS and the framework README describe parameters, the two kinds, instantiation and naming, with Voting as the worked example. Errors: none beyond review. | fn-194.4 | — |
