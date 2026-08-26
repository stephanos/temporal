# Veil in the current Umpire model

Status: exploratory architecture note, 2026-08-25. This document records the intended role and
initial adoption shape for Veil. It is deliberately not an implementation specification; exact
interfaces, module names, task boundaries, and delivery sequencing remain to be refined.

## 1. Summary

Veil should be an optional Lean-native verification capability for selected Temporal model
families. It should strengthen the current `model/` pipeline where inductive invariants,
interference reasoning, symbolic search, or SMT-assisted proof provide value beyond the existing
finite planner. It should not replace Umpire's Property, Behavior, Query, Planning, or Artifact
languages, and Umpire should never compile its semantics into generated Veil source.

A family that uses Veil should own three things together:

1. its canonical Umpire target, properties, and semantic identities;
2. a handwritten Veil declaration in the primary Lean project; and
3. an explicit checked binding that states how Veil states, actions, transitions, and properties
   relate to the canonical Umpire model.

Veil results should become provenance-rich verification receipts or replayable counterexamples.
A counterexample must replay through canonical Umpire semantics before it can become a semantic
violation or promoted regression. A successful Veil run must report its trust mode, assumptions,
bounds, omissions, source digest, and semantic digest rather than collapsing into a generic
`verified` Boolean.

## 2. Current position

The current `model/` tree has checked target composition, finite completeness evidence,
deterministic planning, witness and counterexample search, and canonical `ExperimentSpec`
inspection. It does not depend on Veil and does not yet emit the formal-checker receipt described by
C11 in `UMPIRE_COMPONENTS.md`.

The separate `tools/umpire3/model` project is useful prior art. It already has a pinned Veil
dependency, handwritten family declarations, semantic bindings, mutation checks, and proof
receipts. It is not a dependency or semantic oracle for the current model. Reuse should begin with
its lessons and failure modes, not by importing Umpire3 types or artifacts.

There is also an immediate compatibility question: the current model uses Lean 4.33.1, while the
existing Umpire3 Veil project uses Lean 4.28.0. No integration design should be accepted until a
small compatibility proof establishes the supported Veil revision, solver behavior, build cost,
and developer setup for the current toolchain.

The existing follow-up plans do not fill this gap:

- fn-11 adds basic Nexus DSL showcases over existing checking and planning contracts;
- fn-12 decomposes existing Lean tests without changing semantics; and
- fn-13 projects a stable regression into Go and Markdown while explicitly excluding Umpire3 and
  formal-checker reuse.

## 3. Architectural principles

### 3.1 One semantic authority

The current Umpire model remains authoritative for domain vocabulary, target composition,
properties, behaviors, queries, observations, semantic identities, and replay. Veil is a Lean
library and embedded DSL used to express and check an additional view of selected semantics. It is
not a second model catalog, a backend-neutral target language, or an independently meaningful IR.

### 3.2 Handwritten source, checked relationship

Veil declarations are authored Lean source beside the Temporal model family that owns them. Go,
JSON, templates, and code generators must not create or rewrite those declarations. Lean
metaprogramming may reduce local boilerplate, but the resulting declarations and their relationship
to the canonical model must remain inspectable and source-bound.

The binding should identify, at minimum:

- the canonical target and property identities;
- the Umpire and Veil source identities and digests;
- the mapping between initial states, actions, transitions, and observed properties;
- the exact relation being claimed, such as correspondence, simulation, refinement, or a narrower
  preservation statement;
- assumptions, bounds, exclusions, and unsupported vocabulary; and
- the trust mode used to establish the result.

A partial binding must make its omissions explicit. It must not be presented as equivalence merely
because both views accept the same small set of fixtures.

### 3.3 Optional checker layer

Base `Umpire` and ordinary Temporal model imports should remain usable without importing Veil
modules. The eventual package layout should put Veil support behind a distinct import and test
aggregate in the primary model project. Whether that is a separate Lake library, a separate root,
or another acyclic optional layer is an implementation decision, but the dependency direction is
fixed:

```text
Umpire semantics <- Temporal family <- family-owned Veil view and binding
```

Neither `Umpire.Core` nor the Property, Behavior, Query, Artifact, or Planning packages should
depend on a family-specific Veil declaration.

### 3.4 Honest trust and result types

Kernel-reconstructed proof, trusted SMT, testing, bounded symbolic exploration, and concrete trace
replay are different claims. Receipts must preserve that distinction. Timeout, unsupported input,
solver unavailability, incomplete search, stale bindings, and replay disagreement are not success.

The initial result vocabulary should distinguish at least:

- established under a named trust mode;
- violated with a replayable counterexample;
- unknown because the checker or bounds were inconclusive;
- unsupported because the binding cannot represent the requested semantics; and
- invalid because sources, identities, digests, or bindings disagree.

### 3.5 Offline verification only

Veil remains a build, test, and qualification tool. It does not enter Temporal production request
paths, runtime execution, evidence collection, or server binaries. Go may isolate a Lean process
and transport a source-bound receipt, but it does not interpret or reimplement Veil semantics.

## 4. How Veil should be utilized

Veil should be selected per model family and property, not enabled as a blanket second checker for
everything. It is most useful when one or more of these conditions hold:

- the property is naturally stated as an inductive invariant;
- interference between independently modeled mechanisms is central to the claim;
- the current finite planner provides useful examples but cannot justify the desired state-space
  claim within practical bounds;
- symbolic proof or counterexample search materially improves feedback over enumeration; or
- an independent Lean-native view would make a high-value semantic mutation harder to miss.

The ordinary workflow should remain:

1. author and validate the property, behavior, query, and target through public Umpire interfaces;
2. use the existing deterministic planner for fast canonical witnesses and counterexamples;
3. invoke Veil only for a family with an explicit checked binding;
4. bind the Veil result to the same target, property, source, and semantic identities;
5. replay any counterexample through the canonical Umpire transition kernel; and
6. retain a verification receipt whose claim strength remains separate from runtime conformance.

Veil evidence should complement, not replace, direct Lean proofs, checked finite search, mutation
tests, and eventual live execution evidence. Qualification may combine those independent results,
but it must not silently upgrade one kind of evidence into another.

## 5. Proposed first pilot

Adoption should use two proof points rather than attempting a broad migration.

### Phase A: compatibility and ergonomics spike

Use the smallest existing finite target, such as the reusable switch or basic Nexus lifecycle, to
answer only the toolchain questions:

- Can a pinned Veil revision build in the current Lean 4.33.1 project?
- Can Veil remain behind a distinct import and test aggregate?
- Which solver and trust modes work in local and repository checks?
- Are diagnostics understandable to an Umpire model author?
- What are clean-build, incremental-build, and check-time costs?

This spike should not introduce a production semantic claim or a new persisted artifact. Failure
means the dependency or packaging approach must be reconsidered before deeper design work.

### Phase B: meaningful Nexus binding

After compatibility is proven, bind one existing stable Nexus property to a handwritten Veil
declaration. The Workflow-Nexus caller-closure scenario is a plausible candidate because it already
has stable identities, a canonical artifact, multiple semantic steps, and meaningful ownership and
cancellation behavior. The final choice should favor a property with:

- a clear reason to use symbolic or inductive reasoning;
- a small explicit state/action mapping;
- one realistic mutation that the sound declaration rejects;
- one counterexample that can replay through the Umpire kernel; and
- no need to redesign the public DSL or `ExperimentSpec`.

The pilot should establish the binding boundary and trust model, not maximize Veil coverage.

## 6. Verification receipt direction

Formal-checker output should remain separate from `ExperimentSpec`, which describes what may later
be executed. A future versioned verification receipt should reference rather than duplicate the
semantic artifact. Likely fields include:

- receipt and checker format versions;
- checker and dependency revisions;
- target, query, property, source, and semantic identities;
- Veil declaration and binding source digests;
- trust mode, solver mode, assumptions, bounds, and omissions;
- established, violated, unknown, unsupported, or invalid status;
- proof or certificate provenance when available; and
- a counterexample reference plus canonical replay result when violated.

Exact schema ownership and whether the receipt belongs to C11 alone or a broader result envelope
remain open. The receipt must be deterministic, reject stale semantic digests, and avoid embedding
an independently authored copy of Umpire behavior.

## 7. Failure behavior

The integration should fail closed at every semantic boundary:

- an incompatible Veil or Lean revision fails setup or build explicitly;
- unresolved declarations, duplicate mappings, or unsupported state/action shapes invalidate the
  binding;
- a semantic or source digest change makes an older receipt stale;
- solver timeout, resource exhaustion, or unavailable tooling produces `unknown` or an
  infrastructure error, never success;
- a Veil counterexample that cannot replay through canonical Umpire semantics is checker
  disagreement, not a Temporal violation; and
- a mutation that survives both the canonical and Veil checks invalidates the pilot's claimed
  sensitivity and blocks adoption.

## 8. Verification and adoption gates

The eventual implementation plan should include focused checks for:

- direct elaboration of every Veil declaration and binding module;
- positive proof/search results and a nearby failing semantic mutation;
- stale identity, source, digest, and binding rejection;
- concrete counterexample replay through canonical Umpire semantics;
- deterministic receipts across repeated runs;
- honest separation of reconstructed and trusted proof modes;
- an import-graph guard proving ordinary Umpire modules do not import Veil;
- a no-generated-Veil guard covering every family-owned Veil source file; and
- measured local and clean-check cost before adding Veil to a default regression gate.

The initial command should be a focused model check. Promotion into `make umpire-check-regression`
should happen only after compatibility, determinism, runtime, and developer-setup budgets are
explicitly accepted. Expensive or trusted-solver checks may require separate PR and scheduled
profiles, but every skipped check must remain visible rather than being treated as green evidence.

## 9. Non-goals

- Generating Veil source from Umpire declarations, `ExperimentSpec`, Go, JSON, or templates.
- Introducing a general checker-neutral semantic IR.
- Making Veil mandatory for every model family or property.
- Replacing Umpire's finite planner, direct Lean proofs, or canonical replay kernel.
- Importing Umpire3 model, protocol, receipt, or runtime types into the current model.
- Redesigning Property, Behavior, Query, Planning, Observation, or `ExperimentSpec` merely to suit
  Veil.
- Claiming Temporal runtime conformance from a model-checker result.
- Adding Veil to production binaries, runtime endpoints, or deployment paths.
- Porting every existing Umpire3 Veil declaration before one current-model pilot proves value.

## 10. Questions for the detailed plan

1. Which Veil revision supports the current Lean toolchain, and who owns upgrades?
2. Should the optional bridge be a new Lake library, a test-only root, or another module boundary?
3. Which Nexus property provides the smallest meaningful symbolic or inductive proof?
4. What binding strength is required for the pilot: executable correspondence, simulation,
   refinement, property preservation, or a deliberately narrower relation?
5. Which trust mode is acceptable in pull requests, scheduled checks, and qualification?
6. What build and check budgets determine whether Veil joins the default regression command?
7. What is the minimal receipt schema, and how should it bind to current semantic identities?
8. Which Umpire3 implementation patterns should be re-derived, extracted later, or explicitly
   avoided?
9. How should a checker disagreement be retained for debugging without becoming a semantic claim?
10. What evidence would justify expanding beyond the first family?

## 11. Relationship to other plans

- `UMPIRE_COMPONENTS.md` C11 is the parent component boundary. This note supplies the missing
  current-model Veil direction but does not change C11's deferred delivery status.
- `UMPIRE_DSL.md` continues to own the authoring languages and canonical semantics. It needs a
  change only if later work adds an author-visible verification query or other public semantic
  contract; a checker-only integration should stay outside the DSL.
- `UMPIRE_LEAN.md` remains the active Umpire3 roadmap and prior art for Veil constraints. It does not
  allocate implementation work for the current model.
- fn-11 through fn-13 retain their existing boundaries. A detailed Veil integration should be
  planned separately after the compatibility and pilot decisions above are resolved.
