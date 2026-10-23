"""Receiver-only fixtures: never runtime/live qualification evidence."""
import sys
import hashlib
import json
from unittest.mock import patch
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver


class LiveReader(unittest.TestCase):
    def test_exact_owned_requirements_and_no_manifest_only_success(self):
        import live
        owned = live.required()
        self.assertEqual(len(owned), 124)
        self.assertEqual(len(set(owned)), 124)
        self.assertIn('test-wasm/complete-behavioral-reach', owned)
        for config in ({}, {'live_corpus': {'roots': list(range(1, 40)), 'status': 'PASS'}}):
            with self.subTest(config=config), self.assertRaises(driver.Refusal):
                live.read_corpus(config, Path('/unexecuted'))

    def test_process_journal_requires_actual_completion_and_exact_argv(self):
        import live
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            argv = ['/python', '/reader.py']
            command = dict(argv=argv, cwd='/repo', pid=42, exit=0, observation='completed', start_ns=1, end_ns=2)
            process = dict(argv=argv, pid=42, active=False, exit=0, observation='completed', start_ns=1, end_ns=2)
            for mutation in ('missing', 'pid', 'argv', 'active', 'inconclusive', 'exit', 'zero-time'):
                with self.subTest(mutation=mutation):
                    c, p = dict(command), dict(process)
                    if mutation == 'missing':
                        with self.assertRaises(driver.Refusal): live.completed_command(c, None, argv, '/repo')
                        continue
                    if mutation == 'pid': p['pid'] = 43
                    if mutation == 'argv': p['argv'] = ['/other']
                    if mutation == 'active': p['active'] = True
                    if mutation == 'inconclusive': p['observation'] = 'inconclusive'
                    if mutation == 'exit': c['exit'] = p['exit'] = 1
                    if mutation == 'zero-time': c['end_ns'] = p['end_ns'] = 1
                    with self.assertRaises(driver.Refusal): live.completed_command(c, p, argv, '/repo')
            live.completed_command(command, process, argv, '/repo')

    def test_admission_requires_both_original_encodings_every_cell_and_no_writes(self):
        import live
        variants = live.tampers()
        operations = ('verify', 'storage-selections', 'reopen', 'replay')
        rows = [dict(variant=v, operation=o, status=2, executable_writes=0) for v in variants for o in operations]
        for mutation in ('missing', 'duplicate', 'success', 'writes', 'unknown'):
            changed = [dict(row) for row in rows]
            if mutation == 'missing': changed.pop()
            if mutation == 'duplicate': changed[-1] = dict(changed[0])
            if mutation == 'success': changed[0]['status'] = 0
            if mutation == 'writes': changed[0]['executable_writes'] = 1
            if mutation == 'unknown': changed[0]['variant'] = 'unknown'
            with self.subTest(mutation=mutation), self.assertRaises(driver.Refusal): live.admission_rows(changed)
        live.admission_rows(rows)

    def test_private35_refuses_declared_iterations_without_actual_raw_comparisons(self):
        import live
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            driver.write(root / 'result.json', dict(schema='gomad.live-cluster-shutdown-reinsertion/v1', conditions='ordinary', unchanged_guest=True,
                iterations=[dict(iteration=n, live_verified=True, completed=2, reverse_insertion=bool(n % 2), world_bytes=1, network_bytes=1) for n in range(16)]))
            with self.assertRaises((driver.Refusal, OSError)): live.private35(root, live.PRIVATE35 + ' ... ok\ntest result: ok. 1 passed; 0 failed; 0 ignored;')

    def test_reach_rejects_unmapped_original_subtests_and_unknown_ignored_tests(self):
        import live
        original = {'wasi/original_test.go#TestOriginal', 'wasi/original_test.go#TestOriginal/subcase'}
        for entries in ({}, {next(iter(original)): {'evidence': ['transferred-live/TestAlias']}},
                        {key: {'evidence': ['unexecuted']} for key in original}):
            with self.subTest(entries=entries), self.assertRaises(driver.Refusal):
                live.reach_coverage(original, entries, {'transferred-live/TestOriginal'})
        live.reach_coverage(original, {key: {'evidence': ['transferred-live/TestOriginal']} for key in original},
                            {'transferred-live/TestOriginal'})


class RawBindingGuards(unittest.TestCase):
    def test_live_role_rejects_source_binary_feature_and_helper_mismatch(self):
        import live
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory); binary = root / 'binary'; binary.write_bytes(b'unit fixture')
            receipt = root / 'build.json'
            config = dict(source_sha256='a' * 64, identity=dict(host='b' * 64, helper='c' * 64),
                live_corpus=dict(roles={'host-fixture':dict(binary=str(binary),build_receipt=str(receipt))}))
            original = dict(exit=0, profile='release', source_sha256='a' * 64, binary_sha256=driver.sha(binary),
                helper_sha256='c' * 64, helper_features=[], role='host-fixture', features=['determinism-fixture'])
            for field, value in [('source_sha256','wrong'),('binary_sha256','wrong'),('helper_sha256','wrong'),
                                 ('helper_features',['diagnostic-profile']),('features',[]),('profile','debug'),('exit',1),('role','release')]:
                current = dict(original, **{field:value}); receipt.write_text(json.dumps(current))
                config['files'] = {str(binary):driver.sha(binary),str(receipt):driver.sha(receipt)}
                with self.subTest(field=field), self.assertRaises(driver.Refusal): live.role(config,'host-fixture')
            receipt.write_text(json.dumps(original)); config['files'][str(receipt)] = driver.sha(receipt)
            with self.assertRaises(driver.Refusal):live.role(config,'host-fixture')
            from unittest.mock import patch
            with patch('main_cli_build.read_build',return_value=original) as reader:
                self.assertEqual(live.role(config,'host-fixture')['binary'],str(binary))
                reader.assert_called_once_with(str(receipt),config,expected_role='host-fixture')

    def test_ordinary_libtests_reject_sealed_legacy_success_claims(self):
        import live
        for name in ('ordinary-helper-test', 'ordinary-host-test'):
            with self.subTest(role=name), tempfile.TemporaryDirectory() as directory:
                root=Path(directory);binary=root/'test-binary';binary.write_bytes(b'synthetic libtest')
                receipt=root/'legacy-build.json'
                receipt.write_text(json.dumps(dict(exit=0,profile='release',source_sha256='s'*64,
                    binary_sha256=driver.sha(binary),helper_sha256='helper',helper_features=[],role=name,features=[])))
                config=dict(source_sha256='s'*64,identity=dict(helper='helper'),
                    files={str(p):driver.sha(p) for p in (binary,receipt)},
                    live_corpus=dict(roles={name:dict(binary=str(binary),build_receipt=str(receipt))}))
                with self.assertRaisesRegex(driver.Refusal,'build receipt'):live.role(config,name)

    def test_private35_original_bounds_cannot_be_relaxed(self):
        import live
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'input.json'
            limits=dict(memory_bytes=256<<20,fuel=4000000000,wall_millis=60000,output_bytes=1<<20,evidence_bytes=64<<20)
            value=dict(configuration=dict(limits=limits,incarnations=[dict(limits=dict(limits)),dict(limits=dict(limits))],
                steps=[dict(action='start',handle=dict(node=n,incarnation=1)) for n in ('a','b')]))
            path.write_text(json.dumps(value));live.private35_input(path)
            for field in limits:
                changed=json.loads(json.dumps(value));changed['configuration']['limits'][field]+=1
                path.write_text(json.dumps(changed))
                with self.subTest(field=field), self.assertRaises(driver.Refusal):live.private35_input(path)

    def test_whole_reach_inventory_uses_original_git_objects_and_literal_sites(self):
        import live
        source=b'package p\nfunc TestOriginal(t *testing.T) {\n fixture := `\nfunc TestPhantom(t *testing.T) {}\n`\n t.Run("literal", func(t *testing.T) {})\n t.Run(fmt.Sprintf("dynamic-%d",n), fn)\n}\n'
        key='tools/gomad/internal/backends/wasm/wasi/original_test.go'
        blob='b'*40;commit='a'*40
        sites={key+'#TestOriginal@'+str(source.index(b't.Run')):['literal'],
               key+'#TestOriginal@'+str(source.index(b't.Run',source.index(b't.Run')+1)):['dynamic-7','dynamic-19']}
        mapping=dict(reference_commit=commit,makefiles={},sources={key:dict(blob=blob,sha256=hashlib.sha256(source).hexdigest())},subtests=sites)
        config=dict(repo='/unit-fixture',live_corpus=dict(original_reference_commit=commit))
        tree=('100644 blob '+blob+'\t'+key+'\0').encode()
        external=[]
        for path,name in live.SLIM_SOURCES.items():
            raw=('package p\nfunc '+name+'(t *testing.T) {}\n').encode();external.append(raw)
            tree+=('100644 blob '+blob+'\t'+path+'\0').encode()
            mapping['sources'][path]=dict(blob=blob,sha256=hashlib.sha256(raw).hexdigest())
        fixture=b'package fixture\nfunc TestExcluded(t *testing.T) { t.Run("excluded", f) }\n'
        fixture_path='tools/gomad/internal/backends/wasm/testdata/fixture/main_test.go'
        tree+=('100644 blob '+blob+'\t'+fixture_path+'\0').encode()
        mapping['sources'][fixture_path]=dict(blob=blob,sha256=hashlib.sha256(fixture).hexdigest())
        makes=[]
        for path in live.reach.MAKEFILES:
            raw=b'test-wasm: test-wasm-slim-refusals\n';makes.append(raw)
            tree+=('100644 blob '+blob+'\t'+path+'\0').encode()
            mapping['makefiles'][path]=dict(blob=blob,sha256=hashlib.sha256(raw).hexdigest())
        with patch.object(live.reach, 'inventory', wraps=live.reach.inventory),patch.object(live.subprocess,'check_output',side_effect=[tree,source,*external,fixture,*makes]):
            actual=live.original_callables(config,mapping)
        self.assertEqual(actual,{key+'#TestOriginal',*[site+'/'+case for site,cases in sites.items() for case in cases],
            *[path+'#'+name for path,name in live.SLIM_SOURCES.items()]})
        changed=json.loads(json.dumps(mapping));changed['subtests'][next(iter(sites))]=['renamed']
        with patch.object(live.subprocess,'check_output',side_effect=[tree,source,*external,fixture,*makes]),self.assertRaises(driver.Refusal):
            live.original_callables(config,changed)

    def test_actual_ignored_oracle_tests_cannot_be_replaced_with_declared_pass(self):
        import live
        with tempfile.TemporaryDirectory() as directory:
            for family in live.AUXILIARY:
                with self.subTest(family=family),self.assertRaises(driver.Refusal):live.auxiliary(Path(directory),family,'PASS')

    def test_final_live_proof_reopens_actual_raw_bytes_and_owning_stage(self):
        import live
        with tempfile.TemporaryDirectory() as directory:
            base=Path(directory);work=base/'live-private35';work.mkdir();payload=work/'raw';payload.write_bytes(b'actual receiver fixture')
            config=dict(root=directory,repo='/repo',source_sha256='a'*64,identity=dict(host='b'*64,helper='c'*64),_config_sha256='d'*64)
            entry=dict(stage='artifact',argv=['/ordinary-test','--exact'])
            stage=dict(stage='artifact',exit=0,status='PASS',source=config['source_sha256'],identity=config['identity'],config_sha256=config['_config_sha256'],subgates=[dict(id='selected',status='PASS',executed=True,exit=0)])
            driver.write(base/'receipt.json',stage)
            receipt=dict(argv=entry['argv'],cwd='/repo',pid=42,exit=0,observation='completed',start_ns=1,end_ns=2)
            process=dict(receipt,active=False)
            for name,value in [('command.json',receipt),('process.json',process)]:driver.write(base/('live-private35.'+name),value)
            for name in ('stdout','stderr'):(base/('live-private35.'+name)).write_bytes(b'raw command output')
            proof=dict(schema='gomad.rust-live-proof/v1',family='private35',source_sha256=config['source_sha256'],identity=config['identity'],config_sha256=config['_config_sha256'],files=live.inventory(work))
            for suffix in ('command.json','process.json','stdout','stderr'):proof[suffix.split('.')[0]+'_sha256']=driver.sha(base/('live-private35.'+suffix))
            proof_path=base/'live-private35.proof.json';driver.write(proof_path,proof)
            with patch.object(live,'planned',return_value=(entry,base,work)),patch.object(live,'subgates',return_value=['selected']),patch.object(live,'read_family',return_value=set()):
                live.read_proof(config,'private35')
                for field,value in [('source_sha256','foreign'),('config_sha256','foreign'),('family','other')]:
                    changed=dict(proof,**{field:value});proof_path.write_text(json.dumps(changed))
                    with self.subTest(field=field),self.assertRaises(driver.Refusal):live.read_proof(config,'private35')
                proof_path.write_text(json.dumps(proof));payload.write_bytes(b'changed')
                with self.assertRaises(driver.Refusal):live.read_proof(config,'private35')
                payload.write_bytes(b'actual receiver fixture');stage['status']='SKIP';(base/'receipt.json').write_text(json.dumps(stage))
                with self.assertRaises(driver.Refusal):live.read_proof(config,'private35')

    def test_runtime_preparation_aliases_retire_only_after_binding_original_targets(self):
        import live
        runtime = live.load('live-backend-runtime')
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory);work=root/'work';work.mkdir();stock=root/'stock';stock.mkdir()
            (stock/'real').write_bytes(b'compiler source')
            (stock/'alias').symlink_to('real')
            private=work/'go-unavailable';private.mkdir();(private/'real').write_bytes(b'compiler source')
            (private/'alias').symlink_to('real');(work/'compiler-link').symlink_to(work/'go')
            with self.assertRaises(driver.Refusal):live.inventory(work)
            runtime.retire_preparation_aliases(work,stock)
            proof=driver.read(work/'retired-preparation-aliases.json')
            self.assertEqual([row['path'] for row in proof['aliases']],['compiler-link','go-unavailable/alias'])
            self.assertFalse((work/'compiler-link').is_symlink());self.assertFalse((private/'alias').is_symlink())
            live.inventory(work)
            (private/'alias').symlink_to('unexpected');(work/'compiler-link').symlink_to(work/'go')
            with self.assertRaises(ValueError):runtime.retire_preparation_aliases(work,stock)
            self.assertTrue((work/'compiler-link').is_symlink())
            self.assertTrue((private/'alias').is_symlink())

    def test_whole_reach_refuses_omitted_original_external_slim_execution(self):
        import live
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'map.json'
            driver.write(path,dict(schema='gomad.original-test-wasm-reach/v1',entries={'original':dict(evidence=['actual'])}))
            config=dict(live_corpus=dict(test_wasm_reach=dict(map=str(path))))
            with patch.object(live,'sealed_input'),patch.object(live,'original_callables',return_value={'original'}):
                with self.assertRaises(driver.Refusal):live.test_wasm_reach(config,{'actual'})

    def test_original_slim_requires_real_tagged_run_not_same_named_ordinary_test(self):
        import live
        original=['/pinned/go','test','-json','-tags','gomad_wasm_slim,test_dep','-timeout=15m',
                  '-run','|'.join('^'+name+'$' for name in live.SLIM_SOURCES.values()),'-count=1','./common/archiver/provider','./temporal','./tests/testcore']
        live.slim_command(original)
        for index,value in ((4,'test_dep'),(5,'-timeout=16m'),(1,'list')):
            argv=original[:];argv[index]=value
            with self.subTest(index=index),self.assertRaises(driver.Refusal):live.slim_command(argv)

    def test_reach_retirement_and_collapse_need_reasons_and_an_evidenced_target(self):
        import live
        root, sub, gone = 'a_test.go#TestRoot', 'a_test.go#TestRoot@1/case', 'b_test.go#TestGoOnly'
        original = {root, sub, gone}
        good = {root: dict(evidence=['rust/gomad/receiver']),
                sub: dict(collapsed_into=root, reason='Named variant runs inside the root receiver.'),
                gone: dict(retired='Go-host-only implementation detail deleted with its code.')}
        live.reach_coverage(original, good, {'rust/gomad/receiver'})
        for key, bad in ((gone, dict(retired='')), (gone, dict(retired='two\nlines')),
                         (gone, dict(retired='reason', evidence=['rust/gomad/receiver'])),
                         (sub, dict(collapsed_into=gone, reason='retired target')),
                         (sub, dict(collapsed_into=root)), (sub, dict(collapsed_into='missing', reason='x')),
                         (root, dict(evidence=[], gap='port'))):
            entries = dict(good); entries[key] = bad
            with self.subTest(key=key, bad=bad), self.assertRaises(driver.Refusal):
                live.reach_coverage(original, entries, {'rust/gomad/receiver'})

    def test_original_double_selected_tests_need_distinct_actual_commands(self):
        import live
        root='original_test.go#TestVolumeConformanceCorpus'
        original={root,root+'/original-make/conformance',root+'/original-make/recursive'}
        event='go/package/TestVolumeConformanceCorpus@stage/validate@run/command-0001'
        entries={key:dict(evidence=[event]) for key in original}
        with self.assertRaises(driver.Refusal):live.reach_coverage(original,entries,{event})
        second='go/package/TestVolumeConformanceCorpus@stage/portable@run/command-0002'
        entries[root+'/original-make/recursive']['evidence']=[second]
        live.reach_coverage(original,entries,{event,second})

    def test_provider_reference_uses_exact_original_snapshot_after_retirement(self):
        import live
        checker=live.load('live-cluster-check')
        expected=(live.PARITY/'reference/live-broker-provider.go.txt').read_bytes()
        self.assertEqual(checker.provider_source(),expected)
        with tempfile.TemporaryDirectory() as directory,patch.object(checker,'CORPUS',Path(directory)):
            path=Path(directory)/'reference/live-broker-provider.go.txt';path.parent.mkdir()
            path.write_bytes(expected+b'changed')
            with self.assertRaises(AssertionError):checker.provider_source()
            path.unlink()
            with self.assertRaises(OSError):checker.provider_source()

    def test_slim_duplicate_and_overridden_go_flags_are_refused(self):
        import live
        pattern='|'.join('^'+name+'$' for name in live.SLIM_SOURCES.values())
        original=['/pinned/go','test','-json','-tags','gomad_wasm_slim,test_dep','-timeout=15m',
                  '-run',pattern,'-count=1','./common/archiver/provider','./temporal','./tests/testcore']
        for extra in (['-timeout=60m'],['-timeout','60m'],['-timeout=15m'],
                      ['-tags=test_dep'],['-tags','test_dep'],['-run','.*'],['-run='+pattern]):
            with self.subTest(extra=extra),self.assertRaises(driver.Refusal):live.slim_command(original+extra)
        for index,value in ((7,'.*'),(9,'./other')):
            changed=original[:];changed[index]=value
            with self.subTest(index=index),self.assertRaises(driver.Refusal):live.slim_command(changed)
        equals=['/pinned/go','test','-json','-tags=gomad_wasm_slim,test_dep','-timeout=15m',
                '-run='+pattern,'-count=1','./common/archiver/provider','./temporal','./tests/testcore']
        live.slim_command(equals)

    def test_inspection_expands_every_actual_decision_from_raw_choice_bytes(self):
        import live
        import copy
        with tempfile.TemporaryDirectory() as directory:
            work=Path(directory);base=work/'inspection-record';(base/'streams').mkdir(parents=True)
            raw=bytearray(5*96)
            for index,kind,flag in ((0,1,1),(1,4,0),(2,2,1),(3,3,0),(4,1,1)):
                raw[index*96+8]=kind;raw[index*96+9]=flag
            (base/'streams/choice').write_bytes(raw)
            boundaries=[dict(trace=trace,decision=ordinal) for ordinal,trace in enumerate((0,1,4))]
            driver.write(base/'result.json',dict(inspection=dict(complete=True,incarnations=[dict(registry=dict(closed=True))],boundaries=boundaries)))
            source=dict(fuel=4000000000,memory_bytes=256<<20,wall_millis=60000,output_bytes=1<<20,evidence_bytes=128<<20,model=dict(mode=1,diagnostics=False))
            driver.write(work/'inspection-record.input.json',source)
            commands=[]
            for ordinal,kind in enumerate((1,2,1)):
                name=f'inspection-kind{kind}-{ordinal}';output=work/name;output.mkdir()
                request=copy.deepcopy(source);request['model']['mode']=2;request['replay_trace_path']=str(base/'streams/choice')
                request['inspection']=dict(metadata_path=str(base/'streams/inspection'),decision=ordinal,dumps=['stacks','heap'],probes=[])
                driver.write(work/(name+'.input.json'),request)
                driver.write(output/'result.json',dict(prefix_verified=True,debug=dict(stacks='actual stopped stack')))
                (output/'heap.pprof').write_bytes(b'actual raw receiver fixture')
                commands.append(dict(name=name))
            cases=live.runtime_boundaries(work,commands)
            self.assertEqual(len(cases),3)
            self.assertIn('live-case/TestCooperativeInspectionEveryDecision/kind2-1',cases)
            with self.assertRaises(driver.Refusal):live.runtime_boundaries(work,commands[:-1])
            path=work/'inspection-kind2-1.input.json';changed=driver.read(path);changed['inspection']['decision']=0;path.write_text(json.dumps(changed))
            with self.assertRaises(driver.Refusal):live.runtime_boundaries(work,commands)

    def test_runtime_subvariant_derivation_is_closed_and_requires_actual_cases(self):
        import live
        root='tools/gomad/internal/backends/wasm/wasi/inspection_integration_test.go#TestCooperativeInspectionEveryDecision'
        site=root+'@6043'
        inventory=dict(sources={},makefiles={},callables={root:dict(original_runs=['recursive'])},
            subtest_sites={site:dict(expression='fmt.Sprintf("kind%d-%d", record.Kind, boundary.Decision)',expansions=[])})
        inventory['callables'].update({path+'#'+name:dict(original_runs=['slim']) for path,name in live.SLIM_SOURCES.items()})
        mapping=dict(reference_commit='a'*40,sources={},makefiles={},subtests={site:dict(derive='runtime-inspection-boundaries')})
        config=dict(repo='/receiver-fixture',live_corpus=dict(original_reference_commit='a'*40))
        with patch.object(live.reach,'inventory',return_value=inventory):
            with self.assertRaises(driver.Refusal):live.original_callables(config,mapping)
            actual=live.original_callables(config,mapping,{'live-case/TestCooperativeInspectionEveryDecision/kind2-1'})
            self.assertIn(site+'/kind2-1',actual)
            mapping['subtests'][site]['derive']='declared-count'
            with self.assertRaises(driver.Refusal):live.original_callables(config,mapping,actual)
