import datetime
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

proof = Path(__file__).resolve().parent
root = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
report = json.loads((proof / 'independent-source-review.json').read_text())
for name, expected in report['artifact_sha256'].items():
    assert hashlib.sha256((proof / name).read_bytes()).hexdigest() == expected, name
evidence = json.loads((proof / 'review-audit-evidence.json').read_text())
for name, expected in evidence['writer_proof_hashes'].items():
    assert hashlib.sha256((proof / name).read_bytes()).hexdigest() == expected, name
for phase in ['baseline', 'final', 'review-final']:
    previous_end = None
    for name in ['private-mode', 'portable-cli', 'boundaries', 'lint', 'errortype', 'gofmt']:
        receipt = json.loads((proof / f'{phase}-{name}.receipt.json').read_text())
        start = datetime.datetime.fromisoformat(receipt['start'])
        end = datetime.datetime.fromisoformat(receipt['end'])
        assert end >= start and (previous_end is None or start >= previous_end)
        previous_end = end
        log = root / receipt['log']
        assert hashlib.sha256(log.read_bytes()).hexdigest() == receipt['log_sha256']
        assert receipt['exit_code'] == (1 if name == 'lint' else 0)
        if name in ['private-mode', 'portable-cli', 'boundaries']:
            assert len(re.findall(r'^--- PASS: ', log.read_text(), re.M)) == {'private-mode': 1, 'portable-cli': 34, 'boundaries': 5}[name]
        if name == 'gofmt':
            assert not log.read_bytes()
result = subprocess.run([sys.executable, str(proof / 'review-audit.py')], cwd=root, check=True, capture_output=True, text=True)
print(json.dumps({'verdict': report['verdict'], 'review_artifact_bindings': len(report['artifact_sha256']), 'writer_artifact_bindings': len(evidence['writer_proof_hashes']), 'serial_receipts': 18, 'source_audit': json.loads(result.stdout), 'read_only': True}, indent=2))
