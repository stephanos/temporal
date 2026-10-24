"""Execute fixed broad native processes and refuse incomplete original-root receiving."""
import importlib
import re
from pathlib import Path

from driver import HERE, Refusal, guard, require, subgates, write
import broad_harness
import native_roles

SCHEMA = 'gomad.broad-stage-facts/v1'
COUNTS = {'cluster-provider': 78, 'cluster-functional': 46, 'retained-cluster': 24}
REMOVED_WIRE = (
    'TestClusterTransportCodecCorpus', 'TestClusterTransportCodecCorpusIsCurrent',
    'TestClusterTransportCodecLiteralStageGuards', 'TestClusterTransportStageCorpus',
    'TestClusterTransportStageCorpusIsCurrent', 'TestClusterTransportStageLiteralGuards',
)
IDENTITY_ROOTS = (
    'TestImplementationIdentityBindsCompiledModelDigest', 'TestImplementationIdentityBindsInspectionOwners',
    'TestImplementationIdentityBindsStorageEvidenceSources', 'TestImplementationIdentityBindsStrictJSONOwner',
    'TestImplementationIdentityFailsClosedOnMissingSource',
)
ERRORS = (Refusal, ValueError, KeyError, TypeError, OSError, ImportError)


def selection(config, label, root):
    require(type(label) is str and label in COUNTS, 'unknown broad original stage')
    root = Path(root)
    require(root == Path(config['root']) / 'stages' / label and root.is_absolute() and root.resolve() == root,
            'foreign broad original stage root')
    expected = subgates(label)
    require(len(expected) == COUNTS[label] and len(set(expected)) == len(expected), 'broad original subgate inventory changed')
    return broad_harness.Stage(label), root, expected


def preflight(config, label, root):
    stage, root, expected = selection(config, label, root)
    guard(config)
    native_roles.pinned(config, HERE / 'semantic-map.json')
    broad_harness.source_inputs(config)
    return dict(label=stage.value, root=str(root), original_roots=expected)


def raw_text(raw, name):
    require(name in raw['roles'], 'missing fixed broad raw process: ' + name)
    return native_roles.regular(raw['roles'][name]['stdout']).read_text()


def document(stdout, schema):
    rows = []
    for line in stdout.split('\n'):
        if schema in line:
            start = line.find('{')
            require(start >= 0, 'missing broad raw JSON envelope')
            row = native_roles.decode(line[start:])
            require(row.get('schema') == schema, 'foreign broad raw JSON schema')
            rows.append(row)
    require(len(rows) == 1, 'missing or duplicate broad raw observation: ' + schema)
    return rows[0]


def blocked(value):
    if isinstance(value, dict):
        for key, item in value.items():
            if key in ('remaining_original_predicates', 'remaining_predicates', 'partial_roots', 'receiving_dependencies', 'dependencies', 'gaps') and item:
                return True
            if blocked(item):
                return True
    elif isinstance(value, list):
        return any(blocked(item) for item in value)
    return False


def closure(expected, readers, errors):
    claims, partial = {}, {}
    for owner, result in readers.items():
        require(isinstance(result, dict) and isinstance(result.get('roots', {}), dict), 'foreign broad typed-reader facts')
        for root, facts in result.get('roots', {}).items():
            if root in expected:
                claims.setdefault(root, []).append((owner, facts))
        for root, facts in result.get('partial_roots', {}).items():
            if root in expected:
                partial.setdefault(root, {})[owner] = facts
        require(isinstance(result.get('dependencies', {}), dict), 'foreign broad dependency facts')
        for root, facts in result.get('dependencies', {}).items():
            if root in expected and facts:
                partial.setdefault(root, {})[owner + '-dependency'] = facts
    observed = []
    for root in expected:
        candidates = claims.get(root, [])
        reasons = []
        if root in REMOVED_WIRE and not (len(candidates) == 1 and candidates[0][0] == 'cluster-wire'
                and importlib.import_module('core_cluster_wire').applicable(root, candidates[0][1])):
            reasons.append(dict(disposition='unreceived-removed-wire-schema-or-literal',
                                reason='The retired Go transport schema and literal stage guards have no explicit native receiving contract.'))
        if not candidates:
            reasons.append(dict(disposition='unreceived-original-root', reason='No typed reader owns the complete original root.'))
        elif len(candidates) != 1:
            reasons.append(dict(disposition='ambiguous-whole-root-owners', owners=[owner for owner, _ in candidates]))
        elif not isinstance(candidates[0][1], (dict, list)) or not candidates[0][1] or blocked(candidates[0][1]):
            reasons.append(dict(disposition='incomplete-original-root', owner=candidates[0][0]))
        if root in partial:
            reasons.append(dict(disposition='remaining-original-predicates', readers=partial[root]))
        if reasons:
            observed.append(dict(id=root, status='REFUSED', executed=False, exit=None, dispositions=reasons))
        else:
            owner, facts = candidates[0]
            observed.append(dict(id=root, status='PASS', executed=True, exit=0, reader=owner, facts=facts))
    return observed, not errors and all(row['status'] == 'PASS' for row in observed)


def typed_readers(config, stage, raw, build):
    readers, errors = {}, {}

    def derive(name, callback):
        try:
            value = callback()
            require(isinstance(value, dict), 'typed broad reader returned non-object facts')
            readers[name] = value
        except ERRORS as error:
            errors[name] = dict(type=type(error).__name__, reason=str(error))

    def module(name):
        return importlib.import_module(name)

    helper = raw_text(raw, 'helper-lib')
    inspection = '\n'.join((helper, raw_text(raw, 'host-inspection'), raw_text(raw, 'host-inspection-scale')))
    lifecycle = '\n'.join((helper, raw_text(raw, 'host-scenario')))
    derive('world', lambda: module('core_world').observations(config, helper))
    derive('network', lambda: module('core_network').observations(config, helper))
    derive('codec', lambda: dict(roots=module('core_codec').observations(config, helper)))
    derive('boot-network', lambda: dict(partial_roots={module('core_boot_network').ROOT:
        module('core_boot_network').observations(config, helper)}))
    derive('presence', lambda: dict(partial_roots=module('core_presence').observations(
        config, raw_text(raw, 'host-presence'))))
    derive('lifecycle', lambda: module('core_lifecycle').observations(
        config, lifecycle, network_frame_cases=readers.get('network', {}).get('frame_cases')))
    if stage is broad_harness.Stage.PROVIDER:
        derive('cluster-wire', lambda: module('core_cluster_wire').observations(config, lifecycle))
    if 'boot-network' in readers and 'lifecycle' in readers:
        root = module('core_boot_network').ROOT
        modern = readers['boot-network']['partial_roots'][root]
        readers['lifecycle']['partial_roots'][root] = dict(
            received=modern['received'], additional_controls=modern['additional_controls'],
            remaining_original_predicates=modern['unreceived'], representation=modern['representation'])
    derive('inspection', lambda: dict(roots=module('core_inspection').observations(
        config, inspection, artifact_roots=[raw['roles']['helper-lib']['raw_artifact_root']])))
    derive('identity', lambda: dict(roots={root: facts for facts in [module('core_identity').observations(
        config, raw_text(raw, 'identity'), build['context'])] for root in IDENTITY_ROOTS}))
    derive('retained', lambda: module('core_retained').observations(config, raw_text(raw, 'host-retained')))
    if 'presence' in readers and 'retained' in readers:
        for root, facts in readers['presence']['partial_roots'].items():
            readers['retained'].get('partial_roots', {}).pop(root, None)
    derive('retained-inputs', lambda: module('core_retained_inputs').observations(config, raw_text(raw, 'host-retained')))
    def decoder():
        root = 'TestRetainedDecodeAcceptsOrdinarySpellingsAndRejectsStrictInput'
        require('retained' in readers and 'retained-inputs' in readers, 'decoder root lacks both independent typed readers')
        ordinary = readers['retained']['spelling_cases']
        require(isinstance(ordinary, list) and len(ordinary) == len({row['case'] for row in ordinary})
                and {row['case'] for row in ordinary} == {
                    'canonical', 'indented', 'member-order', 'escaped-member', 'escaped-value',
                    'trailing-space', 'explicit-optional-null'}, 'decoder root lacks exact ordinary spelling obligations')
        inputs = readers['retained-inputs']
        strict = inputs['supporting_obligations'][root]['strict_cases']
        expected = {'decode/' + name for name in (*module('core_retained_inputs').DECODE_CASES, 'native-valid')}
        require(isinstance(strict, list) and len(strict) == len(set(strict)) and set(strict) == expected,
                'decoder root lacks exact strict mutation obligations')
        facts = {name: inputs['facts'][name] for name in strict}
        require(not blocked(ordinary) and not blocked(facts), 'decoder root retains an original predicate gap')
        return dict(roots={root: dict(ordinary_spellings=ordinary, strict_inputs=facts,
                                     raw_process='host-retained', command=raw['roles']['host-retained']['command'])})

    derive('retained-decoder', decoder)
    derive('retained-matrix', lambda: dict(facts=module('core_retained_matrix').lifecycle_matrix(raw_text(raw, 'host-lifecycle-matrix'))))
    if stage in (broad_harness.Stage.PROVIDER, broad_harness.Stage.RETAINED):
        derive('guests', lambda: module('core_guests').observations(config, raw))
        derive('guest-retention', lambda: module('core_guest_retention').observations(config, raw))
        if 'guest-retention' in readers and 'guests' in readers:
            retention = readers['guest-retention']
            for root, facts in retention['source_closed_receiving'].items():
                require(facts.get('native_receiving_complete') is True and root in retention['partial_roots'],
                        'guest retention omitted its native completion or retired-form disposition')
                readers['guests'].get('partial_roots', {}).pop(root, None)
    writer = raw_text(raw, 'host-writer')
    derive('writer', lambda: dict(roots={'TestRetainedCanonicalWriterChecksEveryWriteAndBase64Close':
        module('core_predicates').retained_writer(document(writer, 'gomad.native-retained-writer-observation/v1'))}))
    derive('fields', lambda: dict(roots={'TestRetainedCanonicalWriterMatchesAllDeclaredFields':
        module('core_predicates').retained_fields(document(writer, 'gomad.native-retained-fields-observation/v1'))}))

    def allocation(name, target):
        owned = raw['roles'][name]
        text = raw_text(raw, name)
        require(re.search(r'(?m)^running 1 test$', text)
                and re.search(r'test result: ok\. 1 passed; 0 failed; 0 ignored; 0 measured; [0-9]+ filtered out;', text),
                'allocation process did not execute its one exact ignored test')
        allocator = module('allocation_receiving')
        rows = [native_roles.decode(line.split(allocator.MARKER, 1)[1]) for line in text.split('\n') if allocator.MARKER in line]
        facts = allocator.derive(rows, target, artifact_root=Path(owned['cwd']))
        if target == 'gomad':
            roots = {'TestRetainedCanonicalDecodeAvoidsWholeBundleEncodingAllocation': dict(
                predicates=[row for row in facts['predicates'] if row['case'] == 'retained-decode'])}
        else:
            roots = {root: dict(predicates=[row for row in facts['predicates'] if row['case'] == case]) for root, case in (
                ('TestWorldOwnerCodecMeasuredCardinality', 'binary-world'),
                ('TestCompressedWorldOwnerMeasuredCardinality', 'compressed-world'))}
        require(all(value['predicates'] for value in roots.values()), 'allocation reader omitted original receiving predicates')
        for value in roots.values():
            value['lineage'] = 'Native selected-thread requested allocations and actual owner round-trip/cardinality; no Go byte or GC metric equivalence.'
        partial = {}
        if target == 'wasmhost':
            partial['TestMountDecodersRejectOversizedLengthsBeforeAllocation'] = dict(
                predicates=[row for row in facts['predicates'] if row['case'] == 'namespace-file-length'],
                remaining_original_predicates=['The original mount request caller path-byte limit and response wire grammar are absent from native namespace-image decoding.'])
        return dict(roots=roots, partial_roots=partial, facts=facts, artifact_root=owned['cwd'])

    derive('host-allocation', lambda: allocation('host-allocation', 'gomad'))
    derive('helper-allocation', lambda: allocation('helper-allocation', 'wasmhost'))
    return readers, errors


def collect(config, label, root):
    stage, root, expected = selection(config, label, root)
    raw = broad_harness.read(config, stage)
    build = broad_harness.read_build(config)
    readers, errors = typed_readers(config, stage, raw, build)
    if label == 'cluster-functional':
        import counterfactual_receiving
        readers['executed-divergence'] = counterfactual_receiving.observations(config)
    observed, complete = closure(expected, readers, errors)
    return dict(schema=SCHEMA, label=label, source_sha256=config['source_sha256'],
                config_sha256=config['_config_sha256'], identity=config['identity'],
                build_context=build['context'], raw_custody=raw, readers=readers, reader_errors=errors,
                subgates=observed, complete=complete)


def accepted(facts):
    require(facts['complete'], 'broad original-root receiving remains incomplete: ' + ', '.join(
        row['id'] for row in facts['subgates'] if row['status'] != 'PASS') +
        ('; reader errors: ' + ', '.join(facts['reader_errors']) if facts['reader_errors'] else ''))
    return facts


def run(config, label, root):
    preflight(config, label, root)
    stage, root, expected = selection(config, label, root)
    try:
        broad_harness.execute_build(config)
        broad_harness.execute(config, stage)
        if stage is broad_harness.Stage.PROVIDER:
            reference = importlib.import_module('cluster_wire_reference')
            reference.collect(config, produce=not reference.owned(config).exists())
        if label == 'cluster-functional':
            import counterfactual_receiving
            counterfactual_receiving.run(config)
        facts = collect(config, label, root)
    except ERRORS as error:
        facts = dict(schema=SCHEMA, label=label, source_sha256=config['source_sha256'],
                     config_sha256=config['_config_sha256'], identity=config['identity'], readers={},
                     reader_errors={'execution': dict(type=type(error).__name__, reason=str(error))},
                     subgates=closure(expected, {}, {'execution': str(error)})[0], complete=False)
    write(root / 'broad-diagnostics.json', facts)
    return accepted(facts)


def read(config, label, root):
    preflight(config, label, root)
    _, root, _ = selection(config, label, root)
    facts = collect(config, label, root)
    require(native_roles.read(root / 'broad-diagnostics.json') == facts, 'broad diagnostics differ from independently reopened raw facts')
    guard(config)
    return accepted(facts)
