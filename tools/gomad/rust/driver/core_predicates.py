"""Pure raw native predicate readers; command custody and full original-root maps stay separate."""
from driver import require
from native_roles import decode
from native_observations import canonical, digest

WRITER_SCHEMA = 'gomad.native-retained-writer-observation/v1'
WRITER_ENCODING = 'gomad.artifact-manifest/v2; raw payload bytes; terminal manifest byte replaces retired Go base64-close boundary'
LAYOUT = b'{"schema":"gomad.artifact-store/v2","target_pool":".targets"}'
ENVELOPE = b'{"format":"gomad.artifact-storage/v2"}'


def raw_bytes(value):
    require(isinstance(value, list) and len(value) <= 64 << 20
            and all(type(byte) is int and 0 <= byte <= 255 for byte in value), 'native byte preimage is missing or malformed')
    return bytes(value)


def retained_writer(value):
    """Reconstruct the fixed short/zero/error writer matrix from all actual raw preimages."""
    require(set(value) == {'schema', 'encoding', 'record_sha256', 'payloads', 'manifest_bytes',
                           'layout_bytes', 'envelope_bytes', 'total_written', 'write_calls', 'cases'}
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
    budgets = [0, 1, 19, 100, 1024, total - 2, total - 1]
    require(len(set(budgets)) == 7 and max(budgets) < total and len(value['cases']) == 14,
            'native writer lacks the original seven stable offsets in both failure modes')
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
    return dict(record_sha256=value['record_sha256'], manifest_sha256=digest(manifest_raw), total_written=total, cases=rows)
