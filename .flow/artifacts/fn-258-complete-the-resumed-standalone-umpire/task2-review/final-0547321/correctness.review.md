**Review scope.** Task fn-258.2 at standalone commit 05473210 (tree 46829511). The only source delta since the previously reviewed ac79114 is the gate change in `dev/xtask/src/gate.rs` plus its controls in `dev/xtask/tests/gate_wiring.rs`. Everything else in the range is evidence: the adaptation verifier extension, history-join verifier extension, freshness receipts, the retained `make validate` proof, and tracker/doc updates.

**What I verified**

- The gate change is nine lines. When the gate runs in required mode, the `tests` stage first executes the existing `scripts/fetch-tlc.sh` through the same `execute` path, returns its non-zero code without running tests, and otherwise falls through to the unchanged `tests()` command. Skip mode is untouched. The `required` flag is set for `validate` or `UMPIRE_BACKENDS=require`, and the parent propagates that env to scheduled stage children, so `--scheduled-stage tests` sees the same decision. This matches the red/green logs and the 30-control wiring run.
- The script is idempotent and repo-contained: it exits 0 quickly when the pinned jar is present with the right SHA-256, so the child gate (which already fetches via `standalone-tlc-tool` setup at `standalone.rs:418`) does not double-download. The jar lands in `build/`, which both the boundary scanner and the standalone input inventory skip, so the parallel `tests` fetch cannot perturb `source-boundary` or `standalone` inventories.
- `verify-adaptation.py` prior-adaptation logic: it binds the fn-257 inventory by SHA-256, rehashes each of the 54 before blobs from the c88 Temporal tree and each after blob from the f7b42ce standalone tree, then checks the allowlist's `prior_adaptation_after_sha256` and the unchanged flag in both directions against current bytes and modes. The allowlist carries exactly 162 prior-adaptation fields, which is 54 × 3. The 51/3 split and the three new refusal controls are computed, not copied.
- `verify-history-join.py` receiver checks: parent of 05473210 is ac79114, changed paths equal exactly the two approved paths in diff-tree order, blobs/modes equal the allowlist pins, and all 14 historical ancestors remain reachable. Counts in the markdown (1,189 / 1,130 / 59 / 499 / 133 / 122, 14 controls) agree with the JSON.
- Current-freshness receipt: core and Kettle `proto-go.sh --check` and the real `source-boundary` stage exit 0 at 0547321 with clean tracked/untracked source.

**Findings**

- **Severity**: P3 · **Confidence**: 75 · **Classification**: introduced · **File:Line**: `.flow/artifacts/.../standalone/source-adaptation.patch` (hunk for `dev/xtask/src/gate.rs:160`) · **R-IDs**: [] · **Problem**: The TLC fetch is reported under the stage name `tests`. In timed mode (`time`, `validate`) stdout gets two `tests` timing lines per run, and a fetch failure is announced as `framework stage tests failed (exit N)` rather than naming the setup step. The script's own stderr still names the real cause, and `timing-controls` already uses this pattern, so this is cosmetic. · **Suggestion**: Pass a distinct name such as `tests/tlc-tool` to `execute` for the fetch step and adjust the one assertion in the new control.
- **Severity**: P3 · **Confidence**: 75 · **Classification**: introduced · **File:Line**: `dev/xtask/tests/gate_wiring.rs` (hunk at `required_profile_reaches_gate_update_and_direct_test_children`) · **R-IDs**: [] · **Problem**: The control now pins that `update` and direct `test-run` under `UMPIRE_BACKENDS=require` still do not prefetch TLC, so those entries keep failing late inside the TLC backend when the jar is absent. That is unchanged behaviour, but the test now encodes it as intended. · **Suggestion**: Either accept as the documented contract or route `update` through the same prefetch; no change needed for this task.

**FYI, non-blocking.** `source-adaptation.patch` was regenerated with zero context lines for readability in the tracker diff; it is not applied anywhere and Git blobs remain the identity authority, so this is fine. The verification scripts and receipts are evidence artifacts, not shipped code.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | met | Closed in fn-258.1; this task reuses the frozen ac60 manifest by SHA-256. |
| R2 | met | History-join checks pass; join tree equals first-parent tree; export, portable and filtered ancestries reachable from 05473210; export selected tree equals frozen source. |
| R3 | met | 1,189 files mapped, 59 pinned adaptations with rationale, 499 protected records identical, Go/proto diffs normalize to identity only, descriptors regenerated and freshness green, source-only child gate passes 17 stages with no sibling dependency, boundary and caller controls retained. |
| R4 | deferred | Consumer work, task 3. |
| R5 | deferred | Paired fresh-target, release and local-edit proofs, task 4. Timing misses are reported, not closed. |
| R6 | deferred | Publication and cutover, task 5. |

Unaddressed R-IDs: []

Suppressed findings: 1
Classification counts: 2 introduced, 0 pre_existing.

```json
{"suppressed_count":{"50":1},"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>