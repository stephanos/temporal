"""Original runtime gate over the ordinary release library and retained readers."""
import hashlib
import json
from pathlib import Path
import time
import struct


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def choice_summary(raw, terminal, require):
    require(raw and len(raw) % 96 == 0 and len(raw) <= 1 << 20, 'malformed runtime choice record framing')
    counts = [0, 0, 0]
    branching, pending, parsed = 0, [], []
    for ordinal in range(len(raw)//96):
        record = raw[ordinal*96:(ordinal+1)*96]
        position, kind, flags, ready, alternatives, selected, data, site = struct.unpack_from('>QBBHIIIQ', record)
        observed, missing = bool(flags & 2), bool(flags & 4)
        require(position == ordinal and 1 <= kind <= 4 and not flags & ~7 and alternatives > selected
                and (not missing or site == 0) and observed == (kind == 3)
                and flags & 3 == (0 if kind == 4 else 2 if observed else 1), 'malformed runtime choice record')
        origin = int.from_bytes(record[32:40], 'big') if observed else 0
        if kind == 4:
            require(2 <= alternatives <= 256 and ready == data == 0 and any(record[32:64])
                    and not any(record[64:]) and selected == len(pending), 'malformed runtime creation metadata')
            pending.append((alternatives, record[32:64]))
        elif observed:
            require(not any(record[40:]) and origin <= ordinal and ready & 1
                    and ready >> 6 <= alternatives - bool(ready & 2), 'malformed runtime select observation')
        else:
            require(ready == 0 and any(record[32:64]) and any(record[64:]), 'malformed runtime decision')
        if kind != 4 and pending:
            require(kind == 1 and len(pending) == alternatives and all(count == alternatives for count, _ in pending)
                    and all(left[1] < right[1] for left, right in zip(pending, pending[1:])),
                    'malformed runtime creation group')
            enabled = b'gomad-choice-alternative-set/v1\0' + alternatives.to_bytes(8, 'big') + b''.join(value for _, value in pending)
            require(hashlib.sha256(enabled).digest() == record[64:], 'runtime creation enabled-set digest mismatch')
            pending.clear()
        if kind <= 3:
            counts[kind-1] += 1
        if kind in (1, 2) and alternatives >= 2:
            branching += 1
        parsed.append((kind, site, missing, alternatives, data, origin, ready))
    require(not pending, 'unterminated runtime creation metadata')
    origins = set()
    for ordinal, (kind, site, missing, alternatives, data, origin, ready) in enumerate(parsed):
        if kind == 3 and data >= 2:
            require(data-1 <= ordinal-origin, 'runtime select origin exceeds trace')
            for step in range(data-1):
                poll = origin+step
                require(parsed[poll][:4] == (2, site, missing, step+2) and poll not in origins,
                        'runtime select poll/result binding mismatch')
                origins.add(poll)
    require(len(terminal) == 16 and int.from_bytes(terminal[:4], 'big') == 1
            and int.from_bytes(terminal[4:8], 'big') > 0, 'malformed runtime finish terminal')
    return dict(records=len(parsed), branching_records=branching, runnable=counts[0], select_poll=counts[1],
        select_result=counts[2], terminal_state='complete', peak_goroutines=int.from_bytes(terminal[4:8], 'big'))


def exact_configuration(configuration, require):
    require(configuration.get('world_transition_bytes') == 64 << 20, 'changed runtime World transition byte bound')
    require(configuration['arguments'] == ['guest', 'literal $arg'], 'changed runtime argv')
    require(configuration['environment'] == dict(PWD='/workspace', TZ='UTC', ONLY='captured', EXTRA_ENTROPY='no_'),
            'changed runtime environment')
    require(configuration['profile'] == 'gomad.wasi-cooperative/v1'
            and configuration['engine'] == 'cranelift-fuel-nan-canonical-nounwind-epoch-v1'
            and not configuration.get('derive_seed_environment', False)
            and configuration.get('storage') is None and configuration.get('network') is None,
            'changed runtime profile/engine/model ownership')
    require(configuration['limits'] == dict(fuel=4_000_000_000, memory_bytes=256 << 20,
            wall_millis=60000, output_bytes=1 << 20, evidence_bytes=128 << 20), 'changed runtime execution limits')
    require(configuration.get('wall_nanos') is None, 'changed runtime exact deadline')
    require(configuration['model'] == dict(entropy_key=[1, 2, 3] + [0]*29,
            working_directory_sha256=digest(b'/workspace'), arguments=dict(bytes=0, count=0, chunks=0),
            environment=dict(bytes=0, count=0, chunks=0), mode=1, limit=1 << 20, diagnostics=True,
            epoch_nanos=946684800000000000, read_step_nanos=1000, now=0,
            limits=dict(calls=10000, transcript_bytes=8 << 20, pending_events=4096)), 'changed runtime model configuration')
    require(configuration['namespace'] == dict(entries=[], writable_directories=[list(b'/workspace'), list(b'/tmp')],
            preopens=[list(b'/tmp'), list(b'/workspace')], working_directory=list(b'/workspace'), stdin=[],
            epoch_nanos=946684800000000000,
            limits=dict(files=1000, descriptors=64, filesystem_bytes=8 << 20, output_bytes=1 << 20)),
            'changed runtime namespace configuration')


def harness(config, root, entry, api, name='harness-identity'):
    require, read = api['require'], api['read']
    executable, receipt = entry['harness'], entry['harness_build_receipt']
    require(executable in config['files'] and receipt in config['files'], 'unsealed runtime release harness')
    build = read(receipt)
    require(build['profile'] == 'release' and build['features'] == [] and build['exit'] == 0
            and build['binary_sha256'] == api['sha'](executable) and build['role'] == config['identity'],
            'runtime harness is not the same ordinary release library')
    identity = json.loads(api['command'](config, root, name, [executable, 'identity']))
    installation = read(config['installations'][entry['installation']])
    require(identity == dict(host_sha256=config['identity']['host'], helper_sha256=config['identity']['helper'],
            runtime_implementation_sha256=installation['runtime_implementation_sha256'],
            engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1', release_role=True),
            'runtime harness linked identities changed')
    require(installation['go_version'] == 'go1.27.1', 'runtime compiler version changed')
    return executable, identity


def observation(config, root, name, artifact, index, linked, api, timeout=None):
    require, read, payload = api['require'], api['read'], api['payload']
    inspected = json.loads(api['command'](config, root, name,
        api['public'](config) + ['inspect', '--json', '--choices', str(artifact)], timeout=timeout))
    if 'runtime_artifact' in api:
        api['runtime_artifact'](Path(artifact), inspected, index)
    record = inspected['artifact']['record']
    require(record['seed'] == index*1000003, 'runtime artifact seed changed')
    require(record['identity']['implementation'] == digest(json.dumps(config['identity'], sort_keys=True,
            separators=(',', ':')).encode()), 'runtime implementation mismatch')
    inputs = json.loads(payload(artifact, inspected, 'evidence/replay.json'))
    for field in ('host_sha256', 'helper_sha256', 'runtime_implementation_sha256'):
        require(inputs[field] == linked[field], 'runtime retained component identity changed: ' + field)
    exact_configuration(inputs['configuration'], require)
    provenance = json.loads(payload(artifact, inspected, 'target/provenance.json'))
    source = api['REPO'] / 'tools/gomad/internal/backends/wasm/testdata/runtime_probes'
    require(provenance['inputs']['kind'] == 'go-run' and provenance['inputs']['source'] == '.'
            and provenance['inputs']['directory'] == str(source)
            and provenance['inputs']['sources'].get(str(source / 'main.go')) == api['sha'](source / 'main.go')
            and provenance['module_sha256'] == record['identity']['target']
            and provenance['build_key'] == record['identity']['toolchain'], 'runtime original guest preparation changed')
    installation = read(config['installations'][config['stages'][f'runtime-{index}']['installation']])
    require(provenance['inputs']['compiler_sha256'] == installation['compiler_sha256']
            and provenance['inputs']['go_version'] == 'go1.27.1'
            and provenance['inputs']['environment']['GOEXPERIMENT'] == 'nogreenteagc',
            'runtime retained compiler/collector changed')
    stdout = payload(artifact, inspected, 'output/stdout')
    diagnostic = payload(artifact, inspected, 'evidence/diagnostic')
    terminal = payload(artifact, inspected, 'evidence/runtime-terminal')
    choice = inspected['artifact']['choices']
    summary = choice_summary(payload(artifact, inspected, 'evidence/choice'), terminal, require)
    require(all(choice[key] == value for key, value in summary.items() if key != 'peak_goroutines'),
            'runtime retained inspection choice summary changed')
    witness = api['runtime_witness'](record, inspected, stdout, diagnostic, terminal)
    witness['summary'] = summary
    baseline = dict(outcome=record['outcome'], streams=record['streams'], summary=summary,
        terminal_sha256=digest(terminal), diagnostic_sha256=digest(diagnostic))
    return record, baseline, witness, stdout


def replay_observation(root, artifact, record, linked, api):
    require, read = api['require'], api['read']
    replay = read(root / 'recordings/replay/report.json')
    require(replay['schema'] == 'gomad.runtime-qualification-replay/v1'
            and replay['diagnostics'] is False and replay['exact'] is True and replay['matched'] is True
            and replay['parent_record_sha256'] == read(Path(artifact) / 'manifest.json')['record_sha256'],
            'runtime forced full-tape diagnostics-off replay mismatch')
    require(replay['outcome']['termination'] == 'exit'
            and replay['outcome']['exit_code'] == record['outcome']['exit_code'], 'runtime replay outcome changed')
    for field in ('host_sha256', 'helper_sha256', 'runtime_implementation_sha256'):
        require(replay[field] == linked[field], 'runtime replay linked identity changed')
    for name in ('stdout', 'stderr', 'transcript', 'choice'):
        path = ('output/' if name in ('stdout', 'stderr') else 'evidence/') + name
        expected = record['streams'].get(path, {}).get('sha256', digest(b''))
        require(digest((root / 'recordings/replay' / name).read_bytes()) == replay['streams'][name] == expected,
                'runtime replay raw payload changed: ' + name)
    return replay


def runtime(config, root, label, entry, api):
    require, read = api['require'], api['read']
    started = time.monotonic()
    def remaining():
        duration = 600 - (time.monotonic() - started)
        require(duration > 0, 'original runtime ten-minute shard exceeded')
        return duration
    index = int(label.split('-')[1])
    require(0 <= index < 32, 'runtime index must be 0..31')
    require(entry['source'] == './internal/backends/wasm/testdata/runtime_probes', 'original runtime guest required')
    require(entry['arguments'] == ['--working-dir', str(api['REPO'] / 'tools/gomad')], 'changed runtime target arguments')
    installation = config['installations'][entry['installation']]
    api['runtime_installation'](read(installation))
    executable, linked = harness(config, root, entry, api)
    installed = read(installation)
    require(installed['admission']['host_executable_sha256'] == config['binary_sha256'],
            'runtime operator installation executable mismatch')
    policy = root / 'guest-config.json'
    api['write'](policy, installed['admission']['guest_configuration'])
    runtime_installation = str(root / 'runtime-installation.json')
    api['command'](config, root, 'install-harness', [executable, 'install',
        '--compiler', installed['compiler_path'], '--runtime-root', installed['runtime_root'],
        '--cache', installed['cache_root'], '--go-version', 'go1.27.1', '--profile', installed['profile'],
        '--fuel', str(installed['fuel']), '--memory-bytes', str(installed['memory_bytes']),
        '--config', str(policy), '--output', runtime_installation], timeout=remaining())
    api['command'](config, root, 'producer', [executable, 'produce', runtime_installation,
        str(api['REPO'] / 'tools/gomad/internal/backends/wasm/testdata/runtime_probes'), str(index),
        str(root / 'recordings')], timeout=remaining(), private_observations=config)
    paths = read(root / 'recordings/artifacts.json')
    require(len(paths) == len(set(paths)) == 25, 'runtime requires 25 distinct fresh retained stores')
    baseline = None
    for number, artifact in enumerate(paths):
        require(Path(artifact).parent == root / 'recordings' / f'instance-{number:02}',
                'runtime artifact borrowed from another instance')
        record, observed, witness, stdout = observation(config, root, f'inspect-{number:02}', artifact, index, linked, api,
                                                       timeout=remaining())
        if baseline is None:
            baseline, first_record = observed, record
        require(observed == baseline, 'runtime fresh instance diverged')
    replay_observation(root, paths[0], first_record, linked, api)
    remaining()
    api['write'](root / 'runtime.json', dict(index=index, seed=index*1000003, instances=25, replays=1,
        source=config['source_sha256'], identity=config['identity'], artifacts=paths, linked=linked,
        harness_sha256=api['sha'](executable), witness=witness, baseline=baseline, stdout=stdout.decode()))
    return api['passed'](label)


def runtime_aggregate(config, root, label, api):
    require, read = api['require'], api['read']
    choices, outputs, exits = set(), set(), set()
    domains = {name: set() for name in ('map ', 'runq ', 'timer callbacks ', 'canary ')}
    first, all_paths = None, set()
    for index in range(32):
        stage = Path(config['root']) / 'stages' / f'runtime-{index}'
        report = read(stage / 'runtime.json')
        require(report['index'] == index and report['seed'] == index*1000003 and report['instances'] == 25
                and report['replays'] == 1 and report['source'] == config['source_sha256']
                and report['identity'] == config['identity'] and len(report['artifacts']) == len(set(report['artifacts'])) == 25,
                'stale runtime shard')
        require(not all_paths.intersection(report['artifacts']), 'runtime shards borrowed artifacts')
        all_paths.update(report['artifacts'])
        executable, linked = harness(config, root, config['stages'][f'runtime-{index}'], api, f'harness-identity-{index}')
        require(report['linked'] == linked and report['harness_sha256'] == api['sha'](executable), 'stale runtime harness')
        for number, artifact in enumerate(report['artifacts']):
            require(Path(artifact).parent == stage / 'recordings' / f'instance-{number:02}', 'runtime artifact borrowed')
            record, baseline, witness, stdout = observation(config, root, f'shard-{index}-{number}', artifact, index, linked, api)
            require(baseline == report['baseline'] and witness == report['witness'] and stdout.decode() == report['stdout'],
                    'runtime aggregate payload/summary/witness changed')
            if first is None:
                first = record['identity']
            require(record['identity'] == first, 'runtime shards are not one frozen candidate')
        replay_observation(stage, report['artifacts'][0], record, linked, api)
        choices.add(witness['choice']); outputs.add(witness['stdout']); exits.add(witness['exit'])
        for prefix, values in domains.items():
            found = [line for line in stdout.decode().splitlines() if line.startswith(prefix)]
            require(found, 'missing aggregate runtime domain: ' + prefix)
            values.update(found)
    require(len(choices) >= 2 and len(outputs) >= 2 and exits == {0, 17} and all(len(values) >= 2 for values in domains.values()),
            'runtime representative seeds lack choice/output/exit/domain variation')
    api['write'](root / 'aggregate.json', dict(choices=len(choices), outputs=len(outputs), exits=sorted(exits),
        domains={key: len(values) for key, values in domains.items()}))
    return api['passed'](label)
