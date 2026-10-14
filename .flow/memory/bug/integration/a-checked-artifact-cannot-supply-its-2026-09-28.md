---
title: A checked artifact cannot supply its own completeness universe
date: "2026-09-28"
track: bug
category: integration
module: model/Umpire/Command/ComposeProofs.lean
tags: [compose, kernel, decide, proof, fn-92]
problem_type: integration
symptoms: Dropping an enabled action with its rows still passed composedTableAgrees and the kernel proof
root_cause: Completeness and reachability quantified over the literal's own action catalog instead of the declared candidates
resolution_type: fix
---

## Problem
The first `composedTableAgrees` took its completeness universe from the literal it was checking:
`literal.states.all (sourceComplete members literal.actions literal.rows)`, with `Reachable`
defined over that same catalog. All three review draws reproduced the gap: a literal with an
enabled action dropped together with its rows and the states only it reaches still passed the
check, and the kernel accepted `ComposedAgreement.ofChecked` on it, so the theorem could not
certify that the walk kept every enabled declared action.

## What Didn't Work
Checking rows against the composition function and reachability over the rows is sound (no row
is invented) but says nothing about rows that were never emitted when the action they belong to
was never emitted either: an artifact cannot be the universe of its own completeness check.

## Solution
`model/Umpire/Command/ComposeProofs.lean`: `composedTableAgrees members candidates literal` takes
the candidates as their own argument; completeness and `Reachable` range over them, and every
catalog action must be a candidate. `model/Umpire/Command/Syntax.lean`: the composed Action union
derives `Umpire.Command.Finite`, and the theorem's candidates are
`candidatesOf ((members (α := Action)).map view.action)`, read off the generated type, never off
the walk. Pinned by a literal with an enabled action dropped whole.

## Prevention
When a decided check certifies an artifact against a semantics, every universe the check
quantifies over (actions, states, starts) must come from the declaration side, not from the
artifact; write the "dropped whole" negative test for each universe before the positive pin.
