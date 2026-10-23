"""Closed native lifecycle observations; process and candidate custody belong to the caller."""
from pathlib import Path
import base64
import hashlib

from driver import require
from native_roles import decode, pinned
from core_predicates import raw_bytes
from native_observations import canonical

SCHEMA = 'gomad.native-lifecycle-predicate/v1'
TEST = 'tools/gomad/rust/wasmhost/src/linked_cluster/lifecycle_predicate_tests.rs'
SOURCES = (TEST, 'tools/gomad/rust/wasmhost/src/linked_cluster.rs',
           'tools/gomad/rust/wasmhost/src/linked_cancellation.rs',
           'tools/gomad/rust/gomad/src/campaign/scenario_preservation_tests.rs',
           'tools/gomad/rust/gomad/src/campaign/scenario_sqlite_predicate_tests.rs',
           'tools/gomad/rust/gomad/src/qualify/workloads/mod.rs',
           'tools/gomad/rust/gomad/src/qualify/workloads/oracles.rs',
           'tools/gomad/rust/gomad/src/campaign/scenario.rs',
           'tools/gomad/rust/wasmhost/src/linked.rs',
           'tools/gomad/rust/wasmhost/src/linked_storage.rs',
           'tools/gomad/rust/wasmhost/src/cluster/handoff/broker.rs',
           'tools/gomad/rust/wasmhost/src/linked_cluster/live_controller_tests.rs',
           'tools/gomad/rust/wasmhost/src/engine.rs',
           'tools/gomad/rust/wasmhost/src/model/reply.rs',
           'tools/gomad/rust/wasmhost/src/namespace/call.rs')

OPERATOR = 'TestOperatorStopsActionsAfterFailedWait'
LAZY = 'TestProviderClusterStartsOnlyRequestedIncarnations'
HOST = 'TestClusterHostTerminationStatus'
CLEANUP = 'TestProviderMandatoryCleanupReapsWithoutLifecycleActions'
MODEL = 'TestClusterRejectsResultWithoutModelEvidence'
ATTRIBUTION = 'TestClusterRejectsUnattributableFrames'
CALLBACK = 'TestClusterRejectsGuestCallbacks'
DRAIN = 'TestClusterShutdownWaitsForObservedFailureDrain'
DEADLINE = 'TestOpenClusterRecordsHelperExecutionDeadline'
RESTART = 'TestRejectedRestartLeavesNoUnownedBrokerSession'
RETRY = 'TestClusterRefusedRestartLeavesIncarnationRetryable'
INTERNAL = 'TestClusterInternalDeadlineIsWatchdog'
GRACE = 'TestClusterShutdownBoundsFinishedClusterByExecutionDeadline'
NETWORK = 'TestClusterNetworkFrameRefusesWhatTheGoBrokerRefuses'
SQLITE = 'TestSQLiteObservationUsesFreshFixedLines'
ORACLE = 'TestSQLiteObservationsForwardToNativeOracleOwner'
BOOT_NETWORK = 'TestProviderBindsEverySelectedBootNetworkLimit'
STORAGE_ROOTS = ('TestStorageCrashBoundOnlyByCrashBefore', 'TestOperatorAcceptsDeclaredStorageCrash')
VARIANTS = {
    OPERATOR: ('trap', 'exit'), LAZY: ('native-lazy-store',),
    HOST: ('cancelled-before-start', 'watchdog', 'cancelled', 'parent deadline'),
    CLEANUP: ('shutdown', 'scenario-error', 'context-cancel', 'action-limit', 'context-cancel-after-actual-output'),
    MODEL: ('missing-model', 'memory-capacity-before-bootstrap', 'memory-capacity-after-bootstrap'), ATTRIBUTION: ('missing-namespace', 'unknown-step', 'zero-incarnation', 'duplicate-handle', 'model-seed'),
    CALLBACK: ('call', 'sync'), DRAIN: ('trap', 'exit', 'held-peer-release'), DEADLINE: ('native-sealed-deadline',),
    RESTART: ('missing-storage-image', 'stale-incarnation', 'live-incarnation'),
    RETRY: ('recovery-image-refusal-and-retry',), INTERNAL: ('internal deadline',),
    GRACE: ('live-noncooperative-actual-five-second-grace',),
    NETWORK: ('valid', 'node without profile', 'node local faults', 'stock node', 'no cluster network', 'restarted incarnation', 'faults without network'),
    SQLITE: ('empty', 'missing-history', 'duplicate-history', 'duplicate-integrity', 'nonnumeric-event', 'trailing-comma', 'duplicate-ack'),
    ORACLE: ('three-native-checks',), BOOT_NETWORK: ('matching', 'byte-mismatch', 'stall-mismatch'),
    STORAGE_ROOTS: ('crash-before', 'crash-after', 'io-error', 'no-storage'),
}
CATALOG = frozenset((root, variant) for root, variants in VARIANTS.items() for variant in variants)

HELD_FAILED_MODULE = b'(module\n        (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))\n        (import "wasi_snapshot_preview1" "clock_time_get" (func $clock (param i32 i64 i32) (result i32)))\n        (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))\n        (import "gomad_wasm_v1" "idle" (func $idle (param i32 i32) (result i32)))\n        (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))\n        (import "wasi_snapshot_preview1" "proc_exit" (func $exit (param i32)))\n        (memory (export "memory") 1)\n        (data (i32.const 128) "mark")\n        (func (export "_start")\n            i32.const 0 i32.const 128 i32.store\n            i32.const 4 i32.const 4 i32.store\n            i32.const 1 i32.const 0 i32.const 1 i32.const 16 call $write drop\n            unreachable))'
HELD_PEER_MODULE = b'(module\n        (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))\n        (import "wasi_snapshot_preview1" "clock_time_get" (func $clock (param i32 i64 i32) (result i32)))\n        (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))\n        (import "gomad_wasm_v1" "idle" (func $idle (param i32 i32) (result i32)))\n        (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))\n        (import "wasi_snapshot_preview1" "proc_exit" (func $exit (param i32)))\n        (memory (export "memory") 1)\n        (data (i32.const 128) "mark")\n        (func (export "_start")\n            i32.const 0 i32.const 128 i32.store\n            i32.const 4 i32.const 4 i32.store\n            i32.const 1 i32.const 0 i32.const 1 i32.const 16 call $write drop\n            i32.const 512 i32.const 16 call $config drop\n    i32.const 995 i32.const 1 i32.store8\n    i32.const 992 i32.const 16 call $finish drop))'

MODEL_CAPACITY_MODULES = {
    'memory-capacity-before-bootstrap': HELD_FAILED_MODULE.removesuffix(b'unreachable))') + b'i32.const 1 memory.grow drop))',
    'memory-capacity-after-bootstrap': HELD_FAILED_MODULE.removesuffix(b'unreachable))') + b'i32.const 512 i32.const 16 call $config drop i32.const 1 memory.grow drop))',
}
MODEL_METADATA_FIELDS = frozenset(('now', 'advances', 'calls', 'records', 'trace_sha256', 'branching_records',
    'tape_records', 'tape_sha256', 'next_expected', 'cursor', 'configured', 'finished', 'inspection_sealed',
    'peak', 'elapsed', 'terminal_base64', 'transcript_bytes', 'transcript_sha256'))
FD_WRITE_MARK_TRANSCRIPT = (b'GMWT\x01\0\0\0' + bytes((8, 0, 0, 0)) + (1).to_bytes(8, 'little')
    + (16).to_bytes(4, 'little') + (1).to_bytes(4, 'little') + (4).to_bytes(4, 'little') + b'mark' + (4).to_bytes(4, 'little'))

BOOT_NETWORK_ORIGINAL_VARIANTS = (
    'matching', 'network free', 'first byte mismatch', 'later byte mismatch',
    'first stall mismatch', 'later stall mismatch', 'restart byte mismatch',
    'restart stall mismatch', 'unused mismatch', 'matching explicit broker',
    'broker byte mismatch', 'broker stall mismatch',
)
BOOT_NETWORK_MISSING_VARIANTS = BOOT_NETWORK_ORIGINAL_VARIANTS[1:]

ORIGINAL_ROOTS = ('TestClusterBootAdmitsStorageCrashSelectionBeforeSealing', 'TestClusterBrokerDurableRestartParity', 'TestClusterEventOrderAccessorIsDetached', 'TestClusterFinishPreservesSessionWithLiveIncarnations', 'TestClusterFinishWaitsForInspectionCleanup', 'TestClusterHostTerminationStatus', 'TestClusterInspectedPrefixRequiresRequestedSealedInstance', 'TestClusterInspectionAbortsNeighborsWithoutCompletingTheirTapes', 'TestClusterInspectionMetadataDoesNotReplaceWorldEvidence', 'TestClusterInternalDeadlineIsWatchdog', 'TestClusterOutputBoundaryParity', 'TestClusterPreservesWholeRunInfrastructureFailure', 'TestClusterProtocolFailureIsNotCancellation', 'TestClusterRefusedRestartLeavesIncarnationRetryable', 'TestClusterRejectsGuestCallbacks', 'TestClusterRejectsResultWithoutModelEvidence', 'TestClusterRejectsUnattributableFrames', 'TestClusterShutdownBoundsFinishedClusterByExecutionDeadline', 'TestClusterShutdownControlOrder', 'TestClusterShutdownLiveNodesEvidenceOrder', 'TestClusterShutdownWaitsForObservedFailureDrain', 'TestClusterStartAndObserveLeaveControllerPaused', 'TestClusterTransportCodecCorpus', 'TestClusterTransportCodecCorpusIsCurrent', 'TestClusterTransportCodecLiteralStageGuards', 'TestClusterTransportStageCorpus', 'TestClusterTransportStageCorpusIsCurrent', 'TestClusterTransportStageLiteralGuards', 'TestClusterWaitSerializesTurnsWithoutBlockingLiveControls', 'TestMarkerBoundaryPreservesSelectedCapacityAndPriorErrors', 'TestOpenClusterRecordsHelperExecutionDeadline', 'TestOperatorAcceptsDeclaredStorageCrash', 'TestOperatorArgumentsSurviveNormalization', 'TestOperatorStopsActionsAfterFailedWait', 'TestOperatorStrictPreflight', 'TestProviderActualCrashDurableRestart', 'TestProviderBindsEverySelectedBootNetworkLimit', 'TestProviderCanceledSplitRevocationCancelsAndReaps', 'TestProviderClusterStartsOnlyRequestedIncarnations', 'TestProviderCompressedIndependentGuestsLifecycleReplay', 'TestProviderDetachedOracleCanarySourceLessReplay', 'TestProviderGroupedAcknowledgmentCrashSourceLessReplay', 'TestProviderIndependentGuestsLifecycleReplay', 'TestProviderMandatoryCleanupReapsWithoutLifecycleActions', 'TestProviderOutputCrashKeepsItsExistingBoundary', 'TestProviderStopReapsGuestAndPeerContinues', 'TestProviderStorageCrashRequiresActualDurableBoundary', 'TestProviderUnboundRevocationOfSplitStorageNodeRestarts', 'TestRejectedRestartLeavesNoUnownedBrokerSession', 'TestSQLiteObservationUsesFreshFixedLines', 'TestSQLiteObservationsForwardToNativeOracleOwner')


def fields(value, names):
    require(isinstance(value, dict) and set(value) == set(names), 'foreign lifecycle fields: ' + repr(names))


def integer(value, maximum=(1 << 64) - 1):
    require(type(value) is int and 0 <= value <= maximum, 'foreign lifecycle integer')
    return value


def handle(node, incarnation=1):
    return dict(node=node, incarnation=incarnation)


def step(action, node):
    return dict(action=action, handle=handle(node))


def authored_steps(actual, expected):
    require(isinstance(actual, list), 'missing authored lifecycle steps')
    for item in actual:
        fields(item, ('action', 'handle')); fields(item['handle'], ('node', 'incarnation'))
        require(isinstance(item['action'], str) and isinstance(item['handle']['node'], str)
                and integer(item['handle']['incarnation']) > 0, 'foreign authored lifecycle step')
    return actual == expected


def activity(value, launched=None, callbacks=0):
    fields(value, ('launched_stores', 'remote_callback_attempts'))
    integer(value['launched_stores']); integer(value['remote_callback_attempts'])
    require(value['remote_callback_attempts'] == callbacks, 'native callback ownership differs')
    if launched is not None:
        require(value['launched_stores'] == launched, 'native actual store count differs')


def stores(value, owner_activity, owners=None, require_created=True):
    require(isinstance(value, list), 'missing actual native store results')
    seen, created = [], 0
    for item in value:
        fields(item, ('handle', 'terminal', 'exit_code', 'message', 'authorization_to_terminal_nanos', 'interrupted', 'stdout'))
        fields(item['handle'], ('node', 'incarnation'))
        node = item['handle']['node']
        require(isinstance(node, str) and 1 <= len(node) <= 128 and node.isascii(), 'foreign lifecycle node')
        require(integer(item['handle']['incarnation']) > 0, 'zero result incarnation')
        identity = node + '/' + str(item['handle']['incarnation'])
        require(identity not in seen, 'duplicate native store result owner')
        seen.append(identity)
        require(isinstance(item['terminal'], str) and isinstance(item['message'], str) and type(item['interrupted']) is bool, 'foreign native store result')
        if item['exit_code'] is not None:
            integer(item['exit_code'], (1 << 32) - 1)
        dropped = item['authorization_to_terminal_nanos']
        if dropped is not None:
            require(integer(dropped) > 0, 'store drop duration is absent or forged')
            created += 1
        else:
            require(not require_created and item['terminal'] == 'inspection_aborted', 'actual store lost its drop observation')
        if item['stdout'] is not None:
            raw_bytes(item['stdout'])
    require(seen == sorted(seen), 'native store ownership order differs')
    if owners is not None:
        require(seen == owners, 'native joined store owner differs')
    activity(owner_activity, created)
    return value


def recovery(observed):
    require(observed['storage_before'] == observed['storage_after_refusal'], 'refused restart changed sealed storage')
    before, retry = observed['storage_before'], observed['retry_storage']
    require(isinstance(before, dict) and isinstance(retry, dict), 'missing restart storage preimages')
    require(set(before) == set(retry) == {'descriptor', 'image', 'context'}, 'foreign restart storage input')
    raw = raw_bytes(observed['recovery_image'])
    require(raw and retry['image'] == list(raw) and before['image'] is None and retry['context'] == before['context'], 'restart recovery bytes differ')
    from core_codec import descriptor_model, recovery_image
    descriptor_model(before['descriptor'], 100, 1000, [dict(ID='data', Path='/workspace', CapacityBytes=4096)])
    recovered = recovery_image(raw)
    require(recovered['model'] == before['descriptor']['model_identity'] and set(recovered['volumes']) == {'data'}
            and recovered['volumes']['data']['mount'] == '/workspace' and recovered['volumes']['data']['capacity'] == 4096,
            'restart recovery image belongs to a foreign storage owner')
    old, new = dict(before['descriptor']), dict(retry['descriptor'])
    image = new.pop('image'); old_image = old.pop('image')
    require(old_image is None and new == old, 'restart changed non-image declaration')
    require(image == dict(bytes=len(raw), chunks=(len(raw) + 65535) // 65536, sha256=hashlib.sha256(raw).hexdigest()), 'restart recovery identity differs')
    require(observed['recovery_input_unchanged'] is True, 'restart image assertion differs')


def sqlite_observation(ack, stdout):
    def line(raw, prefix):
        text = raw.decode('utf-8')
        values = [line[len(prefix):] for line in text.split('\n') if line.startswith(prefix)]
        if not values or not values[0] or len(values[0].encode()) > 1 << 20:
            return None, 'missing or oversized workload observation'
        if len(values) > 1: return None, 'ambiguous workload observation'
        return values[0], None
    ack, error = line(ack, 'ACK ')
    if error: return None, error
    if not ack.isascii() or not ack.isdigit() or not 0 < int(ack) < 1 << 64 or str(int(ack)) != ack:
        return None, 'invalid SQLite acknowledgment'
    integrity, error = line(stdout, 'INTEGRITY ')
    if error: return None, error
    history, error = line(stdout, 'HISTORY ')
    if error: return None, error
    rows = [row.split(':', 1) for row in history.split(',')]
    if any(len(row) != 2 or not row[0].isascii() or not row[0].isdigit()
           or not 0 < int(row[0]) < 1 << 64 or str(int(row[0])) != row[0]
           or not row[1] or any(char in row[1] for char in ':\r\n') for row in rows):
        return None, 'invalid SQLite history'
    checks = dict(events=sorted(row[0] for row in rows) == ['1', '2'], integrity=integrity == 'ok',
                  acknowledged=ack == '2' and all(row[1] == 'committed' for row in rows))
    return {'acknowledged-durable-recovery.' + key: value for key, value in checks.items()}, None


def held_drain(observed):
    o = observed
    fields(o, ('terminal', 'exit_code', 'message', 'failed_module_bytes', 'failed_module_sha256',
        'peer_module_bytes', 'peer_module_sha256', 'held_peer_result', 'store_results', 'activity',
        'probe_millis', 'probe_outcome', 'probe_elapsed_nanos', 'release_to_return_nanos', 'join_error',
        'native_control_entry_points', 'before_control_attempts', 'held_control_attempts', 'after_control_attempts',
        'before_lifecycle_bytes', 'held_lifecycle_bytes', 'after_lifecycle_bytes'))
    for owner, expected in (('failed', HELD_FAILED_MODULE), ('peer', HELD_PEER_MODULE)):
        module = raw_bytes(o[owner + '_module_bytes'])
        require(module == expected and o[owner + '_module_sha256'] == hashlib.sha256(module).hexdigest(),
                'held drain did not execute the closed native WAT owner')
    require(o['terminal'] == 'trap' and o['exit_code'] is None and isinstance(o['message'], str) and o['message'],
            'held drain lost already observed native failure')
    actual = stores(o['store_results'], o['activity'], ['a/1', 'b/1'])
    require(actual[0]['terminal'] == 'trap' and actual[0]['exit_code'] is None and actual[0]['message'] == o['message'],
            'drain changed the original failed result')
    require(actual[1] == o['held_peer_result'] and actual[1]['terminal'] == 'exit' and actual[1]['exit_code'] == 0
            and actual[1]['stdout'] == list(b'mark'), 'held peer result was synthesized or replaced at release')
    require(o['probe_millis'] == 50 and type(o['probe_millis']) is int and o['probe_outcome'] == 'timeout'
            and integer(o['probe_elapsed_nanos']) >= 50_000_000, 'owned peer drain returned before original 50ms release gate')
    require(integer(o['release_to_return_nanos']) < 5_000_000_000 and o['join_error'] is None,
            'released native peer did not join successfully')
    require(o['native_control_entry_points'] == ['activate', 'run', 'command'] and
            [integer(o[key]) for key in ('before_control_attempts', 'held_control_attempts', 'after_control_attempts')] == [4, 4, 4],
            'drain attempted a post-failure native lifecycle control')
    raw = raw_bytes(o['before_lifecycle_bytes'])
    require(raw == raw_bytes(o['held_lifecycle_bytes']) == raw_bytes(o['after_lifecycle_bytes'])
            and raw.startswith(b'GMWIL\x01\0\0'), 'held native drain mutated lifecycle evidence')
    at, events = 8, []
    while at < len(raw):
        require(at + 5 <= len(raw), 'held drain lifecycle frame truncated')
        tag, size = raw[at], int.from_bytes(raw[at+1:at+5], 'little'); at += 5
        require(1 <= tag <= 10 and 0 < size <= 1024 and at + size + 8 <= len(raw), 'held drain lifecycle frame bounds')
        node = raw[at:at+size].decode('utf-8'); at += size
        incarnation = int.from_bytes(raw[at:at+8], 'little'); at += 8
        require((node in ('a', 'b') and incarnation == 1) or (tag == 7 and node == 'controller' and incarnation == 0),
                'held drain lifecycle record has a foreign owner')
        events.append((tag, node))
    starts_and_terminals = [event for event in events if event[0] in (1, 3)]
    require(starts_and_terminals == [(1, 'b'), (3, 'b'), (1, 'a'), (3, 'a')],
            'held native peer or already failed guest did not reach actual terminal ownership')


def capacity_model(variant, o):
    fields(o, ('module_bytes', 'module_sha256', 'request', 'terminal', 'exit_code', 'message', 'fuel_remaining',
        'offered_fuel', 'memory_bytes', 'peak_memory_bytes', 'output_bytes', 'local_model', 'choice_bytes',
        'transcript_bytes', 'executed_steps', 'store_results', 'activity'))
    module = raw_bytes(o['module_bytes'])
    require(module == MODEL_CAPACITY_MODULES[variant] and o['module_sha256'] == hashlib.sha256(module).hexdigest(),
            'capacity model case changed its actual interrupted WAT input')
    request = o['request']; fields(request, ('limits', 'model_configuration', 'arguments', 'environment', 'replay_trace', 'namespace', 'storage'))
    require(request['limits'] == dict(memory_bytes=65536, fuel=1000000, output_bytes=1024, evidence_bytes=1 << 20, wall_timeout_millis=5000)
            and request['arguments'] == [] and request['environment'] == ['PWD=/workspace'] and request['replay_trace'] is None and request['storage'] is None,
            'capacity model request budget or owned inputs changed')
    expected_model = dict(entropy_key=[0]*32, working_directory_sha256=hashlib.sha256(b'/workspace').hexdigest(),
        arguments=dict(bytes=0,count=0,chunks=0), environment=dict(bytes=0,count=0,chunks=0),
        seed=7,mode=1,limit=1048576,diagnostics=False,replay=None,epoch_nanos=946684800000000000,
        read_step_nanos=0,now=1,limits=dict(calls=1048576,transcript_bytes=1048576,pending_events=1024),
        digest_only=False,split=False,inspection=None)
    expected_namespace = dict(entries=[dict(path=list(b'/workspace'),directory=True,readonly=False,data=[])],
        writable_directories=[list(b'/workspace')],preopens=[list(b'/workspace')],working_directory=list(b'/workspace'),stdin=[],
        epoch_nanos=946684800000000000,limits=dict(files=100,descriptors=16,filesystem_bytes=4096,output_bytes=1024))
    require(canonical(request['model_configuration']) == canonical(expected_model) and canonical(request['namespace']) == canonical(expected_namespace),
            'capacity model request namespace/configuration ownership changed')
    require(o['terminal'] == 'capacity' and o['exit_code'] is None and isinstance(o['message'], str)
            and 'guest linear memory capacity' in o['message'], 'interrupted capacity changed terminal classification')
    require(o['offered_fuel'] == 1000000 and o['memory_bytes'] == o['peak_memory_bytes'] == 65536
            and o['output_bytes'] == 4 and 0 < integer(o['fuel_remaining']) <= o['offered_fuel'], 'actual interrupted store resources differ')
    require(authored_steps(o['executed_steps'], [step('start', 'a'), step('wait', 'a')]), 'capacity model fabricated lifecycle actions')
    actual = stores(o['store_results'], o['activity'], ['a/1'])
    require(actual[0]['terminal'] == o['terminal'] and actual[0]['exit_code'] is None and actual[0]['message'] == o['message']
            and actual[0]['stdout'] == list(b'mark'), 'interrupted model metadata lost its native terminal owner')
    model = o['local_model']; fields(model, MODEL_METADATA_FIELDS)
    configured = variant == 'memory-capacity-after-bootstrap'
    require(model['configured'] is configured and model['finished'] is False and model['inspection_sealed'] is False,
            'interrupted owned model lifetime is absent or marked complete')
    require(integer(model['calls']) == 1 + int(configured), 'interrupted model lost completed import calls')
    integer(model['now'])
    for key in ('advances', 'records', 'branching_records', 'tape_records', 'cursor', 'peak', 'elapsed'):
        require(integer(model[key]) == 0, 'interrupted model invented runtime choices or advances')
    require(model['next_expected'] is None and model['tape_sha256'] == model['terminal_base64'] == '', 'interrupted model invented replay/finish state')
    choice, transcript = raw_bytes(o['choice_bytes']), raw_bytes(o['transcript_bytes'])
    require(choice == b'' and transcript == FD_WRITE_MARK_TRANSCRIPT, 'interrupted model lost the exact completed fd_write prefix')
    require(integer(model['transcript_bytes']) == len(transcript) and model['transcript_sha256'] == hashlib.sha256(transcript).hexdigest()
            and model['trace_sha256'] == hashlib.sha256(choice).hexdigest(), 'interrupted model commitments do not bind actual complete evidence bytes')


def validate_case(root, variant, observed, inputs=None):
    require((root, variant) in CATALOG, 'unknown or cross-labeled native lifecycle case')
    o = observed
    if root == MODEL and variant in MODEL_CAPACITY_MODULES:
        capacity_model(variant, o)
    elif root == DRAIN and variant == 'held-peer-release':
        held_drain(o)
    elif root in (OPERATOR, DRAIN):
        names = {'activity', 'exit_code', 'steps', 'terminal', 'store_results'}
        names |= {'incarnations'} if root == OPERATOR else {'joined_incarnations', 'message'}
        fields(o, names)
        code = None if variant == 'trap' else 17
        require(o['terminal'] == variant and o['exit_code'] == code, 'failed wait terminal precedence differs')
        expected = [step('start', 'a'), step('wait', 'a')] if root == OPERATOR else [step('start', 'a'), step('start', 'b'), step('wait', 'a')]
        require(authored_steps(o['steps'], expected), 'failed wait invented or reordered authored actions')
        owners = ['a/1'] if root == OPERATOR else ['a/1', 'b/1']
        require(o['incarnations' if root == OPERATOR else 'joined_incarnations'] == owners, 'failed wait owner preimage differs')
        actual = stores(o['store_results'], o['activity'], owners, require_created=root == OPERATOR)
        require(actual[0]['terminal'] == variant and actual[0]['exit_code'] == code, 'first native result lost terminal')
        if root == DRAIN:
            require(o['message'] == actual[0]['message'], 'drain replaced the original failure')
    elif root == LAZY:
        fields(o, ('declared_nodes', 'incarnations', 'activity', 'store_results'))
        require(o['declared_nodes'] == ['a', 'b'] and o['incarnations'] == ['a/1'], 'lazy start executed dormant owner')
        actual = stores(o['store_results'], o['activity'], ['a/1'])
        require(actual[0]['terminal'] == 'exit' and actual[0]['exit_code'] == 0, 'lazy store did not finish')
    elif root == CLEANUP:
        after = variant == 'context-cancel-after-actual-output'
        names = {'terminal', 'steps', 'joined_incarnations', 'acknowledged_stdout', 'activity', 'store_results'}
        names |= {'store_dropped_before_return'} if after else {'message', 'acknowledged_handle', 'all_stores_dropped_before_return', 'dormant_started'}
        fields(o, names)
        require(authored_steps(o['steps'], [step('start', 'b'), step('start', 'a')]), 'cleanup invented or reordered lifecycle actions')
        require(o['joined_incarnations'] == ['a/1', 'b/1'] if after else integer(o['joined_incarnations']) == 2, 'cleanup result ownership omitted')
        actual = stores(o['store_results'], o['activity'], ['a/1', 'b/1'])
        require(actual[1]['stdout'] == list(b'mark'), 'cleanup never reached actual peer output')
        require(o['acknowledged_stdout'] == ('mark' if after else list(b'mark')), 'cleanup acknowledgment differs')
        terminal = {'shutdown': 'exit', 'scenario-error': 'scenario_failure', 'action-limit': 'capacity'}.get(variant, 'cancelled')
        require(o['terminal'] == terminal, 'cleanup suppressed original terminal')
        require(o['store_dropped_before_return' if after else 'all_stores_dropped_before_return'] is True, 'cleanup drop assertion differs')
        if not after:
            require(o['acknowledged_handle'] == handle('b') and o['dormant_started'] is False, 'cleanup acknowledged dormant owner')
            if variant != 'context-cancel':
                require(all(item['terminal'] == 'stop' and item['interrupted'] for item in actual), 'cleanup failed to interrupt all owned stores')
            if variant == 'scenario-error': require(o['message'] == 'scenario broke', 'scenario failure cause changed')
            if variant == 'action-limit': require(o['message'] == 'scenario action capacity limit=2 used=2 delta=1', 'action capacity accounting changed')
    elif root in (HOST, INTERNAL):
        if variant == 'cancelled-before-start':
            fields(o, ('terminal', 'steps', 'activity', 'store_results'))
            require(o['terminal'] == 'cancelled' and authored_steps(o['steps'], []), 'pre-start cancellation performed actions')
            stores(o['store_results'], o['activity'], [])
        else:
            fields(o, ('terminal', 'exit_code', 'watchdog_timeout', 'cancelled', 'bound_millis', 'parent_cancel_after_millis', 'joined_incarnations', 'all_stores_dropped_before_return', 'activity', 'store_results'))
            watchdog = variant in ('watchdog', 'internal deadline')
            require(o['terminal'] == ('watchdog_timeout' if watchdog else 'cancelled') and o['exit_code'] is None, 'host termination classification changed')
            require(o['watchdog_timeout'] is watchdog and o['cancelled'] is not watchdog, 'host deadline mistaken for parent cancellation')
            require(o['bound_millis'] == (20 if watchdog else 1000) and o['parent_cancel_after_millis'] == (None if watchdog else 20), 'original host termination bounds changed')
            actual = stores(o['store_results'], o['activity'])
            require(integer(o['joined_incarnations']) == len(actual) and len(actual) <= 1 and o['all_stores_dropped_before_return'] is True, 'host termination leaked owned stores')
    elif root in (MODEL, ATTRIBUTION):
        fields(o, ('admission_error',))
        errors = {'missing-model': 'cluster requires helper-owned model', 'missing-namespace': 'cluster requires sealed namespace', 'unknown-step': 'unknown cluster lifecycle handle', 'zero-incarnation': 'invalid or duplicate cluster handle', 'duplicate-handle': 'invalid or duplicate cluster handle', 'model-seed': 'cluster model seed mismatch'}
        require(o['admission_error'] == errors[variant], 'typed owner admission precedence changed')
    elif root == CALLBACK:
        fields(o, ('error_kind', 'message', 'activity'))
        require(o['error_kind'] == 'unsupported' and o['message'] == 'linked cluster forbids transport callbacks', 'callback escaped native owner')
        activity(o['activity'], 0, 1)
    elif root == DEADLINE:
        fields(o, ('sealed_millis', 'bounded_millis', 'zero_refused', 'activity'))
        require(o['sealed_millis'] == 660000 and o['bounded_millis'] == min(660000, 420000) and o['zero_refused'] is True, 'sealed deadline was extended or zero admitted')
        activity(o['activity'], 0)
    elif root in (RESTART, RETRY):
        common = {'refusal', 'recovery_input_unchanged', 'storage_before', 'storage_after_refusal', 'retry_storage', 'recovery_image', 'activity', 'retry_incarnation', 'retry_slot'}
        common |= {'before_incarnation', 'before_slot', 'after_refusal_incarnation', 'after_refusal_slot', 'peer_incarnation', 'peer_slot', 'refused_storage_unchanged'} if root == RESTART else {'refused_incarnation_owned', 'current_after_refusal'}
        fields(o, common)
        missing = root == RETRY or variant == 'missing-storage-image'
        require(o['refusal'] == ('cluster restart lacks recovery image' if missing else 'WASI invalid: broker restart lifecycle'), 'restart refusal changed')
        if root == RESTART:
            require([integer(o[key]) for key in ('before_incarnation', 'before_slot', 'after_refusal_incarnation', 'after_refusal_slot', 'peer_incarnation', 'peer_slot')] == [1, 1, 1, 1, 1, 2] and o['refused_storage_unchanged'] is True, 'restart refusal consumed broker or peer lifecycle')
        else:
            require(integer(o['current_after_refusal']) == 1 and o['refused_incarnation_owned'] is False, 'refused incarnation remained owned')
        require(integer(o['retry_incarnation']) == 2 and integer(o['retry_slot']) == 1 + 2, 'retry consumed an extra incarnation slot')
        activity(o['activity'], 0); recovery(o)
    elif root == GRACE:
        fields(o, ('execution_deadline_millis', 'cleanup_grace_millis', 'elapsed_nanos', 'cleanup_error', 'acknowledged_stdout', 'authored_steps', 'joined_incarnations', 'all_stores_dropped_before_return', 'activity', 'store_results'))
        require(o['execution_deadline_millis'] == 900000 and o['cleanup_grace_millis'] == 5000 and 5_000_000_000 <= integer(o['elapsed_nanos']) < 10_000_000_000, 'actual original five second cleanup bound changed')
        require(o['cleanup_error'] == 'actual scenario failure; cluster cleanup deadline exceeded', 'cleanup masked original failure')
        require(authored_steps(o['authored_steps'], [step('start', 'a')]) and integer(o['joined_incarnations']) == 1 and o['acknowledged_stdout'] == list(b'mark'), 'busy cleanup fabricated actions or skipped real output')
        actual = stores(o['store_results'], o['activity'], ['a/1'])
        require(actual[0]['stdout'] == list(b'mark') and o['all_stores_dropped_before_return'] is True, 'busy store was not reaped')
    elif root == NETWORK:
        if variant == 'valid':
            fields(o, ('admitted', 'profile', 'activity'))
            require(o['admitted'] is True and o['profile'] == 'gomad.simulation-network-stream/v3', 'valid network owner refused')
            activity(o['activity'], 0)
        else:
            names = {'admitted', 'admission_error', 'launched_stores'} | ({'owner'} if variant == 'faults without network' else set())
            fields(o, names)
            errors = {'node without profile': 'cluster boot network profile', 'node local faults': 'broker node local fault schedule', 'stock node': 'cluster requires helper-owned model', 'no cluster network': 'cluster boot network profile', 'restarted incarnation': 'invalid cluster incarnation lifecycle', 'faults without network': 'missing field `model`'}
            require(o['admitted'] is False and o['launched_stores'] == 0 and type(o['launched_stores']) is int and o['admission_error'] == errors[variant], 'network typed owner mutation admitted')
            if variant == 'faults without network': require(o['owner'] == 'network-config-decoder', 'network absence refusal attributed to foreign owner')
    elif root in (SQLITE, ORACLE):
        fields(inputs, ('ack_bytes', 'stdout_bytes'))
        ack, stdout = raw_bytes(inputs['ack_bytes']), raw_bytes(inputs['stdout_bytes'])
        good = b'INTEGRITY ok\nHISTORY 1:committed,2:committed\n'
        outputs = {'empty': b'', 'missing-history': b'INTEGRITY ok\n',
            'duplicate-history': b'INTEGRITY ok\nHISTORY 1:committed\nHISTORY 2:committed\n',
            'duplicate-integrity': b'INTEGRITY ok\nINTEGRITY ok\nHISTORY 1:committed\n',
            'nonnumeric-event': b'INTEGRITY ok\nHISTORY x:committed\n',
            'trailing-comma': b'INTEGRITY ok\nHISTORY 1:committed,\n', 'duplicate-ack': good,
            'three-native-checks': good}
        require(ack == (b'ACK 2\nACK 2\n' if variant == 'duplicate-ack' else b'ACK 2\n') and stdout == outputs[variant], 'original SQLite input literal changed')
        checks, error = sqlite_observation(ack, stdout)
        if root == SQLITE:
            fields(o, ('error',))
            require(error and o['error'] == error, 'SQLite fixed-line ambiguity changed')
        else:
            fields(o, ('schema', 'checks'))
            require(error is None and all(checks.values()) and o['schema'] == 'gomad.rust-workload-oracles/v1'
                    and o['checks'] == checks and all(type(value) is bool for value in o['checks'].values()), 'SQLite forwarding lost a native oracle check')
    elif root == BOOT_NETWORK:
        fields(inputs, ('scenario_base64', 'network'))
        scenario = decode(base64.b64decode(inputs['scenario_base64'], validate=True))
        require(scenario['spec']['nodes'] == [dict(id='a', boot='main', address='10.0.0.1'), dict(id='b', boot='main', address='10.0.0.2')]
                and scenario['actions'] == [step('start', 'a'), step('start', 'b'), step('wait', 'a'), step('wait', 'b'), dict(action='restart', handle=handle('a', 2)), dict(action='wait', handle=handle('a', 2))], 'selected boot scenario owners/actions changed')
        limits = scenario['spec']['limits']
        require(limits['network_connection_bytes'] == 123 and limits['network_stall_nanos'] == 456, 'original scenario network specification changed')
        network = inputs['network']
        require(isinstance(network, dict) and network['profile'] == 'gomad.simulation-network-stream/v3', 'foreign selected network profile')
        expected = {'matching': (123, 456), 'byte-mismatch': (124, 456), 'stall-mismatch': (123, 457)}[variant]
        expected_network = dict(profile='gomad.simulation-network-stream/v3', faults=[],
            limits=dict(listeners=8, connections=8, bytes=1024, connection_bytes=expected[0], stall_nanos=expected[1], transitions=128))
        require(network == expected_network, 'original boot network mutation or unrelated limits changed')
        fields(o, ('configuration', 'request_incarnations', 'error', 'accepted'))
        matching = expected == (limits['network_connection_bytes'], limits['network_stall_nanos'])
        require(o['accepted'] is matching, 'mismatched selected boot network accepted')
        if matching:
            require(o['error'] is None and isinstance(o['configuration'], dict), 'matching boot configuration absent')
            expected_owners = sorted([handle('a'), handle('b'), handle('a', 2)], key=lambda owner: (owner['node'], owner['incarnation']))
            for items in (o['configuration']['incarnations'], o['request_incarnations']):
                require(isinstance(items, list) and sorted([item['handle'] for item in items], key=lambda owner: (owner['node'], owner['incarnation'])) == expected_owners, 'selected network owner omitted or duplicated')
                require(all(item['network'] == network for item in items), 'actual selected incarnation lost spec-bound limits')
        else:
            require(o['configuration'] is None and o['request_incarnations'] == [] and o['error'] == 'WASM boot network limits differ from simulation specification', 'boot mismatch retained a partially bound incarnation')
    elif root == STORAGE_ROOTS:
        fields(o, ('accepted', 'error'))
        require(o['accepted'] is (variant == 'crash-before') and o['error'] == (None if variant == 'crash-before' else 'undeclared WASM lifecycle boundary'), 'storage crash fault declared an unrelated boundary')
    else:
        require(False, 'unhandled closed lifecycle case')
    return dict(original_roots=list(root) if isinstance(root, tuple) else [root], variant=variant)


def parse_rows(stdout, required=CATALOG):
    seen, facts = set(), []
    for line in stdout.split('\n'):
        if SCHEMA not in line:
            continue
        require('{' in line, 'native lifecycle row lacks JSON')
        row = decode(line[line.index('{'):])
        key = 'original_roots' if 'original_roots' in row else 'original_root'
        fields(row, ('schema', key, 'variant', 'observed', *(['inputs'] if 'inputs' in row else [])))
        require(row['schema'] == SCHEMA, 'foreign lifecycle schema')
        root = tuple(row[key]) if key == 'original_roots' and isinstance(row[key], list) else row[key]
        require((isinstance(root, str) or isinstance(root, tuple) and all(isinstance(name, str) for name in root)) and isinstance(row['variant'], str), 'foreign lifecycle identity')
        case = (root, row['variant'])
        require(case in required and case not in seen, 'unknown or duplicate native lifecycle case')
        require(('inputs' in row) == (root in (SQLITE, ORACLE, BOOT_NETWORK)), 'lifecycle input preimages omitted or cross-labeled')
        facts.append(validate_case(root, row['variant'], row['observed'], row.get('inputs')))
        seen.add(case)
    require(seen == set(required), 'native lifecycle harness omitted original cases: ' + repr(sorted(set(required) - seen, key=str)))
    return facts


def observations(config, stdout, network_frame_cases=None):
    source_facts = {}
    for name in SOURCES:
        path = pinned(config, (Path(config['repo']) / name).resolve())
        source_facts[name] = hashlib.sha256(path.read_bytes()).hexdigest()
    production = (Path(config['repo']) / SOURCES[1]).read_text()
    test_source = (Path(config['repo']) / TEST).read_text()
    require('fn store_result_fact' in test_source and '"authorization_to_terminal_nanos":item.execution.authorization_to_terminal_nanos' in test_source,
            'native store result collector no longer exposes actual drop timing')
    require('CLUSTER_STOP_GRACE: Duration = Duration::from_secs(5)' in production and 'watcher.join()' in production.replace('\n', '').replace(' ', ''), 'native cleanup bound/join source contract changed')
    drain_source = production.split('fn drain_running(', 1)[1].split('\nfn ', 1)[0]
    require('match joined(pending)' in drain_source and not any(call in drain_source for call in ('.command(', '.run(', '.activate(', 'stop_live('))
            and 'let join_error = drain_running(running, &mut completed, broker, driven.is_ok());' in production,
            'native final drain no longer uses the control-free owned join seam')
    broker_source = (Path(config['repo']) / 'tools/gomad/rust/wasmhost/src/cluster/handoff/broker.rs').read_text()
    for name in ('activate', 'run', 'command'):
        body = broker_source.split('pub fn ' + name + '(', 1)[1].split('\n    pub ', 1)[0]
        require('self.native_control_attempts' in body and '.fetch_add(1, std::sync::atomic::Ordering::Relaxed)' in body,
                'native broker control attempt counter no longer measures actual entry')
    facts = parse_rows(stdout)
    complete = {OPERATOR, HOST, INTERNAL, CALLBACK, RESTART, RETRY, CLEANUP, DRAIN, ORACLE, STORAGE_ROOTS[0], STORAGE_ROOTS[1]}
    partial = {
        LAZY: ['Original hard-isolation refusal and explicit isolated reference admission require separate receiving evidence.'],
        MODEL: ['The former untrusted capacity result frame without model fields belonged to the retired Go helper transport; no current public native result-frame decoder accepts that representation.'],
        ATTRIBUTION: ['Duplicate JSON keys and corrupt evidence base64 require the retained codec receiving owner.'],
        DEADLINE: ['Original maximum selection over instance 3m/7m and boot 11m/1m declarations is not present in the native sealed-deadline row.'],
        GRACE: ['Original finished/near-deadline/past-deadline/never-opened and failed-finish race table remains distinct from the actual five-second busy-store cleanup case.'],
        SQLITE: ['Original configurable maximum_bytes=1 refusal has no corresponding native observation parameter.'],
        BOOT_NETWORK: [
            'Original provider_test.go TestProviderBindsEverySelectedBootNetworkLimit/' + variant
            + ' has no native per-boot/shared-broker observation; the global native byte/stall mutations do not prove this variant.'
            for variant in BOOT_NETWORK_MISSING_VARIANTS
        ] + ['Original explicit broker mismatches must also leave the caller-owned mismatched broker input unchanged.'],
    }
    frame_names = {'per-node fault schedule', 'bad address', 'unknown link endpoint', 'self link', 'duplicate link', 'unordered fault', 'unknown action'}
    if network_frame_cases is not None:
        require(isinstance(network_frame_cases, dict) and set(network_frame_cases) == frame_names and all(isinstance(v, dict) and isinstance(v.get('actual_error'), str) and v['actual_error'] for v in network_frame_cases.values()), 'network Shared receiving cases omitted')
        complete.add(NETWORK)
    else:
        partial[NETWORK] = ['The original seven Shared network configuration refusals require core_network.frame_cases from the same authenticated harness.']
    roots = {root: dict(cases=[fact for fact in facts if root in fact['original_roots']], sources=source_facts,
                        predicates=['closed original subvariant catalog', 'authored action order', 'actual owner/drop observations', 'native refusal and terminal precedence']) for root in sorted(complete)}
    roots[DRAIN]['native_receive'] = dict(
        ownership='The production final drain joins the held native worker before publishing its actual Completed result.',
        controls='Per-broker attempt counters cover native activate/run/command entry points, including rejected commands; raw lifecycle state remains unchanged.',
        retired_wire_disposition='The former helper IPC run/control channel is retired. These facts close the surviving ownership/control predicate and supply no IPC grammar or frame-codec credit.')
    receiving = {root: dict(remaining_original_predicates=[
        'Original root assertions require their other native receiving owner; no lifecycle-v1 row is assigned to this root.'])
        for root in ORIGINAL_ROOTS if root not in complete and root not in partial}
    cost_projection_dependency = dict(original_root='TestCompletedSummariesDetachCountersWithoutEvidencePayloads',
        schema='gomad.native-cluster-cost-projection/v1',
        source='tools/gomad/rust/gomad/src/fixture_cluster_metrics_tests.rs',
        remaining_original_predicates=['Parent-owned receiver must compare actual per-store counters and evidence payloads before and after summary projection.'])
    partial_roots = {root: dict(cases=[fact for fact in facts if root in fact['original_roots']],
        remaining_original_predicates=missing) for root, missing in partial.items()}
    partial_roots[MODEL]['surviving_native_predicates'] = ['Missing model ownership refused before engine creation.',
        'Actual memory-capacity terminals before/after runtime bootstrap retain complete typed model metadata and exact fd_write evidence commitments.']
    partial_roots[LAZY]['retired_representation_facts'] = ['The alternate goBrokerReference isolation provider was removed; current linked execution has no isolated-reference switch.']
    partial_roots[LAZY]['surviving_native_source'] = dict(source='tools/gomad/rust/gomad/src/campaign/scenario.rs',
        selector='campaign::scenario::fidelity_tests::single_linked_helper_refuses_hard_isolation')
    partial_roots[DEADLINE]['retired_representation_facts'] = ['Typed Request carries one explicit cluster wall_timeout; it has no named instance/boot timeout catalogue from which to select the old maximum.']
    partial_roots[SQLITE]['retired_representation_facts'] = ['Workload::observe has no caller maximum_bytes parameter; current line/oracle limits are fixed at 1MiB.']
    partial_roots[BOOT_NETWORK].update(original_variant_catalog=list(BOOT_NETWORK_ORIGINAL_VARIANTS),
        covered_original_variants=['matching'], native_variant_catalog=list(VARIANTS[BOOT_NETWORK]),
        sources=source_facts)
    return dict(roots=roots, receiving_dependencies=receiving, cost_projection_dependency=cost_projection_dependency,
                partial_roots=partial_roots, facts=facts, sources=source_facts)
