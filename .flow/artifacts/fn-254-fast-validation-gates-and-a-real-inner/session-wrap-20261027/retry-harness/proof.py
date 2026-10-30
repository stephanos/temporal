"""Prepared Linux proof harness. Execution needs an explicitly admitted source HEAD."""
from pathlib import Path
import argparse
import datetime
import hashlib
import json
import os
import re
import shutil
import signal
import subprocess
import sys
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn25416-provider-adapt')
GUARD = 30 * 1024**3
TASK = '.flow/tasks/fn-254-fast-validation-gates-and-a-real-inner.16.md'
SOURCE_PATHS = ['model', 'Makefile', 'mise.toml', 'mise.lock', TASK]


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def write(path, value):
    Path(path).write_text(json.dumps(value, indent=2) + '\n')


def interrupt(*_):
    raise KeyboardInterrupt


def snapshot(root):
    paths = subprocess.check_output(['git', 'ls-files', '-z', '--', *SOURCE_PATHS], cwd=root)
    files = {}
    for raw in paths.split(b'\0'):
        if raw:
            name = os.fsdecode(raw)
            path = root / name
            files[name] = {'symlink': os.readlink(path)} if path.is_symlink() else sha(path)
    untracked = subprocess.check_output(
        ['git', 'ls-files', '--others', '--exclude-standard', '-z', '--', *SOURCE_PATHS], cwd=root
    )
    assert not untracked, 'untracked source inputs require admission'
    return {'head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
            'files': files}


def environment(root, target, tmp):
    env = os.environ.copy()
    for name in list(env):
        if name.startswith(('UMPIRE_', 'CARGO_PROFILE_')) or name in [
            'BASH_ENV', 'TRYBUILD', 'INSTA_UPDATE', 'NEXTEST_PROFILE', 'NEXTEST_TEST_THREADS',
            'CARGO_BUILD_TARGET', 'CARGO_ENCODED_RUSTFLAGS', 'RUSTC_BOOTSTRAP', 'RUSTC_WRAPPER',
            'RUSTC_WORKSPACE_WRAPPER', 'RUSTFLAGS',
        ]:
            env.pop(name, None)
    env.update(CARGO_TARGET_DIR=str(target), CARGO_HOME=str(Path('/home/agent/.cargo').resolve()),
               TMPDIR=str(tmp), CARGO_BUILD_JOBS='1', CARGO_INCREMENTAL='0',
               GOFLAGS='-p=1', GOMAXPROCS='1', UMPIRE_BACKENDS='require', UMPIRE_APALACHE='skip',
               UMPIRE_QUICK_CACHE='0', UMPIRE_CACHE_DISABLE='1', MISE_AUTO_INSTALL='false',
               MISE_EXEC_AUTO_INSTALL='false', MISE_TRUSTED_CONFIG_PATHS=str(root)+':'+str(target))
    env['GOCACHE'] = str(target.parent/'go-cache')
    return env


class Run:
    def __init__(self, options):
        assert sys.platform == 'linux', 'Darwin validation remains deferred'
        self.root = options.source_root.resolve()
        self.initial = snapshot(self.root)
        assert self.initial['head'] == options.expected_head, 'root HEAD is not the admitted candidate'
        self.out = options.output.resolve()
        self.out.mkdir(parents=True, exist_ok=False)
        self.target = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn25416-provider-validation/target').resolve()
        self.tmp = self.out / 'tmp'
        self.tmp.mkdir()
        self.env = environment(self.root, self.target, self.tmp)
        self.growth_paths = [self.target, self.tmp, Path(self.env['GOCACHE'])]
        self.baseline = self.size()
        self.overrides = {}
        write(self.out/'source-before.json', self.initial)
        write(self.out/'harness.json', {'command': sys.argv, 'driver_sha256': sha(__file__),
              'expected_head': options.expected_head, 'free_before_bytes': shutil.disk_usage(self.root).free,
              'free_guard_bytes': GUARD, 'growth_guard_bytes': GUARD, 'macos': 'owner-deferred'})
        from observer import Observer
        assert __import__('ctypes').CDLL(None).prctl(36, 1, 0, 0, 0) == 0
        self.observer = Observer(self.out)

    def size(self):
        from census import apparent_bytes

        def missing(operation, root, path, error):
            with (self.out/'growth-census-transients.jsonl').open('a') as log:
                log.write(json.dumps({'at': time.time(), 'operation': operation,
                    'root_path_hex': root.hex(), 'path_hex': path.hex(),
                    'errno': error.errno})+'\n')

        return sum(apparent_bytes(path, missing) for path in self.growth_paths)

    def source(self):
        expected = {'head': self.initial['head'], 'files': dict(self.initial['files'])}
        expected['files'].update(self.overrides)
        assert snapshot(self.root) == expected, 'frozen source input changed'

    def run(self, name, command, cwd, env=None, timeout=1800, observe=False):
        self.source()
        assert shutil.disk_usage(self.root).free >= GUARD, '30GiB free admission guard'
        env = self.env if env is None else env
        start = time.monotonic()
        receipt = {'command': command, 'cwd': str(cwd), 'started_utc': datetime.datetime.now(
            datetime.timezone.utc).isoformat(), 'source_sha256': sha(self.out/'source-before.json'),
            'environment': {k: v for k, v in env.items() if k.startswith((
                'UMPIRE_', 'CARGO_', 'NEXTEST_', 'MISE_')) or k in ['TMPDIR', 'GOFLAGS', 'GOMAXPROCS']}}
        stdout, stderr = self.out/(name+'.stdout.log'), self.out/(name+'.stderr.log')
        code = None
        child = None
        next_growth = time.monotonic()
        try:
            with stdout.open('wb') as a, stderr.open('wb') as b:
                child = subprocess.Popen(command, cwd=cwd, env=env, stdout=a, stderr=b,
                                         start_new_session=True)
                if observe:
                    self.observer.pid = child.pid
                    write(self.out/'active-pid.json', {'pid': child.pid, 'command': command})
                while True:
                    try:
                        code = child.wait(timeout=1)
                        break
                    except subprocess.TimeoutExpired:
                        free = shutil.disk_usage(self.root).free
                        reason = None
                        if free < GUARD: reason = 'free-space guard'
                        if time.monotonic()-start > timeout: reason = 'timeout guard'
                        if stdout.stat().st_size+stderr.stat().st_size > 64*1024**2:
                            reason = 'output guard'
                        if time.monotonic() >= next_growth:
                            if self.size()-self.baseline > GUARD: reason = 'growth guard'
                            next_growth = time.monotonic()+15
                        if reason:
                            receipt['stopped_for_guard'] = reason
                            os.killpg(child.pid, signal.SIGTERM)
                            try: child.wait(timeout=10)
                            except subprocess.TimeoutExpired:
                                os.killpg(child.pid, signal.SIGKILL)
                                child.wait()
                            code = child.returncode
                            raise RuntimeError(reason)
        except BaseException:
            if child is not None and child.poll() is None:
                os.killpg(child.pid, signal.SIGTERM)
                try: child.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    os.killpg(child.pid, signal.SIGKILL)
                    child.wait()
                code = child.returncode
            raise
        finally:
            receipt.update(exit_code=code, seconds=time.monotonic()-start,
                free_after_bytes=shutil.disk_usage(self.root).free,
                stdout_sha256=sha(stdout), stderr_sha256=sha(stderr))
            write(self.out/(name+'.json'), receipt)
        self.source()
        print(name, code, round(receipt['seconds'], 3), flush=True)
        assert code == 0, name
        return receipt

    def finish(self):
        try:
            self.observer.finish()
        finally:
            write(self.out/'source-after.json', snapshot(self.root))
        self.source()


def doctests(path):
    rows = [list(map(int, row)) for row in re.findall(
        r'\[gate:tests\] test result: ok\. (\d+) passed; (\d+) failed; (\d+) ignored; '
        r'(\d+) measured; (\d+) filtered out;', path.read_text())]
    assert rows, 'separate Cargo doctest evidence missing'
    totals = dict(zip(['passed', 'failed', 'ignored', 'measured', 'filtered_out'], map(sum, zip(*rows))))
    assert totals['failed'] == totals['measured'] == 0
    return {'suite_summaries': len(rows), **totals}


def validate_parent(run, parent, child_root):
    result = json.loads((child_root/'result.json').read_text())
    assert result['complete'] and result['source_unchanged'] and result['child_exit'] == 0
    assert parent['stage'] == 'standalone' and parent['scope'] == 'framework'
    assert parent['state'] == 'success' and parent['exit'] == 0 and parent['count_error'] is None
    assert parent['count_authority'] == 'exclusive-stage'
    assert all(value is None for value in parent['counts'].values()), 'parent double-charged child work'
    links = parent['child_count_receipts']
    assert len(links) == 1
    link = links[0]
    path_bytes = bytes(link['path_bytes'])
    linked = Path(os.fsdecode(path_bytes))
    assert linked == child_root/'stages.jsonl' and os.fsencode(linked) == path_bytes
    if link['path'] is not None: assert os.fsencode(link['path']) == path_bytes
    records = linked.read_bytes()
    assert hashlib.sha256(records).hexdigest() == link['sha256'] == result['stages_sha256']
    assert records == link['records'].encode()
    stages = [json.loads(line) for line in records.splitlines()]
    assert stages == link['stages'] and len(stages) == 17
    assert [row['stage'] for row in stages] == result['expected_stages'] == result['actual_stages']
    assert len({(row['scope'], row['stage']) for row in stages}) == 17
    assert all(row['state'] == 'success' and row['exit'] == 0 and row['count_error'] is None
               and row['count_authority'] == 'exclusive-stage' for row in stages)
    inventory = json.loads((run.out/'child-nextest-all.stdout.log').read_text())
    identities = sorted([suite['package-name'], suite['binary-name'], suite['kind'], name, test['ignored']]
        for suite in inventory['rust-suites'].values() for name, test in suite.get('testcases', {}).items())
    ignored = sum(row[-1] for row in identities)
    assert inventory['test-count'] == len(identities)
    docs = doctests(run.out/'scheduled-standalone.stdout.log')
    counts = next(row['counts'] for row in stages if row['stage'] == 'tests')
    assert counts['discovered'] == len(identities)+docs['passed']+docs['ignored']+docs['filtered_out']
    assert counts['executed'] == len(identities)-ignored+docs['passed']
    assert counts['ignored'] == ignored+docs['ignored'] and counts['skipped'] == docs['filtered_out']
    assert counts['discovered'] == counts['executed']+counts['ignored']+counts['skipped']
    assert docs['passed'] >= 4 and docs['ignored'] >= 1, 'known doctest bodies omitted'
    for row in stages:
        if row['stage'] in ['go-host', 'kettle-go-host']:
            assert all(value is None for value in row['counts'].values()), 'unsupported Go count invented'
    write(run.out/'child-inventory-identities.json', identities)
    return {'accepted': True, 'head': run.initial['head'],
        'parent_receipt': parent, 'child_result': result, 'child_counts': counts, 'doctests': docs,
        'nextest_inventory_count': len(identities), 'nextest_ignored': ignored,
        'child_receipt_sha256': link['sha256'], 'parent_own_counts_not_double_charged': True,
        'macos': 'owner-deferred; no runtime acceptance claim'}


def standalone(options):
    run = Run(options)
    production = Path('/tmp')/('umpire-fn25416-final-'+options.expected_head[:12]+'-'+str(os.getpid()))
    assert not production.exists()
    run.growth_paths.append(production)
    run.env['UMPIRE_STANDALONE_OUTPUT'] = str(production)
    bootstrap = run.target/'xtask-bootstrap'
    binary = bootstrap/'debug/xtask'
    validation = None
    try:
        env = dict(run.env, CARGO_PROFILE_DEV_DEBUG='0')
        run.run('native-bootstrap', ['mise','exec','--','cargo','build','--locked','-p','xtask',
            '--bin','xtask','-p','umpire-guest-driver','--bin','umpire-guest-driver','--profile',
            'dev','--target-dir',str(bootstrap)], run.root/'model/umpire', env=env, timeout=900)
        receipts = run.out/'parent-receipts'
        receipts.mkdir()
        env = dict(run.env, UMPIRE_GATE_RECEIPTS=str(receipts), UMPIRE_GATE_SCOPE='framework',
            UMPIRE_GATE_STAGE='standalone', UMPIRE_GATE_ORDER='17',
            UMPIRE_GATE_EXECUTABLE=str(binary), UMPIRE_GATE_CONTROLLER_EXECUTABLE=str(binary))
        run.run('scheduled-standalone', [str(binary),'gate','--scheduled-stage','standalone'],
                run.root/'model/umpire', env=env, timeout=3600, observe=True)
        children = list(production.glob('proof-*'))
        assert len(children) == 1
        child = children[0]
        copied_env = environment(child/'framework', child/'target', child/'target/tmp')
        copied_env.update(NEXTEST_PROFILE='required', NEXTEST_TEST_THREADS='4')
        run.run('child-nextest-all', ['mise','exec','--','cargo','nextest','list','--workspace',
            '--locked','--message-format','json','--run-ignored','all'], child/'framework', env=copied_env)
        parent = json.loads((receipts/'framework/standalone.json').read_text())
        validation = validate_parent(run, parent, child)
        write(run.out/'native-binaries.json', {str(path):sha(path) for path in [
            binary, bootstrap/'debug/umpire-guest-driver']})
    finally:
        run.finish()
    assert validation is not None
    validation['cleanup'] = json.loads((run.out/'owned-cleanup.json').read_text())
    assert validation['cleanup']['all_observed_owned_pids_absent']
    validation['source_after_sha256'] = sha(run.out/'source-after.json')
    write(run.out/'validation.json', validation)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source-root', type=Path, default=ROOT)
    parser.add_argument('--expected-head', required=True)
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--execute', action='store_true')
    options = parser.parse_args()
    assert re.fullmatch('[0-9a-f]{40}', options.expected_head)
    if not options.execute:
        print('Prepared only; no source inspection, process, build or proof execution requested.')
        return
    signal.signal(signal.SIGTERM, interrupt)
    standalone(options)


if __name__ == '__main__':
    main()
