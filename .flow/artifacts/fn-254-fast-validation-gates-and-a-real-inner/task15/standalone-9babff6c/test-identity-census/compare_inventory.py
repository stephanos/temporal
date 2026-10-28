"""Compare an actual fresh framework inventory against accepted60f identities plus explicit renames."""
import argparse
import json
from pathlib import Path

HERE = Path(__file__).resolve().parent

def identities(value):
    if isinstance(value, list):
        return value
    return sorted([suite['package-name'], suite['binary-name'], suite['kind'], name, case['ignored']]
                  for suite in value['rust-suites'].values()
                  for name, case in suite.get('testcases', {}).items())

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('fresh_inventory', type=Path)
    parser.add_argument('--output', required=True, type=Path)
    options = parser.parse_args()
    baseline = json.loads((HERE/'baseline-child-inventory-identities.json').read_text())
    mapping = json.loads((HERE/'anticipated-identity-map.json').read_text())
    fresh = identities(json.loads(options.fresh_inventory.read_text()))
    assert len(fresh) == len(set(map(tuple, fresh))), 'duplicate fresh identity'
    renames = {tuple(row['baseline_runtime_identity']): tuple(row['anticipated_current_identity'])
               for row in mapping['framework_and_driver_renames']}
    expected = {renames.get(tuple(row), tuple(row)) for row in baseline}
    actual = set(map(tuple, fresh))
    missing = sorted(expected - actual)
    result = {'accepted': not missing, 'scope': 'framework baseline identity retention only',
              'baseline_rows': len(baseline), 'actual_fresh_rows': len(fresh),
              'actual_fresh_ignored': sum(row[-1] for row in fresh),
              'allowed_rename_count': len(renames), 'missing_translated_baseline_identities': missing,
              'actual_additional_identities': sorted(actual - expected),
              'fresh_inventory': str(options.fresh_inventory),
              'limits': 'Not a substitute for standalone stage/count/doctest/PID validation; Temporal source map needs its own actual inventory.'}
    options.output.write_text(json.dumps(result, indent=2)+'\n')
    print(json.dumps(result, indent=2))
    return 0 if result['accepted'] else 1

if __name__ == '__main__':
    raise SystemExit(main())
