import pathlib
import subprocess
import sys

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task-60').resolve()
if pathlib.Path.cwd().resolve() != ROOT:
    raise SystemExit('Wrong workspace')
RUNNER = pathlib.Path(__file__).with_name('run_gate.py')
LINT = '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0'
ERRORTYPE = '/tmp/fn109-lint-tools.ZdNe1t50/errortype'
FOCUSED = 'go test -tags test_dep -count=1 -json -run "^(TestTargetFixtureOutputPreservesProcessOutcome|TestRunCapturesTargetExitAndBothStreams|TestRunReplaysLogicalChoiceAcrossPhysicalRunQueueOrder|TestRunReplaysSelectPermutationAcrossSeededPhysicalOrder|TestRunPreservesChoicePrefixRNGPosition|TestRunTargetInheritsChoiceTapeReadOnly|TestRunRejectsExhaustedChoiceTapeBeforeTargetMarker|TestRunRejectsChoiceMetadataMismatchBeforeTargetMarker)$" ./runner/internal/execution'
CONFIGURED = f'{LINT} run --verbose --build-tags test_dep --timeout 10m --fix=false --config={ROOT}/.github/.golangci.yml ./runner/internal/execution'
ORIGINAL = f'make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false GOLANGCI_LINT={LINT} ERRORTYPE={ERRORTYPE}'
mode = sys.argv[1]
if mode == 'baseline':
    gates = [
        ('baseline-output-controls', 'tools/gomad3', 'go test -tags test_dep -count=1 -json -run "^TestTargetFixtureOutputPreservesProcessOutcome$" ./runner/internal/execution'),
        ('baseline-focused', 'tools/gomad3', FOCUSED),
        ('baseline-configured-lint', 'tools/gomad3', CONFIGURED),
        ('baseline-original-lint', '.', ORIGINAL),
    ]
elif mode == 'final':
    gates = [
        ('final-focused', 'tools/gomad3', FOCUSED),
        ('final-ordinary-execution', 'tools/gomad3', 'go test -tags test_dep -count=1 -json ./runner/internal/execution'),
        ('final-vet', 'tools/gomad3', 'go vet -tags test_dep ./runner/internal/execution'),
        ('final-format', '.', 'test -z "$(gofmt -l tools/gomad3/runner/internal/execution/process_test.go tools/gomad3/runner/internal/execution/process_fixture_output_test.go)" && git diff --check'),
        ('final-configured-lint', 'tools/gomad3', CONFIGURED),
        ('final-fast-lint', '.', f'make lint-code-fast GOLANGCI_LINT_BASE_REV=7b75ae312a6641c6b00afbb63f44ae9a9dd0c2b2 GOLANGCI_LINT_FIX=false GOLANGCI_LINT={LINT} ERRORTYPE={ERRORTYPE}'),
        ('final-original-lint', '.', ORIGINAL),
    ]
else:
    raise SystemExit('Unknown batch')
for name, directory, command in gates:
    result = subprocess.run([sys.executable, str(RUNNER), name, directory, command], cwd=ROOT)
    print(name + ' terminal exit=' + str(result.returncode), flush=True)
    if result.returncode not in [0, 1, 2]:
        raise SystemExit(result.returncode)
