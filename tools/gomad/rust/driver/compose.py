#!/usr/bin/env python3
"""Compose reviewed static bindings with typed runtime inputs, refusing gaps before freeze."""
import argparse
import copy
import re
from pathlib import Path
import sys
import tempfile

from driver import HERE, REPO, Refusal, deadline, mapping, read, require, write
from adapters import CELLS
import final
import live
import static_stages
import primary_stages

SECTIONS = {
    '10-stress': ('stress-parity-lane.py', ['stress/41-cases-four-axes-original-repetitions']),
    '20-profiles': ('final_profiles.py', [
        'terminal/current-host-admission-and-linked-attribution',
        'fn185-R4/runtime-cost-equivalent-release-behavior',
        'fn185-R4/qualified-workload-choice-wasi-network-stdout-stderr-preservation']),
    '40-retained': ('retained', ['retained/20-fresh-observed-replays']),
    '50-operator': ('operator', ['operator/detached-installed-discover-inspect-replay']),
    '60-inspection': ('inspection', ['inspection/rotated-three-node-ringpop-120s-fresh-exact-replay']),
    '90-public-verbs': ('final_public_verbs.py', ['public-verbs/fresh-differential-and-raw-golden-integrity']),
}


def final_sections(python, config_path):
    result = {}
    for name, (endpoint, required) in SECTIONS.items():
        if endpoint == 'stress-parity-lane.py':
            argv = [python, str(live.PARITY / endpoint), '--final-section', '--produce', '--config', str(config_path)]
        elif endpoint in ('retained', 'operator', 'inspection'):
            argv = [python, str(HERE / 'final_retained.py'), '--config', str(config_path), '--section', endpoint]
            if endpoint != 'inspection':
                argv.append('--read-only')
        else:
            argv = [python, str(HERE / endpoint), '--config', str(config_path)]
        result[name] = dict(required=required, argv=argv)
    observed = [item for entry in result.values() for item in entry['required']]
    require(len(observed) == len(set(observed)) and set(observed) == final.remaining_ids(), 'changed final endpoint ownership')
    return result


def validate_stages(config):
    require(set(config['stages']) == set(mapping()), 'missing/foreign original stage labels')
    require(set(config.get('identity', {})) == {'host', 'helper'}
            and all(re.fullmatch('[0-9a-f]{64}', value) and value != '0' * 64
                    for value in config['identity'].values()), 'unknown candidate host/helper identities')
    gaps = []
    for label, entry in config['stages'].items():
        adapter = entry.get('adapter')
        if adapter == 'pending':
            gaps.append(label)
            continue
        if label in primary_stages.OWNED:
            require(adapter == 'primary' and set(entry) <= {'adapter', 'live_families'}, 'fixed primary adapter required: ' + label)
            continue
        expected = ('runtime' if label.startswith('runtime-') and label != 'runtime-aggregate' else
                    'runtime-aggregate' if label == 'runtime-aggregate' else
                    'qualify' if label in CELLS else
                    'sqlite-selection' if label.startswith('sqlite-selection-') else
                    'sqlite-catalogue' if label == 'sqlite-catalogue' else
                    'corpus' if label in ('workload-corpus', 'workload-corpus-after-relocation') else
                    'plan' if label == 'terminal-module' else
                    'retained' if label in ('artifact', 'operator', 'retained-cluster', 'retained-durable-restart') else 'checked-command')
        require(adapter == expected, 'wrong semantic adapter: ' + label)
        if adapter == 'checked-command':
            require(entry.get('checker') and not entry.get('reader'), 'actual semantic reader required: ' + label)
            require(label in static_stages.LABELS, 'missing closed original primary recipe: ' + label)
            endpoint = HERE / 'static_stages.py'
            for key in ('argv', 'checker'):
                argv = entry[key]
                require(len(argv) >= 2 and argv[0] == config['python'] and Path(argv[1]).is_file(),
                        'missing actual Python endpoint: ' + label)
                require(Path(argv[1]).resolve() == endpoint, 'foreign stage endpoint: ' + label)
            require(0 < entry['timeout_s'] <= 3600, 'invalid stage timeout')
        elif adapter == 'runtime':
            for key in ('harness', 'harness_build_receipt'):
                require(Path(entry[key]).is_file(), 'missing actual runtime input: ' + label)
        elif adapter == 'qualify':
            require(entry['spec'] == 'GENERATE' or Path(entry['spec']).is_file(), 'missing workload scenario: ' + label)
            require(entry.get('private_observations') is True, 'actual native preparation capture required: ' + label)
            if CELLS[label]['workload'] in ('sqlite', 'multi-instance'):
                require(entry['checker'] == [config['python'], str(HERE / 'cell_reader.py'), '--config',
                                             config['_future_config'], '--stage', label], 'foreign durable cell reader')
        elif adapter == 'sqlite-catalogue':
            require(entry.get('oracle'), 'actual independent SQLite oracle required')
        elif adapter in ('retained', 'plan'):
            require(False, 'missing closed original primary recipe: ' + label)
    require(not gaps, 'unbound original stage endpoints: ' + ', '.join(gaps))


def bind_primitives(config, config_path):
    runtime = config.get('runtime', {})
    require(isinstance(runtime, dict) and set(runtime) == {'harness', 'harness_build_receipt'}, 'sealed ordinary runtime harness inputs required')
    for index in range(32):
        config['stages'][f'runtime-{index}'] = dict(adapter='runtime', installation='cooperative-runtime',
            source='./internal/backends/wasm/testdata/runtime_probes', arguments=['--working-dir', str(REPO / 'tools/gomad')],
            **runtime)
    config['stages']['runtime-aggregate'] = dict(adapter='runtime-aggregate')
    for label, wanted in CELLS.items():
        installation = wanted['workload'] + '-' + wanted['profile']
        require(installation in config['installations'], 'missing actual mandatory installation: ' + installation)
        entry = dict(adapter='qualify', installation=installation, spec='GENERATE', private_observations=True)
        if wanted['workload'] in ('sqlite', 'multi-instance'):
            entry['checker'] = [config['python'], str(HERE / 'cell_reader.py'), '--config', str(config_path), '--stage', label]
        config['stages'][label] = entry
    for label in ('workload-corpus', 'workload-corpus-after-relocation'):
        config['stages'][label] = dict(adapter='corpus')
    for index in range(4):
        config['stages'][f'sqlite-selection-{index}'] = dict(adapter='sqlite-selection', installation='sqlite-cooperative')
    oracle = config.get('sqlite_oracle')
    require(isinstance(oracle, list) and oracle and all(isinstance(value, str) and value for value in oracle),
            'actual reviewed independent SQLite oracle command required')
    config['stages']['sqlite-catalogue'] = dict(adapter='sqlite-catalogue', oracle=oracle)


def assemble_static(inputs, base, plan_path, temporary):
    """Recompute a provisioned plan without overwriting its immutable GNU binding."""
    actual = plan_path if not plan_path.exists() else temporary / 'reviewed-static-plan.json'
    result = static_stages.assemble(inputs, base, actual, temporary / 'static-bindings.json')
    if actual != plan_path:
        require(read(plan_path) == read(actual), 'provisioned static plan differs from the current fixed recipes or actual tool inputs')
        for entry in result['stages'].values():
            for key in ('argv', 'checker'):
                if key in entry:
                    entry[key] = [str(plan_path) if token == str(actual) else token for token in entry[key]]
        result['inputs'] = sorted({str(plan_path) if token == str(actual) else token for token in result['inputs']})
    return result


def compose(inputs, base, plan_path, bindings_path, config_path):
    require(set(inputs) == {'schema', 'static', 'python'}, 'unknown/missing composition input fields')
    require(inputs['schema'] == 'gomad.driver-composition-inputs/v1', 'unknown composition schema')
    python = static_stages.regular(inputs['python'])
    deadline(base['deadline_s'])
    require(base['sent_epoch'] > 0, 'send epoch must precede composition')
    require(set(base['stages']) == set(mapping()), 'missing/foreign original stage labels')
    require(Path(inputs['static']['driver_config']).resolve() == config_path, 'static/final driver config differs')
    result = copy.deepcopy(base)
    original = copy.deepcopy(base['stages'])
    # Static assembly owns only its 24 labels; restore the independently typed runtime bindings.
    require(not bindings_path.exists(), 'composed bindings output must be fresh')
    with tempfile.TemporaryDirectory(prefix='gomad-composition-', dir=bindings_path.parent) as temporary:
        result = assemble_static(inputs['static'], result, plan_path, Path(temporary))
    result['stages'].update({label: entry for label, entry in original.items() if label not in static_stages.LABELS})
    for label in primary_stages.OWNED:
        result['stages'][label] = dict(adapter='primary')
    for label in CELLS:
        if result['stages'][label].get('adapter') == 'qualify':
            result['stages'][label]['private_observations'] = True
    result['python'] = str(python)
    result['_future_config'] = str(config_path)
    bind_primitives(result, config_path)
    result['final_reader'] = [str(python), str(HERE / 'final.py'), '--config', str(config_path)]
    result['final_sections'] = final_sections(str(python), config_path)
    validate_stages(result)
    require(set(result.get('live_corpus', {}).get('families', {})) == set(live.FAMILIES), 'missing actual live receiver families')
    for name, entry in result['live_corpus']['families'].items():
        label = entry['stage']
        require(label in result['stages'], 'foreign live receiver stage')
        families = result['stages'][label].setdefault('live_families', [])
        if name not in families:
            families.append(name)
    for entry in result['final_sections'].values():
        require(Path(entry['argv'][1]).is_file(), 'missing actual final reader: ' + entry['argv'][1])
    for key in ('final_profiles', 'final_retained', 'stress', 'public_verbs', 'private_observations'):
        require(result.get(key), 'missing actual final input binding: ' + key)
    result['inputs'] = sorted(set(result['inputs']) | {str(python), str(HERE / 'compose.py'),
        str(HERE / 'final_public_verbs.py'), str(HERE / 'primary_stages.py'), str(HERE / 'cell_reader.py'), str(HERE / 'native_roles.py')} | {entry['argv'][1] for entry in result['final_sections'].values()})
    result.pop('_future_config')
    write(bindings_path, result)
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for key in ('inputs', 'base', 'plan', 'output', 'config'):
        parser.add_argument('--' + key, required=True, type=Path)
    args = parser.parse_args()
    try:
        compose(read(args.inputs), read(args.base), args.plan.resolve(), args.output.resolve(), args.config.resolve())
        return 0
    except (Refusal, OSError, ValueError, KeyError) as error:
        print('REFUSED: ' + str(error), file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
