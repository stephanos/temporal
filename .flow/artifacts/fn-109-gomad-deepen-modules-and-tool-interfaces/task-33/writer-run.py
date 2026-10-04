import datetime
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
SOURCE = 'tools/gomad3/artifact/opened_test.go'
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
LINT = '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0'
ERROR = '/tmp/fn109-lint-tools.ZdNe1t50/errortype'

def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def snapshot():
    paths = subprocess.check_output(['git', 'ls-files', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml'], cwd=ROOT, text=True).splitlines()
    values = {path: sha(ROOT / path) for path in paths if path != SOURCE}
    protected = dict(files=len(values), aggregate_sha256=hashlib.sha256(json.dumps(values, sort_keys=True).encode()).hexdigest())
    assert protected == json.loads((OUT / 'root-admission.json').read_text())['protected'], protected
    return dict(source_sha256=sha(ROOT / SOURCE), protected=protected)

def run(name, argv, cwd=ROOT / 'tools/gomad3', goflags=''):
    assert not (OUT / (name + '.json')).exists(), name
    before = snapshot()
    environment = os.environ.copy()
    for key in ('GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOROOT'):
        environment.pop(key, None)
    environment.update(PATH=str(Path(GO).parent) + ':' + environment['PATH'], GOWORK='off', GOTOOLCHAIN='local', GOPROXY='off', GOFLAGS=goflags)
    start = datetime.datetime.now(datetime.timezone.utc).isoformat()
    clock = time.monotonic()
    with (OUT / (name + '.log')).open('x') as log:
        result = subprocess.run(argv, cwd=cwd, env=environment, stdout=log, stderr=subprocess.STDOUT, timeout=600, check=False)
    after = snapshot()
    log = (OUT / (name + '.log')).read_text()
    receipt = dict(command=argv, cwd=str(cwd), environment={key: environment.get(key) for key in ('GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOFLAGS', 'GOMADSEED', 'GOMAD3_CHILD_SEED')}, go_path=GO, started_at=start, elapsed_seconds=round(time.monotonic()-clock, 3), exit_code=result.returncode, source_before=before, source_after=after, stable=before == after, tools_sha256={path: sha(path) for path in (GO, LINT, ERROR)}, config_sha256=sha(ROOT / '.github/.golangci.yml'), log_sha256=sha(OUT / (name + '.log')), top_level_run=sum(line.startswith('=== RUN   ') and '/' not in line for line in log.splitlines()), top_level_pass=sum(line.startswith('--- PASS: ') and '/' not in line for line in log.splitlines()))
    (OUT / (name + '.json')).write_text(json.dumps(receipt, indent=2) + '\n')
    assert before == after, 'source moved during command'
    print(json.dumps(dict(name=name, exit_code=result.returncode, elapsed_seconds=receipt['elapsed_seconds'], top_level_run=receipt['top_level_run'], top_level_pass=receipt['top_level_pass'], source_sha256=before['source_sha256'])))
    return result.returncode

if __name__ == '__main__':
    name = sys.argv[1]
    if name == 'baseline':
        run('baseline-package', [GO, 'test', '-count=1', '-tags', 'test_dep', '-v', './artifact/...'])
        run('baseline-lint', [LINT, 'run', '--config', str(ROOT / '.github/.golangci.yml'), '--build-tags', 'test_dep', '--fix=false', './artifact'])
        run('baseline-errortype', [ERROR, '-tags', 'test_dep', './artifact'])
    elif name == 'characterization':
        run(name, [GO, 'test', '-count=1', '-tags', 'test_dep', '-v', './artifact', '-run', '^TestDeepCopy(ArrayReferences|ContainerAndScalarValues|RejectsUnsupportedKinds)$'])
    elif name == 'audit':
        run('writer-audit', ['python3', str(OUT / 'writer-audit.py')], ROOT)
    elif name == 'final':
        run('final-package', [GO, 'test', '-count=1', '-tags', 'test_dep', '-v', './artifact/...'])
        run('final-focused', [GO, 'test', '-count=1', '-tags', 'test_dep', '-v', './artifact', '-run', '^Test(DeepCopy.*|CloneManifestSharesNoMemory|ManifestCopiesCannotChangeOpenedHandle|ArtifactReferenceHoldsNoOpenResource|PublishedReferenceMatchesOpenedHandle|OpenedArtifact.*|ClosedArtifactRejectsPayloadAccess|SyncDirectoryContextPreservesPrimaryResults|PrivatePayload.*|PublishRemovesStagingAfterNonregularSourceFailure|VerifySharedPayload.*|DamagedSharedTargetFailsOpenAndLaterPublication|CopiedArtifactOpensWithoutItsStore)$'])
        run('final-boundaries', [GO, 'test', '-count=1', '-tags', 'test_dep', '-v', '.', '-run', '^(TestPackageArchitecture|TestPublicPackagesDoNotExportTypeAliases|TestArchitecturePublicSignatureFixtures|TestRunnerRequestsCompileInExternalModule|TestRunnerExternalConsumerCompiles)$'])
        run('final-errortype', [ERROR, '-tags', 'test_dep', './artifact'])
        run('final-lint', [LINT, 'run', '--config', str(ROOT / '.github/.golangci.yml'), '--build-tags', 'test_dep', '--fix=false', './artifact'])
        run('final-gofmt', [str(Path(GO).parent / 'gofmt'), '-d', str(ROOT / SOURCE)])
        run('final-diff-check', ['git', 'diff', '--check', '--', SOURCE], ROOT)
        run('final-validation', ['make', 'validate', 'GOFLAGS=-tags=test_dep -count=1'], goflags='-tags=test_dep -count=1')
