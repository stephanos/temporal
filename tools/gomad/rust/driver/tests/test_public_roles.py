"""Typed public-role boundaries; all build records here are synthetic test fixtures."""
import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import public_roles as gate
from driver import Refusal, read, sha


class PublicRoles(unittest.TestCase):
    def fixture(self, root):
        inventory = root / 'source.sha256'
        inventory.write_text('source inventory fixture\n')
        document = dict(schema=gate.SCHEMA, source_inventory=str(inventory), source_inventory_sha256=sha(inventory), roles={})
        receipts = {}
        for name, role in gate.ROLES.items():
            path = root / (role + '.json')
            binary = root / (role + '.binary')
            binary.write_bytes(('synthetic compiled role ' + role).encode())
            receipt = dict(role=role, profile='release', exit=0, features=gate.main_cli_build.ROLES[role],
                helper_features=gate.main_cli_build.helper_features(role), source_sha256='s'*64, head='h'*40, symlinks={},
                tools_plan='/synthetic/tools.json', gnu_plan='/synthetic/gnu.json', binary=str(binary), binary_sha256=sha(binary),
                host_sha256='host', helper_sha256='helper', inputs={str(binary):sha(binary)})
            path.write_text(json.dumps(receipt))
            receipts[str(path)] = receipt
            document['roles'][name] = dict(build_receipt=str(path), sha256=sha(path))
        path = root / 'builds.json'; path.write_text(json.dumps(document))
        def typed(path, expected_role=None):
            value = read(path)
            gate.require(value['role'] == expected_role, 'wrong closed role')
            return value
        return path, document, inventory, typed

    def test_all_seven_roles_require_typed_builds_and_complete_frozen_inputs(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);path,document,inventory,typed=self.fixture(root)
            with patch.object(gate.main_cli_build,'read_build',side_effect=typed) as actual, \
                 patch.object(gate,'source_inventory',return_value=inventory.read_text()):
                built,paths=gate.read_builds(path)
                self.assertEqual(set(built),set(gate.ROLES))
                self.assertEqual(actual.call_count,6)
                config=dict(source_sha256='s'*64,head='h'*40,symlinks={},files={p:sha(p) for p in paths},
                    binary=built['release']['binary'],binary_sha256=built['release']['binary_sha256'],identity=dict(host='host',helper='helper'))
                gate.read_builds(path,frozen=config)
                config['files'].pop(document['roles']['stress']['build_receipt'])
                with self.assertRaisesRegex(Refusal,'fully frozen'):gate.read_builds(path,frozen=config)

    def test_legacy_claims_foreign_roles_and_resealed_feature_changes_refuse(self):
        for mutation in ('legacy','missing','wrong-role','features','helper-features','source','receipt-hash'):
            with self.subTest(mutation=mutation),tempfile.TemporaryDirectory() as temporary:
                root=Path(temporary);path,document,inventory,typed=self.fixture(root)
                if mutation=='legacy':document['roles']['stress']={'features':gate.FEATURES['stress'],'exit':0,'profile':'release'}
                elif mutation=='missing':document['roles'].pop('stress')
                else:
                    target=Path(document['roles']['stress']['build_receipt']);receipt=read(target)
                    if mutation=='wrong-role':receipt['role']='release'
                    if mutation=='features':receipt['features']=[]
                    if mutation=='helper-features':receipt['helper_features']=[]
                    if mutation=='source':receipt['head']='foreign'
                    target.write_text(json.dumps(receipt))
                    if mutation!='receipt-hash':document['roles']['stress']['sha256']=sha(target)
                    else:document['roles']['stress']['sha256']='0'*64
                path.write_text(json.dumps(document))
                with patch.object(gate.main_cli_build,'read_build',side_effect=typed), \
                     patch.object(gate,'source_inventory',return_value=inventory.read_text()),self.assertRaises(Refusal):
                    gate.read_builds(path)

    def test_original_static_helper_build_must_be_independently_received_and_equal(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);binary=root/'helper';binary.write_bytes(b'synthetic helper')
            work=root/'stages/validate/static';work.mkdir(parents=True)
            result=dict(schema='gomad.driver-static-build/v1',role='conformance-reference',features=['conformance','reference-helper'],
                requested_features=['conformance'],binary=str(binary),binary_sha256=sha(binary))
            report=dict(actions=[dict(recipe=dict(kind='build-role'),results=[result])])
            (work/'result.json').write_text(json.dumps(report))
            consumed=root/'static-roles/conformance-reference/build.json';consumed.parent.mkdir(parents=True)
            consumed.write_text(json.dumps(result))
            config=dict(root=str(root),stages=dict(validate=dict(argv=['fixed','--plan','/plan'])))
            with patch('static_stages.authenticate_result') as actual:
                gate.static_reference(config,dict(binary_sha256=sha(binary)))
                actual.assert_called_once_with(Path('/plan'),'validate')
                forged=dict(result,binary='/different/consumed/helper')
                consumed.write_text(json.dumps(forged))
                with self.assertRaisesRegex(Refusal,'consumed a different'):gate.static_reference(config,dict(binary_sha256=sha(binary)))
                consumed.write_text(json.dumps(result))
                result['features']=['conformance'];(work/'result.json').write_text(json.dumps(report))
                with self.assertRaises(Refusal):gate.static_reference(config,dict(binary_sha256=sha(binary)))
