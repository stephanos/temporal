#!/usr/bin/env python3
"""Required fn233 callback acceptance, separate from the frozen rewrite ladder."""
import argparse
import contextlib
import io
import json
import shlex
from pathlib import Path

from driver import HERE, check_files, load_module, read, require, sha
import interception_streaming as streaming
import main_cli_build
import native_roles
import receiving

REQUIRED = ('controlled-compiled-go-marker-release-exit', 'closed-consumer-partial-lifecycle')
FIELDS = {'schema', 'main_build', 'harness_build', 'installation', 'streaming_root', 'closed_root', 'inputs', 'closed_build', 'closed_installation'}


def bindings(value):
    require(set(value) == FIELDS and value['schema'] == 'gomad.interception-callback-bindings/v1',
            'missing/foreign fn233 callback acceptance bindings')
    for key in FIELDS - {'schema'}:
        path = Path(value[key])
        require(path.is_absolute() and path.resolve() == path and not path.is_symlink(),
                'fn233 callback binding must be canonical: ' + key)
    require(value['streaming_root'] != value['closed_root'], 'fn233 callback proofs must be distinct')
    require(value['installation'] != value['closed_installation'], 'fn233 ordinary/fixture installations must be distinct')
    recipe = read(native_roles.regular(Path(value['streaming_root']) / 'recipe.json'))
    for key in ('main_build', 'harness_build', 'installation'):
        require(recipe[key] == value[key], 'fn233 callback proof borrowed another typed binding: ' + key)
    return recipe


def closed_context(value, config, main):
    fixture = main_cli_build.read_build(value['closed_build'], expected_role='host-fixture')
    require(all(fixture[key] == main[key] for key in ('head','source_sha256','symlinks',
                'helper_sha256','runtime_implementation_sha256')),
            'fn233 closed-consumer fixture is not the same source and ordinary helper candidate')
    require(fixture['host_sha256'] != main['host_sha256'] and fixture['binary'] != main['binary'],
            'fn233 closed-consumer fixture must retain its distinct host identity and executable')
    files = dict(config['files'])
    installation = read(native_roles.regular(value['closed_installation']))
    ordinary = read(native_roles.regular(value['installation']))
    require(all(installation[key] == ordinary[key] for key in
                ('compiler_path','compiler_sha256','go_version','profile','runtime_implementation_sha256')),
            'fn233 ordinary/fixture installations differ in compiler/profile/runtime')
    additions = dict(fixture['inputs'])
    additions.update(installation['admission']['compiler_inputs'])
    for name in ('closed_build','closed_installation'):
        additions[value[name]] = sha(value[name])
    for name,digest in additions.items():
        require(name not in files or files[name] == digest, 'fn233 callback candidate input conflict')
        files[name] = digest
    closed = dict(config,root=value['closed_root'],binary=fixture['binary'],
        binary_sha256=fixture['binary_sha256'],identity=dict(host=fixture['host_sha256'],helper=fixture['helper_sha256']),files=files)
    receiving.Retained(closed).installed(installation,installation['admission']['guest_configuration'],1 << 50,4 << 30,
        expected_build_tags=installation.get('build_tags',[]))
    require(installation['runtime_implementation_sha256'] == main['runtime_implementation_sha256'],
            'fn233 closed-consumer fixture installation runtime differs')
    check_files(files)
    return closed,fixture,installation


def preparation_lineage(value, checker, fixture, installation, closed):
    case = checker.private_json(Path(value['inputs']))['callback']
    receipt = checker.private_json(Path(case['receipt']))
    require(receipt['schema'] == 'gomad.rust-preparation-receipt/v1'
            and type(receipt['process_id']) is int and receipt['process_id'] > 0
            and type(receipt['guest_executions']) is int and type(receipt['exit_code']) is int
            and receipt['guest_executions'] == receipt['exit_code'] == 0
            and receipt['command'][:2] == [fixture['binary'],'run-guest'] and receipt['command'][-1:] == ['stream']
            and receipt['module_sha256'] == closed['module_sha256']
            and sha(case['receipt']) == case['receipt_sha256'] == closed['origin']['launch']['preparation_receipt_sha256'],
            'fn233 closed-consumer lacks its actual fixture Go-driver preparation receipt')
    invocation = closed['origin']['invocation']
    argv = [bytes.fromhex(argument).decode() for argument in invocation['driver_argv']]
    require(invocation['original_argv'] == invocation['driver_argv']
            and invocation['compiler_sha256'] == installation['compiler_sha256']
            and argv[:2] == [installation['compiler_path'],'run'] and argv.count('-exec') == 1
            and argv[-2:] == ['./cmd/gomad/testdata/seedfree','stream']
            and shlex.split(argv[argv.index('-exec')+1]) == [fixture['binary'],'run-guest'],
            'fn233 closed-consumer original Go driver is not the selected fixture launcher')


def closed_candidate(config, installation, observation):
    artifact = Path(observation['artifact'])
    retained = receiving.Retained(config)
    manifest = retained.artifact(artifact)
    provenance = read(native_roles.regular(artifact / 'target/provenance.json'))
    origin = observation['origin']
    require(set(origin) == {'schema','origin','invocation','module_sha256','preparation_sha256','launch'}
            and origin['schema'] == 'gomad.intercepted-origin/v1' and origin['origin'] == 'intercepted-go',
            'fn233 callback acceptance requires the original intercepted-origin schema')
    require(provenance.get('origin') == origin,
            'fn233 closed-consumer intercepted origin differs from the authenticated launch')
    # The shared qualification reader deliberately rejects interception extensions.
    # Only this already-authenticated origin is projected out for structural identity checks.
    # This borrowed view never rewrites evidence or supplies qualification credit.
    projected = {key:value for key,value in provenance.items() if key != 'origin'}
    retained.provenance(dict(provenance=projected))
    inputs = provenance['inputs']
    source = str(Path(config['repo']) / 'tools/gomad/cmd/gomad/testdata/seedfree/main.go')
    require(inputs['kind'] == 'go-run' and inputs['source'] == './cmd/gomad/testdata/seedfree'
            and inputs['directory'] == str(Path(config['repo']) / 'tools/gomad')
            and inputs['arguments'] == ['stream'] and inputs['go_version'] == 'go1.27.1'
            and inputs['tags'] == sorted(set(installation.get('build_tags', [])) | {'test_dep'})
            and source in config['files'] and inputs['sources'].get(source) == config['files'][source],
            'fn233 closed consumer replaced the frozen compiled-Go fixture')
    require(provenance['configuration']['profile'] == installation['profile']
            and inputs['compiler_path'] == installation['compiler_path']
            and inputs['compiler_sha256'] == installation['compiler_sha256']
            and inputs['runtime_implementation'] == installation['runtime_implementation_sha256']
            and inputs['compiler'] and all(installation['admission']['compiler_inputs'].get(name) == digest
                for name,digest in inputs['compiler'].items()),
            'fn233 closed-consumer compiler/profile/runtime differs from selected installation')
    stock = Path(installation['compiler_path']).parent.parent
    for name, digest in inputs['sources'].items():
        path = Path(name)
        require(path.is_absolute() and '..' not in path.parts
                and (config['files'].get(name) == digest or path.is_relative_to(stock/'src')
                     or path.is_relative_to(stock/'pkg/include'))
                and sha(native_roles.regular(path)) == digest,
                'fn233 closed-consumer selected source inventory changed: ' + name)
    require(manifest['record_sha256'] == observation['record_sha256']
            and manifest['record']['identity']['target'] == observation['module_sha256'],
            'fn233 closed-consumer raw artifact differs from authenticated observation')


def evaluate(value):
    recipe = bindings(value)
    root = Path(value['streaming_root'])
    config, main, _, installation, _ = streaming.context(recipe, root)
    controlled = streaming.evaluate(root)
    closed_config, fixture, closed_installation = closed_context(value,config,main)
    # Reopen the actual public launcher's raw process/FD-close evidence, never its summary.
    checker_path = HERE.parents[1] / 'testdata/parity/interception-callback-check.py'
    require(config['files'].get(str(checker_path)) == sha(checker_path),
            'fn233 callback reader is not from the typed candidate')
    checker = load_module('fn233_closed_callback_acceptance', checker_path)
    with contextlib.redirect_stdout(io.StringIO()):
        closed = checker.main(['--binary', fixture['binary'], '--installation', value['closed_installation'],
            '--inputs', value['inputs'], '--work', value['closed_root'], '--read-only'])
    require(closed['origin']['invocation']['installation_sha256'] == closed_config['files'][value['closed_installation']],
            'fn233 closed-consumer sealed origin names another installation')
    preparation_lineage(value,checker,fixture,closed_installation,closed)
    closed_candidate(closed_config,closed_installation,closed)
    check_files(closed_config['files'])
    # All typed roles reopen the complete source/HEAD/build custody after both raw readers.
    main_cli_build.read_build(value['main_build'], expected_role='release')
    main_cli_build.read_build(value['harness_build'], expected_role='interception-harness')
    main_cli_build.read_build(value['closed_build'], expected_role='host-fixture')
    return dict(schema='gomad.fn233-callback-acceptance/v1', source_sha256=main['source_sha256'],
        identity=config['identity'], closed_identity=closed_config['identity'], required=list(REQUIRED), controlled=controlled, closed=closed)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--bindings', type=Path, required=True)
    args = parser.parse_args()
    print(json.dumps(evaluate(read(native_roles.regular(args.bindings))), sort_keys=True))


if __name__ == '__main__':
    main()
