"""Reconstruct native retained admission and oracle facts from actual raw preimages."""
import base64
import copy
from pathlib import PurePosixPath

from driver import require
from native_roles import decode
from native_observations import canonical, digest
from core_world import decode_world, expand

FILE = 'evidence/cluster-replay.json'
MATRIX_MARKER = 'GOMAD_RETAINED_LIFECYCLE_MATRIX '
MUTATIONS = ('missing-action', 'mutated-action', 'missing-module', 'foreign-module',
             'boot-module-differs', 'changed-limits', 'changed-environment', 'changed-helper-budget',
             'changed-choice-budget', 'malformed-storage-selection', 'changed-imports',
             'missing-model-implementation', 'changed-model-implementation', 'changed-world-profile',
             'changed-world-encoding', 'trailing-world-byte', 'invalid-world-header',
             'corrupt-world-digest', 'changed-runtime-binding', 'changed-original-record', 'changed-root-identity')
BOUNDARIES = {'verify', 'storage-projection', 'prepare-reopen', 'public-cli-replay'}


def b64(text, maximum=128 << 20):
    require(isinstance(text, str) and len(text) <= (maximum + 2) // 3 * 4, 'retained preimage exceeds byte bound')
    value = base64.b64decode(text, validate=True)
    require(len(value) <= maximum and base64.b64encode(value).decode() == text, 'retained noncanonical base64')
    return value


def fields(value, expected, message):
    require(isinstance(value, dict) and set(value) == set(expected), message)


def same(left, right):
    return canonical(left) == canonical(right)


def safe_name(name):
    require(isinstance(name, str) and name and not name.startswith('/') and '\\' not in name
            and all(part not in ('', '.', '..') for part in name.split('/'))
            and str(PurePosixPath(name)) == name, 'foreign retained payload path')


def artifact(value, *, authenticate=True):
    fields(value, ('manifest_base64', 'record_base64', 'payloads_base64'), 'foreign retained raw artifact')
    manifest_raw, record_raw = b64(value['manifest_base64']), b64(value['record_base64'])
    manifest, record = decode(manifest_raw), decode(record_raw)
    fields(manifest, ('schema', 'files', 'record', 'record_sha256'), 'foreign retained manifest')
    fields(record, ('schema', 'identity', 'execution_identity', 'profile', 'seed', 'arguments', 'environment',
                    'limits', 'virtual_time_nanos', 'outcome', 'streams'), 'foreign native record')
    fields(record['identity'], ('target', 'toolchain', 'platform', 'implementation', 'configuration'), 'foreign native identity')
    require(record['schema'] == 'gomad.execution-record/v2', 'foreign native record lineage')
    require(manifest['schema'] == 'gomad.artifact-manifest/v2' and manifest_raw == canonical(manifest), 'retained manifest lineage or canonical bytes changed')
    payloads = {name: b64(raw) for name, raw in value['payloads_base64'].items()}
    require(set(payloads) == set(manifest['files']) and payloads.get('record.json') == record_raw, 'retained raw payload roster changed')
    for name, raw in payloads.items():
        safe_name(name)
        entry = manifest['files'][name]
        fields(entry, ('bytes', 'sha256', 'module'), 'foreign native payload descriptor')
        require(type(entry['bytes']) is int and entry['bytes'] == len(raw)
                and entry['sha256'] == digest(raw) and entry['module'] is (name == 'target/module.wasm'),
                'retained manifest does not bind actual bytes: ' + name)
    require(isinstance(record.get('streams'), dict), 'missing actual record streams')
    for name, stream in record['streams'].items():
        require(name in payloads and type(stream['bytes']) is int and stream['bytes'] == len(payloads[name])
                and stream['sha256'] == digest(payloads[name]), 'native record stream does not bind actual payload: ' + name)
    violations = []
    if record_raw != canonical(record):
        violations.append('noncanonical-record')
    if manifest['record_sha256'] != digest(record_raw):
        violations.append('record-sha256')
    if not same(manifest['record'], record):
        violations.append('record-projection')
    if record.get('execution_identity') != digest(canonical(record.get('identity'))):
        violations.append('execution-identity')
    if authenticate:
        require(not violations, 'retained canonical record authentication failed: ' + repr(violations))
    if 'target/module.wasm' in payloads:
        require(record['identity']['target'] == digest(payloads['target/module.wasm']), 'actual module differs from record target')
    return dict(manifest=manifest, record=record, payloads=payloads, violations=violations)


def snapshot(entries, actual):
    require(isinstance(entries, dict) and entries, 'missing actual directory preimage')
    prefixes = [name[:-len('manifest.json')] for name in entries if name.endswith('/manifest.json')]
    require(len(prefixes) == 1, 'retained snapshot did not observe exactly one artifact')
    prefix = prefixes[0]
    raw_files = dict(actual['payloads'])
    raw_files['manifest.json'] = canonical(actual['manifest'])
    raw_files['storage.json'] = b'{"format":"gomad.artifact-storage/v2"}'
    for name, raw in raw_files.items():
        require(entries.get(prefix + name) == dict(bytes=len(raw), sha256=digest(raw)),
                'directory snapshot is not the actual raw artifact: ' + name)
    for name, entry in entries.items():
        safe_name(name)
        require(entry == {'directory': True} or (set(entry) == {'bytes', 'sha256'}
                and type(entry['bytes']) is int and entry['bytes'] >= 0
                and isinstance(entry['sha256'], str) and len(entry['sha256']) == 64
                and all(char in '0123456789abcdef' for char in entry['sha256'])), 'foreign directory entry')


def native_baseline(actual):
    inputs = decode(actual['payloads'][FILE])
    configuration = inputs['configuration']
    require(inputs['schema'] == 'gomad.cluster-replay-inputs/v1'
            and configuration['world_encoding'] == configuration['world_profile'] == 'gomad/world/compressed-snapshot/v2'
            and configuration['profile'] == actual['record']['profile'] == 'gomad.wasi-cooperative/v1'
            and actual['record']['identity']['configuration'] == digest(canonical(configuration)),
            'baseline lacks its actual bound native configuration')
    require([item['handle'] for item in configuration['incarnations']] == [
        dict(node='a', incarnation=1), dict(node='b', incarnation=1), dict(node='a', incarnation=2)],
        'matrix did not retain the actual three-incarnation lifecycle')
    require(sorted(canonical(item['handle']) for item in inputs['retained']) == sorted(canonical(item['handle']) for item in configuration['incarnations']),
            'actual retained handle coverage changed')
    broker = configuration['broker']
    decode_world(expand(actual['payloads']['evidence/world']), dict(seed=broker['seed'], **broker['limits']))
    return inputs



def reseal(payloads, record, template):
    payloads = dict(payloads)
    for name in (FILE, 'evidence/world'):
        if name in payloads:
            record['streams'][name]['bytes'] = len(payloads[name])
            record['streams'][name]['sha256'] = digest(payloads[name])
    payloads['record.json'] = canonical(record)
    manifest = copy.deepcopy(template)
    manifest['record'] = record
    manifest['record_sha256'] = digest(payloads['record.json'])
    manifest['files'] = {name: dict(bytes=len(raw), sha256=digest(raw), module=template['files'][name]['module'])
                         for name, raw in payloads.items()}
    return payloads, manifest


def expected_mutation(before, case):
    payloads = dict(before['payloads'])
    inputs = decode(payloads[FILE])
    record = copy.deepcopy(before['record'])
    c = inputs['configuration']
    item = c['incarnations'][0]
    zeros = '0' * 64
    if case == 'missing-action': c['steps'].pop()
    elif case == 'mutated-action': c['steps'][0]['handle']['incarnation'] += 1
    elif case == 'missing-module': item['module']['sha256'] = ''
    elif case == 'foreign-module': item['module']['sha256'] = zeros
    elif case == 'boot-module-differs': c['incarnations'][1]['module']['sha256'] = zeros
    elif case == 'changed-limits': c['broker']['limits']['max_transitions'] += 1
    elif case == 'changed-environment': item['namespace']['epoch_nanos'] += 1
    elif case == 'changed-helper-budget': item['limits']['wall_millis'] += 1
    elif case == 'changed-choice-budget': item['model']['limit'] += 1
    elif case == 'malformed-storage-selection': item['storage_crash_selection'] = {'unknown': [0, 0]}
    elif case == 'changed-imports': item['imports'] = []
    elif case == 'missing-model-implementation': inputs['runtime_implementation_sha256'] = ''
    elif case == 'changed-model-implementation': inputs['runtime_implementation_sha256'] = zeros
    elif case == 'changed-world-profile': c['world_profile'] = 'unknown'
    elif case == 'changed-world-encoding': c['world_encoding'] = 'unknown'
    elif case in ('trailing-world-byte', 'invalid-world-header', 'corrupt-world-digest'):
        raw = bytearray(payloads['evidence/world'])
        if case == 'trailing-world-byte': raw.append(0)
        else: raw[0 if case == 'invalid-world-header' else 32] ^= 1
        payloads['evidence/world'] = bytes(raw)
    elif case == 'changed-runtime-binding': item['choice_identity']['ToolchainBuildKey'] = 'changed'
    else: require(case in ('valid', 'changed-original-record', 'changed-root-identity'), 'foreign original lifecycle mutation')
    payloads[FILE] = canonical(inputs)
    payloads, manifest = reseal(payloads, record, before['manifest'])
    if case == 'changed-original-record':
        payloads['record.json'] += b'\n'
    elif case == 'changed-root-identity':
        record['execution_identity'] = zeros
        payloads['record.json'] = canonical(record)
        manifest['record'] = record
        manifest['record_sha256'] = digest(payloads['record.json'])
    for name, raw in payloads.items():
        manifest['files'][name]['bytes'] = len(raw)
        manifest['files'][name]['sha256'] = digest(raw)
    return payloads, manifest


def boundary_facts(boundaries, *, admitted, actual=None):
    fields(boundaries, BOUNDARIES, 'retained matrix omitted an original receiving boundary')
    for name, row in boundaries.items():
        fields(row, ('error', 'value', 'entries_before', 'entries_after', 'entry_count_before', 'entry_count_after'), 'foreign retained boundary observation')
        require(row['entries_before'] == row['entries_after'] and type(row['entry_count_before']) is int
                and row['entry_count_before'] == row['entry_count_after'] == len(row['entries_before']),
                'retained admission mutated private files before refusing: ' + name)
        if actual is not None:
            snapshot(row['entries_before'], actual)
        if admitted:
            require(row['error'] is None and isinstance(row['value'], dict), 'genuine retained positive control refused')
            if name == 'verify':
                fields(row['value'], ('execution_identity',), 'foreign verify observation')
                require(actual is None or row['value']['execution_identity'] == actual['record']['execution_identity'], 'verify returned foreign native identity')
            elif name == 'prepare-reopen':
                require(row['value'] == dict(exact=True, available=True), 'retained preparation was not exact and available')
            elif name == 'storage-projection':
                fields(row['value'], ('selection_entries', 'selection_bytes_base64'), 'foreign selection projection')
                require(type(row['value']['selection_entries']) is int and row['value']['selection_entries'] == 0
                        and b64(row['value']['selection_bytes_base64']) == b'[]', 'native baseline selection projection changed')
            if name == 'public-cli-replay':
                require(row['value']['status'] == 0 and b'reproduced=true' in b64(row['value']['output_base64']), 'retained public replay did not reproduce')
        else:
            require(isinstance(row['error'], str) and row['error'] and row['value'] is None, 'mutated retained input admitted')
    return dict(boundaries=sorted(boundaries), admitted=admitted)


def lifecycle_matrix(stdout):
    rows = [decode(line.split(MATRIX_MARKER, 1)[1]) for line in stdout.split('\n') if MATRIX_MARKER in line]
    expected = ('valid', *MUTATIONS, 'module-nonembedding')
    require([row.get('case') for row in rows] == list(expected), 'original retained mutation inventory omitted, duplicated or reordered')
    facts = []
    original = None
    for row in rows[:-1]:
        fields(row, ('schema', 'case', 'native_mapping', 'native_identity', 'original_record_base64', 'before', 'after', 'boundaries'), 'foreign retained mutation evidence')
        require(row['schema'] == 'gomad.retained-lifecycle-matrix/v1' and isinstance(row['native_mapping'], str), 'foreign retained mutation lineage')
        before = artifact(row['before'])
        after = artifact(row['after'], authenticate=False)
        native_baseline(before)
        if original is None: original = before
        require(before == original, 'matrix substituted the original raw artifact between mutations')
        require(before['payloads']['record.json'] == b64(row['original_record_base64'])
                and same(before['record']['identity'], row['native_identity']), 'retained matrix replaced original identity/record')
        payloads, manifest = expected_mutation(before, row['case'])
        require(payloads == after['payloads'] and same(manifest, after['manifest']), 'retained matrix did not execute the exact original mutation: ' + row['case'])
        facts.append(dict(case=row['case'], **boundary_facts(row['boundaries'], admitted=row['case'] == 'valid', actual=after)))
    row = rows[-1]
    fields(row, ('schema', 'case', 'native_identity', 'small', 'before', 'after', 'boundaries', 'retained_bytes', 'nonmodule_payload_bytes', 'original_record_bytes', 'module_bytes', 'bound_bytes', 'bound_mapping'), 'foreign nonembedding evidence')
    small, before, after = (artifact(row[name]) for name in ('small', 'before', 'after'))
    require(row['schema'] == 'gomad.retained-lifecycle-matrix/v1'
            and same(row['native_identity'], before['record']['identity']), 'foreign nonembedding native identity')
    require(small == original, 'nonembedding comparison replaced the original small publication')
    native_baseline(small)
    native_baseline(before)
    module = before['payloads']['target/module.wasm']
    require(wasm_custom_padding(module), 'large target lacks actual custom section padding')
    require(module.startswith(b'\0asm\x01\0\0\0') and len(module) > 1 << 20 and len(small['payloads']['target/module.wasm']) < len(module), 'nonembedding module did not actually grow')
    require(after['payloads'] == {name: raw for name, raw in before['payloads'].items() if name != 'target/module.wasm'}, 'module-free retention changed a nonmodule byte')
    require(row['retained_bytes'] == len(before['payloads'][FILE]) and row['module_bytes'] == len(module)
            and row['original_record_bytes'] == len(before['payloads']['record.json']), 'retained nonembedding size lacks raw preimage')
    nonmodule = sum(len(raw) for name, raw in before['payloads'].items() if name not in ('target/module.wasm', 'record.json'))
    bound = row['original_record_bytes'] * 4 // 3 + (64 << 10)
    require(row['nonmodule_payload_bytes'] == nonmodule <= bound == row['bound_bytes'], 'original retained encoded lower bound changed')
    inputs = decode(before['payloads'][FILE])
    require(all(item['module']['sha256'] == digest(module) and item['module']['bytes'] == len(module) for item in inputs['configuration']['incarnations']), 'retained metadata lost large target binding')
    facts.append(dict(case=row['case'], nonmodule_payload_bytes=nonmodule, original_bound=bound,
                      **boundary_facts(row['boundaries'], admitted=True, actual=before)))
    return facts



def wasm_custom_padding(module):
    """Read bounded unsigned LEB section frames, independently locating real custom padding."""
    require(module.startswith(b'\0asm\x01\0\0\0'), 'foreign large Wasm header')
    position = 8
    padding = False
    def number():
        nonlocal position
        value = 0
        for shift in range(0, 35, 7):
            require(position < len(module), 'truncated native Wasm size')
            byte = module[position]
            position += 1
            value |= (byte & 127) << shift
            if not byte & 128:
                require(value <= 0xffffffff, 'Wasm section size overflow')
                return value
        require(False, 'oversized native Wasm LEB')
    while position < len(module):
        kind = module[position]
        position += 1
        size = number()
        end = position + size
        require(kind <= 12 and end <= len(module), 'invalid large Wasm section bounds')
        if kind == 0:
            length = number()
            require(position + length <= end, 'custom section name exceeds section')
            name = module[position:position + length]
            position += length
            if name == b'gomad-retention-padding-v1':
                padding = end - position > (1 << 20) - 128 and not any(module[position:end])
        position = end
    return padding
