#!/usr/bin/env python3
"""Exercise the inspection driver's generated argv against the actual public CLI parser."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location('onebox_inspection', Path(__file__).with_name('onebox-inspection-check.py'))
gate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(gate)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--binary-sha256', required=True)
    args = parser.parse_args()
    binary = args.binary.resolve(strict=True)
    gate.require(gate.digest(binary) == args.binary_sha256, 'CLI binary digest mismatch')
    repo = Path(__file__).resolve().parents[4]
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_')}
    with tempfile.TemporaryDirectory(prefix='onebox-parser-') as temporary:
        root = Path(temporary)
        scenario = root / 'scenario.json'
        scenario.write_bytes(b'not JSON')
        command = list(map(str, gate.explore_command(binary, root / 'installation-absent', root, repo,
                                                    repo / 'service/history/parser-test', scenario)))
        result = subprocess.run(command, cwd=repo, env=environment, capture_output=True, timeout=30)
        combined = result.stdout + result.stderr
        gate.require(result.returncode == 2 and b'scenario must contain bounded valid JSON' in combined,
                     f'generated inspection command failed before scenario admission: exit={result.returncode} output={combined!r}')
        old_command = command.copy()
        old_command[old_command.index('--scenario'):old_command.index('--scenario')] = ['--choice-bytes', '0']
        refused = subprocess.run(old_command, cwd=repo, env=environment, capture_output=True, timeout=30)
        gate.require(refused.returncode == 2 and any(marker in refused.stdout + refused.stderr for marker in
                     (b'--choice-bytes requires --choices', b'invalid byte size')),
                     'actual CLI did not distinguish the rejected explicit choice override')
        print(json.dumps(dict(binary=str(binary), binary_sha256=args.binary_sha256,
                              admitted_scenario_sentinel_exit=result.returncode, rejected_explicit_choice_exit=refused.returncode)))


if __name__ == '__main__':
    main()
