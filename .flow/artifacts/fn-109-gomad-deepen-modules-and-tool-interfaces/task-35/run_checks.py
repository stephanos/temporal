import datetime
import hashlib
import json
import os
from pathlib import Path
import shlex
import subprocess
import sys
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
PROOF = Path(__file__).resolve().parent
MODULE = ROOT / 'tools/gomad3'
ADMISSION = json.loads((PROOF / 'root-admission.json').read_text())
GO_BIN = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin'
TOOLS = {
    GO_BIN + '/go': '1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64',
    '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0': 'acacd04faf1d17489a890a4589ae36a62484c1076cf5fbcb7a33274cbc6a6bdc',
    '/tmp/fn109-lint-tools.ZdNe1t50/errortype': 'db481b4086fb85962e98ae625984f2560a883dce91f8b7d8c705414c207be8cc',
    str(ROOT / '.github/.golangci.yml'): '2abff492b6a1aeaaded801bccc2d8ab85ed366608862b0b9a5dd5311ae0fed43',
}
ENV_SET = {'PATH': GO_BIN + ':/usr/local/bin:/usr/bin:/bin', 'GOENV': 'off', 'GOFLAGS': '', 'GOWORK': 'off', 'GOTOOLCHAIN': 'local', 'GOPROXY': 'off', 'GOMAXPROCS': '2'}
ENV_UNSET = ['GOROOT', 'GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED', 'LINT_TEST_BASE_REV']


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def freeze():
    assert all(digest(ROOT / p) == h for p, h in ADMISSION['protected_files'].items())
    assert all(digest(p) == h for p, h in TOOLS.items())
    return {'admission_sha256': digest(PROOF / 'root-admission.json'), 'protected_count': len(ADMISSION['protected_files']), 'protected_unchanged': True, 'sources': {p: digest(ROOT / p) for p in ADMISSION['source_paths']}, 'tools': TOOLS}


phase = sys.argv[1]
assert phase in ('baseline', 'controls', 'preservation', 'final', 'audit')
commands = [
    ('package', 'go test -count=1 -tags test_dep -v ./runner/internal/corpus'),
    ('focused', "go test -count=1 -tags test_dep -v ./runner/internal/corpus -run '^TestCorpus(PublishesCanonicalSnapshotOnlyAfterMatchingReplay|RejectsIdentityChangesAndNonMatchingReplay|RejectsPreviousAndFutureSchemaBeforeChangedIdentity|RejectsEntryCapacityOverflowBeforeOpeningCases|RejectsCaseWithChangedEnvironment|RejectsMalformedAndNoncanonicalCurrentSchema|AllowsOnlyOneWriter|CasesShareOneTarget|KeepsItsTargetUntilNoCaseSharesIt|ByteCapCountsTheSharedTargetOnce|ReadSnapshotPreservesResultsAndErrorOrder|ValidateEntryPreservesResultsAndErrorOrder|ValidationFailureDoesNotPublish|CanonicalSnapshotBaseline)$'"),
    ('boundaries', "go test -count=1 -tags test_dep -v . -run '^(TestPackageArchitecture|TestPublicPackagesDoNotExportTypeAliases|TestArchitecturePublicSignatureFixtures|TestRunnerRequestsCompileInExternalModule|TestRunnerExternalConsumerCompiles)$'"),
    ('lint', '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./runner/internal/corpus'),
    ('errortype', 'go vet -tags test_dep -vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype -style-check=false ./runner/internal/corpus'),
    ('gofmt', 'gofmt -d runner/internal/corpus/corpus.go runner/internal/corpus/corpus_test.go'),
]
if phase in ('controls', 'preservation'):
    commands = [commands[1]]
if phase == 'audit':
    commands = [('environment', 'go env GOOS GOARCH GOVERSION GOROOT GOMOD GOWORK GOENV GOTOOLCHAIN GOPROXY GOFLAGS'), ('source', 'python3 ' + str(PROOF / 'source_audit.py'))]
environment = dict(os.environ)
for key in ENV_UNSET:
    environment.pop(key, None)
environment.update(ENV_SET)
for name, command in commands:
    before = freeze()
    if phase == 'baseline':
        assert before['sources'] == ADMISSION['source_before_sha256']
    started = time.monotonic()
    start = datetime.datetime.now(datetime.timezone.utc).isoformat()
    log = PROOF / f'{phase}-{name}.log'
    with log.open('wb') as output:
        result = subprocess.run(shlex.split(command), cwd=MODULE, env=environment, stdout=output, stderr=subprocess.STDOUT, timeout=600)
    after = freeze()
    receipt = {'command': command, 'cwd': str(MODULE), 'environment': {'set': ENV_SET, 'unset': ENV_UNSET}, 'source_before': before, 'source_after': after, 'stability': before == after, 'start': start, 'end': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'elapsed_seconds': round(time.monotonic() - started, 3), 'exit_code': result.returncode, 'log': str(log.relative_to(ROOT)), 'log_sha256': digest(log)}
    (PROOF / f'{phase}-{name}.receipt.json').write_text(json.dumps(receipt, indent=2) + '\n')
    print(f'{phase}-{name}: exit={result.returncode} elapsed={receipt["elapsed_seconds"]} stable={before == after}', flush=True)
    assert before == after
    assert result.returncode == (1 if name == 'lint' and phase == 'baseline' else 0), receipt
    if name == 'gofmt':
        assert log.stat().st_size == 0
