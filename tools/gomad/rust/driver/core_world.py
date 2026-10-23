"""Independent frozen-corpus and canonical World binary evidence reader."""
import base64
import hashlib
import json
from pathlib import Path
import zlib

from driver import require
import native_roles

MARKER = 'GOMAD_CORE_OBSERVATION '
SCHEMA = 'gomad.core-predicate-observation/v1'
CONFIG = ('seed', 'max_requests', 'max_events', 'max_queued_events', 'max_transitions', 'max_payload_bytes', 'max_string_bytes')
WORLD_SOURCES = ('choice.go', 'codec.go', 'compressed.go', 'compressed_decode.go', 'compressed_wire_generated.go', 'errors.go', 'implementation.go', 'queue.go', 'recording.go', 'replay.go', 'snapshot.go', 'types.go', 'world.go')
CLOCK_SOURCES = ('simulation_broker.go', 'simulation_progress.go', 'simulation_time.go')


def same(left, right):
    return json.dumps(left, sort_keys=True, separators=(',', ':')) == json.dumps(right, sort_keys=True, separators=(',', ':'))


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def u(value, size=8):
    return value.to_bytes(size, 'big')


def frame(raw):
    if isinstance(raw, str):
        raw = raw.encode()
    return u(len(raw)) + raw


def unbase(value):
    require(isinstance(value, str), 'World bytes must be base64 text')
    raw = base64.b64decode(value, validate=True)
    require(base64.b64encode(raw).decode() == value, 'noncanonical World base64')
    return raw


class Reader:
    def __init__(self, raw):
        self.raw, self.at = raw, 0
        self.config = dict(max_string_bytes=1024, max_payload_bytes=384 << 20)

    def take(self, size):
        require(0 <= size <= len(self.raw) - self.at, 'truncated World binary frame')
        start = self.at
        self.at += size
        return self.raw[start:self.at]

    def number(self, size=8):
        return int.from_bytes(self.take(size), 'big')

    def count(self, maximum):
        value = self.number()
        require(value <= maximum and value <= len(self.raw) - self.at, 'World count exceeds bound')
        return value

    def text(self, tags=None):
        count = self.number(4)
        require(count <= (max(map(len, tags)) if tags else self.config['max_string_bytes']), 'World text exceeds bound')
        value = self.take(count).decode('utf8')
        require(tags is None or value in tags, 'unknown World owner tag')
        return value

    def payload(self):
        return self.take(self.count(self.config['max_payload_bytes']))

    def request(self):
        result = {key: self.text() for key in ('kind', 'adapter', 'resource_kind', 'key')}
        result.update(priority=self.number(2), payload=self.payload())
        return result

    def readiness(self):
        return dict(request_id=self.number(), at=self.number(), kind=self.text(), payload=self.payload(), equivalence_class=self.text())

    def quiescence(self):
        result = dict(kind=self.text(('delivered', 'deadlock', 'idle')), before=self.number(), after=self.number())
        result['deliveries'] = [dict(request_id=self.number(), event_id=self.number(), at=self.number(), kind=self.text(), payload=self.payload()) for _ in range(self.count(self.config['max_events']))]
        result['blocked'] = [self.number() for _ in range(self.count(self.config['max_requests']))]
        return result


def request_frame(request):
    return b''.join(frame(request[key]) for key in ('kind', 'adapter', 'resource_kind', 'key')) + u(request['priority'], 4) + frame(request['payload'])


def readiness_frame(value):
    return u(value['request_id']) + u(value['at']) + frame(value['kind']) + frame(value['payload']) + frame(value['equivalence_class'])


def quiescence_frame(value):
    result = frame(value['kind']) + u(value['before']) + u(value['after']) + u(len(value['deliveries']))
    for delivery in value['deliveries']:
        result += u(delivery['request_id']) + u(delivery['event_id']) + u(delivery['at']) + frame(delivery['kind']) + frame(delivery['payload'])
    return result + u(len(value['blocked'])) + b''.join(u(blocked) for blocked in value['blocked'])


def decode_world(raw, config=None, *, summary=False, measured_requests=None):
    """Decode bounded GMWSNAP1, authenticating every transition and full state framing."""
    require(isinstance(raw, bytes) and 0 < len(raw) <= 384 << 20, 'World expanded byte bound')
    reader = Reader(raw)
    require(reader.take(8) == b'GMWSNAP1' and reader.number(4) == 1, 'World binary header')
    observed_config = {key: reader.number(4 if key == 'max_string_bytes' else 8) for key in CONFIG}
    require(config is None or config == observed_config, 'World configuration identity differs')
    require(all(observed_config[key] > 0 for key in CONFIG[1:]) and observed_config['max_queued_events'] <= observed_config['max_events'], 'World configuration bounds')
    reader.config = observed_config
    result = dict(config=observed_config)
    for key in ('now', 'next_request_id', 'next_event_id', 'next_transition', 'payload_bytes', 'replay_position', 'replay_count'):
        result[key] = reader.number()
    require(result['replay_position'] == result['replay_count'] == 0, 'unexpected World replay progress')
    result['transcript_digest'], result['state_digest'] = reader.take(32).hex(), reader.take(32).hex()
    state = hashlib.sha256(b'gomad/world/state/v1\0' + u(1, 4) + b''.join(u(observed_config[key], 4 if key == 'max_string_bytes' else 8) for key in CONFIG))
    state.update(b''.join(u(result[key]) for key in ('now', 'next_request_id', 'next_event_id', 'next_transition', 'payload_bytes')))
    requests = []
    last_request = 0
    count = reader.count(observed_config['max_requests'])
    result['request_count'] = count
    state.update(u(count))
    for _ in range(count):
        item = dict(id=reader.number(), request=reader.request(), state=reader.text(('pending', 'queued', 'canceled', 'delivered')), event_id=reader.number())
        require(item['id'] == last_request + 1, 'World request order')
        last_request = item['id']
        if measured_requests is not None:
            require(item == dict(id=last_request, request=measured_request(), state='delivered', event_id=last_request), 'measured World terminal request differs')
        state.update(u(item['id']) + request_frame(item['request']) + frame(item['state']) + u(item['event_id']))
        if not summary:
            requests.append(item)
    events = []
    last_event = 0
    count = reader.count(observed_config['max_events'])
    result['event_count'] = count
    state.update(u(count))
    for _ in range(count):
        item = dict(id=reader.number(), readiness=reader.readiness(), state=reader.text(('queued', 'canceled', 'delivered')))
        require(item['id'] == last_event + 1, 'World event order')
        last_event = item['id']
        if measured_requests is not None:
            require(item == dict(id=last_event, readiness=measured_readiness(last_event), state='delivered'), 'measured World terminal event differs')
        state.update(u(item['id']) + readiness_frame(item['readiness']) + frame(item['state']))
        if not summary:
            events.append(item)
    transitions = []
    count = reader.count(observed_config['max_transitions'])
    result['transition_count'] = count
    state.update(u(count))
    previous = sha(b'')
    kinds = {1: 'register', 2: 'ready', 3: 'cancel', 4: 'quiesce'}
    for index in range(count):
        item = dict(sequence=reader.number(), previous_digest=reader.take(32).hex(), digest=reader.take(32).hex())
        tag = reader.number(1)
        require(tag in kinds and item['sequence'] == index + 1 and item['previous_digest'] == previous, 'World transition sequence/chain/tag')
        item['kind'] = kinds[tag]
        if tag == 1:
            item.update(request=reader.request(), request_id=reader.number())
            body = request_frame(item['request']) + u(item['request_id'])
        elif tag == 2:
            item.update(readiness=reader.readiness(), event_id=reader.number())
            body = readiness_frame(item['readiness']) + u(item['event_id'])
        elif tag == 3:
            item.update(request_id=reader.number(), cancellation=dict(request_id=reader.number(), event_id=reader.number(), status=reader.text(('won', 'already-canceled', 'already-delivered'))))
            cancel = item['cancellation']
            body = u(item['request_id']) + u(cancel['request_id']) + u(cancel['event_id']) + frame(cancel['status'])
        else:
            item['quiescence'] = reader.quiescence()
            body = quiescence_frame(item['quiescence'])
        head = u(item['sequence']) + frame(item['kind']) + frame(previous)
        require(sha(b'gomad/world/transcript/v1\0' + frame(previous) + head + body) == item['digest'], 'World transition digest differs')
        state.update(frame(head + frame(item['digest']) + body))
        previous = item['digest']
        if measured_requests is not None:
            measured_transition(item, index)
        if not summary:
            transitions.append(item)
    state.update(frame(previous))
    require(reader.at == len(raw), 'World trailing binary bytes')
    require(previous == result['transcript_digest'] and state.hexdigest() == result['state_digest'], 'World canonical state/transcript digest differs')
    require(result['next_transition'] == count + 1 and result['next_request_id'] == result['request_count'] + 1 and result['next_event_id'] == result['event_count'] + 1, 'World next identity')
    if not summary:
        result.update(requests=requests, events=events, transitions=transitions)
    result['expanded_sha256'] = sha(raw)
    return result


def expand(envelope):
    require(isinstance(envelope, bytes) and len(envelope) >= 64, 'World envelope truncated')
    reader = Reader(envelope)
    require(reader.take(8) == b'GMWORLD2' and reader.number(4) == 64 and reader.number(4) == 0, 'World compressed header')
    size, compressed, digest = reader.number(), reader.number(), reader.take(32)
    require(0 < size <= 384 << 20 and compressed == len(envelope) - 64 and compressed <= 64 << 20, 'World envelope bounds')
    decoder = zlib.decompressobj(31)
    raw = decoder.decompress(envelope[64:], size + 1)
    require(len(raw) == size and decoder.eof and not decoder.unused_data and not decoder.unconsumed_tail and bytes.fromhex(sha(raw)) == digest, 'World envelope expansion/digest/trailing bytes')
    return raw

# Exact 1d695061b6382009d0659238c652a5a765b1d5f5 document closure, including inputs and expected observations.
CORPUS_CLOSURE = {
    'world': (65, '31e5a5d66d4e943615f1d4a3753cff738667262f410011c00a20abb32043a575'),
    'clock': (56, '300275a5a1418986ac927211899826c04cff1986befa5a41b841f6922ef42b8e'),
}
CASES = {'world.corpus-replay', 'clock.corpus-replay', 'generated_clock_repeated_execution',
         'final_world_wire_owner_round_trip', 'binary_world_foreign_configuration', 'binary_world_owner_profile_round_trip'}
ROOT_CASES = {
    'TestWorldConformanceCorpusIsCurrent': ('world.corpus-replay',),
    'TestWorldConformanceDriftPinIsCurrent': ('world.corpus-replay',),
    'TestWorldConformanceFinalWireRoundTrips': ('final_world_wire_owner_round_trip',),
    'TestClockConformanceCorpusIsCurrent': ('clock.corpus-replay',),
    'TestClockConformanceDriftPinIsCurrent': ('clock.corpus-replay',),
    'TestClockConformanceGenerationIsDeterministic': ('generated_clock_repeated_execution',),
    'TestBinaryWorldEvidenceUsesSharedOwnerDecoder': ('binary_world_foreign_configuration', 'binary_world_owner_profile_round_trip'),
}


def corpus_inputs(config, kind):
    directory = Path(config['repo']) / 'tools/gomad/rust/wasmhost/tests/testdata' / (kind + '_conformance')
    paths = sorted(directory.glob('*.json'))
    closure = hashlib.sha256()
    documents = {}
    pin_path = native_roles.pinned(config, directory / ('PINNED_GO_' + kind.upper() + '_SHA256'))
    pin = pin_path.read_text()
    require(pin.endswith('\n') and pin.count('\n') == 1, 'foreign World/clock pin framing')
    for path in paths:
        native_roles.pinned(config, path)
        raw = path.read_bytes()
        closure.update(path.name.encode() + b'\0' + u(len(raw)) + raw)
        doc = native_roles.decode(raw)
        require(doc['schema'] == 'gomad.' + kind + '-conformance/v1' and doc['name'] == path.stem
                and doc[kind + '_implementation_sha256'] + '\n' == pin, 'foreign World/clock corpus header')
        documents[path.stem] = doc
    require((len(paths), closure.hexdigest()) == CORPUS_CLOSURE[kind], 'World/clock corpus differs from the complete immutable original closure')
    require({doc['source'] for doc in documents.values()} == {'scripted', 'generated', 'recorded'}, 'World/clock source categories differ')
    names = WORLD_SOURCES if kind == 'world' else CLOCK_SOURCES
    prefix = b'gomad-world-implementation/v1\n' if kind == 'world' else b'gomad-simulation-clock-implementation/v1\n'
    implementation = prefix
    sources = {}
    for name in names:
        path = native_roles.pinned(config, Path(config['repo']) / 'tools/gomad/internal/model' / kind / name)
        sources[name] = sha(path.read_bytes())
        implementation += (name + ':' + sources[name] + '\n').encode()
    require(pin == 'sha256:' + sha(implementation) + '\n', 'World/clock drift pin differs from recomputed original implementation source')
    return documents, dict(corpus_sha256=closure.hexdigest(), implementation_sha256=pin.strip(), source_inputs=sources)


def fixed_owner(raw):
    doc = decode_world(raw)
    require(doc['config'] == dict(zip(CONFIG, (7, 4, 4, 1, 100000, 32 << 20, 1024))), 'shared World owner configuration differs')
    require(len(doc['requests']) == len(doc['events']) == 3 and len(doc['transitions']) == 9, 'shared World owner cardinality differs')
    for index in range(3):
        identity = index + 1
        request = dict(kind='runtime_net_stream', adapter='wasm-broker', resource_kind='incarnation', key='active/1', priority=0, payload=bytes(32))
        readiness = dict(request_id=identity, at=946684800000000000 + identity, kind='runtime_net_stream', payload=b'', equivalence_class='')
        registered, ready, quiesced = doc['transitions'][index * 3:index * 3 + 3]
        require(registered['kind'] == 'register' and registered['request'] == request and registered['request_id'] == identity
                and ready['kind'] == 'ready' and ready['readiness'] == readiness and ready['event_id'] == identity, 'shared owner register/ready workload differs')
        result = quiesced['quiescence']
        require(quiesced['kind'] == 'quiesce' and result == dict(kind='delivered', before=readiness['at'] - 1, after=readiness['at'],
                deliveries=[dict(request_id=identity, event_id=identity, at=readiness['at'], kind='runtime_net_stream', payload=b'')], blocked=[]), 'shared owner actual quiescence differs')
        require(doc['requests'][index] == dict(id=identity, request=request, state='delivered', event_id=identity)
                and doc['events'][index] == dict(id=identity, readiness=readiness, state='delivered'), 'shared owner terminal request/event differs')
    require([doc[key] for key in ('now', 'next_request_id', 'next_event_id', 'next_transition', 'payload_bytes')] == [946684800000000003, 4, 4, 10, 192], 'shared owner terminal counters differ')
    return doc


def observations(config, stdout):
    documents = {}
    sources = {}
    for kind in ('world', 'clock'):
        documents[kind], sources[kind] = corpus_inputs(config, kind)
    rows = [native_roles.decode(line[line.index(MARKER) + len(MARKER):]) for line in stdout.split('\n') if MARKER in line]
    rows = [row for row in rows if row.get('case') in CASES]
    expected = {}
    for kind in ('world', 'clock'):
        for name, doc in documents[kind].items():
            expected[(kind + '.corpus-replay', name)] = 1 + int(kind == 'world' and doc['final'] is not None)
            if kind == 'clock' and doc['source'] == 'generated':
                expected[('generated_clock_repeated_execution', name)] = 1
            if kind == 'world' and doc['final'] is not None:
                expected[('final_world_wire_owner_round_trip', name)] = 1
    expected.update({('binary_world_foreign_configuration', 8): 1, ('binary_world_foreign_configuration', 7): 1,
                     ('binary_world_owner_profile_round_trip', None): 1})
    seen = {}
    facts = {case: [] for case in CASES}
    owner_rows = []
    for row in rows:
        require(set(row) == {'schema', 'case', 'inputs', 'observed'} and row['schema'] == SCHEMA, 'foreign World observation envelope')
        case, inputs, actual = row['case'], row['inputs'], row['observed']
        name = inputs.get('corpus')
        key = (case, inputs.get('seed') if case == 'binary_world_foreign_configuration' else name)
        require(key in expected, 'unknown World/clock observation case')
        seen[key] = seen.get(key, 0) + 1
        require(seen[key] <= expected[key], 'duplicate World/clock observation')
        if case.endswith('.corpus-replay'):
            kind = case.split('.')[0]
            doc = documents[kind][name]
            require(same(inputs, dict(corpus=name, document=doc)), 'World/clock replay substituted frozen corpus inputs')
            want = dict(steps=[step['expected'] for step in doc['steps']])
            if kind == 'world':
                want['creation_error'] = doc['creation_error']
            require(same(actual, want), 'World/clock actual API response/state differs from exact original assertion')
            fact = dict(corpus=name, steps=len(doc['steps']), source=doc['source'], creation_error=doc.get('creation_error'))
        elif case == 'generated_clock_repeated_execution':
            doc = documents['clock'][name]
            require(same(inputs, dict(corpus=name, clock=doc['clock'], steps=doc['steps'])), 'generated clock input differs')
            want = [step['expected'] for step in doc['steps']]
            require(want and same(actual, dict(first=want, second=want)), 'generated clock raw repeated responses differ')
            fact = dict(corpus=name, repeated_steps=len(want))
        elif case == 'final_world_wire_owner_round_trip':
            doc = documents['world'][name]
            raw = unbase(actual['expanded_base64'])
            envelope, restored = unbase(actual['envelope_base64']), unbase(actual['restored_base64'])
            require(inputs == dict(corpus=name, expanded_sha256=sha(raw)), 'final World wire input digest differs')
            require(raw == unbase(doc['final']['expanded_base64']) == restored == expand(envelope), 'final World raw wire/expansion differs from original')
            decoded = decode_world(raw, doc['config'])
            state = doc['steps'][-1]['expected']['state']
            require(sha(raw) == doc['final']['expanded_sha256'] and sha(envelope) == doc['final']['envelope_sha256'], 'final World original envelope/raw identity differs')
            require(actual == dict(expanded_base64=base64.b64encode(raw).decode(), envelope_base64=base64.b64encode(envelope).decode(), restored_base64=base64.b64encode(restored).decode(),
                    envelope_sha256=sha(envelope), restored_sha256=sha(restored), state_digest=decoded['state_digest'],
                    next_ids=[decoded['next_request_id'], decoded['next_event_id'], decoded['next_transition']]), 'final World observed owner outputs differ')
            require(decoded['state_digest'] == state['state_digest'] and all(decoded[key] == state[key] for key in ('now', 'payload_bytes', 'next_request_id', 'next_event_id', 'next_transition', 'transcript_digest')), 'final World state differs from original owner')
            fact = dict(corpus=name, expanded_sha256=sha(raw), envelope_sha256=sha(envelope), state_digest=decoded['state_digest'], transitions=len(decoded['transitions']))
        else:
            owner_rows.append(row)
            fact = dict(case=case, seed=inputs.get('seed'))
        facts[case].append(fact)
    require(seen == expected, 'World/clock harness omitted an original raw assertion case')
    owner = next(row for row in owner_rows if row['case'] == 'binary_world_owner_profile_round_trip')
    actual = owner['observed']
    raw, envelope, restored = (unbase(actual[key]) for key in ('expanded_base64', 'envelope_base64', 'restored_base64'))
    decoded = fixed_owner(raw)
    require(raw == restored == expand(envelope), 'shared owner compressed expansion differs')
    require(owner['inputs'] == dict(seed=7, requests=3, wire_sha256=sha(raw)), 'shared owner fixed input differs')
    require(actual == dict(expanded_base64=base64.b64encode(raw).decode(), envelope_base64=base64.b64encode(envelope).decode(), restored_base64=base64.b64encode(restored).decode(),
            envelope_sha256=sha(envelope), restored_sha256=sha(restored), trailing_error='World wire trailing bytes',
            wrong_raw_profile_error='World wire header', wrong_compressed_profile_error='World compressed envelope header'), 'shared owner raw profile refusal differs')
    for row in owner_rows:
        if row['case'] == 'binary_world_foreign_configuration':
            seed = row['inputs']['seed']
            require(row['inputs'] == dict(wire_sha256=sha(raw), seed=seed, max_requests=4 if seed == 8 else 3)
                    and row['observed'] == dict(error='World wire configuration mismatch'), 'foreign owner configuration refusal differs')
    return dict(roots={root: dict(cases=[fact for case in cases for fact in facts[case]], source_inputs=sources['clock' if 'Clock' in root else 'world'])
                            for root, cases in ROOT_CASES.items()}, source_inputs=sources)


def world_document(decoded):
    """Project authenticated wire fields into the original Go Snapshot JSON value."""
    def payload(result, raw):
        if raw:
            result['payload'] = base64.b64encode(raw).decode()
        return result

    def request(value):
        return payload(dict(kind=value['kind'], resource=dict(adapter=value['adapter'], kind=value['resource_kind'], key=value['key']), priority=value['priority']), value['payload'])

    def readiness(value):
        result = payload(dict(requestId=str(value['request_id']), at=str(value['at']), kind=value['kind']), value['payload'])
        if value['equivalence_class']:
            result['equivalenceClass'] = value['equivalence_class']
        return result

    def cancellation(value):
        result = dict(requestId=str(value['request_id']), status=value['status'])
        if value['event_id']:
            result['eventId'] = str(value['event_id'])
        return result

    def quiescence(value):
        result = dict(kind=value['kind'], before=str(value['before']), after=str(value['after']))
        if value['deliveries']:
            result['deliveries'] = [payload(dict(requestId=str(item['request_id']), eventId=str(item['event_id']), at=str(item['at']), kind=item['kind']), item['payload']) for item in value['deliveries']]
        if value['blocked']:
            result['blocked'] = list(map(str, value['blocked']))
        return result

    config = decoded['config']
    limits = dict(zip(('maxRequests', 'maxEvents', 'maxQueuedEvents', 'maxTransitions', 'maxPayloadBytes', 'maxStringBytes'), (config[key] for key in CONFIG[1:])))
    result = dict(schemaVersion=1, config=dict(seed=str(config['seed']), limits=limits), now=str(decoded['now']),
                  nextRequestId=str(decoded['next_request_id']), nextEventId=str(decoded['next_event_id']), nextTransition=str(decoded['next_transition']),
                  payloadBytes=decoded['payload_bytes'], requests=[], events=[], transitions=[], transcriptDigest=decoded['transcript_digest'],
                  replay=dict(cursor=0, expected=0), stateDigest=decoded['state_digest'])
    for value in decoded['requests']:
        item = dict(id=str(value['id']), request=request(value['request']), state=value['state'])
        if value['event_id']:
            item['eventId'] = str(value['event_id'])
        result['requests'].append(item)
    for value in decoded['events']:
        result['events'].append(dict(id=str(value['id']), readiness=readiness(value['readiness']), state=value['state']))
    for value in decoded['transitions']:
        item = dict(sequence=str(value['sequence']), kind=value['kind'], previousDigest=value['previous_digest'], digest=value['digest'])
        if value['kind'] == 'register':
            item['register'] = dict(request=request(value['request']), requestId=str(value['request_id']))
        elif value['kind'] == 'ready':
            item['ready'] = dict(readiness=readiness(value['readiness']), eventId=str(value['event_id']))
        elif value['kind'] == 'cancel':
            item['cancel'] = dict(requestId=str(value['request_id']), cancellation=cancellation(value['cancellation']))
        else:
            item['quiesce'] = dict(result=quiescence(value['quiescence']))
        result['transitions'].append(item)
    return result


def measured_request():
    return dict(kind='runtime_net_stream', adapter='wasm-broker', resource_kind='incarnation', key='active/1', priority=0, payload=bytes(32))


def measured_readiness(identity):
    return dict(request_id=identity, at=946684800000000000 + identity, kind='runtime_net_stream', payload=b'', equivalence_class='')


def measured_transition(value, index):
    identity, operation = index // 3 + 1, index % 3
    body = {key: field for key, field in value.items() if key not in ('sequence', 'previous_digest', 'digest')}
    if operation == 0:
        expected = dict(kind='register', request=measured_request(), request_id=identity)
    elif operation == 1:
        expected = dict(kind='ready', readiness=measured_readiness(identity), event_id=identity)
    else:
        at = 946684800000000000 + identity
        expected = dict(kind='quiesce', quiescence=dict(kind='delivered', before=at - 1, after=at,
                        deliveries=[dict(request_id=identity, event_id=identity, at=at, kind='runtime_net_stream', payload=b'')], blocked=[]))
    require(body == expected, 'measured World actual transition differs from fixed original workload')


def decode_measured_world(raw, requests):
    """Bounded-memory authentication of the fixed measured register/ready/quiesce workload."""
    require(type(requests) is int and requests in (20000, 200000), 'foreign World measured cardinality')
    configuration = dict(zip(CONFIG, (7, requests + 1, requests + 1, 1, max(requests * 3 + 1, 100000), 32 << 20, 1024)))
    result = decode_world(raw, configuration, summary=True, measured_requests=requests)
    require(result['request_count'] == result['event_count'] == requests and result['transition_count'] == requests * 3,
            'World measured cardinalities differ from raw wire')
    require([result[key] for key in ('now', 'next_request_id', 'next_event_id', 'next_transition', 'payload_bytes')]
            == [946684800000000000 + requests, requests + 1, requests + 1, requests * 3 + 1, requests * 64], 'World measured terminal counters differ')
    return result
