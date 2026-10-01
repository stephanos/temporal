---
satisfies: [R1, R2, R3]
---
# fn-111-gomad-consolidate-vocabulary-and-update.1 Verify canonical vocabulary and preserve semantic identifiers

## Description
Implements R1-R3. Reuse the user's consolidation in 83d143293; recover its pre-consolidation parent 29917069e089dc0739ec091b18e99161245b9bd5. Audit the glossary Language section, retained assessments, SPEC definitions and aliases, historical README parity disposition, and the complete ordered semantic identifier inventory including command tables. Correct actual vocabulary defects only. Files: tools/gomad3/SPEC.md, tools/gomad3/README.md, deleted tools/gomad3/GLOSSARY.md; scoped evidence under .flow/artifacts/fn-111-gomad-consolidate-vocabulary-and-update. Anchors: SPEC.md:24 and fn111 spec Terminology Decisions. Quick: revision-bound vocabulary/identifier comparison, current glossary absence, scoped baseline-to-current git diff --check. No executable changes, staging, commits, or worktrees.

## Acceptance
All 25 original Language entries are assessed, all 24 current concepts survive as definitions or aliases, and Parity Case remains explicitly historical. The original 123 semantic identifiers remain unique and in their original order. Independently inspect Target/Prepared Target, Campaign/plan, Choice Trace/Decision Tape, exact/prefix replay, Backend/Fidelity, and World boundaries. Record actual source revisions and file hashes; any missing concept, identifier, or conflated guarantee leaves the task incomplete.

## Done summary
Audited the consolidated vocabulary against pre-consolidation revision 29917069e0 and corrected four vocabulary defects in tools/gomad3/SPEC.md and tools/gomad3/README.md: Fidelity no longer reads as independent of Backend (Model Fidelity on both Backends, Hard Isolation process-only), Campaign is distinguished from its Portable Plan, two remaining "prepared Target" spellings use the canonical Prepared Target, and the README simulation narrative names Parity Case as a historical term. All 25 Language entries are assessed, 24 survive as definitions or aliases, and the 123 semantic identifiers (28 in command tables) are unique and in original order; evidence is verify-vocabulary.py, vocabulary-audit.json, and task1-review.md under .flow/artifacts/fn-111-gomad-consolidate-vocabulary-and-update.

Nothing is committed or staged: the conductor forbade commits, so evidence records "commits": [] and the user commits the changed paths. documentation-audit.json still carries the pre-edit SPEC and README hashes; task .2 re-runs verify-documentation.py.

stage: impl-review - ran (raw codex bridge on working-tree diff; commits forbidden)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: python3 .flow/artifacts/fn-111-gomad-consolidate-vocabulary-and-update/verify-vocabulary.py (pre-edit rc=1 on 4 intended checks; post-edit suite_rc=0, errors=[]), git diff --check 29917069e089dc0739ec091b18e99161245b9bd5 -- tools/gomad3/SPEC.md tools/gomad3/README.md tools/gomad3/GLOSSARY.md (rc=0), git diff --check HEAD -- tools/gomad3/SPEC.md tools/gomad3/README.md (rc=0), baseline: green (123/123 identifiers in order, glossary absent, whitespace clean, pre-edit), codex exec -s read-only --skip-git-repo-check -m gpt-5.6-sol -c model_reasoning_effort=high (raw bridge, working-tree diff of SPEC.md, README.md and the two new evidence files against HEAD; round 1; session 01a0f4fc-8ded-7742-88ef-a0a9f6b90714): VERDICT: SHIP, no findings - .flow/artifacts/fn-111-gomad-consolidate-vocabulary-and-update/task1-review.md
- PRs: