#!/usr/bin/env python3
"""Fresh frozen-Go campaign reference; never substitute retained historical views."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import signal
import shutil
import struct
import subprocess
import sys
import tarfile
import tempfile
import time

sys.dont_write_bytecode = True


def equal(actual, expected, label):
    if actual != expected:
        raise ValueError(f'{label}: got {actual!r}, expected {expected!r}')


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def crc_table():
    table = []
    for value in range(256):
        for _ in range(8):
            value = (value >> 1) ^ (0x82f63b78 if value & 1 else 0)
        table.append(value)
    return table


CRC_TABLE = crc_table()


def crc32c(raw):
    value = 0xffffffff
    for byte in raw:
        value = (value >> 8) ^ CRC_TABLE[(value ^ byte) & 255]
    return value ^ 0xffffffff


def frames(root, incomplete=False):
    """Validate the frozen codec's CRC32C framing, lengths, order and raw JSON."""
    raw = (root / 'store/log').read_bytes()
    result, offset = [], 0
    bounds = {1: (1 << 20) + (64 << 10), 2: 64 << 10, 3: 1 << 20,
              4: 1 << 30, 5: 16 << 20, 6: 64 << 20, 7: 64 << 20,
              16: 16 << 20, 18: 16 << 20, 19: 16 << 20}
    while offset < len(raw):
        header = raw[offset:offset + 16]
        if len(header) < 16 and incomplete:
            break
        equal(len(header), 16, 'go.log.header_length')
        equal(header[:6], b'GMLG\x01\x00', 'go.log.magic_version_flags')
        kind, length, checksum = struct.unpack('<HII', header[6:])
        equal(crc32c(header[:12]), checksum, 'go.log.header_crc32c')
        if kind not in bounds or length > bounds[kind]:
            raise ValueError('unknown Go log kind or oversized payload')
        end = offset + 16 + length
        if end + 4 > len(raw) and incomplete:
            break
        equal(len(raw[offset:end + 4]), length + 20, 'go.log.frame_length')
        observed_crc = struct.unpack('<I', raw[end:end + 4])[0]
        if incomplete and end + 4 == len(raw) and crc32c(raw[offset:end]) != observed_crc:
            break
        equal(crc32c(raw[offset:end]), observed_crc, 'go.log.payload_crc32c')
        payload = raw[offset + 16:end]
        value = json.loads(payload)
        result.append(dict(offset=offset, kind=kind, payload=value, payload_raw=payload, payload_sha256=digest(payload)))
        offset = end + 4
    if not incomplete:
        equal(offset, len(raw), 'go.log.complete_boundary')
    return result


def contained(root, relative):
    path = root / relative
    if Path(relative).is_absolute() or '..' in Path(relative).parts or not path.resolve().is_relative_to(root.resolve()):
        raise ValueError('Go evidence reference escapes campaign')
    cursor = root
    for part in Path(relative).parts:
        cursor /= part
        if cursor.is_symlink():
            raise ValueError('Go evidence symlink')
    return path


def authenticate_artifact(root):
    manifest = json.loads((root / 'manifest.json').read_bytes())
    for entry in manifest['files']:
        payload = contained(root, entry['path']).read_bytes()
        equal((len(payload), digest(payload)), (int(entry['size']), entry['sha256'].removeprefix('sha256:')), 'go.artifact.' + entry['path'])
    target = manifest['target']
    equal(digest(contained(root, target['file']).read_bytes()), target['sha256'].removeprefix('sha256:'), 'go.artifact.target')
    return manifest


def authenticate_campaign(root, incomplete=False):
    rows = frames(root, incomplete)
    equal(rows[0]['kind'] in (1, 16), True, 'go.log.envelope_kind')
    equal(rows[0]['payload']['format'] in ('gomad.campaign-log/v1', 'gomad.merged-campaign-log/v1'), True, 'go.log.lineage')
    executions, pending, receipts = [], {}, {}
    selected, parallel, shard = [], 1, None
    published = False
    next_ordinal = 0
    artifact_root = root
    sources = {int(row['index']): row for row in rows[0]['payload'].get('shards', [])}
    source_digest = None
    source_count = 0
    source_metadata = None

    def check_source():
        if source_metadata is not None:
            equal('sha256:' + source_digest.hexdigest(), source_metadata['log_sha256'], 'go.merge.source_log_sha256')
            equal(source_count, int(source_metadata['records']), 'go.merge.source_records')

    def admit(jobs):
        nonlocal next_ordinal
        for job in jobs:
            index, seed = int(job['ordinal']), int(job['seed'])
            equal(index, next_ordinal, 'go.admission.order')
            equal(seed, selected[index], 'go.admission.seed')
            if index in pending:
                raise ValueError('duplicate Go admission')
            pending[index] = seed
            next_ordinal += int(shard['count']) if shard else 1
        if len(pending) > parallel:
            raise ValueError('Go admission exceeds slots')

    for row in rows:
        value, kind = row['payload'], row['kind']
        if kind in (18, 19):
            check_source()
            source_metadata = None
            source_digest = None
            source_count = 0
            if kind == 18:
                source_metadata = sources[int(value['index'])]
                artifact_root = Path(source_metadata['path'])
                source_digest = hashlib.sha256()
        elif source_digest is not None:
            source_digest.update(struct.pack('>HQ', kind, len(row['payload_raw'])))
            source_digest.update(row['payload_raw'])
            source_count += 1
        if kind == 1:
            if pending or receipts:
                raise ValueError('Go source boundary has outstanding work')
            selected = []
            for term in value['selection'].split(','):
                ends = list(map(int, term.split('-')))
                selected.extend(range(ends[0], ends[-1] + 1))
            equal(len(selected), int(value['selection_count']), 'go.selection_count')
            parallel = int(value.get('parallel', 1))
            shard = value.get('shard')
            next_ordinal = int(shard['index']) if shard else 0
        elif kind == 6:
            admit(value['jobs'])
        elif kind == 7:
            execution = value['observation']
            index = int(execution['selection_ordinal'])
            equal(int(execution['seed']), pending[index], 'go.receipt.seed')
            if index in receipts:
                raise ValueError('duplicate Go receipt')
            receipts[index] = execution
        elif kind == 4:
            for execution in value['executions']:
                index = int(execution['selection_ordinal'])
                equal(index < len(selected), True, 'go.execution.bound')
                equal(int(execution['seed']), selected[index], 'go.execution.seed')
                if pending:
                    equal(index, min(pending), 'go.outcome.order')
                    equal(int(execution['seed']), pending.pop(index), 'go.outcome.admitted')
                    receipt = receipts.pop(index)
                    for field in ('seed', 'domain', 'reason', 'termination', 'failure_signature'):
                        equal(execution[field], receipt[field], 'go.outcome.receipt.' + field)
                reference = execution.get('artifact') or execution.get('success_artifact')
                if reference:
                    path = Path(reference)
                    if path.is_absolute():
                        if not path.resolve().is_relative_to(artifact_root.resolve()):
                            raise ValueError('Go execution artifact outside campaign')
                    else:
                        path = contained(artifact_root, reference)
                    manifest = authenticate_artifact(path)
                    if execution.get('failure_signature'):
                        equal(manifest['outcome']['failure_signature'], execution['failure_signature'], 'go.artifact.failure_reference')
                executions.append(execution)
            admit(value.get('admit', []))
        elif kind == 5:
            equal((pending, receipts), ({}, {}), 'go.publish.pending')
            published = True
    check_source()
    if not incomplete:
        equal(published or rows[0]['kind'] == 16, True, 'go.campaign.published')
    indices = [int(item['selection_ordinal']) for item in executions]
    equal(len(indices), len(set(indices)), 'go.execution.unique_ordinals')
    return dict(records=executions, frames=rows, complete=published)


def semantic(records):
    groups = {}
    result = []
    for item in sorted(records, key=lambda row: int(row['selection_ordinal'])):
        domain = item['domain']
        failure = domain != 'success'
        signature = item.get('failure_signature')
        group = None
        if failure:
            if not signature:
                raise ValueError('failure lacks grouping identity')
            group = groups.setdefault(signature, int(item['selection_ordinal']))
        result.append(dict(ordinal=int(item['selection_ordinal']), seed=int(item['seed']),
            outcome='success' if not failure else 'watchdog' if item['termination'] == 'watchdog_timeout' else 'target_failure',
            termination=item['termination'], failure_group=group))
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('baseline', 'source', 'go-compiler', 'work', 'preparation-root'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    for name in ('baseline', 'source', 'go_compiler', 'work', 'preparation_root'):
        setattr(args, name, getattr(args, name).resolve())
    args.work.mkdir(parents=True, mode=0o700, exist_ok=False)
    inventory = (args.baseline / 'inputs.sha256').read_bytes()
    for line in inventory.decode().splitlines():
        expected, relative = line.split()
        equal(digest((args.baseline / relative).read_bytes()), expected, 'baseline.' + relative)
    equal(digest(args.go_compiler.read_bytes()), '1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64', 'pinned.compiler')
    archive = args.baseline / 'source-1d7b37f00a.tar.gz'
    with tarfile.open(archive) as source:
        for member in source.getmembers():
            parts = Path(member.name).parts
            if member.isfile() and parts[1:3] == ('tools', 'gomad'):
                equal(digest((args.source / Path(*parts[1:])).read_bytes()), digest(source.extractfile(member).read()), 'frozen.source.' + member.name)
    binary = args.baseline / 'bin/gomad'
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_') and key not in ('GOROOT', 'GOFLAGS', 'GOMOD', 'GOEXPERIMENT')}
    commands = []

    def receipt(name, command, result, expected):
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        commands.append(dict(name=name, argv=list(map(str, command)), status=result.returncode, expected=expected,
                             stdout_sha256=digest(result.stdout), stderr_sha256=digest(result.stderr)))
        (args.work / 'commands.json').write_text(json.dumps(commands, indent=2) + '\n')
        equal(result.returncode, expected, name + ': ' + result.stderr.decode(errors='replace'))
        return result.stdout

    def run(name, command, expected=0, input_bytes=None, cwd=None):
        result = subprocess.run(list(map(str, command)), env=environment, capture_output=True, timeout=600, input=input_bytes, cwd=cwd)
        return receipt(name, command, result, expected)

    template = Path(__file__).parent / 'reference/campaign.go.txt'
    wrapper_root = Path(tempfile.mkdtemp(prefix='campaign-reference-', dir=args.source / 'tools/gomad'))
    wrapper = args.work / 'go-campaign-api-reference'
    try:
        (wrapper_root / 'main.go').write_bytes(template.read_bytes())
        environment.update(GOENV='off', GOTOOLCHAIN='local', GOWORK='off', CGO_ENABLED='0', GOPROXY='off', GOSUMDB='off', GOCACHE=str(args.preparation_root / 'wrapper-go-cache'))
        run('build-frozen-api-reference', [args.go_compiler, 'build', '-mod=readonly', '-trimpath', '-o', wrapper, '.'], cwd=wrapper_root)
    finally:
        shutil.rmtree(wrapper_root)

    declaration = json.loads((Path(__file__).parent / 'campaign-corpus.json').read_bytes())
    observations = {}
    for case in declaration['cases']:
        name = case['name']
        installation = args.work / (name + '-installation.json')
        run(name + '-install', [binary, 'install', '--compiler', args.go_compiler, '--helper', args.baseline / 'bin/gomad-wasmhost',
            '--profile', 'gomad.wasi-cooperative/v1', '--runtime-root', args.source / 'tools/gomad/internal/model/runtimesource',
            '--cache', args.preparation_root / name, '--fuel', str(1 << 60), '--output', installation])
        prefix = [binary, '--installation', installation]
        flags = ['--json', '--choices', '--parallel=1', '--seeds=' + declaration['seeds'], '--on-failure=all',
            '--keep-successes=all', '--success-limit=8', '--success-bytes=256MiB', '--execution-timeout=30s', '--overall-timeout=10m',
            '--working-dir', args.source / 'tools/gomad' / case['module'], *['--build-tag=' + tag for tag in case['tags']]]
        target = [case['kind'], case['source'], '--', *case['arguments']]

        def locate(destination):
            roots = sorted(path.parent.parent for path in destination.rglob('store/log'))
            equal(len(roots), 1, name + '.campaign_root')
            return roots[0]

        destination = args.work / (name + '-explore')
        run(name + '-explore', [*prefix, 'explore', *flags, '--artifacts', destination, *target], case['status'])
        explored = authenticate_campaign(locate(destination))
        plan = args.work / (name + '-plan')
        run(name + '-public-plan-refusal', [*prefix, 'plan', *flags, '--output', plan, *target], 2)
        request = dict(Operation='plan', Installation=str(installation), WorkingDir=str(args.source / 'tools/gomad' / case['module']),
            Kind=case['kind'], Source=case['source'], Arguments=case['arguments'], BuildTags=case['tags'], Output=str(plan), Artifacts=str(args.work / (name + '-plan-artifacts')))
        run(name + '-runner-api-plan', [wrapper], input_bytes=json.dumps(request).encode())
        shard_roots = []
        shard_records = []
        for index in range(2):
            destination = args.work / f'{name}-shard-{index}'
            run(f'{name}-public-shard-{index}-refusal', [*prefix, 'execute-shard', '--json', '--shard', f'{index}/2', '--artifacts', destination, plan], 2)
            run(f'{name}-runner-api-shard-{index}', [wrapper], case['status'], input_bytes=json.dumps(dict(Operation='execute-shard', Installation=str(installation), Plan=str(plan), Index=index, Count=2, Artifacts=str(destination))).encode())
            root = locate(destination)
            shard_roots.append(root)
            shard_records.extend(authenticate_campaign(root)['records'])
        merged = args.work / (name + '-merged')
        run(name + '-public-merge-refusal', [*prefix, 'merge', '--json', '--output', merged, plan, *shard_roots], 2)
        run(name + '-runner-api-merge', [wrapper], input_bytes=json.dumps(dict(Operation='merge', Installation=str(installation), Plan=str(plan), Shards=list(map(str, shard_roots)), Output=str(merged))).encode())
        merged_records = authenticate_campaign(merged)['records']
        equal(semantic(shard_records), semantic(explored['records']), name + '.shards_semantics')
        equal(semantic(merged_records), semantic(explored['records']), name + '.merge_semantics')

        destination = args.work / (name + '-interrupted')
        command = [*prefix, 'explore', *flags, '--artifacts', destination, *target]
        stdout_path, stderr_path = args.work / (name + '-interrupt.stdout'), args.work / (name + '-interrupt.stderr')
        killed = False
        with stdout_path.open('wb') as stdout, stderr_path.open('wb') as stderr:
            child = subprocess.Popen(list(map(str, command)), env=environment, stdout=stdout, stderr=stderr, start_new_session=True)
            try:
                deadline = time.monotonic() + 600
                while child.poll() is None and time.monotonic() < deadline:
                    roots = list(destination.rglob('store/log')) if destination.exists() else []
                    if len(roots) == 1:
                        root = roots[0].parent.parent
                        observed = frames(root, incomplete=True)
                        if any(row['kind'] == 4 for row in observed) and not any(row['kind'] == 5 for row in observed):
                            with roots[0].open('rb') as stream:
                                os.fsync(stream.fileno())
                            os.killpg(child.pid, signal.SIGKILL)
                            killed = True
                            break
                    time.sleep(0.001)
            finally:
                if child.poll() is None:
                    os.killpg(child.pid, signal.SIGKILL)
                status = child.wait(timeout=60)
        equal(killed, True, name + '.durable_interrupt_observed')
        receipt(name + '-interrupt', command, subprocess.CompletedProcess(command, status, stdout_path.read_bytes(), stderr_path.read_bytes()), -signal.SIGKILL)
        interrupted = locate(destination)
        prefix_records = authenticate_campaign(interrupted, incomplete=True)['records']
        equal(len(prefix_records) > 0, True, name + '.interrupted_prefix')
        run(name + '-resume', [*prefix, 'resume', '--json', interrupted], case['status'])
        resumed = authenticate_campaign(interrupted)
        equal(semantic(resumed['records']), semantic(explored['records']), name + '.resume_semantics')
        observations[name] = dict(explore=semantic(explored['records']), resume=semantic(resumed['records']),
            execute_shard=semantic(shard_records), merge=semantic(merged_records))
    result = dict(schema='gomad.campaign-go-reference/v1', surface={'explore': 'public CLI', 'resume': 'public CLI', 'plan': 'original Runner API; public WASM CLI refuses', 'execute_shard': 'original Runner API; public WASM CLI refuses', 'merge': 'original Runner API; public WASM CLI refuses'}, baseline_inputs_sha256=digest(inventory),
        binary_sha256=digest(binary.read_bytes()), compiler_sha256=digest(args.go_compiler.read_bytes()),
        source_sha256=digest(archive.read_bytes()), driver_sha256=digest(Path(__file__).read_bytes()), template_sha256=digest(template.read_bytes()), wrapper_sha256=digest(wrapper.read_bytes()), observations=observations)
    (args.work / 'campaign-reference.json').write_text(json.dumps(result, indent=2) + '\n')
    print('fresh Go public explore/resume and original Runner API shard/merge authenticated; public WASM shard/merge refusals verified')


if __name__ == '__main__':
    main()
