#!/usr/bin/env python3
"""Produce an actual Go default-success/forced-failure scheduling witness."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess


def prepare(binary, installation, output, run):
    output.mkdir(mode=0o700, parents=True, exist_ok=False)
    guest = output / 'guest'
    guest.mkdir(mode=0o700)
    (guest / 'go.mod').write_text('module gomad.minimize-counterfactual\n\ngo 1.27.1\n')
    (guest / 'failure_test.go').write_text('''package minimizecounterfactual
import ("runtime"; "testing")
func TestWinner(t *testing.T) {
    winners := make(chan int, 2)
    go func() { winners <- 0 }()
    go func() { winners <- 1 }()
    runtime.Gosched()
    winner := <-winners
    <-winners
    noise := make(chan struct{}, 2)
    for range 2 { go func() { runtime.Gosched(); noise <- struct{}{} }() }
    <-noise
    <-noise
    if winner == 0 { t.Fatal("forced scheduling winner") }
}
''')
    for seed in (7, 8, 89):
        artifacts = output / f'seed{seed}'
        run(f'counterfactual-seed{seed}-explore', [binary, '--installation', installation, 'explore', '--json',
            '--strategy=choice-exploration', '--choices', '--seeds', seed, '--parallel=1', '--max-executions=16',
            '--max-choice-depth=4', '--max-exploration-bytes=128MiB', '--on-failure=all', '--keep-successes=all', '--success-limit=32',
            '--success-bytes=512MiB', '--working-dir', guest, '--build-tag=test_dep', '--execution-timeout=30s',
            '--overall-timeout=10m', '--artifacts', artifacts, 'go-test', '.', '--', '-test.run=^TestWinner$', '-test.v'], 1)
        records = [(path.parent, json.loads(path.read_bytes())) for path in artifacts.rglob('manifest.json')]
        defaults = [(path, row) for path, row in records if row.get('artifact_kind') == 'gomad.success/v1'
                    and int(row['selection_ordinal']) == 0]
        failures = sorted(((path, row) for path, row in records if row.get('artifact_kind') == 'gomad.target-failure/v1'
                           and int(row['selection_ordinal']) > 0), key=lambda item: int(item[1]['selection_ordinal']))
        if defaults and failures:
            artifact, manifest = failures[0]
            trace = manifest['choice_profile']['trace']
            result = {'schema': 'gomad.minimize-counterfactual/v1', 'seed': seed, 'artifact': str(artifact),
                      'baseline': str(defaults[0][0]), 'baseline_record_hash': defaults[0][1]['record_hash'],
                      'forced_record_hash': manifest['record_hash'], 'forced_ordinal': int(manifest['selection_ordinal']),
                      'trace_path': str(artifact / trace['file']), 'decisions': int(trace['decisions'])}
            (output / 'witness.json').write_text(json.dumps(result, indent=2) + '\n')
            return result
    raise ValueError('corpus has no actual default-success/forced-failure witness')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'installation', 'output'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_')}
    receipts = []

    def run(name, command, status):
        result = subprocess.run(list(map(str, command)), capture_output=True, timeout=900, env=environment)
        (args.output / (name + '.stdout')).write_bytes(result.stdout)
        (args.output / (name + '.stderr')).write_bytes(result.stderr)
        receipts.append({'name': name, 'command': list(map(str, command)), 'status': result.returncode,
                         'stdout_sha256': hashlib.sha256(result.stdout).hexdigest(),
                         'stderr_sha256': hashlib.sha256(result.stderr).hexdigest()})
        (args.output / 'commands.json').write_text(json.dumps(receipts, indent=2) + '\n')
        if result.returncode != status:
            raise ValueError(f'{name}: status {result.returncode}, expected {status}')
    print(json.dumps(prepare(args.binary.resolve(), args.installation.resolve(), args.output.resolve(), run)))


if __name__ == '__main__':
    main()
