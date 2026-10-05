"""Task-local receipts, adapted from task-13/capture.py."""
import hashlib
import json
import os
import pathlib
import subprocess
import sys
import time

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = pathlib.Path(__file__).resolve().parent
STOCK = pathlib.Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin')


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def sources():
    paths = subprocess.check_output(['git', 'ls-files', '-z'], cwd=ROOT).decode().split('\0')
    return {name: digest(ROOT / name) for name in paths
            if name and not name.startswith('.flow/') and (ROOT / name).is_file()}


def environment():
    env = os.environ.copy()
    for key in ['GOROOT', 'GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED', 'GOEXPERIMENT']:
        env.pop(key, None)
    env.update(GOENV='off', GOWORK='off', GOTOOLCHAIN='local', GOPROXY='off',
               GOSUMDB='off', GOFLAGS='', GOMAXPROCS='2', GOMAD3_STOCK_GO=str(STOCK / 'go'))
    env['PATH'] = str(STOCK) + ':' + env['PATH']
    return env


def recorded_environment(env):
    keys = ['GOENV', 'GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOSUMDB', 'GOFLAGS',
            'GOMAXPROCS', 'GOMAD3_STOCK_GO', 'PATH', 'GOEXPERIMENT', 'GOROOT',
            'GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED', 'GOCACHE', 'GOMODCACHE']
    return {key: env.get(key) for key in keys}


if __name__ == '__main__':
    label = sys.argv[1]
    if label == 'sanitize':
        path = OUT / sys.argv[2]
        info = json.loads(path.read_text())
        info['environment'] = recorded_environment(info['environment'])
        path.write_text(json.dumps(info, indent=2) + '\n')
        print('normalized recorded environment keys', path.name)
    elif label == 'freeze':
        info = {'base_commit': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT).decode().strip(),
                'sources': sources(), 'environment': recorded_environment(environment()),
                'tools': {str(p): digest(p) for p in [STOCK / 'go', STOCK / 'gofmt',
                    ROOT / 'tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz']},
                'user_files': {name: digest(ROOT / name) for name in
                    ['.turbo/plans/gomad3-glossary-update.md', '.turbo/technical-debt.md']}}
        (OUT / 'freeze.json').write_text(json.dumps(info, indent=2) + '\n')
        print('frozen', len(info['sources']), 'tracked product files')
    else:
        paths = [OUT / (label + suffix) for suffix in ['.stdout', '.stderr', '.json']]
        if any(path.exists() for path in paths):
            raise SystemExit('receipt label already exists; use a unique label: ' + label)
        cwd = ROOT / sys.argv[2]
        argv = sys.argv[3:]
        env = environment()
        before = sources()
        start = time.time()
        with (OUT / (label + '.stdout')).open('wb') as stdout, (OUT / (label + '.stderr')).open('wb') as stderr:
            result = subprocess.run(argv, cwd=cwd, env=env, stdout=stdout, stderr=stderr)
        end = time.time()
        after = sources()
        info = {'argv': argv, 'cwd': str(cwd), 'environment': recorded_environment(env), 'exit': result.returncode,
                'started_unix': start, 'ended_unix': end, 'elapsed_seconds': end - start,
                'sources_before_sha256': hashlib.sha256(json.dumps(before, sort_keys=True).encode()).hexdigest(),
                'sources_after_sha256': hashlib.sha256(json.dumps(after, sort_keys=True).encode()).hexdigest(),
                'source_count': len(before),
                'source_changes': [name for name in set(before) | set(after) if before.get(name) != after.get(name)],
                'stdout_sha256': digest(OUT / (label + '.stdout')),
                'stderr_sha256': digest(OUT / (label + '.stderr'))}
        (OUT / (label + '.json')).write_text(json.dumps(info, indent=2) + '\n')
        print(label, 'exit', result.returncode, 'seconds', round(end - start, 3), 'source_changes', info['source_changes'])
        print((OUT / (label + '.stdout')).read_text(errors='replace')[-2000:])
        print((OUT / (label + '.stderr')).read_text(errors='replace')[-2000:])
        sys.exit(result.returncode)
