---
satisfies: [R1, R2, R3, R4]
---
# fn-249-durable-validation-cost-review-at-spec.1 Enforce and integrate durable validation-cost assessment at spec close

## Description
Deliver the parent contract in one implementation iteration through the existing Flow close/config/parser owners, automated-close caller, repository patch/test precedent, project opt-in and agent instructions.

**Size:** M
**Files:** new `tools/flow/validation-review.patch` and `tools/flow/test_validation_review.py`; `.flow/config.json`; `AGENTS.md`. Preserve `tools/flow/committed-dependencies.patch` and its harness. Installed integration targets are the current `flowctl.py` and `make-pr-preflight.sh`; changes there must be reproduced by the repository patch rather than becoming untracked installation-only work.
**Touches:** [tools/flow/**, .flow/config.json, AGENTS.md]

### Approach

1. After committed fn-247.4 closure and root release, reanchor installed `cmd_spec_close`/parser/config and `make-pr-preflight.sh`. Capture current per-target hashes, existing patch state and active gomad installation ownership. Keep task/source work outside other workers' owned paths. The existing close reads task inventory before its lock and lacks multi-file rollback; preserve these limits rather than adding a lifecycle redesign.
2. Add `specClose.validationReviewRequired=false` and optional `--validation-review-file`. Use strict config loading for this setting so invalid JSON/type cannot substitute a permissive default. Under the existing close lock, reload status and validate a new-transition report before the task-write loop. Already-done calls keep existing checks/timestamp behavior and never require or replace a report. Opt-out/no-file calls add no report Git probes. A provided file on a new closure is validated even when policy is off.
3. Keep a minimal v1 JSON schema in ordinary Python helpers, with no schema framework. Required fields are `schema_version=1`, canonical `spec_id`, `gates` and `speedup_decisions`. Gate rows name command/mode, cost kind measured/estimated/unknown, coverage and available evidence; measured/estimated seconds are finite and nonnegative, unknown has a reason. An empty gate list explains why. Decisions name the opportunity, decision and concrete rationale, preserving coverage; no-safe-savings is valid. Include existing review overhead and optional Flow follow-ups/source basis where useful, without requiring new executions, ID allocation or deep provenance. Bound report size at 256 KiB and collections at 128 entries. Wrong types/version/spec, malformed JSON or excess bounds return corrective refusals.
4. Accept uncommitted input under the repo, including `.flow/tmp`. The conventional automated-close input is `.flow/artifacts/<resolved-spec-id>/validation-cost-review.json`, which may be user-authored or staged. Reject path escapes/symlink substitution, parse/hash one exact byte buffer and publish it atomically as `.flow/artifacts/<resolved-spec-id>/validation-cost-review-<sha256>.json` before status writes. An existing artifact must have the exact expected hash/bytes and is never overwritten on mismatch. Store compact artifact path/SHA-256/schema version in the spec JSON and include the validated artifact in modified_paths on every successful new closure, whether newly published, already tracked or reused uncommitted after failure so the normal closure commit makes them durable. No pre-closure commit, HEAD equality, clean-spec gate or mandatory spec-source hash is added; optional source basis remains report content.
5. Extend the actual make-pr close caller to forward conventional input only for enabled new closures, reusing existing config plumbing where available. Missing/invalid input stops before closure staging/commit or PR creation. Preserve opt-out, already-done, dry-run and body-only behavior. Normal close modified_paths always includes the validated artifact, including reuse after failure, for its existing commit fence. Never generate a report, invoke another model or run validation automatically.
6. Add the isolated controls below. Report validation/artifact-publication failure leaves spec/task lifecycle bytes unchanged; acknowledge existing lock/ignore maintenance. Retry while open revalidates input and safely reuses an identical published artifact. A later close-write failure can leave that artifact and existing partially published tasks; no new rollback transaction is claimed. Reopen preserves history but requires explicit report input again. Do not add task epochs, generation attestations or task-inventory race fixes.
7. Stage patched CURRENT copies of both installed targets and run the two focused harnesses against the combined staged installation. Retain known reversible backups; recheck each baseline hash under the conductor's narrow installation lease immediately before atomic per-file replacement. Abort/rebase if another installer changed either baseline. Install compatible CLI first, then preflight, and enable this project's knob last. Preserve earlier installed committed-dependency changes and document partial multi-file recovery. Do not stop other repositories or require global idleness; already-loaded processes continue. Update AGENTS with the quick assessment, input/copy locations, normal report-plus-closure commit, reinstall patch stack and checks. Read the writing-for-agents skill before editing AGENTS. Implementation/install waits for committed fn-247.4 and root release; root may author the plan now in its explicit metadata-only window.

### Investigation targets

**Required** (read-only scout coordinates, reanchor before implementation):

- Installed `flowctl.py:37767-37832` and parser `:57680-57685` for actual close, state writes and flag surfaces.
- Installed `flowctl.py:1388,1608-1639,1870-1911` for config/default/strict read and existing lock.
- Installed `flowctl.py:13601,13672,33658,35223-35240` for raw hashing and local Git object-read conventions; `:31092-31109` for reopen.
- Installed `make-pr-preflight.sh:191-233` and flow-next make-pr workflow close ordering for conventional report forwarding.
- `tools/flow/committed-dependencies.patch` and `test_committed_dependencies.py:12-48` for patch stacking, isolated repos and configurable `FLOWCTL_PY`.
- `AGENTS.md:179-195` and `.flow/config.json` for this project's install/closure instructions and opt-in.
- `.flow/tmp/closure-cost-{repo,spec,memory}-scout.md` and `closure-cost-gap-analysis.md` for evidence, declined scope and limits.

## Acceptance
- [ ] Run the new isolated Python harness with temporary Git repos/CLI copies and no network or real project closures. Cover measured/estimated/unknown costs, justified no-validation/no-safe-savings, uncommitted or staged input, exact copied artifact/hash/reference and modified_paths including both new and reused artifacts. Inject failure after artifact publication but before lifecycle writes, retry through the real close/commit fence and verify that the committed tree contains the referenced report bytes. Prove the normal closure commit includes the report without a preliminary commit.
- [ ] Table-drive missing/malformed/wrong-spec reports, unsafe paths/symlinks, bounds, invalid config, corrupt preexisting artifacts and injected publication failure. Assert unchanged lifecycle bytes and useful text/JSON guidance; no HEAD/spec-clean or arbitrary evidence-tree gate is added.
- [ ] Prove opt-out/no-file compatibility, supplied-file behavior, incomplete-task refusal, repeated-done reference preservation, explicit input on reopen, retry reusing the same content-addressed artifact and existing partial-write failure semantics. No new generation, rollback or task-inventory concurrency contract is claimed.
- [ ] Stub preflight/config/Git/PR boundaries to verify forwarding and failure before staging/PR, plus opt-out/done/dry-run/body-only branches. Assert that close and validation-report forwarding invoke no tests, LLM, additional network fetch or optimization command. Preserve the preflight's existing base-ref refresh fetches; do not forbid them in a full-preflight trace.
- [ ] Run `FLOWCTL_PY=<staged-flowctl.py> python3 tools/flow/test_committed_dependencies.py` plus the new focused harness against the combined patch. Simulate installation baseline mismatch on temporary copies; preserve committed-unpublished predecessor admission and known backups. No Rust/Go full gate is justified for this tooling-only change.
- [ ] Repository patch, safe installed integration, project opt-in and AGENTS/reinstall instructions match the parent. Existing coverage, live proofs, history, other-repo opt-out and active gomad work remain preserved. Retain exact command/output evidence.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
