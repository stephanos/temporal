import sys
from pathlib import Path
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import compose
from driver import Refusal, mapping


class CompositionTests(unittest.TestCase):
    def base(self):
        return dict(python=sys.executable, identity=dict(host='a' * 64, helper='b' * 64), stages={name: dict(adapter='pending') for name in mapping()})

    def test_missing_label(self):
        config = self.base()
        del config['stages']['storage']
        with self.assertRaisesRegex(Refusal, 'missing/foreign'):
            compose.validate_stages(config)

    def test_foreign_label(self):
        config = self.base()
        config['stages']['new-stage'] = dict(adapter='pending')
        with self.assertRaisesRegex(Refusal, 'missing/foreign'):
            compose.validate_stages(config)

    def test_pending_refuses_before_execution(self):
        with self.assertRaisesRegex(Refusal, 'unbound original stage endpoints'):
            compose.validate_stages(self.base())

    def test_unknown_candidate(self):
        config = self.base()
        config['identity']['host'] = '0' * 64
        with self.assertRaisesRegex(Refusal, 'unknown candidate'):
            compose.validate_stages(config)

    def test_wrong_semantic_adapter(self):
        config = self.base()
        config['stages']['runtime-0'] = dict(adapter='corpus')
        with self.assertRaisesRegex(Refusal, 'wrong semantic adapter'):
            compose.validate_stages(config)

    def test_live_exit_only_reader_refused(self):
        config = self.base()
        config['stages']['ordinary'] = dict(adapter='checked-command', reader='static-exit')
        with self.assertRaisesRegex(Refusal, 'fixed primary adapter required: ordinary'):
            compose.validate_stages(config)

    def test_ordinary_and_clusters_use_real_primary_routes_without_new_labels(self):
        config = self.base()
        for label in ('ordinary', 'clusters'):config['stages'][label] = dict(adapter='primary')
        with self.assertRaisesRegex(Refusal, 'unbound original stage endpoints') as error:
            compose.validate_stages(config)
        pending = str(error.exception).split(': ',1)[1].split(', ')
        self.assertNotIn('ordinary',pending)
        self.assertNotIn('clusters',pending)
        self.assertEqual(len(config['stages']),96)
        for label in ('ordinary','clusters'):
            changed = self.base()
            changed['stages'][label] = dict(adapter='primary',argv=['fake'])
            with self.assertRaisesRegex(Refusal,'fixed primary adapter required'):
                compose.validate_stages(changed)

    def test_typed_runtime_and_cells_replace_stale_manual_bindings(self):
        config = self.base()
        config.update(runtime=dict(harness='/sealed/runtime', harness_build_receipt='/sealed/runtime-build.json'),
                      installations={value['workload'] + '-' + value['profile']: '/sealed/installation'
                                     for value in compose.CELLS.values()}, sqlite_oracle=['/sealed/sqlite-oracle'])
        config['stages']['runtime-0'] = dict(adapter='pending', arguments=['/stale/checkout'])
        compose.bind_primitives(config, Path('/sealed/config.json'))
        self.assertEqual(set(config['stages']), set(mapping()))
        runtimes = [name for name, entry in config['stages'].items() if entry['adapter'] == 'runtime']
        self.assertEqual(set(runtimes), {f'runtime-{index}' for index in range(32)})
        self.assertEqual(config['stages']['runtime-aggregate'], dict(adapter='runtime-aggregate'))
        for name in runtimes:
            self.assertEqual(config['stages'][name]['arguments'], ['--working-dir', str(compose.REPO / 'tools/gomad')])
        for name in compose.CELLS:
            self.assertEqual(config['stages'][name]['spec'], 'GENERATE')
            self.assertIs(config['stages'][name]['private_observations'], True)
        self.assertEqual(config['stages']['sqlite-catalogue']['oracle'], ['/sealed/sqlite-oracle'])

    def test_unknown_runtime_role_and_missing_installation_refused(self):
        config = self.base()
        config['runtime'] = dict(harness='/unadmitted')
        with self.assertRaisesRegex(Refusal, 'sealed ordinary runtime'):
            compose.bind_primitives(config, Path('/sealed/config'))
        config['runtime']['harness_build_receipt'] = '/unadmitted-receipt'
        config['installations'] = {}
        with self.assertRaisesRegex(Refusal, 'missing actual mandatory installation'):
            compose.bind_primitives(config, Path('/sealed/config'))

    def test_existing_gnu_pinned_static_plan_is_recomputed_and_never_overwritten(self):
        import tempfile
        import json
        from unittest.mock import patch
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory);temporary=root/'temporary';temporary.mkdir();plan=root/'static-plan.json'
            declared={'schema':'gomad.driver-static-plan/v1','reviewed':'fixedactualtoolsandrecipes'}
            plan.write_text(json.dumps(declared));before=plan.read_bytes()
            def assemble(inputs,base,actual,output):
                actual.write_text(json.dumps(declared))
                return dict(stages={'validate':dict(argv=['python','static','--plan',str(actual)],checker=['python','static','--plan',str(actual)])},
                            inputs=[str(actual),'/actual/tool'])
            with patch.object(compose.static_stages,'assemble',side_effect=assemble):
                result=compose.assemble_static({}, {}, plan, temporary)
            self.assertEqual(plan.read_bytes(),before)
            self.assertEqual(result['stages']['validate']['argv'][-1],str(plan))
            self.assertEqual(result['stages']['validate']['checker'][-1],str(plan))
            self.assertEqual(result['inputs'],sorted([str(plan),'/actual/tool']))
            plan.write_text(json.dumps(dict(declared,reviewed='foreign')))
            with patch.object(compose.static_stages,'assemble',side_effect=assemble),self.assertRaisesRegex(Refusal,'provisioned static plan differs'):
                compose.assemble_static({}, {}, plan, temporary)

    def test_fixed_final_partition_and_order(self):
        sections = compose.final_sections('/sealed/python', Path('/sealed/driver.json'))
        self.assertEqual(len([item for row in sections.values() for item in row['required']]), 8)
        self.assertLess(sorted(sections).index('10-stress'), sorted(sections).index('90-public-verbs'))
        self.assertIn('--read-only', sections['40-retained']['argv'])
        self.assertNotIn('--read-only', sections['60-inspection']['argv'])


if __name__ == '__main__':
    unittest.main()
