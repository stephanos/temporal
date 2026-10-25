#!/usr/bin/env python3
"""Reproducible fn254.1 measurements, admitted explicitly by the conductor."""
import argparse
import datetime
import glob
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import threading
import time

from observer import digest, write

MAIN = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
TARGET = Path('/home/agent/.cache/codex-build/umpire-milestones-20261024/fn24612')
BASE = MAIN / '.flow/tmp/fn254-baseline'
PROBE = BASE / 'probe'
ENV = dict(CARGO_TARGET_DIR=str(TARGET), GOCACHE=str(TARGET / 'go'),
           CARGO_PROFILE_DEV_DEBUG='0', CARGO_PROFILE_TEST_DEBUG='0',
           CARGO_PROFILE_DEV_INCREMENTAL='0', CARGO_PROFILE_TEST_INCREMENTAL='0',
           CARGO_INCREMENTAL='0', CARGO_BUILD_JOBS='2', GOMAXPROCS='2',
           GOFLAGS='-p=2', CGO_ENABLED='0', UMPIRE_APALACHE='skip')


def output(argv, cwd=MAIN, env=None):
    return subprocess.check_output(argv, cwd=cwd, env=env, text=True).strip()


def processes():
    lines = output(['ps', '-eo', 'pid,ppid,etimes,pcpu,rss,args']).splitlines()
    return [line for line in lines if re.search(r'\b(cargo|rustc|rustdoc|java|go|make)\b', line)]


def run(label, argv, cwd, env, merged=False):
    folder = Path(env['FN254_OUT'])
    record = dict(label=label, argv=argv, cwd=str(cwd), started_utc=datetime.datetime.now(datetime.UTC).isoformat(),
                  started_ns=time.monotonic_ns(), exit_code=None, processes_before=processes())
    path = folder / (label + '.json')
    write(path, record)
    samples = []
    stop = threading.Event()

    def monitor():
        while not stop.wait(5):
            samples.append(dict(at_ns=time.monotonic_ns(), processes=processes()))
            write(folder / (label + '-processes.json'), samples)

    thread = threading.Thread(target=monitor)
    thread.start()
    try:
        with (folder / (label + '.stdout')).open('wb') as stdout, (folder / (label + '.stderr')).open('wb') as stderr:
            proc = subprocess.Popen(argv, cwd=cwd, env=env, stdout=stdout, stderr=subprocess.STDOUT if merged else stderr)
            record['pid'] = proc.pid
            write(path, record)
            status = proc.wait()
    finally:
        stop.set()
        thread.join()
    record.update(exit_code=status, ended_ns=time.monotonic_ns(), processes_after=processes(),
                  ended_utc=datetime.datetime.now(datetime.UTC).isoformat())
    record['seconds'] = (record['ended_ns']-record['started_ns'])/1e9
    write(path, record)
    print(json.dumps(dict(label=label, exit_code=status, seconds=record['seconds'])), flush=True)
    return record


def binaries(folder):
    found = {}
    for line in (folder / 'build.stdout').read_text().splitlines():
        obj = json.loads(line)
        if obj.get('reason') == 'compiler-artifact' and obj.get('executable') and obj['profile']['test']:
            path = obj['executable']
            found[path] = dict(package_id=obj['package_id'], target=obj['target'], features=obj['features'],
                               fresh=obj['fresh'], profile=obj['profile'], binary=path, sha256=digest(path),
                               compile_seconds=None, link_seconds=None,
                               build_cost_reason='warm artifact reuse; aggregate Cargo build cannot assign per-binary compile/link cost')
    if not found:
        raise RuntimeError('no test executables discovered')
    return list(found.values())


def inventory(folder, env):
    found = binaries(folder)
    metadata = json.loads(output([env['FN254_REAL_MISE'], 'exec', '--', 'cargo', 'metadata', '--no-deps', '--format-version=1', '--locked'], MAIN / 'model', env))
    write(folder / 'metadata.json', metadata)
    package_roots = {package['id']: Path(package['manifest_path']).parent for package in metadata['packages']}
    tests = []
    for index, binary in enumerate(found):
        identity = {key: binary[key] for key in ['package_id', 'target', 'features', 'binary', 'sha256']}
        lists = {}
        for ignored in [False, True]:
            args = [binary['binary'], '--list', '--format=pretty'] + (['--ignored'] if ignored else [])
            record = run(f'list-{index}-ignored{int(ignored)}', args, package_roots[binary['package_id']], env)
            if record['exit_code'] != 0:
                raise RuntimeError('binary inventory failed')
            text = (folder / (record['label'] + '.stdout')).read_text()
            lists[ignored] = {m[1]: m[2] for m in re.finditer(r'^(.+): (test|benchmark)$', text, re.M)}
        for name, kind in lists[False].items():
            tests.append(dict(identity, name=name, kind=kind, ignored=name in lists[True], backend='require', apalache='skip'))
        if not lists[True].keys() <= lists[False].keys():
            raise RuntimeError('ignored inventory not contained')
    write(folder / 'binaries.json', found)
    write(folder / 'tests.json', tests)
    cases = []
    for package in metadata['packages']:
        package_root = Path(package['manifest_path']).parent
        for source in package_root.rglob('*.rs'):
            if any(segment in source.parts for segment in ['target', 'build', 'archived']):
                continue
            owner = next((parent for parent in source.parents if (parent/'Cargo.toml').is_file()), None)
            if owner != package_root:
                continue
            text = source.read_text()
            if 'trybuild::' not in text:
                continue
            for match in re.finditer(r'\.(compile_fail|pass)\("([^"\n]+)"\)', text):
                matches = sorted(glob.glob(str(package_root / match[2])))
                if not matches:
                    raise RuntimeError(f'empty UI registration {source}: {match[2]}')
                for case in matches:
                    diagnostic = Path(case).with_suffix('.stderr')
                    cases.append(dict(package_id=package['id'], registration=str(source.relative_to(MAIN)),
                                      mode=match[1], pattern=match[2], source=str(Path(case).relative_to(MAIN)),
                                      sha256=digest(case), diagnostic_sha256=digest(diagnostic) if diagnostic.exists() else None))
    write(folder / 'ui-cases.json', cases)
    listings = {}
    doc_targets = {}
    for package in metadata['packages']:
        for target in package['targets']:
            if target['doctest']:
                doc_targets.setdefault(target['name'].replace('-', '_'), []).append((package['id'], target))
    for ignored in [False, True]:
        args = [env['FN254_REAL_MISE'], 'exec', '--', 'cargo', 'test', '--workspace', '--locked', '--doc', '--', '--list', '--format=pretty']
        if ignored:
            args.append('--ignored')
        record = run('doctests-ignored' + str(int(ignored)), args, MAIN / 'model', env, merged=True)
        if record['exit_code'] != 0:
            raise RuntimeError('doctest inventory failed')
        current = None
        names = {}
        for line in (folder/(record['label']+'.stdout')).read_text().splitlines():
            heading = re.match(r'^\s*Doc-tests (\S+)\s*$', line)
            if heading:
                current = heading[1]
                if len(doc_targets.get(current, [])) != 1:
                    raise RuntimeError(f'ambiguous doctest package for {current}')
                continue
            item = re.match(r'^(.+): test$', line)
            if item:
                if current is None:
                    raise RuntimeError('doctest identity before package heading')
                names[(current, item[1])] = doc_targets[current][0]
        listings[ignored] = names
    doctests = []
    for (target_name, name), (package_id, target) in listings[False].items():
        features = sorted({feature for binary in found if binary['package_id']==package_id for feature in binary['features']})
        doctests.append(dict(package_id=package_id, target=target, name=name, ignored=(target_name,name) in listings[True], features=features, backend='require', apalache='skip'))
    if not listings[True].keys() <= listings[False].keys():
        raise RuntimeError('ignored doctest inventory not contained')
    write(folder / 'doctests.json', doctests)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('phase', choices=['build', 'inventory', 'default', 'serial', 'host'])
    parser.add_argument('--run', required=True)
    parser.add_argument('--lease', required=True)
    args = parser.parse_args()
    lease = json.loads(Path(args.lease).read_text())
    if lease.get('target') != str(TARGET) or lease.get('main') != str(MAIN) or not lease.get('admitted'):
        raise RuntimeError('explicit main/cache lease required')
    os.chdir(MAIN)
    source = output(['git', 'rev-parse', 'HEAD:model'])
    if source != lease['model_tree']:
        raise RuntimeError('source model tree changed since lease')
    dirty_source = output(['git', 'status', '--porcelain', '--untracked-files=normal', '--', 'model', 'Makefile', 'mise.toml'])
    if dirty_source:
        raise RuntimeError('measurement source is dirty: ' + dirty_source)
    folder = BASE / 'runs' / args.run
    folder.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, **ENV, FN254_OUT=str(folder))
    env.pop('BASH_ENV', None)
    real_mise = shutil.which('mise')
    env['FN254_REAL_MISE'] = real_mise
    env['FN254_REAL_JAVA'] = output([real_mise, 'which', 'java'])
    env['UMPIRE_BACKENDS'] = 'require'
    for mode in ['mise', 'java', 'cargo-runner', 'serial-runner']:
        destination = PROBE / mode
        if not destination.exists():
            destination.symlink_to('observer.py')
    if args.phase == 'host':
        host = dict(source_commit=output(['git', 'rev-parse', 'HEAD']), model_tree=source,
                    dirty=output(['git', 'status', '--short']), host=output(['uname', '-a']),
                    cores=os.cpu_count(), memory=Path('/proc/meminfo').read_text(),
                    disk=output(['df', '-h', str(MAIN), str(TARGET)]), environment=ENV,
                    rustc=output([real_mise, 'exec', '--', 'rustc', '-Vv']),
                    cargo=output([real_mise, 'exec', '--', 'cargo', '-V']),
                    go=output([real_mise, 'exec', '--', 'go', 'version']), mise=output([real_mise, '--version']),
                    java=env['FN254_REAL_JAVA'], java_sha256=digest(env['FN254_REAL_JAVA']),
                    processes=processes(), lease=lease)
        write(folder / 'host.json', host)
    elif args.phase == 'build':
        record = run('build', [real_mise, 'exec', '--', 'cargo', 'test', '--workspace', '--locked', '--no-run', '--message-format=json', '--timings'], MAIN / 'model', env)
        for report in (TARGET / 'cargo-timings').glob('*.html'):
            shutil.copy2(report, folder / report.name)
        if record['exit_code'] != 0:
            return record['exit_code']
        write(folder / 'binaries.json', binaries(folder))
    elif args.phase == 'inventory':
        inventory(folder, env)
    elif args.phase == 'default':
        host = output([real_mise, 'exec', '--', 'rustc', '-vV'])
        triple = re.search(r'^host: (.+)$', host, re.M)[1]
        env['CARGO_TARGET_' + triple.upper().replace('-', '_') + '_RUNNER'] = str(PROBE / 'cargo-runner')
        env.update(UMPIRE_BACKENDS='skip', UMPIRE_TIME_STEPS='1')
        return run('default', ['make', 'model-check'], MAIN, env)['exit_code']
    elif args.phase == 'serial':
        env['PATH'] = str(PROBE) + os.pathsep + env['PATH']
        found = json.loads((folder / 'binaries.json').read_text())
        failures = []
        for index, binary in enumerate(found):
            package_root = Path(next(package['manifest_path'] for package in json.loads((folder/'metadata.json').read_text())['packages'] if package['id'] == binary['package_id'])).parent
            record = run('serial-' + str(index), [str(PROBE / 'serial-runner'), binary['binary']], package_root, env)
            if record['exit_code'] != 0:
                failures.append(record['label'])
        write(folder / 'serial-result.json', dict(failures=failures, binaries=len(found), exit_code=int(bool(failures))))
        return int(bool(failures))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
