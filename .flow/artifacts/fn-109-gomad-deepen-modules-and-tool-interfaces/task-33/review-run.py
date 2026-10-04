"""Independent review checks, with stable source/protected inputs per command."""
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
BASE = '1ee85b004cfaac41e178784701decbf0dd277968'
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
LINT = '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0'
ERROR = '/tmp/fn109-lint-tools.ZdNe1t50/errortype'


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def old_sha(path):
    item = Path(path)
    if item.is_symlink():
        return hashlib.sha256(os.fsencode(os.readlink(item))).hexdigest()
    return sha(item)


def aggregate(values):
    return dict(files=len(values), aggregate_sha256=hashlib.sha256(json.dumps(values, sort_keys=True).encode()).hexdigest())


def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT, text=True)


def snapshot():
    admission = json.loads((OUT / 'root-admission.json').read_text())
    assert git('rev-parse', 'HEAD').strip() == BASE
    assert git('branch', '--show-current').strip() == 'gomad'
    paths = git('ls-files', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml').splitlines()
    protected = aggregate({p: sha(ROOT / p) for p in paths if p != SOURCE})
    assert protected == admission['protected'], protected
    source = sha(ROOT / SOURCE)
    assert source == '016bb31d600047178bc355784dd2635ef44dbbb6d2b351a8687913a8623b76b4'
    old = git('ls-files', '.flow/artifacts', '.flow/tasks').splitlines()
    old = [p for p in old if '/task-33/' not in p and not p.endswith(('.21.json', '.33.md', '.33.json'))]
    originals = {p: sha(ROOT / p) for p in admission['original_documents']}
    assert originals == admission['original_documents']
    plan = sha(ROOT / '.flow/tmp/artifact-remaining-diagnostics-owner-plan.md')
    assert plan == admission['source_plan_sha256']
    return dict(source_sha256=source, protected=protected, older_evidence_and_tasks=aggregate({p: old_sha(ROOT / p) for p in old}), original_documents=originals, source_plan_sha256=plan, writer_handover_sha256=sha(OUT / 'handover.md'), writer_evidence_sha256=sha(OUT / 'evidence.json'), tools_sha256={p: sha(p) for p in (GO, LINT, ERROR)}, config_sha256=sha(ROOT / '.github/.golangci.yml'), review_runner_sha256=sha(__file__))


if sys.argv[1] == 'serial-all':
    for selected in ('package', 'focused', 'boundaries', 'lint', 'errortype', 'gofmt', 'diff-check', 'writer-audit', 'scope', 'validation'):
        subprocess.check_call([sys.executable, __file__, 'serial-' + selected], cwd=ROOT)
    sys.exit(0)
name = sys.argv[1].removeprefix('serial-')
focused = '^Test(DeepCopy.*|CloneManifestSharesNoMemory|ManifestCopiesCannotChangeOpenedHandle|ArtifactReferenceHoldsNoOpenResource|PublishedReferenceMatchesOpenedHandle|OpenedArtifact.*|ClosedArtifactRejectsPayloadAccess|SyncDirectoryContextPreservesPrimaryResults|PrivatePayload.*|PublishRemovesStagingAfterNonregularSourceFailure|VerifySharedPayload.*|DamagedSharedTargetFailsOpenAndLaterPublication|CopiedArtifactOpensWithoutItsStore)$'
boundaries = '^(TestPackageArchitecture|TestPublicPackagesDoNotExportTypeAliases|TestArchitecturePublicSignatureFixtures|TestRunnerRequestsCompileInExternalModule|TestRunnerExternalConsumerCompiles)$'
commands = {
    'package': [GO, 'test', '-count=1', '-tags', 'test_dep', '-v', './artifact/...'],
    'focused': [GO, 'test', '-count=1', '-tags', 'test_dep', '-v', './artifact', '-run', focused],
    'boundaries': [GO, 'test', '-count=1', '-tags', 'test_dep', '-v', '.', '-run', boundaries],
    'lint': [LINT, 'run', '--config', str(ROOT / '.github/.golangci.yml'), '--build-tags', 'test_dep', '--fix=false', './artifact'],
    'errortype': [ERROR, '-tags', 'test_dep', './artifact'],
    'gofmt': [str(Path(GO).parent / 'gofmt'), '-d', str(ROOT / SOURCE)],
    'diff-check': ['git', 'diff', '--check', BASE, '--', SOURCE],
    'validation': ['make', 'validate', 'GOFLAGS=-tags=test_dep -count=1'],
    'writer-audit': ['python3', str(OUT / 'writer-audit.py')],
    'scope': ['python3', str(OUT / 'root-scope-gate.py')],
}
argv = commands[name]
cwd = ROOT if name in ('diff-check', 'writer-audit', 'scope') else ROOT / 'tools/gomad3'
before = snapshot()
environment = os.environ.copy()
for key in ('GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOROOT'):
    environment.pop(key, None)
environment.update(PATH=str(Path(GO).parent) + ':' + environment['PATH'], GOWORK='off', GOTOOLCHAIN='local', GOPROXY='off', GOFLAGS='-tags=test_dep -count=1' if name == 'validation' else '')
start = datetime.datetime.now(datetime.timezone.utc).isoformat()
clock = time.monotonic()
log_path = OUT / ('review-' + sys.argv[1] + '.log')
receipt_path = OUT / ('review-' + sys.argv[1] + '.json')
assert not log_path.exists() and not receipt_path.exists()
with log_path.open('x') as log:
    result = subprocess.run(argv, cwd=cwd, env=environment, stdout=log, stderr=subprocess.STDOUT, timeout=600, check=False)
after = snapshot()
log = log_path.read_text()
receipt = dict(command=argv, cwd=str(cwd), started_at=start, elapsed_seconds=round(time.monotonic()-clock, 3), exit_code=result.returncode, environment={key: environment.get(key) for key in ('PATH', 'GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOFLAGS', 'GOMADSEED', 'GOMAD3_CHILD_SEED')}, input_before=before, input_after=after, stable=before == after, log_sha256=sha(log_path), top_level_run=sum(line.startswith('=== RUN   ') and '/' not in line for line in log.splitlines()), top_level_pass=sum(line.startswith('--- PASS: ') and '/' not in line for line in log.splitlines()))
receipt_path.write_text(json.dumps(receipt, indent=2) + '\n')
assert before == after, 'inputs moved'
print(json.dumps(dict(name=name, exit_code=result.returncode, elapsed_seconds=receipt['elapsed_seconds'], top_level_run=receipt['top_level_run'], top_level_pass=receipt['top_level_pass'], stable=receipt['stable'], receipt_sha256=sha(receipt_path), log_sha256=receipt['log_sha256'])))
