---
title: Match terminal failure and retained retry history separately
date: "2026-10-24"
track: bug
category: integration
module: model/temporal/feature/activity/temporal.rs
tags: [umpire, programmed-workers, describe, failure-history]
problem_type: integration
symptoms: Describe with contradictory last_failure could still be Explained
root_cause: Attempt-only comparison skipped first-attempt failure and confused current terminal failure with previous retry history
resolution_type: fix
---

## Problem
The implementation review found that Describe could report the programmed terminal failure alongside contradictory info.last_failure and still be Explained. Attempt-one failures skipped history comparison, while the later-attempt check treated terminal current failure as previous-attempt retry history.

## What Didn't Work
One attempt-greater-than-one guard cannot classify both retained retry history and current terminal failure. Comparing only outcome.failure leaves a second authored-content evidence channel unchecked.

## Solution
The activity realization checks terminal Failed outcome and last_failure against the current program row. Nonterminal retry history is checked against the previous row. Separate phase-consistency checks reject contradictory first-attempt history and outcome kinds within the admitted lifecycle, while SDK diagnostic source/stack fields remain distinct from authored response content. The Model is unchanged.

## Prevention
Keep a regression that changes only info.last_failure while preserving the expected terminal outcome. Cover first and later attempts, current terminal failure versus retained retry history, and complete payload metadata equality. Require the negative control to fail before the fix; do not broaden a status-only assertion to accept the contradictory evidence.
