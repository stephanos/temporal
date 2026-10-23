#!/usr/bin/env python3
"""Fail-closed composition of the live reader and separately owned final endpoints."""
import argparse
import json
from pathlib import Path
import sys

from driver import HERE, Refusal, Unexecuted, command, configured, coverage, guard, read, require, sha
import live
import receiving


def merge_reports(config, reports):
    rows = []
    for report in reports:
        require(report['schema'] == 'gomad.rust-driver-final-section/v1'
                and report['source_sha256'] == config['source_sha256'] and report['identity'] == config['identity']
                and not report.get('synthetic', False), 'foreign/synthetic final reader section')
        rows.extend(report['subgates'])
    coverage(read(HERE / 'required-final.json')['subgates'], rows)
    return rows


def remaining_ids():
    required = set(read(HERE / 'required-final.json')['subgates'])
    live_ids, receiving_ids = set(live.required()), set(receiving.OWNED)
    require(not live_ids & receiving_ids and live_ids | receiving_ids <= required,
            'overlapping/unknown owned final endpoints')
    return required - live_ids - receiving_ids


def receiving_report(config):
    # Invoke the independent raw-evidence reader; source presence earns no row.
    rows = receiving.read_receiving(config)
    coverage(receiving.OWNED, rows)
    return dict(schema='gomad.rust-driver-final-section/v1', source_sha256=config['source_sha256'],
                identity=config['identity'], subgates=rows)


def collect(config, root):
    guard(config)
    require(root == Path(config['root']), 'wrong final attempt root')
    remaining = remaining_ids()
    endpoints = config.get('final_sections', {})
    observed = [item for entry in endpoints.values() for item in entry['required']]
    if set(observed) != remaining or len(observed) != len(set(observed)):
        raise Unexecuted('missing/duplicate nonowned final endpoints: ' + repr(sorted(remaining ^ set(observed))))
    reports = [dict(schema='gomad.rust-driver-final-section/v1', source_sha256=config['source_sha256'],
                    identity=config['identity'], subgates=live.read_corpus(config, root)), receiving_report(config)]
    for name, entry in sorted(endpoints.items()):
        require(re_full_name(name), 'invalid final endpoint name')
        argv = entry['argv']
        require(len(argv) >= 2 and argv[1].endswith('.py'), 'final endpoint must be an actual pinned reader')
        for path in argv[:2]: live.sealed_input(config, path)
        raw = command(config, root, 'final-section-' + name, argv + ['--root', str(root)])
        report = json.loads(raw)
        coverage(entry['required'], report['subgates'])
        reports.append(report)
    rows = merge_reports(config, reports)
    guard(config)
    return dict(schema='gomad.rust-driver-final-check/v1', source_sha256=config['source_sha256'],
                identity=config['identity'], subgates=rows)


def re_full_name(value):
    return isinstance(value, str) and value and all(char.isalnum() or char in '-_' for char in value)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path, required=True)
    parser.add_argument('--root', type=Path, required=True)
    args = parser.parse_args()
    try:
        config = configured(args.config)
        config['_config_path'] = str(args.config.resolve())
        config['_config_sha256'] = sha(args.config)
        print(json.dumps(collect(config, args.root.resolve()), separators=(',', ':')))
        return 0
    except (Refusal, OSError, ValueError, KeyError, AssertionError) as error:
        print('REFUSED: ' + str(error), file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
