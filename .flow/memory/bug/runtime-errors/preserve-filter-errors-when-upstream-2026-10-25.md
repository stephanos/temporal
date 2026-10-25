---
title: Preserve filter errors when upstream receives SIGPIPE
date: "2026-10-25"
track: bug
category: runtime-errors
module: model/scripts/retired-names.sh
tags: [bash, pipeline, signals]
problem_type: runtime-error
symptoms: Pipeline returned141 instead of the native filter error
root_cause: Caller treated induced producer SIGPIPE as a genuine upstream failure
resolution_type: fix
---

## Problem
A streaming native filter can return an error before consuming all git grep output. The producer then receives SIGPIPE and the caller returned141, hiding the original filter error2/9.

## What Didn't Work
Giving every upstream status greater than1 unconditional precedence assumed the consumer always drained stdin, as the retired Python implementation did.

## Solution
retired-names.sh yields to a downstream failure only when the upstream status is141; genuine git errors retain precedence. shell_preserves_filter_failure_when_git_grep_gets_sigpipe uses real git grep and input exceeding pipe capacity. It failed141-versus2 before correction and now checks downstream2/9 and simultaneous genuine upstream7/downstream9.

## Prevention
For each streaming pipeline replacement, test early consumer termination with more than a pipe buffer of input and check caller-level statuses, including genuine simultaneous upstream failures. Direct filter tests do not cover this interaction.
