---
satisfies: [R1, R2]
---
# fn-98-gomad-f4-close-the-tests-capability.5 Remove os/exec and os/signal admissions from the modernc libc packs

## Description
Completion review finding (P1): modernc-libc-xsys-v041, -v047 (darwin) and -v047-linux-amd64 admit `import:os/exec` and `import:os/signal` for modernc.org/libc, violating F4's 'no pack admits os/exec' constraint (and the spec's os/exec/os/signal/os/user ban). Rewrite the libc adapter's prepared sources so those imports are gone (the system/popen/signal paths refuse deterministically, like the fx/SDK/otel adapters), regenerate the affected requests/reviews/packs through discover/review/generate (linux pack: regenerate only what can be done from darwin; if the linux pack cannot be regenerated here, record it as needing a linux run and keep its request consistent), add a validation test that rejects any pack admitting os/exec, os/signal, or os/user, and re-run darwin closure analysis of ./tests, compatibility-pack-qualification, core set, and the Temporal set.

## Acceptance
- no pack admits os/exec, os/signal, or os/user; a validation test enforces it
- darwin ./tests closure still 0 blockers; pack qualification, core set, Temporal set pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
