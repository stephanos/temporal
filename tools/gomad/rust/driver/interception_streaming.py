#!/usr/bin/env python3
"""Produce and independently reopen the compiled-Go synchronous callback handshake."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import time
import sys

sys.dont_write_bytecode = True

from driver import HERE, Refusal, check_files, command, read, require, sha, write
import main_cli_build
import native_roles
import preparation
import receiving

MARKER = b'gomad callback marker\n'
LIMITS = dict(fuel=1 << 50, memory_bytes=4 << 30, wall_millis=300000,
              output_bytes=8 << 20, evidence_bytes=159383552)
STREAMS = ('stdout','stderr','transcript','choice')


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def validate_protocol(report, release, streams, record, parent, identity, pid, started, ended):
    require(report['schema'] == 'gomad.interception-streaming-proof/v1'
            and release['schema'] == 'gomad.interception-streaming-release/v1', 'unknown controlled streaming protocol')
    require(type(pid) is int and pid > 0 and type(report['pid']) is int and report['pid'] == release['pid'] == pid,
            'controlled streaming report borrowed another process')
    require(report['parent_record_sha256'] == release['parent_record_sha256'] == parent
            and report['module_sha256'] == record['identity']['target'] and report['identity'] == identity,
            'controlled streaming source/module/component identity changed')
    require(report['not_complete_before_release'] is True and report['exact'] is True,
            'controlled streaming lacks held callback or exact replay')
    require(all(type(report[key]) is int for key in ('marker_callback_ns','marker_observed_ns',
            'release_written_ns','release_received_ns','terminal_ns'))
            and all(type(release[key]) is int for key in ('observed_ns','action_started_ns'))
            and started <= report['marker_callback_ns'] <= report['marker_observed_ns']
                == release['observed_ns'] <= release['action_started_ns'] < report['release_written_ns']
                <= report['release_received_ns'] <= report['terminal_ns'] <= ended,
            'controlled streaming marker/controller action/release/terminal ordering changed')
    require(report['outcome']['termination'] == record['outcome']['termination'] == 'exit'
            and report['outcome'].get('exit_code') == record['outcome']['exit_code'] == 0,
            'controlled streaming did not finish successfully after release')
    require(set(streams) == set(report['streams']) == set(STREAMS)
            and streams['stdout'].startswith(MARKER) and release['marker_sha256'] == digest(MARKER),
            'controlled streaming marker or complete stream set changed')
    for name,raw in streams.items():
        path = ('output/' if name in ('stdout','stderr') else 'evidence/') + name
        entry = record['streams'].get(path,dict(bytes=0,sha256=digest(b'')))
        require(digest(raw) == report['streams'][name] == entry['sha256'] and len(raw) == entry['bytes'],
                'controlled streaming forwarded/replayed payload differs: ' + name)


def context(recipe, root):
    main = main_cli_build.read_build(recipe['main_build'], expected_role='release')
    harness = main_cli_build.read_build(recipe['harness_build'], expected_role='interception-harness')
    require(all(main[key] == harness[key] for key in ('head','source_sha256','symlinks','host_sha256',
            'helper_sha256','runtime_implementation_sha256')), 'controlled streaming builds are not one candidate')
    plan = read(main['tools_plan'])
    source = Path(plan['repo'])
    files = dict(main['inputs'])
    for name,value in harness['inputs'].items():
        require(name not in files or files[name] == value, 'controlled streaming build input conflict')
        files[name] = value
    installation = read(native_roles.regular(recipe['installation']))
    for name,value in installation['admission']['compiler_inputs'].items():
        require(name not in files or files[name] == value, 'controlled streaming compiler input conflict')
        files[name] = value
    for name in (recipe['main_build'],recipe['harness_build'],recipe['installation']):files[name] = sha(name)
    manifest = root/'source-inputs.sha256'
    require(sha(manifest) == main['source_sha256'], 'controlled streaming original source manifest changed')
    files[str(manifest)] = sha(manifest)
    config = dict(repo=str(source), root=str(root), binary=main['binary'], binary_sha256=main['binary_sha256'],
        identity=dict(host=main['host_sha256'],helper=main['helper_sha256']), files=files,
        source_manifest=str(manifest), source_sha256=main['source_sha256'], installations=dict(streaming=recipe['installation']))
    receiving.Retained(config).installed(installation,installation['admission']['guest_configuration'],1 << 50,4 << 30,
        expected_build_tags=installation.get('build_tags',[]))
    require(installation['runtime_implementation_sha256'] == main['runtime_implementation_sha256'],
            'controlled streaming installation runtime changed')
    check_files(files)
    env = dict(PATH='/usr/bin:/bin',HOME=str(root/'home'),TMPDIR=str(root/'tmp'),TZ='UTC',LC_ALL='C',
        GOENV='off',GOTOOLCHAIN='local',GOWORK='off',CGO_ENABLED='0',GOFLAGS='',
        GOMAXPROCS='1',GOCACHE=plan['caches']['go_cache'],GOMODCACHE=plan['caches']['go_mod_cache'],GOPROXY='off',GOSUMDB='off')
    return config,main,harness,installation,env


def baseline_argv(config, recipe, root):
    return [config['binary'],'--installation',recipe['installation'],'explore','--json','--choices',
        '--seeds=1','--parallel=1','--execution-timeout=5m','--overall-timeout=10m',
        '--keep-successes=all','--success-limit=1','--success-bytes=536870912',
        '--artifacts',str(root/'baseline'),'--working-dir',str(Path(config['repo'])/'tools/gomad'),'--build-tag','test_dep',
        'go-run','./cmd/gomad/testdata/seedfree','--','stream']


def authenticate_command(config, root, name, argv, env, recipe):
    row=read(root/(name+'.command.json'))
    process=read(root/(name+'.process.json'))
    require(row['argv'] == argv and row['cwd'] == config['repo'] and type(row['exit']) is int and row['exit'] == 0
            and row['observation'] == 'completed' and row['environment_mode'] == 'exact' and row['clock'] == 'CLOCK_MONOTONIC'
            and row['effective_environment'] == env and process['active'] is False
            and type(row['pid']) is int and row['pid'] > 0 and type(process['exit']) is int
            and process['pid'] == row['pid'] and process['exit'] == row['exit'] and process['stage_pid'] == recipe['pid']
            and all(process[key] == row[key] for key in ('argv','parent_pid','clock','started_monotonic_nanos','reaped_monotonic_nanos','observation','start_ns','resources'))
            and row['parent_pid'] == process['parent_pid'] == recipe['pid']
            and row['start_ns'] <= process['end_ns'] <= row['end_ns']
            and recipe['started_ns'] <= row['started_monotonic_nanos'] < row['reaped_monotonic_nanos'] <= recipe['deadline_ns'],
            'controlled streaming command/process/lifecycle differs from original admission')
    return row


def parent(config, root):
    loader=importlib.util.spec_from_file_location('streaming_campaign_reader',HERE.parents[1]/'testdata/parity/campaign-check.py')
    campaign=importlib.util.module_from_spec(loader);loader.loader.exec_module(campaign)
    paths=list((root/'baseline/campaigns').glob('campaign-*'))
    require(len(paths) == 1, 'controlled streaming requires one actual baseline campaign')
    observed=campaign.authenticate_campaign(paths[0])
    require(observed['plan']['selection'] == '1' and len(observed['executions']) == 1,
            'controlled streaming baseline original seed/instance changed')
    execution=observed['executions'][0]
    require(execution['ordinal'] == 0 and execution['seed'] == 1, 'controlled streaming baseline borrowed another seed')
    artifact=paths[0]/execution['artifact']
    retained=receiving.Retained(config)
    row=read(root/'baseline.command.json')
    retained.capture_preparations(row)
    manifest=retained.artifact(artifact)
    provenance=read(artifact/'target/provenance.json')
    inputs=provenance['inputs']
    installation=read(config['installations']['streaming'])
    require(inputs['kind'] == 'go-run' and inputs['source'] == './cmd/gomad/testdata/seedfree'
            and inputs['directory'] == str(Path(config['repo'])/'tools/gomad') and inputs['arguments'] == ['stream']
            and inputs['go_version'] == 'go1.27.1'
            and inputs['tags'] == sorted(set(installation.get('build_tags',[]))|{'test_dep'}), 'controlled streaming replaced original compiled-Go fixture')
    source=str(Path(config['repo'])/'tools/gomad/cmd/gomad/testdata/seedfree/main.go')
    require(inputs['sources'].get(source) == config['files'].get(source) and source in config['files'],
            'controlled streaming compiled source is not the frozen Go fixture')
    retained.provenance(dict(provenance=provenance))
    sidecar,owner=retained.prepared_observation(paths[0],provenance)
    preparation.Inventory(config).project(sidecar,installation,inputs,owner_pid=owner)
    record=manifest['record']
    require(record['outcome']['class'] == 'success' and record['outcome']['termination'] == 'exit'
            and record['outcome']['exit_code'] == 0 and record['limits'] == LIMITS,
            'controlled streaming baseline outcome or original resource bounds changed')
    return artifact,manifest


def evaluate(root, execute=False):
    recipe=read(native_roles.regular(root/'recipe.json'))
    require(recipe['schema'] == 'gomad.interception-streaming-inputs/v1' and recipe['root'] == str(root)
            and type(recipe['pid']) is int and recipe['pid'] > 0
            and type(recipe['started_ns']) is int and recipe['deadline_ns']-recipe['started_ns'] == 600_000_000_000,
            'controlled streaming original whole admission changed')
    config,main,harness,installation,env=context(recipe,root)
    def remaining():
        value=(recipe['deadline_ns']-time.monotonic_ns())/1e9
        require(value > 0,'controlled streaming original whole deadline exceeded')
        return value
    argv=baseline_argv(config,recipe,root)
    if execute:command(config,root,'baseline',argv,timeout=remaining(),exact_environment=env,private_observations=config)
    authenticate_command(config,root,'baseline',argv,env,recipe)
    artifact,manifest=parent(config,root)
    argv=[harness['binary'],str(artifact),str(root/'controlled')]
    if execute:command(config,root,'controlled',argv,timeout=remaining(),exact_environment=env)
    actual=authenticate_command(config,root,'controlled',argv,env,recipe)
    output=root/'controlled'
    report,release=(read(native_roles.regular(output/name)) for name in ('report.json','controller-release.json'))
    streams={name:native_roles.regular(output/name).read_bytes() for name in STREAMS}
    require((output/'marker').read_bytes() == MARKER,'controlled streaming delivered marker changed')
    identity=dict(host_sha256=main['host_sha256'],helper_sha256=main['helper_sha256'],
        runtime_implementation_sha256=main['runtime_implementation_sha256'],engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1',release_role=True)
    validate_protocol(report,release,streams,manifest['record'],manifest['record_sha256'],identity,
        actual['pid'],actual['started_monotonic_nanos'],actual['reaped_monotonic_nanos'])
    require(actual['pid'] != read(root/'baseline.command.json')['pid'],'controlled streaming reused baseline producer PID')
    require((root/'controlled.stdout').read_bytes() == (root/'controlled.stderr').read_bytes() == b'',
            'controlled streaming harness printed unbound output')
    check_files(config['files'])
    _,source_hash,head,symlinks=main_cli_build.source_state(Path(config['repo']),read(main['gnu_plan'])['git'])
    require((source_hash,head,symlinks) == (main['source_sha256'],main['head'],main['symlinks']),
            'controlled streaming candidate changed during execution')
    if execute:remaining()
    return dict(schema='gomad.interception-streaming-receiving/v1',artifact=str(artifact),
        parent_record_sha256=manifest['record_sha256'],module_sha256=report['module_sha256'],
        identity=identity,producer_pid=actual['pid'],exact=True,controlled_compiled_go=True)


def produce(main_build, harness_build, installation, root):
    started=time.monotonic_ns()
    require(root.is_absolute() and root.resolve() == root and not root.exists(),'controlled streaming requires fresh canonical output')
    main=main_cli_build.read_build(main_build,expected_role='release')
    repo=Path(read(main['tools_plan'])['repo'])
    require(not root.is_relative_to(repo),'controlled streaming outputs must remain outside frozen source')
    root.mkdir(mode=0o700)
    for name in ('home','tmp'):(root/name).mkdir(mode=0o700)
    files,source_hash,head,symlinks=main_cli_build.source_state(repo,read(main['gnu_plan'])['git'])
    require((source_hash,head,symlinks) == (main['source_sha256'],main['head'],main['symlinks']),'controlled streaming candidate changed before admission')
    (root/'source-inputs.sha256').write_text(''.join(value+'  '+name+'\n' for name,value in sorted(files.items())))
    write(root/'recipe.json',dict(schema='gomad.interception-streaming-inputs/v1',root=str(root),
        main_build=str(main_build),harness_build=str(harness_build),installation=str(installation),
        pid=os.getpid(),started_ns=started,deadline_ns=started+600_000_000_000))
    result=evaluate(root,execute=True)
    write(root/'proof.json',result)
    return result


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root',type=Path,required=True)
    parser.add_argument('--read-only',action='store_true')
    for name in ('main-build','harness-build','installation'):parser.add_argument('--'+name,type=Path)
    args=parser.parse_args()
    if args.read_only:result=evaluate(args.root)
    else:
        require(all(getattr(args,name) is not None for name in ('main_build','harness_build','installation')),'actual main/harness build and installation required')
        result=produce(args.main_build,args.harness_build,args.installation,args.root)
    print(json.dumps(result,sort_keys=True))


if __name__=='__main__':main()
