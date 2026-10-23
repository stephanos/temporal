#!/usr/bin/env python3
"""Execute root 7: aliased preparation and source-less runtime canary replay."""
import copy
import json
import os
from pathlib import Path
import runpy
import shutil

common = runpy.run_path(str(Path(__file__).with_name('live-backend-common.py')))
Lane, equal, encode, digest = (common[key] for key in ('Lane', 'equal', 'encode', 'digest'))
MODULE = common['MODULE']



def retire_preparation_aliases(work, stock_root):
    """Retire only authenticated private preparation aliases after detached replay."""
    aliases = []
    compiler = work / 'compiler-link'
    equal(compiler.is_symlink(), True, 'preparation.compiler_alias_present')
    equal(os.readlink(compiler), str(work / 'go'), 'preparation.compiler_alias_target')
    aliases.append(dict(path='compiler-link', target=os.readlink(compiler)))
    private = work / 'go-unavailable'
    equal(private.is_dir() and not private.is_symlink(), True, 'preparation.detached_distribution')
    for path in sorted(private.rglob('*')):
        if not path.is_symlink():
            continue
        original = stock_root / path.relative_to(private)
        equal(original.is_symlink(), True, 'preparation.original_distribution_alias')
        equal(os.readlink(path), os.readlink(original), 'preparation.original_alias_target')
        aliases.append(dict(path=str(path.relative_to(work)), target=os.readlink(path)))
    # Validate the complete set before changing any private alias.
    receipt = dict(schema='gomad.retired-preparation-aliases/v1',
                   after='runtime-detached-exact-replay', stock_root=str(stock_root), aliases=aliases)
    (work / 'retired-preparation-aliases.json').write_bytes(encode(receipt))
    for row in aliases:
        (work / row['path']).unlink()


def main():
    lane = Lane(__doc__)
    installation = lane.installation('runtime')
    selected = json.loads(installation.read_bytes())
    compiler = Path(selected['compiler_path'])
    stock_root = compiler.parent.parent
    private = lane.args.work / 'go'
    # Hard links provide private directory names without duplicating the pinned
    # compiler distribution. No file in either distribution is modified.
    shutil.copytree(stock_root, private, copy_function=os.link, symlinks=True)
    alias = lane.args.work / 'compiler-link'
    alias.symlink_to(private, target_is_directory=True)
    runtime = lane.args.work / 'runtime-owner'
    shutil.copytree(MODULE / 'internal/model/runtimesource', runtime)
    source = lane.args.work / 'source'
    source.mkdir(mode=0o700)
    (source / 'go.mod').write_text('module gomad.wasm.cooperative.fixture\n\ngo 1.27.1\n')
    shutil.copyfile(MODULE / 'internal/backends/wasm/testdata/runtime_probes/main.go', source / 'main.go')
    selected.update(compiler_path=str(alias / 'bin/go'), runtime_root=str(runtime))
    installation.write_bytes(encode(selected))
    preparations = []
    for index in range(2):
        destination = lane.args.work / ('prepare-' + str(index))
        output = lane.run('prepare-' + str(index), ['plan', '--json', '--output',
            lane.args.work / ('plan-' + str(index) + '.json'), '--artifacts', destination,
            '--seeds=7', '--parallel=1', '--execution-timeout=60s', '--overall-timeout=600s',
            '--output-limit=1048576', '--world-transition-limit=1048576', '--choice-bytes=1048576',
            '--choices', '--build-tag=test_dep', '--working-dir', source, 'go-run', '.'],
            installation=installation)
        plan = json.loads(output.stdout)
        equal(plan['guest_executions'], 0, 'preparation.no_execution')
        campaign = Path(plan['campaign'])
        retained = list(campaign.glob('prepared/*/target/provenance.json'))
        equal(len(retained), 1, 'preparation.single_target')
        provenance = json.loads(retained[0].read_bytes())
        module = retained[0].with_name('module.wasm').read_bytes()
        equal(digest(module), provenance['module_sha256'], 'preparation.module_hash')
        preparations.append((provenance, module))
    equal(preparations[0], preparations[1], 'preparation.complete_identity_reuse')
    original = preparations[0][0]
    # The historical alias inventory used the selected distribution name. Its
    # independent counterfactual must differ from canonical inventory identity.
    legacy = copy.deepcopy(original['inputs'])
    changed = 0
    for name in ('compiler', 'sources'):
        entries = legacy[name]
        rewritten = {}
        for path, value in entries.items():
            if path.startswith(str(private) + '/'):
                path = str(alias) + path[len(str(private)):]
                changed += 1
            rewritten[path] = value
        legacy[name] = rewritten
    equal(changed > 0, True, 'legacy_alias.inventory_rewritten')
    equal(digest(encode(legacy)) != original['build_key'], True, 'canonical_not_legacy_alias_key')
    seeds = ','.join(str(index * 1000003) for index in range(32))
    root, output = lane.explore('runtime-canary', ['go-run', '.'], source,
        ['--build-tag=test_dep', '--diagnostics'], installation=installation,
        seeds=seeds, choices=True, overall='600s', status=(1,), on_failure='first')
    report = lane.report(output)
    equal(report['failures'], 1, 'runtime.one_failure')
    artifacts = lane.artifacts(root)
    equal(len(artifacts), 1, 'runtime.one_artifact')
    artifact = artifacts[0]
    record = common['seed']['authenticate_artifact'](artifact)
    equal(record['outcome']['exit_code'], 17, 'runtime.canary_exit')
    equal(record['profile'], 'gomad.wasi-cooperative/v1', 'runtime.exact_profile')
    equal('evidence/choice' in record['streams'], True, 'runtime.choice_tape')
    equal('native' in ''.join(record['streams']), False, 'runtime.no_native_transcript')
    for directory in (source, private, runtime, Path(selected['cache_root'])):
        equal(directory.exists(), True, 'detachment.exists.' + directory.name)
        directory.rename(directory.with_name(directory.name + '-unavailable'))
    terminal = (artifact / 'evidence/runtime-terminal').read_bytes()
    equal(len(terminal), 16, 'runtime.terminal_bytes')
    equal(terminal[:4], (1).to_bytes(4, 'big'), 'runtime.terminal_version')
    lane.replay('runtime-detached-exact-replay', artifact)
    retire_preparation_aliases(lane.args.work, stock_root)
    lane.roots.append(dict(root=7, original='TestIntegratedCooperativeFailureArtifactDetachedReplay',
                           selected_seeds=[index * 1000003 for index in range(32)],
                           build_key=original['build_key'], legacy_alias_build_key=digest(encode(legacy))))
    lane.finish()


if __name__ == '__main__':
    main()
