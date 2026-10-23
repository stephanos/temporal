"""Execute ordinary native suites and the surviving original Go oracle selections.

Complete original callable/subtest equivalence remains the final reach reader's
obligation. A namespace result alone never supplies that coverage.
"""
import json
import re
from pathlib import Path

from driver import HERE, read, require, sha
import live
import reach
import static_stages as static

ROLES = {'host': ('ordinary-host-test', 'gomad'), 'helper': ('ordinary-helper-test', 'gomad_wasmhost')}
NAMESPACE = 'go/go.temporal.io/server/tools/gomad/internal/model/wasi/TestNamespaceConformanceCorpus'
WORKLOAD_TOOLS = 'ordinary-workload-tools-run'


def plan(config):
    path = live.option(config['stages']['validate']['argv'], '--plan')
    require(path, 'ordinary requires its reviewed static compiler plan')
    live.sealed_input(config, path)
    value = read(path)
    require(value['schema'] == 'gomad.driver-static-plan/v1' and value['repo'] == config['repo']
            and value['driver_config'] == config['_config_path']
            and value['stages']['validate'] == static.recipe(value, 'validate'),
            'ordinary static compiler recipe changed')
    return value


def preflight(config):
    value = plan(config)
    for name in ('go',):live.sealed_input(config, value['tools'][name])
    for role, _ in ROLES.values():live.role(config, role)
    live.sealed_input(config, HERE / 'ordinary_receiving.py')
    require(config['live_corpus']['original_reference_commit'] == '1d695061b6382009d0659238c652a5a765b1d5f5',
            'ordinary original Make source commitment changed')


def recipes(config):
    value = plan(config)
    environment = static.command_environment(value, config)
    reference = static.reference_environment(value, config)
    rows = []
    for name, (role, _) in ROLES.items():
        binary = live.role(config, role)['binary']
        for suffix, arguments in (('list', ['--list']), ('ignored', ['--list', '--ignored']),
                                  ('run', ['--test-threads=1'])):
            rows.append(dict(name='ordinary-' + name + '-' + suffix, argv=[binary, *arguments],
                             environment=environment, timeout_s=60 if suffix != 'run' else 900))
    go = value['tools']['go']
    module = str(Path(config['repo']) / 'tools/gomad')
    groups = [
        ('namespace', module, '^(TestNamespaceConformance|TestVolumeConformance)',
         ['./internal/model/wasi'], 'test_dep', 600, True),
        ('builder', module, '^TestRetainedWASMSourcesDoNotImportNativeBuilder$',
         ['./internal/archcheck'], 'test_dep', 900, False),
        ('recursive', module, '.',
         list(static.RETAINED), 'test_dep', 900, True),
        ('recursive-builder', module, '^TestRetainedWASMSourcesDoNotImportNativeBuilder$',
         ['./internal/archcheck'], 'test_dep', 900, False),
        ('slim', config['repo'], '|'.join('^' + name + '$' for name in live.SLIM_SOURCES.values()),
         ['./common/archiver/provider', './temporal', './tests/testcore'], 'gomad_wasm_slim,test_dep', 900, False),
    ]
    for name, working, selector, packages, tags, timeout, needs_reference in groups:
        prefix = [go, '-C', working, 'test', '-json', '-tags', tags, '-count=1', f'-timeout={timeout}s', '-run', selector]
        env = dict(environment, **(reference if needs_reference else {}))
        for suffix, flags in (('list', ['-list', selector]), ('run', [])):
            skip = ['-skip', '^TestNamespaceConformance'] if name == 'recursive' else []
            rows.append(dict(name='ordinary-' + name + '-' + suffix, argv=[*prefix, *flags, *skip, *packages],
                             environment=env, timeout_s=timeout))
    # The receiving Python tests of the qualification controls (guest
    # profiles, build cache, memory admission) have no other executing owner.
    live.sealed_input(config, config['python'])
    rows.append(dict(name=WORKLOAD_TOOLS, argv=[config['python'], '-m', 'unittest', 'discover', '-v', '-s',
                     str(Path(config['repo']) / 'tools/gomad/rust/workload_tools'), '-p', 'test_*.py'],
                     environment=dict(environment, PYTHONDONTWRITEBYTECODE='1'), timeout_s=300))
    return rows


def python_cases(raw):
    lines = raw.decode().splitlines()
    passed = [match.group(1) for line in lines
              if (match := re.fullmatch(r'\w+ \(([\w.]+)\) \.\.\. ok', line))]
    ran = [int(match.group(1)) for line in lines if (match := re.fullmatch(r'Ran (\d+) tests? in [0-9.]+s', line))]
    require(passed and len(set(passed)) == len(passed) and ran == [len(passed)] and lines[-1] == 'OK',
            'ordinary workload-tool tests did not all pass')
    return passed


def collect(config, root):
    actual, deferred = set(), []
    for name, (_, target) in ROLES.items():
        prefix = 'ordinary-' + name
        expected = static.test_names((root / (prefix + '-list.stdout')).read_bytes())
        ignored = static.test_names((root / (prefix + '-ignored.stdout')).read_bytes())
        require(expected and ignored <= expected, 'ordinary native listing is empty or changed')
        passed = static.verify_rust((root / (prefix + '-run.stdout')).read_bytes(), expected, ignored)
        require(passed, 'ordinary native suite selected no actual tests')
        for case in passed:
            identity = 'rust/' + target + '/' + case
            actual.update((identity, identity + '@stage/ordinary@run/' + prefix + '-run'))
        deferred.extend(dict(target=target, test=case, executed=False) for case in sorted(ignored))
    for case in python_cases((root / (WORKLOAD_TOOLS + '.stderr')).read_bytes()):
        identity = 'py/workload_tools/' + case
        actual.update((identity, identity + '@stage/ordinary@run/' + WORKLOAD_TOOLS))
    selections = {}
    for name in ('namespace', 'builder', 'recursive', 'recursive-builder', 'slim'):
        prefix = 'ordinary-' + name
        expected = static.parse_go_inventory((root / (prefix + '-list.stdout')).read_bytes())
        if name == 'recursive':
            expected = {key: value for key, value in expected.items() if not key[1].startswith('TestNamespaceConformance')}
        raw = (root / (prefix + '-run.stdout')).read_bytes()
        roots = static.verify_go(raw, expected, 1)
        identities = set()
        for line in raw.decode().splitlines():
            row = json.loads(line)
            if row.get('Action') == 'pass' and row.get('Test'):
                identity = 'go/' + row['Package'] + '/' + row['Test']
                identities.add(identity)
                actual.update((identity, identity + '@stage/ordinary@run/' + prefix + '-run'))
        require(roots and identities, 'ordinary Go selection executed no roots')
        selections[name] = sorted(identities)
    require(NAMESPACE in selections['namespace'] and live.SLIM_IDS <= set(selections['slim']),
            'ordinary original namespace/slim selections are incomplete')
    require(set(selections['builder']) == set(selections['recursive-builder'])
            and {case for case in selections['namespace'] if '/TestVolumeConformance' in case}
            <= set(selections['recursive']),
            'ordinary duplicated original Make selections lost their distinct runs')
    original = reach.inventory(config['repo'], config['live_corpus']['original_reference_commit'])
    repeated = {name: row['original_runs'] for name, row in original['callables'].items() if len(row['original_runs']) > 1}
    require(len(repeated) == 4, 'original repeated Make selection catalog changed')
    return dict(schema='gomad.ordinary-native-receiving/v1', actual_cases=sorted(actual), deferred=deferred,
                original_counts=original['counts'], original_repeated_selections=repeated, selections=selections,
                complete_original_reach='requires final independently reopened full callable/subtest mapping')


def final_read(config):
    require(isinstance(config.get('root'), str), 'original ordinary stage is not bound')
    import primary_stages
    root = primary_stages.completed_stage(config, 'ordinary')
    report = primary_stages.read_stage(config, 'ordinary', root)
    return report['ordinary_receiving']
