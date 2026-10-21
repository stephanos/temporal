#!/usr/bin/env python3
"""Execute retained Go guests, then verify read-only public symbol projection."""
import argparse
import hashlib
import importlib.machinery
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys

sys.dont_write_bytecode = True

corpus = Path(__file__).resolve().parent
loader = importlib.machinery.SourceFileLoader('parity', str(corpus.parents[1] / 'scripts/behavioral-parity'))
spec = importlib.util.spec_from_loader(loader.name, loader)
parity = importlib.util.module_from_spec(spec)
loader.exec_module(parity)
p = argparse.ArgumentParser()
p.add_argument('--binary', type=Path, required=True)
p.add_argument('--work', type=Path, required=True)
args = p.parse_args()
args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
receipts = []


def command(label, *argv, status=0):
    result = subprocess.run([str(args.binary), *map(str, argv)], capture_output=True, timeout=120)
    receipts.append(dict(label=label, argv=[str(args.binary), *map(str, argv)],
                         status=result.returncode, stdout=result.stdout.decode(), stderr=result.stderr.decode()))
    (args.work / 'commands.json').write_bytes(parity.canonical(receipts))
    assert result.returncode == status, (label, result.returncode, result.stderr)
    return result.stdout


def snapshot(artifact):
    return {str(path.relative_to(artifact)): path.read_bytes()
            for path in artifact.rglob('*') if path.is_file()}


fixture = json.loads((corpus / 'symbolize/execution.json').read_bytes())
expected = {entry['site']: entry['symbol']
            for entry in json.loads((corpus / 'symbolize/expected.json').read_bytes())}
views = []
for repeat in ('first', 'second'):
    root = args.work / repeat
    bundle = json.loads(command('execute-' + repeat, 'fixture-inspect-execution', '--root', root,
                               '--input', corpus / 'symbolize/execution.json'))
    parity.authenticate(root, bundle)
    assert len(bundle['records']) == 6
    repeated = []
    for index, item in enumerate(bundle['records']):
        artifact = root / item['artifact']
        before = snapshot(artifact)
        first = command(f'inspect-{repeat}-{index}', 'inspect', '--json', '--choices', artifact)
        assert first == command(f'repeat-{repeat}-{index}', 'inspect', '--json', '--choices', artifact)
        report = parity.decode(first)
        value = report['artifact']
        record = json.loads(before['record.json'])
        assert value['record'] == record
        assert value['record_sha256'] == item['record_sha256']
        assert value['failure_signature'] == item['failure_signature']
        assert record['execution_identity'] == item['execution_identity']
        case = fixture['cases'][index // 2]
        assert record['identity']['target'] == case['module_sha256']
        assert record['identity']['toolchain'] == case['compiler_sha256']
        assert before['target/module.wasm'] == (corpus / case['module']).read_bytes()
        trace = before['evidence/choice']
        if index < 4:
            assert len(trace) == 16 * 96
        raw_sites = {}
        for offset in range(0, len(trace), 96):
            raw = trace[offset:offset + 96]
            kind = {1: 'runnable', 2: 'select-poll', 3: 'select-result'}[raw[8]]
            site = int.from_bytes(raw[24:32], 'big')
            missing = bool(raw[9] & 4)
            fingerprint = 'missing' if missing else hashlib.sha256(bytes.fromhex(case['module_sha256']) + raw[8:9] + raw[24:32]).hexdigest()
            raw_sites[kind, fingerprint] = (site, missing)
        choices = value['choices']
        if index < 4:
            assert choices['records'] == 16 and choices['decisions'] == 8
            assert len(choices['sites']) == 10
        stripped = case['module'].endswith('guest-stripped.wasm')
        for site in choices['sites']:
            assert set(site) == {'fingerprint', 'kind', 'count', 'maximum_alternatives', 'symbol'}
            offset, missing = raw_sites[site['kind'], site['fingerprint']]
            if index < 4:
                assert site['symbol'] == ({'unavailable': 'no_symbols'} if stripped else expected[offset])
            elif missing:
                assert site['kind'] == 'runnable' and site['symbol'] == {'unavailable': 'site_missing'}
        for decision in choices['replay_decisions']:
            if index < 4:
                assert decision['symbol'] == ({'unavailable': 'no_symbols'} if stripped else expected[decision['site_offset']])
            else:
                assert decision['site_missing'] and decision['symbol'] == {'unavailable': 'site_missing'}
        if index >= 4:
            single, = [site for site in choices['sites'] if site['maximum_alternatives'] == 1]
            assert single['kind'] == 'select-result' and single['count'] == 1
            assert single['symbol'] == {'package': 'reflect', 'function': 'rselect',
                                        'file': 'runtime/select.go', 'line': 629, 'inlined_from': []}
            assert all(decision['kind'] != 'select-result' for decision in choices['replay_decisions'])
            assert any(site['fingerprint'] == 'missing' for site in choices['sites'])
        assert snapshot(artifact) == before
        assert report['path'] == str(artifact)
        report['path'] = '<root>/' + item['artifact']
        repeated.append(report)
    views.append(repeated)
assert views[0] == views[1]


def leb(value):
    result = bytearray()
    while True:
        byte = value & 127
        value >>= 7
        result.append(byte | (128 if value else 0))
        if not value:
            return bytes(result)


def sections(module):
    assert module[:8] == b'\0asm\x01\0\0\0'
    offset = 8
    while offset < len(module):
        section = module[offset]
        offset += 1
        length = shift = 0
        while True:
            byte = module[offset]
            offset += 1
            length |= (byte & 127) << shift
            shift += 7
            if not byte & 128:
                break
        yield section, module[offset:offset + length]
        offset += length


def metadata_variant(original, mode):
    if mode == 'range-guest':
        return range_guest(original)
    output = bytearray(original[:8])
    found = False
    for section, payload in sections(original):
        if section == 0 and payload.startswith(b'\x04name'):
            found = True
            if mode == 'name-mismatch':
                assert payload.count(b'main.positive') == 1
                payload = payload.replace(b'main.positive', b'main.xositive')
            elif mode == 'oversized-name':
                name = b'x' * ((1 << 20) + 1)
                subsection = leb(len(name)) + name
                payload = payload[:5] + b'\0' + leb(len(subsection)) + subsection + payload[5:]
        output.append(section)
        output.extend(leb(len(payload)))
        output.extend(payload)
    assert found
    if mode == 'duplicate-name':
        payload = b'\x04name'
        output.extend(b'\0' + leb(len(payload)) + payload)
    return bytes(output)


def var(data, offset):
    value = shift = 0
    while True:
        byte = data[offset]
        offset += 1
        value |= (byte & 127) << shift
        shift += 7
        assert shift <= 64
        if not byte & 128:
            return value, offset


def constant(value):
    encoded = bytearray(leb(value))
    if encoded[-1] & 64:
        encoded[-1] |= 128
        encoded.append(0)
    return b'\x41' + encoded


def string(value):
    return leb(len(value)) + value


def fixture_request():
    source = (corpus / 'targets/inspect-choice.wat').read_text()
    literal, = re.findall(r'\(data \(i32.const 64\) "([^"]*)"\)', source)
    request = bytes(int(value, 16) for value in re.findall(r'\\([0-9a-f]{2})', literal))
    assert len(request) == 96 and int.from_bytes(request[8:16], 'big') == 42
    return request


def fixture_body(indices):
    return b'\0' + b''.join(constant(address) + constant(length) + b'\x10' + leb(index)
                            + b'\x04\x40\0\x0b'
                            for index, address, length in [(indices[0], 0, 16), (indices[1], 64, 96), (indices[2], 16, 16)]) + b'\x0b'


def fixture_data():
    return b'\0' + constant(16) + b'\x0b\x04\0\0\0\x01\0' + constant(64) + b'\x0b' + string(fixture_request())


def range_guest(original):
    payloads = dict(sections(original))
    assert 8 not in payloads

    def read_string(data, offset):
        length, offset = var(data, offset)
        return data[offset:offset + length], offset + length

    count, offset = var(payloads[2], 0)
    indices = {}
    for index in range(count):
        module, offset = read_string(payloads[2], offset)
        name, offset = read_string(payloads[2], offset)
        assert payloads[2][offset] == 0
        _, offset = var(payloads[2], offset + 1)
        if module == b'gomad_wasm_v1':
            indices[name] = index
    assert offset == len(payloads[2])
    imports = count
    count, offset = var(payloads[7], 0)
    exports = []
    start = None
    for _ in range(count):
        name, offset = read_string(payloads[7], offset)
        kind = payloads[7][offset]
        index, offset = var(payloads[7], offset + 1)
        exports.append((name, kind, index))
        if name == b'_start':
            assert kind == 0
            start = index
    assert offset == len(payloads[7]) and start is not None
    functions, offset = var(payloads[3], 0)
    type_indices = []
    for _ in range(functions):
        index, offset = var(payloads[3], offset)
        type_indices.append(index)
    assert offset == len(payloads[3])
    _, prefix = var(payloads[3], 0)
    changes = {3: leb(functions + 1) + payloads[3][prefix:] + leb(type_indices[start - imports])}
    changes[7] = leb(count) + b''.join(string(name) + bytes([kind])
                                      + leb(imports + functions if name == b'_start' else index)
                                      for name, kind, index in exports)
    count, prefix = var(payloads[10], 0)
    assert count == functions
    code = fixture_body([indices[name] for name in (b'config', b'decision', b'finish')])
    changes[10] = leb(count + 1) + payloads[10][prefix:] + string(code)
    count, prefix = var(payloads[11], 0)
    changes[11] = leb(count + 2) + payloads[11][prefix:] + fixture_data()
    if 12 in payloads:
        declared, end = var(payloads[12], 0)
        assert declared == count and end == len(payloads[12])
        changes[12] = leb(count + 2)
    return original[:8] + b''.join(bytes([section]) + string(changes.get(section, data))
                                  for section, data in sections(original))


def no_pclntab():
    def section(kind, value):
        return bytes([kind]) + leb(len(value)) + value

    imports = b'\x03' + b''.join(string(b'gomad_wasm_v1') + string(name) + b'\0\0'
                                for name in (b'config', b'decision', b'finish'))
    code = fixture_body([0, 1, 2])
    result = b'\0asm\x01\0\0\0' + section(1, b'\x02\x60\x02\x7f\x7f\x01\x7f\x60\0\0')
    result += section(2, imports) + section(3, b'\x01\x01') + section(5, b'\x01\0\x01')
    result += section(7, b'\x02' + string(b'memory') + b'\x02\0' + string(b'_start') + b'\0\x03')
    result += section(10, b'\x01' + string(code)) + section(11, b'\x02' + fixture_data())
    names = b'\x01\x03' + string(b'fixture.start')
    result += section(0, b'\x04name\x01' + string(names))
    assert any(section == 0 and data.startswith(b'\x04name') for section, data in sections(result))
    assert b'\xf1\xff\xff\xff' not in result
    return result


stripped = list(sections((corpus / 'symbolize/guest-stripped.wasm').read_bytes()))
assert not any(section == 0 and data.startswith(b'\x04name') for section, data in stripped)
assert any(section == 11 and b'\xf1\xff\xff\xff\0\0\x01\x08' in data for section, data in stripped)
negative_views = []
for mode, case_index, reason in [('name-mismatch', 0, 'symbol_mismatch'),
                                 ('duplicate-name', 0, 'symbol_mismatch'),
                                 ('oversized-name', 2, 'symbol_table_too_large'),
                                 ('range-guest', 0, 'offset_out_of_range'),
                                 ('no-pclntab', 0, 'no_symbols')]:
    directory = args.work / mode
    (directory / 'input').mkdir(parents=True)
    case = dict(fixture['cases'][case_index])
    original = (corpus / case['module']).read_bytes()
    changed = no_pclntab() if mode == 'no-pclntab' else metadata_variant(original, mode)
    original_code = [payload for section, payload in sections(original) if section == 10]
    if mode in ('no-pclntab', 'range-guest'):
        case = dict(name=mode, fuel=100000, memory_bytes=67108864, wall_millis=30000,
                    modeled=True, runtime=True, choice_limit=4096, termination='exit')
    else:
        assert [payload for section, payload in sections(changed) if section == 10] == original_code
    case.update(module='guest.wasm', module_sha256=hashlib.sha256(changed).hexdigest())
    (directory / 'guest.wasm').write_bytes(changed)
    declaration = dict(fixture, cases=[case])
    input_path = directory / 'input/execution.json'
    input_path.write_bytes(parity.canonical(declaration))
    root = directory / 'published'
    result = json.loads(command(mode, 'fixture-inspect-execution', '--root', root, '--input', input_path))
    parity.authenticate(root, result)
    for index, item in enumerate(result['records']):
        artifact = root / item['artifact']
        before = snapshot(artifact)
        raw = command(f'{mode}-{index}', 'inspect', '--json', '--choices', artifact)
        assert raw == command(f'{mode}-repeat-{index}', 'inspect', '--json', '--choices', artifact)
        report = parity.decode(raw)
        choices = report['artifact']['choices']
        for site in choices['sites']:
            wanted = {'unavailable': 'site_missing' if site['fingerprint'] == 'missing' else reason}
            assert site['symbol'] == wanted
        for decision in choices['replay_decisions']:
            wanted = {'unavailable': 'site_missing' if decision['site_missing'] else reason}
            assert decision['symbol'] == wanted
        assert snapshot(artifact) == before
        assert report['path'] == str(artifact)
        report['path'] = '<root>/' + item['artifact']
        negative_views.append(report)

source = args.work / 'first' / bundle['records'][0]['artifact']
source_before = snapshot(source)
for name in ('module-digest', 'module-unreadable'):
    detached = args.work / name
    shutil.copytree(source, detached)
    module = detached / 'target/module.wasm'
    if name == 'module-digest':
        module.chmod(0o600)
        value = bytearray(module.read_bytes())
        value[-1] ^= 1
        module.write_bytes(value)
    else:
        module.unlink()
    assert not command(name, 'inspect', '--json', '--choices', detached, status=2)
    if name == 'module-digest':
        assert 'identity mismatch' in receipts[-1]['stderr']
assert snapshot(source) == source_before

golden = corpus / 'goldens-inspect/symbols.json'
view = parity.canonical(dict(positive=views[0], negative=negative_views))
if os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1':
    golden.write_bytes(view)
assert golden.read_bytes() == view, 'symbol inspect golden differs; identities must not be normalized'
print('public symbols: 12 fresh Go publications, 10 negative metadata publications, repeated bytes and unchanged artifacts')
