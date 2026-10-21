#!/usr/bin/env python3
"""Exercise public typed seed scenario admission, lifecycle and retained replay."""
import argparse
import base64
import copy
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import os
from pathlib import Path
import shlex
import subprocess
import sys

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent
loader = SourceFileLoader('campaign_check', str(CORPUS / 'campaign-check.py'))
check = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(check)


def authenticate_scenario(root, declared):
    validated = check.authenticate_campaign(root)
    for execution in validated['executions']:
        artifact = root / execution['artifact']
        inputs = check.canonical_file(artifact / 'evidence/cluster-replay.json')
        check.equal(inputs['schema'], 'gomad.cluster-replay-inputs/v1', 'cluster.replay_schema')
        configuration = inputs['configuration']
        check.equal(configuration['nodes'], [node['id'] for node in declared['spec']['nodes']], 'cluster.nodes')
        check.equal(configuration['steps'], declared['actions'], 'cluster.complete_script')
        configured = {(item['handle']['node'], item['handle']['incarnation']): item
                      for item in configuration['incarnations']}
        retained = {(item['handle']['node'], item['handle']['incarnation']): item
                    for item in inputs['retained']}
        check.equal(len(configured), len(configuration['incarnations']), 'cluster.unique_configured_handles')
        check.equal(len(retained), len(inputs['retained']), 'cluster.unique_retained_handles')
        check.equal(set(configured), set(retained), 'cluster.retained_handles')
        expected_handles = {(step['handle']['node'], step['handle']['incarnation'])
                            for step in declared['actions'] if step['action'] in ('start', 'restart')}
        check.equal(set(configured), expected_handles, 'cluster.declared_incarnations')
        bindings = {(item['handle']['node'], item['handle']['incarnation']): item['boot']
                    for item in declared.get('incarnation_boots', [])}
        nodes = {node['id']: node for node in declared['spec']['nodes']}
        boots = {boot['id']: boot for boot in declared['boots']}
        for handle, item in configured.items():
            boot = boots[bindings.get(handle, nodes[handle[0]]['boot'])]
            if boot.get('args') is not None:
                check.equal(item['arguments'][1:], boot['args'], 'cluster.boot_arguments')
            module = item['module']
            raw = check.parity.contained(artifact, module['file']).read_bytes()
            check.equal((len(raw), check.parity.digest(raw)), (module['bytes'], module['sha256']), 'cluster.module')
            check.equal(item['choice_identity']['TargetSHA256'], list(bytes.fromhex(module['sha256'])), 'cluster.choice_target')
            for descriptor in retained[handle]['evidence'].values():
                if not 0 <= descriptor['bytes'] <= descriptor['limit'] <= item['limits']['evidence_bytes']:
                    raise ValueError('cluster evidence descriptor exceeds admitted bounds')
                raw = check.parity.contained(artifact, descriptor['file']).read_bytes()
                check.equal((len(raw), check.parity.digest(raw)),
                            (descriptor['bytes'], descriptor['sha256']), 'cluster.incarnation_payload')
        boundaries = []
        for field, action in (('output_crashes', 'crash'), ('output_stops', 'stop')):
            for item in declared.get(field, []):
                marker = item['marker']
                boundaries.append(dict(handle=item['handle'], action=action,
                    marker=list(base64.b64decode(marker)) if isinstance(marker, str) else marker))
        check.equal(configuration['boundaries'], boundaries, 'cluster.boundaries')
    return validated


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    for name in ('installation', 'input', 'go-compiler', 'preparation-root'):
        parser.add_argument('--' + name, type=Path)
    args = parser.parse_args()
    if args.installation is None and (args.input is None or args.go_compiler is None):
        parser.error('provide --installation or both --input and --go-compiler')
    for name in ('binary', 'installation', 'work', 'input', 'go_compiler', 'preparation_root'):
        value = getattr(args, name)
        if value is not None:
            setattr(args, name, value.resolve())
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    module = CORPUS.parents[1]
    fixtures = module / 'internal/backends/wasm/command/testdata/scenario'
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_')}
    commands = []

    def run(name, arguments, status=0, installation=None):
        selected = args.installation if installation is None else installation
        command = [args.binary, *(['--installation', selected] if selected else []), *arguments]
        result = subprocess.run(list(map(str, command)), env=environment, capture_output=True, timeout=600)
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        commands.append(dict(name=name, argv=list(map(str, command)), status=result.returncode, expected=status,
                             stdout_sha256=check.parity.digest(result.stdout), stderr_sha256=check.parity.digest(result.stderr)))
        (args.work / 'commands.json').write_bytes(check.parity.canonical(commands) + b'\n')
        check.equal(result.returncode, status, name + ': ' + result.stdout.decode(errors='replace') + result.stderr.decode(errors='replace'))
        return result.stdout

    if args.installation is None:
        identity_root = args.work / 'identity'
        bundle = check.parity.decode(run('identity', ['fixture-replay-input', '--root', identity_root,
                                                      '--input', args.input], installation=False))
        check.parity.authenticate(identity_root, bundle)
        identity_artifact = next(identity_root / row['artifact'] for row in bundle['records']
            if check.parity.read(identity_root / row['artifact'] / 'manifest.json')['record']['profile']
                == 'gomad.wasi-cooperative/v1')
        inputs = check.canonical_file(identity_artifact / 'evidence/replay.json')
        configuration = inputs['configuration']
        args.installation = args.work / 'installation.json'
        args.installation.write_bytes(check.parity.canonical(dict(
            schema='gomad.rust-wasm-installation/v1', compiler_path=str(args.go_compiler),
            compiler_sha256=check.parity.digest(args.go_compiler.read_bytes()), go_version='go1.27.1',
            runtime_root=str(module / 'internal/model/runtimesource'),
            cache_root=str(args.preparation_root or args.work.parent / 'scenario-preparation'),
            profile='gomad.wasi-cooperative/v1', fuel=configuration['limits']['fuel'],
            memory_bytes=configuration['limits']['memory_bytes'], helper_sha256=inputs['helper_sha256'],
            host_sha256=inputs['host_sha256'], runtime_implementation_sha256=inputs['runtime_implementation_sha256'],
            model=configuration['model'], namespace=configuration['namespace'],
            storage=configuration.get('storage'), network=None, build_tags=[])))

    def explore(scenario, destination):
        return ['explore', '--json', '--scenario', scenario, '--seeds=7', '--parallel=1', '--on-failure=all',
                '--execution-timeout=30s', '--overall-timeout=10m', '--keep-successes=all', '--success-limit=4',
                '--success-bytes=256MiB', '--artifacts', destination, '--working-dir', module,
                'go-run', './internal/backends/wasm/command/testdata/scenario']

    compiler_calls = args.work / 'refusal-compiler-calls'
    refusal_compiler = args.work / 'refusal-compiler'
    installation = copy.deepcopy(check.canonical_file(args.installation))
    refusal_compiler.write_text('#!/bin/sh\nprintf "compiler invoked\\n" >> '
        + shlex.quote(str(compiler_calls)) + '\nexec '
        + shlex.quote(installation['compiler_path']) + ' "$@"\n')
    refusal_compiler.chmod(0o700)
    installation['compiler_path'] = str(refusal_compiler)
    installation['compiler_sha256'] = check.parity.digest(refusal_compiler.read_bytes())
    refusal_manifest = args.work / 'refusal-installation.json'
    refusal_manifest.write_bytes(check.parity.canonical(installation))
    original = check.parity.read(fixtures / 'durable.json')
    mutations = [
        ('unknown', lambda value: value.__setitem__('extra', True)),
        ('nested-unknown', lambda value: value['spec']['nodes'][0].__setitem__('extra', True)),
        ('capacity', lambda value: value['spec']['limits'].__setitem__('nodes', 1)),
        ('unknown-limit', lambda value: value['spec']['limits'].__setitem__('extra', 1)),
        ('unsorted', lambda value: value['spec']['nodes'].reverse()),
        ('unknown-boot', lambda value: value['spec']['nodes'][0].__setitem__('boot', 'missing')),
        ('mount-mismatch', lambda value: value['boots'][0]['storage']['volumes'][0].__setitem__('CapacityBytes', 1)),
        ('restart-storage', lambda value: value['boots'][1].__setitem__('storage', None)),
        ('missing-boundary', lambda value: value.__setitem__('output_crashes', [])),
        ('marker', lambda value: value['output_crashes'][0].__setitem__('marker', '!!!!')),
        ('unused-binding', lambda value: value['incarnation_boots'][0]['handle'].__setitem__('incarnation', 3)),
        ('incomplete', lambda value: value['actions'].pop()),
        ('running-restart', lambda value: value['actions'][2].__setitem__('action', 'restart')),
        ('unknown-action', lambda value: value['actions'][0].__setitem__('action', 'run')),
    ]
    for name, mutate in mutations:
        value = copy.deepcopy(original)
        mutate(value)
        source = args.work / ('invalid-' + name + '.json')
        source.write_bytes(check.parity.canonical(value))
        run('refuse-' + name, explore(source, args.work / ('invalid-' + name)), 2, refusal_manifest)
        check.equal(compiler_calls.exists(), False, name + '.refused_before_compiler')
    for index, raw in enumerate((b'{} {}', b'{', b'', b'{"faults":null,"faults":null}')):
        source = args.work / f'malformed-{index}.json'
        source.write_bytes(raw)
        run(f'refuse-malformed-{index}', explore(source, args.work / f'malformed-{index}'), 2, refusal_manifest)
        check.equal(compiler_calls.exists(), False, f'malformed-{index}.refused_before_compiler')

    preparation = args.preparation_root or Path(check.canonical_file(args.installation)['cache_root'])
    source = preparation / 'scenario-soft-source'
    source.mkdir(mode=0o700, parents=True, exist_ok=True)
    source_files = {'go.mod': b'module gomad.scenario.softfixture\n\ngo 1.27.1\n',
        'soft_test.go': b'''package softfixture
import ("fmt"; "strings"; "testing"; "time")
func TestAssertion(t *testing.T) { fmt.Println("failed assertion: scenario invariant") }
func TestQuiet(t *testing.T) { fmt.Println("quiet incarnation") }
func TestCrash(t *testing.T) {
    fmt.Println("failed assertion: crash invariant")
    fmt.Println("soft crash boundary")
    time.Sleep(time.Hour)
    t.Fatal("crash boundary failed")
}
func TestTruncated(t *testing.T) {
    fmt.Println("failed assertion: truncated invariant")
    fmt.Print(strings.Repeat("x", 8192))
}
'''}
    if source.is_symlink():
        raise ValueError('generated scenario source directory is a symlink')
    for name, expected in source_files.items():
        path = source / name
        if path.is_symlink():
            raise ValueError('generated scenario source file is a symlink')
        if path.exists():
            check.equal(path.read_bytes(), expected, 'scenario.generated_source.' + name)
        else:
            with path.open('xb') as stream:
                stream.write(expected)
    check.equal(sorted(child.name for child in source.iterdir()), sorted(source_files), 'scenario.generated_source_files')

    def soft_scenario(test, restart=False):
        scenario = copy.deepcopy(original)
        scenario['spec']['nodes'] = [dict(id='a', boot='initial', address='10.0.0.1')]
        scenario['spec']['volumes'] = []
        scenario['boots'] = [dict(id='initial', args=['-test.run=^' + test + '$', '-test.v'])]
        scenario['actions'] = [dict(action='start', handle=dict(node='a', incarnation=1)),
                               dict(action='wait', handle=dict(node='a', incarnation=1))]
        scenario['incarnation_boots'] = []
        scenario['output_crashes'] = []
        if restart:
            scenario['boots'].append(dict(id='recovery', args=['-test.run=^TestQuiet$', '-test.v']))
            scenario['actions'][1]['action'] = 'crash'
            scenario['incarnation_boots'] = [dict(handle=dict(node='a', incarnation=2), boot='recovery')]
            scenario['output_crashes'] = [dict(handle=dict(node='a', incarnation=1),
                marker=base64.b64encode(b'soft crash boundary').decode())]
            scenario['actions'].extend([dict(action='restart', handle=dict(node='a', incarnation=2)),
                                        dict(action='wait', handle=dict(node='a', incarnation=2))])
        return scenario

    soft_views = []
    for repetition in range(2):
        view = {}
        for name, test, report_only, restart, output_limit, status in [
            ('assertion', 'TestAssertion', False, False, None, 1),
            ('report-only', 'TestAssertion', True, False, None, 0),
            ('crash-assertion', 'TestCrash', False, True, None, 1),
            ('crash-report-only', 'TestCrash', True, True, None, 0),
            ('truncated-report-only', 'TestTruncated', True, False, '128', 3),
        ]:
            scenario_path = args.work / (name + '.json')
            scenario_path.write_bytes(check.parity.canonical(soft_scenario(test, restart)))
            destination = args.work / f'{name}-{repetition}'
            arguments = explore(scenario_path, destination)
            arguments[arguments.index('--working-dir') + 1] = source
            target = arguments.index('go-run')
            arguments[target:] = ['go-test', '.']
            if report_only:
                arguments.insert(target, '--soft-assertion-report-only')
            if output_limit:
                arguments.insert(target, '--output-limit=' + output_limit)
            run(f'{name}-{repetition}', arguments, status)
            roots = list((destination / 'campaigns').glob('campaign-*'))
            check.equal(len(roots), 1, name + '.campaign_count')
            validated = authenticate_scenario(roots[0], check.parity.read(scenario_path))
            check.equal(len(validated['executions']), 1, name + '.executions')
            execution = validated['executions'][0]
            artifact = roots[0] / execution['artifact']
            report = check.canonical_file(artifact / 'evidence/soft-assertions.json')
            check.equal(report['schema'], 'gomad.cluster-soft-assertions/v1', name + '.report_schema')
            reports = {(item['handle']['node'], item['handle']['incarnation']): item['report']
                       for item in report['incarnations']}
            first = reports[('a', 1)]
            check.equal(first['report_only'], report_only, name + '.policy')
            check.equal(first['inconclusive'], bool(output_limit), name + '.completeness')
            check.equal(first['messages'], ['truncated invariant' if output_limit else
                'crash invariant' if restart else 'scenario invariant'], name + '.messages')
            check.equal(execution['record']['outcome']['class'],
                'runner_failure' if output_limit else 'success' if report_only else 'target_failure', name + '.class')
            if restart:
                check.equal(sorted(reports), [('a', 1), ('a', 2)] if report_only else [('a', 1)], name + '.handles')
            if not output_limit:
                replayed = run(f'{name}-replay-{repetition}', ['replay', artifact], 0 if report_only else 1)
                check.equal(b'reproduced=true ' in replayed, True, name + '.replay_reproduced')
            view[name] = dict(plan=validated['plan'], journal=validated['journal'],
                              records=check.execution_view(validated), reports=report)

        soft_views.append(view)
    check.equal(soft_views[0], soft_views[1], 'scenario.soft_fresh_repeated_projection')

    views = []
    network_installation = copy.deepcopy(check.parity.read(args.installation))
    limits = check.parity.read(fixtures / 'network.json')['spec']['limits']
    network_installation['network'] = dict(profile='gomad.simulation-network-stream/v3', faults=[], limits={
        name: limits['network_' + name] for name in ('listeners', 'connections', 'bytes', 'connection_bytes', 'stall_nanos', 'transitions')})
    network_manifest = args.work / 'network-installation.json'
    network_manifest.write_bytes(check.parity.canonical(network_installation))
    for repetition in range(2):
        view = {}
        for name in ('durable', 'network'):
            destination = args.work / f'{name}-{repetition}'
            run(f'{name}-{repetition}', explore(fixtures / (name + '.json'), destination),
                installation=network_manifest if name == 'network' else None)
            roots = list((destination / 'campaigns').glob('campaign-*'))
            check.equal(len(roots), 1, name + '.campaign_count')
            root = roots[0]
            validated = authenticate_scenario(root, check.parity.read(fixtures / (name + '.json')))
            check.equal(len(validated['executions']), 1, name + '.executions')
            execution = validated['executions'][0]
            artifact = root / execution['artifact']
            retained = check.canonical_file(artifact / 'evidence/cluster-replay.json')
            handles = [(row['handle']['node'], row['handle']['incarnation']) for row in retained['retained']]
            check.equal(len(handles), 3 if name == 'durable' else 4, name + '.full_lifecycle')
            inspected = check.parity.decode(run(f'{name}-inspect-{repetition}', ['inspect', '--json', root]))
            check.equal(inspected['campaign']['executions'][0]['record'], execution['record'], name + '.inspect.record')
            replayed = run(f'{name}-replay-{repetition}', ['replay', artifact])
            check.equal(b'reproduced=true ' in replayed, True, name + '.replay_reproduced')
            view[name] = dict(plan=validated['plan'], journal=validated['journal'], records=check.execution_view(validated))
        views.append(view)
    check.equal(views[0], views[1], 'scenario.fresh_repeated_projection')

    golden = CORPUS / 'goldens-campaign/scenarios.json'
    regenerate = os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1'
    first = None
    for view in views:
        check.parity.golden(golden, dict(schema='gomad.seed-scenario-golden/v1', cases=view,
                                        soft_assertions=soft_views[0]), regenerate)
        if first is None:
            first = golden.read_bytes()
        else:
            check.equal(golden.read_bytes(), first, 'scenario.second_regeneration_bytes')
    print('public typed seed scenarios, authenticated replay and strict refusals passed')


if __name__ == '__main__':
    main()
