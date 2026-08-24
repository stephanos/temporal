---
satisfies: [R4, R7]
---
# fn-2-agentworkflow-configuration-and-cli.7 Restore legacy single-dash CLI flag compatibility

## Description
Resolve the spec-wide implementation review P1 by preserving the former Go flag parser single-dash long-form syntax across the Cobra command tree. Normalize legacy `-name value` and `-name=value` forms before Cobra parses arguments while retaining current double-dash flags, help, positional validation, stream isolation, and usage-error classification. **Size:** S **Touches:** [tools/agentworkflow/internal/cli/cli.go, tools/agentworkflow/internal/cli/cli_test.go]

## Acceptance
- [ ] Compatibility tests first fail for legacy single-dash long flags and equals forms.
- [ ] Every command accepts its former single-dash long flag spellings without introducing global Cobra state.
- [ ] Double-dash flags, unknown-flag usage errors, positionals, output streams, and exit codes remain compatible.
- [ ] Full tagged module tests/build and task-scoped configured lint pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
