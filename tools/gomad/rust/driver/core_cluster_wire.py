"""Per-assertion retired-wire applicability plus actual typed native owner guards."""
from pathlib import Path

import cluster_wire_reference as reference
import core_lifecycle as lifecycle
import native_roles
from driver import Refusal, require, sha
from native_observations import digest

SCHEMA = 'gomad.cluster-wire-applicability/v1'
ASSERTIONS = {
    reference.ROOTS[0]: ('exact-one-named-fixture', 'strict-fields-and-trailing-eof',
        'schema-name-original-cluster-source-sha', 'target-a-incarnation-two',
        'minimal-result-and-old-prefix-raw-bytes', 'exact-tagged-result-and-invalid-attribution',
        'actual-go-tag-and-attribution-match-fixture'),
    reference.ROOTS[1]: ('actual-original-source-pin', 'actual-go-generator-observation',
        'indented-json-plus-newline-byte-equality', 'regeneration-disabled'),
    reference.ROOTS[2]: ('current-tag-reaches-unsupported-payload-not-attribution',
        'malformed-json-is-not-attribution', 'six-exact-literal-mutations-refused'),
    reference.ROOTS[3]: ('exact-two-named-fixtures', 'strict-fields-and-trailing-eof',
        'filename-matches-case-name', 'schema-source-pin-and-target-a-incarnation-two',
        'current-tag-exact-unsupported-payload-stage', 'malformed-json-exact-invalid-before-attribution',
        'actual-go-boundaries-match-both-fixtures'),
    reference.ROOTS[4]: ('actual-original-source-pin', 'actual-go-two-case-generator',
        'both-indented-json-plus-newline-byte-equalities', 'regeneration-disabled'),
    reference.ROOTS[5]: ('both-original-cases-pass-literal-guards',
        'both-cases-refuse-wrong-stage', 'both-cases-refuse-wrong-target', 'both-cases-refuse-stale-input'),
}
NATIVE = frozenset((lifecycle.ATTRIBUTION, variant) for variant in lifecycle.VARIANTS[lifecycle.ATTRIBUTION]) | {
    (lifecycle.RESTART, 'stale-incarnation'), (lifecycle.CALLBACK, 'call'), (lifecycle.CALLBACK, 'sync')}
SOURCES = ('tools/gomad/rust/wasmhost/src/linked_cluster.rs',
    'tools/gomad/rust/wasmhost/src/linked_cluster/lifecycle_predicate_tests.rs',
    'tools/gomad/rust/wasmhost/src/cluster/handoff/broker.rs')


def native_guards(stdout):
    lifecycle.parse_rows(stdout)
    selected = []
    for line in stdout.split('\n'):
        if lifecycle.SCHEMA not in line:
            continue
        row = native_roles.decode(line[line.index('{'):])
        if (row.get('original_root'), row['variant']) in NATIVE:
            selected.append(row)
    require(len(selected) == len(NATIVE)
            and {(row['original_root'], row['variant']) for row in selected} == NATIVE,
            'missing/duplicate native cluster wire applicability guard')
    return selected


def applicable(root, facts):
    if not (isinstance(facts, dict) and isinstance(facts.get('historical_reference'), dict)
            and isinstance(facts.get('source_bindings'), dict)):
        return False
    complete = (facts.get('schema') == SCHEMA
        and facts.get('original_root') == root and facts.get('original_pass') is False
        and facts.get('historical_reference', {}).get('testcases') == list((*reference.ROOTS, *reference.CASES))
        and facts.get('historical_reference', {}).get('schema') == reference.SCHEMA
        and facts.get('retired_assertions') == [dict(id=name, applicability='retired-framed-helper-boundary',
            historical_original_test=root) for name in ASSERTIONS[root]]
        and isinstance(facts.get('native_guards'), list) and len(facts['native_guards']) == len(NATIVE)
        and facts.get('boundary') == 'typed-native-ownership; no framed JSON peer or tag codec'
        and set(facts.get('source_bindings', {})) == set(SOURCES))
    if not complete:
        return False
    try:
        observed = set()
        for row in facts['native_guards']:
            lifecycle.fields(row, ('schema', 'original_root', 'variant', 'observed'))
            require(row['schema'] == lifecycle.SCHEMA, 'foreign native cluster applicability schema')
            case = (row['original_root'], row['variant'])
            require(case in NATIVE and case not in observed, 'foreign/duplicate native cluster applicability case')
            lifecycle.validate_case(*case, row['observed'])
            observed.add(case)
        return observed == NATIVE
    except (Refusal, KeyError, TypeError, ValueError):
        return False


def observations(config, stdout):
    proof = reference.collect(config)
    guards = native_guards(stdout)
    sources = {name: sha(native_roles.pinned(config, Path(config['repo']) / name)) for name in SOURCES}
    production = (Path(config['repo']) / SOURCES[0]).read_text()
    request = production.split('pub struct Request {', 1)[1].split('\n}', 1)[0]
    require(request.strip().splitlines() == [
        'pub nodes: Vec<String>,', '    pub broker: BrokerConfig,', '    pub network: Option<NetworkConfig>,',
        '    pub incarnations: Vec<Incarnation>,', '    pub steps: Vec<Step>,',
        '    pub boundaries: Vec<Boundary>,', '    pub crash_groups: Vec<CrashGroup>,',
        '    pub wall_timeout: Duration,', '    pub evidence_bytes: u64,'],
        'native cluster admission gained an unreviewed representation')
    require('struct NoRemote(Activity);' in production and 'linked cluster forbids transport callbacks' in production,
            'native cluster callback boundary changed')
    roots = {}
    for root in reference.ROOTS:
        facts = dict(schema=SCHEMA, original_root=root, original_pass=False,
            boundary='typed-native-ownership; no framed JSON peer or tag codec',
            retired_assertions=[dict(id=name, applicability='retired-framed-helper-boundary',
                historical_original_test=root) for name in ASSERTIONS[root]],
            historical_reference=proof, native_guards=guards, source_bindings=sources,
            native_stdout_sha256=digest(stdout.encode()),
            current_scope='Typed admission, callback refusal and unchanged stale-restart ownership only; no original wire byte equivalence.',
            terminal_scope='No fuel, output-counter, missing-terminal or terminal-to-reap credit belongs to these six codec/stage roots.')
        require(applicable(root, facts), 'incomplete cluster wire applicability contract')
        roots[root] = facts
    return dict(roots=roots)
