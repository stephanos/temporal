"""Read retained receiving evidence without executing producers or modifying reports."""
import argparse
import base64
import copy
import gzip
import hashlib
import json
from pathlib import Path
import types
import struct
import sys

import adapters
import sqlite
import receiving_cost
from driver import Refusal, Unexecuted, configured, guard, load_module, read, require, sha


workloads = load_module("receiving_workloads", Path(__file__).resolve().parents[1]/"workload_tools/workloads.py")

OWNED = (
    'runtime/32x25-fresh-and-32-forced-full-replays',
    'mandatory/exact-ten-final-result-receipt-pairs',
    'sqlite/actual-durable-four-state-primary',
    *(f'sqlite/actual-selected-{index}' for index in range(4)),
    'sqlite/actual-catalogue-reader',
    'temporal/unchanged-workflow-type-smoke',
    'temporal/actual-independent-replication',
    'temporal/actual-committed-storage-all-states',
    'temporal/actual-named-reset-retry',
    'temporal/inspection-shard-workflow-onebox',
)


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=False, allow_nan=False).encode()


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def regular(path):
    path = Path(path)
    require(path.is_absolute() and path.is_file() and not any(p.is_symlink() for p in (path, *path.parents)),
            'missing or symlinked retained file: ' + str(path))
    return path


def cloned(module, overrides):
    namespace = dict(vars(module), **overrides)
    for name, value in vars(module).items():
        if isinstance(value, types.FunctionType) and value.__module__ == module.__name__ and name not in overrides:
            namespace[name] = types.FunctionType(value.__code__, namespace, name, value.__defaults__, value.__closure__)
    return types.SimpleNamespace(**namespace)


def unique(pairs):
    value = {}
    for key,item in pairs:
        require(key not in value,'duplicate retained JSON key: '+key)
        value[key] = item
    return value


def decoded(raw, **kwargs):
    kwargs.setdefault('object_pairs_hook',unique)
    return json.loads(raw, **kwargs)


def line(raw, prefix):
    lines = [value[len(prefix):] for value in raw.decode().splitlines() if value.startswith(prefix)]
    require(len(lines) == 1 and 0 < len(lines[0].encode()) <= 1<<20,'missing/duplicate/oversized raw observation')
    return lines[0]


def history(raw, sequential=False):
    value = decoded(line(raw,'GOMAD_TEMPORAL_OBSERVATION '),object_pairs_hook=unique)
    require(set(value) == {'cluster_id','history','event_ids','types'} and value['cluster_id']
            and value['event_ids'] and len(value['event_ids']) == len(value['types'])
            and all(type(item) is int and item > 0 for item in value['event_ids'])
            and all(isinstance(item,str) and item for item in value['types'])
            and base64.b64decode(value['history'],validate=True),'incomplete actual Temporal history')
    require(not sequential or value['event_ids'] == list(range(1,len(value['event_ids'])+1)),
            'Temporal event sequence has omission/duplicate/reordering')
    require(sum(len(canonical(value[key])) for key in ('history','event_ids','cluster_id')) <= 1<<20,
            'Temporal original history evidence bound exceeded')
    return value


def storage_sections(raw):
    require(56 <= len(raw) <= 256<<20 and raw[:8] == b'GMSTORE\x01','native storage header/capacity')
    body = len(raw)-40
    require(struct.unpack_from('<Q',raw,body)[0] == body and hashlib.sha256(raw[:body]).digest() == raw[body+8:],
            'native storage length/checksum mismatch')
    require(struct.unpack_from('<II',raw,8) == (3,6),'native full storage context required')
    result, offset = {}, 16
    for expected in range(1,7):
        require(offset+12 <= body,'truncated native section header')
        tag,size = struct.unpack_from('<IQ',raw,offset)
        offset += 12
        require(tag == expected and offset+size <= body,'unordered/oversized native section')
        result[tag] = raw[offset:offset+size]
        offset += size
    require(offset == body,'trailing native storage bytes')
    return result


def module_metadata(raw):
    require(raw[:8] == b'\0asm\1\0\0\0','actual core WASM header required')
    def number(data,offset):
        value = 0
        for index in range(5):
            require(offset < len(data),'truncated WASM integer')
            byte = data[offset]; offset += 1
            value |= (byte & 127) << (7*index)
            if byte < 128:
                require(value <= (1<<32)-1,'overflowing WASM integer')
                return value,offset
        raise Refusal('oversized WASM integer')
    def text(data,offset):
        size,offset = number(data,offset)
        require(offset+size <= len(data),'truncated WASM import text')
        return data[offset:offset+size].decode(),offset+size
    types,imports,pages,seen = [],[],None,set()
    offset = 8
    while offset < len(raw):
        tag = raw[offset]; offset += 1
        size,offset = number(raw,offset)
        require(offset+size <= len(raw),'truncated WASM section')
        section = raw[offset:offset+size]; offset += size
        if tag not in (1,2,5):
            continue
        require(tag not in seen,'duplicate WASM metadata section')
        seen.add(tag)
        count,cursor = number(section,0)
        if tag == 1:
            for _ in range(count):
                require(cursor < len(section) and section[cursor] == 0x60,'unsupported WASM type')
                cursor += 1
                signature = []
                for _ in range(2):
                    length,cursor = number(section,cursor)
                    values = []
                    for _ in range(length):
                        require(cursor < len(section) and section[cursor] in (0x7f,0x7e,0x7d,0x7c),'unsupported WASM value type')
                        values.append({0x7f:'i32',0x7e:'i64',0x7d:'f32',0x7c:'f64'}[section[cursor]])
                        cursor += 1
                    signature.append(values)
                types.append(signature)
        elif tag == 2:
            for _ in range(count):
                module,cursor = text(section,cursor)
                name,cursor = text(section,cursor)
                require(cursor < len(section) and section[cursor] == 0,'nonfunction WASM import unsupported')
                index,cursor = number(section,cursor+1)
                require(index < len(types),'invalid WASM import type')
                imports.append(dict(module=module,name=name,params=types[index][0],results=types[index][1]))
        else:
            require(count == 1 and pages is None and cursor < len(section),'exactly one WASM memory required')
            flags = section[cursor]; cursor += 1
            require(flags in (0,1),'shared/memory64/custom-page WASM memory unsupported')
            pages,cursor = number(section,cursor)
            if flags:
                maximum,cursor = number(section,cursor)
                require(maximum >= pages,'WASM memory maximum below initial')
        require(cursor == len(section),'trailing WASM metadata bytes')
    require(pages is not None and 1 <= pages <= 65536,'invalid initial WASM memory')
    return dict(imports=imports,initial_memory_pages=pages)


def state_identity(state):
    hasher = hashlib.sha256()
    def sized(raw):
        hasher.update(len(raw).to_bytes(8,'little'))
        hasher.update(raw)
    sized(b'gomad-volume-crash-state/v1')
    sized(state['Volume'].encode())
    paths = []
    require(len(state['Entries']) <= 32768,'native selected full entry bound exceeded')
    for entry in state['Entries']:
        path = entry['Path']
        require(Path(path).is_absolute() and '..' not in Path(path).parts and path not in paths,
                'native selected entry path missing/duplicate/unclean')
        paths.append(path)
        sized(path.encode())
        for field in ('Mode','Kind','ModTime'):
            require(type(entry[field]) is int,'native selected entry numeric field invalid')
            hasher.update((entry[field] % (1<<64)).to_bytes(8,'little'))
        sized(base64.b64decode(entry['Data'],validate=True) if entry['Data'] is not None else b'')
    require(paths == sorted(paths),'native selected entry ordering changed')
    return 'sha256:'+hasher.hexdigest()


def native_state(state, context, volume, maximum):
    require(state['Volume'] == volume and state['PendingSHA256'] == context['PendingSHA256']
            and len(context['Operations']) <= maximum and len(state['SelectedOperations']) <= maximum,
            'native state/context/operation bound mismatch')
    for value in (state['Identity'],state['PendingSHA256']):
        require(isinstance(value,str) and len(value) == 71 and value.startswith('sha256:')
                and all(char in '0123456789abcdef' for char in value[7:]), 'native Go identity encoding')
    require(state_identity(state) == state['Identity'],'native selected full-state identity mismatch')
    operations, previous = {}, 0
    for operation in context['Operations']:
        identity = operation['ID']
        require(type(identity) is int and identity > previous and all(type(dep) is int and 0 < dep < identity
                for dep in operation['Dependencies'] or []),'native pending operation ordering/dependency')
        previous = identity
        operations[identity] = operation
    selected = state['SelectedOperations']
    require(len(set(selected)) == len(selected) and all(identity in operations for identity in selected),
            'duplicate/foreign native selected operation')
    require(all(dep in selected for identity in selected for dep in operations[identity]['Dependencies'] or []
                if dep in operations),'native selection is not dependency closed')


def selected_marker(snapshots, state):
    owners = [row for row in snapshots if row['ID'] == 'sqlite']
    require(len(owners) == 1,'recovered SQLite snapshot owner missing/duplicate')
    expected = [row for row in state['Entries'] if row['Path'] == '/marker']
    actual = [row for row in owners[0]['Persisted'] if row['Path'] == '/marker']
    require(len(expected) <= 1 and len(actual) <= 1 and bool(expected) == bool(actual), 'selected marker presence mismatch')
    if expected:
        entry = expected[0]
        raw = base64.b64decode(entry['Data'],validate=True) if entry['Data'] is not None else b''
        require(entry['Kind'] in (1,2),'selected marker kind not supported')
        wanted = dict(Kind='file' if entry['Kind'] == 1 else 'directory',Mode=entry['Mode'],ModTime=entry['ModTime'],
                      Size=len(raw),DataSHA256='sha256:'+digest(raw) if entry['Kind'] == 1 else '')
        require(all(actual[0][key] == value for key,value in wanted.items()), 'selected marker metadata/content mismatch')


def inspection_policy_projection(policy):
    limits,clock,cwd = policy['Limits'],policy['Clock'],policy['WorkingDirectory']
    model = dict(entropy_key=policy['EntropyKey'],working_directory_sha256=digest(cwd.encode()),
        arguments=dict(bytes=0,count=0,chunks=0),environment=dict(bytes=0,count=0,chunks=0),seed=0,mode=1,
        limit=8<<20,diagnostics=False,epoch_nanos=clock['EpochNanos'],read_step_nanos=clock['ReadStepNanos'],
        now=int(clock['ReadStepNanos']==0),limits=dict(calls=limits['Calls'],transcript_bytes=limits['TranscriptBytes'],pending_events=limits['PendingEvents']))
    entries,roots = {},[]
    def ensure(path,readonly):
        prefix = ''
        for part in path.split('/'):
            if not part: continue
            prefix += '/'+part
            entries.setdefault(prefix,dict(path=list(prefix.encode()),directory=True,readonly=readonly,data=[]))
            require(entries[prefix]['directory'],'captured namespace parent is a file')
    capture = policy.get('CapturedInputs')
    if capture:
        descriptor_raw = base64.b64decode(capture['Descriptor'],validate=True)
        require(digest(descriptor_raw) == capture['Manifest']['SHA256'].removeprefix('sha256:'), 'inspection capture descriptor identity')
        descriptor = decoded(descriptor_raw)
        roots = descriptor['mappings']
        for entry in descriptor['entries']:
            path = entry['path']
            ensure(str(Path(path).parent),True)
            raw = base64.b64decode(capture['Payloads'][entry['payload']],validate=True) if entry['kind']=='file' else b''
            require(entry['kind'] in ('file','directory') and (path not in entries or entry['kind']=='directory' and entries[path]['directory']), 'invalid captured namespace entry')
            if entry['kind']=='file':
                require(len(raw) == int(entry['size']) and 'sha256:'+digest(raw) == entry['sha256'], 'inspection capture raw payload identity')
            entries[path] = dict(path=list(path.encode()),directory=entry['kind']=='directory',readonly=True,data=list(raw))
    writable = policy['WritableDirectories']
    for path in writable:
        ensure(path,False)
        entries[path]['readonly'] = False
    namespace = dict(entries=[entries[path] for path in sorted(entries)],writable_directories=[list(path.encode()) for path in writable],
        preopens=[list(path.encode()) for path in sorted(set(writable+roots))],working_directory=list(cwd.encode()),stdin=[],
        limits=dict(files=limits['Files'],descriptors=limits['Descriptors'],filesystem_bytes=limits['FilesystemBytes'],output_bytes=limits['OutputBytes']),
        epoch_nanos=clock['EpochNanos'])
    environment = dict(value.split('=',1) for value in policy['Environment'])
    network = dict(policy['network_v2'],faults=policy['network_v2'].get('faults',[])) if policy.get('network_v2') else None
    return model,namespace,environment,network


def inspection_configuration(installation, arguments, owner, runtime_sha256):
    environment = dict(installation['admission']['environment'])
    environment.update(PWD=bytes(installation['namespace']['working_directory']).decode(),TZ='UTC',GOMAD_IO_PROFILE='wasm',GOMAD_DEBUG_CANARY='1')
    if owner == 'onebox':
        environment.update(TEMPORAL_ROOT='/workspace',TMPDIR='/tmp',TEMPORAL_TEST_LOG_LEVEL='warn')
    else:
        environment['GOMAD_CHOICE_PROFILE'] = 'gomad-choice-trace/v4'
    model = copy.deepcopy(installation['model'])
    model.pop('seed',None)
    choice = 1<<30 if owner=='onebox' else 8<<20
    model.update(mode=1,limit=choice,diagnostics=False,working_directory_sha256=digest(bytes(installation['namespace']['working_directory'])))
    namespace = copy.deepcopy(installation['namespace'])
    namespace['limits']['output_bytes'] = 1<<20
    nanos = (1800 if owner=='onebox' else 300)*1_000_000_000
    configured = dict(engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1',profile='gomad.wasi-cooperative/v1',
        limits=dict(fuel=installation['fuel'],memory_bytes=installation['memory_bytes'],wall_millis=nanos//1_000_000,
                    output_bytes=1<<20,evidence_bytes=choice+(2<<20)+(128<<20)),arguments=['gomad-target',*arguments],
        wall_nanos=nanos,environment=environment,derive_seed_environment=True,runtime_sha256=runtime_sha256,
        model=model,namespace=namespace,storage=None,network=installation.get('network'))
    if configured['network'] is None:
        del configured['network']
    return configured


def argument(argv, flag):
    require(argv.count(flag) == 1 and argv.index(flag)+1 < len(argv),'missing/duplicate producer argument '+flag)
    return argv[argv.index(flag)+1]


def runtime_selection(configuration, volume, incarnation, state, context):
    owners = [row for row in configuration['incarnations'] if row['handle'] == dict(node=volume,incarnation=incarnation)]
    require(len(owners) == 1,'selected runtime owner missing/duplicate')
    storage = owners[0]['storage']
    raw = canonical({volume:context})
    require(storage['descriptor'].get('crash_selection') == {volume:state['SelectedOperations']}
            and storage['descriptor'].get('crash_context') == dict(bytes=len(raw),chunks=(len(raw)+65535)//65536,sha256=digest(raw))
            and storage['context'] == list(raw),'actual selected runtime state/context is unbound')
    for row in configuration['incarnations']:
        if row is owners[0] or row.get('storage') is None:
            continue
        require(row['storage']['descriptor'].get('crash_selection') is None and row['storage'].get('context') is None,
                'selected context leaked outside original crash owner')


def retry_witness(stdout, network):
    value = decoded(line(stdout,'GOMAD_NETWORK_TEMPORAL_OBSERVATION '),object_pairs_hook=unique)
    ids = value['event_ids']
    require(value['workflow_id'] and value['run_id'] and value['result'] == 'durable timer survived rpc-reset'
            and base64.b64decode(value['history'],validate=True) and ids == list(range(1,len(ids)+1))
            and ids and len(value['types']) == len(ids),'invalid actual named reset history')
    ports = {row['local_port'] for row in value['connections'] if row['opened_nanos'] < 39_999_999_999}
    require(ports,'no SDK stream existed before reset')
    handles,pending = set(),{}
    fault,reset,reconnected = False,False,False
    for operation in network['operations']:
        at,request,result = operation['at'],operation['request'],operation['result']
        observed_fault = operation.get('fault') or {}
        fault |= all(observed_fault.get(key) == item for key,item in dict(name='rpc-reset',action='reset',at_nanos=40_000_000_000).items())
        if request['op'] == 'dial' and result['state'] == 'completed' and not result.get('error') and result.get('handle',0) > 0:
            port = result['local']['port']
            if at < 40_000_000_000 and port in ports:
                handles.add(result['handle'])
            reconnected |= at >= 40_000_000_000 and any(row['opened_nanos'] >= 39_999_999_999 and row['local_port'] == port
                                                      for row in value['connections'])
        if request['op'] in ('read','write'):
            pending[request['request_id']] = request['handle']
        handle = pending.get(request.get('request_id'),0) if request['op'] == 'poll' else request.get('handle',0)
        reset |= at >= 40_000_000_000 and handle in handles and request['op'] in ('read','write','poll') and result.get('error') in (
            'use of closed network connection','connection reset by peer')
    failed,retried = False,False
    for attempt in value['attempts']:
        failed |= attempt['started_nanos'] < 39_999_999_999 <= attempt['ended_nanos'] and attempt['code'] == 'Unavailable'
        retried |= failed and attempt['started_nanos'] >= 39_999_999_999 and attempt['code'] == 'OK'
    require(fault and reset and reconnected and retried,'named reset lacks established stream error, new connection or in-flight retry')


class Retained:
    def __init__(self, config):
        self.config = config
        self.root = Path(config['root'])
        self.commands = {}
        self.adapter = cloned(adapters, dict(command=self.command, write=self.write))

    def contained(self, path):
        path = Path(path)
        require(path.is_absolute() and '..' not in path.parts and path.is_relative_to(self.root),
                'receiving observation outside frozen attempt: ' + str(path))
        return path

    def write(self, path, value):
        self.contained(path)
        require(read(regular(path)) == value, 'retained recomputation differs: ' + str(path))

    def command(self, config, root, name, argv, accepted=(0,), timeout=None):
        require(config == self.config, 'foreign receiving configuration')
        root = self.contained(root)
        row = read(regular(root/(name+'.command.json')))
        process = read(regular(root/(name+'.process.json')))
        require(row['argv'] == argv and row['cwd'] == config['repo'], 'retained command argv/cwd mismatch')
        require(type(row['pid']) is int and row['pid'] > 0 and row['observation'] == 'completed'
                and type(row['exit']) is int and row['exit'] in accepted, 'unexecuted/failed retained command')
        require(type(row['start_ns']) is int and type(row['end_ns']) is int
                and row['end_ns'] >= row['start_ns'] >= config['sent_epoch']*1_000_000_000,
                'retained command outside frozen send interval')
        require(process['active'] is False and type(process['stage_pid']) is int and process['stage_pid'] > 0,
                'incomplete process registration')
        for field in ('argv','pid','start_ns','exit','observation'):
            require(process[field] == row[field], 'retained process/command mismatch: '+field)
        require(row['start_ns'] <= process['end_ns'] <= row['end_ns'], 'retained process end mismatch')
        if timeout is not None:
            require(row['end_ns']-row['start_ns'] <= timeout*1_000_000_000, 'original command bound exceeded')
        require(argv and argv[0] in config['files'], 'retained executable is not sealed')
        stdout = regular(root/(name+'.stdout')).read_bytes()
        stderr = regular(root/(name+'.stderr')).read_bytes()
        if 'stdout_sha256' in row:
            owners = [path/'producer-root.json' for path in (root,*root.parents)
                      if path.is_relative_to(self.root) and (path/'producer-root.json').is_file()]
            if owners:
                owner = read(regular(owners[0]))
                require(process['stage_pid'] == owner['pid'] and row['start_ns'] >= owner['start_ns'],
                        'nested command is not owned by actual producer root/process')
            require(row['stdout'] == str(root/(name+'.stdout')) and row['stderr'] == str(root/(name+'.stderr'))
                    and digest(stdout) == row['stdout_sha256'] and digest(stderr) == row['stderr_sha256']
                    and row['command'] == argv and digest(canonical(argv)) == row['argv_sha256'],
                    'raw nested command streams/argv changed')
        key = str(root/(name+'.command.json'))
        self.commands[key] = row
        if 'report' in argv and '--result' in argv:
            self.result(Path(argv[argv.index('--result')+1]), decoded(stdout))
        if 'inspect' in argv and '--json' in argv:
            inspected = decoded(stdout)['artifact']
            actual = self.artifact(Path(argv[-1]))
            require(all(inspected[field] == actual[field] for field in ('record','files')),
                    'retained public inspect differs from raw manifest')
        return stdout

    def artifact(self, artifact):
        artifact = self.contained(artifact)
        manifest = read(regular(artifact/'manifest.json'))
        require(manifest['schema'] == 'gomad.artifact-manifest/v2'
                and digest(canonical(manifest['record'])) == manifest['record_sha256'], 'raw artifact record identity mismatch')
        modules = []
        for name, entry in manifest['files'].items():
            require(name and not Path(name).is_absolute() and '..' not in Path(name).parts
                    and name not in ('manifest.json','storage.json'), 'unsafe retained payload name')
            if name == 'target/module.wasm':
                require(regular(artifact/name).stat().st_size <= 512<<20,'retained target module exceeds original bound before read')
            raw = regular(artifact/name).read_bytes()
            require(len(raw) == entry['bytes'] and digest(raw) == entry['sha256'], 'raw artifact payload mismatch: '+name)
            require(entry['module'] == (name == 'target/module.wasm'), 'artifact module kind mismatch')
            if entry['module']:
                require(entry['sha256'] == manifest['record']['identity']['target'], 'artifact module identity mismatch')
                modules.append(name)
        require(len(modules) == 1, 'exactly one authenticated native module required')
        for name, stream in manifest['record']['streams'].items():
            require(name in manifest['files'] and all(stream[field] == manifest['files'][name][field]
                    for field in ('bytes','sha256')), 'record stream/payload mismatch: '+name)
        return manifest

    def provenance(self, result):
        provenance = result['provenance']
        inputs, configuration, manifest = provenance['inputs'], provenance['configuration'], result.get('manifest')
        require(provenance['schema'] == 'gomad.rust-wasm-provenance/v1'
                and inputs['schema'] == 'gomad.rust-wasm-build-inputs/v1'
                and digest(canonical(inputs)) == provenance['build_key'], 'qualification build-key preimage mismatch')
        built = dict(configuration,runtime_sha256=None)
        require(inputs['configuration'] == built and inputs['network'] == provenance['network']
                and inputs['scenario'] == provenance['scenario'], 'qualification duplicated input preimages changed')
        compiler = inputs['compiler_path']
        require(isinstance(compiler,str) and Path(compiler).is_absolute()
                and self.config['files'].get(compiler) == inputs['compiler_sha256'], 'qualification compiler path/hash is not frozen')
        require(inputs['module_metadata'] == provenance['module_metadata'], 'qualification metadata preimages disagree')
        choice = provenance['choice_identity']
        profile = {'gomad.wasi-cooperative/v1':'cooperative','gomad.wasi-stock-exploratory/v1':'controlled'}
        require(configuration['profile'] in profile,'qualification profile not admitted')
        require(set(inputs) == set(('arguments','build_flags','compiler','compiler_path','compiler_sha256','configuration','directory','environment','go_version','helper','host','kind','module_metadata','network','runtime','runtime_implementation','scenario','schema','soft_assertion_report_only','source','sources','tags')),
                'qualification build input field inventory changed')
        require(set(provenance)-{'runtime_workload'} == set(('build_key','choice_identity','configuration','inputs','module_metadata','module_sha256','network','scenario','schema')),
                'qualification provenance field inventory changed')
        if configuration['profile'] == 'gomad.wasi-cooperative/v1':
            source = hashlib.sha256(b'gomad-choice-implementation-source-v3')
            for name in ('choice/schema/choicewire.json','choice/schema/choicewire.go.tmpl',
                         'choice/schema/choicewire_runtime.go.tmpl','choice/schema/choicewire_test.go.tmpl',
                         'runtimesource/gomad_semantics.go.txt','choice/trace.go','choice/tape.go'):
                path = Path(self.config['repo'])/'tools/gomad/internal/model'/name
                require(str(path) in self.config['files'],'choice implementation source not frozen')
                raw = regular(path).read_bytes()
                source.update(len(raw).to_bytes(8,'big')); source.update(raw)
            implementation = hashlib.sha256(b'gomad-choice-implementation-v2\0'+source.digest()+bytes.fromhex(provenance['build_key'])).hexdigest()
            require(choice and bytes(choice['ImplementationSHA256']).hex() == implementation == configuration['runtime_sha256']
                    and choice['ToolchainBuildKey'] == provenance['build_key']
                    and bytes(choice['TargetSHA256']).hex() == provenance['module_sha256']
                    and choice['GOOS'] == 'wasip1' and choice['GOARCH'] == 'wasm', 'choice source/toolchain/module/runtime binding changed')
        else:
            require(choice is None and configuration['runtime_sha256'] is None,'controlled profile has cooperative runtime')
        expected = dict(workload_source_sha256=digest(canonical(inputs['sources'])), runtime_sha256=digest(canonical(inputs['compiler'])),
            compiler_sha256=inputs['compiler_sha256'],helper_sha256=inputs['helper'],host_sha256=inputs['host'],
            model_sha256=digest(canonical(configuration.get('model'))), engine_sha256=digest(configuration['engine'].encode()),
            host_config_sha256=digest(canonical(configuration)),module_sha256=provenance['module_sha256'],
            toolchain_sha256=provenance['build_key'],scenario_sha256=digest(bytes(provenance['scenario'])) if provenance['scenario'] is not None else None,
            profile=profile[configuration['profile']],fuel=configuration['limits']['fuel'],memory_bytes=configuration['limits']['memory_bytes'])
        if manifest is not None:
            require(all(manifest[key] == value for key,value in expected.items()),'qualification manifest differs from raw provenance')
        require(inputs['host'] == self.config['identity']['host'] and inputs['helper'] == self.config['identity']['helper'],
                'prepared provenance belongs to foreign release/helper')
        return expected

    def result(self, path, reported):
        path = self.contained(path)
        details = regular(path).stat()
        require(details.st_nlink == 1 and details.st_mode & 0o077 == 0 and details.st_size <= 64<<20,
                'Result must be bounded private regular single-link publication')
        raw = path.read_bytes()
        result = read(path)
        require(raw == canonical(result)+b'\n' and path.name == 'result-'+digest(raw)+'.json',
                'Result canonical bytes/content address mismatch')
        require(reported['result'] == result, 'public report differs from raw Result')
        pairs = list(path.parent.glob(path.name+'.publication-*.json'))
        require(len(pairs) == 1, 'exactly one publication receipt required')
        details = regular(pairs[0]).stat()
        require(details.st_nlink == 1 and details.st_mode & 0o077 == 0 and details.st_size <= 8<<10,
                'receipt must be bounded private regular single-link publication')
        receipt_raw = pairs[0].read_bytes()
        receipt = read(pairs[0])
        require(receipt_raw == canonical(receipt)+b'\n'
                and pairs[0].name == path.name+'.publication-'+digest(receipt_raw)+'.json', 'receipt canonical/content identity mismatch')
        require(receipt == reported['receipt'] and receipt['schema'] == 'gomad.rust-qualification-receipt/v1'
                and receipt['result_sha256'] == digest(raw)
                and type(receipt['operator_process_id']) is int and receipt['operator_process_id'] > 0
                and receipt['operator_process_id'] == result['observations']['operator_process_id'], 'receipt Result/operator binding mismatch')
        require(receipt['linked_after'] is not None and receipt['operator_peak_rss_bytes'] is not None,
                'publication measurement absent')
        manifest = result['manifest']
        source_manifest = regular(self.config['source_manifest']).read_bytes()
        require(result.get('integrated_source_manifest') == source_manifest.decode()
                and digest(source_manifest) == self.config['source_sha256'], 'Result integrated source preimage differs from frozen candidate')
        require(manifest['provisional'] is False and manifest['seed'] == 7 and manifest['repeat'] == 2, 'Result is not original final seed/repeat')
        require(result['schema'] == 'gomad.rust-qualification-result/v1'
                and manifest['integrated_source_sha256'] == self.config['source_sha256'], 'foreign Result source/schema')
        for key, value in self.config['identity'].items():
            require(manifest[key+'_sha256'] == value, 'foreign Result identity: '+key)
        self.provenance(result)
        receiving_cost.validate(result)
        observations, after = result['observations'], receipt['linked_after']
        executed = observations['executed']['linked']
        require(all(type(after[key]) is int and after[key] >= executed[key] for key in ('user_nanos','system_nanos','peak_rss_bytes')),
                'publication raw counters regressed')
        peak = receipt['operator_peak_rss_bytes']
        require(type(peak) is int and 0 < observations['operator_peak_rss_bytes'] <= peak <= 2<<30
                and after['peak_rss_bytes'] <= max(peak,executed['peak_rss_bytes']), 'publication peak attribution mismatch')
        require(reported['publication'] == dict(cpu_nanos=after['user_nanos']+after['system_nanos']-executed['user_nanos']-executed['system_nanos'],
                peak_rss_bytes=peak,status='within'), 'publication tail recomputation mismatch')
        executions = result['report']['executions']
        require(len(executions) == 2 and [row['iteration'] for row in executions] == [1,2], 'exact two Result iterations required')
        report = result['report']
        require(report['schema'] == 'gomad.rust-qualification/v1' and report['seed'] == 7 and report['repeat'] == 2
                and report['command'] and report['failure'] is None, 'incomplete actual qualification report')
        deterministic = len({row['record_sha256'] for row in executions}) == 1
        success = all(row['record']['outcome']['class'] == 'success' for row in executions)
        require(report['deterministic'] == deterministic and deterministic and report['target_success'] == success
                and report['qualified'] == success and report['record_sha256'] == executions[0]['record_sha256']
                and report['first_divergence'] is None, 'Result public verdict differs from raw canonical outcomes')
        paths = set()
        for execution in executions:
            artifact = execution['artifact_path']
            require(artifact not in paths, 'duplicate qualification artifact')
            paths.add(artifact)
            actual = self.artifact(Path(artifact))
            require(module_metadata(regular(Path(artifact)/'target/module.wasm').read_bytes()) == result['provenance']['module_metadata'],
                    'qualification metadata differs from raw module')
            self.journal(Path(execution['campaign_path']), execution['record_sha256'])
            require(actual['record'] == execution['record'] and actual['record_sha256'] == execution['record_sha256'],
                    'Result execution differs from authenticated raw artifact')
            record = actual['record']
            implementation = digest(canonical(self.config['identity']))
            require(record['identity']['implementation'] == implementation
                    and record['identity']['platform'] == digest((manifest['host_os']+'/'+manifest['host_arch']).encode())
                    and record['profile'] == result['provenance']['configuration']['profile'], 'Result current role/platform/profile differs from actual record')
            require(actual['record']['identity']['target'] == manifest['module_sha256']
                    and actual['record']['identity']['toolchain'] == manifest['toolchain_sha256']
                    and actual['record']['seed'] == manifest['seed'], 'qualification module/toolchain/seed differs from execution')
            configured_limits = dict(result['provenance']['configuration']['limits'])
            cluster = manifest['scenario_sha256'] is not None
            if cluster:
                scenario = decoded(bytes(result['provenance']['scenario']))
                configured_limits['evidence_bytes'] = min(configured_limits['evidence_bytes'],scenario['spec']['limits']['scenario_evidence_bytes'])
            else:
                require(record.get('arguments') == result['provenance']['configuration'].get('arguments')
                        and record.get('environment') == result['provenance']['configuration'].get('environment')
                        and record['identity']['configuration'] == manifest['host_config_sha256'], 'Result standalone input binding changed')
            require(record['limits'] == configured_limits,'Result record limits differ from actual prepared request')
            trace_names = ('transcript','choice','diagnostic','storage','network')
            trace = []
            expected_handles = set()
            for name,stream in record['streams'].items():
                if name.startswith('evidence/cluster/'):
                    parts = name.removeprefix('evidence/cluster/').split('/',2)
                    require(len(parts) == 3,'malformed cluster stream owner')
                    expected_handles.add(parts[0]+'/'+parts[1])
                    if parts[2] != 'network' and parts[2] in trace_names:
                        trace.append(stream['bytes'])
                elif name.startswith('evidence/') and name.removeprefix('evidence/') in trace_names:
                    trace.append(stream['bytes'])
            for phase in ('execute','replay'):
                samples = [sample for sample in observations['samples'] if sample['phase'] == phase and sample['iteration'] == execution['iteration']]
                require(samples and receiving_cost.total(sample['trace_bytes'] for sample in samples) == receiving_cost.total(trace)
                        and all(sample['guest_memory_limit_bytes'] == manifest['memory_bytes'] and (sample.get('incarnation') is not None) == cluster for sample in samples),
                        'Result raw trace/memory/store samples differ from actual execution')
                if cluster:
                    require(expected_handles and {sample['incarnation'] for sample in samples} == expected_handles, 'Result raw store samples omit/borrow cluster incarnation')
            replay = execution['replay']
            require(replay and replay['attempted'] is True and replay['matched'] is True
                    and replay['divergence'] is None and replay['choice_replay'] == ('exact' if manifest['profile'] == 'cooperative' else 'none'), 'incomplete full-tape Result replay')
        return result

    def runtime(self):
        paths, pids = set(), set()
        for index in range(32):
            label = 'runtime-'+str(index)
            root = self.root/'stages'/label
            self.adapter.runtime(self.config, root, label, self.config['stages'][label])
            report = read(root/'runtime.json')
            require(not paths.intersection(report['artifacts']), 'runtime shards borrowed artifacts')
            paths.update(report['artifacts'])
            for artifact in report['artifacts']:
                campaign = next((path for path in Path(artifact).parents if path.parent.name == 'campaigns'),None)
                require(campaign is not None,'runtime campaign root missing')
                self.journal(campaign,read(Path(artifact)/'manifest.json')['record_sha256'])
            rows = [read(root/(f'record-{number:02}.command.json')) for number in range(25)]
            require(len({row['pid'] for row in rows}) == 25 and not pids.intersection(row['pid'] for row in rows),
                    'runtime repeated producer PID')
            pids.update(row['pid'] for row in rows)
            replay = read(root/'forced-replay.command.json')
            require(replay['end_ns']-rows[0]['start_ns'] <= 600_000_000_000, 'runtime original ten-minute shard exceeded')
        self.adapter.runtime_aggregate(self.config,self.root/'stages/runtime-aggregate','runtime-aggregate')

    def mandatory(self):
        for label in adapters.CELLS:
            self.adapter.qualify_cell(self.config,self.root/'stages'/label,label,self.config['stages'][label])
        for label in ('workload-corpus','workload-corpus-after-relocation'):
            self.adapter.corpus(self.config,self.root/'stages'/label,label)

    def native_catalogue(self, result, native):
        volume, handle = native['volume'], native['handle']
        for execution in result['report']['executions']:
            artifact = Path(execution['artifact_path'])
            prefix = artifact/f"evidence/cluster/{handle['node']}/{handle['incarnation']}"
            raw = regular(prefix/'storage_state').read_bytes()
            log = regular(prefix/'storage').read_bytes()
            sections = storage_sections(raw)
            require(digest(raw) == native['storage_state_sha256'] and len(sections[5]) == 40
                    and struct.unpack_from('<Q',sections[5])[0] == len(log)
                    and sections[5][8:] == hashlib.sha256(log).digest(), 'native actual storage/log binding mismatch')
            require(decoded(sections[3],object_pairs_hook=unique) == {volume:native['enumeration']}
                    and decoded(sections[6],object_pairs_hook=unique) == {volume:native['context']},
                    'native Result catalogue/context differs from actual GMSTORE sections')
            retained = read(regular(artifact/'evidence/cluster-replay.json'))
            configuration = retained['configuration']
            require(digest(canonical(configuration)) == execution['record']['identity']['configuration'],
                    'native actual cluster configuration binding changed')
            owners = [row for row in configuration['incarnations'] if row['handle'] == handle]
            require(len(owners) == 1,'native actual storage owner missing/duplicate')
            expected = dict(States=32,Operations=8,Depth=8,Bytes=4<<20,Work=1<<16) if volume=='sqlite' else dict(
                States=128,Operations=256,Depth=256,Bytes=256<<20,Work=1<<20)
            require(owners[0]['storage']['descriptor']['crash_limits'] == expected, 'native original crash enumeration bounds changed')

    def sqlite_oracle(self, result):
        for execution in result['report']['executions']:
            artifact = Path(execution['artifact_path'])
            ack = line((artifact/'evidence/cluster/sqlite/2/stdout').read_bytes(),'ACK ')
            recovered = (artifact/'evidence/cluster/sqlite/3/stdout').read_bytes()
            integrity, raw_history = line(recovered,'INTEGRITY '), line(recovered,'HISTORY ')
            require(ack.isdecimal() and str(int(ack)) == ack and int(ack) > 0,'invalid SQLite acknowledgment')
            rows = [row.split(':') for row in raw_history.split(',')]
            require(all(len(row) == 2 and row[0].isdecimal() and str(int(row[0])) == row[0]
                        and int(row[0]) > 0 and row[1] for row in rows), 'invalid SQLite raw history')
            require(len(ack.encode())+len(b'committed')+len(recovered) <= 1<<20, 'SQLite original oracle bound exceeded')
            checks = {'acknowledged-durable-recovery.events': sorted(row[0] for row in rows) == ['1','2'],
                      'acknowledged-durable-recovery.integrity': integrity == 'ok',
                      'acknowledged-durable-recovery.acknowledged': ack == '2' and all(row[1] == 'committed' for row in rows)}
            require(read(regular(artifact/'evidence/workload-oracles.json')) == dict(schema='gomad.rust-workload-oracles/v1',checks=checks)
                    and all(checks.values()), 'actual SQLite durable acknowledgment/history/integrity mismatch')

    def sqlite_primary(self):
        label = 'workload-sqlite-cooperative-success'
        reference = read(regular(self.root/'stages'/label/'result.json'))
        require(sha(regular(reference['path'])) == reference['sha256'], 'SQLite primary Result changed')
        primary = self.adapter.result_report(self.config,self.root/'stages/sqlite-catalogue','primary-report',reference['path'])
        self.sqlite_oracle(primary)
        native = sqlite.catalogue(primary)
        self.native_catalogue(primary,native)
        for state in native['enumeration']['States']:
            native_state(state,native['context'],'sqlite',8)
        return reference, primary, native

    def selected(self, index):
        reference, primary, native = self.sqlite_primary()
        root = self.root/'stages'/f'sqlite-selection-{index}'
        row = read(regular(root/'selection.json'))
        require(row['index'] == index and row['primary_sha256'] == reference['sha256']
                and sha(regular(row['path'])) == row['sha256'], 'stale SQLite selection')
        expected = dict(primary_result_sha256=reference['sha256'],state=native['enumeration']['States'][index],context=native['context'])
        result = self.adapter.result_report(self.config,root,'selected-report',row['path'])
        spec = read(regular(root/'selection-spec.json'))
        template = read(self.config['stages']['workload-sqlite-cooperative-success']['spec'])
        require(spec == sqlite.selection_spec(template,reference['path'],reference['sha256'],expected['state'],expected['context'],str(root/'selected')),
                'SQLite selection changed original request/bounds')
        installation = self.config['installations'][self.config['stages'][f'sqlite-selection-{index}']['installation']]
        response = decoded(self.command(self.config,root,'qualify',adapters.public(self.config,installation)+['qualify','--spec',str(root/'selection-spec.json')]))
        pid = read(root/'qualify.command.json')['pid']
        require(response['result_path'] == row['path'] and row['selection'] == result['native_crash_selection'] == expected
                and row['pid'] == result['observations']['operator_process_id'] == pid and pid != reference['pid'], 'SQLite selected producer/state binding')
        sqlite.matched_inputs(primary,result,spec['RuntimeWorkload'])
        require(sqlite.catalogue(result) == native, 'selected native catalogue changed')
        self.sqlite_oracle(result)
        for execution in result['report']['executions']:
            raw = regular(Path(execution['artifact_path'])/'evidence/cluster/sqlite/3/storage_state').read_bytes()
            selected_marker(decoded(storage_sections(raw)[1]),expected['state'])
        return row, result

    def catalogue(self):
        reference, primary, native = self.sqlite_primary()
        root = self.root/'stages/sqlite-catalogue'
        oracle = self.config['stages']['sqlite-catalogue'].get('oracle')
        if not oracle:
            raise Unexecuted('independent actual native catalogue oracle not bound')
        artifact = primary['report']['executions'][0]['artifact_path']
        inspected = decoded(self.command(self.config,root,'primary-inspect',adapters.public(self.config)+['inspect','--json',artifact]))
        self.write(root/'oracle-input.json',sqlite.native_input(artifact,inspected,native))
        actual = decoded(self.command(self.config,root,'native-Go-oracle',oracle+['--input',str(root/'oracle-input.json')]))
        require(actual == dict(enumeration=native['enumeration'],context=native['context']), 'independent native oracle mismatch')
        hashes, pids, cpu, tail = set(), {reference['pid']}, 0, 0
        for index in range(4):
            row, result = self.selected(index)
            require(row['sha256'] not in hashes and row['pid'] not in pids, 'duplicate SQLite selected producer')
            hashes.add(row['sha256']); pids.add(row['pid'])
            cpu += result['cost']['cpu_nanos']; tail += result['_driver_publication']['cpu_nanos']
        self.write(root/'catalogue.json',dict(primary=reference['sha256'],selected=sorted(hashes),producer_pids=sorted(pids),
            selected_execution_cpu_nanos=cpu,selected_terminal_cpu_nanos=tail,selected_total_cpu_nanos=cpu+tail,primary_cpu_counted=False))

    def journal(self, campaign, record_sha256):
        campaign = self.contained(campaign)
        storage = read(regular(campaign/'storage.json'))
        plan_raw = regular(campaign/'plan.json').read_bytes()
        require(digest(plan_raw) == storage['plan_sha256'], 'campaign plan content identity mismatch')
        raw = regular(campaign/'journal.jsonl').read_bytes()
        require(raw and raw.endswith(b'\n') and len(raw) <= storage['max_bytes'], 'partial/oversized campaign journal')
        previous, found = storage['plan_sha256'], 0
        for ordinal, line in enumerate(raw.splitlines()):
            row = decoded(line, object_pairs_hook=unique)
            require(canonical(row) == line and row['ordinal'] == ordinal and row['previous_sha256'] == previous,
                    'campaign journal canonical/ordinal/link mismatch')
            require(row['sha256'] == digest(canonical({key:row[key] for key in ('ordinal','previous_sha256','event')})),
                    'campaign journal event digest mismatch')
            event = row['event']
            executions = [event['execution']] if event['kind'] == 'receipt' else event.get('executions',[]) if event['kind'] in ('choice','simulation') else []
            for execution in executions:
                if digest(canonical(execution['record'])) == record_sha256:
                    found += 1
            previous = row['sha256']
        require(found == 1 and row['event']['kind'] in ('complete','choice','simulation'),
                'campaign missing unique execution receipt or complete round')

    def nested(self, root, name, argv, accepted=(0,)):
        row = read(regular(self.contained(root)/(name+'.command.json')))
        require('stdout_sha256' in row and 'environment' in row and 'environment_sha256' in row,
                'historical producer lacks authenticated raw observations')
        return self.command(self.config, root, name, argv, accepted=accepted)

    def owner(self, root, expected_argv=None):
        root = self.contained(root)
        owner = read(regular(root/'producer-root.json'))
        require(owner['schema'] == 'gomad.rust-producer-root/v1' and owner['root'] == str(root)
                and owner['cwd'] == self.config['repo'] and type(owner['pid']) is int and owner['pid'] > 0
                and owner['start_ns'] >= self.config['sent_epoch']*1_000_000_000, 'foreign producer root/process')
        if expected_argv is not None:
            require(owner['argv'] == expected_argv[1:], 'producer root argv binding mismatch')
        return owner

    def endpoint(self, label):
        entry = self.config.get('stages',{}).get(label,{})
        if entry.get('adapter') != 'checked-command' or not entry.get('argv'):
            raise Unexecuted(label+': actual receiving producer argv/root is not bound')
        argv = entry['argv']
        require(argv.count('--root') == 1, 'actual producer requires exactly one owned root')
        root = self.contained(Path(argv[argv.index('--root')+1]))
        outer = self.root/'stages'/label
        self.command(self.config,outer,'producer',argv,timeout=entry['timeout_s'])
        owner = self.owner(root,argv)
        observed = read(outer/'producer.command.json')
        require(owner['pid'] == observed['pid'] and observed['start_ns'] <= owner['start_ns'] <= observed['end_ns'],
                'outer producer PID/time/root mismatch')
        return root

    def sources(self, root, workload, staged_source):
        repo = Path(self.config['repo'])
        base = repo/workloads.WASM/'testdata'
        names = {'replication':['temporal_storage/recovery_test.go','temporal_broker/replication_test.go'],
                 'recovery':['temporal_storage/recovery_test.go'],
                 'retry':['network_streams/'+path.name for path in sorted((base/'network_streams').glob('*.go'))]}[workload]
        expected = {str(base/name):sha(regular(base/name)) for name in names}
        require(expected and read(regular(root/'unchanged-source-inputs.json')) == expected
                and all(self.config['files'].get(path) == value for path,value in expected.items()),
                'Temporal original source inventory differs from frozen source')
        source_bytes = b''.join(regular(base/name).read_bytes() for name in names)
        # The retry producer uses its fixed entry fixture for the staging-directory key.
        if workload == 'retry':
            source_bytes = regular(base/'network_streams/temporal_retry_test.go').read_bytes()
        stage = repo/'.tmp'/('fn232-temporal-'+workload+'-'+digest(source_bytes)[:16])
        require(staged_source == './'+str(stage.relative_to(repo)),'Temporal changed original staged source selection')
        require({path.name for path in stage.iterdir()} == {Path(name).name for name in names}, 'Temporal staged source inventory changed')
        for name in names:
            require(regular(stage/Path(name).name).read_bytes() == regular(base/name).read_bytes(), 'Temporal staged source changed original bytes')

    def producer(self, root, workload, mode='success'):
        owner = self.owner(root)
        receipt = read(regular(root/'producer.json'))
        require(receipt['schema'] == 'gomad.rust-mandatory-workload-producer/v1'
                and receipt['workload'] == workload and receipt['mode'] == mode, 'wrong actual producer cell')
        spec = read(regular(root/'qualification-spec.json'))
        target = spec['Campaign']['Target']
        source_manifest = self.config['source_manifest']
        expected = (workloads.support_spec if workload in workloads.SUPPORT_ROWS else workloads.qualification_spec)(
            Path(self.config['repo']),root/'cell',workload,mode=='canary',target['Source'],Path(source_manifest))
        require(spec == expected, 'actual producer changed original request, target, seed, repetition or bounds')
        policy = read(regular(root/'policy.json'))
        capture = policy.get('CapturedInputs')
        expected_policy = (workloads.support_policy if workload in workloads.SUPPORT_ROWS else workloads.policy)(workload,capture)
        require(policy == expected_policy, 'actual producer changed original model policy')
        # Captured inputs are independently pinned in the frozen input set.
        if capture:
            require(owner['argv'].count('--captured-config') == 1,'captured schema path not bound to producer')
            captured_path = owner['argv'][owner['argv'].index('--captured-config')+1]
            require(captured_path in self.config['files'],'captured schema input not frozen')
            captured_value = read(captured_path)
            require(captured_value.get('Config',captured_value).get('CapturedInputs') == capture,'actual captured schema differs from frozen input')
        if workload in ('replication','recovery','retry'):
            self.sources(root,workload,target['Source'])
        installation = read(regular(root/'installation.json'))
        require(installation['fuel'] == (4_000_000_000 if workload=='sqlite' else 1<<50)
                and installation['memory_bytes'] == (256<<20 if workload=='sqlite' else 4<<30)
                and installation['profile'] == 'gomad.wasi-cooperative/v1'
                and installation['admission']['host_executable_sha256'] == self.config['binary_sha256'],
                'actual producer installation role/bounds mismatch')
        for key,value in self.config['identity'].items():
            require(installation[key+'_sha256'] == value,'actual producer installation identity mismatch')
        installed_command = read(regular(root/'install.command.json'))['argv']
        require(installed_command[:2] == [self.config['binary'],'install'], 'foreign installation command')
        for flag,value in (('--output',str(root/'installation.json')),('--profile','gomad.wasi-cooperative/v1'),
                           ('--config',str(root/'policy.json')),('--fuel',str(installation['fuel'])),
                           ('--memory-bytes',str(installation['memory_bytes'])),('--compiler',installation['compiler_path']),
                           ('--runtime-root',installation['runtime_root']),('--cache',installation['cache_root'])):
            require(installed_command.count(flag) == 1 and installed_command[installed_command.index(flag)+1] == value,
                    'producer install argv mismatch '+flag)
        require(len(installed_command) == 18 and installation['compiler_path'] in self.config['files']
                and self.config['files'][installation['compiler_path']] == installation['compiler_sha256'], 'compiler not frozen or extra install controls')
        self.nested(root,'install',installed_command)
        qualify_argv = [self.config['binary'],'--installation',str(root/'installation.json'),'qualify','--spec',str(root/'qualification-spec.json')]
        response = decoded(self.command(self.config,root,'qualify',qualify_argv,accepted=(1,) if mode=='canary' else (0,)))
        path = Path(receipt['result_path'])
        require(response['result_path'] == str(path) and sha(regular(path)) == receipt['result_sha256'], 'actual Result path/hash mismatch')
        reported = decoded(self.nested(root,'report',[self.config['binary'],'report','--result',str(path)]))
        require(receipt['report'] == reported,'producer report changed')
        result = self.adapter.result_report(self.config,root,'report',str(path))
        require(result['observations']['operator_process_id'] == read(root/'qualify.command.json')['pid']
                and read(root/'qualify.process.json')['stage_pid'] == owner['pid'], 'actual producer PID lineage mismatch')
        for key,value in spec['Manifest'].items():
            if key in ('ID','Workload','Cell','Seed','Repeat'):
                require(result['manifest'][key.lower()] == value, 'producer published wrong manifest '+key)
        for key in ('compiler_sha256','helper_sha256','host_sha256'):
            require(result['manifest'][key] == installation[key], 'actual Result installation binding mismatch '+key)
        require(result['manifest']['profile'] == 'cooperative','actual producer profile mismatch')
        return result

    def smoke(self):
        for mode in ('success','canary'):
            label = 'workload-onebox-cooperative-'+mode
            self.adapter.qualify_cell(self.config,self.root/'stages'/label,label,self.config['stages'][label])

    def replication(self):
        results = []
        for mode in ('success','canary'):
            label = 'workload-multi-instance-cooperative-'+mode
            root = self.root/'stages'/label
            self.adapter.qualify_cell(self.config,root,label,self.config['stages'][label])
            reference = read(regular(root/'result.json'))
            result = self.adapter.result_report(self.config,root,'report',reference['path'])
            spec = read(self.config['stages'][label]['spec'])
            scenario = decoded(base64.b64decode(spec['Campaign']['Target']['Scenario'],validate=True))
            require(scenario == workloads.temporal_replication_scenario(mode=='canary'), 'changed actual replication scenario/bounds')
            for execution in result['report']['executions']:
                artifact = Path(execution['artifact_path'])
                checks = {}
                histories = []
                for role in ('active','standby'):
                    expected = history((artifact/f'evidence/cluster/{role}/1/stdout').read_bytes())
                    actual = history((artifact/f'evidence/cluster/{role}/2/stdout').read_bytes())
                    checks[role+'-durable-recovery.history'] = expected['history'] == actual['history']
                    checks[role+'-durable-recovery.events'] = mode=='success' and sorted(actual['event_ids']) == [1,2,3,4,5]
                    checks[role+'-durable-recovery.identity'] = expected['cluster_id'] == actual['cluster_id']
                    histories.append(actual['history'])
                checks['independent-replication-convergence'] = histories[0] == histories[1]
                oracle = read(regular(artifact/'evidence/workload-oracles.json'))
                require(oracle == dict(schema='gomad.rust-workload-oracles/v1',checks=checks)
                        and all(checks.values()) == (mode=='success'), 'replication raw histories/oracle mismatch')
            results.append(result['observations']['operator_process_id'])
        require(len(set(results)) == 2,'replication cells borrowed producer PID')

    def temporal(self, label):
        root = self.endpoint(label)
        if label == 'network-streams':
            result = self.producer(root,'retry')
            for execution in result['report']['executions']:
                artifact = Path(execution['artifact_path'])
                retry_witness((artifact/'output/stdout').read_bytes(),read(artifact/'evidence/network'))
        elif label == 'storage':
            self.recovery(root)
        elif label == 'temporal-inspection':
            outer_argv = self.config['stages'][label]['argv']
            require(argument(outer_argv,'--owner') == 'all','inspection aggregate requires exact three original owners')
            script = str(Path(__file__).resolve().parents[1]/'workload_tools/temporal-inspection.py')
            require(len(outer_argv) > 1 and outer_argv[1] == script and script in self.config['files'], 'inspection aggregate uses foreign producer script')
            for owner in workloads.INSPECTION:
                child_argv = [outer_argv[0],script,'--owner',owner]
                for flag in ('--binary','--repo','--compiler','--runtime-root','--cache'):
                    child_argv += [flag,argument(outer_argv,flag)]
                child_argv += ['--root',str(root/owner)]
                if '--captured-config' in outer_argv:
                    child_argv += ['--captured-config',argument(outer_argv,'--captured-config')]
                self.nested(root,owner,child_argv)
                child = self.owner(root/owner,child_argv)
                require(child['pid'] == read(root/(owner+'.command.json'))['pid'],'inspection child root/PID mismatch')
                self.inspection(root/owner,owner)
        else:
            raise Unexecuted('unsupported receiving endpoint: '+label)

    def recovery(self, root):
        owner = self.owner(root)
        argv = owner['argv']
        require(argv.count('--primary') == 1,'actual recovery primary root missing')
        primary_root = self.contained(Path(argv[argv.index('--primary')+1]))
        primary = self.producer(primary_root,'recovery')
        receipt = read(regular(root/'recovery-receipt.json'))
        primary_path = read(primary_root/'producer.json')['result_path']
        primary_report_argv = [self.config['binary'],'report','--result',primary_path]
        self.nested(root,'primary-report',primary_report_argv)
        self.nested(root,'primary-final-report',primary_report_argv)
        catalogues = primary['native_crash_catalogues']
        require(len(catalogues) == 1 and catalogues[0]['volume'] == 'temporal'
                and catalogues[0]['handle'] == dict(node='temporal',incarnation=1), 'actual primary Temporal catalogue owner')
        native = catalogues[0]
        self.native_catalogue(primary,native)
        enumeration, context = native['enumeration'], native['context']
        require(enumeration['Complete'] is True and not enumeration['Capacity'] and enumeration['States']
                and len(enumeration['States']) <= 128 and len(context['Operations']) <= 256,
                'incomplete actual Temporal catalogue')
        require(receipt['catalogue'] == enumeration and receipt['context'] == context
                and len(receipt['results']) == len(enumeration['States']), 'all original Temporal states required')
        primary_artifact = Path(primary['report']['executions'][0]['artifact_path'])
        primary_record = read(primary_artifact/'manifest.json')['record']
        primary_state = (primary_artifact/'evidence/cluster/temporal/1/storage_state').read_bytes()
        require(digest(primary_state) == native['storage_state_sha256'], 'Temporal native storage state binding')
        expected_prefix = (primary_artifact/'evidence/cluster/temporal/1/choice').read_bytes()
        expected_history = history((primary_artifact/'evidence/cluster/temporal/1/stdout').read_bytes(),sequential=True)
        seen, pids = set(), {primary['observations']['operator_process_id']}
        for index, (state,row) in enumerate(zip(enumeration['States'],receipt['results'])):
            native_state(state,context,'temporal',256)
            require(row['index'] == index and row['state'] == state['Identity'] and row['state'] not in seen,
                    'duplicate/omitted/reordered actual Temporal state')
            seen.add(row['state'])
            cell = root/str(index)
            spec = read(regular(cell/'spec.json'))
            expected = read(primary_root/'qualification-spec.json')
            expected['ArtifactRoot'] = str(cell/'cell')
            expected['Campaign']['Artifacts'] = str(cell/'cell/artifacts')
            expected['RuntimeWorkload'] = dict(kind='temporal-committed-workflow-crash-reopen',primary=dict(
                artifact_path=str(primary_artifact),record_sha256=digest(canonical(primary_record)),
                prefix_sha256=digest(expected_prefix),storage_state_sha256=digest(primary_state),state=state,context=context))
            require(spec == expected,'Temporal selected request changed base scenario or primary context')
            qualify_argv = [self.config['binary'],'--installation',str(primary_root/'installation.json'),
                            'qualify','--spec',str(cell/'spec.json')]
            response = decoded(self.command(self.config,cell,'qualify',qualify_argv))
            require(response['result_path'] == row['result_path'],'Temporal selected qualify response borrowed Result')
            result = self.adapter.result_report(self.config,cell,'report',row['result_path'])
            sqlite.matched_inputs(primary,result,spec['RuntimeWorkload'])
            pid = result['observations']['operator_process_id']
            require(pid not in pids and pid == read(cell/'qualify.command.json')['pid']
                    and sha(regular(row['result_path'])) == row['result_sha256'], 'Temporal selected Result/PID binding')
            pids.add(pid)
            for execution in result['report']['executions']:
                artifact = Path(execution['artifact_path'])
                for stream in ('choice','stdout','storage'):
                    require((artifact/('evidence/cluster/temporal/1/'+stream)).read_bytes() ==
                            (primary_artifact/('evidence/cluster/temporal/1/'+stream)).read_bytes(), 'Temporal primary interrupted prefix changed')
                self.recovery_configuration(primary_artifact,artifact,spec['RuntimeWorkload'],expected_history)
                recovered = history((artifact/'evidence/cluster/temporal/2/stdout').read_bytes(),sequential=True)
                require(recovered == expected_history,'Temporal selected history did not recover exactly')
        for execution in primary['report']['executions']:
            artifact = Path(execution['artifact_path'])
            image = (artifact/'evidence/cluster/temporal/1/recovery_image_primary').read_bytes()
            require(storage_sections((artifact/'evidence/cluster/temporal/1/storage_state').read_bytes())[4] == digest(image).encode(),
                    'Temporal actual image/native state mismatch')
            expected = history((artifact/'evidence/cluster/temporal/1/stdout').read_bytes(),sequential=True)
            for incarnation in (2,3):
                require(history((artifact/f'evidence/cluster/temporal/{incarnation}/stdout').read_bytes(),sequential=True) == expected,
                        'Temporal primary fresh reopen history mismatch')
                require(read(artifact/f'evidence/cluster/temporal/{incarnation}/recovery_image_input') ==
                        dict(source_handle=dict(node='temporal',incarnation=1),image_sha256=digest(image),image_bytes=len(image)),
                        'Temporal primary did not use identical image twice')

    def recovery_configuration(self, primary_artifact, artifact, request, expected_history):
        primary_record = read(regular(primary_artifact/'manifest.json'))['record']
        configuration = read(regular(artifact/'evidence/cluster-replay.json'))['configuration']
        primary_configuration = read(regular(primary_artifact/'evidence/cluster-replay.json'))['configuration']
        require(digest(canonical(configuration)) == read(artifact/'manifest.json')['record']['identity']['configuration'],
                'Temporal selected actual configuration identity mismatch')
        forcing = dict(primary_record_sha256=request['primary']['record_sha256'],prefix_sha256=request['primary']['prefix_sha256'],
                       storage_state_sha256=request['primary']['storage_state_sha256'],state=request['primary']['state'],
                       context=request['primary']['context'],expected=expected_history)
        prefix = primary_artifact/'evidence/cluster/temporal/1'
        for name,field in [('storage','storage_sha256'),('stdout','stdout_sha256'),('recovery_image_primary','primary_image_sha256')]:
            forcing[field] = digest(regular(prefix/name).read_bytes())
        require(configuration['workload'] == dict(kind='temporal-committed-workflow-crash-reopen',forcing=forcing),
                'Temporal selected actual workload does not bind original primary/state/context')
        runtime_selection(configuration,'temporal',1,request['primary']['state'],request['primary']['context'])
        expected = copy.deepcopy(primary_configuration)
        expected['incarnations'] = [row for row in expected['incarnations'] if row['handle']['incarnation'] != 3]
        expected['steps'] = [row for row in expected['steps'] if row['handle']['incarnation'] != 3]
        expected['workload'] = configuration['workload']
        owner = next(row for row in expected['incarnations'] if row['handle'] == dict(node='temporal',incarnation=1))
        raw = canonical(dict(temporal=request['primary']['context']))
        owner['storage']['descriptor']['crash_selection'] = dict(temporal=request['primary']['state']['SelectedOperations'])
        owner['storage']['descriptor']['crash_context'] = dict(bytes=len(raw),chunks=(len(raw)+65535)//65536,sha256=digest(raw))
        owner['storage']['context'] = list(raw)
        require(configuration == expected,'Temporal selected configuration changed unapproved prepared inputs')
        capsule = artifact/'evidence/primary-crash'
        require(regular(capsule/'record.json').read_bytes() == canonical(primary_record), 'Temporal primary record capsule changed')
        require(regular(capsule/'cluster-replay.json').read_bytes() == regular(primary_artifact/'evidence/cluster-replay.json').read_bytes(),
                'Temporal primary configuration capsule changed')
        for name in ('choice','storage_state','storage','stdout','recovery_image_primary'):
            require(regular(capsule/name).read_bytes() == regular(prefix/name).read_bytes(),'Temporal primary capsule payload changed: '+name)

    def installed(self, installation, policy, fuel, memory):
        require(installation['schema'] == 'gomad.rust-wasm-installation/v1'
                and installation['profile'] == 'gomad.wasi-cooperative/v1'
                and installation['fuel'] == fuel and installation['memory_bytes'] == memory,
                'inspection actual installation profile/bounds mismatch')
        require(all(installation[key+'_sha256'] == value for key,value in self.config['identity'].items()),
                'inspection actual installation host/helper mismatch')
        model,namespace,environment,network = inspection_policy_projection(policy)
        require(installation['model'] == model and installation['namespace'] == namespace
                and installation.get('network') == network and installation.get('storage') is None
                and installation.get('build_tags',[]) == [], 'inspection installed original policy projection changed')
        admission = installation['admission']
        require(admission['environment'] == environment and admission['arguments'] == (policy.get('Args') or []),
                'inspection installed argument/environment policy changed')
        require(admission['host_executable_sha256'] == self.config['binary_sha256']
                and admission['guest_configuration'] == policy
                and admission['configuration_sha256'] == digest(canonical(policy))
                and admission['inputs_sha256'] == digest(canonical(admission['compiler_inputs'])),
                'inspection installation release/configuration/toolchain admission mismatch')
        require(admission['compiler_inputs'] and all(self.config['files'].get(path) == value
                for path,value in admission['compiler_inputs'].items()),'inspection compiler inventory not frozen')
        require(self.config['files'].get(installation['compiler_path']) == installation['compiler_sha256'],
                'inspection compiler executable not frozen')
        runtime_pins = {read(path)['runtime_implementation_sha256'] for path in self.config['installations'].values()
                        if read(path)['profile'] == 'gomad.wasi-cooperative/v1'}
        require(runtime_pins == {installation['runtime_implementation_sha256']},'inspection actual runtime implementation differs')

    def inspection(self, root, owner):
        observed_owner = self.owner(root)
        argv = observed_owner['argv']
        require(argument(argv,'--owner') == owner and argument(argv,'--binary') == self.config['binary']
                and argument(argv,'--repo') == self.config['repo'] and argument(argv,'--root') == str(root),
                'inspection owner argv differs from frozen release/root/source')
        repo = Path(self.config['repo'])
        capture = None
        if owner == 'onebox':
            captured_path = argument(argv,'--captured-config')
            require(captured_path in self.config['files'],'inspection captured schema input not frozen')
            value = read(captured_path)
            capture = value.get('Config',value)['CapturedInputs']
        policy = workloads.inspection_policy(owner,capture)
        require(read(regular(root/'policy.json')) == policy,'inspection original policy changed')
        fuel,memory = (1<<50,4<<30) if owner=='onebox' else (50_000_000_000,1<<30)
        installation = read(regular(root/'installation.json'))
        self.installed(installation,policy,fuel,memory)
        installed_command = [self.config['binary'],'install','--compiler',argument(argv,'--compiler'),
            '--runtime-root',argument(argv,'--runtime-root'),'--cache',argument(argv,'--cache'),
            '--output',str(root/'installation.json'),'--profile','gomad.wasi-cooperative/v1','--config',str(root/'policy.json'),
            '--fuel',str(fuel),'--memory-bytes',str(memory)]
        require(installation['compiler_path'] == argument(argv,'--compiler') and installation['runtime_root'] == argument(argv,'--runtime-root')
                and installation['cache_root'] == argument(argv,'--cache'),'inspection installed actual lookup paths changed')
        self.nested(root,'install',installed_command)
        receipt = read(regular(root/'inspection-receipt.json'))
        require(receipt['schema'] == 'gomad.rust-temporal-inspection/v1' and receipt['owner'] == owner,'wrong inspection owner')
        artifact = self.contained(Path(receipt['artifact']))
        require(artifact.is_relative_to(root/'artifacts'),'inspection borrowed foreign producer artifact')
        actual = self.artifact(artifact)
        expected_hashes = {str(path.relative_to(artifact)):sha(regular(path)) for path in artifact.rglob('*') if path.is_file()}
        require(receipt['artifact_sha256'] == expected_hashes,'inspection changed raw artifact')
        stage = repo/'service/history'/('.gomad-inspection-'+owner+'-'+digest(str(root).encode())[:16])
        detached = stage.with_name(stage.name+'-unavailable')
        require(receipt['source_detached'] == str(detached) and detached.is_dir() and not detached.is_symlink() and not stage.exists(),
                'inspection original source was not actually detached')
        original = repo/(workloads.WASM/'testdata/prototype_temporal' if owner=='onebox' else Path('service/history')/owner)
        sources = {path.name:path for path in original.iterdir() if path.is_file() and path.name.endswith('.go')
                   and (owner=='onebox' or not path.name.endswith('_test.go') or path.name=='gomad_debug_test.go')}
        require(sources and read(regular(root/'unchanged-source-inputs.json')) == {str(path):sha(regular(path)) for path in sources.values()}
                and {path.name for path in detached.iterdir()} == set(sources),'inspection original/staged source inventory mismatch')
        for name,path in sources.items():
            require(self.config['files'].get(str(path)) == sha(regular(path)) and regular(detached/name).read_bytes() == path.read_bytes(),
                    'inspection staged source differs from frozen original bytes')
        selection,dump,probe,snapshot = workloads.INSPECTION[owner]
        arguments = ['-test.run=^'+selection+'$','-test.v']
        explore = [self.config['binary'],'--installation',str(root/'installation.json'),'explore','--json','--seeds','7','--parallel','1',
            '--execution-timeout','30m' if owner=='onebox' else '5m','--overall-timeout','45m' if owner=='onebox' else '15m',
            '--terminate-grace','1s','--on-failure','first','--failure-budget','1','--output-limit',str(1<<20),'--world-transition-limit',str(1<<20),
            '--artifacts',str(root/'artifacts'),'--env','GOMAD_DEBUG_CANARY=1']
        if owner=='onebox':
            require(read(regular(root/'scenario.json')) == workloads.inspection_scenario(arguments),'inspection original onebox scenario changed')
            explore += ['--scenario',str(root/'scenario.json')]
            for value in ('TEMPORAL_ROOT=/workspace','TMPDIR=/tmp','TEMPORAL_TEST_LOG_LEVEL=warn'):
                explore += ['--env',value]
        else:
            explore += ['--choices','--choice-bytes',str(8<<20)]
        for tag in workloads.TAGS+['gomad']:
            explore += ['--build-tag',tag]
        explore += ['--working-dir',str(repo),'go-test','./'+str(stage.relative_to(repo)),'--',*arguments]
        explored = decoded(self.nested(root,'explore',explore,accepted=(1,)))
        require(all(explored[key] == 1 for key in ('attempted','failures','retained_evidence')), 'inspection exact retained canary count')
        candidates = [path.parent for path in (root/'artifacts').rglob('manifest.json')
                      if read(regular(path)).get('record',{}).get('outcome',{}).get('class') == 'target_failure']
        require(candidates == [artifact],'inspection actual failure artifact coverage differs')
        provenance = read(regular(artifact/'target/provenance.json'))
        derived = self.provenance(dict(provenance=provenance))
        inputs = provenance['inputs']
        require(inputs['kind'] == 'go-test' and inputs['source'] == './'+str(stage.relative_to(repo))
                and inputs['arguments'] == arguments and inputs['directory'] == str(repo)
                and inputs['tags'] == sorted(set(workloads.TAGS+['gomad']+installation.get('build_tags',[])))
                and derived['fuel'] == fuel and derived['memory_bytes'] == memory
                and inputs['compiler_path'] == installation['compiler_path']
                and inputs['runtime_implementation'] == installation['runtime_implementation_sha256'],
                'inspection raw provenance target/toolchain/profile/limits changed')
        for path,value in inputs['sources'].items():
            source = Path(path)
            require((source.is_relative_to(stage) and sha(regular(detached/source.relative_to(stage))) == value)
                    or self.config['files'].get(path) == value,'inspection transitive source input not frozen')
        prepared = inspection_configuration(installation,arguments,owner,provenance['configuration']['runtime_sha256'])
        require(provenance['configuration'] == prepared,'inspection actual prepared model/namespace/clock/limits differs from original policy')
        if owner=='onebox':
            normalized = workloads.inspection_scenario(arguments)
            normalized['spec'].update(faults=None,exploration=None,scenario_choice_plan=None,replay=None)
            normalized['spec']['nodes'][0].update(config=None,volumes=[])
            normalized['boots'][0]['storage'] = None
            require(decoded(bytes(provenance['scenario'])) == normalized,'inspection prepared original scenario differs')
        else:
            require(provenance['scenario'] is None,'inspection standalone source gained scenario')
        require(module_metadata(regular(artifact/'target/module.wasm').read_bytes()) == provenance['module_metadata'],
                'inspection raw module metadata differs')
        record = actual['record']
        require(record['identity']['target'] == derived['module_sha256'] and record['identity']['toolchain'] == derived['toolchain_sha256']
                and record['identity']['implementation'] == digest(canonical(self.config['identity']))
                and record['profile'] == 'gomad.wasi-cooperative/v1' and record['seed'] == 7
                and record['limits']['fuel'] == fuel and record['limits']['memory_bytes'] == memory
                and record['outcome']['class'] == 'target_failure','inspection actual artifact belongs to foreign executable/target')
        replay_inputs = read(regular(artifact/('evidence/cluster-replay.json' if owner=='onebox' else 'evidence/replay.json')))
        require(replay_inputs['host_sha256'] == self.config['identity']['host'] and replay_inputs['helper_sha256'] == self.config['identity']['helper']
                and replay_inputs['toolchain_sha256'] == derived['toolchain_sha256']
                and replay_inputs['runtime_implementation_sha256'] == installation['runtime_implementation_sha256']
                and digest(canonical(replay_inputs['configuration'])) == record['identity']['configuration'],
                'inspection actual replay configuration/implementation binding changed')
        if owner=='onebox':
            configuration = replay_inputs['configuration']
            require(configuration['steps'] == workloads.inspection_scenario(arguments)['actions']
                    and configuration['workload'] == dict(kind='temporal-onebox-stopped-inspection')
                    and len(configuration['incarnations']) == 1,'inspection onebox actual lifecycle/owner coverage differs')
            incarnation = configuration['incarnations'][0]
            require(incarnation['handle'] == dict(node='history',incarnation=1)
                    and incarnation['model'] == prepared['model'] and incarnation['namespace'] == prepared['namespace']
                    and incarnation['limits'] == prepared['limits'] and incarnation['arguments'] == prepared['arguments']
                    and incarnation['environment'] == dict(prepared['environment'],GOMADSEED='7'), 'inspection onebox actual guest model/policy differs')
        else:
            require(replay_inputs['configuration'] == prepared,'inspection actual standalone replay policy differs')
        campaign = next((path for path in artifact.parents if path.parent.name == 'campaigns'),None)
        require(campaign is not None,'inspection actual campaign root absent')
        self.journal(campaign,actual['record_sha256'])
        replay = self.nested(root,'detached-replay',[self.config['binary'],'replay',str(artifact)],accepted=(1,))
        require(b'reproduced=true' in replay and b'choice-replay=exact' in replay,'inspection full replay mismatch')
        stopped = [self.config['binary'],'replay','--stop=failure','--dump=stacks','--dump=heap','--dump='+dump,'--probe='+probe,
                   '--debug-bytes',str(8<<20),'--dump-bytes',str(4<<20),'--debug-output',str(root/'inspection')]
        if owner=='onebox':
            stopped += ['--instance=history','--incarnation=1','--dump=temporal.shards']
        report = decoded(self.nested(root,'stopped-inspection',stopped+[str(artifact)]))
        require(report == receipt['report'] and report['status'] == 'inspected_prefix' and report['prefix_verified'] is True
                and report['witness_match'] is True and report['output_path'] == str(root/'inspection/debug.json'),
                'inspection prefix/witness/output binding mismatch')
        for name in ('detached-replay','stopped-inspection'):
            environment = read(root/(name+'.command.json'))['environment']
            require(environment == dict(PATH='',GOMAD_WASM_INSTALLATION=str(root/'installation-unavailable.json'),
                GOMAD_STOCK_GO=str(root/'compiler-unavailable'),GOMAD_WASM_RUNTIME_ROOT=str(root/'runtime-unavailable')),
                'inspection compiler/runtime/installation not unavailable')
        hashes = {str(path.relative_to(root/'inspection')):sha(regular(path)) for path in (root/'inspection').rglob('*') if path.is_file()}
        require(receipt['inspection_sha256'] == hashes,'inspection raw debug/heap bytes changed after observation')
        debug = read(regular(self.contained(report['output_path'])))
        require(debug['stacks'] and debug['probes'] and gzip.decompress(regular(root/'inspection/heap.pprof').read_bytes()),
                'inspection lacks useful stacks, heap or probes')
        wanted = {dump:snapshot}
        if owner=='onebox':
            wanted['temporal.shards'] = 'snapshot='
            require({'temporal.shard.snapshot','temporal.mutable_state.snapshot'} <= {item['name'] for item in debug['probes']},
                    'onebox missing original probe owners')
            output = (artifact/'evidence/cluster/history/1/stdout').read_bytes()
            require(b'workflow completed [step 0 attempt 2 step 1 attempt 2]' in output
                    and b'Temporal inspection canary after workflow timers and activity retries completed' in output,
                    'onebox inspection preceded completed timers/retries')
        else:
            require(any(item['name'] == probe for item in debug['probes']), 'inspection registered probe missing')
        for name,marker in wanted.items():
            rows = [item for item in debug['dumps'] if item['name'] == name]
            require(len(rows) == 1 and rows[0]['status'] == 'ok','inspection registered snapshot missing/duplicate')
            body = base64.b64decode(rows[0]['data_base64'],validate=True)
            require(marker.encode() in body and (name != dump or b'values may precede stop' in body), 'inspection original snapshot assertion')


def read_receiving(config):
    """Return only owned rows; missing raw evidence remains explicitly UNEXECUTED."""
    reader = Retained(config)
    reader.adapter.admission(config,reader.root)
    checks = [reader.runtime, reader.mandatory, reader.sqlite_primary,
              *(lambda index=index: reader.selected(index) for index in range(4)), reader.catalogue,
              reader.smoke,
              reader.replication, lambda: reader.temporal('storage'),
              lambda: reader.temporal('network-streams'), lambda: reader.temporal('temporal-inspection')]
    rows = []
    for name, check in zip(OWNED,checks):
        try:
            check()
            rows.append(dict(id=name,status='PASS',executed=True,exit=0))
        except (Refusal, OSError, KeyError, TypeError, ValueError, IndexError, struct.error) as error:
            rows.append(dict(id=name,status='UNEXECUTED' if isinstance(error,(Unexecuted,FileNotFoundError)) else 'FAIL',
                             executed=False,exit=4,reason=str(error)))
    return rows


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config',required=True,type=Path)
    parser.add_argument('--root',required=True,type=Path)
    args = parser.parse_args()
    try:
        config = configured(args.config)
        config['_config_path'] = str(args.config.resolve())
        config['_config_sha256'] = sha(args.config)
        require(str(args.root) == config['root'],'receiving root differs from frozen configuration')
        guard(config)
        rows = read_receiving(config)
        guard(config)
        report = dict(schema='gomad.rust-driver-final-section/v1',source_sha256=config['source_sha256'],
                      identity=config['identity'],subgates=rows)
        print(json.dumps(report,sort_keys=True,separators=(',',':')))
        return 0 if all(row['status'] == 'PASS' for row in rows) else 4
    except (Refusal,OSError,KeyError,TypeError,ValueError,IndexError,struct.error) as error:
        print('receiving: '+str(error),file=sys.stderr)
        return 4


if __name__ == '__main__':
    raise SystemExit(main())
