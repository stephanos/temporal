#!/usr/bin/env python3
"""Capture bounded campaign command evidence without altering source."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
LINT = '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0'
ERROR = '/tmp/fn109-lint-tools.ZdNe1t50/errortype'
CONFIG = ROOT / '.github/.golangci.yml'
ENV = dict(os.environ)
ENV.update(GOWORK='off', GOTOOLCHAIN='local', GOPROXY='off', GOFLAGS='')
ENV['PATH'] = str(Path(GO).parent) + ':' + ENV['PATH']
for name in ('GOMADSEED', 'GOMAD3_CHILD_SEED'):
    ENV.pop(name, None)

def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def snapshot():
    paths = sorted((ROOT / 'tools/gomad3/runner/internal/campaign').glob('*.go'))
    return {str(path.relative_to(ROOT)): digest(path) for path in paths}

def run(name, command, cwd):
    before = snapshot()
    started = datetime.datetime.now(datetime.timezone.utc).isoformat()
    clock = time.monotonic()
    with (OUT / (name + '.log')).open('w') as log:
        result = subprocess.run(command, cwd=cwd, env=ENV, stdout=log, stderr=subprocess.STDOUT)
    elapsed = time.monotonic() - clock
    after = snapshot()
    receipt = dict(command=command, cwd=str(cwd), environment={key: ENV.get(key) for key in
        ('PATH', 'GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOFLAGS', 'GOMADSEED', 'GOMAD3_CHILD_SEED')},
        started=started, ended=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        elapsed_seconds=elapsed, exit=result.returncode, source_before=before, source_after=after,
        stable=before == after, tools={path: digest(path) for path in (GO, LINT, ERROR)},
        config_sha256=digest(CONFIG), log=name + '.log', log_sha256=digest(OUT / (name + '.log')))
    (OUT / (name + '.json')).write_text(json.dumps(receipt, indent=2) + '\n')
    print(json.dumps({key: receipt[key] for key in ('command', 'exit', 'elapsed_seconds', 'stable', 'log')}))

name, kind = sys.argv[1:3]
cwd = ROOT / 'tools/gomad3'
if kind == 'report':
    base = 'd7c6695cff81a1160ee482cb82b62cf592f4f199'
    names = ['baseline-package', 'baseline-focused', 'baseline-lint', 'baseline-errortype',
             'characterization', 'characterization-baseline', 'final-lint',
             'final-predicates-package', 'final-predicates-focused', 'final-predicates-lint',
             'final-predicates-errortype', 'frozen-boundary', 'final-validation']
    receipts = {name: json.loads((OUT / (name + '.json')).read_text()) for name in names}
    def findings(name):
        pattern = r'^(.*\.go):(\d+):(\d+): (.*) \((\w+)\)$'
        return [dict(path=match[0], line=int(match[1]), column=int(match[2]), message=match[3], linter=match[4])
                for match in re.findall(pattern, (OUT / (name + '.log')).read_text(), re.MULTILINE)]
    before, after = findings('baseline-lint'), findings('final-predicates-lint')
    protected_before = json.loads((OUT / 'protected-before.json').read_text())
    protected_after = json.loads((OUT / 'protected-after.json').read_text())
    current = snapshot()
    baseline = receipts['baseline-package']['source_before']
    matches_base = all(value == hashlib.sha256(subprocess.check_output(['git', 'show', base + ':' + path], cwd=ROOT)).hexdigest() for path, value in baseline.items())
    final_names = ['final-predicates-package', 'final-predicates-focused', 'final-predicates-lint', 'final-predicates-errortype', 'frozen-boundary']
    changed = subprocess.check_output(['git', 'diff', base, '--name-only', '--', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml'], cwd=ROOT, text=True).splitlines()
    diff = subprocess.run(['git', 'diff', '--check', '--', 'tools/gomad3/runner/internal/campaign'], cwd=ROOT, capture_output=True, text=True)
    source_checks = dict(baseline_matches_base=matches_base, all_gate_sources_stable=all(item['stable'] for item in receipts.values()),
        final_gate_sources_match_current=all(receipts[name]['source_after'] == current for name in final_names),
        protected_inputs_stable=protected_before['aggregate_sha256'] == protected_after['aggregate_sha256'],
        protected_files=protected_after['files'], protected_sha256=protected_after['aggregate_sha256'],
        original_completion_tests_unchanged=baseline['tools/gomad3/runner/internal/campaign/controller_completion_test.go'] == current['tools/gomad3/runner/internal/campaign/controller_completion_test.go'],
        changed_product_paths=changed, scoped_diff_check_command=['git', 'diff', '--check', '--', 'tools/gomad3/runner/internal/campaign'], scoped_diff_check_exit=diff.returncode)
    result = dict(status='SOURCE_PROGRESS_ONLY', task='fn-109-gomad-deepen-modules-and-tool-interfaces.28',
        base_commit=base, commits=[], commit_range='', requested_model='gpt-6.1-sol', requested_effort='high',
        tier='session (jev-unavailable(no_key))', actual_model=None, actual_model_metadata_available=False,
        platform=protected_after['platform'], source_checks=source_checks,
        gates={name: dict(receipt=name + '.json', log=item['log'], exit=item['exit'], elapsed_seconds=item['elapsed_seconds'], stable=item['stable']) for name, item in receipts.items()},
        pinned_tools=receipts['baseline-lint']['tools'], lint_config_sha256=receipts['baseline-lint']['config_sha256'],
        lint=dict(before=before, after=after, mapped_before=15, mapped_after=0, total_before=len(before), total_after=len(after),
            resolved=[item for item in before if item['linter'] in ('errcheck', 'exhaustive')], introduced=[],
            residual='two original invariant forbidigo findings with unchanged messages and panic bodies',
            intermediate='final-lint.log retains two QF1003 diagnostics from the superseded if/else-if form'),
        source_final={path: current[path] for path in changed},
        root_close=dict(public_and_unix_source=protected_after['pinned_root_source'], unix_returns_nil=True, real_nonnil_close_injection=False,
            limitation='Conditional join ordering is source-inspected; no nonnil os.Root.Close execution is demonstrated.'),
        review=dict(owner='root', backend='codex', status='pending independent fresh source review', formal='conductor-deferred while qualification remains red'),
        remaining=['Original R16/R18/R19 and task3/predecessors/task21 acceptance', 'Fixed supplied first-baseline identity evidence',
                   'Complete/full/formal and qualified darwin/arm64 and linux/amd64 gates', 'Actual patched-runtime/native validation',
                   'Two intentional invariant forbidigo diagnostics', 'Historical broader419 qualification'],
        historical_references=['../task-27/handover.md', '../task-21/preservation-disclosure-2026-10-04.md', '../task-25/root-fast.receipt.json'],
        owned_delegates=0, live_command_handles=0, pending_commands=0, worker_git_writes=0, worker_flow_lifecycle_writes=0)
    assert matches_base and source_checks['all_gate_sources_stable'] and source_checks['final_gate_sources_match_current']
    assert source_checks['protected_inputs_stable'] and source_checks['original_completion_tests_unchanged'] and diff.returncode == 0
    assert len(before) == 17 and len(after) == 2 and all(item['linter'] == 'forbidigo' for item in after)
    print(json.dumps(result, indent=2))
    sys.exit(0)
if kind == 'protect':
    tracked = subprocess.check_output(['git', 'ls-files', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml'], cwd=ROOT, text=True).splitlines()
    paths = [path for path in tracked if not path.startswith('tools/gomad3/runner/internal/campaign/')]
    values = {path: digest(ROOT / path) for path in paths}
    receipt = dict(selection='git tracked Gomad modules, root go.mod/go.sum and lint config excluding campaign source',
        files=len(values), aggregate_sha256=hashlib.sha256(json.dumps(values, sort_keys=True).encode()).hexdigest(),
        captured=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        pinned_root_source={str(Path(GO).parents[1] / 'src/os' / file): digest(Path(GO).parents[1] / 'src/os' / file) for file in ('root.go', 'root_openat.go')},
        platform=subprocess.check_output([GO, 'env', 'GOOS', 'GOARCH'], env=ENV, text=True).splitlines(),
        patched_go_exists=(cwd / '.toolchain/bin/go').exists())
    (OUT / (name + '.json')).write_text(json.dumps(receipt, indent=2) + '\n')
    print(json.dumps(receipt))
    sys.exit(0)
if kind == 'test':
    command = [GO, 'test', '-count=1', '-tags', 'test_dep', *sys.argv[3:]]
elif kind == 'lint':
    command = [LINT, 'run', '--config', str(CONFIG), '--build-tags', 'test_dep', '--fix=false', './runner/internal/campaign']
elif kind == 'errortype':
    command = [ERROR, '-tags', 'test_dep', './runner/internal/campaign']
elif kind == 'validate':
    command = ['make', 'validate', 'GOFLAGS=-tags=test_dep -count=1']
else:
    raise ValueError(kind)
run(name, command, cwd)
