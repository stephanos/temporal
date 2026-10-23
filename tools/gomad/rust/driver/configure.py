#!/usr/bin/env python3
"""Freeze a receiver driver configuration; does not run a qualification gate."""
import argparse
import os
from pathlib import Path
import subprocess
import sys

from driver import FROZEN, HERE, REPO, Unexecuted, deadline, mapping, read, require, sha, write
from adapters import CELLS


def cell_spec(label, evidence, manifest, source_digest):
    wanted = CELLS[label]
    onebox = wanted['workload'] == 'onebox'
    if wanted['workload'] not in ('fixture', 'onebox'):
        raise Unexecuted(label + ': actual durable catalogue/history scenario must come from its Rust owner')
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


def freeze(bindings, output, source_manifest):
    config = dict(bindings)
    deadline(config['deadline_s'])
    from host_admission import STAT, bindings as scratch_bindings
    scratch_bindings(config)
    require(set(config['stages']) == set(mapping()), 'all 96 stage mappings are required')
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
    inputs.update((config['binary'], config['build_receipt']))
    for index in range(32):
        entry = config['stages'][f'runtime-{index}']
        require(entry['adapter'] == 'runtime', 'runtime stage requires its exact release-library gate')
        inputs.update((entry['harness'], entry['harness_build_receipt']))
    for label, wanted in CELLS.items():
        entry = config['stages'][label]
        if entry.get('spec') == 'GENERATE':
            path = output.parent / (label + '.json')
            specification = cell_spec(label, str(Path(config['root']) / 'cells' / label), str(source_manifest), config['source_sha256'])
            write(path, specification)
            entry['spec'] = str(path)
            inputs.add(str(path))
    if config.get('fn222'):
        argv = config['fn222']['argv']
        config['fn222']['argv'] = argv = [str(output) if item == 'FROZEN_CONFIG' else
            str(source_manifest) if item == 'SOURCE_MANIFEST' else item for item in argv]
        require('--support' in argv and '--captured-config' in argv, 'fn222 receiver and captured policy inputs must be explicit')
        inputs.update([argv[0], argv[1], argv[argv.index('--support') + 1], argv[argv.index('--captured-config') + 1]])
    config['files'] = {str(Path(path).resolve()): sha(path) for path in sorted(inputs)}
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
