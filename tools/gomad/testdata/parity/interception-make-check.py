#!/usr/bin/env python3
"""Execute one explicitly granted Make proof stage using supplied immutable inputs.

No compiler/build/guest process runs at import time. Raw outputs precede semantic
projection; this driver neither manufactures fixtures nor changes the Makefile.
"""
import argparse
import hashlib
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import sys

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
CORPUS = json.loads((HERE / 'interception-make-corpus.json').read_text())


def load_checker_module(name, path):
    loader = SourceFileLoader(name, str(path))
    checker = module_from_spec(spec_from_loader(loader.name, loader))
    loader.exec_module(checker)
    return checker


def load_checker():
    return load_checker_module('make_interception', HERE / 'interception-check.py')


def digest(path):
    result = hashlib.sha256()
    with path.open('rb') as source:
        for chunk in iter(lambda: source.read(1 << 20), b''):
            result.update(chunk)
    return result.hexdigest()


def write(path, value):
    path.write_text(json.dumps(value, sort_keys=True, indent=2) + '\n')


def make_args(race=None):
    args = ['make', 'functional-test', 'TEST_SHUFFLE_FLAG=off',
            'PERSISTENCE_TYPE=sql', 'PERSISTENCE_DRIVER=sqlite',
            'TEST_ARGS=-run=' + CORPUS['selector'].replace('$', '$$') + ' -v -count=1', 'TEST_TIMEOUT=5m']
    args += [name + '=' + CORPUS['package'] for name in
             ('FUNCTIONAL_TEST_ROOT', 'FUNCTIONAL_TEST_NDC_ROOT', 'FUNCTIONAL_TEST_XDC_ROOT')]
    if race is not None:
        args.append('TEST_RACE_FLAG=' + race)
    return args


def selected_environment(compiler):
    env = dict(os.environ)
    forbidden = [key for key in env if key.startswith('GOMAD_')]
    if forbidden:
        raise ValueError('remove inherited Gomad controls: ' + ', '.join(sorted(forbidden)))
    for key in ('GOFLAGS', 'GOOS', 'GOARCH', 'GOROOT', 'GOWORK', 'CGO_ENABLED'):
        if env.get(key):
            raise ValueError('remove inherited build override: ' + key)
    env['PATH'] = str(compiler.parent) + os.pathsep + env.get('PATH', '')
    env['GOENV'] = 'off'
    env['GOTOOLCHAIN'] = 'local'
    env['GOWORK'] = 'off'
    return env


def run(argv, cwd, env, output, name, timeout):
    out, err = output / (name + '.stdout'), output / (name + '.stderr')
    write(output / (name + '.command.json'), dict(argv=argv, cwd=str(cwd),
          environment={key: env.get(key) for key in
                       ('PATH', 'GOENV', 'GOTOOLCHAIN', 'GOWORK', 'GOFLAGS', 'GOOS', 'GOARCH', 'CGO_ENABLED')}))
    with out.open('wb') as stdout, err.open('wb') as stderr:
        child = subprocess.Popen(argv, cwd=cwd, env=env, stdout=stdout, stderr=stderr,
                                 start_new_session=True)
        write(output / (name + '.started.json'), dict(pid=child.pid))
        expired = False
        try:
            status = child.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            expired = True
            os.killpg(child.pid, signal.SIGTERM)
            try:
                status = child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(child.pid, signal.SIGKILL)
                status = child.wait()
    result = dict(status=status, timed_out=expired, stdout_sha256=digest(out), stderr_sha256=digest(err))
    write(output / (name + '.result.json'), result)
    if expired:
        raise ValueError(name + ': external deadline expired; no retry')
    return result


def usage_bytes(raw):
    # Go's flag package prints the temporary native executable in its usage line.
    return re.sub(rb'(?m)^Usage of [^\n]+:\n', b'Usage of $HOST_TOOL:\n', raw)


def authenticate_positive(artifacts, binary, compiler_sha256, installation_sha256):
    checker = load_checker()
    rows = []
    wrappers, invocations, launches = set(), set(), set()
    mapping_checker = load_checker_module('make_shim_mapping', HERE / 'interception-shim-check.py')
    for retained in mapping_checker.authenticated_plans(artifacts):
        root = retained['root']
        validated = retained['validated']
        plan = validated['plan']
        prepared_path = retained['artifact']
        prepared = retained['prepared']
        origin = retained['origin']
        executions = validated['executions']
        if not executions:
            continue
        checker.equal(len(executions), 1, 'make.one_seed_per_recipe')
        checker.equal(plan['origin']['origin'], 'intercepted-go', 'make.origin')
        original = [bytes.fromhex(value).decode('utf-8') for value in origin['invocation']['original_argv']]
        checker.equal(original[1], 'test', 'make.original_subcommand')
        if CORPUS['package'] not in original or '-run=' + CORPUS['selector'] not in original:
            raise ValueError('make original argv lacks exact accepted package/selector')
        for argument in ('-persistenceType=sql', '-persistenceDriver=sqlite', '-count=1'):
            if argument not in original:
                raise ValueError('make original argv missing ' + argument)
        for execution in executions:
            if execution['artifact'] is None:
                raise ValueError('make execution was not retained')
            artifact = root / execution['artifact']
            record = checker.campaign.authenticate_artifact(artifact)
            checker.equal(checker.authenticate_origin(artifact, record, plan['origin']), origin, 'make.execution_origin')
            checker.equal(record['outcome']['class'], 'success', 'make.actual_success')
            checker.equal(record['outcome']['exit_code'], 0, 'make.actual_exit')
            for key, expected in dict(memory_bytes=4 << 30, fuel=1 << 50, wall_millis=300_000).items():
                checker.equal(record['limits'][key], expected, 'make.original_limit.' + key)
            stdout = (artifact / 'output/stdout').read_bytes()
            if b'PASS' not in stdout or b'TestFrontendSystemInfo' not in stdout:
                raise ValueError('make retained stdout lacks selected test PASS')
        driver = origin['invocation']
        checker.equal(driver['compiler_sha256'], compiler_sha256, 'make.compiler_binding')
        provenance = checker.campaign.canonical_file(prepared_path / 'target/provenance.json')
        mapping_checker.authenticate_installations(provenance, origin,
            [['go.temporal.io/server/tests/gomadfunctional', installation_sha256]])
        rewritten = [bytes.fromhex(value).decode('utf-8') for value in driver['driver_argv']]
        if not any('run-guest' in value and str(binary) in value for value in rewritten):
            raise ValueError('make driver argv lacks actual Gomad -exec chain')
        wrappers.add(driver['wrapper_run_id'])
        invocations.add(driver['invocation_id'])
        launches.add(origin['launch']['invocation_id'])
        original[0] = '$GO_SHIM'
        rows.append(dict(original_argv=original, origin=origin['origin'], executions=len(executions),
                         outcome='success', configuration_sha256=prepared['identity']['configuration']))
    checker.equal(len(rows), CORPUS['positive_executions'], 'make.three_sequential_recipes')
    checker.equal((len(wrappers), len(invocations), len(launches)), (1, 3, 3), 'make.identity_isolation')
    return rows


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', required=True, type=Path)
    parser.add_argument('--binary-sha256', required=True)
    parser.add_argument('--compiler', required=True, type=Path)
    parser.add_argument('--compiler-sha256', required=True)
    parser.add_argument('--installation', required=True, type=Path)
    parser.add_argument('--installation-sha256', required=True)
    parser.add_argument('--checkout', required=True, type=Path)
    parser.add_argument('--accepted-selector', required=True, choices=[CORPUS['selector']])
    parser.add_argument('--stage', required=True, choices=CORPUS['stages'])
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--timeout', required=True, type=int, help='granted external stage deadline in seconds')
    args = parser.parse_args()
    if args.timeout <= 0:
        parser.error('positive explicit stage deadline required')
    for path, sha in ((args.binary, args.binary_sha256), (args.compiler, args.compiler_sha256),
                      (args.installation, args.installation_sha256)):
        if not path.is_absolute() or path.is_symlink() or not path.is_file() or digest(path) != sha:
            raise ValueError('exact regular immutable input required: ' + str(path))
    if not args.checkout.is_absolute() or not (args.checkout / 'go.mod').is_file():
        raise ValueError('absolute Temporal module checkout required')
    if not args.output.is_absolute():
        raise ValueError('absolute fresh output required')
    if (args.checkout / '.bin/go').exists():
        raise ValueError('.bin/go would shadow the Gomad shim')
    log = args.checkout / 'test.log'
    if log.exists():
        raise ValueError('test.log must be absent; preserve any existing operator log first')
    env = selected_environment(args.compiler)
    args.output.mkdir(mode=0o700, parents=True, exist_ok=False)
    makefile = args.checkout / 'Makefile'
    before = digest(makefile)
    artifacts = args.output / 'artifacts'
    prefix = [str(args.binary), '--installation', str(args.installation), '--seeds', '0',
              '--artifacts', str(artifacts), '--']
    write(args.output / 'inputs.json', dict(binary=str(args.binary), binary_sha256=args.binary_sha256,
          compiler=str(args.compiler), compiler_sha256=args.compiler_sha256,
          installation=str(args.installation), installation_sha256=args.installation_sha256,
          makefile_sha256=before, corpus=CORPUS, stage=args.stage))
    result, semantic = {}, {}
    try:
        if args.stage == 'expansion':
            for name, race in (('default', None), ('race-on', 'on'), ('positive', 'off')):
                command = make_args(race)
                command.insert(1, '-n')
                result[name] = run(prefix + command, args.checkout, env, args.output, name, args.timeout)
                if result[name]['status'] != 0:
                    raise ValueError('Make expansion failed')
                raw = (args.output / (name + '.stdout')).read_bytes()
                tests = [line.decode() for line in raw.splitlines() if b'go test ' in line]
                if len(tests) != 3 or any(CORPUS['selector'] not in line for line in tests):
                    raise ValueError('expected three exact selected Make recipes')
                semantic[name] = [line.replace(str(args.checkout), '$CHECKOUT') for line in tests]
        elif args.stage == 'host-passthrough':
            for name, command in (('generator', ['run', './cmd/tools/protogen', '-h']),
                                  ('linter', ['run', './cmd/tools/lintcode', '-h']),
                                  ('host-env', ['env', 'GOOS', 'GOARCH', 'CGO_ENABLED', 'GOFLAGS'])):
                for mode, child in (('direct', [str(args.compiler)]), ('wrapped', prefix + ['go'])):
                    result[name + '-' + mode] = run(child + command, args.checkout, env, args.output,
                                                   name + '-' + mode, args.timeout)
                direct, wrapped = result[name + '-direct'], result[name + '-wrapped']
                if direct['status'] != 0 or wrapped['status'] != direct['status']:
                    raise ValueError(name + ': host tool status mismatch')
                for stream in ('stdout', 'stderr'):
                    first = (args.output / (name + '-direct.' + stream)).read_bytes()
                    second = (args.output / (name + '-wrapped.' + stream)).read_bytes()
                    if usage_bytes(first) != usage_bytes(second):
                        raise ValueError(name + ': host passthrough output mismatch: ' + stream)
                semantic[name] = dict(status=wrapped['status'], host_passthrough=True)
        else:
            if args.stage == 'direct-race':
                child = ['go', 'test', CORPUS['package'], '-race', '-run=' + CORPUS['selector']]
            else:
                race = {'default-race': None, 'race-on': 'on', 'positive': 'off'}[args.stage]
                child = make_args(race)
            result['child'] = run(prefix + child, args.checkout, env, args.output, 'child', args.timeout)
            if log.exists():
                (args.output / 'test.log').write_bytes(log.read_bytes())
                log.unlink()
            if args.stage == 'positive':
                if result['child']['status'] != 0:
                    raise ValueError('positive Make smoke did not pass')
                semantic['campaigns'] = authenticate_positive(artifacts, args.binary, args.compiler_sha256,
                                                              args.installation_sha256)
                semantic['support_cells'] = 1
            else:
                raw = b''.join((args.output / ('child.' + stream)).read_bytes() for stream in ('stdout', 'stderr'))
                refusals = [line for line in raw.lower().splitlines() if b'race' in line and
                            any(term in line for term in (b'unsupported', b'refus', b'not supported'))]
                expected = 1 if args.stage == 'direct-race' else 3
                if len(refusals) != expected or re.search(rb'(?m)^ok[ \t]', raw):
                    raise ValueError('expected exact race refusals without a passing test')
                if args.stage == 'direct-race' and result['child']['status'] != 2:
                    raise ValueError('direct shim refusal did not retain status 2')
                if list(artifacts.rglob('plan.json')):
                    raise ValueError('race refusal unexpectedly retained a campaign')
                semantic['actual_child_status'] = result['child']['status']
                semantic['race_refused'] = True
        if args.stage != 'positive' and list(artifacts.rglob('plan.json')):
            raise ValueError('non-guest stage unexpectedly retained a campaign')
        if digest(makefile) != before:
            raise ValueError('Makefile changed during proof')
        for path, sha in ((args.binary, args.binary_sha256), (args.compiler, args.compiler_sha256),
                          (args.installation, args.installation_sha256)):
            if digest(path) != sha:
                raise ValueError('immutable input changed during proof')
        write(args.output / 'semantic.json', dict(schema=CORPUS['schema'], stage=args.stage,
                                                makefile_sha256=before, observations=semantic))
        write(args.output / 'receipt.json', dict(accepted=True, commands=result))
    except BaseException as error:
        write(args.output / 'receipt.json', dict(accepted=False, error=str(error), commands=result))
        raise


if __name__ == '__main__':
    main()
