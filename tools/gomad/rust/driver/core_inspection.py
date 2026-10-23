"""Independent receiver for native inspection lineage and bounded search calls."""
import base64
import hashlib
from pathlib import Path

from driver import require
import native_roles

SCHEMA = 'gomad.native-inspection-predicate/v1'
PREFIX = 'TestInspection'
VARIANTS = {
    'CapacityRetainsTemporalScaleBoundariesAndEffects': {'51000-decisions-376000-effects', 'native-retained-scale-admission'},
    'ChoiceMappingUsesDecisionsRatherThanObservations': {'observation-and-inventory', 'production-retained-resolve-gate'},
    'ChoiceMappingUsesPredecisionCoordinateForCreationSites': {'five-creation-groups'},
    'ControlAuthenticatesRawCutAndRegistry': {'choice2'},
    'ControlRefusesUnregisteredAndIncompleteBeforeReplay': {'missing-check-and-incomplete'},
    'InterruptedSelectPrefixKeepsDecisionMapping': {'unmatched-select-poll'},
    'NetworkRawStreamBindsCanonicalIdentity': {'native-terminal-raw-state'},
    'SearchBisectDeclaresConditionalityAndRejectsObservedReversal': {'first_failure', 'already_failing_at_first', 'no_failure', 'monotonicity_violation'},
    'SearchBudgetsAndPredicateErrorsStayInconclusive': {'pass/budget2/verifiedtrue', 'panic/budget10/verifiedtrue', 'timeout/budget10/verifiedtrue', 'divergence/budget10/verifiedtrue', 'pass/budget10/verifiedfalse'},
    'SearchRejectsRequestsBeforeEvaluation': {'unknown-check', 'invalid-range', 'zero-attempts', 'zero-wall', 'unreachable-choice', 'nonmonotonic'},
    'SearchScanDoesNotSkipRecoveredFailures': {'recovered-at2-fails1-and4'},
    'SearchWallBoundIncludesEvaluatorCleanup': {'cleanup-exhausted-wall', 'already-expired', 'external-cancel-joined-store'},
}
ROOT_CASES = {PREFIX + root: variants for root, variants in VARIANTS.items()}


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def binary(value):
    require(isinstance(value, str), 'inspection binary must be base64')
    result = base64.b64decode(value, validate=True)
    require(base64.b64encode(result).decode() == value, 'noncanonical inspection binary')
    return result


class Reader:
    def __init__(self, raw):
        self.raw, self.at = raw, 0

    def take(self, count):
        require(0 <= count <= len(self.raw) - self.at, 'truncated inspection binary')
        result = self.raw[self.at:self.at + count]
        self.at += count
        return result

    def number(self, width=8):
        return int.from_bytes(self.take(width), 'little')

    def handle(self):
        size = self.number(4)
        require(0 < size <= 1024, 'inspection handle bound')
        node = self.take(size).decode()
        require(not any(c in node for c in '\0\r\n'), 'inspection handle framing')
        return dict(node=node, incarnation=self.number())


def decode_evidence(evidence):
    require(evidence['schema'] == 'gomad.wasm-inspection/v3' and evidence['boundary_encoding'] == 'gomad.inspection-boundaries-binary/v1'
            and evidence['lifecycle_encoding'] == 'gomad.inspection-lifecycle-binary/v1' and evidence['state_encoding'] == 'gomad.inspection-state-sha256/v2'
            and evidence['world_encoding'] == 'gomad.inspection-world-sha256/v2', 'foreign inspection lineage')
    frames = Reader(binary(evidence['boundary_frames']))
    require(frames.take(8) == b'GMWIB\1\0\0', 'inspection boundary header')
    count = frames.number(4)
    require(count <= 65536, 'inspection boundary capacity')
    boundaries = []
    for _ in range(count):
        frame = Reader(frames.take(frames.number(4)))
        boundary = dict(handle=frame.handle())
        boundary.update((key, frame.number()) for key in ('decision', 'trace', 'order', 'time_nanos', 'world_bytes', 'lifecycle_bytes'))
        cuts = frame.number(4)
        require(0 < cuts <= 65536, 'inspection cut capacity')
        boundary['cuts'] = []
        for _ in range(cuts):
            cut = dict(handle=frame.handle(), state_encoding=evidence['state_encoding'])
            cut.update((key, frame.number()) for key in ('choice_records', 'choice_bytes', 'wasi_bytes', 'network_bytes', 'storage_bytes', 'stdout_bytes', 'stderr_bytes'))
            clock_size = frame.number(4)
            require(clock_size <= 1024, 'inspection clock bound')
            cut['clock_run'] = base64.b64encode(frame.take(clock_size)).decode()
            for key in ('network_state', 'storage_state', 'recovery_state'):
                cut[key] = base64.b64encode(frame.take(32)).decode()
            runtime = frame.take(128)
            words = lambda at, size=8: int.from_bytes(runtime[at:at + size], 'big')
            counters = [words(40 + i * 8) for i in range(7)]
            require(words(104) <= words(0) and words(112) > 0, 'inspection compulsory witness')
            cut['witness'] = dict(allocations=words(16), allocated_bytes=words(96), gc_cycles=words(24, 4), gc_phase=runtime[28],
                                  goroutines=words(112), run_queue=words(32, 4), global_run_queue=words(120), seeded_draws=sum(counters) % (1 << 64),
                                  seeded_draw_counters=counters, choices=words(104), observations=words(0)-words(104), runtime_state=base64.b64encode(runtime).decode())
            boundary['cuts'].append(cut)
        require(frame.at == len(frame.raw), 'inspection trailing frame')
        boundaries.append(boundary)
    require(frames.at == len(frames.raw), 'inspection trailing boundaries')
    lifecycle = Reader(binary(evidence['lifecycle']))
    require(lifecycle.take(8) == b'GMWIL\1\0\0', 'inspection lifecycle header')
    events = []
    while lifecycle.at < len(lifecycle.raw):
        events.append((lifecycle.number(1), lifecycle.handle()))
    world = binary(evidence['world'])
    require(world.startswith(b'gomad.inspection-world-commitments/v2\0'), 'inspection World header')
    handles = []
    for owner in evidence['incarnations']:
        require(owner['handle'] not in handles, 'duplicate inspection incarnation')
        handles.append(owner['handle'])
        for stream in ('choice', 'wasi', 'network', 'storage', 'stdout', 'stderr'):
            binary(owner['streams'][stream])
    for boundary in boundaries:
        require(boundary['world_bytes'] <= len(world) and boundary['lifecycle_bytes'] <= len(lifecycle.raw), 'inspection raw prefix bound')
        for cut in boundary['cuts']:
            require(cut['handle'] in handles, 'inspection cut owner missing')
            owner = evidence['incarnations'][handles.index(cut['handle'])]
            for stream in ('choice', 'wasi', 'network', 'storage', 'stdout', 'stderr'):
                require(cut[stream + '_bytes'] <= len(binary(owner['streams'][stream])), 'inspection stream prefix bound')
    return boundaries, events


def material(value):
    boundaries, events = decode_evidence(value['evidence'])
    require(boundaries == value['boundaries'], 'inspection decoded boundaries differ from actual API')
    return value['evidence'], boundaries, events


def mapping(evidence, boundaries):
    result = []
    for owner in evidence['incarnations']:
        raw = binary(owner['streams']['choice'])
        require(len(raw) % 96 == 0, 'inspection choice framing')
        coordinates, creation = [], None
        for index in range(len(raw)//96):
            record = raw[index*96:(index+1)*96]
            require(int.from_bytes(record[:8], 'big') == index, 'inspection choice ordinal')
            kind = record[8]
            require(kind in (1, 2, 3, 4), 'inspection choice kind')
            if kind == 4:
                if creation is None:
                    creation = index
            elif kind in (1, 2):
                coordinates.append(index if creation is None else creation)
                creation = None
        actual = [boundary['trace'] for boundary in boundaries if boundary['handle'] == owner['handle']]
        require(actual == coordinates, 'inspection decision/trace mapping differs')
        for boundary in boundaries:
            if boundary['handle'] == owner['handle']:
                cut = next(cut for cut in boundary['cuts'] if cut['handle'] == owner['handle'])
                require(cut['choice_records'] == boundary['trace'] and cut['choice_bytes'] == boundary['trace']*96, 'inspection cut choice prefix differs')
        result.extend(actual)
    return result


def search_report(observed):
    inputs = observed['raw_input']
    request = inputs['search']
    clocks = iter(int(value) for value in inputs['clock_nanos'])
    calls = iter(inputs['calls'])
    cancellations = iter(inputs.get('cancelled', []))
    def cancelled():
        if 'cancelled' not in inputs:
            return False
        value=next(cancellations, None)
        require(type(value) is bool, 'missing search cancellation sample')
        return value
    evaluated, cache = [], {}
    first, last, mode = request['first'], request['last'], request['mode']
    result = dict(status='', evaluations=None, remaining=dict(first=first, last=last), conditional_on_monotonicity=mode == 'bisect')
    def clock():
        value = next(clocks, None)
        require(value is not None and value >= 0, "missing search elapsed clock")
        return value
    def evaluate(choice):
        if choice in cache:
            return cache[choice]
        elapsed = clock()
        if len(evaluated) >= request['attempt_budget'] or elapsed >= request['wall_time'] or cancelled():
            result['status'] = 'incomplete'
            return None
        call = next(calls, None)
        require(isinstance(call, dict), "missing search evaluator call")
        require(call['choice'] == choice and int(call['remaining_nanos']) == request['wall_time'] - elapsed, 'search actual call/budget differs')
        timeout = clock() >= request['wall_time'] or cancelled()
        evaluated.append(dict(choice=choice, outcome='timeout' if timeout else call['outcome'], prefix_verified=call['prefix_verified'], detail='context deadline exceeded' if timeout else call['detail']))
        result['evaluations'] = evaluated
        status = ('incomplete' if timeout else 'replay_divergence' if call['outcome'] == 'divergence' or call['outcome'] in ('pass', 'fail') and not call['prefix_verified'] else 'invariant_evaluation_failure' if call['outcome'] not in ('pass', 'fail') else '')
        if status:
            result['status'] = status
            return None
        passed = call['outcome'] == 'pass'
        cache[choice] = passed
        if mode == 'bisect' and any(not value and passed and position < choice or value and not passed and position > choice for position, value in cache.items()):
            result['status'] = 'monotonicity_violation'
            return None
        return passed
    def finish(failure=None):
        result.pop('remaining')
        result['status'] = 'no_failure' if failure is None else 'already_failing_at_first' if failure == first else 'first_failure'
        if failure is not None:
            result['first_failure'] = failure
    if mode == 'scan':
        for choice in range(first, last+1):
            result['remaining'] = dict(first=choice, last=last)
            value = evaluate(choice)
            if value is None:
                break
            if not value:
                finish(choice)
                break
            if choice == last:
                finish()
    else:
        low, high = first, last
        low_pass = evaluate(low)
        if low_pass is not None:
            high_pass = low_pass if low == high else evaluate(high)
            if high_pass is not None:
                if not low_pass:
                    finish(low)
                elif high_pass:
                    finish()
                else:
                    while high-low > 1:
                        result['remaining'] = dict(first=low+1, last=high)
                        middle = low+(high-low)//2
                        value = evaluate(middle)
                        if value is None:
                            break
                        if value:
                            low = middle
                        else:
                            high = middle
                    else:
                        finish(high)
    require(next(clocks, None) is None and next(calls, None) is None and next(cancellations,None) is None, 'search duplicate or unused raw call/clock')
    require({key:value for key,value in observed.items() if key not in ('raw_input','cleanup')} == result, 'search report differs from independently evaluated calls')
    return result


def refused_mapping(value):
    try:
        mapping(value['evidence'], value['boundaries'])
    except Exception as error:
        from driver import Refusal
        require(isinstance(error, Refusal), 'inspection reader internal failure')
        return
    require(False, 'forged inspection refusal has valid decision mapping')


def commitment(kind, raw):
    domain = b'gomad.inspection-world-sha256/v2\0' if kind=='world' else b'gomad.inspection-state-sha256/v2/network\0' if kind=='network' else b'gomad.inspection-state-sha256/v1/'+kind.encode()+b'\0'
    return base64.b64encode(hashlib.sha256(domain+len(raw).to_bytes(8,'little')+raw).digest()).decode()


SCALE_COMMITMENTS={kind:commitment(kind, b'm'*(64<<10) if kind=='storage' else b'') for kind in ('network','storage','recovery')}


def scale_cut(index, effects, handle):
    runtime=bytearray(128)
    runtime[:8]=index.to_bytes(8,'big');runtime[104:112]=index.to_bytes(8,'big');runtime[112:120]=(1).to_bytes(8,'big')
    witness=dict(allocations=0,allocated_bytes=0,gc_cycles=0,gc_phase=0,goroutines=1,run_queue=0,global_run_queue=0,
                 seeded_draws=0,seeded_draw_counters=[0]*7,choices=index,observations=0,runtime_state=base64.b64encode(runtime).decode())
    return dict(handle=handle,state_encoding='gomad.inspection-state-sha256/v2',choice_records=index,choice_bytes=index*96,wasi_bytes=8+effects*16,
                network_bytes=0,storage_bytes=0,stdout_bytes=0,stderr_bytes=0,clock_run='',network_state=SCALE_COMMITMENTS['network'],
                storage_state=SCALE_COMMITMENTS['storage'],recovery_state=SCALE_COMMITMENTS['recovery'],witness=witness)


def validate_case(root, variant, observed, artifact_roots=()):
    name = root.removeprefix(PREFIX)
    if name.startswith('Search') and name != 'SearchRejectsRequestsBeforeEvaluation':
        result = search_report(observed)
        inputs = observed['raw_input']
        request = inputs['search']
        require(request == dict(check='healthy', first=0, last=4, mode='bisect' if name.startswith('SearchBisect') else 'scan',
                                attempt_budget=2 if variant == 'pass/budget2/verifiedtrue' else 10, wall_time=1000000000), 'search original request differs')
        if name == 'SearchBisectDeclaresConditionalityAndRejectsObservedReversal':
            values = {'first_failure':[True,True,False,False,False], 'already_failing_at_first':[False]*5,
                      'no_failure':[True]*5, 'monotonicity_violation':[False,True,False,False,True]}[variant]
            require(result['status'] == variant and all(call['prefix_verified'] and call['outcome'] == ('pass' if values[call['choice']] else 'fail') for call in inputs['calls']), 'search original bisect predicate differs')
        if name == 'SearchBudgetsAndPredicateErrorsStayInconclusive':
            outcome, _, verified = variant.split('/')
            require(all(call['outcome']==outcome and call['prefix_verified']==(verified=='verifiedtrue') for call in inputs['calls']), 'search original error predicate differs')
        if name == 'SearchScanDoesNotSkipRecoveredFailures':
            require([call['choice'] for call in inputs['calls']] == [0,1] and [call['outcome'] for call in inputs['calls']] == ['pass','fail'] and result['first_failure'] == 1, 'search recovered failure skipped')
        if name == 'SearchWallBoundIncludesEvaluatorCleanup' and variant == 'external-cancel-joined-store':
            cleanup=observed['cleanup']
            require(inputs['cancelled']==[False,True] and len(inputs['calls'])==1 and inputs['calls'][0]['choice']==0
                    and result['status']=='incomplete' and result['evaluations'][0]['outcome']=='timeout'
                    and cleanup['worker_termination']=='cancelled'
                    and int(inputs['clock_nanos'][0])<=int(cleanup['store_joined_nanos'])<=int(inputs['clock_nanos'][1])<=int(cleanup['returned_nanos']), 'external search cancellation failed to join evaluator store before return')
        elif name == 'SearchWallBoundIncludesEvaluatorCleanup':
            require(inputs['clock_nanos'] == (['1000000','1000000000'] if variant == 'cleanup-exhausted-wall' else ['1000000000']), 'search cleanup clock units differ')
        return dict(variant=variant, calls=inputs['calls'], clock_nanos=inputs['clock_nanos'], status=result['status'])
    if variant == 'native-retained-scale-admission':
        path = Path(observed['artifact'])
        require(path.is_absolute() and any(path.is_relative_to(Path(directory)) for directory in artifact_roots), 'inspection retained scale outside assigned custody')
        require(path.resolve() == path, 'inspection retained scale symlink ancestor')
        manifest_raw = native_roles.regular(path/'manifest.json').read_bytes()
        manifest = native_roles.decode(manifest_raw)
        record_raw = native_roles.regular(path/'record.json').read_bytes()
        record = native_roles.decode(record_raw)
        require(manifest == observed['manifest'] and record == observed['record'] == manifest['record']
                and manifest['schema']=='gomad.artifact-manifest/v2' and record['schema']=='gomad.execution-record/v2', 'inspection retained scale native record/manifest differ')
        require(observed['maximum_bytes']==512<<20 and len(record_raw)==observed['record_bytes'] and len(record_raw)<=16<<20
                and manifest['record_sha256']==sha(record_raw), 'inspection retained scale record identity/bound differs')
        total=0; payloads={}
        for name, descriptor in manifest['files'].items():
            require(not Path(name).is_absolute() and all(part not in ('..','.') for part in Path(name).parts), 'inspection retained payload path')
            child=path/name
            require(child.resolve()==child, 'inspection retained payload symlink ancestor')
            raw=native_roles.regular(child).read_bytes()
            require(len(raw)==descriptor['bytes'] and sha(raw)==descriptor['sha256'], 'inspection retained actual payload identity differs')
            payloads[name]=raw; total+=len(raw)
        envelope=native_roles.regular(path/'storage.json').read_bytes()
        require(envelope==b'{"format":"gomad.artifact-storage/v2"}', 'inspection retained scale storage lineage differs')
        require(total==observed['payload_bytes'] and total+len(manifest_raw)+len(envelope)==observed['stored_bytes']<=512<<20,
                'inspection retained scale actual total exceeds original retained bound')
        for name, descriptor in record['streams'].items():
            require(name in payloads and len(payloads[name])==descriptor['bytes'] and sha(payloads[name])==descriptor['sha256'], 'inspection retained record stream binding differs')
        raw=payloads['evidence/inspection']; evidence=native_roles.decode(raw)
        boundaries,_=decode_evidence(evidence)
        owner=evidence['incarnations'][0]
        require(len(boundaries)==51000 and raw==native_roles.regular(Path(observed['raw_file'])).read_bytes() and sha(raw)==observed['raw_sha256'], 'inspection retained scale exact metadata differs')
        require(payloads['evidence/world']==binary(evidence['world']) and payloads['evidence/choice']==payloads['evidence/choice-tape']==binary(owner['streams']['choice'])
                and payloads['evidence/transcript']==binary(owner['streams']['wasi']), 'inspection retained scale omitted raw owner payload')
        return dict(variant=variant, raw_sha256=sha(raw), stored_bytes=observed['stored_bytes'], native_lineage=manifest['schema'])
    if name == 'CapacityRetainsTemporalScaleBoundariesAndEffects':
        path = Path(observed['raw_file'])
        require(path.is_absolute() and any(path.is_relative_to(Path(directory)) for directory in artifact_roots), 'inspection scale file outside assigned custody')
        require(path.resolve()==path, 'inspection scale raw symlink ancestor')
        raw = native_roles.regular(path).read_bytes()
        require(sha(raw) == observed['raw_sha256'] and len(raw) == observed['metadata_bytes'] and len(raw) <= 64<<20, 'inspection scale raw identity/bound')
        evidence = native_roles.decode(raw)
        boundaries, events = decode_evidence(evidence)
        require(evidence['complete'] and len(boundaries) == 51000 and len(events) == 427002, 'inspection scale event inventory differs')
        owner = evidence['incarnations'][0]
        require(len(evidence['incarnations']) == 1 and owner['handle'] == dict(node='history', incarnation=1), 'inspection scale owner')
        require(binary(owner['streams']['choice']) == bytes(51000*96), 'inspection scale complete choice bytes differ')
        wasi = binary(owner['streams']['wasi'])
        require(wasi[:8] == b'GMWT\1\0\0\0' and len(wasi) == observed['wasi_bytes'] == 8+376000*16, 'inspection scale WASI bytes')
        require(wasi[8:] == b''.join(bytes([2,0,0,0]) + number.to_bytes(8,'little') + bytes(4) for number in range(1,376001)), 'inspection scale raw WASI effect inventory differs')
        require(sum(tag == 6 for tag, _ in events) == 376000, 'inspection scale lifecycle model effect inventory differs')
        effects = 0
        for index, boundary in enumerate(boundaries):
            effects += 8 if index < 19000 else 7
            cut = boundary['cuts'][0]
            order=effects+index+2
            require(boundary==dict(handle=owner['handle'],decision=index,trace=index,order=order,time_nanos=index,
                                  world_bytes=len(b'gomad.inspection-world-commitments/v2\0')+(index+1)*40,lifecycle_bytes=8+order*20,
                                  cuts=[scale_cut(index,effects,owner['handle'])]), 'inspection scale omitted or changed any boundary/cut/witness field')
        world_header=b'gomad.inspection-world-commitments/v2\0'
        world_entry=(32).to_bytes(8,'little')+binary(commitment('world',b''))
        require(binary(evidence['world'])==world_header+world_entry*51000, 'inspection scale full World bytes differ')
        expected_events=[(1,owner['handle'])]
        for index in range(51000):
            expected_events.extend([(6,owner['handle'])]*(8 if index<19000 else 7));expected_events.append((2,owner['handle']))
        expected_events.append((3,owner['handle']))
        require(events==expected_events and owner['started_order']==1 and owner['terminal_order']==427002, 'inspection scale complete lifecycle order differs')
        require(effects == 376000 and observed['effects'] == effects and observed['boundaries'] == len(boundaries)
                and observed['lifecycle_bytes'] == len(binary(evidence['lifecycle'])) and observed['choice_bytes'] == 51000*96, 'inspection scale claimed counts differ')
        return dict(variant=variant, raw_sha256=sha(raw), boundaries=len(boundaries), effects=effects)
    if name == 'SearchRejectsRequestsBeforeEvaluation':
        evidence, boundaries, _ = material(dict(evidence=observed['evidence'], boundaries=observed['boundaries']))
        request = observed['search']
        require(observed['error'] and observed['search_entered'] is False, 'search admission missing native refusal')
        invalid = {'unknown-check':request['check']=='absent', 'invalid-range':request['first']>request['last'],
                   'zero-attempts':request['attempt_budget']==0, 'zero-wall':request['wall_time']==0,
                   'unreachable-choice':request['last']>=len(boundaries),
                   'nonmonotonic':request['mode']=='bisect' and not evidence['incarnations'][0]['registry']['entries'][0]['monotonic']}
        require(invalid[variant], 'forged search admission refusal')
        return dict(variant=variant, error=observed['error'])
    if variant == 'production-retained-resolve-gate':
        evidence, boundaries, _ = material(dict(evidence=observed['evidence'], boundaries=observed['boundaries']))
        require(mapping(evidence,boundaries) == [0,1,2,3,4] and observed['expected'] == boundaries[2], 'retained control valid input differs')
        refused_mapping(dict(evidence=evidence,boundaries=observed['invalid_boundaries']))
        require('decision/trace mapping' in observed['error'], 'retained gate missing native error')
        return dict(variant=variant,error=observed['error'])
    evidence, boundaries, events = material(observed['raw'])
    if name.startswith('ChoiceMapping') or name.startswith('Interrupted'):
        coordinates = mapping(evidence,boundaries)
        expected = [0,2,3,4,5] if variant=='observation-and-inventory' else [0,3,6,9,12] if variant=='five-creation-groups' else [0]
        require(coordinates == expected, 'inspection original coordinate inventory differs')
        if variant == 'observation-and-inventory':
            require(len(observed['invalid']) == len(observed['errors']) == 3 and observed['errors']==['inspection decision/trace mapping','incomplete inspection decision boundary inventory','inspection choice prefix length'], 'inspection missing native refusal variants')
            for invalid in observed['invalid']:
                refused_mapping(invalid)
        elif variant=='five-creation-groups':
            refused_mapping(observed['invalid'])
            require(observed['error']=='inspection decision/trace mapping', 'inspection missing creation refusal')
        return dict(variant=variant, coordinates=coordinates)
    if name == 'ControlAuthenticatesRawCutAndRegistry':
        require(mapping(evidence,boundaries)==list(range(5)) and observed['expected']==boundaries[2], 'inspection resolved wrong boundary')
        boundary = boundaries[2]
        require(observed['world_sha256']==sha(binary(evidence['world'])[:boundary['world_bytes']]), 'inspection World prefix digest differs')
        require(observed['stream_sha256'][0]['choice']==sha(binary(evidence['incarnations'][0]['streams']['choice'])[:boundary['cuts'][0]['choice_bytes']]), 'inspection choice prefix digest differs')
        registry = evidence['incarnations'][0]['registry']
        require(observed['registries']==[registry] and registry['closed'] and registry['entries']==[dict(id=0,kind='check',name='healthy',available_order=1,monotonic=True)], 'inspection registry differs')
        require(observed['commands']==[dict(kind=4,id=0)] and observed['changed_boundary']==dict(boundary,order=boundary['order']+1) and observed['error'], 'inspection command/refusal differs')
    elif name == 'ControlRefusesUnregisteredAndIncompleteBeforeReplay':
        require(evidence['complete'] and all(entry['name']!='missing' for owner in evidence['incarnations'] for entry in owner['registry']['entries']), 'inspection missing-check actually registered')
        require(observed['incomplete']['evidence']==dict(evidence,complete=False) and observed['incomplete']['boundaries']==boundaries
                and observed['errors'][0].find('unknown or unavailable')>=0 and observed['errors'][1]=='unsupported or incomplete WASM inspection evidence', 'inspection incomplete refusal input differs')
    elif name == 'NetworkRawStreamBindsCanonicalIdentity':
        network = binary(observed['network_base64'])
        require(sha(network)=='00a0074b5731ede6751ec6de87fdcdc3df56ebd6ef2bf2b1357cfef11c97465a' and network.startswith(b'GMNETEV1'), 'inspection frozen network bytes differ')
        require(binary(evidence['incarnations'][0]['streams']['network'])==network and observed['error'] and observed['trailing_error'], 'inspection network owner/refusals differ')
        commitment = hashlib.sha256(b'gomad.inspection-state-sha256/v2/network\0' + len(network).to_bytes(8,'little') + network).digest()
        terminal = next(cut for cut in boundaries[1]['cuts'] if cut['handle']['node']=='a')
        require(binary(terminal['network_state']) == commitment and terminal['network_bytes']==len(network), 'inspection terminal raw commitment differs')
        invalid = observed['invalid']
        require(invalid['evidence']==evidence, 'inspection network raw owner substituted')
        import copy
        changed = copy.deepcopy(boundaries)
        cut = next(cut for cut in changed[1]['cuts'] if cut['handle']['node']=='a')
        state = bytearray(binary(cut['network_state']));state[0]^=1
        cut['network_state']=base64.b64encode(state).decode()
        require(invalid['boundaries']==changed, 'inspection network mutation differs')
    else:
        require(False,'unhandled inspection root')
    return dict(variant=variant, boundaries=len(boundaries), native_lineage=evidence['schema'])


ORIGINAL_SOURCES = {'tools/gomad/internal/backends/wasm/cluster/inspection_capacity_test.go': '503437cf6842a68ac5375048cae47afdfa9060a77364978c6805d30999900f19', 'tools/gomad/internal/backends/wasm/cluster/inspection_control_test.go': 'dc67d7e1b1fd274587b228572b6668cd3f2ad809e0848c8723b2fba048a02759', 'tools/gomad/internal/backends/wasm/cluster/inspection_network_test.go': '948c5e9a6a42784ab0689f4ecba2584d4a8ae31dc70c75e430adf282d394ac80', 'tools/gomad/internal/backends/wasm/cluster/replay_inspection_search_test.go': '79a1bdba6f2490fda5f7106f937f45a1b03358b223807f5c7c4b929416128039'}


def source_inputs(config):
    source = Path(config.get('inspection_original_source', '/home/agent/evidence/fn232-original-terminal-derived-v3/source'))
    for name, digest in ORIGINAL_SOURCES.items():
        path = native_roles.regular(source / name)
        require(sha(path.read_bytes()) == digest, 'inspection immutable original assertion source differs: ' + name)
    return ORIGINAL_SOURCES


def observations(config, stdout, artifact_roots=()):
    rows=[]
    for line in stdout.split('\n'):
        if SCHEMA in line:
            start=line.find('{')
            require(start>=0, 'inspection raw envelope absent')
            row=native_roles.decode(line[start:])
            require(set(row)=={'schema','original_root','variant','observed'} and row['schema']==SCHEMA, 'inspection raw envelope differs')
            rows.append(row)
    if not rows:
        return {}
    source_inputs(config)
    # Pin the independent fixture before deriving acceptance.
    fixture=Path(config['repo'])/'tools/gomad/rust/wasmhost/tests/testdata/inspection/source-five-decisions.json'
    require(sha(native_roles.pinned(config,fixture).read_bytes())=='b957bbe24a2b98e28e933efee136e50c8310cd9122783376f05ef587c62dbeef', 'inspection frozen fixture differs')
    seen={}; result={}
    for row in rows:
        root,variant=row['original_root'],row['variant']
        require(root in ROOT_CASES and variant in ROOT_CASES[root], 'unknown inspection root/variant')
        require((root,variant) not in seen, 'duplicate inspection raw variant')
        seen[root,variant]=True
        result.setdefault(root,[]).append(validate_case(root,variant,row['observed'],artifact_roots))
    for root in result:
        if root == PREFIX+'CapacityRetainsTemporalScaleBoundariesAndEffects':
            require(len({case['raw_sha256'] for case in result[root]})==1, 'inspection scale retained leg substituted helper fixture')
        require({case['variant'] for case in result[root]}==ROOT_CASES[root], 'missing inspection original subvariant')
    return result
