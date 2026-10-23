"""Synthetic receipt mutations only; actual Go oracle compilation/execution remains mandatory."""
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import sqlite_oracle as oracle
from driver import Refusal, sha


def put(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value) if not isinstance(value, str) else value)


class SQLiteOracle(unittest.TestCase):
    def fixture(self, base):
        repo, sdk, cache, root = (base/name for name in ('source','go','modules','build'))
        for path in (repo,sdk,cache,root,base/'go-cache'):
            path.mkdir()
        go = sdk/'bin/go'; put(go,'synthetic Go compiler');go.chmod(0o700)
        put(sdk/'VERSION','go1.27.1\n');put(sdk/'pkg/tool/linux_arm64/compile','synthetic compiler owner')
        put(repo/'go.mod','module go.temporal.io/server\ngo 1.27.1\n');put(repo/'go.sum','fixture root dependencies')
        put(repo/'tools/gomad/go.mod','module '+oracle.MODULE+'\ngo 1.27.1\n')
        put(repo/'tools/gomad/go.sum',''.join(f'{module} {version} {checksum}\n{module} {version}/go.mod h1:fixturemodsum=\n' for module,(version,checksum) in oracle.DEPENDENCIES.items()))
        put(repo/'tools/gomad/internal/model/cmd/sqlite-native/main.go','package main // fixture\n')
        put(repo/'tools/gomad/internal/model/wasi/storage_projection.go','package wasi // fixture\n')
        put(repo/'tools/gomad/internal/support/fs/read.go','package fs // fixture\n')
        dep = cache/(oracle.DEPENDENCY+'@'+oracle.VERSION)
        put(dep/'json.go','package json // fixture\n')
        for module,(version,checksum) in oracle.DEPENDENCIES.items():
            put(cache/(module+'@'+version)/'go.mod','module '+module)
            for suffix,content in [('.ziphash',checksum),('.zip','synthetic zip'),('.info','{}'),('.mod','module '+module)]:
                put(cache/'cache/download'/module/'@v'/(version+suffix),content)
        plan = dict(schema='gomad.sqlite-go-oracle-plan/v1',repo=str(repo),stock_go=str(go),go_cache=str(base/'go-cache'),module_cache=str(cache))
        plan_path=base/'plan.json';put(plan_path,plan)
        binary=root/'sqlite-native';put(binary,'synthetic binary: never executed');binary.chmod(0o700)
        env=oracle.environment(plan,root)
        for path,raw in oracle.verification_files(plan,root).items():path.write_bytes(raw)
        graph=[dict(ImportPath=oracle.IMPORT,Name='main',Dir=str(repo/'tools/gomad/internal/model/cmd/sqlite-native'),GoFiles=['main.go'],Imports=[oracle.MODULE+'/internal/model/wasi']),
               dict(ImportPath=oracle.MODULE+'/internal/model/wasi',Name='wasi',Dir=str(repo/'tools/gomad/internal/model/wasi'),GoFiles=['storage_projection.go'],Imports=[oracle.DEPENDENCY]),
               dict(ImportPath=oracle.DEPENDENCY,Name='json',Dir=str(dep),GoFiles=['json.go'],Imports=[],Module=dict(Path=oracle.DEPENDENCY,Version=oracle.VERSION))]
        outputs={'version':'go version go1.27.1 linux/arm64\n',
                 'environment':json.dumps(dict(GOROOT=str(sdk),GOVERSION='go1.27.1',GOTOOLDIR=str(sdk/'pkg/tool/linux_arm64'))),
                 'packages':'\n'.join(json.dumps(row) for row in graph),
                 'verify-modules':'all modules verified\n',
                 'build':'',
                 'build-info':f'{binary}: go1.27.1\n\tpath\t{oracle.IMPORT}\n\tmod\t{oracle.MODULE}\t(devel)\n\tdep\t{oracle.DEPENDENCY}\t{oracle.VERSION}\t{oracle.SUM}\n'+''.join('\tbuild\t'+key+'='+value+'\n' for key,value in {'-buildmode':'exe','-compiler':'gc','-tags':'test_dep','-trimpath':'true','CGO_ENABLED':'0','GOOS':'linux','GOARCH':'arm64'}.items())}
        for index,(name,argv) in enumerate(oracle.recipes(plan,root).items()):
            self.journal(root,name,argv,str(repo/'tools/gomad'),env,outputs[name],index+10)
        receipt=dict(schema='gomad.sqlite-go-oracle-build/v1',plan=str(plan_path),plan_sha256=sha(plan_path),repo=str(repo),stock_go=str(go),binary=str(binary),binary_sha256=sha(binary),environment=env,
                     sources=oracle.sources(repo),sdk=oracle.tree(sdk),dependency=oracle.dependency(plan),
                     commands={name:{suffix:sha(root/(name+'.'+suffix)) for suffix in oracle.SUFFIXES} for name in oracle.COMMANDS})
        put(root/'build.json',receipt)
        config=dict(repo=str(repo),primary=dict(compiler=str(go)),sqlite_oracle=dict(build_receipt=str(root/'build.json')))
        config['files']={path:sha(path) for path in oracle.freeze_inputs(config,config['sqlite_oracle'])}
        return config,root,receipt

    def journal(self,root,name,argv,cwd,env,stdout,pid=10):
        row=dict(argv=argv,cwd=cwd,pid=pid,exit=0,observation='completed',start_ns=100,end_ns=200,
                 clock='CLOCK_MONOTONIC',parent_pid=99,started_monotonic_nanos=10,reaped_monotonic_nanos=20,
                 environment_mode='exact',effective_environment=env)
        put(root/(name+'.command.json'),row)
        put(root/(name+'.process.json'),dict(row,active=False,stage_pid=99))
        put(root/(name+'.stdout'),stdout);put(root/(name+'.stderr'),'')

    def test_configure_requires_full_oracle_closure_and_rejects_legacy_argv(self):
        import configure
        with tempfile.TemporaryDirectory() as temporary:
            config,root,_=self.fixture(Path(temporary))
            config['stages']={'sqlite-catalogue':dict(adapter='sqlite-catalogue')}
            expected=oracle.freeze_inputs(config,config['sqlite_oracle'])
            self.assertEqual(configure.sqlite_oracle_inputs(config),expected)
            config['stages']['sqlite-catalogue']['oracle']=['/sealed/rust-host']
            with self.assertRaises(Refusal):configure.sqlite_oracle_inputs(config)
            config['stages']['sqlite-catalogue'].pop('oracle')
            config.pop('sqlite_oracle')
            with self.assertRaises(Refusal):configure.sqlite_oracle_inputs(config)

    def test_closed_build_and_freeze_inputs(self):
        with tempfile.TemporaryDirectory() as temporary:
            config,root,receipt=self.fixture(Path(temporary))
            self.assertEqual(oracle.validate_build(config),receipt)
            self.assertIn(str(Path(receipt['stock_go']).parent.parent/'VERSION'),oracle.freeze_inputs(config,config['sqlite_oracle']))
            for entry in ({'oracle':['/rust/host']},{'build_receipt':str(root/'build.json'),'argv':['/other']}):
                with self.assertRaises(Refusal):oracle.freeze_inputs(config,entry)

    def test_cross_source_compiler_executable_and_omitted_sdk_refuse(self):
        for mutation in ('source','compiler','binary','sdk','module','receipt-extra','command','clock','unreaped','build-info','source-owner','not-executable'):
            with self.subTest(mutation=mutation),tempfile.TemporaryDirectory() as temporary:
                config,root,receipt=self.fixture(Path(temporary))
                if mutation=='source':Path(config['repo']+'/tools/gomad/internal/model/wasi/storage_projection.go').write_text('changed Go algorithm')
                if mutation=='compiler':config['primary']['compiler']='/other/bin/go'
                if mutation=='binary':Path(receipt['binary']).write_text('Rust self-comparator')
                if mutation=='sdk':Path(receipt['stock_go']).parent.parent.joinpath('pkg/tool/linux_arm64/compile').unlink()
                if mutation=='module':Path(receipt['plan']).parent.joinpath('modules',oracle.DEPENDENCY+'@'+oracle.VERSION,'extra.go').write_text('unlisted dependency')
                if mutation=='receipt-extra':put(root/'build.json',dict(receipt,accepted=True))
                if mutation=='command':
                    row=json.loads((root/'build.command.json').read_text());row['argv'][-1]='./internal/rust-substitute';put(root/'build.command.json',row)
                if mutation=='clock':
                    row=json.loads((root/'build.process.json').read_text());row['clock']='wall';put(root/'build.process.json',row)
                if mutation=='unreaped':
                    row=json.loads((root/'build.process.json').read_text());row['active']=True;put(root/'build.process.json',row)
                if mutation=='build-info':(root/'build-info.stdout').write_text('Rust comparator accepts caller data')
                if mutation=='not-executable':Path(receipt['binary']).chmod(0o600)
                if mutation=='source-owner':config['repo']=str(Path(temporary)/'other-candidate')
                with self.assertRaises((Refusal,OSError)):oracle.validate_build(config)

    def test_coherent_receipt_repin_cannot_override_frozen_go_sources(self):
        with tempfile.TemporaryDirectory() as temporary:
            config,root,receipt=self.fixture(Path(temporary))
            source=Path(config['repo'])/'tools/gomad/internal/model/wasi/storage_projection.go'
            source.write_text('coherent other implementation')
            receipt['sources'][str(source)]=sha(source);put(root/'build.json',receipt)
            config['files'][str(root/'build.json')]=sha(root/'build.json')
            with self.assertRaises(Refusal):oracle.validate_build(config)

    def test_coherent_repin_does_not_authorize_changed_recipe_or_clock(self):
        for mutation in ('command','clock','go-main'):
            with self.subTest(mutation=mutation),tempfile.TemporaryDirectory() as temporary:
                config,root,receipt=self.fixture(Path(temporary))
                name='build'
                if mutation=='command':
                    for suffix in ('command.json','process.json'):
                        row=json.loads((root/(name+'.'+suffix)).read_text());row['argv'][-1]='./other-main';put(root/(name+'.'+suffix),row)
                if mutation=='clock':
                    for suffix in ('command.json','process.json'):
                        row=json.loads((root/(name+'.'+suffix)).read_text());row['clock']='CLOCK_REALTIME';put(root/(name+'.'+suffix),row)
                if mutation=='go-main':
                    name='build-info';(root/(name+'.stdout')).write_text((root/(name+'.stdout')).read_text().replace(oracle.IMPORT,'rust.self.comparison'))
                receipt['commands'][name]={suffix:sha(root/(name+'.'+suffix)) for suffix in oracle.SUFFIXES}
                put(root/'build.json',receipt)
                for path in config['files']:config['files'][path]=sha(path)
                with self.assertRaises(Refusal):oracle.validate_build(config)

    def test_actual_go_identity_refusal_prevents_rust_self_comparison_launch(self):
        from unittest.mock import patch
        with tempfile.TemporaryDirectory() as temporary:
            config,root,receipt=self.fixture(Path(temporary));run=Path(temporary)/'run';run.mkdir()
            request=run/'input.json';put(request,{'actual':'authenticated storage'})
            with patch.object(oracle,'command',return_value=b'not a Go executable') as invoke:
                with self.assertRaises(Refusal):oracle.invoke(config,run,request)
                self.assertEqual(invoke.call_count,1)
                self.assertEqual(invoke.call_args.args[3],[receipt['stock_go'],'version','-m',receipt['binary']])

    def test_independent_receiving_catalogue_uses_typed_oracle_without_legacy_argv(self):
        import receiving
        import sqlite
        from unittest.mock import patch
        with tempfile.TemporaryDirectory() as temporary:
            config,build,receipt=self.fixture(Path(temporary))
            attempt=Path(temporary)/'attempt';run=attempt/'stages/sqlite-catalogue';run.mkdir(parents=True)
            config.update(root=str(attempt),sent_epoch=0,binary='/sealed/release',stages={'sqlite-catalogue':dict(adapter='sqlite-catalogue')})
            request=run/'oracle-input.json';put(request,{'authenticated':'primary storage'})
            self.journal(run,'native-Go-oracle-build-info',[receipt['stock_go'],'version','-m',receipt['binary']],config['repo'],receipt['environment'],(build/'build-info.stdout').read_text())
            for suffix in ('process.json','command.json'):
                path=run/('native-Go-oracle-build-info.'+suffix);row=json.loads(path.read_text());row['started_monotonic_nanos']=1;row['reaped_monotonic_nanos']=9;put(path,row)
            self.journal(run,'native-Go-oracle',[receipt['binary'],'--input',str(request)],config['repo'],{'PATH':'/usr/bin:/bin'},json.dumps(dict(enumeration={},context={})))
            identity,_=oracle.run_identity(config,run,request,receipt);put(run/'native-Go-oracle.json',identity)
            reference=dict(sha256='a'*64,pid=5)
            primary=dict(report=dict(executions=[dict(artifact_path=str(attempt/'artifact'))]))
            native=dict(enumeration={},context={})
            selected=[(dict(sha256=f'{index:064x}',pid=index+100),dict(cost=dict(cpu_nanos=1),_driver_publication=dict(cpu_nanos=2))) for index in range(4)]
            put(run/'catalogue.json',dict(primary='a'*64,selected=sorted(row['sha256'] for row,_ in selected),producer_pids=[5,100,101,102,103],selected_execution_cpu_nanos=4,selected_terminal_cpu_nanos=8,selected_total_cpu_nanos=12,primary_cpu_counted=False))
            reader=receiving.Retained(config);actual_command=reader.command
            def command(cfg,root,name,argv,**kwargs):
                if name=='primary-inspect':return b'{}'
                return actual_command(cfg,root,name,argv,**kwargs)
            with patch.object(reader,'sqlite_primary',return_value=(reference,primary,native)),patch.object(reader,'selected',side_effect=lambda index:selected[index]), \
                 patch.object(reader,'command',side_effect=command),patch.object(sqlite,'native_input',return_value={'authenticated':'primary storage'}):
                reader.catalogue()
                self.assertIn(str(run/'native-Go-oracle.command.json'),reader.commands)
                self.assertIn(str(run/'native-Go-oracle-build-info.command.json'),reader.commands)
                put(request,{'authenticated':'changed primary'})
                with self.assertRaises(Refusal):reader.catalogue()

    def test_actual_run_journals_bind_fixed_argv_input_output_and_process(self):
        with tempfile.TemporaryDirectory() as temporary:
            config,root,receipt=self.fixture(Path(temporary));run=Path(temporary)/'run';run.mkdir()
            request=run/'input.json';put(request,{'actual':'authenticated storage'})
            argv=[receipt['binary'],'--input',str(request)]
            self.journal(run,'native-Go-oracle-build-info',[receipt['stock_go'],'version','-m',receipt['binary']],config['repo'],receipt['environment'],(root/'build-info.stdout').read_text())
            row=json.loads((run/'native-Go-oracle-build-info.process.json').read_text());row['started_monotonic_nanos']=1;row['reaped_monotonic_nanos']=9
            put(run/'native-Go-oracle-build-info.process.json',row)
            row=json.loads((run/'native-Go-oracle-build-info.command.json').read_text());row['started_monotonic_nanos']=1;row['reaped_monotonic_nanos']=9
            put(run/'native-Go-oracle-build-info.command.json',row)
            self.journal(run,'native-Go-oracle',argv,config['repo'],{'PATH':'/usr/bin:/bin'},json.dumps(dict(enumeration={},context={})))
            identity,_=oracle.run_identity(config,run,request,receipt);put(run/'native-Go-oracle.json',identity)
            self.assertEqual(oracle.read_run(config,run,request),dict(enumeration={},context={}))
            for mutation in ('input','output','process','actual-go-identity'):
                original={str(p):p.read_bytes() for p in run.iterdir()}
                if mutation=='input':put(request,{'actual':'another storage request'})
                if mutation=='output':put(run/'native-Go-oracle.stdout',dict(enumeration={'accepted':True},context={}))
                if mutation=='actual-go-identity':put(run/'native-Go-oracle-build-info.stdout','not a Go executable')
                if mutation=='process':
                    row=json.loads((run/'native-Go-oracle.process.json').read_text());row['pid']+=1;put(run/'native-Go-oracle.process.json',row)
                with self.subTest(mutation=mutation),self.assertRaises(Refusal):oracle.read_run(config,run,request)
                for path,raw in original.items():Path(path).write_bytes(raw)


if __name__=='__main__':unittest.main()
