#!/usr/bin/env python3
"""Reproduce the released Rust basedirs limitation using an isolated local daemon."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('output', type=Path, help='New directory for sources, cache and receipt')
args = parser.parse_args()
root = args.output.resolve()
root.mkdir(parents=True, exist_ok=False)

def probe(command):
    return subprocess.check_output(command, text=True).strip()

def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

tool = probe(['mise', 'which', 'sccache', '--tool', 'sccache@0.19.0'])
tools_root = Path(__file__).resolve().parents[4] / 'model' / 'umpire'
rustc_command = ['mise', '-C', str(tools_root), 'exec', '--', 'rustup', 'which', 'rustc']
rustc = probe(rustc_command)
environment = {
    key: value for key, value in os.environ.items()
    if not key.startswith('SCCACHE_') and key not in {
        'RUSTC_WRAPPER', 'RUSTC_WORKSPACE_WRAPPER', 'RUSTFLAGS',
        'CARGO_ENCODED_RUSTFLAGS', 'RUSTC_BOOTSTRAP', 'CARGO_TARGET_DIR',
    }
}
source = b'pub fn value() -> u32 { 42 }\n'
for name in ['checkout-a', 'checkout-b']:
    checkout = root / name
    checkout.mkdir()
    (checkout / 'lib.rs').write_bytes(source)

receipt = {
    'purpose': 'Released Rust basedirs limitation, not a completed Cargo integration proof',
    'sccache': {'path': tool, 'version': probe([tool, '--version']), 'sha256': digest(tool)},
    'rustc': {
        'path': rustc, 'resolution_command': rustc_command,
        'version': probe([rustc, '-vV']), 'sha256': digest(rustc),
    },
    'source_sha256': hashlib.sha256(source).hexdigest(),
    'source': source.decode(),
    'results': [],
}
with tempfile.TemporaryDirectory(prefix='fn25419-', dir='/tmp') as sockets:
    environment.update(
        SCCACHE_DIR=str(root / 'cache'), SCCACHE_CONF='/dev/null',
        SCCACHE_MULTILEVEL_CHAIN='disk', SCCACHE_SERVER_UDS=str(Path(sockets) / 'server.sock'),
        SCCACHE_BASEDIRS=f'{root}/checkout-a:{root}/checkout-b', CARGO_INCREMENTAL='0',
    )
    receipt['environment'] = {
        key: value for key, value in environment.items()
        if key.startswith('SCCACHE_') or key in {
            'CARGO_HOME', 'CARGO_INCREMENTAL', 'RUSTUP_HOME', 'RUSTUP_TOOLCHAIN',
        }
    }
    try:
        for index, name in enumerate(['checkout-a', 'checkout-b', 'checkout-a']):
            checkout = root / name
            output = checkout / f'out{index}'
            output.mkdir()
            command = [
                tool, rustc, '--crate-name', 'cache_control', '--crate-type', 'lib',
                '--emit=link,dep-info', '--out-dir', str(output), 'lib.rs',
                '-C', 'debuginfo=1',
            ]
            started = time.monotonic()
            result = subprocess.run(command, cwd=checkout, env=environment, capture_output=True, text=True)
            elapsed = time.monotonic() - started
            statistics_command = [tool, '--show-stats', '--stats-format=json']
            statistics = subprocess.run(statistics_command, env=environment, capture_output=True, text=True, check=True)
            stats = json.loads(statistics.stdout)
            receipt['results'].append({
                'checkout': name, 'cwd': str(checkout), 'command': command,
                'wall_seconds': elapsed, 'status': result.returncode,
                'stdout': result.stdout, 'stderr': result.stderr,
                'statistics_command': statistics_command, 'stats': stats,
            })
            result.check_returncode()
    finally:
        stopped = subprocess.run([tool, '--stop-server'], env=environment, capture_output=True, text=True)
        receipt['owned_daemon_stop'] = {
            'status': stopped.returncode, 'stdout': stopped.stdout, 'stderr': stopped.stderr,
        }
        (root / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n')

counts = [(r['stats']['stats']['cache_hits']['counts'].get('Rust', 0),
           r['stats']['stats']['cache_misses']['counts'].get('Rust', 0)) for r in receipt['results']]
assert counts == [(0, 1), (0, 2), (1, 2)], counts
assert receipt['owned_daemon_stop']['status'] == 0
print(json.dumps({'receipt': str(root / 'receipt.json'), 'cumulative_hits_misses': counts}))
