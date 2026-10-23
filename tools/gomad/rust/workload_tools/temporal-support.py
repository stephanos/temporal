#!/usr/bin/env python3
"""Execute the original fn222 smoke, detached inspection and frontend receiving gate."""
import argparse
import hashlib
import json
import os
import re
from pathlib import Path
import sys
sys.dont_write_bytecode=True
import producer
import workloads


def selected_pass(raw, row):
    selected = "TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds" if row == "smoke" else "TestFrontendSystemInfo"
    marker = re.compile(rb'^[ \t]*--- PASS: ' + re.escape(selected.encode()) + rb' \([0-9]+(?:\.[0-9]+)?s\)[ \t]*$', re.M)
    if len(marker.findall(raw)) != 1 or b"--- SKIP:" in raw or b"no tests to run" in raw:
        raise ValueError("original exact selected workflow/frontend test did not pass exactly once")


def gate_environment(row, measured=False):
    environment = dict(os.environ)
    environment.pop('GOMAD_WASM_QUALIFY_WORKLOADS', None)
    if row == 'smoke' and measured:
        environment['GOMAD_WASM_QUALIFY_WORKLOADS'] = '1'
    return environment


def bounded_runner():
    import importlib.util
    path = Path(__file__).resolve().parents[2] / 'testdata/parity/onebox-gates.py'
    spec = importlib.util.spec_from_file_location('fn222_ordered_onebox', path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['binary', 'repo', 'root', 'compiler', 'runtime-root', 'cache', 'captured-config', 'source-manifest']:
        parser.add_argument('--' + name, required=True, type=Path)
    parser.add_argument('--private-observation-config', type=Path)
    parser.add_argument('--support', type=Path)
    parser.add_argument('--prepare-only', action='store_true')
    args = parser.parse_args()
    command = ['onebox-gates.py']
    for name in ['binary', 'repo', 'root', 'compiler', 'runtime-root', 'cache', 'captured-config', 'source-manifest']:
        command += ['--' + name, str(getattr(args, name.replace('-', '_')).resolve())]
    support = args.support or args.binary.resolve().parent / 'examples/fn222-support'
    command += ['--support', str(support.resolve()), '--inspection', str(Path(__file__).with_name('temporal-inspection.py').resolve())]
    if args.private_observation_config:
        command += ['--private-observation-config', str(args.private_observation_config.resolve())]
    if args.prepare_only:
        command.append('--prepare-only')
    bounded_runner().main(command[1:])


if __name__ == '__main__':
    main()
