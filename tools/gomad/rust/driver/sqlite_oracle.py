#!/usr/bin/env python3
"""Build and reopen the fixed retained-Go SQLite catalogue oracle."""
import argparse
import json
from pathlib import Path
import sys

import live
import preparation
import terminal_controls
import native_roles
from driver import Refusal, command, require, sha, write

PACKAGE = './internal/model/cmd/sqlite-native'
IMPORT = 'go.temporal.io/server/tools/gomad/internal/model/cmd/sqlite-native'
MODULE = 'go.temporal.io/server/tools/gomad'
DEPENDENCY = 'github.com/go-json-experiment/json'
VERSION = 'v0.0.0-20260820222146-c27c302e5fc3'
SUM = 'h1:UADEEmDKgfXbtnGJZ97beY5XLo9ZechG1nlU4KnRrkE='
DEPENDENCIES = {DEPENDENCY: (VERSION, SUM), 'golang.org/x/sys': ('v0.48.0', 'h1:bbX/i/6MgT9BVLM9RT1thmxL04yeTAhbEz4SyadbXoo=')}
PLAN = {'schema', 'repo', 'stock_go', 'go_cache', 'module_cache'}
FIELDS = {'schema', 'plan', 'plan_sha256', 'repo', 'stock_go', 'binary', 'binary_sha256', 'environment', 'sources', 'sdk', 'dependency', 'commands'}
COMMANDS = ('version', 'environment', 'packages', 'verify-modules', 'build', 'build-info')
SUFFIXES = ('command.json', 'process.json', 'stdout', 'stderr')


def read(path):
    path = regular(path)
    require(path.stat().st_size <= 64 << 20, 'SQLite oracle metadata exceeds bound')
    return native_roles.decode(path.read_bytes())


def regular(path):
    path = Path(path)
    require(path.is_absolute() and path.resolve(strict=True) == path and path.is_file()
            and not any(p.is_symlink() for p in (path, *path.parents)) and path.stat().st_size <= 512 << 20,
            'SQLite oracle requires bounded canonical regular files: ' + str(path))
    return path


def tree(root):
    root = Path(root)
    require(root.is_absolute() and root.resolve(strict=True) == root, 'noncanonical SQLite oracle tree')
    return {str(root / name): value['sha256'] for name, value in live.inventory(root).items()}


def sources(repo):
    base = Path(repo) / 'tools/gomad'
    result = {str(regular(directory / name)): sha(directory / name)
              for directory in (Path(repo), base) for name in ('go.mod', 'go.sum')}
    for name in ('internal/model', 'internal/support/fs'):
        result.update(tree(base / name))
    require(str(base / 'internal/model/cmd/sqlite-native/main.go') in result
            and str(base / 'internal/model/wasi/storage_projection.go') in result, 'missing fixed Go SQLite source')
    return result


def plan_value(path):
    value = read(regular(path))
    require(set(value) == PLAN and value['schema'] == 'gomad.sqlite-go-oracle-plan/v1', 'SQLite oracle plan fields differ')
    for name in ('repo', 'go_cache', 'module_cache'):
        p = Path(value[name])
        require(p.is_absolute() and p.resolve(strict=True) == p and p.is_dir(), 'SQLite oracle assigned directory differs')
    go = regular(value['stock_go'])
    require(go.name == 'go' and go.parent.name == 'bin' and go.stat().st_mode & 0o111, 'SQLite oracle requires stock Go distribution')
    require(all(not Path(value[name]).is_relative_to(Path(value['repo'])) for name in ('go_cache', 'module_cache')),
            'SQLite oracle caches must be external')
    return value


def environment(plan, root):
    goroot = Path(plan['stock_go']).parent.parent
    return dict(PATH=str(goroot / 'bin') + ':/usr/bin:/bin', GOROOT=str(goroot),
                GOCACHE=plan['go_cache'], GOMODCACHE=plan['module_cache'], GOTMPDIR=str(root / 'tmp'),
                GOMAXPROCS='1', CGO_ENABLED='0', GOFLAGS='', GOENV='off', GOWORK='off',
                GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off', GOCACHEPROG='')


def verification_files(plan, root):
    # Verify only the two selected source modules, without loading unrelated server dependencies.
    lines = (Path(plan['repo']) / 'tools/gomad/go.sum').read_text().splitlines()
    selected = []
    requirements = []
    for module, (version, checksum) in sorted(DEPENDENCIES.items()):
        requirements.append(f'\t{module} {version}\n')
        for suffix in ('', '/go.mod'):
            wanted = [line for line in lines if line.startswith(module + ' ' + version + suffix + ' ')]
            require(len(wanted) == 1, 'SQLite source dependency sum missing/duplicated')
            if not suffix:
                require(wanted[0] == f'{module} {version} {checksum}', 'SQLite source dependency identity changed')
            selected.extend(wanted)
    return {root / 'verify.mod': ('module gomad.sqlite-oracle-verification\n\ngo 1.27.1\n\nrequire (\n' + ''.join(requirements) + ')\n').encode(),
            root / 'verify.sum': ('\n'.join(selected) + '\n').encode()}


def recipes(plan, root):
    go, binary = plan['stock_go'], str(root / 'sqlite-native')
    return {
        'version': [go, 'version'],
        'environment': [go, 'env', '-json', 'GOROOT', 'GOVERSION', 'GOTOOLDIR'],
        'packages': [go, 'list', '-deps', '-json', '-compiled', '-mod=readonly', '-tags=test_dep', PACKAGE],
        'verify-modules': [go, 'mod', 'verify', '-modfile=' + str(root / 'verify.mod')],
        'build': [go, 'build', '-p=1', '-mod=readonly', '-tags=test_dep', '-trimpath', '-buildvcs=false', '-o', binary, PACKAGE],
        'build-info': [go, 'version', '-m', binary],
    }


def journal(root, name, argv, cwd, env):
    receipt, process = (read(regular(root / (name + '.' + suffix))) for suffix in ('command.json', 'process.json'))
    live.completed_command(receipt, process, argv, str(cwd))
    require(receipt['environment_mode'] == 'exact' and receipt['effective_environment'] == env,
            'SQLite oracle command inherited an unreviewed environment')
    for row in (receipt, process):
        require(row['clock'] == 'CLOCK_MONOTONIC' and type(row['parent_pid']) is int and row['parent_pid'] > 0
                and type(row['started_monotonic_nanos']) is int and type(row['reaped_monotonic_nanos']) is int
                and 0 < row['started_monotonic_nanos'] < row['reaped_monotonic_nanos'], 'SQLite oracle child lacks actual kernel clock')
    require(all(receipt[key] == process[key] for key in ('parent_pid', 'started_monotonic_nanos', 'reaped_monotonic_nanos')),
            'SQLite oracle command/process lifecycle differs')
    for suffix in ('stdout', 'stderr'):
        regular(root / (name + '.' + suffix))
    return (root / (name + '.stdout')).read_bytes(), process


def dependency(plan):
    result = {}
    for module, (version, checksum) in DEPENDENCIES.items():
        root = Path(plan['module_cache']) / (module + '@' + version)
        result.update(tree(root))
        cache = Path(plan['module_cache']) / 'cache/download' / module / '@v'
        require(regular(cache / (version + '.ziphash')).read_text().strip() == checksum,
                'SQLite oracle dependency differs from pinned Go sum')
        for suffix in ('.ziphash', '.zip', '.mod', '.info'):
            path = regular(cache / (version + suffix))
            result[str(path)] = sha(path)
    return result


def package_graph(raw, plan, inputs):
    rows = preparation.documents(raw)
    graph = {row['ImportPath']: row for row in rows}
    require(len(graph) == len(rows) and IMPORT in graph and graph[IMPORT]['Name'] == 'main'
            and graph[IMPORT].get('GoFiles') == ['main.go'], 'foreign/duplicate SQLite Go main')
    reached, pending, modules = set(), [IMPORT], set()
    while pending:
        name = pending.pop()
        if name in reached:
            continue
        require(name in graph, 'SQLite Go dependency graph omits imported package')
        reached.add(name)
        row = graph[name]
        require(not row.get('Error') and not row.get('DepsErrors'), 'SQLite Go package collection failed')
        directory = Path(row['Dir'])
        require(directory.is_absolute() and directory.resolve(strict=True) == directory, 'SQLite Go package directory changed')
        if row.get('Standard'):
            require(directory.is_relative_to(Path(plan['stock_go']).parent.parent / 'src'), 'foreign stock Go source')
        elif name.startswith(MODULE + '/'):
            require(name == IMPORT or name.startswith(MODULE + '/internal/model/') or name == MODULE + '/internal/support/fs',
                    'SQLite oracle imports another host implementation')
            require(directory == Path(plan['repo']) / 'tools/gomad' / name.removeprefix(MODULE + '/'), 'cross-source SQLite Go package')
        else:
            module = row.get('Module', {})
            owner = module.get('Path')
            require(owner in DEPENDENCIES and (name == owner or name.startswith(owner + '/')),
                    'unreviewed SQLite Go dependency')
            version, _ = DEPENDENCIES[owner]
            require(module.get('Version') == version and not module.get('Replace')
                    and directory.is_relative_to(Path(plan['module_cache']) / (owner + '@' + version)),
                    'unreviewed SQLite Go dependency/replacement')
            modules.add(owner)
        require(not row.get('CgoFiles'), 'SQLite oracle requires native pure-Go model')
        for field in ('GoFiles', 'SFiles', 'HFiles', 'SysoFiles', 'EmbedFiles'):
            for relative in row.get(field, []):
                path = directory / relative
                require(path.resolve(strict=True) == path and str(path) in inputs, 'SQLite Go compiled file omitted from closure')
        pending.extend(row.get('ImportMap', {}).get(value, value) for value in row.get('Imports', []))
    require(reached == set(graph), 'SQLite oracle graph contains unexecuted extra packages')
    return modules


def build_info(raw, binary, modules=None):
    lines = raw.decode().splitlines()
    require(lines and lines[0] == str(binary) + ': go1.27.1', 'SQLite oracle executable is not stock Go1.27.1')
    fields, builds = {}, {}
    for line in lines[1:]:
        parts = line.strip().split('\t')
        if parts[0] == 'build':
            key, value = parts[1].split('=', 1)
            require(key not in builds, 'duplicate SQLite Go build setting')
            builds[key] = value
        else:
            fields.setdefault(parts[0], []).append(parts[1:])
    require(fields.get('path') == [[IMPORT]] and fields.get('mod') == [[MODULE, '(devel)']]
            and fields.get('dep') and all(len(row) == 3 and row[0] in DEPENDENCIES
                and tuple(row[1:]) == DEPENDENCIES[row[0]] for row in fields['dep'])
            and len({row[0] for row in fields['dep']}) == len(fields['dep'])
            and DEPENDENCY in {row[0] for row in fields['dep']}, 'SQLite executable embeds another source/module')
    if modules is not None:
        require({row[0] for row in fields['dep']} == modules, 'SQLite executable dependencies differ from actual Go graph')
    required = {'-buildmode': 'exe', '-compiler': 'gc', '-tags': 'test_dep', '-trimpath': 'true', 'CGO_ENABLED': '0'}
    require(all(builds.get(key) == value for key, value in required.items())
            and set(builds) <= set(required) | {'GOOS', 'GOARCH', 'GOAMD64', 'GOARM64', 'GOARM', 'GO386', 'GOMIPS', 'GOMIPS64', 'GOPPC64', 'GORISCV64'},
            'SQLite executable changed fixed compiler flags or native target')
    return builds


def inspect_build(path, config=None):
    path = regular(path)
    receipt = read(path)
    require(set(receipt) == FIELDS and receipt['schema'] == 'gomad.sqlite-go-oracle-build/v1', 'SQLite Go build receipt fields differ')
    plan = plan_value(receipt['plan'])
    require(receipt['plan_sha256'] == sha(receipt['plan']), 'SQLite build input plan changed')
    root, cwd = path.parent, Path(plan['repo']) / 'tools/gomad'
    require(receipt['repo'] == plan['repo'] and receipt['stock_go'] == plan['stock_go']
            and receipt['binary'] == str(root / 'sqlite-native') and sha(regular(receipt['binary'])) == receipt['binary_sha256']
            and Path(receipt['binary']).stat().st_mode & 0o111 and not Path(receipt['binary']).stat().st_mode & 0o022
            and receipt['environment'] == environment(plan, root),
            'SQLite build source/compiler/executable tuple differs')
    actual_sources, sdk, dep = sources(plan['repo']), tree(Path(plan['stock_go']).parent.parent), dependency(plan)
    require(receipt['sources'] == actual_sources and receipt['sdk'] == sdk and receipt['dependency'] == dep,
            'SQLite source/SDK/dependency closed inventory differs')
    require(all(f'{module} {version} {checksum}' in (cwd / 'go.sum').read_text().splitlines()
                for module, (version, checksum) in DEPENDENCIES.items()), 'SQLite module sum is not source pinned')
    inputs = actual_sources | sdk | dep
    for verification, expected in verification_files(plan, root).items():
        require(regular(verification).read_bytes() == expected, 'SQLite verification module changed original dependency pins')
        inputs[str(verification)] = sha(verification)
    paths = {str(path), str(regular(receipt['plan'])), str(regular(receipt['binary']))} | set(inputs)
    parents = set()
    require(set(receipt['commands']) == set(COMMANDS), 'SQLite actual build commands omitted')
    for name, argv in recipes(plan, root).items():
        expected = {suffix: sha(regular(root / (name + '.' + suffix))) for suffix in SUFFIXES}
        require(receipt['commands'][name] == expected, 'SQLite build raw command changed')
        paths.update(str(root / (name + '.' + suffix)) for suffix in SUFFIXES)
        _, process = journal(root, name, argv, cwd, receipt['environment'])
        parents.add(process['parent_pid'])
    require(len(parents) == 1, 'SQLite build commands borrowed different producers')
    terminal_controls.compiler_identity(Path(plan['stock_go']).parent.parent,
        (root / 'version.stdout').read_bytes(), (root / 'environment.stdout').read_bytes())
    require((root / 'verify-modules.stdout').read_bytes() == b'all modules verified\n', 'SQLite source dependencies not verified by actual Go')
    modules = package_graph((root / 'packages.stdout').read_bytes(), plan, inputs)
    info = build_info((root / 'build-info.stdout').read_bytes(), Path(receipt['binary']), modules)
    host = (root / 'version.stdout').read_text().strip().split()[-1].split('/')
    require([info.get('GOOS'), info.get('GOARCH')] == host, 'SQLite oracle must use compiler native host target')
    if config is not None:
        require(plan['repo'] == config['repo'] and plan['stock_go'] == config['primary']['compiler'], 'SQLite oracle belongs to another candidate/compiler')
        if 'final_profiles' in config:
            require(plan['stock_go'] == config['final_profiles']['stock_go'], 'SQLite oracle stock compiler differs from profile compiler')
    return receipt, paths


def freeze_inputs(config, entry):
    require(isinstance(entry, dict) and set(entry) == {'build_receipt'}, 'SQLite requires fixed Go oracle build receipt, no caller argv')
    _, paths = inspect_build(entry['build_receipt'], config)
    return paths


def validate_build(config, entry=None):
    entry = config.get('sqlite_oracle') if entry is None else entry
    paths = freeze_inputs(config, entry)
    for path in paths:
        live.sealed_input(config, path)
    return read(entry['build_receipt'])


def run_identity(config, root, request, receipt):
    info, verified = journal(root, 'native-Go-oracle-build-info', [receipt['stock_go'], 'version', '-m', receipt['binary']], config['repo'], receipt['environment'])
    require(info == (Path(config['sqlite_oracle']['build_receipt']).parent / 'build-info.stdout').read_bytes(),
            'actual SQLite executable no longer matches retained Go build information')
    build_info(info, Path(receipt['binary']))
    raw, process = journal(root, 'native-Go-oracle', [receipt['binary'], '--input', str(request)], config['repo'], {'PATH': '/usr/bin:/bin'})
    require(verified['parent_pid'] == process['parent_pid'] and verified['reaped_monotonic_nanos'] <= process['started_monotonic_nanos'],
            'SQLite Go identity check did not precede actual oracle invocation')
    return dict(schema='gomad.sqlite-go-oracle-run/v1', build_receipt_sha256=sha(config['sqlite_oracle']['build_receipt']),
                binary_sha256=receipt['binary_sha256'], input_sha256=sha(regular(request)),
                stdout_sha256=sha(root / 'native-Go-oracle.stdout'),
                commands={name:{suffix:sha(root / (name + '.' + suffix)) for suffix in SUFFIXES}
                          for name in ('native-Go-oracle-build-info', 'native-Go-oracle')},
                pid=process['pid'], parent_pid=process['parent_pid'], clock=process['clock'],
                started_monotonic_nanos=process['started_monotonic_nanos'], reaped_monotonic_nanos=process['reaped_monotonic_nanos']), raw


def invoke(config, root, request):
    receipt = validate_build(config)
    argv = [receipt['binary'], '--input', str(request)]
    digest = sha(request)
    actual_info = command(config, root, 'native-Go-oracle-build-info', [receipt['stock_go'], 'version', '-m', receipt['binary']], timeout=900, exact_environment=receipt['environment'])
    require(actual_info == (Path(config['sqlite_oracle']['build_receipt']).parent / 'build-info.stdout').read_bytes(),
            'actual SQLite executable differs from authentic Go build before invocation')
    build_info(actual_info, Path(receipt['binary']))
    command(config, root, 'native-Go-oracle', argv, timeout=900, exact_environment={'PATH': '/usr/bin:/bin'})
    require(sha(request) == digest, 'SQLite oracle changed its authenticated request')
    identity, _ = run_identity(config, root, request, receipt)
    write(root / 'native-Go-oracle.json', identity)
    return read_run(config, root, request)


def read_run(config, root, request):
    receipt = validate_build(config)
    identity, raw = run_identity(config, root, request, receipt)
    require(read(root / 'native-Go-oracle.json') == identity, 'SQLite oracle run/input/raw process proof changed')
    result = preparation.documents(raw)
    require(len(result) == 1 and set(result[0]) == {'enumeration', 'context'}, 'SQLite oracle output shape differs')
    return result[0]


def build(plan_path, root):
    plan = plan_value(plan_path)
    require(root.is_absolute() and not root.exists() and root.parent.resolve(strict=True) == root.parent
            and all(not root.is_relative_to(Path(plan[name])) for name in ('repo', 'go_cache', 'module_cache'))
            and not root.is_relative_to(Path(plan['stock_go']).parent.parent), 'fresh canonical external SQLite build root required')
    root.mkdir(mode=0o700)
    (root / 'tmp').mkdir(mode=0o700)
    env = environment(plan, root)
    for path, raw in verification_files(plan, root).items():
        with path.open('xb') as output:
            output.write(raw)
    plan_sha256 = sha(plan_path)
    before = dict(sources=sources(plan['repo']), sdk=tree(Path(plan['stock_go']).parent.parent), dependency=dependency(plan))
    child = dict(repo=str(Path(plan['repo']) / 'tools/gomad'))
    for name, argv in recipes(plan, root).items():
        command(child, root, name, argv, timeout=900, exact_environment=env)
    require(sha(plan_path) == plan_sha256 and before == dict(sources=sources(plan['repo']), sdk=tree(Path(plan['stock_go']).parent.parent), dependency=dependency(plan)),
            'SQLite build inputs changed while compiler executed')
    receipt = dict(schema='gomad.sqlite-go-oracle-build/v1', plan=str(plan_path), plan_sha256=plan_sha256, repo=plan['repo'], stock_go=plan['stock_go'],
                   binary=str(root / 'sqlite-native'), binary_sha256=sha(root / 'sqlite-native'), environment=env, commands={name:{suffix:sha(root / (name + '.' + suffix)) for suffix in SUFFIXES} for name in COMMANDS}, **before)
    write(root / 'build.json', receipt)
    inspect_build(root / 'build.json')
    return root / 'build.json'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('build', 'check'))
    parser.add_argument('--plan', type=Path)
    parser.add_argument('--root', type=Path)
    parser.add_argument('--config', type=Path)
    args = parser.parse_args()
    try:
        if args.mode == 'build':
            require(args.plan is not None and args.root is not None, '--plan and --root required')
            result = str(build(regular(args.plan), args.root))
        else:
            require(args.config is not None, '--config required')
            from driver import configured
            result = validate_build(configured(args.config))
        print(json.dumps(result, separators=(',', ':')))
        return 0
    except (Refusal, OSError, ValueError, KeyError, TypeError) as error:
        print('REFUSED: ' + str(error), file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
