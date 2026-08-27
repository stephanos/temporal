# fn-27-hermetic-ci-execution-and-qualification.6 Expose the ordinary pinned CI test command

## Description
Wire the portability proof into the ordinary pinned Go test command and aggregate repository gate. Workflow configuration delegates to that command with read-only permissions and pinned actions; it does not enumerate semantic checks, accept secrets/OIDC, or expose a second Umpire CI runner or Claim Assessment command.

## Acceptance
- [ ] The ordinary test command is the sole public CI execution surface.
- [ ] Workflow actions and toolchains are pinned with minimal read-only permissions.
- [ ] No semantic flags, profile selector, credentials, cache authority, or custom policy command is added.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
