**Scope reviewed.** Task fn-258.2 (R2/R3) at standalone `05473210`, tree `46829511`: the two-file source delta in `standalone/source-adaptation.patch`, the allowlist/equivalence/history-join evidence and both verifier scripts, the current-freshness and TLC-prefetch control receipts, the `standalone-0547321` validation summaries, the retained initial `ac79114` review, and the MIGRATE/MILESTONES hunks. The sibling `../umpire` checkout is not readable from this sandbox, so byte identity is taken from the committed verifier outputs. Consumer, disk and task5-preparation artifacts were read only for cross-reference and are outside this task's acceptance.

**What checks out (contracts vs. code)**

- Inventory arithmetic agrees everywhere: 1,189 source files, 1,130 unchanged, 59 allowlisted modifications (57 prior + exactly `dev/xtask/src/gate.rs` and `dev/xtask/tests/gate_wiring.rs`), 499 protected records, 133 additions present, 122 deletions absent. Neither new path is admitted by wildcard; both are pinned by exact mode/blob and corroborated by `05473210`.
- The three initial-review findings are resolved as claimed. `current-freshness/receipt.json` binds `proto-go.sh core --check`, `kettle --check` and the real `source-boundary` stage to `05473210` with exit 0 and a clean tree; the empty logs are consistent with the script, which only prints on `diff` mismatch. `compare_prior_adaptations` rehashes all 54 historical before/after blobs from the unchanged fn-257 inventory (SHA bound), checks each claimed prior hash and both directions of the unchanged flag; 51/3 matches the JSON. Fixture-crate source freeze is now explicit in the verifier comment, the prose, and two new negative controls; 9 + 2 + 3 = 14 controls as stated.
- `verify-history-join.py` adds receiver checks that match the prose: parent of `05473210` is `ac79114`, changed paths are exactly the two gate files in `diff-tree` order, blobs/modes match `approved_after`, and all 14 ancestry commits remain reachable. The qualification note correctly scopes "boundary-only" to `ac79114`.
- The gate change does what the allowlist says: in the `tests` arm, when `self.required`, it runs `scripts/fetch-tlc.sh` through the existing `execute` supervisor and returns its non-zero code before `tests()`; skip mode and the test invocation are untouched. The red log (expected `tlc-tool` before `tests`) and the 30-test green log confirm ordering, failure propagation, skip behavior and scheduled receipts. This also fulfils the already-approved standalone README promise that "the gate provisions … checksum-pinned backend tools … as needed", which the prior tree did not honour for the parent required tests stage (in-tree Temporal relies on a manual `model/scripts/fetch-tlc.sh` per `model/README.md:890`).
- Validation summaries agree with raw receipts: 18 parent stages (17 + `standalone`) and 17 child stages all `success`; tests stage 915/912/3/0 in both; `make-validate.log` shows 1679.762 s gate and 24.162 s inner loop, matching MILESTONES and MIGRATE; recovery-only verifier 158 objects / 7,979 members / pass.
- MIGRATE.md and MILESTONES.md correctly state the task as still open pending fresh source review, and do not claim task 4/5 obligations.

## Findings

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/standalone/adaptation-allowlist.json:749`
- **R-IDs**: [R3]
- **Problem**: Acceptance bullet 3 requires every expected difference to name one of identity-rewrite, regenerated-descriptor, quick-selection, portability or boundary-policy rationale. The `dev/xtask/src/gate.rs` reason describes the mechanism ("supervises existing fetch-tlc.sh before tests and propagates setup failure") but names no category; the sibling `gate_wiring.rs` entry does say "portability controls". A reader auditing the allowlist against the acceptance text cannot tell from the entry alone why this difference from the frozen `ac60` source is admitted.
- **Suggestion**: Prefix the reason with the category, e.g. "Portability: a framework-only checkout has no Temporal-side TLC provisioning, so the required tests stage supervises the existing fetch-tlc.sh …". Prose-only; no code or pin change.

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/standalone/source-adaptation.patch:500` (gate.rs `tests` arm)
- **R-IDs**: []
- **Problem**: The prefetch is executed as `self.execute(name, …)` with `name == "tests"`, so its timing line and failure diagnostic are indistinguishable from the test run. `make-validate.log:81-82` shows two `tests` lines (1.149 s and 502.073 s), and a fetch failure reports `framework stage tests failed (exit N)`. The test fixture's `tlc-tool` label comes from the fake script, not from the gate, so the readable receipts do not actually carry a distinct setup label; the child `standalone` stage, by contrast, reports `standalone-tlc-tool` separately. The claims "exact setup failure preserved" and "setup … receipts" hold for the exit code and stage receipt, but not for attribution.
- **Suggestion**: Report the prefetch under a distinct name (e.g. `execute("tests-tlc-tool", …)`) or leave as is and note in the allowlist reason that setup failures surface under the `tests` stage label. Non-blocking either way.

**FYI (not findings)**

- `compare_prior_adaptations` iterates policy entries, so a historical path dropped from `modified_paths` would not be flagged by that check; today 54 == 54 and the main `compare` would still catch any such path that differs from the frozen source, so no failing scenario exists.
- The required inner loop now pays ~1.1 s for the cached fetch-tlc hash check on every required tests run; the loop already misses its 10 s soft target and the miss is reported, so this is a timing observation only.
- `verify-adaptation.py` relies on `assert` for prior-inventory binding; running under `python -O` would silently skip those checks. The documented invocation does not use `-O`.
- Hash bindings in `current-readiness.json` could not be recomputed without a shell; treated as unverified, not wrong.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | deferred | Task 1 (closed); task 2 consumes the `ac60` manifest unchanged (sha `a29ea70…` bound in both verifiers). |
| R2 | met (task-2 share) | `history-join-verification.json`: ordered parents `[5ee8911, a92db06]`, first-parent tree preserved, export selected tree equals frozen source, all historical/portable/export/join ancestors reachable from `05473210`; original `cf8bec4` retained. Commit map is task 1's closed evidence. |
| R3 | met (task-2 share) | Canonical `github.com/temporalio/umpire/{hosts/go,reference/kettle}` identities, `go_package`-only proto edits, pinned regenerated descriptors, exact LICENSE/descriptor boundary policy, root support/docs/CI; current protobuf freshness and source-boundary receipts at `05473210`; full required-TLC `make validate` 18/17 stages green in a fresh target. Release tests and paired proof remain task 4 per the spec's interim-ownership section. |
| R4 | deferred | Task 3. |
| R5 | deferred | Task 4. |
| R6 | deferred | Task 5. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 2 introduced, 0 pre_existing.
Protected-path filter: 0

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>