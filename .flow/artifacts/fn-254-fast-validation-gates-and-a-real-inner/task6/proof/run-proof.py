import os,pathlib,subprocess,time,json,sys
proof=pathlib.Path('/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn254-task6')
candidate=proof/'candidate'
cache=pathlib.Path('/home/agent/.cache/codex-build/umpire-milestones-20261024/fn2546-quick')
def run(name,command,cwd=candidate,target=None,extra=None):
    env={k:v for k,v in os.environ.items() if not (k.startswith('CARGO_PROFILE_') or k in ['BASH_ENV','RUSTFLAGS','CARGO_ENCODED_RUSTFLAGS','RUSTC_WRAPPER','RUSTC_WORKSPACE_WRAPPER','RUSTC_BOOTSTRAP','CARGO_BUILD_TARGET'])}
    target=target or cache/'candidate'
    (target/'tmp').mkdir(parents=True,exist_ok=True)
    env.update(CARGO_TARGET_DIR=str(target),CARGO_BUILD_JOBS='2',CARGO_INCREMENTAL='0',RUST_TEST_THREADS='2',TMPDIR=str(target/'tmp'),MISE_TRUSTED_CONFIG_PATHS=f'{cwd.resolve()}:{target.resolve()}')
    env.update(extra or {})
    started=time.monotonic()
    with (proof/f'{name}.log').open('w') as log:
        result=subprocess.run(command,cwd=cwd,env=env,stdout=log,stderr=subprocess.STDOUT)
    record={'name':name,'command':command,'cwd':str(cwd),'target':str(target),'status':result.returncode,'seconds':round(time.monotonic()-started,3),'environment':{k:env[k] for k in ['CARGO_BUILD_JOBS','CARGO_INCREMENTAL','RUST_TEST_THREADS','MISE_TRUSTED_CONFIG_PATHS','UMPIRE_QUICK_CACHE','UMPIRE_CACHE_DISABLE','UMPIRE_QUICK_CACHE_ROOT','UMPIRE_BACKENDS'] if k in env}}
    with (proof/'runs.jsonl').open('a') as out: out.write(json.dumps(record)+'\n')
    print(json.dumps(record),flush=True)
    return result.returncode
if __name__=='__main__':
    mode=sys.argv[1]
    if mode=='guest-a':
        sys.exit(run(mode,['bash','model/scripts/guest.sh','umpire-temporal-guest',str(proof/'temporal-a.wasm')],target=cache/'guest-a'))
    if mode=='guest-b':
        source=pathlib.Path('/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2546-quick')
        sys.exit(run(mode,['bash','model/scripts/guest.sh','umpire-temporal-guest',str(proof/'temporal-b.wasm')],cwd=source,target=cache/'guest-b'))
    if mode=='prebuild':
        sys.exit(run(mode,['mise','exec','--','cargo','build','--locked','--manifest-path','model/Cargo.toml','-p','umpire-temporal-tests','--test','fixtures','-p','temporal-xtask','--bin','temporal-xtask']))
    raise SystemExit('unknown mode')
