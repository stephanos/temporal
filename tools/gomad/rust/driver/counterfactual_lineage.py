"""One assertion superseded by the accepted manual-cluster failure contract.

Callers must first authenticate all four campaigns, comparisons and public replays.
This module never converts the original result to PASS or supplies runtime evidence.
"""
from driver import require

ROOT = 'TestDeterminismExecutedDivergenceArtifacts'
SOURCE = 'tools/gomad/internal/backends/wasm/backend/determinism_integration_test.go'
AUTHORITY = dict(
    spec='.flow/specs/fn-235-gomad-fail-cluster-runs-fast-and.md',
    sha256='e91e343c4b58881e098774eb93c6baab947b57be90b94968a451ba68536e23d0',
    requirement='R2',
    implementation='a959c074baca3624107f159a8a6aeddd3070c4c6',
    task='fn-235-gomad-fail-cluster-runs-fast-and.1',
)
ASSERTION = dict(source=SOURCE, helper='recordDeterminismArtifact', line=170,
    source_sha256='d3c75d7057a21567cf2bbf29d4cf2080d65288a83084e307eb09ff395d07afe5',
    predicate='every reported incarnation termination equals exit', case='handoff',
    handle=dict(node='b', incarnation=1))


def reconcile(report):
    require(report['status'] == 'FAIL_ORIGINAL_PREDICATE', 'original all-exit failure must remain visible')
    failures = report['original_predicate_failures']
    handoff = [row for row in report['pairs'] if row['name'] == 'handoff']
    require(len(report['pairs']) == 4 and {row['name'] for row in report['pairs']} == {'draw', 'clock', 'handoff', 'output-clock'}
            and len(handoff) == 1, 'lineage cannot replace the four original campaigns')
    expected_artifacts = {handoff[0][role]['artifact'] for role in ('baseline', 'fault')}
    require(len(failures) == len(expected_artifacts) == 2
            and {failure['artifact'] for failure in failures} == expected_artifacts,
            'lineage applies only to both actual handoff artifacts')
    for failure in failures:
        require(failure['original_root'] == ROOT + '/handoff'
                and failure['predicate'] == ASSERTION['predicate']
                and failure['handle'] == ASSERTION['handle'] and failure['expected'] == 'exit'
                and failure['actual'] == 'stop', 'another failed assertion cannot be superseded')
        peer = failure['observation']
        require(peer['handle'] == ASSERTION['handle'] and peer['termination'] == 'stop'
                and peer['exit_code'] is None and peer['interrupted'] is True
                and peer['reason'] == 'WASI stop: cluster failed',
                'peer terminal is not the authorized cluster-failure stop')
    return dict(schema='gomad.original-assertion-applicability/v1', original_status='FAIL_ORIGINAL_PREDICATE',
        assertion=ASSERTION, applicability='SUPERSEDED_BY_ACCEPTED_OWNER_CONTRACT', owner_reference=AUTHORITY,
        current_contract='failed incarnation exits nonzero; its prestarted peer stops and is joined without later controls',
        required_obligations=['all four original campaigns', 'all original divergence predicates',
            'all provider and literal public prefix replays', 'same-stage native trap/exit/held-peer-release drain observations'])


def mapping(entries, applicability):
    """Require assertion-scoped lineage on the exact original root, never an alias."""
    key = SOURCE + '#' + ROOT
    require(key in entries and entries[key].get('assertion_applicability') == applicability,
            'original divergence map lacks exact assertion-scoped owner lineage')
    require(set(entries[key]['evidence']) == {'original-divergence/' + name for name in ('draw', 'clock', 'handoff', 'output-clock')},
            'original divergence root must receive all four actual pairs')
    cases = ('draw', 'clock', 'handoff', 'output-clock')
    require(all(key + '@9714/' + case in entries for case in cases),
            'original divergence dynamic subcase omitted')
    for name, entry in entries.items():
        if 'assertion_applicability' in entry:
            require(name == key, 'handoff lineage cannot waive another original root or subtest')
        if name.startswith(key + '@9714/'):
            case = name.removeprefix(key + '@9714/')
            require(case in ('draw', 'clock', 'handoff', 'output-clock')
                    and entry['evidence'] == ['original-divergence/' + case],
                    'original divergence subcase was substituted or renamed')
