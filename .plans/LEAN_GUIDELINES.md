# Lean Authoring Guidelines

Use these guidelines when creating, changing, or reviewing Lean 4 code. The goal is code that is
idiomatic to its Lean ecosystem and approachable to readers who are still learning Lean.

Readability is a layer over idiomatic Lean, not a replacement for it. Keep standard Lean notation,
proof techniques, and abstractions; explain unfamiliar concepts at module and API boundaries.

## 1. Orient to the project

Before editing:

1. Read `lean-toolchain`, the Lake configuration, the relevant imports, and nearby source files.
2. Determine whether the code follows Lean Std, mathlib, or a project-specific style.
3. Identify the repository's build, test, formatting, and lint commands.
4. Search the project and its dependencies for existing definitions, theorems, instances, and
   notation that cover the requested behavior.

The local project is the source of truth. Mirror its naming, namespaces, imports, formatting, and
proof style when they differ from these general guidelines. Do not change the Lean version or add a
broad import merely to make one proof easier.

This step is complete when the relevant project profile, reusable API, and verification commands
are known.

## 2. Design declarations as interfaces

Treat a theorem statement or definition signature as an API. Prefer a clear, reusable statement
even when a more awkward statement would make its first proof shorter.

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
- Keep domain-specific automation when it communicates the method clearly. Tactics such as `simp`,
  `ring`, `omega`, `linarith`, and `norm_num` are often more informative than their low-level proof
  expansions.
- Use search commands and tactics such as `#check`, `#print`, `exact?`, `apply?`, `simp?`, and
  `aesop?` during exploration when they are available. Replace exploratory commands with the
  resulting stable proof or an intentional final automation call.
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

Give a substantial module a `/-! ... -/` docstring that explains:

- the problem the module addresses;
- its central types and definitions;
- its main invariants or chosen normal form;
- its important entry points;
- non-obvious notation and design decisions.

### Declaration documentation

- Give public definitions and major theorems `/-- ... -/` docstrings.
- State a theorem's mathematical meaning in plain English.
- Explain the concept represented by a definition, its important edge cases, and its observable
  behavior.
- Make each public docstring understandable in an editor hover, without requiring the declaration's
  implementation or surrounding source to be visible.
- Add small, checked examples for important user-facing APIs.

### Comments

- Preserve existing comments when changing or refactoring code.
- Update a comment when the behavior or invariant it describes changes.
- Use comments to explain intent, invariants, proof phases, and non-obvious representation changes.
- Keep comments close to the code they explain.
- Prefer a short explanation of why a step exists over a paraphrase of its Lean syntax.

Documentation is complete when a reader can identify the module's purpose and understand each
public declaration's meaning before reading its implementation or proof.

## 5. Control scope and assumptions

- Keep `open`, `open scoped`, `attribute`, local instances, and option changes as narrow as
  practical.
- Use `classical` or `noncomputable` when the definition or proof genuinely requires it. Scope it
  locally when possible, and document a surprising nonconstructive dependency.
- Use type classes for coherent, reusable structure rather than as shortcuts for passing arbitrary
  local data.
- Finished code contains no `sorry`, `admit`, accidental axioms, or unsound escape hatches. Add an
  `axiom` only when an explicit specification calls for a new assumption.
- Resolve warnings and linter findings or document the specific reason an exception is necessary.
- Check that elaboration did not add stronger hypotheses, unwanted type-class requirements, or an
  unnecessary nonconstructive dependency.

This step is complete when the declaration's assumptions are deliberate, visible in its interface,
and no wider in scope than necessary.

## 6. Verify with Lean

1. Compile the smallest affected target after each logical change.
2. Run the focused tests or checked examples for the changed behavior.
3. Run the repository's normal build, test, formatting, and lint commands before finishing.
4. Inspect warnings, unused imports, unexpected axioms, and unexpectedly slow elaboration.
5. Re-read the theorem statements and public APIs independently of their proofs to catch accidental
   changes in meaning or generality.

Never infer validity from visual inspection alone. Work is complete only after Lean checks every
changed declaration and the repository's required verification passes.

## Reader's key to common Lean syntax

Keep a key like this in project onboarding documentation rather than repeating it in every source
file:

- `(x : α)` means the caller supplies `x`.
- `{x : α}` means Lean normally infers `x`.
- `[C α]` asks type-class synthesis for the structure or capability `C α`.
- `P → Q` transforms evidence of `P` into evidence of `Q`.
- `∀ x, P x` states that `P x` holds for every `x`.
- `∃ x, P x` states that some `x` satisfies `P x`.
- `P ∧ Q`, `P ∨ Q`, and `¬ P` mean and, or, and not.
- `def` introduces data or a computation.
- `theorem` and `lemma` introduce checked proofs.
- `by` begins a tactic proof.
- `:=` separates a declaration's interface from its implementation or proof.

## Upstream mathlib contributions

When contributing to mathlib, follow its current contribution policy in addition to these
guidelines. AI use must be disclosed as required by that policy, and the human contributor must
understand and be able to justify all submitted content. Compose GitHub and Zulip comments in the
human contributor's own words.

## References

- [Lean standard-library style](https://github.com/leanprover/lean4/blob/master/doc/std/style.md)
- [Lean standard-library naming](https://github.com/leanprover/lean4/blob/master/doc/std/naming.md)
- [Lean documentation style](https://github.com/leanprover/lean4/blob/master/doc/style.md)
- [Lean tactic proofs](https://lean-lang.org/theorem_proving_in_lean4/Tactics/)
- [Lean simplifier and simp sets](https://lean-lang.org/doc/reference/latest/The-Simplifier/Simp-sets/)
- [Mathlib style guide](https://leanprover-community.github.io/contribute/style.html)
- [Mathlib naming conventions](https://leanprover-community.github.io/contribute/naming.html)
- [Mathlib contribution and AI policy](https://leanprover-community.github.io/contribute/index.html#use-of-ai)
