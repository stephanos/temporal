---
satisfies: [R4]
---
# fn-233-kettle-service-a-framework-owned-end-to.4 Seeded server mutants and the detection golden

## Description
Wire the seeded defects: `KETTLE_MUTANT` selects exactly one of the enumerated mutants in `mutants.go` (at least: accept an over-long kettle ID, report Boiled before RespondHeating, silently stop rejecting a field, return a wrong status code). `kettle_test.go` runs the live driver once per mutant and asserts the framework names it: an `Accepted` or `Mismatch` in `gaps.txt`, an `Unexplained` from Judge, a differential actual-server gap finding. `expected/mutants.snap.txt` pins mutant → detecting driver → finding.

**Size:** M
**Touches:** `model/umpire/hosts/go/reference/kettle/*.go` (mutant switches and server controls), `model/umpire/reference/kettle/expected/mutants.snap.txt`, `model/umpire/hosts/go/kettle_test.go`, `model/README.md`

### Approach
Each mutant is a one-line switch in the server, not a fork. The test enumerates the mutant list from the server package so a mutant without a test entry fails, and an undetected mutant fails naming the mutant and the driver that should have caught it.

### Differential detection acceptance

Follow the parent's mutant oracle contract. Each mutant uses identical healthy-versus-mutant witness input against fresh server/guest instances. Invalid requests must reach both actual servers despite recorded Annotate violations; only differing actual response classification counts. Removed validation maps to Accepted against a declared rejection; wrong code maps to Mismatch; premature Boiled maps to current-only Judge Unexplained. A no-op mutant, missing test registration, unexpected transport failure or unknown mutant must fail. Use per-server Options in tests, never shared environment mutation. Pin structured findings and name-set equality before golden comparison. Quick: focused Go healthy/mutant tests; final: make model-check.
## Acceptance
- Every enumerated mutant is detected and pinned in `mutants.snap.txt` with its driver and finding (R4).
- An undetected mutant fails naming mutant and driver; a mutant missing from the test fails (R4).
- Mutant runs are part of the Go host test target; `make model-check` green; README gains a short "Kettle service" paragraph.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

