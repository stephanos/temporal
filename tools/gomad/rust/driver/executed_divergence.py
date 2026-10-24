#!/usr/bin/env python3
"""Produce and independently reopen the original four executed divergence campaigns."""
import argparse
import os
from pathlib import Path
import re
import time

from driver import HERE, check_files, command, configured, read, require, sha, write
import live
import main_cli_build
import installed_cluster_cache
from installed_cluster_failure import command_identity

CHECK = live.load('campaign-check')
SCHEMA = 'gomad.original-executed-divergence/v1'
PREPARE = 'campaign::counterfactual::prepare_the_four_original_counterfactual_requests'
PRODUCE = 'campaign::counterfactual::produce_the_four_original_counterfactual_artifact_pairs'
CASES = {'draw': 'draw-counter', 'clock': 'clock', 'handoff': 'world-order', 'output-clock': 'output'}


def roles(cli_path, harness_path, frozen=None):
    cli = main_cli_build.read_build(cli_path, frozen=frozen, expected_role='counterfactual-fixture')
    harness = main_cli_build.read_build(harness_path, frozen=frozen, expected_role='counterfactual-harness')
    for key in ('source_sha256', 'head', 'symlinks', 'host_sha256', 'helper_sha256', 'runtime_implementation_sha256', 'context'):
        require(cli[key] == harness[key], 'counterfactual public CLI and libtest differ: ' + key)
    require(cli['binary'] != harness['binary'] and cli['binary_sha256'] != harness['binary_sha256'],
            'counterfactual public CLI and libtest were conflated')
    return cli, harness


def environment(root, compiler):
    return dict(PATH=str(compiler.parent) + ':/usr/bin:/bin', HOME=str(root / 'home'), TMPDIR=str(root / 'tmp'),
                LC_ALL='C', TZ='UTC', GOENV='off', GOTOOLCHAIN='local', GOWORK='off',
                GOPROXY='off', GOSUMDB='off', CGO_ENABLED='0', RUST_TEST_THREADS='1',
                GOMODCACHE=str(root / 'module-cache'))


def invocation(binary, name):
    return [binary, '--ignored', '--exact', name, '--nocapture', '--test-threads=1']


def retained_tree(root):
    result = {}
    for path in sorted(root.rglob('*')):
        relative = path.relative_to(root)
        if relative.parts[0] in ('home', 'tmp') or relative.parts[:2] == ('prepared', 'cache'):
            continue
        require(not path.is_symlink(), 'symlink in original divergence receiving evidence')
        if path.is_file() and relative.as_posix() != 'result.json':
            result[relative.as_posix()] = dict(bytes=path.stat().st_size, sha256=sha(path))
    return result


def command_result(root, name, argv, env, owner, accepted, seen):
    row = read(root / (name + '.command.json'))
    process = read(root / (name + '.process.json'))
    require(type(row['exit']) is int and row['exit'] in accepted, 'counterfactual command exit changed')
    command_identity(row, process, owner, argv, owner['repo'], row['exit'], env, seen)
    output = (root / (name + '.stdout')).read_bytes()
    selected_test(output, row['exit'])
    return row


def selected_test(output, exit_code):
    summaries = re.findall(rb'^test result: (ok|FAILED)\. ([0-9]+) passed; ([0-9]+) failed; ([0-9]+) ignored;', output, re.MULTILINE)
    expected = [(b'ok', b'1', b'0', b'0')] if exit_code == 0 else [(b'FAILED', b'0', b'1', b'0')]
    require(summaries == expected, 'counterfactual invocation did not execute exactly its selected libtest')


def inspection(result, report):
    require(result['status'] == 'inspected_prefix' and result['prefix_verified'] is True
            and result['witness_match'] is True
            and result['resolved']['choice'] == report['reproducer']['stop']['stop']['value'],
            'original public/provider prefix replay failed')


def dependency_cache(root, recipe, repo):
    require(installed_cluster_cache.verify(root / 'module-cache', repo / 'tools/gomad/go.sum') == recipe['module_cache_inventory']
            and live.inventory(root / 'module-cache') == recipe['module_cache_inventory'],
            'original private dependency cache changed')


def setup(root, cli, cold):
    source = read(root / 'prepared/producer-input.json')
    require(source['schema'] == 'gomad.core-counterfactual-producer/v2'
            and len(source['cases']) == 4 and {case['name'] for case in source['cases']} == set(CASES),
            'original setup did not construct four recipe-only cases')
    caches = []
    for case in source['cases']:
        require(set(case) == {'name', 'campaign', 'original_requested_choice_bytes'},
                'original setup supplied a precompiled preparation receipt')
        installation = read(root / 'prepared' / (case['name'] + '-installation.json'))
        cache = root / 'prepared/cache' / case['name']
        require(installation['cache_root'] == str(cache)
                and installation['host_sha256'] == cli['host_sha256']
                and installation['helper_sha256'] == cli['helper_sha256']
                and case['campaign']['installation'] == str(root / 'prepared' / (case['name'] + '-installation.json')),
                'original case installation/cache/compiled role changed')
        caches.append(str(cache))
    require(len(set(caches)) == 4, 'original fault cases share a prewarmed guest compilation cache')
    if cold:
        require(not list((root / 'prepared').rglob('*.wasm')), 'original setup compiled a guest outside its baseline deadline')
    return dict(schema='gomad.original-counterfactual-setup/v1', guest_compilations=0, guest_executions=0, case_caches=caches)


def pairs(root, cli):
    report = read(root / 'pairs/artifact-pairs.json')
    require(report['schema'] == 'gomad.core-counterfactual-artifact-pairs/v1' and report['qualification'] is False
            and report['source_role'] == 'determinism-fixture' and report['cli'] == cli['binary']
            and report['cli_sha256'] == cli['binary_sha256'] and report['helper'] == cli['helper_sha256'],
            'counterfactual producer supplied another role or qualification claim')
    rows = report['pairs']
    require(len(rows) == 4 and {row['name'] for row in rows} == set(CASES), 'original four fixture cases changed')
    failures = []
    for row in rows:
        case = row['name']
        case_failures = []
        require(row['original_requested_choice_bytes'] == (0 if case == 'handoff' else 4 << 20),
                'original campaign choice retention bound changed')
        for role in ('baseline', 'fault'):
            cell = row[role]
            require({'artifact', 'campaign', 'campaign_result', 'wall_nanos', 'preparation_receipt', 'preparation_sha256'} <= set(cell),
                    'historical direct-run pairs cannot prove original campaign accounting')
            artifact, campaign = Path(cell['artifact']), Path(cell['campaign'])
            require(artifact.is_relative_to(root / 'pairs' / case) and campaign.is_relative_to(root / 'pairs' / case),
                    'counterfactual artifact/campaign borrowed another producer')
            retained = CHECK.authenticate_campaign(campaign)
            plan, executions = retained['plan'], retained['executions']
            require(len(executions) == 1 and executions[0]['seed'] == 7 and executions[0]['ordinal'] == 0
                    and executions[0]['record']['outcome']['class'] == 'target_failure'
                    and campaign / executions[0]['artifact'] == artifact,
                    'original one-failure/one-artifact campaign ledger changed')
            require(plan['selection'] == '7' and plan['parallel'] == 1 and plan['on_failure'] == 'first'
                    and plan['failure_budget'] == 1 and plan['overall_nanos'] == 300_000_000_000
                    and plan['terminate_grace_nanos'] == 100_000_000,
                    'original campaign deadline or failure policy changed')
            result = read(campaign / 'original-explore-result.json')
            require(result == cell['campaign_result'] and result['attempted'] == result['failures'] == 1
                    and len(result['artifacts']) == 1 and 0 < cell['wall_nanos'] <= 300_000_000_000,
                    'original Explore result or preparation-inclusive deadline changed')
            record = CHECK.authenticate_artifact(artifact)
            require(record['identity']['target'] == plan['module_sha256'] and record['seed'] == 7,
                    'original retained artifact differs from campaign execution')
            preparation_path = Path(cell['preparation_receipt'])
            require(preparation_path.is_relative_to(root / 'pairs' / case)
                    and sha(preparation_path) == cell['preparation_sha256'], 'original actual preparation receipt changed')
            prepared = read(preparation_path)
            retained_preparation = CHECK.authenticate_campaign(Path(prepared['campaign']), complete=False)
            require(prepared['schema'] == 'gomad.rust-preparation-receipt/v1' and prepared['guest_executions'] == 0
                    and not retained_preparation['executions'] and prepared['module_sha256'] == record['identity']['target']
                    and retained_preparation['plan']['configuration'] == plan['configuration']
                    and 0 <= prepared['elapsed_nanos'] <= cell['wall_nanos']
                    and prepared['process_id'] == read(root / 'produce.command.json')['pid'],
                    'original baseline deadline excludes its actual guest preparation')
            diagnostics = [(name, stream) for name, stream in record['streams'].items() if name.endswith('/diagnostic')]
            require(len(diagnostics) == (2 if case == 'handoff' else 1)
                    and all(stream['bytes'] > 0 for _, stream in diagnostics),
                    'original fixture lost per-incarnation nonempty diagnostics')
            if case == 'handoff':
                summary = read(artifact / 'evidence/cluster_summary')
                require(len(summary['incarnations']) == 2
                        and {(item['handle']['node'], item['handle']['incarnation']) for item in summary['incarnations']}
                        == {('a', 1), ('b', 1)}, 'original handoff owners changed')
                for item in summary['incarnations']:
                    if item['termination'] != 'exit':
                        case_failures.append(dict(original_root='TestDeterminismExecutedDivergenceArtifacts/handoff',
                            predicate='every reported incarnation termination equals exit', artifact=str(artifact),
                            handle=item['handle'], expected='exit', actual=item['termination'], observation=item))
            else:
                require(record['outcome']['termination'] == 'exit' and record['outcome']['exit_code'] == 1,
                        'original single-owner fixture did not exit one')
        difference = read(root / 'pairs' / case / 'first-divergence.json')
        require(difference == row['verification']['report'] and difference['status'] == 'diverged'
                and difference['ordering'] == 'helper-event' and difference['first']['kind'] == CASES[case]
                and difference['first']['instance'] and difference['first']['incarnation'] == 1
                and difference['first']['coordinate']['Shared'] > 0
                and difference['first']['expected'] != difference['first']['actual'],
                'original first-divergence predicates changed')
        inspection(read(root / 'pairs' / case / 'provider-inspection.json'), difference)
        replay = read(root / 'pairs' / case / 'public-replay.json')
        require(replay['command'] == difference['reproducer']['replay_command']
                and replay['command'][0] == cli['binary'] and row['baseline']['artifact'] in replay['command']
                and type(replay['exit_code']) is int and replay['exit_code'] == 0
                and replay['cli_sha256'] == cli['binary_sha256'], 'literal emitted original public replay changed')
        stdout = root / 'pairs' / case / 'public-replay.stdout'
        stderr = root / 'pairs' / case / 'public-replay.stderr'
        require(sha(stdout) == replay['stdout_sha256'] and sha(stderr) == replay['stderr_sha256'],
                'original public replay raw streams changed')
        inspection(read(stdout), difference)
        original = read(root / 'pairs' / case / 'original-outcome-predicates.json')
        require(original['failures'] == case_failures
                and original['status'] == ('FAIL' if case_failures else 'PASS'),
                'original outcome disposition differs from authenticated incarnation results')
        failures.extend(case_failures)
    require(failures == report['original_predicate_failures']
            and report['status'] == ('FAIL_ORIGINAL_PREDICATE' if failures else 'PASS'),
            'original failed outcome predicates were hidden')
    return report


def verify(path, frozen=None):
    path = Path(path)
    result = read(path)
    root = path.parent
    recipe = read(root / 'recipe.json')
    require(path == root / 'result.json' and result['schema'] == SCHEMA and result['qualification'] is False,
            'foreign original divergence receipt')
    import dry_run
    require(not any(dry_run.is_record(path) for path in (recipe['cli_build'], recipe['harness_build']))
            or (frozen is not None and frozen.get('dry_run') is True),
            'counterfactual dry-run roles require an explicit verified configuration')
    cli, harness = roles(recipe['cli_build'], recipe['harness_build'], frozen=frozen)
    require(result['files'] == retained_tree(root), 'original divergence retained files changed')
    check_files(recipe['inputs'])
    compiler = Path(recipe['compiler'])
    require(live.inventory(compiler.parent.parent) == recipe['compiler_inventory'], 'original stock compiler tree changed')
    dependency_cache(root, recipe, (Path(read(cli['tools_plan'])['repo']) if cli['tools_plan'] else HERE.parents[3]))
    owner = read(root / 'producer.json')
    require(read(root / 'before-produce.json') == setup(root, cli, False), 'original cold setup disposition changed')
    env = environment(root, compiler)
    first = dict(env, GOMAD_CORE_COUNTERFACTUAL_PREPARATION=str(root / 'preparation-recipe.json'))
    second = dict(env, GOMAD_CORE_COUNTERFACTUAL_INPUT=str(root / 'prepared/producer-input.json'),
                  GOMAD_CORE_COUNTERFACTUAL_ROOT=str(root / 'pairs'))
    seen = set()
    command_result(root, 'prepare', invocation(harness['binary'], PREPARE), first, owner, (0,), seen)
    report = pairs(root, cli)
    executed = command_result(root, 'produce', invocation(harness['binary'], PRODUCE), second, owner, (0, 101), seen)
    require(executed['exit'] == (101 if report['original_predicate_failures'] else 0)
            and result['original_status'] == report['status'], 'producer exit contradicts original predicates')
    return result


def produce(cli_build, harness_build, compiler_root, module_cache, root, frozen=None):
    import dry_run
    require(not any(dry_run.is_record(path) for path in (cli_build, harness_build))
            or (frozen is not None and frozen.get('dry_run') is True),
            'counterfactual dry-run roles require an explicit verified configuration')
    cli, harness = roles(cli_build, harness_build, frozen=frozen)
    require(root.is_absolute() and root.resolve() == root and not root.exists(), 'original divergence root must be fresh/canonical')
    repo = (Path(read(cli['tools_plan'])['repo']) if cli['tools_plan'] else HERE.parents[3])
    require(not root.is_relative_to(repo), 'original divergence evidence must remain outside source')
    compiler = compiler_root / 'bin/go'
    require(compiler_root.is_absolute() and compiler.is_file() and not compiler.is_symlink(), 'explicit stock compiler required')
    inputs = {}
    for path, build in ((cli_build, cli), (harness_build, harness)):
        for name, digest in {**build['inputs'], str(path): sha(path)}.items():
            require(name not in inputs or inputs[name] == digest, 'counterfactual build inputs disagree')
            inputs[name] = digest
    root.mkdir(mode=0o700)
    for name in ('home', 'tmp'):(root / name).mkdir(mode=0o700)
    dependency_inventory = installed_cluster_cache.snapshot(module_cache, root / 'module-cache', repo / 'tools/gomad/go.sum')
    write(root / 'producer.json', dict(pid=os.getpid(), repo=str(repo), start_ns=time.time_ns(), started_monotonic_nanos=time.monotonic_ns()))
    write(root / 'recipe.json', dict(cli_build=str(cli_build), harness_build=str(harness_build), compiler=str(compiler),
                                  compiler_inventory=live.inventory(compiler_root), inputs=inputs,
                                  module_cache_source=str(module_cache), module_cache_inventory=dependency_inventory))
    write(root / 'preparation-recipe.json', dict(schema='gomad.original-counterfactual-recipe/v1', root=str(root / 'prepared'),
          compiler=str(compiler), compiler_sha256=sha(compiler), cli=cli['binary'], cli_sha256=cli['binary_sha256']))
    config = dict(repo=str(repo))
    env = environment(root, compiler)
    command(config, root, 'prepare', invocation(harness['binary'], PREPARE), timeout=1200,
            exact_environment=dict(env, GOMAD_CORE_COUNTERFACTUAL_PREPARATION=str(root / 'preparation-recipe.json')))
    check_files(inputs)
    dependency_cache(root, read(root / 'recipe.json'), repo)
    write(root / 'before-produce.json', setup(root, cli, True))
    command(config, root, 'produce', invocation(harness['binary'], PRODUCE), accepted=(0, 101), timeout=3000,
            exact_environment=dict(env, GOMAD_CORE_COUNTERFACTUAL_INPUT=str(root / 'prepared/producer-input.json'),
                                   GOMAD_CORE_COUNTERFACTUAL_ROOT=str(root / 'pairs')))
    check_files(inputs)
    report = pairs(root, cli)
    path = root / 'result.json'
    write(path, dict(schema=SCHEMA, qualification=False, original_status=report['status'], files=retained_tree(root)))
    verify(path, frozen=frozen)
    return path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='mode', required=True)
    producer = sub.add_parser('produce')
    for name in ('cli-build', 'harness-build', 'compiler-root', 'module-cache', 'root'):
        producer.add_argument('--' + name, type=Path, required=True)
    reader = sub.add_parser('verify')
    reader.add_argument('--receipt', type=Path, required=True)
    for endpoint in (producer, reader):
        endpoint.add_argument('--config', type=Path)
    args = parser.parse_args()
    frozen = None
    if args.config is not None:
        frozen = configured(args.config)
        frozen.update(_config_path=str(args.config.resolve()), _config_sha256=sha(args.config))
    print(produce(args.cli_build, args.harness_build, args.compiler_root, args.module_cache, args.root, frozen=frozen)
          if args.mode == 'produce' else verify(args.receipt, frozen=frozen)['original_status'])


if __name__ == '__main__':
    main()
