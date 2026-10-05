#!/usr/bin/env python3
"""Task-owned source freeze and foreground command capture, adapted from task20."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import shlex
import subprocess
import sys
import time

AREA = Path(__file__).resolve().parent
ROOT = AREA.parents[4]
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
ENV = dict(PATH=str(Path(GO).parent) + ':/usr/local/bin:/usr/bin:/bin', GOENV='off', GOWORK='off', GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off', GOFLAGS='', GOMAXPROCS='2')
UNSET = ['GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED']
DOCS = ['tools/gomad3/README.md', 'tools/gomad3/ARCHITECTURE.md', 'tools/gomad3/TUTORIAL.md']
CLAIMS = [(DOCS[0], 'recorded as v2 stable logical decisions.', 'recorded as v3 stable logical decisions.'), (DOCS[0], 'When v2 choice recording is enabled', 'When v3 choice recording is enabled'), (DOCS[1], 'complete v2 trace into a Decision Tape', 'complete v3 trace into a Decision Tape'), (DOCS[2], 'bounded v2 **Choice Trace**', 'bounded v3 **Choice Trace**'), (DOCS[2], 'complete v2 Choice Trace', 'complete v3 Choice Trace'), (DOCS[2], 'replayable v2 choice', 'replayable v3 choice')]
EXPLANATION = """`choice/trace.go` explicitly refuses stored v2 traces because their select
results carry no readiness. Legacy v1 traces remain decodable for inspection,
but `choice/tape.go` returns `ErrReplayUnavailable` when projecting them into
a replay plan. For v3, `ProjectReplayPlan` uses `projectSelectReadiness` to
carry each final `select` result's readiness onto its matching poll decisions.
A poll decision named by no result keeps unknown readiness. Readiness annotates
the tape without becoming a forced decision.

"""

def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT).decode()

def digest(data):
    return hashlib.sha256(data).hexdigest()

def save(name, value):
    path = AREA / name
    if path.exists():
        raise RuntimeError('refusing to overwrite ' + str(path))
    path.write_text(json.dumps(value, indent=2) + '\n')

def closure():
    paths = git('ls-files', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'tests/gomadfunctional', 'Makefile', '.github', '.golangci.yml', '.golangci.yaml', 'go.mod', 'go.sum', 'AGENTS.md', 'MILESTONES.md').splitlines()
    rows = {p: digest((ROOT / p).read_bytes()) for p in paths}
    return {'sha256': digest(json.dumps(rows, sort_keys=True, separators=(',', ':')).encode()), 'files': rows}

def capture(phase):
    env = {**os.environ, **ENV}
    for key in UNSET:
        env.pop(key, None)
    commands = [
        ('platforms', ['grep', '-n', 'darwin/arm64\\|linux/amd64', 'SPEC.md', 'ARCHITECTURE.md', 'README.md'], ROOT / 'tools/gomad3', []),
        ('focused', [GO, 'test', '-json', '-count=1', '-tags', 'test_dep', '.', '-run', '^(TestCurrentVocabularyHasNoLegacyCampaignBoundary|TestMakeTargetsMatchTheirOwnership)$'], ROOT / 'tools/gomad3', ['TestCurrentVocabularyHasNoLegacyCampaignBoundary', 'TestMakeTargetsMatchTheirOwnership']),
        ('choice', [GO, 'test', '-json', '-count=1', '-tags', 'test_dep', './choice', '-run', '^(TestChoiceReadersRejectOtherWireVersions|TestProjectReplayPlanCarriesSelectReadinessOntoPollDecisions|TestProjectDecisionTapeRejectsObservationOnlyAndLegacyTrace)$'], ROOT / 'tools/gomad3', ['TestChoiceReadersRejectOtherWireVersions', 'TestProjectReplayPlanCarriesSelectReadinessOntoPollDecisions', 'TestProjectDecisionTapeRejectsObservationOnlyAndLegacyTrace']),
        ('validate', ['make', '-C', 'tools/gomad3', 'validate'], ROOT, []),
    ]
    receipts = []
    for name, argv, cwd, expected in commands:
        before = closure()['sha256']
        start = datetime.datetime.now(datetime.timezone.utc).isoformat()
        clock = time.monotonic()
        logpath = AREA / (phase + '-' + name + '.log')
        with logpath.open('xb') as log:
            result = subprocess.run(argv, cwd=cwd, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=600)
        logdata = logpath.read_bytes()
        events = [json.loads(line) for line in logdata.decode().splitlines() if line.startswith('{')]
        observed = {row['Test'] for row in events if row.get('Action') == 'pass' and row.get('Test') in expected}
        row = dict(name=name, argv=argv, cwd=str(cwd), environment_overrides=ENV, environment_unset=UNSET, started_at=start, finished_at=datetime.datetime.now(datetime.timezone.utc).isoformat(), elapsed_seconds=time.monotonic()-clock, exit_code=result.returncode, log=logpath.name, log_sha256=digest(logdata), source_before=before, source_after=closure()['sha256'], expected_tests=expected, passed_expected_tests=sorted(observed))
        receipts.append(row)
        print(json.dumps(row), flush=True)
        if result.returncode or observed != set(expected) or row['source_before'] != row['source_after']:
            save(phase + '-receipts.json', receipts)
            raise SystemExit(result.returncode or 1)
    save(phase + '-receipts.json', receipts)

def check(phase):
    before = json.loads((AREA / 'before.json').read_text())
    results = []
    def add(name, passed, details):
        results.append(dict(name=name, passed=bool(passed), details=details))
    for path in DOCS:
        old = git('show', before['base_commit'] + ':' + path)
        new = (ROOT / path).read_text()
        expected = old
        for claimpath, stale, current in CLAIMS:
            if path == claimpath:
                expected = expected.replace(stale, current)
        if path == DOCS[1]:
            expected = expected.replace('Choice Exploration uses forced prefixes from one base Seed.', EXPLANATION + 'Choice Exploration uses forced prefixes from one base Seed.')
        if phase != 'baseline':
            add('all-other-bytes:' + path, new == expected, 'only six replacements and the named architecture paragraph')
        for claimpath, stale, current in CLAIMS:
            if path == claimpath:
                add('current-claim:' + current, current in new and stale not in new, path)
        add('fences:' + path, re.findall(r'^```.*?^```', old, re.M | re.S) == re.findall(r'^```.*?^```', new, re.M | re.S), 'all fenced bytes unchanged')
        add('balanced-fences:' + path, sum(line.startswith('```') for line in new.splitlines()) % 2 == 0, 'balanced')
        remaining = [line for line in old.splitlines() if 'v2' in line and not any(stale in line for p, stale, _ in CLAIMS if p == path)]
        add('unrelated-v2:' + path, remaining == [line for line in new.replace(EXPLANATION, '').splitlines() if 'v2' in line and not any(stale in line for p, stale, _ in CLAIMS if p == path)], remaining)
        errors = []
        for target in re.findall(r'\[[^\]]+\]\(([^)]+)\)', new):
            if '://' in target or target.startswith('mailto:'):
                continue
            file, _, anchor = target.partition('#')
            resolved = (ROOT / path).parent / file if file else ROOT / path
            if not resolved.exists():
                errors.append(target)
            elif anchor and resolved.suffix == '.md':
                headings = [re.sub(r'[^\w\- ]', '', h.lower()).replace(' ', '-') for h in re.findall(r'^#+ (.+)$', resolved.read_text(), re.M)]
                explicit = re.findall(r'<a id="([^"]+)"', resolved.read_text())
                if anchor not in headings + explicit:
                    errors.append(target)
        add('links:' + path, not errors, errors)
    now = closure()
    changed = [p for p, h in before['closure']['files'].items() if now['files'].get(p) != h]
    add('source-preservation', set(changed) <= set(DOCS) and now['files'].keys() == before['closure']['files'].keys(), {'files': len(now['files']), 'changed': changed, 'closure_sha256': now['sha256']})
    add('base-commit', git('rev-parse', 'HEAD').strip() == before['base_commit'], before['base_commit'])
    add('whitespace', subprocess.run(['git', 'diff', '--check', '--', *DOCS], cwd=ROOT).returncode == 0, 'git diff --check')
    add('user-files', all(digest((ROOT / p).read_bytes()) == h for p, h in before['user_files'].items()), list(before['user_files']))
    save(phase + '-doc-check.json', {'passed': all(r['passed'] for r in results), 'checks': results})
    print(json.dumps(results, indent=2))
    raise SystemExit(0 if all(r['passed'] for r in results) else 1)

if sys.argv[1] == 'freeze':
    base = git('rev-parse', 'HEAD').strip()
    rows = closure()
    users = ['.turbo/plans/gomad3-glossary-update.md', '.turbo/technical-debt.md']
    save('before.json', dict(base_commit=base, branch=git('branch', '--show-current').strip(), git_status=git('status', '--short'), closure=rows, git_index=git('ls-files', '-s', '--', *rows['files']), claims=[dict(path=p, before=s, after=c, line=git('show', base + ':' + p).splitlines().index(next(line for line in git('show', base + ':' + p).splitlines() if s in line))+1) for p,s,c in CLAIMS], user_files={p:digest((ROOT/p).read_bytes()) for p in users}, host=dict(system=platform.system(), machine=platform.machine(), patched_go_exists=(ROOT/'tools/gomad3/.toolchain/bin/go').exists()), environment_overrides=ENV, environment_unset=UNSET))
    (AREA / 'base_commit').write_text(base + '\n')
    print('frozen', len(rows['files']), rows['sha256'], base)
elif sys.argv[1] == 'capture':
    capture(sys.argv[2])
elif sys.argv[1] == 'check':
    check(sys.argv[2])
elif sys.argv[1] == 'evidence':
    before = json.loads((AREA / 'before.json').read_text())
    final = closure()
    save('final-source.json', final)
    bindings = {}
    for path, names in {
        'tools/gomad3/choice/trace.go': ['DecodeStoredTrace'],
        'tools/gomad3/choice/tape.go': ['ProjectReplayPlan', 'projectSelectReadiness'],
        'tools/gomad3/choice/legacy_v1.go': ['DecodeLegacyV1Trace', 'decodeLegacyV1Record'],
        'tools/gomad3/choice/readiness_test.go': ['TestChoiceReadersRejectOtherWireVersions', 'TestProjectReplayPlanCarriesSelectReadinessOntoPollDecisions'],
        'tools/gomad3/choice/tape_test.go': ['TestProjectDecisionTapeRejectsObservationOnlyAndLegacyTrace'],
    }.items():
        lines = (ROOT / path).read_text().splitlines()
        bindings[path] = dict(sha256=digest((ROOT/path).read_bytes()), functions={name: next(i+1 for i,line in enumerate(lines) if line.startswith('func '+name+'(')) for name in names})
    env = {**os.environ, **ENV}
    for key in UNSET:
        env.pop(key, None)
    host_commands = []
    for argv in [['uname', '-sm'], [GO, 'version'], [GO, 'env', 'GOHOSTOS', 'GOHOSTARCH', 'GOOS', 'GOARCH', 'GOTOOLCHAIN', 'GOWORK', 'GOPROXY', 'GOSUMDB']]:
        result = subprocess.run(argv, cwd=ROOT, env=env, capture_output=True, text=True, timeout=30)
        host_commands.append(dict(argv=argv, exit_code=result.returncode, stdout=result.stdout, stderr=result.stderr))
        if result.returncode:
            raise SystemExit(result.returncode)
    save('host.json', dict(commands=host_commands, patched_go_exists=(ROOT/'tools/gomad3/.toolchain/bin/go').exists(), environment_overrides=ENV, environment_unset=UNSET))
    baseline = json.loads((AREA/'baseline-receipts.json').read_text())
    final_receipts = json.loads((AREA/'final-receipts.json').read_text())
    lint_path = '.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-3/v041-restoration-20261005/root-integrated-lint.stdout.log'
    claim_rows = [dict(row, final_line=next(i+1 for i,line in enumerate((ROOT/row['path']).read_text().splitlines()) if row['after'] in line)) for row in before['claims']]
    command_strings = [phase + ': ' + shlex.join(row['argv']) + ' (exit ' + str(row['exit_code']) + ')' for phase, rows in [('BASE', baseline), ('FINAL', final_receipts)] for row in rows]
    checks = json.loads((AREA/'final-doc-check.json').read_text())
    command_strings += ['BASE documentation verifier exited 1 on exactly six stale current-version claims', 'FINAL documentation verifier exited 0; all ' + str(len(checks['checks'])) + ' checks passed']
    save('evidence.json', dict(task_id='fn-109-gomad-deepen-modules-and-tool-interfaces.20', status='in_progress', outcome='bounded_source_progress', base_commit=before['base_commit'], commits=[], commit_range=before['base_commit']+'..HEAD', prs=[], tests=command_strings, baseline='green: fresh focused, choice and check-only validation commands', baseline_source_sha256=before['closure']['sha256'], final_source_sha256=final['sha256'], source_inventory='before.json and final-source.json', source_file_count=len(final['files']), source_bindings=bindings, claims=claim_rows, command_receipts=['baseline-receipts.json','final-receipts.json'], host='host.json; Linux/aarch64 stock go1.27.1; patched Go absent; developmental evidence only', historical_lint=dict(path=lint_path, sha256=digest((ROOT/lint_path).read_bytes()), diagnostics=317, errortype='UNREACHED', rerun=False, meaning='retained inherited residuals; executable source unchanged, no fresh lint pass'), qualification=dict(native_darwin=False, formal_review=False, R18_reconciliation=False, D5_acceptance=False, linux_owner='fn-128'), scope=dict(product_paths=DOCS, root_owns_flow=True, root_owns_commit=True, staged_by_worker=False, committed_by_worker=False), terminal_commands=True))
    print(json.dumps(dict(base_commit=before['base_commit'], final_source_sha256=final['sha256'], source_file_count=len(final['files']), evidence=str(AREA/'evidence.json'))))
