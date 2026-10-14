---
title: Test output parsers must accept t.Log file:line decoration
date: "2026-09-27"
track: bug
category: integration
module: tools/umpire/cmd/umpire-repeat/signature.go
tags: [umpire, testpilot, test2json, signature]
problem_type: integration
symptoms: TESTPILOT-SIGNATURE lines printed via t.Log never matched
root_cause: "prefix match ran before stripping t.Log's file.go:NN: decoration"
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/check-unbounded-lean-numbers-before-2026-09-07, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/coverage-findings-must-retain-2026-09-05, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/joint-conflict-scopes-require-realized-2026-09-05, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/moved-conformance-tests-must-not-import-2026-09-06, bug/integration/nested-admission-diagnostics-must-2026-09-05, bug/integration/portable-carriers-must-keep-the-checked-2026-09-09, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reconciliation-must-preserve-authority-2026-09-05, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem
umpire-repeat matched `TESTPILOT-SIGNATURE <json>` only when the trimmed output line started with the prefix. Live tests print the line with `t.Log`, which prepends `file.go:NN: `, so every instrumented failure would have silently fallen back to the assertion-only signature and lost the disposition, verdict, rule and diagnostic fields.

## What Didn't Work
The canned fixture printed the line with fmt.Println (undecorated), so the tests passed against a form the real tests never emit.

## Solution
Strip a leading `file.go:line:` (`leadingLocationPattern`) before `CutPrefix` in `signatureOf` (tools/umpire/cmd/umpire-repeat/signature.go), and make the fixture's signature line `t.Log`-decorated.

## Prevention
When a parser reads test output, generate its fixtures from the real emitter (`t.Log`/`t.Error` through test2json), not from hand-written or fmt-printed lines.
