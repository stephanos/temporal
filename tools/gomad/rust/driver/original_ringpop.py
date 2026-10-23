#!/usr/bin/env python3
"""Fresh, explicitly observed original-source ringpop reference; never historical custody."""
import argparse
import base64
import hashlib
import os
from pathlib import Path
import shutil
import tarfile

from driver import HERE, Refusal, command, read, require, sha, write
import live
import receiving

REFERENCE = '92c69594f21ec09d82b716b04a8eed8352a8fdd4'
CAPTURE_SOURCE = 'tools/gomad/internal/backends/wasm/wasi/cluster.go'
CAPTURE_VARIABLE = 'GOMAD_ORIGINAL_RINGPOP_NETWORK'


def capture_patch(raw):
    require(CAPTURE_VARIABLE.encode() not in raw and b'\t"os"\n' not in raw, 'original capture already present')
    imports = b'\t"io"\n'
    final = b'\t\tsession.result.Network = evidence.Record\n'
    require(raw.count(imports) == raw.count(final) == 1, 'original raw-network capture anchors changed')
    capture = b'''\t\tif destination := os.Getenv("GOMAD_ORIGINAL_RINGPOP_NETWORK"); destination != "" {
\t\t\tfile, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}
\t\t\t_, writeErr := file.Write(session.networkWire)
\t\t\tif err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil {\n\t\t\t\treturn err\n\t\t\t}
\t\t}
'''
    return raw.replace(imports, imports + b'\t"os"\n').replace(final, capture + final)


def inventory(root):
    files, links = {}, {}
    for path in sorted(root.rglob('*')):
        name = str(path.relative_to(root))
        if path.is_symlink():
            require(path.resolve().is_relative_to(root.resolve()), 'original source symlink escapes archive')
            links[name] = os.readlink(path)
        elif path.is_file():
            files[name] = sha(path)
        else:
            require(path.is_dir(), 'original source contains nonregular input')
    require(files, 'empty original source')
    return dict(files=files, symlinks=links)


def archive_inventory(path):
    files, links = {}, {}
    with tarfile.open(path) as archive:
        for member in archive:
            require(not Path(member.name).is_absolute() and '..' not in Path(member.name).parts,
                    'unsafe original source archive member')
            require(member.name not in files and member.name not in links, 'duplicate original source archive member')
            if member.isfile():
                digest = hashlib.sha256()
                with archive.extractfile(member) as handle:
                    while chunk := handle.read(1 << 20): digest.update(chunk)
                files[member.name] = digest.hexdigest()
            elif member.issym():
                links[member.name] = member.linkname
            else:
                require(member.isdir(), 'unsupported original source archive member')
    return dict(files=files, symlinks=links)


def extract_archive(path, root):
    expected = archive_inventory(path)
    for target in expected['symlinks'].values():
        require(target and not Path(target).is_absolute() and '..' not in Path(target).parts,
                'unsafe original source symlink target')
    with tarfile.open(path) as handle:
        handle.extractall(root, filter='data')
    for name, target in expected['symlinks'].items():
        link = root / name
        require(link.is_symlink() and link.resolve().is_relative_to(root.resolve()),
                'original source symlink escaped extraction root')
        observed = os.readlink(link)
        if observed != target:
            require(target.startswith('./') and observed == target[2:],
                    'unexpected original source symlink normalization')
            link.unlink()
            link.symlink_to(target)
    actual = inventory(root)
    require(actual == expected, 'original archive extraction inventory mismatch')
    return actual


def build_argv(plan, root):
    module = root / 'source/tools/gomad'
    return {
        'archive': [plan['git'], '-C', plan['repo'], 'archive', '--format=tar',
                    '--output=' + str(root / 'source.tar'), REFERENCE],
        'host-build': [plan['compiler'], 'build', '-mod=readonly', '-p=1', '-trimpath',
                       '-o', str(root / 'gomad-original'), './cmd/gomad'],
        'helper-build': [plan['cargo'], '+1.99.1', 'build', '--release', '--locked', '--offline',
                         '-j1', '--manifest-path', str(module / 'internal/backends/wasm/wasmhost/Cargo.toml')],
    }


def prepare(plan, root):
    require(plan['schema'] == 'gomad.original-ringpop-build-plan/v1' and plan['reference_commit'] == REFERENCE,
            'wrong original ringpop reference source')
    for name in ('git', 'compiler', 'cargo'):
        path = receiving.regular(plan[name])
        require(plan['files'].get(str(path)) == sha(path), 'unsealed original reference build tool')
    require(not root.exists(), 'original reference preparation must be fresh')
    root.mkdir(mode=0o700, parents=True)
    archive = root / 'source.tar'
    argv = build_argv(plan, root)
    command(plan, root, 'archive', argv['archive'])
    source = root / 'source'
    source.mkdir(mode=0o700)
    before = extract_archive(archive, source)
    captured = source / CAPTURE_SOURCE
    original = captured.read_bytes()
    patched = capture_patch(original)
    captured.write_bytes(patched)
    after = inventory(source)
    require(set(before['files']) == set(after['files']) and before['symlinks'] == after['symlinks']
            and [name for name in before['files'] if before['files'][name] != after['files'][name]] == [CAPTURE_SOURCE],
            'reference capture changed more than its one completed-network observation')
    module = source / 'tools/gomad'
    build = dict(plan, repo=str(module), environment=dict(plan.get('environment', {}),
        GOCACHE=str(root / 'go-cache'), CARGO_TARGET_DIR=plan['cargo_target'], CARGO_INCREMENTAL='0'))
    binary, helper = root / 'gomad-original', root / 'gomad-wasmhost-original'
    command(build, root, 'host-build', argv['host-build'])
    command(build, root, 'helper-build', argv['helper-build'])
    shutil.copyfile(Path(plan['cargo_target']) / 'release/gomad-wasmhost', helper)
    helper.chmod(0o755)
    require(inventory(source) == after, 'original source changed during reference builds')
    inputs = dict(schema='gomad.original-ringpop-inputs/v1', reference_commit=REFERENCE,
        build_plan=plan, source_root=str(source), source_archive=str(archive), source_archive_sha256=sha(archive),
        source_inventory=after, original_source_inventory=before,
        capture=dict(path=CAPTURE_SOURCE, original_sha256=receiving.digest(original), sha256=receiving.digest(patched),
                     environment_variable=CAPTURE_VARIABLE, phase='after original network decoder accepted complete helper bytes'),
        binary=str(binary), binary_sha256=sha(binary), helper=str(helper), helper_sha256=sha(helper),
        compiler=plan['compiler'], compiler_sha256=sha(plan['compiler']),
        build_commands={name: dict(command_sha256=sha(root / (name + '.command.json')),
                                 process_sha256=sha(root / (name + '.process.json')),
                                 stdout_sha256=sha(root / (name + '.stdout')), stderr_sha256=sha(root / (name + '.stderr')))
                        for name in ('archive', 'host-build', 'helper-build')})
    write(root / 'inputs.json', inputs)
    return inputs


def sealed(config):
    entry = config['final_retained']['ringpop_original']
    live.sealed_input(config, entry['inputs'])
    inputs = read(entry['inputs'])
    require(inputs['schema'] == 'gomad.original-ringpop-inputs/v1' and inputs['reference_commit'] == REFERENCE,
            'wrong fresh original reference input lineage')
    for name in ('binary', 'helper', 'compiler', 'source_archive'):
        live.sealed_input(config, inputs[name])
        require(sha(inputs[name]) == inputs[name + '_sha256'], 'original reference executable/archive changed')
    base = Path(entry['inputs']).parent
    source = Path(inputs['source_root'])
    require(source == base / 'source' and inputs['source_archive'] == str(base / 'source.tar')
            and inputs['binary'] == str(base / 'gomad-original') and inputs['helper'] == str(base / 'gomad-wasmhost-original'),
            'original reference build outputs escaped receipt root')
    require(inventory(source) == inputs['source_inventory'], 'original source/capture inventory changed')
    for name, value in inputs['source_inventory']['files'].items():
        require(config['files'].get(str(source / name)) == value, 'original reference source not frozen')
    before, after = inputs['original_source_inventory'], inputs['source_inventory']
    require(before == archive_inventory(inputs['source_archive']), 'reference inventory differs from original archive')
    require(set(before['files']) == set(after['files']) and before['symlinks'] == after['symlinks']
            and [name for name in before['files'] if before['files'][name] != after['files'][name]] == [CAPTURE_SOURCE],
            'original reference contains an undeclared source change')
    captured = source / CAPTURE_SOURCE
    with tarfile.open(inputs['source_archive']) as archive:
        original = archive.extractfile(CAPTURE_SOURCE).read()
    require(capture_patch(original) == captured.read_bytes() and receiving.digest(original) == before['files'][CAPTURE_SOURCE],
            'original reference capture differs from exact archived source patch')
    require(inputs['capture'] == dict(path=CAPTURE_SOURCE, original_sha256=receiving.digest(original),
            sha256=sha(captured), environment_variable=CAPTURE_VARIABLE,
            phase='after original network decoder accepted complete helper bytes'), 'original observation patch custody changed')
    plan = inputs['build_plan']
    require(plan['schema'] == 'gomad.original-ringpop-build-plan/v1' and plan['reference_commit'] == REFERENCE,
            'original build plan lineage changed')
    for name in ('git', 'compiler', 'cargo'):
        live.sealed_input(config, plan[name])
        require(plan['files'].get(plan[name]) == sha(plan[name]), 'original build tool changed')
    require(plan['compiler'] == inputs['compiler'], 'original compiler binding changed')
    argv = build_argv(plan, base)
    require(set(inputs['build_commands']) == set(argv), 'missing original build commands')
    for name, seals in inputs['build_commands'].items():
        for suffix in ('command.json', 'process.json', 'stdout', 'stderr'):
            path = base / (name + '.' + suffix)
            live.sealed_input(config, path)
            require(sha(path) == seals[suffix.split('.')[0] + '_sha256'], 'original reference build receipt changed')
        row, process = read(base / (name + '.command.json')), read(base / (name + '.process.json'))
        live.completed_command(row, process, argv[name], plan['repo'] if name == 'archive' else str(source / 'tools/gomad'))
        expected_environment = plan.get('environment', {}) if name == 'archive' else dict(plan.get('environment', {}),
            GOCACHE=str(base / 'go-cache'), CARGO_TARGET_DIR=plan['cargo_target'], CARGO_INCREMENTAL='0')
        require(row.get('environment', {}) == expected_environment
                and row.get('temporary_paths', {}) == plan.get('temporary_paths', {}), 'original build environment changed')
    return inputs


def bounded(path, limit):
    path = receiving.regular(path)
    require(path.stat().st_size <= limit, 'original reference raw read bound exceeded')
    return path.read_bytes()


def extract(root, helper_sha256):
    manifests = list((root / 'artifacts').rglob('manifest.json'))
    require(len(manifests) == 1, 'original ringpop requires exactly one retained success')
    artifact = manifests[0].parent
    bounded(manifests[0], 8 << 20)
    manifest = read(manifests[0])
    require(manifest['artifact_kind'] == 'gomad.success/v1' and manifest['seed'] == '7',
            'original reference is not the actual seed7 success')
    files = {item['path']: item for item in manifest['files']}
    require(len(files) == len(manifest['files']), 'duplicate original artifact payload')
    for name, descriptor in files.items():
        path = artifact / name
        require(not Path(name).is_absolute() and '..' not in Path(name).parts, 'unsafe original artifact payload')
        require(len(bounded(path, 512 << 20)) == int(descriptor['size']) and 'sha256:' + sha(path) == descriptor['sha256'],
                'original artifact payload integrity mismatch')
    target = manifest['target']
    require(target['file'] in files and target['sha256'] == files[target['file']]['sha256'], 'original module identity mismatch')
    evidence = target['backend']['evidence']
    require(evidence['file'] in files and evidence['sha256'] == files[evidence['file']]['sha256'], 'original backend evidence identity mismatch')
    retained = read(artifact / evidence['file'])
    require(retained['schema'] == 'gomad.wasm-simulation-retained/v2', 'not original typed cluster evidence')
    record = receiving.decoded(base64.b64decode(retained['record'], validate=True))
    execution = record['wasm_execution']
    require(record['outcome'] == 'completed' and record['seed'] == 7
            and execution['virtual_elapsed_nanos'] == 120000000001, 'original ringpop duration/outcome changed')
    owners = execution['incarnations']
    require({(item['handle']['node'], item['handle']['incarnation']) for item in owners} == {('a', 1), ('b', 1), ('c', 1)}
            and len(owners) == 3 and all(not item['interrupted'] for item in owners), 'original ringpop topology incomplete')
    raw = {'world': base64.b64decode(execution['world'], validate=True), 'network': bounded(root / 'network.bin', 512 << 20)}
    require(raw['network'].startswith(b'GMNETEV1'), 'original reference did not capture actual helper network wire')
    for item in owners:
        require('sha256:' + item['module_sha256'] == target['sha256'], 'original owner module mismatch')
        require(item['helper_sha256'] == helper_sha256, 'original owner helper identity mismatch')
        trace = base64.b64decode(item['choice_trace'], validate=True)
        require(trace and len(trace) % 96 == 0, 'original owner lost raw runtime trace records')
        raw['choice/' + item['handle']['node']] = trace
    outputs = {node: b'' for node in ('a', 'b', 'c')}
    for item in record['outputs']:
        require(item['handle']['node'] in outputs and item['handle']['incarnation'] == 1
                and not item['truncated'] and item['discarded_bytes'] == 0, 'original output owner/truncation changed')
        value = base64.b64decode(item['bytes'], validate=True)
        require(len(value) == item['total_bytes'] == item['retained_bytes']
                and 'sha256:' + receiving.digest(value) == item['full_sha256'], 'original output integrity mismatch')
        if item['stream'] == 'stdout': outputs[item['handle']['node']] += value
    for number, node in enumerate(('a', 'b', 'c'), 1):
        for marker in (b'bootstrapped ', b'partitioned view ', b'converged '):
            require(marker + ('10.0.0.%d:7946' % number).encode() in outputs[node],
                    'original ringpop phase/owner output incomplete')
    return artifact, target['sha256'].removeprefix('sha256:'), raw


def collect(config, global_root, read_only=True):
    import final_retained
    inputs = sealed(config)
    root = global_root / 'original-ringpop'
    reader = receiving.Retained(config) if root.exists() else None
    require(reader is not None or not read_only, 'UNEXECUTED: missing fresh original ringpop reference')
    if reader is None:
        root.mkdir(mode=0o700)
        write(root / 'scenario.json', final_retained.scenario())
    require(read(root / 'scenario.json') == final_retained.scenario(), 'original reference scenario changed')
    install = root / 'installation.json'
    argv = [inputs['binary'], 'install', '--compiler', inputs['compiler'], '--helper', inputs['helper'],
        '--runtime-root', str(Path(inputs['source_root']) / 'tools/gomad/internal/model/runtimesource'),
        '--profile', 'gomad.wasi-cooperative/v1', '--config', config['final_retained']['guest_configs']['ringpop'],
        '--cache', str(root / 'preparation'), '--memory-bytes', str(1 << 30), '--fuel', str(1 << 50), '--output', str(install)]
    if reader is None:
        command(config, root, 'install', argv)
    else:
        reader.command(config, root, 'install', argv)
    argv = [inputs['binary'], '--installation', str(install), 'explore', '--json', '--seeds=7', '--parallel=1',
        '--execution-timeout=40m', '--overall-timeout=45m', '--choices', '--choice-bytes=67108864',
        '--output-limit=8388608', '--keep-successes=all', '--success-limit=1', '--success-bytes=4294967296',
        '--artifacts', str(root / 'artifacts'), '--working-dir', str(HERE.parents[1] / 'internal/backends/wasm/testdata/ringpop'),
        '--scenario', str(root / 'scenario.json'), 'go-run', '.']
    observed = dict(config, environment=dict(config.get('environment', {}), **{CAPTURE_VARIABLE: str(root / 'network.bin')}))
    if reader is None:
        command(observed, root, 'explore', argv, timeout=2700)
    else:
        receiving.Retained(observed).command(observed, root, 'explore', argv, timeout=2700)
    for name, expected in (('install', config), ('explore', observed)):
        row = read(root / (name + '.command.json'))
        require(row.get('environment', {}) == expected.get('environment', {})
                and row.get('temporary_paths', {}) == expected.get('temporary_paths', {}),
                'original reference command environment changed')
    artifact, module, raw = extract(root, inputs['helper_sha256'])
    argv = [inputs['binary'], 'inspect', '--json', str(artifact)]
    if reader is None:
        command(config, root, 'inspect', argv)
    else:
        row, process = read(root / 'inspect.command.json'), read(root / 'inspect.process.json')
        live.completed_command(row, process, argv, config['repo'])
        require(row['start_ns'] >= config['sent_epoch'] * 1_000_000_000, 'original inspect borrowed another send interval')
        require(row.get('environment', {}) == config.get('environment', {})
                and row.get('temporary_paths', {}) == config.get('temporary_paths', {}), 'original inspect environment changed')
    reported = receiving.decoded(bounded(root / 'inspect.stdout', 128 << 20))
    manifest = read(artifact / 'manifest.json')
    require(reported['schema'] == 'gomad.inspect/v8' and reported['kind'] == 'artifact' and reported['path'] == str(artifact)
            and reported['artifact']['record_hash'] == manifest['record_hash'] and reported['artifact']['seed'] == 7,
            'original public inspection differs from raw legacy artifact')
    events = [receiving.decoded(line) for line in bounded(root / 'explore.stdout', 8 << 20).splitlines()]
    results = [item for item in events if item['type'] == 'result']
    require(len(results) == 1 and results[0]['classification'] == 'success'
            and all(results[0][name] == 1 for name in ('selected', 'attempted', 'succeeded', 'retained_successes'))
            and results[0]['stop_reason'] == 'seeds_exhausted', 'original public campaign did not complete the one success')
    require([item['path'] for item in events if item['type'] == 'artifact'] == [str(artifact)],
            'original public campaign advertised another artifact')
    paths = {name: str(root / ('reference-' + name.replace('/', '-') + '.bin')) for name in raw}
    for name, value in raw.items():
        if reader is None:
            with Path(paths[name]).open('xb') as handle: handle.write(value)
        require(bounded(paths[name], 512 << 20) == value, 'fresh original raw reference changed after extraction')
    receipt = dict(schema='gomad.ringpop-original-reference/v1', fresh=True, reference_commit=REFERENCE,
        source_inputs_sha256=sha(config['final_retained']['ringpop_original']['inputs']), helper_sha256=inputs['helper_sha256'],
        module_sha256=module, seed=7, virtual_elapsed_nanos=120000000001, scenario=final_retained.scenario(),
        artifact=str(artifact), commands={name: {suffix: sha(root / (name + '.' + suffix))
                        for suffix in ('command.json', 'process.json', 'stdout', 'stderr')}
                  for name in ('install', 'explore', 'inspect')},
        files={name: dict(bytes=len(value), sha256=receiving.digest(value)) for name, value in raw.items()})
    if reader is None: write(root / 'reference.json', receipt)
    require(read(root / 'reference.json') == receipt, 'fresh original reference custody receipt changed')
    return dict(receipt=str(root / 'reference.json'), module_sha256=module, world=paths['world'], network=paths['network'],
                choices={node: paths['choice/' + node] for node in ('a', 'b', 'c')})


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--prepare-plan', type=Path, required=True)
    parser.add_argument('--root', type=Path, required=True)
    args = parser.parse_args()
    prepare(read(args.prepare_plan), args.root.resolve())


if __name__ == '__main__':
    main()
