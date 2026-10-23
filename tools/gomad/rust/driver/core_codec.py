"""Independent native codec observations, authenticated against frozen corpus bytes.

These predicates describe native binary lineage. They do not assert reconstruction
of retired Go JSON documents or run the retired Go corpus generators.
"""
import base64
from pathlib import Path
import re

from driver import require
from native_observations import canonical, digest
import native_roles
from core_predicates import raw_bytes

SCHEMA = 'gomad.core-predicate-observation/v1'
MARKER = 'GOMAD_CORE_OBSERVATION '
OP_FIELDS = {'ID', 'Kind', 'Dependencies', 'Inode', 'Offset', 'Bytes', 'PayloadSHA256', 'EffectSHA256', 'Path', 'Destination'}


def decoded(raw):
    try:
        value=native_roles.decode(raw)
        canonical(value)
        return value
    except (ValueError, TypeError, UnicodeError) as error:
        require(False, 'invalid codec JSON preimage: ' + str(error))


def unsigned(value):
    require(type(value) is int and 0 <= value < 1 << 64, 'foreign storage unsigned integer')
    return value


def typed_context(context):
    require(context is None or isinstance(context, dict), 'foreign storage context map')
    for name, prior in (context or {}).items():
        require(isinstance(name, str) and isinstance(prior,dict) and set(prior) == {'PendingSHA256', 'Operations'}
                and isinstance(prior['PendingSHA256'], str), 'foreign storage context fields')
        operations = prior['Operations']
        require(operations is None or isinstance(operations, list), 'foreign storage operations shape')
        for operation in operations or []:
            require(isinstance(operation,dict) and set(operation) == OP_FIELDS, 'foreign storage operation fields')
            for field in ('ID', 'Inode', 'Offset', 'Bytes'):
                unsigned(operation[field])
            for field in ('Kind', 'PayloadSHA256', 'EffectSHA256', 'Path', 'Destination'):
                require(isinstance(operation[field], str), 'foreign storage operation text')
            dependencies = operation['Dependencies']
            require(dependencies is None or isinstance(dependencies, list), 'foreign dependency shape')
            for dependency in dependencies or []:
                unsigned(dependency)


def context_refusal(volumes, pending, selection, context):
    """Recompute first refusal before any hash/count/projection credit is assigned."""
    unsigned(pending)
    typed_context(context)
    require(selection is None or isinstance(selection, dict), 'foreign storage selection')
    for name, selected in sorted((selection or {}).items()):
        require(isinstance(name, str) and (selected is None or isinstance(selected, list)), 'foreign selected operation shape')
        for identity in selected or []:
            unsigned(identity)
    def refusal(reason):
        return 'WASI invalid: storage crash ' + reason
    for name, selected in sorted((selection or {}).items()):
        if name not in volumes:
            return refusal('selection volume')
        seen = set()
        for identity in selected or []:
            if identity == 0:
                return refusal('selection zero operation')
            if identity in seen:
                return refusal('selection duplicate operation')
            seen.add(identity)
        prior = (context or {}).get(name)
        if not selected and prior is None:
            continue
        if prior is None:
            return refusal('selection requires prior observation')
        operations = {operation['ID']:operation for operation in prior['Operations'] or []}
        for identity in selected or []:
            if identity not in operations:
                return refusal('selection unknown operation')
            if any(dependency in operations and dependency not in seen for dependency in operations[identity]['Dependencies'] or []):
                return refusal('selection is not dependency-closed')
    for name, prior in sorted((context or {}).items()):
        if name not in volumes:
            return refusal('context volume')
        operations = prior['Operations']
        if not re.fullmatch(r'sha256:[0-9a-f]{64}', prior['PendingSHA256']) or operations is None or len(operations) > pending:
            return refusal('context inventory')
        previous = 0
        for operation in operations:
            if operation['ID'] <= previous:
                return refusal('context operation order')
            previous = operation['ID']
            if any(dependency == 0 or dependency >= operation['ID'] for dependency in operation['Dependencies'] or []):
                return refusal('context dependency')
    return None


def context_frames(raw):
    frames = [canonical(dict(type='storage_context_input', sequence=index, data_base64=base64.b64encode(raw[start:start+65536]).decode()))
              for index, start in enumerate(range(0,len(raw),65536))]
    return b'\n'.join(frames+[canonical(dict(type='storage_context_inputs_end'))])+b'\n'


def context_transport(wire, declared):
    require(set(declared) == {'bytes','chunks','sha256'}, 'foreign context descriptor')
    unsigned(declared['bytes']); unsigned(declared['chunks'])
    require(0 < declared['bytes'] <= 64 << 20 and declared['chunks'] == (declared['bytes']+65535)//65536
            and re.fullmatch('[0-9a-f]{64}', declared['sha256']) is not None, 'invalid context descriptor')
    require(wire.endswith(b'\n'), 'unterminated context transport')
    lines = wire.splitlines()
    require(len(lines) == declared['chunks']+1, 'context frame cardinality')
    require(decoded(lines[-1]) == dict(type='storage_context_inputs_end'), 'context terminator')
    chunks=[]
    for index,line in enumerate(lines[:-1]):
        frame=decoded(line)
        require(set(frame)=={'type','sequence','data_base64'} and frame['type']=='storage_context_input'
                and type(frame['sequence']) is int and frame['sequence']==index, 'context frame order/shape')
        try:
            chunk=base64.b64decode(frame['data_base64'],validate=True)
        except (ValueError,TypeError) as error:
            require(False,'invalid context frame base64: '+str(error))
        require(0 < len(chunk) <= 65536 and (index == len(lines)-2 or len(chunk)==65536), 'context chunk size')
        chunks.append(chunk)
    raw=b''.join(chunks)
    require(len(raw)==declared['bytes'] and digest(raw)==declared['sha256'], 'context transport identity')
    return raw


def frozen(config, relative):
    path=Path(config['repo'])/'tools/gomad'/relative
    native_roles.pinned(config,path)
    return path.read_bytes()


def order_catalog(config):
    corpus=decoded(frozen(config,'testdata/parity/storage-crash-validation-order.json'))
    require(corpus['schema']=='gomad.storage-crash-validation-order/v1' and corpus['cases'], 'foreign storage order corpus')
    result={}
    for case in corpus['cases']:
        for reversed_order in (False,True):
            selection=None if case['selection'] is None else dict(case['selection'])
            context=None if case['context'] is None else dict(case['context'])
            expected=context_refusal(corpus['volumes'],corpus['pending_operations'],selection,context)
            require(expected==case['want'], 'frozen storage order corpus contradicts independently derived refusal')
            key=('storage-validation-order-corpus',case['name'],reversed_order)
            require(key not in result,'duplicate frozen storage order case')
            def ordered_wire(entries):
                if entries is None:return None
                pairs=list(reversed(entries)) if reversed_order else entries
                return b'{'+b','.join(canonical(name)+b':'+canonical(value) for name,value in pairs)+b'}'
            result[key]=(selection,context,expected,ordered_wire(case['selection']),ordered_wire(case['context']))
    return result


def operation(identity, path=''):
    return dict(ID=identity,Kind='write',Dependencies=None,Inode=0,Offset=0,Bytes=0,
                PayloadSHA256='',EffectSHA256='',Path=path,Destination='')


def storage_catalog(config):
    """Variable order cases derive only from the authenticated frozen corpus."""
    catalog = order_catalog(config)
    second=operation(2);second['Dependencies']=[1]
    base={'data':dict(PendingSHA256='sha256:'+'a'*64,Operations=[operation(1),second])}
    import copy
    for variant,selection,edit in (('future-operations',[1,2],''),('unknown-operation',[3],''),
            ('dependency-closure',[2],''),('forged-hash',[],'hash'),('nil-inventory',[],'nil'),
            ('repeated-inventory',[],'repeated'),('invented-dependency',[1],'dependency')):
        value=copy.deepcopy(base)
        if edit=='hash':value['data']['PendingSHA256']='a'
        if edit=='nil':value['data']['Operations']=None
        if edit=='repeated':value['data']['Operations']=[operation(1),operation(1)]
        if edit=='dependency':
            changed=operation(1);changed['Dependencies']=[2];value['data']['Operations']=[changed]
        catalog[('storage-context-admission',variant)]=({'data':selection},value,
                context_refusal(['data'],1000,{'data':selection},value))
    catalog[('storage-context-admission-missing',)]=None
    for field,value in (('ID',3),('Kind','forged'),('Dependencies',[]),('Inode',1),('Offset',1),('Bytes',1),
                        ('PayloadSHA256','x'),('EffectSHA256','x'),('Path','x'),('Destination','x')):
        catalog[('storage-context-equality',field)]=value
    for shape in ('null','empty'):
        catalog[('storage-context-equality-shape',shape)]=None
    for variant in ('nil-dependencies','empty-dependencies','dependency','hash','nil-operations','empty-operations','field'):
        catalog[('storage-context-equality-original-dependency',variant)]=None
    catalog[('storage-context-equality-positive',)]=None
    catalog[('storage-context-equality-hash-map',)]=None
    catalog[('storage-large-context-transport',)]=None
    for kind in ('selection','context'):
        for iteration in range(64):
            catalog[('storage-validation-volume-order',kind,iteration)]=None
            catalog[('storage-validation-volume-boot-order',kind,iteration)]=None
    return catalog


def observation_key(row):
    case=row['case'];inputs=row['inputs']
    if case=='storage-context-admission':return (case,inputs['variant'])
    if case=='storage-context-equality':return (case,inputs['field'])
    if case=='storage-context-equality-original-dependency':return (case,inputs['variant'])
    if case=='storage-context-equality-shape':return (case,'null' if inputs['changed']['Operations'] is None else 'empty')
    if case=='storage-validation-order-corpus':
        require(type(inputs['reversed']) is bool,'order reversal must be boolean')
        return (case,inputs['name'],inputs['reversed'])
    if case in ('storage-validation-volume-order','storage-validation-volume-boot-order'):
        unsigned(inputs['iteration'])
        return (case,inputs['kind'],inputs['iteration'])
    return (case,)


def descriptor_model(descriptor,pending,transitions,volumes=None):
    volumes=volumes or [dict(ID='data',Path='/workspace',CapacityBytes=1<<20)]
    model=dict(volumes=volumes,limits=dict(PendingOperations=pending,Transitions=transitions),faults=None,sqlite_lock_databases=None)
    go_model=dict(volumes=volumes,limits=model['limits'])
    require(canonical(descriptor['model'])==canonical(model) and descriptor['model_identity']=='sha256:'+digest(struct_json(go_model)),
            'native storage descriptor model/capacity identity changed')


def storage_observation(config,row,catalog):
    require(set(row)=={'schema','case','inputs','observed'} and row['schema']==SCHEMA,'foreign codec observation')
    key=observation_key(row)
    require(key in catalog,'unknown native storage predicate case')
    case=row['case'];inputs=row['inputs'];observed=row['observed'];expected=catalog[key]
    if case=='storage-context-admission':
        selection,context,error=expected
        require(context_refusal(['data'],1000,inputs['selection'],inputs['context'])==error,'observed admission inputs alter typed refusal')
        require(inputs==dict(variant=key[1],selection=selection,context=context)
                and observed==dict(error=error),'storage admission differs from independent first refusal')
    elif case=='storage-context-admission-missing':
        descriptor=inputs['descriptor']
        descriptor_model(descriptor,1000,10000)
        require(set(inputs)=={'descriptor'} and descriptor['crash_selection']=={'data':[1]}
                and descriptor['crash_context'] is None,'missing-context selection fixture changed')
        error=context_refusal(['data'],1000,descriptor['crash_selection'],None)
        require(observed==dict(refusal=error),'missing prior context did not refuse')
    elif case.startswith('storage-context-equality'):
        import copy
        base=dict(PendingSHA256='sha256:'+'a'*64,Operations=[operation(2)])
        if case in ('storage-context-equality-original-dependency','storage-context-equality-positive'):
            base['Operations'][0]['Dependencies']=[1]
        typed_context({'data':inputs['base']})
        if 'changed' in inputs:typed_context({'data':inputs['changed']})
        require(canonical(inputs['base'])==canonical(base),'storage equality base changed')
        changed=copy.deepcopy(base)
        if case=='storage-context-equality-positive':
            require(inputs==dict(base=base,clone=base) and canonical(observed)==canonical(dict(clone_equal=True,map_equal=True)),'positive clone/map equality failed')
            return key
        if case=='storage-context-equality-original-dependency':
            require(set(inputs)=={'base','variant','changed'},'foreign dependency equality input')
            variant=key[1]
            if variant in ('nil-dependencies','empty-dependencies','dependency'):
                changed['Operations'][0]['Dependencies']={'nil-dependencies':None,'empty-dependencies':[],'dependency':[3]}[variant]
            elif variant=='hash':changed['PendingSHA256']='sha256:'+'b'*64
            elif variant in ('nil-operations','empty-operations'):changed['Operations']=None if variant=='nil-operations' else []
            else:changed['Operations'][0]['EffectSHA256']='sha256:x'
            want=dict(equal_forward=False,equal_reverse=False)
        elif case=='storage-context-equality':
            require(set(inputs)=={'base','field','changed'},'foreign equality input')
            changed['Operations'][0][key[1]]=expected
            want=dict(equal_forward=False,equal_reverse=False)
        elif case=='storage-context-equality-shape':
            require(set(inputs)=={'base','changed'},'foreign shape input')
            changed['Operations']=None if key[1]=='null' else []
            want=dict(equal_forward=False,equal_reverse=False)
        else:
            require(set(inputs)=={'base','changed'},'foreign hash/map input')
            changed['PendingSHA256']='sha256:'+'b'*64
            want=dict(hash_equal=False,map_equal=False,null_empty_equal=False)
        require(canonical(inputs['changed'])==canonical(changed) and changed!=base and canonical(observed)==canonical(want),'exact context equality lost a field or null shape')
    elif case=='storage-large-context-transport':
        require(set(inputs)=={'descriptor','transport_bytes','wire_bytes','typed_context_bytes','transport_sha256','typed_context_sha256'},'missing raw context transport preimage')
        descriptor=inputs['descriptor'];descriptor_model(descriptor,5000,10000)
        raw=raw_bytes(inputs['transport_bytes']);wire=raw_bytes(inputs['wire_bytes'])
        typed=raw_bytes(inputs['typed_context_bytes'])
        expected_context={'data':dict(PendingSHA256='sha256:'+'a'*64,Operations=[operation(identity,'x'*256) for identity in range(1,5001)])}
        require(raw==canonical(expected_context) and len(raw)>1<<20 and descriptor['crash_selection']=={'data':[1]},'large context lost its exact 5000 operations')
        require(context_transport(wire,descriptor['crash_context'])==raw and wire==context_frames(raw),'large transport chunking changed')
        require(decoded(typed)==expected_context and typed==struct_json(expected_context) and inputs['transport_sha256']==digest(raw)
                and inputs['typed_context_sha256']==digest(typed),'context typed/transport identities changed')
        require(len(canonical(descriptor))<=8192 and descriptor['crash_context']['chunks']>=17,'execute descriptor embeds unbounded context')
        require(canonical(observed)==canonical(dict(context_sha256=digest(typed),operation_ids=list(range(1,5001)))), 'large context roundtrip differs')
    elif case=='storage-validation-order-corpus':
        selection,context,error,selection_wire,context_wire=expected
        require(set(inputs)=={'name','reversed','descriptor','context'},'foreign order corpus input')
        descriptor=decoded(inputs['descriptor'])
        order=decoded(frozen(config,'testdata/parity/storage-crash-validation-order.json'))
        descriptor_model(descriptor,order['pending_operations'],100,[dict(ID=name,Path='/'+name,CapacityBytes=512) for name in order['volumes']])
        actual=None if inputs['context'] is None else decoded(inputs['context'])
        require((None if inputs['context'] is None else inputs['context'].encode())==context_wire,'order corpus insertion context changed')
        raw_selection=b'null' if selection_wire is None else selection_wire
        require(b'"crash_selection":'+raw_selection+b',"crash_context":' in inputs['descriptor'].encode(),'order corpus insertion selection changed')
        require(canonical(descriptor['crash_selection'])==canonical(selection) and canonical(actual)==canonical(context),'order corpus raw inputs changed')
        require(observed==dict(refusal=error),'order corpus first refusal differs')
        for field,want in (('selection',selection),('context',context)):
            raw=descriptor['crash_selection'] if field=='selection' else actual
            require(raw==want,'order corpus map mismatch')
    elif case in ('storage-validation-volume-order','storage-validation-volume-boot-order'):
        require(set(inputs)=={'kind','iteration','insertion_ids','descriptor','context'},'missing volume order context')
        kind,iteration=key[1:];ids=['a','b'] if iteration%2==0 else ['b','a']
        require(inputs['insertion_ids']==ids,'volume insertion sequence changed')
        descriptor=inputs['descriptor'];context=inputs['context']
        descriptor_model(descriptor,1000,1000,[dict(ID=name,Path='/'+name,CapacityBytes=512) for name in ('a','b')])
        if kind=='selection':
            require(descriptor['crash_selection']=={'a':[0],'b':[1,1]} and context=={},'volume selection fixture changed')
            context=None
        else:
            expected_context={name:dict(PendingSHA256='invalid' if name=='a' else 'sha256:'+'a'*64,Operations=[operation(0)]) for name in ids}
            require(context==expected_context and descriptor.get('crash_selection') is None,'volume context fixture changed')
        error=context_refusal(['a','b'],1000,descriptor.get('crash_selection'),context)
        require(observed==dict(refusal=error),'volume order first refusal differs')
    return key


ROOTS = {
    'TestStorageCrashContextAdmission': {'storage-context-admission','storage-context-admission-missing'},
    'TestStorageCrashContextEqualIsExact': {'storage-context-equality','storage-context-equality-shape','storage-context-equality-hash-map','storage-context-equality-original-dependency','storage-context-equality-positive'},
    'TestStorageCrashContextLargeChunkTransport': {'storage-large-context-transport'},
    'TestStorageCrashValidationOrderCorpus': {'storage-validation-order-corpus'},
    'TestStorageCrashValidationVolumeOrder': {'storage-validation-volume-order','storage-validation-volume-boot-order'},
}


def observations(config,stdout):
    catalog=storage_catalog(config);seen=set()
    for line in stdout.splitlines():
        if MARKER not in line:continue
        row=decoded(line[line.index(MARKER)+len(MARKER):])
        if not row.get('case','').startswith(('storage-context-admission','storage-context-equality','storage-large-context-transport','storage-validation-')):continue
        key=storage_observation(config,row,catalog)
        require(key not in seen,'duplicate native codec observation')
        seen.add(key)
    require(seen==set(catalog),'native codec harness omitted an exact frozen storage subvariant')
    facts={root:dict(cases=[list(key) for key in sorted(seen) if key[0] in cases],
                      lineage='native typed storage context; exact semantic fields and refusal precedence')
            for root,cases in ROOTS.items()}
    selected=[decoded(line[line.index(MARKER)+len(MARKER):]) for line in stdout.splitlines() if MARKER in line]
    selected=[row for row in selected if row.get('case') in {'storage-selected-baseline','storage-selected-state-recovery','storage-context-divergence','storage-large-observed-selection'}]
    facts.update(selected_observations(selected))
    evidence=[decoded(line[line.index(MARKER)+len(MARKER):]) for line in stdout.splitlines() if MARKER in line]
    facts.update(evidence_observations([row for row in evidence if row.get('case','').startswith('storage.native-')],selected))
    return facts


class ImageReader:
    def __init__(self,raw):
        require(isinstance(raw,bytes) and len(raw)<=64<<20,'native recovery image bound')
        self.raw=raw;self.at=0

    def take(self,count):
        require(0<=count<=len(self.raw)-self.at,'truncated native recovery image')
        raw=self.raw[self.at:self.at+count];self.at+=count
        return raw

    def number(self,width=8):return int.from_bytes(self.take(width),'little')

    def count(self):
        count=self.number();require(count<=len(self.raw)-self.at,'image count exceeds remaining bytes')
        return count

    def blob(self):return self.take(self.count())

    def text(self):
        try:return self.blob().decode('utf-8')
        except UnicodeError:require(False,'invalid recovery UTF-8')

    def numbers(self):return [self.number() for _ in range(self.count())]

    def finish(self):require(self.at==len(self.raw),'trailing native recovery image bytes')

    def object(self):
        inode,mode,kind,mod_time=(self.number() for _ in range(4));data=self.blob()
        require(inode>0 and mode<1<<32 and kind in (1,2) and (kind!=2 or not data),'invalid recovery object')
        return dict(inode=inode,mode=mode,kind=kind,mod_time=mod_time,data=data)

    def operations(self):
        operations=[]
        for _ in range(self.count()):
            identity=self.number();kind=self.text();dependencies=self.numbers();stores=[self.text() for _ in range(self.count())]
            effect_kind=self.text();inode=self.number();parent=self.number();parents=self.numbers()
            offset,size,mode,mod_time=(self.number() for _ in range(4));data=self.blob();presence=self.number()
            require(presence in (0,1) and mode<1<<32,'invalid recovery effect')
            obj=self.object() if presence else None
            namespace=[dict(path=self.text(),inode=self.number()) for _ in range(self.count())]
            require(identity>0 and dependencies==sorted(set(dependencies)) and all(0<dep<identity for dep in dependencies),'invalid recovery dependencies')
            require(size<=256<<20 and offset+len(data)<=256<<20,'recovery persistence byte bound')
            require(effect_kind in ('alloc','write','resize','metadata','namespace'),'unknown recovery persistence effect')
            require(kind==effect_kind and (effect_kind=='alloc')==(obj is not None),'recovery allocation object mismatch')
            operations.append(dict(id=identity,kind=kind,deps=dependencies,stores=stores,effect=dict(kind=effect_kind,inode=inode,parent=parent,
                parents=parents,offset=offset,size=size,mode=mode,mod_time=mod_time,data=data,object=obj,namespace=namespace)))
        require([op['id'] for op in operations]==sorted(set(op['id'] for op in operations)),'recovery operation order')
        return operations


def ordered_pairs(reader,key,value):
    pairs=[(key(),value()) for _ in range(reader.count())]
    require([name for name,_ in pairs]==sorted(set(name for name,_ in pairs)),'recovery map order or duplicate')
    return dict(pairs)


def recovery_image(raw):
    reader=ImageReader(raw)
    require(reader.take(8)==b'GMVOLIMG' and reader.number(4)==1 and reader.number(4)==3,'native recovery image header')
    sections=[]
    for tag in range(1,4):
        require(reader.number(4)==tag,'native recovery image section order')
        sections.append(ImageReader(reader.blob()))
    reader.finish()
    model=sections[0].text();sections[0].finish()
    require(re.fullmatch(r'sha256:[0-9a-f]{64}',model),'invalid recovery model identity')
    reader=sections[1];occurrences=ordered_pairs(reader,reader.text,reader.number);reader.finish()
    reader=sections[2];next_inode=reader.number();generation=reader.number();volumes={};all_inodes={1,2}
    require(next_inode>0 and generation>0,'recovery filesystem counters')
    for _ in range(reader.count()):
        name=reader.text();mount=reader.text();capacity=reader.number()
        require(name not in volumes and (not volumes or name>list(volumes)[-1]),'recovery volume order')
        objects=[reader.object() for _ in range(reader.count())]
        require([obj['inode'] for obj in objects]==sorted(set(obj['inode'] for obj in objects)),'recovery object order')
        objects={obj['inode']:obj for obj in objects}
        require(not (set(objects)&all_inodes),'recovery inode alias');all_inodes.update(objects)
        paths=ordered_pairs(reader,reader.text,reader.number);owned=reader.numbers()
        pending=reader.operations();history=reader.operations();next_operation=reader.number()
        last=ordered_pairs(reader,reader.text,reader.number)
        require(owned==sorted(set(owned)) and set(owned)<=set(objects) and paths.get('/') in objects,'recovery owned/root inventory')
        require(objects[paths['/']]['kind']==2 and all(inode in objects for inode in paths.values()),'recovery path inode inventory')
        require(sum(len(obj['data']) for obj in objects.values())<=capacity and all(inode<next_inode for inode in objects),'recovery inode or byte capacity')
        require(next_operation==len(history)+1 and [op['id'] for op in history]==list(range(1,next_operation))
                and all(op in history for op in pending),'recovery operation history')
        require(last=={store:op['id'] for op in pending for store in op['stores']},'recovery pending store map')
        volumes[name]=dict(mount=mount,capacity=capacity,objects=objects,paths=paths,owned=owned,pending=pending,history=history,next_operation=next_operation,last=last)
    reader.finish()
    return dict(model=model,occurrences=occurrences,next_inode=next_inode,generation=generation,volumes=volumes)


class VolumeHash:
    def __init__(self):
        import hashlib
        self.hash=hashlib.sha256()

    def number(self,value):self.hash.update((value%(1<<64)).to_bytes(8,'little'))

    def blob(self,value):self.number(len(value));self.hash.update(value)

    def text(self,value):self.blob(value.encode())

    def finish(self):return 'sha256:'+self.hash.hexdigest()


def effect_hash(hasher,effect):
    hasher.text(effect['kind']);hasher.number(effect['inode']);hasher.number(effect['parent'])
    for parent in effect['parents']:hasher.number(parent)
    for field in ('offset','size','mode','mod_time'):hasher.number(effect[field])
    hasher.blob(effect['data'])
    if effect['object'] is not None:
        obj=effect['object']
        for field in ('inode','mode','kind','mod_time'):hasher.number(obj[field])
        hasher.blob(obj['data'])
    for part in effect['namespace']:hasher.text(part['path']);hasher.number(part['inode'])


def pending_identity(name,operations):
    hasher=VolumeHash();hasher.text('gomad-volume-pending/v1');hasher.text(name)
    for op in operations:
        hasher.number(op['id']);hasher.text(op['kind'])
        for dependency in op['deps']:hasher.number(dependency)
        effect_hash(hasher,op['effect'])
    return hasher.finish()


def observed_operation(op):
    effect=op['effect'];hasher=VolumeHash();hasher.text('gomad-volume-effect/v1');effect_hash(hasher,effect)
    return dict(ID=op['id'],Kind=op['kind'],Dependencies=op['deps'] or None,Inode=effect['inode'],Offset=effect['offset'],
                Bytes=len(effect['data']) if effect['data'] else effect['size'],
                PayloadSHA256='sha256:'+digest(effect['data']) if effect['data'] else '',EffectSHA256=hasher.finish(),
                Path=effect['namespace'][0]['path'] if effect['namespace'] else '',
                Destination=effect['namespace'][-1]['path'] if len(effect['namespace'])>1 else '')


def entries(volume,crash=False):
    result=[]
    for path,inode in sorted(volume['paths'].items()):
        obj=volume['objects'][inode];time=obj['mod_time'];time=time if time<1<<63 else time-(1<<64)
        common=dict(Path=path,Mode=obj['mode'],Kind=obj['kind'] if crash else ('file' if obj['kind']==1 else 'directory'),ModTime=time)
        if crash:common['Data']=base64.b64encode(obj['data']).decode() if obj['data'] else None
        else:common.update(Size=len(obj['data']),DataSHA256='sha256:'+digest(obj['data']) if obj['kind']==1 else '')
        result.append(common)
    return result


def snapshot_identity(snapshot):
    hasher=VolumeHash();hasher.text('gomad-volume-snapshot/v1');hasher.text(snapshot['ID']);hasher.text(snapshot['Mount']);hasher.number(snapshot['CapacityBytes'])
    for items in (snapshot['Persisted'],snapshot['Volatile']):
        hasher.number(len(items))
        for entry in items:
            hasher.text(entry['Path']);hasher.number(entry['Mode']);hasher.text(entry['Kind']);hasher.number(entry['ModTime'])
            hasher.number(entry['Size']);hasher.text(entry['DataSHA256'])
    hasher.number(snapshot['PendingOperations']);hasher.text(snapshot['PendingSHA256']);hasher.number(snapshot['NextOperation'])
    return hasher.finish()


def recovered_snapshots(image):
    result=[]
    for name,volume in sorted(image['volumes'].items()):
        projected=entries(volume)
        snapshot=dict(ID=name,Mount=volume['mount'],CapacityBytes=volume['capacity'],Persisted=projected,Volatile=projected,
            PendingOperations=0,PendingSHA256=pending_identity(name,[]),NextOperation=volume['next_operation'])
        snapshot['Identity']=snapshot_identity(snapshot);result.append(snapshot)
    return result


def crash_identity(name,projected):
    hasher=VolumeHash();hasher.text('gomad-volume-crash-state/v1');hasher.text(name)
    for entry in projected:
        hasher.text(entry['Path'])
        for field in ('Mode','Kind','ModTime'):hasher.number(entry[field])
        hasher.blob(base64.b64decode(entry['Data'],validate=True) if entry['Data'] is not None else b'')
    return hasher.finish()


def apply_effect(volume,effect):
    import copy
    kind=effect['kind'];inode=effect['inode'];objects=volume['objects'];paths=volume['paths']
    if kind=='alloc':
        require(inode not in objects and effect['object'] is not None and effect['object']['inode']==inode,'invalid selected allocation')
        objects[inode]=copy.deepcopy(effect['object'])
    elif kind in ('resize','write','metadata'):
        require(inode in objects,'selected effect uses absent inode')
        obj=objects[inode]
        if kind=='metadata':obj['mode']=effect['mode']
        else:
            require(obj['kind']==1,'selected file effect uses directory')
            if kind=='resize':obj['data']=(obj['data']+b'\0'*max(0,effect['size']-len(obj['data'])))[:effect['size']]
            else:
                start=effect['offset'];end=start+len(effect['data'])
                require(end<=len(obj['data']),'selected write exceeds resized file')
                obj['data']=obj['data'][:start]+effect['data']+obj['data'][end:]
        obj['mod_time']=effect['mod_time']
    elif kind=='namespace':
        for part in effect['namespace']:
            require(part['path']!='/' and part['path'].startswith('/'),'selected namespace path')
            if part['inode']==0:paths.pop(part['path'],None)
            else:
                require(part['inode'] in objects,'selected namespace inode absent');paths[part['path']]=part['inode']
        if inode in objects:objects[inode]['mod_time']=effect['mod_time']
    else:require(False,'unknown selected effect')


def state_sections(raw):
    import struct
    require(56<=len(raw)<=64<<20 and raw[:8]==b'GMSTORE\x01','storage evidence framing')
    end=len(raw)-40
    require(struct.unpack_from('<Q',raw,end)[0]==end and bytes.fromhex(digest(raw[:end]))==raw[end+8:],'storage evidence footer authentication')
    version,count=struct.unpack_from('<II',raw,8)
    require(version in (2,3) and count==6,'storage context requires complete native sections')
    result={};at=16
    for tag in range(1,7):
        require(at+12<=end,'truncated storage evidence section')
        actual,size=struct.unpack_from('<IQ',raw,at);at+=12
        require(actual==tag and size<=end-at,'unordered storage section')
        section=raw[at:at+size]
        result[tag]=decoded(section) if tag in (1,2,3) or (tag==6 and version==3) else section
        at+=size
    require(at==end,'trailing storage section bytes')
    return result


def selected_baseline(row):
    import copy
    inputs,observed=row['inputs'],row['observed']
    require(set(inputs)=={'context','descriptor'} and set(observed)=={'states','image_bytes','state_bytes','log_bytes'},'missing selected baseline raw bytes')
    descriptor_model(inputs['descriptor'],1000,10000)
    image=recovery_image(raw_bytes(observed['image_bytes']));volume=image['volumes']['data']
    require(set(image['volumes'])=={'data'} and volume['mount']=='/workspace' and volume['capacity']==1<<20,'selected baseline volume changed')
    require(len(volume['pending'])==0 and len(volume['history'])==2 and [op['kind'] for op in volume['history']]==['alloc','namespace'],'selected mkdir operation workload changed')
    context={'data':dict(PendingSHA256=pending_identity('data',volume['history']),Operations=[observed_operation(op) for op in volume['history']])}
    require(inputs['context']==context,'selected baseline context not derived from actual image history')
    require(context['data']['Operations'][1]['Path']=='/db' and context['data']['Operations'][0]['Dependencies'] is None,'selected mkdir path/dependency workload changed')
    possible={}
    for mask in range(4):
        chosen=[op for index,op in enumerate(volume['history']) if mask&(1<<index)]
        selected={op['id'] for op in chosen}
        if any(dep not in selected for op in chosen for dep in op['deps']):continue
        changed=copy.deepcopy(volume)
        for op in chosen:apply_effect(changed,op['effect'])
        projected=entries(changed,True);identity=crash_identity('data',projected)
        possible[identity]=projected
    states=observed['states'];require(isinstance(states,list) and len(states)==len(possible)>=2,'selected state enumeration omitted a distinct recoverable state')
    seen=set()
    for state in states:
        require(set(state)=={'Volume','PendingSHA256','SelectedOperations','Entries','Identity'} and state['Volume']=='data'
                and state['PendingSHA256']==context['data']['PendingSHA256'],'selected state identity/context shape')
        selected=state['SelectedOperations'];require(isinstance(selected,list) and all(type(value) is int for value in selected),'selected operation IDs not integers')
        require(context_refusal(['data'],1000,{'data':selected},context) is None,'selected state violates closure')
        changed=copy.deepcopy(volume)
        for op in volume['history']:
            if op['id'] in selected:apply_effect(changed,op['effect'])
        projected=entries(changed,True);identity=crash_identity('data',projected)
        require(state['Entries']==projected and state['Identity']==identity and identity not in seen,'selected enumeration state does not match independent persistence effects')
        seen.add(identity)
    require(seen==set(possible),'selected enumeration omitted reachable state')
    sections=state_sections(raw_bytes(observed['state_bytes']))
    require(sections[6]==context,'baseline storage evidence context differs')
    original=copy.deepcopy(volume)
    for op in volume['history']:apply_effect(original,op['effect'])
    snapshot=recovered_snapshots(image)[0]
    snapshot.update(Volatile=entries(original),PendingOperations=2,PendingSHA256=context['data']['PendingSHA256'])
    snapshot['Identity']=snapshot_identity(snapshot)
    require(canonical(sections[1])==canonical([snapshot]) and canonical(sections[2])==canonical({'data':context['data']['Operations']}),
            'baseline raw snapshot/history differs from independently replayed mkdir effects')
    return dict(image=image,context=context,states={state['Identity']:state for state in states},state_bytes=raw_bytes(observed['state_bytes']),log_bytes=raw_bytes(observed['log_bytes']))


def struct_json(value):
    import json
    return json.dumps(value,separators=(',',':'),ensure_ascii=False,allow_nan=False).encode()


def authenticated_state(raw,log,partial=False):
    import struct
    sections=state_sections(raw)
    require(sections[5]==struct.pack('<Q',len(log))+bytes.fromhex(digest(log)),'storage evidence log authentication')
    if partial:
        require(sections[4]==b'' and sections[6]==b'\x02','context divergence did not preserve failed-stage identity')
    else:
        require(isinstance(sections[6],dict) and re.fullmatch(b'[0-9a-f]{64}',sections[4]),'successful storage image/context identity')
    return sections


def selected_observations(rows):
    import copy
    baseline_rows=[row for row in rows if row['case']=='storage-selected-baseline']
    require(len(baseline_rows)==1,'selected recovery requires one raw baseline')
    baseline=selected_baseline(baseline_rows[0]);expected_states=baseline['states'];seen=set();divergence=set();large=[]
    baseline_sections=authenticated_state(baseline['state_bytes'],baseline['log_bytes'])
    require(baseline_sections[3]['data']['Complete'] is True and baseline_sections[3]['data']['States']==list(expected_states.values()),'baseline enumeration differs from authenticated storage section')
    for row in rows:
        require(set(row)=={'schema','case','inputs','observed'} and row['schema']==SCHEMA,'foreign selected storage observation')
        inputs,observed=row['inputs'],row['observed'];case=row['case']
        if case=='storage-selected-baseline':continue
        if case=='storage-large-observed-selection':large.append(row);continue
        if case=='storage-selected-state-recovery':
            require(set(inputs)=={'context','state'} and inputs['context']==baseline['context'],'selected recovery context changed')
            state=inputs['state'];identity=state['Identity']
            require(identity in expected_states and state==expected_states[identity] and identity not in seen,'unknown/duplicate selected state')
            require(set(observed)=={'image_bytes','recovery_image_sha256','db_present','snapshots'},'selected recovery raw fields missing')
            raw=raw_bytes(observed['image_bytes']);image=recovery_image(raw)
            require(observed['recovery_image_sha256']==digest(raw),'selected recovery image hash differs')
            require(image['model']==baseline['image']['model'] and image['volumes']['data']['history']==baseline['image']['volumes']['data']['history'],'selected recovery history/model changed')
            require(entries(image['volumes']['data'],True)==state['Entries'],'recovery image differs from selected persistence state')
            require(type(observed['db_present']) is bool and observed['db_present']==any(entry['Path']=='/db' for entry in state['Entries'])
                    and canonical(observed['snapshots'])==canonical(recovered_snapshots(image)),'selected recovered namespace/snapshot differs from independent image')
            seen.add(identity)
        elif case=='storage-context-divergence':
            require(set(inputs)=={'variant','context'} and inputs['variant'] in ('hash','history','missing') and inputs['variant'] not in divergence,'unknown/duplicate divergence case')
            variant=inputs['variant'];expected=copy.deepcopy(baseline['context'])
            if variant=='hash':expected['data']['PendingSHA256']='sha256:'+'0'*64
            elif variant=='history':expected['data']['Operations'][0]['Kind']='forged'
            else:expected['data']['Operations']=[]
            require(inputs['context']==expected and expected!=baseline['context'],'divergence raw inventory did not change original history')
            require(set(observed)=={'failure','image','state_bytes','log_bytes','original_state_bytes','original_log_bytes','state_sha256'},'divergence raw sections missing')
            raw=raw_bytes(observed['state_bytes']);log=raw_bytes(observed['log_bytes'])
            require(raw_bytes(observed['original_state_bytes'])==baseline['state_bytes'] and raw_bytes(observed['original_log_bytes'])==baseline['log_bytes'] and log==baseline['log_bytes'],'divergence lost original observations/log')
            sections=authenticated_state(raw,log,True)
            require(all(sections[tag]==baseline_sections[tag] for tag in (1,2,3)),'divergence lost original snapshot/history/enumeration')
            require(observed['failure']==['invalid','storage crash context diverged from prior observation'] and observed['image'] is None
                    and observed['state_sha256']==digest(raw),'divergence classification/image/hash mismatch')
            divergence.add(variant)
        else:require(False,'foreign selected storage case')
    require(seen==set(expected_states) and divergence=={'hash','history','missing'} and len(large)==1,'selected storage omitted original states/divergences/large selection')
    row=large[0];inputs,observed=row['inputs'],row['observed']
    require(set(inputs)=={'files','path_bytes','context','context_sha256','selection'} and type(inputs['files']) is int and inputs['files']==50
            and type(inputs['path_bytes']) is int and inputs['path_bytes']==4000,'large observed selection cardinality changed')
    require(set(observed)=={'image_bytes','snapshots','image_sha256','snapshots_sha256'},'large observed selection raw image/snapshot missing')
    raw=raw_bytes(observed['image_bytes']);image=recovery_image(raw)
    require(set(image['volumes'])=={'data'},'large selected volume inventory changed')
    volume=image['volumes']['data'];history=volume['history']
    context={'data':dict(PendingSHA256=pending_identity('data',history),Operations=[observed_operation(op) for op in history])}
    require(len(history)==100 and inputs['context']==context and inputs['context_sha256']==digest(canonical(context))
            and len(canonical(context))>1<<20 and inputs['selection']=={'data':list(range(1,101))},'large observed context lost actual history/full selection')
    require(volume['mount']=='/workspace' and volume['capacity']==1<<20 and not volume['pending'],'large selected image configuration/pending mismatch')
    expected_paths={'/'}|{'/'+'\x01'*3998+f'{index:02d}' for index in range(50)}
    require(set(volume['paths'])==expected_paths and all(volume['objects'][inode]['kind']==(2 if path=='/' else 1)
            and not volume['objects'][inode]['data'] for path,inode in volume['paths'].items()),'large selection did not recover all exact 4000-byte file names')
    initial=copy.deepcopy(baseline['image']['volumes']['data'])
    for index in range(50):
        allocation,namespace=history[2*index:2*index+2]
        path='/'+'\x01'*3998+f'{index:02d}'
        require(allocation['kind']=='alloc' and namespace['kind']=='namespace'
                and allocation['effect']['object']['kind']==1 and allocation['effect']['object']['mode']==0o600
                and allocation['effect']['object']['data']==b''
                and namespace['effect']['namespace']==[dict(path=path,inode=allocation['effect']['inode'])],
                'large observed selection did not execute exact 50-file allocation/namespace workload')
        apply_effect(initial,allocation['effect']);apply_effect(initial,namespace['effect'])
    require(entries(initial,True)==entries(volume,True),'large selected image differs from independently replayed creation effects')
    snapshots=recovered_snapshots(image)
    require(canonical(observed['snapshots'])==canonical(snapshots) and observed['image_sha256']==digest(raw)
            and observed['snapshots_sha256']==digest(struct_json(snapshots)),'large selected recovered snapshots differ from raw image')
    return {'TestStorageCrashContextSelectedOwnersAndDivergence':dict(states=sorted(seen),divergences=sorted(divergence),
        lineage='single native storage owner; retired remote/local route has no native branch',
        remaining_original_predicates=['actual engine divergence peak memory must be nonzero']),
        'TestStorageCrashContextLargeObservedSelection':dict(files=50,path_bytes=4000,operations=100,
        lineage='single native storage owner; retired remote/local route has no native branch')}


def state_raw_sections(raw):
    import struct
    require(56<=len(raw)<=64<<20 and raw[:8]==b'GMSTORE\x01','native state framing')
    end=len(raw)-40
    require(int.from_bytes(raw[end:end+8],'little')==end and bytes.fromhex(digest(raw[:end]))==raw[end+8:],'native state authentication')
    version,count=struct.unpack_from('<II',raw,8)
    require((version,count) in ((1,5),(2,6),(3,6)),'native state version/count')
    parts=[];at=16
    for tag in range(1,count+1):
        require(at+12<=end,'native state section truncation')
        actual,size=struct.unpack_from('<IQ',raw,at);at+=12
        require(actual==tag and size<=end-at,'native state section tag/bound')
        parts.append(raw[at:at+size]);at+=size
    require(at==end,'native state trailing bytes')
    return version,parts


def state_envelope(version,parts):
    import struct
    body=b'GMSTORE\x01'+struct.pack('<II',version,len(parts))
    body+=b''.join(struct.pack('<IQ',index,len(part))+part for index,part in enumerate(parts,1))
    return body+struct.pack('<Q',len(body))+bytes.fromhex(digest(body))


def evidence_observations(rows,selected):
    import copy
    required={('storage.native-original-state',None)}
    required.update(('storage.native-state-context',edit) for edit in ('valid','snapshot-hash','operation-history','missing-inventory'))
    required.update(('storage.native-state-context-presence',edit) for edit in ('v1-with-limits','v3-without-limits','forged-log'))
    required.update(('storage.native-partial-state',stage) for stage in (1,2,3))
    mutations=('snapshot-null-entry','snapshot-identity','snapshot-unknown-field','snapshot-duplicate-field','history-duplicate-key',
        'history-unknown-field','enumeration-json','enumeration-identity','enumeration-unknown-field','enumeration-duplicate-key',
        'image-identity','image-uppercase','context-duplicate-key')
    required.update(('storage.native-resealed-section',mutation) for mutation in mutations)
    originals=[row for row in rows if row['case']=='storage.native-original-state']
    require(len(originals)==1,'storage state needs exact original raw baseline')
    original=originals[0];inputs=original['inputs'];observed=original['observed']
    require(set(inputs)=={'storage'} and set(observed)=={'state_bytes','log_bytes','image_bytes'},'original storage state shape')
    raw=raw_bytes(observed['state_bytes']);log=raw_bytes(observed['log_bytes']);image_raw=raw_bytes(observed['image_bytes'])
    baseline=selected_baseline(next(row for row in selected if row['case']=='storage-selected-baseline'))
    require(raw==baseline['state_bytes'] and log==baseline['log_bytes'],'storage evidence baseline differs from independently replayed actual mkdir owner')
    require(recovery_image(image_raw)==baseline['image'],'storage original image differs from actual selected owner baseline')
    _,parts=state_raw_sections(raw);storage=inputs['storage']
    descriptor_model(storage['descriptor'],1000,10000)
    require(storage['descriptor']['crash_limits']==dict(States=16,Operations=8,Depth=8,Bytes=1<<20,Work=256),'native evidence storage crash bounds changed')
    seen={('storage.native-original-state',None)}
    for row in rows:
        case=row['case']
        if case=='storage.native-original-state':continue
        require(set(row)=={'schema','case','inputs','observed'} and row['schema']==SCHEMA,'foreign storage evidence observation')
        inputs,observed=row['inputs'],row['observed'];selector='stage' if case=='storage.native-partial-state' else 'mutation' if case=='storage.native-resealed-section' else 'edit'
        require(set(inputs)=={selector,'state_base64','log_base64','storage'} and set(observed)=={'error'},'storage evidence missing raw inputs')
        key=(case,inputs[selector]);require(key in required and key not in seen,'unknown/duplicate native state case')
        seen.add(key);changed=parts.copy();version=3;expected_storage=copy.deepcopy(storage);expected_log=log
        expected_error=True;error_fragment=None
        if case=='storage.native-state-context':
            context=decoded(parts[5]);edit=inputs['edit']
            if edit=='snapshot-hash':context['data']['PendingSHA256']='sha256:'+'b'*64
            elif edit=='operation-history':context['data']['Operations'][0]['Kind']='forged'
            elif edit=='missing-inventory':context['data']['Operations']=None
            else:expected_error=False
            changed[5]=canonical(context)
            error_fragment='retained storage evidence: crash context' if expected_error else None
        elif case=='storage.native-state-context-presence':
            edit=inputs['edit']
            if edit=='v1-with-limits':version=1;changed=changed[:5]
            elif edit=='v3-without-limits':expected_storage['descriptor']['crash_limits']=None
            else:expected_log=b'forged'
            error_fragment='retained storage log binding' if edit=='forged-log' else 'retained storage evidence: crash context'
        elif case=='storage.native-partial-state':
            stage=inputs['stage'];unsigned(stage);version=2;changed[3]=b'';changed[5]=bytes([stage]);expected_error=stage==3
            error_fragment='retained storage failed stage' if expected_error else None
        else:
            mutation=inputs['mutation']
            if mutation in ('snapshot-null-entry','snapshot-identity','snapshot-unknown-field'):
                value=decoded(parts[0])
                if mutation=='snapshot-null-entry':value[0]['Persisted']=[None]
                elif mutation=='snapshot-identity':value[0]['Identity']='x'
                else:value[0]['unexpected']=1
                changed[0]=canonical(value)
            elif mutation=='snapshot-duplicate-field':changed[0]=parts[0].replace(b'"ID":"data"',b'"ID":"data","ID":"data"',1)
            elif mutation=='history-unknown-field':
                value=decoded(parts[1]);value['data'][0]['unexpected']=1;changed[1]=canonical(value)
            elif mutation=='enumeration-json':changed[2]=b'{'
            elif mutation in ('enumeration-identity','enumeration-unknown-field'):
                value=decoded(parts[2])
                if mutation=='enumeration-identity':value['data']['States'][0]['Identity']='x'
                else:value['data']['unexpected']=1
                changed[2]=canonical(value)
            elif mutation=='image-identity':changed[3]=b'garbage'
            elif mutation=='image-uppercase':changed[3]=b'A'*64
            else:
                section={'history-duplicate-key':1,'enumeration-duplicate-key':2,'context-duplicate-key':5}[mutation]
                payload=canonical(decoded(parts[section])['data'])
                changed[section]=b'{"data":'+payload+b',"data":'+payload+b'}'
            if 'duplicate' in mutation:error_fragment='duplicate JSON key'
            elif 'unknown-field' in mutation:error_fragment='unknown field'
            elif mutation.startswith('image-'):error_fragment='retained storage image identity'
            elif mutation=='snapshot-identity':error_fragment='retained storage snapshot inventory'
            elif mutation=='enumeration-identity':error_fragment='retained storage crash state identity'
        try:
            actual_raw=base64.b64decode(inputs['state_base64'],validate=True);actual_log=base64.b64decode(inputs['log_base64'],validate=True)
        except (ValueError,TypeError):require(False,'storage evidence raw base64 invalid')
        require(actual_raw==state_envelope(version,changed) and actual_log==expected_log and inputs['storage']==expected_storage,'native storage state mutation differs from exact raw original variant')
        error=observed['error']
        require((isinstance(error,str) and bool(error)) if expected_error else error is None,'native storage evidence admission/refusal differs from raw semantics')
        if error_fragment:require(error_fragment in error,'native storage evidence refused at the wrong semantic guard')
    require(seen==required,'native state observation omitted an original context/refusal/presence or structural variant')
    return {'TestStorageCrashContextEvidenceRefusal':dict(cases=[list(key) for key in sorted(seen,key=str)],
        lineage='authenticated native GMSTORE sections; semantic refusals preserve original context/presence invariants')}
