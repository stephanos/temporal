#!/usr/bin/env python3
"""Authenticate retained scope reports before checking their display-only golden."""
import argparse
import copy
import json
import os
from pathlib import Path
import runpy

HERE = Path(__file__).resolve().parent
GOLDEN = HERE / 'goldens-minimize/scoped-display.json'
rounds = runpy.run_path(str(HERE / 'round-check.py'))
canonical = rounds['parity'].canonical


def equal(actual, expected, label):
    if actual != expected:
        raise ValueError(label + ': mismatch')


def projection(minimized, inspected, retained):
    """Select presentation fields; authenticate every omitted record/candidate identity first."""
    equal(retained['schema'], 'gomad.minimization-evidence/v1', 'retained schema')
    raw_view = copy.deepcopy(inspected)
    groups = raw_view.pop('final_forced_by_symbol')
    for reduction in raw_view['accepted']:
        for decision in reduction['removed']:
            symbol = decision.pop('symbol')
            if not isinstance(symbol, dict) or not symbol:
                raise ValueError('removed symbol absent')
    equal(raw_view, retained, 'inspect report versus complete retained minimization block')
    equal(minimized['attempts'], retained['attempts'], 'attempt count')
    equal(minimized['attempt_budget'], retained['attempt_budget'], 'attempt budget')
    equal(len(minimized['accepted']), len(retained['accepted']), 'accepted count')
    equal(sum(group['count'] for group in groups), retained['final_forced_decisions'], 'final group count')
    reductions = []
    symbols = []
    for result, report, raw in zip(minimized['accepted'], inspected['accepted'], retained['accepted']):
        equal(result['kind'], raw['kind'], 'reduction kind')
        for name in ('before_sha256', 'after_sha256'):
            equal(result[name], 'sha256:' + raw[name], 'exact candidate identity ' + name)
        equal(result['removed'], [dict(dimension=item['dimension'], ordinal=item['ordinal'],
              identity='sha256:' + item['identity']) for item in raw['removed']], 'exact removed identities/ordinals')
        for name in ('removed_in_scope', 'removed_out_of_scope'):
            if type(result[name]) is not int or result[name] < 0:
                raise ValueError('invalid scope removal count')
        equal(result['removed_in_scope'] + result['removed_out_of_scope'], len(raw['removed']), 'scope removal total')
        reductions.append(dict(kind=result['kind'], removed_in_scope=result['removed_in_scope'],
            removed_out_of_scope=result['removed_out_of_scope'],
            removed=[dict(dimension=item['dimension'], ordinal=item['ordinal']) for item in result['removed']]))
        symbols.append(dict(kind=report['kind'], removed=[dict(dimension=item['dimension'],
            ordinal=item['ordinal'], symbol=copy.deepcopy(item['symbol'])) for item in report['removed']]))
    return dict(schema='gomad.site-minimizer-display-golden/v1',
        minimize=dict(site_scope=copy.deepcopy(minimized['site_scope']), accepted=reductions),
        inspect=dict(accepted=symbols, final_forced_decisions=retained['final_forced_decisions'],
                     final_forced_by_symbol=copy.deepcopy(groups)))


def authenticated_projection(minimized, inspected, artifact):
    record = rounds['authenticate_artifact'](artifact)
    manifest = rounds['canonical_file'](artifact / 'manifest.json')
    equal(record, manifest['record'], 'authenticated record')
    equal(minimized['record_hash'], manifest['record_sha256'], 'CLI record hash')
    retained = rounds['canonical_file'](artifact / 'evidence/inspection.json')['minimization']
    return projection(minimized, inspected, retained)


def compare(view, golden=GOLDEN):
    equal(canonical(view) + b'\n', Path(golden).read_bytes(), 'reviewed scoped minimizer display golden')


def check(minimized, inspected, artifact):
    view = authenticated_projection(minimized, inspected, artifact)
    compare(view)
    return canonical(view) + b'\n'


def from_scope(scope):
    scope = Path(scope)
    minimized = json.loads((scope / 'scope-failure-minimize.stdout').read_bytes())
    report = json.loads((scope / 'scope-failure-minimum-inspect.stdout').read_bytes())
    artifact = Path(minimized['artifact_path'])
    equal(report['artifact']['record'], rounds['canonical_file'](artifact / 'record.json'), 'inspect record')
    return authenticated_projection(minimized, report['artifact']['minimization'], artifact)


def repeated(views):
    if len(views) < 2:
        raise ValueError('two authenticated projections required before a checked write')
    raw = canonical(views[0]) + b'\n'
    for view in views[1:]:
        equal(canonical(view) + b'\n', raw, 'fresh repeated scoped display projection')
    return raw


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--scope-output', type=Path, action='append', required=True)
    args = parser.parse_args()
    views = [from_scope(scope) for scope in args.scope_output]
    raw = repeated(views)
    if os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1':
        if len({scope.resolve() for scope in args.scope_output}) < 2:
            raise ValueError('distinct authenticated scope outputs required for regeneration')
        GOLDEN.write_bytes(raw)
    else:
        compare(views[0])
    print('authenticated scoped minimizer display goldens match:', len(views))


if __name__ == '__main__':
    main()
