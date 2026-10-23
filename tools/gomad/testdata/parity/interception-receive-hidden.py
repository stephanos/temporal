#!/usr/bin/env python3
"""Retained-only replay in a namespace masking the genuine original build inputs."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys
import time

sys.dont_write_bytecode = True
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader

loader = SourceFileLoader('receiving_hidden_base', str(Path(__file__).with_name('interception-receive.py')))
base = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(base)
equal = base.equal


def protected(path, directories):
    return any(Path(path).is_relative_to(Path(directory)) for directory in directories)


def namespace_argv(inputs, spec_path, root, artifacts):
    hidden = inputs['hidden_directories']
    files = inputs['hidden_installations']
    if not hidden or not files or len(set(hidden + files)) != len(hidden + files):
        raise ValueError('distinct genuine hidden inputs required')
    for name in hidden + files:
        path = Path(name)
        if not path.is_absolute() or len(path.parts) < 4 or path.is_symlink():
            raise ValueError('unsafe hidden input path')
        if name in hidden and not path.is_dir():
            raise ValueError('hidden build directory missing')
        if name in files:
            equal(inputs['files'][name], base.sha(path), 'hidden.installation_custody')
    if any(protected(name, hidden + files) for name in
           [inputs['binary'], inputs['fixture_binary'], inputs['bwrap'], inputs['python'],
            str(spec_path), str(root), *artifacts]):
        raise ValueError('hidden inputs cover replay executable, harness or retained evidence')
    if any(protected(name, [other]) for name in hidden + files for other in hidden + files if name != other):
        raise ValueError('overlapping hidden mount declarations')
    for artifact in artifacts:
        provenance = base.check.campaign.canonical_file(Path(artifact) / 'target/provenance.json')
        compiler_path = provenance['inputs']['compiler_path']
        if not protected(compiler_path, hidden):
            raise ValueError('hidden mounts do not cover original compiler executable')
        for kind in ('compiler', 'sources'):
            paths = provenance['inputs'][kind]
            if not paths or not all(protected(path, hidden) for path in paths):
                raise ValueError('hidden mounts do not cover original ' + kind + ' inputs')
    if inputs['installation'] not in files:
        raise ValueError('selected fixture installation must be masked')
    for name in (inputs['checkout'], inputs['cache'], inputs['compiler']):
        if not protected(name, hidden):
            raise ValueError('hidden mounts do not cover original checkout/cache/compiler')
    for name in files:
        installation = base.read(Path(name))
        for key in ('cache_root', 'runtime_root', 'overlay_module_root', 'compiler_path'):
            if not protected(installation[key], hidden):
                raise ValueError('hidden mounts do not cover installation ' + key)
        compiler_inputs = installation['admission']['compiler_inputs']
        if not compiler_inputs or not all(protected(path, hidden) for path in compiler_inputs):
            raise ValueError('hidden mounts do not cover installation compiler inputs')
    argv = [inputs['bwrap'], '--die-with-parent', '--unshare-user', '--unshare-pid', '--unshare-net',
            '--ro-bind', '/', '/', '--proc', '/proc', '--dev', '/dev']
    for directory in hidden:
        argv += ['--tmpfs', directory]
    for path in files:
        argv += ['--ro-bind', str(root / 'empty'), path]
    return argv + ['--bind', str(root / 'inside'), str(root / 'inside'), '--chdir', '/',
                   inputs['python'], str(root / 'inner.py'), str(root / 'inner.json')]


INNER_SOURCE = '''import hashlib,json,os,subprocess,sys,time
from pathlib import Path

def read(path):
    return json.loads(Path(path).read_bytes())

def write(path,value):
    with path.open('x') as output:
        json.dump(value,output,sort_keys=True,separators=(',',':'))
    path.chmod(0o600)

def sha(path):
    with open(path,'rb') as source:
        return hashlib.file_digest(source,'sha256').hexdigest()

def inner(path):
    spec = read(path)
    root = Path(spec['root'])
    checks = {}
    for name in spec['directories']:
        checks[name] = Path(name).is_dir() and not list(Path(name).iterdir())
    for name in spec['installations']:
        checks[name] = Path(name).is_file() and Path(name).stat().st_size == 0
    write(root / 'inputs-unavailable.json', checks)
    if not all(checks.values()):
        raise ValueError('original build inputs still available inside replay namespace')
    for index, argv in enumerate(spec['commands']):
        # A single external admission covers every replay; inner timeout cannot reset it.
        remaining = (spec['deadline_ns'] - time.monotonic_ns()) / 1e9
        if remaining <= 0:
            raise ValueError('shared receiving deadline expired before hidden replay')
        with (root / (str(index) + '.stdout')).open('xb') as stdout, (root / (str(index) + '.stderr')).open('xb') as stderr:
            child = subprocess.Popen(argv, cwd='/', stdin=subprocess.DEVNULL, stdout=stdout, stderr=stderr)
            write(root / (str(index) + '.started.json'), dict(pid=child.pid, argv=[os.fsencode(value).hex() for value in argv],
                                                                monotonic_ns=time.monotonic_ns()))
            try:
                status = child.wait(timeout=remaining)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait()
                raise ValueError('shared receiving deadline expired during hidden replay')
        write(root / (str(index) + '.result.json'), dict(status=status, monotonic_ns=time.monotonic_ns(),
                   stdout_sha256=sha(root / (str(index) + '.stdout')),
                   stderr_sha256=sha(root / (str(index) + '.stderr'))))
        if status != 0:
            raise ValueError('hidden actual replay failed')


inner(Path(sys.argv[1]))
'''


def prerequisites(inputs):
    for key in ('bwrap', 'python'):
        name = inputs[key]
        equal(base.sha(name), inputs['files'][name], 'hidden.pinned_' + key)


def retained(inputs_path, owner_root):
    inputs = base.pin_inputs(inputs_path)
    rows = []
    for case in ('manual', 'parallel'):
        proof = base.collect(inputs_path, case, owner_root / case)
        rows.extend(proof['artifacts'])
    equal(len(rows), 3, 'hidden.three_actual_go_artifacts')
    return inputs, rows


def commands(inputs, artifacts):
    return [[inputs['fixture_binary'], 'replay', artifact] for artifact in artifacts]


def collect(inputs_path, owner_root, root):
    inputs, artifacts = retained(inputs_path, owner_root)
    prerequisites(inputs)
    admission = base.read(root / 'admission.json')
    equal(admission['inputs_sha256'], base.sha(inputs_path), 'hidden.input_custody')
    equal(admission['root'], str(root), 'hidden.caller_bound_root')
    equal(admission['owner_root'], str(owner_root), 'hidden.caller_bound_owner')
    if not 0 < admission['deadline_ns'] - admission['started_ns'] <= 600_000_000_000:
        raise ValueError('hidden receiving budget exceeds original bounds')
    declaration = dict(root=str(root / 'inside'), directories=inputs['hidden_directories'],
                       installations=inputs['hidden_installations'], commands=commands(inputs, artifacts),
                       deadline_ns=admission['deadline_ns'])
    equal(base.read(root / 'inner.json'), declaration, 'hidden.exact_inner_commands')
    equal((root / 'empty').read_bytes(), b'', 'hidden.empty_file_mount')
    equal((root / 'inner.py').read_text(), INNER_SOURCE, 'hidden.exact_retained_namespace_program')
    argv = namespace_argv(inputs, inputs_path, root, artifacts)
    base.authenticate_command(root / 'run', argv, inputs, admission['deadline_ns'])
    equal(base.read(root / 'inside/inputs-unavailable.json'),
          {path: True for path in inputs['hidden_directories'] + inputs['hidden_installations']},
          'hidden.actual_inside_unavailability')
    for index, argv in enumerate(commands(inputs, artifacts)):
        prefix = root / 'inside' / str(index)
        started = base.read(Path(str(prefix) + '.started.json'))
        result = base.read(Path(str(prefix) + '.result.json'))
        equal(started['argv'], base.encoded(argv), 'hidden.retained_only_no_installation_selector')
        if not 0 < started['pid'] or not admission['started_ns'] <= started['monotonic_ns'] < result['monotonic_ns'] <= admission['deadline_ns']:
            raise ValueError('hidden replay lacks bounded actual process custody')
        equal(result['status'], 0, 'hidden.actual_replay_status')
        for stream, expected in [('stdout', base.EXACT_REPLAY), ('stderr', b'')]:
            path = Path(str(prefix) + '.' + stream)
            equal(path.read_bytes(), expected, 'hidden.exact_reproduction_' + stream)
            equal(base.sha(path), result[stream + '_sha256'], 'hidden.raw_stream_hash')
    equal((root / 'run/stdout').read_bytes(), b'', 'hidden.no_extra_outer_stdout')
    equal((root / 'run/stderr').read_bytes(), b'', 'hidden.no_extra_outer_stderr')
    return dict(schema='gomad.interception-hidden-receiving/v1', accepted=True, artifacts=artifacts,
                hidden_inputs=inputs['hidden_directories'] + inputs['hidden_installations'],
                exact_retained_only_replays=3, final_qualification=False)


def produce(inputs_path, owner_root, root):
    inputs, artifacts = retained(inputs_path, owner_root)
    prerequisites(inputs)
    root.mkdir(mode=0o700)
    (root / 'inside').mkdir(mode=0o700)
    with (root / 'inner.py').open('x') as output:
        output.write(INNER_SOURCE)
    (root / 'inner.py').chmod(0o400)
    with (root / 'empty').open('xb'):
        pass
    start = time.monotonic_ns()
    admission = dict(inputs_sha256=base.sha(inputs_path), root=str(root), owner_root=str(owner_root),
                     started_ns=start, deadline_ns=start + 600_000_000_000)
    base.write(root / 'admission.json', admission)
    base.write(root / 'inner.json', dict(root=str(root / 'inside'), directories=inputs['hidden_directories'],
               installations=inputs['hidden_installations'], commands=commands(inputs, artifacts),
               deadline_ns=admission['deadline_ns']))
    argv = namespace_argv(inputs, inputs_path, root, artifacts)
    base.run(argv, inputs, root / 'run', admission['deadline_ns'])
    result = collect(inputs_path, owner_root, root)
    base.write(root / 'proof.json', result)
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--inputs', type=Path)
    parser.add_argument('--owner-root', type=Path)
    parser.add_argument('--root', type=Path)
    parser.add_argument('--read-only', action='store_true')
    args = parser.parse_args()
    if any(value is None or not value.is_absolute() for value in (args.inputs, args.owner_root, args.root)):
        parser.error('absolute --inputs, --owner-root and --root required')
    result = collect(args.inputs, args.owner_root, args.root) if args.read_only else produce(args.inputs, args.owner_root, args.root)
    print(json.dumps(result, sort_keys=True))


if __name__ == '__main__':
    try:
        main()
    except (ValueError, KeyError, OSError, TypeError) as error:
        print('interception hidden receiving refused: ' + str(error), file=sys.stderr)
        raise SystemExit(1)
