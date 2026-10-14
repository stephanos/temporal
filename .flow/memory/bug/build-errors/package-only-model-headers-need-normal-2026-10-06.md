---
title: Package-only Model headers need normal compilation in every caller
date: "2026-10-06"
track: bug
category: build-errors
module: Makefile
tags: [model, scala, bloop, header]
problem_type: build-error
symptoms: Bloop reports NoInput for a package-only kind header; scalafix may exit0 despite compiler diagnostics
root_cause: Independent check and rewrite compiler invocations retained different compiler modes
resolution_type: fix
---

## Problem
The package-only Nexus kind header emits no TASTy. Bloop reports NoInput for it in Model compilation. The first implementation fixed Model packaging, tests, classpath and scalafix check mode but left the sibling rewrite-mode Makefile caller unchanged. Implementation review found that gap.

## What Didn't Work
Checking only lint-model-models missed fix-model's separate invocation of the same compiler. Both affected scalafix invocations printed compiler errors despite returning exit0, so numeric success alone could not establish compiler admission.

## Solution
Makefile:754 now selects --server=false for the Models rewrite, matching the affected check-mode caller. An isolated snapshot of the identical committed Model sources reproduced the default compiler diagnostics and then ran rewrite mode without those diagnostics under the normal compiler. Gate regressions assert normal compilation for Model package, test and classpath calls. Tools.orFail, the kind classifier, empty-header shape and all unrelated compiler invocations stay unchanged.

## Prevention
When introducing package-only headers, enumerate package, test, classpath, scalafix check and scalafix rewrite callers separately. Inspect printed compiler diagnostics as well as the numeric exit. Preserve inherited tool warnings as separate observations with retained rule-execution evidence; never use them to waive a new compiler error.
