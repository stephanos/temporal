---
satisfies: [R15]
---
# fn-93-simplify-the-lean-model.42 One README and one ARCHITECTURE for model/ (G2)

## Description
Lane G2. Merge `model/README.md` (525), `model/ARCHITECTURE.md` (378) and `model/Umpire/ARCHITECTURE.md` (416) into one README (build, regenerate, run: ~200 lines) and one ARCHITECTURE (~450); write the Go runtime boundary once, pointing at `common/testing/testpilot/*/README.md`; drop the three "Superseded runtime history" sections. Every reader follows the text.

**Size:** M
**Files:** the three docs; `tools/umpire/regression/ci_workflow_test.go:214-228` (re-key the 9 pinned fragments: "testpilot.Prepare(case, Profile)", "Temporal authority remains split", "complete twelve-file conformance tree", "The Testpilot `.proto` files own the Case protocol", "`common/testing/testpilot` owns the Profile/Driver contract", "checks deadline expiry before every transition", "Case, Program, Contract, and Run vocabularies are finite, versioned, and bounded", "Promotion remains generic and review-only", plus the ninth at start); `tools/umpire/internal/retiredvocabulary/check.go:59-65` (`requiredFiles`) and `check_test.go`, `tools/umpire/vocabulary/retired_vocabulary_test.go:209`; `Makefile:976` (README grep); `.plans/lean/LEAN_GUIDELINES.md:26,32,198`, `.plans/UMPIRE4_RESEARCH.md:6`, other inbound anchors; `.plans/index.json` `allowedMissingLinks` for `model/Umpire/ARCHITECTURE.md` inbound links
**Touches:** [model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, tools/umpire/regression/**, tools/umpire/internal/retiredvocabulary/**, tools/umpire/vocabulary/**, Makefile, .plans/lean/LEAN_GUIDELINES.md, .plans/UMPIRE4_RESEARCH.md, .plans/*.md, .plans/index.json]
**Depends on other specs:** fn-88.7, fn-89.6, fn-92.6 edit these docs; fn-94 may edit `common/testing/testpilot/*/README.md`, whose fragments `ci_workflow_test.go:200-213` also pins — do not edit those READMEs here.

### Approach
- Every pinned fragment moves with its key; none is dropped (R15).
- Anchors: list inbound links to the three files (`grep -rn 'model/README.md\|model/ARCHITECTURE.md\|model/Umpire/ARCHITECTURE.md' .plans docs tools`), retarget or add `allowedMissingLinks` with reasons; `go run ./tools/planindex`.

### Quick commands
```sh
go test ./tools/umpire/regression/... ./tools/umpire/internal/retiredvocabulary/... ./tools/umpire/vocabulary/...
make umpire-check-plan-index umpire-check-retired-vocabulary
wc -l model/README.md model/ARCHITECTURE.md
```

## Acceptance
- [ ] One README and one ARCHITECTURE in `model/`; `model/Umpire/ARCHITECTURE.md` gone; no "Superseded runtime history"
- [ ] `ci_workflow_test.go`, `requiredFiles` and its tests, the Makefile grep and the plan-index check green against the moved text
- [ ] Markdown in `model/` measured for R12


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
