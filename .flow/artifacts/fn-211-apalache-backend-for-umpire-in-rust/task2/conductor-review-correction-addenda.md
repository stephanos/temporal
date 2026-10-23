### Correctness axis

Second correctness addendum for `c9d72c6ef6`. Earlier reports remain unchanged.

The two accepted R2 defects are addressed in source: retained file handles replace log-path reopening; captures share a bounded byte budget; Apalache receives diagnostics through `ProcessFailure`; pipe-drain joins are removed. **Walked**. Retained focused Linux tests and clippy pass. **Cited**.

### Should Fix

- **model/umpire/backends/tlc/src/process.rs:324** (Conf 75): The unconditional descendant-timeout test conflicts with universal Darwin no-fork enforcement. Its `subprocess.Popen` fails before `sleep(10)`, producing `Ok(nonzero_status, output)` and failing `.is_err()` — restrict descendant-creation proof to Linux, retaining separate Darwin prevention and platform-neutral direct-helper timeout assertions.

**Executed:** the exact helper under simulated Darwin NPROC0 on Linux exited 1 with `BlockingIOError` after 0.068 seconds. This supports the traced contradiction; it is not native macOS evidence.

No other actionable findings in this delta. Actual macOS execution remains unverified, the final required gate remains pending, and bound-63 evidence remains from the prior run. No repository writes.

### Standards axis

### Standards addendum — `0eda2a2669..c9d72c6ef6`

- New standards findings: **0 Should Fix, 0 Consider**.
- **Walked:** Output capture remains in the shared process owner; `ProcessFailure` carries bounded diagnostics without reopening tool-controlled paths. The obsolete JVM policy path is removed.
- **Executed:** All 38 inventory hashes match; no file newly crosses 1,000 physical lines.

> Out-of-axis observation: **model/scripts/tests/test_backend_process.py:96** — the test accepts `TimeoutExpired` when diagnostic text exists, so a hung preflight can pass; fail explicitly on timeout.

**Limits:** Retained logs are **cited**, not rerun. Native macOS execution and the final required gate remain pending; the Linux NPROC0 mise abort is explicitly recorded as inconclusive for Darwin.

Correctness axis: 1 finding; worst tier Should Fix.
Standards axis: 0 standards findings; worst tier none. Its out-of-axis timeout observation is accepted by the conductor as an additional correctness correction.

