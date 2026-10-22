---
satisfies: [R4]
---
# fn-233-kettle-service-a-framework-owned-end-to.4 Seeded server mutants and the detection golden

## Description
Wire the seeded defects: `KETTLE_MUTANT` selects exactly one of the enumerated mutants in `mutants.go` (at least: accept an over-long kettle ID, report Boiled before RespondHeating, silently stop rejecting a field, return a wrong status code). `kettle_test.go` runs the live driver once per mutant and asserts the framework names it: an `Accepted` or `Mismatch` in `gaps.txt`, an `Unexplained` from Judge, a differential actual-server gap finding. `expected/mutants.snap.txt` pins mutant → detecting driver → finding.

**Size:** M
**Touches:** `model/umpire/hosts/go/reference/kettle/*.go` (mutant switches and server controls), `model/umpire/reference/kettle/expected/mutants.snap.txt`, `model/umpire/hosts/go/kettle*_test.go`, `model/README.md`

### Approach
Each mutant is a one-line switch in the server, not a fork. The test enumerates the mutant list from the server package so a mutant without a test entry fails, and an undetected mutant fails naming the mutant and the driver that should have caught it.

### Differential detection acceptance

Follow the parent's mutant oracle contract. Each mutant uses identical healthy-versus-mutant witness input against fresh server/guest instances. Invalid requests must reach both actual servers despite recorded Annotate violations; only differing actual response classification counts. Removed validation maps to Accepted against a declared rejection; wrong code maps to Mismatch; premature Boiled maps to current-only Judge Unexplained. A no-op mutant, missing test registration, unexpected transport failure or unknown mutant must fail. Use per-server Options in tests, never shared environment mutation. Pin structured findings and name-set equality before golden comparison. Quick: focused Go healthy/mutant tests; final: make model-check.
## Acceptance
- Every enumerated mutant is detected and pinned in `mutants.snap.txt` with its driver and finding (R4).
- An undetected mutant fails naming mutant and driver; a mutant missing from the test fails (R4).
- Mutant runs are part of the Go host test target; `make model-check` green; README gains a short "Kettle service" paragraph.


## Done summary
Kettle's four server-local seeded defects now have actual healthy-versus-mutant witnesses and a detection golden. Invalid prepared bytes reach both fresh servers despite annotation predictions: overlong IDs and removed label validation produce framework Accepted; wrong status produces Mismatch; premature actual Describe before RespondHeating produces current-only Judge Unexplained.

R4 controls assert actual responses and framework findings before golden comparison. TestKettleMutants includes healthy-equivalent no-op servers for every driver; TestKettleMutantMissingWitness guards name-set equality, TestKettleMutantBrokenTransport performs an actual canceled RPC without turning failure into a finding, and TestServeRejectsUnknownMutant refuses invalid startup. Parallel server pairs prove per-server isolation under the race detector. Canonical Kettle Model/report/transcript and extension semantic goldens have no diff. The README records the service and measured cost; no feature-map route changed.

baseline: green (focused healthy Kettle/server tests pre-edit, 19.231 s). All four new live differential witnesses first failed against the still-no-op switches, naming mutant and driver; raw evidence is retained.

Verification: focused healthy/mutant/server controls passed (41.038 s); differential-only golden run passed (20.546 s); server race tests and strict changed-line Go-host lint passed. Root make lint-code-fast used task base b0ac63ff0defbbca6947e8d21fc42987aa3d7f97 with fixes disabled. make model-check passed in 546.668 s at ef3b492d9a, including generic Go host 39.087 s and extension Go host 10.022 s. Later changes are README timing prose and Flow evidence only. R30's 60 s soft gate target remains exceeded. Unchanged extension live slice/gap proofs remain predecessor evidence; this task changed only the Kettle server and its Go controls.

The first green-receipt attempt refused the then-dirty README timing edit; after its commit the exact-command receipt was written and postreview verification honored it. Reviewers could not rerun Go under their read-only sandbox; independent review used committed logs, while worker verification executed the actual suites.

GATE_SKIPPED:unittest:green-receipt b909c861 - postreview verification reused this task's passing make model-check; no executable changes since its pass

stage: impl-review - ran (model: gpt-6.1-sol at high; correctness/contracts/integration all SHIP, no findings; same-family review)
Tier: session (jev-unavailable(no_key)); repository implementer override gpt-6.1-sol at medium.

Durable verification and review: .flow/artifacts/fn-233-kettle-service-a-framework-owned-end-to/task4-{verification,review,gate-receipt}.json and task4-*.log.gz.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ef3b492d9a13377b391e55cb4a69a55f23acfd3b, b909c861473c7cfb0e9747c75e985375117bedad
- Tests: cd model/umpire/hosts/go && go test -tags test_dep ./... -run 'TestKettle|TestHeatingReceipts|TestRejectionsPreserveState|TestMutant|TestServeRejects' -count=1, go -C model/umpire/hosts/go test -tags test_dep -race ./reference/kettle, make model-check, make lint-code-fast GOLANGCI_LINT_BASE_REV=b0ac63ff0defbbca6947e8d21fc42987aa3d7f97 GOLANGCI_LINT_FIX=false, cd model/umpire/hosts/go && ../../../../.bin/golangci-lint-v2.13.1 run --build-tags test_dep --timeout 10m --fix=false --new-from-patch=../../../../.flow/tmp/fn-233/task4-host.patch --config=../../../../.github/.golangci.yml ./..., GATE_SKIPPED:unittest:green-receipt b909c861 - postreview verification reused this task's passing make model-check; no executable changes since its pass
- PRs: