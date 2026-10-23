"""Independent GMNETEV1 framing, Go log reconstruction and network record admission."""
import base64
import copy
import hashlib
import json
from pathlib import Path

from driver import require
import native_roles

MARKER = 'GOMAD_CORE_OBSERVATION '
PROFILE = 'gomad.simulation-network-stream/v3'
LIMITS = dict(listeners=128, connections=128, bytes=8 << 20, connection_bytes=4 << 20,
              stall_nanos=10_000_000_000, transitions=8192)
ROOTS = ('TestBrokerOrdersAreCurrent', 'TestBrokerOrdersRustNetworkEvidenceRebuildsGoDocuments',
         'TestClusterNetworkDecoderRefusesMalformedEvidence', 'TestClusterNetworkRustEvidenceRebuildsTheGoBrokerDocument')

ORIGINAL_COMMIT = '1d695061b6382009d0659238c652a5a765b1d5f5'
ORIGINAL_WASI_PIN = 'sha256:5fddc30fd325f3168d32a0084a3e76e3a10b341da7aee86ddcd9f3b2a8b91bc3'
RELOCATED_WASI_PIN = 'sha256:bd45d4831ca0f417b6411a29b70cf1070a779e5f04591a679c180d925b586d15'
ORIGINAL_CORPUS = {
    'network_cluster_cases': (51, '7cb0497950335edffa3bdaf48de63694bab3174d2ef0302b1f6db96f1c414beb'),
    'broker_orders': (19, '8802ba410039c540f093dce503aa9d63b252bcac038b9ba21ff46595c7ebdada'),
    'cluster_network': (2, 'ffe1f22d278c0f2f8ffa061a8c7d32799fc6a01c085a87d430c4a8b06afeab79'),
}
ORIGINAL_NETWORK_SOURCES = {
    'adapter.go': 'c9ae9add73d213daa877e39d4edbec6959793ac4fc3a002978edadd9c5c67969',
    'identity.go': 'd1c2f50a5fcb6c5070163edd82d373c05bbe194643ed7a420ce1230ee78acd73',
    'simulation.go': '7f92e71b97013fb1b2d1554cab404ad4cfc79d7d7e80f335ef039dadc131d581',
    'stream_status_generated.go': '072fbcdf6533f2ec4e93ab8ff28bf28773449dee6391f7bc6ab6d55a6d225c0e',
    'record_identity.go': 'd95e0923d6013c570ef3e1f7c2b0ab2fe9a7cdd0a2c5d0b67656ea05ad06927d',
}
ORIGINAL_WASI_SOURCE_ANCHORS = {
    'wasi/broker_test.go': '1714a2dd24158696a02b7d882f8f8b7318ac09421e006661722a145422e2728a',
    'wasi/network_cluster_corpus_test.go': '8f1c3a965f8ec6c0b5d6d0a672a941fc16759056385d89af5308c15199193c34',
    'wasi/cluster_network_test.go': 'ab7c1a0403bcd13b6aaf1cb3fd1f489534f688b9a36352f190ec9a8809d534e9',
    'wasi/cluster_network.go': '91b39b0b1b8f6a6b9aa4e23ffaae3812d5b5785a4e2e9f015fcd50b76fa17fed',
    'wasi/broker_network.go': 'ec9a33c783284ca83abe32f41393d38f7f53ef000b23e6fe53eb811905ce1069',
    'wasi/network_v2.go': 'ca5e71708b67c85b80e9235b0ccb4b3304e3e093bca4bf35e877f22f924b3244',
    'wasi/identity.go': 'cc23a2c2fc2e14d3623351d776c89493e8930d59ed78ee3d9a0632608a6b7436',
}


def canonical(value):
    text = json.dumps(value, ensure_ascii=False, separators=(',', ':'))
    for character in '<>&\u2028\u2029':
        text = text.replace(character, '\\u%04x' % ord(character))
    return text.encode()


def same(left, right):
    return json.dumps(left, sort_keys=True, separators=(',', ':')) == json.dumps(right, sort_keys=True, separators=(',', ':'))


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


class Reader:
    def __init__(self, raw):
        self.raw, self.at = raw, 0

    def take(self, count):
        require(count >= 0 and self.at + count <= len(self.raw), 'network wire truncated')
        result = self.raw[self.at:self.at + count]
        self.at += count
        return result

    def number(self):
        value = 0
        for shift in range(0, 70, 7):
            byte = self.take(1)[0]
            require(not (shift == 63 and byte > 1), 'network wire varint overflow')
            value |= (byte & 127) << shift
            if byte < 128:
                require(shift == 0 or byte != 0, 'network wire non-minimal varint')
                return value
        raise ValueError('network wire varint overflow')

    def count(self):
        value = self.number()
        require(value <= len(self.raw) - self.at, 'network count exceeds remaining bytes')
        return value

    def blob(self):
        return self.take(self.number())

    def text(self):
        return self.blob().decode('utf-8')

    def flag(self):
        value = self.take(1)[0]
        require(value in (0, 1), 'network wire flag')
        return bool(value)

    def optional(self, read):
        return [read() for _ in range(self.count())] if self.flag() else None

    def endpoint(self):
        return {key: value for key, value in zip(('node', 'incarnation', 'address', 'port'),
                (self.text(), self.number(), self.text(), self.number())) if value}

    def transition(self):
        value = dict(ordinal=self.number(), kind=self.text(), source=self.endpoint(), destination=self.endpoint())
        for key in ('connection', 'bytes', 'count', 'delay_nanos'):
            number = self.number()
            if number:
                value[key] = number
        value['outcome'] = self.text()
        return value

    def record(self):
        transitions = self.optional(self.transition)
        def node():
            return dict(node=self.text(), address=self.text(), last_incarnation=self.number(),
                        next_listener_port=self.number(), next_client_port=self.number())
        def link():
            return dict(zip(('from', 'to', 'enabled', 'delay_nanos'),
                            (self.text(), self.text(), self.flag(), self.number())))
        def listener():
            return dict(endpoint=self.endpoint(), closed=self.flag())
        def connection():
            return dict(client_bytes=self.number(), server_bytes=self.number(), identity=self.number(),
                        client=self.endpoint(), server=self.endpoint(), closed=self.flag(), reset=self.flag())
        snapshot = dict(nodes=self.optional(node), links=self.optional(link), listeners=self.optional(listener),
                        connections=self.optional(connection), next_connection=self.number(),
                        transition_sha256=self.text(), identity=self.text())
        return dict(transitions=transitions, snapshot=snapshot)

    def operation(self):
        at = self.number()
        request = dict(domain=dict(run=self.number(), node=self.text(), incarnation=self.number(), address=self.text()), op=self.text())
        for key, read in [('handle', self.number), ('host', self.text), ('port', self.number),
                          ('bytes', self.blob), ('count', self.number), ('deadline', self.number)]:
            value = read()
            if value:
                request[key] = base64.b64encode(value).decode() if key == 'bytes' else value
        require(request.get('port', 0) <= (1 << 63) - 1 and request.get('count', 0) <= (1 << 32) - 1, 'network request range')
        request['request_id'] = self.number()
        result = dict(request_id=self.number(), state=self.text())
        handle = self.number()
        if handle:
            result['handle'] = handle
        for key in ('local', 'remote'):
            result[key] = dict(IP=self.text(), Port=self.number())
            require(result[key]['Port'] <= (1 << 63) - 1, 'network address range')
        for key, read in [('bytes', self.blob), ('count', self.number), ('deadline', self.number), ('error', self.text)]:
            value = read()
            if value:
                result[key] = base64.b64encode(value).decode() if key == 'bytes' else value
        require(result.get('count', 0) <= (1 << 63) - 1, 'network result range')
        return dict(at=at, request=request, result=result)

    def quiescence(self):
        signed = lambda n: n if n < (1 << 63) else n - (1 << 64)
        value = dict(kind=self.text(), before=str(signed(self.number())), after=str(signed(self.number())))
        require(value['kind'] in ('delivered', 'deadlock', 'idle'), 'network wire quiescence kind')
        deliveries = []
        for _ in range(self.count()):
            delivery = dict(requestId=str(self.number()), eventId=str(self.number()), at=str(signed(self.number())), kind=self.text())
            payload = self.blob()
            if payload:
                delivery['payload'] = base64.b64encode(payload).decode()
            deliveries.append(delivery)
        blocked = [str(self.number()) for _ in range(self.count())]
        if deliveries:
            value['deliveries'] = deliveries
        if blocked:
            value['blocked'] = blocked
        return value


def u64(value):
    return value.to_bytes(8, 'little')


def string(value):
    raw = value.encode()
    require(len(raw) <= 1 << 20, 'network hash string bound')
    return u64(len(raw)) + raw


def endpoint(value):
    return string(value.get('node', '')) + u64(value.get('incarnation', 0)) + string(value.get('address', '')) + u64(value.get('port', 0))


def record_hashes(record):
    transitions, snapshot = record['transitions'] or [], record['snapshot']
    raw = u64(len(transitions))
    for t in transitions:
        raw += u64(t['ordinal']) + string(t['kind']) + endpoint(t['source']) + endpoint(t['destination'])
        raw += b''.join(u64(t.get(k, 0)) for k in ('connection', 'bytes', 'count', 'delay_nanos')) + string(t['outcome'])
    transition_hash = 'sha256:' + sha(b'gomad-simulation-network-transitions/v3\0' + raw)
    raw = b''
    for key in ('nodes', 'links', 'listeners', 'connections'):
        values = snapshot[key] or []
        raw += u64(len(values))
        for v in values:
            if key == 'nodes':
                raw += string(v['node']) + string(v['address']) + b''.join(u64(v[k]) for k in ('last_incarnation', 'next_listener_port', 'next_client_port'))
            elif key == 'links':
                raw += string(v['from']) + string(v['to']) + u64(v['enabled']) + u64(v['delay_nanos'])
            elif key == 'listeners':
                raw += endpoint(v['endpoint']) + u64(v['closed'])
            else:
                raw += u64(v['identity']) + endpoint(v['client']) + endpoint(v['server'])
                raw += b''.join(u64(v[k]) for k in ('closed', 'reset', 'client_bytes', 'server_bytes'))
    raw += u64(snapshot['next_connection']) + string(snapshot['transition_sha256']) + string('')
    return transition_hash, 'sha256:' + sha(b'gomad-simulation-network-snapshot/v3\0' + raw)


def endpoint_key(e):
    return '%s\0%020d\0%s\0%05d' % (e.get('node', ''), e.get('incarnation', 0), e.get('address', ''), e.get('port', 0))


def lane(t):
    kind = t['kind']
    if kind in ('partition', 'heal', 'delay', 'disconnect', 'reconnect', 'directional_delay'):
        return 'topology'
    if kind in ('listen', 'listener_close'):
        return 'listener\0' + endpoint_key(t['source'])
    if kind in ('dial', 'accept', 'write', 'close', 'reset'):
        return 'connection\0%020d' % t['connection'] if t.get('connection') else 'operation\0' + kind + '\0' + endpoint_key(t['source']) + '\0' + endpoint_key(t['destination'])
    return 'lifecycle\0' + endpoint_key(t['source'])


def valid_endpoint(e, incarnation=True):
    return bool(e.get('node') and e.get('address') and (not incarnation or e.get('incarnation')))


def valid_transition(t):
    kind, outcome, source, dest = t['kind'], t['outcome'], t['source'], t['destination']
    if kind in ('listen', 'dial', 'accept', 'write', 'close', 'listener_close', 'stop', 'crash', 'reset'):
        if not valid_endpoint(source):
            return False
    elif kind in ('partition', 'heal', 'delay', 'disconnect', 'reconnect', 'directional_delay'):
        if not valid_endpoint(source, False) or not valid_endpoint(dest, False) or source['node'] == dest['node']:
            return False
    else:
        return False
    if outcome not in ('ok', 'closed', 'refused', 'deadline', 'reset', 'capacity', 'unsupported', 'held', 'timeout'):
        return False
    if kind == 'reset' or outcome == 'held' or outcome == 'timeout' and kind == 'write':
        if kind not in ('reset', 'write') or not t.get('connection') or not valid_endpoint(dest) or source['node'] == dest['node'] or not 1 <= source.get('port', 0) <= 65535 or not 1 <= dest.get('port', 0) <= 65535:
            return False
        if kind == 'reset' and outcome not in ('reset', 'timeout') or outcome == 'held' and not t.get('bytes'):
            return False
    if outcome == 'timeout' and kind not in ('write', 'reset'):
        return kind == 'dial' and not t.get('connection') and source.get('port', 0) <= 65535 and valid_endpoint(dest, False) and source['node'] != dest['node'] and 1 <= dest.get('port', 0) <= 65535
    return True


def validate_record(record, limits=LIMITS):
    ts, s = record['transitions'] or [], record['snapshot']
    require(record_hashes(record) == (s['transition_sha256'], s['identity']), 'network record identity')
    require(len(ts) <= limits['transitions'] and len(s['listeners'] or []) <= limits['listeners'] and len(s['connections'] or []) <= limits['connections'], 'network record bound')
    require(all(t['ordinal'] == i and valid_transition(t) and (i == 0 or lane(ts[i-1]) <= lane(t)) for i, t in enumerate(ts)), 'network transition semantic admission')
    previous = ''
    for n in s['nodes'] or []:
        require(n['node'] > previous and n['address'] and n['next_listener_port'] and n['next_client_port'], 'network node semantic admission')
        previous = n['node']
    previous = None
    for link in s['links'] or []:
        key = (link['from'], link['to'])
        require(all(key) and key[0] != key[1] and (previous is None or previous < key), 'network link semantic admission')
        previous = key
    previous = None
    for listener in s['listeners'] or []:
        e = listener['endpoint']; key = endpoint_key(e)
        require(valid_endpoint(e) and 1 <= e.get('port', 0) <= 65535 and (previous is None or previous[0] < key or previous[0] == key and previous[1]), 'network listener semantic admission')
        previous = key, listener['closed']
    previous, retained = 0, 0
    for c in s['connections'] or []:
        require(c['identity'] > previous and all(valid_endpoint(c[k]) and 1 <= c[k].get('port', 0) <= 65535 for k in ('client', 'server')), 'network connection semantic admission')
        require(not c['reset'] or c['client_bytes'] == c['server_bytes'] == 0, 'network reset retains bytes')
        for key in ('client_bytes', 'server_bytes'):
            retained += c[key]
            require(c[key] <= limits['connection_bytes'] and retained <= limits['bytes'], 'network byte capacity')
        previous = c['identity']


def decode_network(raw, limit=1 << 20, limits=LIMITS, world=True):
    require(len(raw) <= limit, 'network evidence byte bound')
    r = Reader(raw)
    require(r.take(8) == b'GMNETEV1' and r.number() == 1 and r.text() == PROFILE, 'network evidence header')
    declared, count = r.number(), r.count()
    entries = []
    for _ in range(count):
        tag = r.take(1)[0]
        require(tag in (1, 3), 'network evidence operation tag')
        entries.append(r.operation() if tag == 1 else r.quiescence())
    log = b''.join(canonical(e) + b'\n' for e in entries)
    require(len(log) == declared and len(log) <= limit, 'network evidence log accounting')
    record = r.record()
    require(r.flag(), 'network evidence missing World')
    wire = r.blob()
    require(r.at == len(raw), 'network evidence trailing bytes')
    validate_record(record, limits)
    if world:
        import core_world
        core_world.decode_world(wire)
    return dict(record=record, log=log, world=wire, entries=entries)


def refused(raw, limit=1 << 20, limits=LIMITS):
    try:
        decode_network(raw, limit, limits)
    except (ValueError, UnicodeError, OverflowError):
        return True
    return False


def corpus_inputs(config):
    root = Path(config['repo']) / 'tools/gomad'
    crate = root / 'rust/wasmhost'
    documents, closures = {}, {}
    for directory, expected in ORIGINAL_CORPUS.items():
        base = crate / 'tests/testdata' / directory
        paths = sorted(base.iterdir())
        require(all(path.is_file() and not path.is_symlink() for path in paths), 'network original inventory contains aliases or directories')
        digest = hashlib.sha256()
        documents[directory] = {}
        for path in paths:
            native_roles.pinned(config, path)
            raw = path.read_bytes()
            if directory == 'network_cluster_cases':
                doc = native_roles.decode(raw)
                require(path.suffix == '.json' and doc['schema'] == 'gomad.network-cluster-corpus/v1' and doc['name'] == path.stem
                        and doc['implementation_sha256'] == RELOCATED_WASI_PIN,
                        'network corpus original header or fixed relocated implementation pin changed')
                old = ('"implementation_sha256":"' + RELOCATED_WASI_PIN + '"').encode()
                replacement = ('"implementation_sha256":"' + ORIGINAL_WASI_PIN + '"').encode()
                require(raw.count(old) == 1, 'network corpus original pin substitution is ambiguous')
                raw = raw.replace(old, replacement)
            digest.update(path.name.encode() + b'\0' + len(raw).to_bytes(8, 'big') + raw)
            if path.suffix == '.json':
                documents[directory][path.stem] = native_roles.decode(raw)
        require((len(paths), digest.hexdigest()) == expected, 'network corpus differs from immutable original inventory or content: ' + directory)
        closures[directory] = dict(files=len(paths), sha256=digest.hexdigest())
    sources = {}
    for name, expected in ORIGINAL_NETWORK_SOURCES.items():
        path = native_roles.pinned(config, root / 'internal/model/network' / name)
        sources[name] = sha(path.read_bytes())
        require(sources[name] == expected, 'network retained Go source differs from immutable original: ' + name)
    inventory = ''.join(name + ':' + sources[name] + '\n' for name in sorted(sources) if name != 'record_identity.go').encode()
    network_pin = 'sha256:' + sha(inventory)
    path = native_roles.pinned(config, crate / 'src/network/PINNED_GO_NETWORK_SHA256')
    require(path.read_text() == network_pin + '\n', 'network drift pin differs from recomputed immutable Go source')
    path = native_roles.pinned(config, crate / 'src/cluster/network/PINNED_GO_WASI_SHA256')
    require(path.read_text() == RELOCATED_WASI_PIN + '\n', 'network relocated WASI drift pin changed')
    require(all(doc['network_implementation_sha256'] == network_pin for doc in documents['network_cluster_cases'].values()), 'network corpus Go implementation drift')
    return documents, dict(original_commit=ORIGINAL_COMMIT, corpora=closures, network_source_inputs=sources,
                           network_implementation_sha256=network_pin, original_wasi_source_anchors=ORIGINAL_WASI_SOURCE_ANCHORS,
                           original_wasi_pin=ORIGINAL_WASI_PIN, relocated_wasi_pin=RELOCATED_WASI_PIN,
                           retired_drift_check='Removed Go WASI API implementation is anchored to original source hashes; no claim of regenerating it')


def observations(config, stdout):
    root = Path(config['repo']) / 'tools/gomad/rust/wasmhost'
    def read(name):
        path = root / 'tests/testdata' / name
        native_roles.pinned(config, path.resolve())
        return path.read_bytes()
    documents, original_inputs = corpus_inputs(config)
    corpora, brokers = documents['network_cluster_cases'], documents['broker_orders']
    required = set()
    for name, corpus in corpora.items():
        required.add(('network.corpus.creation' if corpus['creation_error'] else 'network.corpus.initial', name, None))
        required.update(('network.corpus.step', name, index) for index, _ in enumerate(corpus.get('steps') or []))
    required.update(('network.broker.replay', name, None) for name in brokers)
    required.update((case, None, None) for case in ('cluster_network_decoder_valid_fixture', 'cluster_network_decoder_all_prefixes', 'network_shared_record_local_world_limits'))
    required.update(('cluster_network_typed_owner_round_trip', node, None) for node in ('a', 'b'))
    mutations = ['trailing', 'oversize', 'magic', 'version', 'invalid_utf8', 'log_accounting', 'flag', 'world_wire', 'record_identity', 'non_minimal_varint']
    required.update(('cluster_network_decoder_refusal', m, None) for m in mutations)
    required.update(('cluster_network_decoder_refusal', 'unknown_tag', n) for n in (0, 2, 4, 255))
    semantic = ('unknown_transition_kind', 'ordinal_gap', 'listener_port_zero', 'reset_retains_bytes', 'connection_byte_bound', 'total_byte_bound')
    required.update(('network_resealed_semantic_refusal', m, None) for m in semantic)
    cases = {key[0] for key in required}
    rows = [native_roles.decode(line.split(MARKER, 1)[1]) for line in stdout.splitlines() if MARKER in line]
    rows = [row for row in rows if row.get('case') in cases or row.get('case', '').startswith('network.')]
    seen, facts = set(), []
    fixture = read('cluster_network/b.gmnetev1')
    for row in rows:
        require(set(row) == {'schema', 'case', 'inputs', 'observed'} and row['schema'] == 'gomad.core-predicate-observation/v1', 'foreign network observation')
        case, inputs, observed = row['case'], row['inputs'], row['observed']
        key = (case, inputs.get('name'), inputs.get('index'))
        raw = base64.b64decode(inputs['evidence_base64'], validate=True) if 'evidence_base64' in inputs else None
        if case.startswith('network.corpus.'):
            corpus = corpora[inputs['name']]
            expected = dict(error=corpus['creation_error']) if case.endswith('creation') else corpus['initial'] if case.endswith('initial') else corpus['steps'][inputs['index']]['expected']
            require(same(observed, expected), 'network corpus literal output/error/state differs')
            if 'record' in observed:
                validate_record(observed['record'], corpus['config']['model']['limits'])
        elif case == 'network.broker.replay':
            scenario = brokers[inputs['name']]
            require(same(observed['outputs'], scenario['outputs']) and observed['world_base64'] == scenario['world_base64'], 'broker ordered outputs or final World differ')
            import core_world
            core_world.decode_world(base64.b64decode(observed['world_base64'], validate=True))
            nodes = scenario['nodes'] if scenario.get('network') else []
            require(set(observed['evidence']) == set(nodes), 'broker node evidence omitted')
            for node, encoded in observed['evidence'].items():
                wire = base64.b64decode(encoded, validate=True)
                require(wire == read('broker_orders/' + inputs['name'] + '.' + node + '.gmnetev1'), 'broker node network evidence differs')
                decoded = decode_network(wire)
                document = dict(profile=PROFILE, operations=decoded["entries"], native=decoded["record"], world=core_world.world_document(core_world.decode_world(decoded["world"])))
                require(canonical(document) == base64.b64decode(scenario["network_documents_base64"][node], validate=True), "network reconstructed Go broker document differs byte-for-byte")
        elif case == 'cluster_network_typed_owner_round_trip':
            key = (case, inputs['node'], None)
            require(inputs['corpus'] == 'cross-node-ready-poll-and-stale-cancel-binary-fixture' and raw == read('cluster_network/' + inputs['node'] + '.gmnetev1'), 'network typed roundtrip changed fixture')
            value = decode_network(raw)
            require(observed == dict(record=value['record'], log_sha256=sha(value['log']), world_sha256=sha(value['world'])), 'network roundtrip derived bytes differ')
        elif case == 'cluster_network_decoder_valid_fixture':
            key = (case, None, None)
            require(raw == fixture and inputs['sha256'] == sha(raw), 'network valid fixture substituted')
            value = decode_network(raw)
            require(observed == dict(record=value['record'], log_sha256=sha(value['log']), world_sha256=sha(value['world'])), 'network decoded raw preimage differs')
        elif case == 'cluster_network_decoder_all_prefixes':
            key = (case, None, None)
            require(raw == fixture and inputs['prefix_end_exclusive'] == len(raw), 'network prefix domain changed')
            errors = observed['errors']
            require(len(errors) == len(raw) and all(isinstance(e, str) and e for e in errors), 'network raw prefix errors omitted')
            digest = hashlib.sha256()
            for length, error in enumerate(errors):
                require(refused(raw[:length]), 'network reader admitted truncated prefix')
                digest.update(length.to_bytes(8, 'big') + error.encode() + b'\0')
            require(digest.hexdigest() == observed['ordered_error_sha256'], 'network ordered errors commitment differs')
        elif case == 'network_shared_record_local_world_limits':
            key = (case, None, None)
            require(raw == fixture and inputs['local']['limits'] == LIMITS, 'network limits valid base changed')
            shared = dict(LIMITS)
            for name in ('transitions', 'bytes', 'listeners', 'connections'):
                shared[name] *= 2
            require(inputs['shared']['limits'] == shared and inputs['small_shared'] == dict(inputs['shared'], limits=dict(shared, listeners=0)) and inputs['foreign_local'] == dict(inputs['local'], limits=dict(LIMITS, transitions=8193)), 'network exact limits mutations changed')
            value = decode_network(raw, limits=shared)
            require(refused(raw, limits=inputs['small_shared']['limits']), 'network shared bound admitted')
            require(observed['admitted_world_sha256'] == sha(value['world']) and 'record identity or bound' in observed['shared_error'] and 'configuration mismatch' in observed['local_error'], 'network local/shared distinction lost')
        elif case == 'network_resealed_semantic_refusal':
            key = (case, inputs['mutation'], None)
            require(raw is not None and inputs['evidence_sha256'] == sha(raw), 'network semantic preimage omitted')
            # Framing and hashes are authenticated before semantic admission fails.
            r = Reader(raw); r.take(8); r.number(); r.text(); r.number()
            for _ in range(r.count()):
                tag = r.take(1)[0]
                r.operation() if tag == 1 else r.quiescence()
            record_start = r.at
            record = r.record()
            require(raw[:record_start] == fixture[:record_start] and r.flag(), "network semantic mutant changed original header/operations")
            retained_world = r.blob()
            require(r.at == len(raw) and retained_world == decode_network(fixture)["world"], "network semantic mutant changed original World/tail")
            expected = copy.deepcopy(decode_network(fixture)['record'])
            expected_limits = dict(LIMITS)
            mutation = inputs['mutation']
            if mutation == 'unknown_transition_kind': expected['transitions'][0]['kind'] = 'unknown'
            elif mutation == 'ordinal_gap': expected['transitions'][0]['ordinal'] += 1
            elif mutation == 'listener_port_zero': expected['snapshot']['listeners'][0]['endpoint']['port'] = 0
            elif mutation == 'reset_retains_bytes': expected['snapshot']['connections'][0].update(reset=True, client_bytes=1)
            elif mutation == 'connection_byte_bound': expected['snapshot']['connections'][0]['client_bytes'] = LIMITS['connection_bytes'] + 1
            elif mutation == 'total_byte_bound':
                expected_limits['bytes'] = 1
                expected['snapshot']['connections'][0]['client_bytes'] = 2
            expected['snapshot']['transition_sha256'] = record_hashes(expected)[0]
            expected['snapshot']['identity'] = record_hashes(expected)[1]
            require(same(record, inputs['record']) and same(record, expected) and inputs['shared']['limits'] == expected_limits and inputs['local']['limits'] == LIMITS and record_hashes(record) == (record['snapshot']['transition_sha256'], record['snapshot']['identity']), 'network resealed hashes or exact semantic mutation differ')
            require(refused(raw, limits=inputs['shared']['limits']) and observed['error'] and 'identity or bound' not in observed['error'], 'network resealed semantics admitted or hash-only refused')
        elif case == 'cluster_network_decoder_refusal':
            mutation = inputs['mutation']
            profile = 10 + len(PROFILE)
            changed = bytearray(fixture)
            positions = dict(magic=(0, ord('X')), version=(8, 2), invalid_utf8=(10, 255), log_accounting=(profile, fixture[profile] + 1))
            world = fixture.index(b'GMWSNAP1'); flag = world - 1
            while fixture[flag] != 1:
                flag -= 1
            positions.update(flag=(flag, 2), world_wire=(world + 20, fixture[world + 20] ^ 1), record_identity=(fixture.rindex(b'sha256:') + 7, fixture[fixture.rindex(b'sha256:') + 7] ^ 1))
            key = (case, mutation, None)
            if mutation == 'unknown_tag':
                key = (case, mutation, raw[profile + 3]); positions[mutation] = (profile + 3, key[2])
            if mutation in positions:
                at, value = positions[mutation]; changed[at] = value
            elif mutation == 'trailing':
                changed.append(0)
            elif mutation == 'non_minimal_varint':
                changed = fixture[:profile] + bytes((fixture[profile] | 128, fixture[profile + 1] | 128, 0)) + fixture[profile + 2:]
            require(raw == bytes(changed) and inputs['sha256'] == sha(raw) and inputs['bytes'] == len(raw), 'network mutation is not original raw perturbation')
            require(inputs['transcript_limit'] == (len(fixture) - 1 if mutation == 'oversize' else 1 << 20) and refused(raw, inputs['transcript_limit']) and observed['error'], 'network malformed original mutation admitted')
        else:
            raise ValueError('unknown network observation')
        require(key in required and key not in seen, 'unknown or duplicate network case')
        seen.add(key)
        facts.append(dict(case=case, name=key[1], index=key[2]))
    require(seen == required, 'network harness omitted original raw cases: ' + repr(sorted(required - seen, key=str)))
    frame_cases = {'per-node fault schedule': 'reject-local-fault', 'bad address': 'reject-ipv4-range', 'unknown link endpoint': 'reject-unknown-from', 'self link': 'reject-self-link', 'duplicate link': 'reject-duplicate-link', 'unordered fault': 'reject-unordered-faults', 'unknown action': 'reject-unknown-action'}
    return dict(original_inputs=original_inputs, frame_cases={name: dict(corpus=corpus, actual_error=corpora[corpus]['creation_error']) for name, corpus in frame_cases.items()}, roots={root: dict(cases=facts, predicates=['exact corpus literals', 'GMNETEV1 framing and Go JSON log reconstruction', 'native record hash preimages and semantic admission', 'exact broker outputs, World and per-node evidence']) for root in ROOTS}, facts=facts,
                gaps=[dict(root='TestClusterNetworkFrameRefusesWhatTheGoBrokerRefuses', reason='Shared validates central network configuration; node profile, stock-node and incarnation checks require linked prepare owner observations')])
