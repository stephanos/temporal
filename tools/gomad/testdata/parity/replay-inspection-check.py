#!/usr/bin/env python3
"""Replay inspection must admit its grammar and refuse unavailable evidence before execution."""
import argparse
import base64
import gzip
import json
from pathlib import Path
import stat
import struct
import subprocess

p = argparse.ArgumentParser()
p.add_argument('--binary', type=Path, required=True)
p.add_argument('--work', type=Path, required=True)
p.add_argument('--artifact', type=Path, help='fresh identity-matching cooperative artifact with dormant hooks')
p.add_argument('--go-binary', type=Path)
p.add_argument('--go-reference', type=Path)
p.add_argument('--duration-edges', action='store_true')
p.add_argument('--role', choices=['stress-jitter', 'determinism-fixture'])
args = p.parse_args()
args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
receipts = []


def run(name, arguments, binary=None):
    if args.role and binary is None and arguments[0] == 'replay':
        arguments = [arguments[0], '--' + args.role, *arguments[1:]]
    command = [str(binary or args.binary), *map(str, arguments)]
    result = subprocess.run(command, capture_output=True, timeout=180)
    receipts.append(dict(name=name, argv=command, status=result.returncode,
                         stdout=result.stdout.decode(), stderr=result.stderr.decode()))
    (args.work / 'commands.json').write_text(json.dumps(receipts, indent=2) + '\n')
    return result


def grammar():
    produced = run('produce', ['fixture-replay-execution', '--root', args.work / 'inputs',
                              '--corpus', Path(__file__).resolve().parent])
    assert produced.returncode == 0, produced.stderr
    bundle = json.loads((args.work / 'inputs/foundation-result.json').read_bytes())
    artifact = args.work / 'inputs' / bundle['records'][0]['artifact']
    for name, flags, diagnostic in [
    ('stop', ['--stop=choice=0'], 'inspection'),
    ('time', ['--stop=time=1ns'], 'inspection'),
    ('dumps', ['--dump=stacks', '--dump=heap'], 'inspection'),
    ('probe', ['--probe=runtime/*'], 'inspection'),
    ('search', ['--check=healthy', '--choice-range=0:1'], 'inspection'),
    ('search-missing-range', ['--check=healthy'], 'search requires'),
    ('target-half', ['--instance=guest'], 'both instance'),
    ('bounds', ['--debug-bytes=0'], 'invalid byte size'),
    ]:
        result = run(name, ['replay', *flags, artifact])
        assert result.returncode == 2, (name, result.returncode, result.stderr)
        assert diagnostic in result.stderr.decode(), (name, result.stderr)
        assert b'unknown replay option' not in result.stderr, (name, result.stderr)
    for role in ('stress-jitter', 'determinism-fixture'):
        result = run('stock-role-' + role, ['replay', '--' + role, '--stop=choice=0', artifact])
        assert result.returncode == 2 and b'requires a cooperative WASM installation' in result.stderr, result.stderr
    print('replay inspection: bounded grammar and preflight refusals')


if args.role:
    assert args.artifact, 'role inspection requires an explicitly role-recorded artifact'
else:
    grammar()

if args.artifact:
    artifact = args.artifact.resolve()
    for selected in (None, 'stress-jitter', 'determinism-fixture'):
        if selected == args.role:
            continue
        flags = ['--' + selected] if selected else []
        result = run('public-wrong-role-' + (selected or 'default'),
                     ['replay', *flags, '--stop=choice=0', artifact], args.binary)
        assert result.returncode == 3 and b'WASM instrumentation divergence' in result.stderr, result.stderr
        assert not result.stdout, result.stdout
    for name, flags in [('role-conflict', ['--stress-jitter', '--determinism-fixture', '--stop=choice=0']),
                        ('stress-no-stop', ['--stress-jitter']),
                        ('determinism-no-stop', ['--determinism-fixture'])]:
        result = run('public-' + name, ['replay', *flags, artifact], args.binary)
        assert result.returncode == 2 and not result.stdout, (name, result.returncode, result.stderr)
    manifest = json.loads((artifact / 'manifest.json').read_bytes())
    metadata = json.loads((artifact / 'evidence/inspection').read_bytes())
    assert metadata['complete'], 'positive inspection requires complete retained metadata'
    frames = base64.b64decode(metadata['boundary_frames'], validate=True)
    assert frames[:8] == b'GMWIB\x01\0\0'
    count, = struct.unpack_from('<I', frames, 8)
    assert count > 0, 'positive inspection requires retained predecision boundaries'
    boundaries, offset = [], 12
    for _ in range(count):
        length, = struct.unpack_from('<I', frames, offset)
        frame = frames[offset + 4:offset + 4 + length]
        name_bytes, = struct.unpack_from('<I', frame)
        node = frame[4:4 + name_bytes].decode()
        incarnation, decision, trace, order, time, _, _ = struct.unpack_from('<7Q', frame, 4 + name_bytes)
        boundaries.append(dict(node=node, incarnation=incarnation, decision=decision, trace=trace, order=order, time=time))
        offset += 4 + length
    assert offset == len(frames)
    selected = [boundaries[0], boundaries[len(boundaries) // 2], boundaries[-1]]
    go_artifact = None
    if args.go_binary or args.go_reference:
        assert args.go_binary and args.go_reference, 'Go comparison requires --go-binary and --go-reference'
        reference = json.loads(args.go_reference.read_bytes())
        row = next(row for row in reference['records'] if row['mode'] == 'exact' and row['case'] == 'match'
                   and row['seed'] == manifest['record']['seed'])
        go_artifact = Path(row['artifact'])
        go_manifest = json.loads((go_artifact / 'manifest.json').read_bytes())
        assert go_manifest['target']['sha256'].removeprefix('sha256:') == manifest['record']['identity']['target']
    for index, boundary in enumerate(selected):
        destination = args.work / f'debug-{index}'
        result = run(f'inspect-{index}', ['replay', f'--stop=choice={boundary["decision"]}',
                                         '--debug-output', destination, artifact])
        assert result.returncode == 0, (result.returncode, result.stderr)
        report = json.loads(result.stdout)
        assert report['status'] == 'inspected_prefix' and report['prefix_verified'] and report['witness_match'], report
        assert report['resolved'] == dict(instance=boundary['node'], incarnation=boundary['incarnation'],
                                          choice=boundary['decision'], trace=boundary['trace'],
                                          order=boundary['order'], time_nanos=boundary['time']), report
        assert report['prefix_cuts'] and report['artifact_id'] == manifest['record_sha256'], report
        assert stat.S_IMODE(destination.stat().st_mode) == 0o700
        debug = json.loads((destination / 'debug.json').read_bytes())
        assert debug['stacks'] and debug['instance'] == boundary['node'], debug
        assert not debug['truncated'], debug
        assert stat.S_IMODE((destination / 'debug.json').stat().st_mode) == 0o600
        assert gzip.decompress((destination / 'heap.pprof').read_bytes()), 'empty pprof protobuf'
        if go_artifact:
            go_destination = args.work / f'go-debug-{index}'
            go_result = run(f'go-inspect-{index}', ['--installation', args.go_reference.parent / 'exact-installation.json',
                                                  'replay', f'--stop=choice={boundary["decision"]}',
                                                  '--debug-output', go_destination, go_artifact], args.go_binary)
            assert go_result.returncode == 0, (go_result.returncode, go_result.stderr)
            go_report = json.loads(go_result.stdout)
            for field in ['status', 'prefix_verified', 'witness_match', 'resolved', 'prefix_cuts', 'truncated']:
                assert report[field] == go_report[field], (index, field, report[field], go_report[field])
            assert debug == json.loads((go_destination / 'debug.json').read_bytes()), (index, 'debug semantics')
            assert gzip.decompress((destination / 'heap.pprof').read_bytes()) == gzip.decompress((go_destination / 'heap.pprof').read_bytes()), (index, 'heap profile semantics')
    for name, flags in [('unknown-dump', ['--stop=choice=0', '--dump=missing']), ('unknown-probe', ['--stop=choice=0', '--probe=missing/*']),
                         ('unknown-check', ['--check=missing', '--choice-range=0:0'])]:
        result = run(name, ['replay', *flags, artifact])
        assert result.returncode == 2 and b'unknown or unavailable' in result.stderr, (name, result.stderr)
    result = run('time-coordinate', ['replay', '--stop=time=0', '--dump=stacks',
                                    '--debug-output', args.work / 'debug-time', artifact])
    assert result.returncode == 0 and json.loads(result.stdout)['resolved']['choice'] == 0, result.stderr
    print('replay inspection: fresh store stacks, heap, coordinates and retained registry refusals')
    if args.duration_edges:
        result = run('duration-maximum', ['replay', '--stop=time=9223372036854775807ns', artifact])
        assert result.returncode == 2 and b'outside retained inspectable boundaries' in result.stderr, result.stderr
        result = run('duration-negative-zero', ['replay', '--stop=time=-0s', '--dump=stacks',
                                               '--debug-output', args.work / 'debug-negative-zero', artifact])
        assert result.returncode == 0 and json.loads(result.stdout)['resolved']['choice'] == 0, result.stderr
        print('replay inspection: exact whole duration bounds and signed zero')
    registrations = metadata['incarnations'][0]['registry']['entries']
    if any(entry['name'] == 'fixture.before_three' for entry in registrations):
        assert any(boundary['trace'] > boundary['decision'] for boundary in boundaries), 'registered fixture must distinguish trace and choice coordinates'
        last = boundaries[-1]['decision']

        def registered(name, flags, status=0, diagnostic=False):
            destination = args.work / ('registered-' + name)
            selected_flags = [*flags, *(['--debug-output', destination] if diagnostic else [])]
            result = run('registered-' + name, ['replay', *selected_flags, artifact])
            assert result.returncode == status, (name, result.returncode, result.stderr)
            report = json.loads(result.stdout) if result.stdout else None
            debug = json.loads((destination / 'debug.json').read_bytes()) if diagnostic and report else None
            if go_artifact:
                go_destination = args.work / ('go-registered-' + name)
                go_flags = [*flags, *(['--debug-output', go_destination] if diagnostic else [])]
                other = run('go-registered-' + name, ['--installation', args.go_reference.parent / 'exact-installation.json',
                                                     'replay', *go_flags, go_artifact], args.go_binary)
                assert other.returncode == status, (name, other.returncode, other.stderr)
                if report:
                    go_report = json.loads(other.stdout)
                    for field in ['status', 'prefix_verified', 'witness_match', 'resolved', 'prefix_cuts', 'truncated', 'search']:
                        assert report.get(field) == go_report.get(field), (name, field, report.get(field), go_report.get(field))
                    if diagnostic:
                        assert debug == json.loads((go_destination / 'debug.json').read_bytes()), (name, 'registered diagnostic semantics')
            return report, debug

        report, debug = registered('dumps', ['--stop=failure', '--dump=fixture.snapshot', '--dump=fixture.panic',
                                            '--dump=fixture.truncated', '--dump-bytes=128'], diagnostic=True)
        assert report['truncated'] and [dump['status'] for dump in debug['dumps']] == ['ok', 'panic', 'truncated'], debug
        assert b'progress observed' in base64.b64decode(debug['dumps'][0]['data_base64'])
        assert len(base64.b64decode(debug['dumps'][2]['data_base64'])) == 128
        report, debug = registered('probes', [f'--stop=choice={last}', '--dump=fixture.snapshot',
                                             '--probe=fixture.progress', '--probe=fixture.oversized'], diagnostic=True)
        assert len(debug['probes']) == 8 and len(debug['drops']) == 8, debug
        assert [probe['fields'][0]['uint64'] for probe in debug['probes']] == list(range(1, 9)), debug
        scan, _ = registered('scan', ['--check=fixture.before_three', f'--choice-range=0:{last}',
                                      '--search-mode=scan', '--search-attempts=256', '--search-timeout=30s'], 1)
        assert scan['search']['status'] == 'first_failure' and scan['search']['first_failure'] > 0, scan
        bisect, _ = registered('bisect', ['--check=fixture.before_three', f'--choice-range=0:{last}',
                                          '--search-mode=bisect', '--search-attempts=256', '--search-timeout=30s'], 1)
        assert bisect['search']['first_failure'] == scan['search']['first_failure'], (scan, bisect)
        limited, _ = registered('attempt-bound', ['--check=fixture.before_three', f'--choice-range=0:{last}',
                                                  '--search-attempts=1', '--search-timeout=30s'], 1)
        assert limited['search']['status'] == 'incomplete' and len(limited['search']['evaluations']) == 1, limited
        panic, _ = registered('panic-check', ['--check=fixture.panic_check', '--choice-range=0:0'], 1)
        assert panic['search']['status'] == 'invariant_evaluation_failure' and panic['search']['evaluations'][0]['detail'] == 'panic', panic
        refused, _ = registered('nonmonotonic-bisect', ['--check=fixture.recovering', f'--choice-range=0:{last}', '--search-mode=bisect'], 2)
        assert refused is None
        report, debug = registered('frozen-gc', [f'--stop=choice={last}', '--dump=fixture.gc'], diagnostic=True)
        assert debug['dumps'][0]['status'] == 'ok' and b'collection completed' in base64.b64decode(debug['dumps'][0]['data_base64']), debug
        print('replay inspection: registered dumps/probes, trace offsets, scan/bisect, bounds and predicate failure')
