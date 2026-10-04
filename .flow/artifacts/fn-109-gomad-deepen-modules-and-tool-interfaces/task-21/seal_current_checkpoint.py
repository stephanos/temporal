#!/usr/bin/env python3
"""Freeze the verified evidence-only checkpoint without staging or lifecycle writes."""
from datetime import datetime, timezone
import argparse
import hashlib
import json
from pathlib import Path
import stat

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[3]
parser = argparse.ArgumentParser()
parser.add_argument('--check',action='store_true',help='Verify the existing seal without replacing it.')
args = parser.parse_args()
def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()
def read(path):
    return json.loads(path.read_text())
selection = read(HERE/'current-checkpoint-selection.json')
for row in selection['files']:
    path = REPO/row['path']
    assert digest(path) == row['sha256'] and path.stat().st_size == row['bytes'], path
verified = read(HERE/'handover-verification.json')
assert verified['passed'] is True and verified['acceptance_complete'] is False
assert verified['handover_evidence_sha256'] == digest(HERE/'current-qualification-evidence.json')
source = read(HERE/'current-measurement/runs/shipped-source-before-overlay.json')
for row in source:
    path = REPO/'tools/gomad3'/row['path']
    assert digest(path) == row['sha256'] and stat.S_IMODE(path.stat().st_mode) == row['mode'], path
paths = [HERE/name for name in ['current-checkpoint-selection.json',
    'current-qualification-evidence.json','current-qualification-handover.md',
    'handover-verification.json']]
paths += [HERE.parent/'completion-matrix.md',HERE.parent/'qualification-evidence.md',REPO/'MILESTONES.md']
result = dict(frozen_utc=datetime.now(timezone.utc).isoformat(),
    source_candidate=verified['source_candidate'],acceptance_complete=False,
    selected_files=len(selection['files']),selected_bytes=selection['total_bytes'],
    current_shipped_source_paths=len(source),
    artifact_sha256={str(path.relative_to(REPO)):digest(path) for path in paths},
    retention='Retain this seal, the checkpoint selection and handover-verification receipt in addition to the listed lean paths. Bulk outputs remain local and manifest-hash identified. No staging or lifecycle write performed.')
if args.check:
    frozen = read(HERE/'current-checkpoint-freeze.json')
    assert frozen['artifact_sha256'] == result['artifact_sha256']
    assert frozen['selected_files'] == result['selected_files']
    assert frozen['selected_bytes'] == result['selected_bytes']
    assert frozen['current_shipped_source_paths'] == result['current_shipped_source_paths']
    print(json.dumps(dict(passed=True,selected_files=result['selected_files'],selected_bytes=result['selected_bytes'],seal_sha256=digest(HERE/'current-checkpoint-freeze.json'),acceptance_complete=False),indent=2))
else:
    (HERE/'current-checkpoint-freeze.json').write_text(json.dumps(result,indent=2)+'\n')
    print(json.dumps(result,indent=2))
