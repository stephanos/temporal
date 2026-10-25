#!/usr/bin/env python3
"""Task-local stable-libtest and Java launch observers; never a production runner."""
import hashlib
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import sys
import time
import uuid


def write(path, data):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    temp = path.with_suffix('.tmp')
    temp.write_text(json.dumps(data, indent=2) + '\n')
    temp.replace(path)


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def finish_status(status):
    if status < 0:
        signal.signal(-status, signal.SIG_DFL)
        os.kill(os.getpid(), -status)
    return status


def launch(args, serial=False):
    root = Path(os.environ['FN254_OUT'])
    token = uuid.uuid4().hex
    path = root / 'binaries' / (token + '.json')
    binary = str(Path(args[0]).resolve())
    context = root / 'contexts' / (token + '.json')
    env = dict(os.environ, FN254_BINARY=binary, FN254_CONTEXT=str(context))
    argv = args + (['--format=pretty', '--test-threads=1'] if serial else [])
    record = dict(binary=binary, argv=argv, cwd=os.getcwd(), pid=os.getpid(),
                  binary_sha256=digest(binary), mode='serial-observer' if serial else 'cargo-runner',
                  started_ns=time.monotonic_ns(), tests=[], exit_code=None)
    write(path, record)
    if not serial:
        status = subprocess.call(argv, env=env)
    else:
        proc = subprocess.Popen(argv, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        buffer = b''
        active = None
        raw = root / 'binary-output' / (token + '.log')
        raw.parent.mkdir(parents=True, exist_ok=True)
        with raw.open('wb') as output:
            while True:
                chunk = os.read(proc.stdout.fileno(), 4096)
                if not chunk:
                    break
                now = time.monotonic_ns()
                output.write(chunk)
                output.flush()
                buffer += chunk
                while True:
                    if active is None:
                        match = re.search(rb'(?:^|\n)test ([^\r\n]+?) \.\.\. ', buffer)
                        if not match:
                            buffer = buffer[-65536:]
                            break
                        active = dict(name=match[1].decode(errors='replace'), started_ns=now)
                        write(context, active)
                        buffer = buffer[match.end():]
                    match = re.search(rb'(?:^|\n)(ok|FAILED|ignored)(?:[^\r\n]*)\r?\n', buffer)
                    if not match:
                        break
                    active.update(ended_ns=now, seconds=(now-active['started_ns'])/1e9,
                                  status=match[1].decode())
                    record['tests'].append(active)
                    active = None
                    buffer = buffer[match.end():]
            if active is not None:
                active.update(status='incomplete', seconds=None)
                record['tests'].append(active)
        status = proc.wait()
        record['raw_output'] = str(raw)
    record.update(exit_code=status, ended_ns=time.monotonic_ns())
    record['seconds'] = (record['ended_ns'] - record['started_ns']) / 1e9
    write(path, record)
    return finish_status(status)


def java(args):
    root = Path(os.environ['FN254_OUT'])
    token = uuid.uuid4().hex
    directory = root / 'java' / token
    directory.mkdir(parents=True)
    kind = 'tlc-model' if 'tlc2.TLC' in args and '-tool' in args else (
        'tlc-admission' if 'tlc2.TLC' in args else 'java-admission')
    context_path = os.environ.get('FN254_CONTEXT')
    context = json.loads(Path(context_path).read_text()) if context_path and Path(context_path).exists() else None
    record = dict(kind=kind, argv=args, cwd=os.getcwd(), pid=os.getpid(),
                  binary=os.environ.get('FN254_BINARY'), context=context,
                  started_ns=time.monotonic_ns(), exit_code=None, files=[])
    if kind == 'tlc-model':
        candidates = [Path(args[-1] + '.tla')]
        if '-config' in args:
            candidates.append(Path(args[args.index('-config')+1]))
        for source in candidates:
            if source.is_file():
                content = source.read_bytes()
                (directory / source.name).write_bytes(content)
                record['files'].append(dict(name=source.name, sha256=hashlib.sha256(content).hexdigest()))
    write(directory / 'event.json', record)
    status = subprocess.call([os.environ['FN254_REAL_JAVA']] + args)
    record.update(exit_code=status, ended_ns=time.monotonic_ns())
    record['seconds'] = (record['ended_ns'] - record['started_ns']) / 1e9
    write(directory / 'event.json', record)
    return finish_status(status)


def mise(args):
    if args == ['which', 'java']:
        root = Path(os.environ['FN254_OUT'])
        write(root / 'mise-resolution' / (uuid.uuid4().hex + '.json'),
              dict(argv=args, cwd=os.getcwd(), at_ns=time.monotonic_ns()))
        print(Path(__file__).absolute().with_name('java'))
        return 0
    os.execv(os.environ['FN254_REAL_MISE'], [os.environ['FN254_REAL_MISE']] + args)


if __name__ == '__main__':
    mode = Path(sys.argv[0]).name
    if mode == 'mise':
        result = mise(sys.argv[1:])
    elif mode == 'java':
        result = java(sys.argv[1:])
    elif mode == 'cargo-runner':
        result = launch(sys.argv[1:])
    elif mode == 'serial-runner':
        result = launch(sys.argv[1:], serial=True)
    else:
        raise SystemExit('invoke through mise/java/cargo-runner/serial-runner symlink')
    raise SystemExit(result)
