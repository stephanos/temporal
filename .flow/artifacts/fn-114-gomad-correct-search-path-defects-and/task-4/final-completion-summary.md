Implemented fn-114.4 (R4): retain validated forced-prefix replay divergence, commit completed siblings, honor first/budget/all policies, preserve HostError boundaries, and validate retained evidence on inspection and resume. Divergence does not invent an outcome, failure signature, trace, or children. Schema/controller identity changed; older segments fail visibly.

The 45-package full host gate passed on darwin/arm64 in 173.2 seconds, with validate, focused vet, architecture, formatting, and source whitespace checks. The independent review found an observation-boundary gap; two regressions reproduced it, and the fix plus divergence/architecture checks passed in 6.13 seconds. Details: handover-summary.md and review-fix-summary.md. Linux remains unverified; root lint has the documented missing-main/nested-module discovery limitations.

Independent re-review: SHIP, gpt-6-astra high, 2026-10-02T15:17:40.806229Z, session 01a0fd25-c019-7121-af47-bb64c0979f8d. No remaining findings. Implementation commit: e153d05a76083b797525eca6bb685c78f5b623be.

stage: implementation review complete; plan-sync skipped (disabled); tracker sync inactive; sequential shared checkout (worktrees prohibited).
