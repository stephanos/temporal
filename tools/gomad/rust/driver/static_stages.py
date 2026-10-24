#!/usr/bin/env python3
"""Assemble and authenticate the first 24 original qualification stage bindings."""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import sys

from driver import HERE, REPO, check_files, command, mapping, read, require, sha, subgates, write

LABELS = tuple(mapping())[:24]
TOOLS = {'python', 'go', 'cargo', 'rustc', 'rustfmt', 'make', 'golangci', 'errortype', 'true', 'rustdoc', 'cargo_clippy', 'clippy_driver', 'cargo_fmt'}
CACHES = {'cargo_target', 'cargo_home', 'go_cache', 'go_mod_cache'}
RETAINED = ['./internal/model/...', './internal/backends/wasm/cluster',
            './internal/backends/wasm/toolchain/...', './internal/support/...']
HOST_FEATURES = ['', 'determinism-fixture', 'helper-stress-jitter', 'helper-determinism-fixture',
                 'determinism-fixture,helper-stress-jitter', 'determinism-fixture,helper-determinism-fixture']
HELPER_FEATURES = ['reference-helper', 'conformance', 'reference-helper,diagnostic-profile',
                   'reference-helper,diagnostic-fuel-off', 'reference-helper,guest-profile',
                   'reference-helper,guest-profile,diagnostic-fuel-off', 'reference-helper,stress-jitter']
MODES = dict(zip(LABELS[16:22], ['reference-helper', 'conformance',
    'reference-helper,diagnostic-profile', 'reference-helper,diagnostic-fuel-off',
    'reference-helper,guest-profile', 'reference-helper,guest-profile,diagnostic-fuel-off']))


def regular(path):
    path = Path(path)
    require(path.is_absolute() and path.is_file() and not path.is_symlink(), 'nonregular static input: ' + str(path))
    return path


def rust_code(text):
    """Keep token positions while excluding comments and string test fixtures."""
    result, at = list(text), 0
    character_pattern = re.compile(r"'(?:\\(?:x[0-9a-fA-F]{2}|u\{[0-9a-fA-F_]+\}|.)|[^'\\\n])'")
    raw_pattern = re.compile(r'(?:br|r)(#{0,255})"')
    def erase(start, end):
        result[start:end] = ['\n' if char == '\n' else ' ' for char in text[start:end]]
    while at < len(text):
        if text.startswith('//', at):
            end = text.find('\n', at)
            end = len(text) if end < 0 else end
            erase(at, end); at = end
        elif text.startswith('/*', at):
            end, depth = at + 2, 1
            while end < len(text) and depth:
                if text.startswith('/*', end): depth += 1; end += 2
                elif text.startswith('*/', end): depth -= 1; end += 2
                else: end += 1
            require(depth == 0, 'unterminated Rust comment in ignored inventory')
            erase(at, end); at = end
        else:
            character = character_pattern.match(text, at)
            raw = raw_pattern.match(text, at)
            if character:
                end = at + len(character[0])
                erase(at, end); at = end
            elif raw:
                closing = '"' + raw[1]
                end = text.find(closing, at + len(raw[0]))
                require(end >= 0, 'unterminated Rust raw string in ignored inventory')
                end += len(closing)
                erase(at, end); at = end
            elif text[at] == '"':
                end = at + 1
                while end < len(text):
                    if text[end] == '\\': end += 2
                    elif text[end] == '"': end += 1; break
                    else: end += 1
                require(end <= len(text) and text[end - 1] == '"', 'unterminated Rust string in ignored inventory')
                erase(at, end); at = end
            else: at += 1
    return ''.join(result)


def ignored_attributes(text):
    code = rust_code(text)
    rows = []
    for match in re.finditer(r'#\s*\[\s*ignore\b[^\]]*\]', code):
        attribute = text[match.start():match.end()]
        reason = re.fullmatch(r'#\s*\[\s*ignore\s*=\s*("(?:[^"\\]|\\.)*")\s*\]', attribute, re.S)
        require(reason is not None, 'ignored test requires an explicit registered reason')
        name = re.search(r'\bfn\s+(\w+)\s*\(', code[match.end():])
        require(name is not None, 'ignored attribute has no function')
        rows.append((name[1], json.loads(reason[1])))
    return rows


def ignored_inventory(repo):
    declared = read(HERE / 'static-ignored.json')['tests']
    require(read(HERE / 'static-ignored.json')['schema'] == 'gomad.driver-static-ignored/v1', 'unknown ignored schema')
    expected = {(row['source'], row['name'], row['reason']) for row in declared}
    observed = set()
    attributes = 0
    for crate in ('gomad', 'wasmhost'):
        for folder in ('src', 'tests'):
            for path in (repo / 'tools/gomad/rust' / crate / folder).rglob('*.rs'):
                rows = ignored_attributes(path.read_text())
                attributes += len(rows)
                for name, reason in rows:
                    observed.add((str(path.relative_to(repo / 'tools/gomad/rust')), name, reason))
    require(attributes == len(declared) and expected == observed and len(expected) == len(declared), 'unknown/changed/missing ignored test disposition')
    import broad_harness
    for row in declared:
        if 'broad_routes' not in row:
            continue
        routes = row['broad_routes']
        require(isinstance(routes, list) and routes, 'ignored broad test lacks a closed actual route')
        expected_routes = []
        for stage in broad_harness.Stage:
            for invocation in broad_harness.catalog(stage):
                target = broad_harness.target(invocation.role)
                if target.crate != row['crate'] or not invocation.ignored:
                    continue
                selected = (invocation.selector == row['name'] or invocation.selector.endswith('::' + row['name'])) if invocation.exact else (
                    row['source'].endswith('/' + invocation.selector.replace('::', '/') + '.rs'))
                if selected:
                    expected_routes.append(dict(stage=stage.value, invocation=invocation.name,
                                                selector=invocation.selector, features=list(target.features)))
        require(routes == expected_routes and expected_routes, 'ignored broad test changed its fixed executable, features or final selectors')

    final_ids = set(read(HERE / 'required-final.json')['subgates'])
    for row in declared:
        require(row['final_id'] is None or row['final_id'] in final_ids, 'unknown receiving obligation')
    return declared


def static_plan(inputs):
    require(set(inputs) == {'schema', 'repo', 'driver_config', 'tools', 'caches', 'lint_base'}, 'static input fields differ')
    require(inputs['schema'] == 'gomad.driver-static-inputs/v1', 'wrong static input schema')
    require(Path(inputs['repo']).resolve() == REPO, 'static assembler must bind this consolidated checkout')
    require(set(inputs['tools']) == TOOLS and set(inputs['caches']) == CACHES, 'missing/unexpected tools or caches')
    require(re.fullmatch('[0-9a-f]{40}', inputs['lint_base']) is not None, 'lint base must be an exact commit')
    files, tools = {}, {}
    for name, binding in inputs['tools'].items():
        require(set(binding) == {'path', 'sha256'}, 'tool binding fields differ')
        path = regular(binding['path'])
        require(sha(path) == binding['sha256'], 'changed tool: ' + name)
        files[str(path)] = binding['sha256']
        tools[name] = str(path)
    rust_parent = Path(tools['cargo']).parent
    require(all(Path(tools[key]).parent == rust_parent for key in ('rustc', 'rustdoc', 'rustfmt', 'cargo_clippy', 'clippy_driver', 'cargo_fmt')),
            'Rust toolchain tools must share their pinned tool directory')
    require(Path(tools['cargo_clippy']).name == 'cargo-clippy' and Path(tools['clippy_driver']).name == 'clippy-driver'
            and Path(tools['cargo_fmt']).name == 'cargo-fmt',
            'Clippy sibling discovery requires original tool names')
    for path in inputs['caches'].values():
        require(Path(path).is_absolute() and Path(path).resolve() == Path(path) and Path(path).is_dir()
                and not Path(path).is_relative_to(REPO), 'cache must name an assigned canonical external directory')
    require(not Path(inputs['caches']['cargo_target']).is_relative_to(REPO), 'Cargo outputs must stay outside frozen source')
    require(Path(inputs['driver_config']).is_absolute(), 'future frozen driver config must be absolute')
    ignored_inventory(REPO)
    plan = dict(schema='gomad.driver-static-plan/v1', repo=str(REPO), driver_config=inputs['driver_config'],
                tools=tools, caches=inputs['caches'], lint_base=inputs['lint_base'], files=files,
                ignored_sha256=sha(HERE / 'static-ignored.json'))
    # The fixed recipe is data in the reviewed plan; run/read recompute it, never execute caller argv.
    plan['stages'] = {label: recipe(plan, label) for label in LABELS}
    return plan


def prepare_plan(inputs, plan_path):
    plan = static_plan(inputs)
    write(plan_path, plan)
    return plan


def assemble(inputs, base, plan_path, config_path):
    plan = static_plan(inputs)
    require(set(base['stages']) == set(mapping()), 'base must preserve all original 96 labels')
    write(plan_path, plan)
    files, tools = plan['files'], plan['tools']
    result = copy.deepcopy(base)
    result['stages'] = {label: dict(adapter='pending', reason='UNEXECUTED: runtime owner has not composed this endpoint')
                        for label in mapping()}
    for label in LABELS:
        common = ['--plan', str(plan_path.resolve()), '--stage', label]
        result['stages'][label] = dict(adapter='checked-command', timeout_s=3600,
            argv=[tools['python'], str(HERE / 'static_stages.py'), 'run', *common],
            checker=[tools['python'], str(HERE / 'static_stages.py'), 'read', *common])
    result['inputs'] = sorted(set(base['inputs']) | set(files) | {str(plan_path.resolve()),
        str(HERE / 'static_stages.py'), str(HERE / 'static-ignored.json')})
    write(config_path, result)
    return result


def recipe(plan, label):
    require(label in LABELS, 'only first 24 static stages are owned')
    module = str(Path(plan['repo']) / 'tools/gomad')
    workspace = module + '/rust/Cargo.toml'
    go, cargo = plan['tools']['go'], plan['tools']['cargo']
    common = [cargo, 'test', '--manifest-path', workspace, '--release', '--locked', '--offline', '--no-default-features']
    go_common = [go, 'test', '-tags', 'test_dep', '-count=1']
    def step(kind, argv, cwd=module, **extra):
        return dict(kind=kind, argv=argv, cwd=cwd, **extra)
    if label == 'validate':
        return [step('versions', [go, 'version']), step('rust-version', [plan['tools']['rustc'], '--version']),
                step('cargo-version', [cargo, '--version']),
                step('exit', [go, 'run', './internal/model/cmd/protocolgen', '-check']),
                step('go-tests', go_common + ['./internal/model/generation/protocol']),
                step('replay-regeneration', go_common + ['-run', '^TestReplayPlanConformance$', './internal/model/choice']),
                step('cargo-tests', common + ['-p', 'gomad-wasmhost', '--lib'], test_filter='replay_plan'),
                step('build-role', [cargo, 'build', '--manifest-path', workspace, '--release', '--locked', '--offline',
                    '--no-default-features', '-p', 'gomad-wasmhost', '--features', 'conformance'], role='conformance-reference', features=['conformance']),
                step('build-role', [cargo, 'build', '--manifest-path', workspace, '--release', '--locked', '--offline',
                    '--no-default-features', '-p', 'gomad-wasmhost', '--features', 'reference-helper'], role='ordinary-reference', features=['reference-helper']),
                step('go-tests', [go, 'test', '-tags', 'test_dep', '-count=2', *RETAINED], repeats=2, reference=True),
                step('cargo-tests', common + ['-p', 'gomad-wasmhost', '--lib'], repeats=2),
                step('cargo-tests', common + ['-p', 'gomad-wasmhost', '--test', 'inventory'])]
    if label == 'portable':
        return [step('go-tests', go_common + ['-timeout=20m', *RETAINED], reference=True)]
    if label == 'default-build':
        return [step('exit', [go, 'build', './...']),
                step('build-role', [cargo, 'build', '--manifest-path', workspace, '--release', '--locked', '--offline',
                    '--no-default-features', '-p', 'gomad'], role='release', features=[])]
    if label == 'default-vet':
        return [step('exit', [go, 'vet', '-tags', 'test_dep', './...'])]
    if label == 'compile-only':
        return [step('exit', go_common + ['-exec', plan['tools']['true'], '-run', '^$', './...']),
                step('exit', common + ['--workspace', '--no-run'])]
    if label == 'lint-router':
        return [step('go-tests', go_common + ['./cmd/tools/lintcode'], cwd=plan['repo'])]
    if label == 'lint-router-vet':
        return [step('exit', [go, 'vet', '-tags', 'test_dep', './cmd/tools/lintcode'], cwd=plan['repo'])]
    if label == 'actionlint-build':
        return [step('actionlint-build', [go, 'install', 'github.com/rhysd/actionlint/cmd/actionlint@v1.7.7'], cwd=plan['repo'])]
    if label == 'actionlint-version':
        return [step('actionlint-version', ['@actionlint', '-version'], cwd=plan['repo'])]
    if label == 'workflow-lint':
        return [step('workflow-lint', ['@actionlint', '@workflows'], cwd=plan['repo'])]
    if label in ('lint-module', 'lint-root'):
        target = 'lint-code-gomad' if label == 'lint-module' else 'lint-code-fast'
        return [step('exit', [plan['tools']['make'], target, 'GOLANGCI_LINT_BASE_REV=' + plan['lint_base'],
            'GOLANGCI_LINT_FIX=false', 'GOLANGCI_LINT=' + plan['tools']['golangci'],
            'ERRORTYPE=' + plan['tools']['errortype']], cwd=plan['repo'])]
    if label == 'architecture':
        return [step('go-tests', go_common + ['-timeout=15m', './internal/archcheck'], all_platforms=True)]
    if label == 'test-harness':
        return [step('go-tests', go_common + ['./internal/archcheck', './internal/support/...']),
                step('cargo-tests', common + ['-p', 'gomad'])]
    if label == 'world-race':
        return [step('go-tests', [go, 'test', '-race', '-tags=test_dep', '-p=1', '-parallel=1', '-count=1', './internal/model/world/...'])]
    if label == 'rust-lint':
        rows = [step('exit', [cargo, 'fmt', '--manifest-path', workspace, '--all', '--', '--check'])]
        for package, modes in [('gomad', HOST_FEATURES), ('gomad-wasmhost', HELPER_FEATURES)]:
            for features in modes:
                rows.append(step('exit', [plan['tools']['cargo_clippy'], 'clippy', '--manifest-path', workspace, '--release', '--locked', '--offline',
                    '--no-default-features', '-p', package, '--all-targets', '--no-deps', '--features', features, '--', '-D', 'warnings']))
        return rows
    if label in MODES:
        rows = [step('cargo-tests', common + ['-p', 'gomad-wasmhost', '--features', MODES[label]]
                     + (['--lib'] if label == 'rust-conformance-unit' else []))]
        if label == 'rust-default-test':
            rows.append(step('cargo-tests', common + ['-p', 'gomad']))
        return rows
    if label in ('helper-corpus-regen-1', 'helper-corpus-regen-2'):
        return [step('cargo-tests', common + ['-p', 'gomad-wasmhost', '--features', 'reference-helper', '--test', 'helper_corpus'], regenerate=True)]
    raise ValueError('unmapped static stage: ' + label)


def plan_config(path, label):
    plan = read(path)
    require(plan['schema'] == 'gomad.driver-static-plan/v1' and plan['repo'] == str(REPO), 'wrong static plan/source')
    require(label in LABELS and set(plan['stages']) == set(LABELS), 'incomplete static stage bindings')
    require(plan['stages'][label] == recipe(plan, label), 'static argv/reader recipe was changed')
    require(plan['ignored_sha256'] == sha(HERE / 'static-ignored.json'), 'ignored disposition changed')
    ignored_inventory(REPO)
    check_files(plan['files'])
    config = read(plan['driver_config'])
    require(str(path.resolve()) in config['files'] and sha(path) == config['files'][str(path.resolve())], 'unsealed static plan')
    require(str(HERE / 'static_stages.py') in config['files'], 'unsealed static module')
    check_files(config['files'])
    require(config['repo'] == str(REPO), 'wrong frozen candidate')
    return plan, config


def go_listing_argv(argv):
    selector = '.'
    for index, arg in enumerate(argv[2:], 2):
        if arg == '-run':
            require(index + 1 < len(argv), 'missing Go run selector')
            selector = argv[index + 1]
        elif arg.startswith('-run='):
            selector = arg[len('-run='):]
    require('/' not in selector, 'unsupported Go subtest selector in static inventory')
    return [*argv[:2], '-json', '-list', selector or '.', *argv[2:]]


def go_empty_output(package):
    return '?   \t' + package + '\t[no test files]\n'


def parse_go_packages(raw):
    names, packages, terminals, output = {}, set(), {}, {}
    for line in raw.decode().splitlines():
        row = json.loads(line)
        package, action = row['Package'], row.get('Action')
        packages.add(package)
        require(not row.get('Test') and action != 'fail', 'failed or test-bearing Go listing')
        if action in ('pass', 'skip'):
            require(package not in terminals, 'duplicate listed Go package terminal')
            terminals[package] = action
        if action == 'output':
            output[package] = output.get(package, '') + row.get('Output', '')
            name = row.get('Output', '').strip()
            if re.fullmatch(r'(?:Test\w+|Fuzz\w+|Example\w*)', name):
                key = (package, name)
                require(key not in names, 'duplicate listed Go root')
                names[key] = 0
    require(names, 'empty Go root inventory')
    require(set(terminals) == packages, 'missing listed Go package terminal')
    empty = {package for package, action in terminals.items() if action == 'skip'}
    require(all(output.get(package) == go_empty_output(package) for package in empty)
            and not ({package for package, _ in names} & empty), 'changed Go no-test package listing')
    return dict(roots=names, packages=packages, empty_packages=empty)


def parse_go_inventory(raw):
    return parse_go_packages(raw)['roots']


GO_SCOPE_SOURCES = {
    'tools/gomad/internal/archcheck/defaultset_test.go': '245b52ae2b3d33d8f734c7a6c83a3ceda053abeee66ad23876584ee2fb3d9425',
    'tools/gomad/internal/archcheck/architecture_test.go': 'b0e2aeb1a56835776148f94b112fc523a1e6dabdeef0e88d911f5f33d25c5e19',
    'tools/gomad/go.mod': '7951085bb6063ae8925c395345ee8997644cb83c9c9c55963e62cbad6bbf4511',
}
GO_SCOPE_ROOT = ('go.temporal.io/server/tools/gomad/internal/archcheck', 'TestNoDefaultSourceNamesNestedTree')
GO_SCOPE_REASON = 'no nested in-tree module is checked out; nothing to keep out of the default set'


def go_scope_exclusions(inventory):
    if GO_SCOPE_ROOT not in inventory:
        return []
    require(all(sha(REPO / path) == digest for path, digest in GO_SCOPE_SOURCES.items()),
            'original Go scope exclusion source changed')
    module = REPO / 'tools/gomad'
    require(not list((module / 'internal/backends').glob('*/go.mod')), 'nested backend module changes original exclusion scope')
    return [dict(package=GO_SCOPE_ROOT[0], name=GO_SCOPE_ROOT[1], reason=GO_SCOPE_REASON,
                 source_files=GO_SCOPE_SOURCES, working_directory=str(module),
                 replacement='go.temporal.io/server => ../..', nested_backend_modules=[], status='UNEXECUTED')]


def verify_go(raw, expected, repeats, exclusions=(), package_inventory=None):
    packages = package_inventory['packages'] if package_inventory is not None else {pkg for pkg, _ in expected}
    empty = package_inventory['empty_packages'] if package_inventory is not None else set()
    passes, runs, skips, terminals, output = dict(expected), dict(expected), dict(expected), {}, {}
    allowed = {(row['package'], row['name']): row for row in exclusions}
    require(set(allowed) <= set(expected), 'unlisted Go scope exclusion')
    for line in raw.decode().splitlines():
        row = json.loads(line)
        action, test = row.get('Action'), row.get('Test')
        package = row.get('Package')
        key = (package, test)
        require(package in packages, 'unlisted Go package ran')
        require(not test or package not in empty, 'test ran in Go no-test package')
        require(action != 'fail', 'failed Go package or subtest')
        if action == 'output' and test:
            output[key] = output.get(key, '') + row.get('Output', '')
        if action == 'output' and not test and package in empty:
            output[key] = output.get(key, '') + row.get('Output', '')
        if action == 'skip' and test:
            require(key in allowed and allowed[key]['reason'] in output.get(key, ''),
                    'unknown skipped Go package/subtest or changed exclusion reason')
            skips[key] += 1
        if not test:
            if action in ('pass', 'skip'):
                require(package not in terminals, 'duplicate Go package terminal')
                require(action == ('skip' if package in empty else 'pass'), 'unexpected Go package terminal')
                if package in empty:
                    require(output.get(key) == go_empty_output(package), 'changed Go no-test package output')
                terminals[package] = action
            continue
        if '/' not in test and action in ('run', 'pass'):
            require(key in expected, 'unlisted Go root ran')
            target = passes if action == 'pass' else runs
            target[key] += 1
    require(all(runs[key] == repeats and passes[key] == (0 if key in allowed else repeats)
                and skips[key] == (repeats if key in allowed else 0) for key in expected),
            'missing/duplicate Go root execution or scope exclusion')
    require(set(terminals) == packages, 'Go package did not terminate successfully')
    require(all(output.get((package, None)) == go_empty_output(package) for package in empty),
            'changed Go no-test package output')
    return [dict(package=pkg, name=name, repetitions=repeats) for pkg, name in sorted(expected) if (pkg, name) not in allowed]


def test_names(raw):
    names = [line[:-6] for line in raw.decode().splitlines() if line.endswith(': test')]
    require(len(names) == len(set(names)), 'duplicate listed Rust test')
    return set(names)


def verify_rust(raw, expected, ignored):
    observed = re.findall(r'^test (\S+) \.\.\. (ok|ignored(?:[^\n]*)?)$', raw.decode(), re.M)
    names = [name for name, _ in observed]
    require(len(names) == len(set(names)) and set(names) == expected, 'missing/duplicate/unlisted Rust test execution')
    passed = {name for name, status in observed if status == 'ok'}
    skipped = set(names) - passed
    require(skipped == ignored, 'Rust ignored disposition differs from list')
    summaries = re.findall(r'test result: ok\. (\d+) passed; (\d+) failed; (\d+) ignored;', raw.decode())
    require(len(summaries) == 1 and tuple(map(int, summaries[0])) == (len(passed), 0, len(ignored)), 'Rust summary/root inventory mismatch')
    return sorted(passed)


def tree_hashes(directory):
    return {str(path.relative_to(directory)): sha(path) for path in sorted(directory.rglob('*')) if path.is_file()}


def command_environment(plan, config, extra=None):
    environment = {key: value for key, value in config.get('environment', {}).items()
                   if not key.startswith(('GOMAD', 'QUALDRV'))}
    environment.update(PATH=os.pathsep.join([str(Path(plan['tools'][key]).parent) for key in ('go', 'cargo', 'make')])
                       + os.pathsep + os.environ.get('PATH', ''),
        GOCACHE=plan['caches']['go_cache'], GOMODCACHE=plan['caches']['go_mod_cache'],
        CARGO_TARGET_DIR=plan['caches']['cargo_target'], CARGO_HOME=plan['caches']['cargo_home'],
        RUSTC=plan['tools']['rustc'], RUSTDOC=plan['tools']['rustdoc'],
        RUSTFMT=plan['tools']['rustfmt'], CARGO=plan['tools']['cargo'],
        CARGO_BUILD_JOBS='1', RUST_TEST_THREADS='1', GOFLAGS='-p=1')
    environment.update(extra or {})
    return environment


def reference_environment(plan, config):
    result = {}
    for role, variable in [('ordinary-reference', 'GOMAD_WASM_TEST_HELPER'), ('conformance-reference', 'GOMAD_WASM_CONFORMANCE_HELPER')]:
        receipt = read(Path(config['root']) / 'static-roles' / role / 'build.json')
        require(sha(receipt['binary']) == receipt['binary_sha256'] and receipt['source_sha256'] == config['source_sha256'], 'reference role changed')
        result[variable] = receipt['binary']
    result.update(GOMAD_WASM_CONFORMANCE_REQUIRED='1', GOMAD_STOCK_GO=plan['tools']['go'],
                  GOMAD_WASM_STOCK_GO=plan['tools']['go'])
    return result


def action_environment(plan, config, action):
    extra = reference_environment(plan, config) if action.get('reference') else {}
    if action.get('all_platforms'):
        extra['GOMAD_ARCH_ALL_PLATFORMS'] = '1'
    return extra


def rust_test_owner(artifact):
    manifest = Path(artifact['manifest_path'])
    root = REPO / 'tools/gomad/rust'
    require(manifest in (root / 'gomad/Cargo.toml', root / 'wasmhost/Cargo.toml'), 'Rust test Cargo artifact has a foreign crate')
    source = Path(artifact['target']['src_path'])
    require(source.is_absolute() and source.resolve() == source and source.is_relative_to(manifest.parent) and source.suffix == '.rs',
            'Rust test Cargo artifact has a foreign source target')
    return dict(crate=manifest.parent.name, source=str(source.relative_to(root)))


def ignored_test_owner(known, name, owner):
    matches = [row for row in known if row['crate'] == owner['crate']
               and (row['source'].startswith(owner['crate'] + '/src/') or row['source'] == owner['source'])
               and (name.endswith('::' + row['name']) or name == row['name'])]
    require(len(matches) == 1, 'unknown/ambiguous ignored Rust test within Cargo owner: ' + name)
    return matches[0]


class Execution:
    def __init__(self, plan, config, label):
        self.plan, self.config, self.label = plan, config, label
        self.root = Path(config['root']) / 'stages' / label / 'static'
        self.root.mkdir(parents=True, exist_ok=False)
        self.rows, self.deferred, self.roles, self.ordinal = [], [], [], 0

    def call(self, argv, cwd, env=None):
        name = f'command-{self.ordinal:04d}'
        self.ordinal += 1
        environment = command_environment(self.plan, self.config, env)
        configuration = dict(self.config, repo=str(cwd), environment=environment)
        raw = command(configuration, self.root, name, list(map(str, argv)), timeout=3600)
        self.rows.append(dict(name=name, argv=list(map(str, argv)), cwd=str(cwd), environment=environment,
                             stdout_sha256=sha(self.root / (name + '.stdout')), command_sha256=sha(self.root / (name + '.command.json')),
                             process_sha256=sha(self.root / (name + '.process.json'))))
        return raw

    def go(self, action, env=None):
        argv = action['argv']
        listing = self.call(go_listing_argv(argv), action['cwd'], env)
        listing_name = self.rows[-1]['name']
        inventory = parse_go_packages(listing)
        raw = self.call([*argv[:2], '-json', *argv[2:]], action['cwd'], env)
        exclusions = go_scope_exclusions(inventory['roots'])
        roots = verify_go(raw, inventory['roots'], action.get('repeats', 1), exclusions, inventory)
        return dict(kind='go-tests', roots=roots, listing=listing_name, run=self.rows[-1]['name'],
                    repetitions=action.get('repeats', 1), scope_exclusions=exclusions, inventory_sha256=hashlib.sha256(listing).hexdigest())

    def rust(self, action):
        argv, cwd = action['argv'], action['cwd']
        raw = self.call([*argv, '--no-run', '--message-format=json'], cwd)
        binaries = {}
        for line in raw.decode().splitlines():
            row = json.loads(line)
            if row.get('reason') == 'compiler-artifact' and row.get('executable') and row['profile']['test']:
                binary = regular(row['executable'])
                require(binary.is_relative_to(Path(self.plan['caches']['cargo_target'])), 'test executable outside assigned target')
                binaries[str(binary)] = dict(binary_sha256=sha(binary), target=row['target']['name'], cwd=str(Path(row['manifest_path']).parent), **rust_test_owner(row))
        require(binaries, 'no actual Rust test executable')
        total, reports = 0, []
        known = ignored_inventory(REPO)
        for binary, binding in sorted(binaries.items()):
            suffix = [action['test_filter']] if action.get('test_filter') else []
            listed = self.call([binary, '--list', *suffix], binding['cwd'])
            listing_name = self.rows[-1]['name']
            ignored_raw = self.call([binary, '--ignored', '--list', *suffix], binding['cwd'])
            ignored_name = self.rows[-1]['name']
            expected, ignored = test_names(listed), test_names(ignored_raw)
            require(ignored <= expected, 'ignored names absent from test inventory')
            for name in sorted(ignored):
                row = ignored_test_owner(known, name, binding)
                self.deferred.append(dict(row, id=name, obligation=row['final_id'], test=name, status='UNEXECUTED', binary_sha256=binding['binary_sha256'],
                    source_sha256=sha(REPO / 'tools/gomad/rust' / row['source'])))
            for repeat in range(action.get('repeats', 1)):
                env = {'GOMAD_HELPER_CORPUS_REGENERATE': '1'} if action.get('regenerate') else {}
                output = self.call([binary, '--test-threads=1', *suffix], binding['cwd'], env)
                passed = verify_rust(output, expected, ignored)
                total += len(passed)
                reports.append(dict(binary=binary, **binding, tests=passed, ignored=sorted(ignored), repeat=repeat,
                                    listing=listing_name, ignored_listing=ignored_name, run=self.rows[-1]['name']))
                require(sha(binary) == binding['binary_sha256'], 'Rust executable changed during test')
        doctest = None
        # Cargo --no-run omits doctests; preserve their original execution separately.
        if '--test' not in argv and '--bin' not in argv and argv[argv.index('-p') + 1] == 'gomad-wasmhost':
            doc_argv = [arg for arg in argv if arg not in ('--lib',)]
            doc_raw = self.call([*doc_argv, '--doc'], cwd)
            require('test result: ok.' in doc_raw.decode(), 'missing actual doctest summary')
            require(re.search(r'\d+ failed; (?:[1-9]\d*) ignored;', doc_raw.decode()) is None, 'ignored doctest lacks explicit receiving obligation')
            doctest = self.rows[-1]['name']
        require(total > 0, 'empty Rust test suite')
        return dict(kind='cargo-tests', executables=reports, passed=total, doctest=doctest)

    def build(self, action):
        raw = self.call([*action['argv'], '--message-format=json'], action['cwd'])
        name = 'gomad' if action['role'] == 'release' else 'gomad-wasmhost'
        artifacts = [json.loads(line) for line in raw.decode().splitlines()]
        artifacts = [row for row in artifacts if row.get('reason') == 'compiler-artifact' and row.get('executable')
                     and row['target']['name'] == name and not row['profile']['test']]
        require(len(artifacts) == 1, 'missing/duplicate actual build role')
        artifact = artifacts[0]
        source = regular(artifact['executable'])
        require(source.is_relative_to(Path(self.plan['caches']['cargo_target'])), 'role output outside assigned target')
        features = sorted(artifact['features'])
        wanted = sorted(set(action['features']) | ({'reference-helper'} if 'conformance' in action['features'] else set()))
        require(features == wanted, 'compiled build feature set differs')
        root = Path(self.config['root']) / 'static-roles' / action['role']
        root.mkdir(parents=True, exist_ok=False)
        destination = root / name
        shutil.copyfile(source, destination)
        destination.chmod(0o500)
        digest = sha(destination)
        require(digest == sha(source), 'role copy changed')
        if action['role'] == 'release':
            require(digest == self.config['binary_sha256'], 'rebuilt release differs from frozen candidate')
        receipt = dict(schema='gomad.driver-static-build/v1', binary=str(destination), binary_sha256=digest,
            role=action['role'], features=features, requested_features=action['features'], profile='release', exit=0,
            source_sha256=self.config['source_sha256'], rustc_sha256=sha(self.plan['tools']['rustc']),
            cargo_sha256=sha(self.plan['tools']['cargo']), raw_build_sha256=hashlib.sha256(raw).hexdigest())
        write(root / 'build.json', receipt)
        self.roles.append(receipt)
        return receipt

    def execute(self):
        outcomes = []
        for action in self.plan['stages'][self.label]:
            results = []
            first_command = len(self.rows)
            kind, env = action['kind'], action_environment(self.plan, self.config, action)
            if kind == 'go-tests':
                results.append(self.go(action, env))
            elif kind == 'cargo-tests':
                before = tree_hashes(REPO / 'tools/gomad/rust/wasmhost/tests/testdata/helper_transcript') if action.get('regenerate') else None
                results.append(self.rust(action))
                if before is not None:
                    after = tree_hashes(REPO / 'tools/gomad/rust/wasmhost/tests/testdata/helper_transcript')
                    require(before == after and bool(before), 'helper corpus regeneration changed source bytes')
                    if self.label.endswith('-2'):
                        first = read(Path(self.config['root']) / 'stages/helper-corpus-regen-1/static/result.json')
                        require(first['corpus'] == after, 'second helper corpus differs from independent first generation')
                    self.corpus = after
            elif kind == 'build-role':
                results.append(self.build(action))
            elif kind == 'replay-regeneration':
                expected = tree_hashes(REPO / 'tools/gomad/rust/wasmhost/tests/testdata/replay_plan')
                for repeat in range(2):
                    destination = self.root / ('replay-generation-' + str(repeat))
                    results.append(self.go(action, dict(GOMAD_REPLAY_PLAN_CORPUS_REGENERATE='1', GOMAD_REPLAY_PLAN_CORPUS_DIR=str(destination))))
                    require(tree_hashes(destination) == expected, 'replay plan generation differs from source/other pass')
            else:
                argv = action['argv']
                if kind == 'actionlint-build':
                    require('ACTIONLINT_VER := v1.7.7' in (REPO / 'Makefile').read_text(), 'changed original actionlint version')
                    binary = Path(self.config['root']) / 'static-tools/actionlint'
                    binary.parent.mkdir(parents=True, exist_ok=False)
                    env['GOBIN'] = str(binary.parent)
                if kind in ('actionlint-version', 'workflow-lint'):
                    receipt = read(Path(self.config['root']) / 'static-tools/actionlint.json')
                    require(sha(receipt['binary']) == receipt['sha256'], 'built actionlint changed')
                    argv = [receipt['binary'], *argv[1:]]
                    if kind == 'workflow-lint':
                        pairs = [[REPO / '.github/workflows' / (prefix + suffix) for suffix in ('.yml', '-smoke.yml')]
                                 for prefix in ('gomad3', 'gomad')]
                        pairs = [pair for pair in pairs if all(path.is_file() and 'tools/gomad/' in path.read_text() for path in pair)]
                        require(len(pairs) == 1, 'ambiguous/missing original workflow pair')
                        argv = [receipt['binary'], *map(str, pairs[0])]
                raw = self.call(argv, action['cwd'], env)
                if kind == 'versions':require(raw.decode().startswith('go version go1.27.1 '), 'wrong pinned Go version')
                if kind == 'rust-version':require(raw.decode().startswith('rustc 1.99.1 '), 'wrong pinned rustc version')
                if kind == 'cargo-version':require(raw.decode().startswith('cargo 1.99.1 '), 'wrong pinned Cargo version')
                if kind == 'actionlint-build':
                    regular(binary)
                    write(binary.with_suffix('.json'), dict(binary=str(binary), sha256=sha(binary), version='v1.7.7'))
                if kind == 'actionlint-version':require('v1.7.7' in raw.decode().splitlines(), 'changed actionlint version')
                results.append(dict(kind=kind, stdout_sha256=hashlib.sha256(raw).hexdigest()))
            outcomes.append(dict(recipe=action, results=results, commands=[row['name'] for row in self.rows[first_command:]]))
        report = dict(schema='gomad.driver-static-execution/v1', stage=self.label,
            source_sha256=self.config['source_sha256'], identity=self.config['identity'],
            config_sha256=sha(self.plan['driver_config']), plan_sha256=sha(self.plan_path), actions=outcomes, commands=self.rows, deferred_ignored=self.deferred,
            roles=self.roles, corpus=getattr(self, 'corpus', None))
        write(self.root / 'result.json', report)
        return report


def authenticate_result(plan_path, label):
    plan, config = plan_config(plan_path, label)
    root = Path(config['root']) / 'stages' / label / 'static'
    report = read(root / 'result.json')
    require(report['schema'] == 'gomad.driver-static-execution/v1' and report['stage'] == label, 'wrong static result')
    require(report['source_sha256'] == config['source_sha256'] and report['identity'] == config['identity']
            and report['plan_sha256'] == sha(plan_path) and report['config_sha256'] == sha(plan['driver_config']),
            'static result candidate/config/plan changed')
    require([row['recipe'] for row in report['actions']] == plan['stages'][label] and report['commands'],
            'incomplete/reordered static recipes')
    commands = {}
    raw_reports = []
    for index, row in enumerate(report['commands']):
        require(row['name'] == f'command-{index:04d}', 'duplicate/reordered command inventory')
        paths = {suffix: root / (row['name'] + suffix) for suffix in ('.command.json', '.process.json', '.stdout')}
        for suffix, field in [('.command.json', 'command_sha256'), ('.process.json', 'process_sha256'), ('.stdout', 'stdout_sha256')]:
            require(sha(paths[suffix]) == row[field], 'static command bytes changed')
            raw_reports.append(dict(path=str(paths[suffix]), sha256=row[field]))
        recorded, process = read(paths['.command.json']), read(paths['.process.json'])
        require(recorded['argv'] == row['argv'] and recorded['cwd'] == row['cwd'] and recorded['exit'] == 0
                and recorded['observation'] == 'completed' and recorded['pid'] > 0
                and recorded['environment'] == row['environment']
                and recorded['temporary_paths'] == config.get('temporary_paths', {})
                and process['pid'] == recorded['pid'] and process.get('active') is False
                and recorded['end_ns'] >= recorded['start_ns'], 'static command did not complete')
        commands[row['name']] = row
    require([name for action in report['actions'] for name in action['commands']] == list(commands),
            'omitted/reused static commands')
    testcases, deferred, scope_exclusions = [], [], []
    known = ignored_inventory(REPO)
    consumed = set()
    def consume(name, owned, argv, cwd, extra):
        require(name in owned and name not in consumed, 'reused execution or command outside owned recipe')
        row = commands[name]
        require(row['argv'] == argv and row['cwd'] == cwd
                and row['environment'] == command_environment(plan, config, extra), 'changed command argv/cwd/recipe environment')
        consumed.add(name)
    def raw(name, owned):
        require(name in owned, 'raw command belongs to another recipe')
        return (root / (name + '.stdout')).read_bytes()
    for action in report['actions']:
        recipe_row, results, owned = action['recipe'], action['results'], action['commands']
        require(owned and len(results) == (2 if recipe_row['kind'] == 'replay-regeneration' else 1),
                'missing/duplicate recipe result')
        first = commands[owned[0]]
        kind = recipe_row['kind']
        extra = action_environment(plan, config, recipe_row)
        if kind in ('cargo-tests', 'build-role'):
            expected = recipe_row['argv'] + (['--no-run', '--message-format=json'] if kind == 'cargo-tests' else ['--message-format=json'])
            consume(owned[0], owned, expected, recipe_row['cwd'], extra)
        elif kind not in ('go-tests', 'replay-regeneration', 'workflow-lint', 'actionlint-version'):
            if kind == 'actionlint-build':
                extra['GOBIN'] = str(Path(config['root']) / 'static-tools')
            consume(owned[0], owned, recipe_row['argv'], recipe_row['cwd'], extra)
        if kind in ('workflow-lint', 'actionlint-version'):
            receipt = read(Path(config['root']) / 'static-tools/actionlint.json')
            require(sha(receipt['binary']) == receipt['sha256'], 'built actionlint changed before reader')
            expected = [receipt['binary'], '-version']
            if kind == 'workflow-lint':
                pairs = [[REPO / '.github/workflows' / (prefix + suffix) for suffix in ('.yml', '-smoke.yml')]
                         for prefix in ('gomad3', 'gomad')]
                pairs = [pair for pair in pairs if all(path.is_file() and 'tools/gomad/' in path.read_text() for path in pair)]
                require(len(pairs) == 1, 'ambiguous/missing original workflow pair')
                expected = [receipt['binary'], *map(str, pairs[0])]
            consume(owned[0], owned, expected, recipe_row['cwd'], extra)
        for result_index, row in enumerate(results):
            if kind in ('go-tests', 'replay-regeneration'):
                argv = recipe_row['argv']
                require(row['kind'] == 'go-tests' and row['repetitions'] == recipe_row.get('repeats', 1), 'changed Go repetition contract')
                go_extra = dict(extra)
                if kind == 'replay-regeneration':
                    go_extra.update(GOMAD_REPLAY_PLAN_CORPUS_REGENERATE='1',
                                    GOMAD_REPLAY_PLAN_CORPUS_DIR=str(root / ('replay-generation-' + str(result_index))))
                consume(row['listing'], owned, go_listing_argv(argv), recipe_row['cwd'], go_extra)
                consume(row['run'], owned, [*argv[:2], '-json', *argv[2:]], recipe_row['cwd'], go_extra)
                inventory = parse_go_packages(raw(row['listing'], owned))
                exclusions = go_scope_exclusions(inventory['roots'])
                require(row['scope_exclusions'] == exclusions, 'changed Go exclusion witness')
                observed = verify_go(raw(row['run'], owned), inventory['roots'], row['repetitions'], exclusions, inventory)
                scope_exclusions.extend(exclusions)
                require(observed == row['roots'], 'Go roots differ from raw execution')
                for line in raw(row['run'], owned).decode().splitlines():
                    event = json.loads(line)
                    if event.get('Action') == 'pass' and event.get('Test'):
                        testcases.append(dict(id='go/' + event['Package'] + '/' + event['Test'], status='PASS', executed=True, exit=0,
                                              raw=row['run']))
            elif kind == 'cargo-tests':
                require(row['kind'] == kind and row['executables'], 'empty Rust proof')
                artifacts = [json.loads(line) for line in raw(owned[0], owned).decode().splitlines()]
                actual_artifacts = {artifact['executable']: artifact for artifact in artifacts if artifact.get('reason') == 'compiler-artifact'
                                    and artifact.get('executable') and artifact['profile']['test']}
                expected_binaries = set(actual_artifacts)
                require({entry['binary'] for entry in row['executables']} == expected_binaries, 'missing Rust executable')
                require([(entry['binary'], entry['repeat']) for entry in row['executables']]
                        == [(binary, repeat) for binary in sorted(expected_binaries) for repeat in range(recipe_row.get('repeats', 1))],
                        'missing/reordered/duplicate Rust repetitions')
                inventory_refs = {}
                total = 0
                for executable in row['executables']:
                    require(sha(executable['binary']) == executable['binary_sha256'], 'test binary changed before reader')
                    binary = executable['binary']
                    artifact = actual_artifacts[binary]
                    owner = rust_test_owner(artifact)
                    require(all(executable.get(key) == value for key, value in owner.items())
                            and executable['target'] == artifact['target']['name']
                            and executable['cwd'] == str(Path(artifact['manifest_path']).parent),
                            'Rust test executable changed its actual Cargo crate/source target')
                    suffix = [recipe_row['test_filter']] if recipe_row.get('test_filter') else []
                    references = (executable['listing'], executable['ignored_listing'])
                    if executable['repeat'] == 0:
                        inventory_refs[binary] = references
                        consume(executable['listing'], owned, [binary, '--list', *suffix], executable['cwd'], extra)
                        consume(executable['ignored_listing'], owned, [binary, '--ignored', '--list', *suffix], executable['cwd'], extra)
                    else:
                        require(inventory_refs[binary] == references, 'changed shared Rust listing references')
                    run_extra = dict(extra)
                    if recipe_row.get('regenerate'):
                        run_extra['GOMAD_HELPER_CORPUS_REGENERATE'] = '1'
                    consume(executable['run'], owned, [binary, '--test-threads=1', *suffix], executable['cwd'], run_extra)
                    expected = test_names(raw(executable['listing'], owned))
                    ignored = test_names(raw(executable['ignored_listing'], owned))
                    passed = verify_rust(raw(executable['run'], owned), expected, ignored)
                    require(passed == executable['tests'] and sorted(ignored) == executable['ignored'], 'Rust testcase inventory differs')
                    total += len(passed)
                    for name in passed:
                        testcases.append(dict(id='rust/' + executable['target'] + '/' + name, status='PASS', executed=True, exit=0,
                                              raw=executable['run']))
                    if executable['repeat'] == 0:
                        for name in sorted(ignored):
                            entry = ignored_test_owner(known, name, owner)
                            deferred.append(dict(entry, id=name, obligation=entry['final_id'], test=name, status='UNEXECUTED',
                                binary_sha256=executable['binary_sha256'], source_sha256=sha(REPO / 'tools/gomad/rust' / entry['source'])))
                require(total == row['passed'] and total > 0, 'empty/changed Rust passed count')
                for binary in expected_binaries:
                    require(sorted(entry['repeat'] for entry in row['executables'] if entry['binary'] == binary)
                            == list(range(recipe_row.get('repeats', 1))), 'missing/duplicate Rust repetition')
                expects_doc = '--test' not in recipe_row['argv'] and '--bin' not in recipe_row['argv'] and recipe_row['argv'][recipe_row['argv'].index('-p') + 1] == 'gomad-wasmhost'
                require((row['doctest'] is not None) == expects_doc, 'missing/unexpected doctest execution')
                if row['doctest'] is not None:
                    doc_argv = [arg for arg in recipe_row['argv'] if arg != '--lib'] + ['--doc']
                    consume(row['doctest'], owned, doc_argv, recipe_row['cwd'], extra)
                    summaries = re.findall(r'test result: ok\. (\d+) passed; (\d+) failed; (\d+) ignored;', raw(row['doctest'], owned).decode())
                    require(summaries and all(int(failed) == int(ignored) == 0 for _, failed, ignored in summaries), 'missing/ignored/failed doctest')
            elif kind == 'build-role':
                artifacts = [json.loads(line) for line in raw(owned[0], owned).decode().splitlines()]
                target = 'gomad' if recipe_row['role'] == 'release' else 'gomad-wasmhost'
                artifacts = [entry for entry in artifacts if entry.get('reason') == 'compiler-artifact'
                             and entry.get('executable') and entry['target']['name'] == target and not entry['profile']['test']]
                wanted = sorted(set(recipe_row['features']) | ({'reference-helper'} if 'conformance' in recipe_row['features'] else set()))
                require(len(artifacts) == 1 and sorted(artifacts[0]['features']) == wanted and row['features'] == wanted, 'actual compiled role feature mismatch')
                require(row['role'] == recipe_row['role'] and row['requested_features'] == recipe_row['features']
                        and sha(row['binary']) == row['binary_sha256'] and row['source_sha256'] == config['source_sha256'], 'changed build role')
            else:
                require(row['kind'] == kind, 'wrong recipe result kind')
                output = raw(owned[0], owned).decode()
                prefixes = {'versions': 'go version go1.27.1 ', 'rust-version': 'rustc 1.99.1 ', 'cargo-version': 'cargo 1.99.1 '}
                if kind in prefixes:
                    require(output.startswith(prefixes[kind]), 'changed pinned tool version')
                if kind == 'actionlint-version':
                    require('v1.7.7' in output.splitlines(), 'wrong actionlint version')
        if kind == 'replay-regeneration':
            expected = tree_hashes(REPO / 'tools/gomad/rust/wasmhost/tests/testdata/replay_plan')
            require(expected and all(tree_hashes(root / ('replay-generation-' + str(repeat))) == expected for repeat in range(2)),
                    'regenerated replay corpus differs')
        if recipe_row.get('regenerate'):
            expected = tree_hashes(REPO / 'tools/gomad/rust/wasmhost/tests/testdata/helper_transcript')
            require(expected and report['corpus'] == expected, 'regenerated helper corpus differs')
            if label.endswith('-2'):
                first = read(Path(config['root']) / 'stages/helper-corpus-regen-1/static/result.json')
                require(first['corpus'] == expected, 'second helper corpus differs from first')
    require(consumed == set(commands), 'unconsumed raw command cannot count as proof')
    require(deferred == report['deferred_ignored'], 'ignored obligations differ from actual ignored test reports')
    return dict(schema='gomad.rust-driver-check/v1', stage=label, source_sha256=config['source_sha256'],
        config_sha256=sha(plan['driver_config']), identity=config['identity'],
        subgates=[dict(id=name, status='PASS', executed=True, exit=0) for name in subgates(label)],
        testcases=testcases, raw_reports=raw_reports,
        static_execution=dict(path=str(root / 'result.json'), sha256=sha(root / 'result.json')),
        deferred_ignored=deferred, unexecuted_scope_exclusions=scope_exclusions)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='mode', required=True)
    prepare_parser = sub.add_parser('prepare-plan')
    for key in ('inputs', 'plan'):
        prepare_parser.add_argument('--' + key, type=Path, required=True)
    assemble_parser = sub.add_parser('assemble')
    for key in ('inputs', 'base', 'plan', 'output'):
        assemble_parser.add_argument('--' + key, type=Path, required=True)
    for name in ('run', 'read'):
        child = sub.add_parser(name)
        child.add_argument('--plan', type=Path, required=True)
        child.add_argument('--stage', choices=LABELS, required=True)
    args = parser.parse_args()
    if args.mode == 'prepare-plan':
        prepare_plan(read(args.inputs), args.plan.resolve())
    elif args.mode == 'assemble':
        assemble(read(args.inputs), read(args.base), args.plan.resolve(), args.output)
    elif args.mode == 'read':
        print(json.dumps(authenticate_result(args.plan, args.stage), sort_keys=True))
    else:
        plan, config = plan_config(args.plan, args.stage)
        execution = Execution(plan, config, args.stage)
        execution.plan_path = args.plan
        execution.execute()


if __name__ == '__main__':
    main()
