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
args = p.parse_args()
args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
corpus = Path(__file__).resolve().parent
commands = []
loader = SourceFileLoader('cluster_failure_parity', str(corpus.parents[1] / 'scripts/behavioral-parity'))
parity = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(parity)


def run(name, arguments, status=0):
    result = subprocess.run([str(args.binary), *map(str, arguments)], capture_output=True, timeout=15)
    commands.append(dict(name=name, argv=[str(args.binary), *map(str, arguments)],
                         status=result.returncode, stdout=result.stdout.decode(), stderr=result.stderr.decode()))
    (args.work / 'commands.json').write_text(json.dumps(commands, indent=2) + '\n')
    assert result.returncode == status, (name, result.returncode, result.stdout, result.stderr)
    return result.stdout


run('base', ['fixture-replay-cluster-execution', '--root', args.work / 'base', '--corpus', corpus])
base = json.loads((args.work / 'base/foundation-result.json').read_bytes())
parity.authenticate(args.work / 'base', base)
artifact = args.work / 'base' / base['records'][0]['artifact']
manifest = json.loads((artifact / 'manifest.json').read_bytes())
cluster_file = next(name for name in manifest['record']['streams'] if name.endswith('/cluster-replay.json'))
retained = json.loads((artifact / cluster_file).read_bytes())
for name, module, termination, exit_code in [('trap', 'execution-trap.wat', 'trap', None),
                                           ('exit', 'replay-exit-failure.wat', 'exit', 2),
                                           ('capacity', 'execution-modeled.wat', 'capacity', None)]:
    configuration = json.loads(json.dumps(retained['configuration']))
    configuration['incarnations'] = configuration['incarnations'][:2]
    configuration['steps'] = [dict(action=action, handle=dict(node=node, incarnation=1))
                              for action, node in [('start', 'a'), ('start', 'b'), ('wait', 'a'), ('wait', 'b')]]
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
    if name == 'capacity':
        world = (path / 'evidence/world').read_bytes()
        assert world, 'capacity must retain terminal World evidence'
        assert json.loads(world)['schemaVersion'] == 1
    assert b'reproduced=true' in run(name + '-replay', ['replay', path], 1)
    for repetition in range(2):
        repeated_root = args.work / (name + '-repeat-' + str(repetition))
        repeated = json.loads(run(name + '-repeat-' + str(repetition),
            ['fixture-replay-cluster-input', '--root', repeated_root, '--input', input_path]))
        parity.authenticate(repeated_root, repeated)
        repeated_path = repeated_root / repeated['records'][0]['artifact']
        for retained_file in ['record.json', 'manifest.json', *json.loads((path / 'manifest.json').read_bytes())['record']['streams']]:
            assert (path / retained_file).read_bytes() == (repeated_path / retained_file).read_bytes(), retained_file
print('linked cluster guest failures: first outcome retained, peer stopped and replay reproduced')
