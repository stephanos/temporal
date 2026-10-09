import pathlib
import subprocess
import sys

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
RUNNER = pathlib.Path(__file__).with_name('run_gate.py')
LINT = '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0'
ERRORTYPE = '/tmp/fn109-lint-tools.ZdNe1t50/errortype'
BASE = '60ae2649db11f637b851817cd77c3fc814e33d1b'
ORIGINAL = '951c5516e9e7b3066e7e069adda9565cfd68844c'
gates = [
    ('focused', 'tools/gomad3', 'go test -tags test_dep -count=1 -json -run "^(TestBuildCleanup|TestBuildPublishesAndReusesImmutableToolchain|TestBuildInjectedFailuresLeaveNoTemporaryState|TestBuildRejectsOverlayCollisionBeforePatching|TestBuildRejectsUnguardedFailureInjection)" ./toolchain'),
    ('affected-vet', 'tools/gomad3', 'go vet -tags test_dep ./toolchain'),
    ('standalone-errortype', 'tools/gomad3', f'go vet -tags test_dep -vettool={ERRORTYPE} -style-check=false ./toolchain'),
    ('architecture', 'tools/gomad3', 'go test -tags test_dep -count=1 -json -run "^TestPackageArchitecture$" .'),
    ('format-check', '.', 'test -z "$(gofmt -l tools/gomad3/toolchain/build.go tools/gomad3/toolchain/build_cleanup_test.go)" && git diff --check'),
    ('affected-configured-lint', 'tools/gomad3', f'{LINT} run --verbose --build-tags test_dep --timeout 10m --fix=false --config={ROOT}/.github/.golangci.yml ./toolchain'),
    ('make-fast-task-base', '.', f'make lint-code-fast GOLANGCI_LINT_BASE_REV={BASE} GOLANGCI_LINT_FIX=false GOLANGCI_LINT={LINT} ERRORTYPE={ERRORTYPE}'),
    ('make-gomad-original-base', '.', f'make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV={ORIGINAL} GOLANGCI_LINT_FIX=false GOLANGCI_LINT={LINT} ERRORTYPE={ERRORTYPE}'),
]
for name, directory, command in gates:
    result = subprocess.run([sys.executable, str(RUNNER), 'corrected-' + name, directory, command], cwd=ROOT)
    print('corrected-' + name + ' terminal exit=' + str(result.returncode), flush=True)
    if result.returncode not in [0, 1, 2]:
        raise SystemExit(result.returncode)
