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
ENV_SET = {
    'PATH': GO_BIN + ':/usr/local/bin:/usr/bin:/bin',
    'GOENV': 'off', 'GOFLAGS': '', 'GOWORK': 'off',
    'GOTOOLCHAIN': 'local', 'GOPROXY': 'off', 'GOMAXPROCS': '2',
}
ENV_UNSET = ['GOROOT', 'GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED', 'LINT_TEST_BASE_REV']


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def freeze():
    changed = [p for p, h in ADMISSION['protected_files'].items() if digest(ROOT / p) != h]
    assert not changed, changed
    for p, h in TOOLS.items():
        assert digest(p) == h, p
    assert digest(ROOT / '.flow/tmp/cli-remaining-diagnostics-owner-plan.md') == ADMISSION['owner_plan_sha256']
    return {
        'admission_sha256': digest(PROOF / 'root-admission.json'),
        'protected_count': len(ADMISSION['protected_files']),
        'protected_unchanged': True,
        'fixture_sha256': digest(ROOT / ADMISSION['source_path']),
        'tools': TOOLS,
    }


phase = sys.argv[1]
assert phase in ('baseline', 'final')
portable = json.loads((PROOF.parent / 'task-26/focused-cli.receipt.json').read_text())['command']
commands = [
    ('private-mode', "go test -count=1 -tags test_dep ./cmd/gomad/internal/cli -run '^TestCharacterizeUnknownCommandsAndPrivateModes$' -v"),
    ('portable-cli', portable),
    ('boundaries', "go test -count=1 -tags test_dep . -run '^(TestPackageArchitecture|TestPublicPackagesDoNotExportTypeAliases|TestArchitecturePublicSignatureFixtures|TestRunnerRequestsCompileInExternalModule|TestRunnerExternalConsumerCompiles)$' -v"),
    ('lint', '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./cmd/gomad/internal/cli'),
    ('errortype', 'go vet -tags test_dep -vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype -style-check=false ./cmd/gomad/internal/cli'),
    ('gofmt', 'gofmt -d cmd/gomad/internal/cli/characterization_test.go'),
]
environment = dict(os.environ)
for key in ENV_UNSET:
    environment.pop(key, None)
environment.update(ENV_SET)
for name, command in commands:
    before = freeze()
    if phase == 'baseline':
        assert before['fixture_sha256'] == ADMISSION['source_before_sha256']
    start = datetime.datetime.now(datetime.timezone.utc).isoformat()
    started = time.monotonic()
    log = PROOF / f'{phase}-{name}.log'
    signal = None
    with log.open('wb') as output:
        try:
            result = subprocess.run(shlex.split(command), cwd=MODULE, env=environment, stdout=output, stderr=subprocess.STDOUT, timeout=600)
            exit_code = result.returncode
        except subprocess.TimeoutExpired:
            exit_code = 124
            signal = 'timeout'
    after = freeze()
    end = datetime.datetime.now(datetime.timezone.utc).isoformat()
    receipt = {
        'command': command, 'cwd': str(MODULE),
        'environment': {'set': ENV_SET, 'unset': ENV_UNSET},
        'source_before': before, 'source_after': after,
        'stability': before == after,
        'start': start, 'end': end, 'elapsed_seconds': round(time.monotonic() - started, 3),
        'exit_code': exit_code, 'signal': signal,
        'log': str(log.relative_to(ROOT)), 'log_sha256': digest(log),
    }
    (PROOF / f'{phase}-{name}.receipt.json').write_text(json.dumps(receipt, indent=2) + '\n')
    print(f'{phase}-{name}: exit={exit_code} elapsed={receipt["elapsed_seconds"]} stability={receipt["stability"]}', flush=True)
    assert before == after
    assert exit_code == (1 if name == 'lint' else 0), receipt
    if name == 'gofmt':
        assert log.stat().st_size == 0
