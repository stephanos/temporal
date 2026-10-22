#!/usr/bin/env python3
"""Exercise public recovery on authenticated prepare-only seed and round journals."""
import argparse
import fcntl
import hashlib
import json
import os
from pathlib import Path
import runpy
import subprocess


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def snapshot(root):
    return {str(path.relative_to(root)): digest(path.read_bytes())
            for path in root.rglob('*') if path.is_file()}


def check(binary, installation, root, working_dir,
          package='./internal/backends/wasm/qualification/testdata/fixture'):
    binary, installation, root, working_dir = [path.resolve() for path in (binary, installation, root, working_dir)]
    root.mkdir(parents=True)
    parity = runpy.run_path(str(Path(__file__).with_name('campaign-check.py')))
    receipts, observations = [], []
    environment = {key: value for key, value in os.environ.items() if not key.startswith("GOMAD_")}

    def run(name, arguments, status):
        result = subprocess.run([str(binary), *map(str, arguments)], capture_output=True, timeout=240, env=environment, cwd=root)
        (root / (name + '.stdout')).write_bytes(result.stdout)
        (root / (name + '.stderr')).write_bytes(result.stderr)
        receipts.append(dict(name=name, arguments=list(map(str, arguments)), status=result.returncode,
                             stdout_sha256=digest(result.stdout), stderr_sha256=digest(result.stderr)))
        (root / 'commands.json').write_text(json.dumps(receipts, sort_keys=True, indent=2) + '\n')
        if result.returncode != status:
            raise ValueError(f'{name}: status {result.returncode}, expected {status}: {result.stdout!r} {result.stderr!r}')
        return json.loads(result.stdout)

    for kind in ('seed', 'round'):
        output = root / (kind + '-plan.json')
        extra = [] if kind == 'seed' else ['--strategy=choice-exploration', '--choices',
            '--max-executions=1', '--max-choice-depth=1', '--max-exploration-bytes=1MiB']
        prepared = run(kind + '-plan', ['--installation', installation, 'plan', '--json',
            '--output', output, '--artifacts', root / (kind + '-artifacts'),
            '--working-dir', working_dir, '--seeds=7', *extra,
            'go-run', package], 0)
        assert prepared['guest_executions'] == 0
        campaign = Path(prepared['campaign'])
        storage = parity['canonical_file'](campaign / 'storage.json')
        plan = parity['canonical_file'](campaign / 'plan.json')
        assert storage['plan_sha256'] == digest((campaign / 'plan.json').read_bytes())
        metadata = plan if kind == 'seed' else plan['prepared']
        record = parity['authenticate_artifact'](campaign / metadata['prepared_artifact'])
        assert record['outcome']['class'] == 'prepared'
        if kind == 'round':
            runpy.run_path(str(Path(__file__).with_name('round-check.py')))['authenticate'](campaign)
        journal = campaign / 'journal.jsonl'
        assert journal.read_bytes() == b''
        original = snapshot(campaign)
        journal.write_bytes(b'{"event":')
        reports = []
        for attempt, changed in enumerate((True, False)):
            report = run(f'{kind}-recover-{attempt}', ['recover', '--json', campaign.relative_to(root)], 0)
            assert set(report) == {'schema', 'path', 'action', 'changed'}
            assert report['schema'] == 'gomad.recovery/v1'
            assert report['path'] == str(campaign)
            assert report['changed'] is changed
            assert report['action'] == ('repair' if changed else 'none')
            assert snapshot(campaign) == original
            reports.append({key: report[key] for key in ('schema', 'action', 'changed')})
        with (campaign / '.lock').open('rb') as lock:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            locked = snapshot(campaign)
            refused = run(kind + '-locked', ['recover', '--json', campaign], 3)
            assert refused['classification'] == 'runner_failure'
            assert 'writer lock unavailable' in refused['message']
            assert snapshot(campaign) == locked
        journal.write_bytes(b'garbage')
        corrupt = snapshot(campaign)
        invalid = run(kind + '-malformed', ['recover', '--json', campaign], 2)
        assert invalid['classification'] == 'invalid_input'
        assert 'malformed' in invalid['message']
        assert snapshot(campaign) == corrupt
        journal.write_bytes(b'')
        observations.append(dict(kind=kind, preparation_guest_executions=0,
            recovery=reports, repaired_snapshot_matches=True, malformed_refused_unchanged=True,
            locked_status=3, locked_refused_unchanged=True,
            prepared_artifact_authenticated=True, prepared_record=record,
            prepared_record_sha256=digest((campaign / metadata["prepared_artifact"] / "record.json").read_bytes()),
            module_sha256=metadata["module_sha256"]))
    return dict(schema='gomad.recovery-golden/v1', campaigns=observations)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    for flag in ('binary', 'installation', 'work', 'working-dir'):
        parser.add_argument('--' + flag, required=True, type=Path)
    parser.add_argument('--package', default='./internal/backends/wasm/qualification/testdata/fixture')
    args = parser.parse_args()
    print(json.dumps(check(args.binary, args.installation, args.work, args.working_dir, args.package), sort_keys=True))
