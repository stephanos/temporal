---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R8, R9, R10, R11, R12]
---
# fn-154-gomad-simpler-virtual-network-with.12 Reconcile network contracts and verify the frozen integrated candidate

## Description
Reconcile network contracts and verify the frozen integrated candidate (R1, R2, R3, R4, R5, R6, R7, R8, R9, R10, R11, R12). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3/README.md`, `tools/gomad3/SPEC.md`, `tools/gomad3/ARCHITECTURE.md`, `tools/gomad3/architecture_test.go`, `tools/gomad3/deterministicio/profile.go`, `tools/gomad3/toolchain/version/version.json`
**Touches:** [tools/gomad3/README.md, tools/gomad3/SPEC.md, tools/gomad3/ARCHITECTURE.md, tools/gomad3/architecture_test.go, tools/gomad3/deterministicio/profile.go, tools/gomad3/toolchain/version/version.json, tools/gomad3/internal/compatibilitypack/**, tools/gomad3/qualification/**, tools/gomad3/choice/internal/wire/wire_generated.go, tools/gomad3/target/internal/livecap/protocol_generated.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadchoicewire/**, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadlivecap/**]

### Approach

- Update contracts for directed holding, byte bounds, explicit defaults/zero, strict threshold/ties, new dial timeout, graceful EOF/discard and replay migration/waiver. Remove the architecture promise that partitions leave queued deliveries untouched.
- Review affected protocol/toolchain/profile implementation identities against actual generator input lists. Refresh only changed final-input identities/approvals/pins; preserve unrelated canonical bytes and deferred native wording.
- Freeze the integrated source. Run one full host gate, focused runtime/model regressions, generated validation, package architecture, canonical lint and both-source-set static checks. Bind source/tool inputs, commands, results and preserved behavior in one bounded handover.
- Root obtains a fresh independent source review and spec-completion audit, handles Flow/MILESTONES and commits. Newly introduced native gates remain with this owner when unavailable; inherited fn-128/fn-149 gates stay deferred. No planning SHIP or portable test supplies runtime acceptance.

### Investigation targets

**Required:**

- `tools/gomad3/README.md:1233`
- `tools/gomad3/SPEC.md:342`
- `tools/gomad3/ARCHITECTURE.md:77`
- `tools/gomad3/architecture_test.go`
- `tools/gomad3/deterministicio/profile.go:17`
- `tools/gomad3/toolchain/version/version.json`

### Quick commands

```bash
make -C tools/gomad3 validate
(cd tools/gomad3 && go test -tags test_dep -run '^Test(PackageArchitecture|PublicPackagesDoNotExportTypeAliases|PureModulesHaveNoHostEffects|ArchitecturePublicSignatureFixtures|ExactModuleEdges|DomainModulesDoNotExportWireFraming|PublicPackagesDoNotExportForwardingAliases)$' .)
make lint-code-fast
make -C tools/gomad3 lint-code
GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host
make -C tools/gomad3 overlay-test test-simulation
```

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

## Acceptance
- [ ] Current contracts state every intentional behavior/format/replay guarantee change and preserved boundary.
- [ ] Final generated inventory, paired identities, lint, architecture, ordinary host-source coverage and both-source-set static checks have candidate-bound evidence.
- [ ] Required supported runtime/overlay/simulation/framed controls and independent integrated source review pass, or exact missing acceptance remains open.
- [ ] Root verifies all twelve RIDs and standing criteria before completion; no inherited deferral or source checkpoint is represented as new native proof.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
