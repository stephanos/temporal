# Task 20 CLI inventory correction

The CLI guide describes the nine pre-existing public flags identified by the
committed task-21 preservation audit, at their accepted operation boundaries.
Current documentation evidence dates the original checkpoint and separates its
formal SHIP source from this correction's pending review.

Task: fn-109-gomad-deepen-modules-and-tool-interfaces.20
Status: in_progress
Base: 5b8261599475f0cef21b6c7d23d272fc9439b26f
Workspace: /Users/stephan/Workspace/skunkworks/gomad/temporal
Tier: session (jev-unavailable(no_key))

Root owns lifecycle, review, staging and commit under the explicit dispatch
override. REVIEW_MODE is codex, conductor-deferred; no host review backend is
substituted. This worker neither commits nor dispatches formal review. The base
through HEAD range is empty. Source admission and historical reports remain
immutable. The original claim timestamp remains the one Flow reports.

## Scope and evidence

The [current evidence](../../documentation-evidence.md#cli-inventory-correction-2026-10-04)
maps all nine flag descriptions to registration, validation and consumer code.
The worker extends existing guide sections, preserving every shell example,
the full command index and SPEC requirement IDs. No Go, protocol, runtime,
Make, golden, policy, assertion or interface-inventory file changes.

[baseline-doc-check.json](baseline-doc-check.json) retains the red reproduction
before guide edits. All nine flags had zero descriptive paragraphs; the source,
historical-artifact and existing document checks passed. The checker was then
strengthened with required syntax/default/effect terms and source rows per flag.
[final-doc-check.json](final-doc-check.json) retains its final observation.
The pre-edit snapshot covers the committed audit's 978 current module entries
and 2,396 historical artifact files. CLI.md is its only changed source entry;
the other 977 and all historical artifacts remain identical.

The pre-edit Quick commands passed. [baseline-focused.log](baseline-focused.log)
retains the two vocabulary/Make ownership tests, and
[baseline-validate.log](baseline-validate.log) retains `make validate`.
[baseline-platforms.log](baseline-platforms.log), [baseline-d5.json](baseline-d5.json)
and [baseline-fn111.json](baseline-fn111.json) retain the other Quick reads.
Baseline exit codes were observed as 0; baseline timings were not separately
captured. [final-quick-receipts.json](final-quick-receipts.json) retains the final
commands, cwd, environment, real exit codes, elapsed times and stable pre/post
source digest. Both final gates exited 0 against stable source. The focused
command ran both named tests and passed in 0.330 seconds including process
startup; `make validate` passed in 2.422 seconds. `git diff --check` exited 0.
The final document check passed all nine content/source rows and every
preservation, link and fence check.

The host is Linux aarch64, and pinned stock Go reports go1.27.1 linux/arm64.
The patched `.toolchain/bin/go` is absent. Required native darwin/arm64 and
linux/amd64 gates remain incomplete. The existing root lint failure and native
gaps are unchanged. MILESTONES verification instruction 3 scopes this document
correction to document/diff checks and admitted focused Quick commands; no broad
host/lint rerun is needed for unchanged executable source.

R18's Go-interface, independent identity and fixed-baseline preservation gaps,
R19/native qualification, task 19's formal review, inherited D5 and fn-105.5
closure remain open. The old three-draw SHIP applies only to its recorded source,
and this worker makes no fresh review verdict or task-completion claim.

## Delegation and stage

Two read-only scouts traced independent target-input and replay/qualification
domains. Both finished with no writes, tests or running commands. Their requested
model was gpt-6.1-sol at high effort. The worker role was requested at that model
and effort too. No executed-model metadata is available, so actual model is not
inferred from those requests. The single writer made all documentation edits.
Every worker/scout command has exited; no running command is handed back.

stage: impl-review - skipped(policy: explicit conductor-deferred codex review; root owns dispatch and verdict)
