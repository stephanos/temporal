### Correctness axis

Correctness addendum for `0eda2a2669`: the original Should Fix is resolved. No new correctness findings.

- **Walked:** `process.rs:183` isolates the supervisor with `process_group(0)`; `process.py:29–35` installs latch-only handlers before launch and completes cleanup before returning `128 + signal`.
- **Reproduced:** the previously failing owner-group termination now removes both backend and escaped-session descendant PIDs before harness cleanup. Independent owner-group TERM/KILL and direct-supervisor TERM checks passed.
- **Executed:** direct TERM/INT/HUP/QUIT returned 143/130/129/131 respectively, with backend PIDs absent.
- **Cited:** retained focused tests, strict clippy and post-correction actual Apalache/TLC controls pass.

All isolated test processes were cleaned/reaped; repository source remained unchanged. The final required gate is pending. Bound-63 evidence remains from the prior run; actual macOS JVM execution remains unverified.

### Standards axis

### Standards addendum — `5f4d62e9..0eda2a2669`

- New findings: **0 Should Fix, 0 Consider**.
- **Walked:** Signal handling and process-group isolation remain within the shared supervisor. Regression assertions check backend PID disappearance before harness cleanup.
- **Executed:** Updated inventory hashes match; no file newly crosses 1,000 physical lines.
- **Cited:** Retained logs report three Python controls and strict clippy passing. No tests rerun during this read-only audit; native macOS execution and the final required gate remain pending.

Correctness axis: 0 unresolved, 0 new; original Should Fix resolved.
Standards axis: 0 new findings; worst tier none.

