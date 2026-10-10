import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import signal
import shutil
import subprocess
import sys
import time

sys.dont_write_bytecode = True
ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal/.worktrees/fn-109-75-six-attachments-candidate')
PRIMARY = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
PACKET = ROOT / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/scripted-retention-diagnostics-inspection-20261010/worker'
OUTPUT = ROOT / '.flow/tmp/fn10975-evidence'
GO_ROOT = Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64')
GO = str(GO_ROOT / 'bin/go')
TOOLS = Path('/tmp/fn109-lint-tools.ZdNe1t50')
SPEC = '.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md'
SELECTED = ['tools/gomad3/runner/'+name for name in ('diagnostics_test.go','retention_characterization_test.go','inspect_test.go')]
BASE = 'cc0c1c90fa5d2ac3f386c7a13fb51637d144c857'
ORIGINAL = '951c5516e9e7b3066e7e069adda9565cfd68844c'
CONTROL = '^(TestPreparationDependencies(ForwardRealFixtureInputs|OperationErrorsRemainUnchanged|FailuresStopAtOriginalStages|KeepRealDefaultsAndBootstrapGuard)|TestInjectionCharacterization(IsolatedExploreRejectsEverySubstitution|IsolatedPreparationDependencies)|TestPortableProfilePublicGuardsRemainFirst)$'
TABLE = '^TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy$'
SHARED = '^Test(RetentionCapacityExhaustionFailsVisiblyForEveryStrategy|RetentionFailureLeavesNoveltyAndCountersAtTheCommittedState|GuidedAdmissionReplaysBeforeTheCorpusAdvances|RunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs|RunKeepsOneTargetCopyAcrossArtifactsRoundsAndCampaigns)$'
NESTED_LINT_ARGS = ['SHELL=/bin/sh', 'GOLANGCI_LINT_BASE_REV='+ORIGINAL, 'GOLANGCI_LINT_FIX=false', 'GOLANGCI_LINT='+str(TOOLS/'golangci-lint-v2.13.0'), 'ERRORTYPE='+str(TOOLS/'errortype'), 'ALL_TEST_TAGS=test_dep']

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

def sources(extra=()):
    paths = git('ls-files', '-z', '--cached', '--others', '--exclude-standard').split('\0')
    inputs = [ROOT/p for p in paths if p and not p.startswith('.flow/')]
    inputs += list(PACKET.glob('*.py'))
    inputs += list(PACKET.parent.glob('*.md'))
    inputs += [PRIMARY/SPEC, ROOT/SPEC, ROOT/'.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.75.md', ROOT/'.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/retention-helper-next-slice.md', ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/independent-evidence-review.md', ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/binding-successor/admission.md']
    inputs += [ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-74/worker/run.py', ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-74/worker/check-progress-complete.py', ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-74/deadline-failure-disposition-20261010.md']
    inputs += [Path(p) for p in extra]
    result = {}
    for p in sorted(set(inputs)):
        if p.is_symlink():
            result[str(p)] = {'literal_link':os.readlink(p), 'resolved':str(p.resolve()), 'target_sha256':sha(p) if p.is_file() else 'NON_FILE_TARGET'}
        elif p.is_file():
            result[str(p)] = sha(p)
        elif p.is_dir():
            result[str(p)] = 'NON_FILE_DIRECTORY_OR_GITLINK; not a consumed file'
        else:
            result[str(p)] = 'ABSENT_UNMATERIALIZED'
    return result

def tools():
    paths = ['/usr/bin/python3', sys.executable, '/usr/bin/perl', '/usr/bin/bash', '/bin/bash', '/bin/sh', '/usr/bin/make', '/usr/bin/git', '/usr/bin/grep', '/usr/bin/rm', '/usr/bin/ps', '/usr/bin/sed', '/usr/bin/uname', '/usr/bin/cmp', '/usr/bin/sha256sum', '/usr/bin/env', '/usr/bin/find', '/usr/bin/sort', '/usr/bin/head', '/usr/bin/cut', '/usr/bin/xargs', '/usr/bin/dirname', GO, str(GO_ROOT/'bin/gofmt'), str(TOOLS/'golangci-lint-v2.13.0'), str(TOOLS/'errortype')]
    paths += [str(p) for p in (GO_ROOT/'pkg/tool/linux_arm64').iterdir() if p.is_file()]
    result = {p: {'resolved':str(Path(p).resolve()), 'link':os.readlink(p) if Path(p).is_symlink() else None, 'sha256':sha(p)} for p in sorted(set(paths))}
    lookup = str(ROOT/'.bin')+':'+str(GO_ROOT/'bin')+':/usr/bin:/bin'
    result['make_exported_lookup'] = {'PATH':lookup, 'LOCALBIN':'.bin', 'SHELL':'/bin/sh', 'routes':{name:shutil.which(name,path=lookup) for name in ('go','git','grep','rm','find','make','bash','sh','ps','perl')}}
    return result

def environment():
    env = dict(os.environ)
    for key in ('BASH_ENV', 'GOROOT', 'GOMADSEED', 'GOMAD3_CHILD_SEED'):
        env.pop(key, None)
    env.update(PATH=str(GO_ROOT/'bin')+':/usr/bin:/bin', GOCACHE='/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/go-cache', TMPDIR='/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/tmp', GOTMPDIR='/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/tmp', GOMODCACHE='/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc', GOPROXY='file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download', GOSUMDB='off', GOENV='off', GOWORK='off', GOTOOLCHAIN='local', GOFLAGS='', TZ='UTC', CGO_ENABLED='0')
    return env

def settings(env, cwd):
    return json.loads(subprocess.check_output([GO, 'env', '-json'], cwd=cwd, env=env))

def group_members(pgid):
    snapshot = subprocess.check_output(['/usr/bin/ps', '-eo', 'pid=,pgid=,stat=,args=']).decode()
    return [line for line in snapshot.splitlines() if line.split()[1] == str(pgid)]

def run(name, argv, cwd, extra=None, inputs=(), go_probe=True):
    assert Path.cwd().resolve() == ROOT.resolve()
    OUTPUT.mkdir(parents=True, exist_ok=True)
    env = environment()
    env.update(extra or {})
    source_before, tool_before = sources(inputs), tools()
    assert source_before[str(PRIMARY/SPEC)] == '851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c'
    assert tool_before[GO]['sha256'] == '1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64'
    binding = {'workspace':str(ROOT), 'head':git('rev-parse','HEAD').strip(), 'base':BASE, 'argv':argv, 'cwd':str(cwd), 'effective_selected_environment':{k:env[k] for k in ('PATH','GOCACHE','TMPDIR','GOTMPDIR','GOMODCACHE','GOPROXY','GOSUMDB','GOENV','GOWORK','GOTOOLCHAIN','GOFLAGS','TZ','CGO_ENABLED')}, 'inherited_environment_value_hashes':{k:hashlib.sha256(v.encode()).hexdigest() for k,v in os.environ.items()}, 'extra_environment':extra, 'absent':['BASH_ENV','GOROOT','GOMADSEED','GOMAD3_CHILD_SEED'], 'source_manifest':source_before, 'tools_manifest':tool_before, 'actual_go_settings':settings(env,cwd) if go_probe else 'NOT_EXECUTED: non-Go observation boundary', 'wrapper_sha256':sha(__file__), 'wrapper_invocation_argv':sys.argv, 'outer_shell_literal':'cd -- /Users/stephan/Workspace/skunkworks/gomad/temporal/.worktrees/fn-109-75-six-attachments-candidate || exit 1; /usr/bin/env -u BASH_ENV /usr/bin/python3 <capture.py> <gate>', 'wall_bound_seconds':600, 'termination_grace_seconds':15, 'limitations':'Shared caches and nonselected inherited environment remain nonhermetic; executable-file bindings do not bind shared libraries or all tool installations. Developmental linux/arm64 only.'}
    binding_sha = save(name+'-binding.json', binding)
    start = datetime.datetime.now(datetime.timezone.utc).isoformat()
    clock = time.monotonic()
    log = OUTPUT/(name+'.log')
    timed_out = False
    with log.open('xb') as stream:
        process = subprocess.Popen(argv, cwd=cwd, env=env, stdout=stream, stderr=subprocess.STDOUT, start_new_session=True)
        try:
            rc = process.wait(timeout=600)
        except subprocess.TimeoutExpired:
            timed_out = True
            os.killpg(process.pid, signal.SIGTERM)
            try:
                process.wait(timeout=15)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()
            rc = 124
    members = group_members(process.pid)
    if members:
        os.killpg(process.pid, signal.SIGTERM)
        drain_deadline = time.monotonic()+15
        while group_members(process.pid) and time.monotonic() < drain_deadline:
            time.sleep(0.1)
        if group_members(process.pid):
            os.killpg(process.pid, signal.SIGKILL)
    members = group_members(process.pid)
    elapsed = time.monotonic()-clock
    end = datetime.datetime.now(datetime.timezone.utc).isoformat()
    actual_after = settings(env,cwd) if go_probe else 'NOT_EXECUTED: non-Go observation boundary'
    if go_probe:
        before_settings, after_settings = dict(binding['actual_go_settings']), dict(actual_after)
        for value in (before_settings,after_settings):
            value['GOGCCFLAGS'] = re.sub(r'(?<=/tmp/)go-build[0-9]+(?==/tmp/go-build)', 'go-build<VOLATILE>', value['GOGCCFLAGS'])
        stable_settings = before_settings == after_settings
    else:
        stable_settings = 'NOT_EXECUTED: non-Go observation boundary'
    source_after, tool_after = sources(inputs), tools()
    receipt = {'binding_sha256':binding_sha, 'argv':argv, 'cwd':str(cwd), 'start_utc':start, 'end_utc':end, 'elapsed_seconds':elapsed, 'exit_code':rc, 'child_returncode':process.returncode, 'timed_out':timed_out, 'process_group':process.pid, 'remaining_group_members':members, 'log_sha256':sha(log), 'source_before_after_equal':source_before==source_after, 'tools_before_after_equal':tool_before==tool_after, 'go_settings_stable_except_numeric_GOGCCFLAGS':stable_settings, 'source_after':source_after, 'tools_after':tool_after, 'actual_go_settings_after':actual_after, 'go_settings_probe':'EXECUTED' if go_probe else 'NOT_EXECUTED: non-Go observation boundary', 'all_commands_terminal':not members, 'post_helper_window':'source/tools captured after post-command go-env and ps helpers; normalized comparison executes within the bound wrapper'}
    save(name+'.json',receipt)
    print(json.dumps({'name':name, **{k:receipt[k] for k in ('exit_code','elapsed_seconds','log_sha256','source_before_after_equal','tools_before_after_equal','go_settings_stable_except_numeric_GOGCCFLAGS','all_commands_terminal')}}),flush=True)
    assert source_before==source_after and tool_before==tool_after and stable_settings and not members
    return rc


FOCUSED = '^(TestDiagnosticsRetainedWhenSuccessArtifactsAreDiscarded|TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy|TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs|TestOpenReportsSimulationExplorationEvidence|TestOpenReportsSimulationExplorationBoundsAndRemainingWork)$'
CAPACITY = '^TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy$'
BOUNDS = '^TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs$'
COLLATERAL = '^(TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy|TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState|TestGuidedAdmissionReplaysBeforeTheCorpusAdvances|TestRunKeepsOneTargetCopyAcrossArtifactsRoundsAndCampaigns|TestDiagnosticsPlanShardAndGuidanceKeepTheProfile|TestDiagnosticsResumeRestoresTheRecordedProfile)$'
ARCHITECTURE = '^Test(PackageArchitecture|PureModulesHaveNoHostEffects|PublicPackagesDoNotExportTypeAliases|PublicPackagesDoNotExportForwardingAliases|RunnerRequestsCompileInExternalModule|RunnerExecutionInjectionIsPrivate|ArchitectureEffectFixtures|ArchitecturePublicSignatureFixtures|ExactModuleEdges)$'

if __name__ == '__main__':
    name = sys.argv[1]
    module = ROOT/'tools/gomad3'
    test = ['go','test','-tags','test_dep','-count=1','-json']
    commands = {
        'lane-before': (['/usr/bin/python3',str(PACKET/'inspect-lane.py')],ROOT,None,False),
        'gofmt': ([str(GO_ROOT/'bin/gofmt'),'-d',*SELECTED],ROOT,None,False),
        'diff-check': (['/usr/bin/git','diff','--check'],ROOT,None,False),
        'host-vet': ([GO,'vet','-tags','test_dep','./runner','./internal/gomadtool/conformance','./runner/internal/execution'],module,None,True),
        'errortype': ([GO,'vet','-tags','test_dep','-vettool='+str(TOOLS/'errortype'),'-style-check=false','./runner'],module,None,True),
        'runner-lint': ([str(TOOLS/'golangci-lint-v2.13.0'),'run','--config',str(ROOT/'.github/.golangci.yml'),'--build-tags','test_dep','--timeout','10m','--fix=false','./runner'],module,None,True),
        'fast-lint': (['/usr/bin/make','lint-code-fast',*NESTED_LINT_ARGS],ROOT,None,True),
        'architecture': (test+['-run',ARCHITECTURE,'.'],module,None,True),
        'validate': (['/usr/bin/make','-C','tools/gomad3','validate','SHELL=/bin/sh'],ROOT,{'GOCACHE':str(module/'.toolchain/generator-cache')},True),
        'analysis': (['/usr/bin/python3',str(PACKET/'analyze.py')],ROOT,None,False),
        'final-terminal': (['/usr/bin/python3',str(PACKET/'terminal.py')],ROOT,None,False),
    }
    for phase in ('before','after'):
        commands[phase+'-ordinary'] = (test+['./runner'],module,None,True)
        commands[phase+'-focused'] = (test+['./runner','-run',FOCUSED],module,None,True)
        commands[phase+'-controls'] = (test+['-run',CONTROL,'./runner','./deterministicio'],module,None,True)
        commands[phase+'-collateral'] = (test+['-run',COLLATERAL,'./runner'],module,None,True)
        commands[phase+'-cold-bounds'] = (test+['-run',BOUNDS,'./runner'],module,None,True)
        commands[phase+'-aggregate-lint'] = (['/usr/bin/make','lint-code-gomad3',*NESTED_LINT_ARGS],ROOT,None,True)
        for strategy in ('seed','choice-exploration','simulation-exploration'):
            commands[phase+'-cold-capacity-'+strategy] = (test+['-run',CAPACITY+'/.*/^'+strategy+'$','./runner'],module,None,True)
    for goos,goarch in (('linux','amd64'),('darwin','arm64')):
        commands['vet-'+goos+'-'+goarch] = ([GO,'vet','-tags','test_dep','./runner','./internal/gomadtool/conformance','./runner/internal/execution'],module,{'GOOS':goos,'GOARCH':goarch,'CGO_ENABLED':'0'},True)
    argv,cwd,extra,probe = commands[name]
    inputs = list(OUTPUT.glob('*.json'))+list(OUTPUT.glob('*.log')) if name in ('analysis','final-terminal') else []
    code = run(name,argv,cwd,extra,inputs,probe)
    raise SystemExit(code if code >= 0 else 128-code)
