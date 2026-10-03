---
satisfies: [R7]
---
# fn-122-capabilities-and-their-laws.5 Add the law lint kinds and write waiver reasons into the accepted-findings file

## Description
Connect the law sidecar to fn-120's accepted-findings file and add the four law lint kinds, so intended gaps have one source and the gate fails on a waiver that names no law.

**Cross-spec entry gate:** start only after fn-120.3 (model lint, accepted-findings file) is done; fn-120.3 itself waits for fn-114 to close. Verify with `flowctl show fn-120-adopt-what-quint-does-well-named.3`.

**Size:** M
**Files:** `tools/umpire/model/laws.go` (new: sidecar reader) and `tools/umpire/model/lint*.go` (four kinds: law waived with no reason, reason naming no law, capability parameter with no citation, law with one instantiating machine); the gate step that forwards sidecar waivers into the accepted-findings file in fn-120.3's format; lint fixtures (one triggering, one not, per kind); `model/gate/**` wiring.
**Touches:** [tools/umpire/model/**, model/gate/**, model/ir/**]

### Approach
- Reuse fn-120.3's finding record and acceptance file; add kinds, never a second acceptance file. Each `except`/`overriding` reason in a sidecar becomes an acceptance entry keyed by `<machine>.<law>`; lint fails on an acceptance whose law the sidecar's catalog list no longer brings.
- Citation presence, `promises`/`doesNotPromise`, positions and the catalog's instantiating machines are all read from the sidecar (spec Architecture "The law sidecar"); the IR is not changed and no text is parsed from Scala.
- One-instance detection counts instantiating machines per law across all sidecars the gate reads, with the spec's one definition of an instantiating entity.

### Investigation targets
**Required:**
- `.flow/tasks/fn-120-adopt-what-quint-does-well-named.3.md` and the lint command it produced
- `model/ir/*.laws.json` (post tasks 3 and 4) - the sidecar shape
- `tools/umpire/model/checking.go` - reader indexes to reuse
**Optional:**
- `model/README.md` lint section (post fn-120.3)

### Quick commands
```bash
go test -count=1 -tags test_dep ./tools/umpire/model/...
make umpire-check-model
```

### Execution constraints
- No IR schema change; no Case change.
## Acceptance
- [ ] Each `except`/`overriding` reason appears in the accepted-findings file keyed by `<machine>.<law>`, forwarded from the sidecar; a reason naming no catalog law fails the gate.
- [ ] The four lint kinds report kind, machine, message and Scala position read from the sidecar, each with a triggering and a non-triggering fixture; a malformed sidecar is a reader error with no findings.
- [ ] The gate runs lint over every IR file; first-run findings and what was done about each are in the done summary.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
