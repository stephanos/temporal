"""Pure raw native predicate readers; command custody and full original-root maps stay separate."""
from driver import require
from native_roles import decode
from native_observations import canonical, digest

WRITER_SCHEMA = 'gomad.native-retained-writer-observation/v1'
WRITER_ENCODING = 'gomad.artifact-manifest/v2; raw record tail replaces retired Go base64-close; manifest tails are additional native boundaries'
LAYOUT = b'{"schema":"gomad.artifact-store/v2","target_pool":".targets"}'
ENVELOPE = b'{"format":"gomad.artifact-storage/v2"}'


def raw_bytes(value):
    require(isinstance(value, list) and len(value) <= 64 << 20
            and all(type(byte) is int and 0 <= byte <= 255 for byte in value), 'native byte preimage is missing or malformed')
    return bytes(value)


def retained_writer(value):
    """Reconstruct the fixed short/zero/error writer matrix from all actual raw preimages."""
    require(set(value) == {'schema', 'encoding', 'record_sha256', 'payloads', 'manifest_bytes',
                           'layout_bytes', 'envelope_bytes', 'total_written', 'record_tail', 'write_calls', 'cases'}
            and value['schema'] == WRITER_SCHEMA and value['encoding'] == WRITER_ENCODING,
            'foreign native retained writer observation')
    require(raw_bytes(value['layout_bytes']) == LAYOUT and raw_bytes(value['envelope_bytes']) == ENVELOPE,
            'native retained writer storage framing changed')
    require(set(value['payloads']) == {'record.json', 'target/module.wasm', 'output/stdout', 'evidence/binary-writer-tail'},
            'native writer omitted an original raw payload')
    payloads = {name: raw_bytes(raw) for name, raw in value['payloads'].items()}
    manifest_raw = raw_bytes(value['manifest_bytes'])
    record = decode(payloads['record.json'])
    require(payloads['record.json'] == canonical(record) and value['record_sha256'] == digest(payloads['record.json']),
            'native writer record bytes/digest are not their canonical actual preimage')
    require(record['arguments'] == ['quoted="tail"', 'Unicode=☃'] and record['environment'] == {'EMPTY': ''}
            and record['virtual_time_nanos'] == (1 << 64) - 8
            and payloads['evidence/binary-writer-tail'] == bytes(range(256)) + b'\0'
            and payloads['target/module.wasm'] == b'(module)'
            and payloads['output/stdout'] == b'foundation-output',
            'native writer lost original quoting/Unicode/empty/high-integer/binary witnesses')
    expected_record = field_record(257)
    expected_record.update(arguments=['quoted="tail"', 'Unicode=☃'], environment={'EMPTY': ''}, seed=7,
                           virtual_time_nanos=(1 << 64) - 8,
                           outcome={'class': 'success', 'termination': 'exit', 'exit_code': 0, 'reason': ''})
    expected_record['streams']['evidence/binary-writer-tail']['sha256'] = digest(bytes(range(256)) + b'\0')
    require(payloads['record.json'] == canonical(expected_record), 'native writer lost an exact declared record field')
    require(all(type(stream['bytes']) is int and type(stream['records']) is int for stream in record['streams'].values()),
            'native writer stream counts are not typed integers')
    require(record['streams'] == {name: dict(bytes=len(payloads[name]), records=records, sha256=digest(payloads[name]))
                                 for name, records in (('evidence/binary-writer-tail', 1), ('output/stdout', 0))},
            'native writer record lost its actual binary/output streams')
    files = {name: dict(bytes=len(raw), sha256=digest(raw), module=name == 'target/module.wasm') for name, raw in payloads.items()}
    manifest = dict(schema='gomad.artifact-manifest/v2', files=files, record=record, record_sha256=value['record_sha256'])
    require(manifest_raw == canonical(manifest), 'native writer manifest does not authenticate every actual payload and record')
    total = len(LAYOUT) + len(ENVELOPE) + len(manifest_raw) + sum(map(len, payloads.values()))
    require(type(value['total_written']) is int and value['total_written'] == total
            and type(value['write_calls']) is int and value['write_calls'] >= (total + 2) // 3
            and value['write_calls'] > 32,
            'native short writer byte total/count differs from actual serialized files')
    record_tail = len(LAYOUT) + sum(len(payloads[name]) for name in sorted(payloads) if name < 'record.json') + len(payloads['record.json']) - 1
    require(type(value['record_tail']) is int and value['record_tail'] == record_tail,
            'native writer raw record tail does not match the actual publication order')
    budgets = [0, 1, 19, 100, 1024, record_tail, total - 2, total - 1]
    require(len(set(budgets)) == 8 and max(budgets) < total and len(value['cases']) == 16,
            'native writer lacks the original seven offsets plus the additional manifest boundary in both failure modes')
    rows = []
    for row, (mode, budget) in zip(value['cases'], ((mode, budget) for mode in ('zero', 'error') for budget in budgets)):
        require(set(row) == {'mode', 'budget', 'written', 'write_calls', 'failures', 'error', 'committed_before_retry',
                             'retry_record_sha256', 'retry_record_bytes', 'retry_manifest_bytes'}, 'foreign native writer case')
        require(row['mode'] == mode and type(row['budget']) is int and row['budget'] == budget
                and type(row['written']) is int and row['written'] == budget
                and type(row['failures']) is int and row['failures'] == 1
                and type(row['write_calls']) is int and row['write_calls'] >= (budget + 2) // 3 + 1
                and type(row['committed_before_retry']) is int and row['committed_before_retry'] == 0,
                'native writer did not stop exactly at the first actual I/O failure')
        error = 'failed to write whole buffer' if mode == 'zero' else 'injected retained writer error'
        require(isinstance(row['error'], str) and error in row['error'], 'native writer lost the actual I/O error')
        require(raw_bytes(row['retry_record_bytes']) == payloads['record.json']
                and raw_bytes(row['retry_manifest_bytes']) == manifest_raw
                and row['retry_record_sha256'] == value['record_sha256'],
                'native writer atomic retry changed the exact record or payload manifest')
        rows.append(dict(mode=mode, budget=budget, written=row['written']))
    return dict(record_sha256=value['record_sha256'], manifest_sha256=digest(manifest_raw), total_written=total, record_tail=record_tail, cases=rows)


FIELD_SIZES = (0, 1, 2, 3, 1022, 1023, 1024, 1025, 2047, 2048, 2049, 8191, 8192, 8193)


def field_record(size):
    """The fixed native fixture's eleven fields, independently reconstructed from source constants."""
    identity = {name: digest(b'(module)' if name == 'target' else ('foundation-' + name + '-v1').encode())
                for name in ('target', 'toolchain', 'platform', 'implementation', 'configuration')}
    payload = b'\xff' * size
    return dict(schema='gomad.execution-record/v2', execution_identity=digest(canonical(identity)), identity=identity,
                profile='gomad.foundation/v1', seed=(1 << 64) - 1,
                arguments=['<&\u2028', '\u2029&', ''], environment={'<&\u2028': '\u2029', 'EMPTY': ''},
                limits=dict(fuel=100000, memory_bytes=65536, wall_millis=100, output_bytes=1024, evidence_bytes=4096),
                virtual_time_nanos=(1 << 64) - 1,
                outcome={'class': 'success', 'termination': 'exit', 'exit_code': 0, 'reason': 'literal <&\u2028\u2029'},
                streams={name: dict(bytes=len(raw), records=count, sha256=digest(raw))
                         for name, raw, count in (('output/stdout', b'foundation-output', 0),
                                                  ('evidence/binary-writer-tail', payload, 1))})


def retained_fields(value):
    """Read raw field/payload-size facts only; execution custody and original-root coverage are separate."""
    require(set(value) == {'schema', 'encoding', 'cases'}
            and value['schema'] == 'gomad.native-retained-fields-observation/v1'
            and value['encoding'] == 'gomad.execution-record/v2; separate raw binary payloads'
            and isinstance(value['cases'], list) and len(value['cases']) == len(FIELD_SIZES),
            'foreign native retained fields observation')
    rows = []
    for row, size in zip(value['cases'], FIELD_SIZES):
        require(set(row) == {'size', 'record_bytes', 'payload_bytes', 'manifest_bytes'}
                and type(row['size']) is int and row['size'] == size, 'native retained payload size matrix changed')
        payload = raw_bytes(row['payload_bytes'])
        require(payload == b'\xff' * size, 'native retained binary payload changed')
        record = field_record(size)
        record_raw = raw_bytes(row['record_bytes'])
        require(record_raw == canonical(record), 'native retained record lost an exact declared field or integer/Unicode witness')
        payloads = {'record.json': record_raw, 'target/module.wasm': b'(module)', 'output/stdout': b'foundation-output',
                    'evidence/binary-writer-tail': payload}
        manifest = dict(schema='gomad.artifact-manifest/v2', record=record, record_sha256=digest(record_raw),
                        files={name: dict(bytes=len(raw), sha256=digest(raw), module=name == 'target/module.wasm')
                               for name, raw in payloads.items()})
        manifest_raw = raw_bytes(row['manifest_bytes'])
        require(manifest_raw == canonical(manifest), 'native retained fields manifest does not authenticate actual record and payload bytes')
        rows.append(dict(size=size, record_sha256=digest(record_raw), payload_sha256=digest(payload),
                         manifest_sha256=digest(manifest_raw)))
    return dict(cases=rows)
