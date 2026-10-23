"""Bounded files and explicit raw observations, never an executed compiler claim."""
import json
from pathlib import Path

import preparation
import receiving


def complete(fixture):
    root,config,p = fixture.root,fixture.config,fixture.provenance
    def pin(path,raw):
        path=Path(path);path.parent.mkdir(parents=True,exist_ok=True);path.write_bytes(raw)
        config['files'][str(path)]=receiving.digest(raw)
        return str(path)
    module=root/'tools/gomad';runtime=module/'internal/model/runtimesource';cache=root/'cache'
    sdk=root/'sdk';tools=sdk/'pkg/tool/linux_arm64';include=sdk/'pkg/include'
    base={pin(sdk/'VERSION',b'go1.27.1\n'),*(pin(tools/name,name.encode()) for name in ('asm','compile','link')),
          pin(include/'textflag.h',b'// textflag\n'),pin(include/'funcdata.h',b'// funcdata\n')}
    static=set()
    for name in preparation.STATIC:
        path=module/name
        static.add(str(path) if str(path) in config['files'] else pin(path,name.encode()))
    semantics=runtime/'gomad_semantics.go.txt'
    generated=runtime/'gomad_choicewire_generated.go.txt'
    if str(semantics) not in config['files']:pin(semantics,b'package runtime\n')
    pin(generated,b'package runtime\n// generated choicewire\n')
    original=sdk/'src/runtime/runtime.go';pin(original,b'package runtime\n')
    overlay_path=cache/'runtime-overlay'/('e'*64)/'overlay.json'
    replacements={str(original):str(semantics),str(sdk/'src/runtime/new_choice.go'):str(generated)}
    overlay=receiving.canonical(dict(Replace=replacements));pin(overlay_path,overlay)
    compiled=base|static|{str(semantics),str(generated),str(original),str(overlay_path)}
    mod=pin(module/'go.mod',b'module go.temporal.io/server/tools/gomad\n')
    sum_path=pin(module/'go.sum',b'// frozen module sums\n')
    imported=pin(sdk/'src/fmt/print.go',b'package fmt\n')
    assembly=pin(sdk/'src/runtime/asm_wasm.s',b'/* #include "ignored.h" */\n#include "common.h"\n')
    common=pin(sdk/'src/runtime/common.h',b'/* multiline\n#include "ignored.h"\n*/\n#include "textflag.h"\n')
    sources={fixture.source,mod,sum_path,imported,assembly,common,str(include/'textflag.h'),str(original),str(semantics),str(generated)}
    packets=[dict(Dir=str(Path(fixture.source).parent),GoFiles=['main.go'],Module=dict(GoMod=mod),ImportPath='go.temporal.io/server/tools/gomad/internal/backends/wasm/testdata/runtime_probes'),
             dict(Dir=str(sdk/'src/fmt'),GoFiles=['print.go'],ImportPath='fmt'),
             dict(Dir=str(sdk/'src/runtime'),GoFiles=['runtime.go','new_choice.go'],SFiles=['asm_wasm.s'],ImportPath='runtime')]
    listed=b'\n'.join(receiving.canonical(value) for value in packets)+b'\n'
    fixture.installation.update(runtime_root=str(runtime),cache_root=str(cache),overlay_module_root=str(module))
    admitted=compiled|{fixture.installation['compiler_path'],mod,sum_path}
    admission=fixture.installation['admission'];admission['compiler_inputs']={path:config['files'][path] for path in sorted(admitted)}
    admission['inputs_sha256']=receiving.digest(receiving.canonical(admission['compiler_inputs']))
    Path(config['installations']['runtime']).write_bytes(receiving.canonical(fixture.installation))
    p['inputs']['compiler']={path:config['files'][path] for path in sorted(compiled)}
    p['inputs']['sources']={path:config['files'][path] for path in sorted(sources)}
    fixture.reseal()
    executable=p['inputs']['compiler_path'];cwd=str(module)
    env=dict(p['inputs']['environment'],PATH='/compiler/bin',HOME='/compiler/home')
    go_env=receiving.canonical(dict(GOROOT=str(sdk),GOTOOLDIR=str(tools)))
    overlay_output=receiving.canonical(dict(path=str(overlay_path),identity='e'*64,implementation_sha256='d'*64))
    flags=['-mod=readonly','-pgo=off','-overlay',str(overlay_path)]
    go_list=[executable,'list',*flags,'-deps','-json',p['inputs']['source']]
    parent=1234
    compile_key=receiving.digest(receiving.canonical(dict(schema='gomad.native-compilation-inputs/v1',profile=fixture.installation['profile'],
        **{key:p['inputs'][key] for key in ('compiler','compiler_path','compiler_sha256','go_version','sources','runtime','runtime_implementation','kind','source','directory','tags','build_flags','environment')})))
    build_env=dict(env,GOROOT=str(sdk),GOCACHE=str(cache/'compilation-v1'/compile_key/'objects'))
    commands=[('compiler-version',[executable,'version'],env,b'go version go1.27.1 linux/arm64\n',64<<10,cwd),
              ('compiler-environment',[executable,'env','-json','GOTOOLDIR','GOROOT'],env,go_env,64<<10,cwd),
              ('compiler-module-cache',[executable,'env','GOMODCACHE'],env,b'/frozen/modules\n',64<<10,cwd),
              ('runtime-overlay',[executable,'run','-mod=readonly','./internal/backends/wasm/toolchain/cmd/overlay',str(sdk),str(runtime),str(cache/'runtime-overlay')],
               {key:value for key,value in dict(env,GOROOT=str(sdk)).items() if key not in ('GOOS','GOARCH','GOEXPERIMENT')},overlay_output,4<<20,cwd),
              ('selected-package',[executable,'list',*flags,p['inputs']['source']],dict(env,GOROOT=str(sdk)),(packets[0]['ImportPath']+'\n').encode(),1<<20,cwd),
              ('selected-source-closure',go_list,dict(env,GOROOT=str(sdk)),listed,128<<20,cwd),
              ('compile-module',[executable,'build',*flags,'-trimpath','-buildvcs=false','-o',str(cache/f'.rust-preparation-{parent}-0/module.wasm'),p['inputs']['source']],build_env,b'',4<<20,cwd)]
    sidecar=root/'sidecar';(sidecar/preparation.PREFIX).mkdir(parents=True)
    def phase(name,rows,snapshots):
        directory=sidecar/preparation.PREFIX/name;directory.mkdir()
        value=dict(schema='gomad.native-preparation-observations/v1',phase=name,commands=[],snapshots={},blobs={})
        def blob(raw):
            identity=receiving.digest(raw);path=f'{preparation.PREFIX}/{name}/{identity}.bin'
            (sidecar/path).write_bytes(raw);value['blobs'][identity]=dict(path=path,bytes=len(raw),sha256=identity)
            return identity
        for number,(label,argv,environment,stdout,limit,working) in enumerate(rows):
            value['commands'].append(dict(name=label,argv=argv,cwd=working,environment=environment,owner_pid=parent,pid=2000+number,
                start_ns=1000+number*100,end_ns=1050+number*100,exit=0,output_limit=limit,stdout=blob(stdout),stderr=blob(b'')))
        for label,raw in snapshots.items():value['snapshots'][label]=blob(raw)
        (directory/'metadata.json').write_bytes(receiving.canonical(value))
    base_raw=receiving.canonical(sorted(base))
    phase('compile',commands,{'compiler-base-paths':base_raw,'overlay-replacements':overlay})
    phase('admission',[commands[5],('compiler-environment',commands[1][1],dict(env,GOROOT=str(sdk)),go_env,64<<10,cwd)],{'compiler-base-paths':base_raw})
    return sidecar


def outer_seal(fixture):
    import copy
    config,root,p=fixture.config,fixture.root,fixture.provenance
    sidecar=fixture.sidecar
    campaign=sidecar.parent
    # The raw sidecar context is explicit, independent of canonical Store payloads.
    if sidecar.name!='preparation-observations':
        campaign=root/'campaigns/campaign-fixture'
        campaign.mkdir(parents=True,exist_ok=True)
        target=campaign/'preparation-observations'
        sidecar.rename(target);sidecar=target;fixture.sidecar=target
    release=root/'release';release.write_bytes(b'bounded native release stand-in')
    config['binary']=str(release);config['files'][str(release)]=receiving.digest(release.read_bytes())
    config['binary_sha256']=config['files'][str(release)];config['source_sha256']='9'*64
    fixture.installation['admission']['host_executable_sha256']=config['binary_sha256']
    Path(config['installations']['runtime']).write_bytes(receiving.canonical(fixture.installation))
    record=copy.deepcopy(fixture.record)
    record.update(schema='gomad.execution-record/v2',seed=0,virtual_time_nanos=0,execution_identity=receiving.digest(receiving.canonical(record['identity'])),
                  outcome=dict(termination='prepared',exit_code=None,reason='',**{'class':'prepared'}))
    replay=receiving.canonical(dict(schema='gomad.replay-inputs/v1',helper_sha256=config['identity']['helper'],host_sha256=config['identity']['host'],
        toolchain_sha256=p['build_key'],runtime_implementation_sha256=p['inputs']['runtime_implementation'],configuration=p['configuration']))
    record['streams']={'evidence/replay.json':dict(bytes=len(replay),sha256=receiving.digest(replay),records=1)}
    record_hash=receiving.digest(receiving.canonical(record))
    reference='prepared/execution-'+record_hash
    prepared=campaign/reference
    (prepared/'target').mkdir(parents=True);(prepared/'evidence').mkdir()
    (prepared/'target/module.wasm').write_bytes(fixture.module)
    (prepared/'target/provenance.json').write_bytes(receiving.canonical(p))
    (prepared/'evidence/replay.json').write_bytes(replay)
    files={name:dict(bytes=len(raw),sha256=receiving.digest(raw),module=name=='target/module.wasm') for name,raw in
        {'target/module.wasm':fixture.module,'target/provenance.json':receiving.canonical(p),'evidence/replay.json':replay}.items()}
    (prepared/'manifest.json').write_bytes(receiving.canonical(dict(schema='gomad.artifact-manifest/v2',record=record,record_sha256=record_hash,files=files)))
    (campaign/'plan.json').write_bytes(receiving.canonical(dict(schema='gomad.seed-campaign-plan/v2',prepared_artifact=reference)))
    raw_files={str(path.relative_to(sidecar)):dict(bytes=path.stat().st_size,sha256=receiving.digest(path.read_bytes())) for path in sidecar.rglob('*') if path.is_file()}
    for name in raw_files:(sidecar/name).chmod(0o600)
    receipt=dict(schema='gomad.native-preparation-observation-receipt/v1',owner_pid=1234,host_sha256=config['identity']['host'],helper_sha256=config['identity']['helper'],
        prepared_record_sha256=record_hash,module_sha256=receiving.digest(fixture.module),provenance_sha256=receiving.digest(receiving.canonical(p)),
        build_key=p['build_key'],compiler_sha256=receiving.digest(receiving.canonical(p['inputs']['compiler'])),sources_sha256=receiving.digest(receiving.canonical(p['inputs']['sources'])),files=raw_files)
    receipt_path=sidecar/'receipt.json';receipt_path.write_bytes(receiving.canonical(receipt));receipt_path.chmod(0o600)
    entry=dict(path=str(campaign),receipt_path=str(receipt_path),receipt_sha256=receiving.digest(receipt_path.read_bytes()),
               files={str(sidecar/name):entry['sha256'] for name,entry in raw_files.items()})
    return dict(argv=[config['binary'],'--installation',config['installations']['runtime'],'explore'],pid=1234,private_observations=dict(source_sha256=config['source_sha256'],binary_sha256=config['binary_sha256'],campaigns=[entry]))
