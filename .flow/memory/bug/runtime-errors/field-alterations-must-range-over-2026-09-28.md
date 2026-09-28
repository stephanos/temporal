---
title: "Field alterations must range over structure fields, not the reachable catalog"
date: "2026-09-28"
track: bug
category: runtime-errors
module: model/Umpire/Command/Predicate.lean
tags: [umpire, lean, property, compose, fn-92]
problem_type: runtime-error
symptoms: Valid field claims on refining machines and synchronized compositions refused as fixesNothing
root_cause: "Alterations searched the model's state catalog, which holds derived fields and only reachable composed states"
resolution_type: fix
related_to: [bug/runtime-errors/freeze-contract-transitions-when-2026-09-05, bug/runtime-errors/interface-nil-checks-must-cover-every-2026-09-04, bug/runtime-errors/monitor-closure-must-honor-cancellation-2026-09-04, bug/runtime-errors/replay-rejection-lost-to-2026-09-27, bug/runtime-errors/retain-opaque-completion-authority-2026-09-05, bug/runtime-errors/workflow-replay-must-complete-2026-09-05]
---

## Problem
The first field-addressed requirement enumeration (fn-92.7) changed one state field by searching the Model's own state catalog for a state holding every other field unchanged. Two valid claims failed as `fixesNothing`: a claim on a refining machine's own field, because the refinement's abstract state is lowered as a state field and moves with the concrete one; and a claim on a composition whose members move together, because the state with one member's field changed alone is never reachable.

## What Didn't Work
Treating `DeclaredModel.stateFieldValues` over `model.states` as the counterfactual space. It holds derived fields and, for a composition, only reachable states.

## Solution
`Predicate.lean` takes a `StateFields` view: the states to alter within and each state's structure fields only. `Syntax.lean` `stateFieldsView` builds it: for a machine, `model.states` with `structureFields` filtering to the state type's structure fields; for a composition, the product of member machines' states (anonymous constructor over the composed structure) with `memberFields` lowering each member's structure fields as `compose` does.

## Prevention
A counterfactual "change one field" check needs the type's structural field space, not a catalog of observed or reachable states. Pin one derived-field case (refining machine) and one correlated-members case (synchronized composition) whenever alteration is catalog-driven.
