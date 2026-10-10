---
satisfies: [R1, R10]
---
# fn-152-gomad-runner-storage-on-one-append-only.1 Add protected append-log framing and separate replay from writer repair

## Description
Implement R1/R10 in the shared Runner-private log owner.

**Size:** M
**Files:** Core files (3-5 new owner/test files): runner/internal/logstore frame, replay, writer and fault tests. Wider touches: a narrow guarded writable-open/descriptor-lock seam and its tests in internal/hostfs; internal/gomadtool/architecture owner and edge inventory only if required. Framing and protected writer lifecycle form one owner; split with the conductor if the surviving seam exceeds M scope.
**Touches:** [tools/gomad3/runner/internal/logstore/**, tools/gomad3/internal/hostfs/**, tools/gomad3/internal/gomadtool/architecture/**, tools/gomad3/architecture_test.go]

### Approach

Build one Runner-private logstore owner using stdlib JSON, checksums and hostfs locking. Read-only replay captures EOF on an opened file, validates a bounded prefix without acquiring the writer lock, and reports observed tail damage. Exclusive writer-open replays, truncates a recognized final torn frame and syncs before appending. Protect the fixed frame header, including its length and kind, before trusting length arithmetic; a complete invalid header is corruption. Only an independently framed final payload checksum failure or short final frame qualifies for tail repair. Strictly decode known typed payloads and validate complete transactions before applying them. Any partial-write or uncertain sync result poisons the writer until reopen. The guarantees cover append interruption and the defined damage controls, not arbitrary coherent rewrites or checksum collisions.

Port the campaign pinned-root and hostfs.openRegular validation into read and writable opens. Validate the private root and regular single-link log/lock metadata; bind the lock to the opened store/log identity and reject detected replacement before append or truncation. Use held descriptors and root-relative operations, not unchecked pathname reopen. Existing hostfs.Try alone is insufficient: extend only the narrow guarded writable-open/lock seam needed by logstore, preserving other callers. Readers create or modify no object; creation sets private modes only on newly owned objects. Close every acquired handle on rejection.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/runner/internal/campaign/filesystem.go:18`
- `tools/gomad3/runner/internal/campaign/filesystem_fault_test.go:15`
- `tools/gomad3/runner/internal/campaign/open_campaign.go:35`
- `tools/gomad3/runner/internal/campaign/files.go:19`
- `tools/gomad3/internal/hostfs/open.go:23`
- `tools/gomad3/internal/hostfs/open_test.go:31`
- `tools/gomad3/internal/hostfs/open_unix_test.go:12`
- `tools/gomad3/internal/hostfs/lock_unix.go:16`
- `tools/gomad3/internal/gomadtool/architecture/edges.go:54`
- `tools/gomad3/internal/gomadtool/architecture/architecture.go:282`
- `tools/gomad3/internal/gomadtool/architecture/architecture_test.go:202`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/logstore ./internal/hostfs.

Follow the parent spec's Delivery and verification section. Retain exact selectors and current command scope; do not substitute portable coverage for supported-host evidence. If deletion/fixture migration expands the surviving implementation beyond this cohesive owner, stop for conductor scope splitting before implementation.

## Acceptance
- [ ] R1/R10: durable append, strict typed replay and close work with one writer and lock-free readers; a second writer reports in-use.
- [ ] R1: validated header length/kind, checked arithmetic and owner frame bounds prevent a corrupt prefix from being misclassified as a large final torn record; earlier payload damage and valid-framed invalid JSON/state fail as corruption.
- [ ] R1: read-only replay never truncates; writer-open truncates and syncs only the recognized final tail, and reports observed/repaired bytes separately. Reader reports an observed prefix without claiming every complete frame's fsync acknowledged.
- [ ] R1: exhaustive partial-write, before/after-sync, truncate and directory-sync injection preserves every acknowledged commit; an uncertain writer rejects further appends until reopened, and no consumer half-applies a record.
- [ ] R1: reject symlink roots/logs/locks, hard-linked log/lock aliases, nonregular files and wrong modes without mutation. Deterministically inject root, log and lock replacement during open and before append/repair; rejection preserves store and external sentinel contents/modes. Accepted path aliases to one store cannot acquire independent writer ownership. Lock-free readers create or modify no object.
- [ ] R10: stdlib-only dependencies and architecture checks retain campaign/corpus separation and pure exploration engines.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
