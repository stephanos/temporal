#!/usr/bin/env python3
"""Run and independently read the immutable original terminal wire controls."""
import argparse
import json
from pathlib import Path
import sys

import final_profiles
import live
import original_wire_cache
from driver import Refusal, command, configured, read, require, sha, write

PACKAGE = 'go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi'
ROOTS = (
    'TestTerminalExitAuthorizationMismatchAfterStarted',
    'TestTerminalExitExpiredContextDoesNotLaunchHelper',
    'TestTerminalExitMeasurementControls',
    'TestTerminalExitMeasurementSendsCapturedInputsBeforeStarted',
    'TestTerminalExitObserverReceivesCompleteAcceptedFrameBeforeValidation',
)
CASES = (
    *(ROOTS[2] + '/' + name for name in ('prompt-exit', 'slow-preparation', 'delayed-exit',
      'missing-terminal', 'invalid-terminal', 'helper-death', 'post-terminal-watchdog')),
    *(ROOTS[3] + '/' + name for name in ('stock-nil', 'local-nil', 'local-empty',
      'local-empty-string', 'local-populated')),
)


def retained(config, root, name, argv):
    receipt = read(root / (name + '.command.json'))
    process = read(root / (name + '.process.json'))
    live.completed_command(receipt, process, argv, config['repo'])
    for stream in ('stdout', 'stderr'):
        require((root / (name + '.' + stream)).is_file(), 'missing original control command stream')
    require(receipt['environment'] == config.get('environment', {})
            and receipt['temporary_paths'] == config.get('temporary_paths', {}),
            'changed original control command environment')
    return (root / (name + '.stdout')).read_bytes(), process


def run(config, root, name, argv):
    command(config, root, name, argv, timeout=905)
    return retained(config, root, name, argv)


def events(raw):
    """The original assertions own semantics; authenticate their complete execution."""
    observed = {name: [] for name in (*ROOTS, *CASES)}
    package = []
    for line in raw.splitlines():
        event = json.loads(line)
        require(event.get('Package') == PACKAGE, 'foreign original control package')
        action, test = event.get('Action'), event.get('Test')
        require(action not in ('fail', 'skip'), 'failed/skipped original terminal control')
        if test:
            require(test in observed, 'unexpected original terminal control')
            if action in ('run', 'pass'):
                observed[test].append(action)
            else:
                require(action == 'output', 'unexpected terminal control event')
        elif action in ('start', 'pass'):
            package.append(action)
        else:
            require(action in ('output', 'build-output'), 'unexpected package event')
    require(package == ['start', 'pass'] and all(row == ['run', 'pass'] for row in observed.values()),
            'missing/duplicate/unexecuted original terminal control')
    return list(observed)


def compiler_identity(goroot, version, raw):
    compiler = json.loads(raw)
    version_fields = version.decode().strip().split()
    require(len(version_fields) == 4 and version_fields[:3] == ['go', 'version', 'go1.27.1'], 'foreign stock compiler version')
    target = version_fields[3].split('/')
    require(len(target) == 2 and all(value and value.replace('_', '').isalnum() for value in target), 'malformed actual stock compiler host')
    require(compiler == dict(GOROOT=str(goroot), GOVERSION='go1.27.1',
                             GOTOOLDIR=str(goroot / ('pkg/tool/' + '_'.join(target)))),
            'foreign stock compiler installation/host')
    return compiler


def collect(config, root, produce):
    entry = config['terminal_controls_reference']
    execute = run if produce else retained
    derived, source = final_profiles.original_source(config, root, entry, 'original-tree', execute)
    module_cache = original_wire_cache.binding(config, entry)
    go, goroot = entry['go'], Path(entry['go_root'])
    live.sealed_input(config, str(goroot))
    require(Path(go) == goroot / 'bin/go', 'original controls require admitted stock Go installation')
    cwd = str(derived / 'source/tools/gomad/internal/backends/wasm')
    scratch = root / 'scratch'
    if produce:
        scratch.mkdir()
        (scratch / 'go-cache').mkdir()
    require(scratch.is_dir() and not scratch.is_symlink(), 'missing original control scratch')
    environment = original_wire_cache.environment(goroot, scratch, module_cache['root'])
    child = dict(config, repo=cwd, environment=environment, temporary_paths={})
    version, _ = execute(child, root, 'go-version', [go, 'version'])
    require(version.startswith(b'go version go1.27.1 '), 'changed original Go version')
    compiler, _ = execute(child, root, 'go-environment', [go, 'env', '-json', 'GOROOT', 'GOVERSION', 'GOTOOLDIR'])
    compiler = compiler_identity(goroot, version, compiler)
    argv = [go, 'test', '-json', '-v', '-tags', 'test_dep', '-count=1', '-timeout=15m',
            '-run', '^TestTerminalExit', './wasi']
    raw, process = execute(child, root, 'original-terminal-controls', argv)
    names = events(raw)
    require(original_wire_cache.binding(config, entry) == module_cache, 'original terminal dependency cache changed')
    proof = dict(schema='gomad.original-terminal-controls/v1', original_commit=source['original_commit'],
                 original_source=str(derived / 'source.json'), original_source_sha256=sha(derived / 'source.json'),
                 compiler=compiler, compiler_sha256=sha(go), testcases=names, module_cache=module_cache,
                 pid=process['pid'], scope='original framed-helper wire applicability; test-owned hashed shell helpers',
                 source_sha256=config['source_sha256'], identity=config['identity'],
                 helper_factory_sha256=sha(Path(cwd) / 'wasi/process_test.go'),
                 commands={name: {kind: sha(root / (name + '.' + kind))
                                  for kind in ('command.json', 'process.json', 'stdout', 'stderr')}
                           for name in ('original-tree', 'go-version', 'go-environment', 'original-terminal-controls')})
    if produce:
        write(root / 'original-terminal-controls.json', proof)
    else:
        require(read(root / 'original-terminal-controls.json') == proof, 'changed original control proof')
    return dict(schema='gomad.rust-driver-check/v1', stage='terminal-controls',
                source_sha256=config['source_sha256'], identity=config['identity'],
                subgates=[dict(id=name, status='PASS', executed=True, exit=0) for name in ROOTS],
                original_wire_applicability=proof)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('produce', 'check'))
    parser.add_argument('--config', type=Path, required=True)
    parser.add_argument('--root', type=Path, required=True)
    args = parser.parse_args()
    try:
        config = configured(args.config)
        root = args.root.resolve()
        require(root.is_relative_to(Path(config['root']).resolve()) and root.is_dir(), 'foreign control output root')
        print(json.dumps(collect(config, root, args.mode == 'produce'), separators=(',', ':')))
        return 0
    except (Refusal, OSError, ValueError, KeyError, TypeError) as error:
        print('REFUSED: ' + str(error), file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
