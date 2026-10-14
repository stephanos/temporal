---
title: Joined composition keys must be checked injective before keying rows
date: "2026-09-28"
track: bug
category: data
module: model/Umpire/Command/Compose.lean
tags: [compose, umpire, lean, fn-92]
problem_type: data
symptoms: Composed catalog had duplicate keys; walk rows overwritten; spelled-alike domains synchronized
root_cause: String-joined keys and spelling comparisons assumed injectivity without checking
resolution_type: fix
---

## Problem
The first `compose` implementation keyed composed values by string joins and compared synchronized
participants' inputs by their class-key spellings. Review showed three silent corruptions: a `sync:`
name like `agent_resume` duplicated a member action key; member state keys containing `_` made
`a_b`+`c` and `a`+`b_c` the same composed key (the walk's HashMap then overwrote rows); and two
distinct enums both declaring `ok | error` synchronized as if one domain. A Scenario `field.value`
start also parsed the first `-` segment of a member key instead of matching fields structurally.

## What Didn't Work
Rejecting `_` in member *field names* alone: it keeps `<field>_<key>` injective in the field but
not the joined member state keys, and says nothing about sync names.

## Solution
`model/Umpire/Command/Compose.lean` `candidates`: refuse member state keys containing `_`
(`underscoredState`), refuse a sync candidate whose key another candidate has (`duplicateAction`),
and compare classed participants on (input domain declaration names, class keys) from
`Registry.ActionEntry.inputFields`. The composition's Registry entry records each start's held
value spellings so Scenario starts match structurally.

## Prevention
Whenever a key is built by joining keys, prove or check injectivity at the join (separator absent
from every component, generated names checked against every other generated name) before using it
as a map key. Pin collisions with a fixture whose names are spelled alike but declared apart.
