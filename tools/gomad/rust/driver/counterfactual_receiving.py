"""Same-stage four-pair receiving with one explicitly superseded assertion."""
from pathlib import Path

from driver import HERE, command, guard, read, require, sha, write
import counterfactual_lineage as lineage
import executed_divergence as original
import native_roles

STAGE = 'cluster-functional'
IDS = {'original-divergence/' + name for name in original.CASES}
KEYS = {'cli_build', 'harness_build', 'compiler_root', 'module_cache'}


def selection(config):
    selected = config['live_corpus']['test_wasm_reach'].get('executed_divergence')
    require(isinstance(selected, dict) and set(selected) == KEYS,
            'same-stage original four-pair endpoint is not configured')
    require(all(isinstance(value, str) and Path(value).is_absolute() for value in selected.values()),
            'original four-pair inputs must be explicit absolute paths')
    root = Path(config['root']) / 'stages' / STAGE
    return selected, root, root / 'executed-divergence'


def absent(config):
    reach = config['live_corpus']['test_wasm_reach']
    require(isinstance(reach, dict), 'invalid original test-wasm reach declaration')
    if 'executed_divergence' in reach:
        selection(config)
        return None
    import live
    gap = 'fn-232.14: original executed-divergence endpoint is absent; four pairs are unexecuted and uncredited'
    require(not live.REACH_GAPS_BLOCK, gap)
    return dict(status='UNEXECUTED', executed=False, credited=False, actual_cases=[], gap=gap)


def input_paths(selected, repo):
    """Closed explicit input roots for the final freezer; no ambient cache search."""
    require(isinstance(selected, dict) and set(selected) == KEYS, 'incomplete four-pair input declaration')
    paths = external_input_paths(selected, repo)
    for key in ('cli_build', 'harness_build'):
        paths.update(original.main_cli_build.input_paths(Path(selected[key])))
    return paths


def external_input_paths(selected, repo):
    paths = set()
    compiler = Path(selected['compiler_root'])
    require(compiler.is_absolute() and compiler.resolve() == compiler and (compiler / 'bin/go').is_file(),
            'four-pair original stock compiler root is missing or aliased')
    inventory = original.live.inventory(compiler)
    require(len(inventory) <= 100_000 and sum(row['bytes'] for row in inventory.values()) <= 2 << 30,
            'original stock compiler closure exceeds its bounded input inventory')
    paths.update(str(compiler / name) for name in inventory)
    cache = Path(selected['module_cache'])
    inventory = original.installed_cluster_cache.verify(cache, Path(repo) / 'tools/gomad/go.sum')
    paths.update(str(cache / name) for name in inventory)
    paths.update(str(HERE / name) for name in ('counterfactual_lineage.py', 'counterfactual_receiving.py', 'executed_divergence.py'))
    paths.add(str(Path(repo) / lineage.AUTHORITY['spec']))
    return paths


def inputs(config, selected):
    cli, harness = original.roles(Path(selected['cli_build']), Path(selected['harness_build']), frozen=config)
    repo = Path(config['repo'])
    for name in external_input_paths(selected, repo):
        native_roles.pinned(config, Path(name))
    authority = repo / lineage.AUTHORITY['spec']
    native_roles.pinned(config, authority)
    require(sha(authority) == lineage.AUTHORITY['sha256'], 'accepted fn235 R2 authority changed')
    native_roles.pinned(config, HERE / 'counterfactual_lineage.py')
    native_roles.pinned(config, HERE / 'executed_divergence.py')
    return cli


def argv(config, selected, root):
    args = [config['python'], str(HERE / 'executed_divergence.py'), 'produce', '--config', config['_config_path']]
    for key in ('cli_build', 'harness_build', 'compiler_root', 'module_cache'):
        args.extend(['--' + key.replace('_', '-'), selected[key]])
    return args + ['--root', str(root)]


def binding(config):
    return dict(schema='gomad.same-stage-counterfactual/v1', stage=STAGE,
                source_sha256=config['source_sha256'], identity=config['identity'],
                config_sha256=config['_config_sha256'])


def observations(config):
    gap = absent(config)
    if gap is not None: return gap
    selected, stage, root = selection(config)
    cli = inputs(config, selected)
    require(read(stage / 'counterfactual-binding.json') == binding(config),
            'four-pair producer belongs to another qualification cycle')
    original.verify(root / 'result.json', frozen=config)
    report = original.pairs(root, cli)
    applicability = lineage.reconcile(report)
    for row in report['pairs']:
        if row['name'] != 'handoff':continue
        for role in ('baseline', 'fault'):
            summary = read(Path(row[role]['artifact']) / 'evidence/cluster_summary')
            owners = {item['handle']['node']: item for item in summary['incarnations']}
            require(owners['a']['termination'] == 'exit' and owners['a']['exit_code'] == 1
                    and owners['a']['interrupted'] is False and owners['b']['termination'] == 'stop'
                    and owners['b']['reason'] == 'WASI stop: cluster failed',
                    'superseded peer assertion has no originating ordinary guest failure')
    row = read(stage / 'counterfactual.command.json')
    process = read(stage / 'counterfactual.process.json')
    import live
    live.completed_command(row, process, argv(config, selected, root), config['repo'])
    owner = read(root / 'producer.json')
    require(owner['pid'] == row['pid'] and owner['repo'] == config['repo']
            and row['start_ns'] <= owner['start_ns'] <= row['end_ns']
            and row['started_monotonic_nanos'] <= owner['started_monotonic_nanos'] < row['reaped_monotonic_nanos'],
            'four-pair receipt was borrowed from a previous producer')
    return dict(original_status=report['status'], assertion_applicability=applicability,
                actual_cases=sorted(IDS), receipt=str(root / 'result.json'), receipt_sha256=sha(root / 'result.json'))


def run(config):
    guard(config)
    gap = absent(config)
    if gap is not None: return gap
    selected, stage, root = selection(config)
    inputs(config, selected)
    require(not root.exists(), 'four-pair receiving must start with fresh same-stage evidence')
    write(stage / 'counterfactual-binding.json', binding(config))
    command(config, stage, 'counterfactual', argv(config, selected, root), timeout=4800)
    guard(config)
    return observations(config)


def final_read(config):
    import broad_stages
    import core_lifecycle
    from primary_stages import enclose_child
    stage = Path(config['root']) / 'stages' / STAGE
    facts = broad_stages.read(config, STAGE, stage)
    gap = absent(config)
    if gap is not None: return gap
    selected, stage, root = selection(config)
    drain = facts['readers']['lifecycle']['roots'][core_lifecycle.DRAIN]
    require({row['variant'] for row in drain['cases']} == set(core_lifecycle.VARIANTS[core_lifecycle.DRAIN])
            and drain['native_receive'], 'same-stage stronger failure/drain behavior is missing')
    enclose_child(read(stage / 'producer.command.json'), read(stage / 'producer.process.json'),
                  read(stage / 'counterfactual.command.json'))
    return observations(config)
