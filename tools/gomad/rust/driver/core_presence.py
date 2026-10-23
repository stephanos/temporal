"""Modern native presence receipt; retired Go-only representations stay unreceived."""
import copy
from pathlib import Path

from driver import require
from native_roles import decode, regular
from native_observations import canonical, digest
from core_retained_inputs import b64

SCHEMA='gomad.native-retained-presence/v1'
MARKER='GOMAD_PRESENCE_OBSERVATION '
ROOT='TestRetainedOrdinarySpellingKeepsRawAuthenticationAndPresence'
FILE='evidence/cluster-replay.json'
VARIANTS={'optional-none-missing','optional-none-null','optional-some-empty-rejected','optional-some-valid',
          'default-vector-missing','default-vector-empty','default-vector-null','required-vector-empty','required-vector-missing','required-vector-null'}
SPELLINGS={'canonical','pretty','order','escaped'}
DECODER_REFUSALS={'default-vector-null','required-vector-missing','required-vector-null'}
ORIGINAL_SOURCE='tools/gomad/internal/backends/wasm/cluster/retained_admission_test.go'
ORIGINAL_SHA256='0ed405e7c19d13928a3ad5d879c669c882960fe69a1ebc1876f32e73dddd5c46'
DISPOSITIONS=[
    dict(original='Record=[]byte{} versus nil',status='unreceived',reason='No native inline Record byte field; native immutable record.json and manifest descriptors own retained record bytes.'),
    dict(original='Boots=[]retainedBoot{} versus nil',status='unreceived',reason='No native Boots field; native typed incarnations and retained owner descriptors replace boot bundles.'),
    dict(original='OutputStops=nil',status='unreceived',reason='No native OutputStops field; current retained recipe has no corresponding optional vector.'),
    dict(original='Oracles=&OraclePlan{}',status='owned-modern-replacement',reason='Some empty typed native Plan survives decoding but fails current semantic admission; Some minimally valid native policy remains present and admits.'),
]


MODULE_SHA256='7e8bca9c8b7ecd66cfd4e8e121112dc1ddb2a04fcab7d78ce3de50c4b2b26293'


def baseline_identity(value):
    require(set(value)=={'schema','helper_sha256','host_sha256','runtime_implementation_sha256','toolchain_sha256','configuration','retained'},'native presence baseline fields differ')
    require(value['schema']=='gomad.cluster-replay-inputs/v1' and value['toolchain_sha256']==digest(b'gomad.fixture-wat-source/v1'),'presence substituted recorded native fixture identity')
    config=value['configuration']
    handles=[dict(node=node,incarnation=incarnation) for node,incarnation in [('a',1),('b',1),('a',2)]]
    require([item['handle'] for item in config['incarnations']]==handles
            and [item['handle'] for item in value['retained']]==[handles[index] for index in (0,2,1)]
            and config.get('oracle_plan') is None and not config.get('nodes') and config['crash_groups']==[],'presence native baseline owner/presence differs')
    for item in config['incarnations']:
        require(item['module']['file']=='target/module.wasm' and item['module']['sha256']==MODULE_SHA256
                and item['runtime_sha256']==digest(b'inspect-runtime-implementation/v1'),'presence native module/runtime fixture differs')


def shape(baseline,variant):
    value=copy.deepcopy(baseline);config=value['configuration']
    if variant=='optional-none-missing':config.pop('oracle_plan',None)
    elif variant=='optional-none-null':config['oracle_plan']=None
    elif variant=='optional-some-empty-rejected':config['oracle_plan']=dict(schema='gomad.wasm.soft-assertion-observation/v1',maximum_bytes=4096)
    elif variant=='optional-some-valid':
        config['incarnations'][0]['workload_kind']='go-test'
        config['oracle_plan']=dict(schema='gomad.wasm.soft-assertion-observation/v1',maximum_bytes=4096,prefix='',histories=[],convergence=[],soft_assertions=[dict(handle=copy.deepcopy(config['incarnations'][0]['handle']),report_only=False)])
    elif variant=='default-vector-missing':config.pop('nodes',None)
    elif variant=='default-vector-empty':config['nodes']=[]
    elif variant=='default-vector-null':config['nodes']=None
    elif variant=='required-vector-empty':config['crash_groups']=[]
    elif variant=='required-vector-missing':config.pop('crash_groups',None)
    elif variant=='required-vector-null':config['crash_groups']=None
    else:require(False,'unknown native presence variant')
    return value


def native_canonical(value):
    value=copy.deepcopy(value);config=value['configuration']
    if config.get('oracle_plan') is None:config.pop('oracle_plan',None)
    else:
        plan=config['oracle_plan']
        for key,default in [('prefix',''),('histories',[]),('convergence',[]),('soft_assertions',[])]:plan.setdefault(key,default)
    if not config.get('nodes'):config.pop('nodes',None)
    return value


def authentication(raw,manifest,mutations):
    require(isinstance(manifest,dict) and manifest['schema']=='gomad.artifact-manifest/v2','foreign presence manifest')
    descriptor=manifest['files'][FILE];stream=manifest['record']['streams'][FILE]
    require(descriptor['bytes']==stream['bytes']==len(raw) and descriptor['sha256']==stream['sha256']==digest(raw),'presence payload not exactly record and manifest bound')
    require(manifest['record']['schema']=='gomad.execution-record/v2','foreign presence record lineage')
    require(len(mutations)==4 and {item['variant'] for item in mutations}=={'digest','length','path','raw-append'},'presence authentication omitted or duplicated original mutation')
    original=dict(file=FILE,bytes=len(raw),sha256=digest(raw),limit=16<<20)
    for mutation in mutations:
        expected=dict(original);variant=mutation['variant']
        if variant=='digest':expected['sha256']=digest(b'foreign bytes')
        elif variant=='length':expected['bytes']+=1
        elif variant=='path':expected['file']='evidence/foreign.json'
        else:require(b64(mutation['raw_base64'])==raw+b'\n','presence raw-append preimage differs')
        require(mutation['descriptor']==expected and isinstance(mutation['error'],str) and mutation['error'],'presence forged authentication mutation/refusal')
        if variant=='digest':require(expected['sha256']!=descriptor['sha256'],'presence digest mutation did not differ')
        if variant=='path':require(expected['file'] not in manifest['files'],'presence foreign path actually present')


def validate(row):
    require(set(row)=={'schema','original_root','variant','spelling','inputs','observed'} and row['schema']==SCHEMA and row['original_root']==ROOT,'foreign presence envelope')
    variant,spelling=row['variant'],row['spelling']
    require(variant in VARIANTS and spelling in SPELLINGS,'unknown presence variant or spelling')
    inputs,observed=row['inputs'],row['observed']
    require(set(inputs)=={'baseline','raw_base64','manifest'} and set(observed)=={'canonical_base64','decoder_error','admission_error','authentication'},'presence raw inputs/observations missing or foreign')
    baseline_identity(inputs['baseline'])
    raw=b64(inputs['raw_base64']);value=decode(raw)
    require(value==shape(inputs['baseline'],variant),'presence raw variant substituted fixed fixture input')
    require(value['schema']=='gomad.cluster-replay-inputs/v1','foreign retained presence schema')
    if spelling=='canonical':require(raw==canonical(value),'presence canonical spelling differs')
    elif spelling=='pretty':require(raw.startswith(b'{\n') and b'  "' in raw,'presence pretty spelling missing')
    elif spelling=='order':
        require(raw.startswith(b'{"toolchain_sha256":'),'presence reversed top member order missing')
    else:require(b'"\\u0073chema":' in raw and b'\\u0067omad.cluster-replay-inputs' in raw,'presence escaped member/value spelling missing')
    authentication(raw,inputs['manifest'],observed['authentication'])
    if variant in DECODER_REFUSALS:
        require(observed['canonical_base64'] is None and isinstance(observed['decoder_error'],str) and observed['decoder_error'] and observed['admission_error'] is None,'native required/default vector malformed presence was admitted')
        canonical_sha=None
    else:
        encoded=b64(observed['canonical_base64']);expected=native_canonical(value)
        require(encoded==canonical(expected) and observed['decoder_error'] is None,'native optional/vector presence changed on actual canonical re-encode')
        if variant=='optional-some-empty-rejected':require(observed['admission_error']=='invalid typed cluster oracle plan','native empty typed Plan lost pointer presence or bypassed semantic refusal')
        else:require(observed['admission_error'] is None,'native valid presence failed real retained admission')
        canonical_sha=digest(encoded)
    return dict(case=variant+'/'+spelling,raw_sha256=digest(raw),canonical_sha256=canonical_sha,modern_received=True)


def observations(config,stdout):
    rows=[decode(line.split(MARKER,1)[1]) for line in stdout.splitlines() if MARKER in line]
    if not rows:return {}
    source=Path(config.get('presence_original_source','/home/agent/evidence/fn232-original-terminal-derived-v3/source'))/ORIGINAL_SOURCE
    require(digest(regular(source).read_bytes())==ORIGINAL_SHA256,'immutable original retained presence assertions changed')
    seen=set();facts=[]
    for row in rows:
        key=(row.get('variant'),row.get('spelling'))
        require(key not in seen,'duplicate retained presence raw case');seen.add(key)
        facts.append(validate(row))
    require(seen=={(variant,spelling) for variant in VARIANTS for spelling in SPELLINGS},'missing retained presence subvariant or spelling')
    return {ROOT:[dict(cases=facts,modern_received=True,dispositions=DISPOSITIONS,
                      remaining_original_predicates=['retired inline Record, Boots and OutputStops nil/empty representations have no native decoder fields'])]}
