"""Same-artifact retention receipts for the original plain/compressed Go lifecycle.

The removed Go installation/callback wire forms remain explicit original gaps.
"""
import copy
from pathlib import Path

from driver import require
from native_roles import pinned
from native_observations import canonical, digest
from core_codec import decoded, raw_bytes
import core_guests
import core_retained_matrix as matrix

FILE=matrix.FILE
MARKER='GOMAD_GUEST_RETENTION '
SCHEMA='gomad.guest-retention-matrix/v1'
SOURCE='tools/gomad/rust/gomad/src/replay/cluster/retained_lifecycle_matrix_tests.rs'
ROOTS={'lifecycle/compressed=false':core_guests.PLAIN,'lifecycle/compressed=true':core_guests.COMPRESSED}
VARIANTS=('valid',*matrix.MUTATIONS,'foreign-linked-helper','primary-empty','primary-foreign','module-independent','source-less-replay')


def expected_mutation(before,case):
    if case not in ('changed-imports','changed-runtime-binding','foreign-linked-helper','primary-empty','primary-foreign'):
        return matrix.expected_mutation(before,case)
    payloads=dict(before['payloads']);record=copy.deepcopy(before['record'])
    if case in ('primary-empty','primary-foreign'):
        record['identity']['target']='' if case=='primary-empty' else '0'*64
        record['execution_identity']=digest(canonical(record['identity']))
        payloads['record.json']=canonical(record)
        manifest=copy.deepcopy(before['manifest']);manifest['record']=record;manifest['record_sha256']=digest(payloads['record.json'])
        manifest['files']['record.json'].update(bytes=len(payloads['record.json']),sha256=digest(payloads['record.json']))
        return payloads,manifest
    inputs=decoded(payloads[FILE]);item=inputs['configuration']['incarnations'][0]
    if case=='changed-imports':item['imports'][0]['name']+='-changed'
    elif case=='changed-runtime-binding':item['choice_identity']['ToolchainBuildKey']+='-changed'
    else:inputs['helper_sha256']='f'*64
    payloads[FILE]=canonical(inputs)
    return matrix.reseal(payloads,record,before['manifest'])


def artifact(value,original):
    matrix.fields(value,{'manifest_base64','record_base64','payloads_base64','module_preimages'},'foreign same-artifact retained preimage')
    manifest_raw=matrix.b64(value['manifest_base64']);manifest=decoded(manifest_raw)
    matrix.fields(manifest,{'schema','files','record','record_sha256'},'foreign retained manifest fields')
    require(manifest['schema']=='gomad.artifact-manifest/v2' and manifest_raw==canonical(manifest),'noncanonical/foreign retained manifest')
    payloads={name:matrix.b64(raw) for name,raw in value['payloads_base64'].items()}
    for name,reference in value['module_preimages'].items():
        require(name=='target/module.wasm' and name not in payloads,'unexpected original guest module reference')
        raw=original['payloads'][name]
        require(reference==dict(bytes=len(raw),sha256=digest(raw),source='original-artifact'),'retained mutant substituted its actual module preimage')
        payloads[name]=raw
    require(set(payloads)==set(manifest['files']) and payloads.get('record.json')==matrix.b64(value['record_base64']),'retained mutant payload roster changed')
    for name,raw in payloads.items():
        matrix.safe_name(name);descriptor=manifest['files'][name]
        require(type(descriptor.get('bytes')) is int and descriptor==dict(bytes=len(raw),sha256=digest(raw),module=name=='target/module.wasm'),
            'retained mutant descriptor differs from actual raw preimage: '+name)
    record=decoded(payloads['record.json'])
    # Mutated record spellings/identities are deliberately authenticated below against their exact intended delta.
    return dict(manifest=manifest,record=record,payloads=payloads)


def unavailable(value,before):
    matrix.fields(value,{'artifact','read_inputs_base64','bound_bytes','original_module_absent','original_installation_absent'},'foreign module independence receipt')
    after=artifact(value['artifact'],before)
    expected={name:raw for name,raw in before['payloads'].items() if name!='target/module.wasm'}
    manifest=copy.deepcopy(before['manifest']);manifest['files'].pop('target/module.wasm')
    require(after['payloads']==expected and after['manifest']==manifest and not value['artifact']['module_preimages'],
        'metadata independence changed a nonmodule byte or retained hidden module preimages')
    raw=before['payloads'][FILE]
    bound=len(before['payloads']['record.json'])*4//3+(64<<10)
    require(matrix.b64(value['read_inputs_base64'])==raw and type(value['bound_bytes']) is int
            and len(raw)<=bound==value['bound_bytes'],'same-artifact retained metadata embeds module or exceeds original ratio bound')
    require(value['original_module_absent'] is True and value['original_installation_absent'] is True,'retained replay regained original producer input paths')
    return dict(retained_metadata_bytes=len(raw),original_record_bytes=len(before['payloads']['record.json']),bound_bytes=bound,
        removed_module_bytes=len(before['payloads']['target/module.wasm']))


def replay(value,before,root):
    matrix.fields(value,{'original_record_base64','summaries','executed_steps_base64','outcome','projected_evidence','divergence','entries_before','entries_after'},'foreign actual lifecycle replay receipt')
    require(matrix.b64(value['original_record_base64'])==before['payloads']['record.json'],'actual replay lost original canonical record digest preimage')
    steps=decoded(matrix.b64(value['executed_steps_base64']))
    require(steps==core_guests.expected_steps(root)==before['inputs']['configuration']['steps'],'retained actual lifecycle changed original eight actions')
    record=before['record'];outcome=value['outcome'];projected={name:raw_bytes(raw) for name,raw in value['projected_evidence'].items()}
    require(value['divergence'] is None and outcome['termination']==record['outcome']['termination']=='exit'
            and type(outcome.get('exit_code')) is int and outcome['exit_code']==record['outcome']['exit_code']==0
            and outcome['message']==record['outcome']['reason'] and outcome.get('divergence') is None,'actual same-artifact retained replay outcome diverged')
    require(record['virtual_time_nanos']==outcome.get('local_model',{}).get('now',0),'same-artifact retained replay clock diverged')
    for name,stream in record['streams'].items():
        if name==FILE:continue
        prefix='output/' if name.startswith('output/') else 'evidence/'
        require(name.startswith(prefix),'unknown original lifecycle stream namespace')
        key=name[len(prefix):]
        require(key in projected and projected[key]==before['payloads'][name] and stream['sha256']==digest(projected[key]),
            'retained actual execution changed original canonical stream: '+name)
    originals=core_guests.artifact_outputs(before);summaries=value['summaries']
    require(isinstance(summaries,list) and len(summaries)==len(originals)==4,'actual replay did not return four terminal incarnation summaries')
    seen=set()
    for item in summaries:
        matrix.fields(item,{'handle','interrupted','outcome','evidence'},'foreign actual replay incarnation summary')
        owner=item['handle']['node']+'/'+str(item['handle']['incarnation'])
        require(owner in originals and owner not in seen and item['interrupted'] is False,'duplicate/foreign/interrupted actual replay summary')
        terminal=item['outcome'];require(terminal['termination']=='exit' and type(terminal.get('exit_code')) is int and terminal['exit_code']==0,'actual replay incarnation failed')
        original_streams=next(original['evidence'] for original in before['inputs']['retained'] if original['handle']==item['handle'])
        require(set(item['evidence'])==set(original_streams),'retained incarnation summary omitted raw evidence')
        for name,descriptor in original_streams.items():
            require(raw_bytes(item['evidence'][name])==before['payloads'][descriptor['file']],'actual replay summary differs from retained original bytes')
        stdout=raw_bytes(item['evidence']['stdout'])
        require(b'fresh heap 1' in stdout and b'ordered bytes ordered-byte' in stdout,'actual replay reused heap or changed ordered network bytes')
        seen.add(owner)
    require(value['entries_before']==value['entries_after'],'actual source-less replay changed original retained files')
    for name,raw in {**before['payloads'],'manifest.json':canonical(before['manifest']),'storage.json':b'{"format":"gomad.artifact-storage/v2"}'}.items():
        require(value['entries_before'].get(name)==dict(bytes=len(raw),sha256=digest(raw)),'source-less replay directory receipt omitted actual original file: '+name)
    return dict(original_record_sha256=digest(before['payloads']['record.json']),terminal_handles=sorted(seen),canonical_streams=sorted(record['streams']))


def receipts(stdout,originals):
    rows=[decoded(line.split(MARKER,1)[1]) for line in stdout.split('\n') if MARKER in line]
    expected=[(case,variant) for case in ROOTS for variant in VARIANTS]
    require([(row.get('case'),row.get('variant')) for row in rows]==expected,'same-artifact guest retention matrix omitted, duplicated or reordered an original subvariant')
    require(set(originals)==set(ROOTS),'same-artifact retention omitted actual plain/compressed guest artifacts')
    facts={case:{} for case in ROOTS}
    for row in rows:
        matrix.fields(row,{'schema','original_root','case','variant','original_artifact','record_sha256','module_sha256','observed'},'foreign original guest retention receipt')
        case=row['case'];root=ROOTS[case];variant=row['variant'];before=originals[case]['artifact'];value=row['observed']
        require(row['schema']==SCHEMA and row['original_root']==root and row['original_artifact']==originals[case]['path']
                and row['record_sha256']==digest(before['payloads']['record.json'])
                and row['module_sha256']==digest(before['payloads']['target/module.wasm']),'guest retention receipt borrowed another module/profile/record/artifact')
        if variant=='module-independent':fact=unavailable(value,before)
        elif variant=='source-less-replay':fact=replay(value,before,root)
        else:
            matrix.fields(value,{'artifact','boundaries'},'foreign guest retention mutation receipt')
            after=artifact(value['artifact'],before)
            payloads,manifest=(before['payloads'],before['manifest']) if variant=='valid' else expected_mutation(before,variant)
            require(payloads==after['payloads'] and manifest==after['manifest'],'retained guest did not execute exact original same-artifact mutation: '+variant)
            fact=matrix.boundary_facts(value['boundaries'],admitted=variant=='valid',actual=after)
        facts[case][variant]=fact
    return facts


REMOVED_FORMS=[
    'Original external InstalledHelper caller path/bytes and diagnostic naming both installed and retained helper digests are removed; native same-artifact compile-linked-helper rejection is verified separately.',
    'Original sha256:-prefixed caller RetainedModule identity is removed; native same-artifact empty/foreign manifest primary identity rejection is verified separately.',
    'Original gomadsim.Run without an injected engine/callback refusal API has no current native receiving route; this reader does not invent that callback branch.',
]


def observations(config,raw_harness):
    import broad_harness
    guests=core_guests.observations(config,raw_harness)
    stage=broad_harness.Stage(raw_harness['stage']);sealed=broad_harness.read(config,stage)
    require(canonical(sealed)==canonical(raw_harness),'guest retention borrowed a foreign raw process receipt')
    source=pinned(config,(Path(config['repo'])/SOURCE).resolve())
    role=sealed['roles']['host-broad-guest'];files=core_guests.RawFiles(broad_harness.stage_root(config,stage))
    stdout=files.read(Path(role['stdout']),512<<20).decode('utf8')
    guest_rows=core_guests.parse_rows(stdout);originals={}
    for row in guest_rows:
        if row['case'] not in ROOTS:continue
        path=row['observed']['replay']['artifact']
        originals[row['case']]=dict(path=path,artifact=core_guests.read_artifact(files,Path(path)))
    facts=receipts(stdout,originals);custody=files.finish()
    return dict(roots={},source_closed_receiving={ROOTS[case]:dict(cases=value,native_receiving_complete=True) for case,value in facts.items()},
        partial_roots={ROOTS[case]:dict(cases=value,remaining_original_predicates=REMOVED_FORMS) for case,value in facts.items()},
        facts=facts,source_sha256=digest(source.read_bytes()),raw_custody=custody,
        dependencies={root:REMOVED_FORMS for root in ROOTS.values()},
        lineage='Native canonical manifest/configuration and binary World receipts; no retired Go JSON byte equivalence.',
        dependency=dict(actual_original_guests=guests['facts'],same_artifact_native_receivers=True,
            qualification='source-only until the updated original ignored guest selector is executed on the frozen candidate'))
