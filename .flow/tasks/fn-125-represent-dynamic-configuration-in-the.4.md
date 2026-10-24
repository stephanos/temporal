---
satisfies: [R5]
---
# fn-125-represent-dynamic-configuration-in-the.4 Export settings as Quint constants and P params, or refuse

## Description
**Recut 2026-10-18 (one-language series):** Owner-deferred. At revival: lowering goes to fn-203, export to fn-206/207 (fn-205 deferred), Go model valuations to the Scala checker.

Implements R5. The Quint export writes a setting as a `const` bound per instance (one instance per valuation); the P export writes it as a `param` or refuses with `UnsupportedError` at the Scala position.

**Cross-spec entry gate:** fn-114 closed; not concurrent with fn-124.8. Depends on task 3.

**Size:** S
**Files:** `tools/umpire/export/{quint,p,agreement,encode}.go` and tests; the fixture machine from task 2.
**Touches:** [tools/umpire/export/**]

### Approach
- Quint: `const` per setting, an instance per valuation; the agreement check (`agreement.go`) passes on the fixture machine that reads one setting.
- P: `param` with the domain, or `UnsupportedError` carrying the Scala position.
- Machines that read no setting export byte-identically.

### Investigation targets
**Required:**
- `tools/umpire/export/quint.go`, `p.go`, `agreement.go`, `quint_test.go`, `p_test.go`
**Optional:**
- `tools/umpire/export/README.md`

### Quick commands
```bash
go test -count=1 -tags test_dep ./tools/umpire/export/...
```

### Execution constraints
- Existing exports are unchanged.
## Acceptance
- [ ] The Quint export carries settings as constants per valuation, and the agreement check passes on a fixture machine that reads one.
- [ ] The P export carries them as params or refuses with an `UnsupportedError` naming the Scala position.
- [ ] Existing exports are unchanged; export tests and `make lint-code-fast` pass.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: