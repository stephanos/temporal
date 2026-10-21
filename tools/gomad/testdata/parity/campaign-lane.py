#!/usr/bin/env python3
"""Run fresh campaign semantics and the independent exact Rust golden lane."""
import argparse
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
import os
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent


def load(name, path):
    loader = SourceFileLoader(name, str(path))
    module = module_from_spec(spec_from_loader(loader.name, loader))
    loader.exec_module(module)
    return module


check = load('campaign_check', CORPUS / 'campaign-check.py')
reference = load('campaign_reference', CORPUS / 'campaign-reference-check.py')


def semantic(validated):
    groups = {}
    records = []
    for execution in sorted(validated['executions'], key=lambda row: row['ordinal']):
        record = execution['record']
        signature = check.parity.digest(check.parity.canonical(dict(record, seed=0)))
        group = None
        if record['outcome']['class'] != 'success':
            group = groups.setdefault(signature, execution['ordinal'])
        records.append(dict(ordinal=execution['ordinal'], seed=execution['seed'],
            outcome=record['outcome']['class'], termination=record['outcome']['termination'], failure_group=group))
    return records


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'baseline', 'source', 'go-compiler', 'work', 'input', 'preparation-root'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--verb', choices=('explore', 'resume', 'execute-shard', 'merge'), default='explore')
    args = parser.parse_args()
    for name in ('binary', 'baseline', 'source', 'go_compiler', 'work', 'input', 'preparation_root'):
        setattr(args, name, getattr(args, name).resolve())
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_')}
    if os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1':
        environment['GOMAD_GOLDEN_REGENERATE'] = '1'
    receipts = []

    def run(name, command, status=0, extra=None):
        result = subprocess.run(list(map(str, command)), env=dict(environment, **(extra or {})), capture_output=True, timeout=2400)
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        receipts.append(dict(name=name, argv=list(map(str, command)), status=result.returncode, expected=status,
            stdout_sha256=check.parity.digest(result.stdout), stderr_sha256=check.parity.digest(result.stderr)))
        (args.work / 'commands.json').write_bytes(check.parity.canonical(receipts) + b'\n')
        check.equal(result.returncode, status, name + ': ' + result.stderr.decode(errors='replace'))
        return result.stdout

    go = args.work / 'go'
    run('go-reference', [sys.executable, CORPUS / 'campaign-reference-check.py', '--baseline', args.baseline,
        '--source', args.source, '--go-compiler', args.go_compiler, '--work', go,
        '--preparation-root', args.preparation_root / 'go'])
    rust = args.work / 'rust'
    run('rust-public-and-goldens', [sys.executable, CORPUS / 'campaign-check.py', '--binary', args.binary,
        '--go-compiler', args.go_compiler, '--work', rust, '--input', args.input,
        '--preparation-root', args.preparation_root / 'rust'])
    declaration = check.parity.read(CORPUS / 'campaign-corpus.json')
    expected = check.parity.read(go / 'campaign-reference.json')
    check.equal(expected['schema'], 'gomad.campaign-go-reference/v1', 'reference.schema')
    check.equal(expected['binary_sha256'], check.parity.digest((args.baseline / 'bin/gomad').read_bytes()), 'reference.binary')
    check.equal(expected['driver_sha256'], check.parity.digest((CORPUS / 'campaign-reference-check.py').read_bytes()), 'reference.driver')
    prefix = [args.binary, '--installation', rust / 'installation.json']
    module = CORPUS.parents[1]
    observations = {}

    def locate(destination):
        roots = list((destination / 'campaigns').glob('campaign-*'))
        check.equal(len(roots), 1, 'rust.campaign_root')
        return roots[0]

    for case in declaration['cases']:
        name = case['name']
        original_root = locate(rust / (name + '-0'))
        original = check.authenticate_campaign(original_root)
        if name == 'runtime':
            resumed_root = locate(rust / 'interrupted')
            shard_roots = [locate(rust / ('shard-' + str(index))) for index in range(2)]
            merged_root = rust / 'merged'
        else:
            destination = args.work / (name + '-interrupted')
            command = [*prefix, 'explore', '--json', '--choices', '--parallel=1', '--on-failure=all',
                '--seeds=' + declaration['seeds'], '--keep-successes=all', '--success-limit=8', '--success-bytes=256MiB',
                '--execution-timeout=30s', '--overall-timeout=10m', '--artifacts', destination,
                '--working-dir', module / case['module'], *['--build-tag=' + tag for tag in case['tags']],
                case['kind'], case['source'], '--', *case['arguments']]
            run(name + '-interrupt', command, 86, {'GOMAD_FIXTURE_CAMPAIGN_INTERRUPT_AFTER': '2'})
            resumed_root = locate(destination)
            check.authenticate_campaign(resumed_root, complete=False)
            run(name + '-resume', [*prefix, 'resume', '--json', resumed_root], case['status'])
            shard_roots = []
            for index in range(2):
                destination = args.work / f'{name}-shard-{index}'
                run(f'{name}-shard-{index}', [*prefix, 'execute-shard', '--json', '--shard', f'{index}/2',
                    '--artifacts', destination, original_root / 'plan.json'], case['status'])
                shard_roots.append(locate(destination))
            merged_root = args.work / (name + '-merged')
            run(name + '-merge', [*prefix, 'merge', '--json', '--output', merged_root, original_root / 'plan.json', *shard_roots])
        shard_executions = []
        for root in shard_roots:
            shard_executions.extend(check.authenticate_campaign(root)['executions'])
        observations[name] = dict(explore=semantic(original), resume=semantic(check.authenticate_campaign(resumed_root)),
            execute_shard=semantic({'executions': shard_executions}), merge=semantic(check.authenticate_campaign(merged_root)))
    check.parity.check_equal(expected['observations'], observations)
    result = dict(schema='gomad.campaign-semantic-parity/v1', requested_verb=args.verb,
        go_surface=expected['surface'], rust_surface='public CLI for all four verbs', observations=observations,
        public_cli_positive_parity=False,
        exception='Frozen Go WASM plan/execute-shard/merge refuse status 2; positive shard/merge comparison uses unchanged original Runner API')
    (args.work / 'campaign-semantic-parity.json').write_bytes(check.parity.canonical(result) + b'\n')
    print('campaign semantic lane: fresh public explore/resume and original Go Runner API shard/merge match Rust public verbs; public CLI surface discrepancy retained')


if __name__ == '__main__':
    main()
