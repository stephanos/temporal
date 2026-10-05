# Task37 import-order source progress

The admitted import-only correction removes the last qualification-package
gci finding. All diagnostic fixture bodies, assertions, comments and import
identities are byte-preserved. No production code or generator input changed.
Earlier task37 evidence and its old-fixture verifier remain historical and
unchanged; this candidate is bound separately in [evidence](evidence.json).

Actual unfiltered pinned lint: baseline exit 1, exactly one gci; worker and
fresh root final exit 0, zero findings. Qualification: 23 top-level tests,
66 including subtests, no failures/skips. Architecture: 1 pass. Errortype,
gofmt and diff checks pass. Commands, measured durations, raw output and
source/tool/config/context bindings are retained alongside the evidence.
Preservation and command-result logs are losslessly JSON-string encoded to
retain their original diff/argv whitespace without failing Git whitespace checks.
Root verified body equality, protected sources, historical artifacts and
all 11 frozen bindings. No native platform or cleanup-fault proof is claimed.

[Independent review](review.md): zero Critical/Important/Minor findings,
SOURCE_PROGRESS_COMMIT_ONLY. Formal plan SHIP approves admission only.
Full/root-fast/native Darwin/formal, original dependency/preservation,
matched first-baseline and affected-consumer acceptance remains open.
Linux stays with fn-128. Task37 and its parent remain incomplete.

Tier: session (jev-unavailable(no_key)); explicit implementer gpt-6.1-sol/high.
stage: wave dispatch - ran (one source/cache writer)
stage: plan review - ran (SHIP; receipt reports gpt-6.1-sol/high)
stage: source-progress review - ran (fresh context; actual host model metadata unavailable)
stage: impl-review - skipped(policy: original qualification is not green; bounded source review is not formal SHIP)
stage: plan-sync - skipped(config: planSync.enabled=false; no task completed)
stage: completion review - skipped(policy: task and spec acceptance remain open)

Spec: fn-109 — deep modules and tool interfaces, open
Tasks: 2 done / 40; task37 source progress only
Tests: qualification, architecture, errortype, formatting, diff and actual package lint pass
Review: SOURCE_PROGRESS_COMMIT_ONLY
Gates: focused small-fix checks; original full/native/formal acceptance open
Tracker sync: n/a (bridge inactive; sync check ran)
Shipped: 0; no push or PR authorized
Next: continue original source-owned acceptance through /flow-next:work fn-109
