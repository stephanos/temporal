"""Execute and reopen the six immutable Go cluster wire roots; never native credit."""
import argparse
import json
from pathlib import Path
import sys

import final_profiles
import live
import native_roles
import original_wire_cache
import terminal_controls
from driver import Refusal, configured, read, require, sha, write

ROOTS = (
    'TestClusterTransportCodecCorpus', 'TestClusterTransportCodecCorpusIsCurrent',
    'TestClusterTransportCodecLiteralStageGuards', 'TestClusterTransportStageCorpus',
    'TestClusterTransportStageCorpusIsCurrent', 'TestClusterTransportStageLiteralGuards',
)
CASES = tuple(ROOTS[2] + '/' + name for name in (
    'raw-result-kind', 'target', 'incarnation', 'result-payload', 'current-tag-not-stale', 'payload-stage-error')) + tuple(
    ROOTS[5] + '/' + case + '/' + mutation
    for case in ('current-tag-reaches-payload', 'malformed-json-before-attribution')
    for mutation in ('wrong-stage', 'wrong-target', 'stale-input'))
FILES = (
    'tools/gomad/internal/backends/wasm/wasi/cluster.go',
    'tools/gomad/internal/backends/wasm/wasi/cluster_transport_codec_corpus_test.go',
    'tools/gomad/internal/backends/wasm/wasi/cluster_transport_stage_corpus_test.go',
    'tools/gomad/internal/backends/wasm/wasmhost/tests/testdata/cluster_transport_codec/restarted-a-incarnation-two-rejects-old-prefix.json',
    *('tools/gomad/internal/backends/wasm/wasi/testdata/cluster-transport-stage-conformance/' + name + '.json'
       for name in ('current-tag-reaches-payload', 'malformed-json-before-attribution')),
)
SCHEMA = 'gomad.original-cluster-wire-reference/v1'


def events(raw):
    observed = {name: [] for name in (*ROOTS, *CASES)}
    package = []
    for line in raw.splitlines():
        event = native_roles.decode(line)
        require(event.get('Package') == terminal_controls.PACKAGE, 'foreign original cluster wire package')
        action, test = event.get('Action'), event.get('Test')
        require(action not in ('fail', 'skip'), 'failed/skipped original cluster wire assertion')
        if test:
            require(test in observed, 'unexpected original cluster wire test')
            if action in ('run', 'pass'):
                observed[test].append(action)
            else:
                require(action == 'output', 'foreign original cluster wire event')
        elif action in ('start', 'pass'):
            package.append(action)
        else:
            require(action in ('output', 'build-output'), 'foreign original cluster wire package event')
    require(package == ['start', 'pass'] and all(value == ['run', 'pass'] for value in observed.values()),
            'missing/duplicate/unexecuted original cluster wire assertion')
    return list(observed)


def owned(config):
    return Path(config['root']) / 'broad-harness/cluster-wire-reference'


def input_paths(entry):
    import reference_profile
    derived = Path(entry['derived'])
    source = reference_profile.verify(derived)
    goroot = Path(entry['go_root'])
    require(Path(entry['go']) == goroot / 'bin/go', 'cluster reference compiler is outside its installation')
    paths = {entry['git'], str(derived / 'source.json'), str(derived / 'original.tar'),
             str(Path(__file__).parent / 'reference/main.go.txt')}
    paths.update(str(derived / 'source' / name) for name, item in source['original_files'].items()
                 if item['kind'] == 'file')
    paths.update(str(derived / 'source' / name) for name in source['glue'])
    paths.update(str(goroot / name) for name in live.inventory(goroot))
    paths.update(original_wire_cache.input_paths(entry))
    require(all(Path(path).is_absolute() and Path(path).is_file() and not Path(path).is_symlink() for path in paths),
            'missing/nonregular original cluster reference input')
    return paths


def collect(config, produce=False):
    root = owned(config)
    if produce:
        root.mkdir(mode=0o700)
    require(root.is_dir() and root.resolve() == root, 'missing/foreign original cluster wire root')
    entry = config['terminal_controls_reference']
    execute = terminal_controls.run if produce else terminal_controls.retained
    derived, source = final_profiles.original_source(config, root, entry, 'original-tree', execute)
    before = {name: sha(derived / 'source' / name) for name in FILES}
    module_cache = original_wire_cache.binding(config, entry)
    go, goroot = entry['go'], Path(entry['go_root'])
    live.sealed_input(config, str(goroot))
    require(Path(go) == goroot / 'bin/go', 'cluster wire reference requires admitted stock Go')
    scratch = root / 'scratch'
    if produce:
        scratch.mkdir(mode=0o700)
        (scratch / 'go-cache').mkdir(mode=0o700)
    require(scratch.is_dir() and not scratch.is_symlink(), 'missing original cluster wire scratch')
    environment = original_wire_cache.environment(goroot, scratch, module_cache['root'])
    environment.update(GOMAD_CLUSTER_TRANSPORT_CODEC_REGENERATE='', GOMAD_CLUSTER_TRANSPORT_STAGE_REGENERATE='')
    child = dict(config, repo=str(derived / 'source/tools/gomad/internal/backends/wasm'), temporary_paths={},
                 environment=environment)
    version, _ = execute(child, root, 'go-version', [go, 'version'])
    environment, _ = execute(child, root, 'go-environment', [go, 'env', '-json', 'GOROOT', 'GOVERSION', 'GOTOOLDIR'])
    compiler = terminal_controls.compiler_identity(goroot, version, environment)
    argv = [go, 'test', '-json', '-v', '-tags', 'test_dep', '-count=1', '-timeout=15m',
            '-run', '^(' + '|'.join(ROOTS) + ')$', './wasi']
    raw, process = execute(child, root, 'original-cluster-wire', argv)
    names = events(raw)
    require(before == {name: sha(derived / 'source' / name) for name in FILES}, 'original cluster wire source changed')
    # Reopen the entire original archive/tree after the command, not just the six selected files.
    import reference_profile
    require(reference_profile.verify(derived) == source, 'original cluster wire source closure changed')
    live.sealed_input(config, str(goroot))
    require(original_wire_cache.binding(config, entry) == module_cache, 'original cluster wire dependency cache changed')
    proof = dict(schema=SCHEMA, original_commit=source['original_commit'], testcases=names,
        original_source=str(derived / 'source.json'), original_source_sha256=sha(derived / 'source.json'),
        original_files=before, compiler=compiler, compiler_sha256=sha(go), pid=process['pid'], module_cache=module_cache,
        source_sha256=config['source_sha256'], config_sha256=config['_config_sha256'], identity=config['identity'],
        scope='Historical original Go wire tests only; no current native execution or terminal-claim credit.',
        commands={name: {kind: sha(root / (name + '.' + kind)) for kind in
            ('command.json', 'process.json', 'stdout', 'stderr')} for name in
            ('original-tree', 'go-version', 'go-environment', 'original-cluster-wire')})
    path = root / 'original-cluster-wire.json'
    if produce:
        write(path, proof)
    else:
        require(read(path) == proof, 'changed original cluster wire proof')
    return proof


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('produce', 'check'))
    parser.add_argument('--config', type=Path, required=True)
    args = parser.parse_args()
    try:
        config = configured(args.config)
        config['_config_path'] = str(args.config.resolve())
        config['_config_sha256'] = sha(args.config)
        print(json.dumps(collect(config, args.mode == 'produce'), separators=(',', ':')))
        return 0
    except (Refusal, OSError, ValueError, KeyError, TypeError) as error:
        print('REFUSED: ' + str(error), file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
