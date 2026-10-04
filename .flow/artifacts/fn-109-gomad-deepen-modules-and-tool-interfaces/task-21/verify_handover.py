#!/usr/bin/env python3
"""Check evidence scope, links, mappings and immutable source identities."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import stat
import subprocess
import time

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[3]
ARTIFACT = HERE.parent
started = time.monotonic()
parser = argparse.ArgumentParser()
parser.add_argument('--read-only',action='store_true',help='Check without rewriting the frozen verification receipt.')
args = parser.parse_args()
def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()
def read(path):
    return json.loads(path.read_text())
matrix = (ARTIFACT/'completion-matrix.md').read_text()
findings = re.findall(r'^\| (F\d+|S\d+) \|',matrix,re.M)
assert findings == ['F'+str(n) for n in range(1,12)]+['S'+str(n) for n in range(1,6)]
obligations = re.findall(r'^\| (D\d+) \|',matrix,re.M)
assert obligations == ['D'+str(n) for n in range(1,6)]
broken = []
for report in [ARTIFACT/'completion-matrix.md',ARTIFACT/'qualification-evidence.md',HERE/'current-qualification-handover.md',HERE/'current-measurement/measurement.md']:
    for target in re.findall(r'\[[^]]+\]\(([^)]+)\)',report.read_text()):
        if target.startswith(('http:','https:','#')):
            continue
        path = report.parent/target.split('#')[0]
        if not path.exists():
            broken.append(dict(report=str(report.relative_to(REPO)),target=target))
assert not broken, broken
ledger = read(HERE/'native-command-ledger.json')
assert all(not row['executed'] and row['exit_code'] is None for row in ledger['commands'])
for source,expected in ledger['source_sha256'].items():
    assert digest(REPO/source) == expected, source
for row in ledger['disposition_sources']:
    assert digest(REPO/row['path']) == row['current_sha256'], row['path']
audit = HERE/'preservation-audit'
assert digest(audit/'report.md') == '0fc86685ce46a6a2f4b1f1d4550f93e307ca6086216bc1b8d8e881d57d3bfd33'
audit_manifest = audit/'output.sha256'
if not audit_manifest.exists():
    choices = list(audit.glob('*.sha256'))
    assert len(choices) == 1, choices
    audit_manifest = choices[0]
assert digest(audit_manifest) == '028ef71d3631dce74844522a1314f96c4376c7ed375820e9444168b14c515162'
audit_checked = 0
for line in audit_manifest.read_text().splitlines():
    expected,relative = line.split('  ',1)
    assert digest(REPO/relative) == expected, relative
    audit_checked += 1
receipt = read(audit/'focused-preservation-receipt.json')
assert receipt['exit_code'] == 0 and not receipt['skip_lines']
assert len(receipt['expected_top_level_tests']) == 18 and len(receipt['selected_test_names']) == 55
assert digest(audit/'focused-preservation.log') == receipt['log_sha256']
base = '8604c07def0f97b63cbca3864b4c286d6803c4b1'
tracked_changes = subprocess.run(['git','diff','--name-only',base],cwd=REPO,stdout=subprocess.PIPE,text=True,check=True).stdout.splitlines()
assert all(path == 'MILESTONES.md' or path.startswith('.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/') for path in tracked_changes), tracked_changes
milestone_diff = subprocess.run(['git','diff','--unified=0',base,'--','MILESTONES.md'],cwd=REPO,stdout=subprocess.PIPE,text=True,check=True).stdout
assert len(re.findall(r'^@@ ',milestone_diff,re.M)) == 2
source_inventory = read(HERE/'current-measurement/runs/shipped-source-before-overlay.json')
for row in source_inventory:
    path = REPO/'tools/gomad3'/row['path']
    assert digest(path) == row['sha256'], row['path']
    assert stat.S_IMODE(path.stat().st_mode) == row['mode'], row['path']
evidence = read(HERE/'current-qualification-evidence.json')
assert evidence['base_commit'] == base and evidence['commits'] == []
assert evidence['status'] == 'in_progress' and evidence['acceptance_complete'] is False
assert evidence['bounded_measurement']['completed'] is True
assert evidence['preservation']['complete'] is False
assert evidence['native_qualification']['complete'] is False
assert evidence['lint']['result'] == 'failed'
for relative,expected in evidence['artifact_sha256'].items():
    assert digest(REPO/relative) == expected, relative
result = dict(passed=True,finding_rows=len(findings),obligations=len(obligations),source_files=len(source_inventory),native_command_rows=len(ledger['commands']),preservation_output_hashes=audit_checked,tracked_changes=tracked_changes,source_candidate=base,acceptance_complete=False,handover_evidence_sha256=digest(HERE/'current-qualification-evidence.json'),elapsed_seconds=time.monotonic()-started)
if not args.read_only:
    (HERE/'handover-verification.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result,indent=2))
