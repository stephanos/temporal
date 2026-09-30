---
title: Channel catalogs and visible results must survive source lifting
date: "2026-09-30"
track: bug
category: integration
module: model/scalav2/lifter/Lift.scala
tags: [modelir, channels, refinement, source-admission]
problem_type: integration
symptoms: Visible stutter outcomes and unsupported channel declarations silently changed semantics
root_cause: New declarations omitted outcome visibility and finite message/policy admission
resolution_type: fix
---

A source lifter must preserve the semantic parts of a native declaration, or refuse its unsupported form at the declaration. fn-107.2 initially carried only fact visibility in a refinement, so a state-preserving result with a visible outcome could pass as a stutter. Channel lifting also reduced a declared finite Int catalog to an unbounded type, and treated unknown order/loss expressions as default policy cases.

The fixes add a separate typed visible-outcome projection through native refinement, modelir and lifting; a stutter may emit neither a projected fact nor outcome. A supported Int channel carries the declared Finite.upTo range. Nonrepresentable Int/list message catalogs receive source-linked refusals. Channel policy parsing names both supported cases explicitly and rejects computed forms rather than guessing their meanings.

Regression coverage should include a projected-outcome stutter, finite message round trips, nonrepresentable catalogs, and valid Scala computed policy expressions that the portable subset cannot admit. Keep source-lifter admission separate from arbitrary-IR validation and semantic execution; success at the frontend does not certify a raw backend IR file.
