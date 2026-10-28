Wrapping up: I've traced the one real code change (the TLC prefetch in the required tests stage) against the unchanged gate, scheduler, standalone-child and fetch-script code, and checked the verifier scripts and acceptance evidence. Writing the review now.

**Scope reviewed.** Task fn-258.2 (R2/R3) at standalone `05473210`, tree `46829511`. The tracker diff is artifact-only; the standalone source was read through `standalone/source-adaptation.patch` (relative to export `a92db061`) and the in-tree base `model/umpire/dev/xtask/src/gate.rs`, which is byte-identical to the standalone base apart from allowlisted seams. The sibling `../umpire` checkout is not readable from this session, so Git byte identity rests on the committed verifier outputs. Consumer, R6 preparation and cleanup artifacts were only cross-referenced.

**What I verified on the integration axis**

- **The gate.rs seam meets unchanged code correctly.** `execute(name, …)` and `script(…)` exist unchanged at `model/umpire/dev/xtask/src/gate.rs:85-105`; the prefetch runs under the same `command()` environment (`UMPIRE_BACKENDS=require`, cwd = root) and the script resolves `<root>/scripts/fetch-tlc.sh`, writing `<root>/build/tlc/tla2tools-1.7.4.jar`, which is exactly the path the TLC runner refuses on (`backends/tlc/src/runner.rs:165`, and the ac79114 failure log line 234). `required` reaches `--scheduled-stage` children via `run()` at `gate.rs:338-339`, so the parent's scheduled `tests` stage prefetches; the real 054 `make validate` confirms 18 parent stages green.
- **Why only the standalone needed it.** In-tree `make model-validate` goes through `model/scripts/time-gate.sh`, which never fetches TLC and relies on a pre-provisioned `model/umpire/build/tlc` (documented at `model/README.md:890`). The standalone `Makefile` calls `xtask validate` directly on a cold checkout, so the parent `tests` stage was the only required path without provisioning. The in-tree copy is untouched, so no regression there.
- **Idempotent with the standalone child's existing setup.** `standalone.rs:418` already runs `scripts/fetch-tlc.sh` as the `standalone-tlc-tool` setup command, so the child's `tests` stage now re-runs it; `fetch-tlc.sh:36-43` short-circuits on an existing checksum-valid jar. Concurrent cold fetches use `mktemp` plus atomic `mv`, so parallel stages cannot observe a partial jar.
- **No inventory race.** The standalone source inventory excludes `build` (`standalone.rs:126`), so a jar appearing mid-run cannot trigger "standalone controls changed source inputs".
- **Receipts and timing output.** Receipts are committed by `scheduler::scheduled` around the whole stage closure, not by `execute`, so the setup failure yields one `tests` receipt with exit 37 (test asserts this, 30/30 green). In timed mode the stage name prints twice on stdout; `timing-controls` already does the same (`gate.rs:134-140`, visible at gate.log lines 57-58), so no consumer is broken.
- **Skip mode and other entries unchanged.** `UMPIRE_BACKENDS` unset or `skip` leaves `tests` exactly as before; `update`/`time` without require are untouched, matching the updated expectations in `gate_wiring.rs`.
- **Verifier scripts.** `compare_prior_adaptations` now rehashes all 54 historical before/after blobs against the frozen c88 manifest and f7b42ce tree, binds the inventory SHA-256, and checks both unchanged-flag directions; 51 unchanged / 3 changed matches the allowlist. The arithmetic reconciles: 54 + 3 + 2 = 59 adaptations, 1189 − 59 = 1130. `verify-history-join.py` proves 054 is the direct child of ac79114 with exactly the two approved paths at their pinned blobs/modes; `current_receiver_allowlist_sha256` equals the equivalence report's `allowlist_sha256`.
- **Initial three findings resolved.** Current core/Kettle `proto-go.sh --check` and real `source-boundary` receipts at 054 (`current-freshness/receipt.json`, exit 0, clean tree); prior-hash verification with 14 negative controls; explicit fixture-crate freeze comment and prose.

## Findings

No P0/P1 introduced findings.

- **Severity**: P3 · **Confidence**: 75 · **Classification**: introduced · **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/standalone/source-adaptation.patch` (hunk for `dev/xtask/README.md`) · **R-IDs**: [R3]
  **Problem**: The standalone `dev/xtask/README.md` (an allowlisted adaptation touched in this task) still describes `validate` as only "forcing required TLC" and does not say the required `tests` stage now provisions the pinned jar via `fetch-tlc.sh`, including on direct `gate --stage tests` and in scheduled children. A reader following the CLI contract will not expect network access or `build/tlc` writes from the `tests` stage.
  **Suggestion**: One sentence in the stages section noting that required-mode `tests` runs `scripts/fetch-tlc.sh` first and propagates its exit code as the stage status.

**FYI (not findings)**
- `required_tests_provision_tlc_before_running_and_preserve_setup_failure` is a four-deep loop running 16 subprocess matrices; readable enough for a wiring test, but a table-driven `[(timed, scheduled, backend, failed)]` would halve the nesting.
- The allowlist reason for `gate.rs` describes the mechanism rather than naming the "portability" category the acceptance bullet enumerates; the equivalence prose supplies the rationale, so the bullet is satisfied.
- `current-readiness.json` binding digests for the equivalence/history JSON files could not be recomputed without a shell; treat as unverified rather than wrong.

## Pre-existing issues (not blocking this verdict)

- [P3, confidence 75, introduced=false] `model/umpire/dev/xtask/src/gate.rs:389-393` — `update` with `UMPIRE_BACKENDS=require` calls `gate.tests()` directly, so on a cold checkout the TLC fixture refuses with "run bash scripts/fetch-tlc.sh" rather than provisioning. The new test matrix deliberately encodes this (`update` expects no `tlc-tool`), and the in-tree flow has the same gap, so it is a recorded decision, not a regression.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | deferred | Task 1 closed; task 2 consumes its ac60 manifest unchanged (sha `a29ea70…` bound in both verifiers). |
| R2 | met (task-2 share) | `history-join-verification.json`: join `c71c309` parents `[5ee8911, a92db06]`, first-parent tree preserved, export selected tree equals frozen source, all 14 historical/filtered ancestors reachable from 054, exactly two approved receiver paths. |
| R3 | met | Canonical Go identities, `go_package`-only proto edits, pinned regenerated descriptors, 499 protected records exact, 133 additions present / 122 deletions absent, boundary controls retained, current freshness and source-boundary receipts at 054 exit 0, full required-TLC `make validate` 18 parent / 17 child stages green. |
| R4 | deferred | Task 3. |
| R5 | deferred | Task 4 (named check/time/release/paired proofs remain open, as the artifacts state). |
| R6 | deferred | Task 5. |

Unaddressed R-IDs: []

Suppressed findings: 1
Classification counts: 1 introduced, 1 pre_existing.
Protected-path filter: 0

```json
{"suppressed_count":{"50":1},"classification_counts":{"introduced":1,"pre_existing":1},"unaddressed":[]}
```

<verdict>SHIP</verdict>