# Gomad

## Read before work

- Read [README.md](README.md), [SPEC.md](SPEC.md), and [ARCHITECTURE.md](ARCHITECTURE.md) for the current contracts.
- For backend changes, also read the [WASM README](internal/backends/wasm/README.md) and [qualification contract](internal/backends/wasm/QUALIFICATION.md).

## Rules

- Use WASM for guest execution.
- Keep execution and replay deterministic.
- Authenticate retained evidence.
- Keep resource limits explicit.
- Prefer typed contracts and shared validators.
- Test real boundaries before expensive runs.
- Match completion claims to execution evidence.

## Verification

Bind determinism claims to retained evidence for the exact candidate, host, profile, and workload.
Final qualification and independent receiving follow fn-246 and its prerequisites in [MILESTONES.md](../../MILESTONES.md).
