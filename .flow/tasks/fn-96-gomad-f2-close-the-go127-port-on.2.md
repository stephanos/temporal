---
satisfies: [R4]
---
# fn-96-gomad-f2-close-the-go127-port-on.2 Attempt the DTrace clock audit and record the outcome

## Description
Run `make -C tools/gomad3 clock-audit` if root is available non-interactively; otherwise record that it did not run.

## Acceptance
- milestone status states whether the audit ran and its result

## Done summary
The DTrace clock audit did not run: the session has no root (`sudo -n true` fails with "a password is required"). `make -C tools/gomad3 clock-audit` passed its generator and validation checks and found the darwin/arm64 toolchain ready (key 85c444f9...). It stopped with exit 2 at `gomad3 clock audit requires root DTrace privileges`. That confirms the fix in task .1 (68d36aadfe): the audit is now blocked only by privileges. For the next step, a person with root runs `sudo make -C tools/gomad3 clock-audit`. The upgrade dossier keeps `qualified=false` for the clock audit until then. The spec says this does not fail the milestone. Task .4 records it in the milestone status. This task made no code changes.

stage: impl-review - ran [2026-09-27] codex fan-out (3 draws, all SHIP) on empty diff 6bcbf42ebb..HEAD
## Evidence
- Commits:
- Tests: sudo -n true -> 'sudo: a password is required' (rc=1), make -C tools/gomad3 clock-audit -> rc=2; toolchain ready (darwin/arm64, key 85c444f98ab905a782983191f7d47dd4ce9687a606051d1c87f9ffdf71a39f16); stopped at 'gomad3 clock audit requires root DTrace privileges', baseline: none (spec defines no Quick commands; outcome-recording task, empty diff)
- PRs: