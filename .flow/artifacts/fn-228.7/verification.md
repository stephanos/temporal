# Runner and Run documentation checkpoint verification

Baseline: none. Task .7 and its parent spec define no Quick commands. Pre-edit HEAD was
`62fc6611eeaa2b39330263c2e5c4fa9eb25c6a55`; it is persisted in `.flow/tmp/base_commit`.
No executable source, schema, Model, worker, test, gate or MILESTONES.md changed.
The current executable source remains `fad276dfc399766a28b51d4262829f90be6e73b6`.

`python3 .flow/tmp/fn2287-check.py` exited 0 after comparing 24 direct-package non-test
Go line counts with the documentation inventory, checking added local links/anchors,
checking active crate spellings and hashing the committed guest. Counts use newline
count (`wc -l`) including comments/blanks and excluding `_test.go`; generated packages
are separate. The first check caught an inherited helper error: CLI is 61 lines,
not 60. The documentation was corrected; original 1,759/3,205 target totals did not change.
`git diff --check` exited 0. The local checking script/log are retained in `.flow/tmp`.

The Flow classifier returned exit 1, `FULL: unmatched: MIGRATE.md`. Its conservative
path classifier does not recognize that root document as documentation. This is not a
tier-B pass and no GATE_SKIPPED line or green receipt is minted. Manual scope inspection
shows only DESIGN, README, MIGRATE and this evidence document; the explicitly authorized
documentation-only scope uses these bounded checks, without repeating expensive Model/live
gates. The primary review receives this limitation. No Quick-command gate is silently
skipped because none is defined. The committed task-2 verification is the attributed
source for existing actual exit-0 Model/live timings; it is not a newly executed gate
or a baseline receipt reused past the roadmap commit. Broad legacy tooling remains red.

Read-only conductor inspection corrected attribution of 2,427 identities to the whole
umbrella, not solely its 437.061s full-Testpilot stage. It supplied no execution credit.

## Independent package recount

```text
   36 common/testing/testpilot/duration
 1183 common/testing/testpilot/temporal/act
  263 common/testing/testpilot/temporal/provision
 1723 model/umpire/hosts/go/runner
  624 model/umpire/hosts/go/host
  315 tools/canary/authority
  213 tools/canary/cmd/umpire-canary
 1821 tools/canary/controller
  237 tools/canary/policy
  162 tools/canary/preflight
  255 tools/canary/recovery
  110 tools/canary/testharness
   61 tools/umpire/cmd/umpire
   43 tools/umpire/temporaldriver
  101 model/temporal/hosts/go/reference/worker
   76 model/temporal/hosts/go/reference/worker/raw
  149 model/temporal/hosts/go/reference/worker/sdk
   33 model/temporal/hosts/go
  915 model/umpire/hosts/go/gen/umpire/annotations/v1
 1464 model/umpire/hosts/go/gen/umpire/model/v1
  342 model/umpire/hosts/go/gen/umpire/realization/v1
 1290 model/umpire/hosts/go/gen/umpire/run/v1
 1834 model/umpire/reference/kettle/gen/kettle/v1
  812 model/temporal/hosts/go/umpire/temporal/v1
All 24 package counts, added local links/anchors and current crate names verified.
Committed guest hash verified.
```

The homologous generated totals are 5,734 → 5,845 for the old generic schema family
(including relocated Kettle), 425 → 812 for Temporal, and 6,159 → 6,657 combined.
Final R40/fn-228 R7 and both parent specs remain open; non-Go execution remains unproven.
