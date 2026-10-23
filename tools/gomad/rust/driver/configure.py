#!/usr/bin/env python3
"""Freeze a receiver driver configuration; does not run a qualification gate."""
import argparse
import os
from pathlib import Path
import subprocess
import sys

from driver import FROZEN, HERE, REPO, Unexecuted, deadline, mapping, read, require, sha, write
from adapters import CELLS


def cell_spec(label, evidence, manifest, source_digest, bindings=None):
    wanted = CELLS[label]
    onebox = wanted['workload'] == 'onebox'
    if wanted['workload'] not in ('fixture', 'onebox'):
        if bindings is None:
            raise Unexecuted(label + ': actual durable catalogue/history scenario must come from its Rust owner')
        from driver import load_module
        workloads = load_module('configured_cell_workloads', HERE.parent / 'workload_tools/workloads.py')
        workload = 'replication' if wanted['workload'] == 'multi-instance' else 'sqlite'
        source, working_dir = None, None
        if workload == 'replication':
            originals = [REPO / workloads.WASM / 'testdata' / name for name in
                         ('temporal_storage/recovery_test.go', 'temporal_broker/replication_test.go')]
            import hashlib
            key = hashlib.sha256(b''.join(path.read_bytes() for path in originals)).hexdigest()[:16]
            source = './.tmp/fn232-temporal-replication-' + key
            working_dir = str(Path(bindings['root']) / 'stages' / label / 'source-module')
        specification = workloads.qualification_spec(REPO, Path(evidence), workload, wanted['cell'] == 'canary',
            source, Path(manifest), guest_profile=bindings.get('environment', {}).get('GOMAD_WASM_QUALIFICATION_GUEST_PROFILE', ''),
            source_working_dir=working_dir)
        specification['Manifest']['Profile'] = wanted['profile']
        require(specification['Manifest']['IntegratedSourceSHA256'] == 'sha256:' + source_digest.removeprefix('sha256:'),
                'cell source manifest differs from frozen source')
        return specification
    target = dict(Backend='wasm', Kind='go-test' if onebox else 'go-run',
                  Source='./tests' if onebox else '.',
                  WorkingDir=str(REPO if onebox else REPO / 'tools/gomad/internal/backends/wasm/qualification/testdata/fixture'),
                  BuildTags=['disable_grpc_modules', 'gomad_wasm_slim', 'sqlite3_dotlk', 'test_dep'] if onebox else ['test_dep'],
                  PreparationRoot=str(Path(evidence) / 'preparation'))
    selected = '[' if wanted['cell'] == 'canary' else '^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$'
    target['Args'] = (['-test.v', '-test.count=1', '-test.timeout=72h', '-test.run=' + selected,
                       '-persistenceType=sql', '-persistenceDriver=sqlite'] if onebox else
                      ['canary'] if wanted['cell'] == 'canary' else [])
    campaign = dict(Target=target, ExecutionTimeout=900_000_000_000 if onebox else 60_000_000_000,
                    OverallTimeout=1_800_000_000_000 if onebox else 180_000_000_000,
                    TerminateGrace=1_000_000_000 if onebox else 100_000_000, OutputLimit=256 << 20 if onebox else 1 << 20,
                    WorldTransitionLimit=32 << 20 if onebox else 1 << 20, OnFailure='all', FailureBudget=1,
                    Artifacts=str(Path(evidence) / 'artifacts'))
    if onebox:
        campaign.update(ChoiceTraceLimit=64 << 20, SuccessBytesLimit=2 << 30,
                        Environment=['TEMPORAL_ROOT=/workspace', 'TMPDIR=/tmp', 'TEMPORAL_TEST_LOG_LEVEL=warn'])
    return dict(Manifest=dict(Provisional=False, IntegratedSourceManifestPath=manifest,
                             IntegratedSourceSHA256='sha256:' + source_digest.removeprefix('sha256:'), ID=wanted['id'], Workload=wanted['workload'],
                             Profile=wanted['profile'], Cell=wanted['cell'], Seed=7, Repeat=2),
                Campaign=campaign, ArtifactRoot=evidence)


def native_role_inputs(entry):
    require(isinstance(entry, dict) and set(entry) <= {'binary', 'build_receipt', 'admission'}
            and {'binary', 'build_receipt'} <= set(entry), 'native role requires binary/build_receipt and optional admission')
    require(all(Path(path).is_absolute() and str(Path(path).resolve()) == path for path in entry.values()),
            'native role binding paths must be canonical absolute files')
    receipt_path = Path(entry['build_receipt'])
    receipt = read(receipt_path)
    require(receipt['schema'] == 'gomad.native-measurement-build/v1', 'native role requires the full typed build receipt')
    paths = {str(receipt_path), entry['binary'], receipt['executable']} | set(receipt['inputs'])
    if entry['binary'] != receipt['executable']:
        require(entry.get('admission'), 'published native role requires its actual admission')
    if entry.get('admission'):
        admission_path = Path(entry['admission'])
        admission = read(admission_path)
        require(admission['schema'] == 'gomad.native-measurement-admission/v1'
                and admission['build_receipt'] == str(receipt_path) and admission['executable'] == entry['binary'],
                'native role admission does not publish this exact build/binary')
        paths.add(str(admission_path))
        for name in ('artifact-identity', 'staged-identity'):
            paths.update(str(admission_path.parent / (name + '.' + suffix)) for suffix in ('stdout', 'stderr', 'json'))
    require(all(Path(path).is_absolute() and Path(path).is_file() and not Path(path).is_symlink() for path in paths),
            'missing/nonregular static native role input')
    return paths


def native_bootstrap_inputs(config):
    binding = config.get('native_measurement', {})
    require(set(binding) == {'admission'}, 'native measurement bootstrap admission required')
    admission = read(binding['admission'])
    return native_role_inputs(dict(binary=admission['executable'], build_receipt=admission['build_receipt'], admission=binding['admission']))


def native_profile_inputs(config):
    profiles = config['final_profiles']
    paths = set()
    for name in ('measure', 'ordinary', 'diagnostic'):
        paths.update(native_role_inputs(profiles[name]))
    for field in ('terminal_helper_receipt', 'terminal_diagnostic_helper_receipt'):
        if field in profiles:
            admission = read(profiles[field])
            paths.update(native_role_inputs(dict(binary=admission['executable'], build_receipt=admission['build_receipt'], admission=profiles[field])))
    return paths


def freeze_final_cycle(config, output):
    """Pin static inputs, then PLAN, then its all-verb binding; outputs have no hash cycle."""
    require(config.get('stress_cycle', 'GENERATE') == 'GENERATE', 'stress cycle must be generated for this freeze')
    require(set(config['stress']) == {'spec', 'binary', 'build_receipt', 'producer_root', 'producer_name'},
            'missing/unknown actual stress producer inputs')
    for key in ('spec', 'binary', 'build_receipt'):
        path = config['stress'][key]
        require(config['files'].get(path) == sha(path), 'unsealed actual stress input: ' + key)
    template = Path(config['public_verbs']['bindings'])
    require(config['files'].get(str(template)) == sha(template), 'unsealed public verb binding template')
    declared = read(template)
    require(declared['schema'] == 'gomad.all-verb-bindings/v1' and set(declared) == {'schema', 'files', 'trees'},
            'foreign public verb binding template')
    # Relocating the binding document must preserve every static input's original resolution.
    for binding in declared['files'].values():
        binding['path'] = str((template.parent / binding['path']).resolve())
    for binding in declared['trees'].values():
        for key in ('path', 'inventory'):
            binding[key] = str((template.parent / binding[key]).resolve())
    fields = ('repo', 'root', 'source_manifest', 'source_sha256', 'sent_epoch', 'deadline_s', 'identity', 'binary_sha256', 'stress')
    plan = {key: config[key] for key in fields}
    plan.update(schema='gomad.stress-cycle/v1', files=dict(config['files']))
    plan_path = output.parent / 'stress-cycle.json'
    binding_path = output.parent / 'public-verbs-bindings.json'
    require(not plan_path.exists() and not binding_path.exists(), 'final cycle outputs must be fresh')
    write(plan_path, plan)
    config['stress_cycle'] = str(plan_path)
    config['files'][str(plan_path)] = sha(plan_path)
    declared['files']['stress-cycle'] = dict(path=str(plan_path), sha256=sha(plan_path))
    write(binding_path, declared)
    config['public_verbs']['bindings'] = str(binding_path)
    config['files'][str(binding_path)] = sha(binding_path)


def freeze(bindings, output, source_manifest):
    config = dict(bindings)
    deadline(config['deadline_s'])
    from host_admission import STAT, bindings as scratch_bindings
    scratch_bindings(config)
    require(set(config['stages']) == set(mapping()), 'all 96 stage mappings are required')
    require(all(entry.get('adapter') != 'pending' for entry in config['stages'].values()),
            'unbound original stage endpoints must be composed before freezing')
    require(config.get('final_reader') and config.get('final_sections'), 'actual final readers must be composed before freezing')
    output = output.resolve()
    source_manifest = source_manifest.resolve()
    require(not output.exists() and not source_manifest.exists(), 'freeze outputs must be fresh')
    paths = subprocess.check_output(['git', '-C', str(REPO), 'ls-files', '-z']).decode().split('\0')
    source_files = sorted(str(REPO / path) for path in paths if path and not path.startswith('.flow/')
                          and (REPO / path).is_file() and not (REPO / path).is_symlink())
    require(source_files, 'empty source inventory')
    with source_manifest.open('x') as handle:
        for path in source_files:
            require('\n' not in path, 'unsupported source filename')
            handle.write(sha(path) + '  ' + path + '\n')
    config.update(schema='gomad.rust-qualification-driver/v1', repo=str(REPO),
                  head=subprocess.check_output(['git', '-C', str(REPO), 'rev-parse', 'HEAD'], text=True).strip(),
                  source_manifest=str(source_manifest), source_sha256=sha(source_manifest),
                  symlinks={str(REPO / path): os.readlink(REPO / path) for path in paths if path and (REPO / path).is_symlink()})
    inputs = {str(STAT)} | set(config.pop('inputs')) | set(source_files) | {str(source_manifest)}
    inputs.update(str(path) for path in FROZEN.glob('*.py'))
    inputs.update(str(path) for path in FROZEN.glob('*.sh'))
    inputs.update((str(FROZEN / 'dag.tsv'), str(FROZEN.parent / 'q1-serial/run-qualification.sh')))
    inputs.update(config['installations'].values())
    inputs.update(read(path)['compiler_path'] for path in config['installations'].values())
    for installation in config['installations'].values():
        inventory = read(installation)['admission']['compiler_inputs']
        require(isinstance(inventory, dict) and inventory, 'actual installation compiler inventory required')
        for path, expected in inventory.items():
            require(Path(path).is_absolute() and Path(path).is_file() and not Path(path).is_symlink()
                    and sha(path) == expected, 'changed installation compiler input: ' + path)
            inputs.add(path)
    inputs.update((config['binary'], config['build_receipt']))
    inputs.update(native_bootstrap_inputs(config))
    inputs.update(native_profile_inputs(config))
    for token in config['stages']['sqlite-catalogue'].get('oracle') or []:
        if Path(token).is_absolute():
            require(Path(token).is_file() and not Path(token).is_symlink(), 'nonregular independent oracle input')
            inputs.add(token)
    for index in range(32):
        entry = config['stages'][f'runtime-{index}']
        require(entry['adapter'] == 'runtime', 'runtime stage requires its exact release-library gate')
        inputs.update((entry['harness'], entry['harness_build_receipt']))
    for label, wanted in CELLS.items():
        entry = config['stages'][label]
        if entry.get('spec') == 'GENERATE':
            path = output.parent / (label + '.json')
            specification = cell_spec(label, str(Path(config['root']) / 'cells' / label), str(source_manifest), config['source_sha256'], bindings=config)
            write(path, specification)
            entry['spec'] = str(path)
            inputs.add(str(path))
    if config.get('fn222'):
        argv = config['fn222']['argv']
        config['fn222']['argv'] = argv = [str(output) if item == 'FROZEN_CONFIG' else
            str(source_manifest) if item == 'SOURCE_MANIFEST' else item for item in argv]
        require('--support' in argv and '--captured-config' in argv, 'fn222 receiver and captured policy inputs must be explicit')
        inputs.update([argv[0], argv[1], argv[argv.index('--support') + 1], argv[argv.index('--captured-config') + 1]])
    inputs.add(config['public_verbs']['bindings'])
    inputs.update(config['stress'][key] for key in ('spec', 'binary', 'build_receipt'))
    config['files'] = {str(Path(path).resolve()): sha(path) for path in sorted(inputs)}
    import terminal_inputs
    terminal_spec = output.parent / 'terminal-module-input.json'
    require(not terminal_spec.exists(), 'terminal preparation request must be fresh')
    write(terminal_spec, terminal_inputs.preparation_spec(config))
    config['terminal_module_spec'] = str(terminal_spec)
    config['files'][str(terminal_spec)] = sha(terminal_spec)
    inputs.add(str(terminal_spec))
    freeze_final_cycle(config, output)
    import final_public_verbs
    final_public_verbs.binding(config)
    import native_roles
    for name, role in (('measure', {'kind': 'release'}), ('ordinary', {'kind': 'release'}),
                       ('diagnostic', {'kind': 'diagnostic', 'features': ['diagnostic-profile']})):
        entry = config['final_profiles'][name]
        native_roles.validate_build(config, entry['build_receipt'], role)
        if entry.get('admission'):
            native_roles.validate_admission(config, entry['admission'], role)
    import primary_stages
    observed = dict(config, _config_path=str(output))
    for label, entry in config['stages'].items():
        if entry.get('adapter') == 'primary':
            primary_stages.preflight(observed, label, Path(config['root']) / 'stages' / label)
    for entry in config['final_sections'].values():
        require(all(path in config['files'] for path in entry['argv'][:2]), 'unsealed actual final endpoint')
    for path in inputs:
        require(Path(path).is_absolute() and not Path(path).is_symlink(), 'non-regular frozen input: ' + path)
    write(output, config)
    write(output.with_suffix(output.suffix + '.sha256.json'), dict(path=str(output), sha256=sha(output)))
    return config


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--bindings', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--source-manifest', type=Path, required=True)
    args = parser.parse_args()
    try:
        freeze(read(args.bindings), args.output, args.source_manifest)
        print(sha(args.output))
        return 0
    except (OSError, ValueError, KeyError) as error:
        print('REFUSED: ' + str(error), file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
