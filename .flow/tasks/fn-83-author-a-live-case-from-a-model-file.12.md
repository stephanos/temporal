---
satisfies: [R12]
---
# fn-83-author-a-live-case-from-a-model-file.12 Vocabulary declarations: an enum command and a generated Setup

## Description
Shrink a Model file's vocabulary declarations to the names an author means (R12). Two changes:

1. **An `enum` command.** Today every domain is an `inductive … where | … deriving BEq, DecidableEq, Repr`; the `deriving` clause is required by the `model` command but means nothing to an author.

   ```lean
   /-- Waiting recognizes an observed change; it does not cause the operation to change. -/
   enum Action
     | awaitStart
     | awaitSuccess
   ```

   expands to exactly the `inductive` the file declares today, with `deriving BEq, DecidableEq, Repr`.

2. **No `Setup` in the author's file.** The `model` command resolves a type literally named `Setup` in scope that must have exactly one constructor, and nothing else in the file mentions it: the `scenario` line `operation starts scheduled` names the *State*. The `model` command generates the Setup domain itself.

**Size:** M
**Files:** the Umpire command module from .10 (the `enum` macro beside the five Model commands; the `model` elaborator's Setup resolution), `model/Temporal/Feature/Nexus/Success/Model.lean` (four `enum` domains, no `Setup`), `RaceSyntaxTests.lean` (drops its `Setup`), the command tests (`#guard_msgs` and `#guard` pins), `.flow/tasks/fn-83-author-a-live-case-from-a-model-file.6.md` and `.8.md` (wording: "`enum` or `inductive` declarations", no `Setup`)
**Touches:** [model/Umpire/**, model/Temporal/Feature/Nexus/Success/**, .flow/tasks/fn-83-author-a-live-case-from-a-model-file.6.md, .flow/tasks/fn-83-author-a-live-case-from-a-model-file.8.md]

### Approach
**`enum`**
- A `macro`, not an elaborator: it resolves nothing. Sketch: `doc?:(docComment)? &"enum" name:ident ("|" (docComment)? ident)+ : command` → `inductive $name where … deriving BEq, DecidableEq, Repr`. The leading `&"enum"` is a non-reserved keyword so no identifier named `enum` breaks; none exists in `model/` today (only comments and a string literal).
- Preserve the type doc comment and any per-constructor doc comment. Constructor order is exactly the authored order: it is the ordered domain AUT-09 names, and Actions must stay sorted, so the macro never reorders. Unsorted Actions keep failing in the `model` command with the existing located message.
- Plain `inductive` stays admitted. Keep `RaceSyntaxTests.lean`'s four domains on `inductive` so both spellings stay pinned; convert `Model.lean` to `enum`.

**Generated Setup**
- The `model` command emits the single-constructor Setup type itself, scoped to the model so two models in one namespace cannot collide (for example `lifecycle.Setup`), and passes its constructor where it passes the resolved one today.
- The Setup key is part of the model's canonical description (`Umpire/Model/Canonical.lean`, `"setups"`), which feeds fingerprints. Both existing command-authored models name their Setup constructor after their start state (`Setup.scheduled` for `starts [scheduled]`, `Setup.queued` for `starts [queued]`), so deriving the constructor name from the start state keeps their bytes. With several start states, pick a deterministic rule (for example the first in sorted `starts`) and record it; state whether a fixed name would have moved bytes.
- The `model` command no longer looks up `Setup` in scope. A file that still declares its own `Setup` is harmless unless it collides with the generated name; if it can, reject in place naming the generated declaration.
- `Temporal.Feature.Nexus.Race.*` declares its own `Setup` through the expert path and is out of scope.

**Both**
- Byte pin: every checked-in fixture, golden, Definition ID and fingerprint is unchanged.
- Spec fit: AUT-09 already admits domains a command derives from author-declared enum-like inductives; an `enum` is an author declaration. The generated Setup is a single-constructor domain the command derives from the author's `starts` list, which AUT-09's "derived from declarations the author wrote" covers; if review disagrees, draft the wording in `.8`'s AUT-09 amendment rather than adding a spec edit here.

### Investigation targets
**Required:**
- the Umpire command module from .10 — the `model` elaborator (`domainConstructors … Setup`, the `setupConstructor` match) and the five commands' `#guard_msgs` pins
- `model/Umpire/Model/Canonical.lean` — how the setup key enters the canonical description
- `model/Temporal/Feature/Nexus/Success/Model.lean`, `RaceSyntaxTests.lean` — the declarations to convert
- `.plans/LEAN_GUIDELINES.md` — macro and doc-comment conventions

### Key context
- .5 and .6 add new Model files and .8 writes the tutorial; they depend on this task so they use `enum` and omit `Setup` from the start.
## Acceptance
- [ ] `enum Name | a | b` elaborates to `inductive Name where | a | b deriving BEq, DecidableEq, Repr`, keeping the type and constructor doc comments and the authored order
- [ ] `Model.lean` declares its domains with `enum` and declares no `Setup`; `RaceSyntaxTests.lean` keeps `inductive` for its four domains and declares no `Setup`; both elaborate through the `model` command
- [ ] The `model` command generates a model-scoped Setup whose constructor name follows a recorded deterministic rule; two models in one namespace elaborate without a collision, pinned by `#guard`
- [ ] Unsorted `enum Action` still rejects with the existing located message; a colliding hand-declared Setup (if reachable) rejects in place; both pinned by `#guard_msgs`
- [ ] Every checked-in fixture, golden, Definition ID and fingerprint is byte-identical
- [ ] `.6` and `.8` task wording admits `enum` declarations and no longer mentions a `Setup` declaration
- [ ] `cd model && lake build`, `make lint-model`, `make umpire-check-goldens`, `make umpire-check-case-runtime-conformance` pass
## Done summary
Two shrinkings of what a Model file writes before the five commands start.

**`enum`** (`model/Umpire/Command/Syntax.lean`): a macro, not an elaborator, expanding to exactly the
`inductive ... deriving BEq, DecidableEq, Repr` a Model file wrote four times. It resolves nothing
and reorders nothing, so unsorted Actions still fail in the `model` command with the same located
message (pinned with an `enum`-declared domain as well as the existing `inductive` one). A plain
`inductive` is still admitted, and `RaceSyntaxTests.lean` keeps that spelling so both stay pinned.

Grammar deviation, recorded: a member's own doc comment goes **after** its bar
(`| /-- ... -/ awaitStart`), not before it. Before the bar it cannot be distinguished from the doc
comment of whatever declaration follows the `enum` -- the repetition swallows it and the next
command fails to parse. Both `atomic((docComment)? "|")` and the bare
`((docComment)? "|" ident)+` form were tried and neither backtracks correctly. The type's own doc
comment is in the conventional place.

**No `Setup` in a Model file.** Nothing else in one mentioned it -- `operation starts scheduled`
names the State -- so an author was declaring scaffolding whose only requirement was to exist with
one constructor. The `model` command generates `<model>.Setup`, scoped under the Model's own name so
two Models in one namespace cannot collide, with one constructor named after the Model's **first
start state**.

That rule is what keeps bytes: the canonical setup key feeds `Umpire/Model/Canonical.lean`'s
`setups` and every fingerprint built on it, and both command-authored Models already named their
Setup constructor after their start state (`Setup.scheduled` for `starts [scheduled]`,
`Setup.queued` for `starts [queued]`). A fixed name such as `Setup.setup` would have moved both.
With several start states the rule is still "the first in the declared `starts` list", which the
`model` command already requires to be sorted, so it is deterministic.

A file that declares its own `Setup` is harmless, because the generated one is `<model>.Setup`; a
`#guard`-backed test declares one alongside a Model. The unreachable
"a Model takes its setup domain from a type named `Setup`" diagnostic is deleted.

Byte pin: `make umpire-check-case-runtime-conformance` and `make umpire-check-goldens` are clean
with no regeneration.

`make umpire-check-regression` is exit 0 end to end (571 Lean jobs, 9 passing live identities);
`make lint-model` reports 0 findings outside generated `Temporal/API/Proto.lean`.

Task wording for `.6` and `.8` now admits `enum` declarations and no `Setup`.

Swept in, not mine: `model/Temporal/Feature/Nexus/DESIGN.md` (425 lines, a design specimen headed
"Nothing here compiles, and no module imports it"), left untracked in the shared checkout by the
parallel session and staged by `git add -A`.

Review: SHIP after one NEEDS_WORK round. The P0 was real and mine -- task `.8`'s tutorial outline
still said `inductive`s -- and is fixed. The P3 dead `setupDomainMessage` was valid and is deleted.
The remaining P3 is the swept DESIGN.md.
Pinned reviewer `claude:claude-fable-5-1:high` is account-limited for this session, so both rounds
ran on `claude:claude-sonnet-4-5:high` -- a same-family fallback, not an equivalent cross-family
review.

stage: impl-review - ran, 2 rounds (model: claude-sonnet-4-5, high; fable pinned but account-limited)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f126f41a94, 1d96b80031
- Tests: cd model && mise exec -- lake build, make umpire-check-case-runtime-conformance (no regeneration needed), make umpire-check-goldens, make lint-model (0 findings outside generated Temporal/API/Proto.lean), make umpire-check-regression (exit 0)
- PRs: