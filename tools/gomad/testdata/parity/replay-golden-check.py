#!/usr/bin/env python3
"""Authenticate repeated executions before comparing the declared replay view."""
import argparse
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import os
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True
corpus = Path(__file__).resolve().parent
loader = SourceFileLoader('parity', str(corpus.parents[1] / 'scripts/behavioral-parity'))
parity = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(parity)
p = argparse.ArgumentParser()
p.add_argument('--binary', type=Path, required=True)
p.add_argument('--work', type=Path, required=True)
p.add_argument('--input', type=Path)
args = p.parse_args()
args.work = args.work.resolve()
args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
receipts = []


def run(name, command, status=0):
    result = subprocess.run(list(map(str, command)), capture_output=True, timeout=120)
    receipts.append(dict(name=name, argv=list(map(str, command)), status=result.returncode,
                         expected=status, stdout=result.stdout.decode(), stderr=result.stderr.decode()))
    (args.work / 'commands.json').write_bytes(parity.canonical(receipts))
    assert result.returncode == status, (name, result.returncode, result.stderr)
    return result.stdout


views = []
for repetition in range(2):
    root = args.work / str(repetition)
    command = ([args.binary, 'fixture-replay-input', '--root', root, '--input', args.input]
               if args.input else [args.binary, 'fixture-replay-execution', '--root', root, '--corpus', corpus])
    bundle = parity.decode(run(f'produce-{repetition}', command))
    parity.authenticate(root, bundle)
    assert len(bundle['records']) == (4 if args.input else 6), 'missing declared replay corpus'
    rows = []
    for index, item in enumerate(bundle['records']):
        artifact = root / item['artifact']
        raw = (root / item['record_file']).read_bytes()
        manifest = (artifact / 'manifest.json').read_bytes()
        verify = run(f'verify-{repetition}-{index}', [args.binary, 'replay', '--verify-only', artifact])
        assert verify == f'gomad: verified {artifact}\n'.encode()
        status = int(item['outcome'] != 'success')
        replay = run(f'replay-{repetition}-{index}', [args.binary, 'replay', artifact], status)
        assert b'reproduced=true' in replay
        observed = root / f'observed-{index}'
        repeated = run(f'observed-{repetition}-{index}',
            [args.binary, 'replay', '--observed', observed, artifact], status)
        assert replay == repeated
        streams = {}
        for name in ('stdout', 'stderr', 'backend-evidence.bin'):
            payload = (observed / name).read_bytes()
            streams[name] = dict(bytes=len(payload), sha256=parity.digest(payload))
        rows.append(dict(record=raw.decode(), manifest=manifest.decode(), status=status,
            verify=f'gomad: verified <output>/{item["artifact"]}\n',
            replay=replay.decode(), observed=streams))
    views.append(dict(schema='gomad.replay-golden/v1', records=rows,
                      ledger=(root / bundle['ledger_file']).read_text()))
parity.check_equal(views[0], views[1])
parity.golden(corpus / 'goldens-replay' / ('runtime.json' if args.input else 'replay.json'), views[0],
              os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1')
print('replay golden: independent raw integrity and repeated exact records, manifests, ledgers and reports agree')
