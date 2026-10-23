#!/usr/bin/env python3
"""Run the fixed all-verb lanes once and authenticate their retained raw evidence."""
import argparse
import importlib.machinery
import importlib.util
import json
from pathlib import Path
import sys

from driver import HERE, Refusal, command, configured, coverage, guard, read, require, sha
import live

OWNED = ['public-verbs/fresh-differential-and-raw-golden-integrity']
SCRIPT = HERE.parents[1] / 'scripts/all-verb-parity'


def orchestrator():
    loader = importlib.machinery.SourceFileLoader('final_all_verb', str(SCRIPT))
    spec = importlib.util.spec_from_loader(loader.name, loader)
    module = importlib.util.module_from_spec(spec)
    loader.exec_module(module)
    return module


def binding(config):
    entry = config['public_verbs']
    require(set(entry) == {'bindings'}, 'public verbs requires only the sealed binding path')
    path = Path(entry['bindings'])
    live.sealed_input(config, str(path))
    live.sealed_input(config, str(SCRIPT))
    lane = orchestrator()
    files, trees = lane.load_bindings(path, sha(path))
    require(str(files['release']) == config['binary'] and sha(files['release']) == config['binary_sha256'],
            'public verb release differs from frozen candidate')
    for item in files.values():
        live.sealed_input(config, str(item))
    declared = read(path)
    for item in declared['trees'].values():
        inventory = lane.absolute(item['inventory'], path.parent)
        live.sealed_input(config, str(inventory))
        tree = lane.absolute(item['path'], path.parent)
        for line in inventory.read_text().splitlines():
            relative = line.split(maxsplit=1)[1].lstrip(' *')
            live.sealed_input(config, str(tree / relative))
    goldens = {str(item): sha(item) for item in sorted(lane.CORPUS.rglob('*'))
               if item.is_file() and any(part.startswith('golden') for part in item.parts)}
    require(goldens, 'missing raw golden corpus')
    for item in goldens:
        live.sealed_input(config, item)
    return path, lane, goldens


def authenticate_lanes(config, work, lane):
    commands = read(work / 'commands.json')
    endpoints = [('foundation', 'foundation-check.py'), ('release-exclusion', 'foundation-release-check.py'),
                 ('inspect', 'inspect-lane.py'), ('replay', 'replay-lane.py'),
                 ('explore', 'exploration-lane.py'), ('resume', 'exploration-lane.py'),
                 ('execute-shard', 'campaign-lane.py'), ('merge', 'campaign-lane.py'),
                 ('minimize', 'minimize-reference-lane.py'), ('qualify-accounting', 'qualify-accounting-check.py'),
                 ('qualify-report', 'qualify-lane.py'), ('installation', 'installation-lane.py')]
    names = [name for name, _ in endpoints]
    names += [verb + '-' + lineage for verb in ('qualify-set', 'merge-set', 'compare-support') for lineage in ('go', 'rust')]
    names += ['stress-roles', 'stress']
    require([row['name'] for row in commands] == names, 'missing/duplicate/reordered fixed parity commands')
    endpoints += [('stress-roles', 'stress-role-check.py'), ('stress', 'stress-parity-lane.py')]
    by_name = {row['name']: row for row in commands}
    files, trees = lane.load_bindings(Path(config['public_verbs']['bindings']), sha(config['public_verbs']['bindings']))
    for name, endpoint in endpoints:
        row = by_name[name]
        argv = row['argv']
        require(argv[:2] == [config['python'], str(lane.CORPUS / endpoint)]
                and row['status'] == row['expected'] == 0, 'foreign fixed parity endpoint: ' + name)
        flags = argv[2:]
        if name == 'release-exclusion':
            require(flags == [str(files['release']), str(work / name)], 'wrong release exclusion inputs')
            continue
        require(len(flags) % 2 == 0 and len(set(flags[::2])) == len(flags[::2])
                and all(flag.startswith('--') for flag in flags[::2]), 'invalid/duplicate parity flags')
        values = dict(zip(flags[::2], flags[1::2]))
        expected_flags = {
            'foundation': 'binary work',
            'inspect': 'binary baseline source go-compiler work',
            'replay': 'binary baseline source go-compiler work stress-binary determinism-binary release-binary preparation-root cluster-baseline cluster-source',
            'explore': 'binary baseline source go-compiler verb input work preparation-root divergence-binary conformance-helper go-base',
            'resume': 'binary baseline source go-compiler verb input work preparation-root divergence-binary conformance-helper go-base',
            'execute-shard': 'binary baseline source go-compiler verb input work preparation-root',
            'merge': 'binary baseline source go-compiler verb input work preparation-root',
            'minimize': 'baseline source go-compiler rust-binary output preparation-root golden',
            'qualify-accounting': 'binary work',
            'qualify-report': 'go-binary go-helper go-source go-compiler binary input work cells preparation-root',
            'installation': 'go-binary go-helper go-source go-compiler binary work preparation-root role-binary',
            'stress-roles': 'ordinary stress diagnostic',
            'stress': 'go-binary go-helper go-source go-compiler binary input ordinary-binary work',
        }
        require(set(values) == {'--' + item for item in expected_flags[name].split()}, 'missing/unknown fixed parity flags')
        for flag, value in values.items():
            if flag in ('--source', '--go-source', '--cluster-source'):
                tree_name = 'cluster-source' if flag == '--cluster-source' else 'source'
                require(value == str(work / 'derived-sources' / tree_name), 'foreign derived Go source')
            elif flag in ('--go-binary', '--go-helper'):
                require(value == str(trees['baseline'] / 'bin' / ('gomad' if flag == '--go-binary' else 'gomad-wasmhost')),
                        'foreign original Go executable')
            elif flag == '--cells':
                require(value == 'fixture,fixture-canary,sqlite,onebox,cluster', 'partial qualification differential')
            elif flag == '--golden':
                require(value == str(lane.CORPUS / 'goldens-minimize/semantic.json'), 'foreign raw golden')
            elif flag == '--preparation-root':
                suffix = {'execute-shard': 'execute-shard', 'merge': 'merge', 'qualify-report': 'qualify'}.get(name, name)
                require(value == str(work / 'preparation' / suffix), 'foreign preparation root')
        if name not in ('stress-roles',):
            require(values.get('--work', values.get('--output')) == str(work / name), 'foreign parity output root')
        if name in ('explore', 'resume', 'execute-shard', 'merge'):
            require(values.get('--verb') == name, 'wrong distinct parity verb')
        for flag, role in (('--binary', 'release' if name == 'installation' else 'stress-release' if name == 'stress' else 'fixture'),
                           ('--rust-binary', 'fixture'), ('--ordinary', 'release'), ('--stress', 'stress-release'),
                           ('--diagnostic', 'determinism'), ('--ordinary-binary', 'release'),
                           ('--stress-binary', 'stress'), ('--determinism-binary', 'determinism'),
                           ('--release-binary', 'release'), ('--divergence-binary', 'divergence'),
                           ('--conformance-helper', 'reference-helper'), ('--go-compiler', 'compiler'),
                           ('--input', 'replay-input'), ('--go-base', 'go-base'), ('--role-binary', 'stress-release')):
            if flag in values:
                require(values[flag] == str(files[role]), 'foreign parity role input: ' + name + flag)
        for flag, tree in (('--baseline', 'baseline'), ('--cluster-baseline', 'cluster-baseline')):
            if flag in values:
                require(values[flag] == str(trees[tree]), 'foreign parity reference: ' + name)
    for verb in ('qualify-set', 'merge-set', 'compare-support'):
        for lineage, binary in (('go', trees['baseline'] / 'bin/gomad'), ('rust', files['release'])):
            name = verb + '-' + lineage
            row = by_name[name]
            require(row['argv'] == [str(binary), verb] and row['expected'] == row['status'] == 2
                    and not (work / (name + '.stdout')).read_bytes()
                    and any(marker in (work / (name + '.stderr')).read_bytes() for marker in
                            (b'unknown command', b'unknown gomad command', b'unsupported command')),
                    'missing/changed original unsupported verb comparison')
    nested = {'inspect': 'fresh-parity', 'replay': 'fresh-parity', 'explore': '', 'resume': '',
              'execute-shard': '', 'merge': '', 'minimize': '', 'qualify-report': '', 'installation': '', 'stress': ''}
    for name, relative in nested.items():
        command_root = work / name / relative
        lane.authenticate_commands(command_root, checked_in_lane=name == 'installation')
        for row in read(command_root / 'commands.json'):
            require(isinstance(row.get('argv'), list) and len(row['argv']) >= 2
                    and all(isinstance(value, str) and value for value in row['argv']),
                    'missing actual nested parity command argv')
            executable = Path(row['argv'][0])
            allowed = {str(item) for item in files.values()} | {str(trees['baseline'] / 'bin/gomad'),
                       str(trees['baseline'] / 'bin/gomad-wasmhost'), config['python']}
            require(str(executable) in allowed or executable.is_relative_to(work),
                    'foreign nested parity executable')
    receipts = {
        'inspect': ('fresh-parity/inspect-result.json', 'gomad.inspect-parity/v1'),
        'replay': ('fresh-parity/replay-reference-result.json', 'gomad.replay-reference-parity/v1'),
        'explore': ('exploration-semantic-parity.json', 'gomad.exploration-semantic-parity/v1'),
        'resume': ('exploration-semantic-parity.json', 'gomad.exploration-semantic-parity/v1'),
        'execute-shard': ('campaign-semantic-parity.json', 'gomad.campaign-semantic-parity/v1'),
        'merge': ('campaign-semantic-parity.json', 'gomad.campaign-semantic-parity/v1'),
        'minimize': ('minimize-reference-result.json', 'gomad.minimize-reference-parity/v1'),
        'stress': ('stress-parity-result.json', 'gomad.stress-semantic-parity/v1'),
    }
    for name, (relative, schema) in receipts.items():
        receipt = lane.require_receipt(work / name, relative, schema)
        require(len(receipt) > 1, 'schema-only parity receipt')
    expected_coverage = {verb: [verb] for verb in ('inspect', 'replay', 'explore', 'resume', 'execute-shard', 'merge', 'minimize', 'stress')}
    expected_coverage.update({verb: ['qualify-report'] for verb in ('qualify', 'report')})
    expected_coverage.update({verb: ['installation'] for verb in ('install', 'discover', 'doctor', 'plan', 'recover', 'analyze')})
    expected_coverage.update({verb: [dict(lineage=lineage, status=2, availability='unsupported') for lineage in ('go', 'rust')]
                              for verb in ('qualify-set', 'merge-set', 'compare-support')})
    require(read(work / 'all-verb-result.json')['coverage'] == expected_coverage, 'foreign public verb coverage attribution')


def authenticate(config, root, argv, goldens, lane):
    work = root / 'public-verbs'
    live.completed_command(read(root / 'public-verbs.command.json'), read(root / 'public-verbs.process.json'),
                           argv, config['repo'])
    report = read(work / 'all-verb-result.json')
    require(report['schema'] == 'gomad.all-verb-parity/v1' and report['status'] == 'passed'
            and report['bindings_sha256'] == sha(config['public_verbs']['bindings']), 'foreign/incomplete all-verb aggregate')
    require(set(report['coverage']) == set(lane.PUBLIC_VERBS)
            and all(report['coverage'].values()), 'missing actual public verb lanes')
    inventory = report['retained_files']
    require(inventory and 'commands.json' in inventory and 'derived-source-inputs.json' in inventory,
            'missing raw parity inventory')
    actual = {str(item.relative_to(work)) for item in work.rglob('*') if item.is_file()}
    require(actual == set(inventory) | {'all-verb-result.json'}, 'incomplete/foreign raw parity files')
    for name, digest in inventory.items():
        relative = Path(name)
        require(not relative.is_absolute() and '..' not in relative.parts, 'unsafe parity inventory')
        item = work / relative
        require(not item.is_symlink() and item.is_file() and sha(item) == digest, 'changed raw parity evidence: ' + name)
    lane.authenticate_commands(work)
    authenticate_lanes(config, work, lane)
    for path, digest in goldens.items():
        require(Path(path).is_file() and sha(path) == digest, 'raw golden corpus changed')
    # Reopen all pinned original custody/source inputs after the fresh differential.
    lane.load_bindings(Path(config['public_verbs']['bindings']), sha(config['public_verbs']['bindings']))
    rows = [dict(id=OWNED[0], status='PASS', executed=True, exit=0)]
    coverage(OWNED, rows)
    return dict(schema='gomad.rust-driver-final-section/v1', source_sha256=config['source_sha256'],
                identity=config['identity'], subgates=rows)


def collect(config, root):
    guard(config)
    require(root == Path(config['root']), 'wrong public verbs attempt root')
    path, lane, goldens = binding(config)
    python = config['python']
    live.sealed_input(config, python)
    argv = [python, str(SCRIPT), '--bindings', str(path), '--bindings-sha256', sha(path),
            '--work', str(root / 'public-verbs')]
    command(config, root, 'public-verbs', argv, timeout=config['deadline_s'])
    result = authenticate(config, root, argv, goldens, lane)
    guard(config)
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path, required=True)
    parser.add_argument('--root', type=Path, required=True)
    args = parser.parse_args()
    try:
        config = configured(args.config)
        config.update(_config_path=str(args.config.resolve()), _config_sha256=sha(args.config))
        print(json.dumps(collect(config, args.root.resolve()), separators=(',', ':')))
        return 0
    except (Refusal, OSError, ValueError, KeyError, AssertionError) as error:
        print('REFUSED: ' + str(error), file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
