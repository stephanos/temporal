### Correctness axis

## Quality Audit — Correctness axis: task 2, `0e88ae54..5f4d62e9`

### Summary

- Files changed: 28 · Critical 0 · Should Fix 1 · Consider 0 · Ship: ⚠️ Fix first

### Should Fix

- **model/umpire/backends/tlc/src/process.py:34** (Conf 75): The supervisor inherits the owner’s process group and has no SIGTERM handler. Group termination kills both while the backend survives in its new session — isolate the supervisor from caller-group signals, route catchable termination through cleanup, and test group termination.

### Evidence and verification limits

- **Executed:** launched an isolated owner → supervisor → sleeping backend, then called `os.killpg(owner.pid, SIGTERM)`. Owner exited `-15`, supervisor wait status was `15`, and `/proc/<backend>/stat` remained `S`. All isolated processes were subsequently killed/reaped; no repository writes.
- **Cited:** retained logs report 8 Apalache + 13 TLC unit tests and 9 actual controls passing, including five Kettle properties at bound 63. Full required gate remains pending; actual macOS Temurin execution remains unverified.

### Test Gaps

- Owner process-group termination and supervisor signal handling.
- Actual macOS JVM admission, execution and cleanup.

### Test Budget

- Added test lines remain below 2× implementation additions; existing controls retain assertions and strengthen `NotRun` propagation.

### What's Good

- **Walked:** checked native-depth increment, cut refusal, pinned completion parsing, aggregate trace limits and failure propagation prevent inconclusive runs from becoming safety passes.

### Standards axis

## Quality Audit — Standards axis: fn-211 task 2

### Summary

- Files changed: 28 · Should Fix 0 · Consider 0 · Blocking: none possible (standards axis).
- No actionable standards findings in `0e88ae54..5f4d62e9`.

### What's Good

- **Walked:** Java admission and subprocess supervision share TLC’s existing owner; Apalache adds no parallel orchestration framework or third-party dependency.
- **Executed:** No changed file crosses 1,000 physical lines; all 38 inventory paths are unique and their recorded hashes match.

### Verification Limits

- **Cited:** Retained logs report 8 Apalache tests, 13 TLC tests, 6 installer tests, and strict clippy passing. This audit did not rerun tests or the full gate.
- **Cited:** `platform-boundary.md` explicitly limits Darwin evidence to policy wiring tested on Linux. Native macOS execution and the final required model-time gate remain pending.

Correctness axis: 1 finding; worst tier Should Fix.
Standards axis: 0 findings; worst tier none.

