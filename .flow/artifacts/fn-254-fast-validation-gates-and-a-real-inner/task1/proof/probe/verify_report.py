#!/usr/bin/env python3
"""Check report transcriptions and references without executing project code."""
import ast
import json
from pathlib import Path
import re
import sys

report = Path(sys.argv[1]).resolve()
main = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
text = report.read_text()
receipt = json.loads((main/'.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/validation.json').read_text())
for stage in receipt['stages']:
    row = re.search(r'^\| ' + re.escape(stage['name']) + r' \| (.+?) \|', text, re.M)
    assert row, stage['name']
    assert float(row[1].replace('*','')) == stage['seconds'], (stage['name'], row[1], stage['seconds'])
for link in re.findall(r'\]\(([^)]+)\)', text):
    assert (report.parent/link).exists(), link
for script in Path(__file__).parent.glob('*.py'):
    ast.parse(script.read_text())
for task in range(2,14):
    assert re.search(r'^\| \d+ \| \.'+str(task)+r' ', text, re.M), task
assert '24→14' in text and 'code-derived prediction' in text
assert '790 passed, 0 failed, 4 inherited ignored' in text
assert 'No fresh gate, build, test inventory or TLC measurement' in text
historical = json.loads((main/'.flow/tmp/fn254-baseline/historical/summary.json').read_text())
assert len(historical['binary_runs']) == 77
for entry in historical['inputs']:
    import hashlib
    path = main/entry.get('preserved', entry['path'])
    assert hashlib.sha256(path.read_bytes()).hexdigest() == entry['sha256'], str(path)
print('report stage transcriptions, source links, 12 task priorities, retained hashes and scratch Python syntax verified')
