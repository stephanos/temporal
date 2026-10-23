#!/usr/bin/env python3
"""Source controls for the original forged-context peer without a result frame."""
import base64
import copy
import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.dont_write_bytecode = True


def load(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


observe = load('live-broker-observe')
checker = load('live-broker-cluster-check')
IDENTITY = 'sha256:' + 'a' * 64
INDEX = dict(schema='gomad.live-broker-observation-index/v2',
             network_capture=dict(derived_go_implementation_sha256=IDENTITY))


def fixture():
    collected = dict(status='collected', sink_retained=False, state_error='', done_closed=True,
                     classified=True, reaped=True, group_gone=True, wire_bytes=1, document_bytes=1)
    uncollected = dict(collected, status='host_cleanup_without_result', sink_retained=True,
                       state_error='missing incarnation result', wire_bytes=0, document_bytes=0)
    result = dict(Termination='invalid', Message='storage crash context diverged from prior observation',
                  StorageImage=None, Reaped=True, NetworkTranscript='Ag==', ExitCode=None,
                  Stdout=None, Stderr=None, Transcript=None, StorageTranscript=None,
                  RuntimeInterrupted=False, ChoiceTrace=dict(Bytes=None))
    return dict(test='TestClusterBrokerDurableRestartParity/storage-crash/seed-7',
                go_owner_implementation_sha256=IDENTITY,
                states=[dict(handle=dict(ID='a', Incarnation=1), result=result, streams={},
                             network_wire='AQ==', collection=collected),
                        dict(handle=dict(ID='b', Incarnation=1), result=dict(Termination='', Reaped=True, NetworkTranscript=None),
                             streams={}, network_wire=None, collection=uncollected)],
                configuration=dict(Instances=[dict(ID='a', Request=dict(StorageCrashContext={
                    'a-store': dict(PendingSHA256='sha256:' + '0' * 64, Operations=[])})), dict(ID='b', Request={})]),
                steps=[dict(action='start', node='a', incarnation=1), dict(action='start', node='b', incarnation=1),
                       dict(action='wait', node='a', incarnation=1)],
                result=dict(GroupGone=True, World=None, Frames=dict(Results=1)), network_wire=None)


class CollectionStatus(unittest.TestCase):
    def test_host_cleanup_requires_every_original_state_fact(self):
        row = fixture()
        observe.validate_row(row, INDEX)
        for key, value in [('sink_retained', False), ('state_error', ''), ('done_closed', False),
                           ('classified', False), ('reaped', False), ('group_gone', False),
                           ('wire_bytes', 1), ('document_bytes', 1), ('status', 'unknown')]:
            with self.subTest(key=key):
                bad = copy.deepcopy(row)
                bad['states'][1]['collection'][key] = value
                with self.assertRaises(ValueError):
                    observe.validate_row(bad, INDEX)
        for state in row['states']:
            bad = copy.deepcopy(row)
            del bad['states'][row['states'].index(state)]['collection']
            with self.assertRaises(ValueError):
                observe.validate_row(bad, INDEX)
        for field, value in [('Termination', 'exit'), ('Reaped', False), ('NetworkTranscript', 'AQ==')]:
            bad = copy.deepcopy(row)
            bad['states'][1]['result'][field] = value
            with self.assertRaises(ValueError):
                observe.validate_row(bad, INDEX)

    def test_negative_exception_requires_the_exact_original_case(self):
        row = fixture()
        self.assertEqual(checker.forged_storage_cleanup(row), ('b', 1))
        mutations = [lambda r: r.update(test='TestClusterBrokerDurableRestartParity/crash/seed-7'),
                     lambda r: r['states'][0]['result'].update(Message='other'),
                     lambda r: r['states'][0]['result'].update(Termination='exit'),
                     lambda r: r['states'][0]['result'].update(StorageImage='unexpected'),
                     lambda r: r['result'].update(GroupGone=False),
                     lambda r: r['result']['Frames'].update(Results=2),
                     lambda r: r['steps'].append(dict(action='wait', node='b', incarnation=1)),
                     lambda r: r['configuration']['Instances'][0]['Request']['StorageCrashContext']['a-store'].update(PendingSHA256='sha256:' + '1' * 64)]
        for mutate in mutations:
            bad = copy.deepcopy(row)
            mutate(bad)
            with self.assertRaises((AssertionError, ValueError)):
                checker.forged_storage_cleanup(bad)

    def test_collected_network_and_actual_process_cleanup_remain_required(self):
        row = fixture()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            node = root / 'incarnations/a/1'
            node.mkdir(parents=True)
            (node / 'network').write_bytes(b'\x01')
            summary = dict(schema='gomad.live-cluster-observation/v1', execution_returned=True,
                           outcome=dict(termination='invalid', message=row['states'][0]['result']['Message']),
                           executed_steps=[dict(action=s['action'], handle=dict(node=s['node'], incarnation=1)) for s in row['steps']],
                           incarnations=[dict(handle=dict(node='a', incarnation=1), interrupted=False,
                                              outcome=dict(termination='invalid', message=row['states'][0]['result']['Message']))])
            (root / 'result.json').write_text(json.dumps(summary))
            report = checker.compare(root, row, process_exit=0)
            self.assertEqual(report['uncollected_go_peer'], dict(node='b', incarnation=1))
            self.assertEqual(report['go_collected_results'], 1)
            for exit_code in (None, 2):
                with self.assertRaises(AssertionError):
                    checker.compare(root, row, process_exit=exit_code)
            (node / 'network').write_bytes(b'\x02')
            with self.assertRaisesRegex(AssertionError, 'network'):
                checker.compare(root, row, process_exit=0)


if __name__ == '__main__':
    unittest.main()
