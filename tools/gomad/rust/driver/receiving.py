"""Read retained receiving evidence without executing producers or modifying reports."""
import argparse
import base64
import gzip
import hashlib
import json
from pathlib import Path
import types
import struct
import sys

import adapters
import sqlite
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
        inputs, configuration, manifest = provenance['inputs'], provenance['configuration'], result['manifest']
        require(provenance['schema'] == 'gomad.rust-wasm-provenance/v1'
                and inputs['schema'] == 'gomad.rust-wasm-build-inputs/v1'
                and digest(canonical(inputs)) == provenance['build_key'], 'qualification build-key preimage mismatch')
        built = dict(configuration,runtime_sha256=None)
        require(inputs['configuration'] == built and inputs['network'] == provenance['network']
                and inputs['scenario'] == provenance['scenario'], 'qualification duplicated input preimages changed')
        choice = provenance['choice_identity']
        profile = {'gomad.wasi-cooperative/v1':'cooperative','gomad.wasi-stock-exploratory/v1':'controlled'}
        require(configuration['profile'] in profile,'qualification profile not admitted')
        require(set(inputs) == set(('arguments','build_flags','compiler','compiler_sha256','configuration','directory','environment','go_version','helper','host','kind','network','runtime','runtime_implementation','scenario','schema','soft_assertion_report_only','source','sources','tags')),
                'qualification build input field inventory changed')
        require(set(provenance)-{'runtime_workload'} == set(('build_key','choice_identity','configuration','inputs','module_sha256','network','scenario','schema')),
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
        require(all(manifest[key] == value for key,value in expected.items()),'qualification manifest differs from raw provenance')

    def result(self, path, reported):
        path = self.contained(path)
        raw = regular(path).read_bytes()
        result = read(path)
        require(raw == canonical(result)+b'\n' and path.name == 'result-'+digest(raw)+'.json',
                'Result canonical bytes/content address mismatch')
        require(reported['result'] == result, 'public report differs from raw Result')
        pairs = list(path.parent.glob(path.name+'.publication-*.json'))
        require(len(pairs) == 1, 'exactly one publication receipt required')
        receipt_raw = regular(pairs[0]).read_bytes()
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
        require(result['schema'] == 'gomad.rust-qualification-result/v1'
                and manifest['integrated_source_sha256'] == self.config['source_sha256'], 'foreign Result source/schema')
        for key, value in self.config['identity'].items():
            require(manifest[key+'_sha256'] == value, 'foreign Result identity: '+key)
        self.provenance(result)
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
        require(len(executions) == 2 and [row['iteration'] for row in executions] == [0,1], 'exact two Result iterations required')
        paths = set()
        for execution in executions:
            artifact = execution['artifact_path']
            require(artifact not in paths, 'duplicate qualification artifact')
            paths.add(artifact)
            actual = self.artifact(Path(artifact))
            self.journal(Path(execution['campaign_path']), execution['record_sha256'])
            require(actual['record'] == execution['record'] and actual['record_sha256'] == execution['record_sha256'],
                    'Result execution differs from authenticated raw artifact')
            require(actual['record']['identity']['target'] == manifest['module_sha256']
                    and actual['record']['identity']['toolchain'] == manifest['toolchain_sha256']
                    and actual['record']['seed'] == manifest['seed'], 'qualification module/toolchain/seed differs from execution')
            replay = execution['replay']
            require(replay and replay['attempted'] is True and replay['matched'] is True
                    and replay['divergence'] is None and replay['choice_replay'] == 'exact', 'incomplete full-tape Result replay')
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

    def nested(self, root, name, accepted=(0,)):
        row = read(regular(self.contained(root)/(name+'.command.json')))
        require('stdout_sha256' in row and 'environment' in row and 'environment_sha256' in row,
                'historical producer lacks authenticated raw observations')
        return self.command(self.config, root, name, row['argv'], accepted=accepted)

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
        self.nested(root,'install')
        qualify_argv = [self.config['binary'],'--installation',str(root/'installation.json'),'qualify','--spec',str(root/'qualification-spec.json')]
        response = decoded(self.command(self.config,root,'qualify',qualify_argv,accepted=(1,) if mode=='canary' else (0,)))
        path = Path(receipt['result_path'])
        require(response['result_path'] == str(path) and sha(regular(path)) == receipt['result_sha256'], 'actual Result path/hash mismatch')
        reported = decoded(self.nested(root,'report'))
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
            for owner in workloads.INSPECTION:
                self.nested(root,owner)
                child = self.owner(root/owner,read(root/(owner+'.command.json'))['argv'])
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
        self.nested(root,'primary-report'); self.nested(root,'primary-final-report')
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
            self.nested(cell,'qualify')
            result = self.adapter.result_report(self.config,cell,'report',row['result_path'])
            pid = result['observations']['operator_process_id']
            require(pid not in pids and pid == read(cell/'qualify.command.json')['pid']
                    and sha(regular(row['result_path'])) == row['result_sha256'], 'Temporal selected Result/PID binding')
            pids.add(pid)
            for execution in result['report']['executions']:
                artifact = Path(execution['artifact_path'])
                for stream in ('choice','stdout','storage'):
                    require((artifact/('evidence/cluster/temporal/1/'+stream)).read_bytes() ==
                            (primary_artifact/('evidence/cluster/temporal/1/'+stream)).read_bytes(), 'Temporal primary interrupted prefix changed')
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

    def inspection(self, root, owner):
        self.owner(root)
        receipt = read(regular(root/'inspection-receipt.json'))
        require(receipt['schema'] == 'gomad.rust-temporal-inspection/v1' and receipt['owner'] == owner,'wrong inspection owner')
        artifact = Path(receipt['artifact'])
        actual = self.artifact(artifact)
        expected_hashes = {str(path.relative_to(artifact)):sha(regular(path)) for path in artifact.rglob('*') if path.is_file()}
        require(receipt['artifact_sha256'] == expected_hashes,'inspection changed raw artifact')
        detached = Path(receipt['source_detached'])
        require(detached.is_dir() and not detached.is_symlink() and not detached.with_name(detached.name.removesuffix('-unavailable')).exists(),
                'inspection source was not detached')
        selection,dump,probe,snapshot = workloads.INSPECTION[owner]
        installation = read(regular(root/'installation.json'))
        require(installation['fuel'] == (1<<50 if owner=='onebox' else 50_000_000_000)
                and installation['memory_bytes'] == (4<<30 if owner=='onebox' else 1<<30), 'inspection original resource bounds')
        self.nested(root,'install')
        explored = decoded(self.nested(root,'explore',accepted=(1,)))
        require(all(explored[key] == 1 for key in ('attempted','failures','retained_evidence')), 'inspection exact retained canary count')
        command = read(root/'explore.command.json')['argv']
        for flag,value in (('--execution-timeout','30m' if owner=='onebox' else '5m'),
                           ('--overall-timeout','45m' if owner=='onebox' else '15m'),('--seeds','7'),('--parallel','1')):
            require(command.count(flag) == 1 and command[command.index(flag)+1] == value,'inspection changed original bound '+flag)
        require(command[-2:] == ['-test.run=^'+selection+'$','-test.v'],'inspection changed original selection')
        replay = self.nested(root,'detached-replay',accepted=(1,))
        require(b'reproduced=true' in replay and b'choice-replay=exact' in replay,'inspection full replay mismatch')
        report = decoded(self.nested(root,'stopped-inspection'))
        require(report == receipt['report'] and report['status'] == 'inspected_prefix'
                and report['prefix_verified'] is True and report['witness_match'] is True,'inspection prefix/witness mismatch')
        for name in ('detached-replay','stopped-inspection'):
            environment = read(root/(name+'.command.json'))['environment']
            require(environment == dict(PATH='',GOMAD_WASM_INSTALLATION=str(root/'installation-unavailable.json'),
                GOMAD_STOCK_GO=str(root/'compiler-unavailable'),GOMAD_WASM_RUNTIME_ROOT=str(root/'runtime-unavailable')),
                'inspection compiler/runtime/installation not unavailable')
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
