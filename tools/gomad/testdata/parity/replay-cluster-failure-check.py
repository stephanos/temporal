#!/usr/bin/env python3
"""Require first guest failure to stop the linked lifecycle and retain its outcome."""
import argparse
import hashlib
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True

p = argparse.ArgumentParser()
p.add_argument('--binary', type=Path, required=True)
p.add_argument('--work', type=Path, required=True)
p.add_argument('--compiler', type=Path, required=True)
p.add_argument('--source', type=Path, required=True)
args = p.parse_args()
args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
corpus = Path(__file__).resolve().parent
commands = []
loader = SourceFileLoader('cluster_failure_parity', str(corpus.parents[1] / 'scripts/behavioral-parity'))
parity = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(parity)


def run(name, arguments, status=0, executable=None, cwd=None):
    command = [str(executable or args.binary), *map(str, arguments)]
    result = subprocess.run(command, capture_output=True, timeout=60, cwd=cwd)
    commands.append(dict(name=name, argv=command,
                         status=result.returncode, stdout=result.stdout.decode(), stderr=result.stderr.decode()))
    (args.work / 'commands.json').write_text(json.dumps(commands, indent=2) + '\n')
    assert result.returncode == status, (name, result.returncode, result.stdout, result.stderr)
    return result.stdout


module_root = args.source / 'tools/gomad'
wrapper = module_root / 'cmd' / ('fn232-world-reference-' + hashlib.sha256(str(args.work).encode()).hexdigest()[:12])
wrapper.mkdir(mode=0o700)
world_decoder = args.work / 'world-reference'
try:
    (wrapper / 'main.go').write_bytes((corpus / 'reference/replay-world.go.txt').read_bytes())
    run('world-decoder-build', ['build', '-trimpath', '-o', world_decoder, './' + str(wrapper.relative_to(module_root))],
        executable=args.compiler, cwd=module_root)
finally:
    (wrapper / 'main.go').unlink()
    wrapper.rmdir()

run('base', ['fixture-replay-cluster-execution', '--root', args.work / 'base', '--corpus', corpus])
base = json.loads((args.work / 'base/foundation-result.json').read_bytes())
parity.authenticate(args.work / 'base', base)
artifact = args.work / 'base' / base['records'][0]['artifact']
manifest = json.loads((artifact / 'manifest.json').read_bytes())
cluster_file = next(name for name in manifest['record']['streams'] if name.endswith('/cluster-replay.json'))
retained = json.loads((artifact / cluster_file).read_bytes())
for name, module, termination, exit_code in [('trap', 'execution-trap.wat', 'trap', None),
                                           ('exit', 'replay-exit-failure.wat', 'exit', 2),
                                           ('capacity', 'execution-modeled.wat', 'capacity', None),
                                           ('trap-prefix', 'execution-trap.wat', 'trap', None)]:
    configuration = json.loads(json.dumps(retained['configuration']))
    configuration['incarnations'] = configuration['incarnations'][:2]
    configuration['steps'] = [dict(action=action, handle=dict(node=node, incarnation=1))
                              for action, node in [('start', 'a'), ('start', 'b'), ('wait', 'a')]]
    if name != 'trap-prefix':
        configuration['steps'].append(dict(action='wait', handle=dict(node='b', incarnation=1)))
    module_path = corpus / 'targets' / module
    raw = module_path.read_bytes()
    module_sha = hashlib.sha256(raw).hexdigest()
    first = configuration['incarnations'][0]
    if name == 'capacity':
        first['limits']['output_bytes'] = 1
        first['namespace']['limits']['output_bytes'] = 1
    first['module'].update(bytes=len(raw), sha256=module_sha)
    first['choice_identity']['TargetSHA256'] = list(bytes.fromhex(module_sha))
    second = configuration['incarnations'][1]
    second['module']['file'] = 'target/peer.wat'
    second_module = corpus / 'targets/inspect-choice.wat'
    modules = {'target/module.wasm': str(module_path), 'target/peer.wat': str(second_module)}
    input_path = args.work / (name + '-input.json')
    input_path.write_text(json.dumps(dict(schema='gomad.cluster-fixture-input/v1',
        toolchain_sha256=retained['toolchain_sha256'], configuration=configuration, modules=modules)))
    output = args.work / name
    report = json.loads(run(name + '-produce', ['fixture-replay-cluster-input', '--root', output, '--input', input_path]))
    parity.authenticate(output, report)
    path = output / report['records'][0]['artifact']
    record = json.loads((path / 'record.json').read_bytes())
    assert record['outcome']['termination'] == termination, record['outcome']
    assert record['outcome']['exit_code'] == exit_code, record['outcome']
    summary = json.loads((path / 'evidence/cluster_summary').read_bytes())
    assert summary['incarnations'][0]['termination'] == termination, summary
    assert summary['incarnations'][1]['termination'] == 'stop', summary
    actual_input = json.loads((path / cluster_file).read_bytes())
    assert actual_input['configuration']['steps'] == configuration['steps'], 'retained authored lifecycle changed'
    if name == 'capacity':
        world = (path / 'evidence/world').read_bytes()
        assert world, 'capacity must retain terminal World evidence'
        decoded = json.loads(run('capacity-world-decode', [path / 'evidence/world'], executable=world_decoder))
        assert decoded['schemaVersion'] == 1, decoded
    assert b'reproduced=true' in run(name + '-replay', ['replay', path], 1)
    for repetition in range(2):
        repeated_root = args.work / (name + '-repeat-' + str(repetition))
        repeated = json.loads(run(name + '-repeat-' + str(repetition),
            ['fixture-replay-cluster-input', '--root', repeated_root, '--input', input_path]))
        parity.authenticate(repeated_root, repeated)
        repeated_path = repeated_root / repeated['records'][0]['artifact']
        for retained_file in ['record.json', 'manifest.json', *json.loads((path / 'manifest.json').read_bytes())['record']['streams']]:
            assert (path / retained_file).read_bytes() == (repeated_path / retained_file).read_bytes(), retained_file
    if name == 'capacity':
        successful_prefix = json.loads(input_path.read_bytes())
        first = successful_prefix['configuration']['incarnations'][0]
        first['limits']['output_bytes'] = 1024
        first['namespace']['limits']['output_bytes'] = 1024
        complete_path = args.work / 'successful-complete-input.json'
        complete_path.write_text(json.dumps(successful_prefix))
        complete_root = args.work / 'successful-complete'
        complete = json.loads(run('successful-complete-control', ['fixture-replay-cluster-input',
            '--root', complete_root, '--input', complete_path]))
        parity.authenticate(complete_root, complete)
        complete_artifact = complete_root / complete['records'][0]['artifact']
        complete_record = json.loads((complete_artifact / 'record.json').read_bytes())
        assert complete_record['outcome'] == {'class': 'success', 'termination': 'exit', 'exit_code': 0, 'reason': ''}, complete_record['outcome']
        complete_summary = json.loads((complete_artifact / 'evidence/cluster_summary').read_bytes())
        assert all(item['termination'] == 'exit' and item['exit_code'] == 0 for item in complete_summary['incarnations']), complete_summary
        successful_prefix['configuration']['steps'] = configuration['steps'][:3]
        success_path = args.work / 'successful-open-prefix-input.json'
        success_path.write_text(json.dumps(successful_prefix))
        run('successful-open-prefix-refused', ['fixture-replay-cluster-input', '--root',
            args.work / 'successful-open-prefix', '--input', success_path], 2)
        assert 'incomplete cluster incarnation lifecycle' in commands[-1]['stderr']
print('linked cluster guest failures: first outcome retained, peer stopped and replay reproduced')
