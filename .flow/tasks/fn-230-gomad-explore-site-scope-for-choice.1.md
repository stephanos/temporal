---
satisfies: [R1, R3, R4, R5, R6, R9]
---
# fn-230-gomad-explore-site-scope-for-choice.1 Site scope flag, pattern validation and resolution against the base trace

Touches: tools/gomad/internal/cli/explore.go, cli/plan.go, tools/gomad/internal/runner/options/, tools/gomad/internal/runner/internal/exploration/site_scope.go, CLI.md

## Description
Add `--site-scope` to explore and plan for `--strategy=choice-exploration` only. Validate patterns (import path grammar, optional `/...` suffix, non-empty, no whitespace) at parse time with status 2. After the base execution, symbolize every site in its trace via the symbolize package, resolve patterns to a fingerprint set, and store `{patterns, resolved_sha256, resolved_count}` in the plan payload and campaign record; a module without symbols fails with status 2 before execution; an empty resolved set stops round one with `site_scope_empty` after publishing the base artifact. Plan validation refuses a malformed scope block on open.

## Acceptance
Invalid patterns and wrong strategies exit 2 with named reasons; a stripped module exits 2 before any execution; an empty scope publishes the base artifact and stops with `site_scope_empty`; the plan and campaign record carry the scope block; without the flag the plan bytes are unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
