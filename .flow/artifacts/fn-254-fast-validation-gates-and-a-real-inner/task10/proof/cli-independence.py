#!/usr/bin/env python3
"""Run from the task checkout after integrating the framework gate source."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time

checkout = Path.cwd().resolve()
framework = checkout / 'model/umpire'
proof = checkout / '.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task10/proof'
target = Path('/home/agent/.cache/codex-build/umpire-milestones-20261024/fn25410-gate/parent')
env = os.environ.copy()
for key in ['BASH_ENV', 'CARGO_PROFILE_DEV_DEBUG', 'CARGO_PROFILE_TEST_DEBUG', 'RUSTFLAGS', 'CARGO_ENCODED_RUSTFLAGS', 'CARGO_BUILD_TARGET', 'RUSTC_WRAPPER', 'RUSTC_WORKSPACE_WRAPPER']:
    env.pop(key, None)
env.update(CARGO_TARGET_DIR=str(target), CARGO_BUILD_JOBS='2', RUST_TEST_THREADS='2', CARGO_INCREMENTAL='0')
commands = []
def run(label, argv):
    start = time.monotonic()
    with (proof / (label + '.log')).open('wb') as output:
        completed = subprocess.run(argv, cwd=framework, env=env, stdout=output, stderr=subprocess.STDOUT)
    commands.append(dict(label=label, argv=list(map(str,argv)), cwd=str(framework), exit=completed.returncode, seconds=time.monotonic()-start))
    if completed.returncode:
        raise RuntimeError(f'{label}: exit {completed.returncode}')
def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()
def snapshot():
    artifacts = {}
    for path in sorted((target / 'guest-cache').glob('*/wasm32-wasip1/guest/**/*.wasm')):
        stat = path.stat()
        artifacts[str(path.relative_to(target))] = dict(sha256=digest(path), size=stat.st_size, mtime_ns=stat.st_mtime_ns, inode=stat.st_ino)
    return dict(cli_sha256=digest(target/'debug/xtask'), driver_sha256=digest(target/'debug/umpire-guest-driver'), lock_sha256=digest(framework/'Cargo.lock'), namespaces=sorted(p.name for p in (target/'guest-cache').iterdir()), artifacts=artifacts)

build = ['mise','exec','--','cargo','build','--locked','-p','xtask','--bin','xtask','-p','umpire-guest-driver','--bin','umpire-guest-driver','--profile','dev','--target-dir',str(target)]
source = framework / 'dev/xtask/src/main.rs'
original = source.read_bytes()
result = dict(commands=commands, env={k:env[k] for k in ['CARGO_TARGET_DIR','CARGO_BUILD_JOBS','RUST_TEST_THREADS','CARGO_INCREMENTAL']}, source=str(source.relative_to(checkout)), mutation='retained command-help diagnostic; compiled CLI code only', source_sha256=hashlib.sha256(original).hexdigest())
try:
    run('cli-initial-build', build)
    run('cli-initial-guest', [str(target/'debug/xtask'),'guest','umpire-stock-guest',str(target/'stock.wasm'),'--check'])
    result['before'] = snapshot()
    modified = original.replace(b'commands: gate, fmt, update, time, validate,', b'commands (compiled CLI-only independence control): gate, fmt, update, time, validate,')
    assert modified != original
    source.write_bytes(modified)
    result['mutated_source_sha256'] = digest(source)
    run('cli-mutated-build', build)
    result['after_build'] = snapshot()
    assert result['before']['cli_sha256'] != result['after_build']['cli_sha256']
    assert result['before']['driver_sha256'] == result['after_build']['driver_sha256']
    run('cli-mutated-guest', [str(target/'debug/xtask'),'guest','umpire-stock-guest',str(target/'stock.wasm'),'--check'])
    result['after_guest'] = snapshot()
    assert result['before']['driver_sha256'] == result['after_guest']['driver_sha256']
    assert result['before']['lock_sha256'] == result['after_guest']['lock_sha256']
    assert result['before']['namespaces'] == result['after_guest']['namespaces']
    assert result['before']['artifacts'] == result['after_guest']['artifacts']
    assert result['before']['artifacts']
    result['status'] = 'passed'
finally:
    source.write_bytes(original)
    result['source_restored'] = digest(source) == result['source_sha256']
    (proof/'cli-independence.json').write_text(json.dumps(result,indent=2)+'\n')
run('cli-restored-build', build)
result['restored'] = snapshot()
assert result['restored']['cli_sha256'] == result['before']['cli_sha256']
(proof/'cli-independence.json').write_text(json.dumps(result,indent=2)+'\n')
