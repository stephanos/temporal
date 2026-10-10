import datetime
import hashlib
import json
import os
import re
from pathlib import Path
import shutil
import subprocess
import sys
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal/.worktrees/fn-109-73-unix-mode-candidate')
PRIMARY = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
PACKET = ROOT / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-73/worker'
OUTPUT = ROOT / '.flow/tmp/fn10973-evidence'
GO_ROOT = Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64')
GO = str(GO_ROOT / 'bin/go')
TOOLS = Path('/tmp/fn109-lint-tools.ZdNe1t50')
SPEC = '.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md'
SELECTED = 'tools/gomad3/runner/runner_mode_unix_test.go'
BASE = 'ca4d8b88cf95da0b0a911efdc3a18f89bcbb68ae'
ORIGINAL = '951c5516e9e7b3066e7e069adda9565cfd68844c'
CONTROL = '^Test(PreparationDependenciesForwardRealFixtureInputs|PreparationDependenciesOperationErrorsRemainUnchanged|PreparationDependenciesFailuresStopAtOriginalStages|PreparationDependenciesKeepRealDefaultsAndBootstrapGuard|InjectionCharacterizationIsolatedPreparationDependencies|InjectionCharacterizationIsolatedExploreRejectsEverySubstitution)$'
LINT_ARGS = ['SHELL=/bin/sh', 'GOLANGCI_LINT_BASE_REV='+ORIGINAL, 'GOLANGCI_LINT_FIX=false', 'GOLANGCI_LINT='+str(TOOLS/'golangci-lint-v2.13.0'), 'ERRORTYPE='+str(TOOLS/'errortype'), 'ALL_TEST_TAGS=test_dep']

def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def save(name, value):
    path = OUTPUT/name
    with path.open('x') as stream:
        json.dump(value, stream, sort_keys=True, indent=2)
        stream.write('\n')
    return sha(path)

def git(*args):
    return subprocess.check_output(['/usr/bin/git', *args], cwd=ROOT).decode()

def sources():
    paths = git('ls-files', '-z', '--cached', '--others', '--exclude-standard').split('\0')
    inputs = [ROOT/p for p in paths if p and not p.startswith('.flow/')]
    inputs += list((PACKET).glob('*.py'))
    inputs += [PRIMARY/SPEC, ROOT/SPEC, ROOT/'.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.73.md', PACKET.parent/'admission.md', PACKET.parent/'worker-admission-20261010.md', ROOT/'.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/remaining-scripted-fixture-survey.md']
    return {str(p): sha(p) for p in sorted(set(inputs)) if p.is_file()}

def tools():
    paths = ['/usr/bin/python3', sys.executable, '/usr/bin/perl', '/usr/bin/bash', '/bin/sh', '/usr/bin/make', '/usr/bin/git', '/usr/bin/grep', '/usr/bin/sed', '/usr/bin/uname', '/usr/bin/cmp', '/usr/bin/sha256sum', '/usr/bin/env', GO, str(GO_ROOT/'bin/gofmt'), str(TOOLS/'golangci-lint-v2.13.0'), str(TOOLS/'errortype')]
    paths += [str(p) for p in (GO_ROOT/'pkg/tool/linux_arm64').iterdir() if p.is_file()]
    return {str(Path(p).resolve()): sha(p) for p in sorted(set(paths))}

def environment():
    env = dict(os.environ)
    for key in ('BASH_ENV', 'GOROOT', 'GOMADSEED', 'GOMAD3_CHILD_SEED'):
        env.pop(key, None)
    env.update(PATH=str(GO_ROOT/'bin')+':/usr/bin:/bin', GOCACHE='/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/go-cache', TMPDIR='/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/tmp', GOTMPDIR='/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/tmp', GOMODCACHE='/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc', GOPROXY='file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download', GOSUMDB='off', GOENV='off', GOWORK='off', GOTOOLCHAIN='local', GOFLAGS='', TZ='UTC', CGO_ENABLED='0')
    return env

def settings(env, cwd):
    return json.loads(subprocess.check_output([GO, 'env', '-json'], cwd=cwd, env=env))

def run(name, argv, cwd, extra=None):
    assert Path.cwd().resolve() == ROOT.resolve()
    env = environment()
    env.update(extra or {})
    source_before, tool_before = sources(), tools()
    assert source_before[str(PRIMARY/SPEC)] == '851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c'
    assert tool_before[GO] == '1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64'
    assert source_before[str(ROOT/'cmd/tools/lintcode/main.go')] == 'f7ecb5ea8c1686defa55141fdfbcec7d56212ed6af22972a1802966c5dc6f761'
    binding = {'workspace': str(ROOT), 'head': git('rev-parse', 'HEAD').strip(), 'base': BASE, 'argv': argv, 'cwd': str(cwd), 'effective_selected_environment': {k:env[k] for k in ('PATH','GOCACHE','TMPDIR','GOTMPDIR','GOMODCACHE','GOPROXY','GOSUMDB','GOENV','GOWORK','GOTOOLCHAIN','GOFLAGS','TZ','CGO_ENABLED')}, 'extra_environment': extra, 'absent': ['BASH_ENV','GOROOT','GOMADSEED','GOMAD3_CHILD_SEED'], 'source_manifest': source_before, 'tools_manifest': tool_before, 'actual_go_settings': settings(env,cwd), 'wrapper_sha256': sha(__file__), 'limitations': 'Reused shared Go/build/module caches; selected environment and executable files bound, inherited unrelated environment and shared libraries not exhaustively captured; no hermetic or native qualification claim.'}
    binding_sha = save(name+'-binding.json', binding)
    start = datetime.datetime.now(datetime.timezone.utc).isoformat()
    clock = time.monotonic()
    log = OUTPUT/(name+'.log')
    with log.open('xb') as stream:
        try:
            result = subprocess.run(argv, cwd=cwd, env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=600)
            rc = result.returncode
        except subprocess.TimeoutExpired:
            rc = 124
    elapsed = time.monotonic()-clock
    end = datetime.datetime.now(datetime.timezone.utc).isoformat()
    source_after, tool_after = sources(), tools()
    actual_after = settings(env,cwd)
    before_settings = dict(binding['actual_go_settings'])
    after_settings = dict(actual_after)
    for value in (before_settings, after_settings):
        value['GOGCCFLAGS'] = re.sub(r'(?<=/tmp/)go-build[0-9]+(?==/tmp/go-build)', 'go-build<VOLATILE>', value['GOGCCFLAGS'])
    stable_settings = before_settings == after_settings
    receipt = {'binding_sha256':binding_sha, 'argv':argv,'cwd':str(cwd),'start_utc':start,'end_utc':end,'elapsed_seconds':elapsed,'exit_code':rc,'log_sha256':sha(log),'source_before_after_equal':source_before==source_after,'tools_before_after_equal':tool_before==tool_after,'go_settings_before_after_equal':binding['actual_go_settings']==actual_after,'go_settings_stable_except_GOGCCFLAGS':stable_settings,'source_after':source_after,'tools_after':tool_after,'actual_go_settings_after':actual_after,'all_commands_terminal':True}
    save(name+'.json',receipt)
    print(json.dumps({k:receipt[k] for k in ('exit_code','elapsed_seconds','log_sha256','source_before_after_equal','tools_before_after_equal','go_settings_before_after_equal')}), flush=True)
    assert source_before==source_after and tool_before==tool_after and stable_settings
    return rc

if __name__ == '__main__':
    phase = sys.argv[1]
    module = ROOT/'tools/gomad3'
    if phase in ('before', 'after'):
        if not (OUTPUT/(phase+'-selected.json')).exists():
            run(phase+'-selected', [GO,'test','-tags','test_dep','-count=1','-json','-timeout','90s','-run','^TestRunEnforcesBatchModesIndependentOfUmask$','./runner'], module)
        run(phase+'-controls', [GO,'test','-tags','test_dep','-count=1','-json','-timeout','90s','-run',CONTROL,'./runner'], module)
        run(phase+'-public-guard', [GO,'test','-tags','test_dep','-count=1','-json','-timeout','90s','-run','^TestPortableProfilePublicGuardsRemainFirst$','./deterministicio'], module)
        run(phase+'-ordinary', [GO,'test','-tags','test_dep','-count=1','-json','-timeout','300s','./runner'], module)
        run(phase+'-aggregate-lint', ['/usr/bin/make','lint-code-gomad3',*LINT_ARGS], ROOT)
    elif phase == 'gates':
        run('gofmt', [str(GO_ROOT/'bin/gofmt'),'-d',SELECTED], ROOT)
        run('host-vet', [GO,'vet','-tags','test_dep','./runner','./internal/gomadtool/conformance','./runner/internal/execution'], module)
        run('errortype', [str(TOOLS/'errortype'),'-tags','test_dep','./runner'], module)
        run('runner-lint', [str(TOOLS/'golangci-lint-v2.13.0'),'run','--config',str(ROOT/'.github/.golangci.yml'),'--build-tags','test_dep','./runner'], module)
        run('fast-lint', ['/usr/bin/make','lint-code-fast',*LINT_ARGS], ROOT)
        for goos,goarch in (('linux','amd64'),('darwin','arm64')):
            run('vet-'+goos+'-'+goarch,[GO,'vet','-tags','test_dep','./runner','./internal/gomadtool/conformance','./runner/internal/execution'],module,{'GOOS':goos,'GOARCH':goarch})
        run('architecture', [GO,'test','-tags','test_dep','-count=1','-json','-timeout','300s','-run','^Test(PackageArchitecture|PureModulesHaveNoHostEffects|PublicPackagesDoNotExportTypeAliases|PublicPackagesDoNotExportForwardingAliases|RunnerRequestsCompileInExternalModule|RunnerExecutionInjectionIsPrivate|ArchitectureEffectFixtures|ArchitecturePublicSignatureFixtures|ExactModuleEdges)$','.'],module)
