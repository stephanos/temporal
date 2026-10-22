#!/usr/bin/env python3
"""Fresh public install/discover/doctor/refusal parity and strict Rust goldens.

Only the declared absolute manifest/cache/tool path operands are tokenized.
Implementation, compiler, configuration and runtime identities remain literal.
Every subprocess retains original output bytes and digest receipts.
"""
import argparse
import base64
import copy
import hashlib
import json
import os
from pathlib import Path
import runpy
import shutil
import stat
import subprocess
import sys

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent


def encode(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False, allow_nan=False).encode()


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def equal(actual, expected, path):
    if type(actual) is not type(expected):
        raise ValueError(f'{path}: type {type(actual).__name__}, expected {type(expected).__name__}')
    if isinstance(actual, dict):
        equal(set(actual), set(expected), path + '.fields')
        for key in sorted(actual):
            equal(actual[key], expected[key], path + '.' + key)
    elif isinstance(actual, (list, tuple)):
        equal(len(actual), len(expected), path + '.length')
        for index, (left, right) in enumerate(zip(actual, expected)):
            equal(left, right, f'{path}[{index}]')
    elif actual != expected:
        raise ValueError(f'{path}: got {actual!r}, expected {expected!r}')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'go-binary', 'go-helper', 'go-compiler', 'go-source', 'preparation-root', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--grammar-only', action='store_true')
    args = parser.parse_args()
    for key, value in vars(args).items():
        if isinstance(value, Path):
            setattr(args, key, value.resolve())
    args.work.mkdir(parents=True, exist_ok=False)
    input_paths = [args.binary, args.go_binary, args.go_helper, args.go_compiler,
                   CORPUS / 'installation-corpus.json', CORPUS / 'installation-lane.py',
                   CORPUS / 'plan-check.py', CORPUS / 'recover-check.py', CORPUS / 'installation-numeric-check.py',
                   CORPUS / 'campaign-reference-check.py', CORPUS.parents[1] / 'scripts/behavioral-parity']
    input_digests = {str(path): digest(path.read_bytes()) for path in input_paths}
    (args.work / 'inputs.json').write_bytes(encode(input_digests) + b'\n')
    parity = runpy.run_path(str(CORPUS.parents[1] / 'scripts/behavioral-parity'))
    decode = parity['decode']
    declaration = decode((CORPUS / 'installation-corpus.json').read_bytes())
    equal(declaration['schema'], 'gomad.installation-corpus/v1', 'corpus.schema')
    equal({row['name'] for row in declaration['grammar']}, {'missing-output', 'unknown-flag', 'discover-arguments',
        'analyze-bare', 'analyze-help', 'analyze-arguments', 'doctor-bare', 'doctor-help', 'doctor-arguments'}, 'corpus.grammar_cases')
    equal(declaration['profiles'], ['gomad.wasi-stock-exploratory/v1', 'gomad.wasi-cooperative/v1'], 'corpus.profiles')
    equal(len({row['name'] for row in declaration['grammar']}), len(declaration['grammar']), 'corpus.unique')
    equal(set(declaration['required']), {'helper-default', 'helper-matching', 'helper-mismatch', 'compiler-missing',
        'compiler-version', 'publication-collision', 'go-lineage', 'explicit-precedence', 'environment-precedence',
        'sibling', 'invalid-explicit', 'invalid-environment', 'removed-build-inputs', 'changed-compiler',
        'configured-limits', 'temporal-policy', 'captured-payload-mismatch', 'captured-descriptor-mismatch', 'schema-symlink', 'changed-runtime', 'missing-runtime', 'external-helper-never-executed', 'seed-plan', 'scoped-plan', 'seed-recovery', 'round-recovery', 'duplicate-configuration-member', 'numeric-grammar', 'clock-defaults', 'configuration-aliases', 'configuration-nulls', 'captured-descriptor-aliases'}, 'corpus.required')
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_') and key != 'GOROOT'}
    environment.update(GOENV='off', GOWORK='off', GOTOOLCHAIN='local', GOMAXPROCS='2', GOPROXY='off', GOSUMDB='off')
    receipts = []

    def run(name, argv, status, extra=None):
        result = subprocess.run(list(map(str, argv)), capture_output=True, env=dict(environment, **(extra or {})), timeout=240)
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        receipts.append(dict(name=name, argv=list(map(str, argv)), status=result.returncode,
                             stdout_sha256=digest(result.stdout), stderr_sha256=digest(result.stderr)))
        (args.work / 'commands.json').write_bytes(encode(receipts) + b'\n')
        equal(result.returncode, status, name + '.status')
        return result

    def discover(name, binary, manifest=None, extra=None):
        prefix = ['--installation', manifest] if manifest else []
        response = run(name, [binary, *prefix, 'discover'], 0, extra)
        value = decode(response.stdout)
        equal(set(value), {'path', 'source', 'sha256', 'manifest', 'capabilities'}, name + '.response_fields')
        raw = Path(value['path']).read_bytes()
        equal(value['sha256'].removeprefix('sha256:'), digest(raw), name + '.manifest_digest')
        equal(value['manifest'], decode(raw), name + '.manifest_bytes')
        return value

    fixture = CORPUS.parents[1] / 'internal/backends/wasm/qualification/testdata/fixture'
    views = []
    for attempt in range(2):
        receipt_start = len(receipts)
        root = args.work / f'run-{attempt}'
        root.mkdir()
        grammar = []
        for case in declaration['grammar']:
            observed = []
            for lineage, binary in (('go', args.go_binary), ('rust', args.binary)):
                prefix = ['--installation', root / 'missing.json'] if case.get('selected') else []
                result = run(f'{lineage}-{case["name"]}-{attempt}', [binary, *prefix, *case['argv']], case['status'])
                equal(result.stdout, b'', case['name'] + '.stdout')
                if case.get('unsupported'):
                    equal(any(marker in result.stderr for marker in (b'unknown command', b'unknown gomad command', b'unsupported command')),
                          True, case['name'] + '.unsupported')
                else:
                    equal(case['stderr_contains'].encode() in result.stderr, True, case['name'] + '.stderr')
                observed.append(dict(name=case['name'], status=result.returncode, stdout=''))
            equal(observed[0], observed[1], case['name'] + '.parity')
            grammar.append(observed[1])
        doctors = []
        for lineage, binary in (('go', args.go_binary), ('rust', args.binary)):
            result = run(f'{lineage}-doctor-default-{attempt}',
                [binary, 'doctor', '--json', '--artifacts', root / 'doctor-artifacts'], 2)
            doctor = decode(result.stdout)
            equal(set(doctor), {'schema', 'available', 'host', 'supported_platforms', 'boundary_manifest_version',
                'io_inventory_sha256', 'io_implementation_sha256', 'adapters', 'installation_source', 'toolchain_root',
                'artifact_directory', 'repair_instruction', 'checks'}, lineage + '.doctor.fields')
            equal(doctor['available'], False, lineage + '.doctor.unavailable')
            equal(doctor['artifact_directory'], str(root / 'doctor-artifacts'), lineage + '.doctor.artifacts')
            doctor['artifact_directory'] = '$RUN/doctor-artifacts'
            doctors.append(doctor)
        equal(doctors[0], doctors[1], 'doctor.default_parity')
        if args.grammar_only:
            views.append(dict(schema='gomad.installation-golden/v1', grammar=grammar, doctor=doctors[1]))
            continue
        manifests = {}
        installed = []
        for profile, configured, temporal in [(profile, False, False) for profile in declaration['profiles']] + [('gomad.wasi-cooperative/v1', True, False), ('gomad.wasi-cooperative/v1', False, True)]:
            short = 'temporal' if temporal else 'configured' if configured else 'stock' if 'stock' in profile else 'cooperative'
            configuration = None
            if configured:
                configuration = dict(Profile=profile, Args=['fixture-argument'], Environment=['PWD=/workspace', 'TZ=UTC'],
                    WorkingDirectory='/workspace', WritableDirectories=['/workspace', '/tmp'],
                    Clock=dict(EpochNanos=946684800000000000, ReadStepNanos=0),
                    Limits=dict(OutputBytes=1048576, TranscriptBytes=8388608, Calls=10000, Descriptors=64,
                                Files=1000, FilesystemBytes=8388608, PendingEvents=4096))
                (root / 'configuration.json').write_bytes(encode(configuration))
            pairs, capability_pair = [], []
            for lineage, binary, helper in (('go', args.go_binary, args.go_helper), ('rust', args.binary, None)):
                output = root / f'{lineage}-{short}.json'
                command = [binary, 'install', '--compiler', args.go_compiler, '--cache', args.preparation_root / (lineage + '-cache'),
                           '--output', output]
                if temporal:
                    command += ['--temporal-checkout', args.go_source, '--temporal-package', './tests/gomadfunctional']
                else:
                    command += ['--profile', profile]
                if helper:
                    command += ['--helper', helper]
                if short != 'stock':
                    command += ['--runtime-root', args.go_source / 'tools/gomad/internal/model/runtimesource']
                if configured:
                    command += ['--config', root / 'configuration.json', '--fuel', '0x75bcd15', '--memory-bytes', '0x8000000',
                                '--compile-cache', args.preparation_root / 'compiled-code']
                run(f'{lineage}-install-{short}-{attempt}', command, 0)
                equal(stat.S_IMODE(output.stat().st_mode), 0o600, lineage + '.manifest.mode')
                value = discover(f'{lineage}-discover-{short}-{attempt}', binary, output)
                manifest = value['manifest']
                capability_pair.append(value['capabilities'])
                manifests[lineage, short] = output
                if lineage == 'go':
                    options = manifest['options']
                    semantic = dict(profile=options['Config']['Profile'], fuel=options['Fuel'], memory_bytes=options['MemoryBytes'],
                                    go_version=options['GoVersion'], compiler_sha256=options['CompilerSHA256'].removeprefix('sha256:'))
                else:
                    semantic = {key: manifest[key] for key in ('profile', 'fuel', 'memory_bytes', 'go_version', 'compiler_sha256')}
                    equal(semantic['compiler_sha256'], digest(args.go_compiler.read_bytes()), 'compiler.actual_bytes')
                    installed.append(dict(manifest=project_manifest(manifest, root, args), capabilities=value['capabilities']))
                original = manifest['options']['Config'] if lineage == 'go' else manifest['admission']['guest_configuration']
                for field in ('Profile', 'Clock', 'Limits', 'WorkingDirectory', 'WritableDirectories', 'Environment'):
                    semantic[field] = original[field]
                semantic['Args'] = original.get('Args') or []
                if temporal:
                    authenticate_captured(original)
                    semantic['guest_configuration'] = original
                    semantic['build_tags'] = manifest['options']['BuildTags'] if lineage == 'go' else manifest['build_tags']
                    equal((semantic['fuel'], semantic['memory_bytes']), (1 << 50, 4 << 30), 'temporal.default_limits')
                if configured:
                    for field in configuration:
                        equal(original[field], configuration[field], lineage + '.configuration.' + field)
                pairs.append(semantic)
                run(f'{lineage}-collision-{short}-{attempt}', command, 3)
            equal(pairs[0], pairs[1], short + '.installation_parity')
            installed[-1]['go_capabilities'] = capability_pair[0]
            go_caps, rust_caps = map(dict, capability_pair)
            equal(go_caps.pop('ChoiceProfiles'), ['gomad-choice-trace/v3'] if short != 'stock' else None,
                  short + '.go_choice_format')
            equal(rust_caps.pop('ChoiceProfiles'), ['gomad-choice-trace/v4'] if short != 'stock' else None,
                  short + '.rust_choice_format')
            equal(rust_caps['Commands'].count('plan'), 1, 'capabilities.authorized_plan_receiver')
            rust_caps['Commands'] = [verb for verb in rust_caps['Commands'] if verb != 'plan']
            equal(go_caps, rust_caps, short + '.capability_parity')
        numeric_check = runpy.run_path(str(CORPUS / 'installation-numeric-check.py'))['check']
        numeric_go = numeric_check(args.go_binary, args.go_compiler, root / 'go-numeric', args.go_helper)
        numeric_rust = numeric_check(args.binary, args.go_compiler, root / 'rust-numeric')
        equal(numeric_go, numeric_rust, 'installation.numeric_parity')
        equal(len(numeric_go), 12, 'installation.numeric_case_count')
        clocks = clock_cases(args, root, run, decode)
        descriptor_cases = captured_descriptor_cases(args, manifests, root, run, decode)
        configuration_refusals(args, manifests, root, run, decode)
        seed_plan = runpy.run_path(str(CORPUS / 'plan-check.py'))['check'](
            args.binary, manifests['rust', 'cooperative'], root / 'seed-plan',
            fixture, '.', go_binary=args.go_binary, go_installation=manifests['go', 'cooperative'],
            go_working_dir=fixture, support_dir=CORPUS.parents[1])
        recovery = runpy.run_path(str(CORPUS / 'recover-check.py'))['check'](
            args.binary, manifests['rust', 'cooperative'], root / 'recovery', fixture, '.')
        go_recovery = {kind: go_recovery_check(args, manifests['go', 'cooperative'], root, run, decode, kind)
                       for kind in ('seed', 'round')}
        for campaign in recovery['campaigns']:
            equal([report['changed'] for report in campaign['recovery']],
                  [report['changed'] for report in go_recovery[campaign['kind']]['reports']], campaign['kind'] + '.recovery.parity')
        rust_manifest = manifests['rust', 'cooperative']
        go_manifest = manifests['go', 'cooperative']
        duplicate_manifest = root / 'duplicate-configuration-member.json'
        original_manifest = rust_manifest.read_bytes()
        needle = b'"guest_configuration":{'
        equal(original_manifest.count(needle), 1, 'manifest.configuration_object_count')
        duplicate_manifest.write_bytes(original_manifest.replace(needle,
            needle + b'"Profile":' + encode('gomad.wasi-cooperative/v1') + b',', 1))
        duplicate_manifest.chmod(0o600)
        run(f'rust-duplicate-configuration-member-{attempt}',
            [args.binary, '--installation', duplicate_manifest, 'discover'], 3)
        resolution = []

        def selected(lineage, requested, value, manifest):
            equal(value['path'], str(manifest.resolve()), lineage + '.' + requested + '.selected_path')
            equal(value['sha256'], ('sha256:' if lineage == 'go' else '') + digest(manifest.read_bytes()),
                  lineage + '.' + requested + '.selected_digest')
            equal(value['source'], 'explicit' if lineage == 'go' else requested, lineage + '.' + requested + '.reported_source')
            resolution.append(dict(lineage=lineage, requested_origin=requested, reported_origin=value['source'],
                selected_path='$RUN/' + str(manifest.relative_to(root)), manifest_sha256=value['sha256']))

        for lineage, binary, manifest in (('go', args.go_binary, go_manifest), ('rust', args.binary, rust_manifest)):
            for verb in ('doctor', 'analyze'):
                refusal = run(f'{lineage}-installed-{verb}-{attempt}', [binary, '--installation', manifest, verb, '--help'], 2)
                equal(refusal.stdout, b'', lineage + '.' + verb + '.installed_stdout')
            default_analyze = run(f'{lineage}-default-analyze-{attempt}', [binary, 'analyze', '--help'], 2)
            equal(default_analyze.stdout, b'', lineage + '.analyze.default_stdout')
            value = discover(f'{lineage}-explicit-{attempt}', binary, manifest,
                             {'GOMAD_WASM_INSTALLATION': str(root / 'missing.json')})
            selected(lineage, 'explicit', value, manifest)
            value = discover(f'{lineage}-environment-{attempt}', binary, extra={'GOMAD_WASM_INSTALLATION': str(manifest)})
            selected(lineage, 'environment', value, manifest)
            run(f'{lineage}-invalid-explicit-{attempt}', [binary, '--installation', root / 'missing.json', 'discover'], 3,
                {'GOMAD_WASM_INSTALLATION': str(manifest)})
            sibling_dir = root / (lineage + '-sibling')
            sibling_dir.mkdir()
            sibling_binary = sibling_dir / 'gomad'
            shutil.copy2(binary, sibling_binary)
            shutil.copy2(manifest, sibling_dir / 'gomad-wasm.json')
            value = discover(f'{lineage}-sibling-{attempt}', sibling_binary)
            selected(lineage, 'executable-sibling', value, sibling_dir / 'gomad-wasm.json')
            run(f'{lineage}-invalid-environment-{attempt}', [sibling_binary, 'discover'], 3,
                {'GOMAD_WASM_INSTALLATION': str(root / 'missing.json')})
        run(f'rust-go-lineage-{attempt}', [args.binary, '--installation', go_manifest, 'discover'], 3)
        base = [args.binary, 'install', '--compiler', args.go_compiler, '--cache', args.preparation_root / 'rust-cache']
        run(f'rust-helper-matching-{attempt}', [*base, '--helper', root / 'rust-sibling/gomad', '--output', root / 'matching.json'], 0)
        matching = discover(f'rust-helper-matching-discover-{attempt}', args.binary, root / 'matching.json')
        equal(matching['manifest']['admission']['explicit_helper'],
              dict(path=str(root / 'rust-sibling/gomad'), sha256=digest(args.binary.read_bytes())), 'helper.explicit_pin')
        hostile = root / 'hostile-helper'
        marker = root / 'external-helper-was-executed'
        hostile.write_text('#!/bin/sh\necho executed > "' + str(marker) + '"\nexit 0\n')
        hostile.chmod(0o700)
        run(f'rust-helper-never-executed-{attempt}', [*base, '--helper', hostile, '--output', root / 'hostile.json'], 2)
        equal(marker.exists(), False, 'helper.external_never_executed')
        run(f'rust-helper-missing-{attempt}', [*base, '--helper', root / 'missing-helper', '--output', root / 'helper-missing.json'], 2)
        run(f'rust-config-missing-{attempt}', [*base, '--config', root / 'missing-config', '--output', root / 'config-missing.json'], 2)
        malformed = root / 'malformed-config.json'
        malformed.write_bytes(b'{')
        run(f'rust-config-malformed-{attempt}', [*base, '--config', malformed, '--output', root / 'config-malformed.json'], 2)
        run(f'rust-helper-mismatch-{attempt}', [*base, '--helper', args.go_helper, '--output', root / 'mismatch.json'], 2)
        run(f'rust-compiler-missing-{attempt}', [args.binary, 'install', '--compiler', root / 'missing-compiler',
            '--cache', args.preparation_root / 'rust-cache', '--output', root / 'missing.json'], 2)
        run(f'rust-compiler-version-{attempt}', [*base, '--go-version', 'go0.0.0', '--output', root / 'version.json'], 2)
        private = root / 'private-inputs'
        private.mkdir()
        compiler_alias = private / 'compiler'
        (compiler_alias / 'bin').mkdir(parents=True)
        private_compiler = compiler_alias / 'bin/go'
        shutil.copy2(args.go_compiler, private_compiler)
        for entry in args.go_compiler.parent.parent.iterdir():
            if entry.name == 'bin':
                continue
            destination = compiler_alias / entry.name
            if entry.is_dir():
                destination.symlink_to(entry, target_is_directory=True)
            else:
                shutil.copy2(entry, destination)
        private_runtime = private / 'runtimesource'
        shutil.copytree(args.go_source / 'tools/gomad/internal/model/runtimesource', private_runtime)
        detached_path = root / 'detached.json'
        run(f'rust-detached-install-{attempt}', [args.binary, 'install', '--compiler', private_compiler,
            '--runtime-root', private_runtime, '--profile', 'gomad.wasi-cooperative/v1',
            '--cache', root / 'detached-cache', '--output', detached_path], 0)
        before = discover(f'rust-detached-before-{attempt}', args.binary, detached_path)
        runtime_file = next(path for path in sorted(private_runtime.rglob('*.go')) if path.is_file())
        runtime_bytes = runtime_file.read_bytes()
        for kind in ('changed', 'missing'):
            if kind == 'changed':
                runtime_file.write_bytes(runtime_bytes + b'\n')
            else:
                runtime_file.unlink()
            run(f'rust-{kind}-runtime-plan-{attempt}', [args.binary, '--installation', detached_path, 'plan', '--json',
                '--output', root / (kind + '-runtime-plan.json'), '--artifacts', root / (kind + '-runtime-artifacts'),
                '--working-dir', fixture, 'go-run', '.'], 3)
            equal((root / (kind + '-runtime-plan.json')).exists(), False, kind + '_runtime.no_plan_published')
            runtime_file.write_bytes(runtime_bytes)
        private_compiler.write_bytes(b'changed compiler bytes')
        run(f'rust-changed-compiler-plan-{attempt}', [args.binary, '--installation', detached_path, 'plan', '--json',
            '--output', root / 'refused-plan.json', '--artifacts', root / 'refused-artifacts',
            '--working-dir', fixture, 'go-run', '.'], 3)
        equal((root / 'refused-plan.json').exists(), False, 'changed_compiler.no_plan_published')
        changed = discover(f'rust-changed-compiler-{attempt}', args.binary, detached_path)
        equal(changed, before, 'discover.changed_build_input_does_not_change_manifest_identity')
        shutil.rmtree(compiler_alias)
        shutil.rmtree(private_runtime)
        after = discover(f'rust-removed-build-inputs-{attempt}', args.binary, detached_path)
        equal(after, before, 'discover.removed_build_inputs')
        commands = [dict(case=row['name'].replace(root.name, 'run') if root.name in row['name']
                              else row['name'].removesuffix(f'-{attempt}'), status=row['status'])
                    for row in receipts[receipt_start:]]
        views.append(dict(schema='gomad.installation-golden/v1', grammar=grammar, doctor=doctors[1],
                          installations=installed, seed_plan=seed_plan, recovery=recovery, go_recovery=go_recovery, numeric=numeric_rust, clocks=clocks, descriptors=descriptor_cases, resolution=resolution, commands=commands))
    equal({str(path): digest(path.read_bytes()) for path in input_paths}, input_digests, 'installation.unchanged_inputs')
    equal(views[0], views[1], 'installation.repeated_agreement')
    golden = CORPUS / 'goldens-installation' / ('grammar.json' if args.grammar_only else 'installation.json')
    regenerate = os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1'
    golden.parent.mkdir(exist_ok=True)
    for view in views:
        parity['golden'](golden, view, regenerate)
        if view is views[0]:
            first = golden.read_bytes()
    equal(golden.read_bytes(), first, 'installation.second_regeneration_bytes')
    print(f'installation: {len(receipts)} direct public commands plus authenticated child lanes; repeated strict golden agreement')


def clock_cases(args, root, run, decode):
    base = dict(Args=['fixture-argument'], Environment=['PWD=/workspace', 'TZ=UTC'],
        WorkingDirectory='/workspace', WritableDirectories=['/workspace', '/tmp'],
        Limits=dict(OutputBytes=1048576, TranscriptBytes=8388608, Calls=10000, Descriptors=64,
                    Files=1000, FilesystemBytes=8388608, PendingEvents=4096))
    cases = [
        ('omitted-read-step', 'gomad.wasi-cooperative/v1', {'EpochNanos': 946684800000000000}, 0,
         {'EpochNanos': 946684800000000000, 'ReadStepNanos': 0}),
        ('empty-clock', 'gomad.wasi-cooperative/v1', {}, 0, {'EpochNanos': 0, 'ReadStepNanos': 0}),
        ('missing-clock', 'gomad.wasi-cooperative/v1', None, 0, {'EpochNanos': 0, 'ReadStepNanos': 0}),
        ('stock-missing-clock', 'gomad.wasi-stock-exploratory/v1', None, 2, None),
        ('stock-zero-step', 'gomad.wasi-stock-exploratory/v1', {'EpochNanos': 946684800000000000, 'ReadStepNanos': 0}, 2, None),
        ('epoch-overflow', 'gomad.wasi-cooperative/v1', {'EpochNanos': 9223372036854775808, 'ReadStepNanos': 0}, 2, None),
        ('read-step-overflow', 'gomad.wasi-cooperative/v1', {'EpochNanos': 0, 'ReadStepNanos': 1000000001}, 2, None),
    ]
    configured = []
    for name, profile, clock, expected, defaults in cases:
        config = dict(base, Profile=profile)
        if clock is not None:
            config['Clock'] = clock
        configured.append((name, profile, config, expected, defaults))
    canonical = dict(base, Profile='gomad.wasi-cooperative/v1',
                     Clock=dict(EpochNanos=946684800000000000, ReadStepNanos=0))
    def mixed(value):
        return {''.join(char.upper() if index % 2 == 0 else char.lower() for index, char in enumerate(key)): mixed(item)
                for key, item in value.items()} if isinstance(value, dict) else value
    additions = [('mixed-all-typed', mixed(canonical), 0, canonical['Clock'])]
    config = copy.deepcopy(canonical)
    config['ClocK'] = config.pop('Clock')
    additions.append(('kelvin-clock', config, 0, canonical['Clock']))
    config = copy.deepcopy(canonical)
    config['Limitſ'] = config.pop('Limits')
    additions.append(('long-s-limits', config, 0, canonical['Clock']))
    for name, clock in [('null-clock', None), ('null-numerics', {'EpochNanos': None, 'ReadStepNanos': None})]:
        config = dict(canonical, Clock=clock)
        additions.append((name, config, 0, {'EpochNanos': 0, 'ReadStepNanos': 0}))
    config = copy.deepcopy(canonical)
    config['Cl_ock'] = config.pop('Clock')
    additions.append(('delimiter-root', config, 2, None))
    config = copy.deepcopy(canonical)
    config['Clock']['Epoch_Nanos'] = config['Clock'].pop('EpochNanos')
    additions.append(('delimiter-nested', config, 2, None))
    config = copy.deepcopy(canonical)
    config['clock'] = config['Clock']
    additions.append(('duplicate-alias-root', config, 2, None))
    config = copy.deepcopy(canonical)
    config['Clock']['epochnanos'] = config['Clock']['EpochNanos']
    additions.append(('duplicate-alias-nested', config, 2, None))
    config = copy.deepcopy(canonical)
    config['Clock']['Unknown'] = 1
    additions.append(('unknown-clock', config, 2, None))
    for name, field, value, expected in [
        ('null-entropy-key', 'EntropyKey', None, 0),
        ('null-captured-inputs', 'CapturedInputs', None, 0),
        ('null-captured-manifest', 'CapturedInputs', {'Manifest': None}, 0),
        ('null-capture-limits', 'CapturedInputs', {'Manifest': {'Limits': None}}, 0),
        ('null-argument-item', 'Args', [None], 0),
        ('null-limits', 'Limits', None, 2),
        ('entropy-short', 'EntropyKey', [0] * 31, 2),
        ('entropy-long', 'EntropyKey', [0] * 33, 2),
        ('null-entropy-items', 'EntropyKey', [None] * 32, 0),
    ]:
        config = copy.deepcopy(canonical)
        config[field] = value
        additions.append((name, config, expected, canonical['Clock'] if not expected else None))
    config = copy.deepcopy(canonical)
    config['Limits']['OutputBytes'] = None
    additions.append(('null-output-limit', config, 2, None))
    configured.extend((name, 'gomad.wasi-cooperative/v1', config, expected, defaults)
                      for name, config, expected, defaults in additions)
    equal(len(configured), 27, 'configuration.clock_and_alias_case_count')
    views = []
    for name, profile, config, expected, defaults in configured:
        config_path = root / ('clock-' + name + '.json')
        config_path.write_bytes(encode(config))
        observed = []
        for lineage, binary, helper in [('go', args.go_binary, args.go_helper), ('rust', args.binary, None)]:
            output = root / (lineage + '-clock-' + name + '-installation.json')
            command = [binary, 'install', '--compiler', args.go_compiler,
                '--cache', args.preparation_root / (lineage + '-cache'), '--profile', profile,
                '--config', config_path, '--output', output, '--fuel', '123456789', '--memory-bytes', '134217728']
            if profile == 'gomad.wasi-cooperative/v1':
                command += ['--runtime-root', args.go_source / 'tools/gomad/internal/model/runtimesource']
            if helper:
                command += ['--helper', helper]
            result = run(f'{lineage}-clock-{name}-{root.name}', command, expected)
            observed.append(result.returncode)
            if expected:
                equal(output.exists(), False, name + '.no_installation')
            else:
                manifest = decode(output.read_bytes())
                if lineage == 'go':
                    go_config = manifest['options']['Config']
                    go_clock = go_config['Clock']
                    equal(go_clock, defaults, name + '.go_zero_defaults')
                else:
                    original = manifest['admission']['guest_configuration']
                    equal(original, config, name + '.original_configuration')
                    equal(manifest['admission']['configuration_sha256'], digest(encode(original)), name + '.configuration_preimage')
                    model_clock = dict(EpochNanos=manifest['model']['epoch_nanos'], ReadStepNanos=manifest['model']['read_step_nanos'])
                    equal(model_clock, go_clock, name + '.model_clock_parity')
                    equal(manifest['model']['entropy_key'], go_config['EntropyKey'], name + '.entropy_key_parity')
                    equal(manifest['admission']['arguments'], go_config['Args'] or [], name + '.arguments_parity')
                    equal(manifest['admission']['environment'], dict(item.split('=', 1) for item in go_config['Environment'] or []),
                          name + '.environment_parity')
        equal(observed, [expected, expected], name + '.status_parity')
        views.append(dict(case=name, status=expected, configuration=config,
            **(dict(go_clock=go_clock, model_clock=model_clock,
                    configuration_sha256=manifest['admission']['configuration_sha256'], admitted_configuration=original,
                    entropy_key=manifest['model']['entropy_key'], arguments=manifest['admission']['arguments'],
                    environment=manifest['admission']['environment'])
               if not expected else {})))
    return views


def captured_descriptor_cases(args, manifests, root, run, decode):
    original = decode(manifests['go', 'temporal'].read_bytes())['options']['Config']
    canonical_namespace = decode(manifests['rust', 'temporal'].read_bytes())['namespace']
    raw = base64.b64decode(original['CapturedInputs']['Descriptor'], validate=True)
    needle = b'"entries":'
    equal(raw.count(needle), 1, 'descriptor.entries_object_count')
    cases = [
        ('alias', raw.replace(needle, b'"Entries":', 1), 0),
        ('duplicate-alias', raw.replace(needle, b'"Entries":[],"entries":', 1), 2),
        ('delimiter', raw.replace(needle, b'"entr_ies":', 1), 2),
    ]
    views = []
    for name, descriptor, expected in cases:
        config = copy.deepcopy(original)
        config['CapturedInputs']['Descriptor'] = base64.b64encode(descriptor).decode()
        config['CapturedInputs']['Manifest']['Bytes'] = len(descriptor)
        config['CapturedInputs']['Manifest']['SHA256'] = 'sha256:' + digest(descriptor)
        config_path = root / ('descriptor-' + name + '.json')
        config_path.write_bytes(encode(config))
        statuses = []
        for lineage, binary, helper in [('go', args.go_binary, args.go_helper), ('rust', args.binary, None)]:
            output = root / (lineage + '-descriptor-' + name + '-installation.json')
            command = [binary, 'install', '--compiler', args.go_compiler,
                '--cache', args.preparation_root / (lineage + '-cache'), '--profile', 'gomad.wasi-cooperative/v1',
                '--runtime-root', args.go_source / 'tools/gomad/internal/model/runtimesource',
                '--config', config_path, '--output', output, '--fuel', str(1 << 50), '--memory-bytes', str(4 << 30)]
            if helper:
                command += ['--helper', helper]
            result = run(f'{lineage}-descriptor-{name}-{root.name}', command, expected)
            statuses.append(result.returncode)
            if expected:
                equal(output.exists(), False, name + '.no_installation')
            else:
                manifest = decode(output.read_bytes())
                if lineage == 'rust':
                    equal(manifest['admission']['guest_configuration'], config, name + '.original_configuration')
                    equal(manifest['admission']['configuration_sha256'], digest(encode(config)), name + '.configuration_preimage')
                    equal(manifest['namespace'], canonical_namespace, name + '.unchanged_namespace')
        equal(statuses, [expected, expected], name + '.descriptor_status_parity')
        views.append(dict(case=name, status=expected, configuration=config,
            descriptor_sha256=digest(descriptor), descriptor_bytes=len(descriptor),
            **(dict(admitted_configuration=manifest['admission']['guest_configuration'],
                    configuration_sha256=manifest['admission']['configuration_sha256'], namespace=manifest['namespace'])
               if not expected else {})))
    return views


def configuration_refusals(args, manifests, root, run, decode):
    original = decode(manifests['go', 'temporal'].read_bytes())['options']['Config']
    payload = copy.deepcopy(original)
    first = next(iter(payload['CapturedInputs']['Payloads']))
    payload['CapturedInputs']['Payloads'][first] = base64.b64encode(b'changed captured payload').decode()
    descriptor = copy.deepcopy(original)
    raw = base64.b64decode(descriptor['CapturedInputs']['Descriptor'])
    descriptor['CapturedInputs']['Descriptor'] = base64.b64encode(raw + b' ').decode()
    for case, config in [('payload-mismatch', payload), ('descriptor-mismatch', descriptor)]:
        config_path = root / (case + '.json')
        config_path.write_bytes(encode(config))
        for lineage, binary, helper in [('go', args.go_binary, args.go_helper), ('rust', args.binary, None)]:
            output = root / (lineage + '-' + case + '-installation.json')
            command = [binary, 'install', '--compiler', args.go_compiler,
                '--cache', args.preparation_root / (lineage + '-cache'), '--profile', 'gomad.wasi-cooperative/v1',
                '--runtime-root', args.go_source / 'tools/gomad/internal/model/runtimesource',
                '--config', config_path, '--output', output]
            if helper:
                command += ['--helper', helper]
            run(f'{lineage}-{case}-{root.name}', command, 2)
            equal(output.exists(), False, case + '.no_installation')
    checkout = root / 'symlink-checkout'
    schema = checkout / 'schema/sqlite'
    schema.mkdir(parents=True)
    (schema / 'untrusted.sql').symlink_to(args.go_source / 'schema/sqlite')
    for lineage, binary, helper in [('go', args.go_binary, args.go_helper), ('rust', args.binary, None)]:
        output = root / (lineage + '-symlink-installation.json')
        command = [binary, 'install', '--compiler', args.go_compiler, '--cache', args.preparation_root / (lineage + '-cache'),
            '--runtime-root', args.go_source / 'tools/gomad/internal/model/runtimesource',
            '--temporal-checkout', checkout, '--temporal-package', './tests/gomadfunctional', '--output', output]
        if helper:
            command += ['--helper', helper]
        run(f'{lineage}-schema-symlink-{root.name}', command, 2)
        equal(output.exists(), False, 'schema_symlink.no_installation')


def authenticate_captured(configuration):
    captured = configuration['CapturedInputs']
    equal(set(captured), {'Manifest', 'Descriptor', 'Payloads'}, 'captured.fields')
    raw = base64.b64decode(captured['Descriptor'], validate=True)
    manifest = captured['Manifest']
    equal(digest(raw), manifest['SHA256'].removeprefix('sha256:'), 'captured.descriptor_sha256')
    equal(len(raw), manifest['Bytes'], 'captured.descriptor_bytes')
    descriptor = json.loads(raw)
    expected = set()
    for entry in descriptor['entries']:
        if entry.get('payload'):
            payload = base64.b64decode(captured['Payloads'][entry['payload']], validate=True)
            equal(digest(payload), entry['sha256'].removeprefix('sha256:'), 'captured.payload_sha256.' + entry['path'])
            equal(len(payload), int(entry['size']), 'captured.payload_bytes.' + entry['path'])
            expected.add(entry['payload'])
    equal(set(captured['Payloads']), expected, 'captured.exact_payloads')


def go_recovery_check(args, installation, root, run, decode, kind):
    """Fresh Go campaign with Go-native CRC framing; never use Rust journals."""
    fixture = CORPUS.parents[1] / 'internal/backends/wasm/qualification/testdata/fixture'
    target = '.'
    artifacts = root / ('go-' + kind + '-recovery-artifacts')
    trace_flags = ['--choices'] if decode(installation.read_bytes())['options']['Config']['Profile'] == 'gomad.wasi-cooperative/v1' else []
    strategy = trace_flags if kind == 'seed' else ['--strategy=choice-exploration', '--choices', '--max-executions=1',
        '--max-choice-depth=1', '--max-exploration-bytes=1MiB']
    run(f'go-{kind}-recovery-explore-{root.name}', [args.go_binary, '--installation', installation,
        'explore', '--json', '--seeds=7', '--parallel=1', '--on-failure=all', '--keep-successes=all',
        '--success-limit=1', '--success-bytes=256MiB', '--execution-timeout=120s', '--overall-timeout=240s',
        '--artifacts', artifacts, '--working-dir', fixture, *strategy, 'go-run', target,
        '--', '-test.run=^TestPrepareOnlyNeverExecuted$'], 0)
    logs = list(artifacts.rglob('store/log'))
    equal(len(logs), 1, 'go.recovery.campaign_count')
    journal = logs[0]
    campaign = journal.parent.parent
    reference = runpy.run_path(str(CORPUS / 'campaign-reference-check.py'))
    authenticate = reference['authenticate_campaign'] if kind == 'seed' else lambda path: authenticate_go_round(reference, path)
    authenticated = authenticate(campaign)
    run(f'go-{kind}-recovery-inspect-{root.name}', [args.go_binary, '--installation', installation, 'inspect', '--json', campaign], 0)
    original = journal.read_bytes()
    with journal.open('ab') as writer:
        writer.write(b'GMLG')
    reports = []
    for attempt, changed in enumerate((True, False)):
        response = run(f'go-{kind}-recovery-{root.name}-{attempt}', [args.go_binary, '--installation', installation,
            'recover', '--json', campaign], 0)
        report = decode(response.stdout)
        equal(set(report) - {'action'}, {'schema', 'path', 'changed', 'before', 'after'}, 'go.recovery.fields')
        equal(report['changed'], changed, 'go.recovery.changed')
        equal(journal.read_bytes(), original, 'go.recovery.original_journal')
        authenticate(campaign)
        run(f'go-{kind}-recovery-inspect-{root.name}-{attempt}', [args.go_binary, '--installation', installation, 'inspect', '--json', campaign], 0)
        reports.append(dict(schema=report['schema'], action=report.get('action'), changed=report['changed'],
                            before=report['before'], after=report['after']))
    return dict(reports=reports, **(dict(rounds=authenticated) if kind == 'round' else {}))


def authenticate_go_round(reference, campaign):
    rows = reference['frames'](campaign, exploration=True)
    openings = [row['payload'] for row in rows if row['kind'] == 8]
    equal(len(openings), 1, 'go.round.opening_count')
    equal(openings[0]['strategy'], 'choice-exploration', 'go.round.strategy')
    chain, pending, rounds = openings[0]['initial_state_sha256'], None, []
    for row in rows:
        value = row['payload']
        if row['kind'] == 9:
            equal(pending, None, 'go.round.uncommitted_start')
            pending = value
        if row['kind'] != 10:
            continue
        segment = value['segment']
        equal(int(pending['round']), len(rounds), 'go.round.start_order')
        equal(int(value['round']['index']), len(rounds), 'go.round.order')
        equal(segment['index'], len(rounds), 'go.round.segment_order')
        equal(segment['before_sha256'], chain, 'go.round.state_chain')
        unsigned = dict(segment)
        claimed = unsigned.pop('sha256')
        # Go hashes its struct-field order and omits the empty sha256 member.
        raw = json.dumps(unsigned, separators=(',', ':'), ensure_ascii=False).encode()
        equal(claimed, 'sha256:' + digest(b'gomad-choice-exploration-round-segment/v2\0' + raw), 'go.round.segment_sha256')
        equal(len(value['executions']), int(pending['candidates']), 'go.round.execution_count')
        for execution in value['executions']:
            for key in ('artifact', 'success_artifact'):
                if execution.get(key):
                    path = Path(execution[key])
                    reference['authenticate_artifact'](path if path.is_absolute() else campaign / path)
        rounds.append(dict(index=segment['index'], before_sha256=segment['before_sha256'],
                           after_sha256=segment['after_sha256'], sha256=claimed, executions=len(value['executions'])))
        chain, pending = segment['after_sha256'], None
    equal(pending, None, 'go.round.final_uncommitted_start')
    equal(len(rounds) > 0, True, 'go.round.fresh_committed_round')
    return rounds


def project_manifest(manifest, root, args):
    paths = {'compiler_path', 'runtime_root', 'overlay_module_root', 'cache_root', 'compile_cache_root', 'helper_path'}
    kept = {'schema', 'compiler_sha256', 'go_version', 'profile', 'fuel', 'memory_bytes', 'helper_sha256', 'host_sha256',
            'runtime_implementation_sha256', 'model', 'namespace', 'storage', 'network', 'build_tags', 'admission'}
    equal(set(manifest) - paths, kept, 'manifest.declared_fields')
    result = dict(manifest)
    admission = dict(manifest['admission'])
    equal(set(admission), {'compiler_inputs', 'inputs_sha256', 'guest_configuration', 'configuration_sha256',
        'host_executable_sha256', 'explicit_helper', 'environment', 'arguments', 'compile_cache_root'}, 'admission.declared_fields')
    pinned = {Path(path) for path in admission['compiler_inputs']}
    equal(args.go_compiler in pinned, True, 'admission.compiler_executable_listed')
    equal({'VERSION', 'asm', 'compile', 'link'} <= {path.name for path in pinned}, True, 'admission.compiler_distribution_listed')
    equal(any(path.suffix == '.h' for path in pinned), True, 'admission.assembler_headers_listed')
    if manifest['profile'] == 'gomad.wasi-cooperative/v1':
        runtime = args.go_source / 'tools/gomad/internal/model/runtimesource'
        expected_runtime = {path for path in runtime.rglob('*') if path.is_file()}
        equal(bool(expected_runtime) and expected_runtime <= pinned, True, 'admission.complete_runtime_inventory')
    for path, expected in admission['compiler_inputs'].items():
        equal(digest(Path(path).read_bytes()), expected, 'admission.compiler_inputs.' + path)
    equal(digest(args.binary.read_bytes()), admission['host_executable_sha256'], 'admission.host_executable_bytes')
    equal(digest(encode(admission['compiler_inputs'])), admission['inputs_sha256'], 'admission.inputs_preimage')
    equal(digest(encode(admission['guest_configuration'])), admission['configuration_sha256'], 'admission.configuration_preimage')
    equal(admission['explicit_helper'], None, 'admission.default_linked_helper')
    if admission['compile_cache_root']:
        actual = Path(admission['compile_cache_root'])
        equal(actual.is_relative_to(args.preparation_root), True, 'admission.compile_cache_root.containment')
        admission['compile_cache_root'] = '$PREPARATION/' + str(actual.relative_to(args.preparation_root))
    result['admission'] = admission
    for key in paths & set(manifest):
        path = manifest[key]
        if path is None:
            continue
        if key == 'compiler_path':
            equal(path, str(args.go_compiler), 'manifest.compiler_path')
            result[key] = '$COMPILER'
        elif key == 'runtime_root':
            if path:
                equal(path, str(args.go_source / 'tools/gomad/internal/model/runtimesource'), 'manifest.runtime_root')
            result[key] = '$RUNTIME' if path else path
        elif key == 'overlay_module_root':
            if path:
                equal(Path(path).resolve(), CORPUS.parents[1], 'manifest.overlay_module_root')
            result[key] = '$MODULE' if path else path
        elif key == 'helper_path':
            result[key] = '$BINARY' if path else path
        elif path:
            actual = Path(path)
            prefix = args.preparation_root if actual.is_relative_to(args.preparation_root) else root
            equal(actual.is_relative_to(prefix), True, 'manifest.' + key + '.containment')
            result[key] = ('$PREPARATION/' if prefix == args.preparation_root else '$RUN/') + str(actual.relative_to(prefix))
    return result


if __name__ == '__main__':
    main()
