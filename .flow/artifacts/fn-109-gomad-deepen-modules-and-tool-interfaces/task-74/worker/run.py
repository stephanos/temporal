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
ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal/.worktrees/fn-109-74-retention-candidate')
PRIMARY = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
PACKET = ROOT / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-74/worker'
OUTPUT = ROOT / '.flow/tmp/fn10974-evidence'
GO_ROOT = Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64')
GO = str(GO_ROOT / 'bin/go')
TOOLS = Path('/tmp/fn109-lint-tools.ZdNe1t50')
SPEC = '.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md'
SELECTED = 'tools/gomad3/runner/retention_characterization_test.go'
BASE = '305182879f727f0e925483c41b8b8f0679770621'
ORIGINAL = '951c5516e9e7b3066e7e069adda9565cfd68844c'
CONTROL = '^(TestPreparationDependencies(ForwardRealFixtureInputs|OperationErrorsRemainUnchanged|FailuresStopAtOriginalStages|KeepRealDefaultsAndBootstrapGuard)|TestInjectionCharacterization(IsolatedExploreRejectsEverySubstitution|IsolatedPreparationDependencies)|TestPortableProfilePublicGuardsRemainFirst)$'
TABLE = '^TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy$'
SHARED = '^Test(RetentionCapacityExhaustionFailsVisiblyForEveryStrategy|RetentionFailureLeavesNoveltyAndCountersAtTheCommittedState|GuidedAdmissionReplaysBeforeTheCorpusAdvances|RunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs|RunKeepsOneTargetCopyAcrossArtifactsRoundsAndCampaigns)$'
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

def sources(extra=()):
    paths = git('ls-files', '-z', '--cached', '--others', '--exclude-standard').split('\0')
    inputs = [ROOT/p for p in paths if p and not p.startswith('.flow/')]
    inputs += list(PACKET.glob('*.py'))
    inputs += list(PACKET.parent.glob('*.md'))
    inputs += [PRIMARY/SPEC, ROOT/SPEC, ROOT/'.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.74.md', ROOT/'.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/retention-helper-next-slice.md', ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/independent-evidence-review.md', ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/binding-successor/admission.md']
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

def run(name, argv, cwd, extra=None, inputs=()):
    assert Path.cwd().resolve() == ROOT.resolve()
    OUTPUT.mkdir(parents=True, exist_ok=True)
    env = environment()
    env.update(extra or {})
    source_before, tool_before = sources(inputs), tools()
    assert source_before[str(PRIMARY/SPEC)] == '851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c'
    assert tool_before[GO]['sha256'] == '1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64'
    binding = {'workspace':str(ROOT), 'head':git('rev-parse','HEAD').strip(), 'base':BASE, 'argv':argv, 'cwd':str(cwd), 'effective_selected_environment':{k:env[k] for k in ('PATH','GOCACHE','TMPDIR','GOTMPDIR','GOMODCACHE','GOPROXY','GOSUMDB','GOENV','GOWORK','GOTOOLCHAIN','GOFLAGS','TZ','CGO_ENABLED')}, 'inherited_environment_value_hashes':{k:hashlib.sha256(v.encode()).hexdigest() for k,v in os.environ.items()}, 'extra_environment':extra, 'absent':['BASH_ENV','GOROOT','GOMADSEED','GOMAD3_CHILD_SEED'], 'source_manifest':source_before, 'tools_manifest':tool_before, 'actual_go_settings':settings(env,cwd), 'wrapper_sha256':sha(__file__), 'wall_bound_seconds':600, 'termination_grace_seconds':15, 'limitations':'Shared caches and nonselected inherited environment remain nonhermetic; executable-file bindings do not bind shared libraries or all tool installations. Developmental linux/arm64 only.'}
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
    actual_after = settings(env,cwd)
    before_settings, after_settings = dict(binding['actual_go_settings']), dict(actual_after)
    for value in (before_settings,after_settings):
        value['GOGCCFLAGS'] = re.sub(r'(?<=/tmp/)go-build[0-9]+(?==/tmp/go-build)', 'go-build<VOLATILE>', value['GOGCCFLAGS'])
    stable_settings = before_settings == after_settings
    source_after, tool_after = sources(inputs), tools()
    receipt = {'binding_sha256':binding_sha, 'argv':argv, 'cwd':str(cwd), 'start_utc':start, 'end_utc':end, 'elapsed_seconds':elapsed, 'exit_code':rc, 'child_returncode':process.returncode, 'timed_out':timed_out, 'process_group':process.pid, 'remaining_group_members':members, 'log_sha256':sha(log), 'source_before_after_equal':source_before==source_after, 'tools_before_after_equal':tool_before==tool_after, 'go_settings_stable_except_numeric_GOGCCFLAGS':stable_settings, 'source_after':source_after, 'tools_after':tool_after, 'actual_go_settings_after':actual_after, 'all_commands_terminal':not members, 'post_helper_window':'source/tools captured after post-command go-env and ps helpers; normalized comparison executes within the bound wrapper'}
    save(name+'.json',receipt)
    print(json.dumps({'name':name, **{k:receipt[k] for k in ('exit_code','elapsed_seconds','log_sha256','source_before_after_equal','tools_before_after_equal','go_settings_stable_except_numeric_GOGCCFLAGS','all_commands_terminal')}}),flush=True)
    assert source_before==source_after and tool_before==tool_after and stable_settings and not members
    return rc

if __name__ == '__main__':
    phase = sys.argv[1]
    module = ROOT/'tools/gomad3'
    if phase == 'baseline':
        run('before-ordinary',['go','test','-tags','test_dep','-count=1','-json','./runner'],module)
    elif phase in ('before','after'):
        for strategy in ('seed','choice-exploration','simulation-exploration'):
            run(phase+'-cold-'+strategy,['go','test','-tags','test_dep','-count=1','-json','-run',TABLE+'/.*/^'+strategy+'$','./runner'],module)
        run(phase+'-table',['go','test','-tags','test_dep','-count=1','-json','-run',TABLE,'./runner'],module)
        run(phase+'-controls',['go','test','-tags','test_dep','-count=1','-json','-run',CONTROL,'./runner','./deterministicio'],module)
        run(phase+'-shared',['go','test','-tags','test_dep','-count=1','-json','-run',SHARED,'./runner'],module)
        if phase == 'after':
            run('after-ordinary',['go','test','-tags','test_dep','-count=1','-json','./runner'],module)
        run(phase+'-aggregate-lint',['/usr/bin/make','lint-code-gomad3',*LINT_ARGS],ROOT)
