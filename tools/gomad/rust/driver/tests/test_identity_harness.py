"""Closed harness raw custody positives and tampering; never invoke Rust or Cargo."""
import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
sys.path.insert(0,str(Path(__file__).resolve().parent))
import identity_harness as gate
import core_identity
from driver import Refusal,sha
from test_core_identity import fixture as identity_fixture,stdout


def fixture(root):
    config,context,rows=identity_fixture(root)
    repo=Path(config['repo']);attempt=root/'cycle';work=attempt/'stages/ordinary/identity-harness'
    work.mkdir(parents=True)
    distribution=root/'rust';tools={}
    for name in ('cargo','rustc','rustdoc'):
        path=distribution/'bin'/name;path.parent.mkdir(parents=True,exist_ok=True);path.write_text('pinned '+name);path.chmod(0o755);tools[name]=str(path)
    (distribution/'lib').mkdir();(distribution/'lib/compiler.so').write_text('pinned compiler implementation')
    home=root/'cargo-home'
    for name in ('src','cache','index'):
        path=home/'registry'/name;path.mkdir(parents=True);(path/'input').write_text('frozen dependency '+name)
    linker=root/'linker';linker.write_text('pinned linker');linker.chmod(0o755)
    python=root/'python';python.write_text('pinned Python')
    plan=root/'plan.json';plan.write_text(json.dumps(dict(schema='gomad.driver-static-plan/v1',repo=str(repo),driver_config=str(root/'driver.json'),
        tools=tools,caches=dict(cargo_home=str(home)))))
    native_plan=root/'gnu-toolchain.json'
    native_plan.write_text(json.dumps(dict(root=str(root/'gnu'),machine='aarch64-linux-gnu',linker=str(linker),
        programs=dict(ar=str(linker),ranlib=str(linker)),files={str(p):sha(p) for p in distribution.rglob('*') if p.is_file()})))
    config.update(root=str(attempt),python=str(python),_config_path=str(root/'driver.json'),_config_sha256='config',source_sha256='source',identity=dict(host='host',helper='helper'),sent_epoch=1,
                  stages=dict(validate=dict(argv=[str(python),'static','--plan',str(plan)])),identity_harness=dict(toolchain_plan=str(native_plan)))
    config['files'].update({str(p):sha(p) for p in root.rglob('*') if p.is_file() and not p.is_relative_to(attempt)})
    for name in ('home','tmp'):(work/name).mkdir()
    executable=work/'target/release/deps/implementation_identity-abcd';executable.parent.mkdir(parents=True);executable.write_bytes(b'synthetic compiled test artifact');executable.chmod(0o755)
    generated=work/'target/release/build/gomad-wasmhost-abcd/out';generated.mkdir(parents=True)
    files=core_identity.source_inputs(config)
    for name,raw in {'identity-compiler.txt':context['compiler_utf8'],'identity-bindings.tsv':''.join(k+'\t'+v+'\n' for k,v in sorted(context['bindings'].items())),
                     'measurement-source-inputs.tsv':gate.native_roles.source_manifest(files)}.items():(generated/name).write_text(raw)
    crate=repo/'tools/gomad/rust/wasmhost';package='path+'+crate.as_uri()+'#gomad-wasmhost@0.1.0'
    frames=[dict(reason='build-script-executed',package_id=package,out_dir=str(generated)),
            dict(reason='compiler-artifact',package_id=package,manifest_path=str(crate/'Cargo.toml'),executable=str(executable),
                 target=dict(name='implementation_identity',kind=['test'],crate_types=['bin'],src_path=str(crate/'tests/implementation_identity.rs')),
                 features=[],fresh=False,profile=dict(opt_level='3',debug_assertions=False,test=True)),dict(reason='build-finished',success=True)]
    (crate/'tests').mkdir(exist_ok=True);(crate/'tests/implementation_identity.rs').write_text('fixed integration harness source')
    config['files'][str(crate/'tests/implementation_identity.rs')]=sha(crate/'tests/implementation_identity.rs')
    def observed(name,argv,env,output,pid):
        (work/(name+'.stdout')).write_text(output);(work/(name+'.stderr')).write_text('')
        row=dict(argv=argv,cwd=config['repo'],pid=pid,exit=0,observation='completed',start_ns=1_100_000_000,end_ns=1_200_000_000,
                 environment=env,effective_environment=env,environment_mode='exact',temporary_paths={},clock='CLOCK_MONOTONIC',
                 parent_pid=100,started_monotonic_nanos=20,reaped_monotonic_nanos=80)
        process=dict(row,stage_pid=100,active=False)
        (work/(name+'.command.json')).write_text(json.dumps(row));(work/(name+'.process.json')).write_text(json.dumps(process))
    base=gate.environment(config,work);build_env=gate.environment(config,work,context['bindings']['TARGET'])
    for index,(name,argv,output) in enumerate((('rustc-version',[tools['rustc'],'--version'],context['compiler_utf8']),
        ('rustc-verbose',[tools['rustc'],'-vV'],context['compiler_utf8']+'host: '+context['bindings']['TARGET']+'\nrelease: 1.99.1\n'),
        ('cargo-version',[tools['cargo'],'--version'],'cargo 1.99.1 (synthetic)\n'))):observed(name,argv,base,output,201+index)
    observed('cargo-build',gate.cargo_command(config),build_env,'\n'.join(json.dumps(frame) for frame in frames),204)
    build=gate.build_evidence(config,work);(work/'build-receipt.json').write_text(json.dumps(build))
    (work/'admitted').mkdir();admitted=work/'admitted/implementation_identity';admitted.write_bytes(executable.read_bytes());admitted.chmod(0o755)
    admission=dict(schema=gate.ADMISSION,build_receipt=str(work/'build-receipt.json'),build_receipt_sha256=sha(work/'build-receipt.json'),
        cargo_artifact=str(executable),cargo_artifact_sha256=sha(executable),executable=str(admitted),executable_sha256=sha(admitted))
    (work/'admission.json').write_text(json.dumps(admission))
    text='running 5 tests\n'+''.join('test '+name+' ... ok\n' for name in core_identity.TESTS)+stdout(rows)+'\ntest result: ok. 5 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.01s\n'
    observed('identity-test',[str(admitted),'--nocapture'],build_env,text,205)
    observed('producer',gate.producer_argv(config,work,'produce'),base,'',100)
    for suffix in ('command','process'):
        path=work/('producer.'+suffix+'.json');row=json.loads(path.read_text());row.update(parent_pid=90,started_monotonic_nanos=10,reaped_monotonic_nanos=100)
        if suffix=='process':row['stage_pid']=90
        path.write_text(json.dumps(row))
    return config,work


class HarnessCustody(unittest.TestCase):
    def setUp(self):
        # The complete GNU closure has its own real parser tests; these cases isolate Cargo/harness custody.
        selected=patch.object(gate.native_gnu,'inspect',side_effect=lambda config,tools:(Path(config['identity_harness']['toolchain_plan']),json.loads(Path(config['identity_harness']['toolchain_plan']).read_text())))
        selected.start();self.addCleanup(selected.stop)

    def test_prefreeze_missing_cache_or_changed_closure_refuses_without_execution(self):
        for mutation in ('cache', 'GNU-digest'):
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as temporary:
                config,root=fixture(Path(temporary));gate.preflight(config)
                if mutation=='cache':config['files'].pop(next(path for path in config['files'] if '/registry/cache/' in path))
                if mutation=='GNU-digest':config['files'][next(path for path in config['files'] if path.endswith('/compiler.so'))]='0'*64
                with self.assertRaises(Refusal):gate.preflight(config)

    def test_positive_closed_build_admission_process_and_eighteen_facts(self):
        with tempfile.TemporaryDirectory() as temporary:
            config,root=fixture(Path(temporary))
            with patch.object(gate,'guard'):
                result=gate.collect(config,root)
            self.assertEqual(result['facts']['case_count'],18)
            self.assertNotIn('status',result)

    def test_cargo_artifact_feature_source_profile_and_target_tampering(self):
        for mutation in ('features','source','profile','target','duplicate','unfinished'):
            with self.subTest(mutation=mutation),tempfile.TemporaryDirectory() as temporary:
                config,root=fixture(Path(temporary));path=root/'cargo-build.stdout';frames=[json.loads(line) for line in path.read_text().splitlines()]
                if mutation=='features':frames[1]['features']=['reference-helper']
                if mutation=='source':frames[1]['target']['src_path']='/foreign/test.rs'
                if mutation=='profile':frames[1]['profile']['test']=False
                if mutation=='target':frames[1]['executable']='/foreign/implementation_identity-abcd'
                if mutation=='duplicate':frames.insert(0,copy.deepcopy(frames[1]))
                if mutation=='unfinished':frames[-1]['success']=False
                path.write_text('\n'.join(json.dumps(frame) for frame in frames))
                with patch.object(gate,'guard'),self.assertRaises(Refusal):gate.collect(config,root)

    def test_compiler_source_and_generated_build_context_cannot_be_claimed(self):
        for mutation in ('compiler','binding','source','distribution'):
            with self.subTest(mutation=mutation),tempfile.TemporaryDirectory() as temporary:
                config,root=fixture(Path(temporary));_,generated=gate.artifact(config,root)
                if mutation=='compiler':(root/'rustc-version.stdout').write_text('rustc 2.0.0 (caller)\n')
                if mutation=='binding':(generated/'identity-bindings.tsv').write_text('PROFILE\tdebug\nTARGET\tforeign\n')
                if mutation=='source':
                    path=Path(config['repo'])/'tools/gomad/rust/wasmhost/src/lib.rs';path.write_text('changed source');config['files'][str(path)]=sha(path)
                if mutation=='distribution':config['files'].pop(next(path for path in config['files'] if path.endswith('/compiler.so')))
                with patch.object(gate,'guard'),self.assertRaises(Refusal):gate.collect(config,root)

    def test_resealed_foreign_process_and_effective_environment_refuse(self):
        for mutation in ('parent','clock','environment'):
            with self.subTest(mutation=mutation),tempfile.TemporaryDirectory() as temporary:
                config,root=fixture(Path(temporary))
                for suffix in ('command','process'):
                    path=root/('identity-test.'+suffix+'.json');row=json.loads(path.read_text())
                    if mutation=='parent':row['parent_pid']=777;row['stage_pid']=777
                    if mutation=='clock':row['clock']='CLOCK_REALTIME'
                    if mutation=='environment':row['effective_environment']['LD_PRELOAD']='/foreign/library'
                    path.write_text(json.dumps(row))
                with patch.object(gate,'guard'),self.assertRaises(Refusal):gate.collect(config,root)

    def test_admitted_bytes_and_missing_case_do_not_become_success_claims(self):
        for mutation in ('admission','cases'):
            with self.subTest(mutation=mutation),tempfile.TemporaryDirectory() as temporary:
                config,root=fixture(Path(temporary))
                if mutation=='admission':(root/'admitted/implementation_identity').write_bytes(b'foreign')
                else:
                    path=root/'identity-test.stdout';text=path.read_text();text='\n'.join(line for line in text.splitlines() if 'identity.empty-closure' not in line);path.write_text(text)
                with patch.object(gate,'guard'),self.assertRaises(Refusal):gate.collect(config,root)


if __name__=='__main__':unittest.main()
