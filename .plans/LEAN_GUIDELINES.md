# Lean Authoring Guidelines

Use these guidelines when creating, changing, or reviewing Lean 4 code. The goal is code that is
idiomatic to its Lean ecosystem and approachable to readers who are still learning Lean.

Readability is a layer over idiomatic Lean, not a replacement for it. Keep standard Lean notation,
proof techniques, and abstractions; explain unfamiliar concepts at module and API boundaries.

For a read-only review, use the authoring rules as evaluation criteria and run only applicable
non-mutating checks. Do not edit or regenerate files merely to complete the authoring workflow;
report the changes and checks the implementation still needs.

## 1. Orient to the project

Before editing or evaluating a change:

1. Locate every Lake workspace that directly compiles or imports the changed source. Read each
   workspace's `lean-toolchain` and Lake configuration, plus the relevant imports and nearby source
   files. A repository may contain multiple consumer workspaces with different toolchains and
   dependencies.
2. Classify each target as authored or generated. Honor ownership markers, generated-file headers,
   and project documentation. When implementing a generated Lean change, change the owning
   generator or input, regenerate the complete owned surface, and run its staleness and downstream
   checks.
3. Determine whether the code follows Lean Std, mathlib, or a project-specific style, and identify
   the intended public API from its documented facades and module boundaries.
4. Identify each consumer workspace's build, test, formatting, and lint commands, plus any
   repository-level generation or regression gate that covers the changed surface.
5. Search the project and its dependencies for existing definitions, theorems, instances, and
   notation that cover the requested behavior.

The local project is the source of truth. Mirror its naming, namespaces, imports, formatting, and
proof style when they differ from these general guidelines. Do not change the Lean version or add a
broad import merely to make one proof easier.

This step is complete when file ownership, every consumer workspace, their available libraries and
tactics, the reusable API, and the applicable verification gates are known.

## 2. Design declarations as interfaces

Treat a theorem statement or definition signature as an API. Prefer a clear, reusable statement
even when a more awkward statement would make its first proof shorter.

In this guide, *public* means part of an intentional exported API identified by the project's
facades and documentation, not every declaration that happens to lack the `private` modifier.

- Give public declaration arguments and return values explicit types.
- Put operations and theorems in the namespace of their principal type. This makes names
  predictable and enables natural dot notation.
- Follow the project's naming conventions. When it follows Lean Std or mathlib, use:
  - `UpperCamelCase` for types and propositions;
  - `lowerCamelCase` for functions and values;
  - `snake_case` for theorems and proof-valued declarations.
- Prefer ordinary inputs and hypotheses to the left of the colon:

  ```lean
  theorem result (x : α) (h : P x) : Q x := by
    ...
  ```

- Use conventional short names such as `α`, `x`, and `h` in small generic scopes. Use domain
  names when a value has a stable role or a proof spans enough lines that short names become
  ambiguous.
- Use the library's established normal forms in theorem statements and results.
- Introduce notation, coercions, type-class instances, or attributes only as part of a coherent,
  reusable API.
- Add `@[simp]` only when a theorem reliably rewrites expressions toward an agreed canonical form.
  The right-hand side should be structurally or semantically simpler than the left-hand side.

### Deep modules

Encapsulate a complicated representation behind a small semantic API. Provide the constructors,
accessors, elimination principles, extensionality lemmas, and simplification lemmas that callers
need. Keep representation-specific lemmas private or clearly separated from the public interface.

Downstream proofs should normally use API lemmas instead of unfolding implementation details.
Repeated uses of `unfold`, `erw`, representation-specific `change`, or a cleanup `rfl` after an API
operation are signals to look for a missing lemma.

The design step is complete when callers can use the declaration through stable, semantic names
without knowing its representation.

## 3. Write legible proofs

Expose the proof's main argument and let automation handle routine leaves.

- Use a direct proof term for genuinely simple proofs.
- Use `calc` for equality, inequality, and other transitive reasoning chains.
- Use tactic mode for multi-step proofs, with one conceptual step per line.
- Use `have`, `suffices`, or `show` to name important intermediate statements. Give a nontrivial
  intermediate fact an explicit type.
- Use bullets or named `case` blocks when a tactic creates multiple goals.
- Prefer semantic library lemmas over unfolding definitions or relying on incidental definitional
  equality.
- Use `simpa using ...` when the remaining difference is routine normalization.
- Keep domain-specific automation when the active imports provide it and it communicates the method
  clearly. Tactics such as `simp`, `ring`, `omega`, `linarith`, and `norm_num` can be more
  informative than their low-level proof expansions, but some are ecosystem-specific. Verify the
  narrow import that provides a tactic; do not broaden imports or add a dependency solely to gain
  access to it.
- Use search commands and tactics such as `#check`, `#print`, `exact?`, `apply?`, `simp?`, and
  `aesop?` during exploration when the active imports provide them. Replace exploratory commands
  with the resulting stable proof or an intentional final automation call.
- Preserve command-based checks such as `#check`, `#guard`, and `#guard_msgs` when elaboration,
  diagnostics, imports, or visibility are the behavior intentionally tested by a dedicated test
  module.
- In `simp`, name the definitions or nonstandard lemmas that explain the important reduction. Keep
  a clear terminal `simp` instead of replacing it with a generated wall of default `simp only`
  lemmas.
- Expand long semicolon chains, broad `<;>` expressions, and deeply nested anonymous proof terms
  when they obscure which goal is being solved.

For example, prefer visible structural reasoning when it is the point of the proof:

```lean
example (P Q : Prop) (h : P ∧ Q) : Q ∧ P := by
  obtain ⟨hP, hQ⟩ := h
  exact ⟨hQ, hP⟩
```

A proof is complete when its structure communicates why the result holds, its automation is
intentional, and Lean accepts it without placeholders.

## 4. Add a human-readable boundary

Use documentation to bridge Lean syntax and the project's domain without translating every line of
code.

### Module documentation

Give a new substantial module, or one whose purpose or API boundary materially changes, a
`/-! ... -/` docstring that explains:

- the problem the module addresses;
- its central types and definitions;
- its main invariants or chosen normal form;
- its important entry points;
- non-obvious notation and design decisions.

### Declaration documentation

- Give new or materially changed public definitions and major theorems `/-- ... -/` docstrings.
- State a theorem's mathematical meaning in plain English.
- Explain the concept represented by a definition, its important edge cases, and its observable
  behavior.
- Make each public docstring understandable in an editor hover, without requiring the declaration's
  implementation or surrounding source to be visible.
- Add small, checked examples when they are the clearest documentation or regression for an
  important user-facing API being created or materially changed.

Keep existing module and declaration documentation accurate when the behavior it describes
changes. In learner-facing onboarding or tutorial documentation, include a locally tailored key for
non-obvious Lean syntax rather than repeating a generic key in source files.

### Comments

- Preserve existing comments when changing or refactoring code.
- Update a comment when the behavior or invariant it describes changes.
- Use comments to explain intent, invariants, proof phases, and non-obvious representation changes.
- Keep comments close to the code they explain.
- Prefer a short explanation of why a step exists over a paraphrase of its Lean syntax.

Documentation is complete when a reader can identify the module's purpose and understand the
meaning of each new or materially changed public declaration before reading its implementation or
proof. Do not expand a focused change into documentation work on unrelated declarations.

## 5. Control scope and assumptions

- Keep `open`, `open scoped`, `attribute`, local instances, and option changes as narrow as
  practical.
- Use `classical` or `noncomputable` when the definition or proof genuinely requires it. Scope it
  locally when possible, and document a surprising nonconstructive dependency.
- Use type classes for coherent, reusable structure rather than as shortcuts for passing arbitrary
  local data.
- Resolve warnings and linter findings introduced by the change. For unrelated pre-existing output
  from a required check, verify it against a known baseline and report it without expanding the
  task. If no reliable baseline exists, or project policy requires that gate to be green, treat the
  failure as a blocker unless the responsible owner explicitly accepts a waiver.
- Check that elaboration did not add stronger hypotheses, unwanted type-class requirements, or an
  unnecessary nonconstructive dependency.

### Proof trust

Choose proof techniques according to the declaration's assurance boundary:

- Proof terms whose dependencies stay within the declaration's approved axiom baseline are the
  default for reusable and load-bearing theorems.
- `native_decide` relies on compiler evaluation and adds an axiom dependency: some toolchains use
  `Lean.ofReduceBool`, while newer ones create an auditable axiom for each invocation. Use it only
  where the local trust policy accepts the entire native path, commonly in tests or private
  computation witnesses. Native evaluation also trusts the compiler, runtime, and the semantic
  correctness of native implementations selected through `@[implemented_by]` or `@[extern]`;
  `#print axioms` records acceptance of compiler trust but cannot validate those substitutions.
  Trace whether a private witness feeds a public or load-bearing value; the witness's visibility
  does not contain its trust dependency. For a load-bearing declaration whose policy does not
  explicitly accept that path, use a non-native proof.
- Finished authored proofs contain no placeholder terms or tactics, such as uses of Lean's `sorry`
  or `admit` syntax. Add an `axiom` only when an explicit specification calls for that assumption
  and requires it to be disclosed.

Audit changed trust-bearing declarations with `#print axioms Fully.Qualified.name`,
`Lean.Util.CollectAxioms`, or a project checker documented to compute an equivalent transitive
inventory. Take the approved baseline from an explicit local trust policy or checker, not merely
from existing use or passing tests. If no baseline exists, compare an existing declaration's
inventory before and after the change and reject new dependencies unless the specification
explicitly approves and documents them. For a new trust-bearing declaration, introduce no axiom
dependency until its assurance boundary is explicitly stated. `sorryAx` is always a failure in
finished authored code. Compiler-trust dependencies—including bespoke `native_decide` axioms,
`Lean.trustCompiler`, `Lean.ofReduceBool`, and `Lean.ofReduceNat`—and custom axioms are acceptable
only when the local trust policy or explicit specification permits them. `Classical.choice`,
`propext`, and `Quot.sound` are not automatically failures once a project boundary exists; their
acceptability depends on that declared boundary.

This step is complete when the declaration's assumptions and trust dependencies are deliberate,
audited, visible at the relevant boundary, and no wider in scope than necessary.

## 6. Verify with Lean

1. Compile the smallest affected target in the owning workspace after each logical change.
2. For new or materially changed observable computational behavior, add or update an executable
   Lean regression with positive and relevant failure cases. Put it in a Lean test declaration or
   project doctest that an applicable verification command actually checks; an unchecked Markdown
   fence is not a regression. A proof-only refactor that preserves the theorem statement and
   behavior need not add a test.
3. Run the focused tests or checked examples for the changed behavior.
4. Run the applicable normal build, test, formatting, and lint commands for every consumer
   workspace. Run a repository-level generation, regression, or integration gate when project
   documentation identifies it as covering the changed surface.
5. Inspect warnings, unused imports, axiom-audit results, and unexpectedly slow elaboration.
6. Re-read the theorem statements and public APIs independently of their proofs to catch accidental
   changes in meaning or generality.

Never infer validity from visual inspection alone. An implementation is complete only after Lean
checks every changed declaration and every applicable required verification gate passes, except
for a verified pre-existing failure handled by project policy or an explicitly accepted waiver. A
read-only review is complete after the appropriate non-mutating checks run and any unexecuted
authoring checks are reported.

## Upstream mathlib contributions

When contributing to mathlib, follow its current contribution and AI policy. The human contributor
must disclose AI use as required and understand and justify all submitted content. An agent must
not generate, rewrite, or post GitHub or Zulip comments; hand that communication to the human
contributor.

## References

For technical references, select the source tag or manual version matching each affected
`lean-toolchain`; the unversioned links below are discovery pointers. Treat contribution-policy
links as intentionally current.

- [Lean standard-library style](https://github.com/leanprover/lean4/blob/master/doc/std/style.md)
- [Lean standard-library naming](https://github.com/leanprover/lean4/blob/master/doc/std/naming.md)
- [Lean documentation style](https://github.com/leanprover/lean4/blob/master/doc/style.md)
- [Lean tactic proofs](https://lean-lang.org/theorem_proving_in_lean4/Tactics/)
- [Lean simplifier and simp sets](https://lean-lang.org/doc/reference/latest/The-Simplifier/Simp-sets/)
- [Lean axioms and `native_decide`](https://lean-lang.org/doc/reference/latest/Axioms/)
- [Lean proof validation](https://lean-lang.org/doc/reference/latest/ValidatingProofs/)
- [Mathlib style guide](https://leanprover-community.github.io/contribute/style.html)
- [Mathlib naming conventions](https://leanprover-community.github.io/contribute/naming.html)
- [Mathlib contribution and AI policy](https://leanprover-community.github.io/contribute/index.html#use-of-ai)
