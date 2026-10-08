# Source-task resumption and bounded owner decisions, 2026-10-07

The owner requested unblocking and ultimately completing the source tasks, then selected the current `gomad` branch. The 58 previously blocked source tasks return to `todo` with their existing dependencies and acceptance intact except for the two expressly approved decisions below. fn-128 and fn-149 remain deferred; no PR, push or CI action is authorized.

## Approved format/controller migrations

The owner answered “Yes, recognize only those approved migrations” to recognizing the already-approved fn-114 trace/controller migrations within fn-109 preservation. R18 therefore recognizes fn-114.11's Choice Trace v3 and explicit v2 refusal, and fn-114.12's controller/select-poll migration and recorded controller-v2 journal refusal. Verify each change against its actual approved owner contract and retain its identity/refusal controls.

This decision authorizes no decoder restoration, new migration, default change, feature removal, golden rewrite or blanket preservation waiver. Matched first-baseline and fixed-identity preservation remain required for all unaffected behavior. Existing source/API migrations retain their original authorization and proof requirements. The historical selected-v041 retirement receives no waiver; its later restoration must be reconciled.

## Approved two-site invariant-panic exception

The owner answered “Yes, allow only those two exact sites” to a lint exception for the existing pre-mutation panics in `SeedController.Complete` in `tools/gomad3/runner/internal/campaign/controller.go`:

- `panic("gomad3: completed an inactive campaign attempt")`
- `panic("gomad3: completed a campaign attempt without a classification")`

Keep the exact messages, ordering, conditions, panic mechanism and zero-mutation/rejection tests unchanged. Scope the lint exception to this file and these exact source statements; retain actual-tool negative controls for another statement and another path. This supersedes fn-109.28's suppression/policy-change prohibition only for those two approved sites. No other panic, error-string, output, cleanup, load or watchdog finding receives an exception.

## History and source acceptance

The pre-reset live states, blocker reasons and dependency lists are retained in [before-reset.json](before-reset.json). Original task bodies are immutable at `bc548110b9321df59d757d0e0e5c0fea464c002b`. The resumption restores each original task's historical summary/evidence after the CLI reset and adds the authoritative current-work note through `flowctl task set-spec`; historical blocked/source-progress narration does not change current `todo` status.

No reset establishes completed acceptance. Retain portable coverage, original-base lint, generated validation, both-source-set static checks, byte equivalence, applicable non-native measurements, predecessor/source-review requirements and native-owner links. Complete each task only through `flowctl done` with verified evidence and review. The two unrelated `.turbo` documents remain excluded from commits.
