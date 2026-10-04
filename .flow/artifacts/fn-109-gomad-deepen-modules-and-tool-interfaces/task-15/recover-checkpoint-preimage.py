#!/usr/bin/env python3
"""Recover the exact characterized fixture before task 16's callback wiring."""
import hashlib
import json
from pathlib import Path
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
SOURCE = 'tools/gomad3/runner/internal/execution/simulation_progress_fixture_test.go'
CURRENT = '12cbd8911c03b5ed4e0717397fec2742508daab3e28b8e91a6ca376671db8add'
EXPECTED = '35ed448549b3aa5d6ce959d86a631b37979056642144e31274e18bcbfb0e8e5c'


def main():
    data = (ROOT / SOURCE).read_bytes()
    assert hashlib.sha256(data).hexdigest() == CURRENT
    replacements = [
        (b'_ = coordinator.time.progress.apply(modelRequestDispatched{coordinator: coordinator.coordinator})',
         b'coordinator.time.deliverExternal(coordinator.coordinator)'),
        (b'coordinator.time.progress.apply(modelAbandonedResponseDiscarded{coordinator: coordinator.coordinator, arrivals: frame.Arrivals})',
         b'coordinator.time.acknowledgeExternal(coordinator.coordinator, frame.Arrivals)'),
    ]
    for before, after in replacements:
        assert data.count(before) == 1
        data = data.replace(before, after)
    assert hashlib.sha256(data).hexdigest() == EXPECTED
    scratch = Path(subprocess.check_output([
        'mktemp', '-d', '/tmp/fn109-task15-preimage.XXXXXXXX']).decode().strip())
    destination = scratch / SOURCE
    destination.parent.mkdir(parents=True)
    destination.write_bytes(data)
    report = dict(task=15, path=SOURCE, current_sha256=CURRENT,
                  recovered_sha256=EXPECTED, scratch=str(scratch),
                  replacements=[dict(before=a.decode(), after=b.decode()) for a, b in replacements],
                  provenance='Reverse only two task-16 callback-wiring substitutions; complete task-15 evidence hash matches.',
                  shared_source_writes=False, git_mutations=False,
                  native_qualification=False)
    (OUT / 'checkpoint-preimage.json').write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report))


if __name__ == '__main__':
    main()
