---
satisfies: [R3]
---
# fn-257-extract-the-standalone-umpire.3 Prepare the portable standalone root and canonical Go identities

## Description
Adapt the exported independent workspace without rebuilding its already local gate.

**Size:** M
**Files:** exported README/docs/AGENTS/LICENSE/Makefile/CI, Go manifests/imports and protocol schema options/generated descriptors.
**Touches:** [../umpire/**, .flow/artifacts/fn-257-extract-the-standalone-umpire/**]

### Approach
- Copy root LICENSE and write focused agent guidance, command aliases and CI using existing cargo xtask gate/fmt/update/time/validate. Preserve Java for TLC and all current pins/configuration.
- Use github.com/temporalio/umpire/hosts/go and github.com/temporalio/umpire/reference/kettle consistently across module declarations, imports, replacements and go_package options. Regenerate core and Kettle descriptors together when generation is admitted; preserve wire names/tags and ABI.
- Repair root-relative documentation and preserve selected DESIGN/GUEST_BUILDS decisions and measurement provenance through focused text/revision links. Remove historical provenance copies in a distinct adaptation commit.
- Publish the exact consumer-facing interface map for task 4.

### Investigation targets
**Required:** model/umpire/README.md; model/umpire/hosts/go/go.mod; model/umpire/reference/kettle/go.mod; model/umpire/scripts/proto-go.sh; model/umpire/protocol/proto/README.md.

### Verification
Source preparation may proceed during the hold. Preserve task 2's source checkpoint and semantic golden manifest; execution/protobuf regeneration evidence is not fabricated while deferred.

## Acceptance
- [ ] Standalone entry points, attribution, guidance and CI target the existing framework-owned tooling.
- [ ] Both Go identities and schema options are coherent, and generated-code work is completed and checked before final admission.
- [ ] Interface and artifact ownership maps are ready for actual consumers; semantic payloads stay unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
