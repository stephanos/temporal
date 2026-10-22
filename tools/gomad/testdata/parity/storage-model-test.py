#!/usr/bin/env python3
"""Frozen Go StorageModel JSON identities survive observation dictionary ordering."""
import copy
import hashlib
import importlib.util
import json
from pathlib import Path
import sys
import unittest

sys.dont_write_bytecode = True


def load(name):
    path = Path(__file__).with_name(name + '.py')
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


broker = load('live-broker-cluster-check')
FAILED_MODEL = '{"limits":{"PendingOperations":1024,"Transitions":10000},"volumes":[{"CapacityBytes":8388608,"ID":"a-store","Path":"/workspace"}]}'
GO_MODEL = b'{"volumes":[{"ID":"a-store","Path":"/workspace","CapacityBytes":8388608}],"limits":{"PendingOperations":1024,"Transitions":10000}}'
GO_IDENTITY = 'sha256:12bc292756770d0d4c93ca63088d089d0c60789ef0853373c044716d1212f63b'


class StorageModelIdentity(unittest.TestCase):
    def test_original_failed_cell(self):
        model = json.loads(FAILED_MODEL)
        original = copy.deepcopy(model)
        descriptor = broker.storage({'Config': {'storage': model}})['descriptor']
        self.assertEqual(descriptor['model_identity'], GO_IDENTITY)
        self.assertIs(descriptor['model'], model)
        self.assertEqual(model, original)

    def test_typed_go_json_layout(self):
        codec = load('storage-model')
        scenario = load('campaign-scenario-check')
        model = json.loads(FAILED_MODEL)
        cases = [
            (model, GO_MODEL),
            ({'volumes': [{'Path': '/workspace', 'ID': 'a-store', 'CapacityBytes': 8388608}],
              'limits': {'Transitions': 10000, 'PendingOperations': 1024}}, GO_MODEL),
            ({'volumes': None, 'limits': {'Transitions': 2, 'PendingOperations': 1},
              'faults': None, 'sqlite_lock_databases': []},
             b'{"volumes":null,"limits":{"PendingOperations":1,"Transitions":2}}'),
            ({'volumes': [], 'limits': {'Transitions': 2, 'PendingOperations': 1}, 'faults': []},
             b'{"volumes":[],"limits":{"PendingOperations":1,"Transitions":2}}'),
            ({'volumes': [], 'limits': {'Transitions': 2, 'PendingOperations': 1},
              'faults': [{'bytes': 0, 'action': 'fail', 'occurrence': 3, 'operation': 'write',
                          'resource': 'a', 'name': '<&>\u2028\u2029'},
                         {'bytes': 7, 'action': 'partial', 'occurrence': 4, 'operation': 'read',
                          'resource': 'a', 'name': 'é'}], 'sqlite_lock_databases': ['<&>\u2028\u2029']},
             '{"volumes":[],"limits":{"PendingOperations":1,"Transitions":2},"faults":[{"name":"\\u003c\\u0026\\u003e\\u2028\\u2029","resource":"a","operation":"write","occurrence":3,"action":"fail"},{"name":"é","resource":"a","operation":"read","occurrence":4,"action":"partial","bytes":7}],"sqlite_lock_databases":["\\u003c\\u0026\\u003e\\u2028\\u2029"]}'.encode()),
        ]
        for model, expected in cases:
            with self.subTest(model=model):
                original = copy.deepcopy(model)
                self.assertEqual(codec.encode(model), expected)
                self.assertEqual(scenario.storage_model.encode(model), expected)
                descriptor = broker.storage({'Config': {'storage': model}})['descriptor']
                self.assertEqual(descriptor['model_identity'], 'sha256:' + hashlib.sha256(expected).hexdigest())
                self.assertIs(descriptor['model'], model)
                self.assertEqual(model, original)


if __name__ == '__main__':
    unittest.main()
