#!/usr/bin/env python3
"""Exercise public replay admission and retained pinned-root reads."""
import argparse
import hashlib
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

sys.dont_write_bytecode = True


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--work', type=Path, required=True)
    args = parser.parse_args()
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    corpus = Path(__file__).resolve().parent
    loader = SourceFileLoader('behavioral_parity', str(corpus.parent.parent / 'scripts/behavioral-parity'))
    parity = module_from_spec(spec_from_loader(loader.name, loader))
    loader.exec_module(parity)
    receipts, failures = [], []

    def run(name, argv, status=0, reason=None, match=None):
        command = [str(args.binary), *map(str, argv)]
        marker = args.work / (name + '-engine-start')
        assert not marker.exists(), marker
        environment = dict(os.environ, GOMAD_FIXTURE_GUEST_START_FILE=str(marker.resolve()))
        result = subprocess.run(command, capture_output=True, text=True, timeout=60,
                                env=environment)
        entered = marker.read_text() if marker.exists() else None
        expect_entry = (status in (0, 1) and argv[0] in ('replay', 'fixture-replay-pinned')
                        and '--verify-only' not in argv)
        receipts.append(dict(name=name, argv=command, status=result.returncode,
                             expected=status, stdout=result.stdout, stderr=result.stderr,
                             engine_start_file=str(marker.resolve()),
                             engine_entry=entered, expected_engine_entry=expect_entry))
        (args.work / 'commands.json').write_text(json.dumps(receipts, indent=2) + '\n')
        if (result.returncode != status or reason and reason not in result.stderr
                or match and match not in result.stdout or status == 2 and
                (result.stdout or not result.stderr) or
                entered != ('linked execution entered\n' if expect_entry else None)):
            failures.append(name)
        return result

    produced = run('publish-replay', ['fixture-replay-execution', '--root', args.work / 'inputs',
                                     '--corpus', corpus])
    assert produced.returncode == 0, ('fresh replay producer', produced.stderr)
    bundle = json.loads((args.work / 'inputs/foundation-result.json').read_bytes())
    parity.authenticate(args.work / 'inputs', bundle)
    assert len(bundle['records']) == 6, 'incomplete replay corpus'
    sources = {}
    modeled_observed = None
    for index, row in enumerate(bundle['records']):
        path = args.work / 'inputs' / row['artifact']
        manifest = json.loads((path / 'manifest.json').read_bytes())
        profile = manifest['record']['profile']
        sources.setdefault(profile, path)
        inputs = json.loads((path / 'evidence/replay.json').read_bytes())
        if profile == 'gomad.wasi-stock-exploratory/v1' and inputs['configuration']['model']:
            modeled_observed = path
        module = path / 'target/module.wasm'
        assert module.stat().st_nlink > 1, 'producer lost intentional module-pool links'
        run(f'pool-verify-{index}', ['replay', '--verify-only', path])
        run(f'pool-replay-{index}', ['replay', path],
            status=0 if row['outcome'] == 'success' else 1, match='reproduced=true')
    source = sources['gomad.wasi-stock-exploratory/v1']
    exact = sources['gomad.wasi-cooperative/v1']
    assert modeled_observed is not None, 'missing modeled observed replay fixture'

    def clone(name, original=source):
        path = args.work / name
        shutil.copytree(original, path)
        return path

    def seal(path, payloads=None, change=None):
        manifest = json.loads((path / 'manifest.json').read_bytes())
        for name, raw in (payloads or {}).items():
            (path / name).write_bytes(raw)
            entry = dict(bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest())
            manifest['files'][name].update(entry)
            if name in manifest['record']['streams']:
                manifest['record']['streams'][name].update(entry)
        if change:
            change(manifest)
        raw = canonical(manifest['record'])
        (path / 'record.json').write_bytes(raw)
        manifest['record_sha256'] = hashlib.sha256(raw).hexdigest()
        manifest['files']['record.json'].update(bytes=len(raw), sha256=manifest['record_sha256'])
        (path / 'manifest.json').write_bytes(canonical(manifest))

    def refused(name, mutate, reason=None, original=source):
        path = clone(name, original)
        mutate(path)
        for suffix, options in [('verify', ['--verify-only']), ('execute', [])]:
            run(name + '-' + suffix, ['replay', *options, path], 2, reason)
        return path

    def namespace_limit(path, field, value):
        inputs = json.loads((path / 'evidence/replay.json').read_bytes())
        configuration = inputs['configuration']
        if field == 'epoch_nanos':
            configuration['namespace'][field] = value
            configuration['model'][field] = value
        else:
            configuration['namespace']['limits'][field] = value
        if field == 'output_bytes':
            configuration['limits']['output_bytes'] = value

        def identity(manifest):
            record = manifest['record']
            if field == 'output_bytes':
                record['limits']['output_bytes'] = value
            record['identity']['configuration'] = hashlib.sha256(canonical(configuration)).hexdigest()
            record['execution_identity'] = hashlib.sha256(canonical(record['identity'])).hexdigest()

        seal(path, {'evidence/replay.json': canonical(inputs)}, change=identity)

    for name, field, value in [('namespace-descriptors-minimum', 'descriptors', 4),
                               ('namespace-files-maximum', 'files', (1 << 20) + 1),
                               ('namespace-filesystem-maximum', 'filesystem_bytes', (1 << 30) + 1),
                               ('namespace-output-zero', 'output_bytes', 0),
                               ('namespace-epoch-maximum', 'epoch_nanos', 1 << 63)]:
        refused(name, lambda path, field=field, value=value: namespace_limit(path, field, value),
                original=modeled_observed)

    run('detached-copy', ['replay', clone('detached')], match='reproduced=true')
    link = args.work / 'root-link'
    link.symlink_to(source, target_is_directory=True)
    for suffix, options in [('verify', ['--verify-only']), ('execute', [])]:
        run('root-symlink-' + suffix, ['replay', *options, link], 2, 'type')
    run('root-substitution', ['fixture-root-substitution', clone('root-substitution')],
        2, 'changed while opening')
    run('pinned-path-replacement', ['fixture-replay-pinned', clone('pinned-path-replacement')],
        match='reproduced=true')
    for name, relative, mode in [('root-mode', '.', 0o755),
                                 ('directory-mode', 'evidence', 0o755),
                                 ('payload-mode', 'evidence/replay.json', 0o644),
                                 ('manifest-mode', 'manifest.json', 0o644)]:
        refused(name, lambda path, relative=relative, mode=mode: (path / relative).chmod(mode),
                'mode violation')
    refused('missing-module', lambda path: (path / 'target/module.wasm').unlink(), 'missing')
    refused('missing-replay', lambda path: (path / 'evidence/replay.json').unlink(), 'missing')
    refused('changed-replay', lambda path: (path / 'evidence/replay.json').write_bytes(b'x'), 'mismatch')

    def corrupt(path):
        file = path / 'evidence/replay.json'
        raw = file.read_bytes()
        file.write_bytes(b'x' + raw[1:])

    refused('same-size-digest-corruption', corrupt, 'mismatch')
    refused('payload-hardlink', lambda path: os.link(path / 'evidence/replay.json',
                                                   args.work / 'payload-hardlink-outside'), 'hard link')
    refused('module-untrusted-hardlink', lambda path: os.link(path / 'target/module.wasm',
                                                            args.work / 'module-hardlink-outside'), 'pool')

    def symlink(path):
        file = path / 'evidence/replay.json'
        file.unlink()
        file.symlink_to(source / 'evidence/replay.json')

    refused('payload-symlink', symlink, 'type')

    def directory_symlink(path):
        os.rename(path / 'evidence', path / 'evidence-original')
        (path / 'evidence').symlink_to(source / 'evidence', target_is_directory=True)

    refused('directory-symlink', directory_symlink)

    def fifo(path):
        file = path / 'evidence/replay.json'
        file.unlink()
        os.mkfifo(file, 0o600)

    refused('payload-fifo', fifo, 'type')
    refused('manifest-traversal', lambda path: seal(path, change=lambda manifest:
            manifest['files'].update({'../outside': manifest['files'].pop('evidence/replay.json')})),
            'unsafe payload path')
    refused('malformed-manifest-digest', lambda path: seal(path, change=lambda manifest:
            manifest['files']['evidence/replay.json'].update(sha256='z' * 64)))
    refused('noncanonical-replay', lambda path: seal(path, {'evidence/replay.json':
            b' ' + (path / 'evidence/replay.json').read_bytes()}), 'noncanonical')
    refused('malformed-replay-json', lambda path: seal(path, {'evidence/replay.json': b'{'}))

    def replay_identity(path, field):
        value = json.loads((path / 'evidence/replay.json').read_bytes())
        value[field] = '0' * 64
        seal(path, {'evidence/replay.json': canonical(value)})

    for field in ['helper_sha256', 'host_sha256', 'toolchain_sha256']:
        refused('changed-' + field, lambda path, field=field: replay_identity(path, field), 'identity mismatch')

    def oversized(path, name):
        size = (16 << 20) + 1
        file = path / name
        with file.open('wb') as handle:
            handle.truncate(size)
        digest = hashlib.sha256()
        with file.open('rb') as handle:
            while block := handle.read(1 << 20):
                digest.update(block)

        def change(manifest):
            entry = dict(bytes=size, sha256=digest.hexdigest())
            manifest['files'][name].update(entry)
            manifest['record']['streams'][name].update(entry)

        seal(path, change=change)

    refused('oversized-replay', lambda path: oversized(path, 'evidence/replay.json'), 'caller bound')
    refused('malformed-choice', lambda path: seal(path, {'evidence/choice': b'invalid-choice'}),
            original=exact)

    def choice_descriptor(path, field, value):
        details = json.loads((path / 'evidence/inspection.json').read_bytes())
        details['choices']['trace'][field] = value
        seal(path, {'evidence/inspection.json': canonical(details)})

    refused('choice-traversal', lambda path: choice_descriptor(path, 'file', '../outside'), original=exact)
    refused('choice-digest', lambda path: choice_descriptor(path, 'sha256', '0' * 64), original=exact)
    refused('choice-caller-bound', lambda path: choice_descriptor(path, 'limit', 1), original=exact)

    def choice_helper(path):
        details = json.loads((path / 'evidence/inspection.json').read_bytes())
        details['choices']['implementation_sha256'] = '0' * 64
        seal(path, {'evidence/inspection.json': canonical(details)})

    refused('choice-helper-identity', choice_helper, 'identity mismatch', original=exact)

    def semantic_choice(path):
        raw = bytearray((path / 'evidence/choice').read_bytes())
        assert len(raw) == 96, 'semantic admission case requires its one actual choice record'
        raw[12:16] = bytes(4)
        details = json.loads((path / 'evidence/inspection.json').read_bytes())
        details['choices']['trace'].update(bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest())
        seal(path, {'evidence/choice': raw, 'evidence/inspection.json': canonical(details)})

    refused('resealed-semantic-choice', semantic_choice, 'malformed_trace', original=exact)
    refused('noncanonical-inspection', lambda path: seal(path, {'evidence/inspection.json':
            b' ' + (path / 'evidence/inspection.json').read_bytes()}), original=exact)
    assert not failures, failures
    print('public replay fault matrix: authenticated admission and pinned reads passed')


if __name__ == '__main__':
    main()
