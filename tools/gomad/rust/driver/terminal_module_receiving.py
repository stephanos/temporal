"""Pure reader for the exact terminal module preparation and its actual outer command.

Module preparation is separate from the later Temporal execution request. No processes
are launched here, and generated compiler inputs are never promoted into frozen source.
"""
import copy
import hashlib
import os
from pathlib import Path
import re
import stat

from driver import require
import native_observations as native
import preparation
import native_roles
import primary_stages
import receiving

PROFILE = 'gomad.wasi-cooperative/v1'
TAGS = ['disable_grpc_modules', 'test_dep', 'sqlite3_dotlk']
PAYLOADS = dict(module='module.wasm', provenance='module-provenance.json', installation='installation.json',
                preparation_observations='preparation-observations.json', preparation_receipt='preparation-observations/receipt.json')


def exact(actual, expected, reason):
    require(native.canonical(actual) == native.canonical(expected), reason)


def original_configuration():
    return dict(Profile=PROFILE, Args=None, Environment=['PWD=/workspace'], WorkingDirectory='/workspace',
        WritableDirectories=['/workspace','/tmp'], CapturedInputs=dict(Manifest=dict(Bytes=0,Entries=0,File='',
        Limits=dict(DirectoryEntries=0,Files=0,PathBytes=0,Requests=0,SingleFileBytes=0,TotalBytes=0),Mappings=None,
        NotExist=0,Schema='',SHA256='',TotalBytes=0),Descriptor=None,Payloads=None),Stdin=None,EntropyKey=[0]*32,
        Clock=dict(EpochNanos=946684800000000000,ReadStepNanos=0),Limits=dict(OutputBytes=1<<20,
        TranscriptBytes=8<<20,Calls=10000,Descriptors=64,Files=1000,FilesystemBytes=8<<20,PendingEvents=4096))


def runtime_identity(config):
    hasher=hashlib.sha256(b'gomad-wasm-runtime-implementation/v1\0')
    inventory=preparation.Inventory(config)
    for name in native_roles.RUNTIME:
        raw=inventory.file_bytes(Path(config['repo'])/'tools/gomad/internal/backends/wasm/toolchain'/name)
        hasher.update(len(raw).to_bytes(8,'big'));hasher.update(raw)
    return hasher.hexdigest()


def normalized_configuration(installation, runtime_sha256):
    model=copy.deepcopy(installation['model']);model.pop('seed',None)
    model.update(mode=1,limit=1<<30,diagnostics=False)
    namespace=copy.deepcopy(installation['namespace']);namespace['limits']['output_bytes']=1<<20
    environment=dict(installation['admission']['environment'],TZ='UTC',GOMAD_IO_PROFILE='wasm',GOMAD_CHOICE_PROFILE='gomad-choice-trace/v4')
    return dict(engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1',profile=PROFILE,
        limits=dict(fuel=1<<50,memory_bytes=4<<30,wall_millis=1800000,output_bytes=1<<20,evidence_bytes=(1<<30)+(2<<20)+(128<<20)),
        arguments=['gomad-target'],wall_nanos=1800000000000,world_transition_bytes=64<<20,
        environment=environment,derive_seed_environment=True,runtime_sha256=runtime_sha256,
        model=model,namespace=namespace,storage=None,network=None)


def snapshot(path, maximum, mode=0o600, body=False):
    path=native.absolute(path)
    descriptor=os.open(path,os.O_RDONLY|os.O_NOFOLLOW|os.O_NONBLOCK|os.O_CLOEXEC)
    def identity(info):
        return (info.st_dev,info.st_ino,info.st_size,info.st_uid,info.st_mode,info.st_nlink,info.st_mtime_ns,info.st_ctime_ns)
    try:
        before=os.fstat(descriptor)
        require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1 and before.st_uid == os.geteuid()
                and 0 <= before.st_size <= maximum and (mode is None or stat.S_IMODE(before.st_mode) == mode),
                'terminal module requires bounded private unaliased publication')
        hasher=hashlib.sha256();parts=[];total=0
        while chunk:=os.read(descriptor,min(1<<20,maximum-total+1)):
            total+=len(chunk);require(total <= maximum,'terminal module input grew over bound')
            hasher.update(chunk)
            if body:parts.append(chunk)
        require(identity(before) == identity(os.fstat(descriptor)) == identity(path.lstat()) and total == before.st_size,
                'terminal module input changed during read')
        return dict(bytes=total,sha256=hasher.hexdigest()),b''.join(parts) if body else None
    finally:
        os.close(descriptor)


def private(path, maximum):
    return snapshot(path,maximum)[0]


def document(path, maximum=native.METADATA_LIMIT, mode=0o600):
    identity,raw=snapshot(path,maximum,mode,body=True)
    return native.decoded(raw),identity


def closed_sidecar(path, names):
    directories={str(parent) for name in names for parent in Path(name).parents if str(parent) != '.'}
    pending=[(path,'')];seen=set()
    while pending:
        current,relative=pending.pop()
        info=current.stat()
        require(stat.S_ISDIR(info.st_mode) and stat.S_IMODE(info.st_mode) == 0o700 and info.st_uid == os.geteuid(),
                'terminal raw directory mode/owner changed')
        with os.scandir(current) as children:
            for child in children:
                name=str(Path(relative)/child.name)
                require(not child.is_symlink(),'terminal raw symlink')
                if child.is_dir(follow_symlinks=False):
                    require(name in directories,'terminal raw directory is unlisted')
                    pending.append((Path(child.path),name))
                else:
                    require(child.is_file(follow_symlinks=False) and name in names,'terminal raw file is unlisted')
                    seen.add(name)
    require(seen == names,'terminal raw payload missing')


def invocation(config):
    require(config['stages']['terminal-module'] == dict(adapter='primary'),'terminal module requires fixed primary stage')
    for path,identity in ((config['binary'],config['binary_sha256']),(config['source_manifest'],config['source_sha256']),
                          (config['terminal_module_spec'],config['files'].get(config['terminal_module_spec']))):
        require(identity and config['files'].get(path) == identity and native.snapshot(path,native.MODULE_LIMIT)['sha256'] == identity,
                'terminal module command input is not frozen')
    spec,_=document(config['terminal_module_spec'],mode=None)
    require(set(spec) in ({'schema','repo','stock_go','toolchain_cache'}, {'schema','repo','stock_go','toolchain_cache','temporary_directory'})
            and spec['schema'] == 'gomad.native-terminal-module-preparation-input/v1' and spec['repo'] == config['repo'],
            'terminal module closed specification changed')
    compiler=native.absolute(spec['stock_go'])
    require(config['files'].get(str(compiler)) == native.snapshot(compiler,native.MODULE_LIMIT)['sha256'], 'terminal module compiler is not frozen')
    native.absolute(spec['toolchain_cache'])
    if 'temporary_directory' in spec:
        require(isinstance(spec['temporary_directory'],str) and Path(spec['temporary_directory']).is_absolute()
                and '\0' not in spec['temporary_directory'], 'terminal module temporary directory is not explicit absolute')
    executable=primary_stages.native_bootstrap(config)
    root=Path(config['root'])/'stages/terminal-module/primary/module'
    return [executable,'--prepare-terminal-module',config['terminal_module_spec'],str(root)],spec,root


def sidecar(config, owner, root, descriptor, provenance):
    path=root/'preparation-observations'
    info=native.absolute(path).stat()
    require(stat.S_ISDIR(info.st_mode) and stat.S_IMODE(info.st_mode) == 0o700 and info.st_uid == os.geteuid(), 'terminal sidecar requires private owner directory')
    receipt,identity=document(path/'receipt.json')
    require(set(receipt) == native.RECEIPT_FIELDS and receipt['schema'] == 'gomad.native-preparation-observation-receipt/v1'
            and type(owner) is int and owner > 0 and type(receipt['owner_pid']) is int and receipt['owner_pid'] == owner,
            'terminal module receipt belongs to another actual producer')
    inputs=provenance['inputs']
    commitments=dict(host_sha256=config['identity']['host'],helper_sha256=config['identity']['helper'],
        prepared_record_sha256=descriptor['provenance_sha256'],provenance_sha256=descriptor['provenance_sha256'],
        module_sha256=descriptor['module_sha256'],build_key=provenance['build_key'],
        compiler_sha256=native.digest(native.canonical(inputs['compiler'])),sources_sha256=native.digest(native.canonical(inputs['sources'])))
    require(all(receipt[key] == value for key,value in commitments.items()), 'terminal module raw receipt commitments differ')
    table=receipt['files'];required={preparation.PREFIX+'/'+phase+'/metadata.json' for phase in ('compile','admission')}
    require(type(table) is dict and 2 <= len(table) <= 34 and required <= set(table), 'terminal module bounded complete raw table required')
    files={};blobs=0
    for name,expected in table.items():
        match=re.fullmatch(r'target/preparation-inputs/(compile|admission)/(metadata[.]json|[0-9a-f]{64}[.]bin)',name)
        require(match and set(expected) == {'bytes','sha256'} and type(expected['bytes']) is int and 0 <= expected['bytes'] <= native.RAW_LIMIT
                and re.fullmatch('[0-9a-f]{64}',expected['sha256']), 'terminal module raw table entry changed')
        observed=private(path/name,native.METADATA_LIMIT if match[2] == 'metadata.json' else native.RAW_LIMIT)
        exact(observed,expected,'terminal module raw bytes changed')
        if match[2] != 'metadata.json':
            blobs+=1;require(match[2][:-4] == observed['sha256'],'terminal raw content address changed')
        files[str(path/name)]=observed['sha256']
    require(blobs <= 32,'terminal raw blob count changed')
    closed_sidecar(path,{'receipt.json',*table})
    require(private(path/'receipt.json',native.METADATA_LIMIT) == identity,'terminal receipt changed during raw read')
    return files


def frozen_compiler_inputs(config, spec):
    matching=[]
    for name in config['installations'].values():
        value=native.decoded(preparation.Inventory(config).file_bytes(name))
        if (value['profile'] == PROFILE and value['cache_root'] == spec['toolchain_cache']
                and value['compiler_path'] == spec['stock_go']
                and value['runtime_root'] == str(Path(config['repo'])/'tools/gomad/internal/model/runtimesource')
                and value['host_sha256'] == config['identity']['host'] and value['helper_sha256'] == config['identity']['helper']):
            matching.append(value['admission']['compiler_inputs'])
    require(matching,'terminal compiler cache has no frozen current installation witness')
    require(all(native.canonical(value) == native.canonical(matching[0]) for value in matching),
            'terminal frozen installation compiler inventories disagree')
    return matching[0]


def context(config, owner, spec, root):
    info=native.absolute(root).stat()
    require(stat.S_ISDIR(info.st_mode) and stat.S_IMODE(info.st_mode) == 0o700 and info.st_uid == os.geteuid(), 'terminal module output root requires private owner')
    descriptor,identity=document(root/'terminal-module.json')
    require(set(descriptor) == {'schema','recipe','configuration','memory_bytes','fuel',*PAYLOADS,*(key+'_sha256' for key in PAYLOADS)}
            and descriptor['schema'] == 'gomad.native-terminal-module/v1', 'terminal module descriptor fields/schema changed')
    exact(descriptor['recipe'],dict(kind='go-test',source='./tests',directory=config['repo'],build_tags=TAGS,target_arguments=[]), 'original terminal compilation recipe changed')
    exact(descriptor['configuration'],original_configuration(),'original terminal preparation configuration changed')
    require(type(descriptor['memory_bytes']) is int and descriptor['memory_bytes'] == 4<<30
            and type(descriptor['fuel']) is int and descriptor['fuel'] == 1<<50,'original terminal memory/fuel changed')
    for key,name in PAYLOADS.items():
        require(descriptor[key] == str(root/name),'terminal module borrowed another payload path')
        require(private(root/name,native.MODULE_LIMIT if key == 'module' else native.METADATA_LIMIT)['sha256'] == descriptor[key+'_sha256'], 'terminal module descriptor payload hash changed')
    provenance,_=document(descriptor['provenance']);installation,_=document(descriptor['installation'])
    observations,_=document(descriptor['preparation_observations'])
    exact(observations,dict(schema='gomad.native-measurement-preparation-observations/v1',artifact_root=str(root/'preparation-observations'),
        record='module-provenance.json',record_sha256=descriptor['provenance_sha256'],receipt='preparation-observations/receipt.json'),'terminal module observation root differs')
    require(set(installation)-{'excluded_build_tags'} == {'schema','admission','compiler_path','compiler_sha256','go_version','runtime_root','overlay_module_root','cache_root','profile','fuel','memory_bytes','helper_sha256','host_sha256','runtime_implementation_sha256','model','namespace','storage','network','build_tags'},
            'terminal installation field inventory changed')
    require(installation['schema'] == 'gomad.rust-wasm-installation/v1' and installation['profile'] == PROFILE
            and installation['compiler_path'] == spec['stock_go'] and installation['cache_root'] == spec['toolchain_cache']
            and installation['runtime_root'] == str(Path(config['repo'])/'tools/gomad/internal/model/runtimesource')
            and installation['overlay_module_root'] == str(Path(config['repo'])/'tools/gomad')
            and installation['fuel'] == 1<<50 and installation['memory_bytes'] == 4<<30
            and installation['host_sha256'] == config['identity']['host'] and installation['helper_sha256'] == config['identity']['helper']
            and installation['go_version'] == 'go1.27.1'
            and installation['runtime_implementation_sha256'] == runtime_identity(config) and installation['storage'] is None and installation['network'] is None
            and installation['build_tags'] == [] and not installation.get('excluded_build_tags'), 'terminal module installation tuple differs')
    admission=installation['admission']
    require(set(admission) == {'compiler_inputs','inputs_sha256','guest_configuration','configuration_sha256','host_executable_sha256','explicit_helper','environment','arguments','compile_cache_root'}, 'terminal admission field inventory changed')
    exact(admission['guest_configuration'],original_configuration(),'terminal installation captured a selected execution configuration')
    require(admission['configuration_sha256'] == native.digest(native.canonical(original_configuration())) and admission['arguments'] == []
            and admission['explicit_helper'] is None and admission['compile_cache_root'] is None
            and admission['host_executable_sha256'] == native.snapshot(primary_stages.native_bootstrap(config),native.MODULE_LIMIT)['sha256'], 'terminal installation admission changed')
    policy=original_configuration();policy.pop('CapturedInputs')
    model,namespace,environment,_=receiving.inspection_policy_projection(policy)
    exact([installation['model'],installation['namespace'],admission['environment']],[model,namespace,environment], 'terminal installed native policy projection changed')
    compiler=admission['compiler_inputs']
    exact(compiler,frozen_compiler_inputs(config,spec),'terminal compiler inventory differs from frozen installation')
    require(compiler and admission['inputs_sha256'] == native.digest(native.canonical(compiler))
            and all(config['files'].get(path) == value and native.snapshot(path,native.MODULE_LIMIT)['sha256'] == value for path,value in compiler.items()), 'terminal compiler/generated auxiliary is not frozen')
    require(installation['compiler_sha256'] == config['files'].get(spec['stock_go']), 'terminal compiler digest differs from frozen input')
    receiving.Retained(config).provenance(dict(provenance=provenance))
    inputs=provenance['inputs']
    require(provenance['scenario'] is None and provenance['network'] is None and 'runtime_workload' not in provenance
            and inputs['kind'] == 'go-test' and inputs['source'] == './tests' and inputs['directory'] == config['repo']
            and inputs['arguments'] == [] and inputs['runtime_implementation'] == installation['runtime_implementation_sha256'] and inputs['tags'] == sorted(TAGS) and inputs['soft_assertion_report_only'] is False,
            'terminal module provenance selection changed')
    exact(provenance['configuration'],normalized_configuration(installation,provenance['configuration']['runtime_sha256']), 'terminal native preparation extensions changed')
    require(inputs['environment'].get('GOMAXPROCS') == '1' and inputs['environment'].get('GOWASM') == ''
            and inputs['environment'].get('GOTMPDIR') == spec.get('temporary_directory')
            and inputs['build_flags'] == ['-mod=readonly','-trimpath','-buildvcs=false','-pgo=off','-p=1'], 'terminal scoped compiler controls changed')
    _,module=snapshot(descriptor['module'],native.MODULE_LIMIT,body=True);metadata=receiving.module_metadata(module)
    require(native.digest(module) == provenance['module_sha256'] == descriptor['module_sha256'] and metadata == provenance['module_metadata'], 'terminal actual WASM metadata differs')
    required={'config','decision','observation','finish','idle'};seen=set()
    for item in metadata['imports']:
        if item['module'].startswith('gomad_'):
            require((item['module'] == 'gomad_wasm_v1' and item['name'] in required
                    or (item['module'],item['name']) in {('gomad_wasm_v2','net_stream'),('gomad_debug_v1','exchange')})
                    and item['params'] == ['i32','i32'] and item['results'] == ['i32'], 'terminal cooperative import is unlisted or mistyped')
            if item['module'] == 'gomad_wasm_v1':seen.add(item['name'])
    require(seen == required,'terminal cooperative import set incomplete')
    files=sidecar(config,owner,root,descriptor,provenance)
    preparation.Inventory(config).project(root/'preparation-observations',installation,inputs,owner_pid=owner)
    require(any(Path(path).is_relative_to(Path(config['repo'])/'tests') for path in inputs['sources']), 'terminal selected ./tests source missing')
    return descriptor,identity,files


def capture(config, command_pid, argv, stdout):
    expected,spec,root=invocation(config)
    require(argv == expected and len(stdout) <= 64<<10,'terminal module actual command differs from fixed invocation')
    exact(native.decoded(stdout),dict(result=str(root/'terminal-module.json')), 'terminal module stdout descriptor path changed')
    descriptor,identity,files=context(config,command_pid,spec,root)
    return dict(schema='gomad.native-terminal-module-observation-seal/v1',source_sha256=config['source_sha256'],
        binary_sha256=config['binary_sha256'],producer_binary=argv[0],producer_binary_sha256=native.snapshot(argv[0],native.MODULE_LIMIT)['sha256'],
        module_root=str(root),descriptor_sha256=identity['sha256'],stdout_sha256=native.digest(stdout),
        receipt_sha256=descriptor['preparation_receipt_sha256'],files=files)


def read_module(config, root, path):
    root=Path(root)
    require(root == Path(config['root'])/'stages/terminal-module' and Path(path) == root/'primary/module/terminal-module.json', 'terminal module receiving endpoint changed')
    row,_=document(root/'primary-final.command.json',mode=None)
    process,_=document(root/'primary-final.process.json',mode=None)
    require(type(row['exit']) is int and row['exit'] == 0 and row['observation'] == 'completed' and row['cwd'] == config['repo']
            and type(row['pid']) is int and row['pid'] > 0 and type(row['start_ns']) is int and type(row['end_ns']) is int
            and row['end_ns'] >= row['start_ns'] >= config['sent_epoch']*1_000_000_000
            and 0 < row['start_ns'] and row['end_ns']-row['start_ns'] <= 3600_000_000_000
            and row['environment'] == config.get('environment',{}) and row['temporary_paths'] == config.get('temporary_paths',{})
            and type(process['pid']) is int and process['pid'] == row['pid'] and process['argv'] == row['argv'] and process['active'] is False
            and type(process['stage_pid']) is int and process['stage_pid'] > 0
            and type(process['start_ns']) is int and type(process['end_ns']) is int
            and process['start_ns'] == row['start_ns'] and row['start_ns'] <= process['end_ns'] <= row['end_ns']
            and type(process['exit']) is int and process['exit'] == 0 and process['observation'] == 'completed', 'terminal actual outer command/process did not complete')
    _,stdout=snapshot(root/'primary-final.stdout',64<<10,mode=None,body=True)
    seal=capture(config,row['pid'],row['argv'],stdout)
    exact(row['private_observations'],seal,'terminal module lacks its exact actual outer observation seal')
    commands,_,_=preparation.Inventory(config).observed(root/'primary/module/preparation-observations','admission')
    require(all(row['start_ns'] <= command['start_ns'] <= command['end_ns'] <= row['end_ns'] for command in commands.values()), 'terminal admission observations lie outside actual outer command')
    descriptor,identity=document(path)
    require(identity['sha256'] == seal['descriptor_sha256'],'terminal descriptor changed after authenticated capture')
    return descriptor
