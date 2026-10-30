import json
from pathlib import Path
import sys
root = Path(__file__).resolve().parent
manifest = json.loads((root / 'manifest.json').read_text())
rows = [json.loads(line) for line in Path(sys.argv[1]).read_text().splitlines()]
actual = {Path(row['path']).name: row for row in rows}
assert len(actual) == len(rows) == len(manifest), (len(actual), len(rows), len(manifest))
for name, expected in manifest.items():
    row = actual[name]
    assert row['outcome'] == expected['outcome'], (name, expected, row)
    assert expected['message'] in row['message'], (name, expected, row)
    if expected['outcome'].endswith('refusal') or expected['outcome'] == 'parse_error':
        assert row['line'] > 0 and row['column'] > 0 and row['snippet'] and row['message'], row
print(json.dumps({'expected_outcomes_verified': len(rows), 'admitted': sum(row['outcome'] == 'admitted' for row in rows)}))
