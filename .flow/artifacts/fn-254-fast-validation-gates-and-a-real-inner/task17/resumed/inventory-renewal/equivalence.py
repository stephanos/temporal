#!/usr/bin/python3
"""Compare supplied inventory receipts only. Does not invoke Cargo or execute test bodies."""
import argparse
import json
from pathlib import Path
import re


def listed(path):
    names = [match.group(1) for line in Path(path).read_text().splitlines() if (match := re.fullmatch(r'(.+): (?:test|benchmark)', line))]
    assert len(names) == len(set(names)), 'duplicate libtest identity in raw listing'
    return set(names)


def cargo(rows, workspace):
    result = set()
    for row in rows:
        names, ignored = listed(row['all_list']), listed(row['ignored_list'])
        assert ignored <= names
        kind = 'lib' if row['kind'] in ['rlib', 'cdylib', 'dylib', 'staticlib'] else row['kind']
        identities = {(workspace, row['package'], kind, row['binary'], name, name in ignored) for name in names}
        assert not result.intersection(identities), 'duplicate Cargo harness identity'
        result.update(identities)
    return result


def nextest(inventory, workspace):
    result, selected = set(), set()
    for suite in inventory['rust-suites'].values():
        if suite['status'] != 'listed':
            assert not suite.get('testcases'), 'nonlisted suite unexpectedly carries identities'
            continue
        for name, case in suite['testcases'].items():
            row = (workspace, suite['package-name'], suite['kind'], suite['binary-name'], name, case['ignored'])
            if case['filter-match']['status'] == 'matches':
                assert row not in result, 'duplicate nextest selected identity'
                result.add(row)
                if not case['ignored']:
                    selected.add(row)
    return result, selected


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--cargo-lists', type=Path, required=True, help='JSON rows: package,kind,binary,all_list,ignored_list')
    parser.add_argument('--nextest-json', type=Path, required=True)
    parser.add_argument('--workspace', required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--historical-fixture-map', type=Path, help='Only for pre-task5 historical Cargo rows; not for current-source Cargo listing')
    options = parser.parse_args()
    old = cargo(json.loads(options.cargo_lists.read_text()), options.workspace)
    mapping = None
    if options.historical_fixture_map:
        mapping = json.loads(options.historical_fixture_map.read_text())
        expanded = set()
        for row in old:
            if row[1:5] == ('umpire-fixtures', 'test', 'fixtures', 'fixtures'):
                assert not row[5]
                expanded.update((*row[:4], name, False) for name in mapping['new_fixture_cases'])
            else:
                expanded.add(row)
        old = expanded
    current, selected = nextest(json.loads(options.nextest_json.read_text()), options.workspace)
    comparison = dict(scope='identity listing only; not suite execution', equal=old == current, missing=sorted(old - current), extra=sorted(current - old), ignored_cargo=sorted(row for row in old if row[5]), ignored_nextest=sorted(row for row in current if row[5]), current_default_selected=sorted(selected), historical_fixture_mapping=bool(mapping))
    options.output.write_text(json.dumps(comparison, indent=2) + '\n')
    assert comparison['equal'], comparison


if __name__ == '__main__':
    main()
