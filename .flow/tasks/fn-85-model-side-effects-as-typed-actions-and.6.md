---
satisfies: [R6]
---
# fn-85-model-side-effects-as-typed-actions-and.6 Refinement: refines and map checked by the forward simulation

## Description
A machine that declares `refines:` and `map:` is checked against the product machine by the bounded forward simulation inside `Umpire.ImplementationLink`, with same-named values mapped by default, fields the product lacks mapped to `hidden`, and the step mapping derived (a mapped row is a product step, a stutter when the mapped states are equal, or a rejection). A Property on the product machine is checked on the protocol machine's paths (R6).

**Size:** M
**Files:** `model/Umpire/Command/Syntax.lean` (`refines:`, `map:` keys on `machine`), `model/Umpire/Command/Authoring.lean` (derive the value and step mappings; build the `StepPreservation` witness by `decide` over the finite vocabulary), `model/Umpire/ImplementationLink/Language.lean` (a refinement entry point reusing `initialForward`/`stepForward` without the Feature-to-System naming), `model/Umpire/Search/Admission.lean` (a Property on the product machine admits against the refining machine's paths), `model/Umpire/ImplementationLink/Tests/*.lean`, the Nexus command specimens (`#guard_msgs` for the four rejections and a `#guard` on the checked refinement)
**Touches:** [model/Umpire/Command/**, model/Umpire/ImplementationLink/**, model/Umpire/Search/**, model/Temporal/Feature/Nexus/**]

### Approach
- The forward simulation's `initialForward`/`stepForward` are `Prop` fields; the command synthesizes the witness by `decide` over the finite vocabulary (the product and protocol state spaces from task .3 are finite and bounded), never by an authored proof; if `decide` is too slow on the section 3 machines, use `native_decide` as `Property.checked` does and record the measured time.
- `ImplementationLinkRequiredCoverage` demands a mapping or a Known Gap for every action, outcome and observation: derive those maps by name too, and reject an unmapped one at the refinement with its name.
- A refinement is not an Implementation Link (SEM-08); name the entry point `Refinement.check` and keep `checkImplementationLink` for Feature-to-System.
- Rejections: mapping to an undeclared product value; a protocol value with no same-named product value and no map entry; a row whose mapped states are neither a product step nor equal; `map:` without `refines:`.

### Investigation targets
**Required:**
- `model/Umpire/ImplementationLink/Language.lean:22,81,408-457,520-620,681-696,1159` — mappings, `StepPreservation`, the simulation, obligations, `checkImplementationLink`
- `model/Umpire/ImplementationLink/Application.lean:20-147,731-830` — status and result shapes
- `model/Temporal/Feature/Nexus/DESIGN.md` section 2.5 and the section 3 `map:` block
- `model/Umpire/Search/Admission.lean:180-205` — `admit`

**Optional:**
- `model/Temporal/System/Nexus/ImplementationLink.lean` — the only Implementation Link today (kept by fn-86; do not edit)

### Key context
- fn-86 later re-anchors the Feature-to-System link on the product machine this task checks; keep the product machine's element definitions addressable by name.

## Acceptance
- [ ] `nexusProtocol refines: nexusProduct` with the section 3 `map:` checks with a derived step mapping; the witness is synthesized (no authored proof in the Model file), pinned by `#guard`
- [ ] the four R6 rejections reject in place, pinned by `#guard_msgs`
- [ ] `terminalIsFinal` declared on `nexusProduct` is checked on `nexusProtocol`'s paths and a Query over the protocol machine may `find:` it
- [ ] `lake build Umpire.ImplementationLink.Tests TemporalModelTests` green; elaboration time recorded; `make lint-model` green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
