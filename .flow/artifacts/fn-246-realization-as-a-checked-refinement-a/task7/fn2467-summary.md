Implemented R2's typed capabilities, one shared action filler and internal evidence views across direct, generic, renamed and generated consumers. The review correction now propagates typed admission refusals unchanged through Reach and retained-prefix validation, keeping admitted predicate rejection distinct from unsupported evidence.

Tier: gpt-6.1-sol at medium (explicit AGENTS implementer; judge unavailable: no_key).

baseline: green — original pre-edit six-crate suite exit 0 (190 passed; one existing ignored live import); initial committed source 42e301bee4 passed 201 tests before the resumed correction. The new regression intentionally failed before the fix and was committed separately. No build command ran during the conductor's cache hold.

Final verification on corrected source: six-crate suite exit 0 (202 passed; one unchanged ignored explicit live-snapshot import); strict six-crate all-targets/all-features clippy exit 0; workspace fmt exit 0; all six current standard guests plus native panic reports exit 0; fresh Kettle Go module exit 0 (34.321s), retaining accepted transcript, parity and mutant goldens unchanged. Prior initial-task evidence remains preserved separately.

The new regression verifies original MalformedRequest/Unsupported code and message, repeated refusal without freezing, pending-request and retained-history preservation, immutable earlier reports, and successful retry for both a new Reach observation and old Reach evidence revalidated by Current. NoPredicate now explicitly admits its custom view and defaults to a rejecting predicate; its intended Unexplained assertion is unchanged. The optional NoEvidence adapter instead returns its exact unsupported refusal. Realize::reach_evidence now returns Result<EvidenceMatch, Refusal>; direct consumers were migrated.

Current/Candidates retain Snapshot-owned admission rather than a forced generic Rpc decoder. Expanded generated snapshot controls pass for both unconditional Current and Candidates handlers: crossed request types, malformed captured Any and absent required response refuse before handlers; response diagnostics retain precedence; explicitly optional absent capture remains allowed. Existing live required-identity checks still precede feature delegation. Stock/non-RPC adapters, Model policy, catalog-owned source and the conductor-reserved macro diagnostic table were not changed. See fn2467-artifacts/logs/review-fix-characterization.md for the validation boundaries and defect-route record.

Defect route:
- prior fixes: local path history identifies 42e301bee4; local branch symptom search found the owned reproduction; conductor assigned this correction exclusively. External PR/tracker search not done for the isolated conductor-owned review fix.
- diagnosis: successful Reach projection, enabled B and correlated pending read produced Unexplained instead of the exact admission refusal because Err was converted to Reject. The identical control passes after Result propagation, without Model changes.
- introduced by: 42e301bee485b60095395b9e4447c33fbd2caf72. Bisect not done: direct introduction identified; previous API had no typed admission Result for this reproduction.
- base: 7911549dae0b6d3c8bdec0498ecf2fa7c57db52a contains the failing reproduction (exit 101, review-fix-red.log). head: 79beb81e4c2cf9d15dd772fa549004d8057b0902 passes the same control and all 9 Reach controls (review-fix-green.log).
- live: library regression proof; six current guests and fresh Kettle transport checks pass. Broader joined/live gates remain conductor-owned.

Complete source range: 046922c29e1355824cc989575dfe24b42e622bca..79beb81e4c2cf9d15dd772fa549004d8057b0902. Correction range: 42e301bee485b60095395b9e4447c33fbd2caf72..79beb81e4c2cf9d15dd772fa549004d8057b0902. All changes are committed, source is clean, and every command/helper is terminal. Task remains in_progress for conductor integration and resumption of its primary review session; no worker review verdict or lifecycle mutation occurred.

Classification remains FULL (exit 1, Cargo.lock), with no docs-only skip or shared gate receipt. Embedded guest promotion/freshness, the conductor's reserved cross-lane diagnostic fix and broad joined model/live gates remain conductor obligations. No .8/.9 formats, .10 frontier, History producer or fn-237.4 wrapper/journal/gates/live proof was implemented.

Proof logs, exact commands/exits/SHA256s, original and corrected guest artifacts, compiler diagnostics and both source patches: /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/milestones-resume-20261024/fn2467-artifacts/proof-manifest.json.

stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)
