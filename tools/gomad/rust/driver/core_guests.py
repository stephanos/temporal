"""Read actual original Go guest bytes; broad_harness owns executable/process custody."""
import base64
import os
from pathlib import Path, PurePosixPath
import stat

from driver import require
from native_roles import decode, pinned
from native_observations import canonical, digest

MARKER = 'GOMAD_RETAINED_OBSERVATION '
FILE = 'evidence/cluster-replay.json'
SOURCE = 'tools/gomad/rust/gomad/src/replay/cluster/broad_guest_tests.rs'
SOFT = 'TestSoftAssertionActualClusterGuestAndRetainedReplay'
CAPACITY = 'TestSoftAssertionActualClusterOutputCapacityRemainsInfrastructureFailure'
PLAIN = 'TestProviderIndependentGuestsLifecycleReplay'
COMPRESSED = 'TestProviderCompressedIndependentGuestsLifecycleReplay'
CANARY = 'TestProviderDetachedOracleCanarySourceLessReplay'
GROUPED = 'TestProviderGroupedAcknowledgmentCrashSourceLessReplay'
PINS = {
    'tests/gomadfunctional/testdata/softassert/softassert_test.go': '3bc1fe2c5b6cc5f59cce0dec742bec77efa0a931e5976c7dec741b0b3a1274a5',
    'tools/gomad/internal/backends/wasm/cluster/testdata/service/main.go': '35278d6687f89f68716a020ecadfe6c6aaa73bed5c0741963cd50f9d87fb13d6',
    'tools/gomad/internal/backends/wasm/cluster/testdata/service/output_capacity_test.go': '4c144724dc5c299d679f761cc24f00a43bd03f67d652a74e0520a024558afbaf',
}
CASES = {
    **{f'soft/seed={seed}/report-only={str(policy).lower()}': (SOFT, 'softassert', seed, policy, 4096)
       for seed, policy in ((7, False), (19, False), (7, True))},
    **{f'capacity/report-only={str(policy).lower()}/maximum={maximum}': (CAPACITY, 'capacity', 7, policy, maximum)
       for policy, maximum in ((False, 4096), (True, 4096), (True, 1))},
    'lifecycle/compressed=false': (PLAIN, 'lifecycle', 7, False, 4096),
    'lifecycle/compressed=true': (COMPRESSED, 'lifecycle', 7, False, 4096),
    'canary/enabled=false': (CANARY, 'canary', 7, False, 4096),
    'canary/enabled=true': (CANARY, 'canary', 7, False, 4096),
    'grouped/replay': (GROUPED, 'grouped', 7, False, 4096),
}


def fields(value, expected):
    require(isinstance(value, dict) and set(value) == set(expected), 'foreign actual Go guest fields: ' + repr(expected))


def same(left, right):
    return canonical(left) == canonical(right)


def uint(value):
    require(type(value) is int and 0 <= value < 1 << 64, 'foreign actual Go guest integer')
    return value


def b64(value):
    require(isinstance(value, str) and len(value) <= 180 << 20, 'foreign actual Go raw bytes')
    raw = base64.b64decode(value, validate=True)
    require(base64.b64encode(raw).decode() == value, 'noncanonical actual Go base64')
    return raw


def name(value):
    require(isinstance(value, str) and value and '\\' not in value and not value.startswith('/')
            and str(PurePosixPath(value)) == value and all(part not in ('', '.', '..') for part in value.split('/')),
            'foreign actual Go retained filename')
    return value


class RawFiles:
    def __init__(self, root):
        self.root = Path(root)
        require(self.root.is_absolute() and self.root.resolve() == self.root, 'foreign guest evidence root')
        self.seen = {}

    def read(self, path, maximum=512 << 20):
        path = Path(path)
        require(path.is_relative_to(self.root) and path.resolve() == path
                and not any(parent.is_symlink() for parent in (path, *path.parents)), 'guest evidence escaped raw custody')
        descriptor = os.open(path, os.O_RDONLY | os.O_CLOEXEC | os.O_NOFOLLOW)
        try:
            before = os.fstat(descriptor)
            require(stat.S_ISREG(before.st_mode) and 0 <= before.st_size <= maximum, 'guest raw file type or byte bound')
            chunks, total = [], 0
            while chunk := os.read(descriptor, 1 << 20):
                total += len(chunk)
                require(total <= maximum, 'guest raw file grew over bound')
                chunks.append(chunk)
            raw = b''.join(chunks)
            after = os.fstat(descriptor)
            key = lambda info: (info.st_dev, info.st_ino, info.st_size, info.st_mtime_ns, info.st_ctime_ns)
            require(key(before) == key(after) and len(raw) == after.st_size, 'guest raw file changed while read')
        finally:
            os.close(descriptor)
        fact = (key(after), digest(raw))
        require(str(path) not in self.seen or self.seen[str(path)] == fact, 'guest raw file changed between readers')
        self.seen[str(path)] = fact
        return raw

    def finish(self):
        for path in tuple(self.seen):
            self.read(path)
        return {path: fact[1] for path, fact in self.seen.items()}


def artifact_bytes(manifest_raw, record_raw, payloads):
    manifest, record = decode(manifest_raw), decode(record_raw)
    fields(manifest, ('schema', 'files', 'record', 'record_sha256'))
    require(manifest['schema'] == 'gomad.artifact-manifest/v2' and manifest_raw == canonical(manifest)
            and record_raw == canonical(record) and manifest['record'] == record
            and manifest['record_sha256'] == digest(record_raw) and payloads.get('record.json') == record_raw,
            'actual guest manifest/record bytes are not authenticated canonical lineage')
    require(set(payloads) == set(manifest['files']), 'guest raw payload inventory differs')
    for key, raw in payloads.items():
        name(key)
        entry = manifest['files'][key]
        fields(entry, ('bytes', 'sha256', 'module'))
        require(uint(entry['bytes']) == len(raw) and entry['sha256'] == digest(raw)
                and type(entry['module']) is bool and entry['module'] == (key == 'target/module.wasm'), 'guest payload raw authentication differs')
    require(record['execution_identity'] == digest(canonical(record['identity']))
            and record['identity']['target'] == digest(payloads['target/module.wasm']), 'guest primary/record identity differs')
    for key, stream in record['streams'].items():
        fields(stream, ('bytes', 'sha256', 'records'))
        require(key in payloads and uint(stream['bytes']) == len(payloads[key]) and uint(stream['records']) >= 0
                and stream['sha256'] == digest(payloads[key]), 'guest record stream differs from raw bytes')
    inputs = decode(payloads[FILE])
    require(inputs['schema'] == 'gomad.cluster-replay-inputs/v1'
            and record['identity']['configuration'] == digest(canonical(inputs['configuration'])), 'guest retained configuration identity differs')
    for item in inputs['configuration']['incarnations']:
        descriptor=item['module']
        require(descriptor['file']=='target/module.wasm' and descriptor['bytes']==len(payloads['target/module.wasm'])<=descriptor['limit']<=512<<20
                and descriptor['sha256']==record['identity']['target'], 'guest incarnation lost primary module binding')
    require(payloads['target/module.wasm'].startswith(b'\0asm\x01\0\0\0'), 'original Go guest is not a binary WASM module')
    return dict(manifest=manifest, record=record, inputs=inputs, payloads=payloads)


def read_artifact(files, path):
    path = Path(path)
    require(files.read(path / 'storage.json',1<<20)==b'{"format":"gomad.artifact-storage/v2"}', 'actual guest storage envelope lineage differs')
    manifest_raw = files.read(path / 'manifest.json', 64 << 20)
    manifest = decode(manifest_raw)
    payloads = {name(key): files.read(path / key) for key in manifest['files']}
    return artifact_bytes(manifest_raw, payloads['record.json'], payloads)


def handle(node, incarnation=1):
    return dict(node=node, incarnation=incarnation)


def step(action, node, incarnation=1):
    return dict(action=action, handle=handle(node, incarnation))


def owners(configuration):
    return [item['handle']['node'] + '/' + str(item['handle']['incarnation']) for item in configuration['incarnations']]


def expected_steps(root):
    if root == SOFT:
        return [step(action, node) for node in ('a', 'b') for action in ('start', 'wait')]
    if root == CAPACITY: return [step('start', 'a'), step('wait', 'a')]
    if root == CANARY: return [step(action, node) for action in ('start', 'wait') for node in ('a', 'b')]
    if root in (PLAIN, COMPRESSED):
        return [step('start', 'a'), step('start', 'b'), step('wait', 'b'), step('wait', 'a'),
                step('restart', 'a', 2), step('restart', 'b', 2), step('wait', 'b', 2), step('wait', 'a', 2)]
    return [step('start', 'a'), step('start', 'b'), step('crash', 'a'), step('crash', 'b'),
            step('restart', 'a', 2), step('restart', 'b', 2), step('wait', 'a', 2), step('wait', 'b', 2)]


def configuration(case, c):
    root, _, seed, policy, maximum = CASES[case]
    wanted = ['a/1'] if root == CAPACITY else ['a/1', 'b/1']
    if root in (PLAIN, COMPRESSED, GROUPED): wanted += ['a/2', 'b/2']
    require(owners(c) == wanted and c['nodes'] == (['a'] if root == CAPACITY else ['a', 'b'])
            and c['steps'] == expected_steps(root) and uint(c['broker']['seed']) == seed, 'original guest owners/seed/actions differ')
    require(c.get('workload') is None and c.get('simulation_original') is None and c['boundaries'] == [], 'actual guest changed workload/boundary intent')
    require(c['broker']['pause_on_settle'] is False, 'actual guest changed broker pause semantics')
    if root == COMPRESSED:
        require(c['world_encoding'] == c['world_profile'] == 'gomad/world/compressed-snapshot/v2', 'explicit compressed guest profile lost')
        broker = dict(max_requests=2<<20, max_events=2<<20, max_queued_events=250000,
                      max_transitions=8<<20, max_payload_bytes=64<<20, max_string_bytes=(1<<32)-1)
    else:
        require(c['world_encoding'] == '' and c.get('world_profile') in ('', None), 'plain guest acquired foreign World profile')
        broker = dict(max_requests=2<<20, max_events=2<<20, max_queued_events=65536,
                      max_transitions=8<<20, max_payload_bytes=64<<20, max_string_bytes=1024)
    require(same(c['broker']['limits'], broker), 'original guest broker bounds changed')
    network=dict(profile='gomad.simulation-network-stream/v3',faults=[],limits=dict(listeners=128,connections=128,bytes=8<<20,connection_bytes=4<<20,stall_nanos=10_000_000_000,transitions=10000))
    links=[dict(**{'from':a,'to':b},enabled=True,delay_nanos=0) for a,b in (('a','b'),('b','a'))] if root in (PLAIN,COMPRESSED) else []
    require(same(c['network'],dict(model=network,addresses={node:f'10.0.0.{index+1}' for index,node in enumerate(c['nodes'])},links=links,strict_topology=True,faults=[])), 'original guest shared network intent changed')
    timeout = 300000 if root == SOFT else 60000
    for item in c['incarnations']:
        limits = item['limits']; model = item['model']; namespace = item['namespace']
        require(same(item['network'],network), 'original guest per-owner network limits changed')
        require(limits['fuel'] == 4000000000 and limits['memory_bytes'] == 256<<20 and limits['wall_millis'] == timeout
                and limits['output_bytes'] == (4 if root == CAPACITY else 1<<20), 'original actual Go runtime bounds changed')
        require(model['mode'] == 1 and model['limit'] == 8<<20 and model['diagnostics'] is True
                and 'seed' not in model and model['epoch_nanos'] == 946684800000000000 and model['read_step_nanos'] == 0
                and model['entropy_key'] == [1,2,3] + [0]*29
                and same(model['limits'],dict(calls=100000, transcript_bytes=32<<20, pending_events=4096)), 'original actual Go model inputs changed')
        require(namespace['working_directory'] == list(b'/workspace') and namespace['writable_directories'] == [list(b'/workspace'), list(b'/tmp')]
                and namespace['epoch_nanos'] == 946684800000000000
                and same(namespace['limits'],dict(files=1024, descriptors=128, filesystem_bytes=8<<20, output_bytes=4 if root == CAPACITY else 1<<20)), 'original guest namespace changed')
        require(item['environment']['PWD'] == '/workspace' and item['environment']['TZ'] == 'UTC', 'original guest environment changed')
        node, incarnation = item['handle']['node'], item['handle']['incarnation']
        expected = ['soft-assertion','-test.v','-assertion-mode=late'] if root == SOFT else ['capacity','-test.run=TestOutputCapacity'] if root == CAPACITY else ['service']
        if root in (PLAIN, COMPRESSED): expected += ['server'] if node == 'a' else ['client','10.0.0.1']
        elif root == CANARY: expected += ['observation'] + (['canary'] if node == 'b' and case.endswith('true') else [])
        elif root == GROUPED: expected += ['durable' if incarnation == 1 else 'recover']
        require(item['arguments'] == expected and item['workload_kind'] == ('go-test' if root in (SOFT,CAPACITY) else 'go-run'), 'original guest program/arguments/workload changed')
        if root == GROUPED:
            descriptor = item['storage']['descriptor']
            from core_codec import descriptor_model
            descriptor_model(descriptor, 1024, 10000, [dict(ID=node, Path='/workspace', CapacityBytes=8<<20)])
        else: require(item['storage'] is None, 'ordinary guest acquired storage')
    plan = c.get('oracle_plan')
    if policy:
        require(same(plan,dict(schema='gomad.wasm.soft-assertion-observation/v1', maximum_bytes=maximum,
                            histories=[], convergence=[], prefix='', soft_assertions=[dict(handle=item['handle'],report_only=True) for item in c['incarnations']])), 'report-only policy not bound to original Go-test owners')
    elif root != CANARY: require(plan is None, 'default guest has a synthetic opt-out policy')
    if root == CANARY:
        require(plan==dict(schema='gomad.wasm.history-observation/v1',prefix='GOMAD_HISTORY_OBSERVATION ',maximum_bytes=4096,soft_assertions=[],
            histories=[dict(name='fixture.recovered',expected=handle('a'),actual=handle('b'),expected_events=['1','2'])],
            convergence=[dict(name='fixture.converged',handles=[handle('a'),handle('b')])]), 'original history oracle intent changed')
    if root == GROUPED:
        require(c['crash_groups'] == [dict(nodes=['a','b'], marker=list(b'committed durable-a'))], 'group acknowledgment marker/owners changed')
    else: require(c['crash_groups'] == [], 'ordinary guest acquired crash groups')
    return root


def oracle(name, passed, evidence, *, soft=False, policy=False, inconclusive=False):
    values = [dict(label=key,value=base64.b64encode(raw).decode(),full_sha256=digest(raw)) for key,raw in evidence]
    value = dict(name=name,passed=passed,inconclusive=inconclusive,soft_assertion=soft,report_only=policy,
                 evidence=values,failure_identity='',identity='')
    if not passed:value['failure_identity']=digest(canonical(dict(name=name,evidence=values)))
    value['identity']=digest(canonical(value))
    return value


def report(oracles):
    failed = [item for item in oracles if not item['passed']]
    return dict(schema='gomad.cluster-oracles/v1',outcome='oracle_failed' if failed else 'completed',
                reason='soft_assertion' if any(item['soft_assertion'] for item in failed) else 'oracle_failed' if failed else '',
                failure_identity=digest(canonical(failed)) if failed else '',oracles=oracles)


def soft_oracle(owner, stdout, stderr, policy, complete=True):
    messages=[]
    for raw in (stdout,stderr):
        text=raw.decode('utf8')
        for line in text.split('\n'):
            if line.startswith('{'):
                value=decode(line)
                tagged=value.get('failed-assertion',value.get('FailedAssertion',False))
                message=value.get('msg',value.get('message',''))
                require(type(tagged) is bool and isinstance(message,str), 'foreign original logger output')
                if tagged or message.startswith('failed assertion: '):
                    require(message.startswith('failed assertion: '), 'original soft logger message lacks assertion prefix')
                    messages.append(message[len('failed assertion: '):])
            elif 'failed assertion: ' in line: messages.append(line.split('failed assertion: ',1)[1])
    messages=sorted(set(messages))
    require(all(messages) and all(len(message.encode())<=4096 for message in messages), 'foreign original static soft assertion')
    if not messages and complete and not policy:return None
    evidence=[('instance',owner.encode()),('messages',canonical(messages)),('status',b'observed' if complete else b'inconclusive')]
    return oracle('soft-assertion.instance-'+digest(owner.encode()),complete and (not messages or policy),evidence,soft=True,policy=policy,inconclusive=not complete)


def history_report(outputs, canary):
    histories=[]
    for owner in ('a/1','b/1'):
        rows=[line[len(b'GOMAD_HISTORY_OBSERVATION '):] for line in outputs[owner]['stdout'].split(b'\n') if line.startswith(b'GOMAD_HISTORY_OBSERVATION ')]
        require(len(rows)==1 and len(rows[0])<=4096, 'history line missing/ambiguous/oversized')
        value=decode(rows[0]);fields(value,('cluster_id','history','event_ids','types'))
        expected=b'different history' if canary and owner=='b/1' else b'committed history'
        require(value==dict(cluster_id='independent-oracle-reference',history=base64.b64encode(expected).decode(),event_ids=[1,2],types=['started','completed']), 'actual original Go history observation differs')
        histories.append(value)
    a,b=histories
    operation=lambda value:canonical([dict(id='committed',actor='workflow',kind='workflow-history',invocation=1,completion=2,output=value['history'])])
    return report([
        oracle('fixture.recovered.history',a['history']==b['history'],[('actual',operation(b)),('expected',operation(a))]),
        oracle('fixture.recovered.events',True,[('actual',canonical(['1','2'])),('expected',canonical(['1','2']))]),
        oracle('fixture.recovered.identity',True,[('actual',b['cluster_id'].encode()),('expected',a['cluster_id'].encode())]),
        oracle('fixture.converged',a['history']==b['history'],[('a-1',b64(a['history'])),('b-1',b64(b['history']))])])


def artifact_outputs(artifact):
    inputs,payloads=artifact['inputs'],artifact['payloads']
    summary=decode(payloads['evidence/cluster_summary'])
    require(summary['schema']=='gomad.cluster-observation/v1', 'actual guest terminal summary lineage differs')
    require(summary['world_sha256']==digest(payloads['evidence/world']) and summary['network_sha256']==digest(payloads['evidence/network']), 'guest summary world/network raw hashes differ')
    from core_world import expand, decode_world
    world=payloads['evidence/world']
    decode_world(expand(world) if world.startswith(b'GMWORLD2') else world, dict(seed=inputs['configuration']['broker']['seed'], **inputs['configuration']['broker']['limits']))
    actual={}
    for item in inputs['retained']:
        owner=item['handle']['node']+'/'+str(item['handle']['incarnation'])
        require(owner not in actual, 'duplicate actual guest output owner')
        streams={}
        for key,descriptor in item['evidence'].items():
            raw=payloads[descriptor['file']]
            require(descriptor['bytes']==len(raw)<=descriptor['limit'] and descriptor['sha256']==digest(raw), 'actual retained per-owner descriptor differs')
            streams[key]=raw
        rows=[row for row in summary['incarnations'] if row['handle']==item['handle']]
        require(len(rows)==1 and rows[0]['interrupted']==item['interrupted'], 'actual retained terminal summary owner differs')
        terminal=rows[0]
        for key,descriptor in terminal['streams'].items():
            require(key in streams and descriptor==dict(bytes=len(streams[key]),sha256=digest(streams[key])), 'guest summary stream not raw bound')
        require(terminal['offered_output_bytes']==len(streams.get('stdout',b''))+len(streams.get('stderr',b'')), 'retained successful output is incomplete')
        actual[owner]=dict(stdout=streams.get('stdout',b''),stderr=streams.get('stderr',b''),terminal=terminal)
    return actual


def grouped_snapshots(value, artifact, configuration):
    """Merge only the latest actual per-owner snapshot streams, never configured models."""
    import re
    from core_codec import pending_identity, snapshot_identity, state_raw_sections, recovery_image, recovered_snapshots, observed_operation
    fields(value, ('schema', 'incarnations'))
    require(value['schema'] == 'gomad.grouped-latest-storage/v2' and isinstance(value['incarnations'], list),
            'missing actual grouped storage snapshot schema')
    latest = {}
    for item in configuration['incarnations']:
        owner = item['handle']
        latest[owner['node']] = max(latest.get(owner['node'], 0), uint(owner['incarnation']))
    expected = [handle(node, incarnation) for node, incarnation in sorted(latest.items())]
    require(expected == [handle('a', 2), handle('b', 2)]
            and [item.get('handle') for item in value['incarnations']] == expected,
            'grouped snapshots omitted, duplicated or selected a stale actual owner')
    snapshots = []
    for item in value['incarnations']:
        fields(item, ('handle', 'state_bytes', 'log_bytes', 'image_bytes', 'storage_state_base64', 'storage_log_base64', 'storage_image_base64'))
        node = item['handle']['node']
        owner = node + '/' + str(item['handle']['incarnation'])
        state, log = b64(item['storage_state_base64']), b64(item['storage_log_base64'])
        image_raw = b64(item['storage_image_base64'])
        require(uint(item['state_bytes']) == len(state) and uint(item['log_bytes']) == len(log) and uint(item['image_bytes']) == len(image_raw),
                'grouped actual storage byte counts differ from preimages')
        require(state == artifact['payloads'][f'evidence/cluster/{owner}/storage_state']
                and log == artifact['payloads'][f'evidence/cluster/{owner}/storage']
                and image_raw == artifact['payloads'].get(f'evidence/cluster/{owner}/storage_image'),
                'grouped producer snapshots differ from their actual retained owner streams')
        version, sections = state_raw_sections(state)
        require(version == 1 and len(sections) == 5 and decode(sections[2]) is None
                and sections[3] == digest(image_raw).encode('ascii')
                and sections[4] == len(log).to_bytes(8, 'little') + bytes.fromhex(digest(log)),
                'grouped storage snapshot lost complete original framing, image or log binding')
        volumes, history = decode(sections[0]), decode(sections[1])
        require(isinstance(volumes, list) and len(volumes) == 1 and isinstance(history, dict)
                and set(history) == {node}, 'grouped actual storage snapshot/history inventory differs')
        snapshot = volumes[0]
        fields(snapshot, ('ID', 'Mount', 'CapacityBytes', 'Persisted', 'Volatile', 'PendingOperations',
                          'PendingSHA256', 'NextOperation', 'Identity'))
        require(snapshot['ID'] == node and snapshot['Mount'] == '/workspace'
                and uint(snapshot['CapacityBytes']) == 8 << 20 and uint(snapshot['NextOperation']) > 0
                and uint(snapshot['PendingOperations']) <= 1024
                and isinstance(snapshot['PendingSHA256'], str)
                and re.fullmatch(r'sha256:[0-9a-f]{64}', snapshot['PendingSHA256']),
                'grouped actual snapshot has a foreign volume owner or original bound')
        operations = history[node]
        require(isinstance(operations, list) and len(operations) <= 10000,
                'grouped actual snapshot operation history is missing or exceeds original bound')
        ids = []
        for operation in operations:
            fields(operation, ('ID', 'Kind', 'Dependencies', 'Inode', 'Offset', 'Bytes',
                               'PayloadSHA256', 'EffectSHA256', 'Path', 'Destination'))
            identity = uint(operation['ID'])
            require(0 < identity < snapshot['NextOperation'] and operation['Kind'] in ('alloc', 'write', 'resize', 'metadata', 'namespace'),
                    'grouped actual snapshot history operation identity differs')
            for key in ('Inode', 'Offset', 'Bytes'): uint(operation[key])
            require(isinstance(operation['Path'], str) and isinstance(operation['Destination'], str)
                    and isinstance(operation['EffectSHA256'], str) and re.fullmatch(r'sha256:[0-9a-f]{64}', operation['EffectSHA256'])
                    and isinstance(operation['PayloadSHA256'], str)
                    and (operation['PayloadSHA256'] == '' or re.fullmatch(r'sha256:[0-9a-f]{64}', operation['PayloadSHA256'])),
                    'grouped actual snapshot history payload/effect schema differs')
            dependencies = operation['Dependencies']
            require(dependencies is None or isinstance(dependencies, list)
                    and all(0 < uint(dependency) < identity for dependency in dependencies)
                    and dependencies == sorted(set(dependencies)), 'grouped actual snapshot history dependencies differ')
            ids.append(identity)
        require(ids == sorted(set(ids)) and snapshot['PendingOperations'] <= len(operations),
                'grouped actual snapshot history order/pending count differs')
        if snapshot['PendingOperations'] == 0:
            require(snapshot['PendingSHA256'] == pending_identity(node, []),
                    'grouped actual snapshot empty pending commitment differs')
        for key in ('Persisted', 'Volatile'):
            entries = snapshot[key]
            require(isinstance(entries, list) and entries, 'grouped actual snapshot lacks persisted/volatile entries')
            paths = []
            for entry in entries:
                fields(entry, ('Path', 'Mode', 'Kind', 'ModTime', 'Size', 'DataSHA256'))
                require(isinstance(entry['Path'], str) and entry['Path'].startswith('/')
                        and uint(entry['Mode']) < 1 << 32 and entry['Kind'] in ('file', 'directory')
                        and type(entry['ModTime']) is int and -(1 << 63) <= entry['ModTime'] < 1 << 63,
                        'grouped actual snapshot entry schema differs')
                uint(entry['Size'])
                require(isinstance(entry['DataSHA256'], str) and
                        (bool(re.fullmatch(r'sha256:[0-9a-f]{64}', entry['DataSHA256'])) if entry['Kind'] == 'file'
                         else entry['Size'] == 0 and entry['DataSHA256'] == ''), 'grouped actual snapshot entry data identity differs')
                paths.append(entry['Path'])
            require(paths == sorted(set(paths)) and sum(entry['Size'] for entry in entries) <= snapshot['CapacityBytes'],
                    'grouped actual snapshot entry order/capacity differs')
            durable = [entry for entry in entries if entry['Path'] == '/value']
            require(len(durable) == 1 and durable[0]['Kind'] == 'file' and durable[0]['Size'] == len(b'durable-a')
                    and durable[0]['DataSHA256'] == 'sha256:' + digest(b'durable-a'),
                    'grouped actual snapshot does not preserve acknowledged durable bytes')
        require(snapshot['Identity'] == snapshot_identity(snapshot), 'grouped actual snapshot identity differs from its raw entries/history counters')
        image = recovery_image(image_raw)
        declared = next(boot for boot in configuration['incarnations'] if boot['handle'] == item['handle'])
        require(image['model'] == declared['storage']['descriptor']['model_identity'] and set(image['volumes']) == {node},
                'grouped actual storage image belongs to a foreign model or volume owner')
        volume = image['volumes'][node]
        require(snapshot['PendingOperations'] == 0 and volume['pending'] == []
                and same(snapshot, recovered_snapshots(image)[0]),
                'grouped read-only recovered snapshot differs from actual retained storage image')
        require(same(history[node], [observed_operation(operation) for operation in volume['history']]),
                'grouped actual snapshot history differs from decoded retained storage image')
        snapshots.append(dict(handle=item['handle'], snapshot=snapshot, state_bytes=len(state), log_bytes=len(log), image_bytes=len(image_raw),
                              storage_state_sha256=digest(state), storage_log_sha256=digest(log), storage_image_sha256=digest(image_raw)))
    require([item['snapshot']['ID'] for item in snapshots] == ['a', 'b'],
            'grouped latest actual snapshots do not contain the original two distinct volumes')
    return snapshots


def retained_oracle_report(root, configuration, artifact, observed, outputs):
    file = 'evidence/cluster_oracles'
    if root in (PLAIN, COMPRESSED, GROUPED):
        require(configuration.get('oracle_plan') is None and observed is None
                and all(file not in mapping for mapping in (artifact['payloads'], artifact['manifest']['files'], artifact['record']['streams'])),
                'storage guest manufactured an oracle report, plan or retained oracle payload')
        outcome = artifact['record']['outcome']
        require(outcome['class'] == 'success' and outcome['termination'] == 'exit' and type(outcome['exit_code']) is int and outcome['exit_code'] == 0,
                'storage guest without oracle evidence did not actually succeed')
        return None
    if root == SOFT:
        policy = next((item['report_only'] for item in (configuration.get('oracle_plan') or {}).get('soft_assertions', [])), False)
        expected = report([soft_oracle(owner, outputs[owner]['stdout'], outputs[owner]['stderr'], policy) for owner in owners(configuration)])
        require(len(expected['oracles']) == 2 and all(b64(item['evidence'][1]['value']) == b'["invalid fixture state"]' for item in expected['oracles']),
                'actual original soft assertion message changed')
    else:
        require(root == CANARY, 'foreign actual guest oracle owner')
        expected = history_report(outputs, configuration['incarnations'][1]['arguments'][-1] == 'canary')
    require(same(observed, expected) and file in artifact['payloads']
            and same(decode(artifact['payloads'][file]), expected), 'oracle report does not rederive from actual retained stdout/stderr')
    return expected


def validate_case(row, artifact=None):
    fields(row,('schema','original_root','case','inputs','observed'))
    require(row['schema']=='gomad.retained-reader-observation/v1' and row['case'] in CASES, 'foreign actual guest observation')
    case=row['case'];root,family,seed,policy,maximum=CASES[case]
    require(row['original_root']==root, 'cross-labeled actual original Go root')
    inputs,o=row['inputs'],row['observed']
    fields(inputs,('original_commit','sources','module_sha256','toolchain_sha256','prepared_provenance_sha256','configuration'))
    require(inputs['original_commit']=='1d695061b6382009d0659238c652a5a765b1d5f5', 'foreign original Go source revision')
    selected=[key for key in PINS if ('softassert' in key if root==SOFT else 'service/main.go' in key or root==CAPACITY and 'output_capacity' in key)]
    require(inputs['sources']==[dict(path=key,sha256=PINS[key]) for key in selected], 'original source pin roster changed')
    c=inputs['configuration'];configuration(case,c)
    if root==CAPACITY:
        fields(o,('outcome','stdout_base64','choice_bytes','assessment','publication_error'))
        outcome=o['outcome'];stdout=b64(o['stdout_base64']);assessment=o['assessment']
        require(outcome['termination']=='capacity' and outcome['message'] in ('output capacity','WASI capacity: offered-output-bytes') and stdout==b'PASS'
                and uint(outcome['output_bytes'])==4 and uint(o['choice_bytes'])==0, 'actual capacity guest did not exercise accepted PASS4/missing-choice')
        expected=report([soft_oracle('a/1',stdout,b'',policy,False)])
        failure=assessment['capacity_failure']
        require(failure['termination']=='capacity' and failure['message']=='output capacity'
                and failure['soft_assertions']==dict(messages=[],report_only=policy,inconclusive=True), 'capacity annotation broadened authoritative failure')
        detail=failure['detail']
        if maximum==1:
            require(sum(len(b64(item['value'])) for item in expected['oracles'][0]['evidence'])>maximum
                    and assessment['oracles']==[] and assessment['oracle_error']=='capacity: resource=oracle_evidence_bytes'
                    and detail==assessment['oracle_error'], 'oracle capacity budget was hidden')
        else:require(same(assessment['oracles'],expected['oracles']) and assessment['oracle_error']=='' and detail=='choice trace is not complete', 'capacity evidence incorrectly proves completeness')
        require(o['publication_error']=='capacity: output capacity; '+detail, 'capacity retained publication refusal changed')
        return dict(original_root=root,case=case,offered_bytes=4,retained_replay_refused=True)
    fields(o, ('report','replay') if root==SOFT else ('producer','report','replay','latest_storage') if root==GROUPED else ('producer','report','replay'))
    require(artifact is not None, 'actual guest omitted retained raw artifact')
    replay=o['replay'];record=artifact['record'];retained=artifact['inputs']
    require(canonical(replay['manifest'])==canonical(artifact['manifest']) and replay['record_sha256']==digest(artifact['payloads']['record.json'])
            and canonical(replay['retained_inputs'])==canonical(retained) and canonical(retained['configuration'])==canonical(c)
            and record['seed']==seed and record['identity']['toolchain']==inputs['toolchain_sha256']
            and record['identity']['target']==inputs['module_sha256'], 'guest observation not bound to actual retained record')
    require(replay['original_module_absent'] is True and replay['installation_absent'] is True, 'source-less native replay retained original input paths')
    requested=root!=SOFT or seed==7
    if requested:
        status=0 if root in (PLAIN,COMPRESSED,GROUPED) or root==CANARY and case.endswith('false') or root==SOFT and policy else 1
        require(type(replay['public_replay_exit']) is int and replay['public_replay_exit']==status
                and replay['public_replay_output']==f'gomad: reproduced=true diagnostic=false result={"success" if status==0 else "target_failure"} choice-replay=exact\n', 'public replay did not reproduce actual stored success/failure exactly')
    else:require(replay['public_replay_exit'] is None and replay['public_replay_output']=='', 'seed19 invented a replay branch')
    outputs=artifact_outputs(artifact)
    require(sorted(outputs)==sorted(owners(c)), 'actual artifact lost or invented guest owners')
    for owner,item in outputs.items():
        if root==GROUPED and owner.endswith('/1'):
            require(item['terminal']['interrupted'] and b'committed durable-a' in item['stdout'], 'group crash preceded original acknowledgment')
        else:
            require(item['terminal']['termination']=='exit' and item['terminal']['exit_code']==0 and not item['terminal']['interrupted'], 'original actual Go guest did not complete')
        if root==SOFT:require(b'PASS' in item['stdout'], 'soft assertion fixture did not pass ordinary Go tests')
        if root in (PLAIN,COMPRESSED):require(b'fresh heap 1' in item['stdout'] and b'ordered bytes ordered-byte' in item['stdout'], 'independent lifecycle reused guest heap or reordered original stream')
        if root==GROUPED and owner.endswith('/2'):require(b'fresh heap 1\nrecovered durable-a' in item['stdout'], 'group recovery reused heap or lost durable volume')
    if root!=SOFT:
        producer=o['producer'];fields(producer,('activity','incarnations','executed_steps'))
        require(same(producer['executed_steps'],expected_steps(root)), 'actual native execution reordered/invented lifecycle actions')
        require(producer['activity']==dict(launched_stores=len(outputs),remote_callback_attempts=0), 'actual producer launched a dormant/foreign owner')
        require([item['handle'] for item in producer['incarnations']]==[item['handle'] for item in retained['retained']], 'producer owner order differs from actual retained replay')
        for item in producer['incarnations']:
            owner=item['handle']['node']+'/'+str(item['handle']['incarnation'])
            require(b64(item['stdout_base64'])==outputs[owner]['stdout'], 'actual producer stdout differs from retained replay input')
            for key,descriptor in item['evidence'].items():
                raw=artifact['payloads'][f'evidence/cluster/{owner}/{key}']
                require(descriptor==dict(bytes=len(raw),sha256=digest(raw)), 'actual producer evidence differs from retained raw stream')
    expected = retained_oracle_report(root, c, artifact, o['report'], outputs)
    wanted='success' if expected is None or expected['outcome']=='completed' else 'target_failure'
    require(record['outcome']['class']==wanted, 'stored outcome class differs from actual oracle truth')
    facts = dict(original_root=root,case=case,record_sha256=digest(artifact['payloads']['record.json']),failure_identity=expected['failure_identity'] if expected is not None else None,public_replay=requested)
    if root == GROUPED:
        facts['latest_volume_snapshots'] = grouped_snapshots(o['latest_storage'], artifact, c)
        facts['volume_snapshot_count'] = len(facts['latest_volume_snapshots'])
    return facts


def preparation(files, family_root, row, config, owner_pid):
    inputs=row['inputs'];family=CASES[row['case']][1]
    raw=files.read(family_root/'prepared-provenance.json',64<<20);value=decode(raw)
    require(raw==canonical(value) and digest(raw)==inputs['prepared_provenance_sha256']
            and value['schema']=='gomad.rust-wasm-provenance/v1'
            and value['build_key']==inputs['toolchain_sha256']==digest(canonical(value['inputs']))
            and value['module_sha256']==inputs['module_sha256'], 'actual guest preparation provenance changed')
    module=files.read(family_root/'prepared-module.wasm.unavailable')
    require(module.startswith(b'\0asm\x01\0\0\0') and digest(module)==inputs['module_sha256'], 'prepared original module escrow differs')
    require(not (family_root/'prepared-module.wasm').exists() and not (family_root/'installation.json').exists(), 'public replay can reopen original module/installation')
    installed=decode(files.read(family_root/'installation.json.unavailable',16<<20))
    original=files.read(family_root/'original-config.json',16<<20)
    require(original==canonical(decode(original)) and installed['admission']['guest_configuration']==decode(original), 'sealed actual guest configuration differs from original raw config')
    compiler=config['final_profiles']['stock_go']
    require(installed['compiler_path']==compiler and installed['compiler_sha256']==config['files'][compiler], 'actual guest compiler is not the frozen stock role')
    prepared=value['inputs']
    require(prepared['compiler_path']==compiler and prepared['compiler_sha256']==config['files'][compiler]
            and prepared['go_version']=='go1.27.1' and prepared['kind']==('go-test' if family in ('softassert','capacity') else 'go-run')
            and prepared['tags']==(['test_dep'] if family=='softassert' else []), 'actual guest preparation target/tags/compiler differs')
    for source in inputs['sources']:
        path=str((Path(config['repo'])/source['path']).resolve())
        require(prepared['sources'].get(path)==source['sha256'], 'actual Go compile omitted original pinned source')
    proof=family_root/'preparation-proof'
    for phase in ('compile','admission'):
        metadata_path=proof/f'target/preparation-inputs/{phase}/metadata.json'
        metadata=decode(files.read(metadata_path,16<<20))
        require(metadata['schema']=='gomad.native-preparation-observations/v1' and metadata['phase']==phase, 'actual Go preparation phase missing')
        for key,blob in metadata['blobs'].items():
            require(blob['path']==f'target/preparation-inputs/{phase}/{key}.bin' and blob['sha256']==key, 'actual Go preparation blob path substituted')
            raw=files.read(proof/blob['path'])
            require(uint(blob['bytes'])==len(raw) and digest(raw)==key, 'actual Go preparation raw blob changed')
        for command in metadata['commands']:
            require(type(command['exit']) is int and command['exit']==0 and command['owner_pid']==owner_pid
                    and uint(command['pid'])>0 and uint(command['end_ns'])>=uint(command['start_ns']), 'actual Go compiler command lacks successful owner/process custody')
            require(command['stdout'] in metadata['blobs'] and command['stderr'] in metadata['blobs'], 'actual Go command output raw bytes missing')
        if phase=='compile':
            commands=[command for command in metadata['commands'] if command['name']=='compile-module']
            require(len(commands)==1, 'actual fixture has no one successful original compiler invocation')
            argv=commands[0]['argv']
            require(argv[0]==compiler and argv[1:3]==(['test','-c'] if family in ('softassert','capacity') else ['build','-mod=readonly'])
                    and argv[-1]==prepared['source'] and '-trimpath' in argv and '-overlay' in argv,
                    'actual Go compilation command is not the original source target')
    return value


def parse_rows(stdout):
    rows=[];seen=set()
    for line in stdout.split('\n'):
        if MARKER not in line:continue
        row=decode(line.split(MARKER,1)[1])
        require(isinstance(row,dict) and row.get('case') in CASES and row['case'] not in seen, 'unknown or duplicate actual Go guest case')
        seen.add(row['case']);rows.append(row)
    require(seen==set(CASES), 'actual guest source root variants omitted: '+repr(sorted(set(CASES)-seen)))
    return rows


def observations(config, raw_harness):
    """Only admit artifacts enclosed by the fixed release host-lib raw command role."""
    import broad_harness
    require(isinstance(raw_harness,dict) and raw_harness.get('schema')=='gomad.broad-harness-raw/v1', 'guest reader needs sealed broad raw harness custody')
    stage=broad_harness.Stage(raw_harness['stage'])
    require(stage in (broad_harness.Stage.PROVIDER,broad_harness.Stage.RETAINED), 'guest reader called from foreign broad stage')
    sealed=broad_harness.read(config,stage)
    require(canonical(sealed)==canonical(raw_harness), 'guest reader borrowed a foreign harness receipt')
    role=sealed['roles']['host-broad-guest']
    require(role['role']=='host-lib' and role['features']==['determinism-fixture'] and role['ignored'] is True
            and role['argv']==broad_harness.test_argv(config,stage,'host-broad-guest'), 'guest reader substituted the admitted executable/selection')
    source_facts={}
    for key in (SOURCE,*PINS):
        path=pinned(config,(Path(config['repo'])/key).resolve())
        source_facts[key]=digest(path.read_bytes())
        if key in PINS:require(source_facts[key]==PINS[key], 'original immutable Go fixture changed')
    root=broad_harness.stage_root(config,stage)/'guest-root'
    files=RawFiles(broad_harness.stage_root(config,stage))
    stdout=files.read(Path(role['stdout']),512<<20).decode('utf8')
    rows=parse_rows(stdout);facts=[]
    for row in rows:
        case=row['case'];_,family,_,_,_=CASES[case];family_root=root/family
        on_disk=files.read(family_root/(case.replace('/','-')+'.json'),64<<20)
        require(on_disk==canonical(row), 'printed actual guest observation differs from raw retained observation file')
        preparation(files,family_root,row,config,role['pid'])
        actual=None
        if row['original_root']!=CAPACITY:
            path=Path(row['observed']['replay']['artifact'])
            require(path.parent==family_root/case and path.name and path.resolve()==path, 'guest artifact escaped exact original case root')
            actual=read_artifact(files,path)
        facts.append(validate_case(row,actual))
    signatures=[fact['failure_identity'] for fact in facts if fact['case'] in ('soft/seed=7/report-only=false','soft/seed=19/report-only=false')]
    require(len(signatures)==2 and signatures[0]==signatures[1] and signatures[0], 'original seed7/19 static soft assertion failure signature differs')
    custody=files.finish()
    complete={SOFT,CAPACITY,CANARY,GROUPED}
    remaining={PLAIN:['Original retained mutation/refusal readers and module independence/nonembedding bound require the separate core_retained receiver.'],
               COMPRESSED:['Original retained mutation/refusal readers and module independence/nonembedding bound require the separate core_retained receiver.']}
    return dict(roots={key:dict(cases=[fact for fact in facts if fact['original_root']==key],sources=source_facts) for key in sorted(complete)},
                partial_roots={key:dict(cases=[fact for fact in facts if fact['original_root']==key],remaining_original_predicates=value) for key,value in remaining.items()},
                facts=facts,sources=source_facts,raw_custody=custody,process=role)
