# Generating Go from Lean: landscape and a path for Umpire

_Research snapshot: 2026-08-28. Sources are official language documentation and source code, project-owned repositories, and papers from the original authors. “Not found” below means not found in the reviewed public sources; it is not a claim that no private experiment exists._

## Conclusion

Generating Go **from a model written in Lean** is feasible today. Generating Go from **arbitrary Lean programs** is a substantially different compiler project, and I found no maintained general Lean-to-Go backend.

The strongest path for Umpire is therefore not to modify Lean's compiler into a Go compiler. It is to define a deliberately small, effect-aware domain language in Lean, prove the domain transitions and lowering, generate a small typed Go AST, and keep Temporal-specific persistence, locking, protobuf, metrics, and RPC integration in hand-written Go adapters. Workflow Reset is a good first domain if the generated boundary is the reset **decision kernel/plan**, not the entire current orchestration function. Replication is an even stronger eventual application for state-machine modeling, but a much larger first target because ordering, retry, duplicate delivery, conflict resolution, and partial failure all need explicit semantics.

There is credible adjacent work:

- Lean itself has an optimizing, extensible compiler written mostly in Lean, with C and LLVM backends, but no Go backend in its built-in backend vocabulary.
- Fugue is a direct, public Lean-to-Go research prototype for Distributed PlusCal. It uses a small Go AST and printer and is unusually close to the Replication use case, but its proved result currently stops before the final Go translation.
- Trust-Lean demonstrates the exact “model a target subset as an AST, translate, pretty-print, and prove properties” pattern for small C and Rust subsets in Lean. It is useful precedent, not a production general-purpose extractor.
- Isabelle/HOL has a real standalone Go code-generation target. It is the closest direct precedent for mapping a proof assistant's functional language to Go.
- Fiat-Crypto generates Go from Coq for a narrow cryptographic domain and is unusually candid that the Go stringification backend itself is not proved.
- Axon demonstrates proof-producing/translation-validating compiler passes in Lean, but for a small language targeting ARM64 rather than Lean or Go.

The important distinction is:

| Claim | What establishes it |
|---|---|
| The output is syntactically valid Go | A typed Go AST, printer/parser round-trip, `go/parser`, `gofmt`, and `go test`/`go build` |
| The Go-looking AST means the same thing as the Lean domain model | A formal semantics for the source and target subsets plus a lowering simulation/refinement theorem |
| Actual Go execution preserves that target semantics | A trusted Go compiler/runtime assumption, or a formal Go-subset semantics connected to the emitted program; ordinary compilation tests alone do not prove this |
| Arbitrary executable Lean can become equivalent Go | A general compiler that handles Lean's erasure, closures, inductives, polymorphism, partial application, runtime primitives, and effects; no such maintained Go backend was found |

## What Lean provides now

As of this snapshot, Lean 4.33.1 is the latest stable release listed by the official reference; 4.34.0 is still a release candidate ([release index](https://lean-lang.org/doc/reference/latest/releases/)). Lean's documented pipeline parses into `Syntax`, elaborates into kernel terms, kernel-checks those terms, then compiles executable definitions. The normal module build emits one C file per Lean module and compiles it to native code ([Elaboration and Compilation](https://lean-lang.org/doc/reference/latest/Elaboration-and-Compilation/)). Lake also exposes LLVM bitcode facets ([Lake reference](https://lean-lang.org/doc/reference/latest/Build-Tools-and-Distribution/Lake/)).

The current compiler architecture is useful raw material:

- The new LCNF pipeline was completed end-to-end in Lean 4.30 when C emission was ported to LCNF ([4.30 compiler notes](https://lean-lang.org/doc/reference/latest/releases/v4.30.0/)). The compiler lowers elaborated declarations through base, monomorphic, and impure LCNF phases and then into the lower IR ([`LCNF/Main.lean`](https://github.com/leanprover/lean4/blob/v4.33.1/src/Lean/Compiler/LCNF/Main.lean)).
- Compiler passes are represented as Lean values and can be installed, removed, or replaced with the global `cpass` attribute. Installers can add passes before/after named passes or at a phase's end ([`LCNF/PassManager.lean`](https://github.com/leanprover/lean4/blob/v4.33.1/src/Lean/Compiler/LCNF/PassManager.lean), [`LCNF/Passes.lean`](https://github.com/leanprover/lean4/blob/v4.33.1/src/Lean/Compiler/LCNF/Passes.lean)).
- Final impure declarations and imported signatures are stored in environment extensions. The C emitter collects those declarations and renders them ([`LCNF/PhaseExt.lean`](https://github.com/leanprover/lean4/blob/v4.33.1/src/Lean/Compiler/LCNF/PhaseExt.lean), [`LCNF/EmitC.lean`](https://github.com/leanprover/lean4/blob/v4.33.1/src/Lean/Compiler/LCNF/EmitC.lean)).
- `leanir` is a separate code-generation process that loads serialized module/compiler state, resumes postponed compilation, and invokes the C emitter. It is a particularly relevant architectural template for an experimental external emitter ([`LeanIR.lean`](https://github.com/leanprover/lean4/blob/v4.33.1/src/LeanIR.lean)).
- The built-in Lake backend type has exactly `c`, `llvm`, and `default`; Go is not a built-in target ([`Lake/Config/LeanConfig.lean`](https://github.com/leanprover/lean4/blob/v4.33.1/src/lake/Lake/Config/LeanConfig.lean)). The LLVM emitter lowers Lean's low IR via LLVM bindings to bitcode ([`IR/EmitLLVM.lean`](https://github.com/leanprover/lean4/blob/v4.33.1/src/Lean/Compiler/IR/EmitLLVM.lean)).

This makes a prototype technically possible in two ways:

1. A custom Lean/Lake generator can operate directly on a domain AST and emit `.go` files. This uses only ordinary Lean programming and a custom Lake target.
2. An experimental compiler plugin/process can observe LCNF declarations and emit Go, following `leanir` and the pass-extension mechanism.

The first is much safer. Final impure LCNF is already committed to Lean's runtime model: explicit boxing/unboxing, constructor allocation and projection, partial/full application, reset/reuse, reference-count operations, borrowed arguments, module initialization, and foreign calls are visible in the IR and C emitter. Translating all of that to idiomatic Go would amount to implementing a new Lean runtime/backend, not merely writing a pretty-printer. Intercepting an earlier pure/monomorphic phase avoids some runtime operations but still requires stable handling of closures, algebraic data types, polymorphism/monomorphization, and Lean primitives.

The FFI route is a third option rather than extraction: Lean can export C-ABI functions and interoperate with any language that can call the C ABI. The official manual warns that this FFI was designed for internal use and is unstable, and documents the Lean object representation, ownership rules, and module/thread initialization obligations ([Lean FFI reference](https://lean-lang.org/doc/reference/latest/Run-Time-Code/Foreign-Function-Interface/)). Go could call a generated Lean C library through cgo, but that ships the Lean runtime and does not produce native, maintainable Go logic. It is useful for differential or oracle testing, less attractive for replacing a core Temporal subsystem.

### Trust boundary

Kernel-checking a theorem about a Lean function does not make Lean's generated machine code part of the kernel proof. The official reference separates kernel checking from compilation, and the Lean FAQ explains that executing compiled code extends the trusted computing base to the compiler, runtime, backend, and linked external code ([Lean FAQ](https://lean-lang.org/faq/)). Lean FRO's current compiler roadmap likewise describes its compiler results in relation to the “unverified state of the art” and plans further deep code-generator work ([Lean FRO Year 4 Part 1 roadmap](https://lean-lang.org/fro/roadmap/y4-1/)).

For Umpire, the assurance statement should always name the exact boundary: “Lean proved the domain lowering,” “the generated Go passed parser/type/build tests,” or “the actual Go program refines the model.” Those are not interchangeable.

## Relevant projects

### No maintained general Lean-to-Go backend found

The official Lean backend and Lake sources expose C and LLVM only. Searches of public Lean packages, GitHub projects, papers, and code-generation discussions found no maintained project that accepts general Lean 4 definitions and emits equivalent Go packages. There may be unpublished or private projects, and a small repository may have escaped indexing, so this should be read as the current public landscape rather than a proof of absence.

The likely explanation is technical fit. Lean's code generator is designed around functional programs, closures, algebraic data, erasure, reference counting, and Lean runtime primitives. Go supplies garbage collection, closures, interfaces, generics, and goroutines, but it does not directly supply Lean's runtime representation or proof-relevant compilation invariants. A backend must either:

- reproduce Lean runtime behavior in Go, yielding non-idiomatic generated code and a large compatibility surface;
- translate only a restricted Lean subset; or
- compile a purpose-built domain IR whose semantics and Go mapping are controlled by the project.

The third choice matches Workflow Reset and Replication best.

### Fugue: a direct domain-specific Lean-to-Go compiler

[Fugue](https://github.com/Mesabloo/fugue) is the most directly relevant project found. Bergeron, Cirstea, and Merz are building a compiler from Distributed PlusCal to native Go, with the language syntax, semantics, and compiler passes implemented in Lean. Their published work proves correctness of the first lowering, from Guarded PlusCal to Network PlusCal; the formal Go semantics and correctness of the final Network PlusCal-to-Go translation remain future work ([authors' project/publication page](https://g-bergeron.github.io/), [WPTE 2025 paper](https://wpte2025.github.io/pre-proceedings.pdf)).

The public repository follows the architecture proposed here:

- [`GoCal/Syntax.lean`](https://github.com/Mesabloo/fugue/blob/main/PlusCalCompiler/GoCal/Syntax.lean) defines a deliberately small Go-oriented target AST/IR.
- [`GoCal/Pretty.lean`](https://github.com/Mesabloo/fugue/blob/main/PlusCalCompiler/GoCal/Pretty.lean) renders that AST.
- [`NetworkToGoCal/PlusCal.lean`](https://github.com/Mesabloo/fugue/blob/main/PlusCalCompiler/Passes/NetworkToGoCal/PlusCal.lean) lowers the network-level model into GoCal.
- [`Main.lean`](https://github.com/Mesabloo/fugue/blob/main/Main.lean) wires the pipeline into an executable compiler.

It is explicitly a work in progress, not a production backend. At this snapshot the repository has a very short history, no releases, an unfinished tuple-printing proof (`sorry`), and an update-expression path that still emits a TODO marker ([commit history](https://github.com/Mesabloo/fugue/commits/main/), [`GoCal/Pretty.lean`](https://github.com/Mesabloo/fugue/blob/main/PlusCalCompiler/GoCal/Pretty.lean)). Even with that qualification, Fugue answers the core “is anyone doing this?” question affirmatively and validates a narrow `Lean semantics → distributed-algorithm IR → Go AST/source` strategy. Its subject matter makes it more architecturally relevant to Temporal replication than a numeric or cryptographic generator.

### Trust-Lean: the closest public Lean AST/emitter pattern

[Trust-Lean](https://github.com/lambdaclass/trust-lean) is a small, new Lean project for verified DSL compilation through a shared imperative IR to C and Rust. Its history is short (the inspected repository has 16 commits from February through June 2026), so it should be treated as research/prototype evidence rather than a mature dependency.

Its architecture is directly relevant:

- Source DSLs implement lowering and a `CodeGenSound` obligation into an imperative `Stmt` IR ([`CodeGenSound.lean`](https://github.com/lambdaclass/trust-lean/blob/aaf67f6ce98690042318f90573c2b3077b88123b/TrustLean/Typeclass/CodeGenSound.lean)).
- The generic `Pipeline.sound` theorem establishes the source-to-core-IR result, but the file explicitly says backend emission is outside that theorem's trusted boundary ([`Pipeline.lean`](https://github.com/lambdaclass/trust-lean/blob/aaf67f6ce98690042318f90573c2b3077b88123b/TrustLean/Pipeline.lean)).
- Its later “MicroRust” work models a small Rust-like target AST (in fact shared with its MicroC AST), translates to it, defines a target evaluator, and proves a source/target simulation under explicit well-formedness, identifier-injectivity, and fuel hypotheses ([`MicroRust/Defs.lean`](https://github.com/lambdaclass/trust-lean/blob/aaf67f6ce98690042318f90573c2b3077b88123b/TrustLean/MicroRust/Defs.lean), [`MicroRust/Simulation.lean`](https://github.com/lambdaclass/trust-lean/blob/aaf67f6ce98690042318f90573c2b3077b88123b/TrustLean/MicroRust/Simulation.lean)).
- It also proves that its canonical printer followed by its own parser recovers the target AST, subject to well-formedness/disambiguation predicates ([`MicroRust/RoundtripMaster.lean`](https://github.com/lambdaclass/trust-lean/blob/aaf67f6ce98690042318f90573c2b3077b88123b/TrustLean/MicroRust/RoundtripMaster.lean)).

This may be close to the friend's Rust example. The key qualification is that the public project models a deliberately tiny Rust-shaped language: assignments, arrays, calls, sequencing, conditionals, loops, break/continue, and returns. Its semantics are a project-defined evaluator, not all of Rust and not a proof about `rustc`. That is still the right basic design: a deep target module with a small interface, explicit semantics, lowering theorem, and printer round-trip.

### Axon: proof-producing and translation-validating compiler passes in Lean

[Axon](https://github.com/rinard/Axon) is a verified compiler in Lean for a small `WhileLang → TAC → ARM64` pipeline. Its optimizers emit certificates checked by a proved checker, so optimization heuristics themselves need not be in the trusted core. Its project documentation names an end-to-end behavior theorem, an instruction encoder with encodability and decode/emit results, and explicit remaining floating-point and `native_decide` assumptions. The accompanying author paper describes the testing, credible-compilation, verification, and audit workflow ([Rinard 2026](https://arxiv.org/abs/2605.01660)).

For Umpire, the reusable idea is not the language or target. It is **translation validation**: let a generator or optimizer produce a candidate reset/replication plan or Go IR, then require a small verified checker to validate a certificate against the source semantics. This can be easier to evolve than proving every sophisticated optimization function correct.

### Lean4Lean: stronger checking, not extraction

[Lean4Lean](https://github.com/digama0/lean4lean) implements a Lean 4 kernel/external checker in Lean and proves the implementation against an abstract typing specification. It can recheck `.olean` declarations and has checked mathlib, but it is not a compiler backend and does not generate Go. It strengthens confidence in Lean proof checking and provides useful environment/replay machinery; it does not close the generated-code gap ([project paper](https://arxiv.org/abs/2403.14064)).

### Other Lean extraction/backend experiments

[Peregrine/lambda-box](https://peregrine-project.github.io/) is the main current effort to share a verified extraction middle-end across proof assistants. It has Lean, Agda, and Rocq frontends and CakeML, C, Rust, and OCaml backends; Go is not among its published targets, and only some frontends/backends share the middle-end's verification status. Dima's primary implementation report describes a Lean `Expr`-to-lambda-box erasure implemented in Lean and then routed through Rocq's extraction pipeline. The prototype handles much pure Lean core but omits important systems features such as `IO`, asynchronous tasks, and fixed-width integers, and the report identifies Lean compiler modularity as an obstacle ([Compiling Lean programs with Rocq's extraction pipeline](https://www.normalesup.org/~sdima/2025_extraction_report.pdf)). This is the strongest evidence that reuse of a language-neutral extraction IR is possible, and also evidence that Workflow Reset cannot simply be sent through an existing general extractor.

[Qed](https://github.com/JacobAsmuth/qed) contains a direct Lean IR-to-JavaScript backend used by a verified web UI framework ([`Js/Backend.lean`](https://github.com/JacobAsmuth/qed/blob/master/Js/Backend.lean)). It is useful engineering precedent for consuming Lean compiler IR, host bindings, and differential native/target tests, but it does not claim an end-to-end semantic-preservation proof for the transpiler. It shows that an alternative Lean backend can be built without changing the upstream compiler; it does not remove the runtime and proof obligations that a Go backend would inherit.

### Isabelle/HOL's Go target: the closest direct precedent

Stübinger and Hupel added Go as a fifth target to Isabelle/HOL's code generator. Their translation starts from Isabelle's shared functional IR, Thingol, and maps it through a shallow embedding into Go. The difficult areas are exactly the ones a general Lean backend would face: algebraic data types, pattern matching, higher-order functions, type classes, and the mismatch between a functional source and imperative Go. They package it as a standalone theory that can be imported into an Isabelle development ([authors' paper](https://lars.hupel.info/pub/go-codegen.pdf), [AFP entry source outline](https://isa-afp.org/browser_info/current/AFP/Go/outline.pdf)).

This establishes that proof-assistant-to-Go extraction is practical. It does not make Lean-to-Go a solved problem: Isabelle has a mature language-neutral code-generation IR and target framework, while Lean's current final IR is coupled to its runtime and optimization strategy.

### Fiat-Crypto: narrow verified generation with an honest Go-backend gap

[Fiat-Crypto](https://github.com/mit-plv/fiat-crypto) synthesizes finite-field arithmetic from Coq and includes C, Rust, Go, Java, JSON, and Zig stringification backends. It is strong evidence that narrow, proof-driven Go generation can be operationally useful. Its own backend status table is also a valuable warning: the internal compiler establishes bounds and transformations, but the Go backend has no proof that its target AST/stringification preserves the internal semantics; the generated Go is build-checked, not semantically proved. The repository explicitly says only its Bedrock2 internal AST backend has the relevant AST proof, and even its Bedrock2-to-C string conversion is unproved ([Fiat-Crypto README](https://github.com/mit-plv/fiat-crypto#status-of-backends), [`CLI.v`](https://github.com/mit-plv/fiat-crypto/blob/master/src/CLI.v)).

The lesson is to keep three gates separate in Umpire: lowering proof, emitter proof, and real Go conformance/build evidence.

## Modeling Go as a target

Go's standard library already gives an excellent external validation boundary. `go/ast` declares the syntax-tree types and explicitly supports constructing trees directly; `go/parser` parses source into those ASTs; `go/format` formats Go source; and `go/types` type-checks packages ([`go/ast`](https://pkg.go.dev/go/ast), [`go/parser`](https://pkg.go.dev/go/parser), [`go/format`](https://pkg.go.dev/go/format), [`go/types`](https://pkg.go.dev/go/types)). The language syntax and semantics are specified by the official Go specification ([Go specification](https://go.dev/ref/spec)).

There are two sensible emitter shapes:

1. **Lean-native target AST and printer.** Define only the Go subset that Umpire generation needs, with smart constructors that make malformed nodes unrepresentable. Prove identifier safety, declaration uniqueness, import completeness, control-flow well-formedness, and printer/parser round-trip. This keeps the structural generator in Lean and is the best path toward semantic theorems.
2. **Lean semantic IR → JSON → small Go renderer using `go/ast`.** This immediately gains the standard printer and type checker, and sharply limits hand-written string generation. The JSON bridge and renderer remain trusted/unproved unless modeled, but the operational path is simpler.

A hybrid is attractive: make Lean's `Go` module the canonical typed AST, serialize that AST to a schema-versioned format, and have a tiny Go program map it one-for-one into `go/ast` and call `format.Node`. Differential tests can parse the result back and compare a position-free canonical AST. The renderer is then a deep, independently testable module rather than formatting logic scattered through proofs.

The generated subset should initially avoid clever Go features. Prefer named structs, tagged unions encoded as a tag plus payload structs, explicit `switch`, deterministic slices, concrete integer widths, and plain functions. Avoid reflection, `unsafe`, implicit map-order behavior, goroutines, and interface-heavy sum types until their semantics are modeled.

## Application to Workflow Reset

The current reset path is not one pure function. The inspected production slices total roughly 1,944 lines across [`api/resetworkflow/api.go`](../../service/history/api/resetworkflow/api.go), [`ndc/resetter.go`](../../service/history/ndc/resetter.go), and [`ndc/workflow_resetter.go`](../../service/history/ndc/workflow_resetter.go), with more code in mutable-state, history, transaction, and persistence dependencies. They combine:

- namespace/request validation and rate limiting;
- leases and high-priority locking for base/current runs;
- request-id deduplication and UUID/time generation;
- version-history lookup and branch forking;
- mutable-state rebuilding;
- terminating or updating the current execution;
- reapplying eligible signals/updates and post-reset operations;
- generating history events and tasks;
- multi-workflow persistence transactions, cleanup, metrics, and logging.

Trying to generate this whole orchestration directly would require a semantics for Temporal persistence, locks, failures, clocks, UUIDs, protobufs, and side effects before the first useful result. The better deep-module boundary is:

```text
ResetSnapshot + ResetRequest + Policy
                 |
                 v
       decideReset : Except ResetError ResetPlan
                 |
                 v
       hand-written Go executor/adapters
                 |
                 v
     persistence + mutable state + history + tasks
```

`ResetSnapshot` should contain only the immutable semantic facts needed to decide: base/current identifiers and statuses, reset point, relevant version-history information, pending-child/activity facts, request dedup state, eligible reapplied events, and configured policy. `ResetPlan` should be an ordered, typed effect script: fork branch, rebuild through event/version, terminate current run if required, create reset run, reapply event, apply post-reset option, emit event/task, and commit atomically. The hand-written executor resolves each command against existing Temporal interfaces and owns retryable operational errors.

Useful first theorems include:

- accepted reset points are within the base history and identify an existing version-history item;
- the new run/branch begins at the specified cut and never mutates the base branch;
- deduplicated requests do not create a second reset run;
- only policy-eligible signals/updates are reapplied, in a defined order;
- the plan never both preserves and terminates the same current run;
- plan execution is idempotent under an explicit request/transaction identity assumption;
- every successful plan establishes a small postcondition over base/current/reset executions.

The executor itself can then be verified incrementally through a Go-side simulation harness: an in-memory interpreter for the same command algebra, differential traces against Lean's evaluator, and existing Temporal unit/integration tests. Full refinement of real persistence can come later.

## Application to Replication

Replication is a natural transition-system problem, but the local surface is much larger: the inspected Go files under `service/history/replication`, relevant `service/history/ndc`, and replication APIs total about 54,834 lines including tests. The domain includes transport, flow control, ack/progress tracking, task conversion, application, batching, retry/resend, DLQ, event importing, mutable-state conflict resolution, and several task kinds.

The promising generated kernels are narrower:

- **task admission/classification:** duplicate, stale, applicable, needs backfill, needs resend, or DLQ;
- **version-history reconciliation:** compute the common point and required local/remote history action;
- **per-workflow application plan:** transform a snapshot plus one replication task into a command script;
- **progress/ack transitions:** prove monotonicity and that acknowledged ranges correspond to completed policy states;
- **batching legality:** prove a batch preserves the sequential result under explicit commutativity/independence conditions.

The formal model must make delivery assumptions explicit: at-least-once delivery, reordering bounds, duplicate tasks, missing history, concurrent local mutation, failover versions, crash points, retry, and eventual availability. Good target properties are conditional rather than magical: duplicate application is observationally idempotent; ack levels are monotone; a task is never acknowledged before its required durable effects; conflicts select a history consistent with the version-history rule; and replicas converge when the same finite task/history set is eventually delivered under stated fairness assumptions.

Networking, goroutine scheduling, rate limiting, metrics, and persistence clients should remain adapters at first. Generate or verify the deterministic decisions those mechanisms act on.

## Recommended architecture

### 1. A small effect-aware semantic core

Create a project-owned Lean library with modules roughly like:

```text
Umpire.Reset.Model       -- immutable semantic DTOs
Umpire.Reset.Semantics   -- reference transition/effect interpreter
Umpire.Reset.Properties  -- invariants and refinement theorems
Umpire.Go.AST            -- only the required Go subset
Umpire.Go.WellFormed     -- names, types, imports, control-flow predicates
Umpire.Go.Emit           -- AST serialization/printing
Umpire.Go.Semantics      -- optional target-subset evaluator
Umpire.Reset.GoLower     -- model/IR to Go AST plus correctness theorem
```

This is a deep module: the changing details of Go formatting and declaration construction stay behind a small interface such as `lowerResetKernel : ResetModule → Except GenError Go.Package` and `emitPackage : Go.Package → ByteArray`.

### 2. Make effects values

Do not model Temporal APIs as arbitrary Lean `IO`. Define an effect algebra whose constructors name semantic operations and whose interpreter is parameterized. The Lean reference interpreter can use a pure world state; generated Go can return a plan or call a narrow interface. This makes partial failure and crash boundaries visible and testable.

For effects that cannot be made atomic, model the protocol state explicitly. A theorem about a single pure transition is not a theorem about a multi-write production transaction unless the transaction assumption is stated.

### 3. Generate owned files only

Generate self-contained internal packages/files with a clear header and stable public boundary. Keep protobuf conversion, existing interfaces, observability, and persistence in ordinary hand-written Go. Never regenerate mixed hand-written files. CI should fail on a dirty regeneration diff.

Every generation run should perform:

1. Lean build and axiom/sorry audit for named load-bearing theorems.
2. Deterministic generation from a pinned Lean toolchain and generator version.
3. Go parse/format round-trip.
4. `go/types` or ordinary package compilation.
5. Targeted Go unit tests and model/generated differential vectors.
6. Existing reset or replication tests for the integrated adapter.

### 4. State the certificate and TCB

Emit a machine-readable manifest binding source-model hash, generator hash/toolchain, theorem names, target-AST hash, formatted Go hash, and test commands. This is not itself a proof, but it prevents a proved artifact from being confused with a later edited file.

The initial trusted base will still include Lean's kernel, axioms used by the theorem, the serializer/Go renderer, Go parser/type checker/compiler/runtime, and hand-written adapters. A target-AST simulation theorem removes the lowering from that list; a printer round-trip theorem removes many syntax mistakes; neither proves the Go compiler.

## Suggested experiment

Do one vertical slice before committing to a Go backend:

1. Extract a pure reset decision with meaningful branching—for example, reset-point validation plus current-run/dedup/reapply classification—into immutable input/output types.
2. Implement the reference decision in Lean and prove two or three concrete invariants.
3. Define the minimum Go AST needed for that function and generate an internal Go package.
4. Compare the existing Go behavior, Lean evaluator, and generated Go across table tests and generated edge cases.
5. Integrate through a hand-written adapter without deleting the existing implementation; run both in shadow/differential mode where feasible.
6. Decide based on proof effort, generated-code reviewability, build ergonomics, and behavioral parity whether to expand toward a full `ResetPlan`.

Only after that should the project consider sharing the generator infrastructure with a replication kernel. A general LCNF-to-Go backend would be an interesting independent research project, but it is neither necessary nor the shortest path to verified Workflow Reset or Replication.

## Bottom line

The friend's target-AST approach is sound in shape and has close public analogues in Fugue and Trust-Lean. The missing piece is not printing Go: it is choosing and proving the semantic boundary. For Umpire, the highest-leverage boundary is a verified Lean decision kernel that generates a typed Go plan/reducer, surrounded by existing hand-written Temporal infrastructure. That can deliver useful assurance incrementally. Whole-program Lean-to-Go extraction, or generated end-to-end Temporal orchestration, would begin as compiler research rather than ordinary implementation work.
