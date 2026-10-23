# Shared front end: one Model reading for the native checker and the lifter (model)

## Conversation Evidence

> user (2026-10-21): "yes we need both checkers but can they share code?"
> assistant: not the part that assigns meaning, if independence is kept; the attribute/registration grammar could be shared.
> user (2026-10-21): "I'll accept the risk if we can share more code"
> user (2026-10-21): "the only diff should be that the native checker actas like an interpreter and the lifter genrates TLA+ (and later Apalache)"

## Goal & Context

fn-227 showed every DSL construct being implemented twice: in `umpire-dsl-macros` (native adapters, registration, domains, starts) and again in `umpire-lift` (source admission of the same attributes, types, patterns and domains before TLA+ emission). Roughly half of fn-227's growth was the second copy. The owner accepts the risk of a shared misreading in exchange for sharing code: both checkers read a Model through one front end, and differ only in their back end. The native checker executes the Model; the lifter emits TLA+ for TLC and, later, Apalache (fn-211).

With the Model about to expand dramatically and the goal of eventually meeting Quint's capabilities, per-construct cost dominates; one front end makes each new construct roughly half the work.

## Architecture & Data Models

- **One front end crate** reads `#[machine]` modules: attribute grammar (`#[on]`, `#[invariant]`, `#[postcondition]`, `#[assumption]`, `#[scope]`, `#[progress]`, `#[starts]`, `#[domain]`, `#[inputs]`, `derive(World/State/Action/Finite)`), registration, typed World/state/action schema, domains and admitted expression/statement subset, with located diagnostics. Proc macros and the lifter both depend on it.
- **Native back end:** the macros generate the executable Model from the front end's admitted structure; the native checker explores by executing it.
- **Emission back end:** the lifter emits TLA+ (and fn-211's Apalache projection) from the same admitted structure.
- **Open design question for .1:** whether the shared structure stops at admission (native still runs the compiled Rust bodies) or extends to a shared semantic IR that the native checker interprets. The owner's phrasing ("acts like an interpreter") points toward the latter; .1 measures both against Model ergonomics, wasm guest/Realization execution, performance and code size, and recommends one.
- Typed replay of every backend counterexample through the native Model stays: it remains the cross-check between back ends.

## Acceptance Criteria

- **R1:** A written design (in `model/design/`) compares a shared admission front end and a shared interpreted IR, with measured duplication in today's macros and lifter, the expected line reduction, effects on authoring, wasm guest/Realization, performance and the five TLC configuration families, and a recommendation. Errors: none beyond the review.
- **R2:** One front end crate owns attribute/registration/schema/domain reading and the admitted subset; `umpire-dsl-macros` and `umpire-lift` both consume it and no longer parse Model attributes or admit types independently. Errors: unsupported forms produce one located diagnostic from the front end, identical for both consumers.
- **R3:** All checker/report goldens, TLA+/cfg goldens and TLC agreement are byte-identical, or each change is justified by the design; typed counterexample replay is unchanged.
- **R4:** Measured line reduction and `make model-time` output are recorded; MILESTONES and DESIGN.md describe the new boundary.

## Boundaries

- No change to Model authoring beyond what the design justifies; plain Rust stays.
- No backend dropped; TLC stays required and fn-211 consumes the shared front end.
- Not started before fn-227.14 lands, so it builds on the simplified macros and value handoff.

## Decision Context

2026-10-23, owner: "let's enable the deferred ones and add them to the end". Reactivate tasks .5 and .6 after fn-228, followed by fn-237. This supersedes the 2026-10-22 deferral; retain the differential-check requirement and the measured 2x slowdown / 25% per-construct gate before changing the default. Queue order does not add dependency edges.

2026-10-22, owner confirmed ("Keep deferred"): fn-238.4's review measured that Option A halves per-construct cost only for declaration-level constructs, while Option B (.5/.6) adds an interpreter case for every body-level construct on top of lowering and emission. .5 and .6 are deferred until a concrete need (checker speed, a construct rustc cannot run) appears; the owner may revive them.

2026-10-21, owner: share all code except the back end, accepting that a front-end misreading would affect both checkers. This supersedes fn-227's and fn-211's "independent source admission" / "no shared interpretation" wording for future work; counterexample replay remains the cross-check.
