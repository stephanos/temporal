#!/usr/bin/env python3
"""Replay fresh retained cluster executions through the public CLI."""
import argparse
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
from pathlib import Path
import subprocess


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--work', type=Path, required=True)
    args = parser.parse_args()
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    corpus = Path(__file__).resolve().parent
    receipts = []

    def run(name, arguments, status=0, match=None):
        command = [str(args.binary), *map(str, arguments)]
        result = subprocess.run(command, capture_output=True, text=True, timeout=60)
        receipts.append(dict(name=name, argv=command, status=result.returncode,
                             stdout=result.stdout, stderr=result.stderr))
        (args.work / 'commands.json').write_text(json.dumps(receipts, indent=2) + '\n')
        assert result.returncode == status, (name, result.returncode, result.stdout, result.stderr)
        assert match is None or match in result.stdout, (name, result.stdout, result.stderr)
        return result

    run('publish-cluster', ['fixture-replay-cluster-execution', '--root', args.work / 'inputs',
                            '--corpus', corpus])
    bundle = json.loads((args.work / 'inputs/foundation-result.json').read_bytes())
    loader = SourceFileLoader('behavioral_parity', str(corpus.parent.parent / 'scripts/behavioral-parity'))
    parity = module_from_spec(spec_from_loader(loader.name, loader))
    loader.exec_module(parity)
    parity.authenticate(args.work / 'inputs', bundle)
    assert bundle['records'], 'missing cluster replay corpus'
    for index, row in enumerate(bundle['records']):
        artifact = args.work / 'inputs' / row['artifact']
        verified = run(f'verify-{index}', ['replay', '--verify-only', artifact], status=2)
        assert 'retained WASM cluster replay does not support --verify-only, --observed, or --toolchain-root' in verified.stderr
        run(f'observed-{index}', ['replay', '--observed', args.work / f'observed-{index}', artifact], status=2)
        run(f'replay-{index}', ['replay', artifact],
            status=0 if row['outcome'] == 'success' else 1, match='reproduced=true')
    print('public cluster replay: fresh retained executions match')


if __name__ == '__main__':
    main()
