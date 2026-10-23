"""Read exact selected-incarnation network admission; old boot-map scope stays explicit."""
import copy
import hashlib
import json

from driver import require
import native_roles

MARKER = 'GOMAD_CORE_OBSERVATION '
ROOT = 'TestProviderBindsEverySelectedBootNetworkLimit'
VARIANTS = ('matching', 'network free', 'first byte mismatch', 'later byte mismatch',
            'first stall mismatch', 'later stall mismatch', 'restart byte mismatch',
            'restart stall mismatch', 'matching explicit broker', 'matching restart', 'independent local capacities', 'broker byte mismatch', 'broker stall mismatch')
MODEL = dict(profile='gomad.simulation-network-stream/v3', limits=dict(listeners=8, connections=8,
             bytes=1024, connection_bytes=123, stall_nanos=456, transitions=128), faults=[])


def same(left, right):
    return json.dumps(left, sort_keys=True, separators=(',', ':')) == json.dumps(right, sort_keys=True, separators=(',', ':'))


def expected(variant):
    require(variant in VARIANTS, 'foreign selected boot network variant')
    shared = copy.deepcopy(MODEL)
    selected = [dict(node='a', incarnation=1, network=copy.deepcopy(MODEL)), dict(node='b', incarnation=1, network=copy.deepcopy(MODEL))]
    if variant.startswith('restart ') or variant == 'matching restart':
        selected.append(dict(node='a', incarnation=2, network=copy.deepcopy(MODEL)))
    field = 'stall_nanos' if 'stall' in variant else 'connection_bytes'
    if variant.startswith('broker '):
        shared['limits'][field] += 1
    for prefix, index in (('first ', 0), ('later ', 1), ('restart ', 2)):
        if variant.startswith(prefix):
            selected[index]['network']['limits'][field] += 1
    if variant == 'independent local capacities':
        for key in ('bytes', 'transitions', 'listeners', 'connections'):
            selected[1]['network']['limits'][key] *= 2
    caller = dict(shared=shared if variant != 'network free' else None, selected=[dict(item, network=None if variant == 'network free' else item['network']) for item in selected])
    config = None if variant == 'network free' else dict(model=shared, addresses={}, links=[], faults=[], strict_topology=False)
    declared = dict(nodes=['a', 'b'], shared=config, broker_seed=7, selected=[], steps=[])
    for item in selected:
        declared['selected'].append(dict(item, network=None if variant == 'network free' else item['network'], model_seed=7))
        declared['steps'] += [dict(action='start' if item['incarnation'] == 1 else 'restart', handle=dict(node=item['node'], incarnation=item['incarnation'])), dict(action='wait', handle=dict(node=item['node'], incarnation=item['incarnation']))]
    return caller, declared


def observations(config, stdout):
    del config
    rows = [native_roles.decode(line.split(MARKER, 1)[1]) for line in stdout.split('\n') if MARKER in line]
    rows = [row for row in rows if row.get('case') == 'boot-network.selected-model-binding']
    seen, facts = set(), []
    for row in rows:
        require(set(row) == {'schema', 'case', 'inputs', 'observed'} and row['schema'] == 'gomad.core-predicate-observation/v1', 'foreign boot network envelope')
        inputs, actual = row['inputs'], row['observed']
        require(set(inputs) == {'original_root', 'variant', 'caller_before', 'caller_after', 'declared'} and inputs['original_root'] == ROOT, 'foreign boot network input declaration')
        variant = inputs['variant']
        require(variant in VARIANTS and variant not in seen, 'foreign or duplicate boot network variant')
        caller, declared = expected(variant)
        require(same(inputs['caller_before'], caller) and same(inputs['caller_after'], caller) and same(inputs['declared'], declared), 'boot network raw selected model input or caller ownership changed')
        # Limits may differ in other capacity dimensions; these two bind every selected boot.
        mismatch = any(item['network'] is not None and any(item['network']['limits'][key] != declared['shared']['model']['limits'][key] for key in ('connection_bytes', 'stall_nanos')) for item in declared['selected'])
        if mismatch:
            require(actual == dict(error='cluster boot network limits'), 'selected network mismatch was admitted, overwritten or misclassified')
        else:
            selected = sorted([dict(node=item['node'], incarnation=item['incarnation'], network=item['network']) for item in declared['selected']], key=lambda item:(item['node'],item['incarnation']))
            bound = dict(shared=declared['shared'], selected=selected, owners=['a', 'b'])
            require(set(actual) == {'error', 'bound', 'activity', 'shared_json_bytes', 'shared_json_sha256'} and actual['error'] == '' and same(actual['bound'], bound) and same(actual['activity'], dict(launched_stores=0, remote_callback_attempts=0)), 'selected network binding lost owners/models or created stores during admission')
            raw = actual['shared_json_bytes']
            require(isinstance(raw, list) and all(type(byte) is int and 0 <= byte <= 255 for byte in raw), 'network bound identity preimage missing')
            raw = bytes(raw)
            require(hashlib.sha256(raw).hexdigest() == actual['shared_json_sha256'] and same(native_roles.decode(raw), declared['shared']), 'bound shared network identity differs from actual selected model')
        seen.add(variant)
        facts.append(dict(variant=variant, selected=[dict(node=item['node'], incarnation=item['incarnation']) for item in declared['selected']], error=actual['error'], caller_unchanged=True))
    require(seen == set(VARIANTS), 'boot network harness omitted actual selected-incarnation cases')
    controls = {'matching restart', 'independent local capacities'}
    return dict(root=ROOT, received=[fact for fact in facts if fact['variant'] not in controls], additional_controls=[fact for fact in facts if fact['variant'] in controls], unreceived=[dict(original_variant='unused mismatch', reason='Native helper Request contains only selected incarnations; scenario::bind clones one network Option to all selected boots and offers no per-boot network override for unused definitions')],
                representation='Original Provider.Boots/IncarnationBoots and separate Spec limit inputs retired; native shared NetworkConfig is bound to each selected incarnation NetworkModel before linked preparation')
