---
satisfies: [R3, R4, R7, R8]
---

# fn-20-local-execution-semantic-conformance.7 Run the bounded live proof and synchronize current-model documentation

## Description
Complete R3/R4/R7/R8 with one fn-19-to-fn-20 caller-closure proof, public usage docs, architecture handoff, and honest roadmap status.

**Size:** M
**Files:** `tools/umpire/runevaluation/live_test.go`, `tools/umpire/runevaluation/README.md`, `model/README.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [tools/umpire/runevaluation/live_test.go, tools/umpire/runevaluation/README.md, model/README.md, model/ARCHITECTURE.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach
- Produce one real closed four-member caller-closure set through fn-19, check it through the actual sibling pair, publish/reopen the six-member set, and independently assert cleanup/source closure plus the API/history-backed Property result and diagnostic obligations.
- Run the full corruption/ambiguity suite before the green live control and require exact deterministic rerun destination.
- Document the two offline command inputs, six-member output, checker-pair build, statuses, Limits, three independent result dimensions, evidence dispositions, fail-closed cases, and copy-paste root command.
- Update C7, Milestone B, and pilot step 5 only after the live and mutation evidence genuinely passes; retain remote/Observation Evaluation and replay/promotion gaps.
- Run all focused suites plus the stable regression gate; add no CI workflow or model-local Make target.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.7.md` — live four-member producer/harness
- `model/README.md:93-98,149-158` — current model/runtime handoff text
- `model/ARCHITECTURE.md:90-103,147-159` — Feature/System/Tool ownership and lifecycle
- `.plans/UMPIRE4_COMPONENTS.md:330-363,613-640` — C7 and Milestone B status
- `.plans/UMPIRE4_COMPONENTS.md:700-715` — pilot sequencing/status

## Acceptance
- [ ] One bounded real local run yields a reopened six-member set whose Result proves the intended API/history Property and explicitly binds cleanup/source closure.
- [ ] Corruption/ambiguity controls run first and every repeated valid run/check produces the expected stable semantic and publication identities.
- [ ] Documentation gives copy-paste usage and explains why operational success is not semantic satisfaction.
- [ ] Roadmap claims only the one local scenario and keeps replay/promotion, formal checking, and non-local/release Observation Evaluation open.
- [ ] Focused Lean/Go, direct/root command, and `make umpire-check-regression` checks pass without prohibited dependencies or new CI.

## Done summary
Completed the bounded fn-19-to-fn-20 caller-closure proof through the real producer, fixed Lean checker pair, and six-member publication/reopen path. The live gate runs independent corruption and ambiguity controls first, proves cleanup and four-source closure, preserves all input members byte-for-byte, and requires deterministic semantic and publication identities across repeated checks. The exact Nexus adapter now binds the cancellation claim to the unique `realize` command, accepted status, exact schema, run identity, and history workflow/operation correlations; misbound or duplicate claims remain typed evidence and produce publishable `conflict` Results with no Property evaluation.

Published copy-paste CLI/Make usage and synchronized current-model architecture, component, milestone, and pilot documentation without claiming replay, promotion, or non-local Observation Evaluation. Focused Lean/Go tests, the pre-live control group, the real live proof, full package, race, fuzz, vet, model lint, and the full 240-job Umpire regression are green. Nonmutating full repository lint retains the inherited 1,392 findings with zero findings on fn20.7 task paths. Codex review session `01a05398-0add-7ce1-adf0-c4ff385a84c3` returned SHIP in round 3 after both P1 findings were fixed, with zero surviving findings and all R1-R9 requirements met.
## Evidence
- Commits: 1f4a31003, c5daecaa6, 4a4a5dcd3, 0214de305, 21272d2a6, 750d48760
- Tests: mise exec -- lake build Temporal.Tool.RunEvaluationTests, mise exec -- lake build Temporal.Tool.RunEvaluationMutationTests, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH go test -count=1 -tags test_dep -run '^(TestRawArtifactMutationFailsAtAdmission|TestCheckerRequestSeparatesRuntimeAndCheckedMappings|TestCheckerResponseRejectsConsistentCheckedProfileDriftAtTheProtocolBoundary|TestRealCheckerObservationMutationMatrix|TestRealCheckerMisboundParticipantCancellationEvidenceIsSemanticConflict|TestRealCheckerPartialEvidencePublishesAnInMemoryResult|TestRealCheckerSiblingIsDeterministic|TestRealCheckerSiblingAdmitsExactAcceptedSet|TestRealCheckerCancellationPublishesNoPartialSet)$' ./tools/umpire/runevaluation, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH go test -count=1 -tags test_dep -run '^TestBoundedLiveCallerClosureEvaluation$' ./tools/umpire/runevaluation, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH go test -count=1 -tags test_dep ./tools/umpire/runevaluation, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH go test -race -count=1 -tags test_dep ./tools/umpire/runevaluation, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH go test -tags test_dep -run '^$' -fuzz '^FuzzDecodeCheckerResponse$' -fuzztime=10s ./tools/umpire/runevaluation, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH go vet -tags test_dep ./tools/umpire/runevaluation, make lint-model, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH make umpire-check-regression, INHERITED_RED: make lint-code GOLANGCI_LINT_FIX=false reports 1392 repository findings and zero fn20.7 task-path findings
- PRs: