---
satisfies: [R11]
---
# fn-93-simplify-the-lean-model.39 Scenario takes machine: and an optional starts: (F3, F4)

## Description
Lane F3 and F4. `scenario` (~`Syntax.lean:555-573`) takes `machine:` like `property`; `model:` on `scenario` is rejected with a located error naming `machine:` (copy `property`'s `retiredModelKeyMessage` pattern ~504-509). `starts:` is optional when the machine declares exactly one start state; with two or more, omitting it is a located error listing them. Rewrite the 37 scenarios (18 in `Success/Tests`, 7 in `Caller/Model`, 7 in AUTHORING.md ~617-668, rest elsewhere) and update the quoted AUTHORING.md regions in the same commit.

**Size:** M
**Files:** `model/Umpire/Command/Syntax.lean`, `model/Umpire/Command/Tests/**`, `model/Temporal/Feature/Nexus/Success/Tests.lean`, `model/Temporal/Feature/Nexus/Caller/Model.lean`, other scenario sites, `model/AUTHORING.md` (scenarios block, prose ~612-615), vocabulary gate only if a compound token is retired
**Touches:** [model/Umpire/Command/Syntax.lean, model/Umpire/Command/Tests/**, model/Temporal/Feature/**, model/AUTHORING.md]
**Depends on other specs:** fn-92.2 rewrites `starts:` resolution; re-read at start.

### Approach
- Check Go-side authoring checks (`make umpire-check-testpilot-authoring`, `tools/umpire/authoring/**`) for quoted `scenario … model:` text before renaming.
- Pin both new errors with `#guard_msgs`; they are the spec's two deliberate new texts.
- A `#guard` per shorthand compares records with the long form.

### Investigation targets
**Required:**
- `model/Umpire/Command/Syntax.lean:495-575`
- `model/AUTHORING.md:600-670`
- `tools/umpire/authoring/drift_test.go`

### Quick commands
```sh
cd model && lake build
go test ./tools/umpire/authoring/...
make umpire-check-goldens umpire-check-regression
```

## Acceptance
- [ ] `scenario` takes `machine:`; `model:` rejected with a pinned located error naming `machine:`
- [ ] `starts:` optional with one start state; ambiguous omission is a pinned located error
- [ ] Every scenario rewritten; AUTHORING.md drift test green; IDs and fixtures byte-identical


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
