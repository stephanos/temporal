"""Independent closed native retained-input readers, with explicit original-root gaps."""
import base64
import copy
import json
import re

from driver import require, Refusal
from native_observations import canonical, digest
from core_codec import decoded, raw_bytes, unsigned

SCHEMA='gomad.cluster-replay-inputs/v1'
ROW_SCHEMA='gomad.retained-reader-observation/v1'
MARKER='GOMAD_RETAINED_OBSERVATION '
FILE='evidence/cluster-replay.json'
DECODE_CASES=('duplicate-top','duplicate-nested','unknown-top','unknown-nested','duplicate-model-value',
    'duplicate-environment-map','excessive-nesting','oversized','missing-module','wrong-module-type','foreign-schema',
    'legacy-schema','trailing-value','truncated','invalid-utf8','host-faults','host-scenarios','host-history',
    'host-observations','host-oracles','go-served-calls','guest-callbacks')
BIND_CASES=('helper','host','runtime','toolchain','engine','profile','primary-path','primary-digest',
    'missing-incarnations','missing-retained','foreign-retained','duplicate-incarnation')
INPUT_FIELDS={'schema','helper_sha256','host_sha256','runtime_implementation_sha256','toolchain_sha256','configuration','retained'}


def b64(value,maximum=64<<20):
    require(isinstance(value,str) and len(value)<=(maximum+2)//3*4,'retained raw base64 bound/type')
    try:raw=base64.b64decode(value,validate=True)
    except (ValueError,TypeError):require(False,'malformed retained base64')
    require(len(raw)<=maximum and base64.b64encode(raw).decode()==value,'noncanonical retained base64')
    return raw


def same(left,right):return canonical(left)==canonical(right)


def fields(value,names):require(isinstance(value,dict) and set(value)==set(names),'foreign retained observation fields')


def error(value,expected=None):
    require(isinstance(value,str) and bool(value),'native refusal missing')
    if expected is not None:require(expected in value,'native refusal reached wrong independently derived guard')


def decoder_mutation(raw,case):
    value=decoded(raw)
    if case=='duplicate-top':return b'{"schema":"duplicate",'+raw[1:]
    if case=='duplicate-nested':return raw.replace(b'"configuration":{',b'"configuration":{"engine":"duplicate",',1)
    if case=='duplicate-model-value':return raw.replace(b'"model":{',b'"model":{"native_value":1,"native_value":2,',1)
    if case=='duplicate-environment-map':return raw.replace(b'"environment":{',b'"environment":{"PWD":"/first","PWD":"/second",',1)
    if case=='excessive-nesting':return raw.replace(b'"model":{}',b'"model":'+b'['*129+b'0'+b']'*129,1)
    if case=='oversized':return b' '*((16<<20)+1)
    if case=='trailing-value':return raw+b'{}'
    if case=='truncated':return raw[:-1]
    if case=='invalid-utf8':return raw[:2]+b'\xff'+raw[3:]
    if case in ('foreign-schema','legacy-schema'):value['schema']='gomad.other/v1' if case=='foreign-schema' else 'gomad.wasm-simulation-retained/v1'
    elif case=='unknown-top':value['unknown']=True
    elif case=='unknown-nested':value['configuration']['unknown']=True
    elif case=='missing-module':value['configuration']['incarnations'][0].pop('module')
    elif case=='wrong-module-type':value['configuration']['incarnations'][0]['module']=1
    else:
        name={'host-faults':'faults','host-scenarios':'scenarios','host-history':'history','host-observations':'observations',
            'host-oracles':'oracles','go-served-calls':'go_filesystem_calls','guest-callbacks':'guest_callbacks'}.get(case)
        require(name is not None,'unknown retained decoder mutation');value['configuration'][name]=[{}]
    return canonical(value)


def binding_mutation(baseline,case):
    value=copy.deepcopy(baseline);configuration=value['configuration']
    replacements={'helper':('helper_sha256',b'resealed helper'),'host':('host_sha256',b'foreign host'),
        'runtime':('runtime_implementation_sha256',b'foreign runtime'),'toolchain':('toolchain_sha256',b'foreign toolchain')}
    if case in replacements:
        name,raw=replacements[case];value[name]=digest(raw)
    elif case=='engine':configuration['engine']+='-changed'
    elif case=='profile':configuration['profile']='foreign'
    elif case=='primary-path':configuration['incarnations'][0]['module']['file']='target/foreign.wasm'
    elif case=='primary-digest':configuration['incarnations'][0]['module']['sha256']=digest(b'foreign module')
    elif case=='missing-incarnations':configuration['incarnations']=[]
    elif case=='missing-retained':value['retained']=[]
    elif case=='foreign-retained':value['retained'][0]['handle']['node']='foreign'
    elif case=='duplicate-incarnation':configuration['incarnations'].append(copy.deepcopy(configuration['incarnations'][0]))
    else:require(False,'unknown retained binding mutation')
    return value


def history_cases():
    good=b'{"cluster_id":"cluster","history":"YQ==","event_ids":[1],"types":["started"]}'
    return {'missing':b'','duplicate-lines':b'LIVE '+good+b'\nLIVE '+good,
        'duplicate-key':b'LIVE '+good[:-1]+b',"history":"Yg=="}',
        'unknown-key':b'LIVE '+good[:-1]+b',"unknown":true}', 'incomplete':b'LIVE {}',
        'trailing-value':b'LIVE '+good+b' {}',
        'invalid-utf8':b'LIVE {"cluster_id":"cluster","history":"YQ==","event_ids":[1],"types":["\xff"]}',
        'valid':b'LIVE '+good}


def history(raw):
    matches=[line[5:] for line in raw.split(b'\n') if line.startswith(b'LIVE ')]
    require(len(matches)==1 and len(matches[0])<=1<<20,'ambiguous/missing/oversized raw history')
    value=decoded(matches[0]);fields(value,{'cluster_id','history','event_ids','types'})
    require(isinstance(value['cluster_id'],str) and value['cluster_id'] and b64(value['history']) and isinstance(value['event_ids'],list)
            and value['event_ids'] and all(type(identity) is int and 0<identity<1<<63 for identity in value['event_ids'])
            and isinstance(value['types'],list) and len(value['types'])==len(value['event_ids'])
            and all(isinstance(name,str) and name for name in value['types']),'incomplete history')
    return value


def oracle(name,passed,pairs,soft=False,report_only=False,inconclusive=False):
    evidence=[dict(label=label,value=base64.b64encode(raw).decode(),full_sha256=digest(raw)) for label,raw in pairs]
    value=dict(name=name,passed=passed,inconclusive=inconclusive,soft_assertion=soft,report_only=report_only,
        evidence=evidence,failure_identity='',identity='')
    if not passed:value['failure_identity']=digest(canonical(dict(name=name,evidence=evidence)))
    value['identity']=digest(canonical(value))
    return value


def valid_history_report():
    operation=[dict(id='committed',actor='workflow',kind='workflow-history',invocation=1,completion=2,output='YQ==')]
    oracles=[oracle('live-history.history',True,[('actual',canonical(operation)),('expected',canonical(operation))]),
        oracle('live-history.events',False,[('actual',canonical(['1'])),('expected',canonical(['1','2']))]),
        oracle('live-history.identity',True,[('actual',b'cluster'),('expected',b'cluster')])]
    return dict(schema='gomad.cluster-oracles/v1',outcome='oracle_failed',reason='oracle_failed',
        failure_identity=digest(canonical([oracles[1]])),oracles=oracles)


def directory_unchanged(observed):
    before,after=observed['directory_before'],observed['directory_after']
    require(isinstance(before,dict) and before and same(before,after),'retained admission changed actual directory inventory')
    for path,entry in before.items():
        require(isinstance(path,str) and path and all(part not in ('','..','.') for part in path.split('/')),'unsafe observed retained path')
        if entry['kind']=='directory':fields(entry,{'kind'})
        elif entry['kind']=='file':
            fields(entry,{'kind','bytes','sha256'});unsigned(entry['bytes']);require(re.fullmatch('[0-9a-f]{64}',entry['sha256']),'invalid directory file commitment')
        else:require(False,'unexpected retained directory alias/type')
    top=sum('/' not in path for path in before)
    require(type(observed['entries_before']) is int and type(observed['entries_after']) is int
            and observed['entries_before']==observed['entries_after']==top,'directory counters do not match actual raw roster')
    if 'compiled_cache_exists' in observed:require(observed['compiled_cache_exists'] is False and not any(path.split('/')[0]=='compiled-cache' for path in before),'admission created compiled cache')


def metadata(raw):
    value=decoded(raw);fields(value,INPUT_FIELDS)
    require(value['schema']==SCHEMA and raw==canonical(value),'native retained baseline is not typed canonical current lineage')
    for name in ('helper_sha256','host_sha256','runtime_implementation_sha256','toolchain_sha256'):
        require(re.fullmatch('[0-9a-f]{64}',value[name]),'retained baseline identity shape')
    configuration=value['configuration']
    allowed={'original_limits','oracle_plan','workload','simulation_original','nodes','engine','profile','world_encoding','world_profile','limits','wall_nanos','broker','network','incarnations','steps','boundaries','crash_groups'}
    require(set(configuration)<=allowed and all(name in configuration for name in ('engine','profile','limits','broker','network','incarnations','steps','boundaries','crash_groups')),'foreign native configuration fields')
    require(configuration['profile']=='gomad.wasi-cooperative/v1' and len(configuration['incarnations'])==len(value['retained'])==1
            and 'workload' not in configuration and 'oracle_plan' not in configuration,'retained metadata baseline changed')
    item=configuration['incarnations'][0]
    allowed={'workload_kind','handle','module','imports','storage_crash_selection','limits','wall_nanos','arguments','environment','runtime_sha256','model','namespace','storage','network','choice_identity'}
    require(set(item)<=allowed and isinstance(item['module'],dict),'foreign native incarnation fields')
    require(item['handle']=={'node':'sqlite','incarnation':2} and item['model']=={} and item['environment']=={}
            and item['module']==dict(file='target/module.wasm',bytes=8,sha256=digest(b'\0asm\1\0\0\0'),limit=512<<20),'native metadata primary module fixture changed')
    require(value['toolchain_sha256']==digest(b'retained-reader-test-toolchain') and item['choice_identity']['ToolchainBuildKey']==value['toolchain_sha256']
            and item['choice_identity']['TargetSHA256']==list(bytes.fromhex(item['module']['sha256'])),'metadata native module/toolchain choice identity not bound')
    require(same(value['retained'],[dict(handle=item['handle'],interrupted=False,evidence={})]),'metadata retained handle roster changed')
    return value


def decode_row(case,row,raw):
    fields(row['inputs'],{'payload_base64'});actual=b64(row['inputs']['payload_base64'],17<<20)
    observed=row['observed']
    if case=='native-valid':
        fields(observed,{'reencoded_base64'});require(actual==raw==b64(observed['reencoded_base64']),'native typed valid decode/reencode differs')
        return
    require(case in DECODE_CASES and actual==decoder_mutation(raw,case),'retained decoder mutation lacks exact original raw variant')
    fields(observed,{'error','prepare_error','public_status','public_error','entries_before','entries_after','directory_before','directory_after'})
    expected={'foreign-schema':'unsupported retained cluster replay input schema','legacy-schema':'unsupported retained cluster replay input schema',
        'duplicate-top':'duplicate object key','duplicate-nested':'duplicate object key','duplicate-model-value':'duplicate object key','duplicate-environment-map':'duplicate object key',
        'excessive-nesting':'nesting','oversized':'artifact payload exceeds caller bound','unknown-top':'unknown field `unknown`','unknown-nested':'unknown field `unknown`',
        'missing-module':'missing field `module`','wrong-module-type':'invalid type: integer','trailing-value':'trailing data','truncated':'invalid syntax','invalid-utf8':'unicode'}
    unknown={'host-faults':'faults','host-scenarios':'scenarios','host-history':'history','host-observations':'observations','host-oracles':'oracles','go-served-calls':'go_filesystem_calls','guest-callbacks':'guest_callbacks'}
    fragment=expected[case] if case in expected else 'unknown field `'+unknown[case]+'`'
    error(observed['error'],fragment)
    require(observed['prepare_error']==observed['error'] and type(observed['public_status']) is int and observed['public_status']==2
        and observed['error'] in observed['public_error'],'strict retained decoder/prepare/public CLI refusal disagrees')
    directory_unchanged(observed)


def binding_row(case,row,baseline):
    fields(row['inputs'],{'retained_inputs','record'});inputs=row['inputs']['retained_inputs'];record=row['inputs']['record']
    require(case in BIND_CASES and same(inputs,binding_mutation(baseline,case)),'retained binding case is not exact original resealed mutation')
    require(record['schema']=='gomad.execution-record/v2' and record['identity']['configuration']==digest(canonical(inputs['configuration']))
            and record['execution_identity']==digest(canonical(record['identity'])),'binding mutation was not coherently resealed through config/execution identities')
    require(record['identity']['target']==digest(b'\0asm\1\0\0\0') and record['identity']['toolchain']==baseline['toolchain_sha256']
            and record['profile']==baseline['configuration']['profile'] and same(record['limits'],baseline['configuration']['limits'])
            and record['seed']==baseline['configuration']['broker']['seed'],'binding original record fixture changed')
    fields(row['observed'],{'error','prepare_error','entries_before','entries_after','compiled_cache_exists','directory_before','directory_after'})
    expected=('retained cluster primary module identity mismatch' if case in ('primary-path','primary-digest') else
        'retained cluster incarnation count mismatch' if case in ('missing-incarnations','missing-retained','duplicate-incarnation') else
        'retained cluster handle mismatch' if case=='foreign-retained' else 'WASM retained cluster implementation/configuration identity mismatch')
    require(row['observed']['error']==row['observed']['prepare_error']==expected,'retained bind first refusal differs from independent identity/roster mismatch')
    directory_unchanged(row['observed'])


def policy_row(case,row):
    fields(row['observed'],{'error'})
    if case in ('','go-run','go-test','unknown'):
        require(row['inputs']==dict(workload_kind=case),'recorded workload policy input changed')
        want=None if case=='go-test' else 'invalid recorded cluster workload kind' if case=='unknown' else 'soft assertion policy requires a recorded Go-test workload'
    else:
        require(case in ('unknown-handle','duplicate-policy'),'unknown workload policy case');fields(row['inputs'],{'oracle_plan'})
        policy=dict(handle=dict(node='foreign' if case=='unknown-handle' else 'sqlite',incarnation=2),report_only=True)
        plan=dict(schema='gomad.wasm.soft-assertion-observation/v1',prefix='',maximum_bytes=4096,histories=[],convergence=[],soft_assertions=[policy]*(2 if case=='duplicate-policy' else 1))
        require(same(row['inputs']['oracle_plan'],plan),'soft policy mutation does not preserve exact original handle/duplicate variant')
        want='soft assertion policy requires a recorded Go-test workload'
    require(row['observed']==dict(error=want),'native opt-out did not independently validate typed Go-test workload policy')


def capacity_row(case,row):
    match=re.fullmatch(r'(infrastructure|capacity|exit)/(.*)/report-only=(false|true)/maximum=(1|4096)',case)
    require(match is not None,'unknown retained capacity case');termination,message,flag,maximum=match.groups();report_only=flag=='true';maximum=int(maximum)
    require((termination,message) in (('infrastructure','output capacity'),('capacity','transcript capacity'),('exit',''),('capacity','output capacity')),'unknown narrow capacity terminal/message')
    inputs=row['inputs'];fields(inputs,{'handle','stdout_base64','stderr_base64','termination','message','offered_bytes','oracle_plan'})
    stdout,stderr=b64(inputs['stdout_base64']),b64(inputs['stderr_base64'])
    handle=dict(node='sqlite',incarnation=2)
    plan=dict(schema='gomad.wasm.soft-assertion-observation/v1',prefix='',maximum_bytes=maximum,histories=[],convergence=[],soft_assertions=[dict(handle=handle,report_only=report_only)])
    require(inputs['handle']==handle and stdout==b'PASS' and stderr==b'failed assertion: invalid {state}' and inputs['termination']==termination and inputs['message']==message
            and type(inputs['offered_bytes']) is int and inputs['offered_bytes']==len(stdout)+len(stderr) and same(inputs['oracle_plan'],plan),'capacity raw output/counters/policy fixture changed')
    complete=termination!='capacity';instance=b'sqlite/2'
    assessed=oracle('soft-assertion.instance-'+digest(instance),complete and report_only,
        [('instance',instance),('messages',canonical(['invalid {state}'])),('status',b'observed' if complete else b'inconclusive')],
        soft=True,report_only=report_only,inconclusive=not complete)
    expected=dict(oracles=[] if maximum==1 else [assessed],oracle_error='capacity: resource=oracle_evidence_bytes' if maximum==1 else '')
    if termination=='capacity' and message=='output capacity':
        expected['capacity_failure']=dict(termination='capacity',message='output capacity',soft_assertions=dict(messages=['invalid {state}'],report_only=report_only,inconclusive=True),
            detail='capacity: resource=oracle_evidence_bytes' if maximum==1 else 'choice trace is not complete')
    require(same(row['observed'],expected),'capacity annotation or oracle budget differs from raw observed literal message/completeness')


def authentication_row(case,row,raw):
    fields(row['observed'],{'error'})
    if case=='raw-bytes':
        fields(row['inputs'],{'original_base64','changed_base64'})
        require(b64(row['inputs']['original_base64'])==raw and b64(row['inputs']['changed_base64'])==raw+b'\n','retained raw authentication mutation changed')
        error(row['observed']['error'],'artifact file exceeds bound')
        return
    fields(row['inputs'],{'payload_base64','descriptor'})
    descriptor=dict(file=FILE,bytes=len(raw),sha256=digest(raw),limit=16<<20)
    if case=='digest':descriptor['sha256']=digest(b'foreign bytes')
    elif case=='bytes':descriptor['bytes']+=1
    elif case=='path':descriptor['file']='evidence/foreign.json'
    elif case=='limit':descriptor['limit']=32<<20
    else:require(False,'unknown raw authentication case')
    require(b64(row['inputs']['payload_base64'])==raw and same(row['inputs']['descriptor'],descriptor),'raw descriptor mutation differs from authenticated native baseline')
    error(row['observed']['error'], 'missing retained cluster payload' if case=='path' else
        'cluster payload exceeds caller bound or has invalid identity' if case=='limit' else 'retained cluster payload identity mismatch')


def module_row(case,row):
    fields(row['inputs'],{'original_base64','file','descriptor','source_kind','source_bytes_base64','link_target'})
    fields(row['observed'],{'error','compiled_cache_exists'})
    inputs=row['inputs'];raw=b64(inputs['original_base64']);require(raw==b'\0asm\1\0\0\0' and inputs['file']=='target/module.wasm'
        and same(inputs['descriptor'],dict(bytes=8,sha256=digest(raw),module=True)),'module original bytes/descriptor changed')
    if case in ('tampered-bytes','truncated'):
        require(inputs['source_kind']=='file' and b64(inputs['source_bytes_base64'])==(b'foreign!' if case=='tampered-bytes' else raw[:-1]) and inputs['link_target'] is None,'module raw changed bytes missing')
        error(row['observed']['error'],'artifact payload identity mismatch' if case=='tampered-bytes' else 'artifact payload size mismatch')
    elif case=='missing-source':
        require(inputs['source_kind'] is None and inputs['source_bytes_base64'] is None and inputs['link_target'] is None,'module missing-file raw metadata changed');error(row['observed']['error'])
    elif case=='symbolic-link-source':
        require(inputs['source_kind']=='symlink' and inputs['source_bytes_base64'] is None and isinstance(inputs['link_target'],str)
            and inputs['link_target'].endswith('/genuine-module'),'module symlink refusal lost actual file type/target');error(row['observed']['error'])
    else:require(False,'unknown retained module source case')
    require(row['observed']['compiled_cache_exists'] is False,'module admission created compilation cache')


def storage_row(case,row):
    if case=='native-ownership':
        fields(row['inputs'],{'selection','requested_incarnation_base64'});fields(row['observed'],{'selection_error','durable_none','original','durable_default'})
        selected=dict(volume=[1,2],empty=[])
        item=decoded(b64(row['inputs']['requested_incarnation_base64']))
        require(same(row['inputs']['selection'],selected) and same(item['storage_crash_selection'],selected) and item['storage'] is None,
            'native storage selection refusal missing its actual unowned-volume input')
        require(row['observed']==dict(selection_error='storage crash selection requires declared storage volumes',durable_none=True,original={},durable_default=True),
            'native default/empty selection or undeclared volume refusal changed')
        return
    require(case=='native-deep-detachment','unknown storage selection case')
    fields(row['inputs'],{'original_incarnation_base64'});fields(row['observed'],{'projected','detached','dependency_detached','reprojected','original_after_base64'})
    raw=b64(row['inputs']['original_incarnation_base64']);item=decoded(raw)
    require(raw==canonical(item) and b64(row['observed']['original_after_base64'])==raw,'native storage projection changed its original retained bytes')
    selection=dict(volume=[1,2],empty=[])
    require(same(item['storage_crash_selection'],selection),'storage projection selection fixture changed')
    storage=item['storage'];require(storage is not None,'storage projection original declaration absent')
    context=canonical({'volume':{'PendingSHA256':'a'*64,'Operations':[{'ID':2,'Dependencies':[1]}]}})
    require(raw_bytes(storage['context'])==context and storage['descriptor']['crash_context']==dict(chunks=1,bytes=len(context),sha256=digest(context))
            and storage['descriptor']['crash_limits']==dict(States=32,Operations=8,Depth=8,Bytes=4096,Work=65536),
            'storage projection lost original context dependency/capacity raw witnesses')
    projected=copy.deepcopy(storage);projected['descriptor']['crash_selection']=selection
    detached=copy.deepcopy(projected);detached['descriptor']['crash_selection']['volume'][0]=99;detached['descriptor']['crash_selection'].pop('empty')
    detached['descriptor']['crash_limits']['States']=1;detached['context'][0]=99
    dependency=copy.deepcopy(projected);value=decoded(context);value['volume']['Operations'][0]['Dependencies'][0]=99;dependency['context']=list(canonical(value))
    require(same(row['observed']['projected'],projected) and same(row['observed']['detached'],detached)
            and same(row['observed']['dependency_detached'],dependency) and same(row['observed']['reprojected'],projected),
            'native storage projection aliases its source selections/context/dependencies/limits or did not perform exact perturbations')


SQLITE_MUTATIONS={
    'unknown-schema':('schema','unknown'),'arbitrary-prefix':('prefix','arbitrary '),'mixed-history':('histories',[{'name':'mixed'}]),
    'missing-ack-node':('acknowledged',dict(node='missing',incarnation=2)),'same-handles':('recovered',dict(node='sqlite',incarnation=2)),
    'empty-events':('expected_events',[]),'noncanonical-event':('expected_events',['01']),
    'ambiguous-value':('expected_value','comma,ambiguous'),'unchecked-integrity':('expected_integrity','unchecked')}


def sqlite_row(case,row):
    baseline=dict(kind='sqlite-durable-recovery',selection=None)
    if case=='native-valid':
        require(same(row['inputs'],baseline) and same(row['observed'],dict(workload=baseline)),'closed native SQLite workload positive control changed')
    else:
        require(case in SQLITE_MUTATIONS,'unknown native SQLite intent case');name,value=SQLITE_MUTATIONS[case]
        expected=dict(baseline,**{name:value});require(same(row['inputs'],expected),'native SQLite caller-intent mutation changed')
        fields(row['observed'],{'error'});error(row['observed']['error'],'unknown field `'+name+'`')


def admission_row(row):
    fields(row['inputs'],{'retained_inputs_base64','record','actual_execution'})
    inputs=decoded(b64(row['inputs']['retained_inputs_base64']));record=row['inputs']['record'];actual=row['inputs']['actual_execution']
    require(inputs['schema']==SCHEMA and record['schema']=='gomad.execution-record/v2' and record['identity']['configuration']==digest(canonical(inputs['configuration']))
            and record['execution_identity']==digest(canonical(record['identity'])),'actual retained admission record/configuration not authenticated')
    fields(actual,{'outcome','evidence'});outcome=actual['outcome'];evidence=actual['evidence']
    require(record['outcome']['termination']==outcome['termination'] and record['outcome']['exit_code']==outcome.get('exit_code')
            and record['outcome']['reason']==outcome['message'] and outcome.get('divergence') is None,'actual retained replay has an independently visible outcome divergence')
    for name in ('cluster_summary','cluster_oracles','workload-oracles.json','stdout','stderr','transcript','choice','diagnostic','network','storage','storage_state','storage_image'):
        raw=raw_bytes(evidence[name]) if name in evidence else b''
        stream=record['streams'].get(('output/' if name in ('stdout','stderr') else 'evidence/')+name)
        require(digest(raw)==(stream['sha256'] if stream is not None else digest(b'')),'actual source-less retained replay stream diverged: '+name)
    require(record['virtual_time_nanos']==outcome.get('local_model',{}).get('now',0),'actual retained virtual-time observation diverged')
    fields(row['observed'],{'divergence','termination','exit_code','entries_before','entries_after','reopened_inputs_base64','compiled_cache_exists','directory_before','directory_after'})
    observed=row['observed']
    require(observed['divergence'] is None and observed['termination']==outcome['termination'] and observed['exit_code']==outcome.get('exit_code')
        and b64(observed['reopened_inputs_base64'])==b64(row['inputs']['retained_inputs_base64']),'retained admission changed private raw inputs or hid actual divergence')
    directory_unchanged(observed)


def expected_cases():
    result={'history/'+case for case in history_cases()}
    result|={'policy/'+case for case in ('','go-run','go-test','unknown','unknown-handle','duplicate-policy')}
    result|={f'capacity/{terminal}/{message}/report-only={str(flag).lower()}/maximum={maximum}' for terminal,message in
        (('infrastructure','output capacity'),('capacity','transcript capacity'),('exit',''),('capacity','output capacity')) for flag in (False,True) for maximum in (1,4096)}
    result|={'authentication/'+case for case in ('digest','bytes','path','limit','raw-bytes')}
    result|={'decode/'+case for case in (*DECODE_CASES,'native-valid')}
    result|={'binding/'+case for case in BIND_CASES}
    result|={'module/'+case for case in ('tampered-bytes','truncated','missing-source','symbolic-link-source')}
    result|={'storage-selection/native-ownership','storage-selection/native-deep-detachment','admission/actual-replay'}
    result|={'sqlite-intent/'+case for case in (*SQLITE_MUTATIONS,'native-valid')}
    return result


PARTIAL={
    'TestBoundRetainedReplayRejectsBeforeWritingInputs':('binding/', ['original empty retained payload under both empty/foreign primary identities must refuse through every public admission route']),
    'TestRetainedAdmissionConsumesPrivateInputsOnlyOnce':('admission/', ['Go admission retry/consumed-buffer/provider-clear branches are retired; native ownership-by-value is a new lineage, not the same runtime refusal']),
    'TestRetainedPrimaryBindingRejectsMissingAndForeignModules':('binding/', ['original sha256-prefix primary spellings and no-incarnation original record are not reproduced by the native typed module reference matrix']),
    'TestRetainedReplayRejectsTamperedOrMismatchedTargetModule':('module/', ['original caller-supplied Copy/symlink-copy callbacks are retired; native reads an authenticated capability directly']),
    'TestRetainedReplayRequiresTheInstalledHelper':('binding/', ['old external installation missing/changed executable bytes/path branches retired under compile-linked helper lineage']),
    'TestRetainedSchemaV1IsRefusedWithVersionedError':('decode/', ['old legacy envelope Verify/Replay/Selection versioned re-record diagnostic requires the outer-lineage reader']),
    'TestRetainedLifecycleRejectsHostCallbacksBeforeReplay':('decode/', ['new typed config forbids callback fields; original callback collections/provider routes retired']),
    'TestClusterRetentionRefusesGoServedCalls':('decode/', ['retired Go service/callback counter branches have no native execution route']),
    'TestRetainedStorageSelectionProjectionIsDetached':('storage-selection/', ['nil selection entries are not representable in the new typed native map; original full shared preflight rejection requires separate admission proof']),
    'TestSQLiteOraclePlanRequiresSealedIntent':('sqlite-intent/', ['new fixed SQLite workload schema refuses caller plan fields; native workload-owner handle/event/value/integrity validation is not exposed by these schema-only cases']),
}


def observations(config,stdout):
    rows={};expected=expected_cases()
    prefixes=('history/','policy/','capacity/','authentication/','decode/','binding/','admission/','module/','storage-selection/','sqlite-intent/')
    for line in stdout.split('\n'):
        if MARKER not in line:continue
        row=decoded(line[line.index(MARKER)+len(MARKER):])
        if not row.get('case','').startswith(prefixes):continue
        fields(row,{'schema','case','inputs','observed'})
        require(row['schema']==ROW_SCHEMA and row['case'] in expected and row['case'] not in rows,'unknown/duplicate retained native input observation')
        rows[row['case']]=row
    require(set(rows)==expected,'native retained inputs omitted exact original case/subvariant catalog')
    raw=b64(rows['decode/native-valid']['inputs']['payload_base64']);baseline=metadata(raw);facts={}
    for name,row in rows.items():
        group,case=name.split('/',1)
        if group=='decode':decode_row(case,row,raw)
        elif group=='binding':binding_row(case,row,baseline)
        elif group=='history':
            fields(row['inputs'],{'stdout_base64'});actual=b64(row['inputs']['stdout_base64']);require(actual==history_cases()[case],'history ambiguous input bytes differ from exact original variant')
            if case=='valid':require(same(row['observed'],valid_history_report()),'native valid history report differs from independent actual/event/identity evidence')
            else:
                try:history(actual)
                except Refusal:error(row['observed']['error'])
                else:require(False,'ambiguous native history independently admitted')
        elif group=='policy':policy_row(case,row)
        elif group=='capacity':capacity_row(case,row)
        elif group=='authentication':authentication_row(case,row,raw)
        elif group=='module':module_row(case,row)
        elif group=='storage-selection':storage_row(case,row)
        elif group=='sqlite-intent':sqlite_row(case,row)
        elif group=='admission':admission_row(row)
        facts[name]=dict(input_sha256=digest(canonical(row['inputs'])),observation_sha256=digest(canonical(row['observed'])),validated_native_predicate=True)
    roots={
        'TestHistoryObservationRejectsAmbiguousInputs':dict(cases=sorted(name for name in facts if name.startswith('history/'))),
        'TestSoftAssertionClusterOptOutRequiresRecordedTestWorkload':dict(cases=sorted(name for name in facts if name.startswith('policy/'))),
        'TestSoftAssertionOutputCapacityAnnotationDoesNotBroadenOtherFailures':dict(cases=sorted(name for name in facts if name.startswith('capacity/'))),
    }
    partial={root:dict(cases=sorted(name for name in facts if name.startswith(prefix)),remaining_predicates=gaps,
        lineage='native typed/compile-linked retained lineage; retired Go callbacks/materialization are not asserted as equivalent branches')
        for root,(prefix,gaps) in PARTIAL.items()}
    return dict(facts=facts,roots=roots,partial_roots=partial,
        dependencies={root:value['remaining_predicates'] for root,value in partial.items()},
        supporting_obligations={
            'TestRetainedDecodeAcceptsOrdinarySpellingsAndRejectsStrictInput':dict(strict_cases=sorted(name for name in facts if name.startswith('decode/')),complete=False),
            'TestSoftAssertionActualClusterOutputCapacityRemainsInfrastructureFailure':dict(annotation_cases=sorted(name for name in facts if name.startswith('capacity/')),complete=False),
            'TestSoftAssertionCapacityPrecedesFinalizationErrors':dict(annotation_cases=sorted(name for name in facts if name.startswith('capacity/')),complete=False)})
