import pathlib
import subprocess
import sys

ROOT = pathlib.Path('/tmp/gomad-fn109-parallel.pmgezCtg/task-59').resolve()
if pathlib.Path.cwd().resolve() != ROOT:
    raise SystemExit('Wrong workspace')
RUNNER = pathlib.Path(__file__).with_name('run_gate.py')
LINT = '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0'
ERRORTYPE = '/tmp/fn109-lint-tools.ZdNe1t50/errortype'
FOCUSED = 'go test -tags test_dep -count=1 -json -run "^(TestRunChoiceExplorationDivergencePoliciesAndInspection|TestRunChoiceExplorationKeepsOtherErrorsAsHostErrors|TestProcessExplorationCompletionKeepsRunnerDomainFallback|TestRunChoiceExplorationResumePreservesDivergenceIdentity|TestRunChoiceExplorationRetainsOnlyPrefixMismatchReasons|TestCompletionFaultsKeepReasonPrecedenceAndEvidence|TestCancellationIsAHostFailure|TestCompletionProjectsWorldCoverageAndChoicesForEveryStrategy)$" ./runner'
gates = [
    ('focused-after', 'tools/gomad3', FOCUSED),
    ('affected-vet', 'tools/gomad3', 'go vet -tags test_dep ./runner'),
    ('standalone-errortype', 'tools/gomad3', f'go vet -tags test_dep -vettool={ERRORTYPE} -style-check=false ./runner'),
    ('architecture-source-sets', 'tools/gomad3', 'go test -tags test_dep -count=1 -json -run "^(TestPackageArchitecture|TestPublicPackagesDoNotExportTypeAliases|TestPureModulesHaveNoHostEffects|TestDomainModulesDoNotExportWireFraming|TestPublicPackagesDoNotExportForwardingAliases|TestHostPackageVet)$" .'),
    ('runner-ownership', 'tools/gomad3', 'go test -tags test_dep -count=1 -json -run "^TestRunnerExecutionInjectionIsPrivate$" .'),
    ('generated-validation', '.', 'make -C tools/gomad3 validate'),
    ('format-check', '.', 'test -z "$(gofmt -l tools/gomad3/runner/choice_exploration_divergence_test.go tools/gomad3/runner/completion_characterization_test.go)" && git diff --check'),
    ('affected-configured-lint', 'tools/gomad3', f'{LINT} run --verbose --build-tags test_dep --timeout 10m --fix=false --config={ROOT}/.github/.golangci.yml ./runner'),
    ('make-fast-task-base', '.', f'make lint-code-fast GOLANGCI_LINT_BASE_REV=f830467fcb712a83ba504c7dd43e2beb7f2cf223 GOLANGCI_LINT_FIX=false GOLANGCI_LINT={LINT} ERRORTYPE={ERRORTYPE}'),
    ('make-gomad-original-base', '.', f'make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false GOLANGCI_LINT={LINT} ERRORTYPE={ERRORTYPE}'),
]
for name, directory, command in gates:
    result = subprocess.run([sys.executable, str(RUNNER), name, directory, command], cwd=ROOT)
    print(name + ' terminal exit=' + str(result.returncode), flush=True)
    if result.returncode not in [0, 1, 2]:
        raise SystemExit(result.returncode)
