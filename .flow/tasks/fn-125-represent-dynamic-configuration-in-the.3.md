---
satisfies: [R4]
---
# fn-125-represent-dynamic-configuration-in-the.3 Bind settings per Query with under, and key, count and answer one Query per valuation

## Description
Implements R4 (spec Part A). A Query binds settings with `under`: `s := v` fixes one value, `s.each` ranges over the domain. A ranged Query is one Query per valuation, keyed `<query>-<valuation key>` like an action class keyed by its inputs, and checked, answered, counted and lowered on its own.

**Cross-spec entry gate:** fn-114 closed; not concurrent with fn-124.8. Depends on task 2.

**Size:** M
**Files:** `model/umpire/` Query DSL (`under`); `model/lifter/**`; `ir.proto` (`Query.under`, repeated `SettingBinding {setting, values, position}`) and generated Go; `tools/umpire/model/**` and `tools/umpire/checker/**` (per-valuation Query expansion, keys, totals); results, witnesses and receipts (valuation key); `tools/umpire/lower/**` (per-valuation key passes through); `model/SEMANTICS.md` (Query totals multiplier).
**Touches:** [model/umpire/**, model/lifter/**, model/ir/**, proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**, tools/umpire/model/**, tools/umpire/checker/**, tools/umpire/explore/**, tools/umpire/lower/**, model/SEMANTICS.md]

### Approach
- `under` takes fixes and ranges; several bound settings cover the product. `total` is the per-valuation count times the number of valuations.
- A Query must bind every setting its machine reads; it may also bind one the machine does not read (the behavior holds under each value, which is the switch's claim today).
- A Query that binds no setting keeps its key, total, answer and Definition ID.
- Results, witnesses and receipts carry the valuation key; the lowered-Case key is unchanged here (one Case per valuation and the Testpilot side come in task 7).
- Refuse at the Query's line, naming the setting: a setting the machine reads left unbound, a value outside the domain, a setting bound twice.

### Investigation targets
**Required:**
- Query declaration and `total` (fn-112.11) in `model/umpire` and the lifter
- `tools/umpire/model/` Query expansion and keying (action-class keys by input)
- `model/SEMANTICS.md` Query totals section
**Optional:**
- `model/temporal/features/nexuscaller/Queries.scala` (the first ranged Query, task 7)

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir model/cases
make umpire-check-model
go test -count=1 -tags test_dep ./tools/umpire/...
```

### Execution constraints
- Queries that bind no setting are byte-identical in IR, answers, totals, receipts and Cases.

## Acceptance
- [ ] `under` fixes (`:=`) or ranges (`each`) a setting; a ranged Query expands to one Query per valuation keyed `<query>-<valuation key>`, each checked, answered and counted separately; `total` multiplies by the number of valuations.
- [ ] Results, witnesses and receipts carry the valuation key; SEMANTICS states the rule and the totals multiplier.
- [ ] An unbound read setting, a value outside the domain and a setting bound twice are refused at the Query's line, naming the setting.
- [ ] Queries binding no setting are byte-identical; model gate, tooling tests and `make lint-code-fast` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
