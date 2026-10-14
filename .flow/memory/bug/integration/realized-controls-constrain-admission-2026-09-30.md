---
title: Realized controls constrain admission evidence compatibility
date: "2026-09-30"
track: bug
category: integration
module: model/scalav2/specimens/activity.md
tags: [evidence, causal-order, admission]
problem_type: integration
symptoms: An evidence-negative trace was labeled inconclusive despite decisive hold controls
root_cause: The oracle omitted realized control ordering from compatible executions
resolution_type: documentation
related_to: [bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/coverage-findings-must-retain-2026-09-05, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/keyed-capture-operands-must-be-checked-2026-09-09, bug/integration/portable-carriers-must-keep-the-checked-2026-09-09, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04]
---

Removing an internal admission-commit observation does not automatically make an activity pause race inconclusive. A realized hold-before-dispatch, completed pause, and subsequent release constrain the compatible executions: pre-pause admission may be excluded by the control and initial custody rules. The specimen originally conflated this held scenario with an uncontrolled concurrent pause/poll scenario.

Keep the uncontrolled evidence-negative oracle separate from the held-control oracle. Report inconclusive only when the surviving public and control evidence admits executions with different contract verdicts. The fn-107 task1 native review identified this distinction; the revised A10 oracle and same-session SHIP review confirm the correction.
