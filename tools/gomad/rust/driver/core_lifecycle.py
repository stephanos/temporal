"""Closed native lifecycle observations; process and candidate custody belong to the caller."""
from pathlib import Path
import base64
import hashlib

from driver import require
from native_roles import decode, pinned
from core_predicates import raw_bytes

SCHEMA = 'gomad.native-lifecycle-predicate/v1'
TEST = 'tools/gomad/rust/wasmhost/src/linked_cluster/lifecycle_predicate_tests.rs'
SOURCES = (TEST, 'tools/gomad/rust/wasmhost/src/linked_cluster.rs',
           'tools/gomad/rust/wasmhost/src/linked_cancellation.rs',
           'tools/gomad/rust/gomad/src/campaign/scenario_preservation_tests.rs',
           'tools/gomad/rust/gomad/src/campaign/scenario_sqlite_predicate_tests.rs',
           'tools/gomad/rust/gomad/src/qualify/workloads/mod.rs',
           'tools/gomad/rust/gomad/src/qualify/workloads/oracles.rs')

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
    MODEL: ('missing-model',), ATTRIBUTION: ('missing-namespace', 'unknown-step', 'zero-incarnation', 'duplicate-handle', 'model-seed'),
    CALLBACK: ('call', 'sync'), DRAIN: ('trap', 'exit'), DEADLINE: ('native-sealed-deadline',),
    RESTART: ('missing-storage-image', 'stale-incarnation', 'live-incarnation'),
    RETRY: ('recovery-image-refusal-and-retry',), INTERNAL: ('internal deadline',),
    GRACE: ('live-noncooperative-actual-five-second-grace',),
    NETWORK: ('valid', 'node without profile', 'node local faults', 'stock node', 'no cluster network', 'restarted incarnation', 'faults without network'),
    SQLITE: ('empty', 'missing-history', 'duplicate-history', 'duplicate-integrity', 'nonnumeric-event', 'trailing-comma', 'duplicate-ack'),
    ORACLE: ('three-native-checks',), BOOT_NETWORK: ('matching', 'byte-mismatch', 'stall-mismatch'),
    STORAGE_ROOTS: ('crash-before', 'crash-after', 'io-error', 'no-storage'),
}
CATALOG = frozenset((root, variant) for root, variants in VARIANTS.items() for variant in variants)

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
        values = [line[len(prefix):] for line in text.splitlines() if line.startswith(prefix)]
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


def validate_case(root, variant, observed, inputs=None):
    require((root, variant) in CATALOG, 'unknown or cross-labeled native lifecycle case')
    o = observed
    if root in (OPERATOR, DRAIN):
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
    for line in stdout.splitlines():
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
    facts = parse_rows(stdout)
    complete = {OPERATOR, HOST, INTERNAL, CALLBACK, DRAIN, RESTART, RETRY, CLEANUP, ORACLE, BOOT_NETWORK, STORAGE_ROOTS[0], STORAGE_ROOTS[1]}
    partial = {
        LAZY: ['Original hard-isolation refusal and explicit isolated reference admission require separate receiving evidence.'],
        MODEL: ['Interrupted terminal model completeness requires the retained result/model receiving owner.'],
        ATTRIBUTION: ['Duplicate JSON keys and corrupt evidence base64 require the retained codec receiving owner.'],
        DEADLINE: ['Original maximum selection over instance 3m/7m and boot 11m/1m declarations is not present in the native sealed-deadline row.'],
        GRACE: ['Original finished/near-deadline/past-deadline/never-opened and failed-finish race table remains distinct from the actual five-second busy-store cleanup case.'],
        SQLITE: ['Original configurable maximum_bytes=1 refusal has no corresponding native observation parameter.'],
    }
    frame_names = {'per-node fault schedule', 'bad address', 'unknown link endpoint', 'self link', 'duplicate link', 'unordered fault', 'unknown action'}
    if network_frame_cases is not None:
        require(isinstance(network_frame_cases, dict) and set(network_frame_cases) == frame_names and all(isinstance(v, dict) and isinstance(v.get('actual_error'), str) and v['actual_error'] for v in network_frame_cases.values()), 'network Shared receiving cases omitted')
        complete.add(NETWORK)
    else:
        partial[NETWORK] = ['The original seven Shared network configuration refusals require core_network.frame_cases from the same authenticated harness.']
    roots = {root: dict(cases=[fact for fact in facts if root in fact['original_roots']], sources=source_facts,
                        predicates=['closed original subvariant catalog', 'authored action order', 'actual owner/drop observations', 'native refusal and terminal precedence']) for root in sorted(complete)}
    receiving = {root: dict(remaining_original_predicates=[
        'Original root assertions require their other native receiving owner; no lifecycle-v1 row is assigned to this root.'])
        for root in ORIGINAL_ROOTS if root not in complete and root not in partial}
    return dict(roots=roots, receiving_dependencies=receiving, partial_roots={root: dict(cases=[fact for fact in facts if root in fact['original_roots']], remaining_original_predicates=missing) for root, missing in partial.items()}, facts=facts, sources=source_facts)
