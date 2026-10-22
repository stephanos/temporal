---
satisfies: [R2, R4]
---
# fn-221-gomad-functional-suite-triage-sweep-on.2 Run the full sweep and retain the support matrix

## Description
Run the harness over every package under tests on a quiet host with recorded host identity, fixed sorted package order, and the recorded budgets. Write .flow/artifacts/fn221-support-matrix-<date>.md plus a JSON sidecar (shape decided and recorded here) with one row per test function: package, name, class, reason, seed, budget, candidate identity (helper digest, compiler, module identity). Tests cut off by the overall budget are recorded as not run. The first paragraph states this is an observation on one candidate, not a support claim.

**Size:** L
**Files:** .flow/artifacts/fn221-*.md, .flow/artifacts/fn221-*.json

**Touches:** .flow/artifacts/fn221-*.md, .flow/artifacts/fn221-*.json

## Acceptance
- [ ] Every enumerated test function has exactly one class and, for non-pass classes, a reason string
- [ ] Candidate and host identity recorded
- [ ] Not-run tests distinguished from failures
- [ ] Observation-only statement present; QUALIFICATION.md untouched

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

### Rerun in progress, stopped by owner, 2026-10-22 (conductor note)
Conductor decision (owner-delegated): rerun only the 344 load-sensitive units, the 337 helper-link-count infrastructure rows plus the 7 wall-clock timeouts, on the same candidate `1d7b37f00a`. Merge the new rows into `evidence/fn221/full-1d7b37f00a`; the other 1144 rows stand as observed. Guest outcomes are deterministic on the modeled clock, so only wall-clock-bound classes depend on host load.
- **Run root:** `/home/agent/wt/evidence/fn221/rerun-20261022T1705Z`. To resume, follow its `RESUME.md`: rerun the harness command in `command.txt` against the same journal, not the launcher.
- **Kept for the resume:** the source worktree `/home/agent/wt/fn221-rerun-src` (clean, detached at `1d7b37f00a`) and the artifact worktree `/home/agent/wt/fn221-matrix`.
- **Compiler:** the Go 1.27.1 tree was copied to `evidence/fn221/rerun-tools/go1.27.1` because the original path was removed. All 2217 provenance-listed toolchain files hash identical, and the candidate key matched. The conductor accepted this.
- **Progress:** 4 of 344 done, all former timeouts. Each is now capacity-exceeded (WASI network-v2-evidence-bytes) inside the 3m budget; flag this in the artifact. 340 remain: 3 timeouts and 337 infrastructure.
- **Before the merge:** `merge-check.py` must accept multiple journal headers. Ignore the abandoned attempt in `rerun-20261022T1509Z`.
