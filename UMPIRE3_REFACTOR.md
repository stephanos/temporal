---
status: complete
---

# Umpire3 structural refactor

The structural refactor is complete. There are no remaining implementation milestones in this
plan; future product work belongs in `UMPIRE_LEAN2.md` or a new focused plan.

## End state

- `scenario` owns sparse authoring, generated domain vocabulary, compilation, enumeration, and
  explanations.
- `execution` owns the Environment seam, prepared Environment identity, lifecycle, evidence
  qualification, cleanup, and Results.
- `profile` is the sole deployment-authority module. Environment adapters may narrow its authority
  and capabilities but cannot broaden them.
- `temporal` owns Temporal client, worker, SDK participant adapter, public-history observation, and
  exact-once lifecycle behavior. `participant` retains only participant-program semantics and its
  `Runner` port.
- `replay` owns replay bundles, strict encoding and decoding, redaction, corpus persistence,
  reproduction, and drift classification. The external `umpire3/replay-bundle/v1` format is
  unchanged.
- `campaign.Run` owns candidate or mutation selection, execution, exact-identity minimization,
  bundle capture, replay, and ordinary regression promotion. The mutation gate validates this
  canonical report.
- `process.Run` and `Supervisor` use the same bounded subprocess-attempt implementation, including
  process-group termination and CPU, memory, timeout, and output enforcement.
- `internal/command` owns supported command dispatch, bounded file I/O, diagnostics, connection
  flags, and dependency injection. `umpire3-run` and `umpire3-qualify` are thin compatibility
  entry points.
- Layout tests enforce the intended public modules and import directions. Removed packages have no
  forwarding compatibility wrappers.

## Preserved contracts

- Lean remains the semantic authority; no Umpire2 implementation is imported.
- Existing JSON versions, field names, replay bytes, redaction, file permissions, CLI flags, and
  qualification meaning remain stable.
- Generated semantic identifiers and external compiler/runtime version strings remain stable even
  though their Go implementation modules moved.
- Umpire2 and Umpire3 root tests remain independent side-by-side copies.

## Verification

Run from the repository root:

```sh
go test -count=1 -tags test_dep ./tests/umpire3/...
go test -count=1 -tags test_dep ./tests -run '^TestUmpire3' -timeout 20m
make umpire3-check
make lint-code
```
