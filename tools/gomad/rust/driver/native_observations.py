"""Immediately seal private preparation sidecars from explicit native campaigns.

This captures raw commitments, not execution qualification or full Go projection.
Volatile observations never enter canonical Record, Store or plan identities.
"""
import hashlib
import json
import os
from pathlib import Path
import re
import stat

METADATA_LIMIT = 16 << 20
RAW_LIMIT = 128 << 20
MODULE_LIMIT = 512 << 20
RECEIPT_FIELDS = {'schema', 'owner_pid', 'host_sha256', 'helper_sha256', 'prepared_record_sha256',
                  'module_sha256', 'provenance_sha256', 'build_key', 'compiler_sha256', 'sources_sha256', 'files'}
PHASE_PREFIX = 'target/preparation-inputs/'


def require(value, reason):
    if not value:
        raise ValueError(reason)


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False, allow_nan=False).encode()


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def unique(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, 'duplicate private observation JSON key')
        result[key] = value
    return result


def decoded(raw):
    return json.loads(raw, object_pairs_hook=unique,
                      parse_constant=lambda value: (_ for _ in ()).throw(ValueError('nonfinite private observation JSON')))


def absolute(path):
    path = Path(path)
    require(path.is_absolute() and '..' not in path.parts and path.resolve(strict=True) == path,
            'noncanonical private observation path: ' + str(path))
    require(not any(parent.is_symlink() for parent in (path, *path.parents)), 'symlink in private observation path')
    return path


def snapshot(path, maximum, mode=None):
    path = absolute(path)
    descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_CLOEXEC)
    try:
        before = os.fstat(descriptor)
        require(stat.S_ISREG(before.st_mode) and 0 <= before.st_size <= maximum, 'private observation file bound/type')
        if mode is not None:
            require(stat.S_IMODE(before.st_mode) == mode and before.st_uid == os.geteuid(), 'private observation file ownership/mode')
        hasher, total = hashlib.sha256(), 0
        while chunk := os.read(descriptor, 1 << 20):
            total += len(chunk)
            require(total <= maximum, 'private observation grew over bound')
            hasher.update(chunk)
        after = os.fstat(descriptor)
        require((before.st_dev, before.st_ino, before.st_size, before.st_mtime_ns, before.st_ctime_ns)
                == (after.st_dev, after.st_ino, after.st_size, after.st_mtime_ns, after.st_ctime_ns)
                and total == before.st_size, 'private observation changed during capture')
        return dict(bytes=total, sha256=hasher.hexdigest())
    finally:
        os.close(descriptor)


def document(path, maximum=METADATA_LIMIT, mode=None, canonical_required=True):
    identity = snapshot(path, maximum, mode)
    raw = Path(path).read_bytes()
    require(len(raw) == identity['bytes'] and digest(raw) == identity['sha256'], 'private document changed during capture')
    value = decoded(raw)
    require(not canonical_required or raw == canonical(value), 'noncanonical private/native document')
    return value, identity


def child_path(root, name):
    relative = Path(name)
    require(isinstance(name, str) and name and not relative.is_absolute()
            and '..' not in relative.parts and str(relative) == name, 'unclean private/native relative reference')
    return absolute(root / relative)


def commitments(config, campaign):
    plan, _ = document(campaign / 'plan.json')
    metadata = plan.get('prepared', plan)
    artifact = child_path(campaign, metadata['prepared_artifact'])
    manifest, _ = document(artifact / 'manifest.json')
    require(manifest['schema'] == 'gomad.artifact-manifest/v2', 'foreign prepared manifest')
    record = manifest['record']
    record_hash = digest(canonical(record))
    stored_record, _ = document(artifact / 'record.json')
    require(manifest['record_sha256'] == record_hash and stored_record == record,
            'prepared record hash/bytes mismatch')
    require(record['identity']['implementation'] == digest(canonical(config['identity'])), 'foreign prepared implementation')
    module = snapshot(artifact / 'target/module.wasm', MODULE_LIMIT)
    provenance, provenance_file = document(artifact / 'target/provenance.json')
    for name, identity in [('target/module.wasm', module), ('target/provenance.json', provenance_file)]:
        entry = manifest['files'][name]
        require(entry['bytes'] == identity['bytes'] and entry['sha256'] == identity['sha256'], 'prepared payload identity mismatch')
    inputs = provenance['inputs']
    require(provenance['schema'] == 'gomad.rust-wasm-provenance/v1'
            and provenance['module_sha256'] == module['sha256'] and record['identity']['target'] == module['sha256']
            and inputs['host'] == config['identity']['host'] and inputs['helper'] == config['identity']['helper']
            and provenance['build_key'] == digest(canonical(inputs)), 'prepared provenance/candidate mismatch')
    return dict(prepared_record_sha256=record_hash, module_sha256=module['sha256'],
                provenance_sha256=provenance_file['sha256'], build_key=provenance['build_key'],
                compiler_sha256=digest(canonical(inputs['compiler'])), sources_sha256=digest(canonical(inputs['sources'])))


def closed_tree(root, wanted):
    present, directories_seen = set(), set()
    expected_directories = {str(parent) for name in wanted for parent in Path(name).parents if str(parent) != '.'}
    for directory, directories, names in os.walk(root, followlinks=False):
        require(not any((Path(directory) / child).is_symlink() for child in (*directories, *names)), 'private sidecar symlink')
        present.update(str((Path(directory) / child).relative_to(root)) for child in names)
        directories_seen.update(str((Path(directory) / child).relative_to(root)) for child in directories)
    require(present == wanted and directories_seen == expected_directories, 'private sidecar unlisted/missing payload')


def capture(config, command_pid, explicit_campaign_paths):
    require(type(command_pid) is int and command_pid > 0, 'private capture actual command PID required')
    require(config['binary'] in config['files'] and config['files'][config['binary']] == config['binary_sha256']
            and snapshot(config['binary'], MODULE_LIMIT)['sha256'] == config['binary_sha256'], 'private capture binary changed/unsealed')
    require(config['source_manifest'] in config['files'] and config['files'][config['source_manifest']] == config['source_sha256']
            and snapshot(config['source_manifest'], MODULE_LIMIT)['sha256'] == config['source_sha256'], 'private capture source changed/unsealed')
    require(explicit_campaign_paths and len(explicit_campaign_paths) == len(set(map(str, explicit_campaign_paths))),
            'empty/duplicate private campaign contexts')
    root = absolute(config['root'])
    campaigns = []
    for name in explicit_campaign_paths:
        campaign = absolute(name)
        require(campaign.is_relative_to(root) and campaign != root, 'private campaign outside owning attempt')
        sidecar = absolute(campaign / 'preparation-observations')
        info = sidecar.stat()
        require(sidecar.is_dir() and stat.S_IMODE(info.st_mode) == 0o700 and info.st_uid == os.geteuid(), 'private sidecar ownership/mode')
        receipt_path = sidecar / 'receipt.json'
        receipt, receipt_identity = document(receipt_path, mode=0o600)
        require(set(receipt) == RECEIPT_FIELDS and receipt['schema'] == 'gomad.native-preparation-observation-receipt/v1',
                'private native receipt fields/schema')
        require(type(receipt['owner_pid']) is int and receipt['owner_pid'] == command_pid
                and receipt['host_sha256'] == config['identity']['host']
                and receipt['helper_sha256'] == config['identity']['helper'], 'private receipt foreign owner/candidate')
        expected = commitments(config, campaign)
        require(all(receipt[key] == value for key, value in expected.items()), 'private receipt prepared commitments mismatch')
        table = receipt['files']
        require(isinstance(table, dict) and 2 <= len(table) <= 34, 'private raw file table bound')
        required = {PHASE_PREFIX + phase + '/metadata.json' for phase in ('compile', 'admission')}
        require(required <= set(table), 'private compile/admission metadata missing')
        blobs = 0
        files = {}
        total = 0
        for relative, identity in table.items():
            match = re.fullmatch(r'target/preparation-inputs/(compile|admission)/(metadata[.]json|[0-9a-f]{64}[.]bin)', relative)
            require(match and set(identity) == {'bytes', 'sha256'} and type(identity['bytes']) is int
                    and 0 <= identity['bytes'] <= RAW_LIMIT and re.fullmatch('[0-9a-f]{64}', identity['sha256']), 'private raw file name/identity')
            path = child_path(sidecar, relative)
            observed = snapshot(path, METADATA_LIMIT if match[2] == 'metadata.json' else RAW_LIMIT, 0o600)
            require(observed == identity, 'private raw payload identity mismatch')
            total += identity['bytes']
            require(total < 1 << 64, 'private raw byte sum overflow')
            if match[2] != 'metadata.json':
                blobs += 1
                require(match[2][:-4] == identity['sha256'], 'private raw content-address mismatch')
            files[str(path)] = identity['sha256']
        require(blobs <= 32, 'private raw blob count')
        for phase in ('compile', 'admission'):
            metadata, _ = document(sidecar / (PHASE_PREFIX + phase + '/metadata.json'), mode=0o600, canonical_required=False)
            require(metadata['schema'] == 'gomad.native-preparation-observations/v1' and metadata['phase'] == phase
                    and isinstance(metadata['commands'], list) and metadata['commands'], 'private phase metadata schema/commands')
            owners = {entry['owner_pid'] for entry in metadata['commands']}
            require(len(owners) == 1 and all(type(owner) is int and owner > 0 for owner in owners)
                    and (phase != 'admission' or owners == {command_pid}), 'private admission/phase foreign owner')
        closed_tree(sidecar, {'receipt.json', *table})
        require(snapshot(receipt_path, METADATA_LIMIT, 0o600) == receipt_identity, 'private receipt changed during capture')
        campaigns.append(dict(path=str(campaign), receipt_path=str(receipt_path),
                              receipt_sha256=receipt_identity['sha256'], files=files))
    return dict(source_sha256=config['source_sha256'], binary_sha256=config['binary_sha256'], campaigns=campaigns)


def option(argv, name):
    values = []
    for index, argument in enumerate(argv):
        if argument == name:
            require(index + 1 < len(argv), 'private selector missing option value')
            values.append(argv[index + 1])
        elif argument.startswith(name + '='):
            values.append(argument.split('=', 1)[1])
    require(len(values) == 1, 'private selector requires one ' + name)
    return values[0]


def from_command(config, command_pid, argv, stdout):
    if len(argv) > 1 and argv[1] == '--prepare-terminal-module':
        import terminal_module_receiving
        return terminal_module_receiving.capture(config, command_pid, argv, stdout)
    if len(argv) == 6 and argv[1] == 'produce':
        require(any(entry.get('harness') == argv[0] for entry in config['stages'].values())
                and config['files'].get(argv[0]) == snapshot(argv[0], MODULE_LIMIT)['sha256'],
                'private runtime capture is not the frozen release harness')
        return capture(config, command_pid, [Path(argv[5]) / 'preparation'])
    require(argv[0] == config['binary'], 'private capture is not the admitted native executable')
    index = 3 if len(argv) > 1 and argv[1] == '--installation' else 1
    require(index < len(argv) and argv[index] in ('plan', 'qualify', 'explore'), 'unsupported private native selector')
    verb, arguments = argv[index], argv[index + 1:]
    lines = stdout.splitlines()
    require(lines, 'private native command has no actual JSON response')
    response = decoded(lines[-1])
    if verb == 'qualify':
        spec, _ = document(option(arguments, '--spec'), canonical_required=False)
        workspace = absolute(spec['ArtifactRoot'])
        result_path = absolute(response['result_path'])
        require(result_path.is_relative_to(workspace), 'private Result outside declared workspace')
        result, _ = document(result_path, 64 << 20)
        require(result['schema'] == 'gomad.rust-qualification-result/v1'
                and result['manifest']['integrated_source_sha256'] == config['source_sha256']
                and result['observations']['operator_process_id'] == command_pid, 'private Result source/operator mismatch')
        report = result['report']
        repeat = result['manifest']['repeat']
        executions = report['executions']
        require(type(repeat) is int and 2 <= repeat <= 32 and report['repeat'] == repeat
                and report.get('failure') is None and len(executions) == repeat
                and [entry['iteration'] for entry in executions] == list(range(1, repeat + 1)),
                'private Result incomplete/reordered execution campaign set')
        paths = [entry['campaign_path'] for entry in executions]
    elif verb == 'plan':
        workspace = absolute(option(arguments, '--artifacts'))
        require(response['plan'] == option(arguments, '--output'), 'private plan output differs from actual argv')
        require(response['schema'] == 'gomad.plan-result/v1', 'private plan response schema')
        preparation, identity = document(response['preparation_receipt'], 32 << 20)
        require(preparation['schema'] == 'gomad.rust-preparation-receipt/v1'
                and identity['sha256'] == response['preparation_receipt_sha256'] and preparation['process_id'] == command_pid
                and preparation['campaign'] == response['campaign'], 'private preparation response/PID mismatch')
        paths = [response['campaign']]
    else:
        require(response['type'] == 'result', 'private explore response type')
        workspace = absolute(option(arguments, '--artifacts'))
        paths = [response['campaign_path']]
    require(paths and all(absolute(path).is_relative_to(workspace) for path in paths), 'private native campaign outside declared workspace')
    seal = capture(config, command_pid, paths)
    if verb == 'plan':
        expected = commitments(config, absolute(paths[0]))
        plan_identity = snapshot(response['plan'], METADATA_LIMIT)
        require(preparation['plan'] == response['plan']
                and preparation['plan_sha256'] == response['plan_sha256'] == plan_identity['sha256']
                and preparation['module_sha256'] == response['module_sha256'] == expected['module_sha256']
                and preparation['provenance_sha256'] == expected['provenance_sha256']
                and preparation['record_sha256'] == expected['prepared_record_sha256'], 'private plan/preparation commitments mismatch')
    return seal


def verify_snapshot(config, command_pid, seal):
    """Verify the already-sealed hashes, including after compiler/source removal.

    The caller must authenticate this seal inside the actual command witness;
    this function neither adopts a replacement seal nor grants execution credit.
    """
    require(set(seal) == {'source_sha256', 'binary_sha256', 'campaigns'}
            and seal['source_sha256'] == config['source_sha256']
            and seal['binary_sha256'] == config['binary_sha256'] and seal['campaigns'], 'foreign/empty private outer seal')
    seen = set()
    for campaign in seal['campaigns']:
        root = absolute(campaign['path'])
        require(str(root) not in seen and root.is_relative_to(Path(config['root'])), 'duplicate/foreign sealed campaign')
        seen.add(str(root))
        receipt_path = root / 'preparation-observations/receipt.json'
        require(campaign['receipt_path'] == str(receipt_path), 'changed sealed receipt path')
        receipt, identity = document(receipt_path, mode=0o600)
        require(identity['sha256'] == campaign['receipt_sha256'] and receipt['owner_pid'] == command_pid,
                'sealed private receipt hash/owner changed')
        expected = {str(child_path(receipt_path.parent, relative)): item['sha256'] for relative, item in receipt['files'].items()}
        require(expected == campaign['files'], 'sealed private raw file set changed')
        closed_tree(receipt_path.parent, {'receipt.json', *receipt['files']})
        for path, wanted in campaign['files'].items():
            require(snapshot(path, METADATA_LIMIT if path.endswith('/metadata.json') else RAW_LIMIT, 0o600)['sha256'] == wanted,
                    'sealed private raw bytes changed')
    return seal
