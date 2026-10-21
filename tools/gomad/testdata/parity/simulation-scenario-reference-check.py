#!/usr/bin/env python3
"""Compare real forced ready-start executions with the frozen Go WASM scenario producer."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


def encoded(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('source', 'compiler', 'baseline', 'base', 'campaign', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    args.work.mkdir(parents=True, exist_ok=False)
    source = args.source.resolve() / 'tools/gomad'
    receipts = []
    environment = {**os.environ, 'GOTOOLCHAIN': 'local', 'GOWORK': 'off', 'GOENV': 'off',
                   'GOMAXPROCS': '2', 'CGO_ENABLED': '0', 'GOCACHE': str(args.work.resolve() / 'go-cache')}
    environment.pop('GOROOT', None)

    def run(name, command, cwd=None):
        result = subprocess.run(list(map(str, command)), cwd=cwd, env=environment,
                                capture_output=True, timeout=600)
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        receipts.append(dict(name=name, argv=list(map(str, command)), status=result.returncode))
        (args.work / 'commands.json').write_bytes(encoded(receipts))
        assert result.returncode == 0, (name, result.returncode, result.stderr.decode(errors='replace'))
        return result.stdout

    paths = sorted(args.campaign.rglob('simulation/record.json'))
    assert paths
    exemplar = paths[0].parent.parent
    retained = json.loads((exemplar / 'evidence/cluster-replay.json').read_bytes())
    base = json.loads(args.base.read_bytes())
    base['HelperPath'] = str(args.baseline.resolve() / 'bin/gomad-wasmhost')
    base['HelperSHA256'] = hashlib.sha256(Path(base['HelperPath']).read_bytes()).hexdigest()
    base['ModulePath'] = str((exemplar / 'target/module.wasm').resolve())
    base['Runtime']['Choice']['ExecutionIdentity'] = retained['configuration']['incarnations'][0]['choice_identity']
    base['Runtime']['Diagnostics'] = False
    base['Timeout'] = 30_000_000_000
    input_path = args.work.resolve() / 'input.json'
    input_path.write_bytes(encoded(base))
    reference = args.work.resolve() / 'go-scenario-reference'
    template = Path(__file__).with_name('reference') / 'simulation-scenario-live.go.txt'
    with tempfile.TemporaryDirectory(prefix='simulation-reference-', dir=source / 'internal/runner') as temporary:
        shutil.copyfile(template, Path(temporary) / 'main.go')
        run('build-frozen-go', [args.compiler.resolve(), 'build', '-mod=readonly', '-trimpath', '-o', reference, '.'], temporary)
    go = json.loads(run('actual-go-executions', [reference, input_path]))
    rust = {}
    for path in paths:
        record = json.loads(path.read_bytes())
        forced = record['exploration_plan']['overrides'] or []
        if forced and not any(value['dimension'] == 'scenario' for value in forced):
            continue
        rank = next((value['selected'] for value in forced if value['dimension'] == 'scenario'), 0)
        replay = json.loads((path.parent.parent / 'evidence/cluster-replay.json').read_bytes())
        order = [step['handle']['node'] for step in replay['configuration']['steps'] if step['action'] == 'start']
        assert record['outcome'] == 'completed'
        rust[rank] = order
    assert set(rust) == {0, 1}, rust
    for observed in go:
        assert observed['record']['outcome'] == 'completed', observed
        assert observed['start_order'] == rust[observed['rank']], (observed, rust)
    (args.work / 'observations.json').write_bytes(encoded(dict(
        schema='gomad.actual-scenario-producer-parity/v1',
        go_producer='frozen public model.RunWASM with actual model.Choose driving real starts',
        go=go, rust_start_orders=rust)))


if __name__ == '__main__':
    main()
