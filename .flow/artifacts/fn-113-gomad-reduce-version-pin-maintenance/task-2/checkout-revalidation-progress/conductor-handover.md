Committed-scope checkpoint prepared against 717a2de678c743e0f47e07608f94dada3d315330; no task completion or formal SHIP.

The three-line guard rejects same-byte symlink drift before publication, preserves appeared precedence and same-byte regular replacement, and leaves fileDigest, Recover and the remaining check/publication race unchanged. Only transaction.go and the additive public Run test change. Module/source pins, guards, signatures, comments, generators, schemas, locks and journal retirement are unchanged.

Worker unchanged-production RED passes five controls and fails the symlink control; frozen-source GREEN and all portable adapterregen tests pass 40 leaves with zero skips. Conductor focused verification passes six leaves. Fresh independent same-family Codex review found no issues, passed all six final controls and reproduced the single baseline failure with identical final tests using an exact-production overlay. Reviewed candidate and manifest hashes still match. Review receipt and raw logs retain commands, exits and actual shell timings.

Scoped vet, errortype, check-only validate, four architecture checks, formatting and changed-line fast lint pass. Scoped baseline/final lint have identical two unchecked lock.Release findings. Configured lint retains 302 findings before diff filtering; full cleanliness is not claimed. Flow validates 21 specs and 190 tasks with two inherited warning groups. Exact bindings are in conductor-proof.json; task-preservation.json retains the original acceptance/history/evidence comparison.

Source-review SHA256: f2c41636920651d51c84824f74c64cec5fbd1d97042829d6ea77e2cf579470af
Conductor-proof SHA256: 181e70e49136bbd3c69942bac28415ca06f2eeff43f84d75b6b57abd67507463
Reviewer-focused SHA256: f1acd5ce01744a39f0ce42cc5d5672b9ba2768e7bdf74f4ae268a9d21b1a665b
Reviewer-red SHA256: 54907615486fd77578a84b25dc2b4c44ce730e998f6259a652400dbb1aa0f673

Task fn-113.2 is blocked on its unchanged original requirements. Task1 dependency, native Darwin, full/default/functional/affected-consumer and formal acceptance remain open wherever unproved. Linux stays deferred and unverified under fn128. No patched driver or platform evidence was fabricated; no unavailable-driver checks were repeated.

The worker discloses its missing /usr/bin/time launcher, metadata-path probes and initial diagnostic-only test revision. Conductor read-only probes initially used a nonexistent short task filename and invalid fn113.2 alias; corrected canonical paths/IDs supplied subsequent context and no state was changed by failed probes. A metadata rewrite patch using delete+add against one path was rejected without changes, then corrected to Update File. An rg trailing-whitespace search returned 1 for no matches after successful Flow validation; this was not a failed validation. No setup/intermediate observation was counted as a passing gate.

Spec: fn-113-gomad-reduce-version-pin-maintenance
Tasks: fn-113.2 blocked; no task completed by this checkpoint
Tests: portable adapterregen and focused RED/GREEN; standards results above
Review: source-progress-acceptable only (same-family Codex)
Gates: native/full/default/functional/affected/formal remain open; configured lint red
Tracker sync: n/a (bridge inactive)
Shipped: 0 (no PR or push)
Next: continue original milestone acceptance on a supported native Darwin host; retain Linux deferral
stage: source-progress-review - ran (model: gpt-6.1-sol at high)
stage: impl-review - skipped(policy: original Quick/native/full gates not green)
stage: plan-sync - skipped(config: planSync.enabled=false; no completed tasks)
stage: completion-review - skipped(policy: original task/spec acceptance remains open)
