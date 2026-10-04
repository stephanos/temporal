---
satisfies: [R5]
---
# fn-125-represent-dynamic-configuration-in-the.4 Export settings as Quint constants and P params, or refuse

## Description
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
