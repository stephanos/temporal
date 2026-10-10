---
satisfies: [R2, R10]
---
# fn-155-gomad-syscall-level-io-boundary-from.5 Run the determinism soak with the boundary on for single-process and multi-node workloads

## Description
Add the boundary and adapter exclusion to soak grouping and run the existing soak over the single-process and multi-node workloads on the platform in use.

**Size:** M
**Files:** `qualification/soak/soak.go`, `qualification/soak/manifest.go`, `tools/gomad3integration/qualification/soak.json` (or a boundary soak manifest), root `Makefile` soak target if a new manifest is added; retained soak report under `.flow/artifacts/fn-155-gomad-syscall-level-io-boundary-from/`.
**Touches:** [tools/gomad3/qualification/soak/**, tools/gomad3integration/qualification/**, Makefile, .flow/artifacts/fn-155-gomad-syscall-level-io-boundary-from/soak/**]

### Approach
- Include the I/O boundary and excluded adapters in soak batch keys alongside workload, seed, platform and build key (fn-112 soak; spec-scout note).
- Use the existing soak defaults (seeds, repeat, batches from `soak.json`); first divergence is reported by `classifyBatch` (`soak.go:504-520`) and `writeDiffer` (`:619`).
- Record the other qualified platform's proof as deferred to fn-128 or fn-149.

## Acceptance
- [ ] Soak keys include the boundary and adapter exclusion; a test shows runs with different boundaries never share a batch.
- [ ] The soak passes for the single-process and multi-node workloads on the platform in use, or any divergence is reported with its first differing event.
- [ ] The soak report is retained with its platform and toolchain identity; the other platform is recorded as deferred.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
