#!/usr/bin/env python3
"""Execute transferred WASI roots 29–31 against unchanged compiled Go guests."""
import argparse
import base64
import copy
import hashlib
import json
from pathlib import Path
import struct
import subprocess
import time


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def records(raw):
    assert len(raw) % 96 == 0, 'runtime records must be whole wire records'
    return [bytearray(raw[index:index + 96]) for index in range(0, len(raw), 96)]


def exact(left, right):
    for name in ('stdout', 'choice', 'transcript'):
        assert left['streams'].get(name, b'') == right['streams'].get(name, b''), name


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'compiler', 'choices-module', 'runtime-module', 'inspection-module', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--runtime-identity', required=True, help='actual BuildOverlay identity of these guest modules')
    parser.add_argument('--root', choices=['29', '30', '31', 'all'], default='all')
    args = parser.parse_args()
    args.work = args.work.resolve()
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    identity = args.runtime_identity
    assert len(identity) == 64 and int(identity, 16) != 0
    commands = []

    def request(module, diagnostics=True):
        module = module.resolve()
        digest = sha(module.read_bytes())
        return dict(module_path=str(module), module_sha256=digest,
            choice_identity=dict(TargetSHA256=list(bytes.fromhex(digest)), ToolchainBuildKey=identity,
                GOOS='wasip1', GOARCH='wasm', ImplementationSHA256=list(bytes.fromhex(sha(identity.encode())))),
            memory_bytes=256 << 20, fuel=4000000000, wall_millis=60000,
            output_bytes=1 << 20, evidence_bytes=128 << 20,
            arguments=['guest', 'literal $arg'], environment=['PWD=/workspace', 'TZ=UTC', 'ONLY=captured'],
            model=dict(entropy_key=[1, 2, 3] + [0] * 29, working_directory_sha256=sha(b'/workspace'),
                arguments=dict(bytes=0, count=0, chunks=0), environment=dict(bytes=0, count=0, chunks=0),
                seed=7, mode=1, limit=1 << 20, diagnostics=diagnostics,
                epoch_nanos=946684800000000000, read_step_nanos=1000, now=0,
                limits=dict(calls=10000, transcript_bytes=8 << 20, pending_events=4096)),
            namespace=dict(entries=[], writable_directories=[list(b'/workspace'), list(b'/tmp')],
                preopens=[list(b'/tmp'), list(b'/workspace')], working_directory=list(b'/workspace'), stdin=[],
                limits=dict(files=1000, descriptors=64, filesystem_bytes=8 << 20, output_bytes=1 << 20),
                epoch_nanos=946684800000000000))

    def run(name, specification, expected='exit'):
        input_path = args.work / (name + '.input.json')
        input_path.write_text(json.dumps(specification, sort_keys=True) + '\n')
        destination = args.work / name
        command = [str(args.binary.resolve()), 'fixture-live-runtime', '--root', str(destination), '--input', str(input_path)]
        started = time.monotonic()
        result = subprocess.run(command, capture_output=True, timeout=180)
        commands.append(dict(name=name, argv=command, status=result.returncode, elapsed_seconds=time.monotonic() - started,
            stdout=result.stdout.decode(), stderr=result.stderr.decode()))
        (args.work / 'commands.json').write_text(json.dumps(commands, indent=2) + '\n')
        assert result.returncode == 0, (name, result.returncode, result.stderr)
        report = json.loads((destination / 'result.json').read_bytes())
        report['directory'] = destination
        report['streams'] = {path.name: path.read_bytes() for path in (destination / 'streams').iterdir()}
        assert report['outcome']['termination'] == expected, (name, report['outcome'])
        assert report['execution_returned'], 'fresh linked store and watchdog must be joined before return'
        return report

    def replay(specification, source, prefix=None):
        changed = copy.deepcopy(specification)
        changed['replay_trace_path'] = str(source['directory'] / 'streams/choice')
        changed['model']['mode'] = 3 if prefix is not None else 2
        if prefix is not None:
            changed['prefix'] = prefix
        return changed

    if args.root in ('30', 'all'):
        original = request(args.choices_module)
        observed = run('choices-record', original)
        assert observed['outcome']['exit_code'] == 0
        assert observed['choices']['runnable'] and observed['choices']['select_poll']
        assert len(observed['streams']['diagnostic']) // 96 == observed['choices']['records']
        assert b'timer' in observed['streams']['stdout']
        forced = replay(original, observed)
        forced['model']['diagnostics'] = False
        exact(observed, run('choices-exact-diagnostics-off', forced))
        for name, reason in [('missing', 6), ('extra', 7), ('kind', 1), ('site', 2), ('enabled', 5)]:
            trace = records(observed['streams']['choice'])
            target = next(index for index, record in enumerate(trace) if record[8] == 1)
            if name == 'missing':
                trace = []
            elif name == 'extra':
                trace.append(bytearray(trace[target]))
                struct.pack_into('>Q', trace[-1], 0, len(trace) - 1)
            else:
                if name in ('kind', 'enabled'):
                    start = target
                    while start > 0 and trace[start - 1][8] == 4:
                        start -= 1
                    removed = target - start
                    del trace[start:target]
                    for index, record in enumerate(trace):
                        struct.pack_into('>Q', record, 0, index)
                        if record[8] == 3 and struct.unpack_from('>Q', record, 32)[0] >= target:
                            struct.pack_into('>Q', record, 32, struct.unpack_from('>Q', record, 32)[0] - removed)
                    target = start
                if name == 'kind':
                    trace[target][8] = 2
                elif name == 'site':
                    trace[target][9] &= ~4
                    struct.pack_into('>Q', trace[target], 24, struct.unpack_from('>Q', trace[target], 24)[0] + 1)
                else:
                    trace[target][64:96] = bytes.fromhex(sha(b'inapplicable alternatives'))
            mutated = args.work / ('mutated-' + name + '.bin')
            mutated.write_bytes(b''.join(trace))
            changed = copy.deepcopy(forced)
            changed['replay_trace_path'] = str(mutated)
            result = run('choices-' + name, changed, 'divergence')
            assert result['outcome']['divergence']['Reason'] == reason, (name, result['outcome'])

    if args.root in ('31', 'all'):
        original = request(args.runtime_module)
        original['environment'].append('EXTRA_ENTROPY=no_')
        base = run('runtime-record', original)
        assert base['outcome']['exit_code'] in (0, 17)
        for witness in (b'entropy ', b'map ', b'runq ', b'timer callbacks ', b'timer parking',
                        b'note negative park wake', b'collector automatic finalizer cleanup 1 1', b'canary '):
            assert witness in base['streams']['stdout'], witness
        diagnostic = records(base['streams']['diagnostic'])
        altered = copy.deepcopy(diagnostic)
        for record in altered:
            struct.pack_into('>Q', record, 64, struct.unpack_from('>Q', record, 64)[0] + 1)
        mutated = args.work / 'altered-diagnostic.bin'
        mutated.write_bytes(b''.join(altered))
        changed = copy.deepcopy(original)
        changed['compare_diagnostic_path'] = str(mutated)
        difference = run('runtime-diagnostic-divergence', changed)['diagnostic_difference']
        assert difference['ordinal'] == 0 and 'runtime_rand_draws' in difference['fields'], difference
        frozen = copy.deepcopy(original)
        frozen['model']['read_step_nanos'] = 0
        frozen['model']['now'] = 1
        parked = run('runtime-frozen-clock-parking', frozen)
        for witness in (b'timer parking', b'note negative park wake', b'collector automatic finalizer cleanup 1 1'):
            assert witness in parked['streams']['stdout'], witness
        first, separator, rest = base['streams']['stdout'].partition(b'\n')
        assert separator
        for name in ('application-draws', 'application-key', 'runtime-seed'):
            changed = copy.deepcopy(original)
            if name == 'application-draws':
                changed['environment'][-1] = 'EXTRA_ENTROPY=yes'
            elif name == 'application-key':
                changed['model']['entropy_key'][31] = 99
            else:
                changed['model']['seed'] += 1
            result = run('runtime-' + name, changed)
            changed_first, separator, changed_rest = result['streams']['stdout'].partition(b'\n')
            assert separator
            if name == 'runtime-seed':
                assert changed_first == first and changed_rest != rest
            else:
                if name == 'application-key':
                    assert changed_first != first
                assert changed_rest == rest and result['streams']['choice'] == base['streams']['choice']
                counters = records(result['streams']['diagnostic'])
                assert len(counters) == len(diagnostic)
                for index, (actual, expected) in enumerate(zip(counters, diagnostic)):
                    assert actual[40:88] == expected[40:88], (name, index, 'six seeded draw counters')
        last = base['choices']['replay_decisions'][-1]
        assert last['kind'] == 'select-poll' and last['alternatives'] == 2
        alternate = run('runtime-alternate-canary', replay(original, base,
            dict(cutoff=last['ordinal'] + 1, override=dict(ordinal=last['ordinal'], rank=1 - last['selected'], reconstruct=False))))
        assert alternate['outcome']['exit_code'] in (0, 17) and alternate['outcome']['exit_code'] != base['outcome']['exit_code']
        forced = replay(original, alternate)
        forced['model']['diagnostics'] = False
        repeated = run('runtime-alternate-exact-diagnostics-off', forced)
        assert repeated['outcome']['exit_code'] == alternate['outcome']['exit_code']
        exact(alternate, repeated)

    if args.root in ('29', 'all'):
        original = request(args.inspection_module, diagnostics=False)
        base = run('inspection-record', original)
        assert base['outcome']['exit_code'] == 1
        assert not base['streams'].get('debug') and not base['streams'].get('diagnostic')
        metadata = base['inspection']
        assert metadata['complete'] and metadata['incarnations'][0]['registry']['closed']
        forced = replay(original, base)
        repeated = run('inspection-exact', forced)
        for stream in ('choice', 'transcript', 'inspection'):
            assert repeated['streams'][stream] == base['streams'][stream], stream
        boundaries = metadata['boundaries']
        assert boundaries and any(boundary['trace'] > boundary['decision'] for boundary in boundaries)
        seen = set()

        def inspect(name, boundary, dumps=(), probes=(), expected='inspected_prefix', fuel=None, wall=None):
            changed = copy.deepcopy(forced)
            changed['inspection'] = dict(metadata_path=str(base['directory'] / 'streams/inspection'),
                decision=boundary['decision'], dumps=list(dumps), probes=list(probes))
            if fuel is not None:
                changed['fuel'] = fuel
            if wall is not None:
                changed['wall_millis'] = wall
            return run(name, changed, expected)

        wire = records(base['streams']['choice'])
        for ordinal, boundary in enumerate(boundaries):
            index = boundary['trace']
            while wire[index][8] == 4:
                index += 1
            kind = wire[index][8]
            assert boundary['decision'] == ordinal and wire[index][9] & 1
            seen.add(kind)
            stopped = inspect(f'inspection-kind{kind}-{ordinal}', boundary, ('stacks', 'heap'))
            assert stopped['prefix_verified'] and stopped['debug']['stacks']
            heap = stopped['directory'] / 'heap.pprof'
            assert heap.stat().st_size
            if ordinal == 0:
                command = [str(args.compiler.resolve()), 'tool', 'pprof', '-top', str(heap)]
                result = subprocess.run(command, capture_output=True, timeout=60)
                (args.work / 'heap-pprof.stdout').write_bytes(result.stdout)
                (args.work / 'heap-pprof.stderr').write_bytes(result.stderr)
                commands.append(dict(name='heap-pprof', argv=command, status=result.returncode,
                    stdout=result.stdout.decode(), stderr=result.stderr.decode()))
                (args.work / 'commands.json').write_text(json.dumps(commands, indent=2) + '\n')
                assert result.returncode == 0, result.stderr
        assert {1, 2} <= seen
        last = boundaries[-1]
        result = inspect('inspection-registered-dumps-frozen-GC', last,
            ('fixture.snapshot', 'fixture.panic', 'fixture.truncated', 'fixture.gc'))
        assert result['prefix_verified']
        dumps = result['debug']['dumps']
        assert [dump['status'] for dump in dumps] == ['ok', 'panic', 'truncated', 'ok']
        payload = base64.b64decode(dumps[0]['data_base64'])
        assert b'application lock is held' in payload and b'progress observed' in payload
        assert b'collection completed' in base64.b64decode(dumps[3]['data_base64'])
        result = inspect('inspection-allocation-free-probes', last, probes=('fixture.progress', 'fixture.oversized'))
        assert result['prefix_verified']
        probes, drops = result['debug']['probes'], result['debug']['drops']
        assert len(probes) == 8 and len(drops) == 8
        for index, probe in enumerate(probes):
            assert len(probe['fields']) == 3 and probe['fields'][0]['uint64'] == index + 1
            assert probe['decision'] <= last['decision'] and probe['trace'] <= last['trace']
        result = inspect('inspection-allocating-probe-diverges-without-diagnostics', last,
            probes=('fixture.allocating',), expected='instrumentation_divergence')
        assert result['streams'].get('stdout', b'') == base['streams'].get('stdout', b'')
        assert not result['streams'].get('diagnostic') and not result['streams'].get('debug')
        actual = result['inspection']['boundaries'][-1]['cuts'][0]['witness']
        expected = last['cuts'][0]['witness']
        assert actual['gc_cycles'] == expected['gc_cycles'] and actual['allocated_bytes'] > expected['allocated_bytes']
        pressure = inspect('inspection-fuel-pressure', last, ('fixture.timeout',), expected='probe_fuel_limit',
            fuel=original['fuel'] - base['outcome']['fuel_remaining'] + 1000000)
        assert pressure['outcome']['fuel_remaining'] == 0
        inspect('inspection-nonyielding-dump-reaped', last, ('fixture.timeout',),
            expected='watchdog_timeout', fuel=1 << 50, wall=10000)
    (args.work / 'passed.json').write_text(json.dumps(dict(roots=args.root, commands=len(commands))) + '\n')


if __name__ == '__main__':
    main()
