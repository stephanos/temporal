# fn-27-hermetic-ci-execution-and-qualification.4 Prove Artifact Checksum and Behavior Fingerprint parity

## Description
Add independent golden and mutation checks proving the local and CI subjects have identical v2 bytes, Artifact Checksum, Definition IDs, and Behavior Fingerprints. Allow only declared runtime transport identities to differ; reject semantic recompilation, format crossing, incomplete closure, and nondeterministic generation.

## Acceptance
- [ ] Stable Artifact and model identities are identical across local and CI use.
- [ ] Each representative byte, checksum, fingerprint, version, and closure mutation is rejected.
- [ ] Tests do not share the production checksum/fingerprint oracle.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
